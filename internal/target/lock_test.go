package target

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
)

// The point of the lock: while one run owns the repository, a second must be
// refused rather than allowed to edit the same worktree. flock is per open file
// description, so two collectors in ONE process contend exactly as two processes
// do -- which is what makes this testable without spawning fixpoint twice.
func TestLockRepoExcludesASecondRun(t *testing.T) {
	repo := gitRepo(t)
	first := New(config.Target{Path: repo})
	release, err := first.LockRepo(t.Context())
	if err != nil {
		t.Fatalf("LockRepo() on a free repository: %v", err)
	}

	if _, err := New(config.Target{Path: repo}).LockRepo(t.Context()); err == nil {
		t.Fatal("a second run took the lock while the first held it; both would then edit one worktree")
	} else {
		// The message has to be actionable: which repository, and who holds it.
		if !strings.Contains(err.Error(), repo) {
			t.Errorf("contention error must name the repository, got: %v", err)
		}
		if !strings.Contains(err.Error(), strconv.Itoa(os.Getpid())) {
			t.Errorf("contention error must name the holder (pid %d), got: %v", os.Getpid(), err)
		}
	}

	// Released, the repository is available again -- a lock that outlived its run
	// would wedge every later run on the same checkout.
	release()
	release2, err := New(config.Target{Path: repo}).LockRepo(t.Context())
	if err != nil {
		t.Fatalf("LockRepo() after release: %v, want the repository to be free again", err)
	}
	release2()
}

// The lock file lives in the git directory, never in the worktree: in the worktree
// it would dirty the tree the clean check guards, be swept into a round commit by
// `git add -A`, and need a git exclusion of its own.
func TestLockRepoDoesNotTouchTheWorktree(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Path: repo})
	release, err := c.LockRepo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	if clean, err := c.GitClean(t.Context()); err != nil || !clean {
		t.Errorf("holding the lock left the tree dirty (clean=%v err=%v): %s", clean, err, git(t, repo, "status", "--porcelain"))
	}
	if _, err := os.Stat(filepath.Join(repo, ".git", lockName)); err != nil {
		t.Errorf("expected the lock file inside .git: %v", err)
	}
}

// A directory that is not a git repository has no repository to lock, and the
// caller must hear about it rather than proceed unlocked.
func TestLockRepoFailsOutsideAGitRepository(t *testing.T) {
	if _, err := New(config.Target{Path: t.TempDir()}).LockRepo(t.Context()); err == nil {
		t.Error("LockRepo() succeeded outside a git repository; a run would then hold no lock at all")
	}
}
