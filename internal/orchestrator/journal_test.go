package orchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/logstore"
	"github.com/dsaiko/fixpoint/internal/model"
)

// journal returns the run journal's records. It reads the file the run actually
// wrote rather than an in-memory copy, because durability on disk is the feature.
func (f *fixture) journal() []model.JournalEvent {
	f.t.Helper()
	base := f.cfg.Logs.StaticBase()
	runs, err := os.ReadDir(base)
	if err != nil {
		f.t.Fatalf("no run directory under %s: %v", base, err)
	}
	if len(runs) != 1 {
		f.t.Fatalf("expected one run dir under %s, got %d", base, len(runs))
	}
	b, err := os.ReadFile(filepath.Join(base, runs[0].Name(), logstore.JournalName))
	if err != nil {
		f.t.Fatalf("read journal: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	out := make([]model.JournalEvent, 0, len(lines))
	for i, line := range lines {
		var ev model.JournalEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			f.t.Fatalf("journal line %d is not valid JSON (%v): %s", i+1, err, line)
		}
		out = append(out, ev)
	}
	return out
}

// discarded decodes the run's round_discarded record and asserts its reason. Each
// abnormal exit leaves a stash the operator did not ask for, and the journal is
// the only place "which stash is this and why" is answerable -- so every discard
// site's own behavior test checks its record here, rather than one central test
// re-staging four different failures.
func (f *fixture) discarded(reason string) model.JournalRoundDiscarded {
	f.t.Helper()
	var d model.JournalRoundDiscarded
	payload(f.t, f.journal(), model.EvRoundDiscarded, &d)
	if d.Reason != reason {
		f.t.Errorf("round_discarded reason = %q, want %q", d.Reason, reason)
	}
	return d
}

// types is the journal's event sequence, which is the thing worth asserting: the
// summary already reports the final counts, and what it cannot report is the ORDER
// transitions happened in.
func journalTypes(events []model.JournalEvent) []string {
	out := make([]string, 0, len(events))
	for _, ev := range events {
		out = append(out, ev.Type)
	}
	return out
}

// payload decodes the first record of the given type into v. The journal stores a
// per-type payload under Data, so reading one back means naming the type you expect.
func payload(t *testing.T, events []model.JournalEvent, typ string, v any) model.JournalEvent {
	t.Helper()
	for _, ev := range events {
		if ev.Type != typ {
			continue
		}
		b, err := json.Marshal(ev.Data)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, v); err != nil {
			t.Fatalf("decode %s payload: %v", typ, err)
		}
		return ev
	}
	t.Fatalf("no %s record in the journal: %v", typ, journalTypes(events))
	return model.JournalEvent{}
}

// convergingLifecycle is the journal sequence of the run both
// TestRunJournalRecordsTheRoundLifecycle and
// TestRunSurvivesAJournalThatCannotBeWritten drive: one finding fixed and
// committed, then a clean round that ends the run. It is shared because the
// second test's property -- that a broken journal suppresses only the WARNING,
// never a write -- is exactly "every transition in this sequence was still
// attempted", and a count alone cannot say that.
var convergingLifecycle = []string{
	model.EvRunStarted,
	model.EvRoundStarted, model.EvReviewFinished, model.EvIssuesAggregated,
	model.EvFixFinished, model.EvRoundCommitted,
	model.EvRoundStarted, model.EvReviewFinished, model.EvIssuesAggregated,
	model.EvRoundClean,
	model.EvRunFinished,
}

// A converging run's journal must reconstruct the loop: review, aggregate, fix,
// commit, then a clean round that ends it. This is the property the summary cannot
// provide, because the summary is one whole-run write at the end.
func TestRunJournalRecordsTheRoundLifecycle(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("off by one")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, reviewResponse(t)) // round 2: clean

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatal(err)
	}

	events := f.journal()
	got := journalTypes(events)
	want := convergingLifecycle
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("journal sequence =\n  %v\nwant\n  %v", got, want)
	}

	// Round numbers must place each record in its iteration; a run-level record has
	// none, so a reader can tell "the run" from "a round".
	for _, ev := range events {
		switch ev.Type {
		case model.EvRunStarted, model.EvRunFinished:
			if ev.Round != 0 {
				t.Errorf("%s carries round %d, want none", ev.Type, ev.Round)
			}
		default:
			if ev.Round == 0 {
				t.Errorf("%s (seq %d) carries no round", ev.Type, ev.Seq)
			}
		}
	}

	var fix model.JournalFixFinished
	payload(t, events, model.EvFixFinished, &fix)
	if fix.Fixed != 1 || fix.Rejected != 0 || fix.Issues != 1 {
		t.Errorf("fix_finished = %+v, want 1 issue, 1 fixed, 0 rejected", fix)
	}
	if fix.Agent != "mock" {
		t.Errorf("fix_finished agent = %q, want mock", fix.Agent)
	}

	var commit model.JournalRoundCommitted
	payload(t, events, model.EvRoundCommitted, &commit)
	if commit.SHA == "" {
		t.Error("round_committed carries no SHA; it is the only durable pointer to the round's work")
	}
	if commit.Partial {
		t.Error("a normal round must not be recorded as partial")
	}

	var fin model.JournalRunFinished
	payload(t, events, model.EvRunFinished, &fin)
	if fin.Termination != model.TermConverged || fin.Rounds != 2 {
		t.Errorf("run_finished = %+v, want converged after 2 rounds", fin)
	}
}

// The gate's verdict is the one fact in the loop no model produced, so it has to be
// in the journal: the baseline (what was already failing before the run) and each
// attempt over the coder's edits.
func TestRunJournalRecordsVerification(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	events := f.journal()

	var base model.JournalVerifyFinished
	payload(t, events, model.EvVerifyBaseline, &base)
	if !base.Passed || len(base.Checks) != 1 || base.Checks[0].Name != "build" {
		t.Errorf("verify_baseline = %+v, want the build check passing on the pristine tree", base)
	}
	if base.Attempt != "" {
		t.Errorf("verify_baseline attempt = %q, want none: the baseline is not an attempt over a fix", base.Attempt)
	}

	ev := payload(t, events, model.EvVerifyFinished, &base)
	if base.Attempt != model.VerifyAttemptInitial {
		t.Errorf("verify_finished attempt = %q, want %q", base.Attempt, model.VerifyAttemptInitial)
	}
	if !base.Passed || len(base.Blocking) != 0 {
		t.Errorf("verify_finished = %+v, want a passing gate with nothing blocking", base)
	}
	if ev.Round != 1 {
		t.Errorf("verify_finished round = %d, want 1", ev.Round)
	}
}

// A round discarded by the gate must say so, with the reason and whether the work is
// recoverable. This is the exit that leaves a stash the operator did not ask for, so
// "which stash is this and why" has to be answerable from the journal.
func TestRunJournalRecordsDiscardedRound(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1})
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.breakBuildOn(2, "broken.txt")
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "still done"}))

	if _, err := f.orchestrator().Run(t.Context()); err == nil {
		t.Fatal("expected the run to fail after the gate rejected the round")
	}
	events := f.journal()

	var d model.JournalRoundDiscarded
	payload(t, events, model.EvRoundDiscarded, &d)
	if d.Reason != model.DiscardVerifyFailed {
		t.Errorf("discard reason = %q, want %q", d.Reason, model.DiscardVerifyFailed)
	}
	if !d.Stashed {
		t.Error("the coder's edits were stashed; the journal must say so or they look lost")
	}
	if len(d.Checks) == 0 {
		t.Error("discard record names no failing check")
	}
	if got := journalTypes(events); strings.Contains(strings.Join(got, ","), model.EvRoundCommitted) {
		t.Errorf("a discarded round must not be recorded as committed: %v", got)
	}

	// Both gate attempts are recorded: the correction attempt is the run's second
	// coder invocation, and a reader must be able to see that it happened and failed.
	attempts := 0
	for _, ev := range events {
		if ev.Type == model.EvVerifyFinished {
			attempts++
		}
	}
	if attempts != 2 {
		t.Errorf("verify_finished records = %d, want 2 (initial + correction)", attempts)
	}
}

// The journal must record which issues the per-round cap made wait, and not merely
// how many: aging exists to bound that wait, and the starvation bug it fixed is only
// visible by following an id across rounds.
func TestRunJournalRecordsDeferredIssues(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, MaxFindingsPerRound: 1})
	// Two DISTINCT problems: aFinding reuses one file and line, so two of those would
	// aggregate into a single issue and the cap would never bite.
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 1, Title: "first"},
		model.ReviewFinding{Category: "bugs", Severity: "low", File: "other.go", Line: 90, Title: "second"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatal(err)
	}

	var d model.JournalIssuesDeferred
	payload(t, f.journal(), model.EvIssuesDeferred, &d)
	if d.Cap != 1 || d.Deferred != 1 {
		t.Errorf("issues_deferred = %+v, want a cap of 1 deferring 1 issue", d)
	}
	if len(d.IDs) != 1 {
		t.Errorf("issues_deferred IDs = %v, want the one deferred issue named", d.IDs)
	}
}

// Corroboration is the panel's strongest signal and is invisible in the raw
// observation count, so the aggregation record carries it explicitly.
func TestRunJournalRecordsCorroboration(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	// Two DIFFERENT agents: corroboration counts distinct agents, so one agent
	// reporting twice is a duplicate, not agreement.
	f.cfg.Roles.Review.Prompts = []config.ReviewLens{
		{Agent: "mock", Prompt: f.cfg.Roles.Review.Prompts[0].Prompt},
		{Agent: "mock2", Prompt: f.cfg.Roles.Review.Prompts[0].Prompt},
	}
	f.cfg.Agents["mock2"] = f.cfg.Agents["mock"]
	// Both reviewers report the same problem; the coder then sees one issue.
	f.respond(1, reviewResponse(t, aFinding("same bug")))
	f.respond(2, reviewResponse(t, aFinding("same bug")))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatal(err)
	}

	var agg model.JournalIssuesAggregated
	payload(t, f.journal(), model.EvIssuesAggregated, &agg)
	if agg.Observations != 2 || agg.Issues != 1 {
		t.Errorf("issues_aggregated = %+v, want 2 observations grouped into 1 issue", agg)
	}
	if agg.Corroborated != 1 {
		t.Errorf("Corroborated = %d, want 1: two agents agreeing is the fact worth recording", agg.Corroborated)
	}
}

// A run refused by a gate journals the refusal and NOTHING else. No transition
// record may precede the symlink check that authorizes artifact writes, because
// rounds commit afterwards; the closing record is exempt because nothing commits
// after it -- the same reasoning that permits the unconditional summary write.
func TestRunRefusedByTrustGateJournalsOnlyTheRefusal(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1})
	f.cfg.Loop.TrustedTarget = false // the fixture asserts trust; withdraw it
	f.respond(1, reviewResponse(t))

	if _, err := f.orchestrator().Run(t.Context()); err == nil {
		t.Fatal("expected the untrusted-target gate to refuse the run")
	}

	events := f.journal()
	if got := journalTypes(events); len(got) != 1 || got[0] != model.EvRunFinished {
		t.Fatalf("journal = %v, want only %s: no transition record may be written before the symlink check", got, model.EvRunFinished)
	}
	var fin model.JournalRunFinished
	payload(t, events, model.EvRunFinished, &fin)
	if fin.Termination != model.TermError || fin.Rounds != 0 {
		t.Errorf("run_finished = %+v, want an error termination with no rounds", fin)
	}
	if !strings.Contains(fin.Error, "trusted-target") {
		t.Errorf("run_finished must name the refusal, got %q", fin.Error)
	}
}

// A journal write failure must NEVER fail the run. The journal is an audit
// artifact, and the alternative -- failing the run to protect it -- would make the
// observability feature the most likely cause of a lost round, discarding fixes that
// already passed verification and were committed. The warning is emitted once, so a
// full disk cannot bury the run's real output under a line per transition.
//
// The failure is injected through o.journalWrite rather than by breaking the
// filesystem: a read-only path still succeeds for root, which CI often is, so a
// filesystem-based version of this test would silently assert nothing.
func TestRunSurvivesAJournalThatCannotBeWritten(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("off by one")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, reviewResponse(t)) // round 2: clean -> converge

	var mu sync.Mutex
	var lines []string
	logf := func(format string, args ...any) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, fmt.Sprintf(format, args...))
	}
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, logf)
	if err != nil {
		t.Fatal(err)
	}
	// Record what each transition TRIED to write, not just how many tried: an
	// implementation that gave up after the fifth failure would satisfy a count
	// assertion while losing every later record, including the ones that say how
	// the run ended.
	var attempted []string
	o.journalWrite = func(typ string, _ int, _ any) error {
		attempted = append(attempted, typ)
		return errors.New("no space left on device")
	}

	before := f.commitCount()
	sum, err := o.Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want a broken journal to be survivable", err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged", sum.Termination)
	}
	if got := f.commitCount(); got != before+1 {
		t.Errorf("commit count = %d, want %d: the verified fix must still be committed", got, before+1)
	}
	// Every transition still tries, in order: giving up part-way would lose the
	// records a journal that recovers (a freed disk) could still have held. The
	// expected sequence is the one TestRunJournalRecordsTheRoundLifecycle asserts on
	// a working journal, because this test drives the same run.
	if strings.Join(attempted, ",") != strings.Join(convergingLifecycle, ",") {
		t.Errorf("attempted journal writes =\n  %v\nwant one per transition\n  %v", attempted, convergingLifecycle)
	}
	warnings := 0
	for _, l := range lines {
		if strings.Contains(l, "journal unavailable") {
			warnings++
			if !strings.Contains(l, "no space left on device") || !strings.Contains(l, "continues") {
				t.Errorf("the warning must carry the cause and say the run continues: %q", l)
			}
		}
	}
	if warnings != 1 {
		t.Errorf("journal warnings = %d, want exactly 1 for the whole run", warnings)
	}
}
