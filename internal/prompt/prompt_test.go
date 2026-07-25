package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

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
