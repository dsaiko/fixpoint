package orchestrator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
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

// writeAgentScript writes an executable that ignores its stdin and runs body.
func writeAgentScript(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\ncat >/dev/null\n"+body), 0o700); err != nil {
		t.Fatal(err)
	}
	return p
}

// The whole pipeline end to end on an ungated project: plan, scaffold, two
// task sessions, markers-free happy path -- every processed task leaves a
// commit with the durable trailers, and the summary speaks §5.4's vocabulary.
func TestRunImplementEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	designDir := t.TempDir()
	design := "# Game\n\n## One\n\nbuild the one.\n\n## Two\n\nbuild the two.\n"
	if err := os.WriteFile(filepath.Join(designDir, "DESIGN.md"), []byte(design), 0o600); err != nil {
		t.Fatal(err)
	}

	plan := `{
	  "schema_version": 1,
	  "project": {"name": "game", "summary": "a game"},
	  "coverage": [
	    {"heading": "## One", "tasks": ["T01"]},
	    {"heading": "## Two", "tasks": ["T02"]}
	  ],
	  "tasks": [
	    {"id": "T01", "title": "one", "goal": "the one", "acceptance": ["one exists"], "files": ["one.txt"], "depends_on": [], "design_refs": ["## One"]},
	    {"id": "T02", "title": "two", "goal": "the two", "acceptance": ["two exists"], "files": ["two.txt"], "depends_on": ["T01"], "design_refs": ["## Two"]}
	  ]
	}`
	planner := writeAgentScript(t, "planner", "cat <<'REPLY'\n<plan>\n"+plan+"\n</plan>\nREPLY\n")
	// Each coder session writes one unique source file and reports implemented.
	coder := writeAgentScript(t, "coder",
		"printf 'work\\n' > \"src_$$_$(date +%s).txt\"\nsleep 1\n"+
			"cat <<'REPLY'\n<implement>\n{\"task\": \"T\", \"status\": \"implemented\", \"notes\": \"done\", \"files_touched\": []}\n</implement>\nREPLY\n")

	promptDir := t.TempDir()
	planPrompt := filepath.Join(promptDir, "implement-plan.md")
	taskPrompt := filepath.Join(promptDir, "implement-task.md")
	if err := os.WriteFile(planPrompt, []byte("{{.Design}}\n{{.MaxTasks}}\n{{.OutputContract}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(taskPrompt, []byte("{{.ID}}\n{{.PlanShape}}\n{{.PriorFailure}}\n{{.OutputContract}}"), 0o600); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "project")
	noPing := false
	cfg := &config.Config{
		Target: config.Target{Mode: "directory", Path: designDir, Document: "DESIGN.md"},
		Roles: config.Roles{
			Planner: config.RoleRef{Agent: "planner", Prompt: "implement-plan", PromptPath: planPrompt},
			Coder:   config.RoleRef{Agent: "coder", Prompt: "implement-task", PromptPath: taskPrompt},
		},
		Agents: map[string]config.Agent{
			"planner": {Command: []string{planner}, PromptVia: "stdin", Timeout: config.Duration(time.Minute)},
			"coder":   {Command: []string{coder}, PromptVia: "stdin", Timeout: config.Duration(time.Minute), CanEdit: true},
		},
		Loop:       config.Loop{CommitPolicy: config.CommitPerFix, TrustedTarget: true, MaxIterations: 1},
		Verify:     config.Verify{Policy: config.VerifyOff},
		Create:     config.Create{Out: out},
		PingAgents: &noPing,
		Logs: config.Logs{
			Dir:             filepath.Join(t.TempDir(), "logs", "{timestamp}", "round-{round}"),
			Formats:         []string{"md", "json", "raw"},
			Pattern:         "{role}-{agent}-{prompt}-round-{round}.{ext}",
			SummaryPattern:  "summary.{ext}",
			TimestampFormat: "20060102-150405",
		},
	}
	cfg.Implement.MaxTasks = 40 // discriminator is roles.planner; defaults come from applyDefaults in Load, absent here
	cfg.Implement.MaxFilesPerTask = 12
	cfg.Implement.MaxTaskAttempts = 2
	cfg.Implement.MaxVacuousFrac = 0.34
	cfg.Implement.MaxRunDuration = config.Duration(time.Hour)
	cfg.Implement.MaxTaskBytes = 1 << 20
	cfg.Implement.CleanCheck = config.CleanCheckOff

	logf, logs := captureLog()
	o, err := New(&config.Loaded{Config: cfg, Source: config.Source{Config: "t.yaml"}}, logf)
	if err != nil {
		t.Fatal(err)
	}
	var sum model.RunSummary
	if err := o.runImplement(t.Context(), &sum); err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, logs())
	}
	if sum.Termination != model.TermImplemented {
		t.Fatalf("termination = %q, want implemented\nlog:\n%s", sum.Termination, logs())
	}
	if len(sum.Tasks) != 2 || sum.Tasks[0].Outcome != outcomeImplemented || sum.Tasks[1].Outcome != outcomeImplemented {
		t.Fatalf("tasks = %+v", sum.Tasks)
	}
	if sum.Deliverable != out {
		t.Errorf("deliverable = %q", sum.Deliverable)
	}

	log := gitOutAt(t, out, "log", "--format=%s%n%(trailers:key=Fixpoint-Task,valueonly)")
	for _, want := range []string{"fixpoint: T01 — one", "fixpoint: T02 — two", "fixpoint: initialize implementation"} {
		if !strings.Contains(log, want) {
			t.Errorf("history lacks %q:\n%s", want, log)
		}
	}
	for _, f := range []string{"DESIGN.md", "PLAN.md", "PLAN.json", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("bootstrap file %s: %v", f, err)
		}
	}
	// The delivered DESIGN.md is byte-exact (§4.3).
	got, err := os.ReadFile(filepath.Join(out, "DESIGN.md"))
	if err != nil || string(got) != design {
		t.Errorf("DESIGN.md not byte-exact: %v", err)
	}
	// An ungated run's trailers say skipped, never passed.
	body := gitOutAt(t, out, "log", "-1", "--format=%B", "--grep", "T02")
	if !strings.Contains(body, "Verification: skipped") {
		t.Errorf("T02 body:\n%s", body)
	}
}

func gitOutAt(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, b)
	}
	return string(b)
}
