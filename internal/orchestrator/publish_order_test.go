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
// the way the orchestrator builds it: the review body (with the finding's
// location), an inline comment, and the thread and triage replies (SanitizeText,
// then the signature, then publishedText).
func publishedPaths(file string) map[string]string {
	const text = publishedPayloadText
	const sig = "_fixpoint_"
	it := model.Issue{Severity: "high", Title: text, File: file, Line: 9, Description: text, Suggestion: text}
	body := review.RenderBody(review.BodyInput{
		Decision:  review.Decision{Outcome: review.ChangesRequested},
		Issues:    []model.Issue{it},
		Signature: sig,
	})
	return map[string]string{
		"body":   publishedText(body),
		"inline": publishedText(review.RenderInline(it, sig)),
		"reply":  publishedText(forge.SanitizeText(text) + "\n\n" + sig),
	}
}

// The live shapes, raw. Every one of them is escaped in agent text wherever it
// stands, so finding one anywhere in the published text -- inside a code span or
// out -- means an escape was skipped.
var livePayload = []string{"@victim", "<details", "![x]"}

const (
	publishedPayload     = "@victim <details> ![x](https://evil.example/leak)"
	publishedPayloadText = "see `token=abcdefgh`x and `" + publishedPayload + "` and " + publishedPayload
)

// No agent string reaches a forge with a live mention, tag or image in it, however
// the forge pairs its code spans. The inside of a span is escaped like prose, so
// that pairing is not something the sanitizer has to predict: an agent span, a
// location span, and a publish-time mask that eats a span's closing backtick all
// publish inert text.
func TestPublishedTextCarriesNoLivePayload(t *testing.T) {
	const payload = publishedPayload

	t.Run("built-in rules", func(t *testing.T) {
		for name, got := range publishedPaths("a`" + payload + ".go") {
			for _, live := range livePayload {
				if strings.Contains(got, live) {
					t.Errorf("%s: published %q carries %q", name, got, live)
				}
			}
			if strings.Contains(got, "abcdefgh") {
				t.Errorf("%s: published %q still carries the secret", name, got)
			}
		}
		// The location keeps its span: a backtick in the path is not a way out of it.
		const loc = "`a&#96;@<!---->victim &lt;details> !<!---->[x](https://evil.example/leak).go:9`"
		if got := publishedPaths("a`" + payload + ".go")["body"]; !strings.Contains(got, loc) {
			t.Errorf("body %q, want the location quoted whole as %q", got, loc)
		}
	})

	// An operator's logs.redact pattern can run on over `[REDACTED]:9` and the
	// location's closing backtick, which leaves the path's text in the open.
	t.Run("operator pattern eats a closing backtick", func(t *testing.T) {
		agent.SetExtraRedactions([]*regexp.Regexp{regexp.MustCompile(`(secret=)\S+`)})
		t.Cleanup(func() { agent.SetExtraRedactions(nil) })
		body := publishedPaths(payload + " secret=abc.go")["body"]
		if strings.Contains(body, "[REDACTED]:9`") {
			t.Fatalf("precondition: the mask must eat the location's closing backtick: %q", body)
		}
		for name, got := range publishedPaths(payload + " secret=abc.go") {
			for _, live := range livePayload {
				if strings.Contains(got, live) {
					t.Errorf("%s: published %q carries %q", name, got, live)
				}
			}
		}
	})
}
