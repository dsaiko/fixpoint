package forge

import (
	"strings"
	"testing"
)

// The marker has to survive a round trip through the thing that decides whether a
// conversation is live: a reply this tool wrote must be recognized as its own, and
// nothing a person is likely to write must be.
func TestAMachineReplyIsRecognizedAndAHumanOneIsNot(t *testing.T) {
	machine := "Fixed in a.go:7.\n\n🤖 Answered by AI panel · claude-coder · run 20260807-153512\n" +
		ReplyMarker("20260807-153512")
	if !IsMachineReply(machine) {
		t.Errorf("a reply this tool posted was not recognized:\n%s", machine)
	}
	for name, human := range map[string]string{
		"plain":                   "I still think this is wrong.",
		"quoting our signature":   "> 🤖 Answered by AI panel · claude-coder · run 20260807-153512\n\nNo, look again.",
		"an unrelated html note":  "<!-- todo: revisit -->\nnot convinced",
		"the words without a tag": "ai-panel run 20260807-153512",
	} {
		t.Run(name, func(t *testing.T) {
			if IsMachineReply(human) {
				t.Errorf("a person's comment was mistaken for ours, so their question would go unanswered:\n%s", human)
			}
		})
	}
}

// A marker written by an EARLIER run must be recognized by a later one, which does
// not know that run's id. Matching the shape rather than the value is the whole
// reason this is a pattern and not a string comparison.
func TestAnyRunsMarkerIsRecognized(t *testing.T) {
	for _, id := range []string{"20260101-000000", "20991231-235959", ""} {
		if !IsMachineReply("answer\n" + ReplyMarker(id)) {
			t.Errorf("a marker from run %q was not recognized", id)
		}
	}
}

// It must render as nothing. That is the entire reason it is an HTML comment
// rather than a visible token: the reader sees the answer exactly as written.
//
// Checked as the property markdown renderers actually implement -- a well-formed
// comment that opens with <!-- and closes with the FIRST --> -- because a marker
// containing a stray > or -- would either display or swallow the text after it.
func TestTheMarkerIsAWellFormedCommentThatCannotSwallowText(t *testing.T) {
	m := ReplyMarker("20260807-153512")
	if !strings.HasPrefix(m, "<!--") || !strings.HasSuffix(m, "-->") {
		t.Fatalf("marker is not an HTML comment: %q", m)
	}
	// Exactly one terminator, at the end: an earlier one would end the comment and
	// leave the rest of the marker on the page.
	if strings.Count(m, "-->") != 1 {
		t.Errorf("marker %q contains more than one comment terminator", m)
	}
	// And nothing after it, so a reply ending in the marker ends in nothing visible.
	body := "the answer\n\n" + m
	visible := strings.TrimSpace(markerPattern.ReplaceAllString(body, ""))
	if visible != "the answer" {
		t.Errorf("removing the marker leaves %q, want just the answer: it is not rendering as nothing", visible)
	}
}

// A run id reaches this function from the log store, and a marker that could carry
// a newline would break out of its own comment and put bookkeeping on the page.
func TestTheMarkerStaysOnOneLine(t *testing.T) {
	if m := ReplyMarker("2026-08-07\nrogue"); strings.Contains(strings.TrimSuffix(m, "-->"), "\n") {
		t.Errorf("marker spans lines: %q", m)
	}
}

// A finding's identity has to be readable back out of what was posted, and only
// from comments that are ours by BOTH halves. A marker copied into somebody
// else's comment would otherwise let a third party suppress a finding from every
// future review of the pull request -- quieter, and worse, than a duplicate.
func TestPublishedFindingsReadsOurOwnMarkersOnly(t *testing.T) {
	ours := ThreadComment{Author: "dsaiko", Body: "**HIGH** — a defect\n" + FindingMarker("20260808-120000", "abc123def456")}
	forged := ThreadComment{Author: "stranger", Body: "looks fine to me\n" + FindingMarker("20260808-120000", "deadbeef0000")}
	unmarked := ThreadComment{Author: "dsaiko", Body: "just a comment"}
	threads := []Thread{{ID: "1", Comments: []ThreadComment{ours, forged, unmarked}}}

	got := PublishedFindings(threads, "dsaiko")
	if !got["abc123def456"] {
		t.Error("our own published finding was not recognized, so it will be posted again")
	}
	if got["deadbeef0000"] {
		t.Error("a marker in a third party's comment suppressed a finding from every future review")
	}
	if len(got) != 1 {
		t.Errorf("published = %v, want exactly ours", got)
	}
	// And with no login established, nothing is claimed to be published: a duplicate
	// is visible, a silently suppressed finding is not.
	if n := len(PublishedFindings(threads, "")); n != 0 {
		t.Errorf("published %d finding(s) with no authenticated account, want 0", n)
	}
}

// Both markers say "this message is ours", but only one says "we answered".
//
// An inline review comment is a question this tool ASKED, sitting in a thread
// nobody has replied to. Reading it as an answer made a fix run skip every
// conversation the review run before it had just opened -- measured on a real
// pull request: 12 published findings, all skipped, none fixed, none answered.
func TestAPublishedFindingIsOursButIsNotAnAnswer(t *testing.T) {
	finding := "**HIGH** — a defect\n" + FindingMarker("20260808-120000", "abc123")
	if !HasMarker(finding) {
		t.Error("a published finding is not recognized as written by this tool")
	}
	if IsMachineReply(finding) {
		t.Error("a finding we published is a question, not an answer: a fix run must still pick it up")
	}
	reply := "Fixed in a.go:7.\n" + ReplyMarker("20260808-120000")
	if !IsMachineReply(reply) || !HasMarker(reply) {
		t.Error("an answer must read as both ours and an answer")
	}
}

// Every reply posted before findings carried markers has no finding field, so the
// rule has to read those pull requests correctly without re-posting anything.
func TestAMarkerWithNoKindIsAReply(t *testing.T) {
	if !IsMachineReply("answered\n<!-- ai-panel run 20260807-153512 -->") {
		t.Error("a marker written before findings were marked must still read as a reply")
	}
}
