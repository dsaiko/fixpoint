package implement

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// censusRepo builds a repo with one committed file and a .gitignore covering
// ignored/ -- the smallest tree that exercises every census bucket.
func censusRepo(t *testing.T) string {
	t.Helper()
	gitAvailable(t)
	out := filepath.Join(t.TempDir(), "repo")
	col := collectorFor(t, out)
	files := map[string][]byte{
		"tracked.txt": []byte("v1\n"),
		".gitignore":  []byte(GitignoreContent([]string{"ignored/"})),
	}
	_, release, err := Scaffold(t.Context(), col, out, files, "init", "-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(release)
	return out
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTakeCensus(t *testing.T) {
	dir := censusRepo(t)
	write(t, dir, "new.txt", "fresh\n")
	write(t, dir, "tracked.txt", "v2\n")
	write(t, dir, "ignored/cache.bin", "blob\n")

	c, err := TakeCensus(t.Context(), dir, 0)
	if err != nil {
		t.Fatalf("TakeCensus() = %v", err)
	}
	if _, ok := c.Untracked["new.txt"]; !ok {
		t.Errorf("new.txt not censused as untracked: %+v", c)
	}
	if _, ok := c.Modified["tracked.txt"]; !ok {
		t.Errorf("tracked.txt not censused as modified: %+v", c)
	}
	if _, ok := c.Untracked["ignored/cache.bin"]; ok {
		t.Error("an ignored path entered the un-ignored census")
	}
	if c.Bytes <= 0 {
		t.Errorf("Bytes = %d", c.Bytes)
	}

	// A deletion is a recorded state: empty digest, not absence.
	os.Remove(filepath.Join(dir, "tracked.txt"))
	c, err = TakeCensus(t.Context(), dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := c.Modified["tracked.txt"]; !ok || d != "" {
		t.Errorf("deleted tracked file censused as %q, %v", d, ok)
	}
}

// The bound trips on sizes, before hashing, and names the whole limit.
func TestCensusByteBound(t *testing.T) {
	dir := censusRepo(t)
	write(t, dir, "big.txt", strings.Repeat("x", 4096))
	_, err := TakeCensus(t.Context(), dir, 1024)
	var bb ByteBoundError
	if !errors.As(err, &bb) {
		t.Fatalf("want ByteBoundError, got %v", err)
	}
	if bb.Limit != 1024 || bb.Bytes < 4096 {
		t.Errorf("bound error = %+v", bb)
	}
}

func TestIgnoredCensusAndDiff(t *testing.T) {
	dir := censusRepo(t)
	write(t, dir, "ignored/old.bin", "old\n")
	write(t, dir, ".fixpoint/journal.jsonl", "{}\n")
	pre, err := TakeIgnoredCensus(t.Context(), dir)
	if err != nil {
		t.Fatalf("TakeIgnoredCensus() = %v", err)
	}
	if _, ok := pre["ignored/old.bin"]; !ok {
		t.Errorf("ignored file missing from the census: %v", pre)
	}
	for path := range pre {
		if strings.HasPrefix(path, ".fixpoint/") {
			t.Errorf("the artifact root leaked into the ignored census: %s", path)
		}
	}

	write(t, dir, "ignored/new.bin", "new\n")
	// A same-size in-place rewrite with a bumped mtime is build churn: reported.
	future := time.Now().Add(time.Hour)
	write(t, dir, "ignored/old.bin", "OLD\n")
	os.Chtimes(filepath.Join(dir, "ignored", "old.bin"), future, future)

	post, err := TakeIgnoredCensus(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	created, modified := DiffIgnored(pre, post)
	if !reflect.DeepEqual(created, []string{"ignored/new.bin"}) {
		t.Errorf("created = %v", created)
	}
	if !reflect.DeepEqual(modified, []string{"ignored/old.bin"}) {
		t.Errorf("modified = %v", modified)
	}
}

// The three buckets of §5.2 step 7: mutation fails the task, gate_generated
// commits with attribution, output is removed. Pure function, no git.
func TestClassifyGateDiff(t *testing.T) {
	pre := Census{
		Untracked: map[string]string{"src/new.go": "aaa"},
		Modified:  map[string]string{"main.go": "bbb"},
	}
	post := Census{
		Untracked: map[string]string{
			"src/new.go": "MUTATED", // gate rewrote the coder's new file
			"go.sum":     "ccc",     // gate-maintained committed file
			"out.bin":    "ddd",     // build output
		},
		Modified: map[string]string{
			"main.go":  "bbb", // untouched
			"other.go": "eee", // gate modified a tracked file the coder did not touch
		},
	}
	d := ClassifyGateDiff(pre, post, []string{"go.sum"})
	if !reflect.DeepEqual(d.MutatedSources, []string{"other.go", "src/new.go"}) {
		t.Errorf("MutatedSources = %v", d.MutatedSources)
	}
	if !reflect.DeepEqual(d.GateGenerated, []string{"go.sum"}) {
		t.Errorf("GateGenerated = %v", d.GateGenerated)
	}
	if !reflect.DeepEqual(d.Output, []string{"out.bin"}) {
		t.Errorf("Output = %v", d.Output)
	}

	// A gate updating a gate_generated file IN PLACE is attribution, not mutation.
	pre2 := Census{Untracked: map[string]string{"go.sum": "v1"}, Modified: map[string]string{}}
	post2 := Census{Untracked: map[string]string{"go.sum": "v2"}, Modified: map[string]string{}}
	d2 := ClassifyGateDiff(pre2, post2, []string{"go.sum"})
	if len(d2.MutatedSources) != 0 || !reflect.DeepEqual(d2.GateGenerated, []string{"go.sum"}) {
		t.Errorf("in-place lockfile rewrite = %+v", d2)
	}

	// The gate deleting a coder file is mutation.
	d3 := ClassifyGateDiff(Census{Untracked: map[string]string{"a.go": "x"}}, Census{Untracked: map[string]string{}}, nil)
	if !reflect.DeepEqual(d3.MutatedSources, []string{"a.go"}) {
		t.Errorf("deletion = %+v", d3)
	}
}
