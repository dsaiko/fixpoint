package model

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Unknown severities must sort LAST. Severity is a scheduling input: the
// per-round cap keeps the worst N issues, so if an invented severity ranked
// first, a reviewer could jump the queue ahead of a real critical just by
// making up a word.
func TestSeverityRankPutsUnknownLast(t *testing.T) {
	last := len(Severities)
	for _, s := range []string{"", "urgent", "critical!", "p0", "blocker"} {
		if got := SeverityRank(s); got != last {
			t.Errorf("SeverityRank(%q) = %d, want %d (unknown must sort last)", s, got, last)
		}
	}
	if SeverityRank("critical") != 0 {
		t.Error("critical must rank first")
	}
	if !WorseSeverity("critical", "low") || WorseSeverity("low", "critical") {
		t.Error("WorseSeverity must order critical above low")
	}
	// A real severity must always outrank an invented one.
	for _, s := range Severities {
		if !WorseSeverity(s, "made-up") {
			t.Errorf("%q must outrank an unknown severity", s)
		}
	}
}

// Severity arrives as free text from a model, so case and surrounding whitespace
// cannot decide whether a finding is valid or how it is scheduled.
func TestSeverityNormalizesInput(t *testing.T) {
	for _, s := range []string{"HIGH", " high ", "High", "\thigh\n"} {
		if !ValidSeverity(s) {
			t.Errorf("ValidSeverity(%q) = false, want true", s)
		}
		if got, want := SeverityRank(s), SeverityRank("high"); got != want {
			t.Errorf("SeverityRank(%q) = %d, want %d", s, got, want)
		}
	}
	if ValidSeverity("catastrophic") {
		t.Error("ValidSeverity must reject a severity outside the closed set")
	}
}

// The vocabulary must be declared exactly once in the tree.
//
// This guard exists because the duplication has happened TWICE: first as two
// declarations inside the orchestrator (which fixpoint's own reviewers caught), then
// again when the issue ledger grew a private copy of the same four strings. Both
// times the copies agreed at the moment they were written, so nothing failed --
// which is precisely why review found it and tests did not. A second declaration is
// a latent scheduling bug: two packages disagreeing about whether "critical"
// outranks "high" would silently reorder which findings reach the coder.
func TestSeverityVocabularyDeclaredOnce(t *testing.T) {
	root := filepath.Join("..", "..")
	self := filepath.Join(root, "internal", "model", "severity.go")

	var offenders []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip VCS and build output; nothing under them is source we own.
			if name := d.Name(); name == ".git" || name == ".fixpoint" {
				return fs.SkipDir
			}
			return nil
		}
		// Tests may name severities freely -- that is how they assert behavior.
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Clean(path) == filepath.Clean(self) {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		// The quoted literal specifically: the review contract's prose rubric
		// ("- critical: exploitable now...") is deliberate and separately pinned by
		// TestReviewContractMatchesSeverityVocabulary, and it is not a Go string
		// literal, so it does not match here.
		if strings.Contains(string(data), `"critical"`) {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Errorf("severity literals outside model/severity.go: %v\n"+
			"Use model.Severities / SeverityRank / ValidSeverity instead -- a second copy has silently drifted twice before.", offenders)
	}
}
