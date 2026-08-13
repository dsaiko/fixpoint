package implement

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/gitenv"
)

// gitOpTimeout bounds each plumbing call; these are local reads over a
// repository fixpoint created, so anything slower than this is wedged.
const gitOpTimeout = 2 * time.Minute

// gitRun executes one hardened git command in dir. The repostate and census
// code runs plumbing reads only -- no network, no hooks (SafeConfigArgs pins
// them off), no writes -- so this deliberately does not reach for the
// collector: the collector is the WRITE path, and keeping the read path
// separate keeps "deterministic, git plumbing only" checkable at a glance.
func gitRun(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitOpTimeout)
	defer cancel()
	full := append(gitenv.SafeConfigArgs(), args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = dir
	cmd.Env = gitenv.Harden(nil)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
