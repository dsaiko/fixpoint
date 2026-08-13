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

func gitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// The scaffold claims the directory, hardens the repository, and commits
// exactly the bootstrap files -- the whole §5.1 contract in one pass.
func TestScaffold(t *testing.T) {
	gitAvailable(t)
	out := filepath.Join(t.TempDir(), "prsi")
	col := target.New(config.Target{Mode: "directory", Path: out})

	files := map[string][]byte{
		"DESIGN.md":  []byte("# design\n"),
		"PLAN.md":    []byte("# plan\n"),
		"PLAN.json":  []byte("{}\n"),
		".gitignore": []byte(GitignoreContent([]string{"dist/"})),
	}
	sha, err := Scaffold(t.Context(), col, out, files, "fixpoint: initialize", "Fixpoint-Phase: bootstrap")
	if err != nil {
		t.Fatalf("Scaffold() = %v", err)
	}
	if sha == "" {
		t.Fatal("no bootstrap SHA")
	}
	if got := gitOut(t, out, "show", "--name-only", "--format=", "HEAD"); !strings.Contains(got, "DESIGN.md") || !strings.Contains(got, ".gitignore") {
		t.Errorf("bootstrap commit files:\n%s", got)
	}
	if branch := gitOut(t, out, "branch", "--show-current"); branch != "main" {
		t.Errorf("branch = %q, want main", branch)
	}
	if hooks := gitOut(t, out, "config", "core.hooksPath"); hooks != ".git/fixpoint-hooks" {
		t.Errorf("core.hooksPath = %q", hooks)
	}
	if who := gitOut(t, out, "config", "user.name"); who != "fixpoint" {
		t.Errorf("user.name = %q", who)
	}
	for _, f := range []string{"exclude", "attributes"} {
		b, err := os.ReadFile(filepath.Join(out, ".git", "info", f))
		if err != nil || len(b) != 0 {
			t.Errorf(".git/info/%s: err=%v len=%d, want an empty known baseline", f, err, len(b))
		}
	}
	// The claim is exclusive: a second scaffold into the same path refuses.
	if _, err := Scaffold(t.Context(), col, out, files, "x", "y"); err == nil {
		t.Error("scaffolding over an existing directory did not refuse")
	}
}

// CommitExact rebuilds the index from the computed path set: pre-staged junk
// never lands, deletions stage, and --allow-empty carries an outcome marker.
func TestCommitExact(t *testing.T) {
	gitAvailable(t)
	out := filepath.Join(t.TempDir(), "repo")
	col := target.New(config.Target{Mode: "directory", Path: out})
	files := map[string][]byte{"a.txt": []byte("a\n"), "b.txt": []byte("b\n")}
	if _, err := Scaffold(t.Context(), col, out, files, "init", "-"); err != nil {
		t.Fatal(err)
	}

	// A session edits both files and stages one of them plus a stray; fixpoint
	// commits ONLY the computed set {a.txt}.
	os.WriteFile(filepath.Join(out, "a.txt"), []byte("a2\n"), 0o644)
	os.WriteFile(filepath.Join(out, "b.txt"), []byte("b2\n"), 0o644)
	os.WriteFile(filepath.Join(out, "stray.txt"), []byte("x\n"), 0o644)
	gitOut(t, out, "add", "b.txt", "stray.txt") // the untrusted index
	sha, err := col.CommitExact(t.Context(), "fixpoint: T01", "Fixpoint-Task: T01", []string{"a.txt"}, false)
	if err != nil || sha == "" {
		t.Fatalf("CommitExact() = %q, %v", sha, err)
	}
	if got := gitOut(t, out, "show", "--name-only", "--format=", "HEAD"); strings.TrimSpace(got) != "a.txt" {
		t.Errorf("committed %q, want exactly a.txt", got)
	}

	// A deletion in the computed set stages as a deletion.
	os.Remove(filepath.Join(out, "b.txt"))
	os.WriteFile(filepath.Join(out, "stray.txt"), []byte("x2\n"), 0o644) // still not ours
	if _, err := col.CommitExact(t.Context(), "fixpoint: T02", "Fixpoint-Task: T02", []string{"b.txt"}, false); err != nil {
		t.Fatalf("deletion commit: %v", err)
	}
	if ls := gitOut(t, out, "ls-files", "b.txt"); ls != "" {
		t.Errorf("b.txt still tracked after a deletion commit: %q", ls)
	}

	// An outcome marker: empty, tool-authored, no tree change (§5.4).
	before := gitOut(t, out, "rev-parse", "HEAD^{tree}")
	sha, err = col.CommitExact(t.Context(), "fixpoint: T03 — skipped [blocked]",
		"Fixpoint-Task: T03\nFixpoint-Outcome: blocked", nil, true)
	if err != nil || sha == "" {
		t.Fatalf("marker commit: %q, %v", sha, err)
	}
	if after := gitOut(t, out, "rev-parse", "HEAD^{tree}"); after != before {
		t.Error("an outcome marker changed the tree")
	}
	if msg := gitOut(t, out, "log", "-1", "--format=%B"); !strings.Contains(msg, "Fixpoint-Outcome: blocked") {
		t.Errorf("marker trailers missing: %q", msg)
	}
}

func TestGitignoreContent(t *testing.T) {
	got := GitignoreContent([]string{"node_modules/", " ", "dist/"})
	for _, want := range []string{".fixpoint/\n", "node_modules/\n", "dist/\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("gitignore lacks %q:\n%s", want, got)
		}
	}
}
