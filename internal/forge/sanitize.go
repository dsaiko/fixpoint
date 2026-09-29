package forge

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/dsaiko/fixpoint/internal/agent"
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
//     content from a reader while leaving it in the data. Secrets are masked
//     (agent.RedactSecrets, the operator's logs.redact included) right after, and
//     BEFORE the code spans are paired: a mask can swallow a closing backtick, and
//     the pairing that decides which text is left raw must be the one published.
//  2. HTML comment delimiters are escaped BEFORE anything else is inserted. An
//     unescaped `<!--` in agent text would open a comment that swallows the rest of
//     the document -- including the signature -- and step 4 inserts comments of its
//     own, which such a payload could otherwise close.
//  3. Issue references are broken, so a forge's closing-keyword grammar
//     (`Closes #42`, and the full-URL spelling of the same thing) cannot act.
//  4. Mentions are broken, so a review cannot notify arbitrary people.
//
// Steps 1-4 are the rule for prose and live in sanitizeInline, but for the
// redaction, which runs once over the whole string before it. Inside a code span
// -- here and in CodeSpan -- only steps 1 and 2 run (see below). Three more run
// only here, because they are about BLOCKS --
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
//  7. Image syntax is broken back into a link (breakImages). Not because it
//     reaches past the string -- it does not -- but because it is the one inline
//     construct that acts with no reader: rendering it FETCHES a URL the agent
//     chose.
//
// What is deliberately NOT escaped is the rest of inline markup: emphasis, links,
// headings, balanced code spans. Those cannot reach past the string they are in
// and do nothing until a human clicks them, findings use them constantly, and
// escaping them would make fixpoint misquote its own evidence in exchange for
// nothing a reader could not already have been told in plain prose.
//
// Nor is the INSIDE of an inline code span, beyond steps 1 and 2 (see codeSpans):
// no reference or mention is broken there, which is safe ONLY while the span is
// certain to pair the same way on the forge.
// A forge renders a span literally -- it decodes no entity, hides no comment,
// links no mention -- so `List<String>` escaped came out as `List&lt;String>` and
// `@Override` as `@<!---->Override`: a misquote with nothing bought. Step 2 still
// runs there because the markers a later review reads back are matched in the RAW
// body (see PublishedFindings), where a span is no protection. Fenced blocks are
// NOT exempted, and that is not an oversight: the signature joins its lines after
// sanitizing, which turns a fence's content back into inline text, and a fence's
// extent depends on list and quote containers the string cannot see.
//
// A caller that puts two agent strings in ONE paragraph must sanitize them in one
// call. Each call's span pairing holds only for the text it saw, and an unpaired
// backtick at the end of one string pairs, on the forge, with the first one in the
// next -- turning the rest of that span into live prose.
func SanitizeText(s string) string {
	s = agent.RedactSecrets(model.StripControl(s))
	spans, ok := codeSpans(s)
	if !ok {
		return sanitizeSpans(s, nil) // the pairing is not certain, so all of it is treated as prose
	}
	out := sanitizeSpans(s, spans)
	if agent.RedactSecrets(out) != out {
		// Every published path redacts once more (publishedText, logstore), and here
		// that pass would CHANGE text whose span content was left raw: an escape
		// inside a span (`-->` to `--&gt;`) can grow a value past a rule's minimum, and
		// a mask that runs on to a closing backtick re-pairs everything after it. With
		// the spans no longer certain, none of them is exempted.
		return sanitizeSpans(s, nil)
	}
	return out
}

// sanitizeSpans applies the prose rules everywhere but the given code spans,
// whose content keeps only the comment escape.
func sanitizeSpans(s string, spans [][2]int) string {
	var b strings.Builder
	at := 0
	for _, sp := range spans {
		b.WriteString(sanitizeProse(s[at:sp[0]]))
		b.WriteString(escapeHTMLComments(s[sp[0]:sp[1]]))
		at = sp[1]
	}
	b.WriteString(sanitizeProse(s[at:]))
	return closeOpenFence(b.String())
}

// sanitizeProse is every per-string rule for text outside a code span.
func sanitizeProse(s string) string {
	s = sanitizeInline(s)
	s = escapeRawHTML(s)
	return breakImages(s)
}

// codeSpans finds the inline code spans of s, backticks included, and reports
// whether a forge is CERTAIN to pair them the same way.
//
// The pairing is CommonMark's: a run of N backticks opens, the next run of exactly
// N closes, an opener with no closer is literal, and a backslash escapes a
// backtick outside a span. That is not enough on its own, because exempting a span
// is only safe where the forge agrees it IS one -- text taken for code that the
// forge renders as prose is an unescaped mention or tag. So any shape where the
// two could disagree gives up on the whole string (ok false), which is the old
// escape-everything behavior:
//
//   - a span over a newline: the forge pairs within a paragraph, and a blank line,
//     a heading or a list item between the two runs ends it;
//   - a span holding a `|`: a table row is split into cells before spans are
//     paired;
//   - an opener whose unbroken run of prose holds `<`, `:`, `@` or `www.`: an
//     autolink starting there takes the backticks into its URL (an email autolink
//     may carry one in its local part), and those are the characters that start
//     one;
//   - an opener with `](`, `$` or `[[` anywhere in the prose before it: a link
//     destination or title, GitLab's `$`...`$` and `$...$` math and a wikilink
//     each read raw text up to their own closer, backticks included, before any
//     span there can form.
func codeSpans(s string) ([][2]int, bool) {
	var spans [][2]int
	prev := 0    // where the prose before the next opener begins
	raw := false // whether that prose so far could open a construct that reads raw text
	for i := 0; i < len(s); {
		if s[i] == '\\' && i+1 < len(s) && isASCIIPunct(s[i+1]) {
			i += 2
			continue
		}
		if s[i] != '`' {
			i++
			continue
		}
		open := backtickRunEnd(s, i)
		n, closer := open-i, -1
		for k := open; k < len(s); {
			if s[k] != '`' {
				k++
				continue
			}
			e := backtickRunEnd(s, k)
			if e-k == n {
				closer = k
				break
			}
			k = e
		}
		if closer < 0 {
			i = open // no closer: these backticks are literal text
			continue
		}
		prose := s[prev:i]
		raw = raw || strings.Contains(prose, "](") || strings.Contains(prose, "$") || strings.Contains(prose, "[[")
		if raw || strings.ContainsAny(s[open:closer], "\n|") || autolinkCanStart(prose) {
			return nil, false
		}
		spans = append(spans, [2]int{i, closer + n})
		prev = closer + n
		i = prev
	}
	return spans, true
}

// autolinkCanStart reports whether the prose right before a backtick could begin
// an autolink that runs over it: an autolink stops only at whitespace, so only the
// unbroken tail of the prose matters.
func autolinkCanStart(prose string) bool {
	tail := prose[strings.LastIndexAny(prose, " \n")+1:]
	return strings.ContainsAny(tail, "<:@") || strings.Contains(strings.ToLower(tail), "www.")
}

func backtickRunEnd(s string, i int) int {
	for i < len(s) && s[i] == '`' {
		i++
	}
	return i
}

// isASCIIPunct is the set CommonMark lets a backslash escape.
func isASCIIPunct(c byte) bool {
	return strings.IndexByte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", c) >= 0
}

// sanitizeInline is steps 1-4, the part of the rule that holds in all prose. It
// does NOT run inside a code span: there only steps 1 and 2 do (SanitizeText's
// span branch, CodeSpan), and leaving mentions and references unbroken is safe
// only because the forge is certain to render that text as a span. Text that is
// not guaranteed to be one needs all of this. One function rather than call-site
// copies: the copies are how one of them ends up a rule behind.
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

// breakImages turns an agent-authored image back into an ordinary link.
//
// An image is the only inline markup that acts without a reader. A link to
// `https://attacker.example/<data>` sits there until somebody clicks it; the same
// URL written as `![x](...)` is FETCHED the moment the markdown is rendered. So a
// finding whose text an injected pull request wrote can carry a host secret --
// encoded past the redactor's heuristics -- out to a server of its choosing, both
// when the review is posted and when the operator opens review-body.md to decide
// whether to post it. That is a request nobody made, which is what separates it
// from every other construct this function leaves alone.
//
// The break is the empty comment breakMentions uses, for the same reason: it
// renders as nothing, so the reader still sees `![x](url)` as the agent wrote it,
// while the `!` no longer sits against the `[` that would make it an image. What is
// left is a link, with the destination visible and a click required.
//
// Inserting rather than backslash-escaping the `!` is deliberate. `\![x](url)` is
// already an inert link, and prefixing another backslash would give `\\![x](url)`
// -- a literal backslash followed by a live image, so a payload could turn the
// defense into the attack by writing the first backslash itself.
//
// It runs here and not in sanitizeInline because no image can form inside a code
// span, where the comment would render verbatim and misquote the path.
func breakImages(s string) string { return strings.ReplaceAll(s, "![", "!<!---->[") }

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
// Inside the span only the rules that hold THERE run, for the reason SanitizeText
// exempts a span's content: no tag, image, mention or reference can form inside
// one, and escaping them would misquote the path -- `@types/foo.d.ts` printed as
// `@<!---->types/foo.d.ts`, `a<b.txt` as `a&lt;b.txt`. The comment escape stays,
// because a marker is read out of the raw body, span or no span, and so does the
// redaction, first: a mask applied later could run on over the closing delimiter.
//
// A line break becomes a space, which is what a span renders one as anyway. Kept,
// it is a way out: a blank line ends the paragraph and the span with it, and what
// follows -- a `<details>` or a fence -- is live in the document around it.
func CodeSpan(s string) string {
	s = agent.RedactSecrets(strings.ReplaceAll(model.StripControl(s), "\n", " "))
	return strings.ReplaceAll(escapeHTMLComments(s), "`", "&#96;")
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
//
// The path is read the way git wrote it, not the way its defaults write it. The
// diff comes from the operator's own git, whose config can change the prefixes
// (diff.noprefix, diff.mnemonicPrefix's `w/`, diff.dstPrefix) and quotes any
// non-ASCII name by default (core.quotePath). Trimming a literal "b/" stored those
// files under a key no finding ever names, and their anchors were dropped without
// a word. See newSidePath.
func AddressableLines(diff string) map[string]map[int]bool {
	out := map[string]map[int]bool{}
	var path, header, renamed string
	var newLine int
	inHunk := false
	for _, line := range strings.Split(diff, "\n") {
		switch {
		// INSIDE a hunk, content wins over every header pattern. A file header can only
		// appear between hunks, and testing for one first misread an added line whose
		// own text begins "++ " -- which renders in a unified diff as "+++ ..." -- as
		// the start of a new file. Everything after it was then recorded under a path
		// taken from that line's content, so the anchor map named files the change
		// never touched and lost the ones it did. The easy trigger is not exotic: any
		// pull request that adds a .patch or .diff fixture contains such lines.
		case inHunk && (strings.HasPrefix(line, "+") || strings.HasPrefix(line, " ")):
			if out[path] == nil {
				out[path] = map[int]bool{}
			}
			out[path][newLine] = true
			newLine++
		case inHunk && strings.HasPrefix(line, "-"):
			// Removed: it exists only on the old side, which RIGHT comments cannot name.
		case inHunk && strings.HasPrefix(line, "\\"):
			// "\ No newline at end of file" -- a note about the previous line.
		case strings.HasPrefix(line, "diff --git "):
			header, renamed, path, inHunk = strings.TrimPrefix(line, "diff --git "), "", "", false
		case !inHunk && (strings.HasPrefix(line, "rename to ") || strings.HasPrefix(line, "copy to ")):
			// The one place git names the new path with no prefix at all.
			_, to, _ := strings.Cut(line, " to ")
			renamed = unquoteGitPath(to)
		case strings.HasPrefix(line, "+++ "):
			// "+++ b/path" -- and "+++ /dev/null" for a deletion, which has no side to
			// comment on.
			path = newSidePath(strings.TrimPrefix(line, "+++ "), header, renamed)
			header, renamed, inHunk = "", "", false
		case strings.HasPrefix(line, "@@"):
			start, ok := hunkNewStart(line)
			inHunk = ok && path != ""
			newLine = start
		case !inHunk:
			continue
		default:
			// Inside a hunk only the three prefixes above are content, and anything else
			// ends it. Counting by "not a header I recognize" instead let `diff --git`,
			// `index` and `similarity index` lines advance the counter and hand back
			// anchors one past the end of the hunk -- exactly the kind of line a forge
			// refuses, taking the whole review with it.
			inHunk = false
		}
	}
	return out
}

// newSidePath is the repository path a "+++ " line names, whatever prefix the
// operator's git gave it.
//
// No prefix can be stripped by name, because none is fixed: `b/` by default,
// nothing under diff.noprefix, `w/` or `i/` under diff.mnemonicPrefix, anything at
// all under diff.dstPrefix. What IS fixed is that the `diff --git` header names
// the same file on both sides, each under its own prefix, so the path is what the
// two sides share at the end, from a directory boundary. A rename names two files,
// and git spells the new one out on its `rename to` line instead -- unprefixed.
// Only with neither (a bare patch, no header) is `b/` assumed.
func newSidePath(dst, header, renamed string) string {
	// git ends the name with a tab when it holds a space, so a patch tool can tell
	// where it stops.
	dst = strings.TrimRight(dst, "\t\r")
	switch {
	case dst == "/dev/null":
		return ""
	case renamed != "":
		return renamed
	}
	name := unquoteGitPath(dst)
	if src, ok := strings.CutSuffix(header, " "+dst); ok {
		if p := sharedPathTail(unquoteGitPath(src), name); p != "" {
			return p
		}
	}
	return strings.TrimPrefix(name, "b/")
}

// sharedPathTail is the longest common suffix of a and b that starts a path
// component in both: "x/y.go" for "a/x/y.go" and "b/x/y.go", all of "x.go" for
// two unprefixed "x.go".
func sharedPathTail(a, b string) string {
	n := 0
	for n < len(a) && n < len(b) && a[len(a)-1-n] == b[len(b)-1-n] {
		n++
	}
	tail := b[len(b)-n:]
	for tail != "" {
		i, j := len(a)-len(tail), len(b)-len(tail)
		if (i == 0 || a[i-1] == '/') && (j == 0 || b[j-1] == '/') {
			return tail
		}
		_, rest, ok := strings.Cut(tail, "/")
		if !ok {
			return ""
		}
		tail = rest
	}
	return ""
}

// unquoteGitPath undoes core.quotePath: a name git put in double quotes carries C
// escapes, octal bytes for anything outside ASCII among them, which are exactly
// Go's own. Anything else is returned as it came.
func unquoteGitPath(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
	}
	return s
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
