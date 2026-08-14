package implement

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/gitenv"
)

// gitOpTimeout bounds each plumbing call; these are local reads over a
// repository fixpoint created, so anything slower than this is wedged.
const gitOpTimeout = 2 * time.Minute

// Git is the implement pipeline's git handle: the read path's plumbing calls,
// carrying the SAME hardened environment the write-target's Collector runs
// with.
//
// The environment is a constructor argument rather than something this package
// derives, because the alternative was measured to be wrong (review run
// 20260813-180828, i12). The read path used to build its own env from
// gitenv.Harden(nil) -- the full os.Environ() with the operator's global and
// system git config still in effect -- while the write path ran with
// credentials stripped and GIT_CONFIG_GLOBAL/SYSTEM pinned to /dev/null. Two
// of these calls are not plumbing reads at all: RestoreControlArtifacts runs
// `git checkout` and CleanCheck runs `git clone`, both of which WRITE a
// worktree, and a checkout applies the clean/smudge filters that a
// `.gitattributes` names. The coder authors that .gitattributes; the operator's
// global config supplies the filter definition (git-lfs is the canonical one);
// the filter then runs with whatever environment git inherited. That is exactly
// the class the write path's hardening closes, and the read path was outside
// it. One env, constructed once by the orchestrator and given to both, is what
// keeps them from drifting again -- including when the next pin is added.
//
// The operator-config pins are applied by run() rather than trusted to arrive in
// env, so a zero Git is safe too: the value the constructor carries decides only
// which CREDENTIALS are in scope, and the hardening cannot be lost by a caller
// that builds its env some other way.
type Git struct {
	env []string
}

// NewGit returns the handle for a write-target whose git commands must run with
// env -- the orchestrator's one hardened environment, the same slice the
// Collector is given.
func NewGit(env []string) Git { return Git{env: env} }

// maxGitOutput caps each stream. A census of a large tree is the biggest of
// these by far; 4 MB matches what the write path allows itself.
const maxGitOutput = 4 << 20

// run executes one git command in dir under the handle's environment.
//
// Through agent.Supervise, like every other subprocess in the program, and the
// two things that buys are not optional here (review run 20260814-012440):
//
// Containment. Supervise sets Setpgid, cancels by killing the GROUP, and sets
// WaitDelay. The first version used exec.CommandContext plus CombinedOutput,
// which owns neither: os/exec creates the pipes, cmd.Wait joins its copy
// goroutines, and with WaitDelay at its zero default that join is unbounded. A
// `git clone` whose index-pack child (or a smudge filter the cloned
// .gitattributes named) still holds the pipe's write end therefore blocks
// forever -- past gitOpTimeout, past max_run_duration, past the operator's first
// Ctrl-C -- and the surviving child keeps writing into the very tree CleanCheck
// is about to run the gate over.
//
// Separated streams. Every caller here parses the result as machine-readable
// data: NUL-delimited `status --porcelain`, `ls-files -z`, a bare
// `config --local core.hooksPath` that lands straight in RepoState. Folding
// stderr in meant one git warning on a SUCCESSFUL command corrupted a parsed
// value. target.Collector.runInput documents this same hazard and has always
// kept the streams apart; this path was written without it.
func (g Git) run(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitOpTimeout)
	defer cancel()
	full := append(gitenv.SafeConfigArgs(), args...)
	cmd := exec.CommandContext(ctx, gitenv.Tool("git"), full...)
	cmd.Dir = dir
	cmd.Env = gitenv.Harden(gitenv.NoOperatorConfig(g.env))
	outBuf := agent.NewBoundedBuffer(maxGitOutput, agent.TruncationMarker(maxGitOutput))
	errBuf := agent.NewBoundedBuffer(maxGitOutput, agent.TruncationMarker(maxGitOutput))
	if _, err := agent.Supervise(ctx, cmd, outBuf, errBuf); err != nil {
		// stderr on the failure path only, where it is the diagnosis rather than
		// something a parser will read.
		return outBuf.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return outBuf.String(), nil
}

// IsAncestor reports whether base is an ancestor of head -- the §5.2 step 4
// distinction between "the coder committed on top" (recoverable by soft
// reset) and "the coder rewrote history" (a run stop).
func (g Git) IsAncestor(ctx context.Context, dir, base, head string) bool {
	_, err := g.run(ctx, dir, "merge-base", "--is-ancestor", base, head)
	return err == nil
}
