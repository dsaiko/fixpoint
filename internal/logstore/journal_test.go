package logstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dsaiko/fixpoint/internal/model"
)

// readJournal returns the run journal's records, asserting that the file is valid
// JSONL: exactly one well-formed object per line, which is the property that lets a
// killed run's journal still be read to its last complete record.
func readJournal(t *testing.T, dir string) []model.JournalEvent {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(singleRunDir(t, dir), JournalName))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	out := make([]model.JournalEvent, 0, len(lines))
	for i, line := range lines {
		var ev model.JournalEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("journal line %d is not valid JSON (%v): %s", i+1, err, line)
		}
		out = append(out, ev)
	}
	return out
}

// Every record is stamped with the schema version and a gap-free sequence number.
// Seq -- not the timestamp -- is what orders the file: the wall clock can repeat
// within a timestamp interval and can move backwards.
func TestJournalStampsVersionAndSequence(t *testing.T) {
	s, dir := newStore(t, "md")
	for i := range 3 {
		if err := s.Journal(model.EvRoundStarted, i+1, model.JournalRoundStarted{
			Assignments: []string{"review-bugs->mock"},
		}); err != nil {
			t.Fatal(err)
		}
	}
	events := readJournal(t, dir)
	if len(events) != 3 {
		t.Fatalf("got %d records, want 3 (one line per event)", len(events))
	}
	for i, ev := range events {
		if ev.V != model.JournalVersion {
			t.Errorf("record %d: V = %d, want %d", i, ev.V, model.JournalVersion)
		}
		if ev.Seq != i+1 {
			t.Errorf("record %d: Seq = %d, want %d (sequence must be gap-free)", i, ev.Seq, i+1)
		}
		if ev.Round != i+1 {
			t.Errorf("record %d: Round = %d, want %d", i, ev.Round, i+1)
		}
		if ev.At.IsZero() {
			t.Errorf("record %d: At is zero", i)
		}
	}
}

// A run-level event carries no round. Emitting round 0 instead would make "the run"
// indistinguishable from a round numbered zero.
func TestJournalOmitsRoundForRunEvents(t *testing.T) {
	s, dir := newStore(t, "md")
	if err := s.Journal(model.EvRunStarted, 0, model.JournalRunStarted{Config: "task.yaml"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(singleRunDir(t, dir), JournalName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"round"`) {
		t.Errorf("a run-level record must omit round entirely, got: %s", raw)
	}
}

// The journal is written from the parallel reviewer goroutines of a round, so
// concurrent appends must neither interleave within a line nor reuse a sequence
// number -- a torn line would make the whole file unreadable past that point.
func TestJournalConcurrentAppendsStayIntact(t *testing.T) {
	s, dir := newStore(t, "md")
	const n = 40
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Journal(model.EvReviewFinished, i+1, model.JournalReviewFinished{
				Observations: i,
				Errors:       []string{fmt.Sprintf("reviewer %d failed", i)},
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	events := readJournal(t, dir)
	if len(events) != n {
		t.Fatalf("got %d records, want %d", len(events), n)
	}
	seen := map[int]bool{}
	for _, ev := range events {
		if seen[ev.Seq] {
			t.Errorf("sequence number %d was reused", ev.Seq)
		}
		seen[ev.Seq] = true
	}
	for i := 1; i <= n; i++ {
		if !seen[i] {
			t.Errorf("sequence number %d is missing; the sequence must be gap-free", i)
		}
	}
}

// A payload encoding/json cannot serialize writes no record, so it must not spend a
// sequence number either: the next event has to be Seq 1, or the ordering key callers
// are told is gap-free would start with a hole nothing explains.
func TestJournalFailedMarshalKeepsSequence(t *testing.T) {
	s, dir := newStore(t, "md")
	if err := s.Journal(model.EvFixFinished, 1, func() {}); err == nil {
		t.Fatal("Journal accepted an unserializable payload, want an error")
	}
	if err := s.Journal(model.EvFixFinished, 1, model.JournalFixFinished{Agent: "mock"}); err != nil {
		t.Fatal(err)
	}
	events := readJournal(t, dir)
	if len(events) != 1 {
		t.Fatalf("got %d records, want 1", len(events))
	}
	if events[0].Seq != 1 {
		t.Errorf("Seq = %d, want 1 (a failed marshal must not consume a number)", events[0].Seq)
	}
}

// Payloads carry agent-authored text, and a prompt-injected agent can smuggle a
// credential into a reviewer error or a coder failure message. The journal is a
// durable artifact, so it must mask like every other one -- and stay parseable
// afterwards, or redaction would trade a leak for an unreadable audit trail.
func TestJournalRedactsSecrets(t *testing.T) {
	s, dir := newStore(t, "md")
	const secret = "ghp_0123456789abcdefghijklmnopqrstuvwxyz"
	if err := s.Journal(model.EvFixFinished, 1, model.JournalFixFinished{
		Agent: "mock",
		Error: "coder quoted " + secret + " into its failure",
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(singleRunDir(t, dir), JournalName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) {
		t.Errorf("journal leaked the credential-shaped value: %s", raw)
	}
	if !strings.Contains(string(raw), "[REDACTED]") {
		t.Errorf("journal missing the redaction mask: %s", raw)
	}
	readJournal(t, dir) // still valid JSONL after masking
}

// The journal holds the reviewed material's findings and the coder's output, so it
// gets the same owner-only permissions as the rest of the run's artifacts.
func TestJournalIsOwnerOnly(t *testing.T) {
	s, dir := newStore(t, "md")
	if err := s.Journal(model.EvRunStarted, 0, model.JournalRunStarted{Config: "task.yaml"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(singleRunDir(t, dir), JournalName))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("journal mode = %o, want 600", perm)
	}
}

// JournalPath is reported to the operator, so it must name the directory actually
// claimed -- which can carry a collision suffix -- rather than the template.
func TestJournalPathMatchesTheClaimedRunDir(t *testing.T) {
	s, dir := newStore(t, "md")
	if got := s.JournalPath(); !strings.HasSuffix(got, JournalName) {
		t.Errorf("JournalPath() = %q, want it to end in %s", got, JournalName)
	}
	if err := s.Journal(model.EvRunStarted, 0, nil); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(singleRunDir(t, dir), JournalName)
	if got := s.JournalPath(); got != want {
		t.Errorf("JournalPath() = %q, want the claimed run dir's %q", got, want)
	}
}

// The suffix only appears when the rendered directory is already taken, and the
// path is reported at run start -- before any artifact write has claimed one. So
// JournalPath has to do the claiming itself: reporting the bare template here
// would name a directory belonging to the other run, whose journal an operator
// would then read instead of this one's.
func TestJournalPathClaimsTheSuffixedRunDirBeforeAnyWrite(t *testing.T) {
	s, _ := newStore(t, "md")
	// Stand in for a run started within the same timestamp interval.
	taken := s.runDir
	if err := os.MkdirAll(taken, 0o700); err != nil {
		t.Fatal(err)
	}

	got := s.JournalPath()
	if want := filepath.Join(taken+"-2", JournalName); got != want {
		t.Fatalf("JournalPath() = %q, want the collision-suffixed %q", got, want)
	}
	if err := s.Journal(model.EvRunStarted, 0, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(got); err != nil {
		t.Errorf("the journal did not land at the reported path: %v", err)
	}
}

// JournalPath is called from the run's own goroutine while the reviewer goroutines
// are already journaling, so the read of the claimed directory must be ordered
// against the write that claims it -- every caller has to be told the one directory
// the records are actually in. The collision is what makes that write happen at all,
// so it has to be set up here too.
func TestJournalPathIsSafeAlongsideConcurrentAppends(t *testing.T) {
	s, _ := newStore(t, "md")
	taken := s.runDir
	if err := os.MkdirAll(taken, 0o700); err != nil {
		t.Fatal(err)
	}
	const n = 16
	paths := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(2)
		go func() {
			defer wg.Done()
			paths[i] = s.JournalPath()
		}()
		go func() {
			defer wg.Done()
			if err := s.Journal(model.EvRoundStarted, i+1, nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	want := filepath.Join(taken+"-2", JournalName)
	for i, got := range paths {
		if got != want {
			t.Errorf("JournalPath() from goroutine %d = %q, want the single claimed %q", i, got, want)
		}
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("the records did not land at the reported path: %v", err)
	}
}
