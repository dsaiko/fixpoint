package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// makeCommand builds a `make` invocation against the project root. The
// environment is filtered rather than inherited: make imports environment
// variables as make variables, so an ambient POST or PR would otherwise decide
// what the "default" invocation expands to.
func makeCommand(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not installed; the Makefile targets cannot be run here")
	}
	cmd := exec.Command("make", append([]string{"--no-print-directory"}, args...)...)
	cmd.Dir = "../.."
	cmd.Env = nil
	for _, kv := range os.Environ() {
		switch strings.SplitN(kv, "=", 2)[0] {
		case "POST", "PR", "MAKEFLAGS", "MFLAGS":
			continue
		}
		cmd.Env = append(cmd.Env, kv)
	}
	return cmd
}

// makeDryRun expands one target's recipe without running any of it, so the test
// reads the argv the operator would get.
func makeDryRun(t *testing.T, args ...string) string {
	t.Helper()
	cmd := makeCommand(t, append([]string{"-n"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// fixpointCommand returns the fields of the single line that invokes the built
// binary. Matching the whole line and requiring exactly one keeps the assertions
// below from passing on the usage guard or on a prerequisite's recipe.
func fixpointCommand(t *testing.T, recipe string) []string {
	t.Helper()
	var found []string
	for _, line := range strings.Split(recipe, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "./fixpoint ") {
			found = append(found, line)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one ./fixpoint invocation, got %d:\n%s", len(found), recipe)
	}
	return strings.Fields(found[0])
}

// `run` was the documented entry point for the whole review -> fix -> verify ->
// commit cycle. What replaced it only prints an explanation, so the recipe has to
// end non-zero: a wrapper, alias or CI step still calling it must see a failure
// rather than read a no-op as a completed run. This is the one target run for
// real rather than expanded, because the exit status is the whole point of it --
// and it is safe to run, since it touches nothing but stdout.
func TestRunTargetExitsNonZero(t *testing.T) {
	out, err := makeCommand(t, "run").CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("make run succeeded (err=%v); a no-op must not report a completed run:\n%s", err, out)
	}
	if code := exit.ExitCode(); code != 2 {
		t.Errorf("make run exited %d, want 2 -- the usage-error code the PR= guards use:\n%s", code, out)
	}
	// The exit status is only half of it: the operator still has to be told what
	// to run instead, so the explanation and the target listing must survive too.
	for _, want := range []string{"There is no 'make run'", "fix-code:", "review-pr:"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("make run no longer prints %q:\n%s", want, out)
		}
	}
}

func hasArg(argv []string, want string) bool {
	for _, arg := range argv {
		if arg == want {
			return true
		}
	}
	return false
}

// -post is the flag every reply path is gated on, and replies go out under the
// operator's own identity. The Makefile makes it opt-in through POST=1, so both
// halves of that conditional are load-bearing: dropping it would publish by
// default, and misplacing it would silently swallow replies that were asked for.
func TestFixPRPostsOnlyWhenAskedTo(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		argv := fixpointCommand(t, makeDryRun(t, "fix-pr", "PR=170"))
		if hasArg(argv, "-post") {
			t.Errorf("make fix-pr PR=170 passes -post; posting must stay opt-in: %v", argv)
		}
	})
	t.Run("POST=1", func(t *testing.T) {
		argv := fixpointCommand(t, makeDryRun(t, "fix-pr", "PR=170", "POST=1"))
		if !hasArg(argv, "-post") {
			t.Errorf("make fix-pr PR=170 POST=1 omits -post; the replies would never be sent: %v", argv)
		}
	})
	// The trust flags travel with the same recipe line, and the narrow one is
	// what keeps a pull request's own content from being trusted.
	argv := fixpointCommand(t, makeDryRun(t, "fix-pr", "PR=170"))
	for _, want := range []string{"fix-pr", "-pr", "170", "--allow-untrusted-fix", "--trusted-bundle"} {
		if !hasArg(argv, want) {
			t.Errorf("make fix-pr PR=170 lost %q: %v", want, argv)
		}
	}
	if hasArg(argv, "--trusted-target") {
		t.Errorf("make fix-pr PR=170 asserts --trusted-target over an externally authored tree: %v", argv)
	}
}
