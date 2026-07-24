package verify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
)

func cfg(timeout time.Duration, cmds ...config.VerifyCommand) config.Verify {
	return config.Verify{Policy: config.VerifyMustPass, Timeout: config.Duration(timeout), Commands: cmds}
}

func TestRunRecordsPassAndFail(t *testing.T) {
	rep := Run(t.Context(), cfg(time.Minute,
		config.VerifyCommand{Name: "ok", Run: []string{"true"}},
		config.VerifyCommand{Name: "bad", Run: []string{"false"}},
	), t.TempDir())

	if len(rep.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(rep.Results))
	}
	if !rep.Results[0].Passed {
		t.Error("a zero-exit command must be recorded as passed")
	}
	if rep.Results[1].Passed {
		t.Error("a non-zero exit must be recorded as failed")
	}
	if rep.Passed() {
		t.Error("Report.Passed() must be false when a required command failed")
	}
	if got := rep.Failures(); len(got) != 1 || got[0].Name != "bad" {
		t.Errorf("Failures() = %v, want just [bad]", got)
	}
}

// A command that cannot be started at all -- a missing binary, a typo in the
// config -- must fail loudly. Treating it as a pass would turn a misconfigured
// gate into no gate, silently.
func TestRunUnstartableCommandFails(t *testing.T) {
	rep := Run(t.Context(), cfg(time.Minute,
		config.VerifyCommand{Name: "ghost", Run: []string{"definitely-not-a-real-binary-xyz"}},
	), t.TempDir())
	r := rep.Results[0]
	if r.Passed {
		t.Error("an unstartable command must not be reported as passing")
	}
	if r.Err == "" {
		t.Error("an unstartable command must record why it could not run")
	}
}

// Optional commands are recorded but never gate the round.
func TestOptionalCommandDoesNotBlock(t *testing.T) {
	rep := Run(t.Context(), cfg(time.Minute,
		config.VerifyCommand{Name: "lint", Run: []string{"false"}, Optional: true},
	), t.TempDir())
	if !rep.Passed() {
		t.Error("an optional failure must not make the report fail")
	}
	if len(rep.Failures()) != 0 {
		t.Error("an optional failure must not appear in Failures()")
	}
	if rep.Results[0].Passed {
		t.Error("the result itself must still record that it failed")
	}
}

// A hung check must not hang the run.
func TestRunTimesOutAndKillsTheCommand(t *testing.T) {
	start := time.Now()
	rep := Run(t.Context(), cfg(300*time.Millisecond,
		config.VerifyCommand{Name: "hang", Run: []string{"sleep", "60"}},
	), t.TempDir())
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Fatalf("timeout was not enforced: took %s", elapsed)
	}
	r := rep.Results[0]
	if r.Passed {
		t.Error("a timed-out command must not pass")
	}
	if !strings.Contains(r.Err, "timed out") {
		t.Errorf("Err = %q, want a timeout diagnostic", r.Err)
	}
}

// no_regressions is what makes fixpoint usable on a repository that is already
// red: a check failing before the run may keep failing, but one that passed must
// not start failing.
func TestRegressionsComparesAgainstBaseline(t *testing.T) {
	baseline := Report{Results: []Result{
		{Name: "test", Passed: true},
		{Name: "lint", Passed: false}, // already broken before fixpoint ran
	}}
	after := Report{Results: []Result{
		{Name: "test", Passed: false}, // newly broken -- this run's fault
		{Name: "lint", Passed: false}, // still broken -- not this run's fault
	}}
	got := after.Regressions(baseline)
	if len(got) != 1 || got[0].Name != "test" {
		t.Fatalf("Regressions() = %v, want just [test]", names(got))
	}
	if b := after.Blocking(config.VerifyNoRegressions, baseline); len(b) != 1 || b[0].Name != "test" {
		t.Errorf("Blocking(no_regressions) = %v, want just [test]", names(b))
	}
	// must_pass ignores the baseline: both failures block.
	if b := after.Blocking(config.VerifyMustPass, baseline); len(b) != 2 {
		t.Errorf("Blocking(must_pass) = %v, want both failures", names(b))
	}
	if b := after.Blocking(config.VerifyOff, baseline); len(b) != 0 {
		t.Errorf("Blocking(off) = %v, want nothing", names(b))
	}
}

// A command absent from the baseline counts as a regression: with no evidence it
// ever passed, the safe reading is that this run broke it. The unsafe reading
// would let a newly added check fail forever without ever blocking.
func TestRegressionsTreatsUnknownCommandAsRegression(t *testing.T) {
	after := Report{Results: []Result{{Name: "brand-new", Passed: false}}}
	if got := after.Regressions(Report{}); len(got) != 1 {
		t.Errorf("Regressions() = %v, want the unknown command treated as a regression", names(got))
	}
}

// Output is captured (both streams), bounded, and reported for diagnosis.
func TestRunCapturesCombinedOutput(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "noisy.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho to-stdout\necho to-stderr >&2\nexit 7\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	rep := Run(t.Context(), cfg(time.Minute,
		config.VerifyCommand{Name: "noisy", Run: []string{script}},
	), dir)
	r := rep.Results[0]
	if r.ExitCode != 7 {
		t.Errorf("ExitCode = %d, want 7", r.ExitCode)
	}
	for _, want := range []string{"to-stdout", "to-stderr"} {
		if !strings.Contains(r.Output, want) {
			t.Errorf("Output missing %q (both streams must be captured):\n%s", want, r.Output)
		}
	}
}

// Verification output is persisted and fed back to the coder, so a secret the
// build printed must be masked on the way through.
func TestRunRedactsSecretsInOutput(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "leak.sh")
	secret := "ghp" + "_ABCDEFGHIJKLMNOPQRSTUVWXYZ0"
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"token "+secret+"\"\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	rep := Run(t.Context(), cfg(time.Minute,
		config.VerifyCommand{Name: "leak", Run: []string{script}},
	), dir)
	if strings.Contains(rep.Results[0].Output, secret) {
		t.Errorf("a credential in build output reached the report unredacted:\n%s", rep.Results[0].Output)
	}
}

// The block handed back to the coder must name the command and show the output,
// so the coder can reproduce the failure rather than guess at it.
func TestFormatForCoderIsActionable(t *testing.T) {
	got := FormatForCoder([]Result{
		{Name: "test", Argv: []string{"go", "test", "./..."}, ExitCode: 1, Output: "FAIL: TestFoo"},
	})
	for _, want := range []string{"test", "go test ./...", "exit status 1", "FAIL: TestFoo"} {
		if !strings.Contains(got, want) {
			t.Errorf("FormatForCoder() missing %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "Do not disable") {
		t.Error("the coder must be told not to weaken a check to get past it")
	}
}

// Cancellation stops the pass rather than working through every remaining command.
func TestRunStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	rep := Run(ctx, cfg(time.Minute,
		config.VerifyCommand{Name: "one", Run: []string{"true"}},
		config.VerifyCommand{Name: "two", Run: []string{"true"}},
	), t.TempDir())
	if len(rep.Results) > 1 {
		t.Errorf("got %d results, want the pass to stop after the first on cancellation", len(rep.Results))
	}
}

func names(rs []Result) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Name)
	}
	return out
}
