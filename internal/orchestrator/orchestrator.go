// Package orchestrator runs the review->fix loop described in
// fixpoint.yaml: assign lenses to agents, run reviewers in parallel, hand
// findings to the coder, commit each fix round, and stop on convergence.
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/issue"
	"github.com/dsaiko/fixpoint/internal/logstore"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
	"github.com/dsaiko/fixpoint/internal/review"
	"github.com/dsaiko/fixpoint/internal/target"
	"github.com/dsaiko/fixpoint/internal/verify"
)

// Orchestrator drives one run: it owns the collector, the log store, and the
// parsed prompt templates, and executes rounds until a termination condition.
type Orchestrator struct {
	cfg       *config.Config
	source    config.Source
	collector *target.Collector
	logs      *logstore.Store
	// templates is keyed by prompt NAME, the same identity carried on assignments
	// and written into log filenames -- not by resolved path, so there is one
	// notion of "which lens is this" everywhere.
	templates map[string]*template.Template
	logf      func(format string, args ...any)
	// progress declares the run's STRUCTURE -- which phase a line belongs to -- so
	// the sink can render it. Optional: nil means every call degrades to an ordinary
	// logf line, which is what the tests (and anything embedding the orchestrator)
	// get without arranging for a renderer.
	progress Phaser
	// ci is what the forge reported about the reviewed head, when a provider could
	// be asked. Zero value means "not known", which never blocks a verdict but is
	// recorded in its reasons -- see internal/review.CI.
	ci review.CI
	// material is the round's collected diff, kept so the poster can tell which
	// lines a forge will accept a comment on.
	material string
	// commissioned are findings triage accepted from the pull request's comments,
	// waiting to join round 1. They are kept here rather than injected on the spot
	// because triage runs before the first round exists.
	commissioned []model.Finding
	// triageStep is that pass's I/O record, held until there is a round to bill it
	// to. Triage runs before round 1 and still costs tokens; a step that vanished
	// from the summary would make the run's reported cost wrong.
	triageStep model.StepStat
	// threads are the pull request's OPEN UNANSWERED review conversations, read
	// once at start. Empty for every target that is not a pull request, and for a
	// forge that cannot be asked.
	//
	// A thread leaves this list the moment it has been answered -- declined by
	// triage, or replied to by the session that fixed it. That is what makes a
	// conversation answerable at most once per run: conversations() renders this
	// list into every coder prompt and postReplies refuses an id that is not in it,
	// so a thread still here after N sessions is one no session has answered.
	threads []forge.Thread
	// commissionedThreads are the threads triage turned into issues, so a reply can
	// be matched against the session that owes it. Each such thread is answered by
	// the ONE session fixing its issue; another session naming it is answering a
	// question that was not asked of it.
	commissionedThreads map[string]bool
	// gitExclude holds the logs dir as a repo-relative path when it lives
	// inside target.path, so round commits and clean checks never touch the
	// run's own logs.
	gitExclude []string
	// verifyBaseline is how the project's checks behaved before any fix round, so
	// a pre-existing failure is not blamed on this run. Captured once at startup.
	verifyBaseline verify.Report
	// verifyEnv is the environment the gate's commands run with: fixpoint's, minus
	// the agents' credentials. Verify commands are argv the TARGET can supply, so
	// running them with everything the agents deliberately do not see is the one
	// place the env filtering could be walked around. Computed once at startup.
	verifyEnv []string
	// ledger groups raw observations into issues and carries their state across
	// rounds, so a problem two agents both reported costs one slot, not two.
	ledger *issue.Ledger
	// overrides names the CLI assertions that shaped this run, for the journal:
	// fix rounds are authorized by flag, never by config, so the bundle files alone
	// do not explain why editing was permitted.
	overrides []string
	// journalWarn keeps a broken journal to ONE warning. A full disk would
	// otherwise emit a line per transition and bury the run's real output.
	journalWarn sync.Once
	// preflight* memoize the guards. Both run() and Ping call them -- Ping because
	// -check-live reaches it WITHOUT going through run() -- and on the run path
	// that would otherwise repeat the git probes and, worse, print the
	// repo-supplied-git-config WARNING twice for one target. The guards are pure
	// reads of a target that does not change mid-run, so one answer holds.
	//
	// preflightAgents records the STRENGTH of the memoized answer: --check runs the
	// weaker no-agent variant (see PreflightGuardsNoAgent), and that answer must not
	// satisfy a later agent-invoking caller, which inspects strictly more.
	//
	// The memo is deliberately INVALIDATED once, by recheckPreflightGuards, at the
	// `gh pr checkout` inside Prepare: "the target does not change mid-run" does not
	// hold across a branch switch, because git config is branch-conditional.
	preflightMu     sync.Mutex
	preflightDone   bool
	preflightAgents bool
	preflightErr    error
	// warnedGuard keeps each guard warning to one line per run. The guards run a
	// second time after a pr checkout (recheckPreflightGuards), and a target whose
	// trust the operator asserted would otherwise print the identical warning
	// twice; a checkout that ACTIVATES further settings produces a different
	// message, which still gets printed.
	warnedGuard map[string]bool
	// heartbeatEvery is how often runAgent's progress line fires. It is a field
	// defaulting to heartbeatDefault rather than a bare const so a test can shorten
	// it: the tick body is the only thing that makes the heartbeat goroutine touch
	// logf, so without a seam the join below it (hb.Wait) is unobservable and could
	// be deleted with the whole suite still green.
	heartbeatEvery time.Duration
	// journalWrite is how a transition reaches the journal. It is a field holding
	// o.logs.Journal rather than a direct call, so a test can inject a failing
	// writer: "a journal failure never fails the run" is a load-bearing property
	// (its whole point is that losing an audit artifact must not discard fixes that
	// already passed verification), and a filesystem cannot be relied on to fail on
	// demand -- a read-only path still succeeds for root, which CI often is.
	journalWrite func(typ string, round int, data any) error
}

// openJournal writes the run's first record and reports where the journal lives.
//
// It is called from the middle of run(), not the top, because this is the first
// point at which writing a TRANSITION record is safe. The symlink check that
// immediately precedes it is what establishes that an artifact write lands inside
// the lexical logs exclusion rather than somewhere a later `git add -A` would sweep
// into a commit -- and every record from here on is followed by rounds that commit.
// Journaling the gates above would mean writing through exactly the redirected path
// that check exists to refuse.
//
// Run's closing EvRunFinished is deliberately NOT gated this way, so a run refused
// by a gate still leaves a one-record journal naming the refusal. That write carries
// no commit-sweep exposure, because nothing commits after it. The one refusal it is
// NOT safe after is the symlink check's own: writing outside the target is the other
// half of what that check prevents, so Run skips both closing writes on that path.
func (o *Orchestrator) openJournal() {
	o.journal(model.EvRunStarted, 0, model.JournalRunStarted{
		Config:        o.source.Config,
		Mode:          string(o.cfg.Target.Mode),
		Path:          o.cfg.Target.Path,
		Strategy:      string(o.cfg.Roles.Review.Strategy),
		ReviewOnly:    o.cfg.Loop.ReviewOnly,
		MaxIterations: o.cfg.Loop.MaxIterations,
		Overrides:     o.overrides,
	})
	if p := o.logs.JournalPath(); p != "" {
		o.logf("journal: %s", p)
	}
}

// journal records one state transition, best-effort.
//
// A journal failure never fails the run. The journal is an audit artifact; losing
// it must not discard fixes that already passed verification and were committed.
// The inverse -- failing the run to protect the audit trail -- would make the
// observability feature the most likely cause of a lost round.
func (o *Orchestrator) journal(typ string, round int, data any) {
	if err := o.journalWrite(typ, round, data); err != nil {
		o.journalWarn.Do(func() {
			o.logf("WARNING: run journal unavailable (%v); the run continues without it", err)
		})
	}
}

// New sets up the log store, parses every referenced prompt and renders it once
// with the zero value of its role's data type (so a placeholder belonging to the
// wrong role fails now rather than mid-run), and wires the collector and its
// log-directory exclusion. It returns an orchestrator ready to Run.
//
// New does NOT perform the config-level startup validation (agents defined and
// on PATH, strategy satisfiable): that lives in config.Validate, which the
// caller must run beforehand.
//
// It takes the whole *config.Loaded rather than a Config and a Source, because a
// run is defined by all three of its parts: the effective configuration, the files
// it was built from, and the flag assertions that authorized it. Splitting them at
// this boundary is how the last one went unrecorded.
func New(l *config.Loaded, logf func(string, ...any)) (*Orchestrator, error) {
	cfg, source := l.Config, l.Source
	logs, err := logstore.New(cfg.Logs)
	if err != nil {
		return nil, err
	}
	// Parse every referenced prompt up front (part of startup validation) and
	// render it once with the zero value of its role's data type, so a
	// placeholder belonging to the other role fails now instead of mid-run.
	templates := map[string]*template.Template{}
	load := func(name, path string, data any) error {
		// Parsing is cached per name, but the render check runs for every
		// (name, role) pairing: the same file may serve both roles, and
		// skipping the second role would let its placeholders fail mid-run.
		t, ok := templates[name]
		if !ok {
			var err error
			t, err = prompt.Load(path)
			if err != nil {
				return err
			}
		}
		if _, err := prompt.Render(t, data); err != nil {
			return err
		}
		templates[name] = t
		return nil
	}
	if cfg.Roles.Coder.Prompt != "" {
		if err := load(cfg.Roles.Coder.Prompt, cfg.Roles.Coder.PromptFile(), prompt.FixData{}); err != nil {
			return nil, err
		}
	}
	if cfg.Review.Refute != "" {
		if err := load(cfg.Review.Refute, cfg.Review.RefutePath, prompt.RefuteData{}); err != nil {
			return nil, err
		}
	}
	if t := cfg.Roles.Triage; t.Prompt != "" {
		if err := load(t.Prompt, t.PromptPath, prompt.TriageData{}); err != nil {
			return nil, err
		}
	}
	if j := cfg.Roles.Judge; j.Prompt != "" {
		if err := load(j.Prompt, j.PromptPath, prompt.JudgeData{}); err != nil {
			return nil, err
		}
	}
	for _, l := range cfg.Roles.Review.Prompts {
		if err := load(l.Prompt, l.PromptFile(), prompt.ReviewData{}); err != nil {
			return nil, err
		}
	}
	o := &Orchestrator{
		cfg:          cfg,
		source:       source,
		collector:    target.New(cfg.Target),
		logs:         logs,
		templates:    templates,
		logf:         logf,
		ledger:       issue.NewLedger(),
		overrides:    l.Overrides.Applied(),
		journalWrite: logs.Journal,
		verifyEnv:    agent.EnvWithoutCredentials(cfg.Agents),

		heartbeatEvery: heartbeatDefault,
	}
	// A `gh pr checkout` invalidates the preflight's verdict on the target, so the
	// gates run again the moment it lands -- before Prepare's own next git command.
	o.collector.OnCheckout(o.recheckPreflightGuards)
	// Exclude the template's literal prefix, not the rendered path: the rendered
	// path changes every run (and every round), so only the static base is a
	// stable pathspec for commits, clean checks, and collection.
	if rel, err := logsDirWithin(cfg.Logs.StaticBase(), cfg.Target.Path); err == nil && rel != "" {
		// "." would mean "exclude the whole repository": GitClean and Commit
		// would then see every tree as clean and silently skip all changes.
		if rel == "." {
			return nil, fmt.Errorf("logs.dir %s: its literal prefix resolves to target.path itself; put the logs under a subdirectory so round commits and clean checks can exclude them", cfg.Logs.Dir)
		}
		o.gitExclude = []string{rel}
		// Also keep the logs out of the material the collector gathers, so later
		// rounds never review the run's own prompts and outputs (directory walks
		// and untracked-file listings), not just the git commits/clean checks.
		o.collector.ExcludeLogs(rel)
	}
	return o, nil
}

// Phaser receives the run's structure: which block is open, and what closed it.
// internal/runlog implements it; anything that only wants the text can ignore it
// and read logf, which every phase call also has a plain equivalent for.
type Phaser interface {
	Rule(format string, args ...any)
	Phase(format string, args ...any)
	EndPhase(format string, args ...any)
	Progress(format string, args ...any)
}

// WithProgress installs the structure sink and returns o, so a caller can build
// and configure in one expression. Safe to call with nil.
func (o *Orchestrator) WithProgress(p Phaser) *Orchestrator {
	o.progress = p
	return o
}

// rule, phase, endPhase and progress are the four structural log calls. Each
// falls back to an ordinary line when no sink is installed, so the orchestrator
// never has to ask whether one is -- and a test reading the log still sees every
// word, just without the shape.
func (o *Orchestrator) rule(format string, args ...any) {
	if o.progress != nil {
		o.progress.Rule(format, args...)
		return
	}
	o.logf("=== "+format+" ===", args...)
}

func (o *Orchestrator) phase(format string, args ...any) {
	if o.progress != nil {
		o.progress.Phase(format, args...)
		return
	}
	o.logf(format, args...)
}

func (o *Orchestrator) endPhase(format string, args ...any) {
	if o.progress != nil {
		o.progress.EndPhase(format, args...)
		return
	}
	o.logf(format, args...)
}

func (o *Orchestrator) progressf(format string, args ...any) {
	if o.progress != nil {
		o.progress.Progress(format, args...)
		return
	}
	o.logf(format, args...)
}

// logsDirWithin returns the logs dir as a path relative to the target root if
// it lives inside it, "" otherwise.
//
// It measures twice. The LEXICAL form comes first and wins whenever it says
// "inside", because it is the form checkLogsNotSymlinked walks: a logs path that
// is lexically inside the target but reached through a symlink must still set the
// exclusion, so that check can see it and refuse the run.
//
// Only when the lexical form says "outside" is the CANONICAL form tried, because
// the two operands are not normalized the same way: config.ProjectRoot returns a
// symlink-resolved root (which a relative logs.dir is anchored to), while
// Config.anchor leaves an absolute target.path exactly as written. So
// `target.path: /tmp/checkout` with the default logs.dir compares
// /private/tmp/checkout/.fixpoint against /tmp/checkout -- one physical directory
// under two spellings, which Rel reads as two separate trees. Dropping the
// exclusion there is not benign: git runs with cmd.Dir = target.path, i.e. the
// same physical repository, so the run's own prompts and raw outputs would be
// staged into a round commit, collected as material for later rounds, and read as
// dirt by the clean checks. internal/target resolves both sides for exactly this
// case (fileScope, WorktreeOutOfScope).
//
// Retrying canonically can only ADD an exclusion, never remove one, so it cannot
// weaken either check that consumes the result.
func logsDirWithin(logsDir, root string) (string, error) {
	absLogs, err := filepath.Abs(logsDir)
	if err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if rel, ok := relWithin(absRoot, absLogs); ok {
		return rel, nil
	}
	rel, _ := relWithin(resolveExisting(absRoot), resolveExisting(absLogs))
	return rel, nil
}

// relWithin returns child as a slash-separated path relative to parent, and
// whether it lies inside parent at all. "." (child IS parent) counts as inside;
// the caller refuses that shape with its own message.
func relWithin(parent, child string) (string, bool) {
	rel, err := filepath.Rel(parent, child)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// resolveExisting returns path with the symlinks in it resolved, resolving the
// deepest leading part that exists on disk and re-appending the rest.
// filepath.EvalSymlinks refuses a path with a missing component, and the logs
// directory usually does not exist yet -- the first round creates it -- while the
// symlinked component that makes two spellings of one directory disagree
// (/tmp -> /private/tmp) is always one that does exist. A path where nothing
// resolves keeps its lexical form: there is nothing better to compare.
func resolveExisting(path string) string {
	rest := ""
	for cur := path; ; {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(resolved, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return path
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// errLogsRedirected marks the symlinked-logs refusal so Run can recognize it and
// suppress its own closing writes. Those writes go through the very path the check
// refused, so they would create a directory and drop artifacts wherever the
// checkout pointed the logs dir -- the second half of what the check exists to
// prevent (see checkLogsNotSymlinked and Run).
var errLogsRedirected = errors.New("logs directory is redirected by a symlink")

// checkLogsNotSymlinked verifies that no path component of the in-target logs
// directory is a symlink, so artifacts cannot be redirected outside the lexical
// logs exclusion (o.gitExclude) and swept into a round commit, nor written outside
// the target at all. It is a no-op when the logs dir lives outside the target
// (o.gitExclude unset) or when the logs path does not exist yet (it will then be
// created as a real directory). Called after Prepare because a PR checkout can
// change the path from a plain directory into a symlink -- and again from Run
// before the closing writes, which happen even when the run ended before that
// first call (see Run).
func (o *Orchestrator) checkLogsNotSymlinked() error {
	if len(o.gitExclude) == 0 {
		return nil
	}
	root, err := filepath.Abs(o.cfg.Target.Path)
	if err != nil {
		return err
	}
	cur := root
	for _, comp := range strings.Split(filepath.ToSlash(o.gitExclude[0]), "/") {
		cur = filepath.Join(cur, comp)
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // not yet created; ensureDir will make it a real directory
			}
			return fmt.Errorf("inspect logs path %s: %w", cur, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: logs path component %s is a symlink; the prepared target (e.g. a PR) may have redirected the logs directory to bypass the artifact exclusion and leak prompts/outputs into a commit. Point logs.dir outside the target, or remove the symlink", errLogsRedirected, cur)
		}
	}
	return nil
}

// Run executes the full loop and always writes a run summary before
// returning. The returned summary's Termination field says how it ended.
func (o *Orchestrator) Run(ctx context.Context) (*model.RunSummary, error) {
	sum := &model.RunSummary{
		StartedAt:  time.Now(),
		ConfigPath: o.source.Config,
		Sources: model.RunSources{
			Config:  o.source.Config,
			Extends: o.source.Extends,
			Agents:  o.source.Agents,
			Prompts: o.source.Prompts,
		},
		Mode:                string(o.cfg.Target.Mode),
		PR:                  o.cfg.Target.PR,
		Path:                o.cfg.Target.Path,
		Strategy:            string(o.cfg.Roles.Review.Strategy),
		ReviewOnly:          o.cfg.Loop.ReviewOnly,
		MaxIterations:       o.cfg.Loop.MaxIterations,
		MaxFindingsPerRound: o.cfg.Loop.MaxFindingsPerRound,
		Overrides:           o.overrides,
		CommitPolicy:        o.cfg.Loop.CommitPolicy,
		Coder:               o.cfg.Roles.Coder.Agent,
	}
	err := o.run(ctx, sum)
	sum.FinishedAt = time.Now()
	// Recognize the refusal BEFORE recordRunError, which returns nil for a
	// cancellation and would erase both the marker and the text.
	redirected, refusal := errors.Is(err, errLogsRedirected), err
	if err != nil {
		err = recordRunError(ctx, sum, err)
	}
	// run() only reaches checkLogsNotSymlinked when Prepare SUCCEEDED, but the
	// checkout Prepare performs is itself what can plant the symlink: a pr run whose
	// `gh pr checkout` turned the in-target logs dir into a link and then failed (or
	// was canceled) on a later Prepare step returns an error that is not
	// errLogsRedirected, and used to write both closing artifacts straight through
	// the planted link. The closing writes are the thing being protected, so decide
	// here -- where they happen -- rather than trusting how the run ended. The check
	// is a no-op unless the logs dir lives inside the target.
	if !redirected {
		if cerr := o.checkLogsNotSymlinked(); errors.Is(cerr, errLogsRedirected) {
			redirected, refusal = true, cerr
			// A run that otherwise SUCCEEDED has no other way to report this: the
			// refusal suppresses both closing writes, so leaving err nil would exit 0
			// and print a converged outcome for a run whose journal and summary do not
			// exist. recordRunError demotes that outcome to LoopTermination and marks
			// the run itself failed. An error already recorded stands -- it came first,
			// and this one is reported to the log either way.
			if err == nil {
				err = recordRunError(ctx, sum, cerr)
			}
		}
	}
	// A run refused because the logs path is symlinked writes NOTHING: both closing
	// writes reach Store.ensureDir, which resolves the link at syscall time and would
	// create the run directory -- plus the journal and summary in it -- at whatever
	// path the checkout pointed the logs dir to. Keeping artifacts out of such a path
	// is half of what the check is for, so the refusal is reported to the log instead.
	if redirected {
		o.logf("WARNING: no journal or summary written: %v", refusal)
		return sum, err
	}
	// Journal the outcome BEFORE writing the summary, so the ordering on disk
	// matches reality: the summary is a whole-run rewrite that can itself fail,
	// and its failure is exactly when the journal has to already hold the verdict.
	o.journal(model.EvRunFinished, 0, model.JournalRunFinished{
		Termination:     sum.Termination,
		LoopTermination: sum.LoopTermination,
		Rounds:          len(sum.Rounds),
		Error:           sum.Error,
	})
	if mdPath, werr := o.logs.Summary(sum); werr != nil {
		o.logf("WARNING: failed to write summary: %v", werr)
	} else {
		o.logf("summary written: %s", mdPath)
	}
	return sum, err
}

// recordRunError records a failed run in the summary and returns what Run should
// hand back to the CLI: nil for a clean interruption, the error otherwise.
//
// It applies to EVERY non-nil error, including one raised after the loop already
// set a termination. The closing round runs after that point, so a failed final
// reviewer, coder, collection, or commit used to reach the CLI while the summary
// and the journal still recorded converged / all-rejected / max-iterations with no
// error -- durable artifacts claiming success for work that failed.
//
// Cancellation mid-step (e.g. Ctrl-C while the coder runs) surfaces as that step's
// error -- typically a kill message from agent.Run. It is recorded as an
// interruption, like the loop's own ctx checks do, so the summary distinguishes
// "user stopped it" from "it failed". A dirty tree left behind because the
// interrupt-time stash failed is a hard failure, not a clean interruption: it is
// surfaced and recorded even though ctx was canceled, so the user is never told
// the run stopped cleanly while unreconciled edits sit in the tree.
func recordRunError(ctx context.Context, sum *model.RunSummary, err error) error {
	if ctx.Err() != nil && !errors.Is(err, errInterruptedTreeDirty) {
		// A cancellation during the closing round is still an interruption of the
		// RUN, so markInterrupted demotes the loop's own outcome to LoopTermination
		// rather than letting "converged" stand and exit 0 over closing work the
		// operator stopped part-way.
		markInterrupted(sum)
		return nil
	}
	// Keep the loop's own outcome visible: "the loop converged but the closing
	// round failed" is a different fact from "the run failed", and the exit status
	// has to report the failure either way.
	if sum.Termination != "" && sum.Termination != model.TermError {
		sum.LoopTermination = sum.Termination
	}
	sum.Termination = model.TermError
	sum.Error = err.Error()
	return err
}

// markInterrupted records that the operator stopped the run, keeping whatever the
// LOOP had already decided visible in LoopTermination.
//
// The closing phase runs after the loop has set its termination, so a Ctrl-C there
// used to leave "converged" standing: the run reported a clean finish and exited 0
// (model.ExitCode) while configured closing work was left undone -- against the
// CLI's contract that an interruption exits 1. "The loop converged but the operator
// stopped the closing phase" is the same shape as "the loop converged but the
// closing round failed", so it is recorded the same way.
//
// An outcome that is already interrupted or error is left alone: it is not the
// loop's own verdict to preserve, and copying it into LoopTermination would make
// the summary print the same word twice.
func markInterrupted(sum *model.RunSummary) {
	if sum.Termination != "" && sum.Termination != model.TermInterrupted && sum.Termination != model.TermError {
		sum.LoopTermination = sum.Termination
	}
	sum.Termination = model.TermInterrupted
}

func (o *Orchestrator) run(ctx context.Context, sum *model.RunSummary) error {
	o.warnArgModePrompts()
	o.warnInheritedEnv()
	o.warnTargetSuppliedCommand()

	// Enforce the fix-round trust gate; see checkFixTrust for the rationale.
	if err := o.checkFixTrust(); err != nil {
		return err
	}

	if err := o.PreflightGuards(ctx); err != nil {
		return err
	}

	// Cheap read-only git checks first. Fix rounds commit; require a clean
	// git tree so every round commit contains exactly the coder's changes and
	// nothing of yours. PR review requires a clean tree too, even review-only:
	// gh pr checkout can preserve unrelated tracked edits and untracked files,
	// which Collect would otherwise fold into the PR diff and misattribute the
	// resulting findings to the PR.
	requireCleanTree := !o.cfg.Loop.ReviewOnly || o.cfg.Target.Mode == config.ModePR
	if !o.cfg.Loop.ReviewOnly {
		// A probe that could not answer is not an answer: surface it rather than
		// reporting a repository as missing because git was canceled or wedged.
		isRepo, err := o.collector.IsGitRepo(ctx)
		if err != nil {
			return err
		}
		if !isRepo {
			return fmt.Errorf("target.path %s is not a git repository; fix rounds commit each round and require git (use review_only for a report-only run)", o.cfg.Target.Path)
		}
	}
	// A run that writes claims the repository: root check, whole-repository lock,
	// clean tree. Review-only directory runs read and never write, so they skip all
	// three -- they neither take the lock nor are blocked by one.
	if requireCleanTree {
		release, err := o.claimRepo(ctx)
		if err != nil {
			return err
		}
		defer release()
	}

	// Then ping (spends a little on each agent), then Prepare (may mutate
	// state -- pr checkout): each stage fails before the next spends more.
	if o.cfg.Ping() {
		o.phase("PREFLIGHT  pinging %d agent(s)", len(o.activeAgentNames()))
		if err := o.Ping(ctx); err != nil {
			o.endPhase("PREFLIGHT  failed")
			return err
		}
		o.endPhase("PREFLIGHT  every agent responded")
	}

	// Prepare can switch branches (pr mode runs gh pr checkout), which invalidates
	// what the preflight above learned: git config is branch-conditional, so an
	// includeIf "onbranch:..." can activate an execution-capable setting that was
	// inert on the branch the preflight inspected. The gates therefore run again
	// from inside Prepare, the instant the checkout lands and before Prepare's own
	// next git command -- see recheckPreflightGuards, wired in New.
	if err := o.collector.Prepare(ctx); err != nil {
		return err
	}

	// The logs exclusion (o.gitExclude) is a LEXICAL path fixed before Prepare,
	// but Prepare can check out a PR that turns the in-target logs directory into
	// a symlink pointing back into the repository. Artifacts written through such
	// a symlink land under a different physical path that the lexical exclusion
	// misses, so `git add -A` would stage them and a fix commit push prompts and
	// raw agent output (which can carry secrets past the redactor). Recheck here,
	// after the checkout that could have introduced the symlink and before any
	// artifact is written.
	if err := o.checkLogsNotSymlinked(); err != nil {
		return err
	}

	o.openJournal()

	// The pre-Prepare clean check above protects the checkout itself, but
	// Prepare then switches branches (pr mode runs gh pr checkout) without
	// rechecking. A checkout can leave the new branch dirty -- previously
	// ignored files surface, checkout hooks run -- and those pre-existing
	// changes would then be attributed to the coder and swept into a round
	// commit by git add -A. Re-check for fix runs so a dirty checked-out tree
	// aborts before any coder edits. PR review-only re-checks too, so the
	// checkout's leftovers never reach the collected diff (see requireCleanTree).
	if requireCleanTree {
		if err := o.ensureCleanTree(ctx, "working tree is dirty after preparing the target (e.g. gh pr checkout); commit, stash, or clean the checked-out branch so the review sees only the PR's changes (and round commits contain only the coder's fixes)"); err != nil {
			return err
		}
	}

	// Capture the verification baseline on the pristine tree: after the clean-tree
	// checks (so nothing of the operator's is in it) and before any coder edits (so
	// a pre-existing failure is attributable to the project, not to this run).
	o.captureVerifyBaseline(ctx)

	// WHICH commit all of the above is about, pinned before any of it is read, and
	// which repository that commit is in -- a commit alone does not name a destination.
	o.recordReviewedHead(ctx, sum)
	o.recordReviewedRepo(ctx, sum)

	// What the forge already knows about this head, read once before the panel
	// runs so the reviewers' own context and the verdict see the same answer.
	o.readForgeChecks(ctx, sum.ReviewedHead)
	o.readForgeThreads(ctx)
	// Decide what those conversations commission, before any of them has been read
	// three times by three coder sessions and answered by none. Optional: without
	// roles.triage the threads stay context, which is what every run before this did.
	o.triageConversations(ctx)

	// Where the run's commits begin, for a per_run squash at the end. Captured on the
	// same pristine tree as the verification baseline: everything after this point is
	// the run's own work and nothing of the operator's.
	runBase, err := o.resolveRunBase(ctx)
	if err != nil {
		return err
	}

	cleanStreak := 0
	for round := 1; round <= o.cfg.Loop.MaxIterations; round++ {
		done, err := o.runRound(ctx, round, sum, &cleanStreak)
		if err != nil {
			return err
		}
		if done {
			return o.finishRun(ctx, sum, runBase)
		}
	}
	sum.Termination = model.TermMaxIterations
	return o.finishRun(ctx, sum, runBase)
}

// resolveRunBase reports the commit the run starts from, for the per_run squash
// in squashRun -- or "" for a run that cannot commit at all.
//
// Only a run that commits has a base worth resolving. A review-only run never
// does, so squashRun skips itself for one rather than reading the unresolved "" as
// an unborn branch -- and asking for HEAD would break the
// one target that has no HEAD to give: a directory that is not a repository. That
// target is supported on purpose (Collect falls back to a filesystem walk), which
// is why every git probe run() makes before this point -- IsGitRepo,
// WorktreeOutOfScope, guardUntrustedGitConfig -- fails soft on "not a repository"
// rather than aborting. HeadSHA rightly does not: outside a repository git exits
// 128, and reading that as an unborn branch would hand SquashSince an empty base
// and rewrite history as a root commit. So the call is skipped instead. Fix runs
// still resolve it, and run() has already refused them unless the target is a
// repository.
func (o *Orchestrator) resolveRunBase(ctx context.Context) (string, error) {
	if o.cfg.Loop.ReviewOnly {
		return "", nil
	}
	return o.collector.HeadSHA(ctx)
}

// recordReviewedHead pins the commit this run reviews, for the posting paths.
//
// In pr mode the checked-out HEAD is the pull request's head: `gh pr checkout` put
// it there, and the clean-tree check just proved nothing else has touched the tree.
// Recording it is what lets a post -- now, or later through -post-run -- refuse when
// the pull request has moved, so a review of one commit can never become an approval
// of another. Only pr mode has a head to bind to; a directory run posts nothing.
//
// A failure is a warning, not a run failure: the review is still worth producing,
// and the posting path fails closed on the missing SHA rather than publishing an
// unbound verdict.
func (o *Orchestrator) recordReviewedHead(ctx context.Context, sum *model.RunSummary) {
	if o.cfg.Target.Mode != config.ModePR {
		return
	}
	head, err := o.collector.HeadSHA(ctx)
	if err != nil {
		o.logf("WARNING: could not record which commit is under review (%v); publishing this review will refuse rather than post it against a commit that may have moved", err)
		return
	}
	sum.ReviewedHead = head
}

// recordReviewedRepo pins WHICH repository this run reviews, alongside the commit.
//
// The head says what the review is about; it does not say where it belongs. All a
// later -post-run has to go on otherwise is sum.Path and sum.PR, and a path is not
// an identity: the checkout there can be repointed at another repository on the same
// forge, or the directory reused for one, and the replay would then submit to pull
// request PR of THAT repository -- with the head check satisfied by anyone who opens
// a request proposing the reviewed commit, which is public. Recording the repository
// gh resolved for this checkout is what lets the replay refuse a destination the
// panel never read. See forge.RepoID.
//
// A failure is a warning rather than a run failure, exactly like the head's: the
// review is still worth producing, and the posting path fails closed on the missing
// identity rather than publishing to a repository it cannot vouch for.
func (o *Orchestrator) recordReviewedRepo(ctx context.Context, sum *model.RunSummary) {
	if o.cfg.Target.Mode != config.ModePR {
		return
	}
	repo := forge.RepoID(ctx, o.cfg.Target.Path)
	if repo == "" {
		o.logf("WARNING: could not record which repository is under review; publishing this review later with -post-run will refuse rather than submit it to whatever repository that checkout points at by then")
		return
	}
	sum.ReviewedRepo = repo
}

// finishRun runs the closing phase and then applies a per_run squash over
// everything the run committed, loop rounds and closing passes together.
//
// The squash goes last for the same reason the closing phase runs at all: it is the
// point where the run is finally done editing. Squashing earlier would collapse
// commits the closing passes then build on.
//
// A closing-phase interruption is the one way an unvouched-for tree reaches this far:
// a failure returns the error above, but runFinalPhase deliberately reports a
// cancellation as nil (an interruption is a termination, not a run failure), so the
// skip has to be made here. It is checked explicitly rather than left to the first
// git call in squashRun failing on the canceled context: that is an incidental guard
// in another package, and a squashRun that resolved the head some other way -- from a
// cache, or on a fresh context as the reconcile paths do -- would silently start
// rewriting history over exactly the tree squashRun says must not be rewritten.
func (o *Orchestrator) finishRun(ctx context.Context, sum *model.RunSummary, runBase string) error {
	if err := o.runFinalPhase(ctx, sum, runBase); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return nil //nolint:nilerr // interruption is a normal termination, not a run failure
	}
	return o.squashRun(ctx, sum, runBase)
}

// squashRun collapses the whole run into one commit for commit_policy: per_run.
//
// It runs only on a normal termination, which is all that reaches it: a failure
// returns before it and finishRun skips it on an interruption, rightly so --
// rewriting history over a tree nobody vouched for would destroy the per-fix commits
// an operator needs in order to see how far the run actually got.
func (o *Orchestrator) squashRun(ctx context.Context, sum *model.RunSummary, runBase string) error {
	if o.cfg.Loop.CommitPolicy != config.CommitPerRun {
		return nil
	}
	// A review-only run has nothing to squash, and must not reach SquashSince: it is
	// the one case where runBase is "" for a reason other than an unborn branch, and
	// resolveRunBase skipped resolving it precisely because such a run never commits.
	// The head != runBase guard below cannot tell the two apart -- a run that STARTED
	// unborn and then committed legitimately squashes onto the empty base as a root
	// commit -- so an unguarded review-only run would rewrite the branch onto a root
	// commit built from the current index, leaving the repository's whole history
	// reachable only through the reflog. From a run the operator asked to be
	// read-only. `-review-only` over a config that sets per_run is enough to get
	// here, so the skip belongs on this side rather than in config.Validate.
	if o.cfg.Loop.ReviewOnly {
		return nil
	}
	head, err := o.collector.HeadSHA(ctx)
	if err != nil {
		return err
	}
	if head == runBase {
		return nil // nothing was committed
	}
	agg := &model.RoundRecord{Round: len(sum.Rounds)}
	var salvaged []*model.RoundRecord
	for i := range sum.Rounds {
		agg.Fixed += sum.Rounds[i].Fixed
		agg.Rejected += sum.Rounds[i].Rejected
		agg.Findings = append(agg.Findings, sum.Rounds[i].Findings...)
		// A round that salvaged put edits in this tree that no verdict covers, and the
		// Fixed/Rejected sections squashTo renders cannot say so.
		if sum.Rounds[i].CoderError != "" {
			salvaged = append(salvaged, &sum.Rounds[i])
		}
	}
	var sha string
	if len(salvaged) > 0 {
		sha, err = o.squashSalvagedRun(ctx, agg, salvaged, runBase)
	} else {
		sha, err = o.squashTo(ctx, agg, runBase, agg.Round)
	}
	if err != nil {
		return err
	}
	// The per-fix commits this replaced are gone, so point the round that produced the
	// run's last commit at what now holds every round's work -- otherwise the summary
	// cites SHAs no longer in the history. CommitSHA has to be cleared alongside
	// Commits on every other round: the scoreboard falls back to CommitSHA when
	// Commits is empty (logstore/runtable.go), so a leftover per-fix SHA there would
	// be counted as a commit of its own and printed under "Committed:", both of them
	// naming an object git can no longer resolve. Re-pointing the LAST round instead
	// would hand the squash to a clean round that committed nothing.
	last := -1
	for i := range sum.Rounds {
		if len(sum.Rounds[i].Commits) > 0 || sum.Rounds[i].CommitSHA != "" {
			last = i
		}
		sum.Rounds[i].Commits = nil
		sum.Rounds[i].CommitSHA = ""
	}
	// head != runBase with no round claiming a commit should not happen -- every
	// commit path records one -- but the squash exists either way, and dropping it
	// from the summary entirely would report a run that committed as one that did not.
	if last < 0 {
		last = len(sum.Rounds) - 1
	}
	if last >= 0 {
		sum.Rounds[last].CommitSHA = sha
		sum.Rounds[last].Commits = []string{sha}
	}
	if len(salvaged) > 0 {
		o.logf("run: squashed %d round(s) of fixes, including partial salvage work from %d round(s), into %s", agg.Round, len(salvaged), shortSHA(sha))
	} else {
		o.logf("run: squashed %d round(s) of fixes into %s", agg.Round, shortSHA(sha))
	}
	return nil
}

// runFixSessions hands each of the round's active issues to its OWN coder session,
// verifying and committing each one, and reports how many produced a commit.
//
// One issue per session is what makes a per-fix commit honest, and it is the shape
// under every commit policy: the policy only regroups the commits afterwards. A
// session handed eight issues edits files for all eight at once and reports verdicts
// with no file attribution (model.FixResult carries id, verdict, detail -- nothing
// else), so those edits cannot be split apart after the fact. Working one at a time
// also means the verify gate names the FIX that broke the build rather than the
// round, and each fix is revertable on its own.
//
// It is also why loop.max_findings_per_round now defaults to unlimited: that cap was
// sized to one session's 30m timeout, and no session receives more than one issue.
func (o *Orchestrator) runFixSessions(ctx context.Context, rec *model.RoundRecord, history []model.RoundRecord, allowSalvage bool) (salvaged bool, committed int, err error) {
	// Snapshot the working set: setIssueVerdict mutates rec.Issues as each session
	// reports, and activeIssues would then shrink under the loop.
	batch := activeIssues(rec)
	// Where the tree stood when this round's reviewers read it. Each session below
	// commits, so later sessions are handed findings written against a tree that has
	// since moved -- see staleFiles.
	reviewedAt, err := o.collector.HeadSHA(ctx)
	if err != nil {
		return false, committed, err
	}
	for _, it := range batch {
		if ctx.Err() != nil {
			// The operator asked to stop between sessions, so do not spend another one.
			// Nothing is half-done at this point -- every fix so far is verified and
			// committed and the tree is clean -- so there is nothing to stash, and Run
			// softens a bare cancellation into a clean interruption.
			return false, committed, ctx.Err()
		}
		fixedBefore := rec.Fixed
		salvaged, replies, err := o.fix(ctx, rec, history, allowSalvage, []model.Issue{it}, o.staleFiles(ctx, reviewedAt, it))
		if err != nil {
			return false, committed, o.withdrawUncommittedFix(rec, it.ID, err)
		}
		if salvaged {
			// The coder died and its partial work was committed as a salvage round;
			// verdicts for the rest are unknown, so stop and let the next round
			// re-review everything.
			return true, committed, nil
		}
		// A cancellation between the coder finishing and this commit must not leave a
		// committed fix (or a dirty tree) sitting on top of a stop request. Recheck
		// immediately before touching the tree and route the cancellation through the
		// same stash-and-interrupt reconciliation, on a fresh context.
		if ctx.Err() != nil {
			return false, committed, o.withdrawUncommittedFix(rec, it.ID, o.reconcileInterrupt(rec.Round, ctx.Err())) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
		}
		// Reconcile the verdict against the TREE before trusting it. The coder's
		// report is a model's claim about its work, and one issue per session makes
		// the claim checkable: whether these edits exist is not a question about the
		// round as a whole any more, it is a question about this fix.
		clean, err := o.collector.GitClean(ctx, o.gitExclude...)
		if err != nil {
			if ctx.Err() != nil {
				return false, committed, o.withdrawUncommittedFix(rec, it.ID, o.reconcileInterrupt(rec.Round, err)) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
			}
			return false, committed, o.withdrawUncommittedFix(rec, it.ID, err)
		}
		if rec.Fixed == fixedBefore {
			// Rejected. The verdict is recorded; edits, if any, belong to no verdict
			// and must not reach a commit -- but they are this session's alone (the
			// tree was clean when it started), so they are stashed and the loop moves
			// to the next issue rather than ending the run with the remaining issues
			// unheard. Only a stash failure aborts.
			//
			// The replies go through the same gate as a committed session's, which
			// drops them and logs the skip: this is the case the gate was written for,
			// and it is recorded before the stash so an abort there does not swallow it.
			o.answerConversations(ctx, it, false, replies)
			if !clean {
				if err := o.reconcileRejectedSession(ctx, rec, it); err != nil {
					return false, committed, err
				}
			}
			o.endPhase("FIX %s  rejected by the coder", it.ID)
			continue
		}
		if clean {
			return false, committed, o.withdrawUncommittedFix(rec, it.ID,
				fmt.Errorf("round %d: coder reported a fix for %s but left the working tree unchanged", rec.Round, it.ID))
		}
		did, err := o.verifyAndCommitFix(ctx, rec, it)
		if err != nil {
			o.endPhase("FIX %s  failed: %v", it.ID, err)
			return false, committed, o.withdrawUncommittedFix(rec, it.ID, err)
		}
		if did {
			committed++
		}
		o.answerConversations(ctx, it, did, replies)
		o.closeFix(it.ID, did)
	}
	return false, committed, nil
}

// closeFix ends a fix block with what became of the issue. Its own function so
// the outcome is stated in one place -- and so runFixSessions, which is already at
// the complexity limit, does not grow a branch for a log line.
func (o *Orchestrator) closeFix(issueID string, committed bool) {
	if committed {
		o.endPhase("FIX %s  committed", issueID)
		return
	}
	o.endPhase("FIX %s  not committed; the finding stays open", issueID)
}

// answerConversations posts this session's replies, but only once its fix is
// COMMITTED.
//
// Until that point the work can still be withdrawn: the gate can fail, the
// correction attempt can revert it, the edits can end up stashed. Posting on the
// coder's say-so alone put a claim on somebody's pull request, under the
// operator's identity, that nothing had verified.
//
// A rejected issue's replies are dropped rather than posted. An answer explaining
// why nothing was changed would often be welcome, but it cannot be told apart here
// from one claiming work that did not land, and over-posting is the failure being
// fixed. The skip is logged so it is a decision rather than a disappearance.
func (o *Orchestrator) answerConversations(ctx context.Context, it model.Issue, committed bool, replies []model.FixReply) {
	if len(replies) == 0 {
		return
	}
	if !committed {
		o.logf("%d conversation repl(y|ies) for %s were not posted: the fix did not commit", len(replies), it.ID)
		return
	}
	// Every conversation linked to this issue, not just the primary one: merging
	// records a second commissioning comment on Issue.Also and the coder is told to
	// answer all of them, so an issue's session legitimately owes replies to each.
	own := make(map[string]bool, len(it.Also)+1)
	for _, c := range it.Conversations() {
		own[c.Thread] = true
	}
	o.postReplies(ctx, own, replies)
}

// staleFiles reports which of the issue's files have been committed to since
// reviewedAt -- the tree the round's reviewers actually read. Every fix session
// commits, so the Nth session of a round opens a finding written against a tree
// that N-1 commits have since changed.
//
// Best-effort by design: it only adds a caution to the prompt, so a git failure
// here must not fail the round. On error the session simply runs without the
// warning, exactly as it did before this existed.
func (o *Orchestrator) staleFiles(ctx context.Context, reviewedAt string, it model.Issue) []string {
	if reviewedAt == "" {
		return nil
	}
	changed, err := o.collector.ChangedSince(ctx, reviewedAt)
	if err != nil || len(changed) == 0 {
		return nil
	}
	// Every file the issue names, not just the headline one: corroborating
	// observations from other reviewers routinely point at a different site of the
	// same problem, and any of them moving is worth the same caution.
	seen := map[string]bool{}
	out := make([]string, 0, len(it.Observations)+1)
	for _, f := range append([]model.Finding{{File: it.File}}, it.Observations...) {
		if f.File == "" || seen[f.File] || !changed[f.File] {
			continue
		}
		seen[f.File] = true
		out = append(out, f.File)
	}
	return out
}

// flattenField collapses agent-authored text onto a single line before it lands
// in a commit message. `git commit -m` takes its arguments literally, so a value
// carrying newlines forges extra message lines -- and a line shaped like
// `Signed-off-by: Someone <s@org>` becomes a real git trailer, attributing the
// commit to people who never made it. RedactSecrets does not stop that: a trailer
// carries no credential keyword. Reviewer-authored category/title/file and
// coder-authored verdict detail are all reachable by a prompt injection on a run
// over untrusted content, and the trust gate authorizes file edits, not commit
// metadata.
//
// Flattening alone is not enough for a value that ends up ALONE on its own line:
// one line is exactly the shape of a trailer, so the collapsed text can still
// read as `Signed-off-by: ...`. Every such line is written as a `- ` bullet (see
// writeVerdictSection and verifyAndCommitFix) -- git's trailer parser rejects a
// line whose token carries a space, so a bullet can never become a trailer.
//
// The bullet stops trailers and nothing else. A forge's closing keywords (`Closes
// #42`) are matched ANYWHERE in a message, at no particular line shape, so the
// commit -- the one artifact meant to be pushed -- would close third-party issues
// chosen by an injection. Line shape cannot answer that; the reference can, so
// `#` or `GH-` followed by a digit is rewritten with a space in it. That breaks
// the `KEYWORD #N` / `KEYWORD GH-N` grammar, and `owner/repo#42` with it, while
// still recording readably the number the agent named. The same grammar accepts a
// full issue/pull URL, and the repository slug that form needs is public -- the
// injected text lives in the very repository under review -- so that spelling is
// broken the same way. Only the `/issues/N`-shaped tail of a URL is touched, so a
// legitimate link in a commit body survives unless it is itself a closable
// reference.
//
// Nor does flattening defang the text: ESC and the bidi overrides are not
// unicode.IsSpace, so strings.Fields passes them straight through, and git stores
// a message verbatim. A commit is the most-read artifact fixpoint produces -- it
// is pushed, and every later `git log`/`git show` renders it in someone's terminal
// -- so it gets the same escaping the prompts, logs and scoreboard get: OSC 52
// must not reach a reader's clipboard, and a bidi override must not make the
// finding read as something the reviewer never wrote.
func flattenField(s string) string {
	// forge.BreakReferences is shared with the review body: a commit message and a
	// review comment are both read by a forge's closing-keyword grammar, so the rule
	// has one definition rather than two that can drift.
	return forge.BreakReferences(agent.EscapeTerminal(strings.Join(strings.Fields(s), " ")))
}

// verifyAndCommitFix puts ONE fix through the gate and commits it. Same contract as
// a round commit -- nothing lands unverified -- with the granularity moved down to
// the fix, so a failing check names the change that caused it.
func (o *Orchestrator) verifyAndCommitFix(ctx context.Context, rec *model.RoundRecord, it model.Issue) (committed bool, err error) {
	if blocked, err := o.verifyRound(ctx, rec, it); err != nil {
		return false, err
	} else if blocked {
		return false, o.rejectUnverifiedRound(ctx, rec)
	}
	// A correction attempt can revert the edits entirely, leaving a clean tree after
	// verification passed. The verdict is already recorded, so put the issue back
	// rather than let the summary claim a fix no commit contains.
	//
	// A failed check is not a clean tree: it means fixpoint does not KNOW whether the
	// correction reverted the fix, and committing on that ignorance is exactly what
	// this check exists to prevent. Surface it -- the caller withdraws the verdict.
	clean, cerr := o.collector.GitClean(ctx, o.gitExclude...)
	if cerr != nil {
		if ctx.Err() != nil {
			return false, o.reconcileInterrupt(rec.Round, cerr) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
		}
		return false, cerr
	}
	if clean {
		o.reopenFixedIssue(rec, it.ID)
		o.logf("round %d: %s left nothing to commit -- the verification correction reverted it; the finding is reopened", rec.Round, it.ID)
		return false, nil
	}
	// The subject embeds the reviewer-authored title, so it gets the same two
	// treatments the body does: flattenField first (see there), then redaction.
	header := strings.NewReplacer(
		"{round}", strconv.Itoa(rec.Round),
		"{fixed}", "1",
		"{rejected}", "0",
		"{issue}", it.ID,
		"{title}", flattenField(it.Title),
	).Replace(o.fixCommitMessage())
	var body strings.Builder
	fmt.Fprintf(&body, "%s (%s, %s) %s\n", it.ID, flattenField(it.Category), flattenField(it.Severity), flattenField(it.Loc()))
	// Who asked for this change, when it was not the panel. A commissioned fix comes
	// from a comment somebody left on the pull request, and whoever reads this commit
	// later -- in a bisect, in a blame, in a release note -- should be able to see
	// that without reconstructing it from a conversation that may be resolved by then.
	// Stated as a bullet rather than a `token: value` line for the same reason the
	// detail below is: a trailing one of those IS a git trailer.
	if it.Origin.FromConversation() {
		who := flattenField(it.Origin.Author)
		if who == "" {
			who = "an unnamed commenter"
		}
		if it.Origin.External {
			fmt.Fprintf(&body, "\n- requested in conversation %s by %s, who is not the account this run posts under\n",
				flattenField(it.Origin.Thread), who)
		} else {
			fmt.Fprintf(&body, "\n- requested in conversation %s by %s\n", flattenField(it.Origin.Thread), who)
		}
	}
	// The detail is the body's LAST line, and a lone `token: value` line at the end
	// of a message IS a git trailer -- so it goes out as a bullet, the same shape
	// writeVerdictSection uses, which git's trailer parser can never accept.
	if d := flattenField(issueVerdictDetail(rec, it.ID)); d != "" {
		body.WriteString("\n- " + d + "\n")
	}
	// Redact the subject as well as the body: it is agent-authored text bound for
	// a pushed commit, exactly what squashTo redacts for the same reason.
	sha, err := o.collector.Commit(ctx, agent.RedactSecrets(header), agent.RedactSecrets(body.String()), o.gitExclude...)
	if err != nil {
		if ctx.Err() != nil {
			return false, o.reconcileInterrupt(rec.Round, err) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
		}
		return false, o.reconcileFailedCommit(ctx, rec.Round, err)
	}
	rec.CommitSHA = sha // the round's last commit; a squash replaces it
	rec.Commits = append(rec.Commits, sha)
	o.logf("round %d: fixed %s, committed %s", rec.Round, it.ID, shortSHA(sha))
	o.journal(model.EvRoundCommitted, rec.Round, model.JournalRoundCommitted{SHA: sha, Fixed: 1})
	return true, nil
}

// fixCommitMessage is the header for a per-fix commit. The configured
// commit_message describes a ROUND, so under per_fix -- where each commit is one
// issue -- a message reading "3 fixed" on a single-fix commit would be wrong. A
// caller who wants their own wording puts {issue}/{title} in commit_message.
func (o *Orchestrator) fixCommitMessage() string {
	m := o.cfg.Loop.CommitMessage
	if strings.Contains(m, "{issue}") || strings.Contains(m, "{title}") {
		return m
	}
	return "fixpoint: {issue} — {title}"
}

// squashRound regroups the round's per-fix commits per loop.commit_policy. per_fix
// keeps them; per_round collapses the round into the single commit fixpoint has
// always produced; per_run leaves them for the end of the run.
//
// The squash is a pure regrouping -- it re-commits the index and leaves the
// worktree untouched -- so the content is byte-for-byte what the per-fix commits
// already put through the gate one at a time.
func (o *Orchestrator) squashRound(ctx context.Context, rec *model.RoundRecord, base string, committed int) error {
	if committed == 0 || o.cfg.Loop.CommitPolicy != config.CommitPerRound {
		return nil
	}
	sha, err := o.squashTo(ctx, rec, base, rec.Round)
	if err != nil {
		return err
	}
	rec.CommitSHA = sha
	rec.Commits = []string{sha} // the per-fix commits it replaced no longer exist
	o.logf("round %d: squashed %d fix commit(s) into %s", rec.Round, committed, shortSHA(sha))
	return nil
}

// squashSalvagedRound applies commit_policy: per_round to a round that ended in a
// salvage. runRound's salvaged branch returns before squashRound -- the remaining
// verdicts are unknown, so it skips finalize -- and a round whose earlier issues
// were each committed before a later coder session failed would otherwise leave
// those per-fix commits and the salvage commit standing side by side under a policy
// that asks for one commit per round.
//
// The squashed message keeps the salvage header and the coder's error alongside the
// verdicts that did land, because the resulting commit holds both: verified per-fix
// work AND partial work nobody reported a verdict for.
func (o *Orchestrator) squashSalvagedRound(ctx context.Context, rec *model.RoundRecord, base string) error {
	// One commit since base is the salvage commit alone -- it already carries exactly
	// this message, so there is nothing to regroup.
	if len(rec.Commits) < 2 || o.cfg.Loop.CommitPolicy != config.CommitPerRound {
		return nil
	}
	n := len(rec.Commits)
	var body strings.Builder
	writeVerdictSection(&body, "Fixed", rec.Findings, model.VerdictFixed)
	if rec.Rejected > 0 {
		body.WriteString("\n")
		writeVerdictSection(&body, "Rejected", rec.Findings, model.VerdictRejected)
	}
	// What is left undecided is what carries no verdict: the issue the coder died on,
	// plus any it never reached. Not activeIssues -- that still includes the issues
	// earlier sessions of this round fixed, which the Fixed section above already
	// names.
	body.WriteString("\n" + salvageBody(rec.CoderError, pendingIssues(rec)))
	sha, err := o.collector.SquashSince(ctx, base, salvageHeader(rec.Round), agent.RedactSecrets(body.String()), o.gitExclude...)
	if err != nil {
		return err
	}
	rec.CommitSHA = sha
	rec.Commits = []string{sha} // the commits it replaced no longer exist
	o.logf("round %d: squashed %d commit(s), including the partial salvage commit, into %s", rec.Round, n, shortSHA(sha))
	return nil
}

// squashSalvagedRun is squashTo for a per_run squash whose tree contains work a
// failed coder left behind, and it exists for the same reason squashSalvagedRound
// does: the resulting commit holds both verified per-fix work AND partial work no
// verdict accounts for, and the plain round header and body say only the first.
// per_run is the shape that collapses the salvage commit into the run's ONE commit,
// so it is the shape where losing that fact loses it for good.
//
// Each salvaged round gets its own section, named by round: a run can salvage more
// than once, and the coder error and the issues it left undecided differ per round.
// They are reported as of that round -- a later round may well have re-reviewed and
// fixed them, which the Fixed section above says.
func (o *Orchestrator) squashSalvagedRun(ctx context.Context, agg *model.RoundRecord, salvaged []*model.RoundRecord, base string) (string, error) {
	var body strings.Builder
	writeVerdictSection(&body, "Fixed", agg.Findings, model.VerdictFixed)
	if agg.Rejected > 0 {
		body.WriteString("\n")
		writeVerdictSection(&body, "Rejected", agg.Findings, model.VerdictRejected)
	}
	for _, rec := range salvaged {
		fmt.Fprintf(&body, "\nRound %d: %s", rec.Round, salvageBody(rec.CoderError, pendingIssues(rec)))
	}
	// Redacted for the same reason squashTo redacts: agent-authored text bound for a
	// commit that gets pushed.
	return o.collector.SquashSince(ctx, base, salvageHeader(agg.Round), agent.RedactSecrets(body.String()), o.gitExclude...)
}

// roundCommitMessage is the header for a squashed commit. commit_message defaults
// to a per-FIX template now that per_fix is the default policy, and rendering that
// over a squash would produce a header naming one issue for a commit holding many
// (or an empty one). A caller who squashes writes a round-shaped commit_message.
func (o *Orchestrator) roundCommitMessage() string {
	m := o.cfg.Loop.CommitMessage
	if strings.Contains(m, "{issue}") || strings.Contains(m, "{title}") {
		return "fixpoint: round {round} ({fixed} fixed, {rejected} rejected)"
	}
	return m
}

// squashTo collapses everything after base into one commit describing rec.
func (o *Orchestrator) squashTo(ctx context.Context, rec *model.RoundRecord, base string, round int) (string, error) {
	header := strings.NewReplacer(
		"{round}", strconv.Itoa(round),
		"{fixed}", strconv.Itoa(rec.Fixed),
		"{rejected}", strconv.Itoa(rec.Rejected),
	).Replace(o.roundCommitMessage())
	var body strings.Builder
	writeVerdictSection(&body, "Fixed", rec.Findings, model.VerdictFixed)
	if rec.Rejected > 0 {
		body.WriteString("\n")
		writeVerdictSection(&body, "Rejected", rec.Findings, model.VerdictRejected)
	}
	// The body embeds reviewer-authored titles and coder-authored verdict details,
	// which may quote the very secret under review. Unlike the logs, the commit is
	// meant to be pushed and shared, so redact it too (the logstore path does the
	// same for its on-disk artifacts).
	// The same exclusions the per-fix commits used: Commit restores the excluded
	// paths' staged entries into the index after each one, so the squash has to keep
	// them out of its tree just as those commits did.
	return o.collector.SquashSince(ctx, base, header, agent.RedactSecrets(body.String()), o.gitExclude...)
}

// runFinalPhase runs the closing round for `final: true` lenses, after the loop has
// stopped changing the code. See config.ReviewLens.Final for why such a lens waits
// for the finished tree.
//
// It has two parts, because the two kinds of final lens want opposite schedules.
//
// The ACTIONABLE lenses repeat until nothing is left to fix, because
// loop.max_findings_per_round is a limit on ONE CODER SESSION, not a budget for a
// phase: it exists because ~17 issues blew the coder's 30m timeout and ~8 fit.
// Inside the loop that distinction does not matter, since the next round picks up
// whatever was deferred. Here there IS no next round, so a single capped pass would
// report the excess and then drop it -- the closing round would fix 8 of 26 coverage
// gaps and the run would read as complete. Processing more than one session's worth
// means more sessions, not a bigger session. Re-reviewing between passes is not
// waste either: pass 2 sees the tree as pass 1 left it, so it reports what is
// genuinely still missing rather than working from a list computed before the code
// changed -- which is also what makes the phase terminate on its own. The one thing
// pass 2 is NOT shown is the paths loop.final_skip_run_edits hides, recomputed per
// pass so that a pass never reviews the files its predecessor just wrote; see
// runFinalFixPasses' hideRunEdits call.
//
// The ADVISORY lenses then run exactly ONCE, last. They are reports, and a report
// wants to describe the code that actually shipped -- which is only known once the
// actionable half has stopped changing it. Running them per pass would emit one
// maintainability and one design report for every pass, which is precisely the
// per-round waste `final` exists to remove.
//
// The phase runs after every normal termination -- converged, all-rejected, and
// max-iterations alike -- because in all three the loop is done editing. It does
// NOT run after an error or an interruption: the tree is then in a state nobody
// vouched for, and the honest move is to stop rather than start new work on it.
//
// The loop's termination is preserved throughout. The closing phase is extra work
// on an already-decided run, not a new verdict on it, and letting it rewrite
// "converged" into "all-rejected" would report the loop's outcome as something it
// was not. A cancellation is the one thing that does change the run's termination
// -- to interrupted, with the loop's outcome moved to LoopTermination -- because
// that is a statement about the RUN rather than a verdict on the code: see
// markInterrupted.
func (o *Orchestrator) runFinalPhase(ctx context.Context, sum *model.RunSummary, runBase string) error {
	actionable, advisory := o.finalAssignments()
	// Nothing configured for this phase: the run is over exactly as the loop left it.
	if len(actionable)+len(advisory) == 0 || o.cfg.Loop.ReviewOnly {
		return nil
	}
	// A canceled context means the operator asked to stop, so the closing phase is
	// simply not started. Returning nil rather than the cancellation is deliberate:
	// the LOOP already finished, and skipping this work is not a run FAILURE. It is
	// still an interruption, though -- configured closing work was left undone -- so
	// the run is reported as one, with the loop's own outcome kept in
	// LoopTermination. Reporting the loop's "converged" instead would exit 0 on a
	// run the operator cut short.
	if ctx.Err() != nil {
		markInterrupted(sum)
		// An interruption is a termination, not a run failure -- see above.
		return nil
	}
	// Narrow what this phase is shown -- recomputed before every pass, since each
	// pass commits its own fixes and the next one must not be handed them -- then
	// restore full scope: the collector outlives the phase (squashRun and any later
	// reconcile use it), and a hidden set left behind would silently narrow them too.
	defer func() { _, _ = o.collector.HideRunEdits(nil, nil) }()
	if err := o.runFinalFixPasses(ctx, sum, runBase, actionable); err != nil {
		return err
	}
	// The report goes last, over the tree as it finally stands. Skipped on a
	// cancellation: the operator asked to stop, and a report is not worth a session
	// -- but the run is then an interruption, for the reason above.
	if len(advisory) > 0 {
		if ctx.Err() != nil {
			markInterrupted(sum)
			// As above: not a run failure.
			return nil
		}
		// The report describes the tree as it finally stands, so its hidden set is
		// computed against that tree too: everything the actionable passes just
		// committed is this run's own work, and a set computed before those passes ran
		// does not know about it.
		o.hideRunEdits(ctx, runBase)
		if _, err := o.runFinalPass(ctx, sum, advisory, "report"); err != nil {
			return err
		}
	}
	return nil
}

// hideRunEdits applies loop.final_skip_run_edits: the files this run's own commits
// wrote, matching those globs, are kept out of the closing round's material. See
// target.Collector.HideRunEdits for the measurement behind it.
//
// Best-effort, like staleFiles: it narrows a review for economy, so a git failure
// here means the closing round sees the whole tree -- the behavior every earlier
// version had -- rather than the run failing at its last step. What it must not do
// is stay quiet: a reviewer that is not shown a file reports no findings about it,
// which is indistinguishable from a clean bill of health, so both the hidden count
// and a failure to compute it are logged.
func (o *Orchestrator) hideRunEdits(ctx context.Context, runBase string) {
	globs := o.cfg.Loop.FinalSkipRunEdits
	if len(globs) == 0 || runBase == "" {
		return
	}
	// Drop the previous call's set before recomputing. This runs once per closing
	// pass, so on every path that does not install a fresh set -- a cancellation, a
	// git failure -- an earlier pass's narrowing would otherwise stay installed:
	// stale, since it predates the files the intervening pass committed, and the
	// exact opposite of the warning below, which promises the whole tree. Clearing
	// cannot fail with nil arguments.
	_, _ = o.collector.HideRunEdits(nil, nil)
	// A canceled context makes ChangedSince fail instantly, and warning about a
	// phase that is being abandoned anyway tells the operator nothing.
	if ctx.Err() != nil {
		return
	}
	changed, err := o.collector.ChangedSince(ctx, runBase)
	if err != nil {
		// Cancellation can also land between the check above and this git call, which
		// fails it the same way. The set is already cleared, and the same reasoning
		// applies: warn only when git itself is the reason, or the operator is told
		// about the review breadth of a closing round that never runs.
		if ctx.Err() != nil {
			return
		}
		o.logf("WARNING: could not list this run's own edits (%v); the closing round reviews the whole tree, including files it wrote (loop.final_skip_run_edits)", err)
		return
	}
	hidden, err := o.collector.HideRunEdits(changed, globs)
	if err != nil {
		o.logf("WARNING: loop.final_skip_run_edits: %v; the closing round reviews the whole tree, including files it wrote", err)
		return
	}
	if len(hidden) == 0 {
		return
	}
	o.logf("closing round: hiding %d file(s) this run wrote (loop.final_skip_run_edits): %s",
		len(hidden), strings.Join(hidden, ", "))
}

// runFinalFixPasses repeats the actionable half of the closing round until it has
// nothing left to fix, bounded by loop.max_final_passes -- the phase stops on its
// own as soon as a pass finds nothing or fixes nothing, so the bound only catches a
// lens that never runs out of things to say. See config.Loop.MaxFinalPasses for why
// that bound is its own knob (default 1) rather than the loop's max_iterations.
func (o *Orchestrator) runFinalFixPasses(ctx context.Context, sum *model.RunSummary, runBase string, actionable []model.Assignment) error {
	if len(actionable) == 0 {
		return nil
	}
	// A Loop built in code (tests, embedders) never passes through config's
	// defaulting step, and a zero cap there would skip the phase entirely rather
	// than run it the default number of times.
	maxPasses := o.cfg.Loop.MaxFinalPasses
	if maxPasses <= 0 {
		maxPasses = config.DefaultMaxFinalPasses
	}
	for pass := 1; pass <= maxPasses; pass++ {
		// Per pass, not once per phase: every pass commits its own fixes, and those
		// files are as much "this run's own edits" as the loop's are. A set computed
		// before pass 1 would show pass 2 the tests pass 1 just wrote -- the
		// review-the-test-that-was-just-asked-for loop this setting exists to break,
		// and the opposite of what max_final_passes documents.
		o.hideRunEdits(ctx, runBase)
		done, err := o.runFinalPass(ctx, sum, actionable, fmt.Sprintf("pass %d", pass))
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	o.warnFinalPhaseCapped(sum, maxPasses)
	return nil
}

// warnFinalPhaseCapped reports what the closing phase ran out of passes before
// finishing. Staying quiet about THAT is exactly the failure this phase exists to
// avoid: a run that fixed some of its coverage gaps and said nothing about the rest
// reads as complete.
//
// It is only reached when the LAST allowed pass returned done=false, so there is no
// clean exhaustion to keep quiet about -- a pass that found nothing, fixed nothing
// or hit a decided-only round ends the phase with done=true and never gets here.
// done=false leaves behind two INDEPENDENT kinds of loose end, and a pass can leave
// both at once (fix some issues, defer others), so each is detected and reported on
// its own:
//
//   - Issues the pass reported are still unresolved: open, or deferred by the cap
//     and never handed over. A REJECTED issue is neither -- it is decided, and
//     counting it as unfixed work would invent an obligation the run does not have.
//   - The pass committed fixes, which is a reason it asked for another pass: those
//     commits changed the tree, and the review that would have judged them is the
//     pass the cap refused. Silence here reads as a clean finish over a tree whose
//     final edits nobody looked at.
func (o *Orchestrator) warnFinalPhaseCapped(sum *model.RunSummary, passes int) {
	var last model.RoundRecord
	if n := len(sum.Rounds); n > 0 {
		last = sum.Rounds[n-1]
	}
	open, fixed := 0, 0
	for _, it := range last.Issues {
		switch it.StatusOrDefault() {
		case model.VerdictFixed:
			fixed++
		case model.VerdictRejected: // decided; not work left over
		default:
			open++
		}
	}
	// A fix in this phase is only recorded once verifyAndCommitFix committed it, so
	// a fixed issue is itself evidence the tree moved -- checked alongside the round's
	// commits because per_round squashes them into one SHA.
	changed := fixed > 0 || len(last.Commits) > 0 || last.CommitSHA != ""
	var parts []string
	if open > 0 {
		parts = append(parts, fmt.Sprintf("%d issue(s) still open or deferred: they are recorded in the summary and were NOT fixed", open))
	}
	if changed {
		parts = append(parts, "its last pass committed fixes that changed the tree and were NOT re-reviewed")
	}
	// Neither is unreachable through runFinalFixPasses, since a pass with nothing
	// unresolved and nothing committed ends the phase with done=true. Still said out
	// loud rather than returning silently: the cap fired, and that is the one thing
	// this warning exists to never omit.
	if len(parts) == 0 {
		parts = append(parts, "its last pass asked for another look at the tree and did not get one")
	}
	o.logf("WARNING: the closing round stopped after %d pass(es) (loop.max_final_passes); %s",
		passes, strings.Join(parts, "; and "))
}

// runFinalPass is one closing round: review the finished tree, hand the coder up to
// one session's worth, verify, commit. done=true when nothing changed, so another
// pass would ask the same question of the same code.
//
// label names this pass in the log ("pass 2", "report"). An all-advisory pass has no
// findings to fix by construction, so it returns done after the review and never
// reaches the coder.
func (o *Orchestrator) runFinalPass(ctx context.Context, sum *model.RunSummary, asgs []model.Assignment, label string) (done bool, err error) {
	round := len(sum.Rounds) + 1
	o.phase("CLOSING %s  round %d, %d reviewer(s) over the finished tree", label, round, len(asgs))

	material, err := o.collector.Collect(ctx)
	if err != nil {
		return false, err
	}
	rec := model.RoundRecord{Round: round, Assignments: asgs, Final: true}
	o.journal(model.EvRoundStarted, round, model.JournalRoundStarted{
		Assignments: journalAssignments(asgs),
		Final:       true,
	})
	o.review(ctx, &rec, material, sum.Rounds)
	sum.Rounds = append(sum.Rounds, rec)
	recP := &sum.Rounds[len(sum.Rounds)-1]
	recP.Issues = o.ledger.Absorb(round, recP.Findings)
	o.logLedgerConflicts()
	// Same reason as in runRound: an issue that arrives already rejected is never
	// touched by setIssueVerdict, so without this its observations keep an empty
	// verdict and the summary renders them UNRESOLVED while the issues block right
	// below says REJECTED.
	mirrorCarriedVerdicts(recP)
	o.journal(model.EvReviewFinished, round, model.JournalReviewFinished{
		Observations: len(recP.Findings),
		Advisory:     len(recP.Advisory),
		Errors:       recP.ReviewErrors,
	})
	o.journal(model.EvIssuesAggregated, round, model.JournalIssuesAggregated{
		Observations: len(recP.Findings),
		Issues:       len(recP.Issues),
		Corroborated: corroboratedCount(recP.Issues),
	})
	o.endPhase("CLOSING %s  %d finding(s), %d advisory, %d reviewer error(s)",
		label, len(recP.Findings), len(recP.Advisory), len(recP.ReviewErrors))
	// A failed closing reviewer is a failed closing pass, checked BEFORE an empty
	// finding set is read as a completed final review. Nothing follows to catch what
	// a dead reviewer never looked at, so "no findings" from a review that did not
	// finish is not the same statement as "no findings", and proceeding to fix the
	// findings the survivors did report would act on a knowingly partial last look.
	// Repeating the phase does not soften this: a later pass re-reviews the same
	// tree, so it would inherit the same blind spot rather than close it.
	//
	// An ADVISORY-only pass is the exception, because the premise above does not
	// hold for it: it hands nothing to the coder by construction (its output lands
	// in rec.Advisory and rec.Findings stays empty), so there is no partial list to
	// act on and nothing to abort. Failing here would rewrite a run whose loop
	// converged, and whose every fix was verified and committed, into an error over
	// one transient report lens -- contradicting ReviewLens.Advisory's contract that
	// advisory findings are excluded from the termination condition. The failures
	// are recorded on the round either way; here they are only warned about, since
	// the report that ships is incomplete and silence would read as a full one.
	if len(recP.ReviewErrors) > 0 {
		if !allAdvisory(asgs) {
			return false, roundReviewErr(recP)
		}
		o.logf("WARNING: closing %s: %d advisory reviewer(s) failed: %s -- that report is incomplete, but advisory lenses hand nothing to the coder, so the run's outcome is unchanged",
			label, len(recP.ReviewErrors), strings.Join(recP.ReviewErrors, "; "))
	}
	// The operator stopped us between the review and the fix: the phase is over, and
	// the findings this pass just reported are left unfixed, so the RUN is an
	// interruption (the loop's own outcome stays visible in LoopTermination). Ending
	// on the loop's "converged" here would exit 0 over closing work that was cut off.
	if ctx.Err() != nil {
		markInterrupted(sum)
		return true, nil //nolint:nilerr // an interruption is a termination, not a pass failure
	}
	// Nothing to fix: the phase is over and the loop's outcome stands unchanged.
	if len(recP.Findings) == 0 {
		return true, nil
	}

	// The cap still bounds each PASS, because the coder's timeout still bounds each
	// session. What it defers is picked up by the next pass, which is the whole
	// reason this phase repeats -- and deferral aging means a skipped issue outranks
	// fresh ones on the way round, so the tail cannot be starved here either.
	o.deferOverCap(recP)
	// Same guard as runRound: everything the closing reviewers reported was already
	// decided in an earlier round, so there is no work to hand over. Invoking the
	// coder on an empty list would spend a session asking about nothing -- and here
	// it is worse than a wasted session: with allowSalvage=false, any verdict it
	// volunteers for an id it was not given fails applyVerdicts, which would rewrite
	// a run whose loop genuinely converged into an error. Nothing for a later pass
	// either: those verdicts stand, so the phase is done.
	if len(activeIssues(recP)) == 0 {
		o.logf("closing %s: all %d issue(s) were already decided in an earlier round; nothing left to fix", label, len(recP.Issues))
		return true, nil
	}
	// allowSalvage=false: a failed coder's partial edits must not be committed here.
	// Salvage is a promise that the round's commit gets re-reviewed, and the passes
	// here only re-review what the LENS reports -- a half-finished edit it does not
	// mention would ride along unexamined. See discardFailedFix.
	base, err := o.collector.HeadSHA(ctx)
	if err != nil {
		return false, err
	}
	_, committed, err := o.runFixSessions(ctx, recP, sum.Rounds[:len(sum.Rounds)-1], false)
	if err != nil {
		return false, err
	}
	o.logf("closing %s: coder fixed %d, rejected %d", label, recP.Fixed, recP.Rejected)
	if committed == 0 {
		// Nothing was committed, so normally there is nothing for another pass to see:
		// it would re-review identical code and get an identical answer. Two exceptions,
		// the same two finalizeRound makes for the loop.
		//
		// First, nothing committed is not proof of a rejection: a verification
		// correction can revert a reported fix outright, and verifyAndCommitFix then
		// withdraws the verdict and reopens the issue without producing a commit. The
		// code did change under that issue -- the correction reverted the edits, so the
		// tree is back where it started but the defect is still there and undecided --
		// and ending the phase here would leave it unresolved with nobody having said
		// so.
		if open := unrejectedIssues(recP); open > 0 {
			o.logf("closing %s: nothing committed, but %d issue(s) are still open (a reverted fix, not a rejection); continuing", label, open)
			return false, nil
		}
		// Second, a capped pass, where the coder was only ever handed the active issues:
		// if it rejected all of them, the deferred remainder still never reached it, and
		// stopping here would drop exactly the work this repeating phase exists to
		// pick up -- silently, since warnFinalPhaseCapped only fires on the pass cap.
		// The re-review is not wasted on them either: deferral aging promotes what was
		// skipped, so the next pass hands it over first.
		if deferred := deferredFindings(recP); deferred > 0 {
			o.logf("closing %s: coder rejected every active issue, but %d finding(s) held back by the cap never reached it; continuing", label, deferred)
			return false, nil
		}
		return true, nil
	}
	if err := o.squashRound(ctx, recP, base, committed); err != nil {
		return false, err
	}
	// Something changed, so ask again: the next pass judges what is left against the
	// code as it now stands.
	return false, nil
}

// claimRepo takes the repository for a run that will write to it and returns the
// function that gives it back. Three preflight steps, in this order because each
// is cheaper than the next is meaningful without it:
//
//   - The repository ROOT is required. The clean check and staging are scoped to
//     target.path ("." pathspec), but a fix round's commit writes the whole
//     repository index and a PR review runs a repo-wide diff over a repo-wide
//     checkout. A subdirectory target would miss dirt elsewhere in the repo from
//     its target-relative clean check yet still fold those changes into the round
//     commit (fix) or the reviewed diff (pr).
//   - The whole repository is LOCKED for the run, before the clean check and held
//     past the last commit. Every mutating decision the loop makes is derived from
//     a snapshot of the worktree, and a second concurrent fixpoint invalidates all
//     of them silently -- see Collector.LockRepo.
//   - The tree must be CLEAN, so every round commit contains exactly the coder's
//     changes and nothing of the operator's. PR review needs it even review-only:
//     gh pr checkout can preserve unrelated tracked edits and untracked files,
//     which Collect would otherwise fold into the PR diff and misattribute the
//     resulting findings to the PR.
func (o *Orchestrator) claimRepo(ctx context.Context) (func(), error) {
	if atRoot, err := o.collector.AtRepoRoot(ctx); err != nil {
		return nil, err
	} else if !atRoot {
		return nil, fmt.Errorf("target.path %s is a subdirectory of its git repository; fix rounds and PR reviews operate on the whole repository (staging the entire index / diffing the whole checkout), so point target.path at the repository root (or use review_only for a non-pr run)", o.cfg.Target.Path)
	}
	release, err := o.collector.LockRepo(ctx)
	if err != nil {
		return nil, err
	}
	if err := o.ensureCleanTree(ctx, "working tree is dirty; commit or stash your changes first so the review sees only the intended target (and, in a fix run, round commits contain only the coder's fixes)"); err != nil {
		release() // never hold the repository past a failed preflight
		return nil, err
	}
	return release, nil
}

// PreflightGuards runs the two target-integrity gates that must hold before
// fixpoint points git at the target at all: the work-tree redirect check and the
// repo-supplied-git-config trust gate. It exists as its own entry point because
// -check and -check-live reach the target WITHOUT going through run() -- Scope
// runs the same
// commands collection does (`git diff` in git-diff mode, `git ls-files` in
// directory mode), so a repo-supplied filter.<name>.clean would run as a program,
// with fixpoint's inherited environment, from the one command documented as
// invoking no agent, and a core.worktree redirect would make the estimate
// describe a tree outside the target.
//
// Both guards are read-only (`git rev-parse --show-toplevel`, `git config
// --list`), take no lock and mutate nothing, so they are safe on that
// non-mutating path.
//
// It is idempotent (see preflightMu): Ping calls it too, so the run path, which
// guards before its first git command and then pings, probes and warns once.
//
// This is the variant for every entry point that will launch an agent with its
// working directory inside the target -- run() and Ping. That is what decides
// whether the repo-supplied-git-config gate applies at all, because the agents
// run their own `git status`/`git diff`/`git log` there, in every mode. Use
// PreflightGuardsNoAgent only where nothing but fixpoint's own git commands run.
func (o *Orchestrator) PreflightGuards(ctx context.Context) error {
	return o.preflightGuards(ctx, true)
}

// PreflightGuardsNoAgent is the --check variant of PreflightGuards: it applies
// the repo-supplied-git-config gate only to the modes whose OWN git commands
// touch the worktree, and downgrades the pr-mode activatable-filter refusal to a
// warning. --check runs Scope and exits without invoking an agent or a checkout,
// so a directory review-only estimate (`git ls-files`, no content normalized, no
// agent launched in the target) has no code-execution path to gate on, and pr
// mode reports its scope as unknown without running git at all -- and refusing
// either would refuse `fixpoint --check` on every git-lfs checkout the operator
// has not asserted trust in.
func (o *Orchestrator) PreflightGuardsNoAgent(ctx context.Context) error {
	return o.preflightGuards(ctx, false)
}

// recheckPreflightGuards discards the memoized preflight answer and runs the
// gates again. It is registered with the collector (Collector.OnCheckout) so it
// fires at the one point where the target changes under them: Prepare's
// `gh pr checkout`.
//
// The memo holds because the guards read a target that does not change mid-run --
// except across a branch switch. Git config is branch-conditional: an
// `includeIf "onbranch:<pattern>"` in .git/config (or in a file it includes)
// pulls in a whole config file only while that branch is checked out, so a
// crafted checkout can park a filter.<name>.clean, core.sshCommand, credential
// helper or core.worktree redirect in a file that is INERT on the branch the
// preflight inspects and ACTIVE on the PR branch `gh pr checkout` switches to.
// Every git command after the checkout -- the rest of Prepare (its base fetch runs
// a credential helper or ssh command), the post-Prepare `git status`, the
// `git diff` that collects the PR, and each agent's own git commands -- would then
// run that repo-controlled program with fixpoint's inherited environment, in a
// review-only run that passed no trust gate.
//
// It always asks the agent-invoking variant, whatever entry point led here: only
// run() prepares a target, and it goes on to launch agents with their working
// directory inside the checked-out tree.
func (o *Orchestrator) recheckPreflightGuards(ctx context.Context) error {
	o.preflightMu.Lock()
	o.preflightDone, o.preflightAgents, o.preflightErr = false, false, nil
	o.preflightMu.Unlock()
	return o.PreflightGuards(ctx)
}

func (o *Orchestrator) preflightGuards(ctx context.Context, agentsInTarget bool) error {
	o.preflightMu.Lock()
	defer o.preflightMu.Unlock()
	// A memoized no-agent answer does not satisfy an agent-invoking caller: it may
	// have skipped the config gate entirely. Re-probe in that direction only; the
	// reverse (strong answer, weak question) is already conclusive.
	if o.preflightDone && (o.preflightAgents || !agentsInTarget) {
		return o.preflightErr
	}
	o.preflightDone, o.preflightAgents = true, agentsInTarget
	if err := o.guardRedirectedWorktree(ctx); err != nil {
		o.preflightErr = err
		return o.preflightErr
	}
	o.preflightErr = o.guardUntrustedGitConfig(ctx, agentsInTarget)
	return o.preflightErr
}

// guardRedirectedWorktree refuses a target whose git work tree is not the target
// at all. A repo-local `core.worktree` (or GIT_WORK_TREE) points git's work tree
// somewhere else while .git stays put, so every git command fixpoint runs reads
// and writes that other directory: git-diff/pr collection lists the redirected
// tree's diff and files as the material under review, and a directory fix round's
// `git add`/`git commit` stage and commit from it. See Collector.WorktreeOutOfScope
// for why the effective root is what gets judged (a submodule checkout legitimately
// sets core.worktree) and why nothing else on the preflight catches this.
//
// This runs before EVERY mode, including a review-only directory run: that path
// still asks `git ls-files` for its scope whenever the target looks like a work
// tree, so a redirect at an ancestor of target.path would feed it paths from
// outside the target.
//
// It is NOT trust-gated. -trusted-target says "I trust what is in this checkout",
// which is a statement about content; it is not consent to review or commit to a
// different directory than the one named, and a redirect is never what the operator
// asked for.
func (o *Orchestrator) guardRedirectedWorktree(ctx context.Context) error {
	root, err := o.collector.WorktreeOutOfScope(ctx)
	if err != nil {
		return err
	}
	if root == "" {
		return nil
	}
	return fmt.Errorf("target %s is not the git work tree git would operate on: core.worktree (in .git/config, a file it includes, or .git/config.worktree) or GIT_WORK_TREE points the work tree at %s, so every git command fixpoint runs -- diff, ls-files, add, commit -- would read and write files outside the target. Remove the redirect, or set target.path to %s if that is the tree you meant to review", o.cfg.Target.Path, root, root)
}

// guardUntrustedGitConfig closes the code-execution path opened by the target's
// own repo-local .git/config: per-name content filters (filter.<n>.clean/
// smudge/process), diff drivers (diff.external, diff.<driver>.command/textconv),
// core.sshCommand, and credential helpers are programs git runs
// itself, and gitenv's -c overrides cannot neutralize them (their names
// are dynamic). They fire whenever a git command touches the worktree -- git-diff
// mode's `git diff` normalizes files through a clean filter even in a review_only
// run, and every fix round's `git add`/`git status` does the same -- so a target
// that ships its own .git/config (an extracted archive, a crafted checkout) can
// run a repo-controlled program with fixpoint's inherited secrets.
//
// The same key list also covers the settings that execute nothing but redirect
// git's network access -- the http.* section (curloptResolve, proxy, sslVerify, CA
// and pinned-key overrides, client certificates, cookie jars) and the transport
// rewrites next to it -- because the outcome is the same class of loss: a fetch
// that looks like github.com contacts an attacker's host, which the operator's own
// credential helper then authenticates to, and serves whatever objects it likes as
// the reviewed PR. See target.unsafeConfigKey.
//
// The gate applies wherever such a command runs against a target-controlled repo:
// git-diff mode always (its Collect runs `git diff`), directory fix rounds (their
// commits run `git add`/`git status`), and pr mode always -- Prepare's
// `gh pr checkout` writes the worktree, and git runs a configured smudge/process
// filter during checkout. A PR cannot edit .git/config, but it does not have to:
// it supplies the .gitattributes that SELECTS an already-configured filter or diff
// driver, the worktree script such a program may point at, and (for git-lfs) the
// .lfsconfig that redirects where the filter talks to. That half of the path is a
// program the repository ACTIVATES rather than defines, so it is invisible to a
// repo-scoped key list and is handled separately by guardActivatableConfig.
//
// agentsInTarget widens that set to EVERY mode, and is the common case: fixpoint's
// own git commands are not the only ones that run against the target. run() and
// Ping launch each CLI with its working directory inside target.path, and those
// CLIs run `git status`/`git diff`/`git log` while exploring -- which is enough to
// fire a repo-supplied filter.<name>.clean, core.sshCommand or credential helper,
// with fixpoint's inherited environment. Directory review-only is exactly the
// shipped review bundle (defaults.yaml sets mode: directory, review-code.yaml sets
// review_only: true), so without this it would be the one configuration that
// reaches an agent-in-target with no key inspected and no warning printed. Only
// --check, which invokes no agent, gets the narrower set.
//
// When the operator has asserted no trust we refuse; when trust IS asserted we
// still WARN, because trusted_target/allow_untrusted_fix is documented as
// accepting coder prompt-injection risk and an operator must also learn it accepts
// .git/config-driven code execution (a git-lfs repo they trust, or an enforced
// external sandbox, is the intended use).
func (o *Orchestrator) guardUntrustedGitConfig(ctx context.Context, agentsInTarget bool) error {
	touchesGit := agentsInTarget ||
		o.cfg.Target.Mode == config.ModeGitDiff ||
		o.cfg.Target.Mode == config.ModePR ||
		(!o.cfg.Loop.ReviewOnly && o.cfg.Target.Mode == config.ModeDirectory)
	if !touchesGit {
		return nil
	}
	// A failed probe must not silently skip this check: it decides whether a
	// checkout with execution-capable git config is refused.
	isRepo, err := o.collector.IsGitRepo(ctx)
	if err != nil {
		return err
	}
	if !isRepo {
		return nil
	}
	keys, err := o.collector.UnsafeConfig(ctx)
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		if !o.cfg.Loop.TrustedTarget && !o.cfg.Loop.AllowUntrustedFix {
			return fmt.Errorf("target %s has repo-supplied git config (.git/config, a file it includes, or .git/config.worktree) that would run repo-controlled programs, or redirect git's authenticated network access, in ways fixpoint cannot neutralize (%s); git normalizes worktree files through these during diff/add/status/checkout and honors them on every fetch, so this is a code-execution and credential-disclosure path with fixpoint's inherited environment. Review it under an external sandbox (container/VM), or pass -trusted-target if you trust this checkout", o.cfg.Target.Path, strings.Join(keys, ", "))
		}
		o.warnGuardOnce(fmt.Sprintf("WARNING: target %s has repo-supplied git config (.git/config, a file it includes, or .git/config.worktree) that runs repo-controlled programs during git diff/add/status/checkout, or redirects git's authenticated fetches (%s), which fixpoint cannot neutralize; -trusted-target/-allow-untrusted-fix accepts this code-execution and credential-disclosure path (with fixpoint's inherited environment) in addition to coder prompt-injection. Review untrusted checkouts (extracted archives, crafted .git) under an external sandbox.", o.cfg.Target.Path, strings.Join(keys, ", ")))
	}
	return o.guardActivatableConfig(ctx, agentsInTarget)
}

// guardActivatableConfig handles the content filters and diff drivers the target
// can ACTIVATE but does not DEFINE -- the second half of the .gitattributes path
// guardUntrustedGitConfig exists for. UnsafeConfig only sees definitions in the
// repository's own scopes, so on a host where the operator installed one globally
// (`git lfs install` writes filter.lfs.clean/smudge/process into ~/.gitconfig;
// `nbdime config-git --enable --global` writes a diff.<name>.command, and
// pdftotext/exiftool textconv drivers are a common habit) the gate above finds
// nothing while `gh pr checkout` goes on to apply the PR's `.gitattributes` and
// `.lfsconfig`, and git runs that program over PR-controlled content -- before any
// agent sandbox, in a review-only run that passes no trust gate. A textconv driver
// widens the trigger beyond fixpoint's own commands: it fires on the reviewer and
// coder CLIs' `git diff`/`git log -p`/`git show`/`git blame` inside the checkout,
// which the prompts explicitly invite, and Collect's --no-ext-diff --no-textconv
// only covers the command it is on.
//
// pr mode is where that is REFUSED rather than warned about, absent a trust
// assertion. It is the one mode whose content is untrusted by definition and, as
// of the checkout, not yet on disk: the PR supplies the .gitattributes that names
// the filter or driver, the worktree script such a program may run, and the
// .lfsconfig that redirects where a git-lfs filter talks -- all written by the
// `gh pr checkout` that would fire it, so there is nothing for the preflight to
// inspect and "warn and continue" amounts to running the operator's program over
// material nobody has seen yet, with fixpoint's inherited environment. review_only
// does not soften this; it is the default pr path and passes no other trust gate.
//
// The refusal is conditioned on agentsInTarget for the same reason the whole gate
// is: it is the checkout that fires the filter, and only the paths that launch an
// agent in the target perform one. --check (PreflightGuardsNoAgent) deliberately
// does NOT run Prepare -- Collector.Scope reports pr mode as "known only after
// `gh pr checkout`" and issues no git command against the worktree -- so refusing
// it would refuse the one command documented as the cheap, safe thing to run first
// over a checkout it never makes, on any host where the operator ran `git lfs
// install`. It warns instead, like every other non-checkout path.
//
// Everywhere else it stays a warning. There the checkout is one the operator
// pointed fixpoint at (its .gitattributes is already theirs to read), the filter
// or driver definition is their own configuration, and refusing would refuse every
// local run on a git-lfs host -- so the point is that the operator learns which of
// their programs the checkout can reach.
func (o *Orchestrator) guardActivatableConfig(ctx context.Context, agentsInTarget bool) error {
	keys, err := o.collector.ExternalActivatableConfig(ctx)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	if agentsInTarget && o.cfg.Target.Mode == config.ModePR && !o.cfg.Loop.TrustedTarget && !o.cfg.Loop.AllowUntrustedFix {
		return fmt.Errorf("mode pr: content filters and diff drivers configured outside target %s -- in your global or system git config (%s) -- are selected by REPOSITORY content, and in pr mode that content is the PR's: `gh pr checkout` writes the branch's .gitattributes (plus any worktree script the program runs, and the .lfsconfig that redirects where a git-lfs filter talks), so git would run your program over PR-authored file content with fixpoint's inherited environment, before any agent sandbox -- during the checkout itself, and on every later diff/add/status/log/blame, including the reviewer's own. fixpoint cannot neutralize them (their names are dynamic) and cannot inspect the attributes before the checkout that brings them. Review the PR under an external sandbox (container/VM), unset the filter or driver for this run, or pass -trusted-target/-allow-untrusted-fix to accept this path", o.cfg.Target.Path, strings.Join(keys, ", "))
	}
	o.warnGuardOnce(fmt.Sprintf("WARNING: content filters and diff drivers configured outside target %s -- in your global or system git config (%s) -- are selected by REPOSITORY content: a .gitattributes naming one (in the checkout, or in a PR branch `gh pr checkout` writes) makes git run it over repo-controlled file content during checkout/add/status/diff (a textconv driver also during log -p/show/blame, including the agents' own), and a repo-supplied .lfsconfig redirects where a git-lfs filter talks. fixpoint cannot neutralize them (their names are dynamic). Review untrusted checkouts under an external sandbox.", o.cfg.Target.Path, strings.Join(keys, ", ")))
	return nil
}

// warnGuardOnce logs msg unless the identical guard warning has already been
// logged this run; see warnedGuard for why the guards can produce one twice.
// Callers hold preflightMu (the guards' only entry point takes it), so the map
// needs no lock of its own.
func (o *Orchestrator) warnGuardOnce(msg string) {
	if o.warnedGuard[msg] {
		return
	}
	if o.warnedGuard == nil {
		o.warnedGuard = map[string]bool{}
	}
	o.warnedGuard[msg] = true
	o.logf("%s", msg)
}

// checkFixTrust enforces the fix-round trust gate. Fix rounds run the coder --
// an agent that edits files with permission checks disabled and is NOT confined
// to target.path -- and reviewer findings built from in-scope content are
// injected verbatim into its prompt. A prompt-injection payload hidden in any
// reviewed file can steer it into writing anywhere on the machine, so fix rounds
// require an explicit trust signal in EVERY mode (fail-closed by default), not
// just pr:
//   - pr: the PR author is untrusted by definition -> allow_untrusted_fix.
//   - directory/git-diff: the tree may still hold untrusted material (vendored
//     deps, a fetched base_ref, a cloned third-party project) -> trusted_target
//     (or allow_untrusted_fix).
func (o *Orchestrator) checkFixTrust() error {
	if o.cfg.Loop.ReviewOnly {
		return nil
	}
	switch o.cfg.Target.Mode {
	case config.ModePR:
		if !o.cfg.Loop.AllowUntrustedFix {
			return errors.New("mode pr reviews untrusted code, and fix rounds hand its content to a coder that edits files without permission checks; use review_only, or pass -allow-untrusted-fix if you trust the PR author")
		}
	default:
		if !o.cfg.Loop.TrustedTarget && !o.cfg.Loop.AllowUntrustedFix {
			return fmt.Errorf("fix rounds run a coder that edits files with permission checks disabled and is not confined to target.path, so reviewed content could steer it via prompt injection; pass -trusted-target to assert %q holds only code you trust, or use review_only", o.cfg.Target.Path)
		}
	}
	return nil
}

// ensureCleanTree fails with dirtyMsg when the working tree has changes outside
// the git-exclude set. The pre- and post-Prepare gates share it so the single
// GitClean+error pattern lives in one place.
func (o *Orchestrator) ensureCleanTree(ctx context.Context, dirtyMsg string) error {
	clean, err := o.collector.GitClean(ctx, o.gitExclude...)
	if err != nil {
		return err
	}
	if !clean {
		return errors.New(dirtyMsg)
	}
	return nil
}

// runRound executes one review->fix->commit iteration, appending its record to
// sum and threading the consecutive-clean-round streak through cleanStreak. It
// reports done=true when the loop should stop (setting sum.Termination), and an
// error when the round failed. run() is then just preflight plus this loop.
func (o *Orchestrator) runRound(ctx context.Context, round int, sum *model.RunSummary, cleanStreak *int) (done bool, err error) {
	if ctx.Err() != nil {
		sum.Termination = model.TermInterrupted
		return true, nil //nolint:nilerr // interruption is a normal termination, not a round error
	}
	o.rule("round %d/%d", round, o.cfg.Loop.MaxIterations)

	material, err := o.collector.Collect(ctx)
	if err != nil {
		return false, err
	}

	rec := model.RoundRecord{Round: round, Assignments: o.assignments(round)}
	// A fix run with no reviewers this round cannot verify anything: an empty
	// review reports no findings and no errors, which would otherwise read as a
	// clean round and let the loop converge on an unverified fix. Config
	// validation already requires a recurring lens for fix runs, so this is a
	// backstop against any future path that leaves a fix round unassigned.
	if !o.cfg.Loop.ReviewOnly && len(rec.Assignments) == 0 {
		return false, fmt.Errorf("round %d has no reviewer assignments to verify the fix; configure at least one recurring (non-once) reviewer lens", round)
	}
	// Under strategy: rotate the assignment differs every round, and it decides who
	// confirmed a clean result -- which is the whole reason rotation exists.
	o.journal(model.EvRoundStarted, round, model.JournalRoundStarted{
		Assignments: journalAssignments(rec.Assignments),
	})
	o.phase("REVIEW  %d reviewer(s), %d lens(es)", len(panelAgents(rec.Assignments)), len(rec.Assignments))
	o.review(ctx, &rec, material, sum.Rounds)
	// Billed for having RUN, not for what it decided. A pass that declined every
	// conversation, or that failed to parse, spent the same tokens as one that
	// commissioned work -- and the failed one is precisely the pass whose Failed
	// flag the summary must show. Role is set by stepStat, so a non-empty one means
	// the agent was invoked; cleared after so a later round does not bill it again.
	if o.triageStep.Role != "" {
		rec.Steps = append(rec.Steps, o.triageStep)
		o.triageStep = model.StepStat{}
	}
	// What the pull request's comments asked for joins the panel's own reports. From
	// here they are ordinary findings: the ledger groups them (so a comment and a
	// reviewer describing the same defect become one issue, still carrying the
	// conversation to answer), the cap orders them by severity, and each is fixed in
	// its own session behind the verify gate. Offered again every round until the
	// coder decides them -- see mergeCommissioned.
	o.mergeCommissioned(&rec)
	sum.Rounds = append(sum.Rounds, rec)
	recP := &sum.Rounds[len(sum.Rounds)-1]
	o.journal(model.EvReviewFinished, round, model.JournalReviewFinished{
		Observations: len(recP.Findings),
		Advisory:     len(recP.Advisory),
		Errors:       recP.ReviewErrors,
	})

	// Group observations into issues before anything counts them. Two reviewers
	// agreeing is corroboration, not two units of work: without this, agreement
	// consumed two slots against the per-round cap and so REDUCED how many
	// distinct problems a round could fix.
	recP.Issues = o.ledger.Absorb(round, recP.Findings)
	o.logLedgerConflicts()
	mirrorCarriedVerdicts(recP)
	if dup := len(recP.Findings) - len(recP.Issues); dup > 0 {
		o.logf("round %d: %d observation(s) grouped into %d issue(s) (%d corroborating report(s))",
			round, len(recP.Findings), len(recP.Issues), dup)
	}
	o.journal(model.EvIssuesAggregated, round, model.JournalIssuesAggregated{
		Observations: len(recP.Findings),
		Issues:       len(recP.Issues),
		Corroborated: corroboratedCount(recP.Issues),
	})

	o.endPhase("REVIEW  %d finding(s), %d advisory, %d reviewer error(s)",
		len(recP.Findings), len(recP.Advisory), len(recP.ReviewErrors))

	if ctx.Err() != nil {
		sum.Termination = model.TermInterrupted
		return true, nil //nolint:nilerr // interruption is a normal termination, not a round error
	}

	// Review-only runs exactly one round, and ends with a VERDICT rather than a
	// bare success.
	//
	// A partial panel used to fail the run outright, on the argument that
	// reporting success over an incomplete review would let automation trust it.
	// The verdict is a better answer to the same worry and keeps the guarantee:
	// without quorum it cannot approve, so the process still exits non-zero -- but
	// it exits with a reason naming who was missing, instead of an error that
	// cannot tell an infrastructure failure from a reviewer that timed out on one
	// lens of four. See internal/review.
	if o.cfg.Loop.ReviewOnly {
		// Judge the merged set before concluding anything from it: the verdict, the
		// review body and the exit code all read what survives this.
		o.material = material
		o.runRefutation(ctx, recP, material)
		judged := o.runJudge(ctx, recP, material)
		// The verdict is computed either way -- an interrupted review still owes the
		// operator whatever it managed to conclude, and decideVerdict's own callees
		// (postReview) decide for themselves what a canceled context permits.
		postErr := o.decideVerdict(ctx, recP, sum, judged)
		// Checked AFTER decideVerdict, not before it, and that ordering is the whole
		// point: refutation and judging take minutes, and decideVerdict then writes
		// the body and may spend up to the forge timeout posting it, so an interrupt
		// anywhere in that span leaves a review that was never filtered or never
		// published. Recording it as a completed review-only run would exit 0 over
		// exactly that -- a cancellation must never be overwritten with a clean
		// termination, because finishRun honors the canceled context by returning nil
		// and leaves whatever this set standing.
		if ctx.Err() != nil {
			sum.Termination = model.TermInterrupted
			return true, nil //nolint:nilerr // an interruption is a termination, not a round error
		}
		// The review itself finished, so that outcome is recorded BEFORE the posting
		// failure is returned: recordRunError then keeps "review-only" in
		// LoopTermination and marks the run itself failed, which is the honest pair --
		// the panel reached a verdict and the publish the operator asked for did not
		// happen. Nothing is lost by returning here instead of through finishRun: both
		// of its halves are no-ops for a review-only run, by their own design.
		sum.Termination = model.TermReviewOnly
		return true, postErr
	}

	// Termination on clean rounds (advisory findings do not count).
	if len(recP.Findings) == 0 {
		return o.checkCleanStreak(recP, round, sum, cleanStreak), nil
	}
	*cleanStreak = 0

	o.deferOverCap(recP)

	// Nothing left to hand over: every issue the reviewers reported this round was
	// already rejected by the coder in an earlier round (the cap only defers issues
	// while work remains, so an empty workload means exactly that). Invoking the
	// coder on an empty list would spend a session asking about nothing, and looping
	// would just re-report the same decided issues until max_iterations. It is the
	// same terminal state as a round the coder rejected outright -- reported as
	// all-rejected, which exits non-zero because nothing changed.
	if len(activeIssues(recP)) == 0 {
		if len(recP.ReviewErrors) > 0 {
			return false, roundReviewErr(recP)
		}
		o.logf("round %d: all %d issue(s) were already rejected in an earlier round; nothing left to fix", round, len(recP.Issues))
		sum.Termination = model.TermAllRejected
		return true, nil
	}

	// recP is the round just appended, so prior rounds are everything before it.
	prior := sum.Rounds[:len(sum.Rounds)-1]
	// Where the round's commits begin, so commit_policy can regroup them afterwards.
	base, err := o.collector.HeadSHA(ctx)
	if err != nil {
		return false, err
	}
	salvaged, committed, err := o.runFixSessions(ctx, recP, prior, true)
	if err != nil {
		return false, err
	}
	if salvaged {
		// The coder failed but its partial edits were committed; verdicts are
		// unknown, so skip finalize and let the next round re-review. The commits
		// are still regrouped: commit_policy describes how a round's work is laid
		// down, and a round that ended this way laid down commits like any other.
		if err := o.squashSalvagedRound(ctx, recP, base); err != nil {
			return false, err
		}
		return false, nil
	}
	o.logf("round %d: coder fixed %d, rejected %d", round, recP.Fixed, recP.Rejected)
	if err := o.squashRound(ctx, recP, base, committed); err != nil {
		return false, err
	}
	return o.finalizeRound(recP, round, sum, committed)
}

// finalAssignments builds the closing round's work, SPLIT by whether the findings
// get fixed, because the two halves want opposite schedules.
//
// The actionable half drives the repeat: its findings go to the coder, the cap
// bounds each session, and the phase asks again until nothing is left. The advisory
// half is a report, and a report wants to run exactly once, describing the code that
// actually shipped -- which is only known once the actionable half has finished
// changing it. Running both on every pass would print one maintainability and one
// design report per pass, which is the per-round waste `final` exists to remove,
// reintroduced inside the closing round.
//
// Each lens is assigned to EVERY agent it may use, so an unpinned lens gets the
// whole panel: for the actionable half that is breadth on the last look, with no
// next round to catch what one model missed. Pin a lens whose findings are only
// reported -- four overlapping documents is not four times the insight.
func (o *Orchestrator) finalAssignments() (actionable, advisory []model.Assignment) {
	rv := o.cfg.Roles.Review
	for _, l := range rv.Prompts {
		if !l.Final {
			continue
		}
		// A pinned final lens stays pinned: LensAgents returns just that agent.
		for _, a := range rv.LensAgents(l) {
			asg := model.Assignment{Lens: l.Prompt, Agent: a, Advisory: l.Advisory, Pinned: l.Agent != ""}
			if l.Advisory {
				advisory = append(advisory, asg)
			} else {
				actionable = append(actionable, asg)
			}
		}
	}
	return actionable, advisory
}

// deferOverCap enforces loop.max_findings_per_round over ISSUES, not raw
// observations: the worst maxN stay active for the coder and the rest are marked
// deferred. Deferred issues appear in history as still open, so reviewers
// re-report them and they reach the coder in a later round.
//
// Ordering is worst-severity-first with AGING: each round an issue has already
// been deferred promotes it one severity tier. Without that, an unbounded
// generator of medium-severity findings (a coverage lens can always want more
// coverage) starves everything below it forever -- in one five-round run a
// one-line README error was reported three times and never once scheduled. Aging
// bounds the wait: a "low" issue reaches top priority after three skips.
//
// The deferral count is now exact, read from the ledger. It used to be
// approximated from (file, category) because findings had no identity that
// survived a round.
func (o *Orchestrator) deferOverCap(rec *model.RoundRecord) {
	maxN := o.cfg.Loop.MaxFindingsPerRound
	if maxN <= 0 {
		return
	}
	// Only work counts against the cap. An issue the coder rejected in an earlier
	// round arrives already carrying that verdict (the ledger re-reports it so it
	// stays visible, without handing it back), so counting it would let decided
	// issues displace open ones from the round's limited slots -- and deferring one
	// would overwrite the rejection with "deferred", losing the decision.
	var idx []int
	for i := range rec.Issues {
		if coderWork(rec.Issues[i]) {
			idx = append(idx, i)
		}
	}
	if len(idx) <= maxN {
		return
	}
	rank := func(i int) int {
		it := rec.Issues[i]
		// model.SeverityRank sorts unknown severities last, so an invented one
		// cannot jump the queue ahead of a real critical.
		return max(model.SeverityRank(it.Severity)-o.ledger.Deferrals(it.ID), 0)
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ra, rb := rank(idx[a]), rank(idx[b])
		if ra != rb {
			return ra < rb
		}
		// Equal effective rank: whichever has waited longer goes first. The tie
		// would otherwise fall to review order -- deterministic, but it lets a fresh
		// issue edge out one already skipped, which is the starvation aging exists
		// to stop.
		return o.ledger.Deferrals(rec.Issues[idx[a]].ID) > o.ledger.Deferrals(rec.Issues[idx[b]].ID)
	})
	detail := fmt.Sprintf("deferred: fix round capped at %d issue(s) by loop.max_findings_per_round (gains priority each round it is deferred)", maxN)
	deferred := make([]string, 0, len(idx)-maxN)
	for _, i := range idx[maxN:] {
		o.setIssueVerdict(rec, i, model.VerdictDeferred, detail)
		deferred = append(deferred, rec.Issues[i].ID)
	}
	o.logf("round %d: %d issue(s) exceed the per-round cap of %d; %d deferred to later rounds",
		rec.Round, len(idx), maxN, len(idx)-maxN)
	// Record WHICH issues waited, not just how many: aging is meant to bound the
	// wait, and the bug it fixed (a finding reported three times and never once
	// scheduled) is only visible by tracking an id across rounds.
	o.journal(model.EvIssuesDeferred, rec.Round, model.JournalIssuesDeferred{
		Cap:      maxN,
		Active:   maxN,
		Deferred: len(deferred),
		IDs:      deferred,
	})
}

// logLedgerConflicts reports every reviewer-declared issue id the ledger refused
// to honor. A refusal means a reviewer cited an id whose issue was already
// rejected while describing something else, and the observation was filed under
// its own issue instead of being buried under that rejection. Either the reviewer
// is confused about the history or something is steering it, and both are worth
// seeing in the log rather than inferring from an issue count.
func (o *Orchestrator) logLedgerConflicts() {
	for _, c := range o.ledger.TakeConflicts() {
		o.logf("%s", c)
	}
}

// mirrorCarriedVerdicts copies a verdict an issue ALREADY carries out of Absorb
// onto this round's observations of it. Only a previously rejected issue arrives
// that way, and setIssueVerdict never reaches it (the cap and the coder both skip
// non-work issues), so its observations would keep an empty verdict and
// FormatHistory would render them UNRESOLVED. The history preamble tells
// reviewers to re-report anything UNRESOLVED, so the run would solicit the
// re-report of a decided issue every round and the summary would show it as
// unresolved in every round after the one that rejected it.
func mirrorCarriedVerdicts(rec *model.RoundRecord) {
	for _, it := range rec.Issues {
		if it.Verdict == "" {
			continue
		}
		for j := range rec.Findings {
			if rec.Findings[j].IssueID == it.ID {
				rec.Findings[j].Verdict = it.Verdict
				rec.Findings[j].VerdictDetail = it.VerdictDetail
			}
		}
	}
}

// issueFindings returns the round's observations of one issue -- the
// finding-level view of it, for logs that speak in the terms reviewers used.
func issueFindings(rec *model.RoundRecord, issueID string) []model.Finding {
	var out []model.Finding
	for _, f := range rec.Findings {
		if f.IssueID == issueID {
			out = append(out, f)
		}
	}
	return out
}

// setIssueVerdict records a verdict on one of the round's issues, in the ledger,
// and on every observation that reported it -- so history and the run summary keep
// speaking in the terms reviewers used while the coder works from issues.
func (o *Orchestrator) setIssueVerdict(rec *model.RoundRecord, i int, verdict, detail string) {
	it := &rec.Issues[i]
	it.Verdict = verdict
	it.VerdictDetail = detail
	it.Status = verdict
	o.ledger.Record(it.ID, verdict, detail)
	for j := range rec.Findings {
		if rec.Findings[j].IssueID == it.ID {
			rec.Findings[j].Verdict = verdict
			rec.Findings[j].VerdictDetail = detail
		}
	}
}

// issueVerdictDetail reads back the coder's verdict detail for one of the round's
// issues, LIVE from the record.
//
// It cannot be taken from a model.Issue the caller is holding: runFixSessions
// iterates a snapshot taken before any session ran, so its copies still carry the
// empty detail they had then, and setIssueVerdict writes the coder's detail onto
// rec.Issues and rec.Findings -- never onto a copy someone else took. Reading it
// back here is what puts what the coder changed into the per-fix commit body.
func issueVerdictDetail(rec *model.RoundRecord, id string) string {
	for i := range rec.Issues {
		if rec.Issues[i].ID == id {
			return rec.Issues[i].VerdictDetail
		}
	}
	return ""
}

// reopenFixedIssue withdraws ONE fixed verdict and reports whether it found one to
// withdraw, for the cases where a recorded fix did not land: the verification
// correction reverted the edits so there is nothing to commit, or the edits were
// discarded before a commit could be made (see withdrawUncommittedFix). The verdict is
// cleared everywhere setIssueVerdict wrote it -- the round's issue, the ledger, and
// every observation that reported it -- and rec.Fixed is decremented, so the
// summary, the commit-less fix, and the history the next reviewers read agree that
// the issue is still open. An empty verdict renders as UNRESOLVED, which is exactly
// the instruction reviewers need: report it again if it is still there.
//
// Only the issue whose fix was reverted is reopened. Fixes are committed one issue
// at a time, so earlier issues in the round are already in the history; withdrawing
// their verdicts too would mark them open in the ledger and the summary while their
// commits still stand.
//
// Rejections are left alone: a rejection is a judgement, not an edit, so reverting
// edits does not withdraw it.
func (o *Orchestrator) reopenFixedIssue(rec *model.RoundRecord, id string) bool {
	for i := range rec.Issues {
		it := &rec.Issues[i]
		if it.ID != id || it.Verdict != model.VerdictFixed {
			continue
		}
		it.Verdict = ""
		it.VerdictDetail = ""
		it.Status = model.StatusOpen
		o.ledger.Reopen(it.ID)
		for j := range rec.Findings {
			if rec.Findings[j].IssueID == it.ID {
				rec.Findings[j].Verdict = ""
				rec.Findings[j].VerdictDetail = ""
			}
		}
		if rec.Fixed > 0 {
			rec.Fixed--
		}
		return true
	}
	return false
}

// withdrawUncommittedFix rolls back the fixed verdict of the issue a fix session
// was working on when that session ends without a commit, and passes cause back
// through so callers can return it in one expression.
//
// Every abnormal exit between the coder's report and the commit -- the gate
// blocking after a correction attempt, a failed commit, an interruption, a coder
// that reported a fix it never made -- discards the edits (they are stashed, not
// committed). The verdict, the ledger status, the mirrored observation verdicts and
// rec.Fixed were all applied when the coder reported, so leaving them standing
// makes the run summary and the scoreboard claim a fix that no commit contains,
// and the summary is written on the error path too. The issue is still open, so it
// is recorded as open.
//
// It is called from runFixSessions, the loop that owns the issue, because that is
// the only place that knows WHICH issue the discarded edits belonged to: the discard paths
// themselves are shared with round-level exits (a failed reviewer, a salvage) that
// have no single issue to withdraw. Only the current issue is touched -- earlier
// fixes in the round are already committed (see reopenFixedIssue).
//
// A no-op when there is no fixed verdict to withdraw: a rejected issue keeps its
// rejection, since a rejection is a judgement rather than an edit.
func (o *Orchestrator) withdrawUncommittedFix(rec *model.RoundRecord, id string, cause error) error {
	if o.reopenFixedIssue(rec, id) {
		o.logf("round %d: %s was reported fixed but its edits were not committed; the finding is reopened", rec.Round, id)
	}
	return cause
}

// coderWork reports whether one of the round's issues is work for THIS round's
// coder. It is asked BEFORE the coder answers, so the only verdicts an issue can
// already carry are the two that mean "not this round's work":
//
//   - deferred by the per-round cap, so it was never handed over;
//   - rejected in an EARLIER round. The ledger deliberately returns a re-reported
//     rejected issue carrying that verdict so the summary shows it came up again,
//     without resubmitting a decision already made. Handing it back would spend a
//     slot -- and a coder verdict -- on it every round for the rest of the run.
func coderWork(it model.Issue) bool {
	return it.Verdict != model.VerdictDeferred && it.Verdict != model.VerdictRejected
}

// activeIssues returns the issues actually handed to the coder this round: the
// round's work, minus what the cap deferred and what an earlier round already
// rejected. The rejected ones stay in the round record for reporting.
func activeIssues(rec *model.RoundRecord) []model.Issue {
	out := make([]model.Issue, 0, len(rec.Issues))
	for _, it := range rec.Issues {
		if coderWork(it) {
			out = append(out, it)
		}
	}
	return out
}

// pendingIssues returns the round's issues that carry NO verdict at all: what the
// coder died on, plus anything it never reached. Unlike activeIssues it is asked
// AFTER sessions have run, so it must also exclude the issues earlier sessions in
// the same round already fixed -- those have a verdict, and listing them as
// undecided would contradict the Fixed section of the same commit message.
func pendingIssues(rec *model.RoundRecord) []model.Issue {
	out := make([]model.Issue, 0, len(rec.Issues))
	for _, it := range rec.Issues {
		if it.Verdict == "" {
			out = append(out, it)
		}
	}
	return out
}

// allAdvisory reports whether every assignment in a pass is advisory, which makes
// the pass a report: nothing it produces reaches the coder, so it cannot leave a
// fix acting on a partial finding list. An empty list is not such a pass -- the
// callers never build one, and treating "no reviewers" as "all advisory" would
// widen the exemption by accident.
func allAdvisory(asgs []model.Assignment) bool {
	if len(asgs) == 0 {
		return false
	}
	for _, a := range asgs {
		if !a.Advisory {
			return false
		}
	}
	return true
}

// deferredFindings counts the round's observations whose issue the cap deferred.
// Those never reached the coder, which is why an all-rejected round with deferred
// work left is not a terminal state.
func deferredFindings(rec *model.RoundRecord) int {
	n := 0
	for _, f := range rec.Findings {
		if f.Verdict == model.VerdictDeferred {
			n++
		}
	}
	return n
}

// checkCleanStreak advances (or resets) the consecutive-clean-round streak for
// a round that produced no non-advisory findings, and reports done=true once it
// reaches clean_rounds_to_stop (setting TermConverged). A round with reviewer
// errors RESETS the streak: "consecutive clean rounds" means consecutive
// fully-successful clean rounds.
//
// Convergence is also where the run's last chance to mention withheld work is, so
// warnConvergedWithUnresolved is consulted before the verdict is set.
func (o *Orchestrator) checkCleanStreak(rec *model.RoundRecord, round int, sum *model.RunSummary, cleanStreak *int) (done bool) {
	if len(rec.ReviewErrors) == 0 {
		*cleanStreak++
	} else {
		*cleanStreak = 0
		o.logf("round %d had reviewer errors; clean streak reset", round)
	}
	// Convergence is decided here, so the streak is journaled here -- including the
	// reset case, where a clean round produced no progress because a reviewer failed.
	o.journal(model.EvRoundClean, round, model.JournalRoundClean{
		Streak:       *cleanStreak,
		Needed:       o.cfg.Loop.CleanRoundsToStop,
		ReviewErrors: len(rec.ReviewErrors),
	})
	if *cleanStreak >= o.cfg.Loop.CleanRoundsToStop {
		o.warnConvergedWithUnresolved()
		sum.Termination = model.TermConverged
		return true
	}
	o.logf("clean round (%d/%d consecutive needed)", *cleanStreak, o.cfg.Loop.CleanRoundsToStop)
	return false
}

// warnConvergedWithUnresolved names the issues the LEDGER still holds unresolved
// at the moment the loop declares convergence.
//
// The clean-round streak is computed from THIS round's observations, so it only
// knows what reviewers reported now. An issue the cap deferred was withheld from
// the coder deliberately, and it reaches the coder only through a reviewer
// re-reporting it in a later round -- if none does (a `once` lens that already had
// its turn, a rotated panel, a reviewer that simply stopped mentioning it), the
// round reads as clean and the run converges and exits 0 over work fixpoint itself
// held back. An issue a dead coder left with no verdict at all is dropped the same
// way.
//
// The loop is NOT extended for them: they are not in this round's issue list, so a
// further round has nothing to hand over and would only re-ask the review that
// just came back empty. Silence is what costs here, so the record is what this
// restores -- the same job warnFinalPhaseCapped does for the closing phase. FIXED
// and REJECTED are decided and not counted; advisory findings never enter the
// ledger at all.
func (o *Orchestrator) warnConvergedWithUnresolved() {
	issues := o.ledger.Issues()
	parts := make([]string, 0, len(issues))
	for _, it := range issues {
		st := it.StatusOrDefault()
		if st == model.VerdictFixed || st == model.VerdictRejected {
			continue
		}
		loc := it.Loc()
		if loc != "" {
			loc = " " + loc
		}
		parts = append(parts, fmt.Sprintf("%s (%s)%s: %s", it.ID, st, loc, it.Title))
	}
	if len(parts) == 0 {
		return
	}
	o.logf("WARNING: converged with %d issue(s) never resolved -- deferred by the per-round cap or left undecided, and not re-reported since, so no round handed them to the coder: %s",
		len(parts), strings.Join(parts, "; "))
}

// unrejectedIssues counts the round's issues that reached the coder and did NOT
// come back rejected. Unlike coderWork it is asked AFTER the coder answered, so
// anything it counts is still open work: a fixed issue whose verification
// correction reverted the edits arrives here with no verdict at all, because
// reopenFixedIssue withdrew it.
func unrejectedIssues(rec *model.RoundRecord) int {
	n := 0
	for _, it := range rec.Issues {
		if it.Verdict != model.VerdictDeferred && it.Verdict != model.VerdictRejected {
			n++
		}
	}
	return n
}

// finalizeRound recognizes a round's terminal state once every fix in it has been
// verified and committed individually by runFixSessions. Nothing committed means
// the coder rejected everything it was handed -- a successful terminal state, but
// only when the round's review was complete, nothing was held back, and every
// issue really does carry a rejection.
func (o *Orchestrator) finalizeRound(rec *model.RoundRecord, round int, sum *model.RunSummary, committed int) (done bool, err error) {
	if committed > 0 {
		return false, nil
	}
	// Nothing committed: a genuine all-rejected round. It is a successful terminal
	// state only when the round's review was complete; with reviewer errors it may
	// just mean the failed reviewers' findings never arrived.
	if len(rec.ReviewErrors) > 0 {
		return false, roundReviewErr(rec)
	}
	// Nothing committed is not proof of a rejection. A verification correction can
	// revert a reported fix outright, and verifyAndCommitFix then withdraws the
	// verdict and reopens the issue without producing a commit. Ending the run as
	// all-rejected there would report a decision nobody made, over a defect still
	// sitting in the tree -- so keep looping while any issue is still open.
	if open := unrejectedIssues(rec); open > 0 {
		o.logf("round %d: nothing committed, but %d issue(s) are still open (a reverted fix, not a rejection); continuing", round, open)
		return false, nil
	}
	// Deferred findings never reached the coder, so "everything was rejected"
	// does not hold for the round as a whole -- keep looping so they get
	// their turn.
	if deferred := deferredFindings(rec); deferred > 0 {
		o.logf("round %d: coder rejected all active findings but %d deferred remain; continuing", round, deferred)
		return false, nil
	}
	sum.Termination = model.TermAllRejected
	return true, nil
}

// journalAssignments renders the round's lens-to-agent mapping in the same
// "lens→agent" form the summary uses, so the two artifacts read alike.
func journalAssignments(asgs []model.Assignment) []string {
	out := make([]string, 0, len(asgs))
	for _, a := range asgs {
		s := config.LensName(a.Lens) + "->" + a.Agent
		if a.Advisory {
			s += " (advisory)"
		}
		out = append(out, s)
	}
	return out
}

// corroboratedCount is how many issues more than one agent reported independently.
// It is the panel's strongest evidence and is invisible in any raw count.
func corroboratedCount(issues []model.Issue) int {
	n := 0
	for _, it := range issues {
		if len(it.Agents()) > 1 {
			n++
		}
	}
	return n
}

// roundReviewErr aggregates a round's reviewer failures into one error.
func roundReviewErr(rec *model.RoundRecord) error {
	return fmt.Errorf("round %d: %d reviewer(s) failed: %s",
		rec.Round, len(rec.ReviewErrors), strings.Join(rec.ReviewErrors, "; "))
}

// warnInheritedEnv reports agents that opted out of environment filtering. Worth a
// warning rather than silence: the filtered default is what keeps an exported
// secret out of a prompt-injectable reviewer, and inherit_all gives that up for
// one agent without changing anything visible in the run's output.
//
// A warning is the right level only because these declarations are the OPERATOR's:
// an agent file resolved from inside the target cannot set inherit_all at all
// (config.rejectProjectSuppliedInheritAll refuses the run while the configuration
// is compiled), so nothing that reaches here was authored by the reviewed code.
func (o *Orchestrator) warnInheritedEnv() {
	for _, n := range o.activeAgentNames() {
		if !o.cfg.Agents[n].Env.InheritAll {
			continue
		}
		o.logf("WARNING: agent %q sets env.inherit_all -- it receives fixpoint's ENTIRE environment, so every exported secret (cloud credentials, database passwords, tokens for other services) is readable by that agent and can be quoted into a finding. Declare the variables it actually needs under env.pass instead", n)
	}
}

// warnArgModePrompts logs a warning for every agent this run will use that is
// configured prompt_via: arg. Such an agent receives the full prompt on its
// process argument list, where the embedded review material (in git-diff/pr
// mode the diff -- which often IS the credential under review) and any secret a
// reviewer quotes into a finding are world-readable via ps / /proc for the
// invocation's lifetime, bypassing the 0600 log permissions and on-disk
// redaction. The exposure is not silent; stdin is the secure default.
func (o *Orchestrator) warnArgModePrompts() {
	for _, n := range o.activeAgentNames() {
		if o.cfg.Agents[n].PromptVia != config.PromptViaArg {
			continue
		}
		o.logf("WARNING: agent %q uses prompt_via: arg -- the full prompt (reviewed material, plus any secret quoted into a finding) is placed on the process argument list and is readable by other local users via ps / /proc for the run's duration, bypassing the 0600 log permissions and on-disk redaction; prefer prompt_via: stdin for material that may contain secrets", n)
	}
}

// warnTargetSuppliedCommand reports an agent command element that resolves inside
// target.path while the target is a pull request. config.Validate REFUSES that
// combination absent a trust assertion (the PR's `gh pr checkout` decides what
// fixpoint execs as the agent process); with -trusted-target/-allow-untrusted-fix
// the run proceeds, and this is where the operator learns that the assertion also
// accepted direct target-controlled code execution -- not merely the coder
// prompt-injection risk the flags are documented for.
//
// Only pr mode, matching the refusal: in every other mode no checkout replaces
// the file between validation and the invocation that runs it.
func (o *Orchestrator) warnTargetSuppliedCommand() {
	if o.cfg.Target.Mode != config.ModePR {
		return
	}
	for _, n := range o.activeAgentNames() {
		tok := config.TargetSuppliedArg(o.cfg.Agents[n].Argv(), o.cfg.Target.Path)
		if tok == "" {
			continue
		}
		o.logf("WARNING: agent %q has command element %q inside target %s, and in mode pr that path's content is the PR's -- `gh pr checkout` writes the branch before the first round, so PR-authored code runs as the agent process itself, with the credentials this agent declares and before any reviewer sandbox; -trusted-target/-allow-untrusted-fix accepts that on top of coder prompt-injection. Point the command at a binary outside the target, or review under an external sandbox (container/VM)", n, tok, o.cfg.Target.Path)
	}
}

// activeAgentNames returns the sorted, distinct set of agents this run will
// invoke: the coder (unless review-only) plus every active reviewer agent and the
// judge when one is configured. Ping and warnArgModePrompts share it so "the
// agents this run uses" has a single definition. Sorted so per-agent log lines
// and failure lists are deterministic run-to-run, matching review's index-ordered
// collection.
//
// The judge is unconditional once configured: it runs on the review-only path
// too, so it is active whenever it is set. Leaving it out silently dropped all
// four per-agent preflights for it -- prompt_via: arg (the judge prompt embeds
// the material AND every finding's text, so argv exposure is at its worst there),
// env.inherit_all, a target-supplied command, and the ping that would have caught
// an expired login before the panel was paid for rather than after, where a
// failing judge downgrades the verdict to inconclusive.
func (o *Orchestrator) activeAgentNames() []string {
	seen := map[string]bool{}
	var names []string
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	if !o.cfg.Loop.ReviewOnly {
		add(o.cfg.Roles.Coder.Agent)
	}
	for _, a := range o.cfg.Roles.Review.ActiveAgents() {
		add(a)
	}
	if o.cfg.Roles.Judge.Agent != "" {
		add(o.cfg.Roles.Judge.Agent)
	}
	// Triage runs before the first round and its failure costs the operator every
	// answer to every comment, so it belongs in the preflight ping like any other
	// agent the run will actually invoke -- an expired login should fail here rather
	// than after the panel has been paid for.
	if o.cfg.Roles.Triage.Agent != "" {
		add(o.cfg.Roles.Triage.Agent)
	}
	sort.Strings(names)
	return names
}

// pingTimeout caps each preflight ping so a hung CLI cannot stall startup.
const pingTimeout = 3 * time.Minute

// pingTimeoutFor caps an agent's configured timeout to pingTimeout for the
// preflight ping: a hung agent (or one left on the generous default timeout)
// must not stall startup for its full run timeout on each retry. A timeout at or
// below pingTimeout is preserved unchanged.
func pingTimeoutFor(a config.Agent) config.Duration {
	if a.Timeout.Std() > pingTimeout {
		return config.Duration(pingTimeout)
	}
	return a.Timeout
}

// Scope reports how much material a run would review, for --check. It goes
// through the orchestrator's own collector rather than a fresh one so the
// estimate honors the same exclusions the run will apply -- including the logs
// directory, which only the orchestrator knows the location of.
func (o *Orchestrator) Scope(ctx context.Context) (string, error) {
	return o.collector.Scope(ctx)
}

// Ping invokes every distinct agent this run will use with a trivial prompt,
// in parallel, and returns an error naming every agent that failed. Success is
// a zero exit with non-empty stdout (models decorate output; content is not
// checked).
func (o *Orchestrator) Ping(ctx context.Context) error {
	// activeAgentNames includes the write-capable coder unless review-only, and
	// pinging it launches that CLI in target.path with permission checks disabled
	// -- where it loads repository instructions (AGENTS.md, CLAUDE.md) that an
	// untrusted target could weaponize. A real run refuses a fix round without a
	// trust signal (checkFixTrust); -check-live must not offer an unguarded path
	// around that gate, so enforce it here before any agent is invoked.
	if err := o.checkFixTrust(); err != nil {
		return err
	}
	// The target-integrity gates belong here for the same reason: a ping launches
	// each CLI with its working directory INSIDE the target, and those CLIs run
	// `git status`/`git diff`/`git log` while exploring, so a repo-supplied
	// filter.<name>.clean (or core.sshCommand, or a credential helper) executes with
	// fixpoint's inherited environment, and a core.worktree redirect points them at
	// a tree outside the target. That holds in EVERY mode, including directory
	// review-only -- which is why this is PreflightGuards and not the narrower
	// no-agent variant --check uses. -check-live reaches Ping without going through
	// run() -- and it is strictly more invasive than -check, which is gated -- so
	// gating at the ping rather than at the caller keeps every agent-invoking entry
	// point covered. PreflightGuards is idempotent, so run() still probes once.
	if err := o.PreflightGuards(ctx); err != nil {
		return err
	}
	names := o.activeAgentNames()

	type ping struct {
		err error
		dur time.Duration
	}
	results := make([]ping, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			a := o.cfg.Agents[name]
			// Cap once, before the retry loop, so both attempts use the bounded
			// timeout rather than the agent's full run timeout.
			a.Timeout = pingTimeoutFor(a)
			// One retry: CLIs occasionally fail transiently on cold start,
			// and a false-negative preflight aborts a wanted run.
			var err error
			var dur time.Duration
			for range 2 {
				res := agent.Run(ctx, a, "Reply with the single word: OK", o.cfg.Target.Path)
				err = res.Err
				dur += res.Duration
				if err == nil && strings.TrimSpace(res.Stdout) == "" {
					err = errors.New("no output")
				}
				if err == nil || ctx.Err() != nil {
					break
				}
			}
			results[i] = ping{err: err, dur: dur}
		}(i, name)
	}
	wg.Wait()

	var failed []string
	for i, name := range names {
		p := results[i]
		desc := agentDesc(o.cfg.Agents[name])
		if p.err != nil {
			o.logf("ping: %s %s FAILED (%s): %v", name, desc, p.dur.Round(time.Second), p.err)
			failed = append(failed, name)
		} else {
			o.logf("ping: %s %s ok (%s)", name, desc, p.dur.Round(time.Second))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("agent(s) not working: %s", strings.Join(failed, ", "))
	}
	return nil
}

// agentDesc renders an agent's model (and effort, when set) for log lines:
// "[fable, high]" or "[gemma4:31b-cloud]".
func agentDesc(a config.Agent) string {
	if a.Effort != "" {
		return fmt.Sprintf("[%s, %s]", a.Model, a.Effort)
	}
	return fmt.Sprintf("[%s]", a.Model)
}

// assignments pairs lenses with agents for one round, per the strategy.
func (o *Orchestrator) assignments(round int) []model.Assignment {
	rv := o.cfg.Roles.Review
	var out []model.Assignment
	unpinned := 0
	for _, l := range rv.Prompts {
		pinned := l.Agent != ""
		// Give every unpinned lens a rotation index fixed by its position in the
		// full configured prompt list, advancing the counter BEFORE the once-only
		// skip below. Otherwise skipping an unpinned once:true lens after round 1
		// would shift the indices of the recurring lenses that follow it,
		// re-selecting the same agent across rounds and defeating rotation (and
		// letting a clean-round confirmation reuse the previous round's reviewer).
		idx := unpinned
		if !pinned {
			unpinned++
		}
		if l.Once && round > 1 {
			continue // once-per-run lens (e.g. design/architecture): round 1 only
		}
		// A final lens does not run in the loop at all -- finalAssignments runs it
		// once after the loop, on the whole pool. In a review-only run there IS no
		// closing round (no coder to hand anything to), so it runs here instead,
		// which keeps a review-only config a complete preview of its fix sibling.
		if l.Final && !o.cfg.Loop.ReviewOnly {
			continue
		}
		// agents is the shared "who may serve this lens" policy (config.LensAgents):
		// the pinned agent, or the whole pool. The log-collision validator enumerates
		// the same set, so validation cannot drift from what actually runs.
		agents := rv.LensAgents(l)
		switch {
		case pinned: // pinned under every strategy
			out = append(out, model.Assignment{Lens: l.Prompt, Agent: agents[0], Advisory: l.Advisory, Pinned: true})
		// An unpinned final lens fans out to the whole panel regardless of the
		// review strategy, exactly as finalAssignments does it: breadth on the
		// last look, with no next round to catch what one model missed. Only
		// reachable in a review-only run (the skip above removed the rest), and
		// letting rotate pick a single agent here would make that run a subset
		// of its fix sibling's closing round rather than a preview of it.
		case l.Final, rv.Strategy == config.StrategyAll:
			for _, a := range agents {
				out = append(out, model.Assignment{Lens: l.Prompt, Agent: a, Advisory: l.Advisory})
			}
		default: // rotate
			a := agents[(idx+round-1)%len(agents)]
			out = append(out, model.Assignment{Lens: l.Prompt, Agent: a, Advisory: l.Advisory})
		}
	}
	return out
}

// heartbeatDefault is how often a still-running agent invocation emits a
// progress line, so a long silent stretch is visibly alive rather than
// ambiguous (thinking-heavy coders can run 20+ minutes without output).
const heartbeatDefault = 5 * time.Minute

// heartbeat calls log on every tick until done closes. The inner re-check of
// done is load-bearing: when a tick is already queued as the agent finishes,
// both channels are ready and select picks between them at random, so the tick
// branch can win and announce "still running" for an agent that has already
// returned -- while the caller waits on the join for that stale line.
func heartbeat(done <-chan struct{}, ticks <-chan time.Time, log func()) {
	for {
		select {
		case <-done:
			return
		case <-ticks:
			select {
			case <-done:
				return
			default:
			}
			log()
		}
	}
}

// runAgent persists the prompt (before invoking, so a killed or hung agent's
// input is still inspectable), invokes the agent with a heartbeat, and
// returns the result. label prefixes the heartbeat lines, e.g.
// "fix: claude-coder".
func (o *Orchestrator) runAgent(ctx context.Context, label, role, agentName, lensName string, round int, text string) agent.Result {
	if err := o.logs.Prompt(role, agentName, lensName, round, text); err != nil {
		o.logf("WARNING: writing %s prompt log: %v", role, err)
	}
	start := time.Now()
	// Defensive: New always sets the interval, but a hand-built Orchestrator
	// (several tests construct one directly) leaves it zero, and NewTicker(0)
	// panics -- which would take the whole run down over a progress line.
	every := o.heartbeatEvery
	if every <= 0 {
		every = heartbeatDefault
	}
	done := make(chan struct{})
	// Awaited, not just signaled: a tick already inside logf would otherwise
	// still be printing after runAgent returns, interleaving a "still running"
	// line with whatever the caller logs next (or with the end-of-run table).
	var hb sync.WaitGroup
	hb.Add(1)
	go func() {
		defer hb.Done()
		t := time.NewTicker(every)
		defer t.Stop()
		heartbeat(done, t.C, func() {
			o.progressf("%s still running (%s elapsed)", label, time.Since(start).Round(time.Second))
		})
	}()
	res := agent.Run(ctx, o.cfg.Agents[agentName], text, o.cfg.Target.Path)
	close(done)
	hb.Wait()
	return res
}

// stepStat assembles the per-invocation I/O record for the run summary.
func stepStat(role, agentName, lensName string, promptLen int, res agent.Result, failed bool) model.StepStat {
	return model.StepStat{
		Role:        role,
		Agent:       agentName,
		Lens:        lensName,
		PromptBytes: promptLen,
		OutputBytes: len(res.Stdout) + len(res.Stderr),
		DurationMS:  res.Duration.Milliseconds(),
		Failed:      failed,
		Usage:       res.Usage,
	}
}

// runReviewAssignment renders, invokes, parses, and logs one reviewer
// assignment synchronously, returning the raw findings it reported.
func (o *Orchestrator) runReviewAssignment(ctx context.Context, asg model.Assignment, round int, material string, history []model.RoundRecord) ([]model.ReviewFinding, model.StepStat, error) {
	lensName := config.LensName(asg.Lens)
	label := fmt.Sprintf("review: %s via %s", asg.Agent, lensName)
	d := prompt.ReviewData{
		Mode:         o.cfg.Target.Mode,
		Path:         o.cfg.Target.Path,
		Round:        round,
		ModeGuidance: prompt.ModeGuidance(o.cfg.Target.Mode),
		Target:       material,
		History:      prompt.FormatHistory(history),
		// Nothing lens- or agent-specific may reach the prelude: it is what the
		// round's reviewers share a cache prefix on, and one varying byte costs all
		// of it. asg is deliberately not consulted here.
		OutputContract: prompt.ReviewContract,
	}
	d.Prelude = prompt.FormatPrelude(d)
	text, err := prompt.Render(o.templates[asg.Lens], d)
	if err != nil {
		// No agent ran, so there is no output/duration to record; route through
		// stepStat anyway so both paths build the record the same way.
		return nil, stepStat("review", asg.Agent, lensName, len(text), agent.Result{}, true), err
	}
	o.logf("%s starting (prompt %s)", label, logstore.SizeDesc(len(text)))
	res := o.runAgent(ctx, label, "review", asg.Agent, lensName, round, text)
	var out model.ReviewOutput
	parseErr := res.Err
	if parseErr == nil {
		parseErr = agent.ExtractJSON(res.Stdout, "review", &out)
	}
	if parseErr == nil {
		parseErr = validateReviewFindings(out.Findings)
	}
	// A reply that broke the CONTRACT rather than the review gets one chance to
	// restate it. See reformatReview for why this is worth a second invocation.
	var salvage salvageCost
	if parseErr != nil && res.Err == nil && ctx.Err() == nil {
		out, res, salvage, parseErr = o.reformatReview(ctx, label, asg, lensName, round, res, parseErr)
	}

	// Per-step logs (IDs are assigned later; md/json here carry raw findings).
	findings := toFindings(out.Findings, asg, lensName, nil, 0)
	md := logstore.RenderReviewMD(asg.Agent, lensName, round, findings, parseErr)
	o.logStep("review", asg.Agent, lensName, round, parseErr == nil, out, md, res, salvage.raw)
	outBytes := len(res.Stdout) + len(res.Stderr) + salvage.outputBytes
	if parseErr != nil {
		o.logf("%s FAILED after %s (%v)", label, res.Duration.Round(time.Second), parseErr)
	} else {
		o.logf("%s done (%d findings, %s, output %s)", label, len(out.Findings), res.Duration.Round(time.Second), logstore.SizeDesc(outBytes))
	}
	// A salvaged step is billed for both invocations: res already carries the
	// summed usage and duration, and salvage carries the byte counts that cannot
	// live on a single Result. See reformatReview.
	stat := stepStat("review", asg.Agent, lensName, len(text)+salvage.promptBytes, res, parseErr != nil)
	stat.OutputBytes = outBytes
	return out.Findings, stat, parseErr
}

// logStep writes one agent invocation's step logs, downgrading a log-write
// failure to a WARNING (a run must not abort because a log file could not be
// written). parsed is persisted only when ok, so a failed step never writes a
// JSON payload that would read as a clean result.
//
// priorRaw is a preceding invocation billed to this same step -- currently only a
// salvaged review's first attempt (see reformatReview) -- and is written ahead of
// res in the .raw log. Empty for a step that ran once. Without it the .raw of a
// salvaged step would hold only the reformat's reply and silently omit the very
// output whose failure triggered the salvage, which is precisely what the
// fidelity record exists to show.
func (o *Orchestrator) logStep(role, agentName, promptName string, round int, ok bool, out any, md string, res agent.Result, priorRaw string) {
	var parsed any
	if ok {
		parsed = out
	}
	raw := res.Raw(o.cfg.Agents[agentName].Argv())
	if priorRaw != "" {
		// Both attempts, in the order they ran, each under its own banner. res's
		// duration line is the step's summed wall clock (reformatReview merges it),
		// so the second banner says so rather than leaving it to be misread as the
		// reformat's own.
		raw = "=== attempt 1 (output did not meet the contract) ===\n" + priorRaw +
			"\n\n=== attempt 2 (reformat; duration below is the step total) ===\n" + raw
	}
	if err := o.logs.Step(role, agentName, promptName, round, parsed, md, raw); err != nil {
		o.logf("WARNING: writing %s log: %v", role, err)
	}
}

// reformatReview asks a reviewer whose reply broke the output contract to restate
// it, and returns whichever attempt to believe.
//
// Only a CONTRACT failure reaches here -- res.Err == nil, so the agent ran and
// exited cleanly, and what failed was the extractor or the finding validator.
// A crashed, timed-out or rate-limited agent is a different thing and is not
// retried: it has nothing to restate.
//
// The trade is a few hundred tokens against a whole session. Three reviews in this
// project's history were lost to the format rather than the work, each one tens of
// turns and millions of tokens, and each also counted as a reviewer error -- which
// resets the convergence streak and denies the run a clean round it had earned.
//
// On failure the FIRST error is kept, not the second. The first says what the agent
// actually did wrong; a second failure of the same kind adds nothing and the
// operator should not have to read two to learn one. Usage, duration and I/O sizes
// from both invocations are summed either way, so the scoreboard never
// under-reports what a salvage cost.
func (o *Orchestrator) reformatReview(ctx context.Context, label string, asg model.Assignment, lensName string, round int, first agent.Result, firstErr error) (model.ReviewOutput, agent.Result, salvageCost, error) {
	o.logf("%s output did not meet the contract (%v); asking it to restate the block", label, firstErr)
	text := prompt.FormatReformat(first.Stdout, firstErr, prompt.ReviewContract)
	res := o.runAgent(ctx, label+" (reformat)", "review", asg.Agent, config.ReformatLensName(lensName), round, text)
	// Bill both attempts to this step whatever happens: a salvage that fails must
	// not look cheaper than one that succeeds. Usage and wall clock merge onto the
	// Result; the byte counts cannot (Stdout is what the parser reads, so it stays
	// the reformat's own reply) and travel in salvageCost -- as does the first
	// attempt's rendered output, which the step's .raw log would otherwise lose.
	merged := res
	merged.Usage.Add(first.Usage)
	merged.Duration += first.Duration
	cost := salvageCost{
		promptBytes: len(text),
		outputBytes: len(first.Stdout) + len(first.Stderr),
		raw:         first.Raw(o.cfg.Agents[asg.Agent].Argv()),
	}

	var out model.ReviewOutput
	err := res.Err
	if err == nil {
		err = agent.ExtractJSON(res.Stdout, "review", &out)
	}
	if err == nil {
		err = validateReviewFindings(out.Findings)
	}
	if err != nil {
		o.logf("%s reformat also failed (%v); reporting the original contract error", label, err)
		return model.ReviewOutput{}, merged, cost, firstErr
	}
	o.logf("%s reformat recovered %d finding(s) from a reply that would have been discarded", label, len(out.Findings))
	return out, merged, cost, nil
}

// salvageCost is the part of a reformat's cost that cannot be folded into the
// merged agent.Result: the reformat prompt's size, on top of the original review
// prompt, the first attempt's reply size, and the first attempt's rendered .raw
// text. All of it belongs to the one step record the salvage is billed to.
type salvageCost struct {
	promptBytes int
	outputBytes int
	// raw is the first attempt's Result.Raw rendering (already redacted), written
	// ahead of the reformat's in the step's .raw log so the fidelity record still
	// contains the reply that failed the contract.
	raw string
}

// validateReviewFindings enforces the review contract's required fields before
// findings are accepted. It also closes a subtle failure mode in ExtractJSON:
// when a reviewer echoes its prompt and then emits no valid output, extraction
// can fall back to the contract's own example block -- whose placeholder
// severity ("critical|high|medium|low") is not a real value. Rejecting invalid
// severities turns that silent placeholder into an honest reviewer error
// instead of a fabricated finding handed to the coder.
func validateReviewFindings(findings []model.ReviewFinding) error {
	for _, f := range findings {
		if strings.TrimSpace(f.Title) == "" {
			return errors.New("finding has an empty title")
		}
		if !model.ValidSeverity(f.Severity) {
			return fmt.Errorf("finding %q has invalid severity %q (want %s)", f.Title, f.Severity, strings.Join(model.Severities, " | "))
		}
	}
	return nil
}

// review schedules all assigned reviewers in parallel, collects their results
// deterministically, and updates the round record, assigning finding IDs.
func (o *Orchestrator) review(ctx context.Context, rec *model.RoundRecord, material string, history []model.RoundRecord) {
	type result struct {
		asg      model.Assignment
		findings []model.ReviewFinding
		stat     model.StepStat
		err      error
	}
	results := make([]result, len(rec.Assignments))
	var wg sync.WaitGroup
	for i, asg := range rec.Assignments {
		wg.Add(1)
		go func(i int, asg model.Assignment) {
			defer wg.Done()
			findings, stat, err := o.runReviewAssignment(ctx, asg, rec.Round, material, history)
			results[i] = result{asg: asg, findings: findings, stat: stat, err: err}
		}(i, asg)
	}
	wg.Wait()

	seq := 0
	for _, r := range results {
		rec.Steps = append(rec.Steps, r.stat)
		lensName := config.LensName(r.asg.Lens)
		if r.err != nil {
			rec.ReviewErrors = append(rec.ReviewErrors, fmt.Sprintf("%s via %s: %v", r.asg.Agent, lensName, r.err))
			continue
		}
		fs := toFindings(r.findings, r.asg, lensName, &seq, rec.Round)
		if r.asg.Advisory {
			rec.Advisory = append(rec.Advisory, fs...)
		} else {
			rec.Findings = append(rec.Findings, fs...)
		}
	}
}

// toFindings converts model output to findings; when seq is non-nil, IDs
// r<round>.<n> are assigned.
func toFindings(in []model.ReviewFinding, asg model.Assignment, lensName string, seq *int, round int) []model.Finding {
	out := make([]model.Finding, 0, len(in))
	for _, f := range in {
		nf := model.Finding{
			Agent: asg.Agent,
			Lens:  lensName,
			// The reviewer's own cross-round declaration: "this is issue i3, reworded".
			// It has to survive into the observation, because it is the ONLY signal that
			// identifies a finding whose text and line have both moved -- the ledger's
			// fingerprint fallback cannot. Dropping it here silently created a second
			// issue for a re-report, which restarted deferral aging and resubmitted
			// issues the coder had already rejected.
			IssueID:  f.Issue,
			Category: f.Category,
			// Severity is stored in the form ValidSeverity validated, not the form the
			// reviewer typed: it is the one field consumers treat as a closed vocabulary
			// rather than free text, so the raw spelling has no use downstream and does
			// have a cost -- see model.NormalizeSeverity.
			Severity:    model.NormalizeSeverity(f.Severity),
			File:        f.File,
			Line:        f.Line,
			Title:       f.Title,
			Description: f.Description,
			Suggestion:  f.Suggestion,
			Advisory:    asg.Advisory,
			Round:       round,
		}
		if seq != nil {
			*seq++
			nf.ID = fmt.Sprintf("r%d.%d", round, *seq)
		}
		out = append(out, nf)
	}
	return out
}

// errInterruptedTreeDirty marks the one cancellation-time error that must NOT
// be softened into a clean interruption by Run: the coder's edits could not be
// stashed, so the working tree is left dirty and the user must see a hard
// failure (with the reason recorded in the summary), consistent with the
// salvage path's own reconcile-failure handling.
var errInterruptedTreeDirty = errors.New("working tree left dirty after interruption")

// stashForReconcile performs the stash every abnormal-exit path uses to return
// the tree to a clean state, with the two adjustments a cancellation demands.
//
// Cancellation can land at any instant, including between a caller's ctx.Err()
// check and this call. Passing an already-canceled ctx to StashDirty would fail
// its first git operation instantly, so nothing gets stashed and the tree stays
// dirty -- hence the fresh context, exactly as reconcileInterrupt uses (each git
// operation is still bounded by target's own deadline). And a stash failure at
// that moment must carry errInterruptedTreeDirty, or recordRunError softens it
// into a clean interruption: in a loop round that hides a dirty tree behind
// TermInterrupted, and in the CLOSING round -- where a termination like
// "converged" is already set -- it reports a successful run while the coder's
// unreconciled edits sit in the repository and block the next run's preflight.
//
//nolint:contextcheck // deliberate fresh context: the run's context is already canceled and would fail every git operation in the stash
func (o *Orchestrator) stashForReconcile(ctx context.Context, msg string) (bool, error) {
	canceled := ctx.Err() != nil
	if canceled {
		ctx = context.Background()
	}
	stashed, err := o.collector.StashDirty(ctx, msg, o.gitExclude...)
	if err != nil && canceled {
		err = fmt.Errorf("%w: %w", err, errInterruptedTreeDirty)
	}
	return stashed, err
}

// discard is one abnormal exit's description: what happened, what to stash it
// under, and what the caller wants surfaced.
type discard struct {
	round int
	// issue scopes the discard to ONE coder session -- the issue it was handed.
	// Set, it changes the contract: the record is journaled as session_discarded,
	// and a successful stash returns nil because the round continues with its
	// remaining issues. A failed stash is fatal at either scope; the dirty tree
	// is what the next step trips over, whoever left it.
	issue string
	// reason is a model.Discard* constant.
	reason string
	// base is the error to return, already phrased for the operator.
	base error
	// stashMsg labels the stash entry the operator will find.
	stashMsg string
	// checks names the blocking gate commands, when the gate is why this happened.
	checks []string
	// recovered is logged (not returned) when the stash succeeded and the caller
	// treats this exit as recoverable rather than fatal -- a failed commit whose
	// error is surfaced on its own, for instance.
	recovered string
	// returnBase suppresses the "the edits were stashed" suffix, for a caller whose
	// error is passed through verbatim (an interruption, a commit failure).
	returnBase bool
}

// discardEdits is the one path off an abnormal exit: stash the tree back to clean,
// record what happened in the journal, and return the error to surface. With
// d.issue set the discard is session-scoped -- journaled as session_discarded, and
// a successful stash returns nil because the round continues -- but the fail-closed
// half is identical at both scopes.
//
// Every caller used to do this itself, and the copies had diverged in a way that
// only the journal showed. Two of them journaled BEFORE testing the stash error, so
// their record said `stashed: false` and nothing more -- indistinguishable from
// "there was nothing to stash", when in fact the stash had FAILED and the working
// tree was left dirty. That is the single most important thing this record can say:
// it is the one exit where fixpoint knowingly leaves unreconciled edits behind, the
// run summary may never be written, and the next run refuses at preflight over a
// dirty tree the operator never made. Six paths, one of which is right, is a bug
// factory -- so there is now one.
func (o *Orchestrator) discardEdits(ctx context.Context, d discard) error {
	stashed, serr := o.stashForReconcile(ctx, d.stashMsg)
	ev := model.JournalDiscarded{
		Reason:  d.reason,
		Issue:   d.issue,
		Stashed: stashed,
		Checks:  d.checks,
		Error:   d.base.Error(),
	}
	if serr != nil {
		ev.Error = fmt.Sprintf("%v; tree left dirty: %v", d.base, serr)
	}
	// Journaled on BOTH paths, and after the stash result is known: recording the
	// outcome is the point, and a stash failure is the outcome worth recording most.
	event := model.EvRoundDiscarded
	if d.issue != "" {
		event = model.EvSessionDiscarded
	}
	o.journal(event, d.round, ev)
	if serr != nil {
		return fmt.Errorf("%w; and the modified working tree could not be reconciled (it is left dirty): %w", d.base, serr)
	}
	if stashed && d.recovered != "" {
		o.logf("%s", d.recovered)
	}
	// A session-scoped discard that reconciled cleanly is not the run's exit: the
	// caller moves on to the round's remaining issues.
	if d.issue != "" {
		return nil
	}
	if stashed && !d.returnBase {
		return fmt.Errorf("%w. The edits were stashed -- inspect or recover them with `git stash pop`", d.base)
	}
	return d.base
}

// fix hands the round's active findings to the coder and applies its
// verdicts. When the coder fails mid-round (timeout, session limit, malformed
// output) but already edited files, the partial work is committed and
// salvaged=true is returned: the loop continues and the next round re-reviews
// everything. This recovery was performed manually four times before being
// automated here.
//
// allowSalvage is what makes that bargain explicit. It holds only while a LATER
// round will re-review the salvaged commit; the closing round passes false, and a
// failed coder there discards its edits instead (see discardFailedFix).
//
// batch is the issues THIS session handles -- one, under every commit policy. The
// caller decides, so a commit can honestly claim to contain a single fix: a session
// handed eight issues edits files for all eight at once, and nothing in its report
// says which change served which issue.
//
// stale names files an earlier session of this same round has already committed to
// since the reviewers read the tree; empty when nothing moved. See staleFiles.
func (o *Orchestrator) fix(ctx context.Context, rec *model.RoundRecord, history []model.RoundRecord, allowSalvage bool, batch []model.Issue, stale []string) (salvaged bool, replies []model.FixReply, err error) {
	coder := o.cfg.Roles.Coder
	promptName := config.LensName(coder.Prompt)
	label := "fix: " + coder.Agent
	active := batch
	if len(active) == 1 {
		// Qualify the artifact identity by issue. logs.pattern renders one path per
		// (role, agent, prompt, round, timestamp), and timestamp_format is
		// second-granularity by default -- so several fix sessions in one round would
		// otherwise write the same prompt/output files and silently overwrite each
		// other. This is the same axis the review role uses to keep its lenses apart.
		promptName += "-" + active[0].ID
		label += " on " + active[0].ID
	}
	d := prompt.FixData{
		Mode:           o.cfg.Target.Mode,
		Path:           o.cfg.Target.Path,
		Round:          rec.Round,
		Findings:       prompt.FormatIssues(active),
		History:        prompt.FormatHistory(history),
		Stale:          prompt.FormatStale(stale),
		Conversations:  o.conversations(),
		OutputContract: prompt.FixContract,
	}
	text, err := prompt.Render(o.templates[coder.Prompt], d)
	if err != nil {
		return false, nil, err
	}
	o.phase("FIX %s  %s", fixSubject(active), firstLineOf(fixTitle(active)))
	o.logf("%s starting on %d issue(s) (prompt %s)", label, len(active), logstore.SizeDesc(len(text)))
	res := o.runAgent(ctx, label, "fix", coder.Agent, promptName, rec.Round, text)
	var out model.FixOutput
	runErr := res.Err
	if runErr == nil {
		runErr = agent.ExtractJSON(res.Stdout, "fix", &out)
	}

	// The coder contract requires every finding exactly once with a valid
	// verdict; anything else fails the round rather than being silently
	// miscounted (an empty result set must not read as "all rejected").
	if runErr == nil {
		runErr = o.applyVerdicts(rec, out.Results, active)
	}
	// The replies are carried back to the caller rather than posted here. They may
	// only go out once the fix they describe is COMMITTED -- see runFixSessions.
	if runErr == nil {
		replies = out.Replies
	}

	rec.Steps = append(rec.Steps, stepStat("fix", coder.Agent, promptName, len(text), res, runErr != nil))
	// The session's OWN observations, not the round's: this artifact is written per
	// invocation, and the invocation only ever saw `active`. Rendering rec.Findings
	// would attribute the round's other verdicts to this session -- and disagree with
	// the .json log beside it, which carries only this session's FixOutput.
	var seen []model.Finding
	for _, it := range active {
		seen = append(seen, issueFindings(rec, it.ID)...)
	}
	md := logstore.RenderFixMD(coder.Agent, rec.Round, seen, out.Notes, runErr)
	o.logStep("fix", coder.Agent, promptName, rec.Round, runErr == nil, out, md, res, "")
	if runErr != nil {
		o.logf("%s FAILED after %s (%v)", label, res.Duration.Round(time.Second), runErr)
	} else {
		o.logf("%s done (%s, output %s)", label, res.Duration.Round(time.Second), logstore.SizeDesc(len(res.Stdout)+len(res.Stderr)))
	}
	// The coder's self-report, recorded as such. Whether any of it survives is
	// decided by the verify_finished record that follows.
	fixEv := model.JournalFixFinished{
		Agent:    coder.Agent,
		Issues:   len(active),
		Fixed:    rec.Fixed,
		Rejected: rec.Rejected,
	}
	if runErr != nil {
		fixEv.Error = runErr.Error()
	}
	o.journal(model.EvFixFinished, rec.Round, fixEv)
	// A context cancellation (e.g. Ctrl-C) is the user asking to stop, not a
	// salvageable coder failure: committing the coder's edits would defy that
	// request. This holds even when the coder returned valid output (runErr ==
	// nil) but the context was canceled before we could commit -- checking only
	// runErr would miss that race and commit anyway. In every canceled case,
	// reconcile the tree by stashing any edits (kept recoverable via `git
	// stash`) and return the interruption -- run() maps a cancellation-time
	// error to TermInterrupted. A timeout or malformed response (ctx still live)
	// stays on the salvage path below.
	if ctx.Err() != nil {
		interrupted := runErr
		if interrupted == nil {
			interrupted = ctx.Err()
		}
		return false, nil, o.reconcileInterrupt(rec.Round, interrupted) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
	}
	if runErr != nil {
		if !allowSalvage {
			return false, nil, o.discardFailedFix(ctx, rec.Round, runErr)
		}
		salvaged, err := o.salvagePartialFix(ctx, rec, runErr)
		return salvaged, nil, err
	}
	return false, replies, nil
}

// discardFailedFix handles a failed coder in a round nothing follows -- the
// closing round. salvagePartialFix commits a failed coder's partial edits
// specifically because the NEXT round re-reviews them; after the closing round
// there is no next round, so the same commit would leave work no reviewer ever
// looked at sitting in the repository under a run that still reports the loop's
// successful termination. The edits are stashed instead (recoverable with
// `git stash`) and the round fails, the same fail-closed shape as every other
// abnormal exit.
func (o *Orchestrator) discardFailedFix(ctx context.Context, round int, runErr error) error {
	return o.discardEdits(ctx, discard{
		round:     round,
		reason:    model.DiscardFinalCoderFailed,
		base:      fmt.Errorf("closing round %d: coder failed: %w; no round follows to re-review partial work, so its edits were not committed", round, runErr),
		stashMsg:  fmt.Sprintf("fixpoint: closing round %d discarded (coder failed)", round),
		recovered: fmt.Sprintf("closing round %d: coder failed (%v); its edits were stashed, clean tree restored", round, runErr),
	})
}

// reconcileInterrupt stashes any working-tree edits left by a canceled coder
// so the stop request never leaves a committed round or a dirty tree behind,
// and returns cause (the interruption) -- or errInterruptedTreeDirty when the
// stash itself fails. It uses a fresh context because the run's context is
// already canceled; target.Collector still bounds each git subprocess with its
// own operation deadline.
func (o *Orchestrator) reconcileInterrupt(round int, cause error) error {
	// An already-canceled context, because that is what this path IS. It is how
	// stashForReconcile knows to run the stash on a fresh context (the run's would
	// fail every git operation instantly, stashing nothing) and to mark a stash
	// failure with errInterruptedTreeDirty, so recordRunError cannot soften a dirty
	// tree into a clean interruption.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// returnBase: an interruption surfaces as the interruption. The operator asked
	// to stop and does not need the stash narrated back at them.
	return o.discardEdits(ctx, discard{
		round:      round,
		reason:     model.DiscardInterrupted,
		base:       fmt.Errorf("coder round %d interrupted: %w", round, cause),
		stashMsg:   fmt.Sprintf("fixpoint: interrupted round %d", round),
		returnBase: true,
	})
}

// reconcileFailedCommit stashes the edits that Commit's `git add -A` staged
// before the commit itself failed (non-cancellation), so the tree is never left
// staged and dirty -- which would violate the clean-tree invariant and block the
// next run. It mirrors salvagePartialFix's failed-commit handling: return the
// original commit error to surface, or a combined error if the stash also fails.
func (o *Orchestrator) reconcileFailedCommit(ctx context.Context, round int, commitErr error) error {
	// returnBase: the commit error is what the caller surfaces, unchanged -- the
	// stash is cleanup, not part of the diagnosis.
	return o.discardEdits(ctx, discard{
		round:      round,
		reason:     model.DiscardCommitFailed,
		base:       fmt.Errorf("round %d commit failed: %w", round, commitErr),
		stashMsg:   fmt.Sprintf("fixpoint: recovered edits from failed commit in round %d", round),
		recovered:  fmt.Sprintf("round %d: commit failed (%v); edits stashed (recover with `git stash`), clean tree restored", round, commitErr),
		returnBase: true,
	})
}

// reconcileRejectedSession handles a session that rejected its one issue yet left
// edits in the working tree. No verdict claims those edits, so they must not reach
// a commit -- but they are also often the sign of a job done WELL: disproving a
// finding can mean writing the reproducer or probe test that shows it false, and
// the first session this guard ever tripped on had done exactly that. Treating it
// as the run's exit was disproportionate -- it threw away the round's remaining
// issues, and the verified commits already made, over a session that did what was
// asked. So the edits are stashed through the same path every abnormal exit uses
// (recoverable, named by issue), the stash is recorded on the round, and the loop
// moves to the next issue. Only a stash failure still stops the run: that leaves
// the dirty tree the next session's own verdict reconciliation would trip over.
func (o *Orchestrator) reconcileRejectedSession(ctx context.Context, rec *model.RoundRecord, it model.Issue) error {
	if err := o.discardEdits(ctx, discard{
		round:     rec.Round,
		issue:     it.ID,
		reason:    model.DiscardRejectedWithEdits,
		base:      fmt.Errorf("round %d: coder rejected %s yet modified the working tree; refusing to commit edits no verdict accounts for", rec.Round, it.ID),
		stashMsg:  fmt.Sprintf("fixpoint: round %d: session edits from rejected %s", rec.Round, it.ID),
		recovered: fmt.Sprintf("round %d: %s was rejected but its session edited the tree; the edits were stashed (recover with `git stash pop`) and the round continues", rec.Round, it.ID),
	}); err != nil {
		return err
	}
	rec.StashedRejects = append(rec.StashedRejects, it.ID)
	return nil
}

// salvageHeader and salvageBody are the commit message for work a failed coder
// left behind. The salvage commit itself and both squashes that may replace it --
// per_round and per_run -- use them, because the fact a reader must not lose is the
// same in every shape: part of that commit has no verdict behind it. On a per_run
// squash the round number is the run's extent, exactly as it is in the plain
// per_run header, and the body names the rounds that actually salvaged.
func salvageHeader(round int) string {
	return fmt.Sprintf("fixpoint: round %d (partial, coder failed)", round)
}

// pending are the issues the coder was handed and left undecided.
func salvageBody(coderErr string, pending []model.Issue) string {
	var b strings.Builder
	// coderErr can quote the coder's own malformed output, so it is flattened like
	// the finding text below it.
	fmt.Fprintf(&b, "Coder failed before reporting verdicts: %s\n\nIssues left without a verdict:\n", flattenField(coderErr))
	for _, it := range pending {
		fmt.Fprintf(&b, "- [%s] (%s, %s) %s\n", it.ID, flattenField(it.Category), flattenField(it.Severity), flattenField(it.Title))
	}
	return b.String()
}

// salvagePartialFix handles a coder failure (timeout, session limit,
// malformed output). The coder edits files before it reports, so a failure
// can leave real, per-file-complete work in the tree. That work is committed
// as a clearly-labeled partial round and the loop continues -- the next round
// re-reviews everything, so an incomplete state is caught by reviewers rather
// than stranded. Only when nothing can be committed (clean tree, or the commit
// itself fails) does the round become a hard error; a failing commit additionally
// stashes the edits so the tree is never left dirty.
//
// The partial work passes the SAME verification gate as a normal round. A coder
// that died mid-edit is the case most likely to leave a tree that does not build,
// and committing it unverified would break the invariant verifyAndCommit exists to
// hold: every commit a later round builds on, and that convergence can be declared
// over, has passed the gate. "The next round re-reviews everything" is not a
// substitute -- reviewers are models reading content, not a compiler.
//
// Unlike verifyRound this does NOT attempt a coder correction: the coder just
// failed, so re-invoking it would most likely burn another timeout. Verification
// here is a single pass, and a failure preserves the work and stops the run.
func (o *Orchestrator) salvagePartialFix(ctx context.Context, rec *model.RoundRecord, runErr error) (salvaged bool, err error) {
	if blocking, verr := o.verifyPass(ctx, rec, "", model.VerifyAttemptSalvage); verr != nil {
		return false, verr
	} else if len(blocking) > 0 {
		return false, o.rejectUnverifiedSalvage(ctx, rec, runErr, blocking)
	}
	// Redact reviewer-authored finding text before it lands in the pushed
	// commit message, mirroring the normal round commit and the logstore.
	// pendingIssues, not the single-issue batch this session was handed: the caller
	// abandons the round's remaining issues the moment this returns salvaged, so the
	// commit must name every issue the round left undecided -- the one the coder died
	// on plus the ones no session reached. Anything decided already carries a verdict
	// and is excluded. This is the same set the per_round and per_run squashes use, so
	// all three shapes of the salvage message account for the same work.
	sha, cerr := o.collector.Commit(ctx, salvageHeader(rec.Round),
		agent.RedactSecrets(salvageBody(fmt.Sprint(runErr), pendingIssues(rec))), o.gitExclude...)
	if cerr != nil {
		// A cancellation that lands during the salvage commit is a stop request,
		// not a salvageable failure. Passing the already-canceled ctx to
		// StashDirty below would make its git operations fail instantly and return
		// a reconcile error that lacks errInterruptedTreeDirty, which Run would
		// then soften into a clean interruption even though the tree is dirty.
		// Route it through reconcileInterrupt (fresh context) so the tree is
		// stashed, or the dirty tree is flagged as a hard interruption failure.
		if ctx.Err() != nil {
			return false, o.reconcileInterrupt(rec.Round, runErr) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
		}
		// Both failures are the answer: runErr says why the coder stopped, cerr says
		// why its verified partial work could not be kept. Returning only runErr made
		// the API error and the run summary report a malformed output or a timeout while
		// omitting the actual reason nothing landed (a failing commit signature, say).
		base := fmt.Errorf("coder round %d failed: %w; and the partial work it left could not be committed: %w", rec.Round, runErr, cerr)
		stashMsg := fmt.Sprintf("fixpoint: recovered edits from failed round %d", rec.Round)
		stashed, serr := o.stashForReconcile(ctx, stashMsg)
		ev := model.JournalDiscarded{
			Reason:  model.DiscardSalvageCommitFailed,
			Stashed: stashed,
			Error:   base.Error(),
		}
		if serr != nil {
			ev.Error = fmt.Sprintf("%v; tree left dirty: %v", base, serr)
			o.journal(model.EvRoundDiscarded, rec.Round, ev)
			return false, fmt.Errorf("%w; and the modified working tree could not be reconciled (it is left dirty): %w", base, serr)
		}
		o.journal(model.EvRoundDiscarded, rec.Round, ev)
		if stashed {
			o.logf("round %d: coder failed and the salvage commit also failed (%v); edits stashed (recover with `git stash`), clean tree restored", rec.Round, cerr)
		}
		return false, base
	}
	if sha == "" {
		// Clean tree: the coder did nothing before failing -- a genuine error.
		return false, fmt.Errorf("coder round %d failed: %w", rec.Round, runErr)
	}
	rec.CoderError = runErr.Error()
	rec.CommitSHA = sha
	rec.Commits = append(rec.Commits, sha)
	o.logf("round %d: coder failed (%v) but had modified the tree; partial work committed as %s -- continuing, next round re-reviews", rec.Round, runErr, shortSHA(sha))
	// Partial: committed with UNKNOWN verdicts, because the coder died before
	// reporting them. A reader must be able to tell this commit apart from a normal
	// round, since its Fixed count is not a claim anyone made.
	o.journal(model.EvRoundCommitted, rec.Round, model.JournalRoundCommitted{
		SHA:     sha,
		Partial: true,
	})
	return true, nil
}

// shortSHA abbreviates a commit SHA for a log line without ever slicing past its
// end: Commit returns git's own output, but an unguarded sha[:12] would panic on
// a shorter-than-expected value and lose a run (along with the just-committed
// round it was about to report) over a cosmetic detail.
func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// applyVerdicts validates the coder's result set against the round's ISSUES --
// every id exactly once, only known ids, only valid verdicts -- and applies it.
// Any violation fails the round, and nothing is applied unless the whole set is
// valid: the always-written run summary must never carry a partially applied
// response.
//
// Verdicts land on the issue and are mirrored onto every observation that reported
// it, so the summary and the reviewers' history keep speaking in the terms the
// reviewers used.
// batch is what this session was actually handed: a verdict is expected for each of
// those and accepted for nothing else. Deriving the set from the record instead
// would demand verdicts for issues the session never saw, which is what every other
// issue in the round now is.
func (o *Orchestrator) applyVerdicts(rec *model.RoundRecord, results []model.FixResult, batch []model.Issue) error {
	inBatch := make(map[string]bool, len(batch))
	for _, it := range batch {
		inBatch[it.ID] = true
	}
	// Issues that were never handed to the coder -- outside this batch, deferred by
	// the cap, or rejected in an earlier round -- neither expect nor accept a verdict.
	index := map[string]int{}
	expected := 0
	for i := range rec.Issues {
		if !coderWork(rec.Issues[i]) || !inBatch[rec.Issues[i].ID] {
			continue
		}
		index[rec.Issues[i].ID] = i
		expected++
	}
	seen := map[string]bool{}
	for _, r := range results {
		if _, ok := index[r.ID]; !ok {
			return fmt.Errorf("coder referenced unknown issue id %q", r.ID)
		}
		if seen[r.ID] {
			return fmt.Errorf("coder returned issue id %q more than once", r.ID)
		}
		seen[r.ID] = true
		if r.Verdict != model.VerdictFixed && r.Verdict != model.VerdictRejected {
			return fmt.Errorf("coder gave unknown verdict %q for %s (want fixed | rejected)", r.Verdict, r.ID)
		}
	}
	if len(seen) != expected {
		// Range the ordered issues slice, not the index map, so reported ids keep
		// their order and the error text is deterministic.
		var missing []string
		for _, it := range rec.Issues {
			if coderWork(it) && inBatch[it.ID] && !seen[it.ID] {
				missing = append(missing, it.ID)
			}
		}
		return fmt.Errorf("coder did not give a verdict for issue(s): %s", strings.Join(missing, ", "))
	}
	for _, r := range results {
		o.setIssueVerdict(rec, index[r.ID], r.Verdict, r.Detail)
		if r.Verdict == model.VerdictFixed {
			rec.Fixed++
		} else {
			rec.Rejected++
		}
	}
	return nil
}

// writeVerdictSection writes a "<title>:" header followed by one line per
// finding carrying the given verdict, so the fixed and rejected sections of a
// commit body share a single formatting site.
//
// One line per finding is the format, so every agent-authored field is flattened
// -- which is also what keeps an injected newline from forging a trailer.
func writeVerdictSection(b *strings.Builder, title string, findings []model.Finding, verdict string) {
	b.WriteString(title + ":\n")
	for _, f := range findings {
		if f.Verdict == verdict {
			fmt.Fprintf(b, "- [%s] %s — %s\n", flattenField(f.Category), flattenField(f.Title), flattenField(f.VerdictDetail))
		}
	}
}

// captureVerifyBaseline records how the project's own checks behave BEFORE any
// fix round, so a later failure can be attributed. Without it, no_regressions has
// nothing to compare against and fixpoint could only offer must_pass -- which
// would refuse to work on any repository that already has a failing check, i.e.
// most real ones.
//
// A baseline failure is not an error: it is the fact being recorded.
//
// It runs under must_pass too, even though Report.Blocking never reads the
// baseline on that path. Two things justify the gate run: the journal record of
// how the project stood before fixpoint touched it, which is what a later reader
// needs to attribute a failure under ANY policy, and the up-front warning below
// that a red repository under must_pass will have every round refused. That
// warning costs one gate run and saves the alternative -- max_iterations rounds
// of coder work, each one gated, corrected, refused, and stashed.
func (o *Orchestrator) captureVerifyBaseline(ctx context.Context) {
	if !o.cfg.Verify.Enabled() || o.cfg.Loop.ReviewOnly {
		return
	}
	o.phase("BASELINE  capturing %d verification command(s)", len(o.cfg.Verify.Commands))
	// Closed on every path, including the returns in the middle: the block's summary
	// is the last thing said about the baseline, and an unclosed block would indent
	// the whole run under it.
	outcome := "no baseline"
	defer func() { o.endPhase("BASELINE  %s", outcome) }()
	rep := verify.Run(ctx, o.cfg.Verify, o.cfg.Target.Path, o.verifyEnv)
	// A cancellation inside the baseline is ordinary -- it is the longest step before
	// round 1, a full build and test suite over an untouched tree -- and it leaves a
	// SHORT report: the in-flight command carries the interruption the teardown caused
	// and the rest never started. Neither is a fact about the project, so this must not
	// become the baseline, and least of all be narrated as pre-existing failures the
	// policy tolerates. The run aborts immediately after this (resolveRunBase fails on
	// the dead context), so an unset baseline never reaches a Regressions judgement;
	// what would outlive the run is the journal record, so it says interrupted rather
	// than a verdict. Mirrors verifyPass's post-run ctx check, minus the reconcile:
	// captureVerifyBaseline runs before round 1, so there is no round to reconcile.
	if ctx.Err() != nil {
		o.journal(model.EvVerifyBaseline, 0, model.JournalVerifyFinished{
			Policy:      string(o.cfg.Verify.Policy),
			Interrupted: true,
			Checks:      journalChecks(rep.Results),
		})
		o.logf("verify baseline: interrupted (%v) -- no baseline was captured; the commands that had started were stopped by the run, not by the project", ctx.Err())
		outcome = "interrupted; no baseline captured"
		return
	}
	o.verifyBaseline = rep
	// The baseline is what makes every later "blocking" judgement meaningful under
	// no_regressions, so it is recorded rather than only logged: a reader cannot
	// otherwise tell whether a failing check was this run's fault.
	o.journal(model.EvVerifyBaseline, 0, model.JournalVerifyFinished{
		Policy: string(o.cfg.Verify.Policy),
		Passed: rep.Passed(),
		Checks: journalChecks(rep.Results),
	})
	if rep.Passed() {
		outcome = "all checks pass"
		return
	}
	outcome = rep.Summary()
	o.logf("verify baseline: %s", rep.Summary())
	// What a red baseline MEANS depends entirely on the policy, and saying the
	// wrong one here is worse than saying nothing: Report.Blocking reads the
	// baseline only under no_regressions, so under must_pass these very failures
	// block every round and the operator has to be told that up front rather than
	// discovering it when round 1's edits are refused and stashed.
	if o.cfg.Verify.Policy == config.VerifyMustPass {
		o.logf("verify baseline: the failing checks above are pre-existing, but policy must_pass ignores the baseline -- they WILL block every round until they are fixed or marked optional; switch to no_regressions to work on a repository that starts red")
		return
	}
	// Worth stating plainly: under no_regressions these checks are permitted to
	// keep failing, which is easy to misread later as fixpoint ignoring them.
	o.logf("verify baseline: the failing checks above are pre-existing; policy %s permits them to keep failing", o.cfg.Verify.Policy)
	// Except the ones that never ran: those recorded no fact about the project, so
	// verify.Regressions treats them as having no baseline at all and a later
	// failure of theirs blocks the round. Say so here rather than letting the line
	// above imply the gate is merely tolerating them.
	if unrunnable := rep.Unrunnable(); len(unrunnable) > 0 {
		names := make([]string, 0, len(unrunnable))
		for _, res := range unrunnable {
			names = append(names, fmt.Sprintf("%s (%s)", res.Name, res.Err))
		}
		o.logf("verify baseline: %s could not run at all, so there is no usable baseline for them; a later failure of these WILL block the round -- fix the command or mark it optional",
			strings.Join(names, ", "))
	}
}

// verifyRound runs the gate over the coder's edits, giving the coder one bounded
// correction attempt if it fails. It reports blocked=true when the round must not
// be committed.
//
// One retry, not a loop: a coder that cannot make the project's own checks pass
// with the failure output in hand is unlikely to succeed on the third attempt, and
// an unbounded repair loop is how a run silently burns an entire budget.
func (o *Orchestrator) verifyRound(ctx context.Context, rec *model.RoundRecord, fixed model.Issue) (blocked bool, err error) {
	blocking, err := o.verifyPass(ctx, rec, fixed.ID, model.VerifyAttemptInitial)
	if err != nil || len(blocking) == 0 {
		return false, err
	}

	o.logf("round %d verify: %d blocking failure(s) after fixing %s; asking the coder to correct them", rec.Round, len(blocking), fixed.ID)
	if err := o.fixVerification(ctx, rec, blocking, fixed); err != nil {
		return false, err
	}
	blocking, err = o.verifyPass(ctx, rec, fixed.ID, model.VerifyAttemptCorrection)
	return len(blocking) > 0, err
}

// journalChecks reduces gate results to the journal's per-check shape, dropping
// each command's output: the journal is an index of what happened, and the full
// output is already in the round's own artifacts.
func journalChecks(results []model.VerifyResult) []model.JournalCheck {
	out := make([]model.JournalCheck, 0, len(results))
	for _, r := range results {
		out = append(out, model.JournalCheck{
			Name:     r.Name,
			Passed:   r.Passed,
			Optional: r.Optional,
			ExitCode: r.ExitCode,
			Error:    r.Err,
		})
	}
	return out
}

// blockingNames lists the checks that block a commit under the active policy --
// narrower than "the failing checks", since no_regressions permits a check that was
// already failing at the baseline to keep failing.
func blockingNames(blocking []verify.Result) []string {
	out := make([]string, 0, len(blocking))
	for _, r := range blocking {
		out = append(out, r.Name)
	}
	return out
}

// verifyPass runs the verification commands once over the current working tree
// and returns the failures that block under the configured policy. It records the
// results on the round and logs a summary; label distinguishes repeated passes in
// the log.
//
// Extracted so every path that can produce a commit gates on the SAME check.
// verifyRound adds one bounded coder correction on top of this; the salvage path
// deliberately does not (see salvagePartialFix).
//
// issue is the issue whose fix is being gated (empty on a salvage pass, which gates
// a failed coder's partial work rather than one fix), and attempt is one of
// model.VerifyAttempt*: it names the occasion for the journal and selects the log
// line's human suffix, so the machine and human records of one gate run cannot
// drift apart.
func (o *Orchestrator) verifyPass(ctx context.Context, rec *model.RoundRecord, issue, attempt string) ([]verify.Result, error) {
	if !o.cfg.Verify.Enabled() {
		return nil, nil
	}
	rep := verify.Run(ctx, o.cfg.Verify, o.cfg.Target.Path, o.verifyEnv)
	o.logf("round %d verify%s: %s", rec.Round, verifyAttemptLabel[attempt], rep.Summary())
	blocking := rep.Blocking(o.cfg.Verify.Policy, o.verifyBaseline)
	// Appended, not assigned: the gate runs once per fix, so a round holds one record
	// per run. The blocking set is recorded next to the results it was derived from --
	// the baseline is not part of the summary, so nothing downstream could otherwise
	// tell a pass the gate cleared from one it stopped.
	rec.Verify = append(rec.Verify, model.VerifyRun{
		Issue:    issue,
		Attempt:  attempt,
		Results:  rep.Results,
		Blocking: blockingNames(blocking),
	})
	// Journal before the cancellation check: the gate DID run and its verdict is the
	// one fact in the loop no model produced, so an interruption immediately after
	// must not be the reason it goes unrecorded.
	o.journal(model.EvVerifyFinished, rec.Round, model.JournalVerifyFinished{
		Attempt:  attempt,
		Policy:   string(o.cfg.Verify.Policy),
		Passed:   len(blocking) == 0,
		Checks:   journalChecks(rep.Results),
		Blocking: blockingNames(blocking),
	})
	if ctx.Err() != nil {
		return nil, o.reconcileInterrupt(rec.Round, ctx.Err()) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
	}
	return blocking, nil
}

// verifyAttemptLabel is the human suffix for each gate occasion, kept beside the
// machine names so one is never updated without the other.
var verifyAttemptLabel = map[string]string{
	model.VerifyAttemptInitial:    "",
	model.VerifyAttemptCorrection: " (after correction)",
	model.VerifyAttemptSalvage:    " (partial work from the failed coder)",
}

// rejectUnverifiedRound discards a round whose edits do not survive the gate:
// stash the coder's work so the tree returns to its pre-round state, and fail.
//
// Discarding rather than committing is the fail-closed choice, and consistent with
// the rest of the design: committing edits that break the project would put later
// rounds on top of a broken base and let "converged" mean "converged on something
// that does not build". The work is stashed rather than deleted so an operator can
// inspect or recover it.
func (o *Orchestrator) rejectUnverifiedRound(ctx context.Context, rec *model.RoundRecord) error {
	// The blocking set, not every non-optional failure: under no_regressions a check
	// that was already red at the baseline fails without blocking, and naming it here
	// would accuse a check the policy permits while hiding the one that actually
	// caused the discard. The LAST run is the pass that blocked: verifyPass appends one
	// record per run, and earlier runs in this round are the fixes already committed.
	last, _ := rec.LastVerify()
	blocking := last.Blocking
	return o.discardEdits(ctx, discard{
		round:  rec.Round,
		reason: model.DiscardVerifyFailed,
		base: fmt.Errorf("round %d: verification failed after a correction attempt (%s); the round was not committed",
			rec.Round, strings.Join(blocking, ", ")),
		stashMsg: fmt.Sprintf("fixpoint: round %d discarded (verification failed)", rec.Round),
		checks:   blocking,
	})
}

// rejectUnverifiedSalvage handles the one case where a failed coder's partial work
// cannot be kept: it does not pass verification. The work is stashed rather than
// discarded (it may well contain the useful part of a fix an operator wants to
// finish by hand) and the run stops.
//
// Stopping is the point. The alternative -- commit it and let the next round
// re-review -- is what this replaces: it would put every later round on a base
// known to be broken, and the run could then end as "converged" on a tree that
// does not build.
func (o *Orchestrator) rejectUnverifiedSalvage(ctx context.Context, rec *model.RoundRecord, runErr error, blocking []verify.Result) error {
	names := make([]string, 0, len(blocking))
	for _, r := range blocking {
		names = append(names, r.Name)
	}
	return o.discardEdits(ctx, discard{
		round:  rec.Round,
		reason: model.DiscardSalvageFailed,
		base: fmt.Errorf("coder round %d failed (%w) and the partial work it left does not pass verification (%s); it was not committed",
			rec.Round, runErr, strings.Join(names, ", ")),
		stashMsg: fmt.Sprintf("fixpoint: unverified partial work from failed round %d", rec.Round),
		checks:   names,
	})
}

// fixVerification invokes the coder a second time with the verification failures
// in hand. It reuses the coder's own prompt template so the instructions,
// contract, and history stay identical; only the extra Verification block differs.
func (o *Orchestrator) fixVerification(ctx context.Context, rec *model.RoundRecord, blocking []verify.Result, fixed model.Issue) error {
	coder := o.cfg.Roles.Coder
	a := o.cfg.Agents[coder.Agent]
	d := prompt.FixData{
		Mode:  o.cfg.Target.Mode,
		Path:  o.cfg.Target.Path,
		Round: rec.Round,
		// The gate runs per fix, so the only edits in the tree are the ones the coder
		// just made for this issue -- everything earlier is already committed and
		// verified, and the issues it has not been given yet changed nothing. Naming
		// anything else would ask it to correct a check using work it cannot see.
		Findings:       prompt.FormatIssues([]model.Issue{fixed}),
		Verification:   verify.FormatForCoder(blocking),
		OutputContract: prompt.FixContract,
	}
	text, err := prompt.Render(o.templates[coder.Prompt], d)
	if err != nil {
		return fmt.Errorf("render coder prompt for the verification correction: %w", err)
	}
	// Logged under its own prompt name so the correction attempt is a distinct,
	// inspectable artifact rather than overwriting the fix log it follows -- and
	// qualified by issue, since a round now gates (and can correct) each fix in turn.
	lens := "fix-verify-" + fixed.ID
	if err := o.logs.Prompt("fix", coder.Agent, lens, rec.Round, text); err != nil {
		o.logf("WARNING: failed to write the verification-correction prompt: %v", err)
	}
	res := agent.Run(ctx, a, text, o.cfg.Target.Path)
	var out model.FixOutput
	parseErr := agent.ExtractJSON(res.Stdout, "fix", &out)
	stepErr := res.Err
	if stepErr == nil {
		stepErr = parseErr
	}
	// Failed the same way logStep gates below: a run whose output would not parse
	// produced nothing usable, so the step summary must not report it as a success.
	rec.Steps = append(rec.Steps, stepStat("fix", coder.Agent, lens, len(text), res, stepErr != nil))
	// Rendered like the fix step it corrects, so the one attempt where a coder is
	// re-invoked with the gate's failures in hand has a human-readable record: the
	// issue under correction, the coder's notes, and the failure if there was one.
	md := logstore.RenderFixMD(coder.Agent, rec.Round, issueFindings(rec, fixed.ID), out.Notes, stepErr)
	// Routed through logStep so the correction attempt gets the same ok-gating as
	// every other step: a run that failed or whose output would not parse must not
	// persist a zero FixOutput that reads back as a clean "no results" report.
	o.logStep("fix", coder.Agent, lens, rec.Round, stepErr == nil, out, md, res, "")
	// A failed or unparseable correction attempt is not fatal here: the caller
	// re-verifies regardless, and the gate -- not the coder's self-report -- decides
	// whether the round proceeds.
	if res.Err != nil {
		o.logf("round %d: verification-correction attempt failed: %v", rec.Round, res.Err)
	} else if parseErr != nil {
		o.logf("round %d: verification-correction output was unparseable: %v", rec.Round, parseErr)
	}
	return nil
}

// decideVerdict computes a review run's conclusion and records it on the summary.
//
// The inputs are all facts the round already carries: which agents were assigned,
// which of their steps failed, and what survived as issues. Nothing here asks a
// model anything -- see internal/review for why the verdict must not.
//
// The verdict itself never fails; the error is the one from publishing it, passed
// through from postReview so the caller can fail the run over a post the operator
// asked for and did not get.
func (o *Orchestrator) decideVerdict(ctx context.Context, rec *model.RoundRecord, sum *model.RunSummary, judged bool) error {
	// Only failures on GATING lenses may deny a quorum. An advisory lens is
	// reported for a human and QuorumFrom leaves it out of the panel entirely, so
	// counting its failure here would let a lens that gates nothing for the
	// findings gate everything for the verdict: an agent that answered every
	// gating lens and merely timed out on an advisory one would be Missing, and on
	// a small panel that alone turns an approval into INCONCLUSIVE.
	//
	// Assignment.Lens is the configured reference and StepStat.Lens the basename it
	// was logged under, so the two are matched through config.LensName.
	advisory := map[string]bool{}
	for _, a := range rec.Assignments {
		if a.Advisory {
			advisory[a.Agent+"\x00"+config.LensName(a.Lens)] = true
		}
	}
	failed := map[string]int{}
	for _, st := range rec.Steps {
		if st.Role != "review" || !st.Failed {
			continue
		}
		if advisory[st.Agent+"\x00"+st.Lens] {
			o.logf("advisory lens %s failed for %s; it gates nothing, so it does not cost a quorum slot", st.Lens, st.Agent)
			continue
		}
		failed[st.Agent]++
	}
	d := review.Decide(review.Input{
		Issues:       rec.Issues,
		Quorum:       review.QuorumFrom(rec.Assignments, failed),
		CI:           o.ci,
		BlockAt:      o.cfg.Review.BlockAt,
		FilterFailed: !judged,
	})
	sum.Verdict = d.Summary()
	o.phase("VERDICT  %s", strings.ToUpper(strings.ReplaceAll(string(d.Outcome), "_", " ")))
	for _, r := range d.Reasons {
		o.logf("%s", r)
	}
	err := o.writeReviewBody(ctx, rec, sum, d)
	o.endPhase("VERDICT  %s", strings.ToUpper(strings.ReplaceAll(string(d.Outcome), "_", " ")))
	return err
}

// writeReviewBody renders the review document and puts it in the run directory.
//
// It is written even when the verdict is an approval with nothing to say: the file
// is the run's answer, and an operator who has to work out from its ABSENCE
// whether the review ran is being asked the wrong question. Failure to write it is
// a warning, not a run failure -- the verdict is already in the summary, the
// journal and the exit code. WITH -post it is a run failure, because then the file
// is not the only thing lost: see the write path below.
//
// The returned error is the POSTING one, which is a different matter: see
// postReview for why a publish the operator asked for and did not get has to reach
// the exit status.
func (o *Orchestrator) writeReviewBody(ctx context.Context, rec *model.RoundRecord, sum *model.RunSummary, d review.Decision) error {
	panel := map[string]bool{}
	for _, a := range rec.Assignments {
		panel[a.Agent] = true
	}
	agents := make([]string, 0, len(panel))
	for name := range panel {
		agents = append(agents, name)
	}
	sort.Strings(agents)

	// One signature for the whole review, used by the summary AND by every inline
	// comment: an inline comment is read on its own in the Files tab, with no sight
	// of the review it belongs to, so an unsigned one is an unattributed assertion
	// sitting on somebody's code.
	signature := review.Signature(o.cfg.Review.Signature, review.SignatureFacts{
		Agents:  agents,
		Run:     o.logs.RunID(),
		Version: review.Version(),
		Config:  configBaseName(o.source.Config),
		Verdict: string(d.Outcome),
	})
	body := review.RenderBody(review.BodyInput{
		Config:    configBaseName(o.source.Config),
		Target:    describeTarget(o.cfg.Target),
		Decision:  d,
		Issues:    rec.Issues,
		Advisory:  rec.Advisory,
		Panel:     agents,
		Signature: signature,
	})
	// The PUBLISHED bytes, computed once and used for both the file and the post.
	//
	// This used to write agent.RedactSecrets(body) to disk and post the raw body,
	// so a credential a prompt-injected reviewer had quoted into a finding was
	// masked in the 0600 log directory and published verbatim on the pull request.
	// It also falsified the control the whole posting design rests on: an operator
	// reads review-body.md, sees [REDACTED], and publishes believing they checked
	// the bytes that go out. Caught by the panel reviewing this branch.
	//
	// Redaction is what every other artifact already gets; the terminal escaping
	// comes with it because logstore applies both, and passing the same string
	// through keeps the two copies identical rather than merely similar. Both
	// transforms are idempotent, so logstore re-applying them changes nothing.
	published := publishedText(body)
	path, err := o.logs.ReviewBody(published)
	if err != nil {
		// A warning here used to swallow the publish as well: the early return skipped
		// postReview, so a run given -post exited 0 having posted nothing, and an
		// approval was reported as delivered when it never left the machine.
		//
		// It fails rather than posting the bytes it still holds, because the file is
		// half of what posting means here -- it is what the operator inspects, what
		// `-post-run` replays, and the only local record of what went out. Publishing
		// under the operator's identity with no such record is the wrong half to keep.
		if o.cfg.Review.Post {
			return fmt.Errorf("-post was given but the review body could not be written: %w -- nothing was posted", err)
		}
		o.logf("WARNING: failed to write the review body: %v", err)
		return nil
	}
	sum.ReviewBody = path
	o.logf("review body: %s", path)
	// Recorded before any posting, so `-post-run` can publish exactly this review
	// later without re-running the panel -- and so an operator who reads the file
	// first is reading the bytes that will actually go out.
	for _, c := range inlineComments(rec, o.material, signature) {
		sum.ReviewInline = append(sum.ReviewInline, model.ReviewAnchor{Path: c.Path, Line: c.Line, Body: c.Body})
	}
	return o.postReview(ctx, sum, published)
}

// publishedText is the ONE transform between text an agent wrote and text that
// leaves fixpoint -- to a file, to a terminal, or onto a pull request.
//
// It exists as a function because the review has TWO channels out: the summary
// body and the inline comments. When only the body went through this pair, a
// credential a prompt-injected reviewer had quoted into a finding was masked in
// review-body.md and published verbatim in that finding's line comment -- the
// operator read [REDACTED] and approved the post, and the leak went out beside it.
// Naming the transform is what keeps the two channels from drifting again.
//
// Both transforms are idempotent, so logstore re-applying them changes nothing.
func publishedText(s string) string {
	return agent.EscapeTerminalBlock(agent.RedactSecrets(s))
}

// describeTarget names what was reviewed in one phrase, for the review's footer.
func describeTarget(t config.Target) string {
	if t.Mode == config.ModePR && t.PR > 0 {
		return fmt.Sprintf("pull request #%d", t.PR)
	}
	if t.Mode == config.ModeGitDiff && t.BaseRef != "" {
		return "the changes since " + t.BaseRef
	}
	return string(t.Mode)
}

// configBaseName is the config's bare name, as `fixpoint --list` shows it.
func configBaseName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// readForgeChecks asks the forge what its own checks say about the reviewed head,
// and records the answer for the verdict and the review body.
//
// Reading the forge rather than RUNNING the project's checks is the whole point.
// A review run reviews code somebody else wrote: `make test` on a pull request
// executes that pull request's Makefile, which is arbitrary code on this host with
// this operator's credentials. The forge already ran those checks in its own
// sandbox and will tell us the answer for free.
//
// pr mode only, because that is where a pull request exists to ask about. It is
// also entirely best-effort: no gh/glab installed, no remote, a private repo the
// token cannot see, a head that moved out from under the read -- every one of those
// leaves CI unknown, which never blocks a verdict but is stated in its reasons, so
// an approval never silently rests on a check nobody ran, nor on one that ran on a
// commit nobody read.
//
// head is what the answer must be about: recordReviewedHead pinned it just above,
// and the provider refuses rather than reporting the checks of whatever the pull
// request proposes now.
func (o *Orchestrator) readForgeChecks(ctx context.Context, head string) {
	if o.cfg.Target.Mode != config.ModePR || o.cfg.Target.PR <= 0 {
		return
	}
	p := forge.For(ctx, o.cfg.Target.Path)
	if p == nil {
		o.logf("no GitHub or GitLab remote recognized; the verdict will carry no CI evidence")
		return
	}
	checks, err := p.Checks(ctx, o.cfg.Target.Path, o.cfg.Target.PR, head)
	if err != nil {
		o.logf("WARNING: could not read %s checks (%v); the verdict will carry no CI evidence", p.Kind(), err)
		return
	}
	o.ci = review.CI{Known: checks.Known, Failing: checks.Failing, Pending: checks.Pending}
	switch {
	case len(checks.Failing) > 0:
		o.logf("%s checks: FAILING (%s)", p.Kind(), strings.Join(checks.Failing, ", "))
	case len(checks.Pending) > 0:
		o.logf("%s checks: none failing, %d still running (%s)", p.Kind(), len(checks.Pending), strings.Join(checks.Pending, ", "))
	default:
		o.logf("%s checks: all passing", p.Kind())
	}
}

// runRefutation asks every reviewer on the panel to take a position on the MERGED
// finding set, and drops what they unanimously refute.
//
// It exists because agreement cannot sort signal from noise in this panel.
// Measured across 19 runs: under 4% of findings were reported by more than one
// reviewer, and 0 of 25 findings the coder rejected as false positives were among
// them -- so an intersection would filter none of the noise while discarding 237
// of 248 confirmed defects. What is left is to judge each finding on its own
// evidence, which is a better test than counting votes anyway.
//
// Every reviewer sees the same canonical set and nothing about who reported what.
// Attribution would hand a reviewer a reason that is not evidence: "this came from
// the model I have disagreed with twice today" is not a fact about the code.
func (o *Orchestrator) runRefutation(ctx context.Context, rec *model.RoundRecord, material string) {
	if o.cfg.Review.Refute == "" || len(rec.Issues) == 0 || ctx.Err() != nil {
		return
	}
	panel := panelAgents(rec.Assignments)
	if len(panel) == 0 {
		return
	}
	// Only what the gate protects. Everything below the floor goes straight to the
	// judge, which may already drop it alone -- a second opinion there buys nothing
	// it can act on. See ReviewPolicy.RefuteAt for the measurement.
	floor := o.cfg.Review.RefuteFloor()
	subject := issuesAtOrAbove(rec.Issues, floor)
	if len(subject) == 0 {
		o.logf("refutation: skipped -- none of the %d finding(s) is %s or above", len(rec.Issues), floor)
		return
	}
	o.phase("REFUTE  %d reviewer(s) judging %d of %d finding(s) at %s or above",
		len(panel), len(subject), len(rec.Issues), floor)

	type reply struct {
		agent     string
		answered  bool
		positions []model.RefutePosition
		step      *model.StepStat
	}
	replies := make([]reply, len(panel))
	var wg sync.WaitGroup
	for i, name := range panel {
		wg.Add(1)
		// i and name are passed in rather than captured, matching review() and Ping().
		// Per-iteration loop variables would make this correct today, but the slot a
		// goroutine writes decides which agent's positions land in which reply -- and
		// applyRefutations counts one vote per distinct agent -- so the binding is
		// fixed at spawn time instead of resting on what a later edit to the loop body
		// does to i or name.
		go func(i int, name string) {
			defer wg.Done()
			positions, answered, step := o.refuteWith(ctx, name, rec.Round, subject, material)
			replies[i] = reply{agent: name, answered: answered, positions: positions, step: step}
		}(i, name)
	}
	wg.Wait()
	// Collected AFTER the join, into rec in panel order. Appending from inside the
	// goroutines raced -- two refuters could write the same slice index and one
	// step would vanish, taking that reviewer's tokens out of the run's reported
	// cost. This mirrors review(), which already collects into an indexed slice for
	// the same reason; doing it here also makes the step order deterministic
	// instead of dependent on which refuter finished first.
	for _, r := range replies {
		if r.step != nil {
			rec.Steps = append(rec.Steps, *r.step)
		}
	}

	// An interrupt during the fan-out is not a verdict. A killed refuter answers
	// nothing, so aggregating whoever is left reads a PARTIAL panel as the round's
	// judgment: the survivors' positions are counted, and a finding they refuted
	// comes out Contested with their names in ContestedBy -- a recorded doubt from a
	// round most of the panel never finished, which is then persisted and can be
	// published. applyRefutations' responded == panel rule already stops a shrunken
	// electorate from DELETING anything, but it reads a killed reviewer and one that
	// failed on its own as the same fact, and they are not: nobody stopped the second
	// one. Fail closed in the same shape as runJudge -- a round the operator
	// interrupted records no judgment at all, and an operator's Ctrl-C can never
	// be part of what happens to a finding.
	if ctx.Err() != nil {
		o.logf("refutation: interrupted before it could run; every finding stands")
		o.endPhase("REFUTE  interrupted; every finding stands")
		return
	}

	// Keyed by issue AND by the agent that spoke, at most one position each.
	//
	// A flat list let one reviewer forge unanimity: counting position RECORDS rather
	// than distinct agents meant a single prompt-injected refuter emitting four
	// duplicate refutes for one finding satisfied "every responder refuted it" while
	// the other three never mentioned it -- deleting a genuine high before the
	// verdict ever saw it, which on a malicious pull request turns CHANGES_REQUESTED
	// into APPROVE. Identity is the fix: a reviewer gets one vote per finding no
	// matter how many times it says the same thing.
	// Keyed on what the refuters were SHOWN, not on every finding in the round: a
	// position on an id that was below the floor is a position on something this
	// reviewer never read, and counting it would let the round act on a finding it
	// deliberately did not ask about.
	known := make(map[string]bool, len(subject))
	for _, it := range subject {
		known[it.ID] = true
	}
	byIssue := map[string]map[string]model.RefutePosition{}
	responded := 0
	// "This reviewer answered" and "this reviewer listed positions" are separate
	// facts, and conflating them was the second way to forge unanimity: an agent
	// whose reply parsed but omitted the positions key (or set it to null) left a
	// nil slice, was skipped here, and never counted toward `responded`. Three
	// reviewers answering `{}` plus one refuter made responded == 1 and unanimity
	// trivial. An answer with nothing in it is still an answer: it counts as a
	// responder and casts no vote, which is what makes the count safe.
	for _, r := range replies {
		if !r.answered {
			continue
		}
		responded++
		for _, p := range r.positions {
			switch {
			case !known[p.Issue]:
				// An id nobody was shown. It cannot affect a finding that exists, but a
				// reviewer inventing them is worth saying out loud.
				o.logf("WARNING: refute: %s took a position on %q, which was not in the set it was shown; ignored", r.agent, p.Issue)
				continue
			case byIssue[p.Issue] != nil && byIssue[p.Issue][r.agent] != model.RefutePosition{}:
				o.logf("WARNING: refute: %s gave more than one position on %s; only the first counts", r.agent, p.Issue)
				continue
			}
			if byIssue[p.Issue] == nil {
				byIssue[p.Issue] = map[string]model.RefutePosition{}
			}
			byIssue[p.Issue][r.agent] = p
		}
	}
	if responded == 0 {
		// Nobody judged anything. Dropping findings on the strength of an empty round
		// would be the worst possible reading of silence.
		o.logf("refutation: no reviewer returned a usable position; every finding stands")
		o.endPhase("REFUTE  no usable position returned; every finding stands")
		return
	}
	dropped, contested := applyRefutations(rec, byIssue, responded, len(panel), o.logf)
	o.endPhase("REFUTE  %d of %d responder(s), %d dropped, %d contested", responded, len(panel), dropped, contested)
}

// issuesAtOrAbove selects the findings a refutation round is asked about: those
// at or worse than the floor, in the order they were merged.
//
// Undecided only. A finding the coder already fixed or rejected in an earlier
// round is not a claim anyone still needs a position on, and asking would spend a
// reviewer's pass on settled work.
func issuesAtOrAbove(issues []model.Issue, floor string) []model.Issue {
	rank := model.SeverityRank(floor)
	out := make([]model.Issue, 0, len(issues))
	for _, it := range issues {
		switch it.StatusOrDefault() {
		case model.VerdictFixed, model.VerdictRejected:
			continue
		}
		if model.SeverityRank(it.Severity) <= rank {
			out = append(out, it)
		}
	}
	return out
}

// panelAgents lists the distinct non-advisory agents a round assigned, in a
// stable order.
func panelAgents(assignments []model.Assignment) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(assignments))
	for _, a := range assignments {
		if a.Advisory || seen[a.Agent] {
			continue
		}
		seen[a.Agent] = true
		out = append(out, a.Agent)
	}
	sort.Strings(out)
	return out
}

// refuteWith runs one reviewer's refutation pass. The second return says whether
// this reviewer ANSWERED -- which is not the same as whether it took any position,
// and the caller needs both separately to count unanimity safely. A failed refuter
// is not a round failure: it simply does not get a vote. Nor does its silence help
// anyone delete a finding -- unanimity is measured over the whole assigned panel,
// so a reviewer that never answered counts as one that did not refute.
func (o *Orchestrator) refuteWith(ctx context.Context, agentName string, round int, subject []model.Issue, material string) ([]model.RefutePosition, bool, *model.StepStat) {
	label := "refute: " + agentName
	d := prompt.RefuteData{
		Mode:           o.cfg.Target.Mode,
		Path:           o.cfg.Target.Path,
		Round:          round,
		ModeGuidance:   prompt.ModeGuidance(o.cfg.Target.Mode),
		Target:         material,
		Canonical:      prompt.FormatCanonical(subject),
		OutputContract: prompt.RefuteContract,
	}
	d.Prelude = prompt.FormatPrelude(prompt.ReviewData{
		Mode: d.Mode, Path: d.Path, Round: d.Round, ModeGuidance: d.ModeGuidance, Target: d.Target,
	})
	tmpl, ok := o.templates[o.cfg.Review.Refute]
	if !ok || tmpl == nil {
		o.logf("WARNING: %s: refute prompt %q was never loaded; this reviewer casts no position", label, o.cfg.Review.Refute)
		return nil, false, nil
	}
	text, err := prompt.Render(tmpl, d)
	if err != nil {
		o.logf("WARNING: %s: render failed (%v); this reviewer casts no position", label, err)
		return nil, false, nil
	}
	res := o.runAgent(ctx, label, "refute", agentName, o.cfg.Review.Refute, round, text)
	var out model.RefuteOutput
	parseErr := res.Err
	if parseErr == nil {
		parseErr = agent.ExtractJSON(res.Stdout, "review", &out)
	}
	step := stepStat("refute", agentName, o.cfg.Review.Refute, len(text), res, parseErr != nil)
	// Persisted like a review step. The product of this round is JUDGMENT WITH
	// EVIDENCE, and without the artifact none of it survives: a refuter that is
	// outvoted leaves no trace of what it argued, and "who refuted what, on what
	// grounds" cannot be answered even from a run you have in front of you. That
	// was true of the first three runs and made the round impossible to evaluate.
	o.logStep("refute", agentName, o.cfg.Review.Refute, round, parseErr == nil, out,
		logstore.RenderRefuteMD(agentName, round, out.Positions, parseErr), res, "")
	if parseErr != nil {
		o.logf("WARNING: %s failed (%v); this reviewer casts no position", label, parseErr)
		return nil, false, &step
	}
	valid := make([]model.RefutePosition, 0, len(out.Positions))
	for _, p := range out.Positions {
		if !model.ValidPosition(p.Position) {
			o.logf("WARNING: %s returned an unknown position %q on %s; ignored", label, p.Position, p.Issue)
			continue
		}
		p.Position = strings.ToLower(strings.TrimSpace(p.Position))
		// A refutation with no evidence is not a refutation. Only this position
		// DELETES a finding, and the contract, the prompt and RefutePosition's own
		// doc all say the deletion has to be grounded in code -- but nothing checked
		// it, so a bare {"issue":"i1","position":"refute"} counted toward unanimity
		// and could drop a high finding, then leave the record asserting "refuted by
		// every reviewer that judged it: " with nothing after the colon. Dropped
		// here rather than in the aggregation so the reviewer still counts as a
		// responder and casts no vote on this id, which fails closed: the finding is
		// kept and marked contested instead of deleted.
		if p.Position == model.PositionRefute && strings.TrimSpace(p.Evidence) == "" {
			o.logf("WARNING: %s refuted %s with no evidence; ignored -- a refutation must say what in the code disproves it", label, p.Issue)
			continue
		}
		valid = append(valid, p)
	}
	o.logf("%s done (%d position(s), %s)", label, len(valid), res.Duration.Round(time.Second))
	return valid, true, &step
}

// applyRefutations drops the findings every responder refuted, and marks the rest.
//
// UNANIMOUS, not majority. A refutation deletes a finding from the review and
// nothing downstream will look for it again, while a wrongly-kept finding costs a
// human one paragraph. With that asymmetry the bar for deletion belongs at the top:
// one reviewer still standing behind a defect is enough to keep it.
//
// Unanimity is measured over the whole ASSIGNED PANEL -- not over the positions
// that happen to have arrived for one finding, and not over the reviewers that
// merely responded. Both narrower readings let one reviewer delete a finding on its
// own word, and the difference is the whole safety property.
//
// Counting `refuted == len(positions)` made a single refuter enough whenever the
// others simply omitted that id, which is the likely failure: the contract asks for
// a position on every finding and nothing enforces coverage, so a refuter working
// through thirty of them truncates.
//
// Counting against the RESPONDERS left the same hole one step out, because a
// refuter that fails its contract is not counted as a responder at all. Three of a
// four-agent panel timing out or returning unparseable output alongside one that
// refutes made responded == 1, and unanimity trivial again -- and a pull request is
// input the panel reads, so an injection that breaks one model's output format
// while another refutes a security finding is a way to arrange exactly that. The
// finding is then rejected, the judge skips it, and the run can approve. An
// electorate an attacker can shrink is not an electorate: a reviewer that never
// answered is a reviewer that did not refute, and it keeps the finding.
//
// Issues nobody took a position on are untouched, for the same reason. panel is the
// number of reviewers the round asked, from panelAgents.
func applyRefutations(rec *model.RoundRecord, byIssue map[string]map[string]model.RefutePosition, responded, panel int, logf func(string, ...any)) (dropped, contested int) {
	for i := range rec.Issues {
		it := &rec.Issues[i]
		positions := byIssue[it.ID]
		if len(positions) == 0 {
			continue
		}
		refuted, maintained, unsure := 0, 0, 0
		var evidence string
		// Who doubted it, not just how many: the judge gate downstream must be able to
		// tell a second agent's refutation from the judge's own.
		var refuters []string
		// In agent order, not map order: the evidence a dropped finding records is
		// persisted, and which refuter's words it quotes must not depend on a map
		// walk. Two refuters produced a different VerdictDetail on every run.
		names := make([]string, 0, len(positions))
		for name := range positions {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			p := positions[name]
			switch p.Position {
			case model.PositionRefute:
				refuted++
				refuters = append(refuters, name)
				if evidence == "" {
					evidence = p.Evidence
				}
			case model.PositionMaintain:
				maintained++
			case model.PositionUnsure:
				unsure++
				// Deliberately not collected: see the all-unsure branch below.
			}
		}
		switch {
		case refuted == responded && len(positions) == responded && responded == panel:
			it.Status = model.VerdictRejected
			it.Verdict = model.VerdictRejected
			it.VerdictDetail = "refuted by every reviewer on the panel: " + evidence
			dropped++
			logf("refutation: %s dropped -- all %d reviewer(s) on the panel refuted it", it.ID, panel)
		case refuted == len(positions) && len(positions) == responded && responded < panel:
			// Every reviewer that answered refuted it, but some never answered. Their
			// silence is not a refutation, and treating it as one would hand a single
			// refuter the deletion whenever the rest of the panel fails -- which is
			// something the code under review can influence. Kept and flagged.
			it.Contested = true
			it.ContestedBy = refuters
			contested++
			logf("refutation: %s refuted by all %d reviewer(s) that answered, but %d of %d never answered -- kept",
				it.ID, refuted, panel-responded, panel)
		case refuted == len(positions) && len(positions) < responded:
			// Everyone who spoke about this finding refuted it, but not everyone spoke.
			// Silence is not agreement: the reviewers that omitted the id may never have
			// looked at it. Kept and flagged rather than deleted on a partial count.
			it.Contested = true
			it.ContestedBy = refuters
			contested++
			logf("refutation: %s refuted by %d of %d responding reviewer(s), the rest did not say -- kept",
				it.ID, refuted, responded)
		case refuted > 0:
			// Kept, but the disagreement is recorded: a reader deciding what to do about
			// this finding should know somebody who looked did not believe it.
			it.Contested = true
			it.ContestedBy = refuters
			contested++
			logf("refutation: %s contested (%d refute, %d maintain, %d unsure) -- kept", it.ID, refuted, maintained, unsure)
		case unsure == len(positions):
			// Marked for the reader, but NOT recorded as doubt the judge may build on.
			//
			// ContestedBy is what authorizes a judge to drop a merge-blocking finding, and
			// the documented rule is that removing a blocker takes a second agent's
			// EVIDENCE -- refuteWith requires it for a refutation and deliberately does not
			// for an unsure, whose whole meaning is "I could not decide". Letting that
			// satisfy the gate would make the strongest control here reachable by the
			// weakest possible statement: an agent with nothing to say could unlock the
			// deletion of the finding that blocks the merge.
			it.Contested = true
			contested++
			logf("refutation: %s uncertain -- no reviewer could decide it from the evidence; recorded for the reader, not as grounds to drop it", it.ID)
		}
	}
	return dropped, contested
}

// runJudge is the last filter before a review is published: one read-only agent
// sees what survived refutation and decides which findings are worth a human's
// attention.
//
// It is a separate role from the coder, deliberately. The coder already makes this
// judgment in a fix run -- fix.md asks it to reject what is correct but not worth
// fixing -- but the coder is can_edit, and a review- config's whole promise is that
// it never invokes anything that can modify the target. Same judgment, different
// hands.
//
// It became necessary rather than nice-to-have when the verdict gained a hard
// severity gate: if one `high` blocks a merge, something must filter severity
// BEFORE the gate, or the noisiest reviewer decides the outcome. Measured, that
// reviewer is real -- one panel member's highs were rejected half the time.
//
// Fails closed. A judge that dies leaves every finding standing AND prevents an
// approval, because a review whose filter never ran has not been filtered, and
// approving on that basis would be trusting a step that did not happen.
func (o *Orchestrator) runJudge(ctx context.Context, rec *model.RoundRecord, material string) (ran bool) {
	j := o.cfg.Roles.Judge
	if j.Agent == "" || j.Prompt == "" || len(rec.Issues) == 0 {
		return true // not configured is not a failure; there is nothing to fail closed about
	}
	// A judge that was STOPPED is the opposite: it was configured, it did not run, and
	// saying otherwise here approves a review whose filter never happened. This
	// line read `|| ctx.Err() != nil` above the early return and so reported a
	// killed judge as having filtered -- caught by the panel reviewing this branch.
	if ctx.Err() != nil {
		o.logf("judge: interrupted before it could run; every finding stands and the review cannot approve")
		return false
	}
	open := 0
	for _, it := range rec.Issues {
		if it.StatusOrDefault() != model.VerdictRejected {
			open++
		}
	}
	if open == 0 {
		return true
	}
	o.phase("JUDGE  %s weighing %d surviving finding(s)", j.Agent, open)

	d := prompt.JudgeData{
		Mode:           o.cfg.Target.Mode,
		Path:           o.cfg.Target.Path,
		Round:          rec.Round,
		ModeGuidance:   prompt.ModeGuidance(o.cfg.Target.Mode),
		Target:         material,
		Canonical:      prompt.FormatCanonical(undecidedIssues(rec)),
		OutputContract: prompt.JudgeContract,
	}
	d.Prelude = prompt.FormatPrelude(prompt.ReviewData{
		Mode: d.Mode, Path: d.Path, Round: d.Round, ModeGuidance: d.ModeGuidance, Target: d.Target,
	})
	tmpl, ok := o.templates[j.Prompt]
	if !ok || tmpl == nil {
		// Defensive: New loads this template, so reaching here means a hand-built
		// Orchestrator. Rendering a nil template PANICS, which would take down a run
		// over a filter -- fail the filter closed instead.
		o.logf("WARNING: judge prompt %q was never loaded; every finding stands and the review cannot approve", j.Prompt)
		return false
	}
	text, err := prompt.Render(tmpl, d)
	if err != nil {
		o.logf("WARNING: judge: render failed (%v); every finding stands and the review cannot approve", err)
		return false
	}
	res := o.runAgent(ctx, "judge: "+j.Agent, "judge", j.Agent, j.Prompt, rec.Round, text)
	var out model.JudgeOutput
	parseErr := res.Err
	if parseErr == nil {
		parseErr = agent.ExtractJSON(res.Stdout, "review", &out)
	}
	rec.Steps = append(rec.Steps, stepStat("judge", j.Agent, j.Prompt, len(text), res, parseErr != nil))
	// Persisted for the same reason as the refutation: a dropped finding's reason
	// reaches the summary, but a KEPT one's does not, and neither does anything
	// about how the judge weighed the rest.
	o.logStep("judge", j.Agent, j.Prompt, rec.Round, parseErr == nil, out,
		logstore.RenderJudgeMD(j.Agent, rec.Round, out.Verdicts, parseErr), res, "")
	if parseErr != nil {
		o.logf("WARNING: judge failed (%v); every finding stands and the review cannot approve", parseErr)
		o.endPhase("JUDGE  did not finish; every finding stands")
		return false
	}
	kept, dropped := applyJudgment(rec, out.Verdicts, o.cfg.Review.BlockAt, j.Agent, o.logf)
	o.endPhase("JUDGE  %d kept, %d dropped", kept, dropped)
	return true
}

// undecidedIssues are the findings still standing after refutation -- the only
// ones worth a judge's session.
func undecidedIssues(rec *model.RoundRecord) []model.Issue {
	out := make([]model.Issue, 0, len(rec.Issues))
	for _, it := range rec.Issues {
		if it.StatusOrDefault() != model.VerdictRejected {
			out = append(out, it)
		}
	}
	return out
}

// applyJudgment records the arbiter's decisions.
//
// A finding it does not mention is KEPT. The judge is a filter, and a filter that
// removes what it forgot to consider is not a filter -- it is a leak whose size
// depends on how long the reply was.
//
// A drop with no reason is also kept. The reason is the only thing standing between
// a filter and an unaccountable one, and an empty string cannot be disagreed with.
//
// Above all, ONE judge cannot delete a BLOCKING finding on its own word. The judge
// prompt already forbids dropping a high or critical for cost or unlikelihood --
// only for being wrong -- but a prompt is not a control: this single agent reads the
// same untrusted material the panel read, so a pull request carrying injection text
// aimed at the judge could retract exactly the finding that was about to block it,
// and -post-verdict would then approve it. Code cannot check whether the judge
// PROVED a finding wrong, so it checks the one mechanical fact that a second,
// independent agent produced: the refutation round recorded disagreement about it.
// A blocking finding every responding reviewer stood behind survives the judge, and
// the verdict blocks -- the wrong answer costs a human one paragraph, and the other
// wrong answer is an approval nobody gave. Two agents must now agree to remove a
// blocker, one of them in a round that never sees the judge's reasoning.
//
// INDEPENDENT is the operative word, and a boolean could not carry it. The shipped
// pr configuration makes the same agent both a panel refuter and the judge, so
// injection text could buy both halves of the corroboration from one agent: refute
// the finding in the panel round, drop it in the judging round. The doubt therefore
// has to come from an agent that is not the judge -- which is why the refutation
// round records WHO doubted each finding and not merely that somebody did.
func applyJudgment(rec *model.RoundRecord, verdicts []model.JudgeVerdict, blockAt, judge string, logf func(string, ...any)) (kept, dropped int) {
	if blockAt == "" {
		blockAt = model.DefaultBlockAt
	}
	floor := model.SeverityRank(blockAt)
	decided := map[string]model.JudgeVerdict{}
	// Two verdicts on one finding is not a judgment, it is a contract violation, and
	// whichever one arrived last is not more authoritative than the other. Keeping
	// the finding is the same fail-closed reading applied to an unknown verdict
	// below: a judge that answered "keep and drop" did not decide to drop.
	ambiguous := map[string]bool{}
	for _, v := range verdicts {
		if !model.ValidJudgeVerdict(v.Verdict) {
			logf("WARNING: judge returned an unknown verdict %q on %s; the finding stands", v.Verdict, v.Issue)
			continue
		}
		v.Verdict = strings.ToLower(strings.TrimSpace(v.Verdict))
		if _, seen := decided[v.Issue]; seen {
			if !ambiguous[v.Issue] {
				logf("WARNING: judge returned more than one verdict on %s; one finding gets one answer -- kept", v.Issue)
			}
			ambiguous[v.Issue] = true
			continue
		}
		decided[v.Issue] = v
	}
	for i := range rec.Issues {
		it := &rec.Issues[i]
		v, ok := decided[it.ID]
		if !ok || ambiguous[it.ID] || v.Verdict != model.JudgeDrop || it.StatusOrDefault() == model.VerdictRejected {
			continue
		}
		reason := strings.TrimSpace(v.Reason)
		if reason == "" {
			logf("WARNING: judge dropped %s with no reason; a drop nobody can argue with is not a judgment -- kept", it.ID)
			continue
		}
		if model.SeverityRank(it.Severity) <= floor && !doubtedByOther(*it, judge) {
			// Recorded as contested for the reader: the judge is a reviewer of the panel's
			// work, and its dissent is evidence even when it is not authority. Its own
			// name is NOT added to ContestedBy: that list is what the gate above reads,
			// and writing to it here would let this round's refusal authorize the next
			// round's drop on nothing but the judge's repeated opinion.
			it.Contested = true
			logf("judge: %s is %s and no other reviewer refuted it, so one judge may not drop it alone -- kept and contested (%s)",
				it.ID, model.NormalizeSeverity(it.Severity), firstLineOf(reason))
			continue
		}
		it.Status = model.VerdictRejected
		it.Verdict = model.VerdictRejected
		it.VerdictDetail = "judged not worth reporting: " + v.Reason
		dropped++
		logf("judge: %s dropped -- %s", it.ID, firstLineOf(reason))
	}
	for _, it := range rec.Issues {
		if it.StatusOrDefault() != model.VerdictRejected {
			kept++
		}
	}
	return kept, dropped
}

// doubtedByOther reports whether the refutation round recorded doubt about this
// finding from an agent other than the judge.
//
// A judge that refuted the finding itself corroborates nothing: it is the same
// model, reading the same untrusted material, reachable by the same injection. An
// issue carrying Contested from an older summary with no names recorded also fails
// this test, which is the safe direction -- the finding stands.
func doubtedByOther(it model.Issue, judge string) bool {
	for _, name := range it.ContestedBy {
		if name != judge {
			return true
		}
	}
	return false
}

// fixSubject and fixTitle name a coder session's block. One issue per session is
// the rule (see runFixSessions), so the usual answer is that issue's id and title;
// the plural forms exist only for a config that batches, and for the closing
// round's correction pass.
func fixSubject(active []model.Issue) string {
	switch len(active) {
	case 0:
		return "(nothing)"
	case 1:
		return active[0].ID
	default:
		return fmt.Sprintf("%d issues", len(active))
	}
}

func fixTitle(active []model.Issue) string {
	if len(active) != 1 {
		return ""
	}
	return active[0].Title
}

func firstLineOf(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// posterFor is forge.PosterFor behind a variable so a test can observe what would
// be published without a network call. The seam is here rather than in forge
// because what needs asserting is the ORCHESTRATOR's choice of bytes and event,
// not the CLI invocation.
var posterFor = forge.PosterFor

// readerFor is forge.ReaderFor behind a variable, for the same reason.
var readerFor = forge.ReaderFor

// postReview publishes the rendered review on the pull request.
//
// Two assertions, because there are two risk levels and collapsing them would
// price the smaller one at the larger one's rate. -post publishes the findings as
// a COMMENT: the review becomes visible and nothing is spent -- no approval given,
// nobody's merge blocked. -post-verdict additionally lets it carry the verdict,
// which either approves somebody's change or formally requests changes on it, and
// that is a social act as much as a technical one.
//
// Both are command-line flags and neither can come from YAML: the first bundle on
// the search path belongs to the target, so a config key here would let reviewed
// code arrange for a review to be posted under the operator's identity.
//
// A failed post is REPORTED, never swallowed. Reads in this package fail soft
// because a missing datum only weakens the evidence; a write that the operator
// asked for and did not get is the opposite -- silence there would tell them the
// review is on the pull request when it is not.
//
// "Reported" means the RUN fails, which is why this returns an error instead of
// only logging one. A log line is not observable to automation: the caller went on
// to record a normal review-only termination, so `fixpoint review-pr -post` exited
// 0 on an expired token, a 5xx or a timeout, and on the head-moved mismatch
// confirmApproval exists to raise -- the case where an approval IS on the pull
// request and a human has to dismiss it. Two exceptions stay warnings: a run with
// no pull request to post to never had anywhere to publish, and an interruption
// already terminates the run non-zero on its own.
func (o *Orchestrator) postReview(ctx context.Context, sum *model.RunSummary, body string) error {
	if !o.cfg.Review.Post {
		return nil
	}
	if o.cfg.Target.Mode != config.ModePR || o.cfg.Target.PR <= 0 {
		o.logf("WARNING: -post was given but this run reviews no pull request; the review is in %s", sum.ReviewBody)
		return nil
	}
	if ctx.Err() != nil {
		o.logf("WARNING: interrupted before posting; the review is in %s", sum.ReviewBody)
		return nil //nolint:nilerr // the round already records an interruption, which exits non-zero on its own
	}
	p := posterFor(ctx, o.cfg.Target.Path)
	if p == nil {
		// A requested publish that did not happen, so it fails the run like any other
		// -- cmd/fixpoint/postrun.go answers the same situation with exit 1. In pr mode
		// this means the forge remote is not the one PosterFor looks at, not that there
		// is no forge: Prepare already reached the pull request to check it out.
		return fmt.Errorf("-post was given but no GitHub or GitLab remote was recognized; the review is in %s", sum.ReviewBody)
	}
	// forge.EventFor, not a switch here: -post-run publishes the same verdicts from
	// a finished run, and two hand-copied mappings of "which verdict approves
	// somebody's pull request" is one edit away from disagreeing.
	//
	// A summary with no verdict posts as a comment. It should not happen -- the
	// verdict is recorded before this is reached -- but the fallback a missing one
	// deserves is the one that spends nothing.
	outcome := ""
	if sum.Verdict != nil {
		outcome = sum.Verdict.Outcome
	}
	event := forge.EventFor(outcome, o.cfg.Review.PostVerdict)
	// The anchors RECORDED for this review, not a second computation of them.
	// writeReviewBody already stored them, and recomputing here would let the posted
	// review and the replayable one drift apart over the same run.
	inline := make([]forge.InlineComment, 0, len(sum.ReviewInline))
	for _, a := range sum.ReviewInline {
		inline = append(inline, forge.InlineComment{Path: a.Path, Line: a.Line, Body: a.Body})
	}
	// sum.ReviewedHead binds the review to the commit the panel actually read. The
	// poster refuses when the pull request has moved since -- an author who pushes
	// while a review runs must not collect a verdict about the commit before it.
	url, err := p.PostReview(ctx, o.cfg.Target.Path, o.cfg.Target.PR, sum.ReviewedHead, body, event, inline)
	if forge.AnchorRejection(err) {
		// The forge refused the anchors, not the review. Publishing the summary alone
		// is strictly better than publishing nothing: every finding is in it, they
		// just lose their line links. Any other failure is reported below and NOT
		// retried -- it may have been accepted before it failed.
		o.logf("WARNING: %s rejected the inline comments (%v); posting the summary without them", p.Kind(), err)
		url, err = p.PostReview(ctx, o.cfg.Target.Path, o.cfg.Target.PR, sum.ReviewedHead, body, event, nil)
	}
	// Recorded from the URL, not from the absence of an error. A poster that returns
	// a URL has SUBMITTED the review, and one failure arrives after that succeeded:
	// confirmApproval re-reads the head and reports a mismatch as an error on an
	// approval that is already on the pull request. Leaving ReviewPosted empty there
	// would make the summary -- documented as empty when nothing was posted -- deny a
	// review a human has to go and dismiss, and send a later `-post-run` to publish
	// it a second time.
	if err == nil || url != "" {
		sum.ReviewPosted = string(event)
	}
	if err != nil {
		return fmt.Errorf("posting the review to %s failed: %w -- it is written at %s", p.Kind(), err, sum.ReviewBody)
	}
	if url != "" {
		o.logf("review posted to %s as %s: %s", p.Kind(), event, url)
		return nil
	}
	o.logf("review posted to %s as %s", p.Kind(), event)
	return nil
}

// inlineComments anchors each surviving finding to its line, so a reader meets it
// where the code is rather than in a list at the bottom.
//
// Only findings with a file AND a line: a forge has nowhere to hang the rest, and
// including one without a location fails the whole submission. Decided findings
// are left out for the same reason the body omits them -- they are not work
// anybody has to act on.
//
// The text is the finding's own, already sanitized by the body renderer's rules,
// with the severity leading so a reader skimming the Files tab can tell a blocker
// from a note without opening anything.
func inlineComments(rec *model.RoundRecord, diff, signature string) []forge.InlineComment {
	// A forge accepts an anchor only inside the pull request's own diff, and it
	// rejects the WHOLE review when one falls outside -- with a 422 that names
	// nothing. Measured on this project's own pull request: without this filter
	// every anchor was refused, every time, because most findings point at code the
	// change did not touch.
	addressable := forge.AddressableLines(diff)
	out := make([]forge.InlineComment, 0, len(rec.Issues))
	for _, it := range rec.Issues {
		switch it.StatusOrDefault() {
		case model.VerdictFixed, model.VerdictRejected:
			continue
		}
		if it.File == "" || it.Line <= 0 || !addressable[it.File][it.Line] {
			continue
		}
		out = append(out, forge.InlineComment{
			Path: it.File,
			Line: it.Line,
			// Through publishedText, exactly like the body: an inline comment is the
			// second way agent text reaches the pull request, and a finding about a
			// hardcoded credential quotes that credential on a line the diff contains,
			// which is precisely the case that anchors.
			Body: publishedText(review.RenderInline(it, signature)),
		})
	}
	return out
}

// readForgeThreads loads the pull request's OPEN review conversations, so a fix
// run can answer the humans who asked for the change rather than making it
// silently.
//
// Unresolved only, and that filter is the point: a resolved thread is a settled
// question, and handing it to a coder invites it to reopen something a person
// already closed.
//
// Best-effort, like every other read: no gh, no permission, a forge without the
// concept -- all leave the list empty, and a fix run then behaves exactly as it
// did before conversations existed.
func (o *Orchestrator) readForgeThreads(ctx context.Context) {
	if o.cfg.Target.Mode != config.ModePR || o.cfg.Target.PR <= 0 || o.cfg.Loop.ReviewOnly {
		return
	}
	r := readerFor(ctx, o.cfg.Target.Path)
	if r == nil {
		return
	}
	threads, err := r.Threads(ctx, o.cfg.Target.Path, o.cfg.Target.PR)
	if err != nil {
		o.logf("WARNING: could not read the pull request's conversations (%v); the coder will not see them", err)
		return
	}
	// A conversation whose last word is ours is not waiting on anything. Skipping it
	// is what stops a run from answering the same comment again -- and again the run
	// after that -- since a reply does not resolve a thread and every later run reads
	// it as unresolved. The moment a person writes under it, the thread is live again
	// and is read afresh, with the whole exchange including what we said last time.
	live := make([]forge.Thread, 0, len(threads))
	answered := 0
	for _, t := range threads {
		if t.AnsweredByMachine() {
			answered++
			continue
		}
		live = append(live, t)
	}
	o.threads = live
	switch {
	case answered > 0:
		o.logf("%d open conversation(s) on this pull request; %d already carry this tool's answer as the last word and are left alone",
			len(live), answered)
	case len(live) > 0:
		o.logf("%d open conversation(s) on this pull request will be shown to the coder", len(live))
	}
}

// conversations renders the open threads for the coder prompt.
func (o *Orchestrator) conversations() string {
	if len(o.threads) == 0 {
		return ""
	}
	out := make([]prompt.Conversation, 0, len(o.threads))
	for _, t := range o.threads {
		c := prompt.Conversation{ID: t.ID, Path: t.Path, Line: t.Line, Author: t.Author, Body: t.Body}
		for _, m := range t.Comments {
			c.Comments = append(c.Comments, prompt.Comment{Author: m.Author, Body: m.Body})
		}
		out = append(out, c)
	}
	return prompt.FormatConversations(out)
}

// postReplies answers the conversations the coder said its work addressed.
//
// own is the set of threads that commissioned this session's issue -- more than
// one when two comments about the same defect were merged onto it -- and is empty
// for an ordinary panel finding.
//
// Gated on -post, like every other write to a forge: an answer appears under a
// human's comment with the operator's identity on it. Gated on the thread being
// one we actually SHOWED the coder, too -- a reply addressed to an id it invented,
// or to a resolved thread it remembered from somewhere, would post into a
// conversation nobody asked it to touch.
//
// Gated, finally, on the thread still being unanswered, and on a commissioned
// thread being answered by the session that owes it. Every session is a fresh
// agent shown the same list, so without those two rules one human comment
// collects a reply from each of them, all under the operator's name.
//
// Failures are reported per reply and never abort the round: the fix is already
// committed and verified, and losing that over a comment would be the wrong trade.
func (o *Orchestrator) postReplies(ctx context.Context, own map[string]bool, replies []model.FixReply) {
	if len(replies) == 0 || len(o.threads) == 0 {
		return
	}
	if !o.cfg.Review.Post {
		o.logf("%d conversation repl(y|ies) were written but not posted (-post was not given)", len(replies))
		return
	}
	r := readerFor(ctx, o.cfg.Target.Path)
	if r == nil {
		// Said out loud, like answerConversations says its own skip: this was the one
		// path here that discarded a coder's answers with no line at all, so an
		// operator who asked for -post saw nothing happen and nothing explaining it.
		o.logf("WARNING: %d conversation repl(y|ies) were not posted: no GitHub or GitLab remote recognized", len(replies))
		return
	}
	// Signed with the CODER, because that is who is answering. A reply lands in a
	// human's notifications looking exactly like a colleague's, and until now it
	// carried nothing at all to say otherwise -- the one place in this tool where a
	// reader could be misled about who they were talking to. Built once: it is the
	// same run, the same agent, for every thread.
	signature := o.replySignature(o.cfg.Roles.Coder.Agent)
	for _, reply := range replies {
		if !o.threadOpen(reply.Thread) {
			// Covers an invented id, a thread from some other pull request, and one this
			// run has already answered -- a session is shown only the conversations still
			// open, so all three are "not something you were asked about".
			o.logf("WARNING: the coder answered thread %q, which it was not shown as an open conversation; not posted", reply.Thread)
			continue
		}
		if o.commissionedThreads[reply.Thread] && !own[reply.Thread] {
			o.logf("WARNING: conversation %s commissioned a different issue and is answered by that issue's session; this reply is not posted", reply.Thread)
			continue
		}
		// The signature goes AFTER the sanitized agent text and is sanitized
		// separately, exactly as RenderBody places the review's: appending it to text
		// that has not been through the forge funnel yet would let a reply ending in
		// an unclosed HTML comment swallow its own attribution.
		body := publishedText(forge.SanitizeText(reply.Message) + "\n\n" + signature)
		if err := r.Reply(ctx, o.cfg.Target.Path, o.cfg.Target.PR, reply.Thread, body); err != nil {
			// Left open deliberately: nothing was posted, so the conversation is still
			// unanswered and a later session may still answer it.
			o.logf("ERROR: replying to conversation %s failed: %v", reply.Thread, err)
			continue
		}
		o.closeThread(reply.Thread)
		o.logf("replied to conversation %s", reply.Thread)
	}
}

// threadOpen reports whether a conversation is still one this run may answer.
func (o *Orchestrator) threadOpen(id string) bool {
	for _, t := range o.threads {
		if t.ID == id {
			return true
		}
	}
	return false
}

// closeThread takes an answered conversation out of the open list, so no later
// session is shown it and none can post to it a second time.
func (o *Orchestrator) closeThread(id string) {
	for i, t := range o.threads {
		if t.ID == id {
			o.threads = append(o.threads[:i:i], o.threads[i+1:]...)
			return
		}
	}
}
