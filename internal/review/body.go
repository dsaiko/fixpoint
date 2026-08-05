package review

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dsaiko/fixpoint/internal/model"
)

// BodyInput is everything the review body renders from. Like Decide's input it is
// plain data: the renderer performs no I/O and asks nothing, so what a reviewer
// will read can be asserted in a test rather than eyeballed after a run.
type BodyInput struct {
	Config    string // the task config's bare name, for the header
	Target    string // "pr #170", "git-diff", "directory" -- what was reviewed
	Decision  Decision
	Issues    []model.Issue   // everything that survived, blocking or not
	Advisory  []model.Finding // reported for a human; gates nothing
	Signature string          // already rendered; see Signature
	Panel     []string        // agents that reviewed, for the header line
}

// RenderBody produces the review document: the same text whether it is written to
// a file or posted to a pull request.
//
// One renderer for both on purpose. The posted review and the local file must not
// be able to drift, because the local file is what an operator reads to decide
// whether posting is safe -- a body that looked different once posted would make
// that check worthless.
//
// The VERDICT leads. A reader who stops after two lines should have the answer,
// and the reasons that produced it are right beneath, because a verdict nobody can
// audit is not much better than a model's opinion.
func RenderBody(in BodyInput) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## %s\n\n", verdictHeadline(in.Decision.Outcome))
	for _, r := range in.Decision.Reasons {
		fmt.Fprintf(&b, "- %s\n", r)
	}
	b.WriteString("\n")

	blocking, other := split(in.Issues, in.Decision)
	if len(blocking) > 0 {
		fmt.Fprintf(&b, "### Blocking (%d)\n\n", len(blocking))
		for _, it := range blocking {
			writeIssue(&b, it)
		}
	}
	if len(other) > 0 {
		fmt.Fprintf(&b, "### Other findings (%d)\n\n", len(other))
		for _, it := range other {
			writeIssue(&b, it)
		}
	}
	if len(blocking)+len(other) == 0 {
		b.WriteString("No findings.\n\n")
	}

	if len(in.Advisory) > 0 {
		// Advisory notes are excluded from the verdict by contract, so they are
		// rendered apart from the findings rather than mixed in where a reader would
		// reasonably assume they counted.
		fmt.Fprintf(&b, "### Advisory (%d)\n\nReported for a human; these did not affect the verdict.\n\n", len(in.Advisory))
		for _, f := range in.Advisory {
			fmt.Fprintf(&b, "- **%s** — %s\n", mdText(f.Title), mdText(firstSentence(f.Description)))
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n\n")
	if len(in.Panel) > 0 {
		fmt.Fprintf(&b, "Reviewed by %s", strings.Join(mdTexts(in.Panel), ", "))
		if in.Target != "" {
			fmt.Fprintf(&b, " over %s", mdText(in.Target))
		}
		b.WriteString(".\n\n")
	}
	if in.Signature != "" {
		fmt.Fprintf(&b, "%s\n", in.Signature)
	}
	return b.String()
}

func verdictHeadline(o Outcome) string {
	switch o {
	case Approve:
		return "Approved"
	case ChangesRequested:
		return "Changes requested"
	case Inconclusive:
		return "Inconclusive"
	}
	return "Reviewed"
}

// split separates the findings the verdict actually rests on from the rest, so a
// reader is not left to work out which of fourteen findings blocked the merge.
func split(issues []model.Issue, d Decision) (blocking, other []model.Issue) {
	isBlocking := make(map[string]bool, len(d.Blocking))
	for _, it := range d.Blocking {
		isBlocking[it.ID] = true
	}
	for _, it := range issues {
		switch it.StatusOrDefault() {
		case model.VerdictFixed, model.VerdictRejected:
			continue // decided; not something the reader has to act on
		}
		if isBlocking[it.ID] {
			blocking = append(blocking, it)
			continue
		}
		other = append(other, it)
	}
	bySeverity := func(s []model.Issue) {
		sort.SliceStable(s, func(i, j int) bool {
			return model.SeverityRank(s[i].Severity) < model.SeverityRank(s[j].Severity)
		})
	}
	bySeverity(blocking)
	bySeverity(other)
	return blocking, other
}

func writeIssue(b *strings.Builder, it model.Issue) {
	loc := mdText(it.File)
	if it.Line > 0 {
		loc = fmt.Sprintf("%s:%d", loc, it.Line)
	}
	fmt.Fprintf(b, "**%s** · `%s`", strings.ToUpper(mdText(it.Severity)), loc)
	if agents := it.Agents(); len(agents) > 1 {
		// Corroboration is rare enough in practice to be worth saying out loud when
		// it happens: across 19 measured runs under 4% of findings had it.
		fmt.Fprintf(b, " · reported by %d reviewers", len(agents))
	}
	fmt.Fprintf(b, "\n\n%s\n\n", mdText(it.Title))
	if d := strings.TrimSpace(it.Description); d != "" {
		fmt.Fprintf(b, "%s\n\n", mdText(d))
	}
	if s := strings.TrimSpace(it.Suggestion); s != "" {
		fmt.Fprintf(b, "_Suggested:_ %s\n\n", mdText(s))
	}
}

// mdText is the ONE place agent-authored text enters the document, and every
// caller above goes through it. Right now it only strips the control characters
// that would corrupt any renderer; the forge-specific neutralization (@mentions,
// issue references, closing keywords) is layered on top of this by whoever posts
// it, because those hazards exist on a pull request and not in a local file.
//
// Keeping the funnel narrow is the point: a future field rendered directly would
// bypass every protection at once, and a reviewer's prose is written by a model
// that has just read code somebody else controls.
func mdText(s string) string { return model.StripControl(s) }

func mdTexts(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, mdText(s))
	}
	return out
}

// firstSentence keeps an advisory line to one line. Advisory notes are prose and
// often long; the full text is in the run's artifacts.
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, ".\n"); i > 0 {
		return s[:i+1]
	}
	return s
}
