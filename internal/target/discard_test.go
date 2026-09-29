package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
)

// DiscardClean must leave a clean tree even when the attempt's work is STAGED.
// The step 4 soft-reset of a coder commit leaves all of it in the index, and the
// old `checkout -- .` restored from that index while `clean` skips every path
// the index holds, so a staged `.env` survived the discard and assertClean
// stopped the whole run (review run 20260929-113519, i1).
func TestDiscardCleanDropsStagedWork(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, ".gitignore", "ignored.txt\n")
	git(t, repo, "add", ".gitignore")
	git(t, repo, "commit", "-q", "-m", "ignore")
	base := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))

	// A coder commit, soft-reset to base exactly as checkInvariants does: every
	// change it made is now staged.
	writeFile(t, repo, "main.go", "package main // coder\n")
	writeFile(t, repo, ".env", "SECRET=1\n")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-q", "-m", "coder")
	c := New(config.Target{Path: repo})
	if err := c.ResetSoft(t.Context(), base); err != nil {
		t.Fatal(err)
	}
	// Plus unstaged and untracked residue, and an ignored file the discard must
	// leave to finishDiscard.
	writeFile(t, repo, "main.go", "package main // unstaged on top\n")
	writeFile(t, repo, "loose.txt", "x")
	writeFile(t, repo, "ignored.txt", "keep")

	if err := c.DiscardClean(t.Context()); err != nil {
		t.Fatal(err)
	}
	clean, err := c.GitClean(t.Context())
	if err != nil || !clean {
		t.Fatalf("after DiscardClean: clean=%v err=%v, status:\n%s", clean, err, git(t, repo, "status", "--porcelain"))
	}
	if _, err := os.Stat(filepath.Join(repo, ".env")); !os.IsNotExist(err) {
		t.Errorf("the staged .env survived the discard (stat err %v)", err)
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "main.go")); string(b) != "package main\n" {
		t.Errorf("main.go = %q, want the base content", b)
	}
	if got := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD")); got != base {
		t.Errorf("HEAD moved to %s, want %s", got, base)
	}
	if _, err := os.Stat(filepath.Join(repo, "ignored.txt")); err != nil {
		t.Errorf("an ignored file was removed; that is finishDiscard's job, from its census: %v", err)
	}
}

// ChangedSince must name BOTH sides of a rename. git diff detects renames by
// default and lists only the new path, so a finding on the old one looked
// untouched (review run 20260929-113519, i10).
func TestChangedSinceNamesBothSidesOfARename(t *testing.T) {
	repo := gitRepo(t)
	base := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "mv", "main.go", "moved.go")
	git(t, repo, "commit", "-q", "-m", "move")

	c := New(config.Target{Path: repo})
	changed, err := c.ChangedSince(t.Context(), base)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"main.go", "moved.go"} {
		if !changed[p] {
			t.Errorf("ChangedSince omits %s: %v", p, changed)
		}
	}
}
