package orchestrator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/review"
)

// publishedPaths is every way a finding's or a reply's text reaches a forge, built
// the way the orchestrator builds it: the review body, an inline comment, and the
// thread and triage replies (SanitizeText, then the signature, then publishedText).
func publishedPaths(text string) map[string][2]string {
	const sig = "_fixpoint_"
	it := model.Issue{Severity: "high", Title: "t", Description: text, Suggestion: text}
	body := review.RenderBody(review.BodyInput{
		Decision:  review.Decision{Outcome: review.ChangesRequested},
		Issues:    []model.Issue{it},
		Signature: sig,
	})
	inline := review.RenderInline(it, sig)
	reply := forge.SanitizeText(text) + "\n\n" + sig
	out := map[string][2]string{}
	for name, rendered := range map[string]string{"body": body, "inline": inline, "reply": reply} {
		out[name] = [2]string{rendered, publishedText(rendered)}
	}
	return out
}

// publishedText redacts the rendered document once more, and a mask there can
// swallow a code span's closing backtick -- re-pairing every span after it, so the
// text the sanitizer left raw as the INSIDE of a span is published as prose. The
// sanitizer therefore redacts first, and on what it rendered the publish-time pass
// must change nothing.
func TestPublishedTextCannotRepairWhatWasSanitized(t *testing.T) {
	const tail = " and `@victim <details> ![x](https://evil.example/leak)` end"

	t.Run("built-in rule", func(t *testing.T) {
		for name, p := range publishedPaths("see `token=abcdefgh`x" + tail) {
			if strings.Contains(p[1], "abcdefgh") {
				t.Errorf("%s: published %q still carries the secret", name, p[1])
			}
			if p[0] != p[1] {
				t.Errorf("%s: publishing changed the sanitized text:\n rendered  %q\n published %q", name, p[0], p[1])
			}
		}
	})

	// An operator's logs.redact pattern CAN take a backtick, whatever the built-in
	// rules do, so it is what shows the pairing is decided after the mask.
	t.Run("operator pattern", func(t *testing.T) {
		agent.SetExtraRedactions([]*regexp.Regexp{regexp.MustCompile(`(ticket=)\S+`)})
		t.Cleanup(func() { agent.SetExtraRedactions(nil) })
		for name, p := range publishedPaths("see `ticket=abc`x" + tail) {
			got := p[1]
			for _, live := range []string{"@victim", "<details", "![x]"} {
				if strings.Contains(got, live) {
					t.Errorf("%s: published %q carries %q as live prose", name, got, live)
				}
			}
			for _, escaped := range []string{"@<!---->victim", "&lt;details>", "!<!---->[x]"} {
				if !strings.Contains(got, escaped) {
					t.Errorf("%s: published %q, want %q", name, got, escaped)
				}
			}
			if p[0] != got {
				t.Errorf("%s: publishing changed the sanitized text:\n rendered  %q\n published %q", name, p[0], got)
			}
		}
	})
}
