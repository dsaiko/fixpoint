package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/implement"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/target"
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

// The vacuous guard is against the PLAN (§5.4: "N of M tasks"), not against
// the tasks processed so far -- with a running denominator the first
// legitimately satisfied task reads as 50% and aborts the run (review run
// 20260813-124710, i2).
func TestVacuousFraction(t *testing.T) {
	tasks := []model.TaskOutcome{
		{Outcome: outcomeImplemented},
		{Outcome: outcomeSatisfied},
		{Outcome: outcomeSatisfied},
	}
	n, frac := vacuousFraction(tasks, 40)
	if n != 2 || frac < 0.049 || frac > 0.051 {
		t.Errorf("vacuousFraction over a 40-task plan = %d, %g; want 2, 0.05", n, frac)
	}
	// The early-run case the running denominator got wrong: one satisfied task
	// out of three processed, in a plan of ten, is 10% -- under the 34% default.
	if _, frac := vacuousFraction(tasks[:2], 10); frac > 0.34 {
		t.Errorf("one satisfied task early in a 10-task plan tripped the guard at %g", frac)
	}
	if n, frac := vacuousFraction(nil, 0); n != 0 || frac != 0 {
		t.Errorf("empty run reported vacuous work: %d, %g", n, frac)
	}
}

// already_satisfied must name EARLIER tasks that were actually IMPLEMENTED
// (§5.2 step 5). The position check alone is not enough -- citing an earlier
// FAILED task would release the dependents of work nothing built -- and the
// first version of this test asserted only the position, which is why the
// production hole (review run 20260813-124710, i11) survived it.
func TestCoveredByImplemented(t *testing.T) {
	pl := twoTaskPlan()
	implemented := map[string]string{"T01": outcomeImplemented}

	if !coveredByImplemented([]string{"T01"}, pl, 1, implemented) {
		t.Error("an earlier implemented task was rejected")
	}
	cases := map[string]struct {
		ids      []string
		outcomes map[string]string
	}{
		"empty":          {nil, implemented},
		"later task":     {[]string{"T03"}, map[string]string{"T03": outcomeImplemented}},
		"itself":         {[]string{"T02"}, map[string]string{"T02": outcomeImplemented}},
		"unknown id":     {[]string{"T99"}, map[string]string{"T99": outcomeImplemented}},
		"earlier fail":   {[]string{"T01"}, map[string]string{"T01": outcomeFailed}},
		"earlier block":  {[]string{"T01"}, map[string]string{"T01": outcomeBlocked}},
		"earlier skip":   {[]string{"T01"}, map[string]string{"T01": outcomeSkipped}},
		"earlier marker": {[]string{"T01"}, map[string]string{"T01": outcomeSatisfied}},
		"unprocessed":    {[]string{"T01"}, map[string]string{}},
		"one of two":     {[]string{"T01", "T02"}, implemented},
	}
	for name, tc := range cases {
		if coveredByImplemented(tc.ids, pl, 1, tc.outcomes) {
			t.Errorf("coverage by %s (%v, outcomes %v) was accepted", name, tc.ids, tc.outcomes)
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

// implementFixture builds a runnable implement pipeline over a two-section
// design, with scripted planner and coder agents. Every pipeline test shares
// it so the variants differ only in what the agents do and how the gate is
// configured.
type implementFixture struct {
	cfg *config.Config
	out string
	// planJSON is what the scripted planner answers with; newImplementFixture
	// sets the two-task default and a test may replace it before run().
	planJSON string
	planFile string
	logs     func() string
	// backoff overrides the infrastructure retry schedule; nil means the
	// millisecond default every test gets so the breaker is exercised without
	// spending the shipped ~51 minutes.
	backoff []time.Duration
	// diskFree overrides the between-task free-space probe; nil means the real
	// one. Preflight always uses the real one, so a test can leave the run
	// startable and still starve it before a task.
	diskFree func(string) (int64, bool)
}

const twoSectionDesign = "# Game\n\n## One\n\nbuild the one.\n\n## Two\n\nbuild the two.\n"

const twoTaskPlanJSON = `{
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

// implementReply is a coder script body that writes files and answers with the
// given JSON report.
func implementReply(shell, reportJSON string) string {
	return shell + "\ncat <<'REPLY'\n<implement>\n" + reportJSON + "\n</implement>\nREPLY\n"
}

func newImplementFixture(t *testing.T, coderScript string, verify config.Verify) *implementFixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	designDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(designDir, "DESIGN.md"), []byte(twoSectionDesign), 0o600); err != nil {
		t.Fatal(err)
	}
	// The planner's script reads its reply from a file the fixture owns, so a
	// test can change the plan after construction.
	planFile := filepath.Join(t.TempDir(), "plan.json")
	planner := writeAgentScript(t, "planner", "cat <<'REPLY'\n<plan>\nREPLY\ncat "+planFile+"\ncat <<'REPLY'\n</plan>\nREPLY\n")
	coder := writeAgentScript(t, "coder", coderScript)

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
		Verify:     verify,
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
	// applyDefaults runs inside Load, which these tests bypass.
	cfg.Implement.MaxTasks = 40
	cfg.Implement.MaxFilesPerTask = 12
	cfg.Implement.MaxTaskAttempts = 2
	cfg.Implement.MaxVacuousFrac = 0.34
	cfg.Implement.MaxRunDuration = config.Duration(time.Hour)
	cfg.Implement.MaxTaskBytes = 1 << 20
	cfg.Implement.CleanCheck = config.CleanCheckOff
	return &implementFixture{cfg: cfg, out: out, planJSON: twoTaskPlanJSON, planFile: planFile}
}

func (f *implementFixture) run(t *testing.T) (*model.RunSummary, error) {
	t.Helper()
	if err := os.WriteFile(f.planFile, []byte(f.planJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	logf, logs := captureLog()
	f.logs = logs
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "t.yaml"}}, logf)
	if err != nil {
		t.Fatal(err)
	}
	// The shipped schedule waits ~51 minutes across four tries; a test asserting
	// the breaker must exercise the same code without spending them.
	o.infraBackoff = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond, time.Millisecond}
	if f.backoff != nil {
		o.infraBackoff = f.backoff
	}
	if f.diskFree != nil {
		o.diskFree = f.diskFree
	}
	var sum model.RunSummary
	return &sum, o.runImplement(t.Context(), &sum)
}

// The whole pipeline end to end on an ungated project: plan, scaffold, two
// task sessions, every processed task leaving a commit with the durable
// trailers, and the summary speaking §5.4's vocabulary.
func TestRunImplementEndToEnd(t *testing.T) {
	// Each session writes one unique source file and reports implemented.
	f := newImplementFixture(t,
		implementReply("printf 'work\\n' > \"src_$$_$(date +%s).txt\"\nsleep 1",
			`{"status": "implemented", "notes": "done", "files_touched": []}`),
		config.Verify{Policy: config.VerifyOff})

	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Termination != model.TermImplemented {
		t.Fatalf("termination = %q, want implemented\nlog:\n%s", sum.Termination, f.logs())
	}
	if len(sum.Tasks) != 2 || sum.Tasks[0].Outcome != outcomeImplemented || sum.Tasks[1].Outcome != outcomeImplemented {
		t.Fatalf("tasks = %+v", sum.Tasks)
	}
	if sum.Deliverable != f.out {
		t.Errorf("deliverable = %q", sum.Deliverable)
	}

	log := gitOutAt(t, f.out, "log", "--format=%s")
	for _, want := range []string{"fixpoint: T01 — one", "fixpoint: T02 — two", "fixpoint: initialize implementation"} {
		if !strings.Contains(log, want) {
			t.Errorf("history lacks %q:\n%s", want, log)
		}
	}
	for _, name := range []string{"DESIGN.md", "PLAN.md", "PLAN.json", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(f.out, name)); err != nil {
			t.Errorf("bootstrap file %s: %v", name, err)
		}
	}
	// The delivered DESIGN.md is byte-exact (§4.3).
	got, err := os.ReadFile(filepath.Join(f.out, "DESIGN.md"))
	if err != nil || string(got) != twoSectionDesign {
		t.Errorf("DESIGN.md not byte-exact: %v", err)
	}
	// An ungated run's trailers say skipped, never passed.
	body := gitOutAt(t, f.out, "log", "-1", "--format=%B", "--grep", "T02")
	if !strings.Contains(body, "Verification: skipped") {
		t.Errorf("T02 body:\n%s", body)
	}
	// The repository lock was taken and released: a later run can claim it.
	release, err := target.New(config.Target{Mode: "directory", Path: f.out}).LockRepo(t.Context())
	if err != nil {
		t.Errorf("the write-target stayed locked after the run: %v", err)
	} else {
		release()
	}
}

// The GATED path -- the one the shipped implement-go and implement-node
// configs use, and the one the first end-to-end test bypassed (review run
// 20260813-124710, i43/i58). A passing gate commits with a passed trailer.
func TestRunImplementWithAPassingGate(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'ok\\n' > \"src_$$_$(date +%s).txt\"\nsleep 1",
			`{"status": "implemented", "notes": "done", "files_touched": []}`),
		config.Verify{
			Policy:   config.VerifyMustPass,
			Timeout:  config.Duration(time.Minute),
			Commands: []config.VerifyCommand{{Name: "always", Run: []string{"true"}}},
		})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Termination != model.TermImplemented || len(sum.Tasks) != 2 {
		t.Fatalf("termination %q, tasks %+v\nlog:\n%s", sum.Termination, sum.Tasks, f.logs())
	}
	for _, task := range sum.Tasks {
		if task.Gate != "passed" {
			t.Errorf("task %s gate = %q, want passed", task.ID, task.Gate)
		}
	}
	if body := gitOutAt(t, f.out, "log", "-1", "--format=%B"); !strings.Contains(body, "Verification: passed") {
		t.Errorf("a gated commit must record the gate:\n%s", body)
	}
}

// The mandatory credential patterns are the one safety property a config must
// not be able to forget -- and they governed only what agents READ. The
// implement pipeline writes a repository, so a coder that scaffolds an .env
// (ordinary behavior for a node or web project, and steerable from an
// untrusted design) had it committed into the delivered history, where the same
// patterns then hid it from every later review (review run 20260813-180828,
// i3). The attempt fails and names the path instead.
func TestRunImplementRefusesToCommitCredentialShapedFiles(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'ok\\n' > src.txt\nprintf 'TOKEN=sk-live\\n' > .env\nsleep 1",
			`{"status": "implemented", "notes": "scaffolded", "files_touched": []}`),
		config.Verify{Policy: config.VerifyOff})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if !strings.Contains(f.logs(), "credential-shaped file(s)") || !strings.Contains(f.logs(), ".env") {
		t.Errorf("the refusal must name the path it refused:\n%s", f.logs())
	}
	for _, task := range sum.Tasks {
		if task.Outcome == outcomeImplemented {
			t.Errorf("task %s was implemented despite the .env in its census", task.ID)
		}
	}
	// The decisive assertion: nothing in the delivered history carries it.
	if files := gitOutAt(t, f.out, "log", "--all", "--name-only", "--format="); strings.Contains(files, ".env") {
		t.Errorf("a credential-shaped path reached the project's permanent history:\n%s", files)
	}
}

// The gate executes code the coder wrote, so a gate that changes the
// REPOSITORY -- not the worktree -- must stop the run. This is the sharpest
// shape of it (review run 20260813-180828, i48): one line appended to
// .git/info/exclude makes a source file invisible to `git status`, so the
// census that decides what gets committed cannot see it. The gate then passes,
// the commit omits the file, and the pipeline's one claim -- these bytes passed
// the gate -- is false about a tree nobody can reconstruct.
//
// Before the fix the ladder ran only BEFORE the gate and post-gate
// reconciliation looked at worktree files alone, so this was silent, and the
// next task adopted the altered metadata as its own baseline.
func TestRunImplementStopsWhenTheGateMutatesTheRepository(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'ok\\n' > src.txt\nprintf 'hidden\\n' > secret.txt\nsleep 1",
			`{"status": "implemented", "notes": "done", "files_touched": []}`),
		config.Verify{
			Policy:  config.VerifyMustPass,
			Timeout: config.Duration(time.Minute),
			Commands: []config.VerifyCommand{{
				Name: "hide", Run: []string{"sh", "-c", "printf 'secret.txt\\n' >> .git/info/exclude"},
			}},
		})
	sum, err := f.run(t)
	if err == nil {
		t.Fatalf("a gate that edited .git/info/exclude was allowed to commit; tasks %+v\nlog:\n%s", sum.Tasks, f.logs())
	}
	var stop runStopError
	if !errors.As(err, &stop) {
		t.Fatalf("runImplement() = %v, want a repository-invariant run stop", err)
	}
	if !strings.Contains(err.Error(), "the gate changed the repository itself") {
		t.Errorf("the stop must name the gate as the mutator, got: %v", err)
	}
}

// A gate that always fails burns both attempts and the task ends as failed --
// with an outcome MARKER commit, because every processed task leaves exactly
// one commit (§5.4). Its dependent is then skipped, and skips are derived, not
// recorded (review run 20260813-124710, i44/i61).
func TestRunImplementGateFailureMarksAndSkips(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'bad\\n' > \"src_$$_$(date +%s).txt\"\nsleep 1",
			`{"status": "implemented", "notes": "tried", "files_touched": []}`),
		config.Verify{
			Policy:   config.VerifyMustPass,
			Timeout:  config.Duration(time.Minute),
			Commands: []config.VerifyCommand{{Name: "never", Run: []string{"false"}}},
		})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Termination != model.TermIncomplete {
		t.Errorf("termination = %q, want incomplete", sum.Termination)
	}
	if len(sum.Tasks) != 2 {
		t.Fatalf("tasks = %+v", sum.Tasks)
	}
	if sum.Tasks[0].Outcome != outcomeFailed || sum.Tasks[0].Attempts != 2 {
		t.Errorf("T01 = %+v, want failed after 2 attempts", sum.Tasks[0])
	}
	if sum.Tasks[0].SHA == "" {
		t.Error("a failed task left no marker commit; -continue could not replay it")
	}
	if sum.Tasks[1].Outcome != outcomeSkipped {
		t.Errorf("T02 = %+v, want skipped", sum.Tasks[1])
	}
	if sum.Tasks[1].SHA != "" {
		t.Error("a skipped task wrote a commit; skips are derived, never recorded")
	}
	// The marker is empty: the failed session's bytes never entered history.
	if names := gitOutAt(t, f.out, "show", "--name-only", "--format=", sum.Tasks[0].SHA); strings.TrimSpace(names) != "" {
		t.Errorf("the failure marker carries a tree change: %q", names)
	}
	body := gitOutAt(t, f.out, "log", "-1", "--format=%B", sum.Tasks[0].SHA)
	for _, want := range []string{"Fixpoint-Marker: 1", "Fixpoint-Outcome: failed", "Fixpoint-Task: T01"} {
		if !strings.Contains(body, want) {
			t.Errorf("marker body lacks %q:\n%s", want, body)
		}
	}
	// The tree is clean: every non-committing attempt exited through the
	// discard (§5.3).
	if st := gitOutAt(t, f.out, "status", "--porcelain"); strings.TrimSpace(st) != "" {
		t.Errorf("the run left the tree dirty:\n%s", st)
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

// reconcileReport is the §5.2 step-5 contract guard set: it decides, from the
// coder's claim and the tree's state alone, whether an attempt goes on to the
// gate, ends in an outcome, or fails as a contract violation. It shipped
// untested (review run 20260813-124710, i42).
func TestReconcileReport(t *testing.T) {
	pl := twoTaskPlan()
	const idx = 1 // T02, so T01 is an earlier task
	implemented := map[string]string{"T01": outcomeImplemented}
	task := pl.Tasks[idx]

	cases := []struct {
		name     string
		report   taskReport
		clean    bool
		outcomes map[string]string
		wantKind attemptKind
		wantWhy  string // substring; "" means no failure
		wantDone bool
	}{
		{"implemented and dirty proceeds to the gate",
			taskReport{Status: outcomeImplemented}, false, implemented, attemptFailed, "", false},
		{"implemented but clean is a contract violation",
			taskReport{Status: outcomeImplemented}, true, implemented, attemptFailed, "left the working tree unchanged", true},
		{"satisfied with corroboration",
			taskReport{Status: outcomeSatisfied, CoveredBy: []string{"T01"}}, true, implemented, attemptSatisfied, "", true},
		{"satisfied citing a failed task",
			taskReport{Status: outcomeSatisfied, CoveredBy: []string{"T01"}}, true, map[string]string{"T01": outcomeFailed}, attemptFailed, "actually IMPLEMENTED", true},
		{"satisfied without corroboration",
			taskReport{Status: outcomeSatisfied}, true, implemented, attemptFailed, "actually IMPLEMENTED", true},
		{"satisfied but the tree is dirty",
			taskReport{Status: outcomeSatisfied, CoveredBy: []string{"T01"}}, false, implemented, attemptFailed, "disclaimed the work but left changes", true},
		{"blocked with citations",
			taskReport{Status: outcomeBlocked, BlockedOn: []string{"## Rendering"}}, true, implemented, attemptBlocked, "", true},
		{"blocked without citations",
			taskReport{Status: outcomeBlocked}, true, implemented, attemptFailed, "cite the design sections", true},
		{"blocked but the tree is dirty",
			taskReport{Status: outcomeBlocked, BlockedOn: []string{"## Rendering"}}, false, implemented, attemptFailed, "disclaimed the work but left changes", true},
		{"unknown status",
			taskReport{Status: "mostly done"}, false, implemented, attemptFailed, "unknown status", true},
		{"a report about another task",
			taskReport{Task: "T01", Status: outcomeImplemented}, false, implemented, attemptFailed, "answered for task", true},
		{"a report naming its own task is fine",
			taskReport{Task: "T02", Status: outcomeImplemented}, false, implemented, attemptFailed, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verdict, why, done := reconcileReport(task, tc.report, tc.clean, pl, idx, tc.outcomes)
			if done != tc.wantDone {
				t.Fatalf("done = %v, want %v (why %q)", done, tc.wantDone, why)
			}
			if tc.wantWhy == "" {
				if why != "" {
					t.Fatalf("unexpected failure: %q", why)
				}
				if done && verdict.kind != tc.wantKind {
					t.Fatalf("kind = %v, want %v", verdict.kind, tc.wantKind)
				}
				return
			}
			if !strings.Contains(why, tc.wantWhy) {
				t.Fatalf("why = %q, want it to mention %q", why, tc.wantWhy)
			}
		})
	}
}

// The implement run pings the planner and the coder and NOTHING ELSE: the
// reviewer pool inherited from defaults never runs, and pinging it would bill
// four reviewers and die on a quota blackout before the first useful session
// (§7.3). Asserted directly, because the pipeline fixtures disable pings and
// would not notice (review run 20260813-124710, i45).
func TestImplementActiveAgentsAreThePlannerAndTheCoder(t *testing.T) {
	cfg := &config.Config{
		Target: config.Target{Mode: "directory", Path: t.TempDir(), Document: "DESIGN.md"},
		Roles: config.Roles{
			Planner: config.RoleRef{Agent: "planner", Prompt: "p"},
			Coder:   config.RoleRef{Agent: "coder", Prompt: "c"},
			// Inherited from defaults.yaml and inert here.
			Review: config.Review{Strategy: "all", Agents: []string{"claude", "codex", "deepseek-ollama", "minimax-ollama"}},
		},
	}
	o := &Orchestrator{cfg: cfg}
	got := o.activeAgentNames()
	want := []string{"coder", "planner"} // sorted
	if !reflect.DeepEqual(got, want) {
		t.Errorf("activeAgentNames() = %v, want %v -- the inherited pool must never be pinged", got, want)
	}
}

// A blocked report is believed only when a SECOND, independent session agrees
// (§5.4): the blast radius is the whole dependent subtree, so one model's
// opinion is not enough. Untested when it shipped (review run 20260813-124710,
// i44/i56).
func TestRunImplementBlockedNeedsASecondSession(t *testing.T) {
	// Both sessions report blocked with citations, so the outcome stands.
	f := newImplementFixture(t,
		implementReply("sleep 1",
			`{"status": "blocked", "blocked_on": ["## One", "## Two"], "notes": "the design contradicts itself"}`),
		config.Verify{Policy: config.VerifyOff})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Tasks[0].Outcome != outcomeBlocked {
		t.Fatalf("T01 = %+v, want blocked\nlog:\n%s", sum.Tasks[0], f.logs())
	}
	if sum.Tasks[0].Attempts != 2 {
		t.Errorf("blocked was recorded after %d attempt(s); a second session must confirm it", sum.Tasks[0].Attempts)
	}
	if !strings.Contains(sum.Tasks[0].Reason, "## One") {
		t.Errorf("the blocked reason drops its citations: %q", sum.Tasks[0].Reason)
	}
	body := gitOutAt(t, f.out, "log", "-1", "--format=%B", sum.Tasks[0].SHA)
	if !strings.Contains(body, "Fixpoint-Outcome: blocked") {
		t.Errorf("marker body:\n%s", body)
	}
	// Its dependent is skipped, and the run is incomplete.
	if sum.Tasks[1].Outcome != outcomeSkipped || sum.Termination != model.TermIncomplete {
		t.Errorf("T02 = %+v, termination %q", sum.Tasks[1], sum.Termination)
	}
}

// A blocked report the second session does NOT confirm -- it implements the
// task instead -- must not produce a blocked marker: the second opinion is
// what the confirmation is for.
func TestRunImplementBlockedOverturnedBySecondSession(t *testing.T) {
	// First call reports blocked, second writes a file and implements. The
	// counter file survives between sessions because it lives in the agent
	// script's own directory, not the write-target.
	counter := filepath.Join(t.TempDir(), "n")
	script := "n=$(cat " + counter + " 2>/dev/null || echo 0)\n" +
		"echo $((n+1)) > " + counter + "\n" +
		"if [ \"$n\" = 0 ]; then\n" +
		"  cat <<'R1'\n<implement>\n{\"status\": \"blocked\", \"blocked_on\": [\"## One\"], \"notes\": \"unsure\"}\n</implement>\nR1\n" +
		"else\n" +
		"  printf 'work\\n' > \"src_$$_$(date +%s).txt\"; sleep 1\n" +
		"  cat <<'R2'\n<implement>\n{\"status\": \"implemented\", \"notes\": \"it was fine\"}\n</implement>\nR2\n" +
		"fi\n"
	f := newImplementFixture(t, script, config.Verify{Policy: config.VerifyOff})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Tasks[0].Outcome != outcomeImplemented {
		t.Errorf("T01 = %+v, want implemented -- the second session overturned the block\nlog:\n%s", sum.Tasks[0], f.logs())
	}
}

// Infrastructure failures are not task outcomes (§5.4): they burn no attempt,
// write no marker, and two in a row stop the run so a later one can re-enter
// at the same task. Untested when it shipped (review run 20260813-124710,
// i57).
func TestRunImplementInfrastructureBreaker(t *testing.T) {
	// A coder that always dies without output is indistinguishable from a
	// provider outage, which is the point: it is a fact about the morning.
	f := newImplementFixture(t, "exit 1\n", config.Verify{Policy: config.VerifyOff})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Termination != model.TermIncomplete {
		t.Errorf("termination = %q, want incomplete", sum.Termination)
	}
	// Every task is REPORTED, and none is judged: an outage says nothing about
	// the work, so the report says "unreached" rather than leaving the tail of
	// the plan absent from the artifacts entirely (i22).
	for _, task := range sum.Tasks {
		if task.Outcome != outcomeUnreached {
			t.Errorf("an infrastructure outage judged task %s as %q", task.ID, task.Outcome)
		}
	}
	// Only the bootstrap commit exists: no marker was written for the task in
	// flight, which is what makes the stop re-enterable at all.
	if n := strings.Count(gitOutAt(t, f.out, "log", "--format=%s"), "\n"); n != 1 {
		t.Errorf("history has %d commit(s), want only the bootstrap:\n%s", n, gitOutAt(t, f.out, "log", "--format=%s"))
	}
	if !strings.Contains(f.logs(), "infrastructure failure") {
		t.Error("the run did not name the infrastructure failure")
	}
	// The breaker is the terminal case AFTER the retry budget, not instead of
	// it: the shipped code retried with no wait at all, so the second call
	// landed in the same rate-limit window and a 32h run died in seconds
	// (review run 20260813-180828, i42).
	if !strings.Contains(f.logs(), "before trying again") {
		t.Errorf("the run struck out with no backoff between tries:\n%s", f.logs())
	}
	if n := strings.Count(f.logs(), "infrastructure failure ("); n < len(infraBackoff) {
		t.Errorf("only %d infrastructure tries before the breaker, want the whole %d-step budget", n, len(infraBackoff))
	}
}

// 402 is the one provider status where waiting cannot help: the account is out
// of credit, so every retry buys the same answer more slowly. It skips the
// backoff entirely and stops at once.
func TestRunImplementDoesNotWaitOutAPaymentFailure(t *testing.T) {
	f := newImplementFixture(t, "exit 1\n", config.Verify{Policy: config.VerifyOff})
	// Long enough that a run which waited would visibly hang the test.
	f.backoff = []time.Duration{time.Hour, time.Hour, time.Hour, time.Hour}
	// A coder whose CLI reports its PROVIDER refused: a usage envelope carrying
	// the status, which is exactly how the shipped claude agent reports a 429.
	coder := f.cfg.Agents["coder"]
	coder.Command = []string{writeAgentScript(t, "coder402",
		"printf '{\"api_error_status\": 402, \"result\": \"\"}\\n'\nexit 1\n")}
	coder.Usage = config.AgentUsage{Format: "json", Text: "result", ErrorStatus: "api_error_status"}
	f.cfg.Agents["coder"] = coder

	done := make(chan struct{})
	go func() { defer close(done); _, _ = f.run(t) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("a 402 was retried with backoff instead of stopping the run")
	}
	if !strings.Contains(f.logs(), "402") {
		t.Errorf("the stop did not name the payment failure:\n%s", f.logs())
	}
}

// A session that edits a control artifact has it restored and the task failed
// as a contract violation -- the run continues, because the authoritative
// bytes were never lost (§4.3). Untested when it shipped (i60).
func TestRunImplementRestoresAnEditedControlArtifact(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'tampered\\n' > DESIGN.md\nprintf 'work\\n' > src.txt\nsleep 1",
			`{"status": "implemented", "notes": "done"}`),
		config.Verify{Policy: config.VerifyOff})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Tasks[0].Outcome != outcomeFailed {
		t.Errorf("T01 = %+v, want failed", sum.Tasks[0])
	}
	got, err := os.ReadFile(filepath.Join(f.out, "DESIGN.md"))
	if err != nil || string(got) != twoSectionDesign {
		t.Errorf("DESIGN.md was not restored: %q, %v", got, err)
	}
	if st := gitOutAt(t, f.out, "status", "--porcelain"); strings.TrimSpace(st) != "" {
		t.Errorf("the failed attempt left residue:\n%s", st)
	}
}

// Session-created ignored files are deleted before the gate runs and on
// discard, so an attempt's hidden residue never reaches the next one
// (§5.2 step 6, §5.3). Untested when it shipped (i5).
func TestRunImplementDeletesSessionCreatedIgnoredFiles(t *testing.T) {
	// The plan's first task writes a real source file AND an ignored one; the
	// bootstrap .gitignore covers scratch/ via gitignore_seed.
	f := newImplementFixture(t,
		implementReply("mkdir -p scratch && printf 'cache\\n' > scratch/build.tmp\nprintf 'work\\n' > \"src_$$.txt\"\nsleep 1",
			`{"status": "implemented", "notes": "done"}`),
		config.Verify{Policy: config.VerifyOff})
	f.cfg.Implement.GitignoreSeed = []string{"scratch/"}
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Tasks[0].Outcome != outcomeImplemented {
		t.Fatalf("T01 = %+v\nlog:\n%s", sum.Tasks[0], f.logs())
	}
	if _, err := os.Stat(filepath.Join(f.out, "scratch", "build.tmp")); !os.IsNotExist(err) {
		t.Errorf("a session-created ignored file survived the task: %v", err)
	}
	if !strings.Contains(f.logs(), "ignored path(s) deleted before the gate") {
		t.Error("the deletion was not journalled in words the operator can read")
	}
}

// `git add` runs a clean filter that a .gitattributes in the tree names and a
// git config defines, as a child of fixpoint's OWN git process. The
// repository-invariant ladder covers the repo's own config, so the gap i27
// named is the GLOBAL one: a coder writing $HOME/.gitconfig plus the attributes
// file must get nothing executed, and must never see the credentials withheld
// from every agent.
//
// Measured before the fix (review run 20260813-161029): the filter ran and
// captured ANTHROPIC_API_KEY. HOME is redirected for the whole test, so the
// fake global config is the one git reads and the operator's real one is never
// touched.
func TestRunImplementDoesNotRunGloballyDefinedGitFilters(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ANTHROPIC_API_KEY", "sk-secret-value")
	work := t.TempDir()
	canary := filepath.Join(work, "exfiltrated")

	// A filter as a script file, not an inline command: git's config parser
	// treats `;` and `"` in a value as its own syntax, and the point here is
	// the execution, not the quoting.
	filter := filepath.Join(work, "filter.sh")
	if err := os.WriteFile(filter, []byte("#!/bin/sh\nprintf '%s' \"${ANTHROPIC_API_KEY:-none}\" > "+canary+"\ncat\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	script := "cat > .gitattributes <<'ATTRS'\n*.txt filter=evil\nATTRS\n" +
		"cat > " + filepath.Join(home, ".gitconfig") + " <<'CONF'\n" +
		"[filter \"evil\"]\n\tclean = " + filter + "\nCONF\n" +
		"printf 'work\\n' > src.txt\nsleep 1\n"
	f := newImplementFixture(t, implementReply(script,
		`{"status": "implemented", "notes": "done"}`), config.Verify{Policy: config.VerifyOff})

	if _, err := f.run(t); err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if b, err := os.ReadFile(canary); err == nil {
		t.Fatalf("a globally-defined git filter executed during staging and captured %q", strings.TrimSpace(string(b)))
	}
}

// Every commit and every published artifact of the implement pipeline goes
// through agent.RedactSecrets, exactly as every other commit path in the tool
// does. The plan is agent-authored text committed FOREVER into the delivered
// repository, and the planner reads an untrusted design in a directory that may
// hold an .env beside it (review run 20260813-161029, i22/i29).
func TestRunImplementRedactsSecretsInEveryCommittedArtifact(t *testing.T) {
	const secret = "sk-ant-api03-AAAABBBBCCCCDDDDEEEEFFFF0123456789"
	// The planner leaks the credential into the project summary, a task goal
	// and an acceptance criterion; the coder leaks it into a blocked reason.
	plan := strings.ReplaceAll(twoTaskPlanJSON, `"summary": "a game"`,
		`"summary": "a game, key `+secret+`"`)
	plan = strings.ReplaceAll(plan, `"goal": "the one"`, `"goal": "use `+secret+`"`)
	plan = strings.ReplaceAll(plan, `"acceptance": ["one exists"]`, `"acceptance": ["`+secret+` works"]`)

	f := newImplementFixture(t, implementReply("printf 'work\\n' > src.txt\nsleep 1",
		`{"status": "implemented", "notes": "done"}`), config.Verify{Policy: config.VerifyOff})
	f.planJSON = plan

	if _, err := f.run(t); err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	// Nothing in the delivered repository -- committed files or commit messages
	// -- may carry the credential.
	for _, name := range []string{"PLAN.md", "PLAN.json"} {
		b, err := os.ReadFile(filepath.Join(f.out, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), secret) {
			t.Errorf("%s carries the credential verbatim", name)
		}
	}
	if log := gitOutAt(t, f.out, "log", "--format=%B"); strings.Contains(log, secret) {
		t.Error("a commit message carries the credential verbatim")
	}
}

// clean_check is the enforcement of the design's central claim, and `last` is
// the SHIPPED DEFAULT -- yet the wiring was never exercised (review run
// 20260813-161029, i38). A run whose commits satisfy the gate passes it; a run
// whose gate only passes because of an uncommitted ignored file does not, and
// says so.
func TestRunImplementCleanCheck(t *testing.T) {
	// The gate needs marker.txt. The coder writes it into an IGNORED directory
	// and symlinks... no: simpler and truer to the failure -- it writes the
	// file the gate needs under an ignore rule, so the working tree passes and
	// a clone cannot.
	gate := config.Verify{
		Policy:   config.VerifyMustPass,
		Timeout:  config.Duration(time.Minute),
		Commands: []config.VerifyCommand{{Name: "needs", Run: []string{"test", "-f", "hidden/marker.txt"}}},
	}
	f := newImplementFixture(t, implementReply(
		"mkdir -p hidden && printf 'x\\n' > hidden/marker.txt\nprintf 'work\\n' > src.txt\nsleep 1",
		`{"status": "implemented", "notes": "done"}`), gate)
	f.cfg.Implement.GitignoreSeed = []string{"hidden/"}
	f.cfg.Implement.CleanCheck = config.CleanCheckLast

	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	// The tasks themselves pass -- the gate runs in the working tree, where the
	// ignored file exists. The clean check is what catches it.
	if sum.Termination != model.TermIncomplete {
		t.Errorf("termination = %q, want incomplete: HEAD does not build in a clean clone", sum.Termination)
	}
	if !strings.Contains(f.logs(), "clean clone") {
		t.Errorf("the run did not report the clean-clone failure:\n%s", f.logs())
	}
}

// A run whose commits genuinely satisfy the gate passes the same check, so the
// default is not merely a way to fail.
func TestRunImplementCleanCheckPassesOnASoundProject(t *testing.T) {
	gate := config.Verify{
		Policy:   config.VerifyMustPass,
		Timeout:  config.Duration(time.Minute),
		Commands: []config.VerifyCommand{{Name: "needs", Run: []string{"test", "-f", "src.txt"}}},
	}
	// Each session APPENDS a unique line, so the second task leaves the tree
	// dirty too -- writing identical bytes would trip the "reported
	// implementing but changed nothing" contract check instead.
	f := newImplementFixture(t, implementReply("printf 'work %s\\n' \"$$-$(date +%s)\" >> src.txt\nsleep 1",
		`{"status": "implemented", "notes": "done"}`), gate)
	f.cfg.Implement.CleanCheck = config.CleanCheckLast

	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Termination != model.TermImplemented {
		t.Errorf("termination = %q, want implemented\nlog:\n%s", sum.Termination, f.logs())
	}
	if !strings.Contains(f.logs(), "a fresh clone of HEAD passes the gate") {
		t.Error("a passing clean check went unreported")
	}
}

// gate_generated is the third classification category (§5.2 step 7): a file the
// GATE maintains is staged into the task commit with gate attribution, neither
// a mutation failure nor removable output. Never exercised end to end (review
// run 20260813-161029, i51).
func TestRunImplementCommitsGateGeneratedFiles(t *testing.T) {
	gate := config.Verify{
		Policy:  config.VerifyMustPass,
		Timeout: config.Duration(time.Minute),
		// The "gate" behaves like `go mod tidy`: it writes the lockfile.
		Commands: []config.VerifyCommand{{Name: "lock", Run: []string{"sh", "-c", "printf 'locked\\n' > go.sum"}}},
	}
	f := newImplementFixture(t, implementReply("printf 'work\\n' > src.txt\nsleep 1",
		`{"status": "implemented", "notes": "done"}`), gate)
	f.cfg.Implement.GateGenerated = []string{"go.sum"}

	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Tasks[0].Outcome != outcomeImplemented {
		t.Fatalf("T01 = %+v -- a gate-maintained file must not fail the task\nlog:\n%s", sum.Tasks[0], f.logs())
	}
	files := gitOutAt(t, f.out, "show", "--name-only", "--format=", sum.Tasks[0].SHA)
	if !strings.Contains(files, "go.sum") {
		t.Errorf("the gate-maintained lockfile was not committed:\n%s", files)
	}
	if body := gitOutAt(t, f.out, "log", "-1", "--format=%B", sum.Tasks[0].SHA); !strings.Contains(body, "Fixpoint-Gate-Wrote: go.sum") {
		t.Errorf("the commit does not attribute the file to the gate:\n%s", body)
	}
}

// A gate that rewrites the coder's SOURCES is a distinct failure class: tool
// output must never land under the coder's attribution (§5.2 step 7).
func TestRunImplementRefusesAGateThatRewritesSources(t *testing.T) {
	gate := config.Verify{
		Policy:  config.VerifyMustPass,
		Timeout: config.Duration(time.Minute),
		// A formatter in the gate: it rewrites the file the session just wrote.
		Commands: []config.VerifyCommand{{Name: "format", Run: []string{"sh", "-c", "printf 'reformatted\\n' > src.txt"}}},
	}
	f := newImplementFixture(t, implementReply("printf 'work\\n' > src.txt\nsleep 1",
		`{"status": "implemented", "notes": "done"}`), gate)

	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Tasks[0].Outcome != outcomeFailed {
		t.Errorf("T01 = %+v, want failed with gate-mutated-sources", sum.Tasks[0])
	}
	if !strings.Contains(f.logs(), "gate-mutated-sources") {
		t.Errorf("the failure was not named:\n%s", f.logs())
	}
}

// The §5.2 step-4 ladder: a coder that COMMITS is soft-reset back under
// fixpoint's ownership and the work is kept; a coder that rewrites history
// below the base stops the whole run. Neither branch had a pipeline test
// (review run 20260813-161029, i37/i50).
func TestRunImplementSoftResetsACoderCommit(t *testing.T) {
	f := newImplementFixture(t, implementReply(
		"printf 'work\\n' > src.txt\ngit add src.txt >/dev/null 2>&1\ngit -c user.name=c -c user.email=c@c commit -qm 'coder commit' >/dev/null 2>&1\nsleep 1",
		`{"status": "implemented", "notes": "done"}`), config.Verify{Policy: config.VerifyOff})

	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Tasks[0].Outcome != outcomeImplemented {
		t.Fatalf("T01 = %+v -- the work must be kept\nlog:\n%s", sum.Tasks[0], f.logs())
	}
	// The commit is fixpoint's, not the coder's: one commit per task, with our
	// subject and trailers.
	subject := strings.TrimSpace(gitOutAt(t, f.out, "log", "-1", "--format=%s", sum.Tasks[0].SHA))
	if !strings.HasPrefix(subject, "fixpoint: T01") {
		t.Errorf("the coder's commit survived as %q", subject)
	}
	if !strings.Contains(f.logs(), "committed despite being told not to") {
		t.Error("the contract deviation was not journalled")
	}
}

// The pipeline with the largest bill must not discover an unreachable agent
// after the planner session is paid for and the write-target is created and
// locked. runPipeline dispatches implement before run()'s own preflight, so
// implement needs its own call -- and shipped without one (review run
// 20260813-180828, i5).
func TestRunImplementPingsBeforeItSpends(t *testing.T) {
	f := newImplementFixture(t, implementReply("printf 'x\\n' > a.txt\nsleep 1",
		`{"status": "implemented", "notes": "done"}`), config.Verify{Policy: config.VerifyOff})
	ping := true
	f.cfg.PingAgents = &ping
	// A planner that cannot start at all: the ping must catch it.
	planner := f.cfg.Agents["planner"]
	planner.Command = []string{filepath.Join(t.TempDir(), "not-installed")}
	f.cfg.Agents["planner"] = planner

	if _, err := f.run(t); err == nil {
		t.Fatal("an unreachable planner was not caught by preflight")
	}
	if !strings.Contains(f.logs(), "PREFLIGHT") {
		t.Errorf("the implement pipeline never pinged:\n%s", f.logs())
	}
	// Nothing was paid for and nothing was claimed: the refusal came first.
	if _, err := os.Stat(f.out); err == nil {
		t.Errorf("%s was created despite the preflight failure", f.out)
	}
}

// §1's contract is that every task either has a gated commit or is in the
// report with a reason. An early exit used to satisfy neither for the tail of
// the plan (review run 20260813-180828, i22).
func TestRunImplementReportsEveryPlannedTask(t *testing.T) {
	f := newImplementFixture(t, "exit 1\n", config.Verify{Policy: config.VerifyOff})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if len(sum.Tasks) != 2 {
		t.Fatalf("the report covers %d of the plan's 2 tasks: %+v", len(sum.Tasks), sum.Tasks)
	}
	for _, task := range sum.Tasks {
		if task.Reason == "" {
			t.Errorf("task %s is reported as %q with no reason", task.ID, task.Outcome)
		}
	}
}

// A gate command the operator declared environment-dependent is not a verdict
// on the code. Before this, a registry 503 exiting non-zero from `npm ci`
// burned both attempts, wrote a PERMANENT failed marker and skipped every
// dependent task (review run 20260813-222753).
func TestRunImplementTreatsInfraGateFailureAsInfrastructure(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'ok\\n' > src.txt\nsleep 1",
			`{"status": "implemented", "notes": "done"}`),
		config.Verify{
			Policy:  config.VerifyMustPass,
			Timeout: config.Duration(time.Minute),
			Commands: []config.VerifyCommand{
				{Name: "install", Run: []string{"false"}, Infra: true},
			},
		})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	// No marker, and nothing judged: the history must stay re-enterable.
	if n := strings.Count(gitOutAt(t, f.out, "log", "--format=%s"), "\n"); n != 1 {
		t.Errorf("a registry outage reached the history (%d commits):\n%s", n, gitOutAt(t, f.out, "log", "--format=%s"))
	}
	for _, task := range sum.Tasks {
		if task.Outcome == outcomeFailed {
			t.Errorf("task %s was permanently failed by an environment-dependent check", task.ID)
		}
	}
	if !strings.Contains(f.logs(), "environment-dependent check") {
		t.Errorf("the run did not name the failure as environmental:\n%s", f.logs())
	}
}

// The same command WITHOUT the flag stays a verdict: the classification is the
// operator's declaration, not a guess from the exit code.
func TestRunImplementStillFailsOnAnUndeclaredGateFailure(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'ok\\n' > src.txt\nsleep 1",
			`{"status": "implemented", "notes": "done"}`),
		config.Verify{
			Policy:   config.VerifyMustPass,
			Timeout:  config.Duration(time.Minute),
			Commands: []config.VerifyCommand{{Name: "install", Run: []string{"false"}}},
		})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	var failed bool
	for _, task := range sum.Tasks {
		if task.Outcome == outcomeFailed {
			failed = true
		}
	}
	if !failed {
		t.Errorf("an ordinary gate failure stopped being a task failure: %+v", sum.Tasks)
	}
}

// §4.3 names three homes for run state; all three were promises nothing kept.
// status.json is the one that matters while a thirty-hour run is in flight.
func TestRunImplementWritesTheRunStateArtifacts(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'work\\n' > \"src_$$_$(date +%s).txt\"\nsleep 1",
			`{"status": "implemented", "notes": "done"}`),
		config.Verify{Policy: config.VerifyOff})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	// logs.dir is ".../logs/{timestamp}/round-{round}"; the run root is two up.
	runDir := filepath.Dir(filepath.Dir(f.cfg.Logs.Dir))
	entries, err := os.ReadDir(runDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no run directory under %s: %v", runDir, err)
	}
	root := filepath.Join(runDir, entries[0].Name())

	var status struct {
		Planned   int    `json:"planned"`
		Processed int    `json:"processed"`
		Project   string `json:"project"`
		Tasks     []struct {
			ID      string `json:"id"`
			Outcome string `json:"outcome"`
		} `json:"tasks"`
	}
	b, err := os.ReadFile(filepath.Join(root, "status.json"))
	if err != nil {
		t.Fatalf("status.json: %v", err)
	}
	if err := json.Unmarshal(b, &status); err != nil {
		t.Fatalf("status.json is not valid JSON: %v", err)
	}
	if status.Planned != 2 || len(status.Tasks) != len(sum.Tasks) {
		t.Errorf("status.json planned=%d tasks=%d, want 2 and %d", status.Planned, len(status.Tasks), len(sum.Tasks))
	}
	if status.Project != "game" {
		t.Errorf("status.json project = %q", status.Project)
	}

	// plan.json carries the provenance the step artifact cannot: it is logged
	// before fixpoint injects it.
	var plan struct {
		Provenance *struct {
			RunID        string `json:"run_id"`
			DesignSHA256 string `json:"design_sha256"`
		} `json:"provenance"`
	}
	b, err = os.ReadFile(filepath.Join(root, "plan.json"))
	if err != nil {
		t.Fatalf("plan.json: %v", err)
	}
	if err := json.Unmarshal(b, &plan); err != nil {
		t.Fatalf("plan.json is not valid JSON: %v", err)
	}
	if plan.Provenance == nil || plan.Provenance.DesignSHA256 == "" {
		t.Errorf("the canonical plan artifact carries no provenance: %s", b)
	}
}

// The between-task disk guard (i46) was DEAD in the entire suite: the fixture
// bypasses applyDefaults, so MinFreeDisk stayed 0 and `free < 0` was never true
// (review run 20260814-012440). The guard was written, described in a commit
// message, and never once executed by a test.
func TestRunImplementStopsWhenDiskRunsLow(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'work\\n' > \"src_$$_$(date +%s).txt\"\nsleep 1",
			`{"status": "implemented", "notes": "done"}`),
		config.Verify{Policy: config.VerifyOff})
	// Preflight sees the real disk and lets the run start; the between-task probe
	// then reports the filesystem full. Two different probes on purpose -- a
	// threshold high enough to trip the loop would also trip preflight, and the
	// branch under test is the loop's.
	f.cfg.Implement.MinFreeDisk = 1 << 20
	f.diskFree = func(string) (int64, bool) { return 0, true }

	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Termination != model.TermIncomplete {
		t.Errorf("termination = %q, want incomplete", sum.Termination)
	}
	if !strings.Contains(f.logs(), "under implement.min_free_disk") {
		t.Errorf("the stop did not name the threshold:\n%s", f.logs())
	}
	// Nothing was built, and every planned task is still reported (i22).
	if len(sum.Tasks) != 2 {
		t.Fatalf("the report covers %d of 2 planned tasks: %+v", len(sum.Tasks), sum.Tasks)
	}
	for _, task := range sum.Tasks {
		if task.Outcome != outcomeUnreached {
			t.Errorf("task %s = %q, want unreached", task.ID, task.Outcome)
		}
	}
	if n := strings.Count(gitOutAt(t, f.out, "log", "--format=%s"), "\n"); n != 1 {
		t.Errorf("a disk stop committed something: %d commits", n)
	}
}

// implement.max_infra_tries and the ladder backoffSchedule builds from it had no
// test at all: every fixture overrides o.infraBackoff, so the branch never ran,
// and the breaker test counted tries against the package var rather than the
// configured schedule -- it would have passed with the key ignored entirely
// (review run 20260814-012440).
func TestBackoffScheduleFollowsMaxInfraTries(t *testing.T) {
	sched := func(tries int) []time.Duration {
		o := &Orchestrator{cfg: &config.Config{Implement: config.Implement{MaxInfraTries: tries}}}
		return o.backoffSchedule()
	}
	if got := sched(2); !reflect.DeepEqual(got, infraBackoff[:2]) {
		t.Errorf("max_infra_tries=2 -> %v, want the first two rungs %v", got, infraBackoff[:2])
	}
	if got := sched(4); !reflect.DeepEqual(got, infraBackoff) {
		t.Errorf("max_infra_tries=4 -> %v, want the shipped ladder %v", got, infraBackoff)
	}
	// Unset means the shipped ladder, so a config that predates the key behaves
	// as it did.
	if got := sched(0); !reflect.DeepEqual(got, infraBackoff) {
		t.Errorf("max_infra_tries unset -> %v, want the shipped ladder", got)
	}
	// Beyond the declared rungs the last wait REPEATS: raising the key must never
	// silently shorten the waits.
	got := sched(6)
	if len(got) != 6 {
		t.Fatalf("max_infra_tries=6 -> %d rungs, want 6", len(got))
	}
	last := infraBackoff[len(infraBackoff)-1]
	if got[4] != last || got[5] != last {
		t.Errorf("the extra rungs are %v/%v, want the last wait %v repeated", got[4], got[5], last)
	}
}

// gatePhase checks InfraFailures before Passed, so a gate where an environment
// command AND a real check both fail is classified as infrastructure -- the
// attempt is not burned and the run eventually stops on the breaker rather than
// failing the task. That precedence was unstated and unproven: inverting the two
// blocks broke no test (review run 20260814-012440).
func TestRunImplementInfraGateFailureOutranksARealOne(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'ok\\n' > src.txt\nsleep 1",
			`{"status": "implemented", "notes": "done"}`),
		config.Verify{
			Policy:  config.VerifyMustPass,
			Timeout: config.Duration(time.Minute),
			Commands: []config.VerifyCommand{
				{Name: "install", Run: []string{"false"}, Infra: true},
				{Name: "build", Run: []string{"false"}},
			},
		})
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	// Deliberate: while the environment is broken, fixpoint cannot tell whether
	// the build failure is real, so it declines to record a verdict rather than
	// writing a permanent one it might not mean.
	for _, task := range sum.Tasks {
		if task.Outcome == outcomeFailed {
			t.Errorf("task %s was permanently failed while an environment-dependent check was also failing", task.ID)
		}
	}
	if !strings.Contains(f.logs(), "environment-dependent check") {
		t.Errorf("the run did not report the environmental cause:\n%s", f.logs())
	}
}

// repostate.json is where §8 sends an operator diagnosing the loudest failure in
// the tool. It was written and never read back by any test, and the json tags
// the commit called "a durable contract" locked nothing (review run
// 20260814-012440).
func TestRepoStateArtifactCarriesTheDiagnosis(t *testing.T) {
	f := newImplementFixture(t,
		implementReply("printf 'ok\\n' > src.txt\nsleep 1",
			`{"status": "implemented", "notes": "done"}`),
		config.Verify{
			Policy:  config.VerifyMustPass,
			Timeout: config.Duration(time.Minute),
			Commands: []config.VerifyCommand{{
				Name: "hide", Run: []string{"sh", "-c", "printf 'x\\n' >> .git/info/exclude"},
			}},
		})
	if _, err := f.run(t); err == nil {
		t.Fatal("a gate that edited .git/info/exclude did not stop the run")
	}
	runDir := filepath.Dir(filepath.Dir(f.cfg.Logs.Dir))
	entries, err := os.ReadDir(runDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no run directory: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(runDir, entries[0].Name(), "repostate.json"))
	if err != nil {
		t.Fatalf("repostate.json: %v -- §8 points the operator at this file", err)
	}
	// Decoded against the documented field names, not against the Go struct, so
	// a renamed tag fails here rather than silently changing the artifact.
	var rec struct {
		SchemaVersion int      `json:"schema_version"`
		Task          string   `json:"task"`
		When          string   `json:"when"`
		Diff          []string `json:"diff"`
		Before        struct {
			ConfigDigest      string `json:"config_digest"`
			RefsDigest        string `json:"refs_digest"`
			InfoExcludeDigest string `json:"info_exclude_digest"`
		} `json:"before"`
		After struct {
			InfoExcludeDigest string `json:"info_exclude_digest"`
		} `json:"after"`
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatalf("repostate.json is not valid JSON: %v", err)
	}
	if rec.SchemaVersion != 1 || rec.Task == "" || rec.When != "gate" {
		t.Errorf("repostate.json header: %+v", rec)
	}
	if len(rec.Diff) == 0 || !strings.Contains(strings.Join(rec.Diff, "; "), "info/exclude") {
		t.Errorf("the diff does not name the invariant that moved: %v", rec.Diff)
	}
	if rec.Before.ConfigDigest == "" || rec.Before.RefsDigest == "" {
		t.Errorf("the before image is empty, so there is nothing to compare: %+v", rec.Before)
	}
	if rec.Before.InfoExcludeDigest == rec.After.InfoExcludeDigest {
		t.Errorf("before and after agree on the file the gate changed: %q", rec.After.InfoExcludeDigest)
	}
}

// cleanUpInterrupted is the i6 fix: a canceled context made every git command
// fail, so the discard that should have cleaned the tree never ran and an
// interrupted run left the coder's edits behind. It runs on a fresh, bounded
// context now, and the implement pipeline had no test for it -- only the fix
// pipeline did (review run 20260814-012440).
func TestRunImplementCleansTheTreeWhenInterrupted(t *testing.T) {
	// The coder writes, then sleeps long enough for the cancel to land mid-task.
	f := newImplementFixture(t,
		implementReply("printf 'half-done\\n' > leftover.txt\nsleep 30",
			`{"status": "implemented", "notes": "never reached"}`),
		config.Verify{Policy: config.VerifyOff})

	if err := os.WriteFile(f.planFile, []byte(f.planJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	logf, logs := captureLog()
	f.logs = logs
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "t.yaml"}}, logf)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		// Long enough for SCAFFOLD and the coder's write, short of its sleep.
		time.Sleep(4 * time.Second)
		cancel()
	}()
	var sum model.RunSummary
	if err := o.runImplement(ctx, &sum); err == nil {
		t.Fatalf("an interrupted run reported success\nlog:\n%s", f.logs())
	}

	// §8's promise: every committed task stands, nothing further is committed,
	// and the repository is consistent as-is.
	if out := gitOutAt(t, f.out, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Errorf("the interrupted run left the tree dirty:\n%s\nlog:\n%s", out, f.logs())
	}
	if _, err := os.Stat(filepath.Join(f.out, "leftover.txt")); err == nil {
		t.Error("the in-flight attempt's file survived the cleanup")
	}
}
