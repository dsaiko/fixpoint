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
	id := strings.Join(strings.Fields(runID), " ")
	id = strings.NewReplacer("--", "", ">", "").Replace(id)
	return "<!-- ai-panel run " + id + " -->"
}

// markerPattern matches any run's marker, since the reply being tested was
// written by an earlier run with an id this one does not know.
var markerPattern = regexp.MustCompile(`(?i)<!--\s*ai-panel run [^>]*-->`)

// HasReplyMarker reports whether a comment body carries a machine-reply marker.
func HasReplyMarker(body string) bool { return markerPattern.MatchString(body) }
