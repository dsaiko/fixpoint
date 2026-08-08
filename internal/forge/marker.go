package forge

import (
	"regexp"
	"slices"
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
// posts under (see ThreadComment.Ours). A copied marker in somebody else's comment
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

// FindingMarker tags a published FINDING with a stable identity AND the commit it
// was published about, so a later run can tell what it has already said on this
// pull request from what is new.
//
// The identity is the caller's, and it has to be stable across runs for the same
// defect -- the issue ledger's fingerprint is, which is what makes "already
// reported" answerable at all. Without this, running a review twice over one
// commit posts the panel's findings again, and because the panel is not
// deterministic the second review is not even a copy: it overlaps, differs, and
// a reader has no way to tell it is the same code being described twice.
//
// head is the commit the review was produced from (RunSummary.ReviewedHead), and
// it is in the marker because an identity on its own says "this was said once",
// never "this is still what the code does". A pull request lives across pushes: a
// finding fixed on one head and a NEW defect of the same kind at the same path and
// line on a later one hash to the same identity. Recognized without the revision,
// the current finding's description -- the exploit, the reproduction, the
// suggestion -- was withheld from the review body and from the inline comments and
// replaced by a count, while the thread the reader is implicitly pointed at may be
// resolved, outdated, or about code that no longer exists. For a security finding
// that description is the review. So the marker names the revision and
// PublishedFindings hands it back, for the caller to weigh against what has moved
// since (see review.Published).
//
// A marker with NO head -- a run that could not read one, and every marker posted
// before this field existed -- is a statement about an unknown revision.
// PublishedFindings does not collect it, so it withholds nothing: that costs a
// visible duplicate, which is the direction every failure on this path errs in.
//
// Sanitized like the run id, for the same reason: this goes inside a comment, and
// a value that could close one early would print bookkeeping on the page.
func FindingMarker(runID, head, findingID string) string {
	id := markerSafe(findingID)
	if id == "" {
		return ReplyMarker(runID)
	}
	m := "<!-- ai-panel run " + markerSafe(runID)
	if h := markerSafe(head); h != "" {
		m += " head " + h
	}
	return m + " finding " + id + " -->"
}

func markerSafe(s string) string {
	out := strings.Join(strings.Fields(s), " ")
	return strings.NewReplacer("--", "", ">", "").Replace(out)
}

// markerPattern matches any run's marker, since the reply being tested was
// written by an earlier run with an id this one does not know.
var markerPattern = regexp.MustCompile(`(?i)<!--\s*ai-panel run [^>]*-->`)

// findingPattern says a marker is a published FINDING rather than a reply. It
// deliberately does not require the head field, so a marker written before that
// field existed is still not mistaken for one of this tool's answers.
var findingPattern = regexp.MustCompile(`(?i)<!--\s*ai-panel run [^>]*\bfinding ([^\s>]+)\s*-->`)

// publishedPattern pulls the identity AND the commit it was published about out of
// a marker that carries both -- the only form that may withhold a later finding.
// See FindingMarker for why a marker naming no commit does not qualify.
var publishedPattern = regexp.MustCompile(`(?i)<!--\s*ai-panel run [^\s>]+ head ([^\s>]+) finding ([^\s>]+)\s*-->`)

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

// PublishedFindings maps each finding identity already posted on this pull request
// by this tool to the COMMITS it was posted about, from conversations and review
// summaries alike, taking only what is ours by BOTH halves -- marker and authoring
// account.
//
// The commits are half the answer and not decoration. "This identity was published"
// does not mean "the code it describes is still the code that is there": an
// identity covers a path, a line and a title, all of which a later push can restore
// over different code. The caller decides what a given commit still vouches for by
// asking what has moved since it -- see review.Published, which is where that
// judgment lives. A marker naming no commit is not collected at all, so it can
// withhold nothing; see FindingMarker.
//
// Both halves for the same reason AnsweredByMachine needs both: a marker copied
// into somebody else's comment would otherwise let a third party suppress a
// finding from every future review of this pull request, which is a quieter and
// worse outcome than a duplicate.
//
// Both SOURCES because a finding usually has only one of them. An inline comment
// needs a line inside the pull request's own diff, and most findings point at
// code the change did not touch; those are published in the review body alone, as
// is every finding when the forge rejects the anchors and the summary goes out on
// its own. Reading conversations alone recognized the anchored minority and let
// the rest be reprinted in full by every later review.
func PublishedFindings(threads []Thread, reviews []Review, me string) map[string][]string {
	out := map[string][]string{}
	for _, t := range threads {
		for _, c := range t.Comments {
			if c.Ours(me) {
				collectFindings(c.Body, out)
			}
		}
	}
	for _, r := range reviews {
		if r.ours(me) {
			collectFindings(r.Body, out)
		}
	}
	return out
}

// collectFindings adds every identity a body carries, under the commit it was
// published about. Every one, not the first: a review summary lists the whole
// review, so its markers come as a block, and reading one of them would have
// recognized one finding per earlier review.
//
// One identity can arrive with several commits -- two reviews of the same pull
// request said it, on two heads -- and each is kept, because whether any of them
// still describes the current code is the caller's question to answer.
func collectFindings(body string, out map[string][]string) {
	for _, m := range publishedPattern.FindAllStringSubmatch(body, -1) {
		head, id := m[1], m[2]
		if slices.Contains(out[id], head) {
			continue
		}
		out[id] = append(out[id], head)
	}
}
