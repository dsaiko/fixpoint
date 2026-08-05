package forge

import (
	"regexp"
	"strings"

	"github.com/dsaiko/fixpoint/internal/model"
)

// SanitizeText prepares agent-authored text for a place a FORGE will render it.
//
// Every finding fixpoint reports was written by a model that had just read code
// somebody else controls, and a review comment is the first artifact where that
// text stops being a local file and becomes an action on a shared platform: it
// notifies people, it cross-references other work, and it can close issues. Those
// are not formatting concerns, they are the difference between publishing a review
// and publishing whatever an injected payload wanted published under fixpoint's
// name.
//
// It is applied when the document is RENDERED, not when it is posted, so the file
// an operator reads before approving a post is byte-for-byte what gets posted. A
// sanitizer that ran only on the way out would make that inspection worthless.
//
// The order matters and is the rule:
//
//  1. Control characters go first (model.StripControl): bidi overrides that render
//     text as something other than what it says, zero-width characters that hide
//     content from a reader while leaving it in the data.
//  2. HTML comment delimiters are escaped BEFORE anything else is inserted. An
//     unescaped `<!--` in agent text would open a comment that swallows the rest of
//     the document -- including the signature -- and step 4 inserts comments of its
//     own, which such a payload could otherwise close.
//  3. Issue references are broken, so a forge's closing-keyword grammar
//     (`Closes #42`, and the full-URL spelling of the same thing) cannot act.
//  4. Mentions are broken, so a review cannot notify arbitrary people.
func SanitizeText(s string) string {
	s = model.StripControl(s)
	s = escapeHTMLComments(s)
	s = BreakReferences(s)
	return breakMentions(s)
}

// escapeHTMLComments makes a literal comment delimiter render as text.
//
// `&lt;` rather than a lookalike character: it is exactly what markdown renders as
// a literal `<`, it cannot begin a comment, and the reader sees the delimiter the
// agent actually wrote instead of a silently altered string.
func escapeHTMLComments(s string) string {
	s = strings.ReplaceAll(s, "<!--", "&lt;!--")
	return strings.ReplaceAll(s, "-->", "--&gt;")
}

// BreakReferences puts a space inside the issue references a forge's
// closing-keyword grammar accepts, so `Closes #42` stops closing #42 while the
// text still records readably which number the agent named.
//
// Two spellings, because the grammar accepts both and an injected string can write
// either without knowing the repository: `#42` (which is also the tail of
// `owner/repo#42`) and `GH-42`; plus the full issue or pull/merge-request URL,
// whose repository slug is public -- the injected text lives in the repository
// under review. Only the `/issues/N`-shaped tail of a URL is touched, so an
// ordinary link survives unless it is itself a closable reference.
//
// Shared with the commit-body path: a commit message is pushed and a forge reads
// closing keywords out of it too, so both artifacts need exactly this rule and
// must not drift.
func BreakReferences(s string) string {
	return forgeIssueRef.ReplaceAllString(forgeIssueURL.ReplaceAllString(s, "$1 $2"), "$1 $2")
}

var (
	forgeIssueRef = regexp.MustCompile(`(?i)(#|\bGH-)(\d)`)
	forgeIssueURL = regexp.MustCompile(`(?i)(//\S+/(?:issues|pull|pulls|merge_requests)/)(\d)`)
	// mention matches what GitHub and GitLab turn into a notification: an @ that
	// begins a word, followed by a name. Requiring a non-word character before the
	// @ keeps an email address in a stack trace from being mangled -- it is not a
	// mention on either forge.
	mention = regexp.MustCompile(`(^|[^\w@/])@([A-Za-z0-9][-\w]*)`)
)

// breakMentions stops a review from notifying people an injected finding named.
//
// `@<!---->name` is the neutralization because it is INVISIBLE: both forges render
// the empty HTML comment as nothing, so a human reads `@name` exactly as written
// while the linkifier no longer sees a mention. Mangling the text instead --
// dropping the @, spacing it out, swapping in a lookalike -- would make fixpoint
// misquote a finding, and a review that misquotes its own evidence is worse than
// one that pings somebody.
//
// A team handle (`@org/team`) notifies a whole group, and it is matched by the
// same rule: the break lands on the @ itself, so everything after it is inert.
func breakMentions(s string) string { return mention.ReplaceAllString(s, "$1@<!---->$2") }
