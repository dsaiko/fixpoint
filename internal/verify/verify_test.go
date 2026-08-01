package verify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
)

func cfg(timeout time.Duration, cmds ...config.VerifyCommand) config.Verify {
	return config.Verify{Policy: config.VerifyMustPass, Timeout: config.Duration(timeout), Commands: cmds}
}

func TestRunRecordsPassAndFail(t *testing.T) {
	rep := Run(t.Context(), cfg(time.Minute,
		config.VerifyCommand{Name: "ok", Run: []string{"true"}},
		config.VerifyCommand{Name: "bad", Run: []string{"false"}},
	), t.TempDir(), nil)

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
	), t.TempDir(), nil)
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
	), t.TempDir(), nil)
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
	), t.TempDir(), nil)
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

// A baseline entry that could not RUN -- a missing binary, a bad working
// directory, a timeout -- records nothing about the project: it records that the
// gate never executed. Treating it as a pre-existing failure would exempt that
// command for the whole run, so the configured check could keep never running
// while every round is committed as if it had passed.
func TestRegressionsRejectsAnUnusableBaselineEntry(t *testing.T) {
	baseline := Report{Results: []Result{
		{Name: "test", Err: "exec: \"go\": executable file not found in $PATH"},
		{Name: "lint", Err: "timed out after 1m0s"},
		{Name: "build", Passed: false}, // a genuine pre-existing failure
	}}
	after := Report{Results: []Result{
		{Name: "test", Err: "exec: \"go\": executable file not found in $PATH"},
		{Name: "lint", Err: "timed out after 1m0s"},
		{Name: "build", Passed: false},
	}}
	got := names(after.Regressions(baseline))
	if len(got) != 2 || got[0] != "test" || got[1] != "lint" {
		t.Errorf("Regressions() = %v, want the two unusable-baseline commands to block (build is genuinely pre-existing)", got)
	}
	// An unusable baseline command that later succeeds is not a regression.
	fixed := Report{Results: []Result{{Name: "test", Passed: true}}}
	if r := fixed.Regressions(baseline); len(r) != 0 {
		t.Errorf("Regressions() = %v, want none once the command runs and passes", names(r))
	}
	// And the caller can name them, so the operator is not told they are merely
	// "pre-existing failures policy permits".
	if u := names(baseline.Unrunnable()); len(u) != 2 {
		t.Errorf("Unrunnable() = %v, want [test lint]", u)
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
	), dir, nil)
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
	), dir, nil)
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
	), t.TempDir(), nil)
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

// The verify gate is the one execution path whose argv the TARGET can supply (a
// bundle file inside the target shadows the operator's), so it must run with the
// filtered environment the caller hands it and not with fixpoint's own. Without
// this, a verify command reads every agent credential straight out of its
// environment -- before any coder edit -- and the filtering agents get is moot.
func TestRunUsesTheGivenEnvironment(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "SENTINEL-must-not-be-visible")
	dir := t.TempDir()
	script := filepath.Join(dir, "dump.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nenv\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	cmds := cfg(time.Minute, config.VerifyCommand{Name: "dump", Run: []string{script}})

	rep := Run(t.Context(), cmds, dir, agent.EnvWithoutCredentials(nil))
	if !rep.Results[0].Passed {
		t.Fatalf("dump command failed: %+v", rep.Results[0])
	}
	// Assert on the variable NAME, not the value: the redactor masks a known
	// credential's value on its way into the report, which would make a
	// value-based assertion pass even with the key fully present in the process.
	if strings.Contains(rep.Results[0].Output, "ANTHROPIC_API_KEY") {
		t.Errorf("an agent credential reached a verify command:\n%s", rep.Results[0].Output)
	}
	// Guard the guard: with the unfiltered environment the variable IS visible, so
	// the assertion above is really testing the filtering and not a broken dumper.
	inherited := Run(t.Context(), cmds, dir, os.Environ())
	if !strings.Contains(inherited.Results[0].Output, "ANTHROPIC_API_KEY") {
		t.Fatalf("the env dumper prints nothing useful, so the filtering assertion proves nothing:\n%s", inherited.Results[0].Output)
	}
}

// A verify command that exits SUCCESSFULLY after backgrounding a child leaves that
// child running: cmd.Cancel only fires on cancellation. The child is then free to
// edit the repository while the next check, the clean-tree check, or the round
// commit runs -- so a round nobody verified gets committed as a verified one.
// runOne kills the whole process group on every exit path, like agent.Run.
func TestRunKillsBackgroundedChildrenOnSuccess(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "child-survived")
	script := filepath.Join(dir, "leader.sh")
	// The child's pipes are redirected so the leader's exit is not held up by them:
	// this is the case that is reported as PASSED immediately, where nothing else
	// would ever reap the child.
	body := fmt.Sprintf("#!/bin/sh\n( sleep 1; touch '%s' ) >/dev/null 2>&1 &\necho leader done\nexit 0\n", sentinel)
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}

	rep := Run(t.Context(), cfg(time.Minute,
		config.VerifyCommand{Name: "leader", Run: []string{script}},
	), dir, nil)
	if !rep.Results[0].Passed {
		t.Fatalf("the leader exits 0 and must be recorded as passing: %+v", rep.Results[0])
	}

	// Well past the child's own delay: if it were still alive it would have run.
	time.Sleep(2 * time.Second)
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("a backgrounded child survived a successful verification and mutated the directory afterwards")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// boundedBuffer's mutex exists for the WaitDelay case: cmd.Run can return while
// the copy goroutine still drains a pipe a leaked grandchild holds open, so
// String() overlaps a Write. Every other test in this file drives commands
// sequentially and would pass with the locking removed; this one overlaps the two
// and is meaningful only under -race (which `make audit` runs).
func TestBoundedBufferConcurrentWriteString(t *testing.T) {
	var b boundedBuffer
	chunk := []byte(strings.Repeat("x", 4096))
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 2000 {
			if _, err := b.Write(chunk); err != nil {
				t.Errorf("Write: %v", err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for range 2000 {
			_ = b.String()
		}
	}()
	wg.Wait()
}

// Overflow past maxOutput is dropped, not stored, and the reader is told how much
// went missing -- silently truncating would make a coder chase a failure whose
// cause is in the part that vanished.
func TestBoundedBufferTruncatesAndReportsTheDrop(t *testing.T) {
	var b boundedBuffer
	if n, err := b.Write([]byte(strings.Repeat("a", maxOutput))); n != maxOutput || err != nil {
		t.Fatalf("Write returned (%d, %v), want (%d, nil)", n, err, maxOutput)
	}
	// A short write once full, then a longer one: both are accounted for, and Write
	// must still report every byte consumed or io.Copy treats it as a short write.
	if n, err := b.Write([]byte("bb")); n != 2 || err != nil {
		t.Fatalf("Write after full returned (%d, %v), want (2, nil)", n, err)
	}
	if n, err := b.Write([]byte(strings.Repeat("c", 1024))); n != 1024 || err != nil {
		t.Fatalf("Write after full returned (%d, %v), want (1024, nil)", n, err)
	}
	body, marker, found := strings.Cut(b.String(), "\n[...")
	if !found {
		t.Fatalf("String() carries no truncation marker, so the reader is not told anything went missing")
	}
	if body != strings.Repeat("a", maxOutput) {
		t.Errorf("retained output is %d bytes and not exactly the first %d written; output past the cap must be dropped, which is what stops a runaway suite exhausting memory", len(body), maxOutput)
	}
	if !strings.Contains(marker, "dropped") {
		t.Errorf("the truncation must say output was dropped, got: %q", marker)
	}
	// The dropped total is the second and third writes together (2 + 1024 bytes).
	if !strings.Contains(marker, agent.HumanSize(1026)) {
		t.Errorf("the dropped-byte count must be reported (want %s), got: %q", agent.HumanSize(1026), marker)
	}
}
