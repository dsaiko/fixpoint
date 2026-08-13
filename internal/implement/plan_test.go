package implement

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func validPlan() Plan {
	return Plan{
		SchemaVersion: 1,
		Project:       Project{Name: "prsi", Summary: "a card game"},
		Coverage: []CoverageEntry{
			{Heading: "## Rendering", Tasks: []string{"T01"}},
			{Heading: "## Networking", OutOfScope: "deferred to a later phase"},
		},
		Tasks: []Task{
			{ID: "T01", Title: "skeleton", Goal: "a page that loads",
				Acceptance: []string{"index.html renders"}, Files: []string{"index.html"}},
			{ID: "T02", Title: "deck", Goal: "cards exist", DependsOn: []string{"T01"},
				Acceptance: []string{"32 cards"}, Files: []string{"src/deck.js"}, DesignRefs: []string{"## Rendering"}},
		},
	}
}

func rules() Rules {
	return Rules{
		MaxTasks:        40,
		MaxFilesPerTask: 12,
		Outline:         Outline{Level: 2, Headings: []string{"Rendering", "Networking"}},
		CoverageChecked: true,
		MaxTaskAttempts: 2,
		SessionTimeout:  30 * time.Minute,
		GateWorst:       30 * time.Minute,
		MaxRunDuration:  32 * time.Hour,
	}
}

// Every §4.2 rule refuses with a message an operator can act on; each case
// mutates one rule's input. The valid case is the control.
func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		plan    func(*Plan)
		rules   func(*Rules)
		wantErr string
	}{
		{"valid", func(*Plan) {}, func(*Rules) {}, ""},
		{"schema version", func(p *Plan) { p.SchemaVersion = 2 }, func(*Rules) {}, "schema_version"},
		{"no tasks", func(p *Plan) { p.Tasks = nil }, func(*Rules) {}, "no tasks"},
		{"too many tasks", func(*Plan) {}, func(r *Rules) { r.MaxTasks = 1 }, "split the design"},
		{"bad id", func(p *Plan) { p.Tasks[0].ID = "T 1" }, func(*Rules) {}, "must match"},
		{"duplicate id", func(p *Plan) { p.Tasks[1].ID = "T01" }, func(*Rules) {}, "share id"},
		{"forward dependency", func(p *Plan) { p.Tasks[0].DependsOn = []string{"T02"} }, func(*Rules) {}, "backwards only"},
		{"self dependency", func(p *Plan) { p.Tasks[0].DependsOn = []string{"T01"} }, func(*Rules) {}, "backwards only"},
		{"unknown dependency", func(p *Plan) { p.Tasks[1].DependsOn = []string{"T99"} }, func(*Rules) {}, "backwards only"},
		{"no files", func(p *Plan) { p.Tasks[0].Files = nil }, func(*Rules) {}, "names no files"},
		{"too many files", func(*Plan) {}, func(r *Rules) { r.MaxFilesPerTask = 0 }, "max_files_per_task"},
		{"absolute path", func(p *Plan) { p.Tasks[0].Files = []string{"/etc/passwd"} }, func(*Rules) {}, "relative slash path"},
		{"dotdot path", func(p *Plan) { p.Tasks[0].Files = []string{"../outside"} }, func(*Rules) {}, "relative slash path"},
		{"git path", func(p *Plan) { p.Tasks[0].Files = []string{".git/config"} }, func(*Rules) {}, "tool-owned"},
		{"control artifact reserved", func(p *Plan) { p.Tasks[0].Files = []string{"PLAN.json"} }, func(*Rules) {}, "control artifact"},
		{"gitignore reserved", func(p *Plan) { p.Tasks[0].Files = []string{".gitignore"} }, func(*Rules) {}, "control artifact"},
		{"no acceptance", func(p *Plan) { p.Tasks[0].Acceptance = nil }, func(*Rules) {}, "acceptance"},
		{"no goal", func(p *Plan) { p.Tasks[0].Goal = " " }, func(*Rules) {}, "title and a goal"},
		{"uncovered heading", func(p *Plan) { p.Coverage = p.Coverage[:1] }, func(*Rules) {}, "## Networking"},
		{"coverage names ghost task", func(p *Plan) { p.Coverage[0].Tasks = []string{"T99"} }, func(*Rules) {}, "does not exist"},
		{"coverage without reason", func(p *Plan) { p.Coverage[1].OutOfScope = "" }, func(*Rules) {}, "out_of_scope"},
		{"coverage repeats heading", func(p *Plan) {
			p.Coverage = append(p.Coverage, CoverageEntry{Heading: "## Rendering", Tasks: []string{"T01"}})
		}, func(*Rules) {}, "repeats heading"},
		{"coverage skipped when unchecked", func(p *Plan) { p.Coverage = nil }, func(r *Rules) { r.CoverageChecked = false }, ""},
		// design_refs is the same join key coverage uses, and was validated on
		// only one side: an unresolvable pointer reached the coder as the whole
		// of its design context (review run 20260813-180828, i20).
		{"design_refs names a ghost section", func(p *Plan) { p.Tasks[1].DesignRefs = []string{"## Persistence"} }, func(*Rules) {}, "not one of the document's headings"},
		{"design_refs misspells a real section", func(p *Plan) { p.Tasks[1].DesignRefs = []string{"## rendering"} }, func(*Rules) {}, "not one of the document's headings"},
		{"design_refs unchecked with coverage", func(p *Plan) { p.Tasks[1].DesignRefs = []string{"## Nope"} }, func(r *Rules) { r.CoverageChecked = false }, ""},
		{"fit refusal quotes the arithmetic", func(*Plan) {}, func(r *Rules) { r.MaxRunDuration = time.Hour }, "worst-case gate"},
		{"fit skipped without timings", func(*Plan) {}, func(r *Rules) {
			r.SessionTimeout = 0
			r.MaxRunDuration = time.Nanosecond
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, r := validPlan(), rules()
			tc.plan(&p)
			tc.rules(&r)
			err := Validate(p, r)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

// The fit rule counts the gate PER COMMAND (review run 20260813-003817): 25
// tasks at 2 attempts with a 30m session and a 3x10m gate is ~50.8h and must
// refuse a 32h deadline, while the same plan at a 10m single-command gate
// (~34h... still over) -- so assert the boundary precisely at a passing shape.
func TestFitArithmetic(t *testing.T) {
	r := rules()
	p := validPlan()
	// 2 tasks x 2 attempts x (30m + 30m + 1m) = 4h04m: fits 32h.
	if err := Validate(p, r); err != nil {
		t.Fatalf("fitting plan refused: %v", err)
	}
	r.MaxRunDuration = 4 * time.Hour // 4h04m > 4h: refuse
	err := Validate(p, r)
	if err == nil || !strings.Contains(err.Error(), "4h4m") {
		t.Fatalf("Validate() = %v, want the 4h4m worst case quoted", err)
	}
}

// clean_check buys GATE RUNS, and the fit rule has to count them or it is
// back to admitting plans that are arithmetically guaranteed to hit the
// deadline mid-run -- the trap rule 7 exists to close (review run
// 20260813-180828, i43). `every` costs one clone gate per task; `last` costs
// one for the run.
func TestFitCountsCleanCheckCloneRuns(t *testing.T) {
	r := rules() // 30m session, 30m gate, 2 attempts -> 2h02m per task
	base := r.RunWorst(10)
	r.CleanCheck = "last"
	if got, want := r.RunWorst(10), base+30*time.Minute; got != want {
		t.Errorf("clean_check last: RunWorst(10) = %v, want %v", got, want)
	}
	r.CleanCheck = "every"
	if got, want := r.RunWorst(10), base+10*30*time.Minute; got != want {
		t.Errorf("clean_check every: RunWorst(10) = %v, want %v", got, want)
	}

	// And the cap the planner is quoted must shrink with it, because it is the
	// same arithmetic: quoting a cap the validator would then refuse is the
	// drift this shares its implementation to prevent (i13).
	r.MaxRunDuration = 32 * time.Hour
	r.CleanCheck = ""
	loose := r.Admitted(40)
	r.CleanCheck = "every"
	tight := r.Admitted(40)
	if tight >= loose {
		t.Errorf("clean_check every admitted %d, no fewer than %d without it", tight, loose)
	}
	// Whatever it admits, that plan must actually pass validation -- the cap and
	// the rule agreeing is the whole point.
	p := validPlan()
	for len(p.Tasks) < tight {
		id := fmt.Sprintf("T%02d", len(p.Tasks)+1)
		p.Tasks = append(p.Tasks, Task{ID: id, Title: id, Goal: "g", Acceptance: []string{"a"}, Files: []string{id + ".go"}})
		p.Coverage[0].Tasks = append(p.Coverage[0].Tasks, id)
	}
	r.MaxTasks = 40
	if err := validateFit(len(p.Tasks), r); err != nil {
		t.Errorf("the admitted count (%d) does not fit the rule that computed it: %v", tight, err)
	}
	if err := validateFit(tight+1, r); err == nil {
		t.Errorf("one task over the admitted count (%d) was accepted", tight)
	}
}

func TestStripProvenance(t *testing.T) {
	p := validPlan()
	if StripProvenance(&p) {
		t.Error("clean plan reported a provenance deviation")
	}
	p.Provenance = &Provenance{Planner: "forged"}
	if !StripProvenance(&p) || p.Provenance != nil {
		t.Error("planner-emitted provenance survived the strip")
	}
}

// The digest must move when any component of the effective gate moves --
// commands, policy, timeout, or gate_generated (review run 20260813-003817) --
// and must be stable under gate_generated ordering.
func TestVerifyProfile(t *testing.T) {
	base := VerifyProfile([]string{"go test ./..."}, "must_pass", 10*time.Minute, []string{"go.sum"})
	same := VerifyProfile([]string{"go test ./..."}, "must_pass", 10*time.Minute, []string{"go.sum"})
	if base != same {
		t.Error("identical gates digest differently")
	}
	for name, other := range map[string]string{
		"commands":       VerifyProfile([]string{"go vet ./..."}, "must_pass", 10*time.Minute, []string{"go.sum"}),
		"policy":         VerifyProfile([]string{"go test ./..."}, "off", 10*time.Minute, []string{"go.sum"}),
		"timeout":        VerifyProfile([]string{"go test ./..."}, "must_pass", 5*time.Minute, []string{"go.sum"}),
		"gate_generated": VerifyProfile([]string{"go test ./..."}, "must_pass", 10*time.Minute, nil),
	} {
		if other == base {
			t.Errorf("changing %s did not change the profile", name)
		}
	}
	if VerifyProfile(nil, "off", 0, nil) != "ungated" {
		t.Error("a gate with no commands must digest to the literal \"ungated\"")
	}
	a := VerifyProfile([]string{"npm test"}, "must_pass", time.Minute, []string{"a", "b"})
	b := VerifyProfile([]string{"npm test"}, "must_pass", time.Minute, []string{"b", "a"})
	if a != b {
		t.Error("gate_generated ordering changed the profile")
	}
}

func TestPlannerLabel(t *testing.T) {
	if got := PlannerLabel("claude", false, ""); got != "claude" {
		t.Errorf("planner label = %q", got)
	}
	got := PlannerLabel("claude", true, "abcdef0123456789")
	if !strings.Contains(got, "operator-supplied") || strings.Contains(got, "claude") {
		t.Errorf("an operator-supplied plan must never be attributed to the configured planner; got %q", got)
	}
}

func TestRenderMarkdown(t *testing.T) {
	p := validPlan()
	p.Provenance = &Provenance{RunID: "r1", Planner: "claude", Coder: "claude-coder",
		DesignPath: "/d/DESIGN.md", DesignSHA256: "9f2c", VerifyProfile: "abc"}
	md := RenderMarkdown(p, RenderHeader{Coverage: `"##" (2 headings)`, GateCommands: []string{"go test ./..."}})
	for _, want := range []string{"# Plan: prsi", "run: r1", "coverage: \"##\"", "go test ./...",
		"### T01 — skeleton", "out of scope: deferred", "depends on: T01"} {
		if !strings.Contains(md, want) {
			t.Errorf("PLAN.md lacks %q", want)
		}
	}
	ungated := RenderMarkdown(p, RenderHeader{Coverage: "unchecked"})
	if !strings.Contains(ungated, "ungated") || !strings.Contains(ungated, "coverage: unchecked") {
		t.Error("an ungated, unchecked run must say both, verbatim")
	}
}

// The deadline clock starts when the RUN starts -- before the ping, the design
// snapshot and the planner session -- so the fit budget has to cover the same
// interval or rule 7's claim that "the number the refusal quotes and the number
// the run can spend are the same number" is false (review run 20260813-222753).
func TestFitChargesThePlanningItIsClockedAgainst(t *testing.T) {
	r := rules()
	bare := r.RunWorst(5)
	r.PlanOverhead = 35 * time.Minute
	if got, want := r.RunWorst(5), bare+35*time.Minute; got != want {
		t.Errorf("RunWorst with planning overhead = %v, want %v", got, want)
	}
	// And it must cost admissions, not just appear in a message: a run whose
	// planner can spend half an hour has half an hour less for tasks.
	r.PlanOverhead = 0
	loose := r.Admitted(40)
	r.PlanOverhead = 4 * time.Hour
	if tight := r.Admitted(40); tight >= loose {
		t.Errorf("four hours of planning admitted %d tasks, no fewer than %d", tight, loose)
	}
	// The refusal quotes it, so an operator can see where the time went.
	r.PlanOverhead = 35 * time.Minute
	r.MaxRunDuration = time.Hour
	err := validateFit(1, r)
	if err == nil || !strings.Contains(err.Error(), "spent on planning") {
		t.Errorf("validateFit() = %v, want the planning term named", err)
	}
}
