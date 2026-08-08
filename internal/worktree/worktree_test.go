package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/testfixture"
)

// The isolation is the point: an edit in one worktree must be invisible in the
// main checkout and in every other worktree, or parallel coder sessions would be
// editing each other's work and the patches would not compose.
func TestAWorktreeIsolatesItsEdits(t *testing.T) {
	repo := testfixture.GitRepo(t)
	parent := t.TempDir()
	head := strings.TrimSpace(testfixture.GitRun(t, repo, "rev-parse", "HEAD"))

	a, err := Add(t.Context(), repo, parent, head, "a")
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}
	defer a.Close(t.Context())
	b, err := Add(t.Context(), repo, parent, head, "b")
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}
	defer b.Close(t.Context())

	write(t, a.Dir, "main.go", "package main\n\n// fixed in A\n")
	write(t, b.Dir, "other.go", "package main\n\n// added in B\n")

	if got := read(t, repo, "main.go"); strings.Contains(got, "fixed in A") {
		t.Error("an edit in a worktree reached the main checkout")
	}
	if got := read(t, b.Dir, "main.go"); strings.Contains(got, "fixed in A") {
		t.Error("an edit in one worktree reached another")
	}
	if _, err := os.Stat(filepath.Join(repo, "other.go")); err == nil {
		t.Error("a file added in a worktree appeared in the main checkout")
	}
}

// A patch has to carry NEW files too. A coder that adds a file is doing the same
// job as one that edits it, and a patch that omitted new files would apply
// cleanly and leave the fix half-made -- which the gate might even pass.
func TestAPatchCarriesEditsAndNewFiles(t *testing.T) {
	repo := testfixture.GitRepo(t)
	head := strings.TrimSpace(testfixture.GitRun(t, repo, "rev-parse", "HEAD"))
	w, err := Add(t.Context(), repo, t.TempDir(), head, "w")
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}
	defer w.Close(t.Context())

	write(t, w.Dir, "main.go", "package main\n\n// edited\n")
	write(t, w.Dir, "added.go", "package main\n\n// new\n")

	patch, err := w.Patch(t.Context())
	if err != nil {
		t.Fatalf("Patch() = %v", err)
	}
	for _, want := range []string{"main.go", "// edited", "added.go", "// new"} {
		if !strings.Contains(patch, want) {
			t.Errorf("patch is missing %q:\n%s", want, patch)
		}
	}
	// And it applies to the tree it was taken against.
	testfixture.GitRun(t, repo, "apply", "--check", writeTemp(t, patch))
}

// A session that changed nothing produces an empty patch, not an error: that is
// an ordinary outcome -- the coder rejected the issue.
func TestASessionThatChangedNothingYieldsAnEmptyPatch(t *testing.T) {
	repo := testfixture.GitRepo(t)
	head := strings.TrimSpace(testfixture.GitRun(t, repo, "rev-parse", "HEAD"))
	w, err := Add(t.Context(), repo, t.TempDir(), head, "w")
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}
	defer w.Close(t.Context())

	patch, err := w.Patch(t.Context())
	if err != nil {
		t.Fatalf("Patch() = %v", err)
	}
	if strings.TrimSpace(patch) != "" {
		t.Errorf("patch = %q, want empty", patch)
	}
}

// The run's own artifacts live inside the target when logs.dir is there, and they
// must never reach a patch: applying one would commit the run's logs as though
// they were the fix.
func TestAPatchExcludesWhatTheCallerNames(t *testing.T) {
	repo := testfixture.GitRepo(t)
	head := strings.TrimSpace(testfixture.GitRun(t, repo, "rev-parse", "HEAD"))
	w, err := Add(t.Context(), repo, t.TempDir(), head, "w")
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}
	defer w.Close(t.Context())

	if err := os.MkdirAll(filepath.Join(w.Dir, ".fixpoint", "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, w.Dir, filepath.Join(".fixpoint", "run", "step.md"), "an artifact\n")
	write(t, w.Dir, "main.go", "package main\n\n// the actual fix\n")

	patch, err := w.Patch(t.Context(), ".fixpoint")
	if err != nil {
		t.Fatalf("Patch() = %v", err)
	}
	if strings.Contains(patch, ".fixpoint") {
		t.Errorf("the run's own artifacts reached the patch:\n%s", patch)
	}
	if !strings.Contains(patch, "the actual fix") {
		t.Errorf("the fix itself is missing from the patch:\n%s", patch)
	}
}

// Close must leave neither the directory nor git metadata behind: a run makes one
// worktree per fix, so a leak is per-fix rather than per-run.
func TestCloseRemovesTheWorktree(t *testing.T) {
	repo := testfixture.GitRepo(t)
	head := strings.TrimSpace(testfixture.GitRun(t, repo, "rev-parse", "HEAD"))
	w, err := Add(t.Context(), repo, t.TempDir(), head, "w")
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}
	if err := w.Close(t.Context()); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if _, err := os.Stat(w.Dir); !os.IsNotExist(err) {
		t.Errorf("worktree directory survived Close(): %v", err)
	}
	if out := testfixture.GitRun(t, repo, "worktree", "list"); strings.Contains(out, w.Dir) {
		t.Errorf("git still lists the worktree:\n%s", out)
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "patch.diff")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}
