package implement

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
)

// testGit is the handle the package's tests run git through. Deliberately the
// ZERO value: the hardening the tests below assert must hold for a Git nobody
// configured, because that is the shape a future caller will reach for.
var testGit Git

// filterRepo builds a repository that names a clean/smudge filter in its own
// .gitattributes -- the bytes a coder session can write -- and returns it with
// the canary path the filter would write to if it ever ran.
//
// The filter is a script FILE rather than an inline command because git's config
// parser treats `;` and `"` in a value as its own syntax, and what is being
// tested is execution, not quoting.
func filterRepo(t *testing.T) (repo, canary string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ANTHROPIC_API_KEY", "sk-secret-value")

	work := t.TempDir()
	canary = filepath.Join(work, "exfiltrated")
	filter := filepath.Join(work, "filter.sh")
	script := "#!/bin/sh\nprintf '%s' \"${ANTHROPIC_API_KEY:-none}\" > " + canary + "\ncat\n"
	if err := os.WriteFile(filter, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	// The operator's global config, which is where a filter with a dynamic name
	// can be defined: git offers no way to pin `filter.<name>.smudge` off by key.
	conf := "[filter \"evil\"]\n\tclean = " + filter + "\n\tsmudge = " + filter + "\n"
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}

	repo = filepath.Join(work, "project")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, repo, ".gitattributes", "*.txt filter=evil\n")
	write(t, repo, "src.txt", "work\n")
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "seed"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		// Built the same way run() builds it, so the fixture itself cannot be the
		// thing that executes the filter while setting the scene.
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	// Staged through the handle under test: `git add` applies the CLEAN filter.
	if _, err := testGit.run(t.Context(), repo, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := testGit.run(t.Context(), repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "seed files"); err != nil {
		t.Fatal(err)
	}
	return repo, canary
}

// The read path's git must be hardened exactly as the write path's is. Two of
// its calls WRITE a worktree -- CleanCheck's `git clone` and
// RestoreControlArtifacts' `git checkout` -- and a checkout applies the smudge
// filter that the tree's .gitattributes names. The coder authors that file; the
// definition it needs can only come from the operator's global config; so with
// global config off there is nowhere left to define one.
//
// Measured before the fix (review run 20260813-180828, i12): the read path ran
// with gitenv.Harden(nil), i.e. the full process environment and the operator's
// global config still in effect, and the clone executed the filter with
// fixpoint's credentials in its environment.
func TestReadPathDoesNotRunGloballyDefinedGitFilters(t *testing.T) {
	repo, canary := filterRepo(t)

	// The clone: checks out every file, so every smudge filter fires.
	scratch := t.TempDir()
	if _, err := testGit.CleanCheck(t.Context(), repo, scratch, config.Verify{Policy: config.VerifyOff}, nil); err != nil {
		t.Fatalf("CleanCheck() = %v", err)
	}
	if b, err := os.ReadFile(canary); err == nil {
		t.Fatalf("the clean-check clone executed a globally-defined git filter, which captured %q", strings.TrimSpace(string(b)))
	}

	// The checkout: same class, different call.
	write(t, repo, "src.txt", "tampered\n")
	if err := testGit.RestoreControlArtifacts(t.Context(), repo, "HEAD", []string{"src.txt"}); err != nil {
		t.Fatalf("RestoreControlArtifacts() = %v", err)
	}
	if b, err := os.ReadFile(canary); err == nil {
		t.Fatalf("the control-artifact checkout executed a globally-defined git filter, which captured %q", strings.TrimSpace(string(b)))
	}
}

// The plumbing reads carry the same environment, so a filter cannot reach them
// either -- and the read path must not carry credentials it has no use for.
func TestReadPathCarriesNoCredentials(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-secret-value")
	g := NewGit([]string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir()})
	for _, e := range g.env {
		if strings.HasPrefix(e, "ANTHROPIC_API_KEY=") {
			t.Fatal("the read path was handed a credential")
		}
	}
}
