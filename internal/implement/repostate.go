package implement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// gitDirName is the marker whose presence below the worktree root means a
// nested repository.
const gitDirName = ".git"

// RepoState is the repository-invariant snapshot of §5.2 step 4: everything a
// coder session could change about the repository ITSELF -- as opposed to the
// worktree -- captured before the session and compared after it. The list
// includes the census-bearing metadata (info/exclude, info/attributes,
// alternates, shallow) because one line in .git/info/exclude makes a source
// path invisible to git status, so a gate could pass on a file no commit
// contains (review run 20260813-003817).
type RepoState struct {
	// ConfigDigest is the sha256 of .git/config.
	ConfigDigest string
	// HooksPath is the effective core.hooksPath; HooksEmpty is whether the
	// directory it names contains anything.
	HooksPath  string
	HooksEmpty bool
	// RefsDigest covers `git for-each-ref` minus the branch HEAD rides
	// (task commits legitimately move it; step 4 checks HEAD separately) plus
	// the packed-refs file.
	RefsDigest string
	// NestedGit lists any .git file or directory below the worktree root.
	NestedGit []string
	// InfoExcludeDigest and InfoAttributesDigest cover the two metadata files
	// Init wrote empty; "" means the file is absent, which is also a known value.
	InfoExcludeDigest    string
	InfoAttributesDigest string
	// HasAlternates and IsShallow must stay false for the lifetime of the run.
	HasAlternates bool
	IsShallow     bool
}

// SnapshotRepoState reads the invariant surface of the repository at dir.
// excludeBranch is the branch HEAD is expected to move on (normally "main");
// its ref is left out of the digest so an ordinary task commit does not read
// as a ref-list mutation.
func (g Git) SnapshotRepoState(ctx context.Context, dir, excludeBranch string) (RepoState, error) {
	var s RepoState
	gitDir := filepath.Join(dir, gitDirName)

	var err error
	if s.ConfigDigest, err = fileDigest(filepath.Join(gitDir, "config")); err != nil {
		return s, fmt.Errorf("repo state: %w", err)
	}
	// --local: fixpoint's own invocations pin core.hooksPath=/dev/null via -c
	// (gitenv.SafeConfigArgs), and a plain `git config` would read that pin
	// back instead of the repository's durable setting -- which is the thing a
	// session could have edited and the thing the delivered project keeps.
	hooks, err := g.run(ctx, dir, "config", "--local", "core.hooksPath")
	if err != nil {
		// An unset key exits 1; the effective path is then .git/hooks.
		hooks = ".git/hooks"
	}
	s.HooksPath = strings.TrimSpace(hooks)
	s.HooksEmpty = dirEmpty(filepath.Join(dir, filepath.FromSlash(s.HooksPath)))

	refs, err := g.run(ctx, dir, "for-each-ref", "--format=%(refname) %(objectname)")
	if err != nil {
		return s, err
	}
	keep := make([]string, 0, 8)
	for _, line := range strings.Split(strings.TrimSpace(refs), "\n") {
		if line == "" || strings.HasPrefix(line, "refs/heads/"+excludeBranch+" ") {
			continue
		}
		keep = append(keep, line)
	}
	packed, _ := os.ReadFile(filepath.Join(gitDir, "packed-refs"))
	sum := sha256.Sum256([]byte(strings.Join(keep, "\n") + "\x00" + string(packed)))
	s.RefsDigest = hex.EncodeToString(sum[:])

	if s.NestedGit, err = g.findNestedGit(ctx, dir); err != nil {
		return s, err
	}
	if s.InfoExcludeDigest, err = fileDigest(filepath.Join(gitDir, "info", "exclude")); err != nil {
		return s, err
	}
	if s.InfoAttributesDigest, err = fileDigest(filepath.Join(gitDir, "info", "attributes")); err != nil {
		return s, err
	}
	// Lstat: a symlink at either path is a present marker, and the point is
	// presence, not content.
	if _, err := os.Lstat(filepath.Join(gitDir, "objects", "info", "alternates")); err == nil {
		s.HasAlternates = true
	}
	if _, err := os.Lstat(filepath.Join(gitDir, "shallow")); err == nil {
		s.IsShallow = true
	}
	return s, nil
}

// Diff names every invariant that moved since prev, in words the
// repository-invariant failure can print. Empty means intact.
func (s RepoState) Diff(prev RepoState) []string {
	var out []string
	if s.ConfigDigest != prev.ConfigDigest {
		out = append(out, ".git/config changed")
	}
	if s.HooksPath != prev.HooksPath {
		out = append(out, fmt.Sprintf("core.hooksPath changed from %q to %q", prev.HooksPath, s.HooksPath))
	}
	if !s.HooksEmpty && prev.HooksEmpty {
		out = append(out, "the hooks directory is no longer empty")
	}
	if s.RefsDigest != prev.RefsDigest {
		out = append(out, "the ref list changed (a ref other than the working branch moved)")
	}
	if len(s.NestedGit) > 0 && len(prev.NestedGit) == 0 {
		out = append(out, "nested .git appeared: "+strings.Join(s.NestedGit, ", "))
	}
	if s.InfoExcludeDigest != prev.InfoExcludeDigest {
		out = append(out, ".git/info/exclude changed -- the file that decides what the census can see")
	}
	if s.InfoAttributesDigest != prev.InfoAttributesDigest {
		out = append(out, ".git/info/attributes changed")
	}
	if s.HasAlternates && !prev.HasAlternates {
		out = append(out, ".git/objects/info/alternates appeared")
	}
	if s.IsShallow && !prev.IsShallow {
		out = append(out, "a shallow marker appeared")
	}
	return out
}

// fileDigest hashes one file; a missing file digests to "" -- absence is a
// known value, not an error, because Init writes these files empty and a
// deleted file must read as a change.
//
// A SYMLINK digests to a distinct sentinel rather than to its destination's
// bytes. Following it would let a session copy `.git/config` unchanged to an
// external path, replace the repository's file with a link to that copy, and
// pass the invariant with an identical digest -- then change the external file
// afterwards, leaving the DELIVERED repository configured to run
// attacker-controlled hooks or filters on the operator's next ordinary git
// command (review run 20260813-161029). Git metadata is fixpoint's own; a link
// where a file belongs is itself the change worth reporting.
func fileDigest(path string) (string, error) {
	st, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		dest, _ := os.Readlink(path)
		return "symlink:" + dest, nil
	}
	if !st.Mode().IsRegular() {
		return "irregular:" + st.Mode().String(), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// dirEmpty reports whether path is a real, empty DIRECTORY. A symlink is not
// one, however empty its destination: the hooks directory is what
// core.hooksPath resolves to, so a link there points git at a tree nothing in
// this repository controls (same finding as fileDigest's).
func dirEmpty(path string) bool {
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) == 0
}

// findNestedGit walks the UN-IGNORED worktree for .git entries below the root
// -- a nested repository there would turn a stage into a gitlink nothing gated.
//
// Scoped to the un-ignored tree because that is the only part that can reach a
// commit, which is the rule's whole reason for existing. Unscoped it made
// ordinary dependency installation look like tampering (review run
// 20260813-180828, i45): a `.git` under node_modules/ -- any package installed
// from a git URL, any vendored fixture repository -- read as "a nested .git
// appeared" and stopped the RUN, the design's most severe outcome, blaming the
// session for rewriting repository metadata. The trigger is behavior §6
// explicitly encourages. Ignored paths are exempt from every other census rule
// for the same reason (§5.2 step 2): fixpoint cannot reason about build state.
//
// Skipping those subtrees also stops the walk from descending node_modules
// twice per attempt -- ~160 full traversals over a 40-task run.
func (g Git) findNestedGit(ctx context.Context, root string) ([]string, error) {
	ignored, err := g.ignoredDirs(ctx, root)
	if err != nil {
		return nil, err
	}
	var found []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		if d.IsDir() && d.Name() != gitDirName && ignored[filepath.ToSlash(rel)] {
			return filepath.SkipDir
		}
		if d.Name() != gitDirName {
			return nil
		}
		if rel == gitDirName {
			return filepath.SkipDir // the repository's own
		}
		found = append(found, rel)
		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	return found, err
}

// ignoredDirs is the set of wholly-ignored directories, as git reports them.
// --directory collapses a fully-ignored directory to the directory itself, so
// the walk can prune at the top instead of listing every file beneath it.
func (g Git) ignoredDirs(ctx context.Context, dir string) (map[string]bool, error) {
	out, err := g.run(ctx, dir, "ls-files", "--others", "--ignored", "--exclude-standard", "--directory", "-z")
	if err != nil {
		return nil, err
	}
	set := map[string]bool{
		// fixpoint's own scratch inside the project, which is not the coder's
		// and is exempt from the census for the same reason.
		".fixpoint": true,
	}
	for _, p := range strings.Split(strings.Trim(out, "\x00"), "\x00") {
		if p = strings.TrimSuffix(p, "/"); p != "" {
			set[p] = true
		}
	}
	return set, nil
}
