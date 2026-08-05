// Package testfixture holds the integration-test scaffolding shared by the
// main and orchestrator test suites: the scripted mock-agent protocol, the
// canned response builders, and the throwaway git repository. Keeping the
// mock-agent contract (count file, resp-N, side-N.sh, stdin drain) in one place
// means a change to the protocol lands in both suites at once instead of
// silently diverging. The count file gave way to atomic per-call ordinal claims so
// parallel reviewer lenses cannot collide on one ordinal.
// Per-suite config wiring (a *config.Config vs a YAML file)
// stays in each suite; only the reusable machinery lives here.
package testfixture

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/model"
)

// WriteMockScript writes the mock agent into respDir and returns its path. The
// mock replays canned responses in invocation order (resp-1, resp-2, ...),
// claiming a fresh invocation ordinal each call and running an optional
// side-effect script (side-N.sh) before answering. It drains stdin so a
// prompt-piping caller does not block.
//
// The ordinal is claimed with a NOCLOBBER redirect -- `( set -C; : > n-<k> )` in a
// loop -- because the orchestrator fans reviewer lenses out concurrently against one
// shared respDir, so two mock processes must never take the same ordinal. A shared
// count file came first and was a racy read-modify-write that lost invocations and
// served one resp-N twice; `mkdir n-<k>` replaced it and was correct in theory.
//
// It is a shell BUILTIN on purpose. `mkdir n-<k>` broke on Ubuntu 26.04, which ships
// uutils coreutils (the Rust rewrite) as /usr/bin/mkdir: its mkdir checks for the
// path and then creates it, so two concurrent invocations BOTH exit 0 (~78% of races
// on one machine) while only one directory appears. It reports EEXIST correctly when
// run sequentially, which is what made this so quiet. Two reviewers then shared
// ordinal 1 and the coder was served a reviewer's canned response, failing with "no
// <fix> block found in agent output" -- a test failure that looked like a bug in the
// code under test. noclobber is O_EXCL inside the shell itself, so no external
// coreutils implementation can weaken it. Go's own os.Mkdir is unaffected (verified
// atomic on that machine), so production code that claims a directory is fine.
//
// Invocations counts these claim files.
func WriteMockScript(t *testing.T, respDir string) string {
	t.Helper()
	script := filepath.Join(respDir, "mock.sh")
	body := "#!/bin/sh\n" +
		"dir='" + respDir + "'\n" +
		"n=0\n" +
		"while :; do\n" +
		"  n=$((n+1))\n" +
		// The subshell scopes `set -C` and lets the losing racer's "File exists"
		// message be dropped: dash writes it even when the redirect itself is
		// silenced, and it would otherwise pollute every mock agent's stderr.
		"  if ( set -C; : > \"$dir/n-$n\" ) 2>/dev/null; then break; fi\n" +
		"done\n" +
		"cat > /dev/null\n" + // drain the prompt from stdin
		"if [ -x \"$dir/side-$n.sh\" ]; then \"$dir/side-$n.sh\"; fi\n" +
		"cat \"$dir/resp-$n\"\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return script
}

// Respond registers the mock agent's n-th response.
func Respond(t *testing.T, respDir string, n int, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(respDir, fmt.Sprintf("resp-%d", n)), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// EditRepoOn makes the mock agent's n-th invocation edit repo/main.go before
// answering, simulating a coder fix.
func EditRepoOn(t *testing.T, respDir, repo string, n int) {
	t.Helper()
	body := fmt.Sprintf("#!/bin/sh\necho 'fix %d' >> '%s'\n", n, filepath.Join(repo, "main.go"))
	WriteSide(t, respDir, n, body)
}

// WriteSide installs a raw side-effect script (including its #! line) for the
// mock agent's n-th invocation, letting a test script arbitrary repo mutations.
func WriteSide(t *testing.T, respDir string, n int, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(respDir, fmt.Sprintf("side-%d.sh", n)), []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
}

// Invocations reports how many times the mock agent has been called by counting
// the ordinal-claim files (n-<k>) it creates atomically per call. A count lower
// than the number of agent invocations the run made would mean the claim raced and
// two processes shared an ordinal -- see WriteMockScript.
func Invocations(t *testing.T, respDir string) int {
	t.Helper()
	entries, err := os.ReadDir(respDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "n-") {
			n++
		}
	}
	return n
}

// ReviewResponse renders a reviewer's contract-shaped output for the given
// findings (an explicit empty slice when none), wrapped in the <review>
// envelope the extractor looks for.
func ReviewResponse(t *testing.T, findings ...model.ReviewFinding) string {
	t.Helper()
	if findings == nil {
		findings = []model.ReviewFinding{} // the contract requires an explicit []
	}
	b, err := json.Marshal(model.ReviewOutput{Findings: findings})
	if err != nil {
		t.Fatal(err)
	}
	return "I reviewed the code.\n<review>" + string(b) + "</review>\n"
}

// FixResponse renders a coder's contract-shaped output for the given results,
// wrapped in the <fix> envelope the extractor looks for.
func FixResponse(t *testing.T, results ...model.FixResult) string {
	t.Helper()
	b, err := json.Marshal(model.FixOutput{Results: results})
	if err != nil {
		t.Fatal(err)
	}
	return "Done.\n<fix>" + string(b) + "</fix>\n"
}

// AFinding builds a minimal contract-valid review finding with the given title.
func AFinding(title string) model.ReviewFinding {
	return model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 1, Title: title}
}

// GitRepo initializes a throwaway git repository with one committed main.go and
// returns its path.
//
// The operator scopes are pinned empty for the test process, so what the machine
// running the tests has in ~/.gitconfig or /etc/gitconfig cannot change a result.
// That is load-bearing for the external-filter gate (guardActivatableFilters):
// `git lfs install` writes filter.lfs.* into ~/.gitconfig, and without this a
// developer with git-lfs installed would get a different verdict on every pr-mode
// test than CI does. Tests that need an operator-scope setting point
// GIT_CONFIG_GLOBAL at their own file instead.
func GitRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	dir := t.TempDir()
	GitRun(t, dir, "init", "-q")
	GitRun(t, dir, "config", "user.email", "test@example.com")
	GitRun(t, dir, "config", "user.name", "test")
	GitRun(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	GitRun(t, dir, "add", ".")
	GitRun(t, dir, "commit", "-q", "-m", "initial")
	return dir
}

// GitRun runs git in dir, failing the test on error, and returns its combined
// output.
func GitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return string(out)
}
