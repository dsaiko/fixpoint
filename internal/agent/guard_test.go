package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
)

// reapDetachedChild kills the sleeper the leaked-pipe tests deliberately detach
// into their own process group (so the drain grace expires before it exits) and
// waits for it to disappear, so a successful run leaves nothing behind. The
// child publishes its PID to pidFile at startup; tolerate it not being written
// yet since Run can return (once the grace expires) before that write.
func reapDetachedChild(t *testing.T, pidFile string) {
	t.Helper()
	var pid int
	for range 50 {
		if b, err := os.ReadFile(pidFile); err == nil {
			if p, perr := strconv.Atoi(strings.TrimSpace(string(b))); perr == nil && p > 0 {
				pid = p
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid == 0 {
		t.Log("detached child never published its PID; nothing to reap")
		return
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	// Poll until the orphan is gone (reparented to init, which reaps it after the
	// kill). syscall.Kill(pid, 0) returns an error once it no longer exists.
	for range 100 {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("detached child %d still alive after SIGKILL", pid)
}

// KillProcessGroup must no-op on the two states that would otherwise be
// catastrophic when cmd.Cancel fires early: a nil Process (ctx firing before
// Start populates it -> nil deref) and a zero pid (syscall.Kill(-0, ...) would
// signal fixpoint's OWN process group). Reaching the end of the test with
// the process still alive is the proof the guard held.
func TestKillProcessGroupGuard(t *testing.T) {
	if err := KillProcessGroup(&exec.Cmd{}); err != nil { // Process == nil
		t.Errorf("KillProcessGroup(nil Process) = %v, want nil no-op", err)
	}
	if err := KillProcessGroup(&exec.Cmd{Process: &os.Process{Pid: 0}}); err != nil {
		t.Errorf("KillProcessGroup(pid 0) = %v, want nil no-op", err)
	}
}

// A group whose leader has already exited AND been reaped no longer exists, and
// syscall.Kill answers ESRCH. That is "already finished", not a failure, and
// os/exec only recognizes it as such when the error wraps os.ErrProcessDone --
// anything else and it rewrites a command that succeeded into
// `exec: canceling Cmd: no such process`.
func TestKillProcessGroupReportsProcessDone(t *testing.T) {
	cmd := exec.Command("true")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	if err := cmd.Wait(); err != nil { // reaps the leader, emptying the group
		t.Fatalf("Wait() = %v", err)
	}
	err := KillProcessGroup(cmd)
	if !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("KillProcessGroup(reaped leader) = %v, want an error wrapping os.ErrProcessDone", err)
	}
	if errors.Is(err, syscall.ESRCH) {
		t.Errorf("KillProcessGroup leaked the raw errno %v; os/exec does not recognize it as already-finished", err)
	}
}

// stubGroupKill makes every process-group signal answer from the test instead of
// the kernel: sigkill for the containment kill, probe for the signal-0 state
// probe. Nothing is signaled, so it is the only way to drive KillProcessGroup
// through an EPERM -- a reply that needs a group member running under other
// credentials, which a test cannot create. Restoring the real kill is left to
// t.Cleanup so a failing assertion cannot leak the stub into later tests, which
// would then kill nothing at all.
func stubGroupKill(t *testing.T, sigkill, probe error) {
	t.Helper()
	orig := groupKill
	t.Cleanup(func() { groupKill = orig })
	groupKill = func(_ int, sig syscall.Signal) error {
		if sig == 0 {
			return probe
		}
		return sigkill
	}
}

// stubbedPid is a pid handed to KillProcessGroup while groupKill is stubbed. It
// only has to pass the pid > 0 guard: no signal reaches it.
const stubbedPid = 424242

// observeCancelKill replaces the kill Supervise installs as cmd.Cancel with one
// that runs before (given the command) and then records what the real kill
// reported, and returns an accessor for those records. They are what makes the
// interleaving observable rather than assumed: a recorded os.ErrProcessDone means
// that cancel landed on a group whose leader cmd.Wait had already reaped, and an
// empty record means no cancel ran at all. Restoring the original is left to
// t.Cleanup so a failing assertion cannot leak the wrapper into later tests.
func observeCancelKill(t *testing.T, before func(*exec.Cmd)) func() []error {
	t.Helper()
	orig := cancelKill
	t.Cleanup(func() { cancelKill = orig })
	var mu sync.Mutex
	var seen []error
	cancelKill = func(cmd *exec.Cmd) error {
		if before != nil {
			before(cmd)
		}
		err := orig(cmd)
		mu.Lock()
		seen = append(seen, err)
		mu.Unlock()
		return err
	}
	return func() []error {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(seen)
	}
}

// waitLeaderReaped blocks until pid has left the process table entirely. A zombie
// still answers signal 0, so ESRCH here means cmd.Wait has already reaped the
// leader and the group it led is empty -- precisely the state whose ESRCH must
// reach os/exec as os.ErrProcessDone.
func waitLeaderReaped(pid int) error {
	for range 5000 {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return nil
		}
		time.Sleep(2 * time.Millisecond)
	}
	return fmt.Errorf("leader %d still in the process table; it was never reaped", pid)
}

// cancelOnWrite cancels the moment the child produces its first output, which is
// the child's own word that it is running -- so the cancel provably fires while
// the leader is still alive, which is what makes os/exec's watcher take the cancel
// path instead of its already-finished shortcut.
type cancelOnWrite struct {
	cancel func()
	once   sync.Once
}

func (w *cancelOnWrite) Write(p []byte) (int, error) {
	w.once.Do(w.cancel)
	return len(p), nil
}

// The interleaving that matters here: cmd.Wait reaps a leader that exited 0, and
// only then does the context watcher run cmd.Cancel, so the group kill finds
// nothing left. Force it rather than hoping a deadline lands inside it -- hold the
// kill back until the leader is provably reaped, so the cancel always meets an
// empty group. The leader's success must survive that, and it only does because
// KillProcessGroup reports os.ErrProcessDone instead of the raw ESRCH: with the
// errno, os/exec rewrites this run into `exec: canceling Cmd: no such process`.
func TestSuperviseCancelAfterReapKeepsSuccess(t *testing.T) {
	// Reported through a channel because the wait runs on os/exec's context watcher
	// goroutine, not this one.
	reapErrs := make(chan error, 1)
	kills := observeCancelKill(t, func(cmd *exec.Cmd) {
		if err := waitLeaderReaped(cmd.Process.Pid); err != nil {
			select {
			case reapErrs <- err:
			default:
			}
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	// Announces itself, then exits 0 well after the cancel has fired. The held-back
	// kill never reaches it, so its own successful exit is what cmd.Wait reaps.
	cmd := exec.CommandContext(ctx, "sh", "-c", "echo ready; sleep 1")
	_, err := Supervise(ctx, cmd, &cancelOnWrite{cancel: cancel}, nil)

	select {
	case reapErr := <-reapErrs:
		t.Fatalf("forcing the interleaving failed: %v", reapErr)
	default:
	}
	killed := kills()
	if len(killed) != 1 {
		t.Fatalf("the cancel kill ran %d times, want exactly once; the forced interleaving was not reached", len(killed))
	}
	if killErr := killed[0]; !errors.Is(killErr, os.ErrProcessDone) {
		t.Errorf("killing the group of an already-reaped leader = %v, want an error wrapping os.ErrProcessDone", killErr)
	}
	if cmd.ProcessState == nil || !cmd.ProcessState.Success() {
		t.Fatalf("leader state = %v, want a clean exit; the cancel must not have killed it", cmd.ProcessState)
	}
	if err != nil {
		t.Fatalf("Supervise() err = %v for a leader that exited 0 before the cancel; want nil", err)
	}
}

// The mirror of the interleaving above, and the other half of the cancel path:
// the context fires while the leader is still running, so cmd.Cancel's group kill
// lands on a LIVE group and is what ends the command. The child's own first byte
// of output is what triggers the cancel, so this reaches the kill without
// depending on a wall clock -- unlike the sweep below, whose deadlines can only
// aim at that window and can be pushed out of it by load either way.
func TestSuperviseCancelBeforeExitKillsGroup(t *testing.T) {
	kills := observeCancelKill(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	// Announces itself and then sleeps far past anything this test waits for, so the
	// cancel that its announcement triggers always finds the leader alive.
	cmd := exec.CommandContext(ctx, "sh", "-c", "echo ready; sleep 60")
	_, err := Supervise(ctx, cmd, &cancelOnWrite{cancel: cancel}, nil)

	killed := kills()
	if len(killed) != 1 {
		t.Fatalf("the cancel kill ran %d times, want exactly once; the cancel never reached a running leader", len(killed))
	}
	if killErr := killed[0]; killErr != nil {
		t.Errorf("killing the group of a live leader = %v, want nil", killErr)
	}
	if cmd.ProcessState == nil || cmd.ProcessState.Success() {
		t.Errorf("leader state = %v, want an unsuccessful exit; the group kill is what must have ended it", cmd.ProcessState)
	}
	if err == nil {
		t.Error("Supervise() err = nil for a leader the cancel killed; want the kill reported")
	}
	if errors.Is(err, syscall.ESRCH) {
		t.Errorf("Supervise() err = %v; the kill of a live group leaked a raw errno", err)
	}
}

// superviseBaseline is the median wall time of an uncanceled Supervise of a
// trivial command on the machine running the test. The sweep below aims its
// deadlines at that figure because a fixed range only guesses at what a fork,
// exec and exit cost here: guessed deadlines mostly expire before the command
// starts or long after it finished, and the sweep then never reaches the
// interleaving it is named for.
func superviseBaseline(t *testing.T) time.Duration {
	t.Helper()
	runs := make([]time.Duration, 0, 20)
	for range 20 {
		start := time.Now()
		cmd := exec.CommandContext(t.Context(), "true")
		if _, err := Supervise(t.Context(), cmd, io.Discard, nil); err != nil {
			t.Fatalf("Supervise() of an uncanceled command = %v", err)
		}
		runs = append(runs, time.Since(start))
	}
	slices.Sort(runs)
	return runs[len(runs)/2]
}

const sweepIterations = 400

// sweepDeadlines runs one pass of the sweep, asserting what must hold at every
// offset regardless of where that iteration's cancellation actually landed.
func sweepDeadlines(t *testing.T, base time.Duration) {
	t.Helper()
	for i := range sweepIterations {
		// A quarter of the baseline up to twice it: short enough at the start that the
		// kill beats the command, long enough at the end that the command beats it.
		deadline := base/4 + time.Duration(int64(2*base)*int64(i)/sweepIterations)
		ctx, cancel := context.WithTimeout(t.Context(), deadline)
		cmd := exec.CommandContext(ctx, "true")
		_, err := Supervise(ctx, cmd, io.Discard, nil)
		cancel()
		if errors.Is(err, syscall.ESRCH) {
			t.Fatalf("Supervise() err = %v; a completed command was reported as a kill failure", err)
		}
		if err != nil && cmd.ProcessState != nil && cmd.ProcessState.Success() && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Supervise() err = %v for a leader that exited 0; want nil or the context's own error", err)
		}
	}
}

// The two forced tests above pin the interleavings themselves -- the cancel that
// meets a live leader and the one that meets an already-reaped one -- without a
// wall clock. This sweeps deadlines across the whole window in which a trivial
// command starts, exits and is reaped, so cancellation also lands at the offsets
// BETWEEN those two, and requires that no offset surfaces as an ESRCH failure or
// rewrites a successful leader's result.
//
// It also requires the sweep not to be vacuous: every assertion in sweepDeadlines
// holds trivially if each command simply finishes before its deadline, so at least
// one iteration must have canceled a run that was still under way. That is the one
// claim here a scheduler can starve, since a deadline can only aim at the window,
// so a pass that observes nothing re-measures and sweeps again instead of failing:
// both ways a sweep misses entirely -- the machine slower than when the baseline
// was taken, so every deadline expires before Start, or faster, so every command
// beats its deadline -- are drift between the measurement and the sweep, and a
// fresh baseline follows that drift. Only a sweep mis-scaled at every load it is
// measured under fails.
func TestSuperviseSuccessRacingDeadline(t *testing.T) {
	kills := observeCancelKill(t, nil)
	var base time.Duration
	for attempt := 1; attempt <= 3; attempt++ {
		base = superviseBaseline(t)
		sweepDeadlines(t, base)
		if len(kills()) > 0 {
			break
		}
		t.Logf("attempt %d (baseline %v) canceled no run in progress; re-measuring and sweeping again", attempt, base)
	}
	killed := kills()
	if len(killed) == 0 {
		t.Fatalf("no iteration canceled a run in progress (last baseline %v); the sweep never exercised the deadline race", base)
	}
	reaped := 0
	for _, err := range killed {
		if errors.Is(err, os.ErrProcessDone) {
			reaped++
		}
	}
	t.Logf("baseline %v: canceled %d runs in progress out of the %d swept per pass, %d of them after the leader was reaped",
		base, len(killed), sweepIterations, reaped)
}

// stubCleanupKill makes the post-Wait cleanup kill report killErr after the real
// kill has run, so the command is still contained while the test observes what
// Supervise does with a reply it cannot provoke for real.
func stubCleanupKill(t *testing.T, killErr error) {
	t.Helper()
	orig := cleanupKill
	t.Cleanup(func() { cleanupKill = orig })
	cleanupKill = func(cmd *exec.Cmd) error {
		_ = orig(cmd)
		return killErr
	}
}

// The cleanup kill after cmd.Wait is the ORDINARY path -- it runs on every
// command, canceled or not, and is the only containment a leader that exited 0
// ever gets. Off darwin an EPERM from it proves the group still holds a
// descendant this process cannot signal, so discarding it reports a successful,
// contained command while that descendant is still live inside the target
// repository, free to edit it during the verification and commit that follow.
func TestSuperviseReportsFailedCleanupKill(t *testing.T) {
	stubCleanupKill(t, syscall.EPERM)
	cmd := exec.CommandContext(t.Context(), "true")
	_, err := Supervise(t.Context(), cmd, io.Discard, nil)
	if err == nil {
		t.Fatal("Supervise() err = nil for a leader that exited 0 whose group kill failed; the uncontained group must be reported")
	}
	if !errors.Is(err, syscall.EPERM) {
		t.Errorf("Supervise() err = %v, want it to carry the kill's EPERM", err)
	}
	if cmd.ProcessState == nil || !cmd.ProcessState.Success() {
		t.Errorf("leader state = %v, want a clean exit; the failure under test is the kill's, not the leader's", cmd.ProcessState)
	}
}

// The other half of that rule: an empty group is what the cleanup kill normally
// meets, and reporting its os.ErrProcessDone would turn every successful command
// in the program into a failure.
func TestSuperviseIgnoresProcessDoneFromCleanupKill(t *testing.T) {
	stubCleanupKill(t, os.ErrProcessDone)
	if _, err := Supervise(t.Context(), exec.CommandContext(t.Context(), "true"), io.Discard, nil); err != nil {
		t.Errorf("Supervise() err = %v for a cleanup kill that found an empty group; want nil", err)
	}
}

// Supervise owns the output pipes, so a caller that has already set cmd.Stdout
// or cmd.Stderr holds a mistaken idea of where the output goes: Supervise would
// overwrite the writer and silently drop that stream. Fail closed instead of
// running the command.
func TestSuperviseRejectsPresetOutputWriters(t *testing.T) {
	var sink strings.Builder
	preset := exec.Command("true")
	preset.Stdout = &sink
	if _, err := Supervise(t.Context(), preset, io.Discard, nil); err == nil {
		t.Error("Supervise must refuse a cmd whose Stdout is already set")
	}
	preset = exec.Command("true")
	preset.Stderr = &sink
	if _, err := Supervise(t.Context(), preset, io.Discard, nil); err == nil {
		t.Error("Supervise must refuse a cmd whose Stderr is already set")
	}
	if sink.Len() != 0 {
		t.Errorf("the refused commands must not have run; sink = %q", sink.String())
	}
}

// The drain grace must bound how long Run blocks when a grandchild in its OWN
// process group survives the process-group kill and keeps the stdout pipe open.
// Unbounded, the copy goroutine would wait for EOF until that grandchild exits
// (60s here); with the grace, Run returns ~2s after the kill.
func TestRunDrainGraceBoundsLeakedPipe(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl unavailable to spawn a detached pipe-holder")
	}
	// perl puts itself in a fresh process group (setpgrp) and inherits stdout,
	// so the timeout's process-group kill misses it; it holds the pipe for 60s.
	// It publishes its PID so the test can reap it: without that, a passing run
	// leaves the ~60s sleeper behind, polluting the host and piling up across
	// repeated or -race runs.
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	t.Cleanup(func() { reapDetachedChild(t, pidFile) })
	a := config.Agent{
		Command: []string{script(t, "perl -e 'setpgrp(0,0); open(F,\">\",$ARGV[0]) or die; print F $$; close F; sleep 60' "+
			pidFile+" &\nsleep 60")},
		PromptVia: "stdin",
		Timeout:   config.Duration(300 * time.Millisecond),
	}
	done := make(chan Result, 1)
	start := time.Now()
	go func() { done <- Run(t.Context(), a, "", t.TempDir()) }()
	select {
	case res := <-done:
		if res.Err == nil || !strings.Contains(res.Err.Error(), "timed out") {
			t.Fatalf("Run() err = %v, want timeout error", res.Err)
		}
		// The grace is 2s; a generous ceiling still far below the 60s the leaked
		// pipe-holder would otherwise impose.
		if elapsed := time.Since(start); elapsed > 15*time.Second {
			t.Fatalf("Run took %s; the drain grace did not bound the wait on the leaked pipe", elapsed)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Run hung well past the drain grace; leaked-pipe guard is not working")
	}
}
