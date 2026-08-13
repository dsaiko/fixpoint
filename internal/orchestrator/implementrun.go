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
)

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
	out          string // the write-target, claimed by SCAFFOLD
	designBytes  []byte
	designSHA    string
	designPath   string // as the operator named it
	outline      implement.Outline
	material     string // the design, collected and fenced for the planner
	profile      string // the effective-gate digest
	gateCommands []string
	col          *target.Collector // over out; valid after SCAFFOLD
	bootstrapSHA string
	baseline     implement.RepoState
}

// errRunStop is a repository-invariant failure: the run cannot reason about
// the tree any longer, so it stops rather than failing one task (§8).
type runStopError struct{ why string }

func (e runStopError) Error() string { return "repository invariant failed: " + e.why }

// runPipeline dispatches the non-loop pipelines; handled=false means the run
// is loop-shaped and run() continues.
func (o *Orchestrator) runPipeline(ctx context.Context, sum *model.RunSummary) (bool, error) {
	switch {
	case o.cfg.IsCreate():
		return true, o.runCreate(ctx, sum)
	case o.cfg.IsImplement():
		return true, o.runImplement(ctx, sum)
	}
	return false, nil
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

	prep, err := o.prepareImplement(ctx)
	if err != nil {
		return err
	}
	pl, err := o.planPhase(ctx, &rec, prep)
	if err != nil {
		return err
	}
	if err := o.scaffoldPhase(ctx, sum, prep, pl); err != nil {
		return err
	}
	return o.buildPhase(ctx, sum, &rec, prep, pl, started)
}

// prepareImplement is every refusal that must fire before a session is paid
// for: the write-target check, the design snapshot, and the outline.
func (o *Orchestrator) prepareImplement(ctx context.Context) (*implementPrep, error) {
	p := &implementPrep{out: o.cfg.Create.Out}
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

	if o.cfg.Target.Document == "" {
		return nil, errors.New("implement: the target must be a design document file (-target DESIGN.md)")
	}
	p.designPath = filepath.Join(o.cfg.Target.Path, o.cfg.Target.Document)

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

	// The outline comes from the snapshot, never from the planner (§4.2 rule
	// 6); an unreadable outline is a preflight refusal.
	if p.outline, err = implement.ExtractOutline(string(p.designBytes)); err != nil {
		return nil, fmt.Errorf("implement: %w", err)
	}
	o.logf("coverage outline: %q level, %d heading(s)", strings.Repeat("#", p.outline.Level), len(p.outline.Headings))

	if p.material, err = o.snapshotMaterial(ctx, snap); err != nil {
		return nil, err
	}
	p.gateCommands = implement.RenderCommands(o.cfg.Verify)
	p.profile = implement.VerifyProfile(p.gateCommands, string(o.cfg.Verify.Policy), o.cfg.Verify.Timeout.Std(), o.cfg.Implement.GateGenerated)
	return p, nil
}

// planPhase runs the one read-only planner session and validates its product.
// A plan that fails costs exactly one planner session and zero coder sessions.
func (o *Orchestrator) planPhase(ctx context.Context, rec *model.RoundRecord, p *implementPrep) (implement.Plan, error) {
	o.phase("PLAN  one %s session over the design snapshot", o.cfg.Roles.Planner.Agent)
	var pl implement.Plan

	rules := implement.Rules{
		MaxTasks:        o.cfg.Implement.MaxTasks,
		MaxFilesPerTask: o.cfg.Implement.MaxFilesPerTask,
		Outline:         p.outline,
		CoverageChecked: true,
		MaxTaskAttempts: o.cfg.Implement.MaxTaskAttempts,
		SessionTimeout:  o.cfg.Agents[o.cfg.Roles.Coder.Agent].Timeout.Std(),
		GateWorst:       implement.GateWorst(o.cfg.Verify),
		MaxRunDuration:  o.cfg.Implement.MaxRunDuration.Std(),
	}
	admitted := o.cfg.Implement.MaxTasks
	if !rules.FitSkipped() {
		perTask := time.Duration(rules.MaxTaskAttempts) * (rules.SessionTimeout + rules.GateWorst + time.Minute)
		if n := int(rules.MaxRunDuration / perTask); n < admitted {
			admitted = n
		}
	} else {
		o.logf("plan: the fit check is skipped (a session or gate timeout is not available as a number); the deadline is enforced between tasks only")
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
	o.logStep("plan", planner, o.cfg.Roles.Planner.Prompt, 1, perr == nil, pl, planMarkdown(pl, perr), res, "")
	if perr != nil {
		return pl, fmt.Errorf("plan: %w", perr)
	}

	pl.Provenance = &implement.Provenance{
		SchemaVersion: 1,
		RunID:         o.logs.RunID(),
		DesignPath:    p.designPath,
		DesignSHA256:  p.designSHA,
		Planner:       implement.PlannerLabel(planner, false, ""),
		Coder:         o.cfg.Roles.Coder.Agent,
		VerifyProfile: p.profile,
	}
	o.journal("plan_finished", 1, map[string]any{
		"tasks":    len(pl.Tasks),
		"coverage": fmt.Sprintf("%q (%d headings)", strings.Repeat("#", p.outline.Level), len(p.outline.Headings)),
	})
	o.endPhase("PLAN  %d task(s), coverage accounted", len(pl.Tasks))
	return pl, nil
}

// planMarkdown renders the plan step's durable artifact.
func planMarkdown(pl implement.Plan, err error) string {
	if err != nil {
		return fmt.Sprintf("# plan (refused)\n\n%v\n", err)
	}
	return implement.RenderMarkdown(pl, implement.RenderHeader{Coverage: "validated"})
}

// scaffoldPhase claims the write-target and makes the bootstrap commit (§5.1).
func (o *Orchestrator) scaffoldPhase(ctx context.Context, sum *model.RunSummary, p *implementPrep, pl implement.Plan) error {
	o.phase("SCAFFOLD  %s", p.out)
	coverage := fmt.Sprintf("%q (%d headings)", strings.Repeat("#", p.outline.Level), len(p.outline.Headings))
	planJSON, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return err
	}
	files := map[string][]byte{
		"DESIGN.md":  p.designBytes, // byte-exact: the hash must match what review-design approved (§4.3)
		"PLAN.md":    []byte(implement.RenderMarkdown(pl, implement.RenderHeader{Coverage: coverage, GateCommands: p.gateCommands})),
		"PLAN.json":  append(planJSON, '\n'),
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
	sha, err := implement.Scaffold(ctx, p.col, p.out, files, header, body)
	if err != nil {
		return err
	}
	p.bootstrapSHA = sha
	sum.Deliverable = p.out
	if p.baseline, err = implement.SnapshotRepoState(ctx, p.out, "main"); err != nil {
		return err
	}
	o.endPhase("SCAFFOLD  bootstrap %.12s (not gated, no sources)", sha)
	return nil
}

// buildPhase is the task loop (§5.2): serial, one commit per processed task,
// skips derived from the dependency graph, the deadline checked between
// tasks, and an infrastructure circuit breaker so a provider outage never
// reaches the immutable history (§5.4).
func (o *Orchestrator) buildPhase(ctx context.Context, sum *model.RunSummary, rec *model.RoundRecord, p *implementPrep, pl implement.Plan, started time.Time) error {
	o.phase("BUILD  %d task(s), serially", len(pl.Tasks))
	outcomes := map[string]string{}
	infraStrikes := 0
	incomplete := false

	for i, t := range pl.Tasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if elapsed := time.Since(started); elapsed > o.cfg.Implement.MaxRunDuration.Std() {
			o.logf("BUILD: max_run_duration (%s) reached after %s; stopping before %s", o.cfg.Implement.MaxRunDuration.Std(), elapsed.Round(time.Second), t.ID)
			incomplete = true
			break
		}
		if blockedBy := blockedDependency(t, outcomes); blockedBy != "" {
			outcomes[t.ID] = outcomeSkipped
			sum.Tasks = append(sum.Tasks, model.TaskOutcome{ID: t.ID, Title: t.Title, Outcome: outcomeSkipped, Reason: "dependency " + blockedBy})
			o.journal("task_skipped", 1, map[string]any{"id": t.ID, "blocked_by": blockedBy})
			o.logf("task %s skipped: dependency %s did not land", t.ID, blockedBy)
			incomplete = true
			continue
		}
		bad, stopLoop, err := o.processTask(ctx, sum, rec, p, pl, i, outcomes, &infraStrikes)
		if err != nil {
			return err
		}
		if bad {
			incomplete = true
		}
		if stopLoop {
			break
		}
	}
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
func (o *Orchestrator) processTask(ctx context.Context, sum *model.RunSummary, rec *model.RoundRecord, p *implementPrep, pl implement.Plan, i int, outcomes map[string]string, infraStrikes *int) (bad, stopLoop bool, err error) {
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
	outcomes[t.ID] = out.Outcome
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
	if frac := vacuousFraction(sum.Tasks); frac > o.cfg.Implement.MaxVacuousFrac {
		o.logf("BUILD: %.0f%% of processed tasks were already covered by earlier work, over max_vacuous_frac (%.0f%%); the plan was decomposed badly", frac*100, o.cfg.Implement.MaxVacuousFrac*100)
		return true, true, nil
	}
	return bad, false, nil
}

// errInfraBreaker trips after two consecutive infrastructure failures: the
// run stops incomplete with NO marker for the task in flight (§5.4).
var errInfraBreaker = errors.New("infrastructure circuit breaker")

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
			if *infraStrikes >= 2 {
				return out, errInfraBreaker
			}
			o.logf("task %s: infrastructure failure (%s); the attempt is not counted", t.ID, verdict.why)
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
			if !blockedOnce {
				blockedOnce = true
				priorFailure = "a previous session reported this task blocked on " + strings.Join(report.BlockedOn, ", ") + "; verify independently -- implement it if it can be implemented"
				attempt++
				continue
			}
			out.Outcome = outcomeBlocked
			out.Reason = "design_conflict: " + strings.Join(report.BlockedOn, ", ")
			out.Attempts = attempt
			sha, err := o.markerCommit(ctx, p, t, outcomeBlocked, "blocked_on", strings.Join(report.BlockedOn, ","))
			if err != nil {
				return out, err
			}
			out.SHA = sha
			o.journal("task_finished", 1, map[string]any{"id": t.ID, "status": outcomeBlocked, "notes": flattenField(report.Notes)})
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
	preState, err := implement.SnapshotRepoState(ctx, p.out, "main")
	if err != nil {
		return agent.Result{}, report, attemptVerdict{}, err
	}
	preIgnored, err := implement.TakeIgnoredCensus(ctx, p.out)
	if err != nil {
		return agent.Result{}, report, attemptVerdict{}, err
	}

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

	// Infrastructure is not an outcome (§5.4): the provider refused, or the
	// session died leaving nothing.
	if res.Err != nil && (res.ProviderStatus != 0 || strings.TrimSpace(res.Stdout) == "") {
		if err := o.discardAttempt(ctx, p, t.ID, attempt); err != nil {
			return res, report, attemptVerdict{}, err
		}
		return res, report, attemptVerdict{kind: attemptInfra, why: fmt.Sprintf("provider status %d", res.ProviderStatus)}, nil
	}

	// Step 4: the invariant ladder -- HEAD first, then the repository.
	if err := o.checkInvariants(ctx, p, t.ID, base, preState); err != nil {
		return res, report, attemptVerdict{}, err
	}

	// Step 5: reconcile -- control artifacts first (§4.3).
	if changed, err := implement.ControlArtifactsChanged(ctx, p.out, p.bootstrapSHA); err != nil {
		return res, report, attemptVerdict{}, err
	} else if len(changed) > 0 {
		if err := implement.RestoreControlArtifacts(ctx, p.out, p.bootstrapSHA, changed); err != nil {
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
	return o.reconcileAttempt(ctx, p, pl, idx, attempt, preIgnored, res, report)
}

// reconcileAttempt is §5.2 steps 5-8 for a session that returned a report.
func (o *Orchestrator) reconcileAttempt(ctx context.Context, p *implementPrep, pl implement.Plan, idx, attempt int, preIgnored map[string]implement.IgnoredStat, res agent.Result, report taskReport) (agent.Result, taskReport, attemptVerdict, error) {
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

	if verdict, why, done := reconcileReport(t, report, clean, pl, idx); done {
		if why != "" {
			return fail(why)
		}
		return res, report, verdict, nil
	}

	// Step 6: the census, the byte bound, the ignored reconciliation.
	census, err := implement.TakeCensus(ctx, p.out, o.cfg.Implement.MaxTaskBytes.Int64())
	var bb implement.ByteBoundError
	if errors.As(err, &bb) {
		// Checkout-and-clean, not a stash: the oversized tree must not enter
		// the object database (§5.3).
		if derr := p.col.DiscardClean(ctx); derr != nil {
			return res, report, attemptVerdict{}, derr
		}
		if err := o.assertClean(ctx, p); err != nil {
			return res, report, attemptVerdict{}, err
		}
		return res, report, attemptVerdict{kind: attemptFailed, why: bb.Error()}, nil
	}
	if err != nil {
		return res, report, attemptVerdict{}, err
	}
	postIgnored, err := implement.TakeIgnoredCensus(ctx, p.out)
	if err != nil {
		return res, report, attemptVerdict{}, err
	}
	created, modified := implement.DiffIgnored(preIgnored, postIgnored)
	for _, path := range created {
		_ = os.RemoveAll(filepath.Join(p.out, path))
	}
	if len(created) > 0 || len(modified) > 0 {
		o.journal("ignored_paths_diff", 1, map[string]any{"id": t.ID, "created": created, "modified": modified})
		if len(created) > 0 {
			o.logf("task %s: %d session-created ignored path(s) deleted before the gate; the gate recreates what it needs from committed sources", t.ID, len(created))
		}
	}

	// Step 7: the gate.
	gateLabel, gatePaths, why, err := o.gatePhase(ctx, p, t.ID, census)
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

// gatePhase is §5.2 step 7: run the gate, classify every difference, remove
// output after EVERY gate run (a failed gate's droppings must not sit in the
// tree for the next attempt). A non-empty `why` fails the attempt.
func (o *Orchestrator) gatePhase(ctx context.Context, p *implementPrep, taskID string, census implement.Census) (label string, gatePaths []string, why string, err error) {
	if len(o.cfg.Verify.Commands) == 0 {
		return "ungated", nil, "", nil
	}
	rep := verify.Run(ctx, o.cfg.Verify, p.out, agent.EnvWithoutCredentials(o.cfg.Agents))
	post, err := implement.TakeCensus(ctx, p.out, 0)
	if err != nil {
		return "", nil, "", err
	}
	diff := implement.ClassifyGateDiff(census, post, o.cfg.Implement.GateGenerated)
	for _, path := range diff.Output {
		_ = os.RemoveAll(filepath.Join(p.out, path))
	}
	if len(diff.Output) > 0 {
		o.journal("gate_artifacts", 1, map[string]any{"id": taskID, "removed": diff.Output})
		o.logf("task %s: %d un-ignored gate output path(s) removed; add them to implement.gitignore_seed so the next run stops paying for this", taskID, len(diff.Output))
	}
	if len(diff.MutatedSources) > 0 {
		return "", nil, "gate-mutated-sources: " + strings.Join(diff.MutatedSources, ", ") + " -- a formatter belongs in a task, not a gate", nil
	}
	if !rep.Passed() {
		return "", nil, "the gate failed: " + gateFailureSummary(rep), nil
	}
	return "passed", diff.GateGenerated, "", nil
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
	sha, err := p.col.CommitExact(ctx, header, body, paths, false)
	if err != nil {
		return "", err
	}
	return sha, o.assertClean(ctx, p)
}

// reconcileReport is §5.2 step 5: the coder's claim against the tree's state.
// done=false means "implemented, tree dirty: on to the gate"; otherwise the
// verdict stands (why != "" fails the attempt).
func reconcileReport(t implement.Task, report taskReport, clean bool, pl implement.Plan, idx int) (attemptVerdict, string, bool) {
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
		if !coveredByImplemented(report.CoveredBy, pl, idx) {
			return attemptVerdict{}, "already_satisfied must name earlier tasks that actually implemented the work", true
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
		if !isAncestor(ctx, p.out, base, head) {
			return runStopError{fmt.Sprintf("HEAD moved from %.12s to %.12s and the base is not its ancestor", base, head)}
		}
		o.journal("contract_deviation", 1, map[string]any{"id": taskID, "kind": "coder-committed"})
		o.logf("task %s: the coder committed despite being told not to; soft-reset to base, the work is kept", taskID)
		if err := p.col.ResetSoft(ctx, base); err != nil {
			return err
		}
	}
	postState, err := implement.SnapshotRepoState(ctx, p.out, "main")
	if err != nil {
		return err
	}
	if diff := postState.Diff(preState); len(diff) > 0 {
		return runStopError{strings.Join(diff, "; ")}
	}
	return nil
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
	return p.col.CommitExact(ctx, header, strings.Join(lines, "\n"), nil, true)
}

// cleanCheck clones HEAD and runs the gate in the clone (§7.2): the committed
// bytes alone must satisfy the gate.
func (o *Orchestrator) cleanCheck(ctx context.Context, p *implementPrep) error {
	scratch, err := o.logs.ScratchDir("clean-check")
	if err != nil {
		return err
	}
	rep, err := implement.CleanCheck(ctx, p.out, scratch, o.cfg.Verify, agent.EnvWithoutCredentials(o.cfg.Agents))
	passed := err == nil && rep.Passed()
	o.journal("clean_check_finished", 1, map[string]any{"cadence": o.cfg.Implement.CleanCheck, "passed": passed})
	if err != nil {
		return err
	}
	if !rep.Passed() {
		return fmt.Errorf("HEAD does not pass the gate in a clean clone; the working tree carries state the commits do not (%s) -- rerun with implement.clean_check: every to find the culprit task", gateFailureSummary(rep))
	}
	o.logf("clean-check: a fresh clone of HEAD passes the gate")
	return nil
}

// blockedDependency returns the first (transitive) dependency that did not
// land. Skips are DERIVED, never recorded (§5.4): they are a pure function of
// the plan and the failed/blocked outcomes.
func blockedDependency(t implement.Task, outcomes map[string]string) string {
	for _, d := range t.DependsOn {
		switch outcomes[d] {
		case outcomeFailed, "blocked", "skipped":
			return d
		}
	}
	return ""
}

// vacuousFraction is already_satisfied over processed tasks (§5.4): planners
// over-decompose, but a third of the plan being air means it was decomposed
// badly.
func vacuousFraction(tasks []model.TaskOutcome) float64 {
	if len(tasks) == 0 {
		return 0
	}
	vac := 0
	for _, t := range tasks {
		if t.Outcome == outcomeSatisfied {
			vac++
		}
	}
	return float64(vac) / float64(len(tasks))
}

// coveredByImplemented checks an already_satisfied report: every named id
// must be an EARLIER task that actually produced a code commit (§5.2 step 5)
// -- a marker is not an implementation.
func coveredByImplemented(ids []string, pl implement.Plan, idx int) bool {
	if len(ids) == 0 {
		return false
	}
	earlier := map[string]bool{}
	for i := range idx {
		earlier[pl.Tasks[i].ID] = true
	}
	for _, id := range ids {
		if !earlier[id] {
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

// isAncestor reports whether base is an ancestor of head.
func isAncestor(ctx context.Context, dir, base, head string) bool {
	return implement.IsAncestor(ctx, dir, base, head)
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
