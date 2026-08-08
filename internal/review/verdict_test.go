package review

import (
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/model"
)

func issue(id, severity, status string) model.Issue {
	return model.Issue{ID: id, Severity: severity, Status: status, Title: id}
}

func full(panel int) Quorum {
	return Quorum{Panel: panel, Present: panel, Required: Majority(panel)}
}

// The rule, as a table. Order matters and is the rule itself: a blocking finding
// requires changes whether or not the panel was complete, while an APPROVE needs
// everything to line up at once.
func TestDecide(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   Input
		want Outcome
	}{
		{
			"clean and complete approves",
			Input{Issues: []model.Issue{issue("i1", "low", ""), issue("i2", "medium", "")}, Quorum: full(3)},
			Approve,
		},
		{
			"a high blocks",
			Input{Issues: []model.Issue{issue("i1", "high", "")}, Quorum: full(3)},
			ChangesRequested,
		},
		{
			"a critical blocks",
			Input{Issues: []model.Issue{issue("i1", "critical", "")}, Quorum: full(3)},
			ChangesRequested,
		},
		{
			// The panel being incomplete is a reason to doubt SILENCE, never a reason
			// to doubt a finding that was actually made.
			"a high blocks even without quorum",
			Input{Issues: []model.Issue{issue("i1", "high", "")}, Quorum: Quorum{Panel: 3, Present: 1, Required: 2, Missing: []string{"kimi", "deepseek"}}},
			ChangesRequested,
		},
		{
			"clean but no quorum is inconclusive, never an approval",
			Input{Issues: []model.Issue{issue("i1", "low", "")}, Quorum: Quorum{Panel: 3, Present: 1, Required: 2, Missing: []string{"kimi", "deepseek"}}},
			Inconclusive,
		},
		{
			"an empty panel cannot approve",
			Input{Quorum: Quorum{}},
			Inconclusive,
		},
		{
			// The deterministic half of the gate: no model is involved and none can
			// argue with it.
			"failing CI blocks a run with no findings at all",
			Input{Quorum: full(3), CI: CI{Known: true, Failing: []string{"build"}}},
			ChangesRequested,
		},
		{
			"passing CI does not block",
			Input{Quorum: full(3), CI: CI{Known: true}},
			Approve,
		},
		{
			// A judge or a refutation round decided these; they are not outstanding work.
			"rejected and fixed findings do not block",
			Input{Issues: []model.Issue{
				issue("i1", "critical", model.VerdictRejected),
				issue("i2", "high", model.VerdictFixed),
			}, Quorum: full(3)},
			Approve,
		},
		{
			"a lowered floor blocks on medium",
			Input{Issues: []model.Issue{issue("i1", "medium", "")}, Quorum: full(3), BlockAt: "medium"},
			ChangesRequested,
		},
		{
			"a lowered floor still ignores low",
			Input{Issues: []model.Issue{issue("i1", "low", "")}, Quorum: full(3), BlockAt: "medium"},
			Approve,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.in)
			if got.Outcome != tc.want {
				t.Errorf("Outcome = %q, want %q (reasons: %v)", got.Outcome, tc.want, got.Reasons)
			}
			if len(got.Reasons) == 0 {
				t.Error("a verdict with no stated reason cannot be audited")
			}
		})
	}
}

// Strict majority: a tie must not approve. Two of four is exactly the case where
// a naive ">= half" would let half a dead panel speak for the whole.
func TestMajorityIsStrict(t *testing.T) {
	for panel, want := range map[int]int{0: 0, 1: 1, 2: 2, 3: 2, 4: 3, 5: 3} {
		if got := Majority(panel); got != want {
			t.Errorf("Majority(%d) = %d, want %d", panel, got, want)
		}
	}
	half := Quorum{Panel: 4, Present: 2, Required: Majority(4)}
	if half.Met() {
		t.Error("2 of 4 met quorum; a tie must not be enough to approve")
	}
}

// The unit is the AGENT, not the step: under strategy `all` one agent runs every
// lens, so a reviewer that answered three lenses and timed out on the fourth has a
// blind spot exactly the size of that lens and must not count as present.
func TestQuorumCountsAgentsNotSteps(t *testing.T) {
	asg := []model.Assignment{
		{Agent: "claude", Lens: "review-bugs"}, {Agent: "claude", Lens: "review-security"},
		{Agent: "kimi", Lens: "review-bugs"}, {Agent: "kimi", Lens: "review-security"},
		{Agent: "deepseek", Lens: "review-bugs"}, {Agent: "deepseek", Lens: "review-security"},
	}
	q := QuorumFrom(asg, map[string]int{"kimi": 1}) // kimi lost one lens of two
	if q.Panel != 3 {
		t.Errorf("Panel = %d, want 3 agents (not 6 steps)", q.Panel)
	}
	if q.Present != 2 {
		t.Errorf("Present = %d, want 2: an agent that lost any lens is not present", q.Present)
	}
	if len(q.Missing) != 1 || q.Missing[0] != "kimi" {
		t.Errorf("Missing = %v, want [kimi]", q.Missing)
	}
	if !q.Met() {
		t.Error("2 of 3 should meet a quorum of 2")
	}
}

// An agent whose session never produced a step at all must still count against the
// denominator. Sizing the panel from steps instead of assignments would make a
// reviewer that died before writing anything simply disappear, and a panel of one
// survivor would then look complete.
func TestQuorumSizesThePanelFromAssignmentsNotSurvivors(t *testing.T) {
	asg := []model.Assignment{{Agent: "claude"}, {Agent: "kimi"}, {Agent: "deepseek"}}
	q := QuorumFrom(asg, map[string]int{"kimi": 1, "deepseek": 1})
	if q.Panel != 3 || q.Required != 2 || q.Present != 1 {
		t.Errorf("quorum = %+v, want panel 3, required 2, present 1", q)
	}
	if q.Met() {
		t.Error("one surviving reviewer of three met quorum")
	}
}

// Advisory lenses are reported for a human and gate nothing, so an agent running
// only advisory work is not part of the panel a verdict rests on.
func TestQuorumExcludesAdvisoryAssignments(t *testing.T) {
	asg := []model.Assignment{{Agent: "claude"}, {Agent: "reporter", Advisory: true}}
	q := QuorumFrom(asg, nil)
	if q.Panel != 1 {
		t.Errorf("Panel = %d, want 1: an advisory-only agent is not part of the quorum", q.Panel)
	}
}

// The reasons are what a human audits, so the ones that decide the outcome have to
// name the evidence rather than assert a conclusion.
func TestDecisionReasonsNameTheEvidence(t *testing.T) {
	d := Decide(Input{
		Issues: []model.Issue{issue("i7", "high", ""), issue("i9", "low", "")},
		Quorum: Quorum{Panel: 3, Present: 2, Required: 2, Missing: []string{"deepseek"}},
		CI:     CI{Known: true, Failing: []string{"test", "lint"}},
	})
	joined := strings.Join(d.Reasons, " | ")
	for _, want := range []string{"i7", "test, lint", "2 of 3", "deepseek"} {
		if !strings.Contains(joined, want) {
			t.Errorf("reasons %q should mention %q", joined, want)
		}
	}
	if strings.Contains(joined, "i9") {
		t.Errorf("a low finding is not blocking and must not be listed as a reason: %q", joined)
	}
	if len(d.Blocking) != 1 || d.Blocking[0].ID != "i7" {
		t.Errorf("Blocking = %v, want just i7", issueIDs(d.Blocking))
	}
}

// An approval that rests on a review alone, with no CI to corroborate it, has to
// say so -- otherwise the reader assumes the checks were consulted and passed.
func TestApprovalSaysWhenNoCIWasAvailable(t *testing.T) {
	d := Decide(Input{Quorum: full(2), CI: CI{Known: false}})
	if d.Outcome != Approve {
		t.Fatalf("Outcome = %q, want approve", d.Outcome)
	}
	if !strings.Contains(strings.Join(d.Reasons, " "), "no CI status") {
		t.Errorf("reasons %v should record that no CI status was available", d.Reasons)
	}
	if known := Decide(Input{Quorum: full(2), CI: CI{Known: true}}); strings.Contains(strings.Join(known.Reasons, " "), "no CI status") {
		t.Errorf("a run WITH CI must not carry the no-CI caveat: %v", known.Reasons)
	}
}

// Blocking findings render worst-first, because the first one a human reads should
// be the one that matters most.
func TestBlockingFindingsAreOrderedWorstFirst(t *testing.T) {
	d := Decide(Input{Issues: []model.Issue{
		issue("i1", "high", ""), issue("i2", "critical", ""), issue("i3", "high", ""),
	}, Quorum: full(3)})
	got := issueIDs(d.Blocking)
	if len(got) != 3 || got[0] != "i2" {
		t.Errorf("blocking order = %v, want the critical first", got)
	}
}

// The summary's outcome strings and the decision's Outcome values are declared in
// two packages -- model must be readable by tools that do not import the decision
// logic -- so a rename on one side has to fail here rather than silently produce a
// summary whose verdict nothing recognizes.
func TestSummaryOutcomeStringsMatchTheModelConstants(t *testing.T) {
	for outcome, want := range map[Outcome]string{
		Approve:          model.VerdictApprove,
		ChangesRequested: model.VerdictChangesRequested,
		Inconclusive:     model.VerdictInconclusive,
	} {
		if string(outcome) != want {
			t.Errorf("review.%s = %q, model constant = %q", outcome, string(outcome), want)
		}
	}
	d := Decide(Input{Issues: []model.Issue{issue("i1", "high", "")}, Quorum: full(3)})
	sum := d.Summary()
	if sum.Outcome != model.VerdictChangesRequested {
		t.Errorf("Summary().Outcome = %q, want %q", sum.Outcome, model.VerdictChangesRequested)
	}
	if len(sum.Blocking) != 1 || sum.Blocking[0] != "i1" {
		t.Errorf("Summary().Blocking = %v, want [i1]", sum.Blocking)
	}
	if sum.Panel != 3 || sum.Present != 3 || sum.Required != 2 {
		t.Errorf("Summary() quorum = %d/%d req %d, want 3/3 req 2", sum.Present, sum.Panel, sum.Required)
	}
}

// A verdict must only ever make the exit status worse. A review that requested
// changes ends its loop perfectly normally, so without this the process would exit
// 0 and tell CI the branch was fine.
func TestExitCodeAccountsForTheVerdict(t *testing.T) {
	for _, tc := range []struct {
		name        string
		termination string
		outcome     string
		want        int
	}{
		{"approve", model.TermReviewOnly, model.VerdictApprove, 0},
		{"changes requested", model.TermReviewOnly, model.VerdictChangesRequested, model.ExitChangesRequested},
		{"inconclusive", model.TermReviewOnly, model.VerdictInconclusive, model.ExitInconclusive},
		{"an errored run keeps its own code", model.TermError, model.VerdictApprove, 1},
		{"an interrupted run keeps its own code", model.TermInterrupted, model.VerdictApprove, 1},
		{"a fix run has no verdict", model.TermConverged, "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sum := &model.RunSummary{Termination: tc.termination}
			if tc.outcome != "" {
				sum.Verdict = &model.ReviewVerdict{Outcome: tc.outcome}
			}
			if got := model.ExitCodeFor(sum); got != tc.want {
				t.Errorf("ExitCodeFor(%s/%s) = %d, want %d", tc.termination, tc.outcome, got, tc.want)
			}
		})
	}
}
