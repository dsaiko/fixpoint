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
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/gitenv"
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
	// afterCheckout is called by Prepare the moment `gh pr checkout` has switched
	// branches, before Prepare issues another git command. Set via OnCheckout.
	afterCheckout func(context.Context) error
}

// New returns a collector for the configured target.
func New(cfg config.Target) *Collector { return &Collector{cfg: cfg} }

// OnCheckout registers a callback Prepare runs immediately after `gh pr checkout`
// switches branches. An error from it aborts the rest of Prepare.
//
// It exists because a branch switch invalidates what a caller learned about the
// target BEFORE it. Git config is branch-conditional -- an
// `includeIf "onbranch:<pattern>"` in .git/config pulls in a whole config file
// only while a matching branch is checked out -- so a checkout can activate a
// core.sshCommand, credential helper, content filter or core.worktree redirect
// that was inert when the caller's trust gates inspected the target. Prepare's own
// remaining commands are enough to fire some of those (it may `git fetch` the PR's
// base commit, which runs a credential helper or ssh command), so the re-check
// cannot wait for Prepare to return. See orchestrator.recheckPreflightGuards.
func (c *Collector) OnCheckout(fn func(context.Context) error) { c.afterCheckout = fn }

// ExcludeLogs records the run's logs directory (as a target-relative slash
// path) so Collect omits it from directory walks and untracked-file listings.
// An empty rel (logs live outside the target) disables the exclusion.
func (c *Collector) ExcludeLogs(rel string) { c.logsExclude = rel }

// resolveBase turns target.base_ref into the concrete commit the whole run diffs
// against.
//
// A trailing "..." asks for the MERGE BASE of that ref and HEAD, git's own
// notation for "what this branch added". Without it the ref's TIP is used, which
// is the same commit only while the base branch has not moved: once it gains a
// commit this branch does not have, diffing against its tip renders that commit
// backwards, and the panel spends a round reviewing someone else's work presented
// as deletions this branch made.
//
// The suffix lives on base_ref rather than in a second key because the two are one
// decision -- which commit is the base -- and a key that silently does nothing
// unless another key is set is a worse thing to explain. `git rev-parse
// origin/main...HEAD` cannot serve here either: it prints three lines, so the
// pinned base would be a multi-line string and every later `git diff` against it
// would fail.
func (c *Collector) resolveBase(ctx context.Context, ref string) (string, error) {
	base, mergeBase := strings.CutSuffix(ref, "...")
	if mergeBase {
		if base == "" {
			return "", fmt.Errorf("base_ref %q names no ref before the \"...\"", ref)
		}
		out, err := c.git(ctx, "merge-base", base, "HEAD")
		if err != nil {
			return "", fmt.Errorf("resolve merge base of base_ref %q and HEAD: %w: %s", base, err, out)
		}
		return strings.TrimSpace(out), nil
	}
	sha, err := c.git(ctx, "rev-parse", ref)
	if err != nil {
		return "", fmt.Errorf("resolve base_ref %q: %w", ref, err)
	}
	return strings.TrimSpace(sha), nil
}

// Prepare runs once at startup: resolves and pins the diff base (git-diff),
// or checks out the PR branch and pins its merge base (pr).
func (c *Collector) Prepare(ctx context.Context) error {
	switch c.cfg.Mode {
	case config.ModeGitDiff:
		if c.cfg.BaseRef == "" {
			return nil // unstaged working changes; nothing to pin
		}
		sha, err := c.resolveBase(ctx, c.cfg.BaseRef)
		if err != nil {
			return err
		}
		c.baseSHA = sha
	case config.ModePR:
		if _, err := c.git(ctx, "rev-parse", "--git-dir"); err != nil {
			return fmt.Errorf("target.path is not a git repository (mode pr): %w", err)
		}
		out, err := c.run(ctx, "gh", "pr", "checkout", strconv.Itoa(c.cfg.PR))
		if err != nil {
			return fmt.Errorf("gh pr checkout %d: %w: %s", c.cfg.PR, err, out)
		}
		// The tree is now the PR's, and so is whatever branch-conditional git config
		// the checkout activated: re-gate before the next git command below (the base
		// fetch can run a credential helper or ssh command). See OnCheckout.
		if c.afterCheckout != nil {
			if err := c.afterCheckout(ctx); err != nil {
				return err
			}
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
			// --end-of-options: the remote name is a repo-controlled config
			// subsection, so without the terminator a name like
			// `--upload-pack=/tmp/payload` would be parsed by git as an option
			// on fixpoint's own fetch rather than as the repository argument.
			if fout, ferr := c.git(ctx, "fetch", "--no-tags", "--end-of-options", remote, baseOid); ferr != nil {
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

// Scope reports, in one line, how much a run would review -- for --check, before
// any agent is invoked and any money is spent.
//
// It exists because the base of a git-diff run is silent when it is wrong. A
// base_ref that resolves to the wrong commit produces a perfectly valid run over
// the wrong material, and until this existed the first sign of it was a round's
// worth of findings about code you did not touch. The merge-base trap is exactly
// that shape: `origin/main` and `origin/main...` both resolve, and only the diff
// tells you which one you meant.
//
// It deliberately does NOT call Prepare: in pr mode Prepare runs `gh pr checkout`,
// and a validation flag must not switch the operator's branch. That mode is
// therefore reported as unknown rather than checked out to find out.
//
// The pathspec is the same one Collect uses, so the numbers describe what would
// actually be reviewed rather than what git would print unfiltered -- an excluded
// vendor tree must not inflate the estimate it is excluded from.
func (c *Collector) Scope(ctx context.Context) (string, error) {
	switch c.cfg.Mode {
	case config.ModePR:
		return fmt.Sprintf("pr #%d -- scope is known only after `gh pr checkout`, which --check does not run", c.cfg.PR), nil
	case config.ModeGitDiff:
		if c.cfg.BaseRef == "" {
			return "unstaged working-tree changes (no base_ref; review-only)", nil
		}
		base, err := c.resolveBase(ctx, c.cfg.BaseRef)
		if err != nil {
			return "", err
		}
		// The same pathspec Collect builds, symlink aliases and all -- an estimate
		// that skipped them would not describe the material the run reviews.
		specs, err := c.collectPathspec(ctx)
		if err != nil {
			return "", err
		}
		// --no-ext-diff / --no-textconv for the same reason Collect passes them: a
		// repo-controlled diff driver must not be executed by a validation step.
		args := []string{"diff", "--shortstat", "--no-ext-diff", "--no-textconv", base, "--"}
		args = append(args, specs...)
		stat, err := c.git(ctx, args...)
		if err != nil {
			return "", fmt.Errorf("measure diff against %s: %w: %s", shortSHA(base), err, stat)
		}
		if stat = strings.TrimSpace(stat); stat == "" {
			stat = "no changes"
		}
		out := fmt.Sprintf("base_ref %q -> %s; %s", c.cfg.BaseRef, shortSHA(base), stat)
		// Untracked files are part of the material too (Collect lists them), so an
		// estimate that counted only the diff would understate a branch of new files.
		// -z and a NUL-aware scan, not whitespace splitting: git prints one pathname
		// per record, and a valid filename may contain spaces or newlines. Counting
		// fields would report "a b" as two untracked files.
		lsArgs := []string{"ls-files", "--others", "--exclude-standard", "-z", "--"}
		lsArgs = append(lsArgs, specs...)
		n := 0
		// Not swallowed, for the reason Collect gives about the same listing, plus one
		// this path owns: gitScanNUL reports an uncontained git descendant -- a
		// cleanup kill that came back EPERM -- through this error, and an estimate
		// that discarded it would print a clean --check line while a live git sits in
		// the target. A failed scan also cannot honestly be reported as "no untracked
		// files".
		if err := c.gitScanNUL(ctx, func(rel string) {
			if rel != "" {
				n++
			}
		}, lsArgs...); err != nil {
			return "", fmt.Errorf("count untracked files: %w", err)
		}
		if n > 0 {
			out += fmt.Sprintf(", plus %d untracked file(s)", n)
		}
		return out, nil
	case config.ModeDirectory:
		count, _, err := c.listFiles(ctx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d file(s) in scope", count), nil
	}
	return "", fmt.Errorf("unknown mode %q", c.cfg.Mode)
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
		// No -c override can cover these (the driver name is the repository's to
		// choose), so unsafeConfigKey refuses a target that defines one and these flags
		// neutralize the command itself -- belt and braces, since the flags also hold on
		// the operator-configured half a repo-scoped key list cannot see.
		// Keep the run's own logs out of the diff too, not just the untracked
		// listing: a tracked file under the logs dir (a committed .prompt from an
		// earlier run) would otherwise show its full modified content here and be
		// fed back as review material. The same pathspec also drops target.exclude,
		// the mandatory credential patterns -- whose content the diff would
		// otherwise embed verbatim -- and the symlinks whose destination leaves the
		// target; see collectPathspec.
		specs, err := c.collectPathspec(ctx)
		if err != nil {
			return "", err
		}
		args := []string{"diff", "--no-ext-diff", "--no-textconv"}
		if c.baseSHA != "" {
			args = append(args, c.baseSHA)
		}
		args = append(args, "--")
		args = append(args, specs...)
		diff, err := c.git(ctx, args...)
		if err != nil {
			return "", err
		}
		// A failure here (e.g. an invalid core.excludesFile) must not be
		// swallowed: silently treating it as "no untracked files" would drop
		// them from review without reporting the collection was incomplete.
		lsArgs := []string{"ls-files", "--others", "--exclude-standard", "--"}
		lsArgs = append(lsArgs, specs...)
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
	scope, err := c.fileScope()
	if err != nil {
		return 0, "", err
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
		return c.listGitFiles(ctx, scope)
	}
	return c.walkFiles(ctx, scope)
}

// fileScope is what both directory collectors filter against: the compiled
// exclude globs, and the canonical target root every symlink's destination is
// checked against (see symlinkOutOfScope).
type fileScope struct {
	excludes []*regexp.Regexp
	root     string // target.path with every symlink in it resolved
}

// fileScope builds the filter both collectors share. The root is resolved once
// here rather than per file: it is what a symlink's canonical destination is
// compared against, and target.path itself may sit under a symlinked parent
// (/tmp -> /private/tmp), which would otherwise make every path look external.
func (c *Collector) fileScope() (fileScope, error) {
	// EffectiveExcludes adds the mandatory credential patterns, which no config can
	// drop: a reviewer reads any path it is pointed at, so never point it at a key.
	excludes, err := compileGlobs(c.cfg.EffectiveExcludes())
	if err != nil {
		return fileScope{}, fmt.Errorf("target.exclude: %w", err)
	}
	root, err := filepath.EvalSymlinks(c.cfg.Path)
	if err != nil {
		return fileScope{}, fmt.Errorf("resolve target.path %s: %w", c.cfg.Path, err)
	}
	return fileScope{excludes: excludes, root: root}, nil
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
func (c *Collector) listGitFiles(ctx context.Context, scope fileScope) (int, string, error) {
	count := 0
	var sb strings.Builder
	// -z separates paths with NUL, so a filename containing a newline (or one git
	// would otherwise render quoted and escaped) survives intact.
	err := c.gitScanNUL(ctx, func(rel string) {
		if rel == "" {
			return
		}
		rel = filepath.ToSlash(rel)
		if c.skipFile(rel, scope.excludes) {
			return
		}
		// --cached reports index entries, which outlive a file deleted from the
		// worktree. Listing a path an agent cannot open is worse than omitting it.
		fi, err := os.Lstat(filepath.Join(c.cfg.Path, rel))
		if err != nil {
			return
		}
		if fi.Mode()&os.ModeSymlink != 0 && c.symlinkOutOfScope(rel, scope) {
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

// errListingHeldOpen is what a streaming listing reports when its scan had to be
// ended by the drain grace: git exited, its process group was killed, and some
// descendant outside that group still held the stdout write end, so no EOF was
// ever going to arrive. What was read cannot be shown to be the whole listing,
// and git's own exit status says nothing about it.
var errListingHeldOpen = errors.New("stdout held open past the drain grace by a process outside git's process group; the listing may be incomplete")

// gitCleanupKill is the process-group kill gitScanNUL runs after cmd.Wait. It is
// a var solely so a test can make it report the containment failure whose errno
// cannot be produced on demand -- an EPERM needs a descendant running under
// credentials this process cannot signal. Nothing in production reassigns it.
var gitCleanupKill = agent.KillProcessGroup

// scanProgressed reports whether a scan whose progress counter now reads now has
// moved since that counter read mark. The counter is bumped on both sides of the
// callback (see gitScanNUL), so movement is either a new entry taken off the pipe or
// -- on an odd reading -- a callback still in flight over the one already taken.
func scanProgressed(now, mark uint64) bool {
	return now != mark || now%2 == 1
}

// gitScanNUL runs a git command whose stdout is a NUL-delimited list and calls
// fn once per entry as it arrives, so the caller can count an unbounded listing
// without holding it in memory and without the diagnostic output cap c.run
// applies. Everything else -- the safe-config overrides, the hardened
// environment, the operation timeout, the process-group kill, the drain grace on
// the output pipes -- matches c.run, since the reason each of those exists does
// not change with how stdout is consumed. stderr is still buffered, for the error
// message. A listing whose stdout never reached EOF fails with
// errListingHeldOpen even when git itself exited 0.
func (c *Collector) gitScanNUL(ctx context.Context, fn func(string), args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, gitOpTimeout)
	defer cancel()
	full := append(gitenv.SafeConfigArgs(), args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = c.cfg.Path
	cmd.Env = c.probeEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return agent.KillProcessGroup(cmd) }
	// Backstop for a leader that ignores the cancel signal. It has no pipe drain
	// left to bound: both output streams below are files this process owns, so exec
	// runs no copy goroutine and cmd.Wait returns on the leader's exit alone --
	// which is what lets the group kill follow that exit immediately.
	cmd.WaitDelay = 2 * time.Second
	desc := "git " + strings.Join(args, " ")
	// stderr through a pipe this process owns, for the same reason as stdout below:
	// a pipe exec owned would hold cmd.Wait -- and so the process-group kill that
	// follows it -- waiting on whoever still holds the write end, which is exactly
	// the delay that kill exists to prevent.
	errBuf := agent.NewBoundedBuffer(maxRunOutput, agent.TruncationMarker(maxRunOutput))
	errPipe, err := agent.NewOutPipe(errBuf)
	if err != nil {
		return fmt.Errorf("%s: %w", desc, err)
	}
	cmd.Stderr = errPipe.ChildFile()
	// Our own pipe rather than cmd.StdoutPipe, whose read end only cmd.Wait can
	// close -- and Wait cannot be called until the scan finishes. A descendant git
	// left behind holding the write end would then block the scan for the whole
	// gitOpTimeout. Owning both ends lets the waiter below unblock the scan as soon
	// as the leader is gone -- immediately for a descendant the group kill reaches,
	// after agent.PipeDrainGrace for one that escaped it -- and lets the scan drop
	// the pipe on any exit path.
	pr, pw, err := os.Pipe()
	if err != nil {
		errPipe.CloseChild()
		errPipe.Drain()
		return fmt.Errorf("%s: %w", desc, err)
	}
	// Closing twice is harmless (os.File guards it); the defer covers the early
	// returns, the explicit close below ends the scan.
	defer func() { _ = pr.Close() }()
	cmd.Stdout = pw
	startErr := cmd.Start()
	// Drop the parent's write ends: the child and its descendants now hold the only
	// ones, so EOF on pr means every process that could still write is gone.
	_ = pw.Close()
	errPipe.CloseChild()
	if startErr != nil {
		errPipe.Drain()
		return fmt.Errorf("%s: %w", desc, startErr)
	}
	// The leader's exit and the cleanup kill's reply travel separately: the scan
	// error below outranks git's exit status, but must not outrank a kill failure,
	// so the two cannot be pre-joined into one value here.
	type gitExit struct{ wait, kill error }
	waited := make(chan gitExit, 1)
	go func() {
		err := cmd.Wait()
		// Own the whole subprocess lifecycle, not just the leader, exactly as
		// agent.Run and verify.runOne do: cmd.Cancel fires only on cancellation, so
		// a git that exits SUCCESSFULLY after spawning a child leaves it alive in our
		// process group -- free to mutate the repository concurrently with a later
		// clean-tree check or round commit, and, if it inherited stdout, to stall the
		// scan below until the operation timeout. SIGKILL the group on every exit
		// path; os.ErrProcessDone means the group was already empty, the common case.
		//
		// Any other reply is reported rather than dropped, for the reason
		// agent.Supervise gives: off darwin an EPERM proves the group still holds a
		// member this process cannot signal, and a listing that reported success would
		// hand the rest of the round a repository with a live git descendant in it.
		var kill error
		if killErr := gitCleanupKill(cmd); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			kill = fmt.Errorf("kill process group: %w", killErr)
		}
		waited <- gitExit{wait: err, kill: kill}
	}()
	// The scan runs on its own goroutine so this function is never at the mercy of
	// the read. EOF arrives only once EVERY holder of the write end is gone, and a
	// descendant that left our process group -- setsid, or a git that daemonizes --
	// survives the kill above and can hold it open for as long as it likes. Nothing
	// reaches a blocked pr.Read from the outside: the operation timeout reaches the
	// leader and no further, so a scan waiting on that pipe would outlive git
	// indefinitely, hanging the whole collection. Closing the read end out from
	// under it is the only way to end it, exactly as OutPipe.Drain does for stderr.

	// The scan's progress, published so the grace below can bound the ABSENCE of data
	// rather than the scan's completion: what the grace has to distinguish is a listing
	// still being delivered from one that never will be.
	//
	// Bumped BOTH on taking an entry off the pipe and on that entry's fn returning, so
	// an ODD value means fn is in flight. Counting only entries taken would make a
	// single fn call that outlasts one whole grace -- one cold-cache EvalSymlinks is
	// enough, see the grace's own case below -- indistinguishable from a pipe nothing
	// is coming through, and a complete listing would be cut and reported as held open.
	// Cutting cannot help there in any case: the cut path joins this goroutine, which
	// is inside fn, so it waits for exactly the same call it just failed the listing
	// for. A caller whose fn never returns is bounded by ctx, as a slow feed is.
	var progress atomic.Uint64
	scanned := make(chan error, 1)
	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 0, 64<<10), maxGitPath)
		sc.Split(scanNUL)
		for sc.Scan() {
			progress.Add(1)
			fn(sc.Text())
			progress.Add(1)
		}
		scanned <- sc.Err()
	}()
	var scanErr error
	var exit gitExit
	var reaped bool
	// Nothing arms this until the leader has exited AND the group kill has run.
	// Only a descendant that escaped the group can still hold the write end by
	// then, which is precisely what OutPipe.Drain's grace covers -- so the scan gets
	// the same grace rather than being left to the operation timeout. Without it the
	// one interleaving costs stderr two seconds and stdout ten minutes, with git
	// already finished and its exit status sitting in `waited` unread.
	var grace <-chan time.Time
	var graceTimer *time.Timer
	// The progress counter at the instant the grace was last armed, so a fired grace
	// can ask whether the scan moved during it. Monotonic, so equality means not one
	// increment happened -- no entry taken and no fn returned.
	var mark uint64
	// One timer, re-armed in place while the scan keeps making progress, so one Stop
	// covers every exit path -- and the usual one is the scan reaching EOF before the
	// grace fires, which would otherwise leave a live timer behind for every listing
	// this function performs.
	defer func() {
		if graceTimer != nil {
			graceTimer.Stop()
		}
	}()
scan:
	for {
		select {
		case scanErr = <-scanned:
			break scan
		case exit = <-waited:
			// Received once, so this case simply blocks from here on: the grace is armed
			// here and only ever re-armed by the case below.
			reaped = true
			mark = progress.Load()
			graceTimer = time.NewTimer(agent.PipeDrainGrace)
			grace = graceTimer.C
		case <-grace:
			// What this grace bounds is a scan getting NOTHING, not one that is merely
			// slow. By the time git exits it has handed its whole listing to the pipe, so
			// up to a pipe buffer plus the scanner's own buffer of it -- thousands of
			// entries -- can still be unread, and unlike OutPipe.Drain's copy goroutine
			// this one runs the CALLER's fn per entry: listGitFiles' costs a glob sweep,
			// an Lstat, and for a symlink an EvalSymlinks, which on a network filesystem
			// or a cold cache can easily outlast one grace -- a SINGLE one of them, so
			// entries arriving is not the only shape progress takes. Cutting there would
			// discard a listing that was arriving fine and blame an escaped writer that
			// does not exist. So re-arm while the scan moves at all, an entry taken off
			// the pipe or a callback still in flight; a scan blocked on a write end held
			// outside the group does neither and is still cut after a single grace, and
			// one fed slowly forever remains bounded by ctx below.
			if now := progress.Load(); scanProgressed(now, mark) {
				mark = now
				graceTimer.Reset(agent.PipeDrainGrace)
				continue
			}
			_ = pr.Close()
			scanErr = <-scanned
			// The listing could not be read to EOF, so a complete one is
			// indistinguishable from one cut off mid-entry -- and git's exit status
			// cannot tell them apart either, since git itself succeeded. Report it
			// rather than hand the round a scope that may be silently narrow. A scan
			// that reached EOF in the same instant the grace fired keeps its own (nil)
			// outcome: that listing IS known complete.
			if errors.Is(scanErr, os.ErrClosed) {
				scanErr = errListingHeldOpen
			}
			break scan
		case <-ctx.Done():
			_ = pr.Close()
			// Still join the goroutine rather than abandon it: fn writes the caller's
			// state, so nothing may still be calling it once this returns. The close
			// above bounds that wait -- a blocked read fails immediately.
			scanErr = <-scanned
			// Report why the listing ended, not the mechanism that ended it. A scan that
			// had already finished (nil) or failed on its own keeps its own outcome.
			if errors.Is(scanErr, os.ErrClosed) {
				scanErr = ctx.Err()
			}
			break scan
		}
	}
	// Stop reading on every exit path. A scan that stopped early would otherwise
	// leave git blocked on a full pipe; closing the read end fails its next write
	// instead, and the scan error below outranks the exit status that produces.
	_ = pr.Close()
	if !reaped {
		exit = <-waited
	}
	// The group is dead by now, so this collects the last of stderr and guarantees
	// nothing is still writing to errBuf when the error below reads it.
	errPipe.Drain()
	// A scan failure comes first: it means the listing was not read in full, which
	// git's own exit status cannot tell us -- including the nonzero exit our own
	// pr.Close above provokes.
	//
	// A failed cleanup kill is not git's exit status, though, and is never demoted
	// to it. The interleaving that ends a scan by the grace is the very one that
	// yields an EPERM -- a descendant under other credentials, alive in the group,
	// still holding the write end -- so letting the scan error win would drop the
	// one signal proving containment failed and report only that the listing may be
	// short. It outranks the leader's own outcome for the reason agent.Supervise
	// gives, so it is joined rather than replaced.
	if scanErr != nil {
		if exit.kill != nil {
			return fmt.Errorf("%s: reading output: %w; and %w", desc, scanErr, exit.kill)
		}
		return fmt.Errorf("%s: reading output: %w", desc, scanErr)
	}
	if werr := errors.Join(exit.wait, exit.kill); werr != nil {
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

// symlinkOutOfScope reports whether a symlink at the target-relative path rel
// resolves somewhere a reviewer must not be pointed at: outside the canonical
// target root, or onto a path the exclude globs (mandatory credential patterns
// included) remove.
//
// Filtering the pathname alone is not enough, because the name of a symlink says
// nothing about what it opens. A repository can commit an innocent-looking
// `context.txt` pointing at ~/.ssh/id_rsa, another checkout, or a
// credential file just outside the target; listing that alias hands the secret to
// an unsandboxed reviewer that is instructed to read the files in scope -- and
// into its prompt, output, and log artifacts -- while every mandatory pattern
// matches only the alias, never the resolved destination. So the destination is
// what is judged here. An unresolvable link (dangling, a resolution loop, an
// unreadable parent) is dropped too: nothing can read it anyway, and guessing
// where it points is the wrong way to be wrong.
//
// This bounds what collection ADVERTISES; it is not a sandbox. A prompt-injected
// agent can still open any path it likes, which is what the shipped configs'
// security notes and an OS-level sandbox are for.
func (c *Collector) symlinkOutOfScope(rel string, scope fileScope) bool {
	dest, err := filepath.EvalSymlinks(filepath.Join(c.cfg.Path, rel))
	if err != nil {
		return true
	}
	inner, err := filepath.Rel(scope.root, dest)
	if err != nil || inner == ".." || strings.HasPrefix(inner, ".."+string(filepath.Separator)) {
		return true
	}
	// Inside the root, but the destination gets the same filtering the pathname
	// got: a `notes.md -> config/.env` alias must not smuggle in a file the
	// exclusions already removed under its real name.
	return c.skipFile(filepath.ToSlash(inner), scope.excludes)
}

// walkFiles is the non-git fallback: target.path may be any directory, so scope
// comes from the filesystem and only target.exclude narrows it.
//
// It observes ctx: a large tree can take a long time to walk, and Ctrl-C has to
// reach the one collection path that runs no subprocess of its own.
func (c *Collector) walkFiles(ctx context.Context, scope fileScope) (int, string, error) {
	count := 0
	var sb strings.Builder
	excludes := scope.excludes
	// Walk the CANONICAL root, not target.path as written. WalkDir never descends
	// through a symlink -- including the one it is handed -- so a target.path that
	// is itself a link to the checkout (an operator's ~/work -> /mnt/src alias)
	// would yield a single entry for the root and a listing of "Files in scope
	// (0)", which reads like a tree that was reviewed and found empty. Resolving
	// it changes nothing for an ordinary path, and the relative paths this
	// produces are identical either way.
	root := scope.root
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
		// WalkDir never descends through a symlink, so every one of them -- to a
		// file or to a directory -- arrives here as an entry to list. Judge it by
		// where it actually points, not by the name it was committed under.
		if d.Type()&os.ModeSymlink != 0 && c.symlinkOutOfScope(rel, scope) {
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
// both are required before a failure is read as an answer. Matching the English
// message is only sound because probeEnv pins LC_ALL=C for every git command
// fixpoint runs -- see there.
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

// WorktreeOutOfScope returns the work-tree root git would actually operate on
// when that root does not contain target.path, and "" when it does (or when
// there is no work tree here at all). A non-empty result means every git command
// fixpoint runs would read and write a DIFFERENT directory than the one under
// review.
//
// A repo-local `core.worktree` (and GIT_WORK_TREE, and a .git file pointing at a
// gitdir that sets it) redirects git's work tree anywhere on the machine while
// .git stays where it is, so target.path still looks like an ordinary checkout.
// Nothing else catches it: unsafeConfigKey lists programs git EXECUTES, and
// core.worktree runs nothing; `rev-parse --show-prefix` is empty at the redirected
// root, so AtRepoRoot passes; and `--is-inside-work-tree` answers false when the
// redirect points away from target.path, which only makes IsGitRepo's callers skip
// their checks. The consequences are collection and writes outside the target:
// git-diff/pr mode's `git diff` / `ls-files` report the redirected tree's contents
// as the review material (a redirect at ~ hands a reviewer the home directory),
// and a directory fix round's `git add`/`git commit` stage and commit files from
// it.
//
// core.worktree is NOT inherently hostile -- git sets it in every submodule
// checkout, where it names the submodule's own directory -- so this asks git for
// the effective root and judges the RESULT rather than refusing the key outright.
// A root that IS target.path is always fine (plain checkout, submodule, linked
// worktree). A root ABOVE target.path is fine only when git discovered it by
// walking up from target.path, which is the supported "target.path is a
// subdirectory of its repository" case; the same shape produced by a repo-local
// core.worktree is an escape, since `core.worktree = /` would quietly make the
// whole filesystem the tree while target.path stays inside it. Anything else is
// an escape whatever config shape produced it -- including GIT_WORK_TREE.
//
// Like IsGitRepo, only a CONFIRMED "no work tree here" answer -- no repository, or
// a bare one -- is a clean empty result; an unanswered probe is an error, because
// a security check that silently passes when git could not be run is no check.
func (c *Collector) WorktreeOutOfScope(ctx context.Context) (string, error) {
	out, err := c.git(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		if ctx.Err() == nil && (notARepository(err) || noWorkTree(err) || errors.Is(err, exec.ErrNotFound)) {
			return "", nil
		}
		return "", fmt.Errorf("determine the work tree git would use for %s: %w", c.cfg.Path, err)
	}
	top := strings.TrimSpace(out)
	if top == "" {
		return "", nil
	}
	// Compare canonical paths: target.path may sit under a symlinked parent
	// (/tmp -> /private/tmp) and git reports the root it resolved, so the raw
	// strings can differ for the very same directory.
	root, err := filepath.EvalSymlinks(c.cfg.Path)
	if err != nil {
		return "", fmt.Errorf("resolve target.path %s: %w", c.cfg.Path, err)
	}
	// A root that cannot even be resolved is not target.path, and stays out of
	// scope: guessing where it points is the wrong way to be wrong.
	canonicalTop, resolveErr := filepath.EvalSymlinks(top)
	inScope := resolveErr == nil && canonicalTop == root
	if resolveErr == nil && !inScope {
		// Not the target itself, so the only remaining legitimate shape is the
		// walked-up repository root above a subdirectory target. A repo-local
		// core.worktree rules that reading out: with one set, the root git reports
		// is the configured one, not a discovered one.
		configured, err := c.hasRepoWorktreeOverride(ctx)
		if err != nil {
			return "", err
		}
		inner, relErr := filepath.Rel(canonicalTop, root)
		inScope = !configured && relErr == nil &&
			inner != ".." && !strings.HasPrefix(inner, ".."+string(filepath.Separator))
	}
	if inScope {
		return "", nil
	}
	return top, nil
}

// hasRepoWorktreeOverride reports whether the repository itself sets
// core.worktree, i.e. whether the work-tree root git reports was configured by
// the target rather than discovered by walking up from target.path.
func (c *Collector) hasRepoWorktreeOverride(ctx context.Context) (bool, error) {
	keys, err := c.repoScopedConfigKeys(ctx)
	if err != nil {
		return false, err
	}
	return slices.Contains(keys, "core.worktree"), nil
}

// noWorkTree reports whether a failed git command failed because the repository
// has no work tree (a bare repository), as opposed to failing for any other
// reason. Like notARepository, this is a confirmed answer rather than a fault.
func noWorkTree(err error) bool {
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 128 {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "must be run in a work tree")
}

// remoteOrigin is git's conventional default remote name, used as the
// fallback when no remote URL matches the gh base repository.
const remoteOrigin = "origin"

// ghRemote returns the git remote that corresponds to the repository gh treats
// as this checkout's base, so PR base objects are fetched from the right place
// without assuming the remote is named "origin". It matches the base repo's
// canonical URL -- host AND owner/repo, see remoteIdentity -- against the
// configured remote URLs. When gh cannot name the base repository at all it
// falls back to origin and then the sole/first remote; once gh HAS named it, the
// fallback is confined to remotes on the base repository's HOST, so a remote of
// the checkout's choosing on another server is never fetched from. Remotes whose
// name looks like a git option are refused (see below).
func (c *Collector) ghRemote(ctx context.Context) (string, error) {
	out, err := c.git(ctx, "remote")
	if err != nil {
		return "", err
	}
	remotes := strings.Fields(out)
	if len(remotes) == 0 {
		return "", errors.New("no git remotes configured to fetch the PR base from")
	}
	// A remote NAME is a repo-controlled config subsection that ends up as a
	// positional argument to git. Callers pass --end-of-options so such a name
	// cannot be parsed as an option, but a legitimate remote name never starts
	// with "-", so an untrusted checkout that ships one is refused outright
	// rather than fetched from.
	kept := make([]string, 0, len(remotes))
	var optionLike []string
	for _, r := range remotes {
		if strings.HasPrefix(r, "-") {
			optionLike = append(optionLike, r)
			continue
		}
		kept = append(kept, r)
	}
	if len(kept) == 0 {
		return "", fmt.Errorf("refusing to fetch the PR base: the only git remotes have option-like names: %s", strings.Join(optionLike, " "))
	}
	remotes = kept
	if want := c.ghBaseIdentity(ctx); want != "" {
		return c.remoteForIdentity(ctx, remotes, want)
	}
	// gh could not name the base repository (offline, unauthenticated, no GitHub
	// remote), so there is no identity to match against: fall back to git's
	// conventional default and then the sole/first remote.
	r, _ := preferOrigin(remotes)
	return r, nil
}

// preferOrigin picks git's conventional default from a set of equally correct
// remotes, falling back to the first, so the choice does not depend on git's
// listing order. It reports false when there is nothing to choose from.
func preferOrigin(remotes []string) (string, bool) {
	for _, r := range remotes {
		if r == remoteOrigin {
			return remoteOrigin, true
		}
	}
	if len(remotes) == 0 {
		return "", false
	}
	return remotes[0], true
}

// ghBaseIdentity is the host/owner/repo identity of the repository gh treats as
// this checkout's base, or "" when gh cannot name one (offline, unauthenticated,
// no GitHub remote, an unparsable URL).
//
// It asks for the canonical URL rather than nameWithOwner because the URL names
// the HOST the PR's objects live on, and that is the server the fetch has to
// contact; see remoteIdentity for why owner/repo alone is not enough.
func (c *Collector) ghBaseIdentity(ctx context.Context) string {
	raw, err := c.run(ctx, "gh", "repo", "view", "--json", "url", "--jq", ".url")
	if err != nil {
		return ""
	}
	return remoteIdentity(raw)
}

// remoteForIdentity returns the remote to fetch the PR base from, given the
// host/owner/repo identity gh reports for the base repository.
//
// A remote naming that exact identity wins. Otherwise a remote on the same HOST
// is used: the common case is a clone of a fork whose only remote is the fork
// itself while gh resolves the base repository to the parent, and GitHub (like
// other forges) serves fork-network objects, so fetching the base OID from the
// fork remote succeeds. What the host check keeps out is the case the exact match
// exists for: a remote for a DIFFERENT host holding the same owner/repo is what
// an attacker adds to have the PR base fetched -- with the operator's credentials
// -- from a server of the checkout's choosing.
func (c *Collector) remoteForIdentity(ctx context.Context, remotes []string, want string) (string, error) {
	wantHost, _, _ := strings.Cut(want, "/")
	var matches, sameHost []string
	for _, r := range remotes {
		u, err := c.git(ctx, "remote", "get-url", "--end-of-options", r)
		if err != nil {
			continue
		}
		id := remoteIdentity(u)
		switch host, _, _ := strings.Cut(id, "/"); {
		case id == want:
			// Several remotes may legitimately name the same host and repository (a
			// clone plus an explicitly added upstream). They are the same fetch
			// target, so any of them is correct.
			matches = append(matches, r)
		case id != "" && host == wantHost:
			sameHost = append(sameHost, r)
		}
	}
	if r, ok := preferOrigin(matches); ok {
		return r, nil
	}
	if r, ok := preferOrigin(sameHost); ok {
		return r, nil
	}
	return "", fmt.Errorf("refusing to fetch the PR base: no git remote points at the PR's base repository %s or any other repository on %s (remotes: %s)", want, wantHost, strings.Join(remotes, " "))
}

// remoteIdentity reduces a git remote URL -- or the canonical repository URL gh
// reports -- to the lowercased "host/owner/repo" identity, or "" when the URL is
// not of that shape (a local path, a file:// URL, a nested path, a transport
// helper such as ext::). It exists so ghRemote compares identities for EQUALITY,
// host included:
//
//   - a substring test on the URL matches acme/widget inside acme/widget-fork or
//     acme/widgets, and if such a remote sorts first the PR base would be fetched
//     from the wrong repository;
//   - owner/repo alone says nothing about WHICH SERVER holds it, so a checkout
//     that configures https://attacker.example/acme/widget next to the real
//     github.com/acme/widget would be accepted as the PR's base repository and
//     `git fetch` would contact the attacker's host as the operator -- an
//     unintended outbound connection, an SSH/credential-helper prompt against a
//     host of the checkout's choosing, or a probe of an internal endpoint.
//
// Both supported URL forms are handled: scheme://[user[:pass]@]host[:port]/owner/repo[.git]
// and the scp-like [user@]host:owner/repo[.git]. Only the optional .git suffix is
// stripped, since it is the only one git itself treats as decoration.
func remoteIdentity(raw string) string {
	s := strings.TrimSpace(raw)
	var scheme, authority, path string
	if sch, rest, ok := strings.Cut(s, "://"); ok {
		// Everything up to the first slash is the authority (credentials, host, port).
		a, p, ok := strings.Cut(rest, "/")
		if !ok {
			return ""
		}
		scheme, authority, path = strings.ToLower(sch), a, p
	} else if a, p, ok := strings.Cut(s, ":"); ok {
		// scp-like syntax has no port, so the whole remainder is the path.
		authority, path = a, p
	} else {
		return ""
	}
	host := remoteHost(scheme, authority)
	if host == "" {
		return ""
	}
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	owner, repo, ok := strings.Cut(strings.Trim(path, "/"), "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return ""
	}
	return strings.ToLower(host + "/" + owner + "/" + repo)
}

// defaultPorts are the ports a git URL may spell out without naming a different
// endpoint than the scheme already implies. git's two aliases for ssh:// are
// listed too, so a remote spelled with one is not treated as another endpoint.
var defaultPorts = map[string]string{
	"http": "80", "https": "443", "git": "9418",
	"ssh": "22", "git+ssh": "22", "ssh+git": "22",
}

// remoteHost extracts the host[:port] a git URL authority resolves to, lowercased.
//
// Userinfo is dropped at the LAST "@", not the first: git contacts the host after
// it, so a username shaped like a host ("https://github.com@attacker.example/...")
// must resolve to attacker.example rather than being mistaken for github.com.
//
// A port is kept unless it is the scheme's default, so a remote spelled
// ssh://git@github.com:22/acme/widget still matches the https URL gh reports for
// the same repository, while a non-default port -- a different endpoint on that
// host -- stays part of the identity.
func remoteHost(scheme, authority string) string {
	if i := strings.LastIndex(authority, "@"); i >= 0 {
		authority = authority[i+1:]
	}
	host, port := authority, ""
	if strings.HasPrefix(host, "[") {
		// A bracketed IPv6 literal has colons of its own; only what follows the
		// closing bracket can be a port.
		if end := strings.Index(host, "]"); end >= 0 {
			if rest := host[end+1:]; strings.HasPrefix(rest, ":") {
				host, port = host[:end+1], rest[1:]
			}
		}
	} else if h, p, ok := strings.Cut(host, ":"); ok {
		host, port = h, p
	}
	if port != "" && port != defaultPorts[scheme] {
		host += ":" + port
	}
	return strings.ToLower(host)
}

// Commit stages everything except the excluded paths and commits with the
// given header and body. Returns the new commit SHA, or "" if there was
// nothing to commit.
//
// The excluded paths are left exactly as they were, in the worktree AND in the
// index. A non-nil error with a non-empty SHA means the commit landed but that
// restoration did not; the error names the SHA, since callers treat an error as
// "no commit".
func (c *Collector) Commit(ctx context.Context, header, body string, exclude ...string) (sha string, err error) {
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
	// Put the excluded paths' pre-run index entries back on EVERY path out of here,
	// whether or not the commit itself succeeded and whether or not staging even got
	// that far. The add below is what replaces a staged-only excluded version with the
	// worktree one, so once it may have run, a bare return would leave that version
	// lost -- and interruption reconciliation excludes the path too, so nothing else
	// puts it back. On the ordinary path the commit carried the excluded paths' HEAD
	// version, so restoring the entries reproduces exactly the staged diff the run
	// started with, without ever committing an excluded path.
	// Cancellation must not be why the entries stay collapsed: on a dead context every
	// git command fails instantly, and commitStaged may well have recovered a commit
	// that landed. Restore on a fresh context then, still bounded per operation by
	// gitOpTimeout, exactly as committedSHA re-reads HEAD.
	defer func() {
		sha, err = c.restoreExcludedOnExit(ctx, exclude, staged, sha, err)
	}()
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
	return c.commitStaged(ctx, header, body)
}

// stagedIndex is stagedExcluded's snapshot of the index under the excluded
// paths. restore is what says whether the snapshot has to be put back at all;
// it is NOT implied by entries being non-empty, because a staged DELETION is
// represented by the ABSENCE of a record: ls-files emits nothing for it, yet the
// reset in Commit resurrects the deleted path's HEAD entry, so an empty snapshot
// is exactly the state that must be restored.
type stagedIndex struct {
	entries string // raw `ls-files --stage -z` records, empty for a pure deletion
	restore bool
}

// stagedExcluded snapshots the index entries currently under the excluded paths,
// with restore set only when there is something whose loss the add/reset pair in
// Commit could cause. It deliberately reports no restoration in the ordinary case
// -- an index that matches HEAD there is reproduced exactly by the reset -- so a
// normal round commit runs no index surgery at all.
func (c *Collector) stagedExcluded(ctx context.Context, exclude []string) (stagedIndex, error) {
	if len(exclude) == 0 {
		return stagedIndex{}, nil
	}
	specs := literalPathspec(exclude)
	// With a HEAD, the reset in Commit puts each excluded entry back to its HEAD
	// version, so only an index that DIFFERS from HEAD there has anything to
	// preserve. On an unborn branch `reset HEAD` resets to the EMPTY TREE (git
	// treats a literal "HEAD" that does not resolve that way), which drops every
	// excluded entry outright -- so there the whole index under the exclusion is
	// at stake and the diff-index shortcut, which needs a HEAD to run at all,
	// does not apply.
	if c.hasHEAD(ctx) {
		diff, err := c.git(ctx, append([]string{"diff-index", "--cached", "--name-only", "-z", "HEAD", "--"}, specs...)...)
		if err != nil {
			return stagedIndex{}, fmt.Errorf("check for staged changes under the excluded path(s): %w: %s", err, diff)
		}
		if strings.Trim(diff, "\x00") == "" {
			return stagedIndex{}, nil
		}
	}
	out, err := c.git(ctx, append([]string{"ls-files", "--stage", "-z", "--"}, specs...)...)
	if err != nil {
		return stagedIndex{}, fmt.Errorf("read index entries for the excluded path(s): %w: %s", err, out)
	}
	return stagedIndex{entries: out, restore: true}, nil
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
// removed, and how a staged deletion is reproduced rather than resurrected: a
// snapshot with no records at all still has to run, since emptying the index
// under the exclusion is precisely what puts that deletion back.
func (c *Collector) restoreStagedExcluded(ctx context.Context, exclude []string, staged stagedIndex) error {
	if !staged.restore {
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
	sb.WriteString(staged.entries) // already NUL-terminated records
	if sb.Len() == 0 {
		// Nothing staged before, nothing staged now: the index under the exclusion
		// already matches the snapshot, so leave it alone.
		return nil
	}
	if out, err := c.gitInput(ctx, sb.String(), "update-index", "-z", "--index-info"); err != nil {
		return fmt.Errorf("git update-index: %w: %s", err, out)
	}
	return nil
}

// restoreExcludedOnExit puts the excluded paths' index entries back on the way out
// of a commit-building operation and folds a failed restoration into that
// operation's own result. Commit and SquashSince both defer it on their named
// results: both have to take the excluded entries out of the index to keep them out
// of the commit they build, and neither may leave them collapsed afterwards.
//
// Cancellation must not be why the entries stay collapsed: on a dead context every
// git command fails instantly, and the commit may well have landed anyway. Restore
// on a fresh context then, still bounded per operation by gitOpTimeout, exactly as
// committedSHA re-reads HEAD.
func (c *Collector) restoreExcludedOnExit(ctx context.Context, exclude []string, staged stagedIndex, sha string, err error) (string, error) {
	rctx := ctx //nolint:contextcheck // deliberate fresh context below: ctx may be canceled, but the excluded paths' index entries must still be put back
	if rctx.Err() != nil {
		rctx = context.Background()
	}
	rerr := c.restoreStagedExcluded(rctx, exclude, staged)
	switch {
	case rerr == nil:
		return sha, err
	case err != nil:
		return "", fmt.Errorf("%w; and the staged state of the excluded path(s) could not be restored: %w", err, rerr)
	case sha != "":
		// The commit landed: name it, because the returned error means the caller
		// cannot report the SHA itself.
		return sha, fmt.Errorf("commit %s landed but the staged state of the excluded path(s) could not be restored: %w", shortSHA(sha), rerr)
	default:
		return "", fmt.Errorf("the staged state of the excluded path(s) could not be restored: %w", rerr)
	}
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
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		// Only the quiet not-resolvable status is an answer: under -q git exits 1 and
		// says nothing when HEAD does not resolve, which on a repository fixpoint has
		// already locked and checked means an unborn branch. Every other failure --
		// a fatal (exit 128, e.g. a corrupt repository), the gitOpTimeout or a signal
		// killing the process group (no exit status at all), git missing -- is
		// operational and must surface. Reading one of those as "" would hand
		// SquashSince an empty base, and it would rewrite the branch as a ROOT commit,
		// cutting the repository's existing history off from the current branch.
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("resolve HEAD: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// ChangedSince lists the repo-relative paths that differ between base and HEAD.
// It answers one question for the orchestrator: has the code a finding describes
// already moved under it?
//
// A round's reviewers all run against ONE snapshot, and the per-fix sessions that
// follow each commit into the tree -- so by the time session 7 opens its issue, six
// commits have landed since the finding was written. Observed in a real run: three
// findings were rejected as "already handled at HEAD, in the most recent commit
// 390cfcb", a commit made minutes earlier by an earlier session of the SAME pass.
//
// base == "" (an unborn branch had no commits when the round started) means
// everything since is new, which no diff can express, so the caller is told nothing
// is known rather than being handed a wrong answer.
//
// The empty result is an empty map, never nil-with-nil-error: "nothing moved" and
// "cannot say" are both answered by len() == 0 here, and the caller treats them the
// same way -- it omits a caution rather than acting on one.
func (c *Collector) ChangedSince(ctx context.Context, base string) (map[string]bool, error) {
	changed := map[string]bool{}
	if base == "" {
		return changed, nil
	}
	head, err := c.HeadSHA(ctx)
	if err != nil {
		return changed, err
	}
	if head == "" || head == base {
		return changed, nil
	}
	if err := c.gitScanNUL(ctx, func(p string) { changed[p] = true },
		"diff", "--name-only", "-z", base, head); err != nil {
		return nil, fmt.Errorf("list paths changed since %s: %w", base, err)
	}
	return changed, nil
}

// SquashSince replaces every commit after base with a single commit carrying the
// same tree, for loop.commit_policy. It is a pure regrouping: the replacement
// commit is built from the index as it stands, so the content committed here is
// byte-for-byte the content the per-fix commits already put through the verify
// gate one at a time.
//
// The order matters and is the whole point of using commit-tree rather than
// `reset --soft` + `git commit`: write-tree and commit-tree only WRITE OBJECTS,
// leaving HEAD, the index and the worktree untouched, so every failure or
// cancellation before the closing ref move is a no-op (at worst an unreferenced
// commit object that gc collects). Rewinding the branch first would mean a failed
// or canceled commit leaves the branch rewound with the already-verified commits
// reachable only through the reflog. The single `reset --soft` at the end is one
// atomic ref update: it either happens or it does not.
//
// base == "" squashes back to an unborn branch, which is why HeadSHA reports that
// state as an empty string rather than an error; the replacement is then a root
// commit rather than a child of base.
//
// The excluded paths are kept out of the squashed commit and left exactly as they
// were in the index, the same guarantee Commit gives.
func (c *Collector) SquashSince(ctx context.Context, base, header, body string, exclude ...string) (sha string, err error) {
	// Commit deliberately puts the excluded paths' staged entries BACK into the index
	// after each per-fix commit, so the live index is not a tree any of those commits
	// produced: it also carries whatever was staged under the exclusion. Collapse
	// those entries to their HEAD version first -- exactly what Commit's reset does --
	// or the squash would publish staged changes under an excluded path (the run's own
	// logs, a credential-bearing **/*.env) that every commit it replaces left out.
	staged, err := c.stagedExcluded(ctx, exclude)
	if err != nil {
		return "", err
	}
	defer func() {
		sha, err = c.restoreExcludedOnExit(ctx, exclude, staged, sha, err)
	}()
	for _, e := range exclude {
		// :(literal) so a metacharacter-bearing exclude path (e.g. "logs[1]") is reset
		// as that exact path, not a glob -- matching the exclusion pathspec.
		if out, err := c.git(ctx, "reset", "-q", "HEAD", "--", ":(literal)"+e); err != nil {
			return "", fmt.Errorf("git reset excluded %s: %w: %s", e, err, out)
		}
	}
	// The index now holds everything the squashed commits staged and nothing else, so
	// turn it into a tree directly rather than re-running Commit's add: re-staging
	// would pick up anything that arrived in the worktree since, which no verify pass
	// has seen.
	tree, err := c.git(ctx, "write-tree")
	if err != nil {
		return "", fmt.Errorf("git write-tree: %w: %s", err, tree)
	}
	// commit-tree takes the message verbatim, while `git commit -m` cleans it up
	// first; run it through stripspace so a squashed commit reads exactly like the
	// per-fix commits it replaces (no trailing whitespace, no doubled blank lines).
	msg, err := c.gitInput(ctx, header+"\n\n"+body, "stripspace")
	if err != nil {
		return "", fmt.Errorf("git stripspace: %w: %s", err, msg)
	}
	args := []string{"commit-tree", strings.TrimSpace(tree)}
	if base != "" {
		args = append(args, "-p", base)
	}
	// commit-tree, unlike `git commit`, ignores commit.gpgsign and would silently
	// drop the signature the per-fix commits carry in a repository that signs.
	if signed, err := c.git(ctx, "config", "--bool", "--get", "commit.gpgsign"); err == nil && strings.TrimSpace(signed) == "true" {
		args = append(args, "-S")
	}
	out, err := c.gitInput(ctx, msg, args...)
	if err != nil {
		return "", fmt.Errorf("git commit-tree: %w: %s", err, out)
	}
	sha = strings.TrimSpace(out)
	// Capture HEAD before the ref move so an interruption landing between the update
	// and git's exit can still be recognized as a squash that happened -- the same
	// recovery commitStaged does around `git commit`.
	before, err := c.git(ctx, "rev-parse", "HEAD")
	if err != nil && ctx.Err() != nil {
		return "", err
	}
	before = strings.TrimSpace(before)
	// --soft moves the branch and leaves the index and worktree exactly as they
	// are; the index is already the tree just committed, so nothing changes on disk.
	if out, err := c.git(ctx, "reset", "--soft", sha); err != nil {
		// The ref update may have landed anyway and only the command's teardown been
		// cut short -- by ctx cancellation, or by c.git's OWN gitOpTimeout, which
		// fires while the caller's ctx is still live and so cannot be detected from
		// ctx.Err(). Re-read HEAD on a fresh context on ANY failure and accept the
		// squash only if HEAD is the commit just built, so a landed squash is never
		// dropped from the round/run summary while a genuinely failed reset still
		// surfaces its error.
		if c.committedSHA(before) == sha { //nolint:contextcheck // committedSHA deliberately re-reads HEAD on a fresh context: this one may be canceled or timed out, but a landed squash must still be recorded
			return sha, nil
		}
		return "", fmt.Errorf("git reset --soft %s: %w: %s", sha, err, out)
	}
	return sha, nil
}

// commitStaged commits whatever is already in the index, returning the new SHA. It
// is separate so Commit's excluded-path restoration runs on every exit from the
// commit itself, including the cancellation-recovery paths below, and so
// SquashSince can reuse it without re-staging.
func (c *Collector) commitStaged(ctx context.Context, header, body string) (string, error) {
	// Capture HEAD before committing so a kill landing between the commit and the
	// SHA lookup below can still be recognized as a successful commit. The
	// failed-commit recovery reads "HEAD is not before" as "the commit landed", so
	// the snapshot is only usable as evidence when it is TRUSTWORTHY. HeadSHA draws
	// exactly that line: "" with a nil error ONLY for a genuinely unborn HEAD (an
	// empty repo), where no prior commit exists to confuse committedSHA. Every other
	// lookup failure -- ctx canceled mid-lookup, c.git's OWN gitOpTimeout firing
	// while the caller's ctx is still live (and so invisible to ctx.Err()), the
	// process group killed, a broken repository -- says nothing about HEAD, and
	// taking its empty result as a snapshot would make committedSHA compare a
	// pre-existing HEAD against "" and misreport that old commit as newly landed.
	before, err := c.HeadSHA(ctx)
	// A canceled context is worth reporting as itself: the commit below cannot
	// start, let alone land, so there is nothing to recover.
	if err != nil && ctx.Err() != nil {
		return "", err
	}
	// Otherwise still attempt the commit -- a commit worth making must not be lost
	// to a flaky lookup -- but remember the snapshot cannot be used as evidence.
	usableBefore := err == nil
	// --no-verify skips pre-commit/commit-msg hooks explicitly (gitenv.SafeConfigArgs
	// already disables them via core.hooksPath, so this is belt-and-suspenders):
	// an attacker-supplied .git/hooks must never run during a round commit.
	if out, err := c.git(ctx, "commit", "--no-verify", "-m", header, "-m", body); err != nil {
		// A kill landing AFTER git has updated the ref but before it exits cleanly
		// makes cmd.Run report "signal: killed" even though the commit landed. That
		// kill can come from ctx cancellation (KillProcessGroup SIGKILLs git) or from
		// c.git's OWN gitOpTimeout, which fires while the caller's ctx is still live
		// and so cannot be detected from ctx.Err() -- a slow signer or a large index
		// is enough. Re-read HEAD on a fresh context on ANY failure and accept the
		// commit only if HEAD advanced, rather than returning a bare commit error --
		// which would drop a landed commit and let the run summary claim no commit
		// while the commit sits in history and the tree looks clean to interruption
		// reconciliation. A genuinely failed commit leaves HEAD put and still errors.
		//
		// Only with a usable snapshot: unlike SquashSince, which anchors its recovery
		// on the SHA it just built, there is nothing here to compare HEAD against but
		// before. Without a trustworthy one, "HEAD is not before" is not evidence that
		// anything landed, and accepting it would report a PRE-EXISTING commit as this
		// round's -- leaving the coder's staged edits in the tree and skipping
		// reconciliation while the summary and journal claim a clean committed round.
		if usableBefore {
			if sha := c.committedSHA(before); sha != "" { //nolint:contextcheck // committedSHA deliberately re-reads HEAD on a fresh context: this one may be canceled or timed out, but a landed commit must still be recorded
				return sha, nil
			}
		}
		return "", fmt.Errorf("git commit: %w: %s", err, out)
	}
	sha, err := c.git(ctx, "rev-parse", "HEAD")
	if err == nil {
		return strings.TrimSpace(sha), nil
	}
	// The commit above SUCCEEDED, so HEAD has already advanced and whatever killed
	// this lookup says nothing about it: ctx canceled in the window before the
	// lookup, or c.git's OWN gitOpTimeout, which fires while the caller's ctx is
	// still live and so is invisible to ctx.Err(). Recover the landed commit's SHA
	// on a fresh context on ANY lookup failure rather than reporting a bare failed
	// commit -- which would drop the SHA from the run summary and send interruption
	// reconciliation looking for edits that are already committed. A lookup failing
	// because the repository itself is broken recovers nothing and still errors.
	//
	// This path needs no usable snapshot, unlike the failed-commit one above: the
	// commit EXITED CLEANLY, so whatever HEAD reads as now is the commit it made,
	// and before only has to be something HEAD cannot equal (it is "" when the
	// snapshot failed).
	if recovered := c.committedSHA(before); recovered != "" { //nolint:contextcheck // committedSHA deliberately re-reads HEAD on a fresh context: this one may be canceled or timed out, but a landed commit must still be recorded
		return recovered, nil
	}
	return "", err
}

// committedSHA re-reads HEAD on a fresh (still git-op-bounded) context after a
// commit command was interrupted -- by ctx cancellation or by the per-operation
// gitOpTimeout -- returning the new SHA if HEAD advanced past before (the commit
// landed) or "" if it did not. It exists so both the commit command's error path
// and the following rev-parse's error path recover a commit that landed just
// before the kill, instead of dropping its SHA and misreporting the round as a
// failed/interrupted commit.
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
		if err := c.restoreProtectedPath(ctx, p); err != nil {
			// stashed=true alongside the error, for every failure here including a
			// failed lookup: this point is only reached after `git stash push`
			// succeeded, so the coder's work IS in stash@{0} and recoverable with
			// `git stash pop` whichever step of the restoration then failed.
			// Reporting false would make the journal's "stashed: false"
			// indistinguishable from "there was nothing to stash" and hide the
			// recovery path during an already abnormal exit.
			return true, err
		}
	}
	// A clean `git stash` exit does not guarantee the tree actually became clean:
	// a top-level stash does not capture modifications inside a submodule working
	// tree, so reported dirt can outlive the stash. Verify the tree is clean
	// (under the same exclusions) before reporting a successful reconciliation --
	// otherwise interruption/salvage handling would treat a still-dirty tree as
	// clean and the next run would be blocked by dirt this call claimed to clear.
	// stashed=true alongside the error here for the same reason as the protected-path
	// restoration above: `git stash push` has already succeeded, so the coder's work
	// IS in stash@{0} whether the verification failed or found remaining dirt, and a
	// journaled "stashed: false" would be indistinguishable from "there was nothing
	// to stash" and hide the recovery path during an already abnormal exit.
	if clean, err := c.GitClean(ctx, exclude...); err != nil {
		return true, err
	} else if !clean {
		return true, errors.New("git stash exited cleanly but uncommitted changes remain (e.g. modifications inside a submodule working tree, which a top-level stash does not capture); reconcile the tree manually")
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

// collectPathspec is pathspec plus the target.exclude globs, the mandatory
// credential patterns, and every symlink whose DESTINATION is out of scope, for
// the git commands that COLLECT review material.
//
// listFiles applies EffectiveExcludes to directory mode; without this the git
// modes applied neither it nor target.exclude, and they are where it matters
// most: a directory listing only ever names a path, whereas `git diff` embeds the
// full file CONTENT into every reviewer prompt, into each agent CLI's arguments
// or stdin, and into the .prompt/.raw artifacts on disk. A PR that adds a .env or
// a deploy key would hand that key material verbatim to every reviewer, and
// agent.RedactSecrets is shape-based and best-effort. The `glob` magic gives git
// the same `**/` semantics compileGlobs gives the directory walk, and `icase` on
// the mandatory patterns the same case folding (see config.FoldExclude).
//
// Every spec above is name-based, and the name of a symlink says nothing about
// what it opens, so symlinkExcludes adds the resolved-destination check on top --
// the one directory mode already applies in listGitFiles and walkFiles.
//
// The positive "." spec comes from pathspec: collection is scoped to target.path,
// the same scope the clean check and the round commit use.
func (c *Collector) collectPathspec(ctx context.Context) ([]string, error) {
	specs := pathspec(c.excludes())
	for _, g := range c.cfg.EffectiveExcludes() {
		magic := "exclude,glob"
		if config.FoldExclude(g) {
			// The same folding compileGlobs applies, so a PRODUCTION.ENV the walk
			// drops does not arrive here with its content in the diff.
			magic += ",icase"
		}
		specs = append(specs, ":("+magic+")"+g)
	}
	scope, err := c.fileScope()
	if err != nil {
		return nil, err
	}
	aliases, err := c.symlinkExcludes(ctx, scope)
	if err != nil {
		return nil, err
	}
	return append(specs, aliases...), nil
}

// maxSymlinkExcludes bounds how many out-of-scope aliases collectPathspec will
// name. Each one becomes an argv entry on the diff and ls-files commands, and a
// repository can commit an unbounded number of them; past this many the
// collection is refused outright, rather than left to fail with an opaque
// "argument list too long" from exec -- or, worse, silently retried without the
// exclusions that keep the aliases out.
const maxSymlinkExcludes = 4096

// symlinkExcludes returns :(exclude,literal) pathspecs for every symlink in the
// collected scope that resolves somewhere a reviewer must not be pointed at (see
// symlinkOutOfScope).
//
// Without it the git modes filter by NAME alone, and no name-based pattern can
// see through an alias: a pull request that adds a tracked
// `docs/context.txt -> ~/.ssh/id_rsa` matches no mandatory credential pattern, so
// `git diff` renders it into the material handed to every reviewer and the
// untracked listing introduces it as a path to "read directly". Reviewers run
// unsandboxed and are told to read the repository for context, so the key ends up
// in a finding, in the .prompt/.md/.json artifacts, and -- in a fix run -- in the
// commit body, where redaction is shape-based and best-effort. pr mode is the mode
// built for untrusted authors, which is exactly why the check cannot be
// directory-mode-only.
//
// Lstat, not the index mode bits, decides what is a symlink: the worktree is what
// a reviewer would open, and it is what git itself diffs. An entry that cannot be
// stat'ed at all is left alone -- git cannot read it either, so there is no
// content for the collection to embed.
func (c *Collector) symlinkExcludes(ctx context.Context, scope fileScope) ([]string, error) {
	var specs []string
	overflow := 0
	// --cached covers the tracked aliases a PR commits, --others the untracked ones
	// the listing would advertise; -z keeps a path containing a newline intact.
	err := c.gitScanNUL(ctx, func(rel string) {
		if rel == "" {
			return
		}
		rel = filepath.ToSlash(rel)
		// Already dropped by name, so a second spec for it would only crowd argv.
		if c.skipFile(rel, scope.excludes) {
			return
		}
		fi, err := os.Lstat(filepath.Join(c.cfg.Path, rel))
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			return
		}
		if !c.symlinkOutOfScope(rel, scope) {
			return
		}
		if len(specs) >= maxSymlinkExcludes {
			overflow++
			return
		}
		specs = append(specs, ":(exclude,literal)"+rel)
	}, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("list symlinks in scope: %w", err)
	}
	if overflow > 0 {
		return nil, fmt.Errorf(
			"refusing to collect: %d symlinks under %s resolve outside it, %d more than the %d that can be excluded from the diff; narrow the review with target.exclude",
			len(specs)+overflow, c.cfg.Path, overflow, maxSymlinkExcludes)
	}
	return specs, nil
}

// restoreProtectedPath reproduces one excluded path's pre-stash index and
// worktree state from the stash's own trees, keeping the two independent so a
// staged-but-then-modified path keeps both sides: the index from the index
// commit (stash@{0}^2), the worktree from the worktree commit (stash@{0}). A
// path absent from a tree was deleted there at stash time (a plain deletion, or
// the old side of a rename), so its removal is reproduced rather than the file
// being resurrected to HEAD content.
//
// Every failure here leaves the stash itself intact -- see the call site, which is
// what turns that into the "stashed" flag StashDirty reports.
func (c *Collector) restoreProtectedPath(ctx context.Context, p string) error {
	// An operational failure here (cancellation, timeout, unreadable stash
	// object) must abort restoration -- never be mistaken for "the path was
	// deleted at stash time" and silently git rm / os.Remove an excluded path.
	inIndex, err := c.pathInTree(ctx, "stash@{0}^2", p)
	if err != nil {
		return fmt.Errorf("look up excluded path %s in stash index: %w", p, err)
	}
	if inIndex {
		if out, err := c.git(ctx, "restore", "--source=stash@{0}^2", "--staged", "--", p); err != nil {
			return fmt.Errorf("restore excluded path %s index after stash: %w: %s", p, err, out)
		}
	} else if out, err := c.git(ctx, "rm", "-q", "--cached", "--ignore-unmatch", "--", p); err != nil {
		return fmt.Errorf("stage removal of excluded path %s after stash: %w: %s", p, err, out)
	}
	inWorktree, err := c.pathInTree(ctx, "stash@{0}", p)
	if err != nil {
		return fmt.Errorf("look up excluded path %s in stash worktree: %w", p, err)
	}
	if inWorktree {
		if out, err := c.git(ctx, "restore", "--source=stash@{0}", "--worktree", "--", p); err != nil {
			return fmt.Errorf("restore excluded path %s worktree after stash: %w: %s", p, err, out)
		}
	} else if err := os.Remove(filepath.Join(c.cfg.Path, p)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove excluded path %s from worktree after stash: %w", p, err)
	}
	return nil
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
// such a target. gitenv's static pins cannot cover the filter/ssh/credential names
// (they are dynamic); a signing program COULD be force-disabled with
// -c commit.gpgSign=false, but refusing preserves legitimate signed round commits
// for a trusted operator instead of silently dropping their signatures.
//
// The settings that execute nothing but reroute git's network access are the same
// loss on the fetch path, and are judged by unsafeTransportConfigKey.
//
// The DIFF settings are the same class on the read path. diff.external replaces
// git's own diff engine with a program, and diff.<driver>.command/textconv do the
// same for whatever paths a `diff=<driver>` line in .gitattributes selects --
// textconv also runs during `git log -p`, `git grep` and `git blame`, not just
// `git diff`. Collect passes --no-ext-diff --no-textconv so fixpoint's own patch
// generation never fires them, but that flag pair only covers the command it is on:
// the reviewer and coder CLIs run their own `git diff`/`git log -p` inside the
// target, and no -c override can disable a driver whose name the repository
// chooses. Refusing is the only thing that covers both.
func unsafeConfigKey(key string) bool {
	switch {
	case attributeSelectableKey(key):
		return true
	case key == "core.sshcommand" || key == "core.askpass":
		return true
	case key == "diff.external":
		return true
	case strings.HasPrefix(key, "credential.") && strings.HasSuffix(key, ".helper"):
		return true
	case strings.HasPrefix(key, "gpg.") && strings.HasSuffix(key, ".program"):
		return true
	}
	return unsafeTransportConfigKey(key)
}

// unsafeTransportConfigKey reports whether a repo-supplied key changes WHERE git
// connects, WHAT it trusts there, or WHAT it sends -- on the fetch path this guard
// exists to protect, pr mode's `gh pr checkout` and Prepare's `git fetch` of the
// base object. Some of these run a program and some do not; the loss is the same
// either way, because the operator's credential helper answers an auth challenge
// from whatever host the fetch reaches and the objects that host serves become the
// reviewed "PR".
//
//   - url.<base>.insteadOf / pushInsteadOf rewrite the URL git actually contacts.
//     `[url "ext::sh -c <cmd>"] insteadOf = https://github.com/` leaves
//     remote.origin.url an ordinary GitHub URL (so gh still resolves the repo)
//     while every fetch goes through the ext:: helper protocol, which git runs as
//     a shell command. gitenv additionally pins protocol.ext.allow=never so
//     this particular shape is dead even before the guard sees it, but the rewrite
//     can target other transports too and the key belongs on the list.
//   - core.gitProxy is a program git runs for git:// transport.
//   - remote.<name>.uploadPack/receivePack/proxy are programs run for local and
//     file transports.
//   - the http.* section executes nothing and gets there anyway: a repo-local
//     `http.curloptResolve = github.com:443:<attacker ip>` plus
//     `http.sslVerify = false` keeps remote.origin.url an ordinary
//     https://github.com URL (so gh still resolves the PR) while the fetch behind
//     it contacts the attacker's host over an unverified connection. Git looks a
//     credential up by URL, not by resolved address, so the operator's github.com
//     token goes out the moment that host answers with a challenge.
//
// None of it can be neutralized by a -c pin: the names above are dynamic
// (subsections), and the http section's URL-specific forms
// (`http.https://github.com/.sslVerify`) beat any generic pin gitenv could carry,
// since git applies the most specific match. Refusing is the answer.
func unsafeTransportConfigKey(key string) bool {
	switch {
	case key == "core.gitproxy":
		return true
	case strings.HasPrefix(key, "url.") &&
		(strings.HasSuffix(key, ".insteadof") || strings.HasSuffix(key, ".pushinsteadof")):
		return true
	case strings.HasPrefix(key, "remote.") &&
		(strings.HasSuffix(key, ".uploadpack") || strings.HasSuffix(key, ".receivepack") || strings.HasSuffix(key, ".proxy")):
		return true
	case strings.HasPrefix(key, "http.") && !tuningHTTPConfigKey(key):
		return true
	}
	return false
}

// tuningHTTPConfigKey reports whether an http.* setting only tunes how a transfer
// is performed -- buffer sizes, timeouts, connection reuse, protocol version, the
// User-Agent -- and so is none of unsafeTransportConfigKey's business. Everything
// else in the section is refused.
//
// The http section is judged by allowlist, unlike every other class above, because
// its dangerous surface is broad (redirection, TLS trust, client certificates,
// cookie jars, unchallenged credential transmission) and still growing, while its
// harmless surface is this short and stable list. Naming the dangerous keys instead
// would silently trust the next one git adds; this way an unrecognized http setting
// costs a named refusal the operator can read, not a rerouted authenticated fetch.
func tuningHTTPConfigKey(key string) bool {
	// The variable is the part after the optional <url> subsection. git lowercases
	// section and variable names but preserves the subsection verbatim, and a URL
	// subsection contains dots of its own, so the variable is the segment after the
	// LAST dot: both `http.sslverify` and `http.https://github.com/.sslverify` yield
	// "sslverify".
	switch key[strings.LastIndexByte(key, '.')+1:] {
	case "postbuffer", "lowspeedlimit", "lowspeedtime", "maxrequests", "minsessions",
		"version", "useragent", "noepsv",
		"keepaliveidle", "keepaliveinterval", "keepalivecount":
		return true
	}
	return false
}

// attributeSelectableKey reports whether key defines a program that REPOSITORY
// CONTENT can select by name, whichever config scope the definition lives in:
//
//   - a per-name content filter -- filter.<name>.clean on stage-in, .smudge on
//     checkout, .process for a long-running filter -- selected by a
//     `filter=<name>` entry in .gitattributes;
//   - a per-name diff driver -- diff.<name>.command (an external diff program)
//     and diff.<name>.textconv (which also runs during `git log -p`, `git grep`
//     and `git blame`) -- selected by a `diff=<name>` entry in .gitattributes.
//
// .gitattributes IS repository content, so for both classes the definition is
// execution-capable no matter who wrote it, and no -c pin can disable a name the
// repository chooses. unsafeConfigKey refuses such a definition when the
// REPOSITORY supplies it; ExternalActivatableConfig reports it when the operator
// does and the repository can still activate it. The two share this predicate so
// they cannot drift: a key class one of them learns about is a key class the
// other must not miss.
func attributeSelectableKey(key string) bool {
	switch {
	case strings.HasPrefix(key, "filter.") &&
		(strings.HasSuffix(key, ".clean") || strings.HasSuffix(key, ".smudge") || strings.HasSuffix(key, ".process")):
		return true
	case strings.HasPrefix(key, "diff.") &&
		(strings.HasSuffix(key, ".command") || strings.HasSuffix(key, ".textconv")):
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
	all, err := c.repoScopedConfigKeys(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var keys []string
	for _, key := range all {
		if unsafeConfigKey(key) && !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

// ExternalActivatableConfig returns the sorted, de-duplicated content-filter and
// diff-driver keys that are defined OUTSIDE the repository's own scopes -- in the
// operator's global or system git config (or the command scope).
//
// UnsafeConfig deliberately drops those scopes, because a setting the operator
// configured is not the target's doing and refusing on it would refuse every
// target on the host. But for the attribute-selectable classes that reasoning only
// covers half the mechanism: a definition does nothing until a `filter=<name>` or
// `diff=<name>` attribute SELECTS it, and .gitattributes is repository content. So
// a program the operator installed globally -- `git lfs install` writes
// filter.lfs.clean/smudge/process into ~/.gitconfig, `nbdime config-git --enable
// --global` writes a diff.<name>.command, a pdftotext/exiftool textconv driver is
// a common habit, and Git for Windows ships diff.astextplain.textconv in its
// system config -- is still a program a hostile checkout can make git run over its
// own file content, with fixpoint's inherited environment: `gh pr checkout`
// applies the PR's .gitattributes (and its .lfsconfig, which redirects where a
// git-lfs filter talks), and every later git add/status/diff -- plus the reviewer
// and coder CLIs' own diff/log/blame inside the checkout -- re-runs it.
//
// This cannot become a refusal the way a repo-supplied definition does. The
// definition belongs to the operator, the selecting attributes arrive WITH the
// checkout (in pr mode they are not even in the worktree when the preflight runs,
// so there is nothing to cross-check against), and refusing would break every
// host with git-lfs installed. Callers report it instead, so the operator learns
// which of their own programs the checkout is able to activate.
func (c *Collector) ExternalActivatableConfig(ctx context.Context) ([]string, error) {
	entries, err := c.scopedConfigKeys(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.scope == "local" || e.scope == "worktree" {
			continue // the repository's own; UnsafeConfig judges those
		}
		if !attributeSelectableKey(e.key) || seen[e.key] {
			continue
		}
		seen[e.key] = true
		keys = append(keys, e.key)
	}
	sort.Strings(keys)
	return keys, nil
}

// repoScopedConfigKeys lists the config keys the REPOSITORY itself supplies, in
// file order and with duplicates kept (a key may be set more than once). It is
// the shared parse behind UnsafeConfig and the core.worktree check in
// WorktreeOutOfScope; see UnsafeConfig for why the listing must expand includes
// and cover the worktree scope, and why the scope filter is what keeps the
// operator's own global settings from reading as the target's.
func (c *Collector) repoScopedConfigKeys(ctx context.Context) ([]string, error) {
	entries, err := c.scopedConfigKeys(ctx)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		// Everything else (global, system, command -- our own gitenv.SafeConfigArgs -c
		// overrides land in `command`) is the operator's configuration, not the
		// target's, and refusing on it would be a false refusal.
		if e.scope != "local" && e.scope != "worktree" {
			continue
		}
		keys = append(keys, e.key)
	}
	return keys, nil
}

// configEntry is one row of the target's effective git config: the scope git
// attributes the setting to, and the setting's key (values are never returned --
// the checks here judge which programs git would run, not their arguments).
type configEntry struct{ scope, key string }

// scopedConfigKeys lists every config key a git command run inside the target
// would honor, paired with its scope, in file order and with duplicates kept (a
// key may be set more than once). Callers pick the scopes they care about:
// repoScopedConfigKeys keeps what the repository supplies,
// ExternalActivatableConfig keeps what the operator supplies and the repository
// can activate.
func (c *Collector) scopedConfigKeys(ctx context.Context) ([]configEntry, error) {
	// -z with --show-scope: NUL-separated fields alternating "scope" then
	// "key\nvalue" (a valueless key is just "key"). A real repo always has at
	// least the default core.* entries, so a git repo yields a non-error result.
	out, err := c.git(ctx, "config", "--list", "-z", "--show-scope", "--includes")
	if err != nil {
		return nil, fmt.Errorf("inspect target git config: %w", err)
	}
	fields := strings.Split(out, "\x00")
	var entries []configEntry
	for i := 0; i+1 < len(fields); i += 2 {
		scope, entry := fields[i], fields[i+1]
		key := entry
		if nl := strings.IndexByte(entry, '\n'); nl >= 0 {
			key = entry[:nl]
		}
		entries = append(entries, configEntry{scope: scope, key: key})
	}
	return entries, nil
}

// probeEnv is the environment for fixpoint's OWN git/gh subprocesses: the
// safe-config pins, plus a C locale.
//
// The locale pin is load-bearing, not tidiness. git translates its fatal
// messages when built with NLS, and notARepository/noWorkTree read those
// messages to tell an EXPECTED probe result -- "this is not a repository", "this
// is a bare repository" -- apart from a real fault, since exit 128 alone covers
// every fatal. Inheriting LANG/LC_* from whoever started fixpoint would make
// those reads miss under a translated git, turning an ordinary non-repository
// directory into an aborted preflight. LC_ALL beats LANG and every LC_*, and
// gettext ignores LANGUAGE once the locale is C, so nothing else needs clearing;
// exec resolves the duplicate key in favor of the last entry, so this wins over
// an inherited LC_ALL too.
//
// This deliberately does NOT live in gitenv.Harden: that env is also exported
// into the reviewer/coder CLIs, which must keep the user's own locale for their
// output encoding.
func (c *Collector) probeEnv() []string {
	return append(gitenv.Harden(nil), "LC_ALL=C")
}

func (c *Collector) git(ctx context.Context, args ...string) (string, error) {
	return c.run(ctx, "git", append(gitenv.SafeConfigArgs(), args...)...)
}

// gitInput runs a git command that reads its input from stdin (currently only
// `update-index --index-info`), with the same hardening as every other git call.
func (c *Collector) gitInput(ctx context.Context, stdin string, args ...string) (string, error) {
	return c.runInput(ctx, strings.NewReader(stdin), "git", append(gitenv.SafeConfigArgs(), args...)...)
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
	// never see our -c overrides -- still get the safe-config pins applied. gh pr
	// checkout runs `git checkout`, and a repo whose config points core.hooksPath into
	// the worktree (e.g. .githooks) would otherwise run a PR-supplied post-checkout
	// hook with fixpoint's inherited tokens, even in a review-only PR run before any
	// sandbox. GIT_CONFIG_COUNT/KEY/VALUE make git treat these as -c overrides. The
	// locale is pinned too, so the messages callers read stay the ones they parse.
	cmd.Env = c.probeEnv()
	// Stream stdout and stderr into SEPARATE bounded buffers rather than one:
	// callers parse the returned string as a machine-readable value (a SHA, a PR
	// base OID, a remote list, a filename list), and a successful git/gh command
	// that also writes a warning or notice to stderr would otherwise fold that
	// text into the parsed value -- e.g. a gh notice alongside baseRefOid makes PR
	// preparation hand the combined string to git as an invalid object name. On
	// success we return stdout only; stderr surfaces solely in the error. Bounding
	// each buffer still caps memory on a large diff/listing (each stream has its own
	// copy goroutine and its own buffer, so there is no Write x Write race).
	outBuf := agent.NewBoundedBuffer(maxRunOutput, agent.TruncationMarker(maxRunOutput))
	errBuf := agent.NewBoundedBuffer(maxRunOutput, agent.TruncationMarker(maxRunOutput))
	cmd.Stdin = stdin
	// Own the whole subprocess lifecycle, not just the leader, exactly as agent.Run
	// and verify.runOne do -- agent.Supervise is that shared discipline. A git/gh
	// that exits SUCCESSFULLY after spawning a child (a credential helper, a hook,
	// one of gh's internal git calls) must not leave it alive in our process group,
	// free to mutate the repository concurrently with the clean-tree check,
	// verification, or the round commit, or to leak past an aborted run: Supervise
	// kills the group the instant the leader exits, before draining its output.
	//
	// It also reports the leader's own status when only a drain timed out: a
	// `git commit` that landed must not be reported as failed -- that ends the run
	// with no CommitSHA while the commit sits in history, and sends reconciliation
	// looking for edits that are already committed. Callers parse stdout, so a
	// capture cut short surfaces as a parse error on its own rather than as a
	// silently wrong value.
	_, err := agent.Supervise(ctx, cmd, outBuf, errBuf)
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
// separators, * and ? within a segment. A glob config.FoldExclude marks -- the
// mandatory credential patterns -- compiles case-insensitively, which
// collectPathspec pairs with git's `icase` pathspec magic so both matchers drop
// the same files.
//
// A glob carrying NO wildcard at all names a plain directory -- "config/secrets",
// spelled with or without a trailing slash -- and so also matches everything
// BENEATH it. Without that, such an entry protects nothing where it matters most:
// the compiled globs are anchored ^...$ against full paths, and listGitFiles (the
// path taken for any git target) only ever sees FILE paths, because git ls-files
// emits no directory entries. So "config/secrets" would match neither
// config/secrets/prod.key nor anything else, silently, while the same entry pruned
// the directory in the non-git walk -- an operator excluding a directory of
// committed credentials got the protection only on a non-git target.
//
// Wildcard-free is exactly the shape git pathspecs already extend to descendants
// (match_pathspec_item compares such a spec as a leading-directory prefix), so the
// git modes' :(exclude,glob) specs in collectPathspec keep agreeing with the
// directory collectors. A glob carrying a wildcard ANYWHERE is left alone, even
// when the wildcard is not in its last segment: git matches those with wildmatch
// under WM_PATHNAME, where "**/credentials" matches a file named credentials and
// nothing under a directory of that name. Widening them here would both break that
// agreement and quietly delete whole source trees from review -- "**/credentials"
// is a mandatory exclude no config can drop, and credentials/ is an ordinary Go
// package name. Descendants of a matched name need the shape that says so:
// "**/vendor/**".
func compileGlobs(globs []string) ([]*regexp.Regexp, error) {
	res := make([]*regexp.Regexp, 0, len(globs))
	for _, g := range globs {
		var sb strings.Builder
		if config.FoldExclude(g) {
			sb.WriteString("(?i)")
		}
		// The trailing slash is only a spelling of "this is a directory"; drop it so
		// both spellings compile to the same pattern.
		g = strings.TrimSuffix(g, "/")
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
		if g != "" && !strings.ContainsAny(g, "*?") {
			// "(/.*)?" and not "/**": it also matches the "dir/" form walkFiles tests
			// directories with, so the walk still PRUNES the directory instead of
			// descending it to drop each file one at a time.
			sb.WriteString(`(/.*)?`)
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
