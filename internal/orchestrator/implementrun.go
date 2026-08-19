package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/create"
	"github.com/dsaiko/fixpoint/internal/gitenv"
	"github.com/dsaiko/fixpoint/internal/implement"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
	"github.com/dsaiko/fixpoint/internal/target"
	"github.com/dsaiko/fixpoint/internal/verify"
)

// The implement-design pipeline: one read-only planner session turns the
// design document into an ordered task plan, and the coder builds it into a
// fresh repository -- one task, one session, one gated commit. The mechanics
// that are not agent scheduling (plan validation, censuses, the invariant
// snapshot, the scaffold) live in internal/implement; the pipeline is
// specified in docs/design/DESIGN.md and every section reference below is to
// that document.

// Task outcome vocabulary (§5.4). Strings, because they land in trailers,
// journals and the scoreboard verbatim.
const (
	outcomeImplemented = "implemented"
	outcomeSatisfied   = "already_satisfied"
	outcomeBlocked     = "blocked"
	outcomeFailed      = "failed"
	outcomeSkipped     = "skipped"
	// outcomeUnreached is for a task the loop never got to -- the deadline, the
	// breaker, the disk bound or Ctrl-C stopped the run first. Not a judgment
	// about the work: it is what §1's "recorded in the run's report as not
	// built, with a reason" means for the tail of an interrupted plan.
	outcomeUnreached = "unreached"
	// outcomeCarried is an outcome this run ADOPTED from the repository's own
	// history rather than produced (§5.4). A resumed run reports the whole plan,
	// and the reader has to be able to tell which half it actually did.
	outcomeCarried = "carried"
)

// carriedReason renders a carried record for the report: the outcome it holds
// plus whatever reason the original run recorded, so a resumed run's summary
// explains a failed or blocked task it never ran itself.
func carriedReason(r implement.Record) string {
	if r.Reason != "" {
		return fmt.Sprintf("%s (%s), recorded by an earlier run", r.Outcome, r.Reason)
	}
	return r.Outcome + ", recorded by an earlier run"
}

// taskReport is the coder's structured reply.
type taskReport struct {
	Task         string   `json:"task"`
	Status       string   `json:"status"`
	CoveredBy    []string `json:"covered_by"`
	BlockedOn    []string `json:"blocked_on"`
	Notes        string   `json:"notes"`
	FilesTouched []string `json:"files_touched"`
}

// implementPrep is everything the phases share.
type implementPrep struct {
	out         string // the write-target, claimed by SCAFFOLD
	designBytes []byte
	designSHA   string
	designPath  string // as the operator named it
	outline     implement.Outline
	// plannerLabel is what Provenance.Planner records: the configured agent, or
	// "operator-supplied" with the file's digest when -plan was used.
	plannerLabel string
	// coverageChecked is §4.2 rule 6's state for this run: false only when the
	// operator waived it with -no-coverage-check. Carried rather than re-derived
	// so the plan artifact, PLAN.md and the log cannot disagree about whether the
	// check ran.
	coverageChecked bool
	material        string // the design, collected and fenced for the planner
	profile         string // the effective-gate digest
	gateCommands    []string
	col             *target.Collector // over out; valid after SCAFFOLD
	// gitEnv is the ONE hardened environment every git command against the
	// write-target runs with, and git is the read path's handle onto it. Both
	// the Collector and the implement package are handed this same slice: the
	// two used to build their own, and the read path's was weaker (review run
	// 20260813-180828, i12).
	gitEnv []string
	git    implement.Git
	// deadline is when max_run_duration expires, set at BUILD. Zero before
	// then, which is what the backoff reads as "no deadline to overrun".
	deadline     time.Time
	bootstrapSHA string
	// attributed is the last commit FIXPOINT made -- the bootstrap, a task
	// commit, an outcome marker. Anything past it at HEAD is a session's own,
	// unverified and unattributed, and must not survive an interrupted run.
	attributed string
	baseline   implement.RepoState
	// outcomes is every processed task's recorded outcome, by id. Shared state
	// rather than a buildPhase local because the coverage check needs it:
	// "already_satisfied, covered by T04" is only true if T04 was IMPLEMENTED.
	outcomes map[string]string
	// release drops the write-target's repository lock; nil until SCAFFOLD has
	// created the repository there is anything to lock.
	release func()
}

// attemptBase is what the repository and its ignored tree looked like at §5.2
// step 2, carried through the attempt so both the session and the GATE can be
// compared against the same before-picture.
type attemptBase struct {
	head    string
	repo    implement.RepoState
	ignored map[string]implement.IgnoredStat
}

// errRunStop is a repository-invariant failure: the run cannot reason about
// the tree any longer, so it stops rather than failing one task (§8).
type runStopError struct{ why string }

func (e runStopError) Error() string { return "repository invariant failed: " + e.why }

// runPipeline dispatches the non-loop pipelines; handled=false means the run
// is loop-shaped and run() continues.
//
// The logs-symlink guard runs here, BEFORE either pipeline writes its first
// artifact: both write snapshots, prompts and raw agent output into an
// artifact root that sits beside an assignment the target may own, and
// checking only on the way out (the shape the first review of this code found,
// run 20260813-124710) reports the redirect after those bytes have already
// been written wherever the symlink pointed.
func (o *Orchestrator) runPipeline(ctx context.Context, sum *model.RunSummary) (bool, error) {
	if !o.cfg.IsCreate() && !o.cfg.IsImplement() {
		return false, nil
	}
	if err := o.checkLogsNotSymlinked(); err != nil {
		return true, err
	}
	if o.cfg.IsCreate() {
		return true, o.runCreate(ctx, sum)
	}
	return true, o.runImplement(ctx, sum)
}

func (o *Orchestrator) runImplement(ctx context.Context, sum *model.RunSummary) error {
	sum.Implement = true
	started := time.Now()

	// The trust gate, exactly as a fix run enforces it: the coder edits files
	// with permission checks disabled, steered by the design document.
	if err := o.checkFixTrust(); err != nil {
		return err
	}

	rec := model.RoundRecord{Round: 1}
	defer func() {
		// Whatever happened, the sessions are billed.
		sum.Rounds = append(sum.Rounds, rec)
	}()

	// -continue replaces PREFLIGHT, PLAN and SCAFFOLD: the project exists, its
	// plan and design are committed inside it, and what was already built is
	// read back out of its own history.
	if o.cfg.Implement.Continue != "" {
		prep, pl, resume, err := o.resumePhase(ctx, sum)
		if err != nil {
			return err
		}
		defer func() {
			o.cleanUpInterrupted(ctx, prep)
			if prep.release != nil {
				prep.release()
			}
		}()
		return o.buildPhase(ctx, sum, &rec, prep, pl, started, resume)
	}

	prep, err := o.prepareImplement(ctx)
	if err != nil {
		return err
	}
	pl, err := o.planPhase(ctx, &rec, prep)
	if err != nil {
		return err
	}
	// -plan-only stops here, having spent one planner session and claimed
	// nothing. The plan is on disk at the run root; §7.4's composition is to read
	// it, edit it, and hand it back with -plan.
	if o.cfg.Implement.PlanOnly {
		sum.Termination = model.TermPlanned
		o.logf("PLAN-ONLY: %d task(s) validated and written to %s; no directory was created and no coder session was spent",
			len(pl.Tasks), filepath.Join(o.logs.RunDir(), "plan.json"))
		o.logf("review it, then build it with:  -plan %s -out <fresh-dir>", filepath.Join(o.logs.RunDir(), "plan.json"))
		return nil
	}
	if err := o.scaffoldPhase(ctx, sum, prep, pl); err != nil {
		return err
	}
	defer func() {
		// Leave the project the way §8 promises Ctrl-C leaves it: "every
		// committed task stands and was gated; nothing further is committed",
		// and the repository is consistent as-is. On cancellation the in-flight
		// attempt's edits are still in the tree -- checkInvariants fails on the
		// dead context and returns before the discard ever runs -- so the
		// cleanup happens HERE, on a fresh context, exactly as the collector's
		// own commit path restores its index on the way out (review run
		// 20260813-161029). Without it a canceled run left a dirty tree that
		// -continue then refused, and the report said only "interrupted".
		o.cleanUpInterrupted(ctx, prep)
		if prep.release != nil {
			prep.release()
		}
	}()
	return o.buildPhase(ctx, sum, &rec, prep, pl, started, implement.Resume{})
}

// readCommittedArtifacts loads the plan and the design from the BOOTSTRAP
// commit, not the worktree: they are control artifacts nothing may edit, and
// reading the committed blobs means a resumed run cannot be steered by a file
// someone changed between runs. It also re-checks the plan/design pairing the
// bootstrap recorded, so a swapped DESIGN.md is caught here rather than becoming
// the document the run reports it implemented.
func (o *Orchestrator) readCommittedArtifacts(ctx context.Context, p *implementPrep, dir, bootstrap string) (implement.Plan, error) {
	var pl implement.Plan
	planJSON, err := p.git.ShowBlob(ctx, dir, bootstrap, "PLAN.json")
	if err != nil {
		return pl, fmt.Errorf("-continue %s: %w", dir, err)
	}
	if err := json.Unmarshal([]byte(planJSON), &pl); err != nil {
		return pl, fmt.Errorf("-continue %s: the committed PLAN.json does not parse: %w", dir, err)
	}
	designBytes, err := p.git.ShowBlob(ctx, dir, bootstrap, "DESIGN.md")
	if err != nil {
		return pl, fmt.Errorf("-continue %s: %w", dir, err)
	}
	p.designBytes = []byte(designBytes)
	p.designSHA = implement.DigestBytes(p.designBytes)
	p.designPath = filepath.Join(dir, "DESIGN.md")
	if pl.Provenance != nil && pl.Provenance.DesignSHA256 != "" && pl.Provenance.DesignSHA256 != p.designSHA {
		return pl, fmt.Errorf("-continue %s: the committed PLAN.json was written for design %.12s and the committed DESIGN.md hashes to %.12s", dir, pl.Provenance.DesignSHA256, p.designSHA)
	}
	return pl, nil
}

// guardResumedRepository applies the two git guards a resumed project needs
// before the first worktree-touching command. A fresh run gets them from run()'s
// preflight against the reviewed target; a resume must run them against the
// PROJECT, which sat outside fixpoint's lock between runs and which anyone could
// have edited (review run 20260818-234734, two reviewers).
//
// Neither is trust-gated, for the reasons the preflight versions give: a
// core.worktree redirect makes every later git command -- including the coder's
// commits -- read and write a directory the operator never named, and a
// .git/config shipping a filter, diff driver or credential helper is a program
// git runs with fixpoint's environment. The repository-invariant baseline cannot
// substitute for either, because on a resume the baseline is TAKEN FROM this
// tree: tampering done between runs would simply become the accepted state.
func (o *Orchestrator) guardResumedRepository(ctx context.Context, col *target.Collector, dir string) error {
	root, err := col.WorktreeOutOfScope(ctx)
	if err != nil {
		return err
	}
	if root != "" {
		return fmt.Errorf("-continue %s: the repository's work tree points at %s (core.worktree or GIT_WORK_TREE), so every git command -- including the coder's commits -- would read and write outside the project; remove the redirect", dir, root)
	}
	keys, err := col.UnsafeConfig(ctx)
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return fmt.Errorf("-continue %s: the repository's own git config defines %s -- programs git would run with fixpoint's environment; a project left unattended between runs does not get to configure the tool that resumes it. Remove the key(s) and rerun", dir, strings.Join(keys, ", "))
	}
	return nil
}

// resumePhase is §5.5's re-entry: open a project fixpoint built, read what it
// says about itself, and prove it is the one this configuration describes.
//
// Nothing here comes from the original run's artifacts, which may be on another
// machine or long deleted -- the plan, the design and every recorded outcome are
// read out of the repository's own commits. That is what "self-describing"
// means, and it is why §5.4 writes a commit for every processed task even when
// the task changed no files.
//
// The trust boundary is RE-ESTABLISHED rather than inherited. Between runs the
// tree was outside fixpoint's lock and anyone could have touched it, so the
// first-contact checks a fresh directory does not need are run here: the
// repository-invariant baseline is taken now, over this tree, and the lock is
// taken before any of it is read.
func (o *Orchestrator) resumePhase(ctx context.Context, sum *model.RunSummary) (p *implementPrep, pl implement.Plan, resume implement.Resume, retErr error) {
	dir := o.cfg.Implement.Continue
	o.phase("RESUME  %s", dir)

	p = &implementPrep{out: dir, outcomes: map[string]string{}}
	p.gitEnv = gitenv.NoOperatorConfig(o.verifyEnv)
	p.git = implement.NewGit(p.gitEnv)

	if st, err := os.Stat(filepath.Join(dir, ".git")); err != nil || !st.IsDir() {
		return p, pl, resume, fmt.Errorf("-continue %s: not a git repository; -continue resumes a project fixpoint itself built", dir)
	}
	if err := o.preflightPing(ctx); err != nil {
		return p, pl, resume, err
	}

	p.col = target.New(config.Target{Mode: config.ModeDirectory, Path: dir})
	p.col.UseGitEnv(p.gitEnv)
	release, err := p.col.LockRepo(ctx)
	if err != nil {
		return p, pl, resume, fmt.Errorf("-continue %s: %w", dir, err)
	}
	p.release = release
	// Every refusal below hands the lock back on the way out. The caller only
	// registers its release defer AFTER this function succeeds, so an error
	// return that kept the lock would hold the repository -- and leak the
	// descriptor -- for the life of the process; the test suite drives several
	// runs per process, and a second LockRepo then refuses against ourselves
	// (review run 20260818-234734). The named-return check keeps this correct
	// for every return statement, including ones added later.
	defer func() {
		if retErr != nil && p.release != nil {
			p.release()
			p.release = nil
		}
	}()

	if err := o.guardResumedRepository(ctx, p.col, dir); err != nil {
		return p, pl, resume, err
	}

	// Step 0 before anything is believed: a dirty tree means the previous run
	// died mid-attempt, and §5.2 has no way to tell that residue from work.
	// -continue -discard-dirty is §12.7's named answer and is not built, so this
	// says exactly what to do instead.
	if clean, cerr := p.col.GitClean(ctx); cerr != nil {
		return p, pl, resume, cerr
	} else if !clean {
		return p, pl, resume, fmt.Errorf("-continue %s: the working tree is dirty, so a previous run stopped mid-attempt and its residue cannot be told from work; inspect it and `git stash` or `git checkout .` before resuming", dir)
	}

	history, err := p.git.ReadHistory(ctx, dir)
	if err != nil {
		return p, pl, resume, err
	}
	p.bootstrapSHA = history.Bootstrap
	// The interruption anchor, same as a fresh run's: everything at HEAD was
	// admitted from the repository, so HEAD is the last commit this run vouches
	// for. Left empty, cleanUpInterrupted skipped its reset for the whole first
	// resumed task -- a coder that committed on its own and was then interrupted
	// left an ungated commit at HEAD, and the NEXT -continue would have admitted
	// it as a recorded outcome if the session had written itself the trailers
	// (review run 20260818-234734, three reviewers independently).
	p.attributed = history.Head

	if pl, err = o.readCommittedArtifacts(ctx, p, dir, history.Bootstrap); err != nil {
		return p, pl, resume, err
	}

	p.gateCommands = implement.RenderCommands(o.cfg.Verify)
	p.profile = implement.VerifyProfile(p.gateCommands, string(o.cfg.Verify.Policy), o.cfg.Verify.Timeout.Std(), o.cfg.Implement.GateGenerated)
	p.plannerLabel = "carried (-continue)"
	p.coverageChecked = false // the check ran, or did not, on the run that planned

	// The committed plan is held to §4.2 like every other way a plan enters the
	// pipeline. It was the ONE unvalidated entry point (review run
	// 20260818-234734, two reviewers): the admission evidence is trailer text in
	// a repository that sat outside fixpoint's lock, and the rules skipped are
	// load-bearing beyond plan quality -- rule 2's id pattern exists because
	// task ids land in artifact filenames, and rule 4 refuses the paths a misled
	// plan could point a session at. Coverage is unchecked here because the
	// check ran, or did not, on the run that planned; everything else binds.
	if err := implement.Validate(pl, o.planRules(p, 0)); err != nil {
		return p, pl, resume, fmt.Errorf("-continue %s: the committed PLAN.json does not pass validation: %w -- the repository's plan has been altered since fixpoint wrote it", dir, err)
	}

	if resume, err = implement.AdmitResume(history, pl, p.designSHA, p.profile); err != nil {
		return p, pl, resume, fmt.Errorf("-continue %s: %w", dir, err)
	}

	// The baseline is taken over THIS tree, now: whatever the previous run
	// snapshotted describes a repository nobody has watched since.
	if p.baseline, err = p.git.SnapshotRepoState(ctx, dir, "main"); err != nil {
		return p, pl, resume, err
	}

	sum.Deliverable = dir
	sum.Coder = o.cfg.Roles.Coder.Agent
	o.journal("resume_admitted", 1, map[string]any{
		"project": dir, "bootstrap": history.Bootstrap,
		"carried": resume.Index, "planned": len(pl.Tasks),
	})
	if resume.Index >= len(pl.Tasks) {
		o.endPhase("RESUME  every task in the plan is already recorded; nothing to do")
	} else {
		o.endPhase("RESUME  %d of %d task(s) carried; re-entering at %s",
			resume.Index, len(pl.Tasks), pl.Tasks[resume.Index].ID)
	}
	return p, pl, resume, nil
}

// prepareImplement is every refusal that must fire before a session is paid
// for: the write-target check, the design snapshot, and the outline.
func (o *Orchestrator) prepareImplement(ctx context.Context) (*implementPrep, error) {
	p := &implementPrep{out: o.cfg.Create.Out, outcomes: map[string]string{}}
	// Built here, before anything touches the write-target, so no code path can
	// reach git without it: the credential-stripped base with the operator's
	// global and system git config switched off. Safe over this repository in a
	// way it would not be over the operator's, because Init writes the
	// repository's identity locally -- and load-bearing, because a
	// `.gitattributes` the coder writes names a filter whose definition would
	// otherwise come from the operator's global file. NoOperatorConfig returns a
	// fresh slice; o.verifyEnv is shared with every gate run and must not be
	// appended to in place.
	p.gitEnv = gitenv.NoOperatorConfig(o.verifyEnv)
	p.git = implement.NewGit(p.gitEnv)
	// -plan-only claims nothing, so it needs no write-target and must not test
	// one: the point of the mode is to decide whether to spend the run at all,
	// and demanding the directory up front would make an operator name a place
	// for a project they have not agreed to build yet.
	if o.cfg.Implement.PlanOnly {
		return o.prepareDesign(ctx, p)
	}
	if p.out == "" {
		return nil, errors.New("implement: pass -out <directory>; the pipeline builds a new project and must be told where")
	}
	if st, err := os.Stat(p.out); err == nil && st != nil {
		return nil, fmt.Errorf("implement: %s exists; the write-target must be a fresh directory", p.out)
	}
	// -out must not lie inside ANY git repository (§7.3): the enclosing repo
	// would record the new project as a gitlink or absorb its files. Scoped to
	// no particular repository on purpose.
	for dir := filepath.Dir(p.out); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return nil, fmt.Errorf("implement: %s is inside a git repository (%s); the new project must not nest in an existing history", p.out, dir)
		}
		if strings.Contains(filepath.Base(dir), ".fixpoint") {
			return nil, fmt.Errorf("implement: %s is under a .fixpoint artifact directory", p.out)
		}
		if parent := filepath.Dir(dir); parent == dir {
			break
		}
	}
	if free, ok := implement.DiskFree(filepath.Dir(p.out)); ok && free < o.cfg.Implement.MinFreeDisk.Int64() {
		return nil, fmt.Errorf("implement: %d bytes free under %s, below implement.min_free_disk (%s)", free, filepath.Dir(p.out), o.cfg.Implement.MinFreeDisk)
	}

	return o.prepareDesign(ctx, p)
}

// prepareDesign is the half of preflight that is about the DOCUMENT rather than
// the write-target: the agent ping, the snapshot, the design hash and the
// outline. -plan-only runs exactly this much.
func (o *Orchestrator) prepareDesign(ctx context.Context, p *implementPrep) (*implementPrep, error) {
	if o.cfg.Target.Document == "" {
		return nil, errors.New("implement: the target must be a design document file (-target DESIGN.md)")
	}
	p.designPath = filepath.Join(o.cfg.Target.Path, o.cfg.Target.Document)

	// The pipeline with the largest bill was the only one that never checked its
	// agents were reachable (review run 20260813-180828, i5): runPipeline
	// dispatches implement before run()'s own preflight, and runImplement had no
	// copy of it. An expired login or a quota blackout was therefore discovered
	// only after the planner session was paid for and the write-target had been
	// created and locked -- the coder then died twice as infrastructure, the
	// breaker tripped, and the operator was left holding a scaffolded, half-built
	// repository. Placed after the cheap local refusals, before the first spend.
	if err := o.preflightPing(ctx); err != nil {
		return nil, err
	}

	// SNAPSHOT: every phase reads the same bytes (§4.3); the run is long and
	// the odds someone edits the design mid-run are real.
	scratch, err := o.logs.ScratchDir("design")
	if err != nil {
		return nil, err
	}
	// The snapshot budget is the planner's prompt budget: a design that cannot
	// fit the planner's prompt fails HERE, before any session is paid for.
	budget := int64(o.cfg.Agents[o.cfg.Roles.Planner.Agent].PromptBudget)
	if budget <= 0 {
		budget = 1 << 40 // no declared limit: bounded only against a runaway file
	}
	snap, err := create.Snapshot(p.designPath, scratch, nil, budget)
	if err != nil {
		return nil, err
	}
	o.logf("design snapshot: %s (%d bytes)", snap.Entry, snap.Bytes)
	if p.designBytes, err = os.ReadFile(filepath.Join(snap.Dir, snap.Entry)); err != nil {
		return nil, err
	}
	p.designSHA = implement.DigestBytes(p.designBytes)

	if err := o.readOutline(p); err != nil {
		return nil, err
	}

	if p.material, err = o.snapshotMaterial(ctx, snap); err != nil {
		return nil, err
	}
	p.gateCommands = implement.RenderCommands(o.cfg.Verify)
	p.profile = implement.VerifyProfile(p.gateCommands, string(o.cfg.Verify.Policy), o.cfg.Verify.Timeout.Std(), o.cfg.Implement.GateGenerated)
	return p, nil
}

// planRules is §4.2's rule set for this run's configuration. planOverhead is
// what the run spends before the first task -- a planner's timeout on the fresh
// path, zero on a resume, where no planner runs.
func (o *Orchestrator) planRules(p *implementPrep, planOverhead time.Duration) implement.Rules {
	return implement.Rules{
		MaxTasks:        o.cfg.Implement.MaxTasks,
		MaxFilesPerTask: o.cfg.Implement.MaxFilesPerTask,
		Outline:         p.outline,
		CoverageChecked: p.coverageChecked,
		MaxTaskAttempts: o.cfg.Implement.MaxTaskAttempts,
		SessionTimeout:  o.cfg.Agents[o.cfg.Roles.Coder.Agent].Timeout.Std(),
		GateWorst:       implement.GateWorst(o.cfg.Verify),
		MaxRunDuration:  o.cfg.Implement.MaxRunDuration.Std(),
		CleanCheck:      o.cfg.Implement.CleanCheck,
		PlanOverhead:    planOverhead,
	}
}

// planPhase runs the one read-only planner session and validates its product.
// A plan that fails costs exactly one planner session and zero coder sessions.
func (o *Orchestrator) planPhase(ctx context.Context, rec *model.RoundRecord, p *implementPrep) (implement.Plan, error) {
	o.phase("PLAN  one %s session over the design snapshot", o.cfg.Roles.Planner.Agent)
	var pl implement.Plan

	rules := o.planRules(p, o.cfg.Agents[o.cfg.Roles.Planner.Agent].Timeout.Std()+planOverheadSlack)
	// The cap quoted to the planner and the rule that will judge its answer are
	// now the same arithmetic (Rules.Admitted), so the two cannot disagree.
	admitted := rules.Admitted(o.cfg.Implement.MaxTasks)
	if rules.FitSkipped() {
		o.logf("plan: the fit check is skipped (a session or gate timeout is not available as a number); the deadline is enforced between tasks only")
	} else {
		o.logf("plan: at most %d task(s) fit implement.max_run_duration (%s) -- %s per task at %d attempt(s), clean_check %q",
			admitted, o.cfg.Implement.MaxRunDuration.Std(), rules.PerTaskWorst().Round(time.Minute), rules.MaxTaskAttempts, rules.CleanCheck)
	}
	if admitted == 0 {
		return pl, fmt.Errorf("implement: no plan can fit implement.max_run_duration (%s): one task alone needs %s at %d attempt(s) per task and a %s worst-case gate -- raise max_run_duration, lower max_task_attempts, or tighten verify.timeout",
			o.cfg.Implement.MaxRunDuration.Std(), rules.RunWorst(1).Round(time.Minute), rules.MaxTaskAttempts, rules.GateWorst)
	}

	// The configured agent unless -plan replaces it below.
	p.plannerLabel = implement.PlannerLabel(o.cfg.Roles.Planner.Agent, false, "")

	// -plan: the operator supplies the decomposition and no planner session runs.
	// It is held to EVERY rule the planner's answer is held to -- a hand-edited
	// plan is not more trusted for having been typed, and rule 7's arithmetic is
	// what stops a plan that cannot finish from being started.
	if src := o.cfg.Implement.Plan; src != "" {
		pl, err := o.operatorPlan(src, p, rules)
		if err != nil {
			return pl, err
		}
		return o.stampPlan(pl, p)
	}

	d := prompt.PlanData{
		Design:          p.material,
		OutlineHeadings: p.outline.FormattedHeadings(),
		Gate:            p.gateCommands,
		MaxTasks:        admitted,
		OutputContract:  prompt.PlanContract,
	}
	text, err := prompt.Render(o.templates[o.cfg.Roles.Planner.Prompt], d)
	if err != nil {
		return pl, err
	}
	planner := o.cfg.Roles.Planner.Agent
	res := o.runAgentIn(ctx, o.cfg.Target.Path, "plan: "+planner, "plan", planner, o.cfg.Roles.Planner.Prompt, 1, text)
	perr := res.Err
	if perr == nil {
		perr = agent.ExtractJSON(res.Stdout, "plan", &pl)
	}
	if perr == nil {
		if implement.StripProvenance(&pl) {
			o.journal("contract_deviation", 1, map[string]any{"kind": "planner-emitted-provenance"})
			o.logf("plan: the planner emitted a provenance block; discarded -- provenance is fixpoint's to write")
		}
		perr = implement.Validate(pl, rules)
	}
	rec.Steps = append(rec.Steps, stepStat("plan", planner, o.cfg.Roles.Planner.Prompt, len(text), res, perr != nil))
	o.logStep("plan", planner, o.cfg.Roles.Planner.Prompt, 1, perr == nil, pl, planMarkdown(pl, p.coverageLine(), perr), res, "")
	if perr != nil {
		return pl, fmt.Errorf("plan: %w", perr)
	}

	return o.stampPlan(pl, p)
}

// stampPlan writes the provenance this RUN owns onto a validated plan and
// records the canonical artifact. Shared by both ways a plan arrives, because
// everything here is a fact about the invocation rather than about the plan: a
// -plan handback needs the same run id, coder and verify profile that a planner
// session's answer does, and the first version returned before this ran, so the
// supplied plan reached SCAFFOLD with no provenance at all and nil-dereferenced.
func (o *Orchestrator) stampPlan(pl implement.Plan, p *implementPrep) (implement.Plan, error) {
	pl.Provenance = &implement.Provenance{
		SchemaVersion: 1,
		RunID:         o.logs.RunID(),
		DesignPath:    p.designPath,
		DesignSHA256:  p.designSHA,
		Planner:       p.plannerLabel,
		Coder:         o.cfg.Roles.Coder.Agent,
		VerifyProfile: p.profile,
	}
	// The canonical plan as an ARTIFACT, provenance included -- §4.3's
	// `.fixpoint/<ts>/plan.json`. The step artifact logged above is the parsed
	// value before provenance was injected, so it was not this; §4.3 named this
	// home and nothing wrote it (review run 20260813-180828, i23).
	if err := o.writePlanArtifact(pl); err != nil {
		return pl, err
	}
	o.journal("plan_finished", 1, map[string]any{
		"tasks":    len(pl.Tasks),
		"coverage": p.coverageLine(),
	})
	o.endPhase("PLAN  %d task(s), coverage %s", len(pl.Tasks), p.coverageLine())
	return pl, nil
}

// writePlanArtifact records the validated, provenance-stamped plan at the run
// root. Distinct from the project's committed PLAN.json: this one survives on
// the machine that RAN the plan even when the project does not.
func (o *Orchestrator) writePlanArtifact(pl implement.Plan) error {
	b, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}
	_, err = o.logs.RunState("plan.json", append(b, '\n'))
	return err
}

// runStatus is `.fixpoint/<ts>/status.json`: the live view of a run in flight.
type runStatus struct {
	SchemaVersion int                 `json:"schema_version"`
	RunID         string              `json:"run_id"`
	Project       string              `json:"project"`
	Planned       int                 `json:"planned"`
	Processed     int                 `json:"processed"`
	InFlight      string              `json:"in_flight,omitempty"`
	Tasks         []model.TaskOutcome `json:"tasks"`
}

// writeRunState rewrites `.fixpoint/<ts>/status.json` -- the live per-task
// state §4.3 promises, atomically, after every task. It is the only mutable
// state a running implement pipeline exposes: without it the observability of a
// multi-hour unattended run was a stderr line nobody was watching.
//
// A failure here is logged, never returned: losing the observability file must
// not lose the run that was being observed.
func (o *Orchestrator) writeRunState(sum *model.RunSummary, pl implement.Plan, inFlight string) {
	state := runStatus{
		SchemaVersion: 1,
		RunID:         o.logs.RunID(),
		Project:       pl.Project.Name,
		Planned:       len(pl.Tasks),
		Processed:     len(sum.Tasks),
		InFlight:      inFlight,
		Tasks:         sum.Tasks,
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err == nil {
		_, err = o.logs.RunState("status.json", append(b, '\n'))
	}
	if err != nil {
		o.logf("WARNING: could not write status.json (%v); the run continues unobserved", err)
	}
}

// readOutline extracts the design's heading skeleton for §4.2 rule 6.
//
// An unreadable outline is a preflight refusal unless the operator waived the
// rule for this invocation, in which case the run proceeds and SAYS so
// everywhere -- silence would let a report imply a check that never ran.
func (o *Orchestrator) readOutline(p *implementPrep) error {
	p.coverageChecked = !o.cfg.Implement.NoCoverageCheck
	outline, err := implement.ExtractOutline(string(p.designBytes))
	p.outline = outline
	switch {
	case err != nil && p.coverageChecked:
		return fmt.Errorf("implement: %w", err)
	case err != nil:
		o.logf("coverage: UNCHECKED -- %v; -no-coverage-check was given, so the plan is not held to §4.2 rule 6 and no report will claim it was", err)
	case !p.coverageChecked:
		o.logf("coverage: UNCHECKED by -no-coverage-check, though the outline is readable (%q level, %d heading(s))",
			strings.Repeat("#", p.outline.Level), len(p.outline.Headings))
	default:
		o.logf("coverage outline: %q level, %d heading(s)", strings.Repeat("#", p.outline.Level), len(p.outline.Headings))
	}
	return nil
}

// coverageLine is what every report says about §4.2 rule 6 -- the extracted
// outline, or exactly "unchecked". The design requires the word: nothing in a
// run report may imply the check ran when it did not.
func (p *implementPrep) coverageLine() string {
	if !p.coverageChecked {
		return "unchecked"
	}
	return fmt.Sprintf("%q (%d headings)", strings.Repeat("#", p.outline.Level), len(p.outline.Headings))
}

// operatorPlan is the -plan path: load, verify the design pairing, validate
// against §4.2, and stamp the provenance this run owns.
//
// No planner session is spent, so no step is recorded for one -- a run that
// invoked no agent must not appear to have invoked one. What IS recorded is
// where the plan came from: Provenance.Planner says operator-supplied with the
// file's own digest, because naming the configured planner agent as the author
// of a plan it never saw would make the audit trail state a falsehood.
func (o *Orchestrator) operatorPlan(src string, p *implementPrep, rules implement.Rules) (implement.Plan, error) {
	pl, planSHA, err := implement.LoadOperatorPlan(src, p.designSHA)
	if err != nil {
		return pl, err
	}
	// The planner's own provenance block, whatever it claimed, is not this run's.
	// Only the design pairing checked above was the file's to assert; run id,
	// coder and verify profile are facts about THIS invocation.
	implement.StripProvenance(&pl)
	if err := implement.Validate(pl, rules); err != nil {
		return pl, fmt.Errorf("-plan %s: %w", src, err)
	}
	o.journal("plan_supplied", 1, map[string]any{"path": src, "sha256": planSHA, "tasks": len(pl.Tasks)})
	o.logf("plan: %d task(s) from %s (sha256 %.12s), validated against every rule; no planner session was spent", len(pl.Tasks), src, planSHA)
	p.plannerLabel = implement.PlannerLabel(o.cfg.Roles.Planner.Agent, true, planSHA)
	return pl, nil
}

// planMarkdown renders the plan step's durable artifact. coverage is the run's
// answer to rule 6, verbatim: this artifact is read on its own, so "validated"
// hard-coded here would tell an operator the check ran on a run where it was
// waived.
func planMarkdown(pl implement.Plan, coverage string, err error) string {
	if err != nil {
		return fmt.Sprintf("# plan (refused)\n\n%v\n", err)
	}
	return implement.RenderMarkdown(pl, implement.RenderHeader{Coverage: coverage})
}

// scaffoldPhase claims the write-target and makes the bootstrap commit (§5.1).
func (o *Orchestrator) scaffoldPhase(ctx context.Context, sum *model.RunSummary, p *implementPrep, pl implement.Plan) error {
	o.phase("SCAFFOLD  %s", p.out)
	coverage := p.coverageLine()
	planJSON, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}
	// PLAN.md and PLAN.json are agent-authored text committed FOREVER into the
	// delivered repository, so they go through the same redaction every
	// published artifact does -- a planner asked by an injected design to copy
	// a credential from a neighboring .env into a task goal must not have it
	// recorded in the history (review run 20260813-161029). DESIGN.md is
	// exempt and must stay byte-exact: its hash is what ties the project to
	// the document review-design approved, and redacting it would break that
	// match. The design is the operator's own asserted input (-trusted-target),
	// which is precisely what the plan derived from it is not.
	files := map[string][]byte{
		"DESIGN.md":  p.designBytes,
		"PLAN.md":    []byte(agent.RedactSecrets(implement.RenderMarkdown(pl, implement.RenderHeader{Coverage: coverage, GateCommands: p.gateCommands}))),
		"PLAN.json":  []byte(agent.RedactSecrets(string(planJSON)) + "\n"),
		".gitignore": []byte(implement.GitignoreContent(o.cfg.Implement.GitignoreSeed)),
	}
	header := fmt.Sprintf("fixpoint: initialize implementation of %q", flattenField(pl.Project.Name))
	body := strings.Join([]string{
		"Fixpoint-Run: " + o.logs.RunID(),
		"Fixpoint-Producer: fixpoint implement-design",
		"Fixpoint-Phase: bootstrap",
		"Design-SHA256: " + p.designSHA,
		"Planner: " + flattenField(pl.Provenance.Planner),
		"Coder: " + flattenField(pl.Provenance.Coder),
		"Verify-Profile: " + p.profile,
		"Coverage-Check: " + coverage,
	}, "\n")

	p.col = target.New(config.Target{Mode: config.ModeDirectory, Path: p.out})
	// The write-target gets the credential-stripped environment AND has the
	// operator's global and system git config switched off. Measured (review run
	// 20260813-161029): a `.gitattributes` the coder writes plus a filter
	// defined in global config makes `git add` execute that filter, and with
	// global config off there is nowhere left to define one -- the repository's
	// own config is covered by the §5.2 step 4 invariant. Safe here in a way it
	// would not be over an operator's repository: Init writes this repository's
	// identity locally, so nothing fixpoint does needs the global file.
	p.col.UseGitEnv(p.gitEnv)
	// Scaffold takes the repository lock the moment `git init` has made one --
	// before it writes or commits anything -- and hands it back to be held for
	// the rest of the run. Without the lock a concurrent fixpoint run finds an
	// apparently free repository and commits into the same worktree, where
	// checkInvariants would read its commit as the coder's, soft-reset it, and
	// fold a stranger's changes into this task's commit under the wrong
	// attribution (review runs 20260813-124710 and -161029).
	sha, release, err := implement.Scaffold(ctx, p.col, p.out, files, agent.RedactSecrets(header), agent.RedactSecrets(body))
	if err != nil {
		return err
	}
	p.release = release
	p.bootstrapSHA = sha
	p.attributed = sha
	sum.Deliverable = p.out
	if p.baseline, err = p.git.SnapshotRepoState(ctx, p.out, "main"); err != nil {
		return err
	}
	o.endPhase("SCAFFOLD  bootstrap %.12s (not gated, no sources) · repository locked", sha)
	return nil
}

// buildPhase is the task loop (§5.2): serial, one commit per processed task,
// skips derived from the dependency graph, the deadline checked between
// tasks, and an infrastructure circuit breaker so a provider outage never
// reaches the immutable history (§5.4).
func (o *Orchestrator) buildPhase(ctx context.Context, sum *model.RunSummary, rec *model.RoundRecord, p *implementPrep, pl implement.Plan, started time.Time, resume implement.Resume) error {
	infraStrikes := 0
	incomplete := false
	// Everything the history already decided is recorded before the loop starts,
	// so the report of a resumed run covers the whole plan and not just the part
	// this invocation touched. `carried` is §5.4's word for exactly that: an
	// outcome this run adopted rather than produced -- and adopting it includes
	// adopting its VERDICT. A carried failure used to leave `incomplete` false,
	// so resuming a project whose history covered the whole plan terminated
	// `implemented`/exit 0 over a task the original run reported failed/exit 2
	// -- the resumed run contradicting the report the original produced, which
	// is the one thing carried outcomes exist to prevent (review run
	// 20260818-234734, three reviewers independently).
	for _, t := range pl.Tasks[:resume.Index] {
		rc := resume.Carried[t.ID]
		p.outcomes[t.ID] = rc.Outcome
		if rc.Outcome != outcomeImplemented && rc.Outcome != outcomeSatisfied {
			incomplete = true
		}
		sum.Tasks = append(sum.Tasks, model.TaskOutcome{
			ID: t.ID, Title: t.Title, Outcome: outcomeCarried,
			CarriedOutcome: rc.Outcome,
			Reason:         carriedReason(rc), SHA: rc.SHA,
		})
	}
	if resume.Index > 0 {
		o.phase("BUILD  %d task(s) carried from the history, %d to go", resume.Index, len(pl.Tasks)-resume.Index)
	} else {
		o.phase("BUILD  %d task(s), serially", len(pl.Tasks))
	}
	p.deadline = started.Add(o.cfg.Implement.MaxRunDuration.Std())
	stoppedBefore := ""

	for i, t := range pl.Tasks[resume.Index:] {
		i += resume.Index
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if elapsed := time.Since(started); elapsed > o.cfg.Implement.MaxRunDuration.Std() {
			o.logf("BUILD: max_run_duration (%s) reached after %s; stopping before %s", o.cfg.Implement.MaxRunDuration.Std(), elapsed.Round(time.Second), t.ID)
			stoppedBefore = t.ID
			incomplete = true
			break
		}
		// The between-task disk check §8 documents and the code never had
		// (review run 20260813-180828, i46). Disk is the one resource this
		// pipeline grows and never releases -- build output, one stash per
		// failed attempt, the journal -- and running out mid-task surfaces as an
		// opaque git error from a commit or a discard, or as a half-finished
		// discard that the NEXT task reports as fixpoint corrupting its own tree.
		// Named here instead, while the diagnosis is still available.
		if free, ok := o.freeSpace(p.out); ok && free < o.cfg.Implement.MinFreeDisk.Int64() {
			o.logf("BUILD: %s has %s free, under implement.min_free_disk (%s); stopping before %s",
				p.out, config.ByteSize(free), o.cfg.Implement.MinFreeDisk, t.ID)
			o.journal("disk_exhausted", 1, map[string]any{"id": t.ID, "free": free, "min": o.cfg.Implement.MinFreeDisk.Int64()})
			stoppedBefore = t.ID
			incomplete = true
			break
		}
		if blockedBy := implement.BlockedDependency(t, p.outcomes); blockedBy != "" {
			p.outcomes[t.ID] = outcomeSkipped
			sum.Tasks = append(sum.Tasks, model.TaskOutcome{ID: t.ID, Title: t.Title, Outcome: outcomeSkipped, Reason: "dependency " + blockedBy})
			o.journal("task_skipped", 1, map[string]any{"id": t.ID, "blocked_by": blockedBy})
			o.logf("task %s skipped: dependency %s did not land", t.ID, blockedBy)
			incomplete = true
			continue
		}
		o.writeRunState(sum, pl, t.ID)
		bad, stopLoop, err := o.processTask(ctx, sum, rec, p, pl, i, &infraStrikes)
		if err != nil {
			o.writeRunState(sum, pl, "")
			return err
		}
		o.writeRunState(sum, pl, "")
		if bad {
			incomplete = true
		}
		if stopLoop {
			stoppedBefore = t.ID
			break
		}
	}
	// Every task the loop never reached is recorded, with the reason it was
	// never reached. §1's contract is that a task either has a gated commit or
	// is in the report with a reason, and an early exit used to satisfy neither:
	// the run printed "5 task(s) · 5 implemented" for a 20-task plan and nothing
	// anywhere said what the other 15 were (review run 20260813-180828, i22).
	o.recordUnreached(sum, pl, stoppedBefore)
	o.writeRunState(sum, pl, "")
	o.endPhase("BUILD  %d of %d task(s) processed", len(sum.Tasks), len(pl.Tasks))

	if len(o.cfg.Verify.Commands) > 0 && o.cfg.Implement.CleanCheck == config.CleanCheckLast {
		if err := o.cleanCheck(ctx, p); err != nil {
			o.logf("clean-check: %v", err)
			incomplete = true
		}
	}
	if incomplete || len(sum.Tasks) < len(pl.Tasks) {
		sum.Termination = model.TermIncomplete
	} else {
		sum.Termination = model.TermImplemented
	}
	return nil
}

// processTask runs one plan entry to a recorded outcome and reports whether
// the run got worse (bad) and whether the loop must stop (stopLoop). Split
// from buildPhase for the complexity limit only.
func (o *Orchestrator) processTask(ctx context.Context, sum *model.RunSummary, rec *model.RoundRecord, p *implementPrep, pl implement.Plan, i int, infraStrikes *int) (bad, stopLoop bool, err error) {
	t := pl.Tasks[i]
	out, err := o.runTask(ctx, rec, p, pl, i, infraStrikes)
	if err != nil {
		var stop runStopError
		if errors.As(err, &stop) {
			o.journal("repo_invariant_failed", 1, map[string]any{"id": t.ID, "what": stop.why})
			return true, true, err
		}
		if errors.Is(err, errInfraBreaker) {
			o.logf("BUILD: two consecutive infrastructure failures; stopping incomplete before %s -- no marker is written, so a later run re-enters here", t.ID)
			return true, true, nil
		}
		return true, true, err
	}
	p.outcomes[t.ID] = out.Outcome
	sum.Tasks = append(sum.Tasks, out)
	if out.Outcome != outcomeImplemented && out.Outcome != outcomeSatisfied {
		bad = true
	}
	if out.Outcome == outcomeImplemented && len(o.cfg.Verify.Commands) > 0 && o.cfg.Implement.CleanCheck == config.CleanCheckEvery {
		if err := o.cleanCheck(ctx, p); err != nil {
			// At `every` the culprit IS attributable: it is this task.
			o.logf("clean-check after %s: %v", t.ID, err)
			return true, true, nil
		}
	}
	if n, frac := vacuousFraction(sum.Tasks, len(pl.Tasks)); frac > o.cfg.Implement.MaxVacuousFrac {
		o.logf("BUILD: the plan was decomposed badly: %d of %d task(s) were already covered by earlier work, over max_vacuous_frac (%.0f%%)", n, len(pl.Tasks), o.cfg.Implement.MaxVacuousFrac*100)
		return true, true, nil
	}
	return bad, false, nil
}

// freeSpace reports free bytes at path, through the test seam.
func (o *Orchestrator) freeSpace(path string) (int64, bool) {
	if o.diskFree != nil {
		return o.diskFree(path)
	}
	return implement.DiskFree(path)
}

// recordUnreached appends an outcome for every planned task the loop never
// processed, so the report's denominator is the PLAN and not just what was
// attempted. stoppedBefore is the task the run stopped at, when it stopped for
// a reason rather than finishing.
func (o *Orchestrator) recordUnreached(sum *model.RunSummary, pl implement.Plan, stoppedBefore string) {
	seen := make(map[string]bool, len(sum.Tasks))
	for _, t := range sum.Tasks {
		seen[t.ID] = true
	}
	reason := "the run ended before this task"
	if stoppedBefore != "" {
		reason = "the run stopped at " + stoppedBefore
	}
	for _, t := range pl.Tasks {
		if seen[t.ID] {
			continue
		}
		sum.Tasks = append(sum.Tasks, model.TaskOutcome{ID: t.ID, Title: t.Title, Outcome: outcomeUnreached, Reason: reason})
		o.journal("task_unreached", 1, map[string]any{"id": t.ID, "reason": reason})
	}
}

// planOverheadSlack is the allowance for everything before the planner session
// that the run is charged for: the agent ping and the design snapshot.
const planOverheadSlack = 5 * time.Minute

// errInfraBreaker trips after the infrastructure retry budget is spent: the
// run stops incomplete with NO marker for the task in flight (§5.4).
var errInfraBreaker = errors.New("infrastructure circuit breaker")

// infraBackoff is what an infrastructure failure waits before the next try,
// indexed by how many have happened in a row. The budget is deliberately
// bounded and deliberately not instant.
//
// The shipped breaker counted two consecutive failures and retried with NO wait
// between them (review run 20260813-180828, i42), so the second call landed
// inside the same rate-limit window as the first: for a pipeline whose premise
// is a 32-hour unattended run, the most common transient failure of its primary
// dependency ended the run within seconds. An overnight run started at 22:00
// could be dead at 22:04 with one task built.
//
// Four tries, ~51 minutes of total waiting, then the breaker. Long enough to
// ride out an ordinary 429 burst or a provider blip; short enough that a real
// outage does not silently consume the deadline it is charged against.
var infraBackoff = []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute}

// backoffSchedule is the orchestrator's copy, so a test can shrink the waits
// without a package-level variable two tests could race over.
func (o *Orchestrator) backoffSchedule() []time.Duration {
	if len(o.infraBackoff) > 0 {
		return o.infraBackoff
	}
	// implement.max_infra_tries decides how many rungs are used. Beyond the
	// declared ladder the last wait repeats, so raising the key never silently
	// shortens the waits.
	n := o.cfg.Implement.MaxInfraTries
	if n <= 0 || n == len(infraBackoff) {
		return infraBackoff
	}
	out := make([]time.Duration, n)
	for i := range out {
		if i < len(infraBackoff) {
			out[i] = infraBackoff[i]
			continue
		}
		out[i] = infraBackoff[len(infraBackoff)-1]
	}
	return out
}

// waitForInfra sleeps out the strike's backoff, refusing when the wait would
// run past the run's own deadline -- the wait is charged against
// max_run_duration, so it must not silently overrun it. Returns false when the
// run must stop instead of retrying.
func (o *Orchestrator) waitForInfra(ctx context.Context, p *implementPrep, taskID string, strike int) bool {
	schedule := o.backoffSchedule()
	if strike > len(schedule) {
		return false
	}
	wait := schedule[strike-1]
	if !p.deadline.IsZero() && time.Now().Add(wait).After(p.deadline) {
		o.logf("task %s: infrastructure failure and %s of backoff would pass max_run_duration; stopping instead of waiting", taskID, wait)
		return false
	}
	o.logf("task %s: infrastructure failure (%d of %d); waiting %s before trying again", taskID, strike, len(schedule), wait)
	o.journal("infra_backoff", 1, map[string]any{"id": taskID, "strike": strike, "wait": wait.String()})
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// infraStop distinguishes the two reasons waitForInfra gives up. Both used to
// return errInfraBreaker, so Ctrl-C during a backoff was reported as an
// incomplete run at exit 2 instead of an interruption at exit 1 -- the operator
// who stopped the run was told the provider had (review run 20260814-012440).
func infraStop(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errInfraBreaker
}

// runTask processes one task to a terminal outcome, attempts included.
func (o *Orchestrator) runTask(ctx context.Context, rec *model.RoundRecord, p *implementPrep, pl implement.Plan, idx int, infraStrikes *int) (model.TaskOutcome, error) {
	t := pl.Tasks[idx]
	out := model.TaskOutcome{ID: t.ID, Title: t.Title}
	priorFailure := ""
	blockedOnce := false

	for attempt := 1; attempt <= o.cfg.Implement.MaxTaskAttempts; {
		o.journal("task_started", 1, map[string]any{"id": t.ID, "title": t.Title, "attempt": attempt})
		res, report, verdict, err := o.taskAttempt(ctx, rec, p, pl, idx, attempt, priorFailure)
		if err != nil {
			return out, err
		}
		switch verdict.kind {
		case attemptInfra:
			*infraStrikes++
			o.journal("infra_failure", 1, map[string]any{"id": t.ID, "status": res.ProviderStatus, "strike": *infraStrikes})
			o.logf("task %s: infrastructure failure (%s); the attempt is not counted", t.ID, verdict.why)
			// 402 is the one status where waiting cannot help: the account is out
			// of credit or the plan is exhausted, and every retry buys the same
			// answer more slowly. Distinguished from 429/5xx, which are exactly
			// what the backoff exists for.
			if res.ProviderStatus == 402 {
				o.logf("task %s: the provider returned 402 -- no amount of waiting fixes payment; stopping now", t.ID)
				return out, errInfraBreaker
			}
			if !o.waitForInfra(ctx, p, t.ID, *infraStrikes) {
				return out, infraStop(ctx)
			}
			continue // same attempt number: infrastructure is not the task's fault
		case attemptImplemented:
			*infraStrikes = 0
			out.Outcome = outcomeImplemented
			out.SHA = verdict.sha
			out.Attempts = attempt
			out.Gate = verdict.gate
			o.journal("task_committed", 1, map[string]any{"id": t.ID, "sha": verdict.sha})
			o.journal("task_finished", 1, map[string]any{"id": t.ID, "status": outcomeImplemented, "notes": flattenField(report.Notes)})
			return out, nil
		case attemptSatisfied:
			*infraStrikes = 0
			out.Outcome = outcomeSatisfied
			out.Reason = "covered_by=" + strings.Join(report.CoveredBy, ",")
			out.Attempts = attempt
			sha, err := o.markerCommit(ctx, p, t, outcomeSatisfied, "covered_by", strings.Join(report.CoveredBy, ","))
			if err != nil {
				return out, err
			}
			out.SHA = sha
			o.journal("task_finished", 1, map[string]any{"id": t.ID, "status": outcomeSatisfied})
			return out, nil
		case attemptBlocked:
			*infraStrikes = 0
			// A blocked report has the largest blast radius in the taxonomy, so
			// it gets what already_satisfied got: confirmation by a second,
			// independent session before the marker is written (§5.4).
			//
			// Only when an attempt is actually left to spend. Under
			// max_task_attempts: 1 the operator configured away the budget for a
			// second opinion, and the honest answer is to believe the one
			// session and SAY it was uncorroborated -- the first review of this
			// code found the alternative (run 20260813-124710): the confirmation
			// consumed the only attempt, the loop fell through, and a genuine
			// blocked report was recorded as `failed` with a reason claiming a
			// previous session that never ran.
			if !blockedOnce && attempt < o.cfg.Implement.MaxTaskAttempts {
				blockedOnce = true
				priorFailure = "a previous session reported this task blocked on " + strings.Join(report.BlockedOn, ", ") + "; verify independently -- implement it if it can be implemented"
				attempt++
				continue
			}
			out.Outcome = outcomeBlocked
			reason := "design_conflict: " + strings.Join(report.BlockedOn, ", ")
			if !blockedOnce {
				reason += " (uncorroborated: max_task_attempts is 1, so no second session checked it)"
			}
			out.Reason = reason
			out.Attempts = attempt
			sha, err := o.markerCommit(ctx, p, t, outcomeBlocked, "blocked_on", reason)
			if err != nil {
				return out, err
			}
			out.SHA = sha
			o.journal("task_finished", 1, map[string]any{"id": t.ID, "status": outcomeBlocked, "corroborated": blockedOnce, "notes": flattenField(report.Notes)})
			return out, nil
		default: // attemptFailed
			*infraStrikes = 0
			priorFailure = verdict.why
			o.logf("task %s attempt %d failed: %s", t.ID, attempt, verdict.why)
			attempt++
		}
	}
	out.Outcome = outcomeFailed
	out.Reason = verdictReason(priorFailure)
	out.Attempts = o.cfg.Implement.MaxTaskAttempts
	sha, err := o.markerCommit(ctx, p, t, outcomeFailed, "reason", verdictReason(priorFailure))
	if err != nil {
		return out, err
	}
	out.SHA = sha
	o.journal("task_finished", 1, map[string]any{"id": t.ID, "status": outcomeFailed})
	return out, nil
}

type attemptKind int

const (
	attemptFailed attemptKind = iota
	attemptImplemented
	attemptSatisfied
	attemptBlocked
	attemptInfra
)

type attemptVerdict struct {
	kind attemptKind
	why  string
	sha  string
	gate string
}

// taskAttempt is exactly one coder session and one gate run (§5.3): the
// invariant ladder, the reconciliation, the censuses, the gate, the commit.
// Every path that does not reach a commit exits through the discard.
func (o *Orchestrator) taskAttempt(ctx context.Context, rec *model.RoundRecord, p *implementPrep, pl implement.Plan, idx, attempt int, priorFailure string) (agent.Result, taskReport, attemptVerdict, error) {
	t := pl.Tasks[idx]
	var report taskReport

	// Step 0: the clean-tree precondition. A violation is a fixpoint bug, not
	// a task failure (§5.2).
	if clean, err := p.col.GitClean(ctx); err != nil {
		return agent.Result{}, report, attemptVerdict{}, err
	} else if !clean {
		return agent.Result{}, report, attemptVerdict{}, runStopError{"the tree is dirty at the start of a task; the residue is fixpoint's, not the coder's -- report this"}
	}
	// Step 2: the base.
	base, err := p.col.HeadSHA(ctx)
	if err != nil {
		return agent.Result{}, report, attemptVerdict{}, err
	}
	preState, err := p.git.SnapshotRepoState(ctx, p.out, "main")
	if err != nil {
		return agent.Result{}, report, attemptVerdict{}, err
	}
	preIgnored, err := p.git.TakeIgnoredCensus(ctx, p.out)
	if err != nil {
		return agent.Result{}, report, attemptVerdict{}, err
	}
	// What the repository looked like before the session -- and, because the
	// ladder below either finds HEAD unmoved or resets it back to base, what it
	// must still look like when the gate has finished too.
	before := attemptBase{head: base, repo: preState, ignored: preIgnored}

	// Step 3: the coder session.
	d := prompt.TaskData{
		ID: t.ID, Title: t.Title, Goal: t.Goal, Acceptance: t.Acceptance,
		DesignRefs: t.DesignRefs, PlanShape: planShape(pl, idx),
		PriorFailure: priorFailure, Attempt: attempt,
		OutputContract: prompt.ImplementContract,
	}
	text, err := prompt.Render(o.templates[o.cfg.Roles.Coder.Prompt], d)
	if err != nil {
		return agent.Result{}, report, attemptVerdict{}, err
	}
	coder := o.cfg.Roles.Coder.Agent
	label := fmt.Sprintf("task %s (attempt %d)", t.ID, attempt)
	res := o.runAgentIn(ctx, p.out, label, "task", coder, t.ID, 1, text)
	perr := res.Err
	if perr == nil {
		perr = agent.ExtractJSON(res.Stdout, "implement", &report)
	}
	rec.Steps = append(rec.Steps, stepStat("task", coder, t.ID, len(text), res, perr != nil))
	o.logStep("task", coder, t.ID, 1, perr == nil, report, taskMarkdown(t, report, perr), res, "")

	// Step 4 FIRST, whatever the session's exit status: a session that commits
	// and then dies -- a 429 after `git commit`, the timeout firing mid-turn --
	// still moved HEAD, and the ladder is the only thing that notices. Running
	// the infrastructure verdict before it (the shape the first review of this
	// code found, run 20260813-124710) left that commit in the history,
	// unowned: the discard then had nothing to stash, the clean assertion
	// passed, and the next attempt built on top of a commit fixpoint never
	// attributed.
	if err := o.checkInvariants(ctx, p, t.ID, base, preState); err != nil {
		return res, report, attemptVerdict{}, err
	}

	// Infrastructure is not an outcome (§5.4): the provider refused, or the
	// session died leaving nothing.
	if res.Err != nil && (res.ProviderStatus != 0 || strings.TrimSpace(res.Stdout) == "") {
		if err := o.discardAttempt(ctx, p, t.ID, attempt); err != nil {
			return res, report, attemptVerdict{}, err
		}
		return res, report, attemptVerdict{kind: attemptInfra, why: fmt.Sprintf("provider status %d", res.ProviderStatus)}, nil
	}

	// Step 5: reconcile -- control artifacts first (§4.3).
	if changed, err := p.git.ControlArtifactsChanged(ctx, p.out, p.bootstrapSHA); err != nil {
		return res, report, attemptVerdict{}, err
	} else if len(changed) > 0 {
		if err := p.git.RestoreControlArtifacts(ctx, p.out, p.bootstrapSHA, changed); err != nil {
			return res, report, attemptVerdict{}, err
		}
		o.journal("contract_deviation", 1, map[string]any{"id": t.ID, "kind": "control-artifact-edit", "paths": changed})
		if err := o.discardAttempt(ctx, p, t.ID, attempt); err != nil {
			return res, report, attemptVerdict{}, err
		}
		return res, report, attemptVerdict{kind: attemptFailed, why: "the session edited control artifacts: " + strings.Join(changed, ", ")}, nil
	}
	if perr != nil {
		if err := o.discardAttempt(ctx, p, t.ID, attempt); err != nil {
			return res, report, attemptVerdict{}, err
		}
		return res, report, attemptVerdict{kind: attemptFailed, why: "the session broke the output contract: " + perr.Error()}, nil
	}
	return o.reconcileAttempt(ctx, p, pl, idx, attempt, before, res, report)
}

// reconcileAttempt is §5.2 steps 5-8 for a session that returned a report.
func (o *Orchestrator) reconcileAttempt(ctx context.Context, p *implementPrep, pl implement.Plan, idx, attempt int, before attemptBase, res agent.Result, report taskReport) (agent.Result, taskReport, attemptVerdict, error) {
	t := pl.Tasks[idx]
	clean, err := p.col.GitClean(ctx)
	if err != nil {
		return res, report, attemptVerdict{}, err
	}
	fail := func(why string) (agent.Result, taskReport, attemptVerdict, error) {
		if err := o.discardAttempt(ctx, p, t.ID, attempt); err != nil {
			return res, report, attemptVerdict{}, err
		}
		return res, report, attemptVerdict{kind: attemptFailed, why: why}, nil
	}

	if verdict, why, done := reconcileReport(t, report, clean, pl, idx, p.outcomes); done {
		if why != "" {
			return fail(why)
		}
		return res, report, verdict, nil
	}

	// Step 6: the census, the byte bound, the ignored reconciliation.
	census, why, err := o.censusPhase(ctx, p)
	if err != nil {
		return res, report, attemptVerdict{}, err
	}
	if why != "" {
		return res, report, attemptVerdict{kind: attemptFailed, why: why}, nil
	}
	why, cerr := o.refuseCredentialShaped(ctx, p, t.ID, census)
	if cerr != nil {
		return res, report, attemptVerdict{}, cerr
	}
	if why != "" {
		return res, report, attemptVerdict{kind: attemptFailed, why: why}, nil
	}
	postIgnored, err := p.git.TakeIgnoredCensus(ctx, p.out)
	if err != nil {
		return res, report, attemptVerdict{}, err
	}
	created, modified := implement.DiffIgnored(before.ignored, postIgnored)
	// The failures used to be discarded while the log said the paths had been
	// deleted. Ignored paths are invisible to the census and to GitClean, so a
	// path that survived affected the gate and the next attempt with nothing
	// anywhere recording it (review run 20260814-012440).
	if stuck := removeAll(p.out, created); len(stuck) > 0 {
		return res, report, attemptVerdict{}, fmt.Errorf("could not delete session-created ignored path(s) before the gate: %s -- they would be measured as part of the work and carried into the next attempt", strings.Join(stuck, ", "))
	}
	if len(created) > 0 || len(modified) > 0 {
		o.journal("ignored_paths_diff", 1, map[string]any{"id": t.ID, "created": created, "modified": modified})
		if len(created) > 0 {
			o.logf("task %s: %d session-created ignored path(s) deleted before the gate; the gate recreates what it needs from committed sources", t.ID, len(created))
		}
	}

	// Step 7: the gate.
	gateLabel, gatePaths, why, err := o.gatePhase(ctx, p, t.ID, census, before)
	var infraGate infraGateError
	if errors.As(err, &infraGate) {
		// Discarded like any other unfinished attempt, but recorded as
		// infrastructure: no attempt consumed, no marker, and the breaker gets
		// the strike so a registry that is down does not spin the whole plan.
		if derr := o.discardAttempt(ctx, p, t.ID, attempt); derr != nil {
			return res, report, attemptVerdict{}, derr
		}
		return res, report, attemptVerdict{kind: attemptInfra, why: infraGate.Error()}, nil
	}
	if err != nil {
		return res, report, attemptVerdict{}, err
	}
	if why != "" {
		return fail(why)
	}

	// Step 8: the commit -- exactly the computed path set.
	sha, err := o.commitTask(ctx, p, t, attempt, census, gateLabel, gatePaths)
	if err != nil {
		return res, report, attemptVerdict{}, err
	}
	return res, report, attemptVerdict{kind: attemptImplemented, sha: sha, gate: gateLabel}, nil
}

// refuseCredentialShaped fails an attempt whose census names a file the
// target's exclude patterns match -- the config's own list plus the mandatory
// credential patterns no config can drop. A non-empty reason fails the attempt.
//
// Checked before the gate so the gate is not paid for first, and failed rather
// than silently dropped: a project that genuinely needs an .env needs the
// operator to say so, and a task whose work quietly lost a file it wrote would
// miss its acceptance criteria for no stated reason.
//
// Discarded with checkout-and-clean rather than the ordinary stash, for the
// same reason the byte bound uses it (§5.3): a stash is a commit, so stashing a
// secret still writes it into the delivered repository -- buried under
// refs/stash instead of main, which is not the promise. Measured while fixing
// this: the first version routed the failure through the stash and the .env was
// still reachable in the project's history.
func (o *Orchestrator) refuseCredentialShaped(ctx context.Context, p *implementPrep, taskID string, census implement.Census) (string, error) {
	hit, err := p.col.ExcludedPaths(census.Paths())
	if err != nil || len(hit) == 0 {
		return "", err
	}
	o.journal("credential_shaped_paths", 1, map[string]any{"id": taskID, "paths": hit})
	if err := p.col.DiscardClean(ctx); err != nil {
		return "", err
	}
	if err := o.assertClean(ctx, p); err != nil {
		return "", err
	}
	return "the session created credential-shaped file(s), discarded unstashed: " + strings.Join(hit, ", ") +
		" -- these match the patterns that hide a path from every reviewer, so keeping them would bury a secret in the project's history", nil
}

// removeAll deletes each repo-relative path and returns the ones that survived.
// A path that cannot be removed is reported rather than assumed gone: everything
// this deletes is invisible to the census, so nothing downstream would notice.
func removeAll(root string, paths []string) []string {
	var stuck []string
	for _, path := range paths {
		if err := os.RemoveAll(filepath.Join(root, path)); err != nil {
			stuck = append(stuck, fmt.Sprintf("%s (%v)", path, err))
		}
	}
	return stuck
}

// censusPhase is §5.2 step 6's census with its byte bound. A non-empty reason
// fails the attempt: the oversized tree is discarded by checkout-and-clean
// rather than stashed, so the ceiling never writes it into the object database
// (§5.3).
func (o *Orchestrator) censusPhase(ctx context.Context, p *implementPrep) (implement.Census, string, error) {
	census, err := p.git.TakeCensus(ctx, p.out, o.cfg.Implement.MaxTaskBytes.Int64())
	var bb implement.ByteBoundError
	if errors.As(err, &bb) {
		if derr := p.col.DiscardClean(ctx); derr != nil {
			return census, "", derr
		}
		if derr := o.assertClean(ctx, p); derr != nil {
			return census, "", derr
		}
		return census, bb.Error(), nil
	}
	return census, "", err
}

// gatePhase is §5.2 step 7: run the gate, classify every difference, remove
// output after EVERY gate run (a failed gate's droppings must not sit in the
// tree for the next attempt). A non-empty `why` fails the attempt.
func (o *Orchestrator) gatePhase(ctx context.Context, p *implementPrep, taskID string, census implement.Census, before attemptBase) (label string, gatePaths []string, why string, err error) {
	// Enabled(), not just a non-empty list: a config with `verify.policy: off`
	// and commands inherited from its base ran them anyway and could fail tasks
	// on them, while VerifyOff is documented as running nothing (review run
	// 20260814-012440).
	if !o.cfg.Verify.Enabled() {
		return "ungated", nil, "", nil
	}
	rep := verify.Run(ctx, o.cfg.Verify, p.out, agent.EnvWithoutCredentials(o.cfg.Agents))
	// The gate EXECUTES code the coder wrote, so it gets the same scrutiny the
	// session does -- and it shipped without any (review run 20260813-180828,
	// i48): the step 4 ladder ran before the gate and post-gate reconciliation
	// looked only at worktree files, so a test could edit .git/config, plant a
	// hooks directory, or add one line to .git/info/exclude that makes a source
	// file invisible to `git status`. The commit then omits that file while the
	// gate that "verified" it passed, which is the one claim this pipeline
	// exists to make. Worse, nothing noticed later: the next attempt and the
	// next task re-snapshot at step 2 and would adopt the altered metadata as
	// their own baseline.
	//
	// Stricter than the session's ladder on purpose: a session that commits is
	// merely disobedient and gets a soft reset, but a GATE is fixpoint's own
	// configured command, and every difference here -- HEAD included -- means
	// the run can no longer reason about the repository.
	if err := o.checkGateInvariants(ctx, p, taskID, before); err != nil {
		return "", nil, "", err
	}
	post, err := p.git.TakeCensus(ctx, p.out, 0)
	if err != nil {
		return "", nil, "", err
	}
	diff := implement.ClassifyGateDiff(census, post, o.cfg.Implement.GateGenerated)
	if stuck := removeAll(p.out, diff.Output); len(stuck) > 0 {
		return "", nil, "", fmt.Errorf("could not delete gate output: %s -- it would be committed as if the session had written it", strings.Join(stuck, ", "))
	}
	if len(diff.Output) > 0 {
		o.journal("gate_artifacts", 1, map[string]any{"id": taskID, "removed": diff.Output})
		o.logf("task %s: %d un-ignored gate output path(s) removed; add them to implement.gitignore_seed so the next run stops paying for this", taskID, len(diff.Output))
	}
	if len(diff.MutatedSources) > 0 {
		return "", nil, "gate-mutated-sources: " + strings.Join(diff.MutatedSources, ", ") + " -- a formatter belongs in a task, not a gate", nil
	}
	// A command the operator declared environment-dependent failed: that is a
	// fact about the registry, the proxy or the network, not about the code, and
	// the design spent a whole revision establishing that such facts do not
	// become permanent verdicts (review run 20260813-222753). Reported as
	// infrastructure so the caller burns no attempt and writes no marker.
	if bad := rep.InfraFailures(); len(bad) > 0 {
		names := make([]string, 0, len(bad))
		for _, r := range bad {
			names = append(names, r.Name)
		}
		return "", nil, "", infraGateError{checks: names, detail: gateFailureSummary(rep)}
	}
	if !rep.Passed() {
		return "", nil, "the gate failed: " + gateFailureSummary(rep), nil
	}
	return "passed", diff.GateGenerated, "", nil
}

// infraGateError carries a gate failure that is environmental rather than a
// verdict on the work, so the attempt loop can route it exactly as it routes a
// provider refusal.
type infraGateError struct {
	checks []string
	detail string
}

func (e infraGateError) Error() string {
	return "the gate's environment-dependent check(s) failed: " + strings.Join(e.checks, ", ") + " -- " + e.detail
}

// commitTask stages exactly the computed path set and asserts the tree clean
// after the commit -- which is step 0 for the next task.
func (o *Orchestrator) commitTask(ctx context.Context, p *implementPrep, t implement.Task, attempt int, census implement.Census, gateLabel string, gatePaths []string) (string, error) {
	paths := make([]string, 0, len(census.Untracked)+len(census.Modified)+len(gatePaths))
	for path := range census.Untracked {
		if !isControlArtifact(path) {
			paths = append(paths, path)
		}
	}
	for path := range census.Modified {
		if !isControlArtifact(path) {
			paths = append(paths, path)
		}
	}
	paths = append(paths, gatePaths...)
	header := fmt.Sprintf("fixpoint: %s — %s", t.ID, flattenField(t.Title))
	body := taskCommitBody(t, o.logs.RunID(), attempt, p, gateLabel, gatePaths)
	// Redacted like every other commit fixpoint makes: the body carries the
	// planner's goal and acceptance criteria, and the planner read an untrusted
	// design in a directory that may hold an .env beside it. The implement
	// paths were the only ones skipping this (review run 20260813-161029).
	sha, err := p.col.CommitExact(ctx, agent.RedactSecrets(header), agent.RedactSecrets(body), paths, false)
	if err != nil {
		return "", err
	}
	p.attributed = sha
	return sha, o.assertClean(ctx, p)
}

// reconcileReport is §5.2 step 5: the coder's claim against the tree's state.
// done=false means "implemented, tree dirty: on to the gate"; otherwise the
// verdict stands (why != "" fails the attempt).
func reconcileReport(t implement.Task, report taskReport, clean bool, pl implement.Plan, idx int, outcomes map[string]string) (attemptVerdict, string, bool) {
	// The report must be about THIS task. A stale or mismatched reply -- a
	// harness replaying an earlier turn, a model answering the plan instead of
	// the prompt -- would otherwise be committed under this task's trailers,
	// and the first end-to-end test of this pipeline shipped a fixture that
	// returned the wrong id without failing (review run 20260813-124710).
	if id := strings.TrimSpace(report.Task); id != "" && id != t.ID {
		return attemptVerdict{}, fmt.Sprintf("the session answered for task %q, not %q", id, t.ID), true
	}

	switch report.Status {
	case outcomeSatisfied, outcomeBlocked:
		if !clean {
			return attemptVerdict{}, "the session disclaimed the work but left changes behind", true
		}
		if report.Status == outcomeBlocked {
			if len(report.BlockedOn) == 0 {
				return attemptVerdict{}, "a blocked report must cite the design sections in conflict", true
			}
			return attemptVerdict{kind: attemptBlocked}, "", true
		}
		if !coveredByImplemented(report.CoveredBy, pl, idx, outcomes) {
			return attemptVerdict{}, "already_satisfied must name earlier tasks that were actually IMPLEMENTED -- a failed, blocked, skipped or itself-satisfied task covers nothing", true
		}
		return attemptVerdict{kind: attemptSatisfied}, "", true
	case outcomeImplemented:
		if clean {
			return attemptVerdict{}, "the coder reported implementing " + t.ID + " but left the working tree unchanged", true
		}
		return attemptVerdict{}, "", false
	default:
		return attemptVerdict{}, fmt.Sprintf("unknown status %q", report.Status), true
	}
}

// checkInvariants is §5.2 step 4's ladder: HEAD first (equal -> normal;
// descendant -> soft-reset with the work kept; anything else -> run stop),
// then the rest of the repository against the step 2 snapshot.
func (o *Orchestrator) checkInvariants(ctx context.Context, p *implementPrep, taskID, base string, preState implement.RepoState) error {
	head, err := p.col.HeadSHA(ctx)
	if err != nil {
		return err
	}
	if head != base {
		if !p.git.IsAncestor(ctx, p.out, base, head) {
			return runStopError{fmt.Sprintf("HEAD moved from %.12s to %.12s and the base is not its ancestor", base, head)}
		}
		o.journal("contract_deviation", 1, map[string]any{"id": taskID, "kind": "coder-committed"})
		o.logf("task %s: the coder committed despite being told not to; soft-reset to base, the work is kept", taskID)
		if err := p.col.ResetSoft(ctx, base); err != nil {
			return err
		}
	}
	postState, err := p.git.SnapshotRepoState(ctx, p.out, "main")
	if err != nil {
		return err
	}
	if diff := postState.Diff(preState); len(diff) > 0 {
		// §8 tells an operator whose run stopped here to inspect the invariant
		// snapshot, and named a file nothing wrote (review run 20260813-180828,
		// i23). Both sides go down now, because the DIFF is the diagnosis and a
		// one-line journal string is not the pre-image.
		o.writeRepoState(taskID, "session", preState, postState, diff)
		return runStopError{strings.Join(diff, "; ")}
	}
	return nil
}

// repoStateRecord is `.fixpoint/<ts>/repostate.json`: what the repository looked
// like before, what it looks like now, and which invariants moved.
type repoStateRecord struct {
	SchemaVersion int                 `json:"schema_version"`
	Task          string              `json:"task"`
	When          string              `json:"when"`
	Diff          []string            `json:"diff"`
	Before        implement.RepoState `json:"before"`
	After         implement.RepoState `json:"after"`
}

// writeRepoState persists an invariant comparison at the run root: what the
// repository looked like before, what it looks like now, and which invariants
// moved. Best-effort -- the run is already stopping, and the error that matters
// is the one being reported.
func (o *Orchestrator) writeRepoState(taskID, when string, before, after implement.RepoState, diff []string) {
	b, err := json.MarshalIndent(repoStateRecord{
		SchemaVersion: 1, Task: taskID, When: when, Diff: diff, Before: before, After: after,
	}, "", "  ")
	if err == nil {
		_, err = o.logs.RunState("repostate.json", append(b, '\n'))
	}
	if err != nil {
		o.logf("WARNING: could not write repostate.json (%v)", err)
	}
}

// checkGateInvariants is the same comparison after the gate has run project
// code, with no ladder: the gate is not supposed to touch the repository at all,
// so HEAD moving is itself a mutation rather than something to reset away.
func (o *Orchestrator) checkGateInvariants(ctx context.Context, p *implementPrep, taskID string, before attemptBase) error {
	head, err := p.col.HeadSHA(ctx)
	if err != nil {
		return err
	}
	postState, err := p.git.SnapshotRepoState(ctx, p.out, "main")
	if err != nil {
		return err
	}
	diff := postState.Diff(before.repo)
	if head != before.head {
		diff = append(diff, fmt.Sprintf("the gate moved HEAD from %.12s to %.12s", before.head, head))
	}
	if len(diff) == 0 {
		return nil
	}
	o.writeRepoState(taskID, "gate", before.repo, postState, diff)
	o.journal("gate_mutated_repository", 1, map[string]any{"id": taskID, "what": diff})
	return runStopError{"the gate changed the repository itself, not just the worktree: " + strings.Join(diff, "; ")}
}

// cleanUpInterrupted discards whatever an interrupted attempt left behind. It
// runs on a FRESH, bounded context because the run's own is already dead: a
// canceled context makes every git command fail instantly, which is precisely
// how the residue came to be left.
//
// A failure here is logged rather than returned: the run is already ending,
// and the operator needs to be told the tree is dirty far more than the caller
// needs another error.
func (o *Orchestrator) cleanUpInterrupted(ctx context.Context, p *implementPrep) {
	if ctx.Err() == nil || p.col == nil {
		return
	}
	fresh, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()
	// HEAD FIRST, because a clean tree is not the same as an unchanged history.
	// A session that committed and then died -- or was canceled before the step 4
	// ladder could soft-reset it -- leaves its work IN a commit, so GitClean
	// returns true and the discard below never runs. The repository was then
	// unlocked with an unverified, unattributed commit at HEAD, contradicting the
	// §8 promise that only previously gated commits remain (review run
	// 20260814-024946). Soft-reset back to what fixpoint last attributed; the
	// bytes become working-tree changes and the discard takes them from there.
	if p.attributed != "" {
		if head, err := p.col.HeadSHA(fresh); err == nil && head != p.attributed {
			o.logf("the run was interrupted after a session committed on its own; resetting HEAD from %.12s back to %.12s, the last commit fixpoint attributed", head, p.attributed)
			o.journal("interrupted_reset", 1, map[string]any{"from": head, "to": p.attributed})
			if err := p.col.ResetSoft(fresh, p.attributed); err != nil {
				o.logf("WARNING: could not reset HEAD to %.12s (%v); %s carries a commit fixpoint did not make -- inspect it before continuing", p.attributed, err, p.out)
				return
			}
		}
	}
	clean, err := p.col.GitClean(fresh)
	if err != nil || clean {
		return
	}
	o.logf("the run was interrupted with the tree dirty; discarding the in-flight attempt so the project stays consistent")
	if _, err := p.col.StashDirty(fresh, fmt.Sprintf("fixpoint %s: interrupted (discarded)", o.logs.RunID())); err != nil {
		o.logf("WARNING: could not discard the interrupted attempt (%v); %s is left dirty -- `git stash` there before continuing", err, p.out)
		return
	}
	if clean, err := p.col.GitClean(fresh); err != nil || !clean {
		o.logf("WARNING: %s is still dirty after the discard; inspect it before continuing", p.out)
	}
}

// discardAttempt is the single exit for every attempt that does not reach a
// commit (§5.3): stash tracked and untracked changes together, delete
// session-created ignored paths, then assert the tree clean -- the assertion
// is the thing that turns an incomplete discard into a loud failure instead
// of a silent attribution bug.
func (o *Orchestrator) discardAttempt(ctx context.Context, p *implementPrep, taskID string, attempt int) error {
	msg := fmt.Sprintf("fixpoint %s: %s attempt %d (discarded)", o.logs.RunID(), taskID, attempt)
	if _, err := p.col.StashDirty(ctx, msg); err != nil {
		return err
	}
	return o.assertClean(ctx, p)
}

// assertClean is §5.2 step 0's check, reused everywhere a phase promises to
// leave the tree clean. A failure stops the run: the residue is fixpoint's.
func (o *Orchestrator) assertClean(ctx context.Context, p *implementPrep) error {
	clean, err := p.col.GitClean(ctx)
	if err != nil {
		return err
	}
	if !clean {
		return runStopError{"a discard or commit left the tree dirty; the residue is fixpoint's, not the coder's -- report this"}
	}
	return nil
}

// markerCommit writes the empty outcome marker (§5.4): every processed task
// leaves exactly one commit, because the trailers are the resume state.
func (o *Orchestrator) markerCommit(ctx context.Context, p *implementPrep, t implement.Task, outcome, reasonKey, reasonValue string) (string, error) {
	header := fmt.Sprintf("fixpoint: %s — %s [%s]", t.ID, flattenField(t.Title), outcome)
	lines := []string{
		"Fixpoint-Run: " + o.logs.RunID(),
		"Fixpoint-Marker: 1",
		"Fixpoint-Task: " + t.ID,
		"Fixpoint-Outcome: " + outcome,
	}
	if reasonValue != "" {
		lines = append(lines, "Fixpoint-Reason: "+flattenField(clampLine(reasonKey+"="+reasonValue)))
	}
	sha, err := p.col.CommitExact(ctx, agent.RedactSecrets(header), agent.RedactSecrets(strings.Join(lines, "\n")), nil, true)
	if err == nil {
		p.attributed = sha
	}
	return sha, err
}

// cleanCheck clones HEAD and runs the gate in the clone (§7.2): the committed
// bytes alone must satisfy the gate.
func (o *Orchestrator) cleanCheck(ctx context.Context, p *implementPrep) error {
	scratch, err := o.logs.ScratchDir("clean-check")
	if err != nil {
		return err
	}
	rep, err := p.git.CleanCheck(ctx, p.out, scratch, o.cfg.Verify, agent.EnvWithoutCredentials(o.cfg.Agents))
	passed := err == nil && rep.Passed()
	o.journal("clean_check_finished", 1, map[string]any{"cadence": o.cfg.Implement.CleanCheck, "passed": passed})
	if err != nil {
		return err
	}
	// An environment-dependent check failing here says nothing about the commits,
	// exactly as it says nothing during a task (§5.4). The clean check was the one
	// gate that never consulted the classification, so a registry 503 during the
	// final clone turned a completed run into an incomplete one and told the
	// operator their history did not build (review run 20260814-024946).
	//
	// ONLY when nothing else failed, and that condition is the whole of the fix
	// here (review run 20260814-191024). The first version returned before
	// looking at the real checks, so `[{install, infra}, {build}]` with a missing
	// go.mod failed BOTH in the clone and was reported as "the clone proves
	// nothing" -- the run terminated implemented over a HEAD that does not build,
	// which is precisely the state clean_check exists to catch.
	//
	// Deliberately the opposite precedence to gatePhase, because the consequence
	// is opposite. There, suppressing means retrying a task and committing
	// nothing, so "the environment is broken, we cannot tell" is the safe answer.
	// Here, suppressing means declaring the whole run a success. A real failure
	// alongside a broken environment is still a real failure.
	verdicts := 0 // failures that are a claim about the CODE, not the environment
	for _, f := range rep.Failures() {
		if !f.Infra {
			verdicts++
		}
	}
	if bad := rep.InfraFailures(); len(bad) > 0 && verdicts == 0 {
		names := make([]string, 0, len(bad))
		for _, r := range bad {
			names = append(names, r.Name)
		}
		o.journal("clean_check_infra", 1, map[string]any{"checks": names})
		o.logf("clean-check: the environment-dependent check(s) %s could not complete (%s); nothing else failed, so the clone proves nothing either way and the run's verdict is unchanged",
			strings.Join(names, ", "), gateFailureSummary(rep))
		return nil
	}
	if !rep.Passed() {
		return fmt.Errorf("HEAD does not pass the gate in a clean clone; the working tree carries state the commits do not (%s) -- rerun with implement.clean_check: every to find the culprit task", gateFailureSummary(rep))
	}
	o.logf("clean-check: a fresh clone of HEAD passes the gate")
	return nil
}

// vacuousFraction is already_satisfied over the WHOLE PLAN (§5.4: "N of M
// tasks"), returning both numbers so the refusal can quote them.
//
// The denominator is the plan, not the tasks processed so far, and the first
// review of this code found the running denominator (run 20260813-124710):
// with it, the first legitimately-satisfied task at position 2 reads as 50%
// and aborts a 40-task run over one over-decomposition. The guard exists to
// catch a plan that is mostly air, which is only knowable against the plan.
func vacuousFraction(tasks []model.TaskOutcome, planned int) (int, float64) {
	if planned <= 0 {
		return 0, 0
	}
	vac := 0
	for _, t := range tasks {
		// A carried already_satisfied is still an already_satisfied: the guard
		// measures how much of the PLAN decomposed to nothing, and relabelling a
		// task `carried` on resume must not launder it out of that fraction
		// (review run 20260818-234734).
		if t.Outcome == outcomeSatisfied || t.CarriedOutcome == outcomeSatisfied {
			vac++
		}
	}
	return vac, float64(vac) / float64(planned)
}

// coveredByImplemented checks an already_satisfied report: every named id must
// be an EARLIER task that actually produced a code commit (§5.2 step 5) -- a
// marker is not an implementation.
//
// Both halves are load-bearing, and the first review of this code found the
// second one missing (run 20260813-124710): checking only the position lets a
// task cite an earlier FAILED task, collect an already_satisfied marker, and
// release its own dependents even though nothing built the work. That is
// exactly the hole the corroboration rule exists to close, reopened one field
// away from where it was closed.
func coveredByImplemented(ids []string, pl implement.Plan, idx int, outcomes map[string]string) bool {
	if len(ids) == 0 {
		return false
	}
	earlier := map[string]bool{}
	for i := range idx {
		earlier[pl.Tasks[i].ID] = true
	}
	for _, id := range ids {
		if !earlier[id] || outcomes[id] != outcomeImplemented {
			return false
		}
	}
	return true
}

func isControlArtifact(path string) bool {
	for _, ca := range implement.ControlArtifacts {
		if path == ca {
			return true
		}
	}
	return false
}

// planShape lists every task with its position relative to idx, so the coder
// knows what exists without receiving the whole plan (§6).
func planShape(pl implement.Plan, idx int) string {
	var b strings.Builder
	for i, t := range pl.Tasks {
		state := "pending"
		switch {
		case i < idx:
			state = "done or recorded"
		case i == idx:
			state = "THIS TASK"
		}
		fmt.Fprintf(&b, "- %s: %s (%s)\n", t.ID, t.Title, state)
	}
	return b.String()
}

func taskMarkdown(t implement.Task, report taskReport, err error) string {
	if err != nil {
		return fmt.Sprintf("# task %s (failed)\n\n%v\n", t.ID, err)
	}
	return fmt.Sprintf("# task %s — %s\n\nstatus: %s\n\n%s\n", t.ID, t.Title, report.Status, report.Notes)
}

// taskCommitBody renders the task commit's body and trailers (§5.2 step 8).
func taskCommitBody(t implement.Task, runID string, attempt int, p *implementPrep, gateLabel string, gatePaths []string) string {
	var b strings.Builder
	b.WriteString(flattenField(t.Goal) + "\n\nAcceptance:\n")
	for _, a := range t.Acceptance {
		b.WriteString("- " + flattenField(a) + "\n")
	}
	b.WriteString("\n")
	lines := []string{
		"Fixpoint-Run: " + runID,
		"Fixpoint-Task: " + t.ID,
		"Fixpoint-Outcome: implemented",
		fmt.Sprintf("Fixpoint-Attempt: %d", attempt),
		"Design-SHA256: " + p.designSHA,
	}
	if gateLabel == "ungated" {
		lines = append(lines, "Verification: skipped (no operator gates configured)")
	} else {
		lines = append(lines, "Verification: "+gateLabel)
	}
	for _, g := range gatePaths {
		lines = append(lines, "Fixpoint-Gate-Wrote: "+g)
	}
	b.WriteString(strings.Join(lines, "\n"))
	return b.String()
}

func gateFailureSummary(rep verify.Report) string {
	var failed []string
	for _, r := range rep.Results {
		if !r.Passed && !r.Optional {
			failed = append(failed, r.Name)
		}
	}
	return strings.Join(failed, ", ")
}

func verdictReason(why string) string {
	return clampLine(why)
}

// clampLine bounds one trailer value (§5.4: a versioned marker schema with a
// bounded, sanitized detail line; the unabridged text lives in the journal).
func clampLine(s string) string {
	const limit = 200
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return s[:limit]
}
