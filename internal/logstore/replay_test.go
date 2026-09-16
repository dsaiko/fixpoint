package logstore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

func replayStore(t *testing.T, on bool) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := New(config.Logs{
		Dir:             filepath.Join(dir, "{timestamp}", "round-{round}"),
		Formats:         []string{"md"},
		Pattern:         "{role}-{agent}-{prompt}.{ext}",
		SummaryPattern:  "summary.{ext}",
		TimestampFormat: "20060102-150405",
		Replay:          on,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}

// readRecording returns the recording's records, failing if there is none.
func readRecording(t *testing.T, s *Store) []model.ReplayStep {
	t.Helper()
	b, err := os.ReadFile(s.ReplayPath())
	if err != nil {
		t.Fatal(err)
	}
	var out []model.ReplayStep
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var rec model.ReplayStep
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("recording line %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

func TestReplayRecordsWhatTheOrchestratorConsumed(t *testing.T) {
	s, _ := replayStore(t, true)
	res := agent.Result{
		Stdout:   "<review>{}</review>",
		Stderr:   "warming up",
		Duration: 2 * time.Second,
		Usage:    model.Usage{InputTokens: 10, OutputTokens: 5},
	}
	if err := s.Replay("review", "claude", "bugs", 1, "THE PROMPT", res); err != nil {
		t.Fatal(err)
	}
	recs := readRecording(t, s)
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	got := recs[0]
	if got.V != model.ReplayVersion || got.Seq != 1 {
		t.Errorf("v=%d seq=%d, want v=%d seq=1", got.V, got.Seq, model.ReplayVersion)
	}
	if got.Stdout != res.Stdout || got.Stderr != res.Stderr {
		t.Errorf("streams not recorded verbatim: %+v", got)
	}
	if got.DurationMS != 2000 {
		t.Errorf("DurationMS = %d, want 2000", got.DurationMS)
	}
	if got.Usage.InputTokens != 10 {
		t.Errorf("usage not recorded: %+v", got.Usage)
	}
	// The prompt is recorded as a digest and never in full: it embeds the reviewed
	// material, and a second copy of the diff in this file would be a second place
	// a secret can land.
	if got.PromptSHA256 == "" || strings.Contains(string(mustJSON(t, got)), "THE PROMPT") {
		t.Errorf("the prompt text leaked into the recording: %+v", got)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReplayIsANoOpWhenDisabled(t *testing.T) {
	s, dir := replayStore(t, false)
	if err := s.Replay("review", "claude", "bugs", 1, "P", agent.Result{Stdout: "x"}); err != nil {
		t.Fatal(err)
	}
	if p := s.ReplayPath(); p != "" {
		t.Errorf("ReplayPath() = %q with logs.replay off, want empty", p)
	}
	// And nothing was written anywhere under the logs root -- including the run
	// directory, which Replay must not even claim when it is off.
	var found []string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			found = append(found, p)
		}
		return nil
	})
	if len(found) != 0 {
		t.Errorf("disabled Replay still wrote %v", found)
	}
}

// Sequence numbers are the replay loader's ordering key AND its completeness
// check, so they must be gap-free across every invocation of a run.
func TestReplaySequenceIsGapFree(t *testing.T) {
	s, _ := replayStore(t, true)
	for range 5 {
		if err := s.Replay("review", "claude", "bugs", 1, "P", agent.Result{Stdout: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	for i, rec := range readRecording(t, s) {
		if rec.Seq != i+1 {
			t.Fatalf("record %d has seq %d; the loader reads a gap as a truncated recording", i, rec.Seq)
		}
	}
}

func TestReplayRecordsTheErrorAsText(t *testing.T) {
	s, _ := replayStore(t, true)
	res := agent.Result{Err: errors.New("timed out after 30m"), ProviderStatus: 429}
	if err := s.Replay("review", "claude", "bugs", 2, "P", res); err != nil {
		t.Fatal(err)
	}
	got := readRecording(t, s)[0]
	if got.Err != "timed out after 30m" {
		t.Errorf("Err = %q", got.Err)
	}
	if got.ProviderStatus != 429 {
		t.Errorf("ProviderStatus = %d, want 429", got.ProviderStatus)
	}
}

// The recording carries agent-authored text, so it goes through the same
// redaction pass as every other artifact. This is also why a replayed run is not
// a bit-identical re-run, which the package doc states outright.
func TestReplayRedactsTheReply(t *testing.T) {
	s, _ := replayStore(t, true)
	res := agent.Result{Stdout: "the key is sk-ant-abcdefghijklmnopqrstuvwxyz0123"}
	if err := s.Replay("review", "claude", "bugs", 1, "P", res); err != nil {
		t.Fatal(err)
	}
	got := readRecording(t, s)[0]
	if strings.Contains(got.Stdout, "sk-ant-abcdefghijklmnopqrstuvwxyz0123") {
		t.Errorf("a credential-shaped token survived into the recording: %q", got.Stdout)
	}
	if !strings.Contains(got.Stdout, "[REDACTED]") {
		t.Errorf("the reply was not masked: %q", got.Stdout)
	}
}
