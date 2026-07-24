package agent

import (
	"context"
	"errors"
	"os"
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
	// exits SUCCESSFULLY. The child redirects its inherited pipes to /dev/null so
	// it does not hold cmd.Run's stdout open (which would only trip the WaitDelay
	// path). Run must SIGKILL the whole process group on the success exit path, so
	// the child never survives to edit the repo after the leader is gone.
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
