// Package testfixture holds the integration-test scaffolding shared by the
// main and orchestrator test suites: the scripted mock-agent protocol, the
// canned response builders, and the throwaway git repository. Keeping the
// mock-agent contract (count file, resp-N, side-N.sh, stdin drain) in one place
// means a change to the protocol lands in both suites at once instead of
// silently diverging. The count file gave way to atomic per-call ordinal-claim
// directories so parallel reviewer lenses cannot collide on one ordinal.
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
// The ordinal is claimed with `mkdir n-<k>` in a loop: mkdir is atomic on POSIX
// and fails if the directory already exists, so two mock processes running in
// parallel (the orchestrator fans reviewer lenses out concurrently against one
// shared respDir) can never both take the same ordinal -- a cat+printf on a
// shared count file was a racy read-modify-write that lost invocations and served
// one resp-N twice. Invocations counts these claim directories.
func WriteMockScript(t *testing.T, respDir string) string {
	t.Helper()
	script := filepath.Join(respDir, "mock.sh")
	body := "#!/bin/sh\n" +
		"dir='" + respDir + "'\n" +
		"n=0\n" +
		"while :; do\n" +
		"  n=$((n+1))\n" +
		"  if mkdir \"$dir/n-$n\" 2>/dev/null; then break; fi\n" +
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
// the ordinal-claim directories (n-<k>) it creates atomically per call.
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
		if e.IsDir() && strings.HasPrefix(e.Name(), "n-") {
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
func GitRepo(t *testing.T) string {
	t.Helper()
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
