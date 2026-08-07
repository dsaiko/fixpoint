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
//
// Steps 1-4 hold wherever the text lands, so they live in sanitizeInline and are
// shared with CodeSpan. Two more run only here, because they are about BLOCKS --
// constructs that reach past the string into the document around it, which is
// where fixpoint's own words are:
//
//  5. Raw HTML tags are escaped. `<details>` renders everything after it collapsed,
//     and an unclosed `<?` block hides it outright -- either one takes the
//     signature with it, and the signature's whole security property is that it
//     sits outside every region an agent wrote.
//  6. An unclosed code fence is closed. Without it a finding's description turns
//     the rest of the review -- the findings under it and the attribution at the
//     bottom -- into the inside of a code block.
//
// What is deliberately NOT escaped is inline markup: emphasis, links, headings,
// balanced code spans. Those cannot reach past the string they are in, findings
// use them constantly, and escaping them would make fixpoint misquote its own
// evidence in exchange for nothing a reader could not already have been told in
// plain prose.
func SanitizeText(s string) string {
	s = sanitizeInline(s)
	s = escapeRawHTML(s)
	return closeOpenFence(s)
}

// sanitizeInline is the part of the rule that holds in every context, including
// inside a code span where no block can form. One function rather than two
// call-site copies: the copies are how one of them ends up a rule behind.
func sanitizeInline(s string) string {
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

// escapeRawHTML makes an agent's HTML render as the text it wrote.
//
// Both forges pass a subset of raw HTML through, and the dangerous part of that
// subset is not scripting -- their sanitizers handle that -- it is the tags that
// swallow what FOLLOWS them: `<details>` collapses the remainder of the comment
// behind a disclosure triangle, and a `<?`-opened block that is never closed is
// removed along with everything up to the end of the document. Either one hides
// the signature while leaving a review posted under the operator's identity.
//
// Only the `<` is escaped, and only in the shapes that can begin an HTML block:
// a tag name, a closing tag, a processing instruction, a declaration or CDATA. So
// `x < y` and `a <- b` come through as written, and `<nil>` in a stack trace now
// SURVIVES -- a forge used to drop it as an unknown tag.
//
// It deliberately does not match `<!--`: step 2 escaped every comment the agent
// wrote, so the only one left in the string by the time this runs is the invisible
// break breakMentions inserts, and escaping that would make a broken mention
// visible as `@<!---->name`.
func escapeRawHTML(s string) string {
	return htmlTagOpen.ReplaceAllString(s, "&lt;$1")
}

// closeOpenFence terminates a code fence the agent left open.
//
// Each field is sanitized on its own and then written into a document beside
// fixpoint's own text, so an unterminated ``` does not merely garble one finding:
// every finding after it, and the signature, become the contents of that block.
// Closing it costs three characters and keeps a legitimate code block -- which
// findings do carry -- rendering exactly as the agent wrote it.
//
// It errs towards doing nothing. Only fences CommonMark is unambiguous about are
// tracked (at most three spaces of indent, a backtick opener whose info string
// carries no backtick), because a MISSED fence leaves the status quo while an
// imagined one would append a delimiter that opens a block of its own -- the exact
// harm this is here to prevent.
func closeOpenFence(s string) string {
	var open string
	for _, line := range strings.Split(s, "\n") {
		m := codeFence.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		fence, rest := m[1], m[2]
		if open != "" {
			// A closing fence is the same character, at least as long as the opener, and
			// followed by nothing else: "```go" inside a block is content, not the end.
			if fence[0] == open[0] && len(fence) >= len(open) && strings.TrimSpace(rest) == "" {
				open = ""
			}
			continue
		}
		if fence[0] == '`' && strings.Contains(rest, "`") {
			continue // not an opener: a backtick fence's info string may not contain one
		}
		open = fence
	}
	if open == "" {
		return s
	}
	return strings.TrimSuffix(s, "\n") + "\n" + open + "\n"
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
	// htmlTagOpen matches the openers CommonMark treats as the start of raw HTML,
	// minus the comment (see escapeRawHTML on why): a tag or closing tag, `<?`, and
	// a `<!` declaration or CDATA section.
	htmlTagOpen = regexp.MustCompile(`<(/?[A-Za-z]|\?|![A-Za-z\[])`)
	// codeFence splits a candidate fence line into its delimiter and whatever
	// follows, which is the info string on an opener and must be blank on a closer.
	codeFence = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")
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
//
// It composes sanitizeInline rather than SanitizeText because the two BLOCK rules
// have nothing to do here: no HTML tag and no code fence can form inside a span,
// the backtick escape below already denies the only way out of one, and applying
// them anyway would misquote the path -- `a<b.txt` printed as `a&lt;b.txt`, which
// a code span shows verbatim instead of rendering.
func CodeSpan(s string) string {
	return strings.ReplaceAll(sanitizeInline(s), "`", "&#96;")
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
