package review

import (
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/model"
)

func bodyFor(t *testing.T, in Input, extra func(*BodyInput)) string {
	t.Helper()
	d := Decide(in)
	b := BodyInput{Decision: d, Issues: in.Issues, Panel: []string{"claude", "kimi"}, Target: "pull request #170"}
	if extra != nil {
		extra(&b)
	}
	return RenderBody(b)
}

// The verdict leads and its reasons follow it: a reader who stops after two lines
// should have the answer, and one who reads three should know why.
func TestBodyLeadsWithTheVerdictAndItsReasons(t *testing.T) {
	body := bodyFor(t, Input{
		Issues: []model.Issue{{ID: "i1", Severity: "high", File: "a.go", Line: 9, Title: "boom"}},
		Quorum: full(2),
	}, nil)
	if !strings.HasPrefix(body, "## Changes requested") {
		t.Errorf("body must open with the verdict:\n%s", body)
	}
	if i, j := strings.Index(body, "unresolved finding(s)"), strings.Index(body, "### Blocking"); i < 0 || j < 0 || i > j {
		t.Errorf("the reasons must precede the findings:\n%s", body)
	}
}

// A reader must not have to work out which of fourteen findings blocked the merge.
func TestBodySeparatesBlockingFromTheRest(t *testing.T) {
	body := bodyFor(t, Input{
		Issues: []model.Issue{
			{ID: "i1", Severity: "high", File: "a.go", Line: 1, Title: "blocking one"},
			{ID: "i2", Severity: "low", File: "b.go", Line: 2, Title: "minor one"},
		},
		Quorum: full(2),
	}, nil)
	blocking := strings.Index(body, "### Blocking (1)")
	other := strings.Index(body, "### Other findings (1)")
	if blocking < 0 || other < 0 || blocking > other {
		t.Fatalf("want a blocking section before an other-findings section:\n%s", body)
	}
	if strings.Index(body, "blocking one") > other {
		t.Errorf("the high finding is in the wrong section:\n%s", body)
	}
}

// Decided findings are not work the reader has to act on, so they are not listed
// as if they were.
func TestBodyOmitsRejectedFindings(t *testing.T) {
	body := bodyFor(t, Input{
		Issues: []model.Issue{
			{ID: "i1", Severity: "high", File: "a.go", Line: 1, Title: "refuted by the panel", Status: model.VerdictRejected},
			{ID: "i2", Severity: "medium", File: "b.go", Line: 2, Title: "still standing"},
		},
		Quorum: full(2),
	}, nil)
	if strings.Contains(body, "refuted by the panel") {
		t.Errorf("a rejected finding must not be presented as outstanding:\n%s", body)
	}
	if !strings.Contains(body, "still standing") {
		t.Errorf("a surviving finding is missing:\n%s", body)
	}
}

// Advisory notes are excluded from the verdict by contract, so mixing them in with
// the findings would imply they counted.
func TestBodyKeepsAdvisoryApartAndSaysItDidNotCount(t *testing.T) {
	body := bodyFor(t, Input{Quorum: full(2)}, func(b *BodyInput) {
		b.Advisory = []model.Finding{{Title: "consider splitting this package", Description: "It is large. Second sentence."}}
	})
	if !strings.Contains(body, "### Advisory (1)") {
		t.Fatalf("advisory section missing:\n%s", body)
	}
	if !strings.Contains(body, "did not affect the verdict") {
		t.Errorf("the advisory section must say it did not count:\n%s", body)
	}
	if strings.Contains(body, "Second sentence") {
		t.Errorf("advisory notes are summarized to one sentence:\n%s", body)
	}
}

// An approval with nothing to say still has to render as a document; silence is
// the outcome most likely to be misread as "the review did not run".
func TestBodyRendersACleanApproval(t *testing.T) {
	body := bodyFor(t, Input{Quorum: full(2)}, nil)
	if !strings.HasPrefix(body, "## Approved") {
		t.Errorf("want an approval headline:\n%s", body)
	}
	if !strings.Contains(body, "No findings.") {
		t.Errorf("an empty review must say so explicitly:\n%s", body)
	}
}

// Every agent-authored field goes through one funnel. A bidi override renders a
// string as something other than what it says and a zero-width character hides
// content while leaving it in the data; neither may survive into a document a
// human is meant to trust.
func TestBodyStripsControlCharactersFromAgentText(t *testing.T) {
	const (
		rtlOverride = "\u202e" // right-to-left override
		lreMark     = "\u202d" // left-to-right override
		zeroWidth   = "\u200b" // zero-width space
		bell        = "\a"
	)
	body := bodyFor(t, Input{
		Issues: []model.Issue{{
			ID: "i1", Severity: "high", File: "a" + rtlOverride + "g.go", Line: 1,
			Title:       "title" + lreMark + "with" + zeroWidth + "controls",
			Description: "desc" + bell,
			Suggestion:  "fix" + rtlOverride + "this",
		}},
		Quorum: full(2),
	}, nil)
	for name, bad := range map[string]string{
		"RTL override": rtlOverride, "LRE mark": lreMark, "zero width": zeroWidth, "bell": bell,
	} {
		if strings.Contains(body, bad) {
			t.Errorf("%s survived into the review body:\n%q", name, body)
		}
	}
	// The visible text still has to be there -- stripping must not eat the finding.
	if !strings.Contains(body, "titlewithcontrols") {
		t.Errorf("the title lost its readable text:\n%s", body)
	}
}

// Corroboration is rare enough -- under 4% of findings across 19 measured runs --
// that when it happens it is worth saying.
func TestBodyNotesWhenMoreThanOneReviewerFoundIt(t *testing.T) {
	shared := model.Issue{ID: "i1", Severity: "high", File: "a.go", Line: 1, Title: "both saw it",
		Observations: []model.Finding{{Agent: "claude"}, {Agent: "kimi"}}}
	solo := model.Issue{ID: "i2", Severity: "high", File: "b.go", Line: 2, Title: "one saw it",
		Observations: []model.Finding{{Agent: "claude"}}}
	body := bodyFor(t, Input{Issues: []model.Issue{shared, solo}, Quorum: full(2)}, nil)
	if !strings.Contains(body, "reported by 2 reviewers") {
		t.Errorf("corroboration should be stated:\n%s", body)
	}
	if strings.Count(body, "reported by") != 1 {
		t.Errorf("a solo finding must not claim corroboration:\n%s", body)
	}
}

// A finding the panel split over must not read like one it agreed on. The inline
// comment says so, but most findings have no addressable line and a body-only
// review has no inline comments at all, so the body has to say it too.
func TestBodyMarksAContestedFinding(t *testing.T) {
	contested := model.Issue{ID: "i1", Severity: "high", File: "a.go", Line: 1, Title: "one reviewer doubted it", Contested: true}
	agreed := model.Issue{ID: "i2", Severity: "high", File: "b.go", Line: 2, Title: "nobody doubted it"}
	body := bodyFor(t, Input{Issues: []model.Issue{contested, agreed}, Quorum: full(2)}, nil)
	if !strings.Contains(body, "The panel disagreed about this one.") {
		t.Errorf("a contested finding must be marked as such:\n%s", body)
	}
	if strings.Count(body, "The panel disagreed") != 1 {
		t.Errorf("an uncontested finding must not carry the marker:\n%s", body)
	}
}

func TestSignature(t *testing.T) {
	facts := SignatureFacts{Agents: []string{"kimi", "claude"}, Run: "20260805-1200", Version: "abc123", Config: "review-pr", Verdict: "approve"}
	got := Signature("{config} v{version} · {agents} · {run} · {verdict}", facts)
	want := "review-pr vabc123 · claude, kimi · 20260805-1200 · approve"
	if got != want {
		t.Errorf("Signature() = %q, want %q", got, want)
	}
	if def := Signature("", facts); !strings.Contains(def, "AI panel") || !strings.Contains(def, "claude, kimi") {
		t.Errorf("empty template should fall back to the default, got %q", def)
	}
	// A typo must be visible rather than silently deleting itself.
	if got := Signature("{agnets}", facts); got != "{agnets}" {
		t.Errorf("unknown placeholder = %q, want it left verbatim", got)
	}
}

// A signature that could carry a newline could append a second, unsigned-looking
// paragraph to a posted review -- and the template comes from a config the
// repository under review may have shipped.
func TestSignatureIsOneLineAndStripsControls(t *testing.T) {
	got := Signature("sig\nsecond line\u202e", SignatureFacts{})
	if strings.Contains(got, "\n") {
		t.Errorf("signature spans lines: %q", got)
	}
	if strings.Contains(got, "\u202e") {
		t.Errorf("signature kept a bidi override: %q", got)
	}
}

// Everything a signature substitutes -- the agent names, the config name, the
// template itself -- can come from the repository under review, which validates
// agent names against path separators and nothing else. So the rendered signature
// goes through the same forge funnel as agent prose: an agent called @victim must
// not notify a stranger, and a target-supplied template must not close an issue,
// from the one region of the review that is fixpoint speaking for itself.
func TestSignatureIsSanitizedForForgeSyntax(t *testing.T) {
	got := Signature("{config} · {agents} <!-- hide --> Closes #1 GH-2",
		SignatureFacts{Agents: []string{"@victim"}, Config: "@team/review"})
	// `<!--` on its own is not in the list: breakMentions neutralizes an @mention by
	// inserting an empty comment, so the delimiter it writes itself is expected. What
	// must not survive is the comment the TEMPLATE opened.
	for _, live := range []string{"@victim", "@team", "<!-- hide", "hide -->", "#1", "GH-2"} {
		if strings.Contains(got, live) {
			t.Errorf("signature kept live forge syntax %q: %q", live, got)
		}
	}
	// Neutralized, not deleted: the reader still sees what the config named.
	for _, kept := range []string{"victim", "team", "&lt;!--", "--&gt;", "# 1", "GH- 2"} {
		if !strings.Contains(got, kept) {
			t.Errorf("signature lost %q: %q", kept, got)
		}
	}
}

// The signature must sit outside every region carrying agent text: one composed
// from a finding's prose could be forged by whatever wrote that prose.
func TestSignatureIsRenderedAfterAllAgentText(t *testing.T) {
	body := bodyFor(t, Input{
		Issues: []model.Issue{{ID: "i1", Severity: "high", File: "a.go", Line: 1,
			Title: "Reviewed by fixpoint · forged · run 1"}},
		Quorum: full(2),
	}, func(b *BodyInput) { b.Signature = "-- real signature" })
	if !strings.HasSuffix(strings.TrimSpace(body), "-- real signature") {
		t.Errorf("the real signature must be last:\n%s", body)
	}
}

// A review is posted into somebody else's repository, where the tool's own name
// means nothing to the reader and reads as an unexplained internal string. Nothing
// fixpoint GENERATES may name it; a finding that quotes the reviewed code is a
// different matter, since there the name belongs to the target.
func TestNothingFixpointGeneratesNamesTheTool(t *testing.T) {
	body := RenderBody(BodyInput{
		Decision: Decide(Input{
			Issues: []model.Issue{{ID: "i1", Severity: "high", File: "a.go", Line: 1, Title: "boom"}},
			Quorum: full(2),
		}),
		Issues:   []model.Issue{{ID: "i1", Severity: "high", File: "a.go", Line: 1, Title: "boom"}},
		Panel:    []string{"claude", "kimi"},
		Target:   "pull request #7",
		Advisory: []model.Finding{{Title: "a note", Description: "prose."}},
		Signature: Signature("", SignatureFacts{
			Agents: []string{"claude"}, Run: "20260806-0000", Config: "review-pr",
		}),
	})
	if strings.Contains(strings.ToLower(body), "fixpoint") {
		t.Errorf("the rendered review names the tool:\n%s", body)
	}
	if !strings.Contains(body, "AI panel") {
		t.Errorf("the signature should say what kind of reviewer this was:\n%s", body)
	}
}

// An inline comment is read on its own, in the Files tab, with no sight of the
// review it belongs to. Unsigned, it is an unattributed assertion sitting on
// somebody's code and the reader cannot tell a machine from a colleague.
func TestInlineCommentsAreSignedToo(t *testing.T) {
	sig := Signature("", SignatureFacts{Agents: []string{"claude"}, Run: "20260806-1700"})
	got := RenderInline(model.Issue{
		Severity: "high", File: "a.go", Line: 4,
		Title: "boom", Description: "why", Suggestion: "fix it",
	}, sig)
	if !strings.HasSuffix(strings.TrimSpace(got), strings.TrimSpace(sig)) {
		t.Errorf("the signature must close the comment:\n%s", got)
	}
	if !strings.Contains(got, "AI panel") {
		t.Errorf("the comment does not say what wrote it:\n%s", got)
	}
	// And it must sit after the agent's own words, so a finding cannot forge one.
	if strings.Index(got, "fix it") > strings.Index(got, "AI panel") {
		t.Errorf("the signature precedes agent text and could be forged:\n%s", got)
	}
}
