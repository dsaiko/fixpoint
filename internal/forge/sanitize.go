package forge

import (
	"regexp"
	"strconv"
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

// CodeSpan renders one agent-authored string inside a markdown code span.
//
// SanitizeText on its own is not enough here, because the DELIMITERS are part of
// the attack surface: a backtick in the string closes the span early, and
// everything after it stops being quoted text and becomes live markdown in a note
// posted under the operator's identity. So the backtick is escaped too.
//
// It becomes its HTML entity rather than being dropped, for the same reason
// breakMentions keeps the name it neutralizes: the string is evidence -- a file
// path a finding is about -- and a review that silently renames the file it points
// at misquotes itself.
//
// Exported because the review body puts a path in a span too (review.mdCode). Two
// hand-rolled escapes for one rule is how one of them ends up without it.
func CodeSpan(s string) string {
	return strings.ReplaceAll(SanitizeText(s), "`", "&#96;")
}

// AddressableLines reports which lines of which files a forge will accept a
// comment on: the new-file side of every hunk in a unified diff.
//
// Without this the feature never lands. A review comment can only anchor inside
// the pull request's own diff, and most findings point at code the change did not
// touch -- the guard three functions up, the caller in another file. GitHub
// rejects the ENTIRE review when one comment falls outside, with a 422 that names
// nothing, so a single such finding silently costs every anchor on the review.
// Measured on this project's own pull request: every inline comment was refused.
//
// Context lines count, not just additions. A finding about a line the change
// merely moved past is still anchorable, and GitHub accepts it as long as the
// line appears in a hunk.
func AddressableLines(diff string) map[string]map[int]bool {
	out := map[string]map[int]bool{}
	var path string
	var newLine int
	inHunk := false
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			// "+++ b/path" -- and "+++ /dev/null" for a deletion, which has no side to
			// comment on.
			path = strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, "+++ ")), "b/")
			if path == "/dev/null" {
				path = ""
			}
			inHunk = false
		case strings.HasPrefix(line, "@@"):
			start, ok := hunkNewStart(line)
			inHunk = ok && path != ""
			newLine = start
		case !inHunk:
			continue
		// Inside a hunk ONLY these three prefixes are content; anything else ends it.
		// Counting by "not a header I recognize" instead let `diff --git`, `index`
		// and `similarity index` lines advance the counter and hand back anchors one
		// past the end of the hunk -- which is exactly the kind of line a forge
		// refuses, taking the whole review with it.
		case strings.HasPrefix(line, "+"), strings.HasPrefix(line, " "):
			if out[path] == nil {
				out[path] = map[int]bool{}
			}
			out[path][newLine] = true
			newLine++
		case strings.HasPrefix(line, "-"):
			// Removed: it exists only on the old side, which RIGHT comments cannot name.
		case strings.HasPrefix(line, "\\"):
			// "\ No newline at end of file" -- a note about the previous line.
		default:
			inHunk = false
		}
	}
	return out
}

// hunkNewStart pulls the new-file start line out of "@@ -a,b +c,d @@".
func hunkNewStart(header string) (int, bool) {
	plus := strings.Index(header, "+")
	if plus < 0 {
		return 0, false
	}
	rest := header[plus+1:]
	end := strings.IndexAny(rest, ", @")
	if end < 0 {
		return 0, false
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0, false
	}
	return n, true
}
