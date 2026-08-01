package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"unicode/utf8"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

func loadTemplate(t *testing.T, content string) *template.Template {
	t.Helper()
	p := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	tmpl, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func TestLoadAndRender(t *testing.T) {
	tmpl := loadTemplate(t, "Round {{.Round}} mode {{.Mode}}\n{{.Target}}\n{{.OutputContract}}")
	got, err := Render(tmpl, ReviewData{Round: 2, Mode: "pr", Target: "the diff", OutputContract: ReviewContract})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Round 2 mode pr", "the diff", "<review>"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered prompt missing %q:\n%s", want, got)
		}
	}
}

// FixContract is the JSON output contract the coder is bound to. Like the
// review path's ReviewContract assertion above, render it through a FixData and
// assert its shape survives: the <fix> envelope, both verdict examples, and the
// "exactly once" clause. A malformed contract (broken example JSON, a dropped
// instruction) would otherwise render without error and ship undetected.
func TestFixContractRenders(t *testing.T) {
	tmpl := loadTemplate(t, "Round {{.Round}} mode {{.Mode}}\n{{.Findings}}\n{{.OutputContract}}")
	got, err := Render(tmpl, FixData{Round: 3, Mode: "directory", Findings: "the findings", OutputContract: FixContract})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Round 3 mode directory",
		"the findings",
		"<fix>",
		`"verdict": "fixed"`,
		`"verdict": "rejected"`,
		"exactly once",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered fix prompt missing %q:\n%s", want, got)
		}
	}
}

func TestLoadErrors(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.md")); err == nil {
		t.Error("Load(missing) = nil, want error")
	}
	p := filepath.Join(t.TempDir(), "bad.md")
	if err := os.WriteFile(p, []byte("{{.Unclosed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Error("Load(bad template) = nil, want parse error")
	}
}

func TestRenderUnknownFieldFails(t *testing.T) {
	tmpl := loadTemplate(t, "{{.NoSuchField}}")
	if _, err := Render(tmpl, ReviewData{}); err == nil {
		t.Error("Render() = nil, want error for unknown field")
	}
}

// A placeholder belonging to the other role's contract must fail at render,
// not silently produce an empty value.
func TestRenderCrossRolePlaceholderFails(t *testing.T) {
	tmpl := loadTemplate(t, "{{.Findings}}")
	if _, err := Render(tmpl, ReviewData{}); err == nil {
		t.Error("Render(review data) = nil, want error for fix-only .Findings")
	}
	if _, err := Render(tmpl, FixData{}); err != nil {
		t.Errorf("Render(fix data) = %v, want nil", err)
	}
	tmpl = loadTemplate(t, "{{.Target}}")
	if _, err := Render(tmpl, FixData{}); err == nil {
		t.Error("Render(fix data) = nil, want error for review-only .Target")
	}
}

func TestFormatFindings(t *testing.T) {
	findings := []model.Finding{
		{
			ID: "r1.1", Agent: "claude", Lens: "review-bugs",
			Category: "correctness", Severity: "high",
			File: "a/b.go", Line: 42,
			Title:       "off by one",
			Description: "loop misses the last element",
			Suggestion:  "use <=",
		},
		{
			ID: "r1.2", Agent: "codex", Lens: "review-tests",
			Category: "tests", Severity: "low",
			File:  "c.go", // no line, no description/suggestion
			Title: "missing test",
		},
	}
	got := FormatFindings(findings)
	for _, want := range []string{
		"### [r1.1] (correctness, high) a/b.go:42 — off by one",
		"loop misses the last element",
		"Suggested: use <=",
		"(reported by claude via review-bugs)",
		"### [r1.2] (tests, low) c.go — missing test",
		"(reported by codex via review-tests)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FormatFindings missing %q:\n%s", want, got)
		}
	}
	if strings.HasSuffix(got, "\n") {
		t.Error("FormatFindings should not end with a newline")
	}
	if FormatFindings(nil) != "" {
		t.Error("FormatFindings(nil) should be empty")
	}
}

func TestFormatHistory(t *testing.T) {
	if got := FormatHistory(nil); got != "" {
		t.Errorf("FormatHistory(nil) = %q, want empty", got)
	}
	rounds := []model.RoundRecord{
		{
			Round: 1, Fixed: 1, Rejected: 1,
			Findings: []model.Finding{
				{ID: "r1.1", File: "x.go", Line: 3, Title: "bug", Verdict: "fixed", VerdictDetail: "patched"},
				{ID: "r1.2", File: "y.go", Title: "nit", Verdict: "rejected", VerdictDetail: "by design"},
				{ID: "r1.3", File: "z.go", Title: "hang"}, // no verdict
			},
		},
		{Round: 2},
	}
	got := FormatHistory(rounds)
	for _, want := range []string{
		"## History of previous rounds",
		"Round 1 (1 fixed, 1 rejected):",
		"[r1.1] x.go:3 bug — FIXED: patched",
		"[r1.2] y.go nit — REJECTED: by design",
		"[r1.3] z.go hang — UNRESOLVED:",
		"Round 2 (0 fixed, 0 rejected):",
		"- no findings",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FormatHistory missing %q:\n%s", want, got)
		}
	}
}

// History is prepended to every review prompt and used to grow without bound: one
// measured run went from 4.6 KB to 37.7 KB by round 7, against a fixed reviewer
// timeout that then killed three reviewers mid-work. Rounds past the most recent
// few keep only what stops a re-report -- id, location, verdict -- and lose the
// titles and verdict details that make up the bulk.
func TestFormatHistoryCondensesOlderRounds(t *testing.T) {
	var rounds []model.RoundRecord
	for i := 1; i <= historyRoundsInFull+2; i++ {
		rounds = append(rounds, model.RoundRecord{
			Round: i, Fixed: 1,
			Findings: []model.Finding{{
				ID:      fmt.Sprintf("r%d.1", i),
				File:    fmt.Sprintf("f%d.go", i),
				Line:    7,
				Title:   fmt.Sprintf("title of round %d", i),
				Verdict: "rejected", VerdictDetail: fmt.Sprintf("detail of round %d", i),
			}},
		})
	}
	got := FormatHistory(rounds)

	// The two oldest rounds are condensed: the id and the verdict survive, because
	// "do not re-report this" has to outlive the detail that justified it.
	for _, want := range []string{"[r1.1] f1.go:7 — REJECTED", "[r2.1] f2.go:7 — REJECTED"} {
		if !strings.Contains(got, want) {
			t.Errorf("condensed line %q missing:\n%s", want, got)
		}
	}
	for _, gone := range []string{"title of round 1", "detail of round 1", "title of round 2", "detail of round 2"} {
		if strings.Contains(got, gone) {
			t.Errorf("condensed round still carries %q, so history is still growing:\n%s", gone, got)
		}
	}
	// The most recent rounds keep everything.
	for i := 3; i <= historyRoundsInFull+2; i++ {
		for _, want := range []string{fmt.Sprintf("title of round %d", i), fmt.Sprintf("detail of round %d", i)} {
			if !strings.Contains(got, want) {
				t.Errorf("recent round %d lost %q:\n%s", i, want, got)
			}
		}
	}
	// A run short enough to fit is untouched, so nothing is condensed prematurely.
	if short := FormatHistory(rounds[:historyRoundsInFull]); !strings.Contains(short, "title of round 1") {
		t.Errorf("a run within the window must keep full detail:\n%s", short)
	}
}

// Verdict details are the bulk of what survives condensing: a coder argues a
// rejection at length, and history carries one per finding to every reviewer every
// round. Measured on the run this came from, clipping them took the round-7 history
// from 18.4 KB to 7.0 KB (33.6 KB before older rounds were condensed too).
func TestFormatHistoryClipsLongVerdictDetails(t *testing.T) {
	long := strings.Repeat("because the premise is false and here is the citation trail ", 20)
	got := FormatHistory([]model.RoundRecord{{
		Round: 1, Rejected: 1,
		Findings: []model.Finding{
			{ID: "i1", File: "a.go", Title: "t", Verdict: "rejected", VerdictDetail: long},
			{ID: "i2", File: "b.go", Title: "t2", Verdict: "fixed", VerdictDetail: "short one"},
		},
	}})
	if len(got) > 1200 {
		t.Errorf("history is %d B for two findings, want the long detail clipped:\n%s", len(got), got)
	}
	// The headline survives -- that is what a reviewer needs to decide whether to
	// re-report -- and the cut is visible rather than looking like a finished thought.
	if !strings.Contains(got, "because the premise is false") {
		t.Errorf("clipping removed the start of the reason:\n%s", got)
	}
	if !strings.Contains(got, "[…]") {
		t.Errorf("a clipped detail must be marked as cut:\n%s", got)
	}
	// A detail that fits is untouched, mark and all.
	if !strings.HasSuffix(got, "FIXED: short one") {
		t.Errorf("a short detail must pass through verbatim:\n%s", got)
	}
}

// A verdict detail is free-form prose from an agent, so it can hold any UTF-8 --
// non-ASCII quotes, arrows, other scripts. Clipping it at a byte offset would leave
// half a character at the end, and that broken byte then rides in every history block
// of every later round, where an agent CLI may reject the prompt outright.
func TestFormatHistoryClipsOnCharacterBoundaries(t *testing.T) {
	// No spaces, so the word-boundary fallback cannot hide the cut, and the leading
	// ASCII byte puts the 3-byte arrows out of phase with the byte limit.
	got := FormatHistory([]model.RoundRecord{{
		Round: 1, Rejected: 1,
		Findings: []model.Finding{
			{ID: "i1", File: "a.go", Title: "t", Verdict: "rejected", VerdictDetail: "x" + strings.Repeat("→", 300)},
		},
	}})
	if !utf8.ValidString(got) {
		t.Errorf("clipping split a character: history is not valid UTF-8:\n%q", got)
	}
	if !strings.Contains(got, "[…]") {
		t.Errorf("a clipped detail must be marked as cut:\n%s", got)
	}
}

// The reviewers of a round all read one snapshot, then per-fix sessions commit into
// the tree one after another -- so a later session's finding can already be handled.
// The coder is told what moved rather than being left to re-apply a landed fix.
func TestFormatStale(t *testing.T) {
	if got := FormatStale(nil); got != "" {
		t.Errorf("FormatStale(nil) = %q, want empty: an unmoved tree must add nothing to the prompt", got)
	}
	got := FormatStale([]string{"internal/target/target.go", "cmd/fixpoint/main.go"})
	for _, want := range []string{
		"may already be out of date",
		"- internal/target/target.go",
		"- cmd/fixpoint/main.go",
		"Read the CURRENT contents before editing",
		"reject",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FormatStale missing %q:\n%s", want, got)
		}
	}
}

func TestModeGuidance(t *testing.T) {
	for _, mode := range []config.Mode{config.ModeGitDiff, config.ModePR, config.ModeDirectory} {
		if ModeGuidance(mode) == "" {
			t.Errorf("ModeGuidance(%q) is empty", mode)
		}
	}
	if ModeGuidance("other") != "" {
		t.Error("ModeGuidance(other) should be empty")
	}
}

// The rubric in ReviewContract and model.Severities must describe the same
// vocabulary. They are separately authored on purpose -- the rubric is per-severity
// prose that explains what each level MEANS, which generating from a list would
// only make worse -- but they must not drift: a severity the contract teaches
// reviewers to use but the validator rejects turns every finding at that level into
// a reviewer error, and one the validator accepts but the contract never mentions
// gets rated by each lens's own guesswork, which is the inconsistency the rubric
// exists to end.
func TestReviewContractMatchesSeverityVocabulary(t *testing.T) {
	for _, s := range model.Severities {
		// The rubric line for each severity, e.g. "- critical: ".
		if !strings.Contains(ReviewContract, "\n- "+s+": ") {
			t.Errorf("ReviewContract has no rubric line for severity %q; reviewers would rate it by guesswork", s)
		}
	}
	// And the JSON example's severity placeholder must enumerate exactly the same
	// set, since that line is what a reviewer copies.
	want := `"severity": "` + strings.Join(model.Severities, "|") + `"`
	if !strings.Contains(ReviewContract, want) {
		t.Errorf("ReviewContract's example severity placeholder does not match model.Severities; want it to contain %s", want)
	}
	// No rubric line for a severity the validator would reject.
	for _, line := range strings.Split(ReviewContract, "\n") {
		name, _, isRubric := strings.Cut(strings.TrimPrefix(line, "- "), ": ")
		if !isRubric || !strings.HasPrefix(line, "- ") || strings.Contains(name, " ") {
			continue
		}
		if !model.ValidSeverity(name) {
			t.Errorf("ReviewContract teaches severity %q, which model.ValidSeverity rejects", name)
		}
	}
}
