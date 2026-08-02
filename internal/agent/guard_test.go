package agent

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

// When a command completes just as its context expires, cmd.Wait can reap the
// leader before the context watcher runs cmd.Cancel, so the group kill finds
// nothing left. Sweep deadlines across the window where a trivial command
// finishes and require that this interleaving never surfaces as an ESRCH
// failure: the leader's own result must stand.
func TestSuperviseSuccessRacingDeadline(t *testing.T) {
	for i := range 400 {
		ctx, cancel := context.WithTimeout(t.Context(), time.Duration(500+i*10)*time.Microsecond)
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
