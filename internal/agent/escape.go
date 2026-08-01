package agent

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// EscapeTerminal renders untrusted text as something a terminal only DISPLAYS.
// Collapsing whitespace is not enough: ESC, BEL, the C1 controls, and the Unicode
// bidi/formatting characters all survive strings.Fields, and most of what fixpoint
// prints -- config/agent/prompt names and paths from a bundle inside the
// repository under review, git and agent output, a finding an injected reviewer
// wrote -- is authored by the party being judged.
//
// Text like that is printed BEFORE any trust gate applies (the listing, the source
// provenance lines, the refusal that asks the operator for -trusted-target), so a
// repository must not be able to draw the surface the operator decides on: cursor
// controls can scroll or overwrite the other entries away, a bidi override can make
// a path read as somewhere else, and OSC 52 can write the operator's clipboard.
// Escaping keeps the text visible (and reviewable) instead of silently dropping it.
//
// Newlines are escaped along with the rest. A log line here is one timestamped
// event, and an embedded newline is how untrusted text forges a second one; a
// wrapped subprocess error is still whole, just on one greppable line.
func EscapeTerminal(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			// Invalid encoding: a terminal can resynchronize mid-sequence and render
			// bytes that were never a character, so show the byte itself.
			fmt.Fprintf(&b, "\\x%02x", s[i])
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r):
			// Cc (C0, DEL, C1 -- ESC and friends) plus Cf, which is where the bidi
			// overrides and other invisible formatting controls live.
			if r < 0x100 {
				fmt.Fprintf(&b, "\\x%02x", r)
			} else {
				fmt.Fprintf(&b, "\\u%04x", r)
			}
		default:
			b.WriteRune(r)
		}
		i += size
	}
	return b.String()
}
