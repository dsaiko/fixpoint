package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// makeDryRun expands one target's recipe without running any of it, so the test
// reads the argv the operator would get. The environment is filtered rather than
// inherited: make imports environment variables as make variables, so an ambient
// POST or PR would otherwise decide what the "default" invocation expands to.
func makeDryRun(t *testing.T, args ...string) string {
	t.Helper()
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not installed; the Makefile targets cannot be expanded here")
	}
	cmd := exec.Command("make", append([]string{"-n", "--no-print-directory"}, args...)...)
	cmd.Dir = "../.."
	cmd.Env = nil
	for _, kv := range os.Environ() {
		switch strings.SplitN(kv, "=", 2)[0] {
		case "POST", "PR", "MAKEFLAGS", "MFLAGS":
			continue
		}
		cmd.Env = append(cmd.Env, kv)
	}
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
