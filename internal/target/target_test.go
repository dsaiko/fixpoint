package target

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
		{"?.go", "a.go", true},
		{"?.go", "ab.go", false},
		{"**/café.go", "src/café.go", true}, // multi-byte literals
		{".github/**/*.yaml", ".github/workflows/ci.yaml", true},
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

// ghRemote must compare a remote's OWNER/REPO identity for equality, not look for
// the desired repository as a substring of the URL: acme/widget is a substring of
// acme/widget-fork and acme/widgets, and if such a remote sorts first the PR base
// OID would be fetched from the wrong repository -- failing even though a correct
// remote is configured.
func TestRemoteIdentity(t *testing.T) {
	cases := []struct{ url, want string }{
		{"https://github.com/acme/widget.git", "acme/widget"},
		{"https://github.com/acme/widget", "acme/widget"},
		{"https://github.com/Acme/Widget.git\n", "acme/widget"},
		{"https://token:x-oauth-basic@github.com/acme/widget.git", "acme/widget"},
		{"http://ghe.example.com/acme/widget.git", "acme/widget"},
		{"ssh://git@github.com:22/acme/widget.git", "acme/widget"},
		{"git@github.com:acme/widget.git", "acme/widget"},
		{"git@github.com:acme/widget", "acme/widget"},
		// The near misses a substring test would accept.
		{"https://github.com/acme/widget-fork.git", "acme/widget-fork"},
		{"https://github.com/acme/widgets.git", "acme/widgets"},
		{"git@github.com:acme/widget-fork.git", "acme/widget-fork"},
		// Not an owner/repo remote at all: no identity to compare, so ghRemote falls
		// back to origin rather than matching by accident.
		{"/srv/git/widget.git", ""},
		{"file:///srv/git/acme/widget.git", ""},
		{"https://github.com/acme", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := remoteIdentity(tc.url); got != tc.want {
			t.Errorf("remoteIdentity(%q) = %q, want %q", tc.url, got, tc.want)
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
