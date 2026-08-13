package implement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/verify"
)

// ControlArtifactsChanged compares the worktree bytes of the four control
// artifacts against their blobs in the bootstrap commit (§4.3). Returns the
// changed names; the caller restores them and fails the task as a contract
// violation -- session-scoped, because the authoritative bytes were never
// lost.
func ControlArtifactsChanged(ctx context.Context, dir, bootstrapSHA string) ([]string, error) {
	var changed []string
	for _, name := range ControlArtifacts {
		blob, err := gitRun(ctx, dir, "show", bootstrapSHA+":"+name)
		if err != nil {
			// Absent from the bootstrap commit: nothing to protect.
			continue
		}
		work, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || sha256.Sum256(work) != sha256.Sum256([]byte(blob)) {
			changed = append(changed, name)
		}
	}
	return changed, nil
}

// RestoreControlArtifacts puts the bootstrap commit's version of the named
// control artifacts back into the worktree.
func RestoreControlArtifacts(ctx context.Context, dir, bootstrapSHA string, names []string) error {
	if len(names) == 0 {
		return nil
	}
	args := append([]string{"checkout", "-q", bootstrapSHA, "--"}, names...)
	if out, err := gitRun(ctx, dir, args...); err != nil {
		return fmt.Errorf("restore control artifacts: %w: %s", err, out)
	}
	return nil
}

// CleanCheck clones HEAD into a scratch directory and runs the gate in the
// clone (§7.2): the enforcement of the design's central claim -- the
// committed bytes alone satisfy the gate -- put where it can actually be
// checked. The working tree cannot carry that claim; a fresh checkout can.
func CleanCheck(ctx context.Context, repoDir, scratchDir string, vcfg config.Verify, env []string) (verify.Report, error) {
	clone := filepath.Join(scratchDir, "clean-check")
	if err := os.RemoveAll(clone); err != nil {
		return verify.Report{}, err
	}
	// --no-hardlinks: the clone must not share object files with a repository a
	// gate command is about to run inside.
	if out, err := gitRun(ctx, scratchDir, "clone", "-q", "--no-hardlinks", repoDir, clone); err != nil {
		return verify.Report{}, fmt.Errorf("clean-check clone: %w: %s", err, out)
	}
	return verify.Run(ctx, vcfg, clone, env), nil
}

// GateWorst is the fit rule's gate term: the SUM of every configured
// command's timeout, because the verify executor applies the timeout per
// command and runs them sequentially (§4.2 rule 7).
func GateWorst(v config.Verify) time.Duration {
	return time.Duration(len(v.Commands)) * v.Timeout.Std()
}

// RenderCommands renders the gate's argv lists as display strings, for the
// planner prompt and the verify profile.
func RenderCommands(v config.Verify) []string {
	out := make([]string, 0, len(v.Commands))
	for _, c := range v.Commands {
		out = append(out, strings.Join(c.Run, " "))
	}
	return out
}

// DigestBytes is the design fingerprint: sha256, hex.
func DigestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
