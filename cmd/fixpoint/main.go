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
	// Pull a LEADING positional out before flag parsing. Go's flag package stops
	// at the first non-flag argument, so `fixpoint full-review --trusted-target`
	// would otherwise leave the flag unparsed -- and that particular flag is the
	// gate that permits file edits, so silently dropping it would be the worst
	// possible thing to get wrong. Taking it only from position 0 keeps this
	// unambiguous: a flag's value (e.g. `--max-iterations 3`) can never be
	// mistaken for the config name.
	var leading string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		leading, args = args[0], args[1:]
	}

	fs := flag.NewFlagSet("fixpoint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, `fixpoint - automated code-review loop

Usage:
  fixpoint <config> [flags]     run a named config from the bundle search path
  fixpoint ./some.yaml [flags]  run a config by path
  fixpoint --list               list the configs available here

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
	check := fs.Bool("check", false, "validate the configuration and exit without running")
	checkLive := fs.Bool("check-live", false, "validate the configuration, ping every agent, and exit without running")
	if err := fs.Parse(args); err != nil {
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

	if *list {
		return listConfigs(resolver, projectRoot, stdout, stderr)
	}

	name, err := configName(leading, fs.Args(), *cfgPath)
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
	case model.TermConverged, model.TermReviewOnly, model.TermAllRejected:
		return 0
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
func configName(leading string, trailing []string, flagPath string) (string, error) {
	names := trailing
	if leading != "" {
		names = append([]string{leading}, trailing...)
	}
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
// that says only "loaded full-review" hides the thing most worth knowing: whether
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
	for _, name := range sortedKeys(configs) {
		fmt.Fprintf(stdout, "%-20s %s\n", name, configs[name])
	}
	return 0
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
