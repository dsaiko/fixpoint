package review

import (
	"slices"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/issue"
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

// The location is the one agent-authored string the body renders inside a code
// span, and the span's delimiter is part of the attack surface: a backtick in a
// reported path would close it early and turn the rest of the line into live
// markdown in a review posted under the operator's identity.
func TestBodyEscapesBackticksInTheReportedPath(t *testing.T) {
	body := bodyFor(t, Input{
		Issues: []model.Issue{{
			ID: "i1", Severity: "high", Line: 7,
			File:  "pkg/x.go`<img src=x onerror=alert(1)>`",
			Title: "real finding",
		}},
		Quorum: full(2),
	}, nil)
	var loc string
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "**HIGH**") {
			loc = line
		}
	}
	if loc == "" {
		t.Fatalf("the finding lost its location line:\n%s", body)
	}
	// Exactly the two delimiters the renderer wrote: any more and the path escaped.
	if n := strings.Count(loc, "`"); n != 2 {
		t.Errorf("the path broke out of its code span (%d backticks in %q):\n%s", n, loc, body)
	}
	// Escaped, not dropped: the review still quotes the path the finding named, and
	// the line it named survives on the end of it.
	if !strings.Contains(loc, "pkg/x.go&#96;") || !strings.HasSuffix(loc, ":7`") {
		t.Errorf("the reported location was lost from %q:\n%s", loc, body)
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

// A verdict reason reads like fixpoint's own words, but it interpolates strings
// the reviewed repository controls: the names of its failing and pending checks,
// and the agent names in the quorum note. So it goes through the same funnel as
// agent prose -- a check named @victim must not notify a stranger, and one named
// `Closes #42` must not act on an issue, under the operator's identity.
func TestVerdictReasonsAreSanitizedForForgeSyntax(t *testing.T) {
	body := bodyFor(t, Input{
		Quorum: Quorum{Panel: 2, Present: 1, Required: 2, Missing: []string{"@victim"}},
		CI:     CI{Known: true, Failing: []string{"build <!-- hide --> Closes #42"}},
	}, nil)
	for _, live := range []string{"@victim", "<!-- hide", "hide -->", "#42"} {
		if strings.Contains(body, live) {
			t.Errorf("a reason kept live forge syntax %q:\n%s", live, body)
		}
	}
	// Neutralized, not deleted: the reader still sees which check failed and who
	// did not report.
	for _, kept := range []string{"victim", "&lt;!--", "--&gt;", "# 42"} {
		if !strings.Contains(body, kept) {
			t.Errorf("a reason lost %q:\n%s", kept, body)
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
	// Last of everything a reader SEES. What follows is the identity markers, which
	// are HTML comments a forge renders as nothing -- the same order an inline
	// comment uses, where the marker also trails the signature.
	if !strings.HasSuffix(strings.TrimSpace(withoutMarkers(body)), "-- real signature") {
		t.Errorf("the real signature must be last:\n%s", body)
	}
}

// withoutMarkers drops the trailing identity markers, so a test can assert on what
// the reader is shown.
func withoutMarkers(body string) string {
	var kept []string
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "<!-- ai-panel") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
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

// publishedAt is the ordinary case these tests are about: the identities were
// published about the commit now under review, so nothing has moved since and each
// of them still describes the code a reader is looking at. The cases where that is
// NOT true have their own tests below.
func publishedAt(ids ...string) Published {
	at := make(map[string][]string, len(ids))
	for _, id := range ids {
		at[id] = []string{"c0ffeec0ffee"}
	}
	return Published{At: at, Moved: map[string]map[string]bool{"c0ffeec0ffee": {}}}
}

// Two reviews of one commit is a legitimate thing to want -- a second panel sees
// what the first missed -- but repeating what is already posted is not. And
// because the panel is not deterministic the repeat would not even read as a copy:
// it overlaps, differs in wording, and a reader cannot tell it is one finding
// described twice.
//
// What is left out is COUNTED. A body showing three findings where an earlier one
// showed thirty, with nothing saying the difference is history, reads as a project
// that has just been cleaned up.
func TestABodyOmitsWhatThePullRequestAlreadyCarriesAndSaysHowMuch(t *testing.T) {
	old := model.Issue{ID: "i1", Severity: "high", Title: "already said", File: "a.go", Line: 1, Fingerprint: "a.go#L1"}
	fresh := model.Issue{ID: "i2", Severity: "high", Title: "new this time", File: "b.go", Line: 2, Fingerprint: "b.go#L2"}
	in := BodyInput{
		Decision:         Decision{Outcome: ChangesRequested, Reasons: []string{"1 unresolved finding"}},
		Issues:           []model.Issue{old, fresh},
		AlreadyPublished: publishedAt(FindingID(old)),
	}
	got := RenderBody(in)

	if strings.Contains(got, "already said") {
		t.Errorf("a finding already on the pull request was repeated:\n%s", got)
	}
	if !strings.Contains(got, "new this time") {
		t.Errorf("a new finding must still be published:\n%s", got)
	}
	if !strings.Contains(got, "1 further finding(s) are already reported") {
		t.Errorf("the omission must be counted, or a short list reads as a clean bill of health:\n%s", got)
	}
}

// Everything already said, nothing new: the body says exactly that rather than
// "No findings", which would claim the opposite of the truth.
func TestABodyWithNothingNewSaysSo(t *testing.T) {
	it := model.Issue{ID: "i1", Severity: "medium", Title: "known", File: "a.go", Line: 1, Fingerprint: "a.go#L1"}
	got := RenderBody(BodyInput{
		Decision:         Decision{Outcome: ChangesRequested},
		Issues:           []model.Issue{it},
		AlreadyPublished: publishedAt(FindingID(it)),
	})
	if strings.Contains(got, "No findings.") {
		t.Errorf("a pull request with an outstanding finding must not be told there are none:\n%s", got)
	}
	if !strings.Contains(got, "not already reported") {
		t.Errorf("the body should say the findings are known, not absent:\n%s", got)
	}
}

// A finding is recognized on the next review only if the pull request carries its
// identity, and for most findings the body is the only place that can. An anchor
// has to fall inside the pull request's own diff and most findings point at code
// the change did not touch; a finding with no location at all can never anchor,
// and when a forge refuses the anchors the whole review goes out as a body. Marked
// only where they anchored, those came back verbatim in every later review.
//
// Read back through forge.PublishedFindings, the same reader the next run uses, so
// the two cannot drift.
func TestTheBodyCarriesTheIdentityOfEveryFindingItPublishes(t *testing.T) {
	anchored := model.Issue{ID: "i1", Severity: "high", Title: "in the diff", File: "a.go", Line: 1, Fingerprint: "a.go#L1"}
	unanchored := model.Issue{ID: "i2", Severity: "medium", Title: "outside the diff", File: "old.go", Line: 400, Fingerprint: "old.go#L400"}
	placeless := model.Issue{ID: "i3", Severity: "low", Title: "nowhere in particular"}
	note := model.Finding{Severity: "low", Title: "an advisory note", Description: "worth knowing."}
	body := RenderBody(BodyInput{
		Decision:     Decision{Outcome: ChangesRequested, Reasons: []string{"3 unresolved findings"}},
		Issues:       []model.Issue{anchored, unanchored, placeless},
		Advisory:     []model.Finding{note},
		Signature:    "-- AI panel",
		RunID:        "20260808-120000",
		ReviewedHead: "0123456789abcdef0123456789abcdef01234567",
	})

	published := forge.PublishedFindings(nil, []forge.Review{{Author: "me", Body: body}}, "me")
	for _, it := range []model.Issue{anchored, unanchored, placeless} {
		// Against the commit the panel read, not merely present: an identity read back
		// without a revision cannot say whether it describes the code that is there now,
		// and this reader is what the next run's delta is built from.
		if !slices.Contains(published[FindingID(it)], "0123456789abcdef0123456789abcdef01234567") {
			t.Errorf("%q was published in the body but the next review cannot recognize it against the reviewed commit (%v):\n%s", it.Title, published[FindingID(it)], body)
		}
	}
	if !slices.Contains(published[AdvisoryID(note)], "0123456789abcdef0123456789abcdef01234567") {
		t.Errorf("an advisory note has no identity, so it is reprinted by every later review:\n%s", body)
	}
	// A marker is bookkeeping, not a second copy of the review: nothing it adds is
	// visible to a reader.
	if strings.Contains(withoutMarkers(body), "ai-panel run") {
		t.Errorf("marker text leaked into the visible review:\n%s", body)
	}
	// And a stranger's review carrying the same body proves nothing: the marker is
	// copyable, the authoring account is not.
	if n := len(forge.PublishedFindings(nil, []forge.Review{{Author: "stranger", Body: body}}, "me")); n != 0 {
		t.Errorf("a copied review body suppressed %d finding(s) from every future review", n)
	}
}

// An advisory note is published in the body and nowhere else, so it is the part of
// a review most exposed to being said again. It is counted apart from the findings
// because it gates nothing: the findings' line tells the reader the verdict
// accounts for what it omitted, which of an advisory note would be untrue.
func TestARepeatedAdvisoryNoteIsOmittedAndCountedOnItsOwn(t *testing.T) {
	said := model.Finding{Severity: "low", Title: "already noted", Description: "old news."}
	fresh := model.Finding{Severity: "low", Title: "newly noted", Description: "new news."}
	got := RenderBody(BodyInput{
		Decision:         Decision{Outcome: Approve},
		Advisory:         []model.Finding{said, fresh},
		AlreadyPublished: publishedAt(AdvisoryID(said)),
	})
	if strings.Contains(got, "already noted") {
		t.Errorf("an advisory note already on the pull request was repeated:\n%s", got)
	}
	if !strings.Contains(got, "newly noted") {
		t.Errorf("a new advisory note must still be published:\n%s", got)
	}
	if !strings.Contains(got, "1 further advisory note(s) are already reported") {
		t.Errorf("the omission must be counted, or the section reads as complete:\n%s", got)
	}
	if strings.Contains(got, "further finding(s)") {
		t.Errorf("an advisory note must not be counted as a finding the verdict accounts for:\n%s", got)
	}
}

// A pull request lives across pushes, and an identity is a path, a line and a
// normalized title -- all three of which a later push can restore over DIFFERENT
// code. An author fixes a high-severity authentication finding, pushes, and a
// defect of the same kind lands at that same place later in the pull request's
// life: recognized on the identity alone, the second review withheld the current
// description -- the exploit, the reproduction, the suggestion -- and printed a
// count saying it was already reported, while the thread the reader is left to
// find may be resolved, outdated, or about code that is gone.
func TestAFindingIsPublishedInFullWhenItsFileMovedSinceItWasSaid(t *testing.T) {
	reintroduced := model.Issue{
		ID: "i1", Severity: "high", Title: "authentication bypass",
		Description: "the session check is skipped for a signed cookie.",
		File:        "auth.go", Line: 118, Fingerprint: "auth.go#L118",
	}
	untouched := model.Issue{
		ID: "i2", Severity: "high", Title: "unchecked error",
		Description: "the write error is dropped.",
		File:        "quiet.go", Line: 4, Fingerprint: "quiet.go#L4",
	}
	// Both were reported on an earlier head; only auth.go has been pushed to since.
	said := Published{
		At: map[string][]string{
			FindingID(reintroduced): {"0ldc0mm1t"},
			FindingID(untouched):    {"0ldc0mm1t"},
		},
		Moved: map[string]map[string]bool{"0ldc0mm1t": {"auth.go": true}},
	}

	got := RenderBody(BodyInput{
		Decision:         Decision{Outcome: ChangesRequested, Reasons: []string{"2 unresolved findings"}},
		Issues:           []model.Issue{reintroduced, untouched},
		AlreadyPublished: said,
	})
	if !strings.Contains(got, "the session check is skipped") {
		t.Errorf("a finding on code pushed since it was reported was withheld as already said:\n%s", got)
	}
	if strings.Contains(got, "the write error is dropped") {
		t.Errorf("a finding on code nobody has touched was repeated:\n%s", got)
	}
	if !strings.Contains(got, "1 further finding(s) are already reported") {
		t.Errorf("exactly the untouched finding should be counted as already reported:\n%s", got)
	}
}

// Carries is the whole of "may this finding be withheld?", so each way of
// answering it no is worth pinning: a commit nothing can be compared against, and
// a finding with no file to compare.
func TestOnlyAFindingAboutUnmovedCodeCountsAsAlreadyCarried(t *testing.T) {
	located := model.Issue{Title: "a defect", File: "a.go", Line: 7, Fingerprint: "a.go#L7"}
	placeless := model.Issue{Title: "the change needs a test"}
	for _, tc := range []struct {
		name string
		it   model.Issue
		p    Published
		want bool
	}{
		{
			name: "nothing moved since it was said",
			it:   located,
			p:    Published{At: map[string][]string{FindingID(located): {"h1"}}, Moved: map[string]map[string]bool{"h1": {}}},
			want: true,
		},
		{
			name: "another file moved, not this one",
			it:   located,
			p:    Published{At: map[string][]string{FindingID(located): {"h1"}}, Moved: map[string]map[string]bool{"h1": {"b.go": true}}},
			want: true,
		},
		{
			name: "its own file moved",
			it:   located,
			p:    Published{At: map[string][]string{FindingID(located): {"h1"}}, Moved: map[string]map[string]bool{"h1": {"a.go": true}}},
			want: false,
		},
		{
			// Force-pushed away, never fetched, git unavailable: nothing is known about what
			// has moved, so nothing is withheld on the strength of it.
			name: "the commit it was said about cannot be diffed",
			it:   located,
			p:    Published{At: map[string][]string{FindingID(located): {"h1"}}, Moved: map[string]map[string]bool{}},
			want: false,
		},
		{
			// Said twice; the later head still vouches for the code, so it is still said.
			name: "one of its commits still vouches for the code",
			it:   located,
			p: Published{
				At:    map[string][]string{FindingID(located): {"h1", "h2"}},
				Moved: map[string]map[string]bool{"h1": {"a.go": true}, "h2": {}},
			},
			want: true,
		},
		{
			// No file, so no file's stillness can vouch for it: it is a statement about the
			// change, and the change has moved.
			name: "a finding with no location, on a pull request pushed to since",
			it:   placeless,
			p:    Published{At: map[string][]string{FindingID(placeless): {"h1"}}, Moved: map[string]map[string]bool{"h1": {"b.go": true}}},
			want: false,
		},
		{
			name: "a finding with no location, on a pull request nothing has moved in",
			it:   placeless,
			p:    Published{At: map[string][]string{FindingID(placeless): {"h1"}}, Moved: map[string]map[string]bool{"h1": {}}},
			want: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.Carries(FindingID(tc.it), tc.it.File); got != tc.want {
				t.Errorf("Carries() = %v, want %v", got, tc.want)
			}
		})
	}
}

// The identity has to survive rewording, or a second run reports the same defect
// again just because two models described it differently -- and it has to survive
// only rewording, or a second run withholds a defect nobody has ever seen because
// an unrelated one was already reported on that line.
func TestFindingIDIsStableAcrossWordingAndUnstableAcrossPlaces(t *testing.T) {
	a := model.Issue{Title: "nil deref on the config pointer", File: "a.go", Line: 7, Fingerprint: "a.go#L7"}
	b := model.Issue{Title: "Config pointer, nil deref!", File: "a.go", Line: 7, Fingerprint: "a.go#L7"}
	c := model.Issue{Title: "nil deref on the config pointer", File: "a.go", Line: 8, Fingerprint: "a.go#L8"}
	if FindingID(a) != FindingID(b) {
		t.Error("two wordings of one finding must share an identity")
	}
	if FindingID(a) == FindingID(c) {
		t.Error("two places must not share an identity")
	}
	if id := FindingID(a); len(id) != 32 || strings.ContainsAny(id, "->< ") {
		t.Errorf("identity %q must be safe inside an HTML comment and wide enough to be worth colliding", id)
	}
}

// The identity decides whether a finding is withheld, and its pre-image is the
// path, line and title of code the pull request's author wrote. A width that can
// be searched offline lets a decoy be built whose identity equals a real finding's,
// which suppresses that finding under a line claiming the verdict accounted for it.
// 16 bytes of SHA-256, pinned here because the cheap thing to write is 6.
func TestTheIdentityIsTooWideToCollideOnPurpose(t *testing.T) {
	id := FindingID(model.Issue{Title: "a defect", File: "a.go", Line: 7, Fingerprint: "a.go#L7"})
	if len(id) != 32 {
		t.Errorf("identity %q is %d hex chars; a suppression key needs 32 (128 bits)", id, len(id))
	}
	note := AdvisoryID(model.Finding{Title: "a note", File: "a.go", Line: 7})
	if len(note) != 32 {
		t.Errorf("advisory identity %q is %d hex chars; it suppresses too", note, len(note))
	}
}

// Advisory notes and blocking findings are looked up in ONE AlreadyPublished set,
// so their identities must not collide. Advisory-ness belongs to the lens that
// found the defect, not to the defect, and the panel is nondeterministic: the same
// defect can be a note one run and a blocking finding the next. Sharing the key
// meant that run's finding was dropped and counted under a line saying the verdict
// accounted for it, while the pull request carried only a note explicitly
// disclaiming the verdict -- the finding's text, severity and suggestion nowhere
// at all.
func TestAnAdvisoryNoteDoesNotSuppressABlockingFindingOnTheSameDefect(t *testing.T) {
	note := model.Finding{Severity: "high", Title: "secret logged in the request handler", File: "a.go", Line: 7}
	same := model.Issue{
		ID: "i1", Severity: "high", Title: "secret logged in the request handler",
		Description: "the bearer token reaches the access log.",
		File:        "a.go", Line: 7, Fingerprint: issue.Fingerprint(note),
	}
	if AdvisoryID(note) == FindingID(same) {
		t.Fatal("a note and a blocking finding of one defect share an identity, so the note withholds the finding")
	}
	got := RenderBody(BodyInput{
		Decision:         Decision{Outcome: ChangesRequested, Reasons: []string{"1 unresolved finding"}},
		Issues:           []model.Issue{same},
		AlreadyPublished: publishedAt(AdvisoryID(note)),
	})
	if !strings.Contains(got, "the bearer token reaches the access log") {
		t.Errorf("a blocking finding was withheld because a previous run reported the defect as an advisory note:\n%s", got)
	}
	if strings.Contains(got, "further finding(s) are already reported") {
		t.Errorf("the finding was counted as already said when nothing on the pull request says it:\n%s", got)
	}
}

// The title half of the identity is a NORMALIZED title, and normalization can
// throw everything away: an ASCII-only tokenizer reduced every non-Latin script to
// nothing, and a title of only filler words reduces to nothing in any script. Two
// such titles on one line then hash alike, so the second defect is withheld from
// the pull request and counted as already reported -- the same suppression the
// location-only identity caused, on the titles a lexical rule cannot key.
func TestTitlesThatNormalizationCannotTellApartStillGetDistinctIdentities(t *testing.T) {
	for _, tc := range []struct{ name, a, b string }{
		{"non-latin script", "空指针解引用", "忽略错误返回值"},
		{"filler words only", "It is not the one", "may be that this can be"},
		{"punctuation only", "!!!", "???"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := model.Issue{Title: tc.a, File: "a.go", Line: 7, Fingerprint: "a.go#L7"}
			b := model.Issue{Title: tc.b, File: "a.go", Line: 7, Fingerprint: "a.go#L7"}
			if FindingID(a) == FindingID(b) {
				t.Errorf("%q and %q share an identity, so one suppresses the other", tc.a, tc.b)
			}
		})
	}
}

// One statement routinely holds two defects -- the nil deref and the unchecked
// error it came from -- which is why the ledger refuses to call a shared location
// identity on its own. The published identity has to refuse it too: a second panel
// finding something NEW on a line the first already commented on is the case this
// whole feature is run for, and suppressing it would be reported as "already
// reported", which is the opposite of the truth.
func TestADifferentDefectOnAnAlreadyReportedLineIsStillPublished(t *testing.T) {
	said := model.Issue{ID: "i1", Severity: "high", Title: "nil deref on the config pointer", File: "a.go", Line: 7, Fingerprint: "a.go#L7"}
	other := model.Issue{ID: "i2", Severity: "high", Title: "error return ignored", File: "a.go", Line: 7, Fingerprint: "a.go#L7"}
	if FindingID(said) == FindingID(other) {
		t.Fatal("two defects on one line must not share an identity")
	}
	got := RenderBody(BodyInput{
		Decision:         Decision{Outcome: ChangesRequested, Reasons: []string{"1 unresolved finding"}},
		Issues:           []model.Issue{said, other},
		AlreadyPublished: publishedAt(FindingID(said)),
	})
	if strings.Contains(got, "nil deref") {
		t.Errorf("the finding already on the pull request was repeated:\n%s", got)
	}
	if !strings.Contains(got, "error return ignored") {
		t.Errorf("a defect nobody has reported must reach the body, not the omitted count:\n%s", got)
	}
}
