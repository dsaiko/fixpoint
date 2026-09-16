package replay

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/model"
)

// rec builds one recording line. The prompt is hashed by the helper so a test
// that cares about the digest can pass the real prompt text and one that does not
// can pass anything.
func rec(seq int, role, agentName, promptName string, prompt, stdout string) model.ReplayStep {
	return model.ReplayStep{
		V:            model.ReplayVersion,
		Seq:          seq,
		At:           time.Now(),
		Role:         role,
		Agent:        agentName,
		Prompt:       promptName,
		Round:        1,
		PromptSHA256: digest(prompt),
		Stdout:       stdout,
		DurationMS:   1234,
	}
}

func digest(s string) string {
	// Deliberately recomputed here rather than exported from the package under
	// test: a test that shares the subject's own hashing helper passes even if that
	// helper hashes the wrong thing.
	h := newSHA()
	h.Write([]byte(s))
	return h.hex()
}

// write lays a recording down in a fresh run directory and returns its path.
func write(t *testing.T, steps ...model.ReplayStep) string {
	t.Helper()
	dir := t.TempDir()
	var b strings.Builder
	for _, s := range steps {
		line, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, model.ReplayName), []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestServeReturnsTheRecordedReply(t *testing.T) {
	dir := write(t, rec(1, "review", "claude", "bugs", "PROMPT", "<review>{}</review>"))
	src, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := src.Steps(); got != 1 {
		t.Fatalf("Steps() = %d, want 1", got)
	}
	res, err := src.Serve("review", "claude", "bugs", 1, "PROMPT")
	if err != nil {
		t.Fatal(err)
	}
	if res.Stdout != "<review>{}</review>" {
		t.Errorf("Stdout = %q", res.Stdout)
	}
	// The recorded duration is restored rather than zeroed: the scoreboard bills a
	// replayed run with the numbers the live one produced, which is what makes a
	// replayed summary comparable to the summary it came from.
	if res.Duration != 1234*time.Millisecond {
		t.Errorf("Duration = %s, want 1.234s", res.Duration)
	}
	if d := src.Divergence(); d.Any() {
		t.Errorf("a fully consumed recording diverged: %+v", d)
	}
}

// Two replies under ONE key is the reviewer-reformat-retry shape. Serving the
// same reply twice would make a retry that fixed a contract violation look like a
// reviewer that never fixed it.
func TestServeOrdersRepeatedKeyBySeq(t *testing.T) {
	dir := write(t,
		rec(2, "review", "claude", "bugs", "P", "second"),
		rec(1, "review", "claude", "bugs", "P", "first"),
	)
	src, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"first", "second"} {
		res, err := src.Serve("review", "claude", "bugs", 1, "P")
		if err != nil {
			t.Fatal(err)
		}
		if res.Stdout != want {
			t.Errorf("Stdout = %q, want %q", res.Stdout, want)
		}
	}
	// A third ask is the replayed run taking more turns than the recorded one.
	if _, err := src.Serve("review", "claude", "bugs", 1, "P"); err == nil {
		t.Fatal("a third Serve on a two-record key succeeded; it must refuse rather than repeat a reply")
	}
}

func TestServeRefusesAnInvocationTheRecordingDoesNotHave(t *testing.T) {
	dir := write(t, rec(1, "review", "claude", "bugs", "P", "x"))
	src, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	// An empty review would read downstream as a CLEAN one, so the refusal here is
	// what keeps a diverged replay from converging on a reviewer that never ran.
	_, err = src.Serve("review", "codex", "bugs", 1, "P")
	if err == nil {
		t.Fatal("Serve invented a reply for an agent the recording never saw")
	}
	if !strings.Contains(err.Error(), "codex") {
		t.Errorf("the error does not name the missing invocation: %v", err)
	}
}

func TestServeRefusesRolesWhoseSideEffectsWereNotRecorded(t *testing.T) {
	dir := write(t, rec(1, "fix", "claude", "fix", "P", "<fix>{}</fix>"))
	src, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = src.Serve("fix", "claude", "fix", 1, "P")
	if !errors.Is(err, ErrNotReviewOnly) {
		t.Fatalf("replaying a fix round was permitted; the coder's edits are not in any recording. err = %v", err)
	}
}

// A prompt that no longer hashes to the recorded one means the reply answers a
// question this build does not ask. It must still be served -- changing a prompt
// is one of the things a replay exists to test -- but it must be reported.
func TestPromptChangeIsServedAndReported(t *testing.T) {
	dir := write(t, rec(1, "review", "claude", "bugs", "OLD PROMPT", "x"))
	src, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Serve("review", "claude", "bugs", 1, "NEW PROMPT"); err != nil {
		t.Fatal(err)
	}
	d := src.Divergence()
	if len(d.PromptChanged) != 1 {
		t.Fatalf("PromptChanged = %v, want one entry", d.PromptChanged)
	}
	if !strings.Contains(d.PromptChanged[0], "bugs") {
		t.Errorf("the report does not name the invocation: %v", d.PromptChanged)
	}
}

func TestDivergenceReportsWhatWasNeverAskedFor(t *testing.T) {
	dir := write(t,
		rec(1, "review", "claude", "bugs", "P", "x"),
		rec(2, "review", "codex", "bugs", "P", "y"),
	)
	src, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Serve("review", "claude", "bugs", 1, "P"); err != nil {
		t.Fatal(err)
	}
	d := src.Divergence()
	if len(d.Unserved) != 1 || !strings.Contains(d.Unserved[0], "codex") {
		t.Fatalf("Unserved = %v, want the codex invocation", d.Unserved)
	}
}

func TestLoadRefusesAnIncompleteRecording(t *testing.T) {
	// Seq 2 missing: the run that wrote this was killed, or the file was truncated.
	// Replaying it would report the reviewer at seq 2 as having answered nothing.
	dir := write(t,
		rec(1, "review", "claude", "bugs", "P", "x"),
		rec(3, "review", "codex", "bugs", "P", "y"),
	)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("a recording with a sequence gap loaded; it must be refused")
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Errorf("the error does not say the recording is incomplete: %v", err)
	}
}

func TestLoadRefusesAnUnknownSchemaVersion(t *testing.T) {
	s := rec(1, "review", "claude", "bugs", "P", "x")
	s.V = model.ReplayVersion + 1
	_, err := Load(write(t, s))
	if err == nil {
		t.Fatal("a recording from a newer schema loaded; a half-understood replay looks real and is not")
	}
}

func TestLoadExplainsAMissingRecording(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Fatal("Load succeeded on a directory with no recording")
	}
	// The actionable half: the run was made with logs.replay off.
	if !strings.Contains(err.Error(), "replay") {
		t.Errorf("the error does not say how to get a recording: %v", err)
	}
}

func TestLoadRefusesAnEmptyRecording(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, model.ReplayName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("an empty recording loaded; there is nothing to replay")
	}
}

// A reviewer's reply can be megabytes, well past bufio.Scanner's default 64 KB
// token. Without the enlarged buffer Load fails on exactly the runs most worth
// replaying.
func TestLoadReadsALongReply(t *testing.T) {
	long := strings.Repeat("x", 512*1024)
	src, err := Load(write(t, rec(1, "review", "claude", "bugs", "P", long)))
	if err != nil {
		t.Fatal(err)
	}
	res, err := src.Serve("review", "claude", "bugs", 1, "P")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Stdout) != len(long) {
		t.Errorf("reply truncated: got %d bytes, want %d", len(res.Stdout), len(long))
	}
}

// REGRESSION (review run 20260916-085129, finding i4). A key the recording has no
// records for never appears in `queued`, so it could not show up in the unserved
// tally -- and a replay that consumed everything it DID have while being refused
// something else reported "matched the recording exactly" over a run that plainly
// diverged.
func TestDivergenceReportsARefusedInvocation(t *testing.T) {
	dir := write(t, rec(1, "review", "claude", "bugs", "P", "x"))
	src, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Consume the whole recording, so Unserved is empty...
	if _, err := src.Serve("review", "claude", "bugs", 1, "P"); err != nil {
		t.Fatal(err)
	}
	// ...then ask for something it never held.
	if _, err := src.Serve("review", "codex", "bugs", 1, "P"); err == nil {
		t.Fatal("precondition: the second Serve should have been refused")
	}
	d := src.Divergence()
	if !d.Any() {
		t.Fatal("a replay that was refused an invocation reported a clean match")
	}
	if len(d.Refused) != 1 || !strings.Contains(d.Refused[0], "codex") {
		t.Errorf("Refused = %v, want the codex invocation", d.Refused)
	}
}
