package model

import (
	"strings"
	"unicode"
)

// StripControl removes the characters that let untrusted text manipulate whatever
// renders it, while leaving the text readable.
//
// Newlines survive because they are structure a reader needs. Tabs and the
// Unicode line/paragraph separators become ordinary spaces: they are layout, and a
// separator that a renderer treats as a line break can make quoted text appear to
// escape the region it was quoted into. Everything else in the control and format
// categories is dropped -- that is where the bidi overrides live, which can render
// a string as something other than what it says, and the zero-width characters
// that hide content from a reader while leaving it in the data.
//
// It lives in model because more than one layer needs exactly this rule: the
// prompt builder applies it to every agent-authored field it quotes back, and the
// review renderer applies it to the same fields on their way into a document a
// human reads. Two implementations of one security rule is one too many.
func StripControl(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n':
			return r
		case r == '\t', unicode.Is(unicode.Zl, r), unicode.Is(unicode.Zp, r):
			return ' '
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r):
			return -1
		}
		return r
	}, s)
}
