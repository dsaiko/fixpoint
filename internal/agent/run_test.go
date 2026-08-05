package agent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
)

// script writes an executable shell script and returns its path.
func script(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "agent.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o700); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunPromptViaStdin(t *testing.T) {
	a := config.Agent{
		Command:   []string{script(t, "cat")},
		PromptVia: "stdin",
		Timeout:   config.Duration(time.Minute),
	}
	res := Run(t.Context(), a, "hello prompt", t.TempDir())
	if res.Err != nil {
		t.Fatalf("Run() err = %v", res.Err)
	}
	if res.Stdout != "hello prompt" {
		t.Errorf("stdout = %q, want the prompt echoed back", res.Stdout)
	}
}

func TestRunPromptViaArg(t *testing.T) {
	a := config.Agent{
		Command:   []string{script(t, `printf '%s' "$1"`)},
		PromptVia: "arg",
		Timeout:   config.Duration(time.Minute),
	}
	res := Run(t.Context(), a, "arg prompt", t.TempDir())
	if res.Err != nil {
		t.Fatalf("Run() err = %v", res.Err)
	}
	if res.Stdout != "arg prompt" {
		t.Errorf("stdout = %q, want the prompt as argv", res.Stdout)
	}
}

func TestRunCapturesStderrAndExitError(t *testing.T) {
	a := config.Agent{
		Command:   []string{script(t, "echo out; echo err >&2; exit 3")},
		PromptVia: "stdin",
		Timeout:   config.Duration(time.Minute),
	}
	res := Run(t.Context(), a, "", t.TempDir())
	if res.Err == nil {
		t.Fatal("Run() err = nil, want non-zero exit error")
	}
	if !strings.Contains(res.Stdout, "out") || !strings.Contains(res.Stderr, "err") {
		t.Errorf("streams: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}
}

func TestRunRunsInDir(t *testing.T) {
	dir := t.TempDir()
	a := config.Agent{
		Command:   []string{script(t, "pwd")},
		PromptVia: "stdin",
		Timeout:   config.Duration(time.Minute),
	}
	res := Run(t.Context(), a, "", dir)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	got, err := filepath.EvalSymlinks(strings.TrimSpace(res.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("cwd = %q, want %q", got, want)
	}
}

func TestRunTimeoutKillsProcessGroup(t *testing.T) {
	// The script backgrounds a long-lived child that inherits the stdout
	// pipe: unless the whole process group is killed, Run blocks on the pipe
	// long after the direct child is gone.
	a := config.Agent{
		Command:   []string{script(t, "sleep 60 &\nsleep 60")},
		PromptVia: "stdin",
		Timeout:   config.Duration(300 * time.Millisecond),
	}
	done := make(chan Result, 1)
	go func() { done <- Run(t.Context(), a, "", t.TempDir()) }()
	select {
	case res := <-done:
		if res.Err == nil || !strings.Contains(res.Err.Error(), "timed out after") {
			t.Fatalf("Run() err = %v, want timeout error", res.Err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after timeout; process group not killed")
	}
}

func TestRunKillsSurvivingDescendantsOnSuccess(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "sentinel")
	// The leader backgrounds a child that writes a sentinel after a delay, then
	// exits SUCCESSFULLY. The child redirects its inherited pipes to /dev/null so it
	// does not hold the output pipe open (the case the next test covers). Run must
	// SIGKILL the whole process group on the success exit path, so the child never
	// survives to edit the repo after the leader is gone.
	a := config.Agent{
		Command:   []string{script(t, "{ sleep 1; echo edited > '"+sentinel+"'; } >/dev/null 2>&1 &\nexit 0")},
		PromptVia: "stdin",
		Timeout:   config.Duration(time.Minute),
	}
	res := Run(t.Context(), a, "", dir)
	if res.Err != nil {
		t.Fatalf("Run() err = %v, want a clean success", res.Err)
	}
	// Wait past the child's delay: if the group was killed the sentinel never
	// appears; if the child survived it has had time to write it.
	time.Sleep(1500 * time.Millisecond)
	if _, err := os.Stat(sentinel); err == nil {
		t.Fatal("background descendant survived a successful leader exit and edited the working tree")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// The same interleaving as above, with the child's `>/dev/null 2>&1` removed --
// which is what a backgrounded child actually does unless it explicitly
// redirects. It inherits stdout, so no EOF arrives on its own even though the
// leader wrote its whole reply and exited 0. Run must report that success and
// keep the reply: treating a drain timeout as a failure throws away a complete
// review or fix. And because the process group is killed the instant the leader
// exits, the pipe closes there: the capture is COMPLETE and carries no
// cut-short note.
func TestRunSucceedsWhenDescendantHoldsOutputPipe(t *testing.T) {
	a := config.Agent{
		Command:   []string{script(t, "sleep 30 &\necho '<fix>ok</fix>'\nexit 0")},
		PromptVia: "stdin",
		Timeout:   config.Duration(time.Minute),
	}
	done := make(chan Result, 1)
	go func() { done <- Run(t.Context(), a, "", t.TempDir()) }()
	select {
	case res := <-done:
		if res.Err != nil {
			t.Fatalf("Run() err = %v, want success: the leader exited 0, only a descendant held the pipe", res.Err)
		}
		if !strings.Contains(res.Stdout, "<fix>ok</fix>") {
			t.Errorf("stdout = %q, want the leader's reply preserved", res.Stdout)
		}
		if strings.Contains(res.Stderr, "held the output pipe open") {
			t.Errorf("stderr = %q: a descendant inside the process group is killed at the leader's exit, so the capture is not cut short", res.Stderr)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Run hung with a descendant on the output pipe")
	}
}

// A descendant that ESCAPED the process group (setpgrp/setsid) survives the kill
// and can hold the output pipe open for as long as it lives. The drain grace is
// what stops that from wedging the run; the leader still exited 0, so Run must
// report the success, keep what it captured, and record that the capture was cut
// where fixpoint stopped draining.
func TestRunSucceedsWhenDetachedDescendantHoldsOutputPipe(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl unavailable to spawn a detached pipe-holder")
	}
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	t.Cleanup(func() { reapDetachedChild(t, pidFile) })
	// The leader waits for the holder to publish its PID before replying and
	// exiting: the PID is written after setpgrp, so by then the holder really has
	// escaped the group the exit is about to have killed.
	a := config.Agent{
		Command: []string{script(t, "perl -e 'setpgrp(0,0); open(F,\">\",$ARGV[0]) or die; print F $$; close F; sleep 60' "+
			pidFile+" &\nn=0\nwhile [ ! -s '"+pidFile+"' ] && [ $n -lt 5 ]; do sleep 1; n=$((n+1)); done\necho '<fix>ok</fix>'\nexit 0")},
		PromptVia: "stdin",
		Timeout:   config.Duration(time.Minute),
	}
	done := make(chan Result, 1)
	go func() { done <- Run(t.Context(), a, "", t.TempDir()) }()
	select {
	case res := <-done:
		if res.Err != nil {
			t.Fatalf("Run() err = %v, want success: the leader exited 0, only a detached descendant held the pipe", res.Err)
		}
		if !strings.Contains(res.Stdout, "<fix>ok</fix>") {
			t.Errorf("stdout = %q, want the leader's reply preserved", res.Stdout)
		}
		if !strings.Contains(res.Stderr, "held the output pipe open") {
			t.Errorf("stderr = %q, want a note recording the cut-short capture", res.Stderr)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Run hung well past the drain grace with a detached descendant on the output pipe")
	}
}

// The deadline can expire AFTER the leader has exited 0, while Run is still
// tearing down: a detached descendant on the output pipe holds the drain for the
// full grace, and a timeout shorter than that fires in the middle of it. The
// leader succeeded and its whole reply is captured, so the invocation is a
// success -- reclassifying it as a timeout would discard a complete review or
// fix, the same damage the drain-timeout handling exists to prevent.
func TestRunKeepsSuccessWhenDeadlineExpiresDuringTeardown(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl unavailable to spawn a detached pipe-holder")
	}
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	t.Cleanup(func() { reapDetachedChild(t, pidFile) })
	// The leader replies and exits well inside the timeout; the drain that follows
	// runs past it, because the escaped holder keeps a write end open.
	a := config.Agent{
		Command: []string{script(t, "perl -e 'setpgrp(0,0); open(F,\">\",$ARGV[0]) or die; print F $$; close F; sleep 60' "+
			pidFile+" &\nn=0\nwhile [ ! -s '"+pidFile+"' ] && [ $n -lt 5 ]; do sleep 1; n=$((n+1)); done\necho '<fix>ok</fix>'\nexit 0")},
		PromptVia: "stdin",
		Timeout:   config.Duration(2 * time.Second),
	}
	done := make(chan Result, 1)
	go func() { done <- Run(t.Context(), a, "", t.TempDir()) }()
	select {
	case res := <-done:
		if res.Err != nil {
			t.Fatalf("Run() err = %v, want success: the leader exited 0 before the deadline, which expired during the drain", res.Err)
		}
		if !strings.Contains(res.Stdout, "<fix>ok</fix>") {
			t.Errorf("stdout = %q, want the leader's reply preserved", res.Stdout)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Run hung with a detached descendant on the output pipe past the deadline")
	}
}

// The process group must be killed on the LEADER's exit, not once the output
// pipes have drained. A child that inherited the pipe keeps it open, so draining
// first would leave the child running for the whole drain -- long enough to edit
// a file after the invocation that spawned it is already recorded as finished,
// which is how an edit no reviewer saw and no check verified reaches the round
// commit.
func TestRunKillsPipeHoldingChildBeforeItCanEditTheTree(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "edited-during-drain")
	// The child holds stdout open (no redirection) and edits the tree a second
	// later; the leader exits 0 at once. Killing the group only after the drain
	// gives it that second.
	a := config.Agent{
		Command:   []string{script(t, "{ sleep 1; echo edited > '"+sentinel+"'; } &\necho done\nexit 0")},
		PromptVia: "stdin",
		Timeout:   config.Duration(time.Minute),
	}
	res := Run(t.Context(), a, "", dir)
	if res.Err != nil {
		t.Fatalf("Run() err = %v, want a clean success", res.Err)
	}
	// Past the child's own delay: if it outlived the leader it has written by now.
	time.Sleep(1500 * time.Millisecond)
	if _, err := os.Stat(sentinel); err == nil {
		t.Fatal("a pipe-holding child survived its leader's exit and edited the working tree")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// The same ordering requirement as above, reached through STDIN instead of the
// output pipes. prompt_via: stdin is the default, so the prompt arrives as an
// ordinary reader; if exec owned that pipe it would run the copy itself and
// cmd.Wait would block on it for the whole WaitDelay whenever a descendant
// inherited fd 0 and the leader exited without consuming the prompt -- giving
// that descendant the same window to edit the tree after the leader is recorded
// as finished. The prompt here is larger than the pipe buffer so the feed cannot
// simply be buffered and forgotten.
func TestRunKillsStdinHoldingChildBeforeItCanEditTheTree(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "edited-during-stdin-feed")
	// The child keeps the prompt pipe on fd 0 -- via fd 3, since a non-interactive
	// shell gives a background job /dev/null for stdin before explicit
	// redirections, where an agent's own spawned child (an MCP server, a language
	// server) simply inherits it -- and redirects its output, so ONLY stdin can hold
	// things up. The leader never reads the prompt and exits 0 at once.
	a := config.Agent{
		Command:   []string{script(t, "exec 3<&0\n{ sleep 1; echo edited > '"+sentinel+"'; } <&3 >/dev/null 2>&1 &\nexit 0")},
		PromptVia: "stdin",
		Timeout:   config.Duration(time.Minute),
	}
	res := Run(t.Context(), a, strings.Repeat("prompt ", 40000), dir)
	if res.Err != nil {
		t.Fatalf("Run() err = %v, want a clean success", res.Err)
	}
	if strings.Contains(res.Stderr, "held the output pipe open") {
		t.Errorf("stderr = %q: no descendant held an OUTPUT pipe, so nothing was cut short", res.Stderr)
	}
	// Past the child's own delay: if the stdin feed postponed the group kill it has
	// written by now.
	time.Sleep(1500 * time.Millisecond)
	if _, err := os.Stat(sentinel); err == nil {
		t.Fatal("a stdin-holding child survived its leader's exit and edited the working tree")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRunContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	dir := t.TempDir()
	// The child signals readiness by touching a sentinel in its cwd; the test
	// cancels only after seeing it, so cancellation always hits a live process
	// rather than racing process startup.
	a := config.Agent{
		Command:   []string{script(t, "touch ready\nsleep 60")},
		PromptVia: "stdin",
		Timeout:   config.Duration(time.Minute),
	}
	done := make(chan Result, 1)
	go func() { done <- Run(ctx, a, "", dir) }()
	ready := filepath.Join(dir, "ready")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child process never signaled readiness")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case res := <-done:
		if res.Err == nil {
			t.Fatal("Run() err = nil, want cancellation error")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

func TestResultRaw(t *testing.T) {
	r := Result{Stdout: "sout", Stderr: "serr", Duration: time.Second}
	raw := r.Raw([]string{"claude", "-p"})
	for _, want := range []string{"$ claude -p", "--- stdout ---\nsout", "--- stderr ---\nserr"} {
		if !strings.Contains(raw, want) {
			t.Errorf("Raw() missing %q:\n%s", want, raw)
		}
	}
	noErr := Result{Stdout: "x"}.Raw([]string{"a"})
	if strings.Contains(noErr, "error:") {
		t.Errorf("Raw() without error should not print an error line:\n%s", noErr)
	}

	// The positive branch: an error prints an "error:" line carrying the message.
	withErr := Result{Stdout: "x", Err: errors.New("timed out after 5m")}.Raw([]string{"a"})
	if !strings.Contains(withErr, "error: timed out after 5m") {
		t.Errorf("Raw() with an error should print the error line:\n%s", withErr)
	}
	// And that line is redacted like the rest of the output: a credential-shaped
	// error message must not survive verbatim.
	credErr := Result{Stdout: "x", Err: errors.New("auth failed: token=abcd1234efgh5678")}.Raw([]string{"a"})
	if strings.Contains(credErr, "abcd1234efgh5678") {
		t.Errorf("Raw() must redact a credential-shaped error message:\n%s", credErr)
	}
	if !strings.Contains(credErr, "error:") || !strings.Contains(credErr, redactionMask) {
		t.Errorf("Raw() should keep the masked error line:\n%s", credErr)
	}
}
