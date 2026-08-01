// Package prompt builds the final prompts sent to agents: the user-editable
// template from prompts/ plus the code-injected output contract, mode
// guidance, target material, findings, and history.
package prompt

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"unicode/utf8"

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
	Mode     config.Mode
	Path     string
	Round    int
	Findings string // concatenated findings to resolve
	History  string // prior rounds' findings + verdicts
	// Verification is empty on the first fix attempt of a round and holds the
	// deterministic gate's failures on the one correction attempt that follows.
	Verification string
	// Stale warns that the file this finding names has already been committed to
	// since the finding was written, by an earlier fix session in the SAME round.
	// Empty when the tree still stands where the reviewers saw it.
	Stale          string
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
      "issue": "<id from the History section if this is the SAME problem, else omit>",
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
Set "issue" only to re-report a problem already listed in History; omit it otherwise.
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

Every ISSUE id you were given must appear exactly once in results. Use verdict
"fixed" for issues you resolved and "rejected" for ones you decided not to act on,
with the reason. Duplicate reports have already been merged into single issues, so
you should not need to reconcile them yourself.
The <fix> block must be the LAST thing you print. The JSON must be valid: no
comments, no trailing commas, no markdown fences inside the block.`

// FormatIssues renders the issues handed to the coder: one entry per distinct
// problem, with every reviewer's description of it beneath.
//
// Corroboration is stated explicitly. Two independent agents reaching the same
// conclusion is the strongest evidence a review panel produces, and the coder
// deciding what is genuine should be told when it is present -- previously it saw
// two separate findings and had to work out for itself that they were one thing.
func FormatIssues(issues []model.Issue) string {
	var sb strings.Builder
	for _, it := range issues {
		fmt.Fprintf(&sb, "### [%s] (%s, %s) %s — %s\n", it.ID, it.Category, it.Severity, it.Loc(), it.Title)
		if agents := it.Agents(); len(agents) > 1 {
			fmt.Fprintf(&sb, "**Reported independently by %d agents (%s)** — corroborated, so treat it as more likely genuine.\n",
				len(agents), strings.Join(agents, ", "))
		}
		if it.Description != "" {
			sb.WriteString(it.Description + "\n")
		}
		if it.Suggestion != "" {
			sb.WriteString("Suggested: " + it.Suggestion + "\n")
		}
		// Additional readings, when they differ: a second description of the same
		// defect often names the cause the first one only gestured at.
		for _, o := range it.Observations {
			if o.Description == it.Description || o.Description == "" {
				continue
			}
			fmt.Fprintf(&sb, "\nAlso reported by %s via %s: %s\n", o.Agent, o.Lens, o.Description)
			if o.Suggestion != "" && o.Suggestion != it.Suggestion {
				sb.WriteString("Suggested: " + o.Suggestion + "\n")
			}
		}
		if it.Deferrals > 0 {
			fmt.Fprintf(&sb, "(deferred in %d earlier round(s) by the per-round cap)\n", it.Deferrals)
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

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

// historyRoundsInFull is how many of the most recent rounds FormatHistory renders
// with titles and verdict details. Older rounds collapse to one line per finding:
// id, location, and verdict.
//
// History is prepended to EVERY review prompt and grew without bound, which made
// the run slower the longer it ran. Measured over one 7-round run: 4.6 KB in round
// 1, 37.7 KB by round 7 -- an 8x growth against a fixed reviewer timeout, and three
// reviewers were killed mid-work at that timeout. The prompt is read in full by
// every reviewer, every round, so this is the one input whose size compounds.
//
// The tail is what a reviewer actually reasons about (what changed, and why the
// last verdicts went the way they did); older rounds only need to keep the reviewer
// from re-reporting decided issues, and an id with a verdict does that in a tenth of
// the bytes. Nothing is dropped outright -- a rejected finding from round 1 is still
// listed in round 9, because "do not re-report this" must survive the whole run.
const historyRoundsInFull = 3

// historyDetailMax caps a verdict detail inside the full window. Even with older
// rounds condensed, this is what is left of the bulk: a coder's rejection is
// argued at length -- the ones in the run that motivated this ran past 700
// characters each, citing commits and line numbers -- and history carries one per
// finding, to every reviewer, every round.
//
// A reviewer reads these to answer one question: has this already been decided,
// and does my evidence beat the reason it was decided that way? The headline
// answers it; the citation trail behind it is for a human reading the summary,
// which keeps the untruncated text. Cutting on a word boundary keeps the tail from
// ending mid-token.
const historyDetailMax = 240

// FormatHistory renders prior rounds for review prompts, so a freshly rotated
// reviewer knows what was already reported, fixed, and rejected. Rounds older than
// historyRoundsInFull are compacted -- see that constant for why.
func FormatHistory(rounds []model.RoundRecord) string {
	if len(rounds) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n## History of previous rounds\n")
	sb.WriteString("Do NOT re-report findings that were rejected below unless you have strong new evidence.\n")
	sb.WriteString("Findings marked FIXED were addressed -- verify the fix rather than re-reporting the original.\n")
	sb.WriteString("Findings marked DEFERRED or UNRESOLVED were NOT yet addressed -- report them again if still present.\n")
	sb.WriteString("Each entry starts with its issue id in brackets. If you report the SAME problem as one of these,\n")
	sb.WriteString("set \"issue\" to that id -- rewording it or pointing at a moved line would otherwise look like a new issue.\n\n")
	// The boundary is counted from the END, so the most recent rounds keep their
	// detail as the run gets longer.
	full := len(rounds) - historyRoundsInFull
	if full < 0 {
		full = 0
	}
	if full > 0 {
		fmt.Fprintf(&sb, "Rounds 1-%d, condensed (id, location, verdict -- ask the tree, not this list, for detail):\n",
			rounds[full-1].Round)
		for _, r := range rounds[:full] {
			for _, f := range r.Findings {
				fmt.Fprintf(&sb, "- [%s] %s — %s\n", historyID(f), f.Loc(), strings.ToUpper(f.VerdictOrDefault()))
			}
		}
		sb.WriteString("\n")
	}
	for _, r := range rounds[full:] {
		fmt.Fprintf(&sb, "Round %d (%d fixed, %d rejected):\n", r.Round, r.Fixed, r.Rejected)
		if len(r.Findings) == 0 {
			sb.WriteString("- no findings\n")
		}
		for _, f := range r.Findings {
			fmt.Fprintf(&sb, "- [%s] %s %s — %s: %s\n", historyID(f), f.Loc(), f.Title, strings.ToUpper(f.VerdictOrDefault()), clip(f.VerdictDetail, historyDetailMax))
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// FormatStale is the warning handed to a coder session whose issue names files
// that later commits in the same round have already touched. Empty when nothing
// the issue points at has moved.
//
// It is a caution, not an instruction to reject: the reviewers of a round all read
// ONE snapshot, and the per-fix sessions that follow commit into the tree one after
// another, so a finding written against that snapshot can arrive already addressed.
// The coder is the only party that can tell "already fixed by the last session"
// from "still broken in a file that happens to have changed", so it is told what
// moved and asked to look before editing.
func FormatStale(files []string) string {
	if len(files) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n## This finding may already be out of date\n")
	sb.WriteString("The reviewers of this round all read the same snapshot of the tree. Since then, EARLIER\n")
	sb.WriteString("fix sessions in this same round have committed changes to the file(s) this finding names:\n\n")
	for _, f := range files {
		sb.WriteString("- " + f + "\n")
	}
	sb.WriteString("\nRead the CURRENT contents before editing. If the problem is already resolved there, reject\n")
	sb.WriteString("the finding and say which commit resolved it -- do not re-apply a fix that has already landed.\n")
	return sb.String()
}

// clip shortens s to at most limit characters, breaking on the last word boundary
// before the limit and marking the cut so the reader knows text is missing rather
// than believing a sentence simply ended there. The limit counts runes, not bytes:
// a detail is free-form prose from an agent and may hold any UTF-8, and cutting
// mid-character would put invalid bytes into every prompt that carries history.
func clip(s string, limit int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	cut := s
	n := 0
	for i := range s {
		if n == limit {
			cut = s[:i]
			break
		}
		n++
	}
	if i := strings.LastIndexAny(cut, " \t\n"); i > len(cut)/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " \t\n.,;:") + " […]"
}

// historyID is the ISSUE id, not the observation id: that is what a reviewer must
// cite to declare a re-report, and what the ledger matches on.
func historyID(f model.Finding) string {
	if f.IssueID != "" {
		return f.IssueID
	}
	return f.ID
}
