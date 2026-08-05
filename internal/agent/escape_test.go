package agent

import "testing"

// EscapeTerminal is the single-line form: everything a terminal would act on --
// including the newline that would forge a second log event -- becomes visible
// text rather than behavior.
func TestEscapeTerminal(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain text", "plain text"},
		{"osc\x1b]52;c;ZXZpbA==\x07", "osc\\x1b]52;c;ZXZpbA==\\x07"},
		{"two\nlines\tapart", "two\\x0alines\\x09apart"},
		{"bidi\u202eflip", "bidi\\u202eflip"},
		{"invalid\xffbyte", "invalid\\xffbyte"},
	}
	for _, tc := range cases {
		if got := EscapeTerminal(tc.in); got != tc.want {
			t.Errorf("EscapeTerminal(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// EscapeTerminalBlock is the document form: same neutralization, but the line
// break and the tab stay themselves so a persisted artifact keeps its shape.
func TestEscapeTerminalBlock(t *testing.T) {
	cases := []struct{ in, want string }{
		{"para one\n\npara two", "para one\n\npara two"},
		{"func f() {\n\treturn\n}", "func f() {\n\treturn\n}"},
		{"osc\x1b]52;c;ZXZpbA==\x07", "osc\\x1b]52;c;ZXZpbA==\\x07"},
		{"crlf\r\n", "crlf\\x0d\n"},
		{"bidi\u202eflip", "bidi\\u202eflip"},
	}
	for _, tc := range cases {
		if got := EscapeTerminalBlock(tc.in); got != tc.want {
			t.Errorf("EscapeTerminalBlock(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// Escaping is idempotent: the summary md re-escapes text the run table already
	// escaped, and that must not double up the backslashes.
	once := EscapeTerminalBlock("osc\x1b]52")
	if twice := EscapeTerminalBlock(once); twice != once {
		t.Errorf("not idempotent: %q then %q", once, twice)
	}
}
