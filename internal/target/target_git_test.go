package target

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
)

// gitRepo creates a temporary git repository with one committed file and
// returns its path.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "test")
	git(t, dir, "config", "commit.gpgsign", "false")
	writeFile(t, dir, "main.go", "package main\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "initial")
	return dir
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsGitRepo(t *testing.T) {
	if ok, err := New(config.Target{Path: gitRepo(t)}).IsGitRepo(t.Context()); err != nil || !ok {
		t.Errorf("IsGitRepo() = %v, %v, want true for a git repo", ok, err)
	}
	// A plain directory is a CONFIRMED answer, not a failure: directory collection
	// falls back to the filesystem walk on it.
	if ok, err := New(config.Target{Path: t.TempDir()}).IsGitRepo(t.Context()); err != nil || ok {
		t.Errorf("IsGitRepo() = %v, %v, want false with no error for a plain directory", ok, err)
	}
}

// A probe that could not answer must NOT be reported as "no repository": directory
// collection would silently switch to the filesystem walk, which ignores
// .gitignore and so reviews a different set of files, and on Ctrl-C would spend the
// cancellation walking a large tree instead of stopping.
func TestIsGitRepoOperationalFailureIsAnError(t *testing.T) {
	repo := gitRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if ok, err := New(config.Target{Path: repo}).IsGitRepo(ctx); err == nil || ok {
		t.Errorf("IsGitRepo() = %v, %v on a canceled context, want false with an error", ok, err)
	}

	// A git that fails for a reason other than "not a repository" is likewise not an
	// answer. Exit 1 with no such message is the shape of an unreadable repository.
	shimGit(t, "rev-parse", "    echo 'fatal: unable to read the index' >&2\n    exit 1")
	if ok, err := New(config.Target{Path: repo}).IsGitRepo(t.Context()); err == nil || ok {
		t.Errorf("IsGitRepo() = %v, %v with a failing rev-parse, want false with an error", ok, err)
	}
}

// The same diversion, seen from the collection path: a failed probe must fail the
// round rather than quietly changing scope to the .gitignore-blind walk.
func TestCollectDirectoryDoesNotFallBackOnAProbeFailure(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, ".gitignore", "ignored/\n")
	writeFile(t, repo, "ignored/vendored.go", "package vendored\n")
	shimGit(t, "rev-parse", "    echo 'fatal: unable to read the index' >&2\n    exit 1")

	out, err := New(config.Target{Mode: "directory", Path: repo}).Collect(t.Context())
	if err == nil {
		t.Fatalf("Collect() = %q, nil; want a failed git-worktree probe to fail collection", out)
	}
	if strings.Contains(out, "vendored.go") {
		t.Errorf("collection fell back to the filesystem walk and picked up a gitignored file:\n%s", out)
	}
}

func TestGitClean(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Path: repo})

	clean, err := c.GitClean(t.Context())
	if err != nil || !clean {
		t.Fatalf("fresh repo: clean=%v err=%v, want clean", clean, err)
	}

	writeFile(t, repo, "dirty.txt", "x")
	clean, err = c.GitClean(t.Context())
	if err != nil || clean {
		t.Fatalf("untracked file: clean=%v err=%v, want dirty", clean, err)
	}

	// The excluded path must not count as dirt (this is how the run's own
	// logs directory is ignored).
	clean, err = c.GitClean(t.Context(), "dirty.txt")
	if err != nil || !clean {
		t.Fatalf("excluded dirt: clean=%v err=%v, want clean", clean, err)
	}
}

func TestPrepareAndCollectGitDiff(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD"})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if c.baseSHA == "" {
		t.Fatal("Prepare did not pin a base SHA")
	}

	// Commit a change AFTER the base is pinned: the diff must still show it.
	writeFile(t, repo, "main.go", "package main\n\nfunc changed() {}\n")
	git(t, repo, "commit", "-aqm", "change")
	writeFile(t, repo, "untracked.txt", "new")

	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Diff against pinned base " + c.baseSHA[:12],
		"func changed()",
		"Untracked files",
		"untracked.txt",
	} {
		if !strings.Contains(material, want) {
			t.Errorf("Collect() missing %q:\n%s", want, material)
		}
	}
}

func TestPrepareGitDiffBadRef(t *testing.T) {
	c := New(config.Target{Mode: "git-diff", Path: gitRepo(t), BaseRef: "no-such-ref"})
	if err := c.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "resolve base_ref") {
		t.Fatalf("Prepare() = %v, want resolve base_ref error", err)
	}
}

func TestCollectGitDiffUnstaged(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "git-diff", Path: repo}) // empty base_ref
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	writeFile(t, repo, "main.go", "package main // edited\n")
	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(material, "Unstaged working-tree changes") || !strings.Contains(material, "// edited") {
		t.Errorf("Collect() = %q", material)
	}
}

func TestCollectDirectory(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "pkg/a.go", "package pkg\n")
	writeFile(t, repo, "vendor/dep/b.go", "package dep\n")
	writeFile(t, repo, "README.md", "hi\n")
	// Scope is denylist-only: everything the exclude globs don't remove is in
	// scope, whatever its extension. README.md is listed here precisely because
	// there is no allowlist to leave it out.
	c := New(config.Target{
		Mode:    "directory",
		Path:    repo,
		Exclude: []string{"**/vendor/**"},
	})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(material, "Files in scope (3):") {
		t.Errorf("Collect() header wrong:\n%s", material)
	}
	for _, want := range []string{"main.go", "pkg/a.go", "README.md"} {
		if !strings.Contains(material, want) {
			t.Errorf("Collect() missing %q:\n%s", want, material)
		}
	}
	for _, notWant := range []string{"vendor", ".git"} {
		if strings.Contains(material, notWant) {
			t.Errorf("Collect() should not list %q:\n%s", notWant, material)
		}
	}
}

// A repository whose ls-files output exceeds maxRunOutput must still be counted
// exactly. Reading the listing through Collector.run capped stdout at 4 MB and
// appended a truncation marker, so every path past the cap vanished from scope
// with no diagnostic and the marker became a final pseudo-path -- leaving the
// header confidently reporting a total that undercounts the tree, which is the
// silent narrowing the denylist-only design exists to prevent.
func TestCollectDirectoryCountsPastTheRunOutputCap(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a repository with several MB of path text")
	}
	repo := gitRepo(t)
	// Long names so the cap is passed with as few files as possible: the point is
	// the total SIZE of the path list, not the file count.
	dirName := strings.Repeat("d", 180)
	base := strings.Repeat("f", 180)
	const dirs, perDir = 120, 120
	for d := range dirs {
		sub := filepath.Join(repo, fmt.Sprintf("%s%03d", dirName, d))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		for f := range perDir {
			// Untracked but not ignored, so --others --exclude-standard lists them
			// without the cost of staging every one.
			if err := os.WriteFile(filepath.Join(sub, fmt.Sprintf("%s%03d.go", base, f)), nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	// main.go from gitRepo is tracked and in scope too.
	want := dirs*perDir + 1
	if bytes := dirs * perDir * (len(dirName) + len(base) + 12); bytes <= maxRunOutput {
		t.Fatalf("the generated listing is only ~%d bytes, which does not exceed the %d-byte cap this guards", bytes, maxRunOutput)
	}

	c := New(config.Target{Mode: "directory", Path: repo})
	count, listing, err := c.listFiles(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Errorf("count = %d, want %d: paths past the output cap were dropped from scope", count, want)
	}
	// The RENDERED listing is still capped -- that is the prompt budget, and it is
	// the one thing that may be cut. It must not carry a run-level truncation
	// marker into the material either.
	if len(listing) > maxMaterial+1024 {
		t.Errorf("rendered listing is %d bytes, want it bounded near %d", len(listing), maxMaterial)
	}
	if strings.Contains(listing, "output truncated") {
		t.Errorf("the run-level truncation marker leaked into the listing as a path:\n%s", listing[max(0, len(listing)-500):])
	}
}

// ExcludeLogs must keep the run's own logs dir out of the collected material,
// in both directory mode (the walk) and git-diff/pr mode (untracked listing),
// so later rounds never review the run's own prompts and outputs.
func TestExcludeLogsFromCollect(t *testing.T) {
	t.Run("directory mode skips the logs dir", func(t *testing.T) {
		repo := gitRepo(t)
		writeFile(t, repo, "logs/run.prompt", "reviewed material and secrets\n")
		writeFile(t, repo, "pkg/a.go", "package pkg\n")
		c := New(config.Target{Mode: "directory", Path: repo})
		c.ExcludeLogs("logs")
		material, err := c.Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(material, "logs/run.prompt") {
			t.Errorf("Collect() listed the run's own logs:\n%s", material)
		}
		if !strings.Contains(material, "pkg/a.go") {
			t.Errorf("Collect() dropped an in-scope file:\n%s", material)
		}
	})

	t.Run("git-diff mode drops logs from untracked files", func(t *testing.T) {
		repo := gitRepo(t)
		c := New(config.Target{Mode: "git-diff", Path: repo}) // empty base_ref
		c.ExcludeLogs("logs")
		if err := c.Prepare(t.Context()); err != nil {
			t.Fatal(err)
		}
		writeFile(t, repo, "logs/run.prompt", "reviewed material\n") // untracked log
		writeFile(t, repo, "new.go", "package main\n")               // untracked source
		material, err := c.Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(material, "logs/run.prompt") {
			t.Errorf("Collect() listed the run's own logs as untracked:\n%s", material)
		}
		if !strings.Contains(material, "new.go") {
			t.Errorf("Collect() dropped a genuine untracked file:\n%s", material)
		}
	})

	t.Run("git-diff mode drops tracked log changes from the diff", func(t *testing.T) {
		repo := gitRepo(t)
		// A log file from an earlier run, committed under the logs dir.
		writeFile(t, repo, "logs/run.prompt", "old prompt\n")
		git(t, repo, "add", "logs/run.prompt")
		git(t, repo, "commit", "-q", "-m", "an earlier run's log")

		c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD"})
		c.ExcludeLogs("logs")
		if err := c.Prepare(t.Context()); err != nil {
			t.Fatal(err)
		}
		// Modify BOTH the tracked log and a real source file after pinning.
		writeFile(t, repo, "logs/run.prompt", "reviewed material and SECRETLOGCONTENT\n")
		writeFile(t, repo, "main.go", "package main // SOURCECHANGE\n")
		git(t, repo, "commit", "-aqm", "edit log and source")

		material, err := c.Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(material, "SOURCECHANGE") {
			t.Errorf("Collect() dropped the genuine source diff:\n%s", material)
		}
		if strings.Contains(material, "logs/run.prompt") || strings.Contains(material, "SECRETLOGCONTENT") {
			t.Errorf("Collect() included the tracked log's path or content in the diff:\n%s", material)
		}
	})
}

// pathspec() wraps each exclude in ":(exclude,literal)" so a logs directory
// whose name contains pathspec metacharacters (the documented example is a dir
// literally named "logs[1]") is matched as that exact path rather than
// interpreted as a glob. Dropping `literal` would read "logs[1]" as a
// character-class glob that matches "logs1" but NOT the literal directory, so
// its files would slip past the exclusion into a clean check, round commit, or
// diff. Every other test uses the metacharacter-free "logs", leaving this
// security-relevant invariant uncovered; assert it for GitClean, Commit, and
// (git-diff) Collect, the three paths that route the exclusion through pathspec.
func TestExcludeDirWithPathspecMetacharacters(t *testing.T) {
	const logs = "logs[1]"

	t.Run("GitClean excludes a metacharacter-named dir", func(t *testing.T) {
		repo := gitRepo(t)
		c := New(config.Target{Path: repo})
		writeFile(t, repo, logs+"/run.raw", "log dirt\n")
		if clean, err := c.GitClean(t.Context(), logs); err != nil || !clean {
			t.Errorf("GitClean(excluding %q) = %v, %v; want clean (its dirt is excluded)", logs, clean, err)
		}
	})

	t.Run("Commit excludes a metacharacter-named dir", func(t *testing.T) {
		repo := gitRepo(t)
		c := New(config.Target{Path: repo})
		writeFile(t, repo, "fixed.go", "package main\n")
		writeFile(t, repo, logs+"/run.raw", "log dirt\n")
		sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body", logs)
		if err != nil {
			t.Fatal(err)
		}
		if sha == "" {
			t.Fatal("expected a commit for fixed.go")
		}
		shown := git(t, repo, "show", "--name-only", "--format=", "HEAD")
		if !strings.Contains(shown, "fixed.go") {
			t.Errorf("commit missing fixed.go:\n%s", shown)
		}
		if strings.Contains(shown, "run.raw") {
			t.Errorf("commit staged the metacharacter-named excluded dir:\n%s", shown)
		}
	})

	t.Run("git-diff Collect excludes a metacharacter-named dir", func(t *testing.T) {
		repo := gitRepo(t)
		c := New(config.Target{Mode: "git-diff", Path: repo}) // empty base_ref
		c.ExcludeLogs(logs)
		if err := c.Prepare(t.Context()); err != nil {
			t.Fatal(err)
		}
		writeFile(t, repo, logs+"/run.prompt", "reviewed material SECRETLOGCONTENT\n")
		writeFile(t, repo, "new.go", "package main\n")
		material, err := c.Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(material, "run.prompt") || strings.Contains(material, "SECRETLOGCONTENT") {
			t.Errorf("Collect() listed the metacharacter-named logs dir:\n%s", material)
		}
		if !strings.Contains(material, "new.go") {
			t.Errorf("Collect() dropped a genuine untracked file:\n%s", material)
		}
	})
}

// A failure enumerating untracked files must surface, not be swallowed as "no
// untracked files": that would silently drop them from review. A git wrapper
// on PATH lets `git diff` through but fails `git ls-files`, so only the new
// error branch can produce the wrapped "list untracked files" error.
func TestCollectUntrackedListingErrorSurfaces(t *testing.T) {
	repo := gitRepo(t)
	binDir := t.TempDir()
	// Shim `git`: forward everything to the real git except `ls-files`, which
	// fails. `which git` resolves the real binary once PATH is shadowed, so pin
	// it up front. The subcommand is matched anywhere in the argument list, since
	// the collector prefixes every git call with -c hardening flags.
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	shim := "#!/bin/sh\n" +
		`for a in "$@"; do if [ "$a" = "ls-files" ]; then echo "boom" >&2; exit 1; fi; done` + "\n" +
		`exec "` + realGit + `" "$@"` + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(shim), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	c := New(config.Target{Mode: "git-diff", Path: repo}) // empty base_ref
	if _, err := c.Collect(t.Context()); err == nil || !strings.Contains(err.Error(), "list untracked files") {
		t.Fatalf("Collect() = %v, want wrapped \"list untracked files\" error", err)
	}
}

func TestCollectUnknownMode(t *testing.T) {
	if _, err := New(config.Target{Mode: "svn", Path: t.TempDir()}).Collect(t.Context()); err == nil {
		t.Fatal("Collect() = nil, want unknown mode error")
	}
}

func TestCommit(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Path: repo})

	// Clean tree: nothing to commit, no error.
	sha, err := c.Commit(t.Context(), "header", "body")
	if err != nil || sha != "" {
		t.Fatalf("clean commit: sha=%q err=%v, want empty sha", sha, err)
	}

	writeFile(t, repo, "fixed.go", "package main\n")
	writeFile(t, repo, "logs/run.raw", "log dirt")
	sha, err = c.Commit(t.Context(), "fixpoint: round 1", "Fixed:\n- bug", "logs")
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) < 12 {
		t.Fatalf("Commit sha = %q", sha)
	}

	msg := git(t, repo, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "fixpoint: round 1") || !strings.Contains(msg, "Fixed:\n- bug") {
		t.Errorf("commit message = %q", msg)
	}
	shown := git(t, repo, "show", "--stat", "--name-only", "HEAD")
	if !strings.Contains(shown, "fixed.go") {
		t.Errorf("commit does not contain fixed.go: %s", shown)
	}
	if strings.Contains(shown, "logs/run.raw") {
		t.Errorf("commit must not contain the excluded logs dir: %s", shown)
	}
	// Only the excluded dirt remains.
	if clean, err := c.GitClean(t.Context(), "logs"); err != nil || !clean {
		t.Errorf("tree not clean after commit (minus logs): clean=%v err=%v", clean, err)
	}
}

func TestSquashSince(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Path: repo})
	base := git(t, repo, "rev-parse", "HEAD")
	base = strings.TrimSpace(base)
	for _, name := range []string{"one.go", "two.go"} {
		writeFile(t, repo, name, "package main\n")
		if _, err := c.Commit(t.Context(), "fix "+name, "body"); err != nil {
			t.Fatal(err)
		}
	}

	sha, err := c.SquashSince(t.Context(), base, "fixpoint: round 1", "Fixed:\n- one\n- two")
	if err != nil {
		t.Fatal(err)
	}
	if head := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD")); head != sha {
		t.Fatalf("HEAD = %s, want the squashed commit %s", head, sha)
	}
	if parent := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD^")); parent != base {
		t.Errorf("squashed commit's parent = %s, want base %s", parent, base)
	}
	// The message must read like a per-fix commit: `git commit -m` cleanup applied.
	if msg := git(t, repo, "log", "-1", "--format=%B"); msg != "fixpoint: round 1\n\nFixed:\n- one\n- two\n\n" {
		t.Errorf("commit message = %q", msg)
	}
	// A squash is a pure regrouping: same content, clean tree, one commit.
	if files := git(t, repo, "show", "--name-only", "--format=", "HEAD"); !strings.Contains(files, "one.go") || !strings.Contains(files, "two.go") {
		t.Errorf("squashed commit files = %q", files)
	}
	if clean, err := c.GitClean(t.Context()); err != nil || !clean {
		t.Errorf("tree not clean after squash: clean=%v err=%v", clean, err)
	}
}

// base == "" squashes back to an unborn branch: the replacement is a root commit.
func TestSquashSinceUnbornBase(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "test")
	git(t, dir, "config", "commit.gpgsign", "false")
	c := New(config.Target{Path: dir})
	writeFile(t, dir, "main.go", "package main\n")
	if _, err := c.Commit(t.Context(), "fix", "body"); err != nil {
		t.Fatal(err)
	}

	sha, err := c.SquashSince(t.Context(), "", "fixpoint: run", "Fixed:\n- one")
	if err != nil {
		t.Fatal(err)
	}
	if head := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD")); head != sha {
		t.Fatalf("HEAD = %s, want the squashed commit %s", head, sha)
	}
	if count := strings.TrimSpace(git(t, dir, "rev-list", "--count", "HEAD")); count != "1" {
		t.Errorf("commit count = %s, want 1 root commit", count)
	}
	if clean, err := c.GitClean(t.Context()); err != nil || !clean {
		t.Errorf("tree not clean after squash: clean=%v err=%v", clean, err)
	}
}

// A squash that cannot create its replacement commit must leave the branch alone:
// rewinding first would strand the already-verified per-fix commits in the reflog
// with their content only staged. The failure is injected through signing --
// commit.gpgsign with a gpg that always fails -- because that breaks exactly the
// commit-creating step and nothing before it.
func TestSquashSinceFailureLeavesTheBranchIntact(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Path: repo})
	base := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	writeFile(t, repo, "one.go", "package main\n")
	if _, err := c.Commit(t.Context(), "fix one.go", "body"); err != nil {
		t.Fatal(err)
	}
	head := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "config", "commit.gpgsign", "true")
	git(t, repo, "config", "gpg.program", "/bin/false")

	if sha, err := c.SquashSince(t.Context(), base, "fixpoint: round 1", "Fixed:\n- one"); err == nil {
		t.Fatalf("SquashSince() = %q, want an error when the commit cannot be created", sha)
	}
	if now := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD")); now != head {
		t.Errorf("HEAD = %s after a failed squash, want the pre-squash commit %s", now, head)
	}
	if clean, err := c.GitClean(t.Context()); err != nil || !clean {
		t.Errorf("tree not clean after a failed squash: clean=%v err=%v", clean, err)
	}
}

// waitForFile polls until path exists (or the test times out), then returns.
// Used to synchronize a cancellation with a git shim that signals via a file.
func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

// r4.1/r5.5: a cancellation SIGKILL that lands AFTER git has updated the ref but
// before the commit command exits cleanly must be recognized as a landed commit
// (recovered via committedSHA), not misreported as a failed/interrupted commit
// that drops the SHA. A git shim runs the real commit, signals the test, then
// blocks so the test can cancel ctx mid-command -- reproducing the kill-after-ref
// -update window on the commit command's own error path.
func TestCommitRecoversLandedCommitOnCancelDuringCommit(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skipf("sh not found: %v", err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	repo := gitRepo(t)
	binDir := t.TempDir()
	sentinel := filepath.Join(binDir, "committing")
	shim := "#!/bin/sh\n" +
		`is_commit=0; for a in "$@"; do if [ "$a" = "commit" ]; then is_commit=1; fi; done` + "\n" +
		`if [ "$is_commit" = 1 ]; then "` + realGit + `" "$@"; st=$?; touch "` + sentinel + `"; sleep 30; exit $st; fi` + "\n" +
		`exec "` + realGit + `" "$@"` + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(shim), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeFile(t, repo, "fixed.go", "package main\n")

	c := New(config.Target{Path: repo})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		waitForFile(t, sentinel)
		cancel() // ref has landed; kill the commit command mid-flight
	}()
	sha, err := c.Commit(ctx, "fixpoint: round 1", "body")
	if err != nil {
		t.Fatalf("Commit() = %v, want the landed commit recovered", err)
	}
	head := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	if sha == "" || sha != head {
		t.Fatalf("Commit() SHA = %q, want landed HEAD %q", sha, head)
	}
	if out := git(t, repo, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Errorf("tree not clean after recovered commit: %q", out)
	}
}

// Same recovery, but the cancellation lands on the rev-parse that reads the new
// SHA AFTER a clean commit (target.go's second committedSHA call site). The shim
// lets the commit finish, then blocks the post-commit rev-parse so the test can
// cancel there; committedSHA must still recover the landed SHA.
func TestCommitRecoversLandedCommitOnCancelDuringRevParse(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skipf("sh not found: %v", err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	repo := gitRepo(t)
	binDir := t.TempDir()
	sentinel := filepath.Join(binDir, "revparsing")
	marker := filepath.Join(binDir, "commit-done")
	// commit runs normally and drops a marker; the FIRST rev-parse seen after the
	// marker exists (the post-commit SHA lookup) removes the marker, signals, and
	// blocks. Removing it first lets committedSHA's own rev-parse pass straight
	// through once this one is killed, so the test does not stall on the shim.
	shim := "#!/bin/sh\n" +
		`op=""; for a in "$@"; do case "$a" in commit) op=commit;; rev-parse) op=revparse;; esac; done` + "\n" +
		`if [ "$op" = commit ]; then "` + realGit + `" "$@"; st=$?; touch "` + marker + `"; exit $st; fi` + "\n" +
		`if [ "$op" = revparse ] && [ -f "` + marker + `" ]; then rm -f "` + marker + `"; touch "` + sentinel + `"; sleep 30; fi` + "\n" +
		`exec "` + realGit + `" "$@"` + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(shim), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeFile(t, repo, "fixed.go", "package main\n")

	c := New(config.Target{Path: repo})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		waitForFile(t, sentinel)
		cancel() // commit landed; kill the post-commit rev-parse
	}()
	sha, err := c.Commit(ctx, "fixpoint: round 1", "body")
	if err != nil {
		t.Fatalf("Commit() = %v, want the landed commit recovered", err)
	}
	head := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	if sha == "" || sha != head {
		t.Fatalf("Commit() SHA = %q, want landed HEAD %q", sha, head)
	}
	if out := git(t, repo, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Errorf("tree not clean after recovered commit: %q", out)
	}
}

// gitSafeConfig points core.hooksPath at /dev/null so a hook shipped in an
// untrusted target's .git never runs with fixpoint's privileges. Regression
// guard for that mitigation: a pre-commit hook (also blocked by --no-verify) and
// a post-commit hook (which --no-verify does NOT skip, so only hooksPath stops
// it) must both stay silent during a Commit.
func TestCommitDoesNotRunGitHooks(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to run hook scripts")
	}
	repo := gitRepo(t)
	hooksDir := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	preRan := filepath.Join(repo, "pre-commit-ran")
	postRan := filepath.Join(repo, "post-commit-ran")
	writeHook(t, hooksDir, "pre-commit", preRan)
	writeHook(t, hooksDir, "post-commit", postRan)

	c := New(config.Target{Path: repo})
	writeFile(t, repo, "fixed.go", "package main\n")
	if _, err := c.Commit(t.Context(), "fixpoint: round 1", "body"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if _, err := os.Stat(preRan); err == nil {
		t.Error("pre-commit hook executed despite the hooks-disabling mitigation")
	}
	if _, err := os.Stat(postRan); err == nil {
		t.Error("post-commit hook executed; the core.hooksPath mitigation is not in effect")
	}
}

// gh pr checkout runs git subprocesses that never see c.git's -c overrides, so a
// repo whose config points core.hooksPath into the worktree could otherwise run a
// PR-supplied hook. gitHardenedEnv propagates the hooks-disabling override to
// every subprocess via GIT_CONFIG_*. Regression guard: a checkout run through
// c.run (the raw path gh's nested git takes -- no -c overrides) must not execute
// a worktree post-checkout hook.
func TestRunEnvDisablesWorktreeHooks(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to run hook scripts")
	}
	repo := gitRepo(t)
	hooksDir := filepath.Join(repo, ".githooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ran := filepath.Join(repo, "post-checkout-ran")
	writeHook(t, hooksDir, "post-checkout", ran)
	// core.hooksPath in the repo's own config, worktree-relative, is exactly the
	// setting a PR can exploit; git(t,...) writes it via .git/config.
	git(t, repo, "config", "core.hooksPath", ".githooks")
	git(t, repo, "branch", "other")

	c := New(config.Target{Path: repo})
	// c.run invokes git WITHOUT the -c overrides, so only the hardened env can
	// disable the hook -- the same conditions gh's internal git runs under.
	if out, err := c.run(t.Context(), "git", "checkout", "other"); err != nil {
		t.Fatalf("checkout: %v: %s", err, out)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("post-checkout hook executed; gitHardenedEnv did not disable hooks for a subprocess lacking -c overrides")
	}
}

// The ext:: transport's URL IS a command git runs, so a remote pointed at one
// turns any fetch into code execution with fixpoint's inherited environment.
// git itself defaults protocol.ext.allow to "never", but that default is
// CONFIGURABLE: a crafted .git/config that sets protocol.ext.allow=always turns
// it back on, and remote.<name>.url is not a key unsafeConfigKey refuses. The
// gitSafeConfig pin is what beats the repo's own value -- on both paths, since
// gh's internal git never sees the -c overrides and gets them from
// GIT_CONFIG_* instead.
func TestFetchDoesNotRunAnExtTransportHelper(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to run the transport helper")
	}
	for _, tc := range []struct {
		name  string
		fetch func(*Collector, string) (string, error)
	}{
		// c.git prepends gitSafeConfig as -c overrides AND sets the hardened env.
		{"git", func(c *Collector, remote string) (string, error) {
			return c.git(t.Context(), "fetch", "--no-tags", remote)
		}},
		// c.run invokes git with NO -c overrides -- the conditions gh's nested git
		// runs under, where only GIT_CONFIG_* from gitHardenedEnv can protect it.
		{"run", func(c *Collector, remote string) (string, error) {
			return c.run(t.Context(), "git", "fetch", "--no-tags", remote)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := gitRepo(t)
			sentinel := filepath.Join(t.TempDir(), "ext-helper-ran")
			helper := filepath.Join(repo, "evil-remote-helper.sh")
			// The helper touches the sentinel and then fails: reaching it at all is
			// the compromise, whether or not it can speak the remote-helper protocol.
			if err := os.WriteFile(helper, []byte("#!/bin/sh\ntouch '"+sentinel+"'\nexit 1\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			git(t, repo, "config", "remote.evil.url", "ext::"+helper)
			// The repo re-enables the transport git disables by default. Without the
			// pin the fetch below runs the helper; with it, git refuses the transport.
			git(t, repo, "config", "protocol.ext.allow", "always")

			out, err := tc.fetch(New(config.Target{Path: repo}), "evil")
			if err == nil {
				t.Fatalf("fetch through an ext:: remote succeeded: %s", out)
			}
			if !strings.Contains(err.Error(), "transport 'ext' not allowed") {
				t.Errorf("want git's ext-transport refusal, got: %v", err)
			}
			if _, serr := os.Stat(sentinel); serr == nil {
				t.Error("the ext:: helper executed; protocol.ext.allow=never is not reaching this git invocation")
			}
		})
	}
}

// writeHook installs an executable git hook that touches sentinel when run.
func writeHook(t *testing.T, hooksDir, name, sentinel string) {
	t.Helper()
	script := "#!/bin/sh\ntouch '" + sentinel + "'\n"
	if err := os.WriteFile(filepath.Join(hooksDir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// The read-only collection diff must not execute a repository-controlled
// diff.external helper: --no-ext-diff keeps a crafted .git/config from turning
// git-diff collection into code execution with fixpoint's credentials.
func TestCollectDoesNotRunExternalDiff(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to run the diff helper")
	}
	repo := gitRepo(t)
	sentinel := filepath.Join(repo, "external-diff-ran")
	evil := filepath.Join(repo, "evil-diff.sh")
	if err := os.WriteFile(evil, []byte("#!/bin/sh\ntouch '"+sentinel+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "config", "diff.external", evil)
	writeFile(t, repo, "main.go", "package main\n\nvar changed = true\n")

	c := New(config.Target{Path: repo, Mode: config.ModeGitDiff})
	if _, err := c.Collect(t.Context()); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("external diff helper executed; the --no-ext-diff mitigation is not in effect")
	}
}

// The read-only collection diff must not execute a .gitattributes-selected
// textconv driver: --no-textconv (r4.2) blocks it. Unlike diff.external (blocked
// by --no-ext-diff), a textconv driver runs per blob to render human-readable
// diffs, so a crafted .gitattributes + diff driver would otherwise turn git-diff
// collection into code execution with fixpoint's credentials before any agent
// sandboxing, even in a review-only run.
func TestCollectDoesNotRunTextconv(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to run the textconv helper")
	}
	repo := gitRepo(t)
	sentinel := filepath.Join(repo, "textconv-ran")
	evil := filepath.Join(repo, "evil-textconv.sh")
	// A textconv command receives the blob path and must print its text; touch a
	// sentinel as the side effect --no-textconv is meant to prevent.
	if err := os.WriteFile(evil, []byte("#!/bin/sh\ntouch '"+sentinel+"'\ncat \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Select a diff driver for *.go via .gitattributes, and give that driver a
	// textconv command. git honors an untracked .gitattributes for diffs.
	git(t, repo, "config", "diff.evil.textconv", evil)
	writeFile(t, repo, ".gitattributes", "*.go diff=evil\n")
	writeFile(t, repo, "main.go", "package main\n\nvar changed = true\n") // modify the tracked *.go

	c := New(config.Target{Path: repo, Mode: config.ModeGitDiff})
	if _, err := c.Collect(t.Context()); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("textconv driver executed; the --no-textconv mitigation is not in effect")
	}
}

// Regression: the excluded logs dir being .gitignore'd must not fail the
// commit. git add refuses pathspecs naming ignored paths (even in :(exclude)
// form), so ignored excludes are dropped from the pathspec -- git skips them
// on its own.
func TestCommitWithGitignoredLogs(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, ".gitignore", "logs/\n")
	git(t, repo, "add", ".gitignore")
	git(t, repo, "commit", "-q", "-m", "ignore logs")
	c := New(config.Target{Path: repo})

	writeFile(t, repo, "fixed.go", "package main\n")
	writeFile(t, repo, "logs/run.raw", "log dirt")
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "Fixed:\n- bug", "logs")
	if err != nil {
		t.Fatalf("Commit with gitignored logs: %v", err)
	}
	if sha == "" {
		t.Fatal("expected a commit")
	}
	shown := git(t, repo, "show", "--stat", "--name-only", "HEAD")
	if !strings.Contains(shown, "fixed.go") {
		t.Errorf("commit does not contain fixed.go: %s", shown)
	}
	if strings.Contains(shown, "logs/run.raw") {
		t.Errorf("commit must not contain the ignored logs dir: %s", shown)
	}
}

// TestPrepareAndCollectPR exercises pr mode against a stub gh CLI: Prepare
// must check out the PR branch, pin the merge base with the PR's base branch,
// and Collect must diff against that pin.
func TestPrepareAndCollectPR(t *testing.T) {
	repo := gitRepo(t)
	git(t, repo, "branch", "-M", "main")
	mainSHA := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "checkout", "-q", "-b", "feature")
	writeFile(t, repo, "main.go", "package main\n\nfunc prChange() {}\n")
	git(t, repo, "commit", "-aqm", "pr change")
	git(t, repo, "checkout", "-q", "main")

	// Stub gh on PATH: checkout switches to the PR branch, view prints the
	// base commit oid; every call is logged for verification. The base commit
	// is already reachable locally, so Prepare never has to fetch it.
	binDir := t.TempDir()
	callLog := filepath.Join(binDir, "calls.log")
	stub := "#!/bin/sh\n" +
		`echo "$@" >> "` + callLog + "\"\n" +
		`case "$1 $2" in` + "\n" +
		`"pr checkout") git checkout -q feature ;;` + "\n" +
		`"pr view") echo ` + mainSHA + " ;;\n" +
		`*) echo "unexpected gh call: $@" >&2; exit 1 ;;` + "\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	c := New(config.Target{Mode: "pr", Path: repo, PR: 7})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if c.baseSHA != mainSHA {
		t.Errorf("baseSHA = %q, want merge base %q", c.baseSHA, mainSHA)
	}
	if head := strings.TrimSpace(git(t, repo, "rev-parse", "--abbrev-ref", "HEAD")); head != "feature" {
		t.Errorf("HEAD = %q, want feature (PR checked out)", head)
	}
	calls, err := os.ReadFile(callLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"pr checkout 7", "pr view 7 --json baseRefOid --jq .baseRefOid"} {
		if !strings.Contains(string(calls), want) {
			t.Errorf("gh calls missing %q:\n%s", want, calls)
		}
	}

	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Diff against pinned base " + mainSHA[:12], "func prChange()"} {
		if !strings.Contains(material, want) {
			t.Errorf("Collect() missing %q:\n%s", want, material)
		}
	}
}

// A failing gh must surface as a Prepare error, not a silent empty base.
func TestPreparePRCheckoutFails(t *testing.T) {
	repo := gitRepo(t)
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte("#!/bin/sh\necho boom >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	c := New(config.Target{Mode: "pr", Path: repo, PR: 7})
	if err := c.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "gh pr checkout") {
		t.Fatalf("Prepare() = %v, want gh pr checkout error", err)
	}
}

func TestAtRepoRoot(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "sub/x.go", "package sub\n")
	if at, err := New(config.Target{Path: repo}).AtRepoRoot(t.Context()); err != nil || !at {
		t.Fatalf("AtRepoRoot(root) = %v, %v; want true", at, err)
	}
	sub := filepath.Join(repo, "sub")
	if at, err := New(config.Target{Path: sub}).AtRepoRoot(t.Context()); err != nil || at {
		t.Fatalf("AtRepoRoot(subdir) = %v, %v; want false", at, err)
	}
}

// Regression: a file that is BOTH gitignored AND already tracked still gets its
// modifications staged by git add -A despite the ignore rule; the exclusion
// must reset it out of the index so it never lands in a round commit.
func TestCommitExcludesTrackedFileUnderIgnoredLogs(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "logs/run.log", "old\n")
	git(t, repo, "add", "-f", "logs/run.log") // force-track under the soon-ignored dir
	writeFile(t, repo, ".gitignore", "logs/\n")
	git(t, repo, "add", ".gitignore")
	git(t, repo, "commit", "-q", "-m", "track a log then ignore the dir")

	writeFile(t, repo, "fixed.go", "package main\n")
	writeFile(t, repo, "logs/run.log", "new secret content\n") // tracked-but-ignored edit

	c := New(config.Target{Path: repo})
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body", "logs")
	if err != nil {
		t.Fatal(err)
	}
	if sha == "" {
		t.Fatal("expected a commit for fixed.go")
	}
	shown := git(t, repo, "show", "--name-only", "--format=", "HEAD")
	if !strings.Contains(shown, "fixed.go") {
		t.Errorf("commit missing fixed.go:\n%s", shown)
	}
	if strings.Contains(shown, "logs/run.log") {
		t.Errorf("commit staged the tracked-but-excluded log file:\n%s", shown)
	}
}

// A round commit must leave the excluded paths' INDEX alone too, not just keep them
// out of the commit. GitClean deliberately ignores the exclusion, so a run may
// legitimately start with staged changes under an excluded path -- and Commit walks
// the caller's real index: `git add -A` replaces the staged version with the
// worktree version and the reset then collapses the entry to HEAD, so a staged-only
// version would be destroyed by a commit that is supposed to leave the path
// untouched (the blob is reachable from nothing afterwards).
func TestCommitPreservesStagedStateUnderExcludedPaths(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "logs/run.log", "committed\n")
	git(t, repo, "add", "logs/run.log")
	git(t, repo, "commit", "-q", "-m", "a tracked log")

	// Staged one thing, then modified further in the worktree: the two differ, so
	// only an exact restore keeps both sides.
	writeFile(t, repo, "logs/run.log", "staged\n")
	git(t, repo, "add", "logs/run.log")
	stagedBlob := strings.Fields(git(t, repo, "ls-files", "--stage", "--", "logs/run.log"))[1]
	writeFile(t, repo, "logs/run.log", "worktree\n")
	writeFile(t, repo, "fixed.go", "package main\n") // the round's own work

	c := New(config.Target{Path: repo})
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body", "logs")
	if err != nil {
		t.Fatal(err)
	}
	if sha == "" {
		t.Fatal("expected a commit for fixed.go")
	}
	if shown := git(t, repo, "show", "--name-only", "--format=", "HEAD"); strings.Contains(shown, "logs/run.log") {
		t.Errorf("the round commit carried an excluded path:\n%s", shown)
	}
	if got := strings.Fields(git(t, repo, "ls-files", "--stage", "--", "logs/run.log"))[1]; got != stagedBlob {
		t.Errorf("staged blob = %s, want %s: the round commit destroyed the staged version of an excluded path", got, stagedBlob)
	}
	if b, err := os.ReadFile(filepath.Join(repo, "logs", "run.log")); err != nil || string(b) != "worktree\n" {
		t.Errorf("worktree content = %q (%v), want it untouched", b, err)
	}
}

// The same guarantee when staging fails PART WAY through: `git add -A` has already
// replaced the staged-only version of an excluded path with the worktree version, so
// a bare return from the failing reset would leave that version unreachable -- and
// interruption reconciliation excludes the path too, so nothing else puts it back.
// Restoration therefore has to run on every path out of Commit, not just after a
// completed commit.
func TestCommitRestoresExcludedIndexWhenStagingFails(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "logs/run.log", "committed\n")
	git(t, repo, "add", "logs/run.log")
	git(t, repo, "commit", "-q", "-m", "a tracked log")

	writeFile(t, repo, "logs/run.log", "staged\n")
	git(t, repo, "add", "logs/run.log")
	stagedBlob := strings.Fields(git(t, repo, "ls-files", "--stage", "--", "logs/run.log"))[1]
	writeFile(t, repo, "logs/run.log", "worktree\n")
	writeFile(t, repo, "fixed.go", "package main\n") // the round's own work

	// Fail the unstaging step only: the add before it runs for real, so the index
	// genuinely holds the worktree version by the time Commit gives up.
	shimGit(t, "reset", "    echo 'boom' >&2\n    exit 1")

	c := New(config.Target{Path: repo})
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body", "logs")
	if err == nil {
		t.Fatal("Commit() = nil, want the failed reset to surface")
	}
	if sha != "" {
		t.Errorf("Commit() sha = %q, want none: no commit was made", sha)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("Commit() err = %v, want it to carry git's stderr", err)
	}
	if got := strings.Fields(git(t, repo, "ls-files", "--stage", "--", "logs/run.log"))[1]; got != stagedBlob {
		t.Errorf("staged blob = %s, want %s: the failed round destroyed the staged version of an excluded path", got, stagedBlob)
	}
}

// The same guarantee for an excluded path staged as an ADDITION: it is absent from
// HEAD, so restoring it means re-adding an entry the reset removed outright.
func TestCommitPreservesStagedAdditionUnderExcludedPaths(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "logs/new.log", "staged addition\n")
	git(t, repo, "add", "logs/new.log")
	stagedBlob := strings.Fields(git(t, repo, "ls-files", "--stage", "--", "logs/new.log"))[1]
	writeFile(t, repo, "fixed.go", "package main\n")

	c := New(config.Target{Path: repo})
	if _, err := c.Commit(t.Context(), "fixpoint: round 1", "body", "logs"); err != nil {
		t.Fatal(err)
	}
	if shown := git(t, repo, "show", "--name-only", "--format=", "HEAD"); strings.Contains(shown, "logs/new.log") {
		t.Errorf("the round commit carried an excluded path:\n%s", shown)
	}
	entry := git(t, repo, "ls-files", "--stage", "--", "logs/new.log")
	if !strings.Contains(entry, stagedBlob) {
		t.Errorf("index entry = %q, want the staged addition %s preserved", entry, stagedBlob)
	}
}

// And for an excluded path staged as a DELETION, the one case with no index entry
// to save: ls-files emits nothing for it, so the snapshot is empty even though the
// reset in Commit resurrects the path's HEAD entry. Restoration has to run anyway
// and empty the index under the exclusion, or the round commit silently un-deletes
// the user's staged removal.
func TestCommitPreservesStagedDeletionUnderExcludedPaths(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "logs/run.log", "committed\n")
	git(t, repo, "add", "logs/run.log")
	git(t, repo, "commit", "-q", "-m", "a tracked log")

	git(t, repo, "rm", "-q", "--", "logs/run.log") // staged deletion under the exclusion
	writeFile(t, repo, "fixed.go", "package main\n")

	c := New(config.Target{Path: repo})
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body", "logs")
	if err != nil {
		t.Fatal(err)
	}
	if sha == "" {
		t.Fatal("expected a commit for fixed.go")
	}
	if shown := git(t, repo, "show", "--name-only", "--format=", "HEAD"); strings.Contains(shown, "logs/run.log") {
		t.Errorf("the round commit carried an excluded path:\n%s", shown)
	}
	if entry := git(t, repo, "ls-files", "--stage", "--", "logs/run.log"); strings.TrimSpace(entry) != "" {
		t.Errorf("index entry = %q, want none: the round commit undid the staged deletion of an excluded path", entry)
	}
}

// A repository with no commits yet is a supported target (HeadSHA and SquashSince
// both treat an unborn branch as a valid starting point), so the FIRST round commit
// must land there with the default logs exclusion in play. `git reset HEAD` resets
// to the empty tree on an unborn branch rather than failing -- which is also why an
// excluded path staged before the run still has to be restored afterwards: there is
// no HEAD version for the reset to leave behind.
func TestCommitOnUnbornBranchWithExcludedPaths(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "test")
	git(t, dir, "config", "commit.gpgsign", "false")

	writeFile(t, dir, "logs/new.log", "staged addition\n")
	git(t, dir, "add", "logs/new.log")
	stagedBlob := strings.Fields(git(t, dir, "ls-files", "--stage", "--", "logs/new.log"))[1]
	writeFile(t, dir, "main.go", "package main\n") // the round's own work

	c := New(config.Target{Path: dir})
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body", "logs")
	if err != nil {
		t.Fatal(err)
	}
	if sha == "" {
		t.Fatal("expected the repository's first commit to land")
	}
	shown := git(t, dir, "show", "--name-only", "--format=", "HEAD")
	if !strings.Contains(shown, "main.go") {
		t.Errorf("first commit does not contain main.go:\n%s", shown)
	}
	if strings.Contains(shown, "logs/new.log") {
		t.Errorf("first commit carried an excluded path:\n%s", shown)
	}
	if entry := git(t, dir, "ls-files", "--stage", "--", "logs/new.log"); !strings.Contains(entry, stagedBlob) {
		t.Errorf("index entry = %q, want the staged addition %s preserved", entry, stagedBlob)
	}
}

// The empty base is reserved for a genuinely unborn branch: SquashSince turns it
// into a ROOT commit, so an operational rev-parse failure reported as "" would cut
// the repository's history off from the branch under per_round/per_run. Only git's
// quiet exit 1 means unborn; a fatal (here, no repository at all) must surface.
func TestHeadSHAUnbornVersusOperationalFailure(t *testing.T) {
	unborn := t.TempDir()
	git(t, unborn, "init", "-q")
	sha, err := New(config.Target{Path: unborn}).HeadSHA(t.Context())
	if err != nil || sha != "" {
		t.Errorf("HeadSHA(unborn) = %q, %v; want \"\", nil", sha, err)
	}

	sha, err = New(config.Target{Path: t.TempDir()}).HeadSHA(t.Context())
	if err == nil {
		t.Errorf("HeadSHA(not a repository) = %q, nil; want an error rather than an unborn-branch answer", sha)
	}
}

func TestStashDirty(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Path: repo})

	if stashed, err := c.StashDirty(t.Context(), "m", "logs"); err != nil || stashed {
		t.Fatalf("StashDirty(clean) = %v, %v; want false, nil", stashed, err)
	}

	writeFile(t, repo, "main.go", "package main // edited\n") // tracked mod
	writeFile(t, repo, "new.txt", "new\n")                    // untracked
	writeFile(t, repo, "logs/x.raw", "log\n")                 // excluded dirt
	stashed, err := c.StashDirty(t.Context(), "recover round 1", "logs")
	if err != nil || !stashed {
		t.Fatalf("StashDirty(dirty) = %v, %v; want true, nil", stashed, err)
	}
	if clean, err := c.GitClean(t.Context(), "logs"); err != nil || !clean {
		t.Fatalf("tree not clean after stash: clean=%v err=%v", clean, err)
	}
	if _, err := os.Stat(filepath.Join(repo, "logs", "x.raw")); err != nil {
		t.Errorf("excluded logs must survive the stash: %v", err)
	}
	if list := git(t, repo, "stash", "list"); !strings.Contains(list, "recover round 1") {
		t.Errorf("stash entry missing: %q", list)
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "main.go")); strings.Contains(string(b), "edited") {
		t.Errorf("tracked edit not stashed away: %q", b)
	}
}

// A tracked file under a gitignored excluded directory must be left untouched by
// StashDirty: git ignore rules do not apply to tracked files, so a naive `git
// stash push -- .` would sweep in (or, with a stale :(exclude) pathspec, fail
// on) that modified file -- exactly the changes StashDirty promises to preserve.
func TestStashDirtyPreservesTrackedFileUnderIgnoredLogs(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "logs/run.log", "old\n")
	git(t, repo, "add", "-f", "logs/run.log") // force-track under the soon-ignored dir
	writeFile(t, repo, ".gitignore", "logs/\n")
	git(t, repo, "add", ".gitignore")
	git(t, repo, "commit", "-q", "-m", "track a log then ignore the dir")

	writeFile(t, repo, "main.go", "package main // edited\n") // coder edit to stash away
	writeFile(t, repo, "logs/run.log", "user edit\n")         // tracked-but-excluded change to keep
	writeFile(t, repo, "logs/output.raw", "run output\n")     // untracked-ignored, left in place

	c := New(config.Target{Path: repo})
	stashed, err := c.StashDirty(t.Context(), "recover round 1", "logs")
	if err != nil || !stashed {
		t.Fatalf("StashDirty = %v, %v; want true, nil", stashed, err)
	}
	// The coder's edit is stashed away, leaving the tree clean (excluding logs).
	if clean, err := c.GitClean(t.Context(), "logs"); err != nil || !clean {
		t.Fatalf("tree not clean after stash: clean=%v err=%v", clean, err)
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "main.go")); strings.Contains(string(b), "edited") {
		t.Errorf("tracked edit outside logs not stashed away: %q", b)
	}
	// The excluded tracked file keeps the user's change, not reverted to HEAD.
	if b, _ := os.ReadFile(filepath.Join(repo, "logs", "run.log")); string(b) != "user edit\n" {
		t.Errorf("tracked file under excluded logs was not preserved: %q", b)
	}
	// The untracked-ignored run output is untouched too.
	if b, _ := os.ReadFile(filepath.Join(repo, "logs", "output.raw")); string(b) != "run output\n" {
		t.Errorf("untracked-ignored file under excluded logs was disturbed: %q", b)
	}
	if list := git(t, repo, "stash", "list"); !strings.Contains(list, "recover round 1") {
		t.Errorf("stash entry missing: %q", list)
	}
}

// StashDirty must leave EVERY kind of tracked change under an ignore-matched
// exclude exactly as it was -- not just an ordinary modification. It reproduces
// the pre-stash worktree AND index state for unstaged modifications, staged
// modifications, deletions, and renames, while still stashing the coder's edit.
func TestStashDirtyPreservesVariedTrackedChangesUnderIgnoredLogs(t *testing.T) {
	repo := gitRepo(t)
	for _, f := range []string{"keep.txt", "del.txt", "stg.txt", "rold.txt"} {
		writeFile(t, repo, "logs/"+f, "orig "+f+"\n")
	}
	git(t, repo, "add", "-f", "logs/keep.txt", "logs/del.txt", "logs/stg.txt", "logs/rold.txt")
	writeFile(t, repo, ".gitignore", "logs/\n")
	git(t, repo, "add", ".gitignore")
	git(t, repo, "commit", "-q", "-m", "track logs then ignore the dir")

	// A coder edit to stash away, plus one of every excluded change kind.
	writeFile(t, repo, "main.go", "package main // edited\n")
	writeFile(t, repo, "logs/keep.txt", "unstaged mod\n") // ' M' unstaged modification
	if err := os.Remove(filepath.Join(repo, "logs", "del.txt")); err != nil {
		t.Fatal(err)
	} // ' D' unstaged deletion
	writeFile(t, repo, "logs/stg.txt", "staged mod\n")
	git(t, repo, "add", "-f", "logs/stg.txt")            // 'M ' staged modification
	git(t, repo, "mv", "logs/rold.txt", "logs/rnew.txt") // 'R ' staged rename

	c := New(config.Target{Path: repo})
	stashed, err := c.StashDirty(t.Context(), "recover round 1", "logs")
	if err != nil || !stashed {
		t.Fatalf("StashDirty = %v, %v; want true, nil", stashed, err)
	}
	if clean, err := c.GitClean(t.Context(), "logs"); err != nil || !clean {
		t.Fatalf("tree not clean after stash: clean=%v err=%v", clean, err)
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "main.go")); strings.Contains(string(b), "edited") {
		t.Errorf("coder edit outside logs not stashed away: %q", b)
	}

	// Worktree: surviving paths keep their content; deleted and renamed-away
	// files stay gone rather than being resurrected to their HEAD content.
	if b, _ := os.ReadFile(filepath.Join(repo, "logs", "keep.txt")); string(b) != "unstaged mod\n" {
		t.Errorf("unstaged modification not preserved: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "logs", "stg.txt")); string(b) != "staged mod\n" {
		t.Errorf("staged modification not preserved: %q", b)
	}
	if _, err := os.Stat(filepath.Join(repo, "logs", "del.txt")); !os.IsNotExist(err) {
		t.Errorf("deleted file was resurrected: err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "logs", "rold.txt")); !os.IsNotExist(err) {
		t.Errorf("renamed-away file was resurrected: err=%v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "logs", "rnew.txt")); string(b) != "orig rold.txt\n" {
		t.Errorf("rename destination missing/wrong: %q", b)
	}

	// Index: the porcelain XY codes must match the pre-stash state exactly, so a
	// staged change stays staged and an unstaged one stays unstaged.
	status := git(t, repo, "status", "--porcelain", "--", "logs")
	for _, want := range []string{" M logs/keep.txt", " D logs/del.txt", "M  logs/stg.txt", "R  logs/rold.txt -> logs/rnew.txt"} {
		if !strings.Contains(status, want) {
			t.Errorf("status missing %q; got:\n%s", want, status)
		}
	}
}

// The most delicate restoreProtectedPath case: an excluded tracked file staged
// with one version and then modified again in the worktree (porcelain "MM").
// StashDirty must reproduce BOTH independent sides -- the index from the stash's
// index commit (stash@{0}^2) and the worktree from the worktree commit
// (stash@{0}) -- so the staged version and the later worktree version each
// survive and the path stays MM, rather than collapsing to a single version.
func TestStashDirtyPreservesStagedThenModifiedExcludedPath(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "logs/run.log", "orig\n")
	git(t, repo, "add", "-f", "logs/run.log") // force-track under the soon-ignored dir
	writeFile(t, repo, ".gitignore", "logs/\n")
	git(t, repo, "add", ".gitignore")
	git(t, repo, "commit", "-q", "-m", "track a log then ignore the dir")

	// A coder edit to stash away.
	writeFile(t, repo, "main.go", "package main // edited\n")
	// The excluded file: stage one version, then modify it again without staging.
	writeFile(t, repo, "logs/run.log", "staged version\n")
	git(t, repo, "add", "-f", "logs/run.log")
	writeFile(t, repo, "logs/run.log", "worktree version\n")

	// Precondition: the excluded path is genuinely MM (staged AND unstaged change).
	if pre := git(t, repo, "status", "--porcelain", "--", "logs"); !strings.Contains(pre, "MM logs/run.log") {
		t.Fatalf("precondition: want MM logs/run.log, got %q", pre)
	}

	col := New(config.Target{Path: repo})
	stashed, err := col.StashDirty(t.Context(), "recover round 1", "logs")
	if err != nil || !stashed {
		t.Fatalf("StashDirty = %v, %v; want true, nil", stashed, err)
	}
	if clean, err := col.GitClean(t.Context(), "logs"); err != nil || !clean {
		t.Fatalf("tree not clean after stash: clean=%v err=%v", clean, err)
	}
	// The coder's edit is stashed away.
	if b, _ := os.ReadFile(filepath.Join(repo, "main.go")); strings.Contains(string(b), "edited") {
		t.Errorf("coder edit outside logs not stashed away: %q", b)
	}
	// The worktree keeps the LATER version.
	if b, _ := os.ReadFile(filepath.Join(repo, "logs", "run.log")); string(b) != "worktree version\n" {
		t.Errorf("worktree version not preserved: %q", b)
	}
	// The index keeps the STAGED version (stage 0 of the index).
	if staged := git(t, repo, "show", ":logs/run.log"); staged != "staged version\n" {
		t.Errorf("staged index version not preserved: %q", staged)
	}
	// And the path is still MM: both sides independently restored.
	if status := git(t, repo, "status", "--porcelain", "--", "logs"); !strings.Contains(status, "MM logs/run.log") {
		t.Errorf("status missing MM logs/run.log after restore; got:\n%s", status)
	}
}

// A top-level `git stash` cannot capture a modified submodule working tree, so
// after stashing the tree can still be dirty. StashDirty must detect this in its
// final clean-verification and return the "uncommitted changes remain" error
// rather than (false, nil) or claiming a clean reconciliation.
func TestStashDirtyStillDirtyAfterStash(t *testing.T) {
	outer := gitRepo(t)
	inner := gitRepo(t) // a separate repo to embed as a submodule
	// -c protocol.file.allow=always is required for local-path submodules in
	// modern git; the submodule brings inner/main.go into outer/sub.
	git(t, outer, "-c", "protocol.file.allow=always", "submodule", "add", inner, "sub")
	git(t, outer, "commit", "-q", "-m", "add submodule")

	// Dirty the submodule's working tree. This shows as " M sub" in the
	// superproject but a top-level stash leaves it in place.
	writeFile(t, outer, "sub/main.go", "package main // changed\n")

	c := New(config.Target{Path: outer})
	stashed, err := c.StashDirty(t.Context(), "recover round 1", "logs")
	if err == nil {
		t.Fatalf("StashDirty = (%v, nil); want error about uncommitted changes remaining", stashed)
	}
	if !strings.Contains(err.Error(), "uncommitted changes remain") {
		t.Fatalf("StashDirty err = %v; want 'uncommitted changes remain'", err)
	}
}

// UnsafeConfig flags the repo-local git settings gitSafeConfig cannot neutralize
// (content filters, sshCommand, credential helpers) so an untrusted checkout is
// refused before any worktree-touching git command runs.
func TestUnsafeConfig(t *testing.T) {
	t.Run("clean repo has none", func(t *testing.T) {
		c := New(config.Target{Path: gitRepo(t)})
		keys, err := c.UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 0 {
			t.Fatalf("UnsafeConfig() = %v, want none", keys)
		}
	})

	t.Run("flags execution-capable settings", func(t *testing.T) {
		repo := gitRepo(t)
		// All three filter suffixes run a repo-controlled program (.clean on
		// stage-in, .smudge on checkout, .process for a long-running filter), so
		// each branch of the filter.<n>.{clean,smudge,process} OR is exercised
		// independently -- a regression dropping any one from the allowlist turns
		// `git diff`/`git add` on a hostile worktree into silent code execution.
		git(t, repo, "config", "filter.evil.clean", "sh -c 'id'")
		git(t, repo, "config", "filter.evil.smudge", "sh -c 'id'")
		git(t, repo, "config", "filter.evil.process", "sh -c 'id'")
		git(t, repo, "config", "core.sshCommand", "sh -c 'id'")
		git(t, repo, "config", "credential.helper", "!sh -c 'id'")
		// core.askPass is the credential prompt git reaches for after GIT_ASKPASS
		// and before SSH_ASKPASS, i.e. the one that fires in the non-interactive
		// context Prepare's `git fetch <remote> <baseOid>` runs in on the pr path.
		git(t, repo, "config", "core.askPass", "./payload")
		// A signing program runs when a fix round's `git commit` signs, so a
		// crafted .git/config pointing gpg.program (or a format-specific signing
		// program) at a payload is a commit-time code-execution path too.
		git(t, repo, "config", "gpg.program", "./payload")
		git(t, repo, "config", "gpg.ssh.program", "./payload")
		c := New(config.Target{Path: repo})
		keys, err := c.UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Join(keys, ",")
		for _, want := range []string{"filter.evil.clean", "filter.evil.smudge", "filter.evil.process", "core.sshcommand", "core.askpass", "credential.helper", "gpg.program", "gpg.ssh.program"} {
			if !strings.Contains(got, want) {
				t.Errorf("UnsafeConfig() = %v, want it to include %q", keys, want)
			}
		}
	})

	// The transport settings are the same class on the path the guard is actually
	// for: pr mode fetches (gh pr checkout, and Prepare's base-object fetch). A
	// url.<base>.insteadOf rewrite leaves remote.origin.url an ordinary GitHub URL
	// -- so nothing else looks wrong -- while routing every fetch through a helper
	// protocol whose URL git executes.
	t.Run("flags transport settings that execute programs", func(t *testing.T) {
		repo := gitRepo(t)
		git(t, repo, "config", "url.ext::sh -c id.insteadOf", "https://github.com/")
		git(t, repo, "config", "url.ext::sh -c id.pushInsteadOf", "https://github.com/")
		git(t, repo, "config", "core.gitProxy", "./payload")
		git(t, repo, "config", "remote.origin.uploadPack", "./payload")
		git(t, repo, "config", "remote.origin.receivePack", "./payload")
		git(t, repo, "config", "remote.origin.proxy", "./payload")
		keys, err := New(config.Target{Path: repo}).UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Join(keys, ",")
		for _, want := range []string{
			"url.ext::sh -c id.insteadof", "url.ext::sh -c id.pushinsteadof",
			"core.gitproxy",
			"remote.origin.uploadpack", "remote.origin.receivepack", "remote.origin.proxy",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("UnsafeConfig() = %v, want it to include %q", keys, want)
			}
		}
	})

	// An `[include] path` in .git/config is expanded by every real git command but
	// NOT by `git config --local --list`: --local names a specific file, and
	// git-config defaults --includes to off in that case. A .git/config whose only
	// content is an include would otherwise look completely clean while
	// diff/add/status/checkout run the filter it pulls in.
	t.Run("flags settings reached through an include", func(t *testing.T) {
		repo := gitRepo(t)
		writeFile(t, repo, ".git/included", "[filter \"evil\"]\n\tclean = sh -c 'id'\n")
		f, err := os.OpenFile(filepath.Join(repo, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString("[include]\n\tpath = included\n"); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		keys, err := New(config.Target{Path: repo}).UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(keys, "filter.evil.clean") {
			t.Errorf("UnsafeConfig() = %v, want it to include filter.evil.clean from the included file", keys)
		}
	})

	// .git/config.worktree applies to every git command in the worktree once
	// extensions.worktreeConfig is set, but it lives in the `worktree` scope, which
	// --local does not read -- a second place inside .git to hide a filter.
	t.Run("flags worktree-scoped settings", func(t *testing.T) {
		repo := gitRepo(t)
		git(t, repo, "config", "extensions.worktreeConfig", "true")
		git(t, repo, "config", "--worktree", "filter.evil.clean", "sh -c 'id'")
		keys, err := New(config.Target{Path: repo}).UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(keys, "filter.evil.clean") {
			t.Errorf("UnsafeConfig() = %v, want it to include filter.evil.clean from .git/config.worktree", keys)
		}
	})

	// Reading every scope to catch the two above must not start refusing targets
	// over the OPERATOR's own configuration: a global credential.helper is theirs,
	// not the repository's, and flagging it would refuse every target on the host.
	t.Run("ignores settings from the operator's global config", func(t *testing.T) {
		repo := gitRepo(t)
		global := filepath.Join(t.TempDir(), "gitconfig")
		if err := os.WriteFile(global, []byte("[credential]\n\thelper = store\n[core]\n\tsshCommand = ssh -i ~/.ssh/id_ed25519\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		// The collector runs git with fixpoint's own environment, so pointing
		// GIT_CONFIG_GLOBAL at the file is what puts it in the `global` scope.
		t.Setenv("GIT_CONFIG_GLOBAL", global)
		keys, err := New(config.Target{Path: repo}).UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 0 {
			t.Fatalf("UnsafeConfig() = %v, want none: global-scope settings are the operator's, not the target's", keys)
		}
	})
}

// The diff embeds full file CONTENT into every reviewer prompt and into the
// on-disk artifacts, so the mandatory credential excludes matter more here than
// in directory mode -- which only ever emits a list of paths. A PR that adds a
// .env or a deploy key must not hand the key material to the reviewers.
func TestCollectGitDiffAppliesExcludes(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD", Exclude: []string{"**/vendor/**"}})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	// Committed inside the diff range: content would otherwise be rendered in full.
	writeFile(t, repo, ".env", "DB_PASS=hunter2-committed\n")
	writeFile(t, repo, "deploy/id_rsa.pem", "-----BEGIN PRIVATE KEY-----\ncommitted-key\n")
	writeFile(t, repo, "vendor/dep/c.go", "package dep // configured-exclude\n")
	writeFile(t, repo, "main.go", "package main // reviewed\n")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-qm", "secrets")
	// And as untracked files, which are listed by path rather than diffed.
	writeFile(t, repo, "sub/.env.local", "TOKEN=hunter2-untracked\n")
	writeFile(t, repo, "sub/ok.txt", "fine\n")

	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{
		"hunter2-committed", ".env",
		"committed-key", "id_rsa.pem",
		"hunter2-untracked", ".env.local",
		"configured-exclude", "vendor/dep/c.go",
	} {
		if strings.Contains(material, leak) {
			t.Errorf("Collect() leaked %q into the review material:\n%s", leak, material)
		}
	}
	// ...while everything else is still collected: the exclusion must not silently
	// narrow the review to nothing.
	for _, want := range []string{"// reviewed", "sub/ok.txt"} {
		if !strings.Contains(material, want) {
			t.Errorf("Collect() missing %q:\n%s", want, material)
		}
	}
}

// ghRemote must select the remote gh treats as the base without assuming
// "origin": match the base repo's nameWithOwner against remote URLs, then fall
// back to origin, then the first remote, and error when there are none.
func TestGhRemote(t *testing.T) {
	installGh := func(t *testing.T, body string) {
		t.Helper()
		binDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte("#!/bin/sh\n"+body), 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	t.Run("no remotes errors", func(t *testing.T) {
		installGh(t, "exit 1\n")
		c := New(config.Target{Path: gitRepo(t)})
		if _, err := c.ghRemote(t.Context()); err == nil || !strings.Contains(err.Error(), "no git remotes") {
			t.Fatalf("ghRemote() = %v, want no-remotes error", err)
		}
	})

	t.Run("nameWithOwner match beats origin fallback", func(t *testing.T) {
		installGh(t, "echo acme/widget\n") // gh repo view --json nameWithOwner
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "origin", "https://github.com/acme/other.git")
		git(t, repo, "remote", "add", "upstream", "https://github.com/acme/widget.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "upstream" {
			t.Fatalf("ghRemote() = %q, %v; want upstream (URL match)", r, err)
		}
	})

	t.Run("origin fallback when gh gives nothing", func(t *testing.T) {
		installGh(t, "exit 1\n")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "upstream", "https://example.com/u.git")
		git(t, repo, "remote", "add", "origin", "https://example.com/o.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "origin" {
			t.Fatalf("ghRemote() = %q, %v; want origin fallback", r, err)
		}
	})

	t.Run("first remote when no origin", func(t *testing.T) {
		installGh(t, "exit 1\n")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "upstream", "https://example.com/u.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "upstream" {
			t.Fatalf("ghRemote() = %q, %v; want first-remote fallback", r, err)
		}
	})
}

func TestPreparePREmptyBaseOid(t *testing.T) {
	repo := gitRepo(t)
	binDir := t.TempDir()
	stub := "#!/bin/sh\ncase \"$1 $2\" in\n\"pr checkout\") : ;;\n\"pr view\") echo '' ;;\n*) exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	c := New(config.Target{Mode: "pr", Path: repo, PR: 7})
	if err := c.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "empty base commit oid") {
		t.Fatalf("Prepare() = %v, want empty-oid error", err)
	}
}

func TestPreparePRNoRemotesForFetch(t *testing.T) {
	repo := gitRepo(t) // no remotes
	binDir := t.TempDir()
	// pr view returns a well-formed but unreachable oid, so cat-file fails and
	// Prepare must fetch -- but there are no remotes to fetch from.
	stub := "#!/bin/sh\ncase \"$1 $2\" in\n\"pr checkout\") : ;;\n" +
		"\"pr view\") echo 0000000000000000000000000000000000000000 ;;\n*) exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	c := New(config.Target{Mode: "pr", Path: repo, PR: 7})
	if err := c.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "no git remotes") {
		t.Fatalf("Prepare() = %v, want no-remotes fetch error", err)
	}
}

// When the PR base tip is not reachable locally, Prepare must fetch it from the
// gh remote before computing the merge base.
func TestPreparePRFetchesUnreachableBase(t *testing.T) {
	// upstream holds a base commit the clone has not fetched.
	upstream := gitRepo(t)
	base := t.TempDir()
	git(t, base, "clone", "-q", upstream, "work")
	repo := filepath.Join(base, "work")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "test")
	git(t, repo, "config", "commit.gpgsign", "false")
	initial := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))

	// Advance upstream: this new commit is the PR base, unreachable in the clone.
	writeFile(t, upstream, "base.go", "package main\n")
	git(t, upstream, "add", ".")
	git(t, upstream, "commit", "-q", "-m", "base advance")
	baseOid := strings.TrimSpace(git(t, upstream, "rev-parse", "HEAD"))

	// The clone's feature branch, off the shared initial commit.
	git(t, repo, "checkout", "-q", "-b", "feature")
	writeFile(t, repo, "main.go", "package main\n\nfunc pr() {}\n")
	git(t, repo, "commit", "-aqm", "pr change")
	git(t, repo, "checkout", "-q", "-")

	if _, err := New(config.Target{Path: repo}).git(t.Context(), "cat-file", "-e", baseOid+"^{commit}"); err == nil {
		t.Fatal("precondition failed: base oid should NOT be reachable before fetch")
	}

	binDir := t.TempDir()
	stub := "#!/bin/sh\ncase \"$1 $2\" in\n\"pr checkout\") git checkout -q feature ;;\n" +
		"\"pr view\") echo " + baseOid + " ;;\n*) exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	c := New(config.Target{Mode: "pr", Path: repo, PR: 7})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatalf("Prepare() = %v, want the fetch path to succeed", err)
	}
	// Merge base of the feature head with the fetched base is the shared commit.
	if c.baseSHA != initial {
		t.Errorf("baseSHA = %q, want the shared merge base %q", c.baseSHA, initial)
	}
	// The fetch made the base object locally reachable.
	if _, err := c.git(t.Context(), "cat-file", "-e", baseOid+"^{commit}"); err != nil {
		t.Errorf("base oid still unreachable; fetch did not run: %v", err)
	}
}

// gh pr view failing (after a successful checkout) must abort Prepare with an
// error naming the view step, not silently proceed with an empty base.
func TestPreparePRViewFails(t *testing.T) {
	repo := gitRepo(t)
	binDir := t.TempDir()
	stub := "#!/bin/sh\ncase \"$1 $2\" in\n" +
		"\"pr checkout\") : ;;\n" +
		"\"pr view\") echo 'view boom' >&2; exit 1 ;;\n" +
		"*) exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	c := New(config.Target{Mode: "pr", Path: repo, PR: 7})
	if err := c.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "gh pr view") {
		t.Fatalf("Prepare() = %v, want gh pr view error", err)
	}
}

// When the PR base tip is unreachable locally, Prepare fetches it from the gh
// remote; a failing fetch must abort with an error naming the fetch step.
func TestPreparePRFetchFails(t *testing.T) {
	repo := gitRepo(t)
	upstream := gitRepo(t) // a real remote that simply lacks the requested oid
	git(t, repo, "remote", "add", "origin", upstream)
	binDir := t.TempDir()
	// pr view returns a well-formed but unreachable oid, so cat-file fails and
	// Prepare fetches it from origin -- which does not have it, so fetch fails.
	// The `*) exit 1` arm also fails gh's repo-view remote lookup, so ghRemote
	// falls back to the origin remote configured above.
	stub := "#!/bin/sh\ncase \"$1 $2\" in\n" +
		"\"pr checkout\") : ;;\n" +
		"\"pr view\") echo 1111111111111111111111111111111111111111 ;;\n" +
		"*) exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	c := New(config.Target{Mode: "pr", Path: repo, PR: 7})
	if err := c.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "fetch PR base") {
		t.Fatalf("Prepare() = %v, want fetch-PR-base error", err)
	}
}

// merge-base failing (the pinned base shares no history with HEAD) must abort
// Prepare with an error naming the merge-base step.
func TestPreparePRMergeBaseFails(t *testing.T) {
	repo := gitRepo(t)
	orig := strings.TrimSpace(git(t, repo, "rev-parse", "--abbrev-ref", "HEAD"))
	// Build an unrelated orphan commit that IS reachable locally (so no fetch is
	// attempted) but shares no ancestry with HEAD, so merge-base exits non-zero.
	git(t, repo, "checkout", "-q", "--orphan", "unrelated")
	writeFile(t, repo, "unrelated.txt", "x\n")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-q", "-m", "orphan root")
	orphan := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "checkout", "-q", "-f", orig)

	binDir := t.TempDir()
	stub := "#!/bin/sh\ncase \"$1 $2\" in\n" +
		"\"pr checkout\") : ;;\n" +
		"\"pr view\") echo " + orphan + " ;;\n" +
		"*) exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	c := New(config.Target{Mode: "pr", Path: repo, PR: 7})
	if err := c.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "merge-base with base") {
		t.Fatalf("Prepare() = %v, want merge-base error", err)
	}
}

func TestTruncate(t *testing.T) {
	small := strings.Repeat("a", 100)
	if truncate(small) != small {
		t.Error("truncate changed a small string")
	}
	big := strings.Repeat("a", maxMaterial+1000)
	got := truncate(big)
	if len(got) >= len(big) || !strings.HasPrefix(got, big[:maxMaterial]) || !strings.Contains(got, "material truncated") {
		t.Errorf("truncate(big): len=%d, marker present=%v", len(got), strings.Contains(got, "truncated"))
	}
	// The marker's size text must be derived from the cap, not hardcoded: assert
	// it embeds HumanSize(maxMaterial) so the material marker cannot silently drift
	// from the constant it bounds (r4.6/r5.6).
	if sz := agent.HumanSize(maxMaterial); !strings.Contains(got, sz) {
		t.Errorf("truncate(big) marker missing cap size %q:\n%s", sz, got[maxMaterial:])
	}

	// The cutoff must not split a multibyte character: "é" (2 bytes) straddles
	// the limit, so the cut backs off to the rune boundary before it.
	straddling := strings.Repeat("a", maxMaterial-1) + "é" + strings.Repeat("b", 100)
	got = truncate(straddling)
	if !utf8.ValidString(got) {
		t.Error("truncate produced invalid UTF-8")
	}
	if !strings.HasPrefix(got, strings.Repeat("a", maxMaterial-1)+"\n") {
		t.Error("truncate did not back off to the rune boundary")
	}
}

// In a git repository, directory-mode scope comes from git, so .gitignore decides
// what counts as source -- the same rule git-diff/pr mode already applies to
// untracked files. Without this, target.exclude has to re-derive every language's
// build directories and silently misses whatever it forgets.
func TestCollectDirectoryHonorsGitignore(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, ".gitignore", "coverage.out\nbuild/\nsecret.env\n")
	writeFile(t, repo, "pkg/a.go", "package pkg\n")
	writeFile(t, repo, "Makefile", "all:\n")
	writeFile(t, repo, "coverage.out", "mode: set\n")
	writeFile(t, repo, "build/artifact.js", "compiled\n")
	writeFile(t, repo, "secret.env", "TOKEN=xyz\n")
	c := New(config.Target{Mode: "directory", Path: repo})
	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// Tracked (main.go) and untracked-but-not-ignored files are in scope, and no
	// exclude glob was configured -- .gitignore alone removed the rest.
	for _, want := range []string{"main.go", "pkg/a.go", "Makefile", ".gitignore"} {
		if !strings.Contains(material, want) {
			t.Errorf("Collect() missing %q:\n%s", want, material)
		}
	}
	for _, notWant := range []string{"coverage.out", "build/artifact.js", "secret.env"} {
		if strings.Contains(material, notWant) {
			t.Errorf("Collect() listed gitignored %q:\n%s", notWant, material)
		}
	}
	if !strings.Contains(material, "Files in scope (4):") {
		t.Errorf("Collect() header wrong:\n%s", material)
	}
}

// credentialFiles are one path per mandatory-exclude pattern shape, at the root and
// nested, plus the ordinary source file that must survive the filtering. Directory
// mode renders this listing into every reviewer prompt, and reviewers run
// unsandboxed and read whatever path they are pointed at, so a credential path
// appearing here is the realistic leak: a reviewer steered by injected content in
// the same tree quotes the key into a finding, which is persisted and echoed into
// the fix commit body.
var credentialFiles = []string{
	".env", ".env.local", "svc/.env.production",
	"key.pem", "certs/server.pem", "certs/bundle.p12", "certs/bundle.pfx",
	"server.key", "certs/tls.key",
	"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", "home/.ssh/id_ed25519",
	".npmrc", ".netrc", ".pgpass", "home/.netrc",
	"credentials", "home/.aws/credentials",
	"secrets.kdbx", "vault/secrets.kdbx",
}

// writeCredentialTree writes every credential shape plus one ordinary source file
// into dir.
func writeCredentialTree(t *testing.T, dir string) {
	t.Helper()
	for _, name := range credentialFiles {
		writeFile(t, dir, name, "SECRET=leaked\n")
	}
	writeFile(t, dir, "pkg/a.go", "package pkg\n")
}

// assertNoCredentials checks the collected material lists the source file and none
// of the credential paths.
func assertNoCredentials(t *testing.T, material string) {
	t.Helper()
	for _, name := range credentialFiles {
		if strings.Contains(material, name) {
			t.Errorf("collected material names credential file %q:\n%s", name, material)
		}
	}
	if !strings.Contains(material, "pkg/a.go") {
		t.Errorf("collected material dropped an ordinary source file:\n%s", material)
	}
}

// The mandatory credential excludes are a safety property, which is why they live
// in code rather than config ("a safety property must not be something a config can
// forget"). This is the test that keeps them working: an EMPTY target.exclude, no
// .gitignore, and every shape of credential path in the tree. It covers BOTH
// collection paths, because they filter at different call sites -- listGitFiles via
// skipFile, walkFiles via matchAny -- and a change to either one, or to compileGlobs'
// "**/" expansion, would otherwise leave the suite green while every directory-mode
// prompt started naming the operator's key files.
func TestCollectDirectoryAlwaysExcludesCredentialFiles(t *testing.T) {
	t.Run("git", func(t *testing.T) {
		repo := gitRepo(t)
		writeCredentialTree(t, repo)
		// TRACKED, which is the harder case: --cached lists index entries no ignore
		// rule can remove, so only the mandatory excludes stand between a tracked
		// id_rsa fixture and the reviewer prompt.
		git(t, repo, "add", "-A")
		material, err := New(config.Target{Mode: "directory", Path: repo, Exclude: nil}).Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assertNoCredentials(t, material)
	})
	t.Run("walk", func(t *testing.T) {
		dir := t.TempDir()
		writeCredentialTree(t, dir)
		material, err := New(config.Target{Mode: "directory", Path: dir, Exclude: nil}).Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assertNoCredentials(t, material)
	})
}

// target.exclude still applies on top of .gitignore, for committed material that
// is not worth reviewing (vendored deps, fixtures).
func TestCollectDirectoryGitExcludeGlobsStillApply(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "vendor/dep/b.go", "package dep\n")
	writeFile(t, repo, "pkg/a.go", "package pkg\n")
	c := New(config.Target{Mode: "directory", Path: repo, Exclude: []string{"**/vendor/**"}})
	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(material, "vendor") {
		t.Errorf("exclude glob not applied to the git listing:\n%s", material)
	}
	if !strings.Contains(material, "pkg/a.go") {
		t.Errorf("Collect() missing pkg/a.go:\n%s", material)
	}
}

// A target.path inside the repository must yield paths relative to that
// subdirectory, matching what the filesystem walk produces. git ls-files reports
// relative to the process working directory, so this pins that contract.
func TestCollectDirectoryGitSubdirectoryTarget(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "sub/deep/x.go", "package deep\n")
	writeFile(t, repo, "sub/y.go", "package sub\n")
	writeFile(t, repo, "outside.go", "package main\n")
	c := New(config.Target{Mode: "directory", Path: filepath.Join(repo, "sub")})
	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"deep/x.go", "y.go"} {
		if !strings.Contains(material, want) {
			t.Errorf("Collect() missing subdir-relative %q:\n%s", want, material)
		}
	}
	if strings.Contains(material, "sub/") || strings.Contains(material, "outside.go") {
		t.Errorf("Collect() leaked repo-root paths:\n%s", material)
	}
}

// git ls-files --cached reports index entries, which outlive a file deleted from
// the worktree. Listing a path no agent can open would be worse than omitting it.
func TestCollectDirectoryGitSkipsDeletedWorktreeFile(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "gone.go", "package main\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "add gone.go")
	if err := os.Remove(filepath.Join(repo, "gone.go")); err != nil {
		t.Fatal(err)
	}
	c := New(config.Target{Mode: "directory", Path: repo})
	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(material, "gone.go") {
		t.Errorf("Collect() listed a deleted worktree file:\n%s", material)
	}
	if !strings.Contains(material, "Files in scope (1):") {
		t.Errorf("count should exclude the deleted file:\n%s", material)
	}
}

// shimGit shadows `git` on PATH with a wrapper that runs body (shell source, which
// must exit) whenever sub appears anywhere in the argument list, and forwards every
// other invocation to the real git. The subcommand is matched anywhere because the
// collector prefixes every git call with -c hardening flags. `which git` would
// resolve the shim once PATH is shadowed, so the real binary is pinned up front.
func shimGit(t *testing.T, sub, body string) {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"for a in \"$@\"; do\n" +
		"  if [ \"$a\" = \"" + sub + "\" ]; then\n" + body + "\n  fi\n" +
		"done\n" +
		"exec \"" + realGit + "\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// Directory mode streams its listing through gitScanNUL, not c.git, so it has its
// own error handling -- and a failure there must surface rather than be reported as
// an empty (or partial) scope, which is exactly the silently-narrowed listing the
// denylist-only design exists to avoid. The wrapped message names the operation and
// carries git's stderr, since that is the only clue to what went wrong.
func TestCollectDirectoryListingErrorSurfaces(t *testing.T) {
	repo := gitRepo(t)
	shimGit(t, "ls-files", "    echo 'boom' >&2\n    exit 1")

	_, err := New(config.Target{Mode: "directory", Path: repo}).Collect(t.Context())
	if err == nil {
		t.Fatal("Collect() = nil, want the failed git listing to surface")
	}
	for _, want := range []string{"list files from git", "boom"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Collect() err = %v, want it to contain %q", err, want)
		}
	}
}

// An entry longer than maxGitPath is an error, never a truncated path: the caller
// counts what fn is handed, so accepting a fragment would report a file that does
// not exist and hide the rest of the listing behind a bogus total. Only the scanner
// can detect this -- git exits 0 here -- so the scan error must win over the exit
// status.
func TestCollectDirectoryOversizedEntryFails(t *testing.T) {
	repo := gitRepo(t)
	// 2 MB of NUL-free output: one entry, twice maxGitPath, so the scanner stops.
	shimGit(t, "ls-files", "    i=0\n    while [ $i -lt 2048 ]; do printf '%1024s' ''; i=$((i+1)); done\n    exit 0")

	_, err := New(config.Target{Mode: "directory", Path: repo}).Collect(t.Context())
	if err == nil {
		t.Fatal("Collect() = nil, want an oversized listing entry to fail the collection")
	}
	for _, want := range []string{"list files from git", "reading output"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Collect() err = %v, want it to contain %q", err, want)
		}
	}
}

// gitScanNUL blocks on the listing until git's stdout reaches EOF, so a canceled
// run must be terminated by the Cancel hook rather than left to finish:
// listGitFiles is called with the run's context and Ctrl-C has to reach it. Asserted on listGitFiles directly, because
// listFiles' work-tree probe is itself a git command and fails first under a
// canceled context, so it would never reach the listing.
func TestListGitFilesCanceledContextReturnsPromptly(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "directory", Path: repo})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	done := make(chan error, 1)
	go func() {
		_, _, err := c.listGitFiles(ctx, nil)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("listGitFiles() = nil on a canceled context, want an error")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("listGitFiles did not return on a canceled context; the scan was not interrupted")
	}
}

// A git that exits SUCCESSFULLY after backgrounding a child leaves that child
// running: cmd.Cancel only fires on cancellation. The child is then free to mutate
// the repository while the clean-tree check, verification, or the round commit
// runs. gitScanNUL kills the whole process group on every exit path, like
// agent.Run and verify.runOne.
func TestCollectDirectoryKillsBackgroundedChildOnSuccess(t *testing.T) {
	repo := gitRepo(t)
	sentinel := filepath.Join(t.TempDir(), "child-survived")
	// The child's pipes go to /dev/null so it does not hold the listing pipe open:
	// this is the case where the scan ends cleanly and nothing else would reap it.
	shimGit(t, "ls-files", "    ( sleep 1; touch '"+sentinel+"' ) >/dev/null 2>&1 &\n"+
		"    printf 'main.go\\0'\n    exit 0")

	material, err := New(config.Target{Mode: "directory", Path: repo}).Collect(t.Context())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if !strings.Contains(material, "main.go") {
		t.Errorf("the listing should still be produced:\n%s", material)
	}
	// Well past the child's own delay: if it were still alive it would have run.
	time.Sleep(2 * time.Second)
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("a backgrounded child survived a successful listing and mutated the directory afterwards")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// The scan reads stdout to EOF, and a child that inherited the write end holds it
// open after git itself exits. Nothing may make that wait unbounded: the whole
// collection would otherwise stall until gitOpTimeout (ten minutes) even though
// the listing is complete and git is gone.
func TestCollectDirectoryChildHoldingStdoutDoesNotStallScan(t *testing.T) {
	repo := gitRepo(t)
	// The child inherits stdout and outlives the leader by a minute. Only its stderr
	// is redirected: that stream is a copy-goroutine pipe, already bounded by
	// WaitDelay, and holding it too would just report the collection as a WaitDelay
	// expiry instead of exercising the stdout stall this test is about.
	shimGit(t, "ls-files", "    printf 'main.go\\0'\n    sleep 60 2>/dev/null &\n    exit 0")

	type result struct {
		material string
		err      error
	}
	done := make(chan result, 1)
	go func() {
		material, err := New(config.Target{Mode: "directory", Path: repo}).Collect(t.Context())
		done <- result{material, err}
	}()
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Collect: %v", got.err)
		}
		if !strings.Contains(got.material, "main.go") {
			t.Errorf("the listing should still be produced:\n%s", got.material)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Collect hung on a descendant holding stdout after git exited; the scan is bounded only by gitOpTimeout")
	}
}

// gitScanNUL hand-copies c.git's hardening (the gitSafeConfig -c overrides and the
// hardened environment) because it runs git itself. core.fsmonitor names a program
// git spawns while listing files, and a target's own .git/config can set it -- so a
// directory-mode collect against a crafted checkout would otherwise be code
// execution with fixpoint's inherited environment, in a mode that never reaches the
// fix-round trust gate. This is the regression guard for dropping either override
// from the streaming path.
func TestCollectDirectoryDoesNotRunFsmonitor(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to run the fsmonitor program")
	}
	repo := gitRepo(t)
	sentinel := filepath.Join(repo, "fsmonitor-ran")
	evil := filepath.Join(repo, "evil-fsmonitor.sh")
	// A v1 fsmonitor program prints the paths it considers dirty; the sentinel is
	// the side effect core.fsmonitor=false must prevent.
	if err := os.WriteFile(evil, []byte("#!/bin/sh\ntouch '"+sentinel+"'\nprintf '/\\0'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "config", "core.fsmonitor", evil)

	material, err := New(config.Target{Mode: "directory", Path: repo}).Collect(t.Context())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if !strings.Contains(material, "main.go") {
		t.Errorf("the listing should still be produced:\n%s", material)
	}
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("the repo-configured fsmonitor program ran; the streaming listing lost gitSafeConfig/gitHardenedEnv")
	}
}

// A non-git target has no .gitignore to consult, so scope falls back to the
// filesystem walk and only target.exclude narrows it.
func TestCollectDirectoryNonGitFallsBackToWalk(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "package a\n")
	writeFile(t, dir, "coverage.out", "mode: set\n")
	writeFile(t, dir, "skipme/b.go", "package b\n")
	c := New(config.Target{Mode: "directory", Path: dir, Exclude: []string{"skipme/**"}})
	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// coverage.out survives here: no git means no .gitignore to remove it.
	for _, want := range []string{"a.go", "coverage.out"} {
		if !strings.Contains(material, want) {
			t.Errorf("walk fallback missing %q:\n%s", want, material)
		}
	}
	if strings.Contains(material, "skipme") {
		t.Errorf("walk fallback ignored the exclude glob:\n%s", material)
	}
}
