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

	c, err := testGit.TakeCensus(t.Context(), dir, 0)
	if err != nil {
		t.Fatalf("testGit.TakeCensus() = %v", err)
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
	c, err = testGit.TakeCensus(t.Context(), dir, 0)
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
	_, err := testGit.TakeCensus(t.Context(), dir, 1024)
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
	pre, err := testGit.TakeIgnoredCensus(t.Context(), dir)
	if err != nil {
		t.Fatalf("testGit.TakeIgnoredCensus() = %v", err)
	}
	if _, ok := pre["ignored/old.bin"]; !ok {
		t.Errorf("ignored file missing from the census: %v", pre)
	}
	for path := range pre {
		if path == ".fixpoint" || strings.HasPrefix(path, ".fixpoint/") {
			t.Errorf("the artifact root leaked into the ignored census: %s", path)
		}
	}

	write(t, dir, "ignored/new.bin", "new\n")
	// A same-size in-place rewrite with a bumped mtime is build churn: reported.
	future := time.Now().Add(time.Hour)
	write(t, dir, "ignored/old.bin", "OLD\n")
	os.Chtimes(filepath.Join(dir, "ignored", "old.bin"), future, future)

	post, err := testGit.TakeIgnoredCensus(t.Context(), dir)
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

// Review run 20260929-141502, i7: logs.dir is configurable, and a run whose
// logs live under an ignored runlogs/ in the project censused its own files --
// the prompts an attempt logs after step 2 read as created, and every discard
// deleted them.
func TestIgnoredCensusExcludesTheNamedArtifactRoots(t *testing.T) {
	dir := censusRepo(t)
	write(t, dir, ".git/info/exclude", "runlogs/\n")
	write(t, dir, "runlogs/20260929/journal.jsonl", "{}\n")
	pre, err := testGit.TakeIgnoredCensus(t.Context(), dir, "runlogs")
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, "runlogs/20260929/round-1/coder-prompt.md", "logged after step 2\n")
	write(t, dir, "ignored/new.bin", "the attempt's\n")
	post, err := testGit.TakeIgnoredCensus(t.Context(), dir, "runlogs")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []map[string]IgnoredStat{pre, post} {
		for path := range c {
			if path == "runlogs" || strings.HasPrefix(path, "runlogs/") {
				t.Errorf("the configured logs root leaked into the ignored census: %s", path)
			}
		}
	}
	if created, _ := DiffIgnored(pre, post); !reflect.DeepEqual(created, []string{"ignored", "ignored/new.bin"}) {
		t.Errorf("created = %v, want only the attempt's own output", created)
	}
}

// Directories enter the ignored census so a discard can delete the ones an
// attempt created (review run 20260929-125352, i4) -- and ONLY those: a
// directory that existed at step 2 is kept, and one holding tracked files is
// never listed, however ignored its contents, since deleting it as "created"
// would take the sources with it.
func TestIgnoredCensusRecordsDirectories(t *testing.T) {
	dir := censusRepo(t)
	write(t, dir, "src/main.go", "package main\n")
	if _, err := testGit.run(t.Context(), dir, "add", "src/main.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := testGit.run(t.Context(), dir, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "src"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, ".git/info/exclude", "*.log\n")
	write(t, dir, "ignored/pkg/old.bin", "old\n")
	if err := os.MkdirAll(filepath.Join(dir, "ignored", "keep-empty"), 0o750); err != nil {
		t.Fatal(err)
	}
	pre, err := testGit.TakeIgnoredCensus(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}

	write(t, dir, "ignored/pkg/sub/new.bin", "new\n") // new dir under a pre-existing one
	write(t, dir, "ignored/fresh/deep/a.bin", "a\n")  // new nested dirs
	if err := os.MkdirAll(filepath.Join(dir, "ignored", "fresh", "empty"), 0o750); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "logs-only/run.log", "l\n") // a directory git reports as wholly ignored
	write(t, dir, "src/debug.log", "l\n")     // an ignored file in a TRACKED directory
	post, err := testGit.TakeIgnoredCensus(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	created, modified := DiffIgnored(pre, post)
	want := []string{
		"ignored/fresh", "ignored/fresh/deep", "ignored/fresh/deep/a.bin",
		"ignored/pkg/sub", "ignored/pkg/sub/new.bin",
		"logs-only", "logs-only/run.log",
		"src/debug.log",
	}
	if !reflect.DeepEqual(created, want) {
		t.Errorf("created = %v\nwant      %v", created, want)
	}
	if len(modified) != 0 {
		t.Errorf("modified = %v; a directory's mtime is churn, not a modification", modified)
	}
	// Recorded at step 2, so the diff above keeps them rather than never seeing
	// them; ignored/keep-empty is kept by never being listed at all.
	for _, kept := range []string{"ignored", "ignored/pkg"} {
		if !pre[kept].Dir {
			t.Errorf("%s is missing from the first census as a directory: %v", kept, pre)
		}
	}
	if _, ok := post["src"]; ok {
		t.Error("src/ holds a tracked file and entered the ignored census, where a discard could delete it")
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

// A gate that regenerates a tracked file the coder DELETED, byte-identical to
// HEAD, drops the path out of the post census entirely. That absence means
// "matches HEAD", not the "" a deletion digests to, so it must read as a
// mutation -- otherwise the commit stages a HEAD-identical path and the
// coder's deletion vanishes from an "implemented" task. Real git on both sides,
// because the bug lived in how the two censuses spell the two states.
func TestClassifyGateDiffRecreatedDeletion(t *testing.T) {
	dir := censusRepo(t)
	if err := os.Remove(filepath.Join(dir, "tracked.txt")); err != nil {
		t.Fatal(err)
	}
	pre, err := testGit.TakeCensus(t.Context(), dir, 0)
	if err != nil {
		t.Fatal(err)
	}

	// The deletion survives the gate: no difference to report.
	same, err := testGit.TakeCensus(t.Context(), dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if d := ClassifyGateDiff(pre, same, nil); len(d.MutatedSources)+len(d.GateGenerated)+len(d.Output) != 0 {
		t.Errorf("an untouched deletion was classified as a difference: %+v", d)
	}

	// The gate recreates it exactly as HEAD has it.
	write(t, dir, "tracked.txt", "v1\n")
	post, err := testGit.TakeCensus(t.Context(), dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := post.Modified["tracked.txt"]; ok {
		t.Fatalf("precondition: a HEAD-identical file should be absent from the census: %+v", post)
	}
	if d := ClassifyGateDiff(pre, post, nil); !reflect.DeepEqual(d.MutatedSources, []string{"tracked.txt"}) {
		t.Errorf("gate recreating a deleted tracked file: MutatedSources = %v, want [tracked.txt]", d.MutatedSources)
	}
	// A gate_generated path recreated this way is the gate's, attributed.
	if d := ClassifyGateDiff(pre, post, []string{"tracked.txt"}); !reflect.DeepEqual(d.GateGenerated, []string{"tracked.txt"}) || len(d.MutatedSources) != 0 {
		t.Errorf("recreated gate_generated path = %+v", d)
	}
}
