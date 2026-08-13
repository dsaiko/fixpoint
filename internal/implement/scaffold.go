package implement

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Repo is the slice of the collector the scaffold needs: the hardened init and
// the exact-path commit. An interface so the scaffold's tests exercise the
// refusal paths without a git repository, while the real caller hands in
// *target.Collector.
type Repo interface {
	Init(ctx context.Context) error
	LockRepo(ctx context.Context) (func(), error)
	CommitExact(ctx context.Context, header, body string, paths []string, allowEmpty bool) (string, error)
}

// Scaffold is the only code in the tool that creates a repository (DESIGN.md
// §3): claim the directory, harden-init it, write the bootstrap files, and
// make the one tool-authored commit that gives every task commit a parent.
//
// os.Mkdir, not MkdirAll: it fails when the path exists, so the check and the
// claim are one syscall -- the same argument create.Publish makes for link(2).
// The caller verified at preflight that out is absent and outside any git
// repository; this claim is the authoritative one.
//
// files maps repository-relative names to bytes; DESIGN.md must be byte-exact
// (altering it would break the hash match with what review-design approved,
// §4.3) and the caller builds the rest. The commit stages exactly these names,
// sorted, so the bootstrap commit's content is a pure function of its inputs.
//
// The repository lock is taken the instant `git init` has made one, BEFORE any
// file is written or committed, and returned to the caller to hold for the rest
// of the run. Locking after the bootstrap commit (the shape the second review
// of this code found, run 20260813-161029) leaves a window in which .git
// exists and the lock does not, so a concurrent fixpoint run can claim an
// apparently free repository while this one is still writing into it.
func Scaffold(ctx context.Context, repo Repo, out string, files map[string][]byte, header, body string) (sha string, release func(), err error) {
	if err := os.Mkdir(out, 0o750); err != nil {
		return "", nil, fmt.Errorf("claim %s: %w -- the write-target must not exist; implement-design builds into a fresh directory", out, err)
	}
	if err := repo.Init(ctx); err != nil {
		return "", nil, err
	}
	release, err = repo.LockRepo(ctx)
	if err != nil {
		return "", nil, err
	}
	// Any failure from here on releases the lock: the caller only holds what it
	// was handed together with a SHA.
	defer func() {
		if err != nil {
			release()
		}
	}()
	names := make([]string, 0, len(files))
	for name := range files {
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
			return "", nil, fmt.Errorf("scaffold: %q is not a safe repository-relative name", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := filepath.Join(out, name)
		if dir := filepath.Dir(p); dir != out {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return "", nil, err
			}
		}
		if err := os.WriteFile(p, files[name], 0o600); err != nil {
			return "", nil, err
		}
	}
	sha, err = repo.CommitExact(ctx, header, body, names, false)
	return sha, release, err
}

// GitignoreContent renders the bootstrap .gitignore: fixpoint's scratch root
// first (later fix-code runs need it ignored, §4.3), then the operator's
// stack seed. A control artifact -- fixpoint writes it once, no session may
// change it (§5.1).
func GitignoreContent(seed []string) string {
	var b strings.Builder
	b.WriteString("# Written by fixpoint implement-design; sessions may not edit this file.\n")
	b.WriteString(".fixpoint/\n")
	for _, s := range seed {
		if t := strings.TrimSpace(s); t != "" {
			b.WriteString(t + "\n")
		}
	}
	return b.String()
}
