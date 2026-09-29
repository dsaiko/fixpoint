package implement

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func commitAll(t *testing.T, dir, msg string) {
	t.Helper()
	if _, err := testGit.run(t.Context(), dir, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := testGit.run(t.Context(), dir, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", msg); err != nil {
		t.Fatal(err)
	}
}

func mustExist(t *testing.T, path, why string) {
	t.Helper()
	if _, err := os.Lstat(path); err != nil {
		t.Errorf("%s: %v", why, err)
	}
}

// Review run 20260929-133423, i7: certs/ holds a tracked README and the
// operator's ignored dev.key. Once the task deletes the README and stages it,
// git reports certs/ wholly ignored -- a directory the first census never
// listed -- and the step 6 RemoveAll that followed took dev.key with it.
func TestDiffIgnoredNeverCreatesADirectoryHoldingFirstCensusFiles(t *testing.T) {
	dir := censusRepo(t)
	write(t, dir, ".git/info/exclude", "*.key\n")
	write(t, dir, "certs/README.md", "certs\n")
	commitAll(t, dir, "certs")
	write(t, dir, "certs/dev.key", "operator secret\n")
	pre, err := testGit.TakeIgnoredCensus(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testGit.run(t.Context(), dir, "rm", "-q", "certs/README.md"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "certs/new.key", "attempt's\n")
	post, err := testGit.TakeIgnoredCensus(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !post["certs"].Dir {
		t.Fatalf("git no longer reports the emptied certs/ as ignored, so this proves nothing: %v", post)
	}
	created, _ := DiffIgnored(pre, post)
	if want := []string{"certs/new.key"}; !slices.Equal(created, want) {
		t.Errorf("created = %v, want %v: certs/ holds a file that was there before the attempt", created, want)
	}
	if left := RemoveCreated(dir, created, post); len(left) != 0 {
		t.Errorf("left = %v", left)
	}
	mustExist(t, filepath.Join(dir, "certs", "dev.key"), "the operator's pre-existing ignored file was deleted")
}

// Deletion is never recursive: a created directory still holding something
// neither census saw is reported and left, with its bytes, while the empty
// directories git never lists are swept.
func TestRemoveCreatedNeverDeletesUncensusedBytes(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "out/new.bin", "new\n")
	write(t, dir, "out/unseen/precious", "not in any census\n")
	if err := os.MkdirAll(filepath.Join(dir, "out", "empty", "deeper"), 0o750); err != nil {
		t.Fatal(err)
	}
	census := map[string]IgnoredStat{"out": {Dir: true}, "out/new.bin": {}}
	left := RemoveCreated(dir, []string{"out", "out/new.bin"}, census)
	if len(left) != 1 || !strings.HasPrefix(left[0], "out (left in place") {
		t.Errorf("left = %v, want out reported as left in place", left)
	}
	mustExist(t, filepath.Join(dir, "out", "unseen", "precious"), "an uncensused file under a created directory was deleted")
	if _, err := os.Lstat(filepath.Join(dir, "out", "new.bin")); !os.IsNotExist(err) {
		t.Errorf("the created file survived: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "out", "empty")); !os.IsNotExist(err) {
		t.Errorf("an empty directory beneath a created one survived: %v", err)
	}
}

// Review run 20260929-133423, i11: a parent swapped for a symlink after the
// census. RemoveAll resolved it and deleted what it named. Both an escape and an
// in-repo redirect are refused, and nothing either one names is touched.
func TestRemoveCreatedRefusesASymlinkedParent(t *testing.T) {
	for _, tc := range []struct{ name, target string }{
		{"outside the repository", filepath.Join(t.TempDir(), "elsewhere")},
		{"inside the repository", "keep"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			victim := tc.target
			if !filepath.IsAbs(victim) {
				victim = filepath.Join(dir, victim)
			}
			write(t, victim, "sub/precious", "not the attempt's\n")
			if err := os.Symlink(tc.target, filepath.Join(dir, "ignored")); err != nil {
				t.Fatal(err)
			}
			census := map[string]IgnoredStat{"ignored/sub": {Dir: true}, "ignored/sub/precious": {}}
			left := RemoveCreated(dir, []string{"ignored/sub", "ignored/sub/precious"}, census)
			if len(left) != 2 || !strings.Contains(left[0], "is a symlink") {
				t.Errorf("left = %v, want both paths refused over the symlink", left)
			}
			mustExist(t, filepath.Join(victim, "sub", "precious"), "a file the symlinked parent named was deleted")
		})
	}
}

// A path whose type is not the censused one is not the thing the census saw,
// so it is refused rather than removed -- a symlink where a directory was, an
// empty directory where a file was.
func TestRemoveCreatedRefusesAChangedType(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "elsewhere")
	write(t, outside, "precious", "x\n")
	if err := os.Symlink(outside, filepath.Join(dir, "was-dir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "was-file"), 0o750); err != nil {
		t.Fatal(err)
	}
	census := map[string]IgnoredStat{"was-dir": {Dir: true}, "was-file": {}}
	left := RemoveCreated(dir, []string{"was-dir", "was-file"}, census)
	if len(left) != 2 {
		t.Errorf("left = %v, want both refused", left)
	}
	for _, p := range []string{filepath.Join(dir, "was-dir"), filepath.Join(dir, "was-file"), filepath.Join(outside, "precious")} {
		mustExist(t, p, "a path whose type changed since the census was removed")
	}
}

// Review run 20260929-133423, i3: an ignored file the attempt replaced with a
// directory diffed as "modified", so the discard removed the new children and
// left the directory for the next gate's `test -d` to pass on.
func TestDiffIgnoredFileReplacedByADirectoryIsCreated(t *testing.T) {
	dir := censusRepo(t)
	write(t, dir, ".git/info/exclude", "scratch\n")
	write(t, dir, "scratch", "a cache file\n")
	pre, err := testGit.TakeIgnoredCensus(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "scratch")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "scratch/sub/f", "the attempt's\n")
	post, err := testGit.TakeIgnoredCensus(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	created, modified := DiffIgnored(pre, post)
	if !slices.Contains(created, "scratch") || slices.Contains(modified, "scratch") {
		t.Fatalf("created = %v, modified = %v: a file replaced by a directory is a replacement", created, modified)
	}
	if left := RemoveCreated(dir, created, post); len(left) != 0 {
		t.Errorf("left = %v", left)
	}
	if _, err := os.Lstat(filepath.Join(dir, "scratch")); !os.IsNotExist(err) {
		t.Errorf("the attempt's directory survived: %v", err)
	}
}
