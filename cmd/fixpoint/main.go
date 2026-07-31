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
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
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

	var logMu sync.Mutex // reviewer goroutines log concurrently
	logf := func(format string, args ...any) {
		logMu.Lock()
		defer logMu.Unlock()
		// Redact before writing: reviewer/coder/git errors flow through here
		// verbatim, and a prompt-injected agent can smuggle a credential into one
		// (e.g. inside an invalid severity that validateReviewFindings echoes back).
		// Persisted logs already mask these; stderr and CI console logs must too.
		msg := agent.RedactSecrets(fmt.Sprintf(format, args...))
		fmt.Fprintf(stderr, "%s %s\n", time.Now().Format("15:04:05"), msg)
	}

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

	// Resolve and default without validating yet: the flags below override config
	// and change what a valid configuration is (e.g. -review-only exempts an
	// all-once lens list from the recurring-lens rule), so validation must run
	// against the EFFECTIVE configuration, after the overrides are applied.
	loaded, err := config.LoadBundle(resolver, name, projectRoot)
	if err != nil {
		logf("config: %v", err)
		return 1
	}
	cfg := loaded.Config
	logSource(logf, loaded)
	overrides{
		reviewOnly:        *reviewOnly,
		maxIter:           *maxIter,
		allowUntrustedFix: *allowUntrustedFix,
		trustedTarget:     *trustedTarget,
	}.apply(cfg)
	if !allowProjectSuppliedExec(loaded, logf) {
		return 1
	}

	if err := cfg.Validate(); err != nil {
		logf("config: %s: %v", loaded.Source.Config, err)
		return 1
	}

	o, err := orchestrator.New(cfg, loaded.Source, logf)
	if err != nil {
		logf("startup validation: %v", err)
		return 1
	}
	if *check {
		logf("configuration OK: %d review lens(es), coder %s, strategy %s",
			len(cfg.Roles.Review.Prompts), cfg.Roles.Coder.Agent, cfg.Roles.Review.Strategy)
		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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
	if err != nil {
		logf("run failed: %v", err)
		return 1
	}
	logf("done: %s after %d round(s)", sum.Termination, len(sum.Rounds))
	switch sum.Termination {
	case model.TermConverged, model.TermReviewOnly:
		return 0
	case model.TermAllRejected:
		// NOT 0. "The coder rejected every finding" is not "the code is clean" --
		// it could equally mean the reviewers are miscalibrated or the coder was
		// unwilling. Nothing changed, so automation keying on exit 0 would read a
		// no-op as a converged run. Distinct code, distinct meaning.
		logf("no changes were made: the coder rejected every finding this round")
		return 3
	case model.TermMaxIterations:
		return 2
	default: // interrupted, error
		return 1
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
		desc := c.Description
		if !c.Runnable {
			desc = strings.TrimSpace(desc + "  (base — for `extends`, not runnable)")
		}
		fmt.Fprintf(stdout, "%-14s %s\n", c.Name, desc)
		fmt.Fprintf(stdout, "%-14s %s\n", "", c.Path)
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

// sanitizeField flattens a value so it cannot break the tab-separated format.
func sanitizeField(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\t", " ")), " ")
}

// overrides are the CLI flags that change the effective configuration. They are
// applied before validation, because they change what a valid configuration is
// (-review-only exempts an all-once lens list from the recurring-lens rule).
type overrides struct {
	reviewOnly        bool
	maxIter           int
	allowUntrustedFix bool
	trustedTarget     bool
}

func (o overrides) apply(cfg *config.Config) {
	// The booleans can only force ON. Turning a gate off is the config's job, so a
	// flag can never silently weaken a run someone configured deliberately.
	if o.reviewOnly {
		cfg.Loop.ReviewOnly = true
	}
	if o.allowUntrustedFix {
		cfg.Loop.AllowUntrustedFix = true
	}
	if o.trustedTarget {
		cfg.Loop.TrustedTarget = true
	}
	// Any nonzero value applies, including a negative one: an explicit
	// -max-iterations -1 must reach Validate so its must-not-be-negative rule
	// rejects the bad input rather than being swallowed as "no flag supplied".
	// Zero stays "use config", per the flag help.
	if o.maxIter != 0 {
		cfg.Loop.MaxIterations = o.maxIter
	}
}

// allowProjectSuppliedExec gates bundle files that were resolved from inside the
// review target and that fixpoint executes -- agent commands and verify commands.
// Those are executable policy, not data: a repository shipping its own
// agents/*.yaml is supplying argv that fixpoint runs, which needs no model and no
// prompt injection to exploit. It therefore requires the same explicit trust
// assertion as letting the coder edit that repository.
func allowProjectSuppliedExec(l *config.Loaded, logf func(string, ...any)) bool {
	supplied := l.ProjectSuppliedExec()
	if len(supplied) == 0 || l.Config.Loop.TrustedTarget {
		return true
	}
	logf("refusing to run: the target supplies its own executable configuration, which fixpoint would execute:")
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
