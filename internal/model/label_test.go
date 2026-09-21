package model

import "testing"

// One vocabulary for both halves of the answer a review run gives. The empty
// case is the one worth pinning: it is reached from postReview's nil-verdict
// gate, and without the branch the withheld reason silently becomes
// "-post-if-approved and the verdict is " -- an empty cell where the fact that
// no verdict was recorded belongs.
func TestVerdictAndEventLabels(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{VerdictApprove, "APPROVE"},
		{VerdictChangesRequested, "CHANGES REQUESTED"},
		{VerdictInconclusive, "INCONCLUSIVE"},
		{"", "NO VERDICT"},
	} {
		if got := VerdictLabel(tc.in); got != tc.want {
			t.Errorf("VerdictLabel(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// Events share the rendering but not the empty-input answer: "no event" is
	// what an empty ReviewPosted means, and its callers test that field instead of
	// asking for a label.
	for _, tc := range []struct{ in, want string }{
		{"comment", "COMMENT"},
		{"approve", "APPROVE"},
		{"request_changes", "REQUEST CHANGES"},
	} {
		if got := EventLabel(tc.in); got != tc.want {
			t.Errorf("EventLabel(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
