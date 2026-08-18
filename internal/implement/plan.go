// Package implement holds the implement-design pipeline's deterministic
// mechanics: the plan as a value (schema, validation, rendering), the design's
// outline for the coverage rule, and the effective-gate digest. The
// orchestrator drives the phases; this package is everything about them that
// is neither agent scheduling nor git. Specified in docs/design/DESIGN.md,
// which was reviewed by review-design across three revisions; the decisions
// embodied here cite the sections that force them.
package implement

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
)

// Plan is the planner's product after validation: the ordered task list and
// the coverage map. The planner owns project, coverage and each task's fields;
// fixpoint owns Provenance and injects it AFTER parsing, so no model can
// author or overwrite it (§4.1).
type Plan struct {
	SchemaVersion int             `json:"schema_version"`
	Project       Project         `json:"project"`
	Coverage      []CoverageEntry `json:"coverage"`
	Tasks         []Task          `json:"tasks"`
	Provenance    *Provenance     `json:"provenance,omitempty"`
}

// Project names what is being built, for commit subjects and the scoreboard.
type Project struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

// CoverageEntry accounts for one design section: mapped to tasks, or ruled out
// of scope with a reason. Every outline heading must appear (§4.2 rule 6).
type CoverageEntry struct {
	Heading    string   `json:"heading"`
	Tasks      []string `json:"tasks"`
	OutOfScope string   `json:"out_of_scope,omitempty"`
}

// Task is one coder session and one commit.
type Task struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Goal       string   `json:"goal"`
	Acceptance []string `json:"acceptance"`
	Files      []string `json:"files"`
	DependsOn  []string `json:"depends_on"`
	DesignRefs []string `json:"design_refs"`
}

// Provenance is fixpoint's, never the planner's. A planner-emitted provenance
// key is discarded before validation with a contract_deviation event -- not
// merged, not trusted, not an error (§4.1). The planner field is never forged:
// an operator-supplied plan is stamped as such (§7.4).
type Provenance struct {
	SchemaVersion int    `json:"schema_version"`
	RunID         string `json:"run_id"`
	DesignPath    string `json:"design_path"`
	DesignSHA256  string `json:"design_sha256"`
	Planner       string `json:"planner"`
	Coder         string `json:"coder"`
	VerifyProfile string `json:"verify_profile"`
}

// ControlArtifacts are the four files no task may name and no session may edit
// (§4.3): the design of record, the two plan renderings, and the exemption set
// that decides what the census can see.
var ControlArtifacts = []string{"DESIGN.md", "PLAN.md", "PLAN.json", ".gitignore"}

// StripProvenance removes a planner-emitted provenance block, reporting
// whether one was present so the caller can journal the contract deviation.
func StripProvenance(p *Plan) bool {
	had := p.Provenance != nil
	p.Provenance = nil
	return had
}

// Rules parameterizes validation with the run's configuration and the
// document's extracted outline. Zero-valued timing fields skip the fit rule
// (the skip is STATED by the caller, §4.2 rule 7).
type Rules struct {
	MaxTasks        int
	MaxFilesPerTask int
	// Outline is the design's extracted skeleton; CoverageChecked false skips
	// rule 6, and every report says so. Set false by -no-coverage-check (§7.4),
	// which is a per-invocation waiver and never a config key -- see
	// config.Implement.NoCoverageCheck.
	Outline         Outline
	CoverageChecked bool
	// Fit inputs: the worst case the configuration permits (§4.2 rule 7).
	MaxTaskAttempts int
	SessionTimeout  time.Duration
	// GateWorst is the SUM of every configured command's timeout -- the verify
	// executor applies the timeout per command and runs them sequentially.
	GateWorst      time.Duration
	MaxRunDuration time.Duration
	// PlanOverhead is what the run spends BEFORE the first task: the preflight
	// ping, the design snapshot and the planner session. Counted because the
	// deadline clock starts when the run starts, not when BUILD does, so a
	// budget that omitted this was measuring a shorter interval than the clock
	// enforced -- and rule 7's own claim is that "the number the refusal quotes
	// and the number the run can spend are the same number" (review run
	// 20260813-222753). A slow planner could otherwise eat a task's worth of a
	// deadline the plan was admitted against.
	PlanOverhead time.Duration
	// CleanCheck is implement.clean_check, because it BUYS GATE RUNS the
	// arithmetic must count: "every" adds a full clone gate after each
	// implemented task, "last" adds one after the loop. Left out of the formula
	// when it shipped, which re-opened the exact trap rule 7 exists to close --
	// a 15-task plan the validator called legal needed ~38h against a 32h
	// deadline at clean_check: every, so it hit the deadline around task 12,
	// after the money was spent (review run 20260813-180828, i43).
	CleanCheck string
}

// PerTaskWorst is the worst case one task may cost: every attempt the
// configuration permits, each paying a full session, a full gate and the
// bookkeeping allowance.
//
// Exported and used by BOTH the validator and the planner prompt's cap.
// They used to compute it separately from the same formula, so a change to
// either drifted: the planner would be told a plan fits that the validator then
// refused, and the refusal would quote arithmetic the run never used (i13).
func (r Rules) PerTaskWorst() time.Duration {
	return time.Duration(r.MaxTaskAttempts) * (r.SessionTimeout + r.GateWorst + taskOverhead)
}

// RunWorst is what a plan of n tasks may cost end to end: every task's worst
// case plus the clean-clone gate runs clean_check buys.
func (r Rules) RunWorst(n int) time.Duration {
	total := r.PlanOverhead + time.Duration(n)*r.PerTaskWorst()
	// The config's own constants, so the arithmetic and the validator that
	// accepts the value cannot disagree about its spelling.
	switch r.CleanCheck {
	case config.CleanCheckEvery:
		total += time.Duration(n) * r.GateWorst
	case config.CleanCheckLast:
		total += r.GateWorst
	}
	return total
}

// Admitted is the largest task count that fits the deadline -- the number the
// planner is given as its cap, computed from the same arithmetic that will
// judge what it returns. Returns ceiling when the fit rule is skipped.
func (r Rules) Admitted(ceiling int) int {
	if r.FitSkipped() {
		return ceiling
	}
	n := 0
	for n < ceiling && r.RunWorst(n+1) <= r.MaxRunDuration {
		n++
	}
	return n
}

// taskOverhead is the per-task bookkeeping allowance in the fit arithmetic:
// census, reconciliation, commit, journal.
const taskOverhead = time.Minute

var idPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,16}$`)

// Validate applies every deterministic rule of §4.2, in order, and returns the
// first refusal. A plan that fails costs exactly one planner session and zero
// coder sessions.
func Validate(p Plan, r Rules) error {
	if p.SchemaVersion != 1 {
		return fmt.Errorf("plan: schema_version %d is not 1; the plan and this tool disagree about what the fields mean", p.SchemaVersion)
	}
	if err := validateTasks(p, r); err != nil {
		return err
	}
	if err := validateCoverage(p, r); err != nil {
		return err
	}
	return validateFit(len(p.Tasks), r)
}

func validateTasks(p Plan, r Rules) error {
	// Rule 1: bounded count. Over the cap is a refusal, not a truncation.
	if len(p.Tasks) < 1 {
		return errors.New("plan: no tasks; a design that decomposes to nothing was not decomposed")
	}
	if len(p.Tasks) > r.MaxTasks {
		return fmt.Errorf("plan: %d tasks exceed implement.max_tasks (%d) -- split the design; a design needing this many sessions is a program, not a run", len(p.Tasks), r.MaxTasks)
	}
	seen := map[string]int{}
	for i, t := range p.Tasks {
		// Rule 2: ids land in log filenames and commit subjects.
		if !idPattern.MatchString(t.ID) {
			return fmt.Errorf("plan: task %d id %q must match %s", i, t.ID, idPattern)
		}
		if j, dup := seen[t.ID]; dup {
			return fmt.Errorf("plan: tasks %d and %d share id %q", j, i, t.ID)
		}
		seen[t.ID] = i
		// Rule 3: backward-only dependencies make the array a valid execution
		// order with no topological sort and a cyclic plan impossible.
		for _, d := range t.DependsOn {
			j, known := seen[d]
			if !known || j == i {
				return fmt.Errorf("plan: task %s depends on %q, which is not an EARLIER task; dependencies point backwards only", t.ID, d)
			}
		}
		if err := validateTaskFiles(t, r); err != nil {
			return err
		}
		// Rule 5: a task nobody can state acceptance criteria for is two tasks
		// or none, and its commit body would say nothing.
		if len(t.Acceptance) == 0 {
			return fmt.Errorf("plan: task %s has no acceptance criteria", t.ID)
		}
		if strings.TrimSpace(t.Title) == "" || strings.TrimSpace(t.Goal) == "" {
			return fmt.Errorf("plan: task %s needs both a title and a goal", t.ID)
		}
	}
	return nil
}

// validateTaskFiles is rule 4: the advisory file list still refuses paths that
// could only come from a misled planner.
func validateTaskFiles(t Task, r Rules) error {
	if len(t.Files) == 0 {
		return fmt.Errorf("plan: task %s names no files; the list is advisory but empty means the planner did not think about where the work lands", t.ID)
	}
	if len(t.Files) > r.MaxFilesPerTask {
		return fmt.Errorf("plan: task %s names %d files, over implement.max_files_per_task (%d)", t.ID, len(t.Files), r.MaxFilesPerTask)
	}
	for _, f := range t.Files {
		switch {
		case f == "", strings.HasPrefix(f, "/"), strings.Contains(f, ".."):
			return fmt.Errorf("plan: task %s file %q must be a relative slash path free of dot-dot segments", t.ID, f)
		case strings.HasPrefix(f, ".git/"), strings.HasPrefix(f, ".fixpoint/"):
			return fmt.Errorf("plan: task %s file %q is under a tool-owned directory", t.ID, f)
		}
		for _, ca := range ControlArtifacts {
			if f == ca {
				return fmt.Errorf("plan: task %s names %s, a control artifact no task may edit (§4.3)", t.ID, ca)
			}
		}
	}
	return nil
}

// validateCoverage is rule 6: every outline heading is accounted for, by task
// or by an explicit out-of-scope reason.
func validateCoverage(p Plan, r Rules) error {
	if !r.CoverageChecked {
		return nil
	}
	ids := map[string]bool{}
	for _, t := range p.Tasks {
		ids[t.ID] = true
	}
	covered := map[string]bool{}
	for i, c := range p.Coverage {
		if covered[c.Heading] {
			return fmt.Errorf("plan: coverage entry %d repeats heading %q", i, c.Heading)
		}
		covered[c.Heading] = true
		if len(c.Tasks) == 0 && strings.TrimSpace(c.OutOfScope) == "" {
			return fmt.Errorf("plan: coverage of %q maps to no task and gives no out_of_scope reason", c.Heading)
		}
		for _, id := range c.Tasks {
			if !ids[id] {
				return fmt.Errorf("plan: coverage of %q names task %q, which does not exist", c.Heading, id)
			}
		}
	}
	var missing []string
	for _, h := range r.Outline.FormattedHeadings() {
		if !covered[h] {
			missing = append(missing, h)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("plan: coverage does not account for %s -- every design section is mapped to tasks or explicitly ruled out of scope", strings.Join(missing, ", "))
	}
	return validateDesignRefs(p, r)
}

// validateDesignRefs holds design_refs to the same key coverage is held to.
//
// The design makes heading text an identity three parties share -- the
// extractor, coverage, and design_refs -- and refuses duplicate outline
// headings for exactly that reason. Validation enforced it on ONE side: a
// planner emitting `design_refs: ["## Networking"]` for a design whose section
// is "## Network protocol" passed, and the unresolvable pointer then reached
// the coder verbatim as the whole of its design context beyond a one-line goal
// (review run 20260813-180828, i20). A wrong ref cost a session and appeared
// nowhere in the report, the journal or the trailers.
func validateDesignRefs(p Plan, r Rules) error {
	known := make(map[string]bool, len(r.Outline.Headings))
	for _, h := range r.Outline.FormattedHeadings() {
		known[h] = true
	}
	for _, t := range p.Tasks {
		for _, ref := range t.DesignRefs {
			if !known[strings.TrimSpace(ref)] {
				return fmt.Errorf("plan: task %s cites design section %q, which is not one of the document's headings -- a reference the coder cannot resolve is the only design context it gets beyond its goal", t.ID, ref)
			}
		}
	}
	return nil
}

// validateFit is rule 7: the plan must fit the deadline at the WORST case the
// configuration permits -- attempts times (session + the sum of per-command
// gate timeouts + overhead). Zero timing inputs mean the caller could not
// resolve the numbers; the check is skipped and the CALLER says so, in the
// same places coverage says so.
func validateFit(tasks int, r Rules) error {
	if r.FitSkipped() {
		return nil
	}
	need := r.RunWorst(tasks)
	if need > r.MaxRunDuration {
		return fmt.Errorf("plan: %d tasks need at least %s at %d attempt(s) per task, a %s worst-case gate, clean_check %q and %s already spent on planning; implement.max_run_duration is %s -- raise it, cut the plan, lower max_task_attempts, or tighten verify.timeout",
			tasks, need.Round(time.Minute), r.MaxTaskAttempts, r.GateWorst, r.CleanCheck, r.PlanOverhead.Round(time.Minute), r.MaxRunDuration)
	}
	return nil
}

// FitSkipped reports whether Validate's fit rule was a no-op for these Rules,
// so the caller can state the skip rather than let silence imply arithmetic
// that never ran.
func (r Rules) FitSkipped() bool {
	return r.SessionTimeout <= 0 || r.MaxRunDuration <= 0
}

// VerifyProfile digests the EFFECTIVE gate -- commands, policy, per-command
// timeout and the gate_generated list -- not the command strings alone:
// continuation uses this digest to assert "finished under the same gate", and
// changing gate_generated alone changes what a gate-written lockfile does
// (review run 20260813-003817). "ungated" is the whole profile of a run with
// no commands.
func VerifyProfile(commands []string, policy string, timeout time.Duration, gateGenerated []string) string {
	if len(commands) == 0 {
		return "ungated"
	}
	gg := append([]string(nil), gateGenerated...)
	sort.Strings(gg)
	var b strings.Builder
	fmt.Fprintf(&b, "policy=%s\ntimeout=%s\n", policy, timeout)
	for _, c := range commands {
		fmt.Fprintf(&b, "cmd=%s\n", c)
	}
	for _, g := range gg {
		fmt.Fprintf(&b, "gate_generated=%s\n", g)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// PlannerLabel is what Provenance.Planner records for an operator-supplied
// plan (§7.4): recording the configured planner agent as the author of a plan
// it never saw would make the audit trail state a falsehood.
func PlannerLabel(agent string, operatorSupplied bool, planSHA string) string {
	if operatorSupplied {
		return fmt.Sprintf("operator-supplied (-plan, sha256 %.12s)", planSHA)
	}
	return agent
}
