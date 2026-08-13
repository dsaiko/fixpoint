package implement

import (
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
