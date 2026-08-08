// Package worktree gives a coder session its own checkout of the target, so
// several of them can work at once without seeing each other's edits.
//
// It exists for one measured reason: in a two-hour run of this tool against its
// own pull request, the coder accounted for 5301 of 7562 seconds -- 70% of the
// wall clock -- and every second of it was sequential. A coder session is not
// CPU-bound; it spends nearly all of that waiting on a provider. Running several
// at once costs the local machine almost nothing.
//
// What it deliberately does NOT parallelize is the verify gate. Four concurrent
// runs of this project's own test suite were measured at 391s against 159s for
// one, so the gate scales at about 1.6x rather than 4x -- and much worse than the
// throughput number suggests, because two fixes that each pass ALONE can fail
// TOGETHER. The property that a failing gate names exactly one fix is what makes
// a round auditable and a single fix revertable, and it survives only if the gate
// stays single-threaded on the real tree. So a worktree here produces a PATCH in
// isolation; applying, verifying and committing it happens exactly as it does
// today, one at a time, in the order the issues were listed.
package worktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Tree is one isolated checkout. Close removes it.
type Tree struct {
	// Dir is where the agent runs.
	Dir string
	// repo is the main checkout that owns this worktree.
	repo string
	// run executes git, injected so tests do not need a real repository for the
	// paths that never touch one.
	run func(ctx context.Context, dir string, args ...string) (string, error)
}

// Add creates a detached worktree of repo at base, under parent.
//
// Detached on purpose: nothing here should move a branch. The worktree exists to
// hold edits for as long as one agent session takes, and its result is read as a
// diff -- a branch would be a second place for the run's state to live, and a
// second thing to reconcile after an interrupt.
func Add(ctx context.Context, repo, parent, base, name string) (*Tree, error) {
	t := &Tree{Dir: filepath.Join(parent, name), repo: repo, run: git}
	if _, err := t.run(ctx, repo, "worktree", "add", "--detach", t.Dir, base); err != nil {
		return nil, fmt.Errorf("worktree add %s at %s: %w", t.Dir, base, err)
	}
	return t, nil
}

// Patch is what the session changed, as a unified diff against the base.
//
// Tracked and untracked both: a coder that adds a file is doing the same job as
// one that edits it, and a patch that silently omitted new files would apply
// cleanly and leave the fix half-made. `git add --intent-to-add` is what puts an
// untracked file into the diff without staging its content.
//
// Empty means the session changed nothing, which is a normal outcome -- a
// rejected issue -- and not an error.
func (t *Tree) Patch(ctx context.Context, exclude ...string) (string, error) {
	if _, err := t.run(ctx, t.Dir, "add", "--intent-to-add", "--", "."); err != nil {
		return "", fmt.Errorf("stage new files in %s: %w", t.Dir, err)
	}
	args := []string{"diff", "--binary"}
	if len(exclude) > 0 {
		args = append(args, "--", ".")
		for _, e := range exclude {
			args = append(args, ":(exclude)"+e)
		}
	}
	out, err := t.run(ctx, t.Dir, args...)
	if err != nil {
		return "", fmt.Errorf("read patch from %s: %w", t.Dir, err)
	}
	return out, nil
}

// Close removes the worktree and its directory.
//
// --force because the session left the tree dirty by design; that is the whole
// product. Errors are returned rather than swallowed so a caller can report a
// leaked directory, but no caller should fail a run over one: the fix it holds
// has already been read out as a patch.
func (t *Tree) Close(ctx context.Context) error {
	if _, err := t.run(ctx, t.repo, "worktree", "remove", "--force", t.Dir); err != nil {
		// The worktree may already be gone (a killed run, a manual cleanup). Removing
		// the directory is what actually matters; prune keeps git's metadata honest.
		_ = os.RemoveAll(t.Dir)
		if _, perr := t.run(ctx, t.repo, "worktree", "prune"); perr != nil {
			return fmt.Errorf("remove worktree %s: %w", t.Dir, err)
		}
	}
	return nil
}

// git runs one git command and returns its stdout.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}
