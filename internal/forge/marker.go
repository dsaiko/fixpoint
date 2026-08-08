package forge

import (
	"regexp"
	"strings"
)

// ReplyMarker tags a reply this tool posted, so a later run can tell its own
// answer from a person's.
//
// It exists because author identity cannot answer that question. Replies go out
// under the operator's account, so "the last comment is mine" is equally true of a
// machine answer and of the operator typing a new request an hour later -- and
// treating the second as already-answered would swallow exactly the message a run
// should act on. Whether a message came from this tool is a fact about the
// MESSAGE, so it is recorded there.
//
// An HTML comment because both forges render one as nothing: a reader sees the
// reply exactly as written, with no machine bookkeeping in it. It deliberately
// does not name fixpoint -- the same reason the visible signature does not, since
// these land in repositories where the tool's own name means nothing -- and it
// carries the run id, which is what makes a reply traceable to the artifacts that
// produced it.
//
// Being invisible is not the same as being hidden: it is in the comment's source
// for anyone who looks, which is the point. On its own it is not a security
// control, because anybody who can comment can copy it: what decides whether a
// comment is this tool's is the marker AND its author being the account this run
// posts under (see Thread.ours). A copied marker in somebody else's comment
// therefore changes nothing about who is recorded as having asked for a change.
// The one thing it can still do is silence the forger's own conversation when the
// account's login could not be read at all, and that is a self-inflicted silence
// rather than an exploit.
func ReplyMarker(runID string) string {
	// The id is the run directory's name, so in practice it is a timestamp -- but it
	// is interpolated into a comment, and a comment that a value can break out of is
	// bookkeeping printed on somebody's pull request. Whitespace collapses to single
	// spaces so the marker stays one line, and the two sequences that can END a
	// comment early are removed rather than escaped: there is nothing inside a marker
	// worth preserving them for.
	return "<!-- ai-panel run " + markerSafe(runID) + " -->"
}

// FindingMarker tags a published FINDING with a stable identity, so a later run
// can tell what it has already said on this pull request from what is new.
//
// The identity is the caller's, and it has to be stable across runs for the same
// defect -- the issue ledger's fingerprint is, which is what makes "already
// reported" answerable at all. Without this, running a review twice over one
// commit posts the panel's findings again, and because the panel is not
// deterministic the second review is not even a copy: it overlaps, differs, and
// a reader has no way to tell it is the same code being described twice.
//
// Sanitized like the run id, for the same reason: this goes inside a comment, and
// a value that could close one early would print bookkeeping on the page.
func FindingMarker(runID, findingID string) string {
	id := markerSafe(findingID)
	if id == "" {
		return ReplyMarker(runID)
	}
	return "<!-- ai-panel run " + markerSafe(runID) + " finding " + id + " -->"
}

func markerSafe(s string) string {
	out := strings.Join(strings.Fields(s), " ")
	return strings.NewReplacer("--", "", ">", "").Replace(out)
}

// markerPattern matches any run's marker, since the reply being tested was
// written by an earlier run with an id this one does not know.
var markerPattern = regexp.MustCompile(`(?i)<!--\s*ai-panel run [^>]*-->`)

// findingPattern pulls the finding identity out of a marker that carries one.
var findingPattern = regexp.MustCompile(`(?i)<!--\s*ai-panel run [^>]*\bfinding ([^\s>]+)\s*-->`)

// HasMarker reports whether a comment was written by this tool -- a reply or a
// published finding.
func HasMarker(body string) bool { return markerPattern.MatchString(body) }

// IsMachineReply reports whether a comment is one of this tool's ANSWERS, as
// opposed to a finding it published.
//
// The distinction is the whole point, and conflating them was a real defect: an
// inline review comment is a question this tool ASKED, sitting in a thread nobody
// has answered yet, and treating it as an answer made a fix run skip the very
// conversations a review run had just opened for it. What the two mean for "is
// anybody waiting on us?" is opposite.
//
// A marker with no finding field is a reply. That is what every reply posted
// before findings were marked at all looks like, so the rule reads existing pull
// requests correctly rather than needing them re-posted.
func IsMachineReply(body string) bool {
	return markerPattern.MatchString(body) && !findingPattern.MatchString(body)
}

// PublishedFindings lists the finding identities already posted on these
// conversations by this tool, from comments that are ours by BOTH halves --
// marker and authoring account.
//
// Both halves for the same reason AnsweredByMachine needs both: a marker copied
// into somebody else's comment would otherwise let a third party suppress a
// finding from every future review of this pull request, which is a quieter and
// worse outcome than a duplicate.
func PublishedFindings(threads []Thread, me string) map[string]bool {
	out := map[string]bool{}
	for _, t := range threads {
		for _, c := range t.Comments {
			if !c.ours(me) {
				continue
			}
			if m := findingPattern.FindStringSubmatch(c.Body); m != nil {
				out[m[1]] = true
			}
		}
	}
	return out
}
