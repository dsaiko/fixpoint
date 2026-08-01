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
	"github.com/dsaiko/fixpoint/internal/issue"
	"github.com/dsaiko/fixpoint/internal/logstore"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
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
// no such exposure for the same reason the unconditional summary write does not:
// nothing commits after it.
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
	if err := load(cfg.Roles.Coder.Prompt, cfg.Roles.Coder.PromptFile(), prompt.FixData{}); err != nil {
		return nil, err
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
	}
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

// logsDirWithin returns the logs dir as a path relative to the target root if
// it lives inside it, "" otherwise.
func logsDirWithin(logsDir, root string) (string, error) {
	absLogs, err := filepath.Abs(logsDir)
	if err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absLogs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

// checkLogsNotSymlinked verifies that no path component of the in-target logs
// directory is a symlink, so artifacts cannot be redirected outside the lexical
// logs exclusion (o.gitExclude) and swept into a round commit. It is a no-op when
// the logs dir lives outside the target (o.gitExclude unset) or when the logs
// path does not exist yet (it will then be created as a real directory). Called
// after Prepare because a PR checkout can change the path from a plain directory
// into a symlink.
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
			return fmt.Errorf("logs path component %s is a symlink; the prepared target (e.g. a PR) may have redirected the logs directory to bypass the artifact exclusion and leak prompts/outputs into a commit. Point logs.dir outside the target, or remove the symlink", cur)
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
		Path:                o.cfg.Target.Path,
		Strategy:            string(o.cfg.Roles.Review.Strategy),
		ReviewOnly:          o.cfg.Loop.ReviewOnly,
		MaxIterations:       o.cfg.Loop.MaxIterations,
		MaxFindingsPerRound: o.cfg.Loop.MaxFindingsPerRound,
		Overrides:           o.overrides,
		Coder:               o.cfg.Roles.Coder.Agent,
	}
	err := o.run(ctx, sum)
	sum.FinishedAt = time.Now()
	if err != nil {
		err = recordRunError(ctx, sum, err)
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
		// A cancellation during the closing round leaves the loop's own outcome
		// standing, exactly as runFinalRound's own ctx checks do.
		if sum.Termination == "" {
			sum.Termination = model.TermInterrupted
		}
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

func (o *Orchestrator) run(ctx context.Context, sum *model.RunSummary) error {
	o.warnArgModePrompts()
	o.warnInheritedEnv()

	// Enforce the fix-round trust gate; see checkFixTrust for the rationale.
	if err := o.checkFixTrust(); err != nil {
		return err
	}

	if err := o.guardUntrustedGitConfig(ctx); err != nil {
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
		o.logf("preflight: pinging agents...")
		if err := o.Ping(ctx); err != nil {
			return err
		}
	}

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

	cleanStreak := 0
	for round := 1; round <= o.cfg.Loop.MaxIterations; round++ {
		done, err := o.runRound(ctx, round, sum, &cleanStreak)
		if err != nil {
			return err
		}
		if done {
			return o.runFinalRound(ctx, sum)
		}
	}
	sum.Termination = model.TermMaxIterations
	return o.runFinalRound(ctx, sum)
}

// runFinalRound runs the closing round for `final: true` lenses, once, after the
// loop has stopped changing the code. See config.ReviewLens.Final for why a lens
// wants this: it asks its question about the FINISHED tree instead of about work
// the next round will rewrite.
//
// It runs after every normal termination -- converged, all-rejected, and
// max-iterations alike -- because in all three the loop is done editing. It does
// NOT run after an error or an interruption: the tree is then in a state nobody
// vouched for, and the honest move is to stop rather than start new work on it.
//
// The loop's termination is preserved across it. The closing round is extra work
// on an already-decided run, not a new verdict on it, and letting it rewrite
// "converged" into "all-rejected" would report the loop's outcome as something it
// was not.
func (o *Orchestrator) runFinalRound(ctx context.Context, sum *model.RunSummary) error {
	asgs := o.finalAssignments()
	// A canceled context means the operator asked to stop, so the closing round is
	// simply not started. Returning nil rather than the cancellation is deliberate:
	// the LOOP already finished and recorded its own outcome, and skipping optional
	// extra work must not rewrite a run that genuinely converged into a failure.
	if len(asgs) == 0 || o.cfg.Loop.ReviewOnly || ctx.Err() != nil {
		return nil //nolint:nilerr // see above: skipping optional work is not a run failure
	}
	round := len(sum.Rounds) + 1
	o.logf("=== final round (%d): %d closing reviewer(s) over the finished tree ===", round, len(asgs))

	material, err := o.collector.Collect(ctx)
	if err != nil {
		return err
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
	o.logf("final round: %d finding(s), %d advisory, %d reviewer error(s)",
		len(recP.Findings), len(recP.Advisory), len(recP.ReviewErrors))
	// A failed closing reviewer is a failed closing round, checked BEFORE an empty
	// finding set is read as a completed final review. Nothing follows to catch what
	// a dead reviewer never looked at, so "no findings" from a review that did not
	// finish is not the same statement as "no findings", and proceeding to fix the
	// findings the survivors did report would act on a knowingly partial last look.
	if len(recP.ReviewErrors) > 0 {
		return roundReviewErr(recP)
	}
	// Nothing to fix, or the operator stopped us between the review and the fix --
	// in both cases the round is over and the loop's outcome stands unchanged.
	if len(recP.Findings) == 0 || ctx.Err() != nil {
		return nil //nolint:nilerr // a cancellation here leaves the loop's own termination intact
	}

	// The per-round cap still applies: it exists because one coder session has a
	// timeout, and that is no less true here. Nothing follows to pick up the
	// remainder, so anything deferred is reported for a human -- which is the same
	// deal the cap has always offered, minus the promise of a next round.
	o.deferOverCap(recP)
	// allowSalvage=false: a failed coder's partial edits must not be committed here.
	// Salvage is a promise that the next round re-reviews the commit, and there is
	// no next round -- see discardFailedFix.
	if _, err := o.fix(ctx, recP, sum.Rounds[:len(sum.Rounds)-1], false); err != nil {
		return err
	}
	o.logf("final round: coder fixed %d, rejected %d", recP.Fixed, recP.Rejected)
	if recP.Fixed == 0 {
		return nil // nothing to commit; the gate and commit below would no-op
	}
	// Same gate and same commit as any other round: a closing round must not be the
	// one that lands unverified edits.
	return o.verifyAndCommit(ctx, recP, round)
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

// guardUntrustedGitConfig closes the code-execution path opened by the target's
// own repo-local .git/config: per-name content filters (filter.<n>.clean/
// smudge/process), core.sshCommand, and credential helpers are programs git runs
// itself, and gitSafeConfig's -c overrides cannot neutralize them (their names
// are dynamic). They fire whenever a git command touches the worktree -- git-diff
// mode's `git diff` normalizes files through a clean filter even in a review_only
// run, and every fix round's `git add`/`git status` does the same -- so a target
// that ships its own .git/config (an extracted archive, a crafted checkout) can
// run a repo-controlled program with fixpoint's inherited secrets.
//
// The gate applies wherever such a command runs against a target-controlled repo:
// git-diff mode always (its Collect runs `git diff`), directory fix rounds (their
// commits run `git add`/`git status`), and pr mode always -- Prepare's
// `gh pr checkout` writes the worktree, and git runs a configured smudge/process
// filter during checkout. A PR cannot edit .git/config, but it does not have to:
// it supplies the .gitattributes that SELECTS an already-configured filter, the
// worktree script such a filter may point at, and (for git-lfs) the .lfsconfig
// that redirects where the filter talks to. Directory review-only walks the
// filesystem and runs no worktree-touching git command, so it has no such path.
//
// When the operator has asserted no trust we refuse; when trust IS asserted we
// still WARN, because trusted_target/allow_untrusted_fix is documented as
// accepting coder prompt-injection risk and an operator must also learn it accepts
// .git/config-driven code execution (a git-lfs repo they trust, or an enforced
// external sandbox, is the intended use).
func (o *Orchestrator) guardUntrustedGitConfig(ctx context.Context) error {
	touchesGit := o.cfg.Target.Mode == config.ModeGitDiff ||
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
	if len(keys) == 0 {
		return nil
	}
	if !o.cfg.Loop.TrustedTarget && !o.cfg.Loop.AllowUntrustedFix {
		return fmt.Errorf("target %s has repo-local git config that would run repo-controlled programs fixpoint cannot neutralize (%s); git normalizes worktree files through these during diff/add/status/checkout, so this is a code-execution path with fixpoint's inherited environment. Review it under an external sandbox (container/VM), or pass -trusted-target if you trust this checkout", o.cfg.Target.Path, strings.Join(keys, ", "))
	}
	o.logf("WARNING: target %s has repo-local git config that runs repo-controlled programs during git diff/add/status/checkout (%s) which fixpoint cannot neutralize; -trusted-target/-allow-untrusted-fix accepts this code-execution path (with fixpoint's inherited environment) in addition to coder prompt-injection. Review untrusted checkouts (extracted archives, crafted .git) under an external sandbox.", o.cfg.Target.Path, strings.Join(keys, ", "))
	return nil
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
	o.logf("=== round %d/%d ===", round, o.cfg.Loop.MaxIterations)

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
	o.review(ctx, &rec, material, sum.Rounds)
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

	o.logf("round %d: %d finding(s), %d advisory, %d reviewer error(s)",
		round, len(recP.Findings), len(recP.Advisory), len(recP.ReviewErrors))

	if ctx.Err() != nil {
		sum.Termination = model.TermInterrupted
		return true, nil //nolint:nilerr // interruption is a normal termination, not a round error
	}

	// Review-only runs exactly one round, and it only counts as a successful
	// review if every reviewer completed: reporting success over a partial
	// round would let automation trust an incomplete review. The round record
	// stays in the summary either way.
	if o.cfg.Loop.ReviewOnly {
		if len(recP.ReviewErrors) > 0 {
			return false, roundReviewErr(recP)
		}
		sum.Termination = model.TermReviewOnly
		return true, nil
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
	salvaged, err := o.fix(ctx, recP, prior, true)
	if err != nil {
		return false, err
	}
	if salvaged {
		// The coder failed but its partial edits were committed; verdicts are
		// unknown, so skip finalize and let the next round re-review.
		return false, nil
	}
	o.logf("round %d: coder fixed %d, rejected %d", round, recP.Fixed, recP.Rejected)

	return o.finalizeFix(ctx, recP, round, sum)
}

// finalAssignments builds the closing round: every final lens on EVERY agent it
// may use. Breadth rather than rotation, because there is no next round to catch
// what one model missed -- this is the last look at the finished code.
func (o *Orchestrator) finalAssignments() []model.Assignment {
	rv := o.cfg.Roles.Review
	var out []model.Assignment
	for _, l := range rv.Prompts {
		if !l.Final {
			continue
		}
		// A pinned final lens stays pinned: LensAgents returns just that agent.
		for _, a := range rv.LensAgents(l) {
			out = append(out, model.Assignment{Lens: l.Prompt, Agent: a, Advisory: l.Advisory, Pinned: l.Agent != ""})
		}
	}
	return out
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

// reopenFixedIssues withdraws this round's fixed verdicts and returns how many it
// withdrew, for the one case where a recorded fix did not land: the verification
// correction reverted every edit, so there is nothing to commit. The verdict is
// cleared everywhere setIssueVerdict wrote it -- the round's issue, the ledger, and
// every observation that reported it -- and rec.Fixed is reset, so the summary, the
// commit-less round, and the history the next reviewers read agree that the issue is
// still open. An empty verdict renders as UNRESOLVED, which is exactly the
// instruction reviewers need: report it again if it is still there.
//
// Rejections are left alone: a rejection is a judgement, not an edit, so reverting
// edits does not withdraw it.
func (o *Orchestrator) reopenFixedIssues(rec *model.RoundRecord) int {
	reopened := 0
	for i := range rec.Issues {
		it := &rec.Issues[i]
		if it.Verdict != model.VerdictFixed {
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
		reopened++
	}
	rec.Fixed = 0
	return reopened
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
		sum.Termination = model.TermConverged
		return true
	}
	o.logf("clean round (%d/%d consecutive needed)", *cleanStreak, o.cfg.Loop.CleanRoundsToStop)
	return false
}

// finalizeFix reconciles the coder's reported verdicts with the actual working
// tree after a fix, then either commits the round or recognizes a genuine
// all-rejected terminal state. Whether to commit is decided from the tree, not
// from the Fixed count alone: a disagreement is a hard error rather than a
// warning, because fixes with no edits mean nothing was really done, and edits
// under an all-rejected verdict would otherwise leave the "all-rejected"
// success termination sitting on top of an uncommitted, dirty tree.
func (o *Orchestrator) finalizeFix(ctx context.Context, rec *model.RoundRecord, round int, sum *model.RunSummary) (done bool, err error) {
	// A cancellation between the coder finishing and this commit must not leave a
	// committed round (or a dirty tree) sitting on top of a stop request. Recheck
	// immediately before touching the tree and route any cancellation through the
	// same stash-and-interrupt reconciliation the fix path uses.
	if ctx.Err() != nil {
		return false, o.reconcileInterrupt(rec.Round, ctx.Err()) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
	}
	clean, err := o.collector.GitClean(ctx, o.gitExclude...)
	if err != nil {
		// Cancellation can land WHILE GitClean runs, after the check above passed
		// (a TOCTOU window). Route the resulting context error through the same
		// interrupt reconciliation, on a fresh context, so a tree left dirty by a
		// canceled coder is stashed (or flagged errInterruptedTreeDirty) rather
		// than returned bare and softened into a clean interruption by Run.
		if ctx.Err() != nil {
			return false, o.reconcileInterrupt(rec.Round, err) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
		}
		return false, err
	}
	if rec.Fixed > 0 && clean {
		return false, fmt.Errorf("round %d: coder reported %d fix(es) but left the working tree unchanged", round, rec.Fixed)
	}
	if rec.Fixed == 0 && !clean {
		return false, o.reconcileRejectedEdits(ctx, round)
	}

	if rec.Fixed > 0 {
		return false, o.verifyAndCommit(ctx, rec, round)
	}

	// rec.Fixed == 0 with a clean tree: a genuine all-rejected round. It is a
	// successful terminal state only when the round's review was complete; with
	// reviewer errors it may just mean the failed reviewers' findings never
	// arrived.
	if len(rec.ReviewErrors) > 0 {
		return false, roundReviewErr(rec)
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

// warnArgModePrompts logs a warning for every agent this run will use that is
// configured prompt_via: arg. Such an agent receives the full prompt on its
// process argument list, where the embedded review material (in git-diff/pr
// mode the diff -- which often IS the credential under review) and any secret a
// reviewer quotes into a finding are world-readable via ps / /proc for the
// invocation's lifetime, bypassing the 0600 log permissions and on-disk
// redaction. The exposure is not silent; stdin is the secure default.
// warnInheritedEnv reports agents that opted out of environment filtering. Worth a
// warning rather than silence: the filtered default is what keeps an exported
// secret out of a prompt-injectable reviewer, and inherit_all gives that up for
// one agent without changing anything visible in the run's output.
func (o *Orchestrator) warnInheritedEnv() {
	for _, n := range o.activeAgentNames() {
		if !o.cfg.Agents[n].Env.InheritAll {
			continue
		}
		o.logf("WARNING: agent %q sets env.inherit_all -- it receives fixpoint's ENTIRE environment, so every exported secret (cloud credentials, database passwords, tokens for other services) is readable by that agent and can be quoted into a finding. Declare the variables it actually needs under env.pass instead", n)
	}
}

func (o *Orchestrator) warnArgModePrompts() {
	for _, n := range o.activeAgentNames() {
		if o.cfg.Agents[n].PromptVia != config.PromptViaArg {
			continue
		}
		o.logf("WARNING: agent %q uses prompt_via: arg -- the full prompt (reviewed material, plus any secret quoted into a finding) is placed on the process argument list and is readable by other local users via ps / /proc for the run's duration, bypassing the 0600 log permissions and on-disk redaction; prefer prompt_via: stdin for material that may contain secrets", n)
	}
}

// activeAgentNames returns the sorted, distinct set of agents this run will
// invoke: the coder (unless review-only) plus every active reviewer agent.
// Ping and warnArgModePrompts share it so "the agents this run uses" has a
// single definition. Sorted so per-agent log lines and failure lists are
// deterministic run-to-run, matching review's index-ordered collection.
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
		case rv.Strategy == config.StrategyAll:
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

// heartbeatEvery is how often a still-running agent invocation emits a
// progress line, so a long silent stretch is visibly alive rather than
// ambiguous (thinking-heavy coders can run 20+ minutes without output).
const heartbeatEvery = 5 * time.Minute

// runAgent persists the prompt (before invoking, so a killed or hung agent's
// input is still inspectable), invokes the agent with a heartbeat, and
// returns the result. label prefixes the heartbeat lines, e.g.
// "fix: claude-coder".
func (o *Orchestrator) runAgent(ctx context.Context, label, role, agentName, lensName string, round int, text string) agent.Result {
	if err := o.logs.Prompt(role, agentName, lensName, round, text); err != nil {
		o.logf("WARNING: writing %s prompt log: %v", role, err)
	}
	start := time.Now()
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(heartbeatEvery)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				o.logf("%s still running (%s elapsed)", label, time.Since(start).Round(time.Second))
			}
		}
	}()
	res := agent.Run(ctx, o.cfg.Agents[agentName], text, o.cfg.Target.Path)
	close(done)
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
		Mode:           o.cfg.Target.Mode,
		Path:           o.cfg.Target.Path,
		Round:          round,
		ModeGuidance:   prompt.ModeGuidance(o.cfg.Target.Mode),
		Target:         material,
		History:        prompt.FormatHistory(history),
		OutputContract: prompt.ReviewContract,
	}
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

	// Per-step logs (IDs are assigned later; md/json here carry raw findings).
	findings := toFindings(out.Findings, asg, lensName, nil, 0)
	md := logstore.RenderReviewMD(asg.Agent, lensName, round, findings, parseErr)
	o.logStep("review", asg.Agent, lensName, round, parseErr == nil, out, md, res)
	o.logf("%s done (%d findings, %s, output %s)", label, len(out.Findings), res.Duration.Round(time.Second), logstore.SizeDesc(len(res.Stdout)+len(res.Stderr)))
	return out.Findings, stepStat("review", asg.Agent, lensName, len(text), res, parseErr != nil), parseErr
}

// logStep writes one agent invocation's step logs, downgrading a log-write
// failure to a WARNING (a run must not abort because a log file could not be
// written). parsed is persisted only when ok, so a failed step never writes a
// JSON payload that would read as a clean result.
func (o *Orchestrator) logStep(role, agentName, promptName string, round int, ok bool, out any, md string, res agent.Result) {
	var parsed any
	if ok {
		parsed = out
	}
	if err := o.logs.Step(role, agentName, promptName, round, parsed, md, res.Raw(o.cfg.Agents[agentName].Argv())); err != nil {
		o.logf("WARNING: writing %s log: %v", role, err)
	}
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
			IssueID:     f.Issue,
			Category:    f.Category,
			Severity:    f.Severity,
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
func (o *Orchestrator) fix(ctx context.Context, rec *model.RoundRecord, history []model.RoundRecord, allowSalvage bool) (salvaged bool, err error) {
	coder := o.cfg.Roles.Coder
	promptName := config.LensName(coder.Prompt)
	label := "fix: " + coder.Agent
	active := activeIssues(rec)
	d := prompt.FixData{
		Mode:           o.cfg.Target.Mode,
		Path:           o.cfg.Target.Path,
		Round:          rec.Round,
		Findings:       prompt.FormatIssues(active),
		History:        prompt.FormatHistory(history),
		OutputContract: prompt.FixContract,
	}
	text, err := prompt.Render(o.templates[coder.Prompt], d)
	if err != nil {
		return false, err
	}
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
		runErr = o.applyVerdicts(rec, out.Results)
	}

	rec.Steps = append(rec.Steps, stepStat("fix", coder.Agent, promptName, len(text), res, runErr != nil))
	md := logstore.RenderFixMD(coder.Agent, rec.Round, rec.Findings, out.Notes, runErr)
	o.logStep("fix", coder.Agent, promptName, rec.Round, runErr == nil, out, md, res)
	o.logf("%s done (%s, output %s)", label, res.Duration.Round(time.Second), logstore.SizeDesc(len(res.Stdout)+len(res.Stderr)))
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
		return false, o.reconcileInterrupt(rec.Round, interrupted) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
	}
	if runErr != nil {
		if !allowSalvage {
			return false, o.discardFailedFix(ctx, rec.Round, runErr)
		}
		return o.salvagePartialFix(ctx, rec, active, runErr)
	}
	return false, nil
}

// discardFailedFix handles a failed coder in a round nothing follows -- the
// closing round. salvagePartialFix commits a failed coder's partial edits
// specifically because the NEXT round re-reviews them; after the closing round
// there is no next round, so the same commit would leave work no reviewer ever
// looked at sitting in the repository under a run that still reports the loop's
// successful termination. The edits are stashed instead (recoverable with
// `git stash`) and the round fails, mirroring reconcileRejectedEdits.
func (o *Orchestrator) discardFailedFix(ctx context.Context, round int, runErr error) error {
	base := fmt.Errorf("closing round %d: coder failed: %w; no round follows to re-review partial work, so its edits were not committed", round, runErr)
	stashMsg := fmt.Sprintf("fixpoint: closing round %d discarded (coder failed)", round)
	stashed, serr := o.collector.StashDirty(ctx, stashMsg, o.gitExclude...)
	ev := model.JournalRoundDiscarded{
		Reason:  model.DiscardFinalCoderFailed,
		Stashed: stashed,
		Error:   base.Error(),
	}
	if serr != nil {
		ev.Error = fmt.Sprintf("%v; tree left dirty: %v", base, serr)
		o.journal(model.EvRoundDiscarded, round, ev)
		return fmt.Errorf("%w; and the modified working tree could not be reconciled (it is left dirty): %w", base, serr)
	}
	o.journal(model.EvRoundDiscarded, round, ev)
	if stashed {
		o.logf("closing round %d: coder failed (%v); its edits were stashed, clean tree restored", round, runErr)
		return fmt.Errorf("%w. The edits were stashed -- inspect or recover them with `git stash pop`", base)
	}
	return base
}

// reconcileInterrupt stashes any working-tree edits left by a canceled coder
// so the stop request never leaves a committed round or a dirty tree behind,
// and returns cause (the interruption) -- or errInterruptedTreeDirty when the
// stash itself fails. It uses a fresh context because the run's context is
// already canceled; target.Collector still bounds each git subprocess with its
// own operation deadline.
func (o *Orchestrator) reconcileInterrupt(round int, cause error) error {
	stashMsg := fmt.Sprintf("fixpoint: interrupted round %d", round)
	stashed, serr := o.collector.StashDirty(context.Background(), stashMsg, o.gitExclude...)
	ev := model.JournalRoundDiscarded{
		Reason:  model.DiscardInterrupted,
		Stashed: stashed,
		Error:   cause.Error(),
	}
	if serr != nil {
		// The tree is LEFT DIRTY. Recording that is the single most useful thing the
		// journal can do here: it is the one exit where fixpoint knowingly leaves
		// unreconciled edits behind, and the summary may not get written at all.
		ev.Error = fmt.Sprintf("%v; tree left dirty: %v", cause, serr)
		o.journal(model.EvRoundDiscarded, round, ev)
		return fmt.Errorf("coder round %d interrupted: %w; and the modified working tree could not be reconciled (it is left dirty): %w: %w", round, cause, serr, errInterruptedTreeDirty)
	}
	o.journal(model.EvRoundDiscarded, round, ev)
	return cause
}

// reconcileFailedCommit stashes the edits that Commit's `git add -A` staged
// before the commit itself failed (non-cancellation), so the tree is never left
// staged and dirty -- which would violate the clean-tree invariant and block the
// next run. It mirrors salvagePartialFix's failed-commit handling: return the
// original commit error to surface, or a combined error if the stash also fails.
func (o *Orchestrator) reconcileFailedCommit(ctx context.Context, round int, commitErr error) error {
	stashMsg := fmt.Sprintf("fixpoint: recovered edits from failed commit in round %d", round)
	stashed, serr := o.collector.StashDirty(ctx, stashMsg, o.gitExclude...)
	ev := model.JournalRoundDiscarded{
		Reason:  model.DiscardCommitFailed,
		Stashed: stashed,
		Error:   commitErr.Error(),
	}
	if serr != nil {
		ev.Error = fmt.Sprintf("%v; tree left dirty: %v", commitErr, serr)
		o.journal(model.EvRoundDiscarded, round, ev)
		return fmt.Errorf("round %d commit failed: %w; and the modified working tree could not be reconciled (it is left dirty): %w", round, commitErr, serr)
	}
	o.journal(model.EvRoundDiscarded, round, ev)
	if stashed {
		o.logf("round %d: commit failed (%v); edits stashed (recover with `git stash`), clean tree restored", round, commitErr)
	}
	return commitErr
}

// reconcileRejectedEdits handles a round whose coder rejected every issue yet
// left edits in the working tree. No verdict claims those edits, so there is
// nothing to commit -- but returning with them still in the tree would violate
// the clean-tree invariant the next run's ensureCleanTree enforces, and block
// that run until an operator cleans up by hand. So they are stashed through the
// same path every other abnormal exit uses, and stay recoverable.
func (o *Orchestrator) reconcileRejectedEdits(ctx context.Context, round int) error {
	base := fmt.Errorf("round %d: coder rejected every finding yet modified the working tree; refusing to commit edits no verdict accounts for", round)
	stashMsg := fmt.Sprintf("fixpoint: round %d discarded (rejected verdicts with edits)", round)
	stashed, serr := o.collector.StashDirty(ctx, stashMsg, o.gitExclude...)
	ev := model.JournalRoundDiscarded{
		Reason:  model.DiscardRejectedWithEdits,
		Stashed: stashed,
		Error:   base.Error(),
	}
	if serr != nil {
		ev.Error = fmt.Sprintf("%v; tree left dirty: %v", base, serr)
		o.journal(model.EvRoundDiscarded, round, ev)
		return fmt.Errorf("%w; and the modified working tree could not be reconciled (it is left dirty): %w", base, serr)
	}
	o.journal(model.EvRoundDiscarded, round, ev)
	if stashed {
		o.logf("round %d: edits left by an all-rejected round were stashed; clean tree restored", round)
		return fmt.Errorf("%w. The edits were stashed -- inspect or recover them with `git stash pop`", base)
	}
	return base
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
func (o *Orchestrator) salvagePartialFix(ctx context.Context, rec *model.RoundRecord, active []model.Issue, runErr error) (salvaged bool, err error) {
	if blocking, verr := o.verifyPass(ctx, rec, model.VerifyAttemptSalvage); verr != nil {
		return false, verr
	} else if len(blocking) > 0 {
		return false, o.rejectUnverifiedSalvage(ctx, rec, runErr, blocking)
	}
	header := fmt.Sprintf("fixpoint: round %d (partial, coder failed)", rec.Round)
	var body strings.Builder
	fmt.Fprintf(&body, "Coder failed before reporting verdicts: %v\n\nIssues it was working on:\n", runErr)
	for _, it := range active {
		fmt.Fprintf(&body, "- [%s] (%s, %s) %s\n", it.ID, it.Category, it.Severity, it.Title)
	}
	// Redact reviewer-authored finding text before it lands in the pushed
	// commit message, mirroring the normal round commit and the logstore.
	sha, cerr := o.collector.Commit(ctx, header, agent.RedactSecrets(body.String()), o.gitExclude...)
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
		stashed, serr := o.collector.StashDirty(ctx, stashMsg, o.gitExclude...)
		ev := model.JournalRoundDiscarded{
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
func (o *Orchestrator) applyVerdicts(rec *model.RoundRecord, results []model.FixResult) error {
	// Issues that were never handed to the coder -- deferred by the cap, or rejected
	// in an earlier round -- neither expect nor accept a verdict.
	index := map[string]int{}
	expected := 0
	for i := range rec.Issues {
		if !coderWork(rec.Issues[i]) {
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
			if coderWork(it) && !seen[it.ID] {
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

// commit stages and commits the coder's changes with the configured header
// and a body composed from the coder's own verdicts.
func (o *Orchestrator) commit(ctx context.Context, rec *model.RoundRecord) (string, error) {
	header := strings.NewReplacer(
		"{round}", strconv.Itoa(rec.Round),
		"{fixed}", strconv.Itoa(rec.Fixed),
		"{rejected}", strconv.Itoa(rec.Rejected),
	).Replace(o.cfg.Loop.CommitMessage)

	var body strings.Builder
	writeVerdictSection(&body, "Fixed", rec.Findings, model.VerdictFixed)
	if rec.Rejected > 0 {
		body.WriteString("\n")
		writeVerdictSection(&body, "Rejected", rec.Findings, model.VerdictRejected)
	}
	// The body embeds reviewer-authored titles and coder-authored verdict
	// details, which may quote the very secret under review. Unlike the logs,
	// the commit is meant to be pushed and shared, so redact it too (the
	// logstore path does the same for its on-disk artifacts).
	return o.collector.Commit(ctx, header, agent.RedactSecrets(body.String()), o.gitExclude...)
}

// writeVerdictSection writes a "<title>:" header followed by one line per
// finding carrying the given verdict, so the fixed and rejected sections of a
// commit body share a single formatting site.
func writeVerdictSection(b *strings.Builder, title string, findings []model.Finding, verdict string) {
	b.WriteString(title + ":\n")
	for _, f := range findings {
		if f.Verdict == verdict {
			fmt.Fprintf(b, "- [%s] %s — %s\n", f.Category, f.Title, f.VerdictDetail)
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
func (o *Orchestrator) captureVerifyBaseline(ctx context.Context) {
	if !o.cfg.Verify.Enabled() || o.cfg.Loop.ReviewOnly {
		return
	}
	o.logf("verify: capturing baseline (%d command(s))", len(o.cfg.Verify.Commands))
	rep := verify.Run(ctx, o.cfg.Verify, o.cfg.Target.Path, o.verifyEnv)
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
		o.logf("verify baseline: all checks pass")
		return
	}
	// Worth stating plainly: under no_regressions these checks are permitted to
	// keep failing, which is easy to misread later as fixpoint ignoring them.
	o.logf("verify baseline: %s", rep.Summary())
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
func (o *Orchestrator) verifyRound(ctx context.Context, rec *model.RoundRecord) (blocked bool, err error) {
	blocking, err := o.verifyPass(ctx, rec, model.VerifyAttemptInitial)
	if err != nil || len(blocking) == 0 {
		return false, err
	}

	o.logf("round %d verify: %d blocking failure(s); asking the coder to correct them", rec.Round, len(blocking))
	if err := o.fixVerification(ctx, rec, blocking); err != nil {
		return false, err
	}
	rec.VerifyRetried = true
	blocking, err = o.verifyPass(ctx, rec, model.VerifyAttemptCorrection)
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
// attempt is one of model.VerifyAttempt*: it names the occasion for the journal and
// selects the log line's human suffix, so the machine and human records of one gate
// run cannot drift apart.
func (o *Orchestrator) verifyPass(ctx context.Context, rec *model.RoundRecord, attempt string) ([]verify.Result, error) {
	if !o.cfg.Verify.Enabled() {
		return nil, nil
	}
	rep := verify.Run(ctx, o.cfg.Verify, o.cfg.Target.Path, o.verifyEnv)
	rec.Verify = rep.Results
	o.logf("round %d verify%s: %s", rec.Round, verifyAttemptLabel[attempt], rep.Summary())
	blocking := rep.Blocking(o.cfg.Verify.Policy, o.verifyBaseline)
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
	failed := make([]string, 0, len(rec.Verify))
	for _, r := range rec.Verify {
		if !r.Optional && !r.Passed {
			failed = append(failed, r.Name)
		}
	}
	stashed, serr := o.collector.StashDirty(ctx, fmt.Sprintf("fixpoint: round %d discarded (verification failed)", rec.Round), o.gitExclude...)
	base := fmt.Errorf("round %d: verification failed after a correction attempt (%s); the round was not committed",
		rec.Round, strings.Join(failed, ", "))
	o.journal(model.EvRoundDiscarded, rec.Round, model.JournalRoundDiscarded{
		Reason:  model.DiscardVerifyFailed,
		Stashed: stashed,
		Checks:  failed,
	})
	if serr != nil {
		return fmt.Errorf("%w; and the working tree could not be restored: %w", base, serr)
	}
	if stashed {
		return fmt.Errorf("%w. The coder's edits were stashed -- recover them with `git stash pop`", base)
	}
	return base
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
	base := fmt.Errorf("coder round %d failed (%w) and the partial work it left does not pass verification (%s); it was not committed",
		rec.Round, runErr, strings.Join(names, ", "))
	stashed, serr := o.collector.StashDirty(ctx, fmt.Sprintf("fixpoint: unverified partial work from failed round %d", rec.Round), o.gitExclude...)
	o.journal(model.EvRoundDiscarded, rec.Round, model.JournalRoundDiscarded{
		Reason:  model.DiscardSalvageFailed,
		Stashed: stashed,
		Checks:  names,
		Error:   runErr.Error(),
	})
	if serr != nil {
		return fmt.Errorf("%w; and the working tree could not be restored: %w", base, serr)
	}
	if stashed {
		return fmt.Errorf("%w. The edits were stashed -- inspect or recover them with `git stash pop`", base)
	}
	return base
}

// fixVerification invokes the coder a second time with the verification failures
// in hand. It reuses the coder's own prompt template so the instructions,
// contract, and history stay identical; only the extra Verification block differs.
func (o *Orchestrator) fixVerification(ctx context.Context, rec *model.RoundRecord, blocking []verify.Result) error {
	coder := o.cfg.Roles.Coder
	a := o.cfg.Agents[coder.Agent]
	d := prompt.FixData{
		Mode:  o.cfg.Target.Mode,
		Path:  o.cfg.Target.Path,
		Round: rec.Round,
		// This runs AFTER the round's verdicts are applied, so activeIssues here is the
		// work whose edits are in the tree -- the issues the coder reported fixing. The
		// ones it rejected changed nothing and cannot be why a check now fails.
		Findings:       prompt.FormatIssues(activeIssues(rec)),
		Verification:   verify.FormatForCoder(blocking),
		OutputContract: prompt.FixContract,
	}
	text, err := prompt.Render(o.templates[coder.Prompt], d)
	if err != nil {
		return fmt.Errorf("render coder prompt for the verification correction: %w", err)
	}
	// Logged under its own prompt name so the correction attempt is a distinct,
	// inspectable artifact rather than overwriting the round's first fix log.
	const lens = "fix-verify"
	if err := o.logs.Prompt("fix", coder.Agent, lens, rec.Round, text); err != nil {
		o.logf("WARNING: failed to write the verification-correction prompt: %v", err)
	}
	res := agent.Run(ctx, a, text, o.cfg.Target.Path)
	rec.Steps = append(rec.Steps, stepStat("fix", coder.Agent, lens, len(text), res, res.Err != nil))
	var out model.FixOutput
	parseErr := agent.ExtractJSON(res.Stdout, "fix", &out)
	if werr := o.logs.Step("fix", coder.Agent, lens, rec.Round, out, prompt.FormatFindings(nil), res.Raw(a.Argv())); werr != nil {
		o.logf("WARNING: failed to write the verification-correction log: %v", werr)
	}
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

// verifyAndCommit runs the deterministic gate over the coder's edits and, if they
// hold up, commits the round. Split out of finalizeFix so the round's terminal
// bookkeeping stays readable next to a phase that has three distinct failure
// exits: blocked by the gate, reverted by the correction, and commit failure.
func (o *Orchestrator) verifyAndCommit(ctx context.Context, rec *model.RoundRecord, round int) error {
	// Verify BEFORE committing. The coder's own report is a model's claim about
	// its work; this is the only check in the loop that is not. A round whose
	// edits break the project must not become a commit that later rounds build
	// on -- and must not count toward convergence.
	if blocked, err := o.verifyRound(ctx, rec); err != nil {
		return err
	} else if blocked {
		return o.rejectUnverifiedRound(ctx, rec)
	}
	// A correction attempt can revert the round's edits entirely, leaving a clean
	// tree after verification passed. Commit would then no-op and the round would
	// report fixes that never landed, so say so rather than passing silently.
	if postClean, cerr := o.collector.GitClean(ctx, o.gitExclude...); cerr == nil && postClean {
		// The findings only STAY open if they are put back: the coder's fixed verdicts
		// are already recorded on the round, mirrored onto its observations and applied
		// to the ledger, so leaving them would make the summary and the next round's
		// reviewer history claim fixes that no commit contains -- contradicting the log
		// line below and letting the run converge on work that was reverted.
		reopened := o.reopenFixedIssues(rec)
		o.logf("round %d: nothing left to commit -- the verification correction reverted the round's edits; %d finding(s) reopened", rec.Round, reopened)
		rec.CoderError = "verification correction reverted the round's edits; nothing was committed"
		return nil
	}
	sha, err := o.commit(ctx, rec)
	if err != nil {
		// Same TOCTOU window: cancellation can interrupt the commit's git add /
		// commit after staging the coder's edits. Reconcile so the staged tree
		// is stashed instead of left staged under a softened interruption.
		if ctx.Err() != nil {
			return o.reconcileInterrupt(rec.Round, err) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
		}
		// A non-cancellation commit failure (e.g. commit signing failed) leaves
		// the coder's edits staged by Commit's `git add -A`. Left as-is, the
		// dirty/staged tree violates the clean-tree invariant and the next run
		// refuses to start. Stash the edits so the tree is clean again.
		return o.reconcileFailedCommit(ctx, rec.Round, err)
	}
	rec.CommitSHA = sha
	if sha != "" {
		o.logf("round %d committed: %s", round, shortSHA(sha))
	}
	o.journal(model.EvRoundCommitted, rec.Round, model.JournalRoundCommitted{
		SHA:   sha,
		Fixed: rec.Fixed,
	})
	return nil
}
