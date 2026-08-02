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
	// O_NOFOLLOW: the lock lives in the TARGET's git directory, which fixpoint
	// already treats as attacker-controllable (see internal/gitenv, and the logs
	// symlink check). An extracted archive or crafted checkout can ship
	// .git/fixpoint.lock as a symlink to ~/.ssh/authorized_keys or any other file
	// the operator can write, and the Truncate(0) below would then destroy it --
	// from a review-only run that passes no trust gate. Without O_NOFOLLOW the open
	// follows the link; with it the open fails (ELOOP) and the run refuses.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, fmt.Errorf("refusing to run: the repository lock path %s is a symlink; fixpoint truncates and rewrites that file, so following it would destroy whatever it points at. Remove it (a genuine fixpoint lock is a regular file) and treat this checkout as untrusted", path)
		}
		return nil, fmt.Errorf("open repository lock %s: %w", path, err)
	}
	// Defense in depth for the shapes O_NOFOLLOW does not cover: a hard link to
	// someone else's file, a fifo that would block a reader, a device node. Only a
	// regular file we own is safe to truncate and rewrite.
	if err := checkLockFile(f, path); err != nil {
		_ = f.Close()
		return nil, err
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

// checkLockFile confirms the OPENED descriptor is a plain file this user owns,
// before the caller truncates it. O_NOFOLLOW rejects the symlink shape; this
// rejects the rest a prepared .git can plant at the path: a hard link to a file
// elsewhere (which no open flag detects, and which truncation would destroy just
// as thoroughly), a fifo, a device node. It stats the descriptor rather than the
// path, so nothing can be swapped in between the check and the write.
func checkLockFile(f *os.File, path string) error {
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("inspect repository lock %s: %w", path, err)
	}
	return checkLockStat(info, path, os.Getuid())
}

// checkLockStat holds checkLockFile's predicates, with the caller's own uid
// passed in rather than looked up: the foreign-owner refusal is otherwise
// unreachable in a test, since a test process has exactly one uid and cannot
// create a file owned by another.
func checkLockStat(info os.FileInfo, path string, uid int) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to run: the repository lock path %s is not a regular file (%s); fixpoint truncates and rewrites that file, and a genuine lock is a plain file. Treat this checkout as untrusted", path, info.Mode().Type())
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil // unknown platform shape; the regular-file check above still held
	}
	if st.Nlink > 1 {
		return fmt.Errorf("refusing to run: the repository lock %s has %d hard links, so it is also some other path; fixpoint truncates and rewrites it, which would destroy that file. Treat this checkout as untrusted", path, st.Nlink)
	}
	if uid >= 0 && int(st.Uid) != uid {
		return fmt.Errorf("refusing to run: the repository lock %s is owned by uid %d, not by the user running fixpoint (uid %d); fixpoint truncates and rewrites that file. Treat this checkout as untrusted", path, st.Uid, uid)
	}
	return nil
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
