package target

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
)

// A PR-supplied `git` inside the target must not run under gh with the forge
// token. gh resolves its internal git through PATH, gitenv.Tool pins only gh
// itself, and in pr mode `gh pr checkout` has just written PR content into the
// target -- so a PATH entry there (direnv PATH_add, $PWD/node_modules/.bin) let
// the reviewed branch receive GITHUB_TOKEN. The helper, agent.PathWithout, had a
// test of its own and was never wired into Collector.runInput (review run
// 20260929-113519, i20).
//
// Driven through a real call site (prIntent's `gh pr view`) and through c.run,
// with a stub gh that prints its PATH and token and then calls `git` by name,
// exactly as the real gh does. Calling the helper would pass with the wiring
// deleted, which is the gap this test exists to close.
func TestForgeCLIRunsWithoutTargetPathEntries(t *testing.T) {
	target := t.TempDir()
	inside := filepath.Join(target, "bin")
	outside := t.TempDir()
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(dir, name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// The PR's git, first on PATH so it wins any lookup that can still see it.
	write(inside, "git", "echo PWNED-with-${GITHUB_TOKEN:-none}\n")
	write(outside, "git", "echo real-git\n")
	write(outside, "gh", "printf 'PATH=%s\\n' \"$PATH\"\n"+
		"printf 'TOKEN=%s\\n' \"${GITHUB_TOKEN:-none}\"\n"+
		"git\n")

	t.Setenv("GITHUB_TOKEN", "ghp_forge")
	sep := string(os.PathListSeparator)
	t.Setenv("PATH", inside+sep+outside)

	c := New(config.Target{Mode: config.ModePR, PR: 7, Path: target})
	c.UseGitEnv(agent.EnvWithoutCredentials(nil))

	check := func(t *testing.T, out string) {
		t.Helper()
		if strings.Contains(out, "PWNED") {
			t.Fatalf("gh ran the target's git with the forge token in reach:\n%s", out)
		}
		if !strings.Contains(out, "real-git") {
			t.Errorf("gh's own git did not resolve to the one outside the target:\n%s", out)
		}
		if !strings.Contains(out, "TOKEN=ghp_forge") {
			t.Errorf("gh lost its own forge credential:\n%s", out)
		}
		var path string
		for _, line := range strings.Split(out, "\n") {
			if v, ok := strings.CutPrefix(line, "PATH="); ok {
				path = v
			}
		}
		entries := filepath.SplitList(path)
		if slices.Contains(entries, inside) {
			t.Errorf("gh's PATH kept %q, inside the target: %q", inside, path)
		}
		if !slices.Contains(entries, outside) {
			t.Errorf("gh's PATH lost %q, outside the target: %q", outside, path)
		}
	}

	t.Run("prIntent", func(t *testing.T) {
		check(t, c.prIntent(t.Context()))
	})
	t.Run("run", func(t *testing.T) {
		out, err := c.run(t.Context(), "gh", "repo", "view")
		if err != nil {
			t.Fatal(err)
		}
		check(t, out)
	})
	// A RELATIVE entry names the target's own bin, because gh runs with cmd.Dir =
	// the target and its `git` lookup resolves `bin` there. Checked from
	// fixpoint's side it resolved against fixpoint's cwd and survived (review run
	// 20260929-125352, i5/i8).
	t.Run("relative", func(t *testing.T) {
		// fixpoint's cwd has a bin of its own, outside the target, so the entry
		// resolves from here and passes the inside-the-target check. It holds no
		// gh, so gh itself still resolves to the stub outside.
		cwd := t.TempDir()
		if err := os.MkdirAll(filepath.Join(cwd, "bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Chdir(cwd)
		t.Setenv("PATH", "bin"+sep+outside)
		// A collector of its own: UseGitEnv snapshots PATH, so c still holds the
		// absolute one.
		rc := New(config.Target{Mode: config.ModePR, PR: 7, Path: target})
		rc.UseGitEnv(agent.EnvWithoutCredentials(nil))
		for name, out := range map[string]func() string{
			"prIntent": func() string { return rc.prIntent(t.Context()) },
			"run": func() string {
				out, err := rc.run(t.Context(), "gh", "repo", "view")
				if err != nil {
					t.Fatal(err)
				}
				return out
			},
		} {
			got := out()
			check(t, got)
			for _, line := range strings.Split(got, "\n") {
				if v, ok := strings.CutPrefix(line, "PATH="); ok && slices.Contains(filepath.SplitList(v), "bin") {
					t.Errorf("%s: gh's PATH kept the relative entry bin: %q", name, v)
				}
			}
		}
	})
	// Every Collector subprocess, not only the forge CLIs: git's own children
	// (ssh, gpg, a credential helper) resolve through the same PATH.
	t.Run("git", func(t *testing.T) {
		// outside first, so the top-level lookup finds the stub that reports. The
		// PATH it is handed is still the collector's own, target entry first.
		write(outside, "git", "printf 'PATH=%s\\n' \"$PATH\"\n")
		t.Setenv("PATH", outside+sep+inside)
		out, err := c.run(t.Context(), "git", "status")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(out, "PATH=") {
			t.Fatalf("the reporting git stub did not run: %q", out)
		}
		if strings.Contains(out, inside) {
			t.Errorf("git's PATH kept an entry inside the target: %q", out)
		}
	})
}
