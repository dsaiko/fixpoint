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
	if !HasReplyMarker(machine) {
		t.Errorf("a reply this tool posted was not recognized:\n%s", machine)
	}
	for name, human := range map[string]string{
		"plain":                   "I still think this is wrong.",
		"quoting our signature":   "> 🤖 Answered by AI panel · claude-coder · run 20260807-153512\n\nNo, look again.",
		"an unrelated html note":  "<!-- todo: revisit -->\nnot convinced",
		"the words without a tag": "ai-panel run 20260807-153512",
	} {
		t.Run(name, func(t *testing.T) {
			if HasReplyMarker(human) {
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
		if !HasReplyMarker("answer\n" + ReplyMarker(id)) {
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
