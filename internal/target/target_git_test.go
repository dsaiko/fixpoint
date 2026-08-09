package target

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
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

// The reverse mistake: an EXPECTED probe result must not be read as a failure
// just because the invoking shell has a locale set. git translates its fatal
// messages when built with NLS, and notARepository matches the English text, so
// without a pinned locale an ordinary non-repository directory would abort
// preflight on any translated git. The shim stands in for that git -- English
// only under LC_ALL=C -- so the test does not depend on which locales the host
// has installed.
func TestIsGitRepoUnderATranslatedGit(t *testing.T) {
	t.Setenv("LC_ALL", "de_DE.UTF-8")
	t.Setenv("LANG", "de_DE.UTF-8")
	t.Setenv("LANGUAGE", "de")
	shimGit(t, "rev-parse", `    if [ "$LC_ALL" = C ]; then
      echo 'fatal: not a git repository (or any of the parent directories): .git' >&2
    else
      echo 'fatal: Kein Git-Repository (oder irgendein Elternverzeichnis): .git' >&2
    fi
    exit 128`)

	if ok, err := New(config.Target{Path: t.TempDir()}).IsGitRepo(t.Context()); err != nil || ok {
		t.Errorf("IsGitRepo() = %v, %v under a translated git, want false with no error", ok, err)
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

// A trailing "..." pins the MERGE BASE of the ref and HEAD, which is the only
// correct base for "what did this branch change" once the base branch has moved.
// Against the branch TIP the commits this branch does not have appear REVERSED --
// the panel would review someone else's work as deletions this branch made, which
// with two machines pushing to one repo happens within hours.
func TestPrepareGitDiffMergeBaseIgnoresCommitsOnlyOnTheBase(t *testing.T) {
	repo := gitRepo(t)
	base := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	// Whatever `git init` named it: init.defaultBranch is user config, so master
	// and main are both possible and neither may be hard-coded here.
	trunk := strings.TrimSpace(git(t, repo, "rev-parse", "--abbrev-ref", "HEAD"))
	git(t, repo, "checkout", "-qb", "feature")
	writeFile(t, repo, "mine.go", "package main\n\nfunc mine() {}\n")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-qm", "mine")
	// The base branch gains a commit this branch does not have.
	git(t, repo, "checkout", "-q", trunk)
	writeFile(t, repo, "theirs.go", "package main\n\nfunc theirs() {}\n")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-qm", "theirs")
	git(t, repo, "checkout", "-q", "feature")

	c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: trunk + "..."})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if c.baseSHA != base {
		t.Fatalf("pinned base = %s, want the merge base %s", c.baseSHA, base)
	}
	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(material, "func mine()") {
		t.Errorf("the branch's own change is missing from the diff:\n%s", material)
	}
	if strings.Contains(material, "theirs") {
		t.Errorf("a commit only on the base branch reached the diff; it would be reviewed backwards:\n%s", material)
	}

	// Without the dots the same ref pins the TIP, and the base-only commit shows up
	// as a deletion -- the behavior the suffix exists to avoid.
	tip := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: trunk})
	if err := tip.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if tip.baseSHA == c.baseSHA {
		t.Fatal("tip and merge base resolved to the same commit; the fixture did not diverge")
	}
	tipMaterial, err := tip.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tipMaterial, "theirs") {
		t.Errorf("tip semantics should show the base-only commit reversed, so this test is not pinning what it claims:\n%s", tipMaterial)
	}
}

func TestPrepareGitDiffMergeBaseErrors(t *testing.T) {
	// Nothing before the dots is operator error, not a ref lookup.
	c := New(config.Target{Mode: "git-diff", Path: gitRepo(t), BaseRef: "..."})
	if err := c.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "names no ref") {
		t.Fatalf("Prepare() = %v, want a names-no-ref error", err)
	}
	// An unknown ref before the dots reports the merge-base failure, not a silent
	// fall back to reviewing the whole tree.
	c = New(config.Target{Mode: "git-diff", Path: gitRepo(t), BaseRef: "no-such-ref..."})
	if err := c.Prepare(t.Context()); err == nil || !strings.Contains(err.Error(), "merge base") {
		t.Fatalf("Prepare() = %v, want a merge-base error", err)
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
	// Shim `git`: forward everything to the real git except the UNTRACKED listing,
	// which fails. `which git` resolves the real binary once PATH is shadowed, so
	// pin it up front. The flags are matched anywhere in the argument list, since
	// the collector prefixes every git call with -c hardening flags. Only
	// `--others` without `--cached` is failed, so this hits the untracked listing
	// and not the symlink-destination scan, which enumerates both sets -- otherwise
	// the collection would fail earlier, for a different reason, and this test would
	// pass without ever reaching the branch it exists to cover.
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	shim := "#!/bin/sh\n" +
		`cached=; others=` + "\n" +
		`for a in "$@"; do` + "\n" +
		`  if [ "$a" = "--cached" ]; then cached=1; fi` + "\n" +
		`  if [ "$a" = "--others" ]; then others=1; fi` + "\n" +
		`done` + "\n" +
		`if [ -n "$others" ] && [ -z "$cached" ]; then echo "boom" >&2; exit 1; fi` + "\n" +
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

// The squash must keep the excluded paths out of its tree, exactly as the per-fix
// commits it replaces did. Commit deliberately restores the excluded paths' staged
// index entries after each commit, so by squash time the live index carries staged
// changes under the exclusion that no commit contained -- and a write-tree over that
// index would publish them (the run's own logs, a credential-bearing .env).
func TestSquashSinceExcludesStagedExcludedPaths(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "logs/run.log", "committed\n")
	git(t, repo, "add", "logs/run.log")
	git(t, repo, "commit", "-q", "-m", "a tracked log")
	base := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))

	// Staged under the exclusion, then modified further in the worktree: neither side
	// belongs in the squash, and both have to survive it.
	writeFile(t, repo, "logs/run.log", "staged\n")
	git(t, repo, "add", "logs/run.log")
	stagedBlob := strings.Fields(git(t, repo, "ls-files", "--stage", "--", "logs/run.log"))[1]
	writeFile(t, repo, "logs/run.log", "worktree\n")

	c := New(config.Target{Path: repo})
	for _, name := range []string{"one.go", "two.go"} {
		writeFile(t, repo, name, "package main\n")
		if _, err := c.Commit(t.Context(), "fix "+name, "body", "logs"); err != nil {
			t.Fatal(err)
		}
	}

	sha, err := c.SquashSince(t.Context(), base, "fixpoint: round 1", "Fixed:\n- one\n- two", "logs")
	if err != nil {
		t.Fatal(err)
	}
	files := git(t, repo, "show", "--name-only", "--format=", sha)
	if strings.Contains(files, "logs/run.log") {
		t.Errorf("the squashed commit carried an excluded path:\n%s", files)
	}
	if !strings.Contains(files, "one.go") || !strings.Contains(files, "two.go") {
		t.Errorf("squashed commit files = %q, want both fixes", files)
	}
	if got := git(t, repo, "show", sha+":logs/run.log"); got != "committed\n" {
		t.Errorf("logs/run.log in the squashed tree = %q, want the pre-run committed version", got)
	}
	// The excluded path is left exactly as it was, in the index AND the worktree.
	if got := strings.Fields(git(t, repo, "ls-files", "--stage", "--", "logs/run.log"))[1]; got != stagedBlob {
		t.Errorf("staged blob = %s, want %s: the squash destroyed the staged version of an excluded path", got, stagedBlob)
	}
	if b, err := os.ReadFile(filepath.Join(repo, "logs", "run.log")); err != nil || string(b) != "worktree\n" {
		t.Errorf("worktree content = %q (%v), want it untouched", b, err)
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

// Same recovery again, but with the caller's context still LIVE: c.git wraps
// every command in its own gitOpTimeout, so that timeout kills the post-commit
// rev-parse without ctx.Err() ever becoming non-nil. A recovery gated on
// cancellation would report a landed commit as a failure and drop its SHA from
// the run summary. The shim stands in for the timeout kill by failing the
// post-commit rev-parse outright -- indistinguishable from here, and it does not
// cost the test ten minutes.
func TestCommitRecoversLandedCommitWhenRevParseFailsWithLiveContext(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skipf("sh not found: %v", err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	repo := gitRepo(t)
	binDir := t.TempDir()
	marker := filepath.Join(binDir, "commit-done")
	// commit runs normally and drops a marker; the FIRST rev-parse seen after the
	// marker exists (the post-commit SHA lookup) removes the marker and fails, so
	// committedSHA's own rev-parse passes straight through afterwards.
	shim := "#!/bin/sh\n" +
		`op=""; for a in "$@"; do case "$a" in commit) op=commit;; rev-parse) op=revparse;; esac; done` + "\n" +
		`if [ "$op" = commit ]; then "` + realGit + `" "$@"; st=$?; touch "` + marker + `"; exit $st; fi` + "\n" +
		`if [ "$op" = revparse ] && [ -f "` + marker + `" ]; then rm -f "` + marker + `"; exit 1; fi` + "\n" +
		`exec "` + realGit + `" "$@"` + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(shim), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeFile(t, repo, "fixed.go", "package main\n")

	c := New(config.Target{Path: repo})
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body")
	if err != nil {
		t.Fatalf("Commit() = %v, want the landed commit recovered", err)
	}
	head := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	if sha == "" || sha != head {
		t.Fatalf("Commit() SHA = %q, want landed HEAD %q", sha, head)
	}
}

// The `git commit` command itself has the same live-context exposure: c.git's own
// gitOpTimeout can kill it after the ref update but before it exits (a large
// index, a slow gpg signer), with the caller's ctx never canceled. A recovery
// gated on ctx.Err() would report the landed commit as "git commit: ..." and the
// orchestrator would withdraw the fix while the commit sits in history. The shim
// runs the real commit and then fails, standing in for that kill.
func TestCommitRecoversLandedCommitWhenCommitFailsWithLiveContext(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skipf("sh not found: %v", err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	repo := gitRepo(t)
	binDir := t.TempDir()
	shim := "#!/bin/sh\n" +
		`is_commit=0; for a in "$@"; do if [ "$a" = "commit" ]; then is_commit=1; fi; done` + "\n" +
		`if [ "$is_commit" = 1 ]; then "` + realGit + `" "$@" || exit $?; exit 1; fi` + "\n" +
		`exec "` + realGit + `" "$@"` + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(shim), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeFile(t, repo, "fixed.go", "package main\n")

	c := New(config.Target{Path: repo})
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body")
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

// The counterpart: a commit that fails WITHOUT moving the ref must still surface
// its error, so dropping the ctx.Err() gate above cannot turn a genuine failure
// into a phantom SHA. The shim fails the commit before the real git ever runs.
func TestCommitFailsWhenCommitFailsWithoutMovingRef(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skipf("sh not found: %v", err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	repo := gitRepo(t)
	binDir := t.TempDir()
	shim := "#!/bin/sh\n" +
		`is_commit=0; for a in "$@"; do if [ "$a" = "commit" ]; then is_commit=1; fi; done` + "\n" +
		`if [ "$is_commit" = 1 ]; then echo "commit refused" >&2; exit 1; fi` + "\n" +
		`exec "` + realGit + `" "$@"` + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(shim), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	head := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	writeFile(t, repo, "fixed.go", "package main\n")

	c := New(config.Target{Path: repo})
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body")
	if err == nil {
		t.Fatalf("Commit() = %q, nil; want an error when the commit never landed", sha)
	}
	if sha != "" {
		t.Errorf("Commit() SHA = %q, want empty on a failed commit", sha)
	}
	if now := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD")); now != head {
		t.Errorf("HEAD moved to %q, want it left at %q", now, head)
	}
}

// The recovery above compares HEAD against a snapshot taken BEFORE the commit, so
// that snapshot has to be trustworthy: the pre-commit rev-parse can fail for
// reasons that say nothing about HEAD (gitOpTimeout with the caller's ctx live, a
// process-group kill), and reading its empty result as "no commit yet" would make
// the PRE-EXISTING HEAD look like the commit this round just made -- a failed
// commit reported as a landed one, with the coder's staged edits left in the tree.
// The shim fails the snapshot lookup (exit 128, not the quiet exit 1 that means an
// unborn branch) and then fails the commit, leaving every other lookup working.
func TestCommitFailsWhenTheSnapshotLookupFailsAndTheCommitDoesNotLand(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skipf("sh not found: %v", err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	repo := gitRepo(t)
	binDir := t.TempDir()
	// The pre-commit snapshot is the only `rev-parse --verify` Commit runs here;
	// the recovery's own lookup is a plain `rev-parse HEAD` and passes through, so
	// a recovery gated on the snapshot alone would happily return the old HEAD.
	shim := "#!/bin/sh\n" +
		`op=""; for a in "$@"; do case "$a" in --verify) op=verify;; commit) op=commit;; esac; done` + "\n" +
		`if [ "$op" = verify ]; then echo "fatal: killed" >&2; exit 128; fi` + "\n" +
		`if [ "$op" = commit ]; then echo "commit refused" >&2; exit 1; fi` + "\n" +
		`exec "` + realGit + `" "$@"` + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(shim), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	head := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	writeFile(t, repo, "fixed.go", "package main\n")

	c := New(config.Target{Path: repo})
	sha, err := c.Commit(t.Context(), "fixpoint: round 1", "body")
	if err == nil {
		t.Fatalf("Commit() = %q, nil; want an error when the commit never landed", sha)
	}
	if sha != "" {
		t.Errorf("Commit() SHA = %q, want empty rather than the pre-existing HEAD %q", sha, head)
	}
	if now := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD")); now != head {
		t.Errorf("HEAD moved to %q, want it left at %q", now, head)
	}
}

// SquashSince's closing `reset --soft` has the same exposure: the ref update can
// land and the command still be cut short -- by gitOpTimeout, with the caller's
// context live. The squash must be reported with its SHA, not lost. The shim runs
// the real reset and then fails, standing in for the kill after the ref moved.
func TestSquashSinceRecoversLandedResetWhenItFailsWithLiveContext(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skipf("sh not found: %v", err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not found: %v", err)
	}
	repo := gitRepo(t)
	c := New(config.Target{Path: repo})
	base := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	writeFile(t, repo, "one.go", "package main\n")
	if _, err := c.Commit(t.Context(), "fix one.go", "body"); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	// Only the closing `reset --soft` is intercepted; the per-exclude `reset -q
	// HEAD -- path` calls and everything else run untouched.
	shim := "#!/bin/sh\n" +
		`soft=0; for a in "$@"; do if [ "$a" = "--soft" ]; then soft=1; fi; done` + "\n" +
		`if [ "$soft" = 1 ]; then "` + realGit + `" "$@" || exit $?; exit 1; fi` + "\n" +
		`exec "` + realGit + `" "$@"` + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(shim), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	sha, err := c.SquashSince(t.Context(), base, "fixpoint: run", "Fixed:\n- one")
	if err != nil {
		t.Fatalf("SquashSince() = %v, want the landed squash recovered", err)
	}
	head := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	if sha == "" || sha != head {
		t.Fatalf("SquashSince() SHA = %q, want landed HEAD %q", sha, head)
	}
	if count := strings.TrimSpace(git(t, repo, "rev-list", "--count", base+"..HEAD")); count != "1" {
		t.Errorf("commits since base = %s, want the single squashed commit", count)
	}
}

// gitenv.SafeConfigArgs points core.hooksPath at /dev/null so a hook shipped in an
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
// PR-supplied hook. gitenv.Harden propagates the hooks-disabling override to
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
		t.Error("post-checkout hook executed; gitenv.Harden did not disable hooks for a subprocess lacking -c overrides")
	}
}

// The ext:: transport's URL IS a command git runs, so a remote pointed at one
// turns any fetch into code execution with fixpoint's inherited environment.
// git itself defaults protocol.ext.allow to "never", but that default is
// CONFIGURABLE: a crafted .git/config that sets protocol.ext.allow=always turns
// it back on, and remote.<name>.url is not a key unsafeConfigKey refuses. The
// gitenv.SafeConfigArgs pin is what beats the repo's own value -- on both paths, since
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
		// c.git prepends gitenv.SafeConfigArgs as -c overrides AND sets the hardened env.
		{"git", func(c *Collector, remote string) (string, error) {
			return c.git(t.Context(), "fetch", "--no-tags", remote)
		}},
		// c.run invokes git with NO -c overrides -- the conditions gh's nested git
		// runs under, where only GIT_CONFIG_* from gitenv.Harden can protect it.
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

// core.alternateRefsCommand is a value git runs through the SHELL. Its
// documentation calls it server-side only, but a client-side fetch enumerates
// the tips of every .git/objects/info/alternates entry to seed negotiation and
// runs this command to do it -- so a crafted checkout that ships an alternates
// file plus the setting turns pr mode's checkout/fetch into code execution with
// fixpoint's inherited environment. The gitenv.SafeConfigArgs pin is what beats the
// repo's own value, on both the -c path and the GIT_CONFIG_* path gh's internal
// git takes.
func TestFetchDoesNotRunAlternateRefsCommand(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to run the alternate-refs command")
	}
	for _, tc := range []struct {
		name  string
		fetch func(*Collector, string) (string, error)
	}{
		// c.git prepends gitenv.SafeConfigArgs as -c overrides AND sets the hardened env.
		{"git", func(c *Collector, remote string) (string, error) {
			return c.git(t.Context(), "fetch", "--no-tags", remote)
		}},
		// c.run invokes git with NO -c overrides -- the conditions gh's nested git
		// runs under, where only GIT_CONFIG_* from gitenv.Harden can protect it.
		{"run", func(c *Collector, remote string) (string, error) {
			return c.run(t.Context(), "git", "fetch", "--no-tags", remote)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := gitRepo(t)
			// src doubles as the fetch source and as the alternate object store the
			// command is invoked for.
			src := gitRepo(t)
			sentinel := filepath.Join(t.TempDir(), "alternate-refs-command-ran")
			evil := filepath.Join(repo, "evil-alternate-refs.sh")
			if err := os.WriteFile(evil, []byte("#!/bin/sh\ntouch '"+sentinel+"'\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			alternates := filepath.Join(repo, ".git", "objects", "info", "alternates")
			if err := os.WriteFile(alternates, []byte(filepath.Join(src, ".git", "objects")+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			git(t, repo, "config", "core.alternateRefsCommand", evil)
			git(t, repo, "config", "remote.src.url", src)

			if out, err := tc.fetch(New(config.Target{Path: repo}), "src"); err != nil {
				t.Fatalf("fetch: %v: %s", err, out)
			}
			if _, serr := os.Stat(sentinel); serr == nil {
				t.Error("the alternate-refs command executed; core.alternateRefsCommand=false is not reaching this git invocation")
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
// rather than (false, nil) or claiming a clean reconciliation -- while still
// reporting stashed=true, because the stash landed and the edits it holds are
// recoverable with `git stash pop`.
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
	if !stashed {
		t.Fatalf("StashDirty stashed = false; want true -- `git stash push` succeeded, so the edits are in stash@{0}")
	}
}

// UnsafeConfig flags the repo-local git settings gitenv.SafeConfigArgs cannot neutralize
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

	// The http section reroutes the very fetch the guard exists for without running
	// any program: resolve github.com to an attacker's address, drop certificate
	// verification, and `gh pr checkout` hands the operator's github.com credential
	// to whoever answers -- while remote.origin.url still reads as ordinary GitHub.
	// The URL-specific forms are included because git prefers the most specific
	// match, so they beat any generic pin gitenv could carry.
	t.Run("flags http settings that reroute or weaken a fetch", func(t *testing.T) {
		repo := gitRepo(t)
		git(t, repo, "config", "http.curloptResolve", "github.com:443:203.0.113.7")
		git(t, repo, "config", "http.proxy", "http://203.0.113.7:8080")
		git(t, repo, "config", "http.sslVerify", "false")
		git(t, repo, "config", "http.sslCAInfo", "./ca.pem")
		git(t, repo, "config", "http.pinnedPubkey", "sha256//AAAA")
		git(t, repo, "config", "http.sslCert", "./client.pem")
		git(t, repo, "config", "http.cookieFile", "./cookies")
		git(t, repo, "config", "http.emptyAuth", "true")
		git(t, repo, "config", "http.followRedirects", "true")
		git(t, repo, "config", "http.extraHeader", "Authorization: Basic Zm9v")
		git(t, repo, "config", "http.https://github.com/.sslVerify", "false")
		keys, err := New(config.Target{Path: repo}).UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Join(keys, ",")
		for _, want := range []string{
			"http.curloptresolve", "http.proxy", "http.sslverify", "http.sslcainfo",
			"http.pinnedpubkey", "http.sslcert", "http.cookiefile", "http.emptyauth",
			"http.followredirects", "http.extraheader",
			"http.https://github.com/.sslverify",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("UnsafeConfig() = %v, want it to include %q", keys, want)
			}
		}
	})

	// ...but an http setting that only tunes how the transfer runs changes neither
	// where git connects nor what it trusts or sends, and repositories do carry those
	// (http.postBuffer is the standard workaround for a large push). Refusing them
	// would be a refusal over a performance knob.
	t.Run("ignores http settings that only tune the transfer", func(t *testing.T) {
		repo := gitRepo(t)
		git(t, repo, "config", "http.postBuffer", "524288000")
		git(t, repo, "config", "http.lowSpeedLimit", "1000")
		git(t, repo, "config", "http.lowSpeedTime", "60")
		git(t, repo, "config", "http.version", "HTTP/1.1")
		git(t, repo, "config", "http.userAgent", "git/fixpoint")
		keys, err := New(config.Target{Path: repo}).UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 0 {
			t.Fatalf("UnsafeConfig() = %v, want none: these http settings only tune the transfer", keys)
		}
	})

	// A diff driver is a program git runs on the READ path, and it is the shape an
	// agent trips rather than fixpoint: Collect passes --no-ext-diff --no-textconv, but
	// the reviewer and coder CLIs run their own `git diff`/`git log -p`/`git blame`
	// inside the target, and the driver name is the repository's to choose so no -c
	// override can disable it.
	t.Run("flags diff drivers that execute programs", func(t *testing.T) {
		repo := gitRepo(t)
		git(t, repo, "config", "diff.external", "./payload")
		git(t, repo, "config", "diff.evil.command", "./payload")
		git(t, repo, "config", "diff.evil.textconv", "./payload")
		keys, err := New(config.Target{Path: repo}).UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Join(keys, ",")
		for _, want := range []string{"diff.external", "diff.evil.command", "diff.evil.textconv"} {
			if !strings.Contains(got, want) {
				t.Errorf("UnsafeConfig() = %v, want it to include %q", keys, want)
			}
		}
	})

	// ...but the rest of the diff section configures git's own engine and runs
	// nothing. Flagging those would refuse ordinary repositories over a stylistic
	// setting, which is how a security gate stops being used at all.
	t.Run("ignores diff settings that run no program", func(t *testing.T) {
		repo := gitRepo(t)
		git(t, repo, "config", "diff.algorithm", "histogram")
		git(t, repo, "config", "diff.noprefix", "true")
		git(t, repo, "config", "diff.evil.cachetextconv", "true")
		keys, err := New(config.Target{Path: repo}).UnsafeConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 0 {
			t.Fatalf("UnsafeConfig() = %v, want none: these diff settings execute nothing", keys)
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

// A content filter the OPERATOR configured globally is still a program the
// REPOSITORY can run: .gitattributes selects a filter by name, and .gitattributes
// is repository content (on the pr path it arrives with `gh pr checkout`, after
// the preflight). UnsafeConfig cannot report it without refusing every target on
// a git-lfs host, so ExternalActivatableConfig reports it separately for a warning.
func TestExternalActivatableConfig(t *testing.T) {
	t.Run("reports globally configured filters", func(t *testing.T) {
		repo := gitRepo(t)
		global := filepath.Join(t.TempDir(), "gitconfig")
		// What `git lfs install` writes into ~/.gitconfig, plus a global
		// credential.helper to show that only the attribute-selectable filters are
		// reported -- the rest of the operator's config is genuinely theirs alone.
		if err := os.WriteFile(global, []byte("[filter \"lfs\"]\n\tclean = git-lfs clean -- %f\n\tsmudge = git-lfs smudge -- %f\n\tprocess = git-lfs filter-process\n[credential]\n\thelper = store\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_CONFIG_GLOBAL", global)
		// Pin the system scope too, so a filter installed host-wide on the machine
		// running the tests cannot make the exact-match assertion below flaky.
		t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
		keys, err := New(config.Target{Path: repo}).ExternalActivatableConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got, want := strings.Join(keys, ","), "filter.lfs.clean,filter.lfs.process,filter.lfs.smudge"; got != want {
			t.Errorf("ExternalActivatableConfig() = %v, want exactly %q", keys, want)
		}
	})

	// A repo-supplied filter is UnsafeConfig's business: it is refused outright,
	// and reporting it here too would warn about a target the guard already
	// stopped (or that the operator explicitly trusted).
	t.Run("ignores filters the repository itself defines", func(t *testing.T) {
		repo := gitRepo(t)
		global := filepath.Join(t.TempDir(), "gitconfig")
		if err := os.WriteFile(global, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_CONFIG_GLOBAL", global)
		t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
		git(t, repo, "config", "filter.evil.clean", "sh -c 'id'")
		keys, err := New(config.Target{Path: repo}).ExternalActivatableConfig(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 0 {
			t.Fatalf("ExternalActivatableConfig() = %v, want none: a repo-scoped filter is UnsafeConfig's to refuse", keys)
		}
	})
}

// Both probes must fail soft on a target that is not a repository at all. The
// orchestrator's repo-supplied-config gate now applies in EVERY mode, so a
// review-only directory run -- the documented way to review a plain folder, which
// Collect serves with a filesystem walk -- reaches this pair, held back only by an
// IsGitRepo short-circuit ahead of them. If that guard ever moves, these calls are
// what decides whether such a run proceeds or dies with "inspect target git
// config", so what they do outside a repository is behavior, not an accident: the
// listing is simply empty there, and an empty listing must read as "no keys" and
// not as a failed probe.
//
// The config sources are pinned to nothing so the assertion holds on its own terms
// -- neither the operator's global config, the host's system config, nor inherited
// GIT_CONFIG_* overrides can be what supplies an entry that keeps the probes quiet.
func TestConfigProbesOutsideARepository(t *testing.T) {
	dir := t.TempDir() // not a git repo
	global := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(global, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_CONFIG_COUNT", "0")

	c := New(config.Target{Path: dir})
	if keys, err := c.UnsafeConfig(t.Context()); err != nil || len(keys) != 0 {
		t.Errorf("UnsafeConfig() = %v, %v outside a repository, want no keys and no error", keys, err)
	}
	if keys, err := c.ExternalActivatableConfig(t.Context()); err != nil || len(keys) != 0 {
		t.Errorf("ExternalActivatableConfig() = %v, %v outside a repository, want no keys and no error", keys, err)
	}
}

// A repo-local core.worktree points git's work tree somewhere else while .git
// stays put, so every git command fixpoint runs -- diff and ls-files during
// collection, add/commit during a fix round -- operates on that other directory
// while target.path still looks like an ordinary checkout. Nothing else on the
// preflight sees it: it executes no program (so unsafeConfigKey has no business
// flagging it), and `rev-parse --show-prefix` is empty at the redirected root, so
// AtRepoRoot passes. WorktreeOutOfScope is the check, and it must judge the
// EFFECTIVE root rather than the key, because git sets core.worktree in every
// legitimate submodule checkout.
func TestWorktreeOutOfScope(t *testing.T) {
	t.Run("plain repository is in scope", func(t *testing.T) {
		repo := gitRepo(t)
		got, err := New(config.Target{Path: repo}).WorktreeOutOfScope(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Fatalf("WorktreeOutOfScope() = %q, want \"\" for an ordinary checkout", got)
		}
	})

	// target.path is allowed to be a subdirectory of its repository (review-only
	// modes support it), and there the root git reports is legitimately an
	// ancestor. Refusing that would break every subdirectory target.
	t.Run("subdirectory of the repository is in scope", func(t *testing.T) {
		repo := gitRepo(t)
		writeFile(t, repo, "sub/f.go", "package sub\n")
		got, err := New(config.Target{Path: filepath.Join(repo, "sub")}).WorktreeOutOfScope(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Fatalf("WorktreeOutOfScope() = %q, want \"\" for a subdirectory of the repository", got)
		}
	})

	// The canonical root is what gets compared: an operator whose target.path runs
	// through a symlinked parent (~/work -> /mnt/src, /tmp -> /private/tmp) must
	// not be told their work tree escaped.
	t.Run("symlinked target.path is in scope", func(t *testing.T) {
		repo := gitRepo(t)
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(repo, link); err != nil {
			t.Fatal(err)
		}
		got, err := New(config.Target{Path: link}).WorktreeOutOfScope(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Fatalf("WorktreeOutOfScope() = %q, want \"\" when target.path reaches the repository through a symlink", got)
		}
	})

	// The attack: .git stays at target.path, the work tree is somewhere else
	// entirely. git then reports that directory's files as the repository's.
	t.Run("redirect to an unrelated directory is reported", func(t *testing.T) {
		repo := gitRepo(t)
		elsewhere := t.TempDir()
		writeFile(t, elsewhere, "secret.txt", "token\n")
		git(t, repo, "config", "core.worktree", elsewhere)

		// Prove the redirect really does divert collection, so this test fails if
		// the check is removed rather than passing on an inert setting.
		if out := git(t, repo, "ls-files", "--others", "--exclude-standard"); !strings.Contains(out, "secret.txt") {
			t.Fatalf("git ls-files = %q, want it to list the redirected tree's files", out)
		}

		got, err := New(config.Target{Path: repo}).WorktreeOutOfScope(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got == "" {
			t.Fatal("WorktreeOutOfScope() = \"\", want the redirected work tree reported")
		}
		if !strings.HasSuffix(got, filepath.Base(elsewhere)) {
			t.Errorf("WorktreeOutOfScope() = %q, want the redirected root %q", got, elsewhere)
		}
	})

	// A redirect to an ANCESTOR of target.path wears the same shape as the
	// legitimate subdirectory case above -- target.path sits inside the reported
	// root -- so containment alone would wave it through, while `core.worktree = /`
	// quietly makes the whole filesystem the tree. A repo-local core.worktree is
	// what tells the two apart: with one set, the root was configured, not
	// discovered.
	t.Run("redirect to an ancestor of the target is reported", func(t *testing.T) {
		repo := gitRepo(t)
		parent := filepath.Dir(repo)
		git(t, repo, "config", "core.worktree", parent)

		got, err := New(config.Target{Path: repo}).WorktreeOutOfScope(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got == "" {
			t.Fatalf("WorktreeOutOfScope() = \"\", want the redirect to %q reported", parent)
		}
	})

	// Neither shape is a work tree at all, so there is nothing to escape from and
	// the caller must not be handed a refusal (or an error) for it: a directory
	// target that is not a repository is an ordinary review.
	t.Run("no work tree is not an escape", func(t *testing.T) {
		bare := t.TempDir()
		git(t, bare, "init", "-q", "--bare")
		for name, path := range map[string]string{"bare repository": bare, "plain directory": t.TempDir()} {
			got, err := New(config.Target{Path: path}).WorktreeOutOfScope(t.Context())
			if err != nil {
				t.Fatalf("%s: WorktreeOutOfScope() err = %v", name, err)
			}
			if got != "" {
				t.Errorf("%s: WorktreeOutOfScope() = %q, want \"\"", name, got)
			}
		}
	})

	// noWorkTree matches git's English "must be run in a work tree" for the same
	// reason notARepository matches its own message, and depends on the same
	// LC_ALL=C pin in probeEnv. Without that pin a bare repository under a
	// translated git would read as a fault rather than an answer, and this
	// security check would abort preflight on an ordinary target. The shim stands
	// in for a git built with NLS -- English only under LC_ALL=C -- so the test
	// does not depend on which locales the host has installed.
	t.Run("bare repository under a translated git is not an escape", func(t *testing.T) {
		bare := t.TempDir()
		git(t, bare, "init", "-q", "--bare")
		t.Setenv("LC_ALL", "de_DE.UTF-8")
		t.Setenv("LANG", "de_DE.UTF-8")
		t.Setenv("LANGUAGE", "de")
		shimGit(t, "rev-parse", `    if [ "$LC_ALL" = C ]; then
      echo 'fatal: this operation must be run in a work tree' >&2
    else
      echo 'fatal: Diese Operation muss in einem Arbeitsverzeichnis ausgeführt werden' >&2
    fi
    exit 128`)

		got, err := New(config.Target{Path: bare}).WorktreeOutOfScope(t.Context())
		if err != nil || got != "" {
			t.Errorf("WorktreeOutOfScope() = %q, %v under a translated git, want \"\" with no error", got, err)
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
	writeFile(t, repo, "production.env", "DB_PASS=hunter2-suffix-spelling\n")
	writeFile(t, repo, "deploy/id_rsa.pem", "-----BEGIN PRIVATE KEY-----\ncommitted-key\n")
	// direnv's file, whose exports carry names no shape-based redaction knows.
	writeFile(t, repo, ".envrc", "export CUSTOM_API_KEY=hunter2-direnv\n")
	writeFile(t, repo, ".direnv/dump/env", "CUSTOM_API_KEY=hunter2-direnv-cache\n")
	writeFile(t, repo, "vendor/dep/c.go", "package dep // configured-exclude\n")
	writeFile(t, repo, "main.go", "package main // reviewed\n")
	// The same secrets under an upper- or mixed-case name: the mandatory patterns
	// are lowercase, and the pathspec magic has to fold or the file's full content
	// lands in the diff -- and so in every reviewer prompt and .prompt artifact.
	writeFile(t, repo, "STAGING.ENV", "DB_PASS=hunter2-upper-suffix\n")
	writeFile(t, repo, "certs/Wildcard.PEM", "-----BEGIN PRIVATE KEY-----\nmixed-case-key\n")
	writeFile(t, repo, "deploy/ID_RSA", "-----BEGIN PRIVATE KEY-----\nupper-case-key\n")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-qm", "secrets")
	// And as untracked files, which are listed by path rather than diffed.
	writeFile(t, repo, "sub/.env.local", "TOKEN=hunter2-untracked\n")
	writeFile(t, repo, "sub/.Env", "TOKEN=hunter2-untracked-mixed\n")
	writeFile(t, repo, "sub/.envrc.local", "export CUSTOM_API_KEY=hunter2-untracked-direnv\n")
	writeFile(t, repo, "sub/ok.txt", "fine\n")

	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{
		"hunter2-committed", ".env",
		"hunter2-suffix-spelling", "production.env",
		"committed-key", "id_rsa.pem",
		"hunter2-untracked", ".env.local",
		"configured-exclude", "vendor/dep/c.go",
		"hunter2-upper-suffix", "STAGING.ENV",
		"mixed-case-key", "Wildcard.PEM",
		"upper-case-key", "ID_RSA",
		"hunter2-untracked-mixed", "sub/.Env",
		"hunter2-direnv", ".envrc",
		"hunter2-direnv-cache", ".direnv/dump/env",
		"hunter2-untracked-direnv", "sub/.envrc.local",
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

// The git modes filter material by pathname only, and the name of a symlink says
// nothing about what opening it yields -- so without a destination check a pull
// request (the mode written for UNTRUSTED authors) can advertise an alias to a
// host secret in every reviewer prompt: the diff renders the committed link and
// the untracked listing introduces it as a path to "read directly", while
// reviewers run unsandboxed and are told to read the repository for context.
func TestCollectGitDiffSkipsSymlinksLeavingTheTarget(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD"})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	// TRACKED and inside the diff range, the case a PR produces: the aliases are
	// committed, so nothing but the destination check keeps them out.
	want, unwanted := symlinkTree(t, repo)
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-qm", "aliases")
	// And one untracked alias, which reaches the material through the "read them
	// directly" listing rather than the diff.
	outside := t.TempDir()
	writeFile(t, outside, "token", "SECRET=leaked\n")
	if err := os.Symlink(filepath.Join(outside, "token"), filepath.Join(repo, "untracked.txt")); err != nil {
		t.Fatal(err)
	}
	unwanted = append(unwanted, "untracked.txt")
	writeFile(t, repo, "sub/ok.txt", "fine\n")

	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range unwanted {
		if strings.Contains(material, name) {
			t.Errorf("collected material names out-of-scope symlink %q:\n%s", name, material)
		}
	}
	// ...while the in-scope alias and ordinary files still arrive: the exclusion
	// must not silently narrow the review.
	for _, name := range append(want, "sub/ok.txt") {
		if !strings.Contains(material, name) {
			t.Errorf("collected material dropped in-scope %q:\n%s", name, material)
		}
	}
}

// ghRemote must select the remote gh treats as the base without assuming
// "origin": match the base repo's canonical URL (host and owner/repo) against
// remote URLs, and only when gh names no base repository fall back to origin and
// then the first remote. It errors when there are no remotes, and when gh named a
// base repository no remote points at.
func TestGhRemote(t *testing.T) {
	installGh := func(t *testing.T, body string) {
		t.Helper()
		binDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte("#!/bin/sh\n"+body), 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	// The host-aware defense below is only as good as the question ghBaseIdentity
	// asks gh: `--json url` names the HOST the PR's objects live on, while the
	// nameWithOwner it replaced reports a bare "acme/widget" that remoteIdentity
	// refuses -- ghRemote would then think gh named no base repository and fall
	// back to a remote of the checkout's choosing. A shim that answers any argv
	// cannot catch that regression, so this one answers ONLY the exact invocation
	// and records what it was really called with; the recording is asserted on
	// cleanup so every subtest below covers it.
	installGhURL := func(t *testing.T, url string) {
		t.Helper()
		const wantArgv = "repo view --json url --jq .url"
		argvFile := filepath.Join(t.TempDir(), "argv")
		installGh(t, "printf '%s' \"$*\" >>"+argvFile+"\n"+
			"[ \"$*\" = '"+wantArgv+"' ] || exit 1\n"+
			"echo "+url+"\n")
		t.Cleanup(func() {
			got, err := os.ReadFile(argvFile)
			if err != nil {
				t.Errorf("gh was never invoked: %v", err)
				return
			}
			if string(got) != wantArgv {
				t.Errorf("gh invoked as %q, want exactly one %q: the canonical URL is what names the host the fetch must contact", got, wantArgv)
			}
		})
	}

	t.Run("no remotes errors", func(t *testing.T) {
		installGh(t, "exit 1\n")
		c := New(config.Target{Path: gitRepo(t)})
		if _, err := c.ghRemote(t.Context()); err == nil || !strings.Contains(err.Error(), "no git remotes") {
			t.Fatalf("ghRemote() = %v, want no-remotes error", err)
		}
	})

	t.Run("base repo URL match beats origin fallback", func(t *testing.T) {
		installGhURL(t, "https://github.com/acme/widget")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "origin", "https://github.com/acme/other.git")
		git(t, repo, "remote", "add", "upstream", "https://github.com/acme/widget.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "upstream" {
			t.Fatalf("ghRemote() = %q, %v; want upstream (URL match)", r, err)
		}
	})

	// The identity must include the host. A checkout can configure a remote for
	// the same owner/repo on a server of its choosing -- and git lists remotes
	// alphabetically, so such a name sorts before the legitimate one. Matching on
	// owner/repo alone would select it and `git fetch` would then contact the
	// attacker's host as the operator.
	t.Run("same owner/repo on another host is not matched", func(t *testing.T) {
		installGhURL(t, "https://github.com/acme/widget")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "attacker", "https://attacker.example/acme/widget.git")
		git(t, repo, "remote", "add", "upstream", "https://github.com/acme/widget.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "upstream" {
			t.Fatalf("ghRemote() = %q, %v; want upstream (host must be part of the identity)", r, err)
		}
	})

	// An scp-like remote for the base repository is the same fetch target as the
	// https URL gh reports, so it must still match rather than be refused.
	t.Run("scp-like remote matches the gh URL", func(t *testing.T) {
		installGhURL(t, "https://github.com/acme/widget")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "upstream", "git@github.com:acme/widget.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "upstream" {
			t.Fatalf("ghRemote() = %q, %v; want upstream (scp-like URL match)", r, err)
		}
	})

	// Once gh HAS named the base repository, a checkout whose remotes all point at
	// ANOTHER HOST must be refused: falling back to origin/first-remote there lets
	// the checkout pick the server the PR base is fetched from.
	t.Run("no remote for the gh base repo host is refused", func(t *testing.T) {
		installGhURL(t, "https://github.com/acme/widget")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "origin", "https://attacker.example/acme/widget.git")
		_, err := New(config.Target{Path: repo}).ghRemote(t.Context())
		if err == nil || !strings.Contains(err.Error(), "no git remote points at") {
			t.Fatalf("ghRemote() = %v, want a refusal naming the unmatched base repository", err)
		}
	})

	// The ordinary fork clone: the only remote is the operator's fork while gh
	// resolves the base repository to the parent. Both live on the same host and
	// the forge serves fork-network objects, so the fork remote is a working fetch
	// target -- refusing here would break `gh pr checkout` runs that worked before.
	t.Run("fork remote on the base repo host is used", func(t *testing.T) {
		installGhURL(t, "https://github.com/acme/widget")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "origin", "https://github.com/me/widget.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "origin" {
			t.Fatalf("ghRemote() = %q, %v; want origin (fork remote on the base host)", r, err)
		}
	})

	// The same-host fallback must not outrank a remote for the base repository
	// itself, and must never reach across hosts even when the fork remote sorts
	// first and is named origin.
	t.Run("exact base repo match beats a same-host fork remote", func(t *testing.T) {
		installGhURL(t, "https://github.com/acme/widget")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "origin", "https://github.com/me/widget.git")
		git(t, repo, "remote", "add", "upstream", "https://github.com/acme/widget.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "upstream" {
			t.Fatalf("ghRemote() = %q, %v; want upstream (exact match beats same-host fallback)", r, err)
		}
	})

	t.Run("same owner/repo on another host is not a same-host fallback", func(t *testing.T) {
		installGhURL(t, "https://github.com/acme/widget")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "origin", "https://attacker.example/acme/widget.git")
		git(t, repo, "remote", "add", "zfork", "https://github.com/me/widget.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "zfork" {
			t.Fatalf("ghRemote() = %q, %v; want zfork (only the base host may serve the fetch)", r, err)
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

	// A remote NAME is a repo-controlled config subsection, so an untrusted
	// checkout can ship one that looks like a git option. It must never be
	// returned as the remote to fetch from -- neither as the first-remote
	// fallback (git sorts it before ordinary names) nor via the URL match.
	t.Run("option-like remote name is skipped", func(t *testing.T) {
		installGhURL(t, "https://github.com/acme/widget")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "--", "--upload-pack=/tmp/payload", "https://github.com/acme/widget.git")
		git(t, repo, "remote", "add", "upstream", "https://github.com/acme/widget.git")
		if r, err := New(config.Target{Path: repo}).ghRemote(t.Context()); err != nil || r != "upstream" {
			t.Fatalf("ghRemote() = %q, %v; want upstream (option-like name skipped)", r, err)
		}
	})

	t.Run("only option-like remotes errors", func(t *testing.T) {
		// The plain shim, not installGhURL: gh is never consulted here because the
		// option-like refusal comes before ghRemote asks for the base repository.
		installGh(t, "echo https://github.com/acme/widget\n")
		repo := gitRepo(t)
		git(t, repo, "remote", "add", "--", "--upload-pack=/tmp/payload", "https://github.com/acme/widget.git")
		_, err := New(config.Target{Path: repo}).ghRemote(t.Context())
		if err == nil || !strings.Contains(err.Error(), "option-like names") {
			t.Fatalf("ghRemote() = %v, want option-like-name refusal", err)
		}
	})
}

// Prepare must refuse a pr-mode run whose only remote has an option-like name
// rather than hand the name to `git fetch`, where git would parse it as an
// option (--upload-pack=<program> is executed by the local transport).
func TestPreparePROptionLikeRemoteRefused(t *testing.T) {
	repo := gitRepo(t)
	git(t, repo, "remote", "add", "--", "--upload-pack=/tmp/payload", "https://github.com/acme/widget.git")
	binDir := t.TempDir()
	// A base oid that is not a local object, so Prepare reaches the fetch.
	const absentOid = "0123456789012345678901234567890123456789"
	stub := "#!/bin/sh\ncase \"$1 $2\" in\n\"pr checkout\") : ;;\n\"pr view\") echo " + absentOid + " ;;\n*) exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	err := New(config.Target{Mode: "pr", Path: repo, PR: 7}).Prepare(t.Context())
	if err == nil || !strings.Contains(err.Error(), "option-like names") {
		t.Fatalf("Prepare() = %v, want option-like-name refusal", err)
	}
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
	"production.env", "svc/docker.env",
	".envrc", "svc/.envrc", ".envrc.local", ".direnv/dump/env",
	"key.pem", "certs/server.pem", "certs/bundle.p12", "certs/bundle.pfx",
	"server.key", "certs/tls.key",
	"store.jks", "certs/app.keystore", "keys/deploy.ppk",
	"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", "home/.ssh/id_ed25519",
	".npmrc", ".netrc", ".pgpass", "home/.netrc",
	".git-credentials", "home/.git-credentials",
	"credentials", "home/.aws/credentials",
	"kubeconfig", "home/.kube/kubeconfig",
	"secrets.kdbx", "vault/secrets.kdbx",
	// Casings the lowercase patterns must still recognize: a Windows-authored
	// branch or an operator's own habit produces them, and a case-sensitive match
	// would leave the one exclusion no config can drop protecting nothing. Each
	// name differs from the lowercase fixtures above by more than its case, so the
	// tree lays out distinct files on a case-INsensitive filesystem too.
	"STAGING.ENV", "svc/.Env", "certs/Wildcard.PEM",
	"home/.ssh/ID_RSA", "deploy/KUBECONFIG", "aws/Credentials",
	"deploy/.ENVRC",
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
// collection paths, because they are separate code -- listGitFiles asks git for the
// listing, walkFiles walks the filesystem, and both filter through skipFile -- and a
// change to either one, or to compileGlobs'
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

// The mandatory patterns name FILES, and must not take a directory that happens to
// share the name with them. "credentials" and "kubeconfig" are ordinary Go package
// and source names (grpc-go's credentials/, aws-sdk-go's aws/credentials/), the
// patterns cannot be switched off by any config, and the material carries no signal
// that something was dropped -- so widening them would silently delete the
// authentication code, the part a reviewer most needs to see, from every prompt.
// It would also split the collectors: git matches ":(exclude,glob)**/credentials"
// with wildmatch, which does not reach inside a directory of that name, so the git
// diff/PR modes would still review what directory mode had stopped showing.
func TestCollectDirectoryKeepsSourceUnderCredentialNamedDirectory(t *testing.T) {
	const src = "internal/credentials/aws.go"
	writeTree := func(t *testing.T, dir string) {
		t.Helper()
		writeFile(t, dir, src, "package credentials\n")
		writeFile(t, dir, "internal/kubeconfig/load.go", "package kubeconfig\n")
		// The file the patterns are actually for, in the same tree: the directory
		// stays, its credential-file namesake still goes.
		writeFile(t, dir, "home/.aws/credentials", "SECRET=leaked\n")
	}
	assert := func(t *testing.T, material string) {
		t.Helper()
		for _, want := range []string{src, "internal/kubeconfig/load.go"} {
			if !strings.Contains(material, want) {
				t.Errorf("collected material dropped source file %q:\n%s", want, material)
			}
		}
		if strings.Contains(material, "home/.aws/credentials") {
			t.Errorf("collected material names a credential file:\n%s", material)
		}
	}
	t.Run("git", func(t *testing.T) {
		repo := gitRepo(t)
		writeTree(t, repo)
		git(t, repo, "add", "-A")
		material, err := New(config.Target{Mode: "directory", Path: repo, Exclude: nil}).Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assert(t, material)
	})
	t.Run("walk", func(t *testing.T) {
		dir := t.TempDir()
		writeTree(t, dir)
		material, err := New(config.Target{Mode: "directory", Path: dir, Exclude: nil}).Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assert(t, material)
	})
}

// A target.exclude entry naming a DIRECTORY has to remove that directory's
// CONTENTS, and has to do it identically on both collection paths. The exclude
// list is the only filter directory mode has, and the entry is what an operator
// writes to keep a directory of arbitrarily named secrets -- the shape
// mandatoryExcludes cannot recognize -- out of every reviewer prompt. It used to
// work only on the non-git walk: the globs are anchored ^...$ against full paths,
// git ls-files emits no directory entries, so nothing on the git path ever tested
// "config/secrets" against anything but the files under it, and the entry silently
// protected nothing on the common target. Both spellings and both collectors are
// asserted here so the two cannot drift apart again.
func TestCollectDirectoryExcludesDirectoryContents(t *testing.T) {
	// Not a credential-shaped name: a *.key would be dropped by the mandatory
	// patterns whatever target.exclude says, and the test would pass without
	// reading the exclude entry at all.
	const secret = "config/secrets/prod.token"
	writeTree := func(t *testing.T, dir string) {
		t.Helper()
		writeFile(t, dir, secret, "TOKEN=leaked\n")
		writeFile(t, dir, "pkg/a.go", "package pkg\n")
	}
	assert := func(t *testing.T, material string) {
		t.Helper()
		if strings.Contains(material, secret) {
			t.Errorf("collected material names excluded directory content %q:\n%s", secret, material)
		}
		if !strings.Contains(material, "pkg/a.go") {
			t.Errorf("collected material dropped an ordinary source file:\n%s", material)
		}
	}
	for _, glob := range []string{"config/secrets", "config/secrets/"} {
		t.Run("git/"+glob, func(t *testing.T) {
			repo := gitRepo(t)
			writeTree(t, repo)
			// TRACKED, the harder case: --cached lists index entries, so no ignore
			// rule stands in for the exclude entry.
			git(t, repo, "add", "-A")
			material, err := New(config.Target{Mode: "directory", Path: repo, Exclude: []string{glob}}).Collect(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			assert(t, material)
		})
		t.Run("walk/"+glob, func(t *testing.T) {
			dir := t.TempDir()
			writeTree(t, dir)
			material, err := New(config.Target{Mode: "directory", Path: dir, Exclude: []string{glob}}).Collect(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			assert(t, material)
		})
	}
}

// The same wildcard-free directory entry has to remove that directory's contents
// in the GIT modes too -- where compileGlobs is never called at all. Collect
// builds its `git diff` and `git ls-files` argv from collectPathspec, so the
// exclusion there is enforced solely by git's own :(exclude,glob) prefix
// semantics, and compileGlobs' widening is only correct because the two agree.
// The git modes are also where a divergence costs the most: the diff embeds full
// file CONTENT, so a missed entry puts config/secrets/prod.token verbatim into
// every reviewer prompt and every .prompt artifact rather than merely naming its
// path -- silently, because the entry looks configured.
//
// The other direction is pinned here as well. "**/credentials" carries a
// wildcard, git matches it with wildmatch under WM_PATHNAME, and widening it to
// descendants would delete an ordinary internal/credentials Go package from
// review. It is a mandatory exclude no config can drop, so it is in force in
// every subtest below and its package must still arrive.
func TestCollectGitModesExcludeDirectoryContents(t *testing.T) {
	// Not a credential-shaped name: a *.key would be dropped by the mandatory
	// patterns whatever target.exclude says, and the test would pass without
	// reading the exclude entry at all.
	const (
		secret        = "config/secrets/prod.token"
		secretContent = "TOKEN=leaked-in-diff"
		newSecret     = "config/secrets/new.token"
	)
	// Everything is committed in the base and MODIFIED inside the diff range, so
	// the diff renders each file's content in full if a spec misses it.
	setup := func(t *testing.T) (repo, base string) {
		t.Helper()
		repo = gitRepo(t)
		writeFile(t, repo, secret, "TOKEN=original\n")
		writeFile(t, repo, "pkg/a.go", "package pkg\n")
		writeFile(t, repo, "internal/credentials/aws.go", "package credentials\n")
		git(t, repo, "add", "-A")
		git(t, repo, "commit", "-qm", "base")
		base = strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
		writeFile(t, repo, secret, secretContent+"\n")
		writeFile(t, repo, "pkg/a.go", "package pkg // reviewed\n")
		writeFile(t, repo, "internal/credentials/aws.go", "package credentials // reviewed-package\n")
		git(t, repo, "add", "-A")
		git(t, repo, "commit", "-qm", "changes")
		// The untracked half is built from the SAME pathspecs but reaches the
		// material through the "read them directly" listing rather than the diff.
		writeFile(t, repo, newSecret, "TOKEN=untracked\n")
		writeFile(t, repo, "sub/ok.txt", "fine\n")
		return repo, base
	}
	assert := func(t *testing.T, material string) {
		t.Helper()
		for _, leak := range []string{secret, secretContent, newSecret} {
			if strings.Contains(material, leak) {
				t.Errorf("collected material leaked excluded directory content %q:\n%s", leak, material)
			}
		}
		for _, want := range []string{
			"pkg/a.go", "// reviewed", "sub/ok.txt",
			// The wildcard exclude must NOT have been widened to descendants.
			"internal/credentials/aws.go", "// reviewed-package",
		} {
			if !strings.Contains(material, want) {
				t.Errorf("collected material dropped %q:\n%s", want, material)
			}
		}
	}
	for _, glob := range []string{"config/secrets", "config/secrets/"} {
		t.Run("git-diff/"+glob, func(t *testing.T) {
			repo, base := setup(t)
			c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: base, Exclude: []string{glob}})
			if err := c.Prepare(t.Context()); err != nil {
				t.Fatal(err)
			}
			material, err := c.Collect(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			assert(t, material)
		})
		t.Run("pr/"+glob, func(t *testing.T) {
			repo, base := setup(t)
			// Stub gh: the PR head is already checked out and its base tip is the
			// local base commit, so Prepare pins it without fetching.
			binDir := t.TempDir()
			stub := "#!/bin/sh\ncase \"$1 $2\" in\n" +
				"\"pr checkout\") : ;;\n" +
				"\"pr view\") echo " + base + " ;;\n" +
				"*) exit 1 ;;\nesac\n"
			if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			c := New(config.Target{Mode: "pr", Path: repo, PR: 7, Exclude: []string{glob}})
			if err := c.Prepare(t.Context()); err != nil {
				t.Fatal(err)
			}
			material, err := c.Collect(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			assert(t, material)
		})
	}
}

// symlinkTree lays out a target directory holding one alias of every shape the
// destination check has to separate, and returns the target-relative paths that
// must survive collection and those that must not. The secret they reach for
// lives in a sibling directory, so an escaping link is a real escape rather than
// a path-string trick.
func symlinkTree(t *testing.T, dir string) (want, unwanted []string) {
	t.Helper()
	outside := t.TempDir()
	writeFile(t, outside, "id_rsa", "PRIVATE KEY\n")
	writeFile(t, dir, "pkg/a.go", "package pkg\n")
	writeFile(t, dir, "config/.env", "TOKEN=leaked\n")
	rel, err := filepath.Rel(dir, filepath.Join(outside, "id_rsa"))
	if err != nil {
		t.Fatal(err)
	}
	links := []struct{ dest, name string }{
		// The headline case: an innocent name, an absolute destination outside the
		// target, and no pattern in the world matches the name it was committed under.
		{filepath.Join(outside, "id_rsa"), "context.txt"},
		{rel, "relative.txt"},       // same escape, spelled relatively
		{outside, "docs"},           // a whole directory outside the target
		{"config/.env", "notes.md"}, // inside the root, but an excluded destination
		{"nowhere", "dangling.txt"}, // unresolvable: nothing can read it anyway
		{"pkg/a.go", "inside.txt"},  // legitimate, and must stay in scope
	}
	for _, l := range links {
		if err := os.Symlink(l.dest, filepath.Join(dir, l.name)); err != nil {
			t.Fatal(err)
		}
		if l.name == "inside.txt" {
			want = append(want, l.name)
			continue
		}
		unwanted = append(unwanted, l.name)
	}
	return append(want, "pkg/a.go"), append(unwanted, "config/.env")
}

// aliasPath returns a symlink pointing at dir, standing in for the ordinary case
// of a target.path reached through a symlinked parent: /tmp -> /private/tmp on
// macOS, or an operator whose checkout lives under a symlinked home or mount.
// The link sits in its own temp dir, so it is not itself part of the tree under
// review.
func aliasPath(t *testing.T, dir string) string {
	t.Helper()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, alias); err != nil {
		t.Fatal(err)
	}
	return alias
}

// A symlink's NAME says nothing about what opening it yields, so filtering the
// pathname alone lets a committed `context.txt -> ~/.ssh/id_rsa` walk straight
// past the mandatory credential patterns into the reviewer's file list -- and the
// reviewer is unsandboxed and told to read the files in scope, so the key lands in
// its prompt, its output, and the run's artifacts. Both collectors must therefore
// judge the resolved destination, and both are covered here for the same reason
// the credential test covers both: they filter at different call sites.
func TestCollectDirectorySkipsSymlinksLeavingTheTarget(t *testing.T) {
	assert := func(t *testing.T, material string, want, unwanted []string) {
		t.Helper()
		for _, name := range unwanted {
			if strings.Contains(material, name) {
				t.Errorf("collected material names out-of-scope symlink %q:\n%s", name, material)
			}
		}
		for _, name := range want {
			if !strings.Contains(material, name) {
				t.Errorf("collected material dropped in-scope %q:\n%s", name, material)
			}
		}
	}
	t.Run("git", func(t *testing.T) {
		repo := gitRepo(t)
		want, unwanted := symlinkTree(t, repo)
		// TRACKED, the harder case: --cached lists index entries no ignore rule can
		// remove, so only the destination check stands between a committed alias and
		// the reviewer prompt.
		git(t, repo, "add", "-A")
		material, err := New(config.Target{Mode: "directory", Path: repo}).Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assert(t, material, want, unwanted)
	})
	t.Run("walk", func(t *testing.T) {
		dir := t.TempDir()
		want, unwanted := symlinkTree(t, dir)
		material, err := New(config.Target{Mode: "directory", Path: dir}).Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assert(t, material, want, unwanted)
	})
	// The same two trees, reached through a symlinked target.path. Destinations
	// are compared fully resolved, so the ROOT has to be resolved too: measure a
	// resolved destination against an unresolved root and filepath.Rel reports
	// every link in the tree as ".."-prefixed, so inside.txt -- a legitimate,
	// in-scope alias -- disappears along with the escapes, silently narrowing the
	// listing that listFiles' denylist-only design exists to keep whole. Both
	// cases above run on a real t.TempDir() path, where resolving the root is a
	// no-op and cannot tell the difference.
	t.Run("git symlinked root", func(t *testing.T) {
		repo := gitRepo(t)
		want, unwanted := symlinkTree(t, repo)
		git(t, repo, "add", "-A")
		material, err := New(config.Target{Mode: "directory", Path: aliasPath(t, repo)}).Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assert(t, material, want, unwanted)
	})
	t.Run("walk symlinked root", func(t *testing.T) {
		dir := t.TempDir()
		want, unwanted := symlinkTree(t, dir)
		material, err := New(config.Target{Mode: "directory", Path: aliasPath(t, dir)}).Collect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		assert(t, material, want, unwanted)
	})
}

// A target.path that cannot be resolved must fail the collection rather than
// produce a listing: with no canonical root there is nothing to judge symlink
// destinations against, and "Files in scope (0)" reads to every later reader as
// a tree that was reviewed and found empty.
func TestCollectDirectoryUnresolvableTargetPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	material, err := New(config.Target{Mode: "directory", Path: dir}).Collect(t.Context())
	if err == nil {
		t.Fatalf("Collect() succeeded for a nonexistent target.path:\n%s", material)
	}
	if !strings.Contains(err.Error(), "resolve target.path") {
		t.Errorf("Collect() error = %v, want it to name the unresolvable target.path", err)
	}
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
	scope, err := c.fileScope()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	done := make(chan error, 1)
	go func() {
		_, _, err := c.listGitFiles(ctx, scope)
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

// The kill above is the listing's only containment, and it runs on the ordinary
// path -- a git that exits 0, no cancellation involved. Off darwin an EPERM from
// it proves the group still holds a member this process cannot signal, so a
// listing that reported success would hand the rest of the round a repository
// with a live git descendant in it. The errno needs a descendant under other
// credentials, which a test cannot create, so the kill is stubbed after the real
// one has run and left nothing behind.
func TestGitScanNULReportsFailedCleanupKill(t *testing.T) {
	repo := gitRepo(t)
	orig := gitCleanupKill
	t.Cleanup(func() { gitCleanupKill = orig })
	gitCleanupKill = func(cmd *exec.Cmd) error {
		_ = orig(cmd)
		return syscall.EPERM
	}
	c := New(config.Target{Mode: "directory", Path: repo})
	var seen int
	err := c.gitScanNUL(t.Context(), func(string) { seen++ }, "ls-files", "--cached", "-z")
	if err == nil {
		t.Fatal("gitScanNUL() = nil after a group kill that reported an uncontained group; want the failure surfaced")
	}
	if !errors.Is(err, syscall.EPERM) {
		t.Errorf("gitScanNUL() err = %v, want it to carry the kill's EPERM", err)
	}
	if seen == 0 {
		t.Error("the listing itself never ran; the test proved nothing about a SUCCESSFUL scan's cleanup")
	}
}

// An empty group -- what the cleanup kill meets on essentially every listing --
// must stay a success: reporting its os.ErrProcessDone would fail every
// collection in the program.
func TestGitScanNULIgnoresProcessDoneFromCleanupKill(t *testing.T) {
	repo := gitRepo(t)
	orig := gitCleanupKill
	t.Cleanup(func() { gitCleanupKill = orig })
	gitCleanupKill = func(cmd *exec.Cmd) error {
		_ = orig(cmd)
		return os.ErrProcessDone
	}
	c := New(config.Target{Mode: "directory", Path: repo})
	if err := c.gitScanNUL(t.Context(), func(string) {}, "ls-files", "--cached", "-z"); err != nil {
		t.Errorf("gitScanNUL() err = %v for a cleanup kill that found an empty group; want nil", err)
	}
}

// The scan reads stdout to EOF, and a child that inherited the write end holds it
// open after git itself exits. Nothing may make that wait unbounded: the whole
// collection would otherwise stall until gitOpTimeout (ten minutes) even though
// the listing is complete and git is gone.
func TestCollectDirectoryChildHoldingStdoutDoesNotStallScan(t *testing.T) {
	repo := gitRepo(t)
	// The child inherits stdout and outlives the leader by a minute. Only its stderr
	// is redirected, so the wait this test measures is the scan's own: holding stderr
	// too would add that stream's drain grace instead of exercising the stdout stall
	// the test is about.
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

// The group kill reaches only the group. A descendant that left it -- setsid, or a
// git that daemonizes -- keeps the inherited stdout open, so the listing pipe never
// reaches EOF and the scan blocks on a writer nothing in this process can signal.
// The leader's own exit has to bound that wait, at the drain grace stderr gets:
// leaving it to the operation timeout stalls the whole round for ten minutes on the
// interleaving that costs stderr two seconds, with git already finished. The
// listing cannot be shown to be complete, so it must fail rather than narrow the
// scope silently.
func TestListGitFilesDetachedChildHoldingStdoutIsBounded(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid unavailable to detach the child from the process group")
	}
	repo := gitRepo(t)
	pidFile := filepath.Join(t.TempDir(), "detached.pid")
	// The child outlives this test by design, so reap it rather than leave a sleeper
	// behind on every run (and every -count=N iteration).
	t.Cleanup(func() { reapDetachedChild(t, pidFile) })
	// setsid puts the child in a session -- and so a process group -- of its own, so
	// the SIGKILL sent to git's group misses it and it keeps the inherited stdout.
	// It publishes its PID so the cleanup above can find it, and execs the sleep so
	// that PID is the process actually holding the pipe. Its stderr goes to
	// /dev/null: holding that too would measure the stderr drain grace instead of
	// the stdout stall under test. The leader lingers a second before exiting so the
	// kill that follows its exit cannot land while setsid is still forking, which
	// would catch the child while it is still in the group.
	shimGit(t, "ls-files", "    printf 'main.go\\0'\n"+
		"    setsid sh -c 'echo $$ > "+pidFile+"; exec sleep 20' 2>/dev/null &\n"+
		"    sleep 1\n    exit 0")

	c := New(config.Target{Mode: "directory", Path: repo})
	scope, err := c.fileScope()
	if err != nil {
		t.Fatal(err)
	}
	// No deadline of its own: the context here is as live as the run's would be, so
	// the grace is the only thing that can end the scan. A test that raced a short
	// deadline against it would pass on a scan still bounded only by gitOpTimeout.
	// Asserted on listGitFiles rather than Collect so what is timed is the listing
	// alone and not the git commands leading up to it.
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, _, err := c.listGitFiles(t.Context(), scope)
		done <- err
	}()
	select {
	case err := <-done:
		// The identity, not merely the presence: an escaped writer holding stdout is
		// WHY the listing ended, and an operator reading this error has to be told
		// that rather than the mechanism (the read end closed under the blocked scan)
		// used to end it.
		if !errors.Is(err, errListingHeldOpen) {
			t.Fatalf("listGitFiles() err = %v, want it to wrap errListingHeldOpen", err)
		}
		if strings.Contains(err.Error(), os.ErrClosed.Error()) {
			t.Errorf("the error reports the mechanism instead of the cause: %v", err)
		}
		// The leader lingers a second and the grace is two; a generous ceiling still
		// far below both the child's 20s and the 10-minute operation timeout.
		if elapsed := time.Since(start); elapsed > 15*time.Second {
			t.Errorf("listGitFiles took %s; the drain grace did not bound the scan on the escaped writer", elapsed)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("listGitFiles hung on a detached descendant holding stdout; the leader's exit does not bound the scan")
	}
}

// The two failures above arrive together on exactly one interleaving -- a
// descendant under other credentials, alive in the group, holding the stdout
// write end -- and that interleaving is the whole reason the cleanup kill exists.
// The scan error outranks git's exit status, but a kill failure is not git's exit
// status: demoting it would tell the operator only that the listing may be short
// while a git child keeps running inside the target with fixpoint's hardened
// environment. Both must survive. The EPERM's errno needs credentials a test
// cannot arrange, so the kill is stubbed after the real one has run; the held-open
// stdout is real.
func TestGitScanNULHeldOpenListingStillReportsFailedCleanupKill(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid unavailable to detach the child from the process group")
	}
	repo := gitRepo(t)
	pidFile := filepath.Join(t.TempDir(), "detached.pid")
	t.Cleanup(func() { reapDetachedChild(t, pidFile) })
	// Same shim as the boundedness test above: the setsid child escapes the group
	// kill and holds the inherited stdout, so the scan can only be ended by the
	// grace, and the leader lingers so the kill cannot land mid-fork.
	shimGit(t, "ls-files", "    printf 'main.go\\0'\n"+
		"    setsid sh -c 'echo $$ > "+pidFile+"; exec sleep 20' 2>/dev/null &\n"+
		"    sleep 1\n    exit 0")
	orig := gitCleanupKill
	t.Cleanup(func() { gitCleanupKill = orig })
	gitCleanupKill = func(cmd *exec.Cmd) error {
		_ = orig(cmd)
		return syscall.EPERM
	}

	c := New(config.Target{Mode: "directory", Path: repo})
	err := c.gitScanNUL(t.Context(), func(string) {}, "ls-files", "--cached", "-z")
	if !errors.Is(err, errListingHeldOpen) {
		t.Errorf("gitScanNUL() err = %v, want it to wrap errListingHeldOpen", err)
	}
	if !errors.Is(err, syscall.EPERM) {
		t.Errorf("gitScanNUL() err = %v; the kill's EPERM was masked by the scan error, so the uncontained group goes unreported", err)
	}
}

// The grace exists for a write end no EOF is ever coming through, so what it has
// to bound is the ABSENCE of output -- not the scan's own work. git hands its whole
// listing to the pipe before it exits, so by the time the leader is reaped (which
// is when the grace is armed) a pipe buffer plus a scanner buffer of it can still be
// unread, and every one of those entries costs fn a glob sweep, an Lstat and
// sometimes an EvalSymlinks: on a network filesystem or a cold page cache that
// residue outlasts one grace. Deadlining the scan there would fail a listing that
// was complete, with an error blaming an escaped descendant that never existed --
// and would truncate the listing to make its own diagnosis true. Entries arrive
// slowly here and nothing holds stdout, so the listing must be delivered in full.
//
// This is the many-fast-callbacks half of that: every individual call is short and it
// is their number that outlasts the grace. The one-long-call half -- a single fn that
// outlasts a whole grace by itself -- is the test below.
func TestGitScanNULSlowCallbackDoesNotFailCompleteListing(t *testing.T) {
	repo := gitRepo(t)
	const entries = 100
	// The whole listing, then an immediate exit: the residue sits in the pipe with the
	// leader already gone. Nothing escapes the process group, so EOF is waiting behind
	// that residue -- the scan just has to get to it.
	shimGit(t, "ls-files", fmt.Sprintf("    i=0\n"+
		"    while [ $i -lt %d ]; do printf 'f%%d.go\\0' $i; i=$((i+1)); done\n"+
		"    exit 0", entries))

	var seen int
	// Derived from the grace rather than hardcoded, so raising that shared constant
	// cannot leave the whole listing finishing before the grace ever fires -- which
	// would keep this test passing while exercising none of the re-arm. Two and a HALF
	// graces' worth of callback spread over 100 entries means the grace armed at the
	// leader's exit has to be re-armed TWICE before the listing is through: sustained
	// progress across more than one window, not the single re-arm one long call would
	// exercise. The half window is what makes that structural rather than incidental:
	// the grace is armed at the REAP, a few milliseconds after the scan started
	// consuming, so with exactly two windows' worth of callback the second firing and
	// EOF would land within the sleeps' own overshoot of each other and many runs would
	// exercise one re-arm, not two. At 2.5 windows the second firing has half a grace of
	// callback still ahead of it. Each individual gap is a fortieth of a grace, so what
	// decides this test is the predicate under test and not timing noise on a loaded
	// machine: a cut on the second grace lands around the 80th entry and fails the count
	// below.
	perEntry := 5 * agent.PipeDrainGrace / (2 * entries)
	c := New(config.Target{Mode: "directory", Path: repo})
	start := time.Now()
	err := c.gitScanNUL(t.Context(), func(string) {
		seen++
		time.Sleep(perEntry)
	}, "ls-files", "--cached", "-z")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("gitScanNUL() err = %v, want nil: the grace bounded the callback's own work instead of the absence of output", err)
	}
	if seen != entries {
		t.Errorf("gitScanNUL delivered %d of %d entries; the listing was cut short", seen, entries)
	}
	// Nothing above can tell a re-armed grace from one that never fired: a listing
	// delivered before the first firing yields the same nil error and the same count.
	// The sleeps are floors, never ceilings, so a loaded machine only pushes this
	// further past the bound -- but a shortened gap, a shrunk entry count, or a grace
	// raised out from under the derivation would leave the test green while exercising
	// no re-arm at all. The same guard the one-long-call test below carries, for the
	// same reason.
	if elapsed < 5*agent.PipeDrainGrace/2 {
		t.Errorf("gitScanNUL returned after %s, sooner than the %s of callback it was supposed to span; the grace never fired twice, so the second re-arm was not exercised", elapsed, 5*agent.PipeDrainGrace/2)
	}
}

// The other shape a slow delivery takes, and the one a count of entries taken off the
// pipe cannot see: ONE callback that outlasts a whole grace by itself. listGitFiles'
// fn does an Lstat and, for a symlink, an EvalSymlinks; a single one of those on a
// hung network mount or a cold cache is enough. If progress were measured only on
// entry, the last entry's counter bump would land BEFORE the grace was armed and
// nothing would move again until fn returned, so the grace would fire on an unchanged
// count and cut a listing that was complete -- reporting errListingHeldOpen and
// blaming an escaped descendant that never existed. Cutting cannot even end the scan
// there: the cut path joins the scan goroutine, which is inside that very call.
//
// The shim lingers after its single entry so the callback is provably in flight when
// the leader is reaped and the grace is armed -- the exact state the re-arm has to
// survive -- and nothing escapes the process group, so EOF is waiting the moment fn
// returns.
func TestGitScanNULSingleLongCallbackDoesNotFailCompleteListing(t *testing.T) {
	repo := gitRepo(t)
	shimGit(t, "ls-files", "    printf 'main.go\\0'\n    sleep 1\n    exit 0")

	var seen int
	// Two graces in one call, derived from the constant so the call outlasts the grace
	// whatever it is set to: the entry is consumed about a second before the leader
	// exits, the grace fires one grace after that -- with fn still a whole grace from
	// returning -- and must re-arm rather than cut.
	c := New(config.Target{Mode: "directory", Path: repo})
	start := time.Now()
	err := c.gitScanNUL(t.Context(), func(string) {
		seen++
		time.Sleep(2 * agent.PipeDrainGrace)
	}, "ls-files", "--cached", "-z")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("gitScanNUL() err = %v, want nil: the grace bounded one callback's own work instead of the absence of output", err)
	}
	if seen != 1 {
		t.Errorf("gitScanNUL delivered %d of 1 entries; the listing was cut short", seen)
	}
	// Without this the test could pass on a callback that never actually outlasted the
	// grace -- a shortened sleep, or a grace raised out from under it -- and would stop
	// exercising the re-arm while still looking green.
	if elapsed < 2*agent.PipeDrainGrace {
		t.Errorf("gitScanNUL returned after %s, sooner than the %s callback it was supposed to wait out; the grace was never in a position to cut", elapsed, 2*agent.PipeDrainGrace)
	}
}

// Re-arming on progress is not permission to keep the scan open indefinitely:
// entries arriving buy another window, they do not disarm the grace. So the other
// half of the predicate is a listing that arrives for a while and then genuinely
// stalls -- consumed past the leader's exit, then a descendant outside the process
// group holding the write end so EOF is never coming -- which must still be cut, on
// the grace that follows the last re-arm. Every other held-open test cuts on the
// FIRST grace, before any re-arm has happened, so nothing else covers the
// re-arm -> cut transition: a re-arm that lost the cut would leave this scan bounded
// only by the ten-minute operation timeout, with git long gone.
func TestGitScanNULReArmedGraceStillCutsAStalledListing(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid unavailable to detach the child from the process group")
	}
	repo := gitRepo(t)
	pidFile := filepath.Join(t.TempDir(), "detached.pid")
	// The child outlives this test by design, so reap it rather than leave a sleeper
	// behind on every run.
	t.Cleanup(func() { reapDetachedChild(t, pidFile) })
	const entries = 40
	// The whole listing up front, then the setsid child that escapes the group kill and
	// keeps the inherited stdout, so EOF never arrives and only the grace can end the
	// scan. It publishes its PID for the cleanup above and its stderr goes to /dev/null,
	// so what is measured is the stdout stall and not the stderr drain. The leader
	// lingers a second -- both so the kill cannot land while setsid is still forking and
	// so the grace is armed while the callback below is still working through the
	// listing, which is what makes the first firing a re-arm rather than a cut.
	shimGit(t, "ls-files", fmt.Sprintf("    i=0\n"+
		"    while [ $i -lt %d ]; do printf 'f%%d.go\\0' $i; i=$((i+1)); done\n"+
		"    setsid sh -c 'echo $$ > "+pidFile+"; exec sleep 30' 2>/dev/null &\n"+
		"    sleep 1\n    exit 0", entries))

	var seen int
	// The wall time of the last entry handed to fn, taken where the implementation
	// counts it (on entry, before the callback's own work), so the gap asserted below is
	// the same quantity the grace measures.
	var last time.Time
	// 40 entries at 50ms is 2s of callback against the leader's 1s linger: consumption is
	// still going when the grace is armed and is over well before that first window
	// closes, so the first firing sees progress and re-arms and the second sees none and
	// must cut.
	c := New(config.Target{Mode: "directory", Path: repo})
	start := time.Now()
	err := c.gitScanNUL(t.Context(), func(string) {
		seen++
		last = time.Now()
		time.Sleep(50 * time.Millisecond)
	}, "ls-files", "--cached", "-z")
	elapsed := time.Since(start)
	// The identity, not merely the presence: the listing could not be read to EOF, so it
	// may be short, and that is what the operator has to be told.
	if !errors.Is(err, errListingHeldOpen) {
		t.Fatalf("gitScanNUL() err = %v, want it to wrap errListingHeldOpen once the re-armed grace found no progress", err)
	}
	// A cut that landed while entries were still being consumed would truncate the
	// listing -- the very failure the re-arm exists to prevent -- and would also make
	// this test's stall a fiction.
	if seen != entries {
		t.Errorf("gitScanNUL delivered %d of %d entries; the grace cut a listing that was still arriving", seen, entries)
	}
	// A full grace must have elapsed with nothing consumed, so the cut cannot come
	// sooner than one grace after the last entry. The tolerance covers only the
	// microseconds between the implementation's counter increment and the timestamp
	// above it; a cut on the FIRST grace lands about a second after the last entry here,
	// nowhere near it.
	if gap := elapsed - last.Sub(start); gap < agent.PipeDrainGrace-100*time.Millisecond {
		t.Errorf("the listing was cut %s after its last entry, less than the %s grace; progress did not re-arm it", gap, agent.PipeDrainGrace)
	}
	// Two windows behind the leader's one-second linger is five seconds; a cut on the
	// first grace would land near three. The ceiling proves the stall was still bounded
	// by the grace and not by the child's own lifetime or gitOpTimeout.
	if elapsed < 2*agent.PipeDrainGrace {
		t.Errorf("gitScanNUL returned after %s, too soon to have re-armed: a single grace from the leader's exit would cut here", elapsed)
	}
	if elapsed > 20*time.Second {
		t.Errorf("gitScanNUL took %s; the re-armed grace stopped bounding the scan", elapsed)
	}
}

// The boundary between the two halves above, and the one the other held-open tests
// step around: the grace arms while a callback is in flight -- so the mark it takes is
// ODD -- and that callback then RETURNS onto a pipe an escaped writer is holding open.
// The counter moves once, to mark+1, and nothing about that increment says the pipe
// delivered anything: it is the end of the very work the previous firing already
// re-armed for. Counting it as progress buys the stalled scan a second full window,
// so the listing is cut two graces after the last thing that actually arrived instead
// of the one the grace promises -- and the re-arm -> cut test above cannot see it,
// since its grace arms between callbacks and its mark is even.
//
// The callback is released by the test rather than timed out of a sleep, so the
// release provably lands after the first firing (which must re-arm, fn being in
// flight) and inside the second window; what is then asserted is the distance from
// that release to the cut, which is one window under the predicate and two without it.
func TestGitScanNULGraceAfterInFlightCallbackReturnsCutsAStalledListing(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid unavailable to detach the child from the process group")
	}
	repo := gitRepo(t)
	pidFile := filepath.Join(t.TempDir(), "detached.pid")
	// The child outlives this test by design, so reap it rather than leave a sleeper
	// behind on every run.
	t.Cleanup(func() { reapDetachedChild(t, pidFile) })
	// One entry, delivered before the leader is anywhere near exiting, so the callback
	// below is provably in flight when the grace is armed. Then the setsid child that
	// escapes the group kill and keeps the inherited stdout, so EOF is never coming and
	// only the grace can end the scan; its stderr goes to /dev/null so what is measured
	// is the stdout stall. The leader lingers a second so the kill cannot land while
	// setsid is still forking.
	shimGit(t, "ls-files", "    printf 'main.go\\0'\n"+
		"    setsid sh -c 'echo $$ > "+pidFile+"; exec sleep 30' 2>/dev/null &\n"+
		"    sleep 1\n    exit 0")

	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var seen int
	// The wall time the callback returned -- the last instant anything at all moved the
	// progress counter -- read back only after gitScanNUL has returned and so joined the
	// goroutine that writes it.
	var returned time.Time
	c := New(config.Target{Mode: "directory", Path: repo})
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		done <- c.gitScanNUL(t.Context(), func(string) {
			seen++
			once.Do(func() { close(entered) })
			<-release
			returned = time.Now()
		}, "ls-files", "--cached", "-z")
	}()
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("gitScanNUL returned before delivering a listing entry: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("git never delivered a listing entry to fn")
	}
	// Half a window past the first firing: the grace is armed at the leader's exit (its
	// one-second linger), fires a window later with fn still blocked, and re-arms. The
	// release then lands in the middle of the second window, so the firing that closes
	// it is half a grace away and the one after it is a grace and a half -- the widest
	// margin either side of the single-grace bound asserted below.
	hold := time.Second + 3*agent.PipeDrainGrace/2
	select {
	case err := <-done:
		t.Fatalf("gitScanNUL returned after %s with fn still in flight: %v", time.Since(start), err)
	case <-time.After(time.Until(start.Add(hold))):
	}
	close(release)

	select {
	case err := <-done:
		elapsed := time.Since(start)
		// The identity, not merely the presence: the listing could not be read to EOF, so
		// it may be short, and that is what the operator has to be told.
		if !errors.Is(err, errListingHeldOpen) {
			t.Fatalf("gitScanNUL() err = %v, want it to wrap errListingHeldOpen once the grace found no progress", err)
		}
		if seen != 1 {
			t.Errorf("gitScanNUL delivered %d of 1 entries", seen)
		}
		gap := elapsed - returned.Sub(start)
		// The claim under test. A callback's return counted as pipe progress re-arms here
		// and the cut lands a grace and a half after it instead of half a grace.
		if gap > agent.PipeDrainGrace {
			t.Errorf("the stalled listing was cut %s after the in-flight callback returned, more than the %s grace: the callback's own return was counted as progress and bought the dead pipe another window", gap, agent.PipeDrainGrace)
		}
		// The other side, so the test cannot pass on a grace that never re-armed at all:
		// a firing that cut while fn was still in flight would be blocked joining that
		// very call, and would return within milliseconds of the release above.
		if gap < agent.PipeDrainGrace/4 {
			t.Errorf("the listing was cut %s after the callback returned, too soon for a firing of its own: the grace cut while fn was in flight and merely waited out the call", gap)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("gitScanNUL never returned once the in-flight callback did; the grace stopped bounding a scan on a held-open pipe")
	}
}

// The re-arm removes the grace as the outer bound on one shape of scan: a writer
// that escaped the process group and TRICKLES, delivering an entry inside every
// window, so every firing sees progress and re-arms forever. Nothing about that
// writer says a listing is still coming -- a looping or stalled git descendant
// that inherited stdout feeds the pipe exactly this way -- so what ends it is the
// operation timeout installed at the top of gitScanNUL, and the only thing keeping
// that from being an unbounded scan is ctx. Every other held-open test cuts on a
// grace, so nothing else exercises a scan the grace never cuts; this pins ctx as
// the bound the re-arm now leans on, and the identity of the error, since a
// listing ended by the deadline was not shown to be held open by anyone.
func TestGitScanNULTrickledListingIsBoundedByContext(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid unavailable to detach the child from the process group")
	}
	repo := gitRepo(t)
	pidFile := filepath.Join(t.TempDir(), "detached.pid")
	// The child outlives this test by design, so reap it rather than leave a trickler
	// behind on every run.
	t.Cleanup(func() { reapDetachedChild(t, pidFile) })
	// Half a grace between entries, derived from the constant so the trickle stays
	// inside the window whatever it is set to: two entries per window leaves room for
	// a loaded machine's scheduling noise before a firing could see none and cut.
	interval := fmt.Sprintf("%g", (agent.PipeDrainGrace / 2).Seconds())
	// The setsid child escapes the group kill and keeps the inherited stdout, and
	// unlike the silent holder the other tests use it keeps WRITING, so the scan makes
	// progress in every window and the grace can only ever re-arm. It publishes its PID
	// for the cleanup above and its stderr goes to /dev/null, so what is measured is the
	// stdout trickle. Each entry is printed from a subshell, whose exit flushes it, so
	// the test does not rest on a particular /bin/sh flushing its printf builtin into a
	// pipe -- block buffering would starve the scan and cut it on the first grace. The
	// loop is bounded well past the deadline below so a failed reap still ends on its
	// own, and the leader lingers a second so the kill cannot land while setsid is
	// still forking.
	shimGit(t, "ls-files", fmt.Sprintf("    printf 'f0.go\\0'\n"+
		"    setsid sh -c 'echo $$ > "+pidFile+"; i=0; while [ $i -lt 60 ]; do (printf \"t%%d.go\\0\" $i); sleep %s; i=$((i+1)); done' 2>/dev/null &\n"+
		"    sleep 1\n    exit 0", interval))

	// Three graces: the leader's one-second linger puts the first firing at three
	// seconds and the second at five, so the deadline lands after TWO re-arms -- a
	// grace that had stopped re-arming would end this scan long before it.
	deadline := 3 * agent.PipeDrainGrace
	ctx, cancel := context.WithTimeout(t.Context(), deadline)
	defer cancel()

	var seen int
	// The wall time of the last entry handed to fn, taken where the implementation
	// counts progress, so the assertion below reads the same quantity the grace does.
	var last time.Time
	c := New(config.Target{Mode: "directory", Path: repo})
	type result struct {
		err     error
		elapsed time.Duration
	}
	done := make(chan result, 1)
	start := time.Now()
	go func() {
		err := c.gitScanNUL(ctx, func(string) {
			seen++
			last = time.Now()
		}, "ls-files", "--cached", "-z")
		done <- result{err: err, elapsed: time.Since(start)}
	}()

	select {
	case got := <-done:
		// errListingHeldOpen here would mean the grace cut after all and the trickle
		// never reached the scan, which would leave the claim untested rather than
		// disproved; context.DeadlineExceeded is what proves ctx did the bounding.
		if !errors.Is(got.err, context.DeadlineExceeded) {
			t.Fatalf("gitScanNUL() err = %v, want it to wrap context.DeadlineExceeded: the trickled scan is bounded by ctx, not the grace", got.err)
		}
		// A scan starved of entries would also end at the deadline, and would look
		// identical from the outside. Entries arriving after the first firing (the
		// leader's linger plus one grace) are what prove the re-arm was exercised.
		if arrived := last.Sub(start); arrived < time.Second+agent.PipeDrainGrace {
			t.Errorf("the last entry arrived %s in, before the first grace would have fired: the scan was starved, not re-armed", arrived)
		}
		if seen < 2 {
			t.Errorf("gitScanNUL saw %d entries; the escaped writer's trickle never reached the scan", seen)
		}
		// The point of the test: bounded by the deadline, nowhere near gitOpTimeout. The
		// slack covers only joining the scan goroutine and reaping the leader.
		if got.elapsed > deadline+5*time.Second {
			t.Errorf("gitScanNUL took %s for a %s deadline; ctx no longer bounds a scan the grace keeps re-arming", got.elapsed, deadline)
		}
	case <-time.After(deadline + 30*time.Second):
		t.Fatal("gitScanNUL hung on a trickling escaped writer; the re-armed grace has no bound left and ctx does not end it")
	}
}

// joinScan's outcome mapping, tested directly because one of its two callers cannot
// reach it any other way. The cancellation path below drives it end to end, but the
// grace branch's call is only entered when scanProgressed says no, which needs an
// even counter -- fn not in flight -- so reaching that call with the scan wedged
// inside fn requires git to deliver an entry in the window between the branch's
// progress.Load and its pr.Close. A wedge that predates the firing re-arms the grace
// instead, which is the ctx-bounded path the tests below cover. So the timeout arm of
// that call has no end-to-end test on purpose, and without this one a future
// simplification back to a bare receive would leave the suite green while restoring
// an unbounded hang in Collect, with gitOpTimeout already spent as ctx and nothing
// above it left to cut.
func TestJoinScanReportsWhyTheListingEnded(t *testing.T) {
	own := errors.New("scanner's own failure")
	cut := errors.New("why the read end was closed")
	tests := []struct {
		name    string
		scanned error
		want    error
	}{
		// The one outcome the close itself manufactures, so it says nothing about the
		// listing and is replaced by the reason we closed.
		{name: "closed under it", scanned: os.ErrClosed, want: cut},
		// A scan that reached EOF in the same instant keeps its own outcome: that
		// listing IS known complete.
		{name: "already finished", scanned: nil, want: nil},
		{name: "failed on its own", scanned: own, want: own},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanned := make(chan error, 1)
			scanned <- tt.scanned
			if got := joinScan(scanned, cut); !errors.Is(got, tt.want) {
				t.Errorf("joinScan() = %v, want %v", got, tt.want)
			}
		})
	}
}

// The bound itself: a scan nobody will ever hear from -- the wedged callback, which
// the close cannot reach -- must cost one grace and then be abandoned, not held onto
// forever.
func TestJoinScanBoundsAScanThatNeverReports(t *testing.T) {
	cut := errors.New("why the read end was closed")
	done := make(chan error, 1)
	start := time.Now()
	// Never sent on, and never closed either: a closed channel would yield a nil
	// outcome and take the receive arm, which is not the arm under test.
	go func() { done <- joinScan(make(chan error), cut) }()

	select {
	case err := <-done:
		if !errors.Is(err, cut) {
			t.Errorf("joinScan() = %v, want %v once the join timed out", err, cut)
		}
		// The wait has to actually be the grace: a join that returned at once would
		// abandon a callback still capable of finishing, which is the whole reason the
		// join exists.
		if elapsed := time.Since(start); elapsed < agent.PipeDrainGrace-100*time.Millisecond {
			t.Errorf("joinScan returned after %s, sooner than the %s grace; it did not wait for the scan at all", elapsed, agent.PipeDrainGrace)
		}
	case <-time.After(agent.PipeDrainGrace + 30*time.Second):
		t.Fatal("joinScan never returned for a scan that never reports; the join is unbounded and hangs the round")
	}
}

// gitScanNUL's scan runs on its own goroutine and calls fn, which writes the
// caller's state (listGitFiles' count and builder). On the cancellation path it
// must JOIN that goroutine rather than abandon it while it still can: a caller
// that returned while fn was mid-call would be racing a goroutine still filling
// state it owns. That join is bounded -- see the test below, which pins the other
// side of the same behavior -- and the window checked here is well inside the
// bound, so what fails this test is dropping the join, not the bound.
//
// The handshake below is what pins that guarantee rather than assuming it: fn
// signals that it is genuinely in flight and then blocks, so the cancel provably
// lands mid-call. A test that merely raced a short deadline against a sleeping fn
// would pass on a worker where the deadline expired before git delivered its first
// entry -- nothing to join, no fn call to race, and an abandoned goroutine would
// look identical.
func TestGitScanNULJoinsScanGoroutineOnCancel(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "directory", Path: repo})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var inFlight atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- c.gitScanNUL(ctx, func(string) {
			inFlight.Add(1)
			defer inFlight.Add(-1)
			once.Do(func() { close(entered) })
			<-release
		}, "ls-files", "--cached", "-z")
	}()

	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("gitScanNUL returned before delivering a listing entry: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("git never delivered a listing entry to fn")
	}
	cancel()
	// The join is the claim under test: with fn still in flight, gitScanNUL must stay
	// blocked no matter that its context is already done.
	select {
	case err := <-done:
		t.Fatalf("gitScanNUL returned with fn still in flight; the scan goroutine was abandoned: %v", err)
	case <-time.After(500 * time.Millisecond):
	}
	close(release)

	select {
	case err := <-done:
		if n := inFlight.Load(); n != 0 {
			t.Errorf("gitScanNUL returned with %d fn call(s) still in flight; the scan goroutine was not joined", n)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("gitScanNUL() err = %v, want it to wrap context.Canceled", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("gitScanNUL did not return once fn was released")
	}
}

// The other side of that join, and the case the re-arming grace created: an fn that
// NEVER returns. With the progress counter bumped on both sides of the callback, an
// odd reading re-arms the grace on every firing, so a scan sitting inside fn is
// never cut -- by design, since cutting would only join the same call. ctx is the
// only bound left on it, which is exactly what gitScanNUL's own comment claims.
//
// That claim holds only if the cancellation path's join is itself bounded. Closing
// the read end ends a scan blocked on pr.Read at once, but says nothing to one
// blocked inside fn -- listGitFiles' does an Lstat and, for a symlink, an
// EvalSymlinks, either of which blocks uninterruptibly on a hard-mounted export.
// An unbounded join there would spend the last bound waiting on the very call ctx
// fired over: gitScanNUL would never return, and with gitOpTimeout already spent as
// its ctx, Collect would hang the round with nothing above it left to cut.
//
// The handshake makes the wedge deterministic rather than raced against a deadline:
// fn signals that it is in flight and then blocks forever, so the cancel provably
// lands mid-call and the goroutine is provably still in it when gitScanNUL returns.
func TestGitScanNULWedgedCallbackStillReturnsOnCancel(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "directory", Path: repo})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	entered, release := make(chan struct{}), make(chan struct{})
	// Released only once the assertions are done, so fn is wedged for the whole test
	// and the scan goroutine still ends rather than being leaked into the next one.
	t.Cleanup(func() { close(release) })
	var once sync.Once
	done := make(chan error, 1)
	go func() {
		done <- c.gitScanNUL(ctx, func(string) {
			once.Do(func() { close(entered) })
			<-release
		}, "ls-files", "--cached", "-z")
	}()

	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("gitScanNUL returned before delivering a listing entry: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("git never delivered a listing entry to fn")
	}
	cancel()

	// One grace is the bound on the join, so the return lands a grace after the
	// cancel; the slack covers reaping the leader and draining stderr behind it.
	// Anything past this is the hang, not a slow worker: the alternative the current
	// shape replaced waits on fn, which here never returns at all.
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("gitScanNUL() err = %v, want it to wrap context.Canceled: the listing ended because ctx did", err)
		}
	case <-time.After(agent.PipeDrainGrace + 30*time.Second):
		t.Fatal("gitScanNUL never returned with fn wedged; the cancellation path joins a goroutine that is inside fn, so ctx bounds nothing and the round hangs")
	}
}

// The goroutine the test above deliberately abandons is only safe if its callback
// reads nothing another goroutine rewrites. It keeps calling fn once per entry
// bufio.Scanner already buffered -- long after gitScanNUL returned and the next phase
// began -- and both streamed listings' callbacks open with skipFile, whose first
// question is whether the path is hidden. HideRunEdits answers that question from
// runFinalPhase's defer, which fires on exactly the path that abandons a listing: a
// closing pass whose Collect failed. Reading c.hidden through the receiver there is a
// concurrent map read and write, which is a FATAL runtime error rather than something
// the run can report -- at the run's last step, after the loop's commits were
// verified and before the journal's run_finished record and the summary are written.
// fileScope's snapshot is what keeps the callback off the field; this pins it, by
// clearing the field while the callback is provably still in flight and requiring the
// verdicts that follow to be the snapshot's.
func TestAbandonedScanCallbackJudgesAgainstTheScopeSnapshot(t *testing.T) {
	repo := gitRepo(t)
	// Enough entries that git's listing is sitting in the scanner's buffer when the
	// cancel lands, so the abandoned goroutine still has callbacks left to make.
	for i := range 300 {
		writeFile(t, repo, fmt.Sprintf("f%03d_test.go", i), "package main\n")
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "a listing worth buffering")

	c := New(config.Target{Mode: "directory", Path: repo})
	const ours = "f000_test.go" // a test file "this run" wrote: hidden from the closing round
	if _, err := c.HideRunEdits(map[string]bool{ours: true}, []string{"**/*_test.go"}); err != nil {
		t.Fatal(err)
	}
	scope, err := c.fileScope()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	entered, release, enough := make(chan struct{}), make(chan struct{}), make(chan struct{})
	// Released only once the field has been rewritten, so every callback after the
	// first one runs in the window the fix is about.
	var once sync.Once
	var drained, stale atomic.Int64
	done := make(chan error, 1)
	go func() {
		done <- c.gitScanNUL(ctx, func(string) {
			wedge := false
			once.Do(func() { wedge = true })
			if wedge {
				close(entered)
				<-release
				return
			}
			if !c.skipFile(ours, scope) {
				stale.Add(1)
			}
			if drained.Add(1) == 20 {
				close(enough)
			}
		}, "ls-files", "--cached", "-z")
	}()

	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("gitScanNUL returned before delivering a listing entry: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("git never delivered a listing entry to fn")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("gitScanNUL() err = %v, want it to wrap context.Canceled", err)
		}
	case <-time.After(agent.PipeDrainGrace + 30*time.Second):
		t.Fatal("gitScanNUL never returned with fn wedged; the abandonment this test is about did not happen")
	}

	// gitScanNUL has returned with its scan goroutine still inside fn -- precisely the
	// state runFinalPhase's defer runs in. Clear the hidden set the way that defer
	// does, then let the goroutine work through what it still holds.
	if _, err := c.HideRunEdits(nil, nil); err != nil {
		t.Fatal(err)
	}
	close(release)
	// Keep rewriting the field while the goroutine drains, so a callback that consulted
	// it directly is a data race the -race build reports as well as a stale verdict.
	// Neither rewrite ever hides `ours` again, so a "skip" verdict can only be the
	// snapshot's.
	for range 50 {
		if _, err := c.HideRunEdits(map[string]bool{"other.go": true}, []string{"other.go"}); err != nil {
			t.Fatal(err)
		}
		if _, err := c.HideRunEdits(nil, nil); err != nil {
			t.Fatal(err)
		}
	}

	select {
	case <-enough:
	case <-time.After(30 * time.Second):
		t.Fatalf("the abandoned scan made only %d callback(s) after the cancel; nothing was left buffered for the rewrite to race", drained.Load())
	}
	if n := stale.Load(); n != 0 {
		t.Errorf("%d abandoned callback(s) judged %s against the rewritten field instead of the scope snapshot; the callback reads state the next phase mutates", n, ours)
	}
}

// gitScanNUL hand-copies c.git's hardening (the gitenv.SafeConfigArgs -c overrides and the
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
		t.Error("the repo-configured fsmonitor program ran; the streaming listing lost gitenv.SafeConfigArgs/gitenv.Harden")
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

// The untracked tally in a --check estimate counts NUL-delimited git records, not
// whitespace-separated words and not lines: a filename containing a space or an
// embedded newline is one file, and a filename that is nothing but a space is
// still a file.
func TestScopeCountsUntrackedFilesAsRecords(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "a b.go", "package a\n")
	writeFile(t, repo, "c d.go", "package c\n")
	writeFile(t, repo, " ", "space\n")
	// Splitting records on "\n" instead of NUL would count this one twice.
	writeFile(t, repo, "e\nf.go", "package e\n")

	got, err := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD"}).Scope(t.Context())
	if err != nil {
		t.Fatalf("Scope() err = %v", err)
	}
	// Field-splitting would say 5: two words each for the spaced and newline names
	// and none at all for the file whose name is a single space. Line-splitting
	// would say 5 too, from the two halves of the newline name.
	if !strings.Contains(got, "plus 4 untracked file(s)") {
		t.Errorf("Scope() = %q, want it to report 4 untracked files", got)
	}
}

// gitScanNUL reports an uncontained git descendant -- a cleanup kill that came
// back EPERM -- through its error, and Scope's untracked tally is a real caller of
// it. Folding that failure into "no untracked files" would print a clean --check
// estimate over a target that still has a live git in it, so Scope must carry it
// out. The kill is stubbed after the real one has run (the errno needs a
// descendant under credentials this process cannot signal) and only for the
// untracked scan itself: Scope's earlier symlink listing scans with --cached and
// would otherwise fail first, leaving the tally's own error path unproven.
func TestScopeReportsFailedCleanupKillFromUntrackedScan(t *testing.T) {
	repo := gitRepo(t)
	writeFile(t, repo, "untracked.go", "package a\n")
	orig := gitCleanupKill
	t.Cleanup(func() { gitCleanupKill = orig })
	gitCleanupKill = func(cmd *exec.Cmd) error {
		err := orig(cmd)
		if slices.Contains(cmd.Args, "--others") && !slices.Contains(cmd.Args, "--cached") {
			return syscall.EPERM
		}
		return err
	}

	got, err := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD"}).Scope(t.Context())
	if err == nil {
		t.Fatalf("Scope() = %q, nil after a group kill that reported an uncontained git; want the failure surfaced", got)
	}
	if !errors.Is(err, syscall.EPERM) {
		t.Errorf("Scope() err = %v, want it to carry the kill's EPERM", err)
	}
}

// The closing round must not be shown the test files the run itself just wrote:
// review-tests then reports on its own output ("this assertion claims more than it
// proves"), which is a loop with no fixed point. HideRunEdits is what narrows it,
// and it must narrow ONLY that -- a matching file the run did not touch, and a
// run-written file that does not match, both stay in scope.
func TestHideRunEditsHidesOnlyTheRunsOwnMatchingFiles(t *testing.T) {
	// BOTH collection paths, because they are different code -- listGitFiles asks git
	// for the listing, walkFiles walks the filesystem -- and their parity is exactly
	// what broke here once: walkFiles matched the exclude globs directly instead of
	// going through skipFile, so the hidden set applied on the git path only. Nothing
	// end-to-end can catch a regression either, which is why the parity is pinned
	// here: resolveRunBase needs a HEAD, so a non-repository target never gets a
	// runBase and the orchestrator's hideRunEdits returns before narrowing anything.
	for _, tc := range []struct {
		name string
		tree func(*testing.T) string
	}{
		{"git", gitRepo},
		// A plain directory, so IsGitRepo is false and listFiles takes walkFiles.
		{"walk", func(t *testing.T) string { t.Helper(); return t.TempDir() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.tree(t)
			writeFile(t, dir, "main.go", "package main\n")     // untouched source
			writeFile(t, dir, "old_test.go", "package main\n") // matches, but not ours
			writeFile(t, dir, "new_test.go", "package main\n") // matches and ours
			writeFile(t, dir, "helper.go", "package main\n")   // ours, does not match
			c := New(config.Target{Mode: "directory", Path: dir})
			if err := c.Prepare(t.Context()); err != nil {
				t.Fatal(err)
			}

			// Exactly what the orchestrator passes: the paths this run's commits changed.
			changed := map[string]bool{"new_test.go": true, "helper.go": true}
			hidden, err := c.HideRunEdits(changed, []string{"**/*_test.go"})
			if err != nil {
				t.Fatal(err)
			}
			if len(hidden) != 1 || hidden[0] != "new_test.go" {
				t.Fatalf("HideRunEdits() = %v, want exactly [new_test.go]", hidden)
			}

			material, err := c.Collect(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(material, "new_test.go") {
				t.Errorf("a test file this run wrote is still in the closing round's material:\n%s", material)
			}
			for _, want := range []string{"old_test.go", "helper.go", "main.go"} {
				if !strings.Contains(material, want) {
					t.Errorf("Collect() dropped %q, which HideRunEdits must not touch:\n%s", want, material)
				}
			}

			// Cleared again: the collector outlives the closing phase, and a hidden set
			// left behind would silently narrow every later collection and clean check.
			if _, err := c.HideRunEdits(nil, nil); err != nil {
				t.Fatal(err)
			}
			material, err = c.Collect(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(material, "new_test.go") {
				t.Errorf("clearing HideRunEdits did not restore full scope:\n%s", material)
			}
		})
	}
}

// The same narrowing has to reach git-diff mode, where it matters more: that mode
// embeds full file CONTENT in the prompt, so an unhidden file costs tokens as well
// as attention. The pathspec is where it happens, and a concrete path must be
// excluded literally -- a filename containing pathspec metacharacters would
// otherwise be reinterpreted as a glob and quietly stay in the diff.
func TestHideRunEditsAppliesToTheGitDiffPathspec(t *testing.T) {
	repo := gitRepo(t)
	base := strings.TrimSpace(git(t, repo, "rev-parse", "HEAD"))
	writeFile(t, repo, "x_test.go", "package main\n// written by this run\n")
	writeFile(t, repo, "x.go", "package main\n// also written by this run\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "the run's own work")

	c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: base})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.HideRunEdits(map[string]bool{"x_test.go": true}, []string{"**/*_test.go"}); err != nil {
		t.Fatal(err)
	}
	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(material, "x_test.go") {
		t.Errorf("the diff still carries a test file this run wrote:\n%s", material)
	}
	if !strings.Contains(material, "x.go") {
		t.Errorf("the diff lost the non-test file this run wrote, which stays in scope:\n%s", material)
	}
}

// A diff answers "is this internally consistent"; it cannot answer "does this do
// what it was for", because the intent is not in it. Commit messages carry the
// author's reasoning per step, which for many projects is the only place a
// decision is written down at all -- and every review before this was blind to it.
func TestCollectCarriesTheCommitMessagesOfTheChangesUnderReview(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD"})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	writeFile(t, repo, "main.go", "package main\n\nfunc changed() {}\n")
	git(t, repo, "commit", "-aqm", "guard the nil deref\n\nThe caller can pass nil once the retry lands, and the guard\nis cheaper than the branch it replaces.")

	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Commit messages of the changes under review",
		"guard the nil deref",
		"cheaper than the branch it replaces", // the BODY, where the reason lives
	} {
		if !strings.Contains(material, want) {
			t.Errorf("Collect() is missing %q:\n%s", want, material)
		}
	}
	// The diff is still the material; the context sits ahead of it.
	if i, j := strings.Index(material, "Commit messages"), strings.Index(material, "Diff against pinned base"); i < 0 || j < 0 || i > j {
		t.Errorf("intent should precede the diff (at %d and %d)", i, j)
	}
}

// The commit list is derived from a HEAD this run MOVES: the orchestrator commits
// each accepted fix, so round 2's diff contains a commit round 1's never saw. A
// list cached at round 1 would sit under a heading claiming to quote the changes
// under review while omitting one of them, with nothing saying so.
func TestCollectRereadsTheCommitMessagesAfterTheRunCommits(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD"})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	writeFile(t, repo, "main.go", "package main\n\nfunc changed() {}\n")
	git(t, repo, "commit", "-aqm", "the change under review")
	if _, err := c.Collect(t.Context()); err != nil { // round 1
		t.Fatal(err)
	}
	writeFile(t, repo, "main.go", "package main\n\nfunc changed() { fixed() }\n")
	git(t, repo, "commit", "-aqm", "fixpoint: i1 -- the round's own fix")

	material, err := c.Collect(t.Context()) // round 2
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"the change under review", "the round's own fix"} {
		if !strings.Contains(material, want) {
			t.Errorf("the commit list is stale, missing %q:\n%s", want, material)
		}
	}
}

// Nothing to say is not a failure. A repository with no commits since the base,
// no gh, or no network simply has no context to add -- which is the state every
// run before this was in, and a round must still work there.
func TestCollectWithoutAnyIntentIsJustTheDiff(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD"})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	writeFile(t, repo, "main.go", "package main\n\nfunc changed() {}\n")

	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(material, "Commit messages") {
		t.Errorf("no commits were made, so nothing should claim to quote them:\n%s", material)
	}
	if !strings.Contains(material, "func changed()") {
		t.Errorf("the diff itself is missing:\n%s", material)
	}
}

// A description is free text somebody else writes and a branch can carry hundreds
// of commits; neither may crowd out the diff, which is the thing being reviewed.
// The cap is STATED, because a reader who cannot tell a truncated description from
// a short one reads the missing half as absent.
func TestClampIntentSaysWhatItDropped(t *testing.T) {
	var b strings.Builder
	for i := range 500 {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	got := clampIntent(b.String(), intentLines, intentBytes)
	if strings.Contains(got, "line 450") {
		t.Error("the clamp did not apply")
	}
	if !strings.Contains(got, "not shown") {
		t.Errorf("a silent truncation reads as a short description:\n%s", got[len(got)-200:])
	}
	if short := "one\ntwo\n"; clampIntent(short, intentLines, intentBytes) != short {
		t.Error("a short text must pass through untouched, with nothing claimed to be missing")
	}
}

// A line cap is not a size cap: git imposes no limit on a commit message, so one
// pasted line passes 400 lines untouched. The intent is written AHEAD of the diff
// and the material is head-truncated, so an intent bounded only by lines would
// take the whole budget and leave the reviewer prose and no code -- which reads as
// a clean round on something nobody saw.
func TestClampIntentBoundsBytesNotJustLines(t *testing.T) {
	oneLine := strings.Repeat("x", maxMaterial)
	got := clampIntent(oneLine, intentLines, intentBytes)
	if len(got) > intentBytes+200 {
		t.Errorf("a single %d-byte line survived the byte cap: got %d bytes", len(oneLine), len(got))
	}
	if !strings.Contains(got, "not shown") {
		t.Errorf("the byte cut must be stated, not silent:\n%s", got[max(0, len(got)-200):])
	}
	// A cut that lands mid-character would put invalid UTF-8 into every prompt.
	if wide := strings.Repeat("é", intentBytes); !utf8.ValidString(clampIntent(wide, intentLines, intentBytes)) {
		t.Error("the byte cap cut inside a multibyte character")
	}
}

// The end-to-end shape of the same thing: an oversized commit message must cost
// itself, never the diff.
func TestCollectKeepsTheDiffWhenTheIntentIsOversized(t *testing.T) {
	repo := gitRepo(t)
	c := New(config.Target{Mode: "git-diff", Path: repo, BaseRef: "HEAD"})
	if err := c.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	writeFile(t, repo, "main.go", "package main\n\nfunc changed() {}\n")
	// Well over maxMaterial, on far fewer than intentLines lines. Passed as a file
	// because a single argument that size exceeds what exec will carry -- which is
	// the only thing stopping it, and no bound git itself imposes.
	msg := filepath.Join(t.TempDir(), "msg.txt")
	if err := os.WriteFile(msg, []byte("a subject\n\n"+strings.Repeat("A", maxMaterial+1_000)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "commit", "-aq", "-F", msg)

	material, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Diff against pinned base", "func changed()"} {
		if !strings.Contains(material, want) {
			t.Errorf("the intent evicted %q from the material (%d bytes collected)", want, len(material))
		}
	}
}
