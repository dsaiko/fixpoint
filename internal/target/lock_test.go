package target

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

// The lock is opened, TRUNCATED, and rewritten inside the target's .git -- which
// fixpoint treats as attacker-controllable. An extracted archive or crafted
// checkout can ship .git/fixpoint.lock as a symlink to any file the operator can
// write, and following it would destroy that file from a review-only run that
// passes no trust gate. Refuse, and leave the pointed-at file untouched.
func TestLockRepoRefusesASymlinkedLockFile(t *testing.T) {
	repo := gitRepo(t)
	victim := filepath.Join(t.TempDir(), "authorized_keys")
	const content = "ssh-ed25519 AAAA... operator@host\n"
	if err := os.WriteFile(victim, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(repo, ".git", lockName)); err != nil {
		t.Fatal(err)
	}

	release, err := New(config.Target{Path: repo}).LockRepo(t.Context())
	if err == nil {
		release()
		t.Fatal("LockRepo() followed a symlinked lock path; it truncates that file, so an arbitrary file would have been destroyed")
	}
	if !strings.Contains(err.Error(), "symlink") || !strings.Contains(err.Error(), lockName) {
		t.Errorf("refusal must name the path and say why, got: %v", err)
	}
	got, rerr := os.ReadFile(victim)
	if rerr != nil {
		t.Fatalf("read the symlink target after the refusal: %v", rerr)
	}
	if string(got) != content {
		t.Errorf("the symlink target was modified: %q, want %q", got, content)
	}
}

// The same write, through the shape O_NOFOLLOW cannot see: a hard link planted at
// the lock path is indistinguishable from a regular file at open time, and
// truncating it destroys the other name for the same inode.
func TestLockRepoRefusesAHardLinkedLockFile(t *testing.T) {
	repo := gitRepo(t)
	victim := filepath.Join(t.TempDir(), "notes.db")
	const content = "important\n"
	if err := os.WriteFile(victim, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(victim, filepath.Join(repo, ".git", lockName)); err != nil {
		t.Skipf("hard links unavailable here: %v", err)
	}

	release, err := New(config.Target{Path: repo}).LockRepo(t.Context())
	if err == nil {
		release()
		t.Fatal("LockRepo() accepted a hard-linked lock path; truncating it destroys the file it shares an inode with")
	}
	got, rerr := os.ReadFile(victim)
	if rerr != nil {
		t.Fatalf("read the hard-link target after the refusal: %v", rerr)
	}
	if string(got) != content {
		t.Errorf("the hard-link target was modified: %q, want %q", got, content)
	}
}

// The third shape a crafted .git can plant: a fifo. O_NOFOLLOW does not see it,
// it has one link so the hard-link check passes, and LockRepo would go on to
// Truncate(0) and write to it -- in a run that passes no trust gate. Nothing
// else in the suite reaches the non-regular-file refusal, so without this test
// dropping the IsRegular check would leave every other lock test green.
func TestLockRepoRefusesAFifoLockFile(t *testing.T) {
	repo := gitRepo(t)
	path := filepath.Join(repo, ".git", lockName)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("fifos unavailable here: %v", err)
	}

	release, err := New(config.Target{Path: repo}).LockRepo(t.Context())
	if err == nil {
		release()
		t.Fatal("LockRepo() accepted a fifo lock path; it truncates and writes that file")
	}
	if !strings.Contains(err.Error(), lockName) || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("refusal must name the path and say why, got: %v", err)
	}
	// The refusal must be the shape check, not an incidental open/flock failure:
	// the fifo is still there, untouched.
	info, serr := os.Lstat(path)
	if serr != nil {
		t.Fatalf("stat the fifo after the refusal: %v", serr)
	}
	if info.Mode().Type()&os.ModeNamedPipe == 0 {
		t.Errorf("the lock path is no longer a fifo (mode %s); it was replaced or rewritten", info.Mode())
	}
}

// A lock file owned by another user is one fixpoint must not truncate, and the
// branch cannot be reached through LockRepo: a test process has a single uid and
// cannot create a file owned by anyone else. checkLockStat takes the caller's uid
// as an argument for exactly this reason, so the predicate is still covered.
func TestCheckLockStatRefusesAForeignOwnedLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), lockName)
	if err := os.WriteFile(path, []byte("pid 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Our own uid: a plain file we own is the only accepted shape.
	if err := checkLockStat(info, path, os.Getuid()); err != nil {
		t.Fatalf("checkLockStat rejected a regular file owned by the caller: %v", err)
	}
	// The same file seen by a different user: refused, naming both uids so the
	// operator can tell who owns it.
	foreign := os.Getuid() + 1
	err = checkLockStat(info, path, foreign)
	if err == nil {
		t.Fatal("checkLockStat accepted a lock file owned by another uid; fixpoint truncates and rewrites it")
	}
	for _, want := range []string{path, strconv.Itoa(foreign)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal must contain %q, got: %v", want, err)
		}
	}
}

// A directory that is not a git repository has no repository to lock, and the
// caller must hear about it rather than proceed unlocked.
func TestLockRepoFailsOutsideAGitRepository(t *testing.T) {
	if _, err := New(config.Target{Path: t.TempDir()}).LockRepo(t.Context()); err == nil {
		t.Error("LockRepo() succeeded outside a git repository; a run would then hold no lock at all")
	}
}
