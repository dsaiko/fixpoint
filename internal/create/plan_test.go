package create

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Anonymous in every prompt, attributable in the artifacts, and deterministic so
// two runs over one pool label the same agent the same way.
func TestLabelsAreDeterministicAndDistinct(t *testing.T) {
	a := Labels([]string{"codex", "claude", "kimi"})
	b := Labels([]string{"kimi", "claude", "codex"})
	if a["claude"] != "A" || a["codex"] != "B" || a["kimi"] != "C" {
		t.Errorf("labels = %v, want sorted A/B/C", a)
	}
	for k, v := range a {
		if b[k] != v {
			t.Errorf("labels differ across orderings: %v vs %v", a, b)
		}
	}
}

// The cap derives from the SMALLEST budget -- the pool is deliberately
// heterogeneous, and a cap derived from anything else overruns exactly the agent
// least able to take it. The floor refuses at startup, before sessions are paid
// for; no declared budgets means no cap, not an invented one.
func TestCapsDeriveFromTheSmallestBudgetAndRefuseUnderTheFloor(t *testing.T) {
	limit, err := Caps([]int{900_000, 400_000, 900_000}, 10_000, 2)
	if err != nil {
		t.Fatalf("Caps() = %v", err)
	}
	if want := (400_000 - 10_000 - 16*1024) / 4; limit != want {
		t.Errorf("cap = %d, want %d (from the 400k budget, not the 900k)", limit, want)
	}
	if _, err := Caps([]int{400_000}, 390_000, 2); err == nil || !strings.Contains(err.Error(), "floor") {
		t.Errorf("an assignment that squeezes proposals under the floor must refuse, got %v", err)
	}
	if limit, err := Caps([]int{0, 0}, 1<<20, 4); err != nil || limit != 0 {
		t.Errorf("no declared budgets should mean no cap, got %d, %v", limit, err)
	}
}

// The clamp states what it dropped and never splits a rune -- a truncated text
// that reads as complete is the failure every clamp in this tool exists to avoid.
func TestClampStatesWhatItDroppedOnARuneBoundary(t *testing.T) {
	s := strings.Repeat("ř", 100) // 2 bytes each
	got := Clamp(s, 51)           // mid-rune
	if !utf8.ValidString(got) {
		t.Errorf("clamp split a rune: %q", got[:60])
	}
	if !strings.Contains(got, "not shown") {
		t.Errorf("the elision must be stated:\n%s", got)
	}
	if short := "fits"; Clamp(short, 100) != short {
		t.Error("a text under the cap must pass through untouched")
	}
	if unbounded := Clamp(s, 0); unbounded != s {
		t.Error("cap 0 means unbounded, not empty")
	}
}
