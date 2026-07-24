// Package target collects the material each review round runs against, and
// provides the git operations the loop depends on (base pinning, clean-tree
// checks, per-round commits).
package target

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
)

// maxMaterial caps the material embedded into prompts. Agents are agentic and
// can read the repository themselves; the cap only bounds the prompt size.
const maxMaterial = 300_000

// gitOpTimeout bounds every git/gh subprocess so a hung credential prompt, a
// blocking commit hook, or a stalled network fetch cannot stall the run
// forever. It is generous enough for a large fetch/checkout but still caps a
// wedged operation the outer context cancel alone could not reach.
const gitOpTimeout = 10 * time.Minute

// Collector produces the per-round review material.
type Collector struct {
	cfg     config.Target
	baseSHA string // pinned at Prepare for git-diff / pr modes
	// logsExclude is the run's own logs directory as a path relative to
	// target.path, when it lives inside the target. Set via ExcludeLogs, it is
	// kept out of directory walks and untracked-file listings so later rounds
	// never review the run's own prompts and outputs.
	logsExclude string
}

// New returns a collector for the configured target.
func New(cfg config.Target) *Collector { return &Collector{cfg: cfg} }

// ExcludeLogs records the run's logs directory (as a target-relative slash
// path) so Collect omits it from directory walks and untracked-file listings.
// An empty rel (logs live outside the target) disables the exclusion.
func (c *Collector) ExcludeLogs(rel string) { c.logsExclude = rel }

// Prepare runs once at startup: resolves and pins the diff base (git-diff),
// or checks out the PR branch and pins its merge base (pr).
func (c *Collector) Prepare(ctx context.Context) error {
	switch c.cfg.Mode {
	case config.ModeGitDiff:
		ref := c.cfg.BaseRef
		if ref == "" {
			return nil // unstaged working changes; nothing to pin
		}
		sha, err := c.git(ctx, "rev-parse", ref)
		if err != nil {
			return fmt.Errorf("resolve base_ref %q: %w", ref, err)
		}
		c.baseSHA = strings.TrimSpace(sha)
	case config.ModePR:
		if _, err := c.git(ctx, "rev-parse", "--git-dir"); err != nil {
			return fmt.Errorf("target.path is not a git repository (mode pr): %w", err)
		}
		out, err := c.run(ctx, "gh", "pr", "checkout", strconv.Itoa(c.cfg.PR))
		if err != nil {
			return fmt.Errorf("gh pr checkout %d: %w: %s", c.cfg.PR, err, out)
		}
		// Pin the PR's exact base commit from GitHub. A hard-coded
		// origin/<baseRefName> fails when the GitHub remote is not named origin
		// and, worse, silently pins the wrong merge base when the local
		// remote-tracking ref is stale. baseRefOid is the base tip the PR's diff
		// is computed against, so the merge base with it reproduces that diff.
		oid, err := c.run(ctx, "gh", "pr", "view", strconv.Itoa(c.cfg.PR), "--json", "baseRefOid", "--jq", ".baseRefOid")
		if err != nil {
			return fmt.Errorf("gh pr view %d: %w: %s", c.cfg.PR, err, oid)
		}
		baseOid := strings.TrimSpace(oid)
		if baseOid == "" {
			return fmt.Errorf("gh pr view %d returned an empty base commit oid", c.cfg.PR)
		}
		// gh pr checkout only fetches the head; the base tip may advance past
		// the merge base, so fetch the object if it isn't already local.
		if _, err := c.git(ctx, "cat-file", "-e", baseOid+"^{commit}"); err != nil {
			remote, rerr := c.ghRemote(ctx)
			if rerr != nil {
				return rerr
			}
			if fout, ferr := c.git(ctx, "fetch", "--no-tags", remote, baseOid); ferr != nil {
				return fmt.Errorf("fetch PR base %s from %s: %w: %s", baseOid, remote, ferr, fout)
			}
		}
		mb, err := c.git(ctx, "merge-base", "HEAD", baseOid)
		if err != nil {
			return fmt.Errorf("merge-base with base %s: %w", baseOid, err)
		}
		c.baseSHA = strings.TrimSpace(mb)
	case config.ModeDirectory:
		// nothing to prepare
	}
	return nil
}

// Collect returns the material for one round.
func (c *Collector) Collect(ctx context.Context) (string, error) {
	switch c.cfg.Mode {
	case config.ModeGitDiff, config.ModePR:
		// --no-ext-diff / --no-textconv keep this read-only collection step from
		// running attacker-controlled programs: a target's .git/config can define
		// diff.external, or .gitattributes can select a diff driver's textconv, and
		// generating patch output would otherwise execute either with fixpoint's
		// credentials -- before any agent sandboxing, even in a review-only run.
		// gitSafeConfig cannot cover these (there is no single -c that disables a
		// gitattributes-driven textconv), so they are neutralized per command here.
		args := []string{"diff", "--no-ext-diff", "--no-textconv"}
		if c.baseSHA != "" {
			args = append(args, c.baseSHA)
		}
		// Keep the run's own logs out of the diff too, not just the untracked
		// listing: a tracked file under the logs dir (a committed .prompt from an
		// earlier run) would otherwise show its full modified content here and be
		// fed back as review material. The pathspec matches the exclusion applied
		// by GitClean/Commit.
		if ex := c.excludes(); len(ex) > 0 {
			args = append(args, "--")
			args = append(args, pathspec(ex)...)
		}
		diff, err := c.git(ctx, args...)
		if err != nil {
			return "", err
		}
		// A failure here (e.g. an invalid core.excludesFile) must not be
		// swallowed: silently treating it as "no untracked files" would drop
		// them from review without reporting the collection was incomplete.
		lsArgs := []string{"ls-files", "--others", "--exclude-standard", "--"}
		lsArgs = append(lsArgs, pathspec(c.excludes())...)
		untracked, err := c.git(ctx, lsArgs...)
		if err != nil {
			return "", fmt.Errorf("list untracked files: %w", err)
		}
		var sb strings.Builder
		if c.baseSHA != "" {
			fmt.Fprintf(&sb, "Diff against pinned base %s:\n\n", c.baseSHA[:12])
		} else {
			sb.WriteString("Unstaged working-tree changes:\n\n")
		}
		sb.WriteString(diff)
		if u := strings.TrimSpace(untracked); u != "" {
			sb.WriteString("\n\nUntracked files (not shown in the diff -- read them directly):\n" + u)
		}
		return truncate(sb.String()), nil
	case config.ModeDirectory:
		count, listing, err := c.listFiles(ctx)
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "Files in scope (%d):\n", count)
		sb.WriteString(listing)
		return truncate(sb.String()), nil
	}
	return "", fmt.Errorf("unknown mode %q", c.cfg.Mode)
}

// excludes returns the repo-relative paths to keep out of collection (currently
// just the run's own logs dir when it lives inside the target).
func (c *Collector) excludes() []string {
	if c.logsExclude == "" {
		return nil
	}
	return []string{c.logsExclude}
}

// listFiles enumerates the files in scope. There is no include/allowlist option
// on purpose: one has to be re-derived per language and silently drops whatever
// it forgets (a Go project's Makefile, go.sum, or .golangci.yml), hiding those
// files from every reviewer with no diagnostic. Scope is therefore everything
// the target.exclude globs do not remove -- and, in a git repository, everything
// .gitignore does not remove either (see listGitFiles).
//
// Both paths return the total number of files in scope and a rendered listing of
// their paths capped at maxMaterial bytes: only a size-bounded prefix of the path
// text is retained, so a tree with an unbounded number of files cannot accumulate
// every path in memory only for truncate to discard the tail. Both enumerate in an
// order fixed by the tree -- WalkDir visits lexically; git ls-files emits its
// tracked and untracked sets each sorted, but as separate runs, so the whole is
// deterministic without being globally lexical. Determinism is what the cap needs:
// the retained prefix is stable across runs without materializing and sorting
// every path first.
func (c *Collector) listFiles(ctx context.Context) (int, string, error) {
	// EffectiveExcludes adds the mandatory credential patterns, which no config can
	// drop: a reviewer reads any path it is pointed at, so never point it at a key.
	excludes, err := compileGlobs(c.cfg.EffectiveExcludes())
	if err != nil {
		return 0, "", fmt.Errorf("target.exclude: %w", err)
	}
	if c.IsGitRepo(ctx) {
		return c.listGitFiles(ctx, excludes)
	}
	return c.walkFiles(excludes)
}

// listGitFiles asks git for the scope instead of walking the filesystem:
// --cached lists tracked files and --others --exclude-standard lists untracked
// ones that .gitignore (plus .git/info/exclude and the global ignore file) does
// not cover. So the project's own declaration of what is not source decides, per
// project, with no configuration -- which is both what git-diff/pr mode already
// does when it lists untracked files, and what keeps target.exclude from having
// to re-derive every language's build directories.
//
// Paths come out relative to the process working directory, which c.git sets to
// target.path, so they are directly comparable to the walk's relative paths even
// when target.path is a subdirectory of the repository.
func (c *Collector) listGitFiles(ctx context.Context, excludes []*regexp.Regexp) (int, string, error) {
	// -z separates paths with NUL, so a filename containing a newline (or one git
	// would otherwise render quoted and escaped) survives intact.
	out, err := c.git(ctx, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return 0, "", fmt.Errorf("list files from git: %w", err)
	}
	count := 0
	var sb strings.Builder
	for _, rel := range strings.Split(out, "\x00") {
		if rel == "" {
			continue
		}
		rel = filepath.ToSlash(rel)
		if c.skipFile(rel, excludes) {
			continue
		}
		// --cached reports index entries, which outlive a file deleted from the
		// worktree. Listing a path an agent cannot open is worse than omitting it.
		if _, err := os.Lstat(filepath.Join(c.cfg.Path, rel)); err != nil {
			continue
		}
		count++
		if sb.Len() < maxMaterial {
			sb.WriteString(rel + "\n")
		}
	}
	return count, sb.String(), nil
}

// skipFile reports whether a target-relative file path is out of scope: inside
// the run's own logs directory, or matched by an exclude glob. The walk applies
// the logs check per directory (and prunes there); a git listing yields only file
// paths, so the same rule is applied by prefix here.
func (c *Collector) skipFile(rel string, excludes []*regexp.Regexp) bool {
	if c.logsExclude != "" && (rel == c.logsExclude || strings.HasPrefix(rel, c.logsExclude+"/")) {
		return true
	}
	return matchAny(excludes, rel)
}

// walkFiles is the non-git fallback: target.path may be any directory, so scope
// comes from the filesystem and only target.exclude narrows it.
func (c *Collector) walkFiles(excludes []*regexp.Regexp) (int, string, error) {
	count := 0
	var sb strings.Builder
	root := c.cfg.Path
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			// Only .git and the run's own logs dir are skipped unconditionally;
			// other hidden directories (.github, ...) stay in scope unless
			// excluded. Skipping logs keeps later rounds from reviewing the
			// run's own prompts and outputs.
			if rel != "." && (d.Name() == ".git" || rel == c.logsExclude || matchAny(excludes, rel+"/")) {
				return filepath.SkipDir
			}
			return nil
		}
		if matchAny(excludes, rel) {
			return nil
		}
		count++
		// Keep counting every match (the header reports the true total) but stop
		// growing the rendered listing once it reaches the cap; truncate appends
		// the notice, and the prompt caller is expected to read the tree directly.
		if sb.Len() < maxMaterial {
			sb.WriteString(rel + "\n")
		}
		return nil
	})
	if err != nil {
		return 0, "", err
	}
	return count, sb.String(), nil
}

func matchAny(res []*regexp.Regexp, path string) bool {
	for _, re := range res {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

// GitClean reports whether the working tree has no uncommitted changes.
// Paths in exclude (repo-relative) are ignored, so e.g. the run's own logs
// directory does not count as dirt.
func (c *Collector) GitClean(ctx context.Context, exclude ...string) (bool, error) {
	args := append([]string{"status", "--porcelain", "--"}, pathspec(exclude)...)
	out, err := c.git(ctx, args...)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

// IsGitRepo reports whether target.path is inside a git work tree.
func (c *Collector) IsGitRepo(ctx context.Context) bool {
	_, err := c.git(ctx, "rev-parse", "--git-dir")
	return err == nil
}

// AtRepoRoot reports whether target.path is the top level of its git
// repository. GitClean and Commit stage with a pathspec relative to
// target.path ("."), but git commit writes the WHOLE index, so a subdirectory
// target would miss staged changes elsewhere in the repo from its clean check
// yet still fold them into the round commit. Fix rounds therefore require
// target.path to be the repository root; --show-prefix is empty exactly there.
func (c *Collector) AtRepoRoot(ctx context.Context) (bool, error) {
	out, err := c.git(ctx, "rev-parse", "--show-prefix")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

// remoteOrigin is git's conventional default remote name, used as the
// fallback when no remote URL matches the gh base repository.
const remoteOrigin = "origin"

// ghRemote returns the git remote that corresponds to the repository gh treats
// as this checkout's base, so PR base objects are fetched from the right place
// without assuming the remote is named "origin". It matches the base repo's
// nameWithOwner against configured remote URLs, falling back to origin and then
// the sole/first remote.
func (c *Collector) ghRemote(ctx context.Context) (string, error) {
	out, err := c.git(ctx, "remote")
	if err != nil {
		return "", err
	}
	remotes := strings.Fields(out)
	if len(remotes) == 0 {
		return "", errors.New("no git remotes configured to fetch the PR base from")
	}
	if nwo, err := c.run(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner"); err == nil {
		if want := strings.ToLower(strings.TrimSpace(nwo)); want != "" {
			for _, r := range remotes {
				if u, err := c.git(ctx, "remote", "get-url", r); err == nil &&
					strings.Contains(strings.ToLower(u), want) {
					return r, nil
				}
			}
		}
	}
	for _, r := range remotes {
		if r == remoteOrigin {
			return remoteOrigin, nil
		}
	}
	return remotes[0], nil
}

// Commit stages everything except the excluded paths and commits with the
// given header and body. Returns the new commit SHA, or "" if there was
// nothing to commit.
func (c *Collector) Commit(ctx context.Context, header, body string, exclude ...string) (string, error) {
	if clean, err := c.GitClean(ctx, exclude...); err != nil {
		return "", err
	} else if clean {
		return "", nil
	}
	// Stage everything under the repo root, then unstage the excluded paths.
	// Passing excludes to `git add` is not viable: it refuses a pathspec that
	// names a .gitignore'd path even in :(exclude) form, yet a bare `git add -A`
	// still stages modifications to an already-tracked file under an ignored
	// path (git ignore rules do not stop updates to tracked files). Adding all,
	// then resetting the excludes out of the index, handles untracked-ignored
	// and tracked-ignored excludes uniformly, so no excluded path is committed.
	if out, err := c.git(ctx, "add", "-A", "--", "."); err != nil {
		return "", fmt.Errorf("git add: %w: %s", err, out)
	}
	for _, e := range exclude {
		// :(literal) so a metacharacter-bearing exclude path (e.g. "logs[1]") is
		// reset as that exact path, not a glob -- matching the exclusion pathspec.
		if out, err := c.git(ctx, "reset", "-q", "HEAD", "--", ":(literal)"+e); err != nil {
			return "", fmt.Errorf("git reset excluded %s: %w: %s", e, err, out)
		}
	}
	// Capture HEAD before committing so a cancellation landing between the commit
	// and the SHA lookup below can still be recognized as a successful commit.
	// rev-parse fails on an unborn HEAD (an empty repo); that is a valid snapshot
	// -- before stays "" and no prior commit exists to confuse committedSHA. But
	// rev-parse ALSO fails if ctx is canceled mid-lookup, and then before would be
	// "" while a prior commit DOES exist: committedSHA would compare that old HEAD
	// against "" and misreport the pre-existing commit as newly landed. So treat a
	// cancellation here as a real error and let the orchestrator reconcile the tree,
	// rather than silently trusting an empty snapshot.
	before, err := c.git(ctx, "rev-parse", "HEAD")
	if err != nil && ctx.Err() != nil {
		return "", err
	}
	before = strings.TrimSpace(before)
	// --no-verify skips pre-commit/commit-msg hooks explicitly (gitSafeConfig
	// already disables them via core.hooksPath, so this is belt-and-suspenders):
	// an attacker-supplied .git/hooks must never run during a round commit.
	if out, err := c.git(ctx, "commit", "--no-verify", "-m", header, "-m", body); err != nil {
		// `git commit` itself runs on the cancellable ctx: if ctx is canceled
		// mid-commit, KillProcessGroup SIGKILLs git, and a kill that lands AFTER git
		// has updated the ref but before it exits cleanly makes cmd.Run report
		// "signal: killed" even though the commit landed. Recover the SHA rather
		// than returning a bare commit error -- which would drop a landed commit and
		// let the run summary claim an interruption with no commit while the commit
		// sits in history and the tree looks clean to interruption reconciliation.
		if ctx.Err() != nil {
			if sha := c.committedSHA(before); sha != "" { //nolint:contextcheck // committedSHA deliberately re-reads HEAD on a fresh context: ctx is canceled but a landed commit must still be recorded
				return sha, nil
			}
		}
		return "", fmt.Errorf("git commit: %w: %s", err, out)
	}
	sha, err := c.git(ctx, "rev-parse", "HEAD")
	if err == nil {
		return strings.TrimSpace(sha), nil
	}
	// A non-cancellation lookup failure is a genuine error to surface.
	if ctx.Err() == nil {
		return "", err
	}
	// The commit above already advanced HEAD, but ctx was canceled in the window
	// before the lookup, failing rev-parse on the dead context. Recover the landed
	// commit's SHA on a fresh context rather than reporting a bare failed commit.
	if recovered := c.committedSHA(before); recovered != "" { //nolint:contextcheck // committedSHA deliberately re-reads HEAD on a fresh context: ctx is canceled but a landed commit must still be recorded
		return recovered, nil
	}
	return "", err
}

// committedSHA re-reads HEAD on a fresh (still git-op-bounded) context after a
// commit command was interrupted by ctx cancellation, returning the new SHA if
// HEAD advanced past before (the commit landed) or "" if it did not. It exists
// so both the commit command's error path and the following rev-parse's error
// path recover a commit that landed just before the SIGKILL, instead of dropping
// its SHA and misreporting the round as a failed/interrupted commit.
func (c *Collector) committedSHA(before string) string {
	after, err := c.git(context.Background(), "rev-parse", "HEAD")
	if err == nil && strings.TrimSpace(after) != before {
		return strings.TrimSpace(after)
	}
	return ""
}

// StashDirty preserves any uncommitted working-tree changes (excluding the
// given paths, e.g. the run's own logs) in a git stash and restores a clean
// tree, reporting whether anything was stashed. It is used to reconcile the
// tree after a failed fix round: the coder may have edited files before
// failing (timeout, malformed output, contract violation), and leaving those
// edits behind breaks the clean-tree invariant and blocks the next run. A
// stash is non-destructive -- the edits stay recoverable via `git stash` --
// unlike a hard reset that would discard a fix whose only fault was a bad
// output envelope.
func (c *Collector) StashDirty(ctx context.Context, message string, exclude ...string) (bool, error) {
	if clean, err := c.GitClean(ctx, exclude...); err != nil {
		return false, err
	} else if clean {
		return false, nil
	}
	// Decide how to keep each excluded path out of the stash. A negative pathspec
	// (:(exclude)e) suffices for a path git does not treat as ignored. But `git
	// stash push --include-untracked` REFUSES a pathspec under an ignored
	// directory, and ignore rules do not stop a MODIFIED TRACKED file there from
	// being swept in by the "." pathspec -- exactly the changes StashDirty
	// promises to leave untouched. So for an ignore-matched exclude, drop it from
	// the stash pathspec (its untracked content is ignored, which
	// --include-untracked skips anyway) and instead restore any tracked, modified
	// descendants from the stash afterwards. --no-index reports the ignore match
	// even when the directory also holds tracked files (a plain check-ignore
	// returns "not ignored" in that case, so the exclusion would wrongly stay in
	// the pathspec and make the whole stash fail).
	var need []string    // non-ignored excludes: kept out via pathspec
	var protect []string // tracked, modified paths under ignore-matched excludes
	for _, e := range exclude {
		if _, err := c.git(ctx, "check-ignore", "--no-index", "-q", e); err != nil {
			need = append(need, e)
			continue
		}
		// --no-renames keeps both sides of a rename as distinct paths, so the
		// deleted old side is protected and reproduced too (rename detection would
		// otherwise collapse it and leave the old file resurrected after restore).
		changed, err := c.git(ctx, "diff", "HEAD", "--name-only", "--no-renames", "-z", "--", e)
		if err != nil {
			return false, fmt.Errorf("list tracked changes under excluded %s: %w", e, err)
		}
		for _, p := range strings.Split(strings.Trim(changed, "\x00"), "\x00") {
			if p != "" {
				protect = append(protect, p)
			}
		}
	}
	args := append([]string{"stash", "push", "--include-untracked", "-m", message, "--"}, pathspec(need)...)
	if out, err := c.git(ctx, args...); err != nil {
		return false, fmt.Errorf("git stash: %w: %s", err, out)
	}
	// Restore tracked changes under ignore-matched excludes that the stash swept
	// up, so those paths are left exactly as they were (still recoverable from
	// the stash). Reproduce each path's full pre-stash state from the stash's own
	// trees, keeping index and worktree independent so a staged-but-then-modified
	// path keeps both sides: the index from the index commit (stash@{0}^2) and
	// the worktree from the worktree commit (stash@{0}). A path absent from a
	// tree was deleted there at stash time (a plain deletion, or the old side of
	// a rename), so reproduce the removal instead of resurrecting the file to its
	// HEAD content -- a plain `checkout stash@{0} -- p` cannot delete and fails
	// outright on such a path, and a checkout+reset would collapse a staged
	// change to unstaged.
	for _, p := range protect {
		if stashed, err := c.restoreProtectedPath(ctx, p); err != nil {
			return stashed, err
		}
	}
	// A clean `git stash` exit does not guarantee the tree actually became clean:
	// a top-level stash does not capture modifications inside a submodule working
	// tree, so reported dirt can outlive the stash. Verify the tree is clean
	// (under the same exclusions) before reporting a successful reconciliation --
	// otherwise interruption/salvage handling would treat a still-dirty tree as
	// clean and the next run would be blocked by dirt this call claimed to clear.
	if clean, err := c.GitClean(ctx, exclude...); err != nil {
		return false, err
	} else if !clean {
		return false, errors.New("git stash exited cleanly but uncommitted changes remain (e.g. modifications inside a submodule working tree, which a top-level stash does not capture); reconcile the tree manually")
	}
	return true, nil
}

// pathspec renders repo-relative exclude paths as git pathspec arguments:
// everything under ".", minus the excluded trees. The exclude specs carry the
// `literal` magic so a directory name containing pathspec metacharacters (e.g.
// a logs dir literally named "logs[1]") is matched as that exact path rather
// than interpreted as a glob -- otherwise its files could slip past the
// exclusion and be swept into a diff, clean check, or round commit.
func pathspec(exclude []string) []string {
	specs := []string{"."}
	for _, e := range exclude {
		specs = append(specs, ":(exclude,literal)"+e)
	}
	return specs
}

// restoreProtectedPath reproduces one excluded path's pre-stash index and
// worktree state from the stash's own trees, keeping the two independent so a
// staged-but-then-modified path keeps both sides: the index from the index
// commit (stash@{0}^2), the worktree from the worktree commit (stash@{0}). A
// path absent from a tree was deleted there at stash time (a plain deletion, or
// the old side of a rename), so its removal is reproduced rather than the file
// being resurrected to HEAD content. The returned bool is the value StashDirty
// should report as "stashed" alongside a non-nil error.
func (c *Collector) restoreProtectedPath(ctx context.Context, p string) (bool, error) {
	// An operational failure here (cancellation, timeout, unreadable stash
	// object) must abort restoration -- never be mistaken for "the path was
	// deleted at stash time" and silently git rm / os.Remove an excluded path.
	inIndex, err := c.pathInTree(ctx, "stash@{0}^2", p)
	if err != nil {
		return false, fmt.Errorf("look up excluded path %s in stash index: %w", p, err)
	}
	if inIndex {
		if out, err := c.git(ctx, "restore", "--source=stash@{0}^2", "--staged", "--", p); err != nil {
			return true, fmt.Errorf("restore excluded path %s index after stash: %w: %s", p, err, out)
		}
	} else if out, err := c.git(ctx, "rm", "-q", "--cached", "--ignore-unmatch", "--", p); err != nil {
		return true, fmt.Errorf("stage removal of excluded path %s after stash: %w: %s", p, err, out)
	}
	inWorktree, err := c.pathInTree(ctx, "stash@{0}", p)
	if err != nil {
		return false, fmt.Errorf("look up excluded path %s in stash worktree: %w", p, err)
	}
	if inWorktree {
		if out, err := c.git(ctx, "restore", "--source=stash@{0}", "--worktree", "--", p); err != nil {
			return true, fmt.Errorf("restore excluded path %s worktree after stash: %w: %s", p, err, out)
		}
	} else if err := os.Remove(filepath.Join(c.cfg.Path, p)); err != nil && !os.IsNotExist(err) {
		return true, fmt.Errorf("remove excluded path %s from worktree after stash: %w", p, err)
	}
	return true, nil
}

// pathInTree reports whether p exists in the given git tree-ish. It uses
// ls-tree rather than `cat-file -e tree:p` so a confirmed-absent path is
// distinguishable from an operational failure: ls-tree exits 0 whether or not
// the path is present (empty output means absent), so a non-nil error is a real
// failure to surface -- a cancellation, timeout, or unreadable stash object --
// not a deletion to reproduce. `cat-file -e` conflated the two into a non-zero
// exit, so a failed lookup during excluded-path restoration would be read as
// "deleted" and the caller would git rm / os.Remove an excluded path.
func (c *Collector) pathInTree(ctx context.Context, tree, p string) (bool, error) {
	out, err := c.git(ctx, "ls-tree", "-r", "--name-only", tree, "--", p)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// gitSafeConfig neutralizes the repo-controlled git settings that make git
// itself execute attacker code, prepended to every git invocation as -c
// overrides (they beat any value in the target's .git/config). A target's .git
// is not always a clean `git clone` -- an extracted archive or a crafted
// checkout can ship its own .git/hooks and .git/config -- and running git there
// would otherwise run those programs with fixpoint's privileges and inherited
// environment (ANTHROPIC_API_KEY, GITHUB_TOKEN, ...), even in a review-only
// git-diff run that never passes the fix-round trust gate. core.hooksPath is
// pointed at a directory that cannot hold an executable hook, and core.fsmonitor
// (a command status/diff/ls-files would spawn) is disabled. This does not cover
// gitattributes-driven filters/diff-drivers or gh's own internal git calls; a
// truly untrusted .git should still be reviewed under an external sandbox.
var gitSafeConfig = []string{
	"-c", "core.hooksPath=/dev/null",
	"-c", "core.fsmonitor=false",
}

// gitHardenedEnv returns the child environment for every git/gh subprocess:
// os.Environ() plus GIT_CONFIG_COUNT/GIT_CONFIG_KEY_n/GIT_CONFIG_VALUE_n entries
// that apply gitSafeConfig to EVERY git process, including the ones gh spawns
// internally (which the -c overrides in c.git never reach). git reads these env
// entries exactly as if they were -c key=value overrides. Any GIT_CONFIG_COUNT
// already in the environment is merged rather than clobbered: its existing
// KEY/VALUE entries are preserved and ours are appended after them, so a caller
// that legitimately set its own overrides keeps them.
func gitHardenedEnv() []string {
	base := 0
	env := make([]string, 0, len(os.Environ())+len(gitSafeConfig))
	for _, e := range os.Environ() {
		if v, ok := strings.CutPrefix(e, "GIT_CONFIG_COUNT="); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				base = n
			}
			continue // dropped; re-added below with the merged count
		}
		env = append(env, e)
	}
	n := base
	// gitSafeConfig is a flat [-c, key=value, -c, key=value, ...] slice; turn each
	// key=value into a GIT_CONFIG_KEY_n / GIT_CONFIG_VALUE_n pair.
	for i := 0; i+1 < len(gitSafeConfig); i += 2 {
		key, val, _ := strings.Cut(gitSafeConfig[i+1], "=")
		env = append(env,
			fmt.Sprintf("GIT_CONFIG_KEY_%d=%s", n, key),
			fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", n, val),
		)
		n++
	}
	if n > 0 {
		env = append(env, fmt.Sprintf("GIT_CONFIG_COUNT=%d", n))
	}
	return env
}

// unsafeConfigKey reports whether a repo-local git config key names a setting
// that makes a later git command execute a repo-controlled program: a per-name
// content filter (filter.<name>.clean/smudge/process) runs when `git diff`
// normalizes a worktree file, core.sshCommand / a credential helper run during a
// fetch, and a signing program (gpg.program, gpg.<format>.program, gpg.ssh.program)
// runs when a fix round's `git commit` signs. On an untrusted checkout that
// shipped its own .git/config these turn a git pass into code execution with
// fixpoint's inherited environment, so callers refuse rather than run git against
// such a target. gitSafeConfig cannot cover the filter/ssh/credential names (they
// are dynamic); a signing program COULD be force-disabled with
// -c commit.gpgSign=false, but refusing preserves legitimate signed round commits
// for a trusted operator instead of silently dropping their signatures.
func unsafeConfigKey(key string) bool {
	switch {
	case strings.HasPrefix(key, "filter.") &&
		(strings.HasSuffix(key, ".clean") || strings.HasSuffix(key, ".smudge") || strings.HasSuffix(key, ".process")):
		return true
	case key == "core.sshcommand":
		return true
	case strings.HasPrefix(key, "credential.") && strings.HasSuffix(key, ".helper"):
		return true
	case strings.HasPrefix(key, "gpg.") && strings.HasSuffix(key, ".program"):
		return true
	}
	return false
}

// UnsafeConfig returns the sorted, de-duplicated repo-local config keys present
// in the target that unsafeConfigKey flags -- the execution-capable settings
// fixpoint cannot neutralize. It reads only .git/config (--local), which runs no
// filter or hook, so it is safe to call before any worktree-touching command.
func (c *Collector) UnsafeConfig(ctx context.Context) ([]string, error) {
	// -z: NUL-separated entries, each "key\nvalue"; a real repo always has at
	// least the default core.* entries, so a git repo yields a non-error result.
	out, err := c.git(ctx, "config", "--local", "--list", "-z")
	if err != nil {
		return nil, fmt.Errorf("inspect target repo-local git config: %w", err)
	}
	seen := map[string]bool{}
	var keys []string
	for _, entry := range strings.Split(out, "\x00") {
		if entry == "" {
			continue
		}
		key := entry
		if nl := strings.IndexByte(entry, '\n'); nl >= 0 {
			key = entry[:nl]
		}
		if unsafeConfigKey(key) && !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (c *Collector) git(ctx context.Context, args ...string) (string, error) {
	return c.run(ctx, "git", append(append([]string{}, gitSafeConfig...), args...)...)
}

func (c *Collector) run(ctx context.Context, name string, args ...string) (string, error) {
	// Honor the run's context so Ctrl-C actually terminates the subprocess
	// (exec.Command ignored it, so a hung git/gh survived cancellation and the
	// orchestrator could never reach its next ctx check), and bound the
	// operation so a wedged credential prompt or commit hook cannot hang forever.
	ctx, cancel := context.WithTimeout(ctx, gitOpTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = c.cfg.Path
	// Harden the environment so the git processes gh spawns internally -- which
	// never see our -c overrides -- still get gitSafeConfig applied. gh pr checkout
	// runs `git checkout`, and a repo whose config points core.hooksPath into the
	// worktree (e.g. .githooks) would otherwise run a PR-supplied post-checkout hook
	// with fixpoint's inherited tokens, even in a review-only PR run before any
	// sandbox. GIT_CONFIG_COUNT/KEY/VALUE make git treat these as -c overrides.
	cmd.Env = gitHardenedEnv()
	// Put the child in its own process group and SIGKILL the whole group on
	// cancel/timeout, so a commit hook or other descendant it spawned cannot
	// keep the operation alive after we try to stop it.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return agent.KillProcessGroup(cmd) }
	// Bound how long Wait blocks after the group kill, so a descendant holding
	// the output pipe open cannot wedge Wait indefinitely.
	cmd.WaitDelay = 2 * time.Second
	// Stream stdout and stderr into SEPARATE bounded buffers rather than one:
	// callers parse the returned string as a machine-readable value (a SHA, a PR
	// base OID, a remote list, a filename list), and a successful git/gh command
	// that also writes a warning or notice to stderr would otherwise fold that
	// text into the parsed value -- e.g. a gh notice alongside baseRefOid makes PR
	// preparation hand the combined string to git as an invalid object name. On
	// success we return stdout only; stderr surfaces solely in the error. Bounding
	// each buffer still caps memory on a large diff/listing (each exec copy
	// goroutine writes its own buffer, so there is no Write x Write race), and
	// BoundedBuffer locks to guard String() below against an abandoned copy
	// goroutine after a WaitDelay expiry.
	outBuf := agent.NewBoundedBuffer(maxRunOutput, agent.TruncationMarker(maxRunOutput))
	errBuf := agent.NewBoundedBuffer(maxRunOutput, agent.TruncationMarker(maxRunOutput))
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	stdout := outBuf.String()
	if err != nil {
		return stdout, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return stdout, nil
}

// maxRunOutput bounds the combined stdout+stderr a single git/gh subprocess may
// buffer in memory. It sits well above maxMaterial so a diff large enough to be
// truncated still yields more than the prompt cap, letting truncate mark the
// material as cut; anything beyond is discarded here so memory stays bounded
// regardless of repository or diff size.
const maxRunOutput = 4 << 20 // 4 MB

func truncate(s string) string {
	if len(s) <= maxMaterial {
		return s
	}
	// Back off to a rune boundary: cutting inside a multibyte character would
	// embed invalid UTF-8 into the prompt.
	cut := maxMaterial
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "\n\n[... material truncated at " + agent.HumanSize(maxMaterial) + " -- read the repository directly for the rest ...]"
}

// compileGlobs converts **-style globs to regexps: ** matches across path
// separators, * and ? within a segment.
func compileGlobs(globs []string) ([]*regexp.Regexp, error) {
	res := make([]*regexp.Regexp, 0, len(globs))
	for _, g := range globs {
		var sb strings.Builder
		sb.WriteString("^")
		i := 0
		for i < len(g) {
			switch {
			case strings.HasPrefix(g[i:], "**/"):
				sb.WriteString(`(.*/)?`)
				i += 3
			case strings.HasPrefix(g[i:], "**"):
				sb.WriteString(`.*`)
				i += 2
			case g[i] == '*':
				sb.WriteString(`[^/]*`)
				i++
			case g[i] == '?':
				sb.WriteString(`[^/]`)
				i++
			default:
				r, size := utf8.DecodeRuneInString(g[i:])
				sb.WriteString(regexp.QuoteMeta(string(r)))
				i += size
			}
		}
		sb.WriteString("$")
		re, err := regexp.Compile(sb.String())
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", g, err)
		}
		res = append(res, re)
	}
	return res, nil
}
