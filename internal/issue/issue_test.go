package issue

import (
	"testing"

	"github.com/dsaiko/fixpoint/internal/model"
)

func obs(agent, lens, category, severity, file string, line int, title string) model.Finding {
	return model.Finding{
		Agent: agent, Lens: lens, Category: category, Severity: severity,
		File: file, Line: line, Title: title,
	}
}

// The duplicate that motivated this package, verbatim from a real five-round run:
// two lenses reported the same racy-ordinal defect at the same file and line under
// DIFFERENT categories and severities. As findings they consumed two of the eight
// slots in that round's cap, so the reviewers agreeing cost a budget slot instead
// of raising confidence.
func TestAbsorbMergesRealWorldDuplicate(t *testing.T) {
	l := NewLedger()
	got := l.Absorb(3, []model.Finding{
		obs("codex", "review-concurrency", "concurrency", "low", "internal/testfixture/testfixture.go", 33,
			"Mock invocation ordinals are allocated with a racy read-modify-write"),
		obs("claude", "review-tests", "tests", "medium", "internal/testfixture/testfixture.go", 33,
			"Mock agent assigns invocation ordinals via a racy read-modify-write"),
	})

	if len(got) != 1 {
		t.Fatalf("got %d issues, want 1: the same defect at the same location must not cost two cap slots", len(got))
	}
	iss := got[0]
	if len(iss.Observations) != 2 {
		t.Errorf("got %d observations, want both preserved as corroboration", len(iss.Observations))
	}
	// Severity is the worst any reviewer assigned: one reviewer seeing it as more
	// serious must not be outvoted, because severity decides scheduling.
	if iss.Severity != "medium" {
		t.Errorf("Severity = %q, want medium (the worst reading)", iss.Severity)
	}
	if agents := iss.Agents(); len(agents) != 2 {
		t.Errorf("Agents() = %v, want both reporting agents", agents)
	}
}

// Category must NOT be part of identity -- the real duplicate above arrived under
// two different categories. This pins that explicitly, because "add the category
// to the key" is a natural-looking change that would silently reintroduce the bug.
func TestFingerprintIgnoresCategory(t *testing.T) {
	a := obs("x", "l", "concurrency", "low", "a.go", 10, "same place")
	b := obs("y", "l", "tests", "low", "a.go", 10, "same place")
	if Fingerprint(a) != Fingerprint(b) {
		t.Error("two categories at one location must share a fingerprint")
	}
}

// Reviewers point at slightly different lines for one defect -- the declaration,
// the use, the enclosing function.
func TestAbsorbMergesNearbyLines(t *testing.T) {
	l := NewLedger()
	got := l.Absorb(1, []model.Finding{
		obs("a", "bugs", "bug", "high", "pkg/x.go", 100, "nil deref on the config pointer"),
		obs("b", "bugs", "bug", "high", "pkg/x.go", 103, "config pointer can be nil here"),
	})
	if len(got) != 1 {
		t.Fatalf("got %d issues, want 1: lines %d apart in one file are one defect", len(got), 3)
	}
}

// ...but not so far apart that unrelated code merges.
func TestAbsorbKeepsDistantLinesApart(t *testing.T) {
	l := NewLedger()
	got := l.Absorb(1, []model.Finding{
		obs("a", "bugs", "bug", "high", "pkg/x.go", 10, "one defect"),
		obs("b", "bugs", "bug", "high", "pkg/x.go", 400, "an entirely different defect"),
	})
	if len(got) != 2 {
		t.Fatalf("got %d issues, want 2: merging distant code would tell the coder to fix one thing when there are two", len(got))
	}
}

// With no line number, identity falls back to the title -- normalized, so word
// order and filler words do not split one issue in two.
func TestAbsorbMatchesNormalizedTitleWhenNoLine(t *testing.T) {
	l := NewLedger()
	got := l.Absorb(1, []model.Finding{
		obs("a", "docs", "maintainability", "low", "README.md", 0, "The review-only claim is wrong"),
		obs("b", "docs", "maintainability", "low", "README.md", 0, "review-only claim is wrong"),
	})
	if len(got) != 1 {
		t.Fatalf("got %d issues, want 1: titles differing only in filler words are one issue", len(got))
	}
}

// Cross-round identity that no lexical rule can recover: this is the severity
// vocabulary issue as actually reported across three rounds -- reworded each time,
// and its line moved as the surrounding code changed. The reviewer declares the
// reference; the fingerprint cannot.
func TestAbsorbHonorsReviewerDeclaredIssueAcrossRounds(t *testing.T) {
	l := NewLedger()
	r1 := l.Absorb(1, []model.Finding{
		obs("codex", "review-maintainability", "maintainability", "medium", "internal/orchestrator/orchestrator.go", 382,
			"Severity vocabulary has two independent declarations"),
	})
	if len(r1) != 1 {
		t.Fatalf("round 1: got %d issues, want 1", len(r1))
	}
	id := r1[0].ID

	// Round 2: reworded, different line, same problem -- declared by the reviewer.
	reReport := obs("claude", "review-maintainability", "maintainability", "low", "internal/orchestrator/orchestrator.go", 788,
		"Severity vocabulary declared twice (severityRank and validSeverities)")
	reReport.IssueID = id
	r2 := l.Absorb(2, []model.Finding{reReport})
	if len(r2) != 1 || r2[0].ID != id {
		t.Fatalf("round 2 should join issue %s, got %+v", id, r2)
	}
	if total := len(l.Issues()); total != 1 {
		t.Errorf("ledger holds %d issues, want 1: a declared re-report must not create a second", total)
	}
	if got := l.Issues()[0].FirstRound; got != 1 {
		t.Errorf("FirstRound = %d, want 1: the issue is as old as its first sighting", got)
	}
}

// Corroboration is a claim about ONE round. Under strategy: rotate a lens is
// deliberately reassigned each round, so an issue that survives a round is seen by
// a different agent next time. If the round copy carried the whole accumulated
// history, the coder prompt would announce "reported independently by 2 agents --
// corroborated" for a problem exactly one agent saw per round, and the journal
// would pair a per-round observation count with a cumulative corroborated count
// (3 observations, 3 issues, 3 corroborated -- impossible within a round).
func TestForRoundScopesObservationsToTheRound(t *testing.T) {
	l := NewLedger()
	l.Absorb(1, []model.Finding{obs("codex", "review-bugs", "bug", "high", "x.go", 10, "the same defect")})
	r2 := l.Absorb(2, []model.Finding{obs("claude", "review-bugs", "bug", "high", "x.go", 10, "the same defect")})

	if len(r2) != 1 {
		t.Fatalf("round 2: got %d issues, want the round-1 issue re-reported", len(r2))
	}
	if got := r2[0].Observations; len(got) != 1 || got[0].Agent != "claude" {
		t.Errorf("round 2 observations = %+v, want only this round's report by claude", got)
	}
	if agents := r2[0].Agents(); len(agents) != 1 {
		t.Errorf("Agents() = %v, want 1: one agent per round is not corroboration", agents)
	}
	// The ledger still keeps the full history -- that is what the summary and the
	// aging heuristics read.
	if got := l.Issues()[0].Observations; len(got) != 2 {
		t.Errorf("ledger observations = %d, want both rounds retained", len(got))
	}
}

// A declared id that does not exist is a model mistake. It must not be trusted as
// an identity, but it must not lose the observation either.
func TestAbsorbIgnoresUnknownDeclaredIssueID(t *testing.T) {
	l := NewLedger()
	o := obs("a", "bugs", "bug", "high", "x.go", 5, "real problem")
	o.IssueID = "i999"
	got := l.Absorb(1, []model.Finding{o})
	if len(got) != 1 {
		t.Fatalf("got %d issues, want the observation kept under a fresh issue", len(got))
	}
	if got[0].ID == "i999" {
		t.Error("a hallucinated id must not be adopted as the issue id")
	}
}

// Deferral counts are now exact, which is what the cap's aging consumes. The
// previous approximation could only guess from (file, category).
func TestRecordTracksDeferralsAndStatus(t *testing.T) {
	l := NewLedger()
	got := l.Absorb(1, []model.Finding{obs("a", "bugs", "bug", "low", "x.go", 1, "nit")})
	id := got[0].ID

	l.Record(id, model.VerdictDeferred, "capped")
	if n := l.Deferrals(id); n != 1 {
		t.Errorf("Deferrals = %d, want 1", n)
	}
	// Re-reported next round: still open, and the deferral history is retained so
	// aging keeps promoting it rather than restarting.
	l.Absorb(2, []model.Finding{obs("b", "bugs", "bug", "low", "x.go", 1, "nit")})
	iss, _ := l.Get(id)
	if iss.StatusOrDefault() != model.StatusOpen {
		t.Errorf("status = %q, want open after a re-report", iss.StatusOrDefault())
	}
	if n := l.Deferrals(id); n != 1 {
		t.Errorf("Deferrals = %d, want the count preserved across rounds", n)
	}
	l.Record(id, model.VerdictDeferred, "capped again")
	if n := l.Deferrals(id); n != 2 {
		t.Errorf("Deferrals = %d, want 2", n)
	}
}

// A re-report means different things depending on how the issue was closed, and
// getting this wrong is costly in both directions.
func TestAbsorbReopenSemantics(t *testing.T) {
	t.Run("rejected stays rejected and is not re-submitted", func(t *testing.T) {
		l := NewLedger()
		got := l.Absorb(1, []model.Finding{obs("a", "bugs", "bug", "low", "x.go", 1, "nit")})
		l.Record(got[0].ID, model.VerdictRejected, "not genuine")
		r2 := l.Absorb(2, []model.Finding{obs("b", "bugs", "bug", "low", "x.go", 1, "nit")})
		// Still surfaced, so the summary shows it came up again -- but carrying the
		// rejection, which keeps it out of the coder's workload. Re-submitting a
		// decided issue would spend a slot every round forever.
		if len(r2) != 1 {
			t.Fatalf("got %d issues, want the re-report surfaced", len(r2))
		}
		if r2[0].Verdict != model.VerdictRejected {
			t.Errorf("Verdict = %q, want it to stay rejected", r2[0].Verdict)
		}
	})

	t.Run("fixed reopens, because a re-report means the fix did not work", func(t *testing.T) {
		l := NewLedger()
		got := l.Absorb(1, []model.Finding{obs("a", "bugs", "bug", "high", "x.go", 1, "real bug")})
		l.Record(got[0].ID, model.VerdictFixed, "fixed it")
		r2 := l.Absorb(2, []model.Finding{obs("b", "bugs", "bug", "high", "x.go", 1, "real bug still here")})
		if len(r2) != 1 {
			t.Fatalf("got %d issues, want 1", len(r2))
		}
		// Treating it as closed would let a failed fix end the run as converged.
		if r2[0].Verdict != "" || r2[0].StatusOrDefault() != model.StatusOpen {
			t.Errorf("issue = %+v, want it reopened for the coder", r2[0])
		}
	})

	t.Run("a new round never inherits the previous round's verdict", func(t *testing.T) {
		l := NewLedger()
		got := l.Absorb(1, []model.Finding{obs("a", "bugs", "bug", "low", "x.go", 1, "nit")})
		l.Record(got[0].ID, model.VerdictDeferred, "capped")
		r2 := l.Absorb(2, []model.Finding{obs("b", "bugs", "bug", "low", "x.go", 1, "nit")})
		if r2[0].Verdict != "" {
			t.Errorf("Verdict = %q, want cleared so this round records its own decision", r2[0].Verdict)
		}
		if l.Deferrals(got[0].ID) != 1 {
			t.Error("the deferral count must survive the reopen; aging depends on it")
		}
	})
}

// Paths are spelled inconsistently by different reviewers.
func TestFingerprintNormalizesPaths(t *testing.T) {
	a := obs("x", "l", "bug", "low", "./pkg/x.go", 7, "t")
	b := obs("y", "l", "bug", "low", "pkg/x.go", 7, "t")
	if Fingerprint(a) != Fingerprint(b) {
		t.Errorf("a leading ./ must not split an issue: %q vs %q", Fingerprint(a), Fingerprint(b))
	}
}

// Observations are tagged in place so the summary and history can keep speaking in
// terms of findings while the coder works from issues.
func TestAbsorbTagsObservationsWithIssueID(t *testing.T) {
	l := NewLedger()
	in := []model.Finding{
		obs("a", "bugs", "bug", "high", "x.go", 1, "one"),
		obs("b", "tests", "tests", "low", "x.go", 1, "one again"),
	}
	l.Absorb(1, in)
	if in[0].IssueID == "" || in[0].IssueID != in[1].IssueID {
		t.Errorf("observations must be tagged with their shared issue: %q, %q", in[0].IssueID, in[1].IssueID)
	}
}

// Proximity alone must not merge: three unrelated defects on consecutive lines of
// one file are three issues. Merging them would tell the coder to fix one thing
// when there are three -- strictly worse than a surviving duplicate, which only
// costs a cap slot.
func TestAbsorbKeepsUnrelatedNeighboursApart(t *testing.T) {
	l := NewLedger()
	got := l.Absorb(1, []model.Finding{
		obs("a", "tests", "tests", "low", "main.go", 1, "low prio"),
		obs("a", "bugs", "bug", "high", "main.go", 2, "high prio"),
		obs("a", "bugs", "bug", "medium", "main.go", 3, "med prio"),
	})
	if len(got) != 3 {
		t.Fatalf("got %d issues, want 3: adjacent lines with unrelated titles are distinct defects", len(got))
	}
}

// An exact line match is identity on its own -- that is the fingerprint, and it is
// what merged the real-world duplicate whose titles were only loosely similar.
func TestAbsorbMergesExactLineRegardlessOfTitle(t *testing.T) {
	l := NewLedger()
	got := l.Absorb(1, []model.Finding{
		obs("a", "x", "bug", "low", "main.go", 42, "completely different words here"),
		obs("b", "y", "tests", "low", "main.go", 42, "nothing alike whatsoever"),
	})
	if len(got) != 1 {
		t.Fatalf("got %d issues, want 1: the same line is the same place", len(got))
	}
}
