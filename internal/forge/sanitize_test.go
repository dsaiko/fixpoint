package forge

import (
	"strings"
	"testing"
)

// A review comment is where a finding stops being a local file and becomes an
// action on a shared platform. Each of these is something an injected finding
// could otherwise make fixpoint do under its own name.
func TestSanitizeTextNeutralizesForgeActions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		in         string
		wantGone   []string // must not survive as a live construct
		wantStayed []string // the reader must still be able to read what was written
	}{
		{
			name:       "closing keyword with a bare reference",
			in:         "This duplicates the work in Closes #42, see there.",
			wantGone:   []string{"Closes #42"},
			wantStayed: []string{"Closes", "42"},
		},
		{
			name:       "owner/repo shorthand",
			in:         "Same as oddin-gg/fujin#170.",
			wantGone:   []string{"fujin#170"},
			wantStayed: []string{"oddin-gg/fujin", "170"},
		},
		{
			name:       "GH- spelling",
			in:         "Fixes GH-7 as well.",
			wantGone:   []string{"GH-7"},
			wantStayed: []string{"Fixes", "7"},
		},
		{
			name:       "full issue URL",
			in:         "Resolves https://github.com/o/r/issues/12 too.",
			wantGone:   []string{"/issues/12"},
			wantStayed: []string{"github.com/o/r/issues/", "12"},
		},
		{
			name:       "user mention",
			in:         "Ask @octocat about this.",
			wantGone:   []string{"@octocat"},
			wantStayed: []string{"octocat"},
		},
		{
			name:       "team mention",
			in:         "cc @acme/security-team",
			wantGone:   []string{"@acme/security-team"},
			wantStayed: []string{"acme/security-team"},
		},
		{
			name:       "html comment that would swallow the rest of the document",
			in:         "harmless <!-- everything after this vanishes",
			wantGone:   []string{"<!--"},
			wantStayed: []string{"everything after this vanishes"},
		},
		{
			name:       "html comment terminator",
			in:         "text --> more",
			wantGone:   []string{"-->"},
			wantStayed: []string{"more"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeText(tc.in)
			for _, gone := range tc.wantGone {
				if strings.Contains(got, gone) {
					t.Errorf("%q survived in %q", gone, got)
				}
			}
			for _, stayed := range tc.wantStayed {
				if !strings.Contains(got, stayed) {
					t.Errorf("%q was lost from %q; the reader must still see what the finding said", stayed, got)
				}
			}
		})
	}
}

// The neutralization has to be invisible when rendered. Mangling the text instead
// would make fixpoint misquote its own evidence, and a review that misquotes the
// finding it is reporting is worse than one that pings somebody.
func TestMentionBreakIsAnEmptyComment(t *testing.T) {
	if got := SanitizeText("ping @octocat"); got != "ping @<!---->octocat" {
		t.Errorf("SanitizeText() = %q, want the empty-comment break", got)
	}
}

// Ordinary text must come through untouched: a sanitizer that rewrites prose
// nobody asked it to rewrite gets turned off.
func TestSanitizeTextLeavesOrdinaryProseAlone(t *testing.T) {
	for _, in := range []string{
		"The comparison at token.go:91 is not constant time.",
		"Use subtle.ConstantTimeCompare instead of ==.",
		"See docs/issues/1-intro.md for the background.", // a path, not a reference
		"Contact build@example.com if this breaks.",      // an address, not a mention
		"The channel is unbuffered, so the send blocks.", // no reference at all
	} {
		if got := SanitizeText(in); got != in {
			t.Errorf("SanitizeText(%q) = %q, want it unchanged", in, got)
		}
	}
}

// Any #N is a live reference on both forges, including one that reads as an
// ordinal in prose. Breaking it costs a space in the text; not breaking it lets
// `Closes #1` through, and there is no way to tell the two apart from the string.
func TestSanitizeTextBreaksEveryNumericReference(t *testing.T) {
	for _, tc := range []struct{ in, ref string }{
		{"#0", "#0"},
		{"#1 is first", "#1"},
		{"issue #999", "#999"},
	} {
		if got := SanitizeText(tc.in); strings.Contains(got, tc.ref) {
			t.Errorf("SanitizeText(%q) = %q, want %q broken", tc.in, got, tc.ref)
		}
	}
}

// A path that merely contains "issues" names no issue: only the URL shape does.
func TestSanitizeTextDoesNotBreakOrdinaryPaths(t *testing.T) {
	const in = "docs/issues/12-notes.md"
	if got := SanitizeText(in); got != in {
		t.Errorf("SanitizeText(%q) = %q; a plain path keeps its spelling", in, got)
	}
}

// The delimiters must be escaped BEFORE the mention break inserts comments of its
// own, or a payload could close the comment fixpoint just opened.
func TestCommentEscapingHappensBeforeMentionsAreBroken(t *testing.T) {
	got := SanitizeText("--> @octocat")
	if strings.Contains(got, "--> ") {
		t.Errorf("a live comment terminator survived: %q", got)
	}
	if !strings.Contains(got, "@<!---->octocat") {
		t.Errorf("the mention was not broken: %q", got)
	}
	// The escaped terminator must not be able to close the inserted comment.
	if strings.Index(got, "--&gt;") > strings.Index(got, "@<!---->") {
		t.Errorf("ordering changed; the escape must precede the insertion: %q", got)
	}
}

// Control characters are handled first, so a bidi override cannot hide the very
// construct the later rules are looking for.
func TestSanitizeTextStripsControlsBeforeMatching(t *testing.T) {
	const rtl = "\u202e"
	got := SanitizeText("Closes " + rtl + "#42")
	if strings.Contains(got, rtl) {
		t.Errorf("bidi override survived: %q", got)
	}
	if strings.Contains(got, "#42") {
		t.Errorf("a reference hidden behind a control character stayed live: %q", got)
	}
}

// BreakReferences is shared with the commit-body path, so it must be usable
// without the rest of the comment rules -- a commit message has no @mentions to
// worry about and its own escaping already ran.
func TestBreakReferencesIsIndependentOfTheCommentRules(t *testing.T) {
	got := BreakReferences("Closes #42 @octocat")
	if strings.Contains(got, "#42") {
		t.Errorf("reference not broken: %q", got)
	}
	if !strings.Contains(got, "@octocat") {
		t.Errorf("BreakReferences must not touch mentions: %q", got)
	}
}

// The verdict-to-event mapping is small but it decides whether fixpoint approves
// somebody's pull request, so it is pinned rather than left to a switch nobody
// reads. There is deliberately no event for an inconclusive review: both that
// exist would be lies about a panel that never reached quorum.
func TestEventsAreDistinctAndNamed(t *testing.T) {
	seen := map[Event]bool{}
	for _, e := range []Event{Comment, EventApprove, EventRequestChanges} {
		if e == "" {
			t.Error("an event with an empty name would silently become a comment")
		}
		if seen[e] {
			t.Errorf("duplicate event value %q", e)
		}
		seen[e] = true
	}
}

// Both providers must satisfy Poster, or PosterFor silently returns nil and a
// -post that the operator asked for turns into a warning about an unrecognized
// remote.
func TestBothProvidersCanPost(*testing.T) {
	var _ Poster = githubProvider{}
	var _ Poster = gitlabProvider{}
}

// The filter is what makes inline comments land at all: a forge rejects the whole
// review when one anchor falls outside the diff, and most findings point at code
// the change did not touch.
func TestAddressableLinesReadsTheNewSideOfEveryHunk(t *testing.T) {
	diff := `diff --git a/keep.go b/keep.go
--- a/keep.go
+++ b/keep.go
@@ -10,3 +10,4 @@ func x() {
 context at 10
-removed, old side only
+added at 11
 context at 12
+added at 13
diff --git a/gone.go b/gone.go
--- a/gone.go
+++ /dev/null
@@ -1,2 +0,0 @@
-all gone
-really gone
`
	got := AddressableLines(diff)
	want := map[int]bool{10: true, 11: true, 12: true, 13: true}
	if len(got["keep.go"]) != len(want) {
		t.Fatalf("keep.go lines = %v, want %v", got["keep.go"], want)
	}
	for line := range want {
		if !got["keep.go"][line] {
			t.Errorf("line %d should be addressable: %v", line, got["keep.go"])
		}
	}
	// A line the change removed exists only on the old side; a RIGHT comment cannot
	// name it, and asking for one is what gets the whole review refused.
	if got["keep.go"][14] {
		t.Error("a line past the hunk was reported as addressable")
	}
	if len(got["gone.go"]) != 0 {
		t.Errorf("a deleted file has no side to comment on: %v", got["gone.go"])
	}
}

// Context lines count, not just additions: a finding about a line the change
// merely moved past is still anchorable.
func TestAddressableLinesIncludesContextNotJustAdditions(t *testing.T) {
	got := AddressableLines("+++ b/a.go\n@@ -5,2 +5,2 @@\n unchanged\n unchanged too\n")
	if !got["a.go"][5] || !got["a.go"][6] {
		t.Errorf("context lines should be addressable: %v", got["a.go"])
	}
}

// Junk must not panic or invent anchors -- the input is a diff of code somebody
// else wrote.
func TestAddressableLinesIgnoresMalformedInput(t *testing.T) {
	for _, in := range []string{"", "not a diff at all", "@@ nonsense @@\n line", "+++ b/a.go\n@@ -x,y +z,w @@\n line"} {
		if got := AddressableLines(in); len(got) != 0 {
			t.Errorf("AddressableLines(%q) = %v, want nothing", in, got)
		}
	}
}
