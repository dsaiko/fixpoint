package target

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// lockName is the lock file's name inside the repository's git directory. It
// lives there, not in the worktree, so it can never dirty the tree, be swept
// into a round commit, or need a git exclusion of its own.
const lockName = "fixpoint.lock"

// LockRepo takes an exclusive, whole-repository lock for the duration of a run
// and returns the function that releases it.
//
// Every mutating decision the loop makes is derived from a SNAPSHOT of the
// worktree: the preflight clean check says the tree holds nothing of the
// operator's, the verify gate says what is in the tree builds, and `git add -A`
// then commits whatever is there. Two fixpoint processes on one repository
// invalidate all three -- both see a clean tree, both launch coders into the same
// files, and edits one process never verified (or another coder's half-finished
// work) land in the other's commit, attributed to its findings. The failure is
// silent, which is the worst kind here: the commit looks like a verified round.
// Competing stash/commit recovery paths additionally collide on git's index lock,
// and one run's reconcile can stash the other's work.
//
// The lock is advisory flock(2) on a file in the git directory, so it is released
// by the kernel if fixpoint is killed -- a stale lock file never wedges the next
// run, which a pidfile would. It is taken without blocking: a second run reports
// who holds the repository and exits rather than queueing for an unknown time
// behind an agent session.
func (c *Collector) LockRepo(ctx context.Context) (func(), error) {
	gitDir, err := c.git(ctx, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return nil, fmt.Errorf("locate the git directory of %s to lock it: %w", c.cfg.Path, err)
	}
	path := filepath.Join(strings.TrimSpace(gitDir), lockName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open repository lock %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		holder := lockHolder(f)
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("another fixpoint run (%s) is already working on %s; a run owns the whole repository (it commits and reconciles the worktree), so wait for it to finish or point this run at a separate checkout", holder, c.cfg.Path)
		}
		return nil, fmt.Errorf("lock repository %s: %w", c.cfg.Path, err)
	}
	// Record who holds it, for the message the next run prints. Best-effort: the
	// lock is held by the descriptor, never by the contents, so a failed write
	// costs a diagnostic and nothing more.
	if err := f.Truncate(0); err == nil {
		_, _ = f.WriteAt([]byte(fmt.Sprintf("pid %d\n", os.Getpid())), 0)
	}
	return func() {
		// Unlock explicitly rather than relying on Close, so the order is visible
		// and a future caller that keeps the file open cannot leak the lock.
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

// lockHolder reads the holder description the owning run wrote, for the contention
// message. Any failure degrades to a generic phrase: naming the repository is the
// point, naming the pid is a convenience.
func lockHolder(f *os.File) string {
	b := make([]byte, 64)
	n, _ := f.ReadAt(b, 0) // a short read is expected (io.EOF); the bytes read still count
	if s := strings.TrimSpace(string(b[:n])); s != "" {
		return s
	}
	return "pid unknown"
}
