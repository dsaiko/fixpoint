package review

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/issue"
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
	// RunID is this run's id, carried by the identity markers the body ends with.
	// See writeFindingMarkers for why the body needs them at all.
	RunID string
	// AlreadyPublished is what this pull request already carries from an earlier
	// review, keyed by FindingID -- and by AdvisoryID for the notes, which are
	// published in the body and nowhere else. They are omitted from the lists and
	// counted in a line of their own, one line per kind: an advisory note gates
	// nothing, and the findings' line says the verdict accounts for what it left
	// out.
	//
	// Counted rather than dropped silently: a review that showed three findings
	// where a previous one showed thirty, with nothing to say the difference is
	// history rather than progress, would read as a project that had just been
	// cleaned up.
	AlreadyPublished map[string]bool
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
		// Through the same funnel as every other line. A reason reads like fixpoint's
		// own words, but it interpolates strings the reviewed repository controls: the
		// names of its failing and pending checks (from `gh pr view`), and the agent
		// names in the quorum note. An unsanitized check called `@victim` or
		// `Closes #42` would act on the forge under the operator's identity from the
		// one region of the document a reader trusts most.
		fmt.Fprintf(&b, "- %s\n", mdText(r))
	}
	b.WriteString("\n")

	blocking, other := split(in.Issues, in.Decision)
	var repeated int
	if len(in.AlreadyPublished) > 0 {
		blocking, repeated = withoutPublished(blocking, in.AlreadyPublished)
		var n int
		other, n = withoutPublished(other, in.AlreadyPublished)
		repeated += n
	}
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
		if repeated > 0 {
			b.WriteString("No findings that are not already reported on this pull request.\n\n")
		} else {
			b.WriteString("No findings.\n\n")
		}
	}
	if repeated > 0 {
		// The count, always: it is what tells a reader that a short list is a delta
		// against what is already here rather than a clean bill of health. The verdict
		// above still counts every surviving finding, including these -- what was
		// already said is still true.
		fmt.Fprintf(&b, "_%d further finding(s) are already reported on this pull request and are not repeated here. The verdict above accounts for them._\n\n", repeated)
	}

	advisory := in.Advisory
	var repeatedAdvisory int
	if len(in.AlreadyPublished) > 0 {
		// The same delta as the findings above, and separately counted: an advisory
		// note gates nothing, so folding it into that line -- which tells the reader
		// the verdict accounts for what it omitted -- would say something untrue about
		// the verdict.
		advisory, repeatedAdvisory = advisoryWithoutPublished(advisory, in.AlreadyPublished)
	}
	if len(advisory) > 0 {
		// Advisory notes are excluded from the verdict by contract, so they are
		// rendered apart from the findings rather than mixed in where a reader would
		// reasonably assume they counted.
		fmt.Fprintf(&b, "### Advisory (%d)\n\nReported for a human; these did not affect the verdict.\n\n", len(advisory))
		for _, f := range advisory {
			fmt.Fprintf(&b, "- **%s** — %s\n", mdText(f.Title), mdText(firstSentence(f.Description)))
		}
		b.WriteString("\n")
	}
	if repeatedAdvisory > 0 {
		fmt.Fprintf(&b, "_%d further advisory note(s) are already reported on this pull request and are not repeated here._\n\n", repeatedAdvisory)
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
	writeFindingMarkers(&b, in.RunID, blocking, other, advisory)
	return b.String()
}

// writeFindingMarkers ends the body with the identity of everything it just said,
// so the NEXT review of this pull request can tell what it has already reported.
//
// The body needs its own markers because an inline comment cannot carry them for
// it. A forge accepts an anchor only inside the pull request's own diff, and
// measured on this project's own pull requests most findings point at code the
// change did not touch -- those, plus every finding with no location at all, plus
// the entire review whenever the forge rejects the anchors and the summary is
// posted alone, exist on the pull request only as these paragraphs. Marked only
// where they anchored, the majority of a review was invisible to the next one and
// came back verbatim, under a body claiming the omissions were accounted for.
//
// Everything RENDERED, including the findings that did get an inline comment: the
// duplicate marker costs nothing, and singling out the unanchored ones would mean
// this list and the anchoring rule had to agree forever.
//
// HTML comments, which both forges render as nothing -- the reader sees the review
// as written. Invisible is not hidden: they are in the source for anyone who
// looks, and being copyable is why a marker alone never proves authorship (see
// forge.PublishedFindings).
func writeFindingMarkers(b *strings.Builder, runID string, blocking, other []model.Issue, advisory []model.Finding) {
	if len(blocking)+len(other)+len(advisory) == 0 {
		return
	}
	b.WriteString("\n")
	for _, group := range [][]model.Issue{blocking, other} {
		for _, it := range group {
			fmt.Fprintf(b, "%s\n", forge.FindingMarker(runID, FindingID(it)))
		}
	}
	for _, f := range advisory {
		fmt.Fprintf(b, "%s\n", forge.FindingMarker(runID, AdvisoryID(f)))
	}
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
	// mdCode, not mdText: the location is rendered inside a code span two lines down,
	// and mdText deliberately leaves backticks alone. A reported path carrying one
	// would close that span early, and everything after it becomes live markdown in a
	// review posted under the operator's identity.
	loc := mdCode(it.File)
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
	if it.Contested {
		// The same notice RenderInline gives, for the same reason: a finding the
		// panel split over must not read like one it agreed on. Most findings never
		// get an inline comment -- they have no addressable line, or the review is
		// posted body-only -- so this section is the only place the reader would
		// learn it.
		b.WriteString("_The panel disagreed about this one._\n\n")
	}
}

// mdText is the ONE place agent-authored text enters the document, and every
// caller above goes through it. Keeping the funnel narrow is the point: a future
// field rendered directly would bypass every protection at once, and a reviewer's
// prose is written by a model that has just read code somebody else controls.
//
// It applies the FORGE rules (see forge.SanitizeText) even when the document is
// only going to a file. That is deliberate: the file is what an operator reads to
// decide whether posting is safe, so it has to be byte-for-byte what would be
// posted. Sanitizing on the way out instead would make that inspection worthless.
func mdText(s string) string { return forge.SanitizeText(s) }

// mdCode is the same funnel for a string that lands inside a code span, where the
// span's own delimiter is part of the attack surface. It is forge.CodeSpan rather
// than a local escape so this and the GitLab inline-note path cannot drift.
func mdCode(s string) string { return forge.CodeSpan(s) }

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

// RenderInline is one finding as a comment on its own line.
//
// Shorter than its entry in the summary and deliberately so: it is read in a diff
// with the code beside it, so the location it would otherwise repeat is already
// on screen. Severity leads, because a reader skimming the Files tab needs to
// tell a blocker from a note without opening anything.
//
// Same mdText funnel as the body: this text goes to the same place under the same
// rules, and having a second path into a forge comment is how one of them ends up
// without the protections.
//
// SIGNED, like the summary. An inline comment is read on its own, in the Files
// tab, with no sight of the review it belongs to -- so without a signature it is
// an unattributed assertion sitting on somebody's code, and the reader cannot tell
// a machine's opinion from a colleague's. The signature is rendered by fixpoint
// and appended after all agent text, for the same reason it is in the body: one
// composed from a finding's prose could be forged by whatever wrote that prose.
func RenderInline(it model.Issue, signature string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** — %s\n\n", strings.ToUpper(mdText(it.Severity)), mdText(it.Title))
	if d := strings.TrimSpace(it.Description); d != "" {
		fmt.Fprintf(&b, "%s\n\n", mdText(d))
	}
	if s := strings.TrimSpace(it.Suggestion); s != "" {
		fmt.Fprintf(&b, "_Suggested:_ %s\n", mdText(s))
	}
	if it.Contested {
		b.WriteString("\n_The panel disagreed about this one._\n")
	}
	if signature != "" {
		fmt.Fprintf(&b, "\n%s\n", signature)
	}
	return b.String()
}

// FindingID is the stable identity of a finding as published on a pull request.
//
// The ledger's fingerprint is what makes "have we already reported this?"
// answerable across runs -- it is derived from the location, or from the title
// when there is no line -- but it is a path and a sentence, which cannot go inside
// an HTML comment. Hashed to a hex string, which can -- see findingID for why the
// digest is not truncated to display size.
//
// The fingerprint ALONE is not that identity, though, and using it would be the
// one failure this whole feature exists to avoid. A located fingerprint is a path
// and a line, and issue.fingerprintMatch pairs it with title agreement precisely
// because one statement routinely holds two defects -- the nil deref and the
// unchecked error it came from. Keyed on the location alone, a second panel's NEW
// finding at a line the first panel already commented on would be dropped from the
// body and from the inline comments, and counted in the "already reported" line:
// the reader told a finding was repeated when in fact it was withheld, on exactly
// the case a second review is run for. So the hash covers the normalized title
// too, the same pair the ledger calls one defect. Two wordings that normalize
// alike still collide; two defects on one line do not, and the worst that costs is
// a visible duplicate.
func FindingID(it model.Issue) string {
	fp := it.Fingerprint
	if fp == "" {
		// An issue that reached here without one still needs an identity, and its
		// location plus title is what the fingerprint would have been built from.
		fp = fmt.Sprintf("%s#L%d", it.File, it.Line)
	}
	return findingID(fp, it.Title)
}

// AdvisoryID is that same identity for an advisory note, which is a Finding and
// so has no ledger fingerprint of its own -- issue.Fingerprint computes the one it
// would have had, from the same location-or-title rule.
//
// It exists because an advisory note is published in the body and nowhere else,
// so without an identity it was the one part of a review that came back in full
// every time.
func AdvisoryID(f model.Finding) string {
	return findingID(issue.Fingerprint(f), f.Title)
}

// findingID hashes the pair the ledger calls one defect. 16 bytes of the digest,
// not the 6 a display id would want: this value is the sole thing that decides
// whether a finding is WITHHELD from a pull request, and its pre-image -- a path,
// a line, and a title -- is chosen by whoever opens that pull request. At 48 bits
// a collision can be searched for offline in hours on commodity hardware, so an
// attacker could land a padding file whose obvious defect hashes to the identity
// of a real finding at the line they intend to backdoor; the first review
// publishes the decoy's marker, and every later review drops the real finding as
// already reported, counted in a line asserting the verdict accounts for it. A
// suppression key has to cost more to collide than the thing it suppresses is
// worth. 128 bits does; the marker is an HTML comment, where the extra 20
// characters cost nothing and the marker pattern already accepts any non-space
// run.
func findingID(fingerprint, title string) string {
	sum := sha256.Sum256([]byte(fingerprint + "#" + issue.NormalizeTitle(title)))
	return hex.EncodeToString(sum[:16])
}

// withoutPublished drops the findings this pull request already carries, and
// reports how many were dropped.
func withoutPublished(issues []model.Issue, published map[string]bool) (kept []model.Issue, dropped int) {
	kept = make([]model.Issue, 0, len(issues))
	for _, it := range issues {
		if published[FindingID(it)] {
			dropped++
			continue
		}
		kept = append(kept, it)
	}
	return kept, dropped
}

// advisoryWithoutPublished is the same for the advisory notes, keyed by AdvisoryID.
func advisoryWithoutPublished(notes []model.Finding, published map[string]bool) (kept []model.Finding, dropped int) {
	kept = make([]model.Finding, 0, len(notes))
	for _, f := range notes {
		if published[AdvisoryID(f)] {
			dropped++
			continue
		}
		kept = append(kept, f)
	}
	return kept, dropped
}
