// Package prompt builds the final prompts sent to agents: the user-editable
// template from prompts/ plus the code-injected output contract, mode
// guidance, target material, findings, and history.
package prompt

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

// ReviewData is the placeholder set available to review prompt templates.
// Review and fix placeholders are separate types on purpose: a template
// referencing the other role's placeholder fails at execution instead of
// silently rendering an empty value.
type ReviewData struct {
	Mode           config.Mode
	Path           string
	Round          int
	ModeGuidance   string
	Target         string // the collected material
	History        string // prior rounds' findings + verdicts
	OutputContract string
}

// FixData is the placeholder set available to the coder prompt template.
type FixData struct {
	Mode           config.Mode
	Path           string
	Round          int
	Findings       string // concatenated findings to resolve
	History        string // prior rounds' findings + verdicts
	OutputContract string
}

// Load parses a prompt template file.
func Load(path string) (*template.Template, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t, err := template.New(path).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return t, nil
}

// Render executes the template with the given data (ReviewData or FixData).
func Render(t *template.Template, d any) (string, error) {
	var sb strings.Builder
	if err := t.Execute(&sb, d); err != nil {
		return "", fmt.Errorf("render %s: %w", t.Name(), err)
	}
	return sb.String(), nil
}

// ModeGuidance returns the one-paragraph steer that differs per target mode.
func ModeGuidance(mode config.Mode) string {
	switch mode {
	case config.ModeGitDiff:
		return "The material below is a git diff of the changes under review. Judge the " +
			"changes and their impact on surrounding code. You are running inside the " +
			"repository -- read any file you need for context."
	case config.ModePR:
		return "The material below is the diff of a pull request checked out locally. " +
			"Judge the changes and their impact on surrounding code. You are running " +
			"inside the repository -- read any file you need for context."
	case config.ModeDirectory:
		return "The material below is a listing of the files in scope. You are running " +
			"inside the repository -- read and explore the listed files yourself; the " +
			"listing is an index, not the content."
	default:
		return ""
	}
}

// ReviewContract is the output contract injected into review prompts. It carries
// the severity rubric as well as the JSON shape: severity decides which findings
// reach the coder when a round is capped, so it is a scheduling input, not a
// label. Left to each lens, the same issue drew "low" from one agent and "high"
// from another in one run, and a missing test outranked a credential-leak gap.
// Keeping the rubric here means every lens inherits it and a new lens cannot
// forget it.
const ReviewContract = `## Severity
Severity decides which findings reach the fixer first when a round is capped, so
rate by impact on the running system:

- critical: exploitable now, or causes data loss or corruption in normal use
- high: wrong behavior, or a security weakness, on a path that is actually reached
- medium: wrong behavior on an unlikely path, or a real defect with a workaround
- low: correct today but fragile, misleading, or undocumented

Rate the defect, not the effort to fix it and not how interesting it is. A
one-line documentation error stays low even though it is trivial to fix. A missing
test for a security control is high — not because tests matter in the abstract,
but because that control can silently stop working with nothing to catch it.

## Required output format
End your response with exactly one <review> block containing valid JSON:

<review>
{
  "findings": [
    {
      "category": "<the category your instructions above told you to set>",
      "severity": "critical|high|medium|low",
      "file": "relative/path.go",
      "line": 42,
      "title": "one-line summary",
      "description": "what is wrong and why it matters",
      "suggestion": "how to fix it"
    }
  ]
}
</review>

If you have no findings, output "findings": [].
The <review> block must be the LAST thing you print. The JSON must be valid:
no comments, no trailing commas, no markdown fences inside the block.`

// FixContract is the output contract injected into the coder prompt.
const FixContract = `## Required output format
End your response with exactly one <fix> block containing valid JSON:

<fix>
{
  "results": [
    {"id": "<finding id>", "verdict": "fixed", "detail": "what you changed and why"},
    {"id": "<finding id>", "verdict": "rejected", "detail": "why this is not a genuine issue"}
  ],
  "notes": "anything else worth recording"
}
</fix>

Every finding id you were given must appear exactly once in results. Use
verdict "fixed" for findings you resolved (a duplicate of a finding you fixed
also counts as "fixed" -- say so in detail). Use "rejected" for findings you
decided not to act on, with the reason.
The <fix> block must be the LAST thing you print. The JSON must be valid: no
comments, no trailing commas, no markdown fences inside the block.`

// FormatFindings renders findings as the markdown block handed to the coder.
func FormatFindings(findings []model.Finding) string {
	var sb strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&sb, "### [%s] (%s, %s) %s — %s\n", f.ID, f.Category, f.Severity, f.Loc(), f.Title)
		if f.Description != "" {
			sb.WriteString(f.Description + "\n")
		}
		if f.Suggestion != "" {
			sb.WriteString("Suggested: " + f.Suggestion + "\n")
		}
		fmt.Fprintf(&sb, "(reported by %s via %s)\n\n", f.Agent, f.Lens)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// FormatHistory renders prior rounds for review prompts, so a freshly rotated
// reviewer knows what was already reported, fixed, and rejected.
func FormatHistory(rounds []model.RoundRecord) string {
	if len(rounds) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n## History of previous rounds\n")
	sb.WriteString("Do NOT re-report findings that were rejected below unless you have strong new evidence.\n")
	sb.WriteString("Findings marked FIXED were addressed -- verify the fix rather than re-reporting the original.\n")
	sb.WriteString("Findings marked DEFERRED or UNRESOLVED were NOT yet addressed -- report them again if still present.\n\n")
	for _, r := range rounds {
		fmt.Fprintf(&sb, "Round %d (%d fixed, %d rejected):\n", r.Round, r.Fixed, r.Rejected)
		if len(r.Findings) == 0 {
			sb.WriteString("- no findings\n")
		}
		for _, f := range r.Findings {
			fmt.Fprintf(&sb, "- [%s] %s %s — %s: %s\n", f.ID, f.Loc(), f.Title, strings.ToUpper(f.VerdictOrDefault()), f.VerdictDetail)
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
