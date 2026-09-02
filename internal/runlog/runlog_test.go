package runlog

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

// fixed makes the output byte-comparable: the timestamp is the only part of a
// line that is not a function of the call.
func fixed(w *bytes.Buffer) *Log {
	l := New(w)
	l.color = false
	l.now = func() time.Time { return time.Date(2026, 8, 6, 23, 30, 17, 0, time.UTC) }
	return l
}

// The shape is the point: a block's opener and closer sit at the margin, its
// contents are indented, and a blank line separates it from what came before. An
// operator scanning the first column should see the run's structure and nothing
// else.
func TestABlockIsVisibleFromTheLeftMargin(t *testing.T) {
	var buf bytes.Buffer
	l := fixed(&buf)
	l.Rulef("round 3/3")
	l.Phasef("REVIEW  3 reviewers")
	l.Printf("claude done (2 findings)")
	l.Progressf("kimi still running (5m0s)")
	l.EndPhasef("REVIEW  4 findings, 1 error")
	l.Printf("after the block")

	// The rule fills a fixed width, so it reads as one continuous separator
	// whatever the round number is.
	want := "\n" +
		"23:30:17 ━━━━ round 3/3 " + strings.Repeat("━", ruleWidth-len("round 3/3")-6) + "\n" +
		"\n" +
		"23:30:17 ▸ REVIEW  3 reviewers\n" +
		"23:30:17     claude done (2 findings)\n" +
		"23:30:17     … kimi still running (5m0s)\n" +
		"23:30:17 ◂ REVIEW  4 findings, 1 error\n" +
		"23:30:17 after the block\n"
	if got := buf.String(); got != want {
		t.Errorf("output\n%q\nwant\n%q", got, want)
	}
}

// A round starting proves the previous phase ended. Without this an unbalanced
// EndPhase would indent everything after it, and the indent is what the reader
// uses to tell a block's contents from its boundaries.
func TestARuleClosesAnOpenPhase(t *testing.T) {
	var buf bytes.Buffer
	l := fixed(&buf)
	l.Phasef("FIX i19")
	l.Rulef("round 2/3")
	l.Printf("at the margin")
	if strings.Contains(buf.String(), "     at the margin") {
		t.Errorf("a rule must close the open phase:\n%s", buf.String())
	}
}

// An agent error arrives with a stack trace in it. Every line of it stays inside
// the block: a continuation at the margin reads as a phase boundary, which is
// exactly the signal this package exists to make trustworthy.
func TestAMultiLineMessageStaysInsideItsBlock(t *testing.T) {
	var buf bytes.Buffer
	l := fixed(&buf)
	l.Phasef("REVIEW")
	l.Printf("claude failed:\ngoroutine 1:\n  main.go:7")
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")[2:] {
		if !strings.HasPrefix(line, "23:30:17     ") && !strings.HasPrefix(line, "             ") {
			t.Errorf("continuation escaped the block: %q", line)
		}
	}
}

// Redaction and terminal escaping belong to the sink, not to a hundred format
// strings: one call site forgetting is how a credential gets printed.
func TestEveryMessagePassesThroughTheTransform(t *testing.T) {
	var buf bytes.Buffer
	l := fixed(&buf).Transform(func(s string) string { return strings.ReplaceAll(s, "sk-secret", "[REDACTED]") })
	l.Rulef("round with sk-secret")
	l.Phasef("PHASE with sk-secret")
	l.Printf("detail with sk-secret")
	l.Progressf("progress with sk-secret")
	l.EndPhasef("end with sk-secret")
	if strings.Contains(buf.String(), "sk-secret") {
		t.Errorf("a message bypassed the transform:\n%s", buf.String())
	}
	if n := strings.Count(buf.String(), "[REDACTED]"); n != 5 {
		t.Errorf("masked %d of 5 messages:\n%s", n, buf.String())
	}
}

// Color never changes what a line says, so a redirected log and a terminal one
// differ only by escapes.
func TestPipedOutputCarriesNoEscapes(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf) // not a character device
	l.now = func() time.Time { return time.Time{} }
	l.Phasef("REVIEW")
	l.Progressf("waiting")
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("escapes reached a non-terminal writer: %q", buf.String())
	}
}

// The panel logs from one goroutine per reviewer while the heartbeat logs from
// its own. Under -race this is what proves the lock covers the whole line rather
// than each Fprintf.
func TestConcurrentWritesDoNotInterleave(t *testing.T) {
	var buf bytes.Buffer
	l := fixed(&buf)
	l.Phasef("REVIEW")
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			l.Printf("reviewer %d reporting a finding", i)
			l.Progressf("reviewer %d still running", i)
		}(i)
	}
	wg.Wait()
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "23:30:17 ") {
			t.Errorf("line was shredded by a concurrent write: %q", line)
		}
	}
}
