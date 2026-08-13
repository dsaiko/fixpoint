package implement

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/target"
)

func collectorFor(t *testing.T, out string) *target.Collector {
	t.Helper()
	return target.New(config.Target{Mode: "directory", Path: out})
}

func stateRepo(t *testing.T) string {
	t.Helper()
	gitAvailable(t)
	out := filepath.Join(t.TempDir(), "repo")
	files := map[string][]byte{"a.txt": []byte("a\n")}
	_, release, err := Scaffold(t.Context(), collectorFor(t, out), out, files, "init", "-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(release)
	return out
}

// Every invariant the snapshot covers must read as a diff when it moves; a
// task commit on the working branch must not (§5.2 step 4).
func TestRepoStateDiff(t *testing.T) {
	dir := stateRepo(t)
	base, err := SnapshotRepoState(t.Context(), dir, "main")
	if err != nil {
		t.Fatalf("SnapshotRepoState() = %v", err)
	}
	if got := base.Diff(base); len(got) != 0 {
		t.Fatalf("self-diff = %v", got)
	}

	// An ordinary commit on main is invisible to the invariant.
	write(t, dir, "a.txt", "a2\n")
	if _, err := collectorFor(t, dir).CommitExact(t.Context(), "c", "-", []string{"a.txt"}, false); err != nil {
		t.Fatal(err)
	}
	after, err := SnapshotRepoState(t.Context(), dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if got := after.Diff(base); len(got) != 0 {
		t.Fatalf("a task commit tripped the invariant: %v", got)
	}

	mutations := []struct {
		name string
		do   func()
		want string
	}{
		{"config edit", func() {
			cmd := exec.Command("git", "config", "user.name", "attacker")
			cmd.Dir = dir
			_ = cmd.Run()
		}, ".git/config"},
		{"hook dropped", func() {
			write(t, dir, ".git/fixpoint-hooks/pre-commit", "#!/bin/sh\n")
		}, "hooks directory"},
		{"foreign ref", func() {
			cmd := exec.Command("git", "branch", "smuggle")
			cmd.Dir = dir
			_ = cmd.Run()
		}, "ref list"},
		{"nested repo", func() {
			if err := os.MkdirAll(filepath.Join(dir, "vendor", ".git"), 0o750); err != nil {
				t.Fatal(err)
			}
		}, "nested .git"},
		{"exclude line", func() {
			write(t, dir, ".git/info/exclude", "secret.go\n")
		}, "info/exclude"},
		{"attributes line", func() {
			write(t, dir, ".git/info/attributes", "*.go filter=x\n")
		}, "info/attributes"},
		{"alternates", func() {
			write(t, dir, ".git/objects/info/alternates", "/elsewhere\n")
		}, "alternates"},
		{"shallow marker", func() {
			write(t, dir, ".git/shallow", "0000\n")
		}, "shallow"},
	}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			pre, err := SnapshotRepoState(t.Context(), dir, "main")
			if err != nil {
				t.Fatal(err)
			}
			m.do()
			post, err := SnapshotRepoState(t.Context(), dir, "main")
			if err != nil {
				t.Fatal(err)
			}
			diff := post.Diff(pre)
			if len(diff) == 0 || !strings.Contains(strings.Join(diff, "; "), m.want) {
				t.Fatalf("diff = %v, want mention of %q", diff, m.want)
			}
		})
	}
}

// Git metadata is fixpoint's own, so a SYMLINK where a file or directory
// belongs is itself the change: following it let a session swap .git/config
// for a link to an identical copy, pass the invariant, and then edit the copy
// -- leaving the delivered repository pointed at hooks nothing here controls
// (review run 20260813-161029, i28).
func TestRepoStateRefusesSymlinkedMetadata(t *testing.T) {
	cases := []struct {
		name string
		do   func(t *testing.T, dir, external string)
		want string
	}{
		{"config swapped for a link to an identical copy", func(t *testing.T, dir, external string) {
			t.Helper()
			cfg := filepath.Join(dir, ".git", "config")
			b, err := os.ReadFile(cfg)
			if err != nil {
				t.Fatal(err)
			}
			copyPath := filepath.Join(external, "config")
			if err := os.WriteFile(copyPath, b, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(cfg); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(copyPath, cfg); err != nil {
				t.Fatal(err)
			}
		}, ".git/config"},
		{"hooks directory swapped for a link to an empty one", func(t *testing.T, dir, external string) {
			t.Helper()
			hooks := filepath.Join(dir, ".git", "fixpoint-hooks")
			empty := filepath.Join(external, "hooks")
			if err := os.MkdirAll(empty, 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(hooks); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(empty, hooks); err != nil {
				t.Fatal(err)
			}
		}, "hooks directory"},
		{"exclude swapped for a link", func(t *testing.T, dir, external string) {
			t.Helper()
			ex := filepath.Join(dir, ".git", "info", "exclude")
			outside := filepath.Join(external, "exclude")
			if err := os.WriteFile(outside, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(ex); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, ex); err != nil {
				t.Fatal(err)
			}
		}, "info/exclude"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := stateRepo(t)
			external := t.TempDir()
			base, err := SnapshotRepoState(t.Context(), dir, "main")
			if err != nil {
				t.Fatal(err)
			}
			tc.do(t, dir, external)
			after, err := SnapshotRepoState(t.Context(), dir, "main")
			if err != nil {
				t.Fatal(err)
			}
			diff := after.Diff(base)
			if len(diff) == 0 || !strings.Contains(strings.Join(diff, "; "), tc.want) {
				t.Fatalf("diff = %v, want it to name %q", diff, tc.want)
			}
		})
	}
}
