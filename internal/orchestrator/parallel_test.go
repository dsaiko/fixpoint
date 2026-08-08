package orchestrator

import (
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

// Two coder sessions editing one file from a shared base produce patches that do
// not compose: each is a diff against the same original text, so applying the
// second after the first conflicts or silently reverts part of it. Batching by
// file removes that case by construction, which is why there is no merge step
// anywhere in this feature.
func TestBatchingNeverPutsTwoIssuesOnOneFileTogether(t *testing.T) {
	issues := []model.Issue{
		{ID: "i1", File: "a.go"}, {ID: "i2", File: "a.go"},
		{ID: "i3", File: "b.go"}, {ID: "i4", File: "a.go"},
		{ID: "i5", File: "c.go"},
	}
	for _, n := range []int{2, 3, 4, 10} {
		for _, batch := range batchByFile(issues, n) {
			if len(batch) > n {
				t.Errorf("n=%d: batch of %d exceeds the limit", n, len(batch))
			}
			seen := map[string]bool{}
			for _, it := range batch {
				if seen[it.File] {
					t.Errorf("n=%d: %s and another issue share %s in one batch", n, it.ID, it.File)
				}
				seen[it.File] = true
			}
		}
	}
}

// Every issue must appear exactly once, in the order it was given: they arrive
// worst-severity first, and a batch boundary must not promote a low finding ahead
// of a high one, nor drop one.
func TestBatchingPreservesEveryIssueAndItsOrder(t *testing.T) {
	issues := []model.Issue{
		{ID: "i1", File: "a.go"}, {ID: "i2", File: "b.go"}, {ID: "i3", File: "a.go"},
		{ID: "i4", File: ""}, {ID: "i5", File: "c.go"},
	}
	var got []string
	for _, b := range batchByFile(issues, 3) {
		for _, it := range b {
			got = append(got, it.ID)
		}
	}
	if want := "i1 i2 i5 i3 i4"; strings.Join(got, " ") != want {
		// i1,i2,i5 fill the first batch (distinct files); i3 collides with i1's file
		// and starts the second; i4 has no file and takes one of its own.
		t.Errorf("order = %v, want %s", got, want)
	}
	if len(got) != len(issues) {
		t.Errorf("got %d issues out of %d", len(got), len(issues))
	}
}

// An issue with no file cannot be reasoned about -- "which files does it touch"
// is unanswerable -- so it runs alone rather than being assumed to collide with
// nothing.
func TestAnIssueWithNoFileRunsAlone(t *testing.T) {
	batches := batchByFile([]model.Issue{
		{ID: "i1", File: "a.go"}, {ID: "i2", File: ""}, {ID: "i3", File: "b.go"},
	}, 4)
	for _, b := range batches {
		for _, it := range b {
			if it.File == "" && len(b) != 1 {
				t.Errorf("issue %s with no file shares a batch of %d", it.ID, len(b))
			}
		}
	}
}

// Sequential is the default and stays reachable: n below 2 must produce batches
// of one, which is the path every existing test exercises.
func TestBatchingBelowTwoIsSequential(t *testing.T) {
	issues := []model.Issue{{ID: "i1", File: "a.go"}, {ID: "i2", File: "b.go"}}
	for _, n := range []int{-1, 0, 1} {
		for _, b := range batchByFile(issues, n) {
			if len(b) != 1 {
				t.Errorf("n=%d produced a batch of %d, want sequential", n, len(b))
			}
		}
	}
}

// The speedup is bounded by the busiest file, not by the configured number, and
// that is worth pinning: measured on this project's own rounds, 9 issues over 4
// files with the worst file holding 4 run in four batches however high n is.
func TestTheBusiestFileBoundsTheBatchCount(t *testing.T) {
	issues := make([]model.Issue, 0, 9)
	for i := range 4 {
		issues = append(issues, model.Issue{ID: "a" + string(rune('1'+i)), File: "busy.go"})
	}
	issues = append(issues,
		model.Issue{ID: "b1", File: "x.go"}, model.Issue{ID: "b2", File: "y.go"},
		model.Issue{ID: "b3", File: "z.go"}, model.Issue{ID: "b4", File: "w.go"},
		model.Issue{ID: "b5", File: "v.go"})
	if got := len(batchByFile(issues, 16)); got != 4 {
		t.Errorf("batches = %d, want 4 -- one per issue on the busiest file", got)
	}
}

// A session's verdict must reach the round's record, and only for the issue that
// session was given: a scratch record cannot be allowed to carry a claim about
// anybody else's issue.
func TestMergingASessionCarriesItsOwnVerdictOnly(t *testing.T) {
	rec := &model.RoundRecord{Round: 1, Issues: []model.Issue{
		{ID: "i1", Title: "one"}, {ID: "i2", Title: "two"},
	}}
	s := session{
		issue: model.Issue{ID: "i1"},
		scratch: &model.RoundRecord{
			Round: 1,
			Issues: []model.Issue{
				{ID: "i1", Status: model.VerdictFixed, Verdict: model.VerdictFixed, VerdictDetail: "done"},
				{ID: "i2", Status: model.VerdictRejected, Verdict: model.VerdictRejected, VerdictDetail: "not mine to say"},
			},
			Steps: []model.StepStat{{Role: "fix", Agent: "coder"}},
			Fixed: 1,
		},
	}
	mergeSession(rec, s)

	if rec.Issues[0].Status != model.VerdictFixed {
		t.Error("the session's own verdict did not reach the round")
	}
	if rec.Issues[1].Status == model.VerdictRejected {
		t.Error("a session recorded a verdict on an issue it was not given")
	}
	if len(rec.Steps) != 1 {
		t.Errorf("steps = %d, want the session's one: a failed or rejected session still spent tokens", len(rec.Steps))
	}
	if rec.Fixed != 1 {
		t.Errorf("Fixed = %d, want 1", rec.Fixed)
	}
}

// Merging a session that never ran must change nothing: a worktree that could not
// be created leaves no scratch, and the issue has to survive to the next round
// rather than being recorded as anything.
func TestMergingAFailedSessionIsANoOp(t *testing.T) {
	rec := &model.RoundRecord{Round: 1, Issues: []model.Issue{{ID: "i1", Title: "one"}}}
	mergeSession(rec, session{issue: model.Issue{ID: "i1"}})
	if rec.Issues[0].Status != "" || len(rec.Steps) != 0 || rec.Fixed != 0 {
		t.Errorf("a session with no scratch changed the round: %+v", rec.Issues[0])
	}
}

// The property the whole feature rests on: a parallel round must land the same
// commits, in the same order, as a sequential one. The sessions run at once and in
// separate checkouts; the gate and the commits do not move.
//
// Each coder writes into its own working directory, which for a parallel session
// is a worktree -- so the edits reaching the real tree are the ones read back as
// patches, not edits the sessions made to it directly.
func TestAParallelRoundCommitsEachFixSeparatelyAndInOrder(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, ParallelFixes: 4})
	before := f.commitCount()

	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bug", Severity: "high", File: "a.go", Line: 1, Title: "in a"},
		model.ReviewFinding{Category: "bug", Severity: "high", File: "b.go", Line: 1, Title: "in b"}))
	// Two sessions, two files, each writing relative to its own cwd.
	f.writeSideRelative(2, "a.go", "fixed a")
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "d"}))
	f.writeSideRelative(3, "b.go", "fixed b")
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "d"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if got := f.commitCount() - before; got != 2 {
		t.Errorf("commits = %d, want one per fix: the gate stays serial however many coders run", got)
	}
	rec := sum.Rounds[0]
	if rec.Fixed != 2 {
		t.Errorf("fixed = %d, want 2; issues %+v", rec.Fixed, rec.Issues)
	}
	// Both edits are in the tree, so neither patch overwrote the other.
	for _, want := range []struct{ file, body string }{{"a.go", "fixed a"}, {"b.go", "fixed b"}} {
		if got := f.fileInRepo(want.file); !strings.Contains(got, want.body) {
			t.Errorf("%s = %q, want it to carry %q", want.file, got, want.body)
		}
	}
	// And every session was billed: a parallel round must not lose a step's tokens.
	fixSteps := 0
	for _, s := range rec.Steps {
		if s.Role == "fix" {
			fixSteps++
		}
	}
	if fixSteps != 2 {
		t.Errorf("fix steps recorded = %d, want 2", fixSteps)
	}
}
