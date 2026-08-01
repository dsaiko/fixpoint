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
		CommitPolicy:        o.cfg.Loop.CommitPolicy,
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

	// Where the run's commits begin, for a per_run squash at the end. Captured on the
	// same pristine tree as the verification baseline: everything after this point is
	// the run's own work and nothing of the operator's.
	runBase, err := o.collector.HeadSHA(ctx)
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

// finishRun runs the closing phase and then applies a per_run squash over
// everything the run committed, loop rounds and closing passes together.
//
// The squash goes last for the same reason the closing phase runs at all: it is the
// point where the run is finally done editing. Squashing earlier would collapse
// commits the closing passes then build on.
func (o *Orchestrator) finishRun(ctx context.Context, sum *model.RunSummary, runBase string) error {
	if err := o.runFinalPhase(ctx, sum); err != nil {
		return err
	}
	return o.squashRun(ctx, sum, runBase)
}

// squashRun collapses the whole run into one commit for commit_policy: per_run.
//
// It runs only on a normal termination, which is all that reaches it: an error or an
// interruption returns before finishRun, and rightly so -- rewriting history over a
// tree nobody vouched for would destroy the per-fix commits an operator needs in
// order to see how far the run actually got.
func (o *Orchestrator) squashRun(ctx context.Context, sum *model.RunSummary, runBase string) error {
	if o.cfg.Loop.CommitPolicy != config.CommitPerRun {
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
	for i := range sum.Rounds {
		agg.Fixed += sum.Rounds[i].Fixed
		agg.Rejected += sum.Rounds[i].Rejected
		agg.Findings = append(agg.Findings, sum.Rounds[i].Findings...)
	}
	sha, err := o.squashTo(ctx, agg, runBase, agg.Round)
	if err != nil {
		return err
	}
	// The per-fix commits this replaced are gone, so point the last round at what now
	// holds its work -- otherwise the summary cites a SHA no longer in the history.
	for i := range sum.Rounds {
		sum.Rounds[i].Commits = nil
	}
	if n := len(sum.Rounds); n > 0 {
		sum.Rounds[n-1].CommitSHA = sha
		sum.Rounds[n-1].Commits = []string{sha}
	}
	o.logf("run: squashed %d round(s) of fixes into %s", agg.Round, shortSHA(sha))
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
	for _, it := range batch {
		if ctx.Err() != nil {
			// The operator asked to stop between sessions, so do not spend another one.
			// Nothing is half-done at this point -- every fix so far is verified and
			// committed and the tree is clean -- so there is nothing to stash, and Run
			// softens a bare cancellation into a clean interruption.
			return false, committed, ctx.Err()
		}
		fixedBefore := rec.Fixed
		salvaged, err := o.fix(ctx, rec, history, allowSalvage, []model.Issue{it})
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
			if !clean {
				if err := o.reconcileRejectedSession(ctx, rec, it); err != nil {
					return false, committed, err
				}
			}
			continue
		}
		if clean {
			return false, committed, o.withdrawUncommittedFix(rec, it.ID,
				fmt.Errorf("round %d: coder reported a fix for %s but left the working tree unchanged", rec.Round, it.ID))
		}
		did, err := o.verifyAndCommitFix(ctx, rec, it)
		if err != nil {
			return false, committed, o.withdrawUncommittedFix(rec, it.ID, err)
		}
		if did {
			committed++
		}
	}
	return false, committed, nil
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
	if clean, cerr := o.collector.GitClean(ctx, o.gitExclude...); cerr == nil && clean {
		o.reopenFixedIssue(rec, it.ID)
		o.logf("round %d: %s left nothing to commit -- the verification correction reverted it; the finding is reopened", rec.Round, it.ID)
		return false, nil
	}
	// The subject embeds the reviewer-authored title, so it gets the same two
	// treatments the body does. Flattening first: `git commit -m` takes the
	// argument literally, so a title carrying newlines would forge extra lines
	// -- trailers like Signed-off-by: among them -- into the message.
	header := strings.NewReplacer(
		"{round}", strconv.Itoa(rec.Round),
		"{fixed}", "1",
		"{rejected}", "0",
		"{issue}", it.ID,
		"{title}", strings.Join(strings.Fields(it.Title), " "),
	).Replace(o.fixCommitMessage())
	var body strings.Builder
	fmt.Fprintf(&body, "%s (%s, %s) %s\n", it.ID, it.Category, it.Severity, it.Loc())
	if it.VerdictDetail != "" {
		body.WriteString("\n" + it.VerdictDetail + "\n")
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
// waste either: pass 2 sees the tests pass 1 wrote, so it reports what is genuinely
// still missing rather than working from a list computed before the code changed --
// which is also what makes the phase terminate on its own.
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
// was not.
func (o *Orchestrator) runFinalPhase(ctx context.Context, sum *model.RunSummary) error {
	actionable, advisory := o.finalAssignments()
	// A canceled context means the operator asked to stop, so the closing phase is
	// simply not started. Returning nil rather than the cancellation is deliberate:
	// the LOOP already finished and recorded its own outcome, and skipping optional
	// extra work must not rewrite a run that genuinely converged into a failure.
	if len(actionable)+len(advisory) == 0 || o.cfg.Loop.ReviewOnly || ctx.Err() != nil {
		return nil //nolint:nilerr // see above: skipping optional work is not a run failure
	}
	if err := o.runFinalFixPasses(ctx, sum, actionable); err != nil {
		return err
	}
	// The report goes last, over the tree as it finally stands. Skipped on a
	// cancellation: the operator asked to stop, and a report is not worth a session.
	if len(advisory) > 0 && ctx.Err() == nil {
		if _, err := o.runFinalPass(ctx, sum, advisory, "report"); err != nil {
			return err
		}
	}
	return nil
}

// runFinalFixPasses repeats the actionable half of the closing round until it has
// nothing left to fix, bounded by the same safety valve as the loop rather than a
// knob of its own -- the phase stops on its own as soon as a pass finds nothing or
// fixes nothing, so the bound only catches a lens that never runs out of things to
// say.
func (o *Orchestrator) runFinalFixPasses(ctx context.Context, sum *model.RunSummary, actionable []model.Assignment) error {
	if len(actionable) == 0 {
		return nil
	}
	for pass := 1; pass <= o.cfg.Loop.MaxIterations; pass++ {
		done, err := o.runFinalPass(ctx, sum, actionable, fmt.Sprintf("pass %d", pass))
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	o.warnFinalPhaseCapped(sum)
	return nil
}

// warnFinalPhaseCapped reports what the closing phase ran out of passes before
// fixing. Staying quiet is exactly the failure this phase exists to avoid: a run
// that fixed some of its coverage gaps and said nothing about the rest reads as
// complete.
func (o *Orchestrator) warnFinalPhaseCapped(sum *model.RunSummary) {
	open := 0
	if n := len(sum.Rounds); n > 0 {
		for _, it := range sum.Rounds[n-1].Issues {
			if it.StatusOrDefault() != model.VerdictFixed {
				open++
			}
		}
	}
	o.logf("WARNING: the closing round stopped after %d pass(es) (loop.max_iterations) with %d issue(s) still open; they are recorded in the summary and were NOT fixed",
		o.cfg.Loop.MaxIterations, open)
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
	o.logf("=== closing round, %s (round %d): %d reviewer(s) over the finished tree ===", label, round, len(asgs))

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
	o.logf("closing %s: %d finding(s), %d advisory, %d reviewer error(s)",
		label, len(recP.Findings), len(recP.Advisory), len(recP.ReviewErrors))
	// A failed closing reviewer is a failed closing pass, checked BEFORE an empty
	// finding set is read as a completed final review. Nothing follows to catch what
	// a dead reviewer never looked at, so "no findings" from a review that did not
	// finish is not the same statement as "no findings", and proceeding to fix the
	// findings the survivors did report would act on a knowingly partial last look.
	// Repeating the phase does not soften this: a later pass re-reviews the same
	// tree, so it would inherit the same blind spot rather than close it.
	if len(recP.ReviewErrors) > 0 {
		return false, roundReviewErr(recP)
	}
	// Nothing to fix, or the operator stopped us between the review and the fix --
	// in both cases the phase is over and the loop's outcome stands unchanged.
	if len(recP.Findings) == 0 || ctx.Err() != nil {
		return true, nil //nolint:nilerr // a cancellation here leaves the loop's own termination intact
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
		// it would re-review identical code and get an identical answer. The exception
		// is a capped pass, where the coder was only ever handed the active issues: if
		// it rejected all of them, the deferred remainder still never reached it, and
		// stopping here would drop exactly the work this repeating phase exists to
		// pick up -- silently, since warnFinalPhaseCapped only fires on the pass cap.
		// The re-review is not wasted on them either: deferral aging promotes what was
		// skipped, so the next pass hands it over first. Same reasoning as
		// finalizeRound's deferred check in the loop.
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
		return fmt.Errorf("target %s has repo-supplied git config (.git/config, a file it includes, or .git/config.worktree) that would run repo-controlled programs fixpoint cannot neutralize (%s); git normalizes worktree files through these during diff/add/status/checkout, so this is a code-execution path with fixpoint's inherited environment. Review it under an external sandbox (container/VM), or pass -trusted-target if you trust this checkout", o.cfg.Target.Path, strings.Join(keys, ", "))
	}
	o.logf("WARNING: target %s has repo-supplied git config (.git/config, a file it includes, or .git/config.worktree) that runs repo-controlled programs during git diff/add/status/checkout (%s) which fixpoint cannot neutralize; -trusted-target/-allow-untrusted-fix accepts this code-execution path (with fixpoint's inherited environment) in addition to coder prompt-injection. Review untrusted checkouts (extracted archives, crafted .git) under an external sandbox.", o.cfg.Target.Path, strings.Join(keys, ", "))
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
		// unknown, so skip finalize and let the next round re-review.
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
	// Awaited, not just signaled: a tick already inside logf would otherwise
	// still be printing after runAgent returns, interleaving a "still running"
	// line with whatever the caller logs next (or with the end-of-run table).
	var hb sync.WaitGroup
	hb.Add(1)
	go func() {
		defer hb.Done()
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
func (o *Orchestrator) fix(ctx context.Context, rec *model.RoundRecord, history []model.RoundRecord, allowSalvage bool, batch []model.Issue) (salvaged bool, err error) {
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
		runErr = o.applyVerdicts(rec, out.Results, active)
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
func (o *Orchestrator) verifyRound(ctx context.Context, rec *model.RoundRecord, fixed model.Issue) (blocked bool, err error) {
	blocking, err := o.verifyPass(ctx, rec, model.VerifyAttemptInitial)
	if err != nil || len(blocking) == 0 {
		return false, err
	}

	o.logf("round %d verify: %d blocking failure(s) after fixing %s; asking the coder to correct them", rec.Round, len(blocking), fixed.ID)
	if err := o.fixVerification(ctx, rec, blocking, fixed); err != nil {
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
	// Recorded next to the results they were derived from: the baseline is not part
	// of the summary, so nothing downstream could otherwise tell a round the gate
	// cleared from one it stopped.
	rec.VerifyBlocking = blockingNames(blocking)
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
	return o.discardEdits(ctx, discard{
		round:  rec.Round,
		reason: model.DiscardVerifyFailed,
		base: fmt.Errorf("round %d: verification failed after a correction attempt (%s); the round was not committed",
			rec.Round, strings.Join(failed, ", ")),
		stashMsg: fmt.Sprintf("fixpoint: round %d discarded (verification failed)", rec.Round),
		checks:   failed,
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
