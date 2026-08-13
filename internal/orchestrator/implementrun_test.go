package orchestrator

import (
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/implement"
	"github.com/dsaiko/fixpoint/internal/model"
)

func twoTaskPlan() implement.Plan {
	return implement.Plan{
		SchemaVersion: 1,
		Tasks: []implement.Task{
			{ID: "T01", Title: "skeleton"},
			{ID: "T02", Title: "game", DependsOn: []string{"T01"}},
			{ID: "T03", Title: "polish", DependsOn: []string{"T02"}},
		},
	}
}

// Skips are derived from the recorded outcomes, never recorded themselves
// (§5.4): a failed dependency skips the whole transitive subtree.
func TestBlockedDependency(t *testing.T) {
	pl := twoTaskPlan()
	outcomes := map[string]string{"T01": "failed"}
	if got := blockedDependency(pl.Tasks[1], outcomes); got != "T01" {
		t.Errorf("blockedDependency(T02) = %q, want T01", got)
	}
	outcomes["T02"] = "skipped"
	if got := blockedDependency(pl.Tasks[2], outcomes); got != "T02" {
		t.Errorf("blockedDependency(T03) = %q, want T02", got)
	}
	if got := blockedDependency(pl.Tasks[0], outcomes); got != "" {
		t.Errorf("blockedDependency(T01) = %q, want none", got)
	}
	// already_satisfied and implemented both count as landed.
	if got := blockedDependency(pl.Tasks[1], map[string]string{"T01": "already_satisfied"}); got != "" {
		t.Errorf("an already_satisfied dependency blocked its dependent: %q", got)
	}
}

func TestVacuousFraction(t *testing.T) {
	tasks := []model.TaskOutcome{
		{Outcome: "implemented"},
		{Outcome: "already_satisfied"},
		{Outcome: "already_satisfied"},
	}
	if got := vacuousFraction(tasks); got < 0.66 || got > 0.67 {
		t.Errorf("vacuousFraction = %g", got)
	}
	if vacuousFraction(nil) != 0 {
		t.Error("empty run reported vacuous work")
	}
}

// already_satisfied must name EARLIER tasks (§5.2 step 5); a later task, an
// unknown id, or an empty list is a contract violation.
func TestCoveredByImplemented(t *testing.T) {
	pl := twoTaskPlan()
	if !coveredByImplemented([]string{"T01"}, pl, 1) {
		t.Error("a legitimate earlier task was rejected")
	}
	for name, ids := range map[string][]string{
		"empty":   nil,
		"later":   {"T03"},
		"itself":  {"T02"},
		"unknown": {"T99"},
	} {
		if coveredByImplemented(ids, pl, 1) {
			t.Errorf("coveredBy %s (%v) accepted", name, ids)
		}
	}
}

func TestPlanShape(t *testing.T) {
	got := planShape(twoTaskPlan(), 1)
	if !strings.Contains(got, "T02: game (THIS TASK)") || !strings.Contains(got, "T01: skeleton (done or recorded)") {
		t.Errorf("planShape:\n%s", got)
	}
}

func TestClampLine(t *testing.T) {
	if got := clampLine("  short  "); got != "short" {
		t.Errorf("clampLine = %q", got)
	}
	if got := clampLine(strings.Repeat("x", 300)); len(got) != 200 {
		t.Errorf("len = %d", len(got))
	}
}

// The commit body carries the marker schema's durable fields, and an ungated
// run says "skipped", never "passed" (§5.2 step 8).
func TestTaskCommitBody(t *testing.T) {
	task := implement.Task{ID: "T01", Title: "skeleton", Goal: "a page", Acceptance: []string{"loads"}}
	p := &implementPrep{designSHA: "9f2c"}
	gated := taskCommitBody(task, "run1", 2, p, "passed", []string{"go.sum"})
	for _, want := range []string{"Fixpoint-Task: T01", "Fixpoint-Outcome: implemented", "Fixpoint-Attempt: 2",
		"Design-SHA256: 9f2c", "Verification: passed", "Fixpoint-Gate-Wrote: go.sum"} {
		if !strings.Contains(gated, want) {
			t.Errorf("body lacks %q:\n%s", want, gated)
		}
	}
	ungated := taskCommitBody(task, "run1", 1, p, "ungated", nil)
	if !strings.Contains(ungated, "Verification: skipped (no operator gates configured)") || strings.Contains(ungated, "passed") {
		t.Errorf("ungated body must say skipped, never passed:\n%s", ungated)
	}
}
