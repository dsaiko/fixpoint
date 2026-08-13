package implement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
)

// guardRepo scaffolds a project with the four control artifacts, as a real run
// does, and returns the directory and the bootstrap SHA.
func guardRepo(t *testing.T) (string, string) {
	t.Helper()
	gitAvailable(t)
	out := filepath.Join(t.TempDir(), "repo")
	files := map[string][]byte{
		"DESIGN.md":  []byte("# design\n"),
		"PLAN.md":    []byte("# plan\n"),
		"PLAN.json":  []byte("{}\n"),
		".gitignore": []byte(GitignoreContent([]string{"ignored/"})),
		"src.txt":    []byte("source\n"),
	}
	sha, err := Scaffold(t.Context(), collectorFor(t, out), out, files, "init", "-")
	if err != nil {
		t.Fatal(err)
	}
	return out, sha
}

// The four control artifacts are protected mechanically, not by declaration
// (§4.3): an edit is DETECTED and the bootstrap bytes RESTORED. Untested when
// it shipped (review run 20260813-124710, i54/i60).
func TestControlArtifactsChangedAndRestored(t *testing.T) {
	dir, sha := guardRepo(t)

	if changed, err := ControlArtifactsChanged(t.Context(), dir, sha); err != nil || len(changed) != 0 {
		t.Fatalf("a clean tree reported %v, %v", changed, err)
	}

	// Every one of them, including a deletion -- absence is a change too.
	write(t, dir, "DESIGN.md", "# tampered\n")
	write(t, dir, "PLAN.json", `{"tasks": []}`)
	write(t, dir, ".gitignore", "*\n")
	if err := os.Remove(filepath.Join(dir, "PLAN.md")); err != nil {
		t.Fatal(err)
	}
	// A source edit is NOT a control-artifact change: the guard must not fire
	// on ordinary work.
	write(t, dir, "src.txt", "edited by the coder\n")

	changed, err := ControlArtifactsChanged(t.Context(), dir, sha)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"DESIGN.md": true, "PLAN.md": true, "PLAN.json": true, ".gitignore": true}
	for _, c := range changed {
		if !want[c] {
			t.Errorf("guard fired on %q, which is not a control artifact", c)
		}
		delete(want, c)
	}
	for missing := range want {
		t.Errorf("an edit to %s went unnoticed", missing)
	}

	if err := RestoreControlArtifacts(t.Context(), dir, sha, changed); err != nil {
		t.Fatalf("RestoreControlArtifacts() = %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "DESIGN.md")); string(got) != "# design\n" {
		t.Errorf("DESIGN.md after restore = %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "PLAN.md")); string(got) != "# plan\n" {
		t.Errorf("a deleted control artifact was not restored: %q", got)
	}
	// The coder's own work survives the restore -- only the artifacts revert.
	if got, _ := os.ReadFile(filepath.Join(dir, "src.txt")); string(got) != "edited by the coder\n" {
		t.Errorf("the restore reverted the session's source edit: %q", got)
	}
	if changed, err := ControlArtifactsChanged(t.Context(), dir, sha); err != nil || len(changed) != 0 {
		t.Errorf("after restore the guard still reports %v, %v", changed, err)
	}
}

// The clean-clone check is the enforcement of the design's central claim --
// the committed bytes alone satisfy the gate (§7.2) -- and it shipped with no
// test at all (review run 20260813-124710, i55). The decisive case is a gate
// that passes in the WORKING TREE because of an uncommitted (ignored) file and
// must fail in the clone.
func TestCleanCheck(t *testing.T) {
	gitAvailable(t)
	dir, _ := guardRepo(t)
	scratch := t.TempDir()

	// A gate that requires needed.txt. It is committed, so the clone passes.
	write(t, dir, "needed.txt", "present\n")
	if _, err := collectorFor(t, dir).CommitExact(t.Context(), "add needed", "-", []string{"needed.txt"}, false); err != nil {
		t.Fatal(err)
	}
	vcfg := config.Verify{
		Policy:   config.VerifyMustPass,
		Timeout:  config.Duration(30_000_000_000),
		Commands: []config.VerifyCommand{{Name: "needs-file", Run: []string{"test", "-f", "needed.txt"}}},
	}
	rep, err := CleanCheck(t.Context(), dir, scratch, vcfg, nil)
	if err != nil {
		t.Fatalf("CleanCheck() = %v", err)
	}
	if !rep.Passed() {
		t.Fatalf("a committed dependency failed in the clone: %+v", rep.Results)
	}

	// Now the hidden-state case: the file the gate needs exists only in the
	// working tree, under an ignore rule. The tree would pass; the clone must
	// not -- that gap is the whole reason this check exists.
	if err := os.Remove(filepath.Join(dir, "needed.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := collectorFor(t, dir).CommitExact(t.Context(), "drop needed", "-", []string{"needed.txt"}, false); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "ignored/needed.txt", "hidden\n")
	write(t, dir, "needed.txt", "hidden in the tree only\n")
	rep, err = CleanCheck(t.Context(), dir, scratch, vcfg, nil)
	if err != nil {
		t.Fatalf("CleanCheck() = %v", err)
	}
	if rep.Passed() {
		t.Error("the clone passed on a file only the working tree has; hidden state would ship undetected")
	}
}

func TestGateWorstSumsEveryCommandTimeout(t *testing.T) {
	v := config.Verify{
		Timeout: config.Duration(10 * 60_000_000_000),
		Commands: []config.VerifyCommand{
			{Name: "build", Run: []string{"true"}},
			{Name: "vet", Run: []string{"true"}},
			{Name: "test", Run: []string{"true"}},
		},
	}
	// Three commands under a 10m per-command timeout is a 30m worst case, not
	// 10m -- the arithmetic §4.2 rule 7 depends on.
	if got := GateWorst(v); got.Minutes() != 30 {
		t.Errorf("GateWorst = %s, want 30m", got)
	}
	if got := GateWorst(config.Verify{Timeout: v.Timeout}); got != 0 {
		t.Errorf("an ungated config has a %s worst case, want 0", got)
	}
}

func TestRenderCommands(t *testing.T) {
	got := RenderCommands(config.Verify{Commands: []config.VerifyCommand{
		{Name: "build", Run: []string{"go", "build", "./..."}},
	}})
	if len(got) != 1 || got[0] != "go build ./..." {
		t.Errorf("RenderCommands = %q", got)
	}
	if !strings.HasPrefix(DigestBytes([]byte("x")), "2d711642") {
		t.Errorf("DigestBytes = %q, want the sha256 of x", DigestBytes([]byte("x")))
	}
}
