package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/model"
)

// wrapperScript writes a stand-in sandbox: it records the argv and the working
// directory it was handed, then execs the real command. A container is not
// available in a unit test, and is not what needs proving -- what needs proving
// is that fixpoint execs the WRAPPER and hands it the agent command, which is the
// whole of fixpoint's side of the contract.
func wrapperScript(t *testing.T) (path, log string) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "wrap.sh")
	log = filepath.Join(dir, "wrapped.log")
	// Its own options end at "--", the way bwrap and docker separate theirs from
	// the command they run. That separator is part of the wrapper's contract, not
	// fixpoint's: fixpoint appends the agent argv and takes no view on how the
	// wrapper finds where its own arguments stop.
	body := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
while [ "$1" != "--" ]; do shift; done
shift
exec "$@"
`, log)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path, log
}

// withSandbox appends a sandbox block to a config the fixture wrote.
func withSandbox(t *testing.T, cfgPath string, command ...string) string {
	t.Helper()
	quoted := make([]string, len(command))
	for i, c := range command {
		quoted[i] = fmt.Sprintf("%q", c)
	}
	body := readFile(t, cfgPath) + fmt.Sprintf("sandbox:\n  command: [%s]\n", strings.Join(quoted, ", "))
	out := filepath.Join(t.TempDir(), "sandboxed.yaml")
	if err := os.WriteFile(out, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return out
}

// The property the feature exists for: every agent is launched THROUGH the
// wrapper, with the agent command as its argument.
func TestSandboxWrapsEveryAgentInvocation(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t))
	wrap, log := wrapperScript(t)
	cfg := withSandbox(t, f.configFile("directory", "", "  review_only: true"), wrap, "--mode", "{{target_mode}}", "--target", "{{target}}", "--")

	var buf bytes.Buffer
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 0 {
		t.Fatalf("run() = %d; stderr:\n%s", got, buf.String())
	}
	lines := strings.Split(strings.TrimSpace(readFile(t, log)), "\n")
	if len(lines) != 1 {
		t.Fatalf("the wrapper ran %d time(s), want 1 (one reviewer invocation):\n%s", len(lines), strings.Join(lines, "\n"))
	}
	got := lines[0]
	// The reviewer is can_edit: false, so the target is handed over read-only.
	if !strings.Contains(got, "--mode ro") {
		t.Errorf("the wrapper was not told the reviewer is read-only: %q", got)
	}
	if !strings.Contains(got, "--target "+f.repo) {
		t.Errorf("the wrapper was not told which target is under review: %q", got)
	}
	// And the agent command follows the wrapper's own arguments.
	if !strings.Contains(got, f.script) {
		t.Errorf("the agent command was not passed to the wrapper: %q", got)
	}
}

// can_edit decides the mode, so a coder is handed the target writable while the
// reviewers beside it are not. That distinction is fixpoint's to make: the
// wrapper cannot know it.
func TestSandboxHandsTheCoderAWritableTarget(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t, aFinding("a real bug")))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	// No repo edit is staged: what is asserted is the argv the wrapper was handed,
	// and the coder is launched through it whether or not its round survives
	// verification.
	wrap, log := wrapperScript(t)
	cfg := withSandbox(t, f.configFile("directory", "", "  max_iterations: 1\n  clean_rounds_to_stop: 1"), wrap, "--mode", "{{target_mode}}", "--who", "{{agent}}", "--")

	var buf bytes.Buffer
	run([]string{"-config", cfg, "-trusted-target"}, &buf, &buf)

	body := readFile(t, log)
	if !strings.Contains(body, "--mode ro --who mock-rev") {
		t.Errorf("the reviewer was not confined read-only:\n%s", body)
	}
	if !strings.Contains(body, "--mode rw --who mock") {
		t.Errorf("the coder was not given a writable target:\n%s", body)
	}
}

// A wrapper binary that does not exist must be refused at startup, under its own
// name -- not reported as a missing agent CLI, which is what the check would say
// if it only ever saw the command behind the wrapper.
func TestSandboxMissingWrapperIsRefusedAtStartup(t *testing.T) {
	f := newFixture(t)
	cfg := withSandbox(t, f.configFile("directory", "", "  review_only: true"), "/nonexistent/sandbox-binary")
	var buf bytes.Buffer
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1; stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "sandbox-binary") {
		t.Errorf("the refusal does not name the missing wrapper:\n%s", buf.String())
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("the refused run still invoked %d agent(s)", got)
	}
}

// A typo in a mount placeholder would confine the wrong path. Refused before
// anything runs.
func TestSandboxUnknownPlaceholderIsRefused(t *testing.T) {
	f := newFixture(t)
	wrap, _ := wrapperScript(t)
	cfg := withSandbox(t, f.configFile("directory", "", "  review_only: true"), wrap, "{{targt}}", "--")
	var buf bytes.Buffer
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1; stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "{{targt}}") {
		t.Errorf("the refusal does not name the typo:\n%s", buf.String())
	}
}
