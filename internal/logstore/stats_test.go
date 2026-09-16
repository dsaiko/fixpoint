package logstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/model"
)

// writeSummary lays one run summary down under root, in its own run directory,
// the way a real run does.
func writeSummary(t *testing.T, root, name string, sum model.RunSummary) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(sum)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "summary-"+name+".json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// reviewRun builds a one-round review run in which each named agent reported one
// issue with the given verdict.
func reviewRun(start time.Time, agentVerdict map[string]string) model.RunSummary {
	rec := model.RoundRecord{Round: 1}
	i := 0
	for name, verdict := range agentVerdict {
		i++
		id := name + "-issue"
		rec.Findings = append(rec.Findings, model.Finding{ID: id, IssueID: id, Agent: name, Lens: "bugs"})
		rec.Issues = append(rec.Issues, model.Issue{ID: id, Status: verdict, Severity: "high"})
		rec.Steps = append(rec.Steps, model.StepStat{
			Role: "review", Agent: name, Lens: "bugs", DurationMS: 1000,
			Usage: model.Usage{InputTokens: 100, OutputTokens: 20, CacheReadTokens: 50},
		})
		rec.Assignments = append(rec.Assignments, model.Assignment{Agent: name, Lens: "bugs"})
	}
	return model.RunSummary{
		StartedAt:   start,
		FinishedAt:  start.Add(time.Minute),
		Termination: model.TermConverged,
		Rounds:      []model.RoundRecord{rec},
	}
}

func TestLoadStatsAggregatesAcrossRuns(t *testing.T) {
	root := t.TempDir()
	day := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	writeSummary(t, root, "run1", reviewRun(day, map[string]string{
		"claude": model.VerdictFixed,
		"codex":  model.VerdictRejected,
	}))
	writeSummary(t, root, "run2", reviewRun(day.AddDate(0, 0, 1), map[string]string{
		"claude": model.VerdictFixed,
	}))

	s, err := LoadStats(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Runs != 2 {
		t.Fatalf("Runs = %d, want 2", s.Runs)
	}
	byName := map[string]AgentStat{}
	for _, a := range s.Agents {
		byName[a.Name] = a
	}
	claude, codex := byName["claude"], byName["codex"]
	if claude.Runs != 2 || codex.Runs != 1 {
		t.Errorf("Runs: claude=%d codex=%d, want 2 and 1", claude.Runs, codex.Runs)
	}
	// The row an operator reads to decide a seat: volume, and what the coder did
	// with it.
	if claude.Issues != 2 || claude.Fixed != 2 {
		t.Errorf("claude issues=%d fixed=%d, want 2 and 2", claude.Issues, claude.Fixed)
	}
	if codex.Rejected != 1 {
		t.Errorf("codex rejected = %d, want 1", codex.Rejected)
	}
	// Usage accumulates across runs, and the three counters stay SEPARATE so a
	// route that folds cache into input cannot be double-counted by the reader.
	if claude.Usage.InputTokens != 200 || claude.Usage.CacheReadTokens != 100 {
		t.Errorf("claude usage = %+v, want in=200 cache_read=100", claude.Usage)
	}
	// Busiest first, so a seat tried once cannot head the table.
	if s.Agents[0].Name != "claude" {
		t.Errorf("row order = %q first; want the agent with the most runs", s.Agents[0].Name)
	}
}

func TestRejectRate(t *testing.T) {
	a := AgentStat{Issues: 4, Rejected: 3}
	got, ok := a.RejectRate()
	if !ok || got != 0.75 {
		t.Errorf("RejectRate() = %v, %v; want 0.75, true", got, ok)
	}
	// No issues means no rate, not a zero one: an agent that reported nothing has
	// not been vindicated.
	if _, ok := (AgentStat{}).RejectRate(); ok {
		t.Error("RejectRate() reported a rate for an agent with no issues")
	}
}

// Summaries are found by content, so an operator who changed logs.summary_pattern
// still gets an answer. A reader that globbed the default name would report
// nothing, which looks exactly like having run nothing.
func TestLoadStatsFindsSummariesUnderAnyFilename(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "20260901-120000")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(reviewRun(time.Now(), map[string]string{"claude": model.VerdictFixed}))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadStats(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Runs != 1 {
		t.Fatalf("Runs = %d, want 1 -- a summary under a non-default filename was missed", s.Runs)
	}
}

// The per-step .json artifacts live under the same root and unmarshal happily
// into a RunSummary full of zero values. Counting one as a run would inflate
// every denominator in the table.
func TestLoadStatsIgnoresStepArtifacts(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "20260901-120000", "round-1")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	step, err := json.Marshal(model.ReviewOutput{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "review-claude-bugs.json"), step, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadStats(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Runs != 0 {
		t.Fatalf("Runs = %d, want 0; a per-step artifact was counted as a run", s.Runs)
	}
}

func TestRenderStatsNamesTheTokenCaveat(t *testing.T) {
	root := t.TempDir()
	writeSummary(t, root, "run1", reviewRun(time.Now(), map[string]string{"claude": model.VerdictFixed}))
	s, err := LoadStats(root)
	if err != nil {
		t.Fatal(err)
	}
	out := RenderStats(s)
	// No summed token column: the table exists to compare agents, and a figure
	// that means different things per row is what would break that.
	if !strings.Contains(out, "double-count") {
		t.Errorf("the rendering does not warn that in/out/cache must not be summed:\n%s", out)
	}
	if !strings.Contains(out, "claude") {
		t.Errorf("the agent row is missing:\n%s", out)
	}
}

func TestRenderStatsSaysSoWhenThereIsNothing(t *testing.T) {
	s, err := LoadStats(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	out := RenderStats(s)
	if !strings.Contains(out, "no runs found") {
		t.Errorf("an empty root renders no explanation:\n%s", out)
	}
}

// A replayed run's usage is the RECORDED usage, already billed against the run it
// came from. Counting it again inflates exactly the per-agent economics this table
// exists to report (review run 20260916-085129, finding i3).
func TestLoadStatsExcludesReplayedRuns(t *testing.T) {
	root := t.TempDir()
	live := reviewRun(time.Now(), map[string]string{"claude": model.VerdictFixed})
	writeSummary(t, root, "run1", live)
	replayed := reviewRun(time.Now(), map[string]string{"claude": model.VerdictFixed})
	replayed.ReplayedFrom = filepath.Join(root, "run1")
	writeSummary(t, root, "run2", replayed)

	s, err := LoadStats(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Runs != 1 {
		t.Fatalf("Runs = %d, want 1 -- the replay must not be counted", s.Runs)
	}
	if s.Replayed != 1 {
		t.Errorf("Replayed = %d, want 1; a skipped run has to be reported, not silently dropped", s.Replayed)
	}
	if got := s.Agents[0].Usage.InputTokens; got != 100 {
		t.Errorf("input tokens = %d, want 100 (the live run's alone)", got)
	}
	if !strings.Contains(RenderStats(s), "were replays and are excluded") {
		t.Error("the rendering does not say runs were skipped")
	}
}

// The coder's failed invocations reach the errors column. computeRunStats credits
// errors from ReviewErrors, which is reviewer-only, so a coder that died mid-fix
// showed a clean row however often it happened (finding i6).
func TestLoadStatsCountsCoderFailures(t *testing.T) {
	root := t.TempDir()
	sum := reviewRun(time.Now(), map[string]string{"claude": model.VerdictFixed})
	sum.Coder = "claude-coder"
	sum.Rounds[0].Steps = append(sum.Rounds[0].Steps, model.StepStat{
		Role: "fix", Agent: "claude-coder", DurationMS: 5000, Failed: true,
		Usage: model.Usage{InputTokens: 10},
	})
	writeSummary(t, root, "run1", sum)

	s, err := LoadStats(root)
	if err != nil {
		t.Fatal(err)
	}
	var coder AgentStat
	for _, a := range s.Agents {
		if a.Name == "claude-coder" {
			coder = a
		}
	}
	if coder.Name == "" {
		t.Fatalf("the coder is missing from the table: %+v", s.Agents)
	}
	if coder.Errors != 1 {
		t.Errorf("coder errors = %d, want 1 -- the most expensive kind of failure must not read as clean", coder.Errors)
	}
}

// A termination string comes off disk, from a summary a repository can ship, and
// is printed under a table whose columns it could otherwise redraw (finding i11).
func TestRenderStatsEscapesTheTerminationLabel(t *testing.T) {
	root := t.TempDir()
	sum := reviewRun(time.Now(), map[string]string{"claude": model.VerdictFixed})
	sum.Termination = "converged\x1b[2J"
	writeSummary(t, root, "run1", sum)
	s, err := LoadStats(root)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(RenderStats(s), "\x1b") {
		t.Error("an escape sequence from a summary reached the rendered table verbatim")
	}
}
