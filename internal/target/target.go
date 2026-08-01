// Package target collects the material each review round runs against, and
// provides the git operations the loop depends on (base pinning, clean-tree
// checks, per-round commits).
package target

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
		// fed back as review material. The same pathspec also drops target.exclude
		// and the mandatory credential patterns, whose content the diff would
		// otherwise embed verbatim -- see collectPathspec.
		args = append(args, "--")
		args = append(args, c.collectPathspec()...)
		diff, err := c.git(ctx, args...)
		if err != nil {
			return "", err
		}
		// A failure here (e.g. an invalid core.excludesFile) must not be
		// swallowed: silently treating it as "no untracked files" would drop
		// them from review without reporting the collection was incomplete.
		lsArgs := []string{"ls-files", "--others", "--exclude-standard", "--"}
		lsArgs = append(lsArgs, c.collectPathspec()...)
		untracked, err := c.git(ctx, lsArgs...)
		if err != nil {
			return "", fmt.Errorf("list untracked files: %w", err)
		}
		var sb strings.Builder
		if c.baseSHA != "" {
			fmt.Fprintf(&sb, "Diff against pinned base %s:\n\n", shortSHA(c.baseSHA))
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

// shortSHA abbreviates a commit SHA for display without ever slicing past its
// end. The pinned base comes from git's own output, so it is normally 40 hex
// characters -- but an unguarded sha[:12] would panic on anything shorter (a
// truncated read, a git wrapper on PATH), turning a cosmetic detail into a lost
// run. Mirrors the guard the summary writer applies to round commit SHAs.
func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
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
	// Only a CONFIRMED non-worktree falls back to the filesystem walk. The two
	// paths do not review the same set of files -- the walk ignores .gitignore --
	// so treating a canceled, timed-out, or otherwise failed probe as "not a git
	// repository" would silently change the review's scope, and on Ctrl-C would
	// spend the cancellation walking the tree instead of stopping.
	isRepo, err := c.IsGitRepo(ctx)
	if err != nil {
		return 0, "", err
	}
	if isRepo {
		return c.listGitFiles(ctx, excludes)
	}
	return c.walkFiles(ctx, excludes)
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
//
// The listing is STREAMED rather than read through c.git: that path caps stdout
// at maxRunOutput, which is right for output that only ever lands in an error
// message and wrong for output that is parsed. A repository whose path list
// exceeds the cap would lose its tail silently, the truncation marker would
// arrive as one final pseudo-path that Lstat rejects and skips, and the header
// would confidently report a total that undercounts the tree -- the exact
// silently-narrowed scope listFiles' denylist-only design exists to avoid.
func (c *Collector) listGitFiles(ctx context.Context, excludes []*regexp.Regexp) (int, string, error) {
	count := 0
	var sb strings.Builder
	// -z separates paths with NUL, so a filename containing a newline (or one git
	// would otherwise render quoted and escaped) survives intact.
	err := c.gitScanNUL(ctx, func(rel string) {
		if rel == "" {
			return
		}
		rel = filepath.ToSlash(rel)
		if c.skipFile(rel, excludes) {
			return
		}
		// --cached reports index entries, which outlive a file deleted from the
		// worktree. Listing a path an agent cannot open is worse than omitting it.
		if _, err := os.Lstat(filepath.Join(c.cfg.Path, rel)); err != nil {
			return
		}
		// Keep counting every match (the header reports the true total) but stop
		// growing the rendered listing once it reaches the cap.
		count++
		if sb.Len() < maxMaterial {
			sb.WriteString(rel + "\n")
		}
	}, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return 0, "", fmt.Errorf("list files from git: %w", err)
	}
	return count, sb.String(), nil
}

// maxGitPath bounds a single NUL-delimited entry gitScanNUL will accept. Well
// past any real path length; an entry longer than this is surfaced as an error
// rather than silently truncated, so a caller counting entries never counts a
// fragment as a path.
const maxGitPath = 1 << 20

// gitScanNUL runs a git command whose stdout is a NUL-delimited list and calls
// fn once per entry as it arrives, so the caller can count an unbounded listing
// without holding it in memory and without the diagnostic output cap c.run
// applies. Everything else -- the safe-config overrides, the hardened
// environment, the operation timeout, the process-group kill, the WaitDelay --
// matches c.run, since the reason each of those exists does not change with how
// stdout is consumed. stderr is still buffered, for the error message.
func (c *Collector) gitScanNUL(ctx context.Context, fn func(string), args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, gitOpTimeout)
	defer cancel()
	full := append(append([]string{}, gitSafeConfig...), args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = c.cfg.Path
	cmd.Env = gitHardenedEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return agent.KillProcessGroup(cmd) }
	cmd.WaitDelay = 2 * time.Second
	errBuf := agent.NewBoundedBuffer(maxRunOutput, agent.TruncationMarker(maxRunOutput))
	cmd.Stderr = errBuf
	desc := "git " + strings.Join(args, " ")
	// Our own pipe rather than cmd.StdoutPipe, whose read end only cmd.Wait can
	// close -- and Wait cannot be called until the scan finishes. A descendant git
	// left behind holding the write end would then block the scan for the whole
	// gitOpTimeout. Owning both ends lets the waiter below unblock the scan as soon
	// as the leader is gone, and lets the scan drop the pipe on any exit path.
	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("%s: %w", desc, err)
	}
	// Closing twice is harmless (os.File guards it); the defer covers the early
	// returns, the explicit close below ends the scan.
	defer func() { _ = pr.Close() }()
	cmd.Stdout = pw
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		return fmt.Errorf("%s: %w", desc, err)
	}
	// Drop the parent's write end: the child and its descendants now hold the only
	// ones, so EOF on pr means every process that could still write is gone.
	_ = pw.Close()
	waited := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		// Own the whole subprocess lifecycle, not just the leader, exactly as
		// agent.Run and verify.runOne do: cmd.Cancel fires only on cancellation, so
		// a git that exits SUCCESSFULLY after spawning a child leaves it alive in our
		// process group -- free to mutate the repository concurrently with a later
		// clean-tree check or round commit, and, if it inherited stdout, to stall the
		// scan below until the operation timeout. SIGKILL the group on every exit
		// path; it is a no-op once the group is empty, which is the common case.
		_ = agent.KillProcessGroup(cmd)
		waited <- err
	}()
	sc := bufio.NewScanner(pr)
	sc.Buffer(make([]byte, 0, 64<<10), maxGitPath)
	sc.Split(scanNUL)
	for sc.Scan() {
		fn(sc.Text())
	}
	scanErr := sc.Err()
	// Stop reading on every exit path. A scan that stopped early would otherwise
	// leave git blocked on a full pipe; closing the read end fails its next write
	// instead, and the scan error below outranks the exit status that produces.
	_ = pr.Close()
	werr := <-waited
	// A scan failure comes first: it means the listing was not read in full, which
	// git's own exit status cannot tell us.
	if scanErr != nil {
		return fmt.Errorf("%s: reading output: %w", desc, scanErr)
	}
	if werr != nil {
		return fmt.Errorf("%s: %w: %s", desc, werr, strings.TrimSpace(errBuf.String()))
	}
	return nil
}

// scanNUL is a bufio.SplitFunc for git's -z output: entries separated by NUL.
// git terminates every entry, so a trailing fragment at EOF means the output was
// cut short; it is still emitted rather than dropped, and the short read shows up
// as a nonzero git exit.
func scanNUL(data []byte, atEOF bool) (int, []byte, error) {
	if i := bytes.IndexByte(data, 0); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
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
//
// It observes ctx: a large tree can take a long time to walk, and Ctrl-C has to
// reach the one collection path that runs no subprocess of its own.
func (c *Collector) walkFiles(ctx context.Context, excludes []*regexp.Regexp) (int, string, error) {
	count := 0
	var sb strings.Builder
	root := c.cfg.Path
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
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
//
// It returns an error rather than a bare false for an OPERATIONAL failure --
// cancellation, the operation timeout, an unreadable .git -- because every caller
// does something materially different with "no git here": directory collection
// switches to a filesystem walk that ignores .gitignore, and the untrusted-config
// guard skips its check entirely. Collapsing a failed probe into false would let a
// canceled Ctrl-C or a wedged git silently change what gets reviewed, and silently
// skip a security check, with no diagnostic.
func (c *Collector) IsGitRepo(ctx context.Context) (bool, error) {
	out, err := c.git(ctx, "rev-parse", "--is-inside-work-tree")
	if err == nil {
		return strings.TrimSpace(out) == "true", nil
	}
	// A confirmed answer: git ran and said this is not a repository (exit 128), or
	// git is not installed at all -- in neither case is there a work tree to list
	// from, so the walk is the right scope and not a silent diversion.
	if ctx.Err() == nil && (notARepository(err) || errors.Is(err, exec.ErrNotFound)) {
		return false, nil
	}
	return false, fmt.Errorf("determine whether %s is a git work tree: %w", c.cfg.Path, err)
}

// notARepository reports whether a failed git command failed because there is no
// repository, as opposed to failing for any other reason. git answers that with
// exit 128 plus a specific message; the exit status alone covers every fatal, so
// both are required before a failure is read as an answer.
func notARepository(err error) bool {
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 128 {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not a git repository") || strings.Contains(msg, "not a work tree")
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
					remoteIdentity(u) == want {
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

// remoteIdentity reduces a git remote URL to the lowercased "owner/repo" identity
// gh reports as nameWithOwner, or "" when the URL is not of that shape (a local
// path, a file:// URL, a nested path). It exists so ghRemote compares identities
// for EQUALITY: a substring test on the URL matches acme/widget inside
// acme/widget-fork or acme/widgets, and if such a remote sorts first the PR base
// would be fetched from the wrong repository.
//
// Both supported URL forms are handled: scheme://[user[:pass]@]host[:port]/owner/repo[.git]
// and the scp-like [user@]host:owner/repo[.git]. Only the optional .git suffix is
// stripped, since it is the only one git itself treats as decoration.
func remoteIdentity(raw string) string {
	s := strings.TrimSpace(raw)
	var path string
	if _, rest, ok := strings.Cut(s, "://"); ok {
		// Everything up to the first slash is the authority (credentials, host, port).
		_, p, ok := strings.Cut(rest, "/")
		if !ok {
			return ""
		}
		path = p
	} else if _, p, ok := strings.Cut(s, ":"); ok {
		// scp-like syntax has no port, so the whole remainder is the path.
		path = p
	} else {
		return ""
	}
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	owner, repo, ok := strings.Cut(strings.Trim(path, "/"), "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return ""
	}
	return strings.ToLower(owner + "/" + repo)
}

// Commit stages everything except the excluded paths and commits with the
// given header and body. Returns the new commit SHA, or "" if there was
// nothing to commit.
//
// The excluded paths are left exactly as they were, in the worktree AND in the
// index. A non-nil error with a non-empty SHA means the commit landed but that
// restoration did not; the error names the SHA, since callers treat an error as
// "no commit".
func (c *Collector) Commit(ctx context.Context, header, body string, exclude ...string) (string, error) {
	if clean, err := c.GitClean(ctx, exclude...); err != nil {
		return "", err
	} else if clean {
		return "", nil
	}
	// Snapshot the excluded paths' index entries before staging touches them.
	// GitClean deliberately ignores the exclusion, so a run may legitimately start
	// with STAGED changes under an excluded path -- and the add/reset pair below
	// walks the caller's real index: `git add -A` replaces a staged-only version
	// with the worktree version, and the reset then collapses the entry to HEAD.
	// Both are needed to keep the excluded path out of the commit, so the staged
	// version is preserved here and put back afterwards instead.
	staged, err := c.stagedExcluded(ctx, exclude)
	if err != nil {
		return "", err
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
	sha, cerr := c.commitStaged(ctx, header, body)
	// Put the excluded paths' pre-run index entries back, whether or not the commit
	// itself succeeded. The commit above carried their HEAD version, so restoring the
	// entries afterwards reproduces exactly the staged diff the run started with,
	// without ever committing an excluded path.
	// Cancellation must not be why the entries stay collapsed: on a dead context every
	// git command fails instantly, and commitStaged may well have recovered a commit
	// that landed. Restore on a fresh context then, still bounded per operation by
	// gitOpTimeout, exactly as committedSHA re-reads HEAD.
	rctx := ctx //nolint:contextcheck // deliberate fresh context below: ctx may be canceled, but the excluded paths' index entries must still be put back
	if rctx.Err() != nil {
		rctx = context.Background()
	}
	if rerr := c.restoreStagedExcluded(rctx, exclude, staged); rerr != nil {
		if cerr != nil {
			return "", fmt.Errorf("%w; and the staged state of the excluded path(s) could not be restored: %w", cerr, rerr)
		}
		// The commit landed: name it, because the returned error means the caller
		// cannot report the SHA itself.
		return sha, fmt.Errorf("commit %s landed but the staged state of the excluded path(s) could not be restored: %w", shortSHA(sha), rerr)
	}
	return sha, cerr
}

// stagedExcluded returns the raw `ls-files --stage -z` records for the index
// entries currently under the excluded paths, or "" when there is nothing whose
// loss the add/reset pair in Commit could cause. It is deliberately empty in the
// ordinary case -- an index that matches HEAD there is reproduced exactly by the
// reset -- so a normal round commit runs no index surgery at all.
func (c *Collector) stagedExcluded(ctx context.Context, exclude []string) (string, error) {
	if len(exclude) == 0 {
		return "", nil
	}
	// No HEAD yet (an unborn branch): the reset in Commit fails before any commit
	// lands, so there is no round commit whose staging could destroy anything.
	if !c.hasHEAD(ctx) {
		return "", nil
	}
	specs := literalPathspec(exclude)
	diff, err := c.git(ctx, append([]string{"diff-index", "--cached", "--name-only", "-z", "HEAD", "--"}, specs...)...)
	if err != nil {
		return "", fmt.Errorf("check for staged changes under the excluded path(s): %w: %s", err, diff)
	}
	if strings.Trim(diff, "\x00") == "" {
		return "", nil
	}
	out, err := c.git(ctx, append([]string{"ls-files", "--stage", "-z", "--"}, specs...)...)
	if err != nil {
		return "", fmt.Errorf("read index entries for the excluded path(s): %w: %s", err, out)
	}
	return out, nil
}

// hasHEAD reports whether HEAD resolves to a commit. A false covers both an unborn
// branch and a failed lookup, which is deliberate here: the caller only uses it to
// decide whether there is any pre-commit index state worth preserving, and every
// later git command in Commit surfaces a real failure on its own.
func (c *Collector) hasHEAD(ctx context.Context) bool {
	out, err := c.git(ctx, "rev-parse", "--verify", "-q", "HEAD")
	return err == nil && strings.TrimSpace(out) != ""
}

// restoreStagedExcluded puts back the index entries stagedExcluded captured. The
// removals and the re-adds go through ONE `update-index --index-info` run, so the
// index is rewritten once: a partial restore is what would actually lose the
// staged version. A mode of 0 tells --index-info to drop a path, which is how an
// entry staging left behind (an untracked file under a non-ignored exclusion) is
// removed, and how a staged deletion is reproduced rather than resurrected.
func (c *Collector) restoreStagedExcluded(ctx context.Context, exclude []string, staged string) error {
	if staged == "" {
		return nil
	}
	current, err := c.git(ctx, append([]string{"ls-files", "--stage", "-z", "--"}, literalPathspec(exclude)...)...)
	if err != nil {
		return fmt.Errorf("read index entries for the excluded path(s): %w: %s", err, current)
	}
	var sb strings.Builder
	for _, rec := range strings.Split(current, "\x00") {
		// Each record is "<mode> <sha> <stage>\t<path>"; the path is everything after
		// the first tab, verbatim under -z.
		if _, path, ok := strings.Cut(rec, "\t"); ok && path != "" {
			sb.WriteString("0 " + nullOID + "\t" + path + "\x00")
		}
	}
	sb.WriteString(staged) // already NUL-terminated records
	if out, err := c.gitInput(ctx, sb.String(), "update-index", "-z", "--index-info"); err != nil {
		return fmt.Errorf("git update-index: %w: %s", err, out)
	}
	return nil
}

// nullOID is git's all-zero object name, the placeholder --index-info wants
// beside a mode of 0 (which is what actually removes the path).
const nullOID = "0000000000000000000000000000000000000000"

// literalPathspec renders exclude paths as :(literal) pathspecs, matching the
// exclusion used everywhere else so a path with pathspec metacharacters (a logs
// dir literally named "logs[1]") is treated as that exact path.
func literalPathspec(exclude []string) []string {
	specs := make([]string, 0, len(exclude))
	for _, e := range exclude {
		specs = append(specs, ":(literal)"+e)
	}
	return specs
}

// HeadSHA returns the current commit, or "" on an unborn branch (a repository
// with no commits yet). The empty string is a valid answer rather than an error:
// the caller uses it as the base to squash back to, and "nothing committed yet"
// is a legitimate starting point.
func (c *Collector) HeadSHA(ctx context.Context) (string, error) {
	out, err := c.git(ctx, "rev-parse", "--verify", "-q", "HEAD")
	if err != nil {
		// `rev-parse --verify -q HEAD` fails ONLY because HEAD does not resolve, which
		// on a repository fixpoint has already locked and checked means an unborn
		// branch -- the empty-string answer documented above. A cancellation is the one
		// other way to get here, and the caller sees it on its next ctx check.
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// SquashSince replaces every commit after base with a single commit carrying the
// same tree, for loop.commit_policy. It is a pure regrouping: `reset --soft` moves
// the branch and leaves the index and worktree exactly as they are, so the content
// committed here is byte-for-byte the content the per-fix commits already put
// through the verify gate one at a time.
//
// base == "" squashes back to an unborn branch, which is why HeadSHA reports that
// state as an empty string rather than an error.
func (c *Collector) SquashSince(ctx context.Context, base, header, body string) (string, error) {
	if base == "" {
		// No commit to reset onto: unstage nothing, just move the branch pointer off
		// its commits by pointing HEAD at an empty tree's parent -- i.e. delete the
		// ref and re-commit the index.
		if out, err := c.git(ctx, "update-ref", "-d", "HEAD"); err != nil {
			return "", fmt.Errorf("git update-ref -d HEAD: %w: %s", err, out)
		}
	} else if out, err := c.git(ctx, "reset", "--soft", base); err != nil {
		return "", fmt.Errorf("git reset --soft %s: %w: %s", base, err, out)
	}
	// The index already holds everything the squashed commits staged, so commit it
	// directly rather than re-running Commit's add/exclude dance: re-staging would
	// pick up anything that arrived in the worktree since, which no verify pass has
	// seen.
	return c.commitStaged(ctx, header, body)
}

// commitStaged commits whatever is already in the index, returning the new SHA. It
// is separate so Commit's excluded-path restoration runs on every exit from the
// commit itself, including the cancellation-recovery paths below, and so
// SquashSince can reuse it without re-staging.
func (c *Collector) commitStaged(ctx context.Context, header, body string) (string, error) {
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

// collectPathspec is pathspec plus the target.exclude globs and the mandatory
// credential patterns, for the git commands that COLLECT review material.
//
// listFiles applies EffectiveExcludes to directory mode; without this the git
// modes applied neither it nor target.exclude, and they are where it matters
// most: a directory listing only ever names a path, whereas `git diff` embeds the
// full file CONTENT into every reviewer prompt, into each agent CLI's arguments
// or stdin, and into the .prompt/.raw artifacts on disk. A PR that adds a .env or
// a deploy key would hand that key material verbatim to every reviewer, and
// agent.RedactSecrets is shape-based and best-effort. The `glob` magic gives git
// the same `**/` semantics compileGlobs gives the directory walk.
//
// The positive "." spec comes from pathspec: collection is scoped to target.path,
// the same scope the clean check and the round commit use.
func (c *Collector) collectPathspec() []string {
	specs := pathspec(c.excludes())
	for _, g := range c.cfg.EffectiveExcludes() {
		specs = append(specs, ":(exclude,glob)"+g)
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
//
// protocol.ext.allow=never kills the ext:: helper protocol, whose URL IS a shell
// command git runs. git's own default for it is already "never", but that default
// is CONFIGURABLE: a crafted .git/config setting protocol.ext.allow=always turns
// the transport back on for exactly the user-initiated fetches fixpoint performs
// (`gh pr checkout`, the base-object fetch), and remote.<name>.url or a repo-local
// url.<ext-url>.insteadOf then routes an ordinary-looking remote into it.
// unsafeConfigKey refuses such a rewrite but does not flag remote.<name>.url; this
// pin beats the repo's value outright and closes the whole class rather than one
// key at a time -- including the shapes that reach git through gh's internal
// calls, since gitHardenedEnv exports these as GIT_CONFIG_*.
var gitSafeConfig = []string{
	"-c", "core.hooksPath=/dev/null",
	"-c", "core.fsmonitor=false",
	"-c", "protocol.ext.allow=never",
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
// normalizes a worktree file, core.sshCommand / a credential helper / core.askPass
// run during a fetch (git resolves a credential prompt as GIT_ASKPASS, then
// core.askPass, then SSH_ASKPASS, then the terminal -- so core.askPass is the one
// that fires in the non-interactive context fixpoint runs Prepare's
// `git fetch <remote> <baseOid>` in), and a signing program (gpg.program,
// gpg.<format>.program, gpg.ssh.program)
// runs when a fix round's `git commit` signs. On an untrusted checkout that
// shipped its own .git/config these turn a git pass into code execution with
// fixpoint's inherited environment, so callers refuse rather than run git against
// such a target. gitSafeConfig cannot cover the filter/ssh/credential names (they
// are dynamic); a signing program COULD be force-disabled with
// -c commit.gpgSign=false, but refusing preserves legitimate signed round commits
// for a trusted operator instead of silently dropping their signatures.
//
// The TRANSPORT settings below are the same class on the fetch path this guard
// exists to protect -- pr mode's `gh pr checkout` and Prepare's `git fetch` of the
// base object:
//
//   - url.<base>.insteadOf / pushInsteadOf rewrite the URL git actually contacts.
//     `[url "ext::sh -c <cmd>"] insteadOf = https://github.com/` leaves
//     remote.origin.url an ordinary GitHub URL (so gh still resolves the repo)
//     while every fetch goes through the ext:: helper protocol, which git runs as
//     a shell command. gitSafeConfig additionally pins protocol.ext.allow=never so
//     this particular shape is dead even before the guard sees it, but the rewrite
//     can target other transports too and the key belongs on the list.
//   - core.gitProxy is a program git runs for git:// transport.
//   - remote.<name>.uploadPack/receivePack/proxy are programs run for local and
//     file transports.
//
// The names are dynamic (subsections), so like the filters they cannot be
// neutralized by a -c override; refusing is the answer.
func unsafeConfigKey(key string) bool {
	switch {
	case strings.HasPrefix(key, "filter.") &&
		(strings.HasSuffix(key, ".clean") || strings.HasSuffix(key, ".smudge") || strings.HasSuffix(key, ".process")):
		return true
	case key == "core.sshcommand" || key == "core.gitproxy" || key == "core.askpass":
		return true
	case strings.HasPrefix(key, "credential.") && strings.HasSuffix(key, ".helper"):
		return true
	case strings.HasPrefix(key, "gpg.") && strings.HasSuffix(key, ".program"):
		return true
	case strings.HasPrefix(key, "url.") &&
		(strings.HasSuffix(key, ".insteadof") || strings.HasSuffix(key, ".pushinsteadof")):
		return true
	case strings.HasPrefix(key, "remote.") &&
		(strings.HasSuffix(key, ".uploadpack") || strings.HasSuffix(key, ".receivepack") || strings.HasSuffix(key, ".proxy")):
		return true
	}
	return false
}

// UnsafeConfig returns the sorted, de-duplicated repo-supplied config keys
// present in the target that unsafeConfigKey flags -- the execution-capable
// settings fixpoint cannot neutralize. It only PARSES config files (no filter,
// no hook, no worktree write), so it is safe to call before any
// worktree-touching command.
//
// It must see every key a real git command would honor from inside .git, which
// rules out the obvious `git config --local --list`:
//
//   - INCLUDES. git-config(1) defaults --includes to off "when a specific file is
//     given (e.g. using --file, --global, etc.)", and --local selects a specific
//     file -- so `git config --local --list` does NOT expand includes. A
//     .git/config whose entire content is `[include] path = hooks/cfg` then reports
//     nothing, while git diff/add/status and gh pr checkout all apply whatever the
//     included file defines.
//   - WORKTREE SCOPE. With extensions.worktreeConfig set, .git/config.worktree
//     applies to every git command in that worktree but lives in the `worktree`
//     scope, which --local does not read.
//
// So it lists ALL scopes with includes expanded and keeps only the two scopes the
// repository itself can write. The scope filter is the point: without it the
// operator's own global credential.helper -- which they configured, and which is
// not repo-controlled -- would be reported as a reason to refuse the target.
// (--show-scope needs git >= 2.26; older git makes every entry unattributable, so
// this fails closed rather than degrading to the --local blind spot above.)
func (c *Collector) UnsafeConfig(ctx context.Context) ([]string, error) {
	// -z with --show-scope: NUL-separated fields alternating "scope" then
	// "key\nvalue" (a valueless key is just "key"). A real repo always has at
	// least the default core.* entries, so a git repo yields a non-error result.
	out, err := c.git(ctx, "config", "--list", "-z", "--show-scope", "--includes")
	if err != nil {
		return nil, fmt.Errorf("inspect target git config: %w", err)
	}
	fields := strings.Split(out, "\x00")
	seen := map[string]bool{}
	var keys []string
	for i := 0; i+1 < len(fields); i += 2 {
		scope, entry := fields[i], fields[i+1]
		// Everything else (global, system, command -- our own gitSafeConfig -c
		// overrides land in `command`) is the operator's configuration, not the
		// target's, and refusing on it would be a false refusal.
		if scope != "local" && scope != "worktree" {
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

// gitInput runs a git command that reads its input from stdin (currently only
// `update-index --index-info`), with the same hardening as every other git call.
func (c *Collector) gitInput(ctx context.Context, stdin string, args ...string) (string, error) {
	return c.runInput(ctx, strings.NewReader(stdin), "git", append(append([]string{}, gitSafeConfig...), args...)...)
}

func (c *Collector) run(ctx context.Context, name string, args ...string) (string, error) {
	return c.runInput(ctx, nil, name, args...)
}

func (c *Collector) runInput(ctx context.Context, stdin io.Reader, name string, args ...string) (string, error) {
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
	cmd.Stdin = stdin
	err := cmd.Run()
	// Own the whole subprocess lifecycle, not just the leader, exactly as agent.Run
	// and verify.runOne do. cmd.Cancel (KillProcessGroup) fires only on
	// cancellation, so a git/gh that exits SUCCESSFULLY after spawning a child --
	// a credential helper, a hook, one of gh's internal git calls -- leaves that
	// child alive in our process group, free to mutate the repository concurrently
	// with the clean-tree check, verification, or the round commit, or to leak past
	// an aborted run. SIGKILL the whole group on every exit path; it is a no-op once
	// the group is empty, which is the common case.
	_ = agent.KillProcessGroup(cmd)
	// That kill comes too late to shorten the wait it follows, so a git/gh that
	// exited 0 after leaving a descendant on the output pipe (a credential-cache
	// daemon from a fetch, a hook's child) still returns exec.ErrWaitDelay. The
	// leader's exit status is authoritative: a `git commit` that landed must not be
	// reported as failed -- that ends the run with no CommitSHA while the commit sits
	// in history, and sends reconciliation looking for edits that are already
	// committed. Callers parse stdout, so a capture cut short surfaces as a parse
	// error on its own rather than as a silently wrong value.
	if agent.SucceededDespiteLeakedPipe(cmd, err) {
		err = nil
	}
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
