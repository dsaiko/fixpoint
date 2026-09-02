// Package runlog renders a run's progress so its structure is visible.
//
// The information was always there and had to be hunted for: every line was one
// flat timestamped string at the same indent, so "the review ended and the fixes
// began" -- the boundary an operator watching a 40-minute run actually looks for --
// was distinguishable only by reading the words. The five lines belonging to one
// fix (start, done, verify, commit, reply) were five peers with nothing tying them
// together.
//
// Three devices, in the order they earn their place:
//
//  1. A blank line before each block. The cheapest separator there is, and the eye
//     finds it without reading.
//  2. Only block openers and closers at the left margin; their contents indented.
//     Scanning the first column then yields the shape of the run and nothing else.
//  3. A closing line that SUMMARIZES. "REVIEW ended, 4 findings, 1 error, 20m" is
//     the sentence the operator was reconstructing by hand.
//
// Color is applied only when the destination is a terminal, and never changes
// what a line SAYS -- a piped or redirected log is byte-identical to what a reader
// sees, minus the escapes.
package runlog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Log is a run's progress output. The zero value is not usable; call New.
//
// Every write holds one mutex, because reviewer goroutines log concurrently and
// the heartbeat writes from its own. That lock also makes a multi-line block --
// blank line, banner -- atomic against an interleaving detail line, which is what
// makes the shape survive a parallel panel.
type Log struct {
	mu    sync.Mutex
	w     io.Writer
	color bool
	// inBlock indents detail lines under an open block, and is what a closer
	// consumes. Deliberately a bool rather than a depth counter: the run has exactly
	// two levels (a round, and the phases inside it), and a counter would silently
	// absorb an unbalanced Phase/EndPhase pair into ever-deeper indentation instead
	// of showing it.
	inBlock bool
	now     func() time.Time
	// transform is applied to every rendered message. It exists so redaction and
	// terminal escaping stay a property of the SINK rather than of each call site:
	// agent text reaches here through a hundred format strings, and one of them
	// forgetting is how a credential gets printed.
	transform func(string) string
}

// New writes to w, coloring only when w is a terminal that has not asked
// otherwise (NO_COLOR, or TERM=dumb).
func New(w io.Writer) *Log {
	return &Log{w: w, color: colorOK(w), now: time.Now, transform: func(s string) string { return s }}
}

// Transform installs the sanitizer every message passes through before it is
// written. It returns l so a logger can be built in one expression.
func (l *Log) Transform(f func(string) string) *Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	if f != nil {
		l.transform = f
	}
	return l
}

// ANSI codes, kept in one place. Dim is used for lines that are progress rather
// than events -- a heartbeat says nothing happened, and should not compete with
// the line that says something did.
const (
	ansiReset = "\x1b[0m"
	ansiBold  = "\x1b[1m"
	ansiDim   = "\x1b[2m"
)

// Printf writes one detail line, indented if a block is open.
func (l *Log) Printf(format string, args ...any) {
	l.write("", detailIndent, fmt.Sprintf(format, args...))
}

// Rulef opens the outermost block -- a round -- with a full-width separator.
//
// It closes any phase still open, because a round starting is proof the previous
// one ended: an unbalanced EndPhasef somewhere must not indent the rest of the run.
func (l *Log) Rulef(format string, args ...any) {
	title := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.inBlock = false
	l.blank()
	bar := strings.Repeat("━", max(0, ruleWidth-len([]rune(title))-6))
	l.emit(ansiBold, "", "━━━━ "+title+" "+bar)
}

// Phasef opens a block. Everything logged until EndPhasef is indented under it.
func (l *Log) Phasef(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.blank()
	l.inBlock = true
	l.emit(ansiBold, "", "▸ "+fmt.Sprintf(format, args...))
}

// EndPhasef closes the block with its summary, at the margin again.
func (l *Log) EndPhasef(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.inBlock = false
	l.emit(ansiBold, "", "◂ "+fmt.Sprintf(format, args...))
}

// Progressf writes a line that reports nothing new -- an agent still running. Dim
// and prefixed, so it recedes behind the lines that carry events.
func (l *Log) Progressf(format string, args ...any) {
	l.write(ansiDim, detailIndent, "… "+fmt.Sprintf(format, args...))
}

// Raw writes pre-formatted, multi-line output (the end-of-run scoreboard) under
// the same lock, so nothing lands mid-table and shreds the column alignment. The
// caller owns its formatting entirely; not even the timestamp is added.
func (l *Log) Raw(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprint(l.w, s)
}

// Logf adapts this Log to the plain func(string, ...any) that most of the code
// still logs through, so a call site only needs changing where it has structure
// to declare.
func (l *Log) Logf() func(string, ...any) { return l.Printf }

const (
	detailIndent = "    "
	ruleWidth    = 72
)

func (l *Log) write(color, indent, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.inBlock {
		indent = ""
	}
	l.emit(color, indent, msg)
}

// emit renders one line. The caller holds the lock.
//
// A message spanning several lines is indented on every one of them: an agent
// error quoting a stack trace would otherwise break out of its block and leave
// the rest of the trace at the margin, where the shape says "phase boundary".
func (l *Log) emit(color, indent, msg string) {
	msg = l.transform(msg)
	stamp := l.now().Format("15:04:05")
	for i, line := range strings.Split(msg, "\n") {
		if i > 0 {
			stamp = strings.Repeat(" ", len(stamp))
		}
		fmt.Fprintf(l.w, "%s %s%s\n", stamp, indent, l.paint(color, line))
	}
}

func (l *Log) paint(color, s string) string {
	if !l.color || color == "" || s == "" {
		return s
	}
	return color + s + ansiReset
}

// blank separates blocks. The caller holds the lock.
func (l *Log) blank() { _, _ = fmt.Fprintln(l.w) }

// colorOK reports whether escapes should be written: a character device, no
// NO_COLOR, and a terminal that can render them.
//
// os.Stat rather than a terminal library, because the question here is only
// "would these escapes reach a screen" -- a pipe, a file and a CI capture all
// answer no, and that is the whole decision.
func colorOK(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
