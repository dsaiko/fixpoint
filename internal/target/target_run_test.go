package target

import (
	"context"
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

// The shared BoundedBuffer's unit-level truncation and concurrency behavior is
// covered by TestBoundedBuffer(ConcurrentWriteString) in the agent package. This
// test exercises target.run's use of it end to end: a subprocess emitting well
// beyond maxRunOutput must still complete successfully, with its output
// truncated rather than surfaced as a write error.
func TestRunTruncatesOversizedSubprocessOutput(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to emit oversized output")
	}
	c := New(config.Target{Path: t.TempDir()})
	// ~5 MB on stdout, exit 0. Before BoundedBuffer reported the full write
	// length, os/exec turned this into io.ErrShortWrite.
	out, err := c.run(t.Context(), "sh", "-c", "head -c 5000000 /dev/zero | tr '\\0' a")
	if err != nil {
		t.Fatalf("run() on oversized output = %v, want success", err)
	}
	if !strings.Contains(out, "output truncated at 4 MB") {
		t.Errorf("oversized output not marked truncated (len=%d)", len(out))
	}
}

// run must return stdout ALONE on success: callers parse it as a SHA, PR base
// OID, remote name, or file list, so a successful command that also writes a
// warning to stderr must not fold that text into the parsed value. On failure
// the stderr must instead surface in the error for diagnosis.
func TestRunSeparatesStdoutFromStderr(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to emit distinct streams")
	}
	c := New(config.Target{Path: t.TempDir()})

	// Success: a machine-readable value on stdout, a warning on stderr.
	out, err := c.run(t.Context(), "sh", "-c", "echo MACHINE_VALUE; echo human-warning >&2")
	if err != nil {
		t.Fatalf("run() = %v, want success", err)
	}
	if strings.TrimSpace(out) != "MACHINE_VALUE" {
		t.Errorf("run() = %q, want exactly the stdout value (stderr must not be folded in)", out)
	}

	// Failure: stderr is included in the returned error so the diagnostic is not lost.
	_, err = c.run(t.Context(), "sh", "-c", "echo OUT; echo ERR_DIAGNOSTIC >&2; exit 3")
	if err == nil || !strings.Contains(err.Error(), "ERR_DIAGNOSTIC") {
		t.Fatalf("run() err = %v, want it to carry the stderr diagnostic", err)
	}
}

// A git/gh that exits SUCCESSFULLY after backgrounding a child leaves that child
// running: cmd.Cancel only fires on cancellation. The child -- a credential
// helper, a hook, one of gh's internal git calls -- is then free to mutate the
// repository while the clean-tree check, verification, or the round commit runs.
// run kills the whole process group on every exit path, like agent.Run.
func TestRunKillsBackgroundedChildOnSuccess(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to spawn a child process")
	}
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "child-survived")
	c := New(config.Target{Path: dir})
	// The child's pipes go to /dev/null so it does not hold run's output pipes open:
	// this is the success path, where the command is reported as done immediately
	// and nothing else would ever reap the child.
	out, err := c.run(t.Context(), "sh", "-c",
		"( sleep 1; touch '"+sentinel+"' ) >/dev/null 2>&1 &\necho leader done\nexit 0")
	if err != nil {
		t.Fatalf("run() = %v, want success", err)
	}
	if !strings.Contains(out, "leader done") {
		t.Errorf("run() = %q, want the leader's stdout", out)
	}
	// Well past the child's own delay: if it were still alive it would have run.
	time.Sleep(2 * time.Second)
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("a backgrounded child survived a successful command and mutated the directory afterwards")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// The process group must be killed on the LEADER's exit, not once the output
// pipes have drained. A child that inherited the pipe keeps it open, so draining
// first would leave it alive for the whole drain -- long enough to mutate the
// repository after the git/gh call that spawned it has already returned, which is
// exactly what the clean-tree check, verification and the round commit rely on
// not happening.
func TestRunKillsPipeHoldingChildBeforeItCanMutateTheRepo(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to spawn a child process")
	}
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "mutated-during-drain")
	c := New(config.Target{Path: dir})
	// No redirection, so the child holds the output pipe open; it mutates the
	// directory a second later, while the leader exits 0 at once.
	out, err := c.run(t.Context(), "sh", "-c",
		"( sleep 1; touch '"+sentinel+"' ) &\necho leader done\nexit 0")
	if err != nil {
		t.Fatalf("run() = %v, want success", err)
	}
	if !strings.Contains(out, "leader done") {
		t.Errorf("run() = %q, want the leader's stdout", out)
	}
	// Past the child's own delay: if it outlived the leader it has run by now.
	time.Sleep(1500 * time.Millisecond)
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("a pipe-holding child survived a completed command and mutated the directory afterwards")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// The same interleaving with the child's redirection removed -- what a
// backgrounded child (a credential-cache daemon left by a fetch, a hook's child)
// does unless it explicitly redirects. It inherits the output pipe, so no EOF
// arrives on its own even though git/gh exited 0. run must report the success: a
// `git commit` that landed reported as failed ends the run with no CommitSHA
// while the commit sits in history.
func TestRunSucceedsWhenDescendantHoldsOutputPipe(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to spawn a child process")
	}
	c := New(config.Target{Path: t.TempDir()})
	type result struct {
		out string
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := c.run(t.Context(), "sh", "-c", "sleep 30 &\necho MACHINE_VALUE\nexit 0")
		done <- result{out, err}
	}()
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("run() = %v, want success: the leader exited 0, only a descendant held the pipe", got.err)
		}
		if strings.TrimSpace(got.out) != "MACHINE_VALUE" {
			t.Errorf("run() = %q, want the leader's stdout", got.out)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("run hung with a descendant on the output pipe")
	}
}

// run honors context cancellation and kills the whole process group: a canceled
// command with a backgrounded child (which inherits the stdout pipe) must return
// promptly rather than blocking on the child until it exits ~60s later.
func TestRunContextCancelKillsProcessGroup(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to spawn a child process")
	}
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	c := New(config.Target{Path: dir})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	start := time.Now()
	// The command signals readiness (so cancellation hits a live process, not a
	// startup race), backgrounds a long child in the same process group, then
	// sleeps itself. Only a process-group kill stops both.
	go func() {
		_, err := c.run(ctx, "sh", "-c", "touch ready\nsleep 60 &\nsleep 60")
		done <- err
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("subprocess never signaled readiness")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("run() err = nil, want cancellation error")
		}
		if elapsed := time.Since(start); elapsed > 15*time.Second {
			t.Fatalf("run took %s; the process-group kill / drain grace did not bound the wait", elapsed)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("run hung well past cancellation; process group not killed")
	}
}

// The drain grace must bound how long run blocks when a grandchild in its OWN
// process group survives the process-group kill and keeps the stdout pipe open.
// Unbounded, the copy goroutine would wait for EOF until that grandchild exits
// (60s here); with the grace, run returns ~2s after the kill.
func TestRunDrainGraceBoundsLeakedPipe(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl unavailable to spawn a detached pipe-holder")
	}
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	t.Cleanup(func() { reapDetachedChild(t, pidFile) })

	// perl puts itself in a fresh process group (setpgrp) and inherits stdout, so
	// the cancel's process-group kill misses it; it holds the pipe for 60s and
	// publishes its PID so the test can reap it rather than leak a 60s sleeper.
	holder := "perl -e 'setpgrp(0,0); open(F,\">\",$ARGV[0]) or die; print F $$; close F; sleep 60' " +
		pidFile + " &\nsleep 60"
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	col := New(config.Target{Path: dir})
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, err := col.run(ctx, "sh", "-c", holder)
		done <- err
	}()
	// Give the holder a moment to detach and inherit the pipe, then cancel.
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("run() err = nil, want cancellation error")
		}
		if elapsed := time.Since(start); elapsed > 15*time.Second {
			t.Fatalf("run took %s; the drain grace did not bound the wait on the leaked pipe", elapsed)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("run hung well past the drain grace; leaked-pipe guard is not working")
	}
}

// reapDetachedChild kills the sleeper that TestRunDrainGraceBoundsLeakedPipe
// deliberately detaches into its own process group (so the drain grace expires
// before it exits) and waits for it to disappear, so a passing test leaves
// nothing behind. The child publishes its PID; tolerate it not being written yet
// since run can return (once the grace expires) before that write.
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
	for range 100 {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("detached child %d still alive after SIGKILL", pid)
}
