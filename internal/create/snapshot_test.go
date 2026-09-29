package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// The property everything else rests on: the phases read the run's own bytes.
// Editing the source after the snapshot must change nothing an agent sees --
// the specification's first draft pinned hashes instead, and its own review
// called that drift detection, not a snapshot.
func TestSnapshotFreezesTheBytes(t *testing.T) {
	src := t.TempDir()
	p := write(t, src, "assignment.md", "design a card game\n")

	snap, err := Snapshot(p, filepath.Join(t.TempDir(), "assignment"), nil, 1<<20)
	if err != nil {
		t.Fatalf("Snapshot() = %v", err)
	}
	if err := os.WriteFile(p, []byte("design a DIFFERENT game\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(snap.Dir, snap.Entry))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "design a card game\n" {
		t.Errorf("the snapshot moved with its source: %q", got)
	}
	if snap.Files != 1 || snap.Bytes != int64(len("design a card game\n")) {
		t.Errorf("reported %d file(s), %d byte(s)", snap.Files, snap.Bytes)
	}
}

// Exclusions are structural: what is not copied cannot be read. A rerun's own
// deliverable, the tool's logs, and VCS metadata are not the assignment.
func TestSnapshotExcludesTheDeliverableLogsAndGit(t *testing.T) {
	src := t.TempDir()
	write(t, src, "brief.md", "the brief")
	write(t, src, "notes/context.md", "context")
	out := write(t, src, "DESIGN.md", "a previous deliverable")
	write(t, src, ".fixpoint/run/journal.jsonl", "{}")
	write(t, src, ".git/config", "[core]")

	snap, err := Snapshot(src, filepath.Join(t.TempDir(), "assignment"), []string{out}, 1<<20)
	if err != nil {
		t.Fatalf("Snapshot() = %v", err)
	}
	for _, absent := range []string{"DESIGN.md", ".fixpoint/run/journal.jsonl", ".git/config"} {
		if _, err := os.Stat(filepath.Join(snap.Dir, absent)); !os.IsNotExist(err) {
			t.Errorf("%s reached the snapshot; a rerun would design against its own previous answer", absent)
		}
	}
	for _, present := range []string{"brief.md", "notes/context.md"} {
		if _, err := os.Stat(filepath.Join(snap.Dir, present)); err != nil {
			t.Errorf("%s missing from the snapshot: %v", present, err)
		}
	}
	if snap.Files != 2 {
		t.Errorf("Files = %d, want 2", snap.Files)
	}
}

// A symlink is neither followed nor replicated -- following one ingests files
// outside the assignment, replicating one lets the snapshot read outside itself
// later. It is counted, because a snapshot that silently dropped part of the
// assignment would misreport what was designed against.
func TestSnapshotSkipsAndCountsSymlinks(t *testing.T) {
	outside := write(t, t.TempDir(), "secret.txt", "not the assignment")
	src := t.TempDir()
	write(t, src, "brief.md", "the brief")
	if err := os.Symlink(outside, filepath.Join(src, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	snap, err := Snapshot(src, filepath.Join(t.TempDir(), "assignment"), nil, 1<<20)
	if err != nil {
		t.Fatalf("Snapshot() = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(snap.Dir, "link.txt")); !os.IsNotExist(err) {
		t.Error("a symlink reached the snapshot")
	}
	if snap.SkippedLinks != 1 {
		t.Errorf("SkippedLinks = %d, want 1", snap.SkippedLinks)
	}
}

// The budget refuses, never truncates: a partial assignment silently passed on
// would be reviewed as though it were whole. The message names the WHOLE budget,
// not whatever remained when the breach happened.
func TestSnapshotRefusesOverTheBudget(t *testing.T) {
	src := t.TempDir()
	write(t, src, "a.md", strings.Repeat("x", 600))
	write(t, src, "b.md", strings.Repeat("y", 600))

	_, err := Snapshot(src, filepath.Join(t.TempDir(), "assignment"), nil, 1000)
	if err == nil {
		t.Fatal("Snapshot() = nil, want a refusal over the budget")
	}
	if !strings.Contains(err.Error(), "1000-byte") {
		t.Errorf("the refusal should name the whole budget: %v", err)
	}
	// Exactly at the budget is allowed -- it is a ceiling, not a strict bound.
	one := t.TempDir()
	write(t, one, "a.md", strings.Repeat("x", 1000))
	if _, err := Snapshot(one, filepath.Join(t.TempDir(), "s2"), nil, 1000); err != nil {
		t.Errorf("Snapshot() at exactly the budget = %v, want nil", err)
	}
}

// A missing assignment is the operator's typo, and it must cost nothing.
func TestSnapshotRefusesAMissingAssignment(t *testing.T) {
	_, err := Snapshot(filepath.Join(t.TempDir(), "nope.md"), t.TempDir(), nil, 1<<20)
	if err == nil || !strings.Contains(err.Error(), "assignment") {
		t.Fatalf("Snapshot() = %v, want an assignment error", err)
	}
}

// An assignment that IS a symlink is refused, not followed and not walked as
// nothing. A config's target.document reaches the snapshot unchecked -- the
// collector only ever sees the copy -- so a followed `brief.md ->
// ~/.aws/credentials` handed the destination to every agent; and a root link to
// a directory was walked as an empty tree, so the run designed against nothing.
func TestSnapshotRefusesASymlinkedAssignment(t *testing.T) {
	outside := t.TempDir()
	secret := write(t, outside, "credentials", "aws_secret_access_key=x")
	write(t, outside, "tree/brief.md", "a brief elsewhere")
	src := t.TempDir()
	fileLink := filepath.Join(src, "brief.md")
	dirLink := filepath.Join(src, "assignment")
	if err := os.Symlink(secret, fileLink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "tree"), dirLink); err != nil {
		t.Fatal(err)
	}
	for _, link := range []string{fileLink, dirLink} {
		dst := filepath.Join(t.TempDir(), "assignment")
		_, err := Snapshot(link, dst, nil, 1<<20)
		if err == nil || !strings.Contains(err.Error(), "is a symlink") {
			t.Errorf("Snapshot(%s) = %v, want the symlink refusal", link, err)
		}
		if b, rerr := os.ReadFile(filepath.Join(dst, "brief.md")); rerr == nil {
			t.Errorf("the link's destination reached the snapshot: %q", b)
		}
	}
	// An ancestor that leaves its directory is the same hole one level up.
	via := filepath.Join(src, "docs")
	if err := os.Symlink(outside, via); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(filepath.Join(via, "credentials"), filepath.Join(t.TempDir(), "a"), nil, 1<<20); err == nil || !strings.Contains(err.Error(), "leaves the directory") {
		t.Errorf("Snapshot() through an escaping ancestor = %v, want the refusal", err)
	}
}

// An exclude spelled canonically still excludes when the assignment is named
// through an alias (the /tmp -> /private/tmp shape: a relative logs.dir anchors
// to the resolved project root, an absolute target.path stays as written). And
// an exclude that CONTAINS the assignment names no part of it and must not
// silently empty the snapshot.
func TestSnapshotExcludesAcrossSpellingsAndIgnoresAncestors(t *testing.T) {
	base := t.TempDir()
	write(t, base, "real/proj/brief.md", "the brief")
	write(t, base, "real/proj/runlogs/old/prompt.md", "a previous run's prompt")
	// A link that stays inside its own directory, so it is an alias, not an escape.
	if err := os.Symlink(filepath.Join(base, "real"), filepath.Join(base, "alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	canon, err := filepath.EvalSymlinks(filepath.Join(base, "real", "proj"))
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(base, "alias", "proj")

	snap, err := Snapshot(src, filepath.Join(t.TempDir(), "s"), []string{filepath.Join(canon, "runlogs")}, 1<<20)
	if err != nil {
		t.Fatalf("Snapshot() = %v", err)
	}
	if _, err := os.Stat(filepath.Join(snap.Dir, "runlogs")); !os.IsNotExist(err) {
		t.Error("the logs reached the snapshot because the exclude was spelled canonically")
	}
	if snap.Files != 1 {
		t.Errorf("Files = %d, want 1", snap.Files)
	}

	snap, err = Snapshot(src, filepath.Join(t.TempDir(), "s"), []string{base}, 1<<20)
	if err != nil {
		t.Fatalf("Snapshot() = %v", err)
	}
	if snap.Files != 2 {
		t.Errorf("an exclude containing the assignment emptied the snapshot: Files = %d, want 2", snap.Files)
	}
}

// A logs.dir whose literal prefix IS the assignment directory is refused. The
// exclusion used to be dropped as "names no part of the assignment", so every
// earlier run's prompts and raw outputs, sitting directly under the assignment,
// were copied in as material for every agent.
func TestSnapshotRefusesAnExcludeThatIsTheAssignment(t *testing.T) {
	base := t.TempDir()
	write(t, base, "real/proj/brief.md", "the brief")
	write(t, base, "real/proj/20260101-000000/round-1/prompt.md", "a previous run's prompt")
	if err := os.Symlink(filepath.Join(base, "real"), filepath.Join(base, "alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	canon, err := filepath.EvalSymlinks(filepath.Join(base, "real", "proj"))
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(base, "alias", "proj")
	// Both spellings: as written, and canonical while the assignment is an alias.
	for _, logs := range []string{src, canon, src + string(filepath.Separator)} {
		dst := filepath.Join(t.TempDir(), "s")
		_, err := Snapshot(src, dst, []string{logs}, 1<<20)
		if err == nil || !strings.Contains(err.Error(), "logs.dir") {
			t.Errorf("Snapshot(exclude %s) = %v, want the refusal", logs, err)
		}
		if _, rerr := os.Stat(filepath.Join(dst, "20260101-000000")); rerr == nil {
			t.Errorf("an earlier run's logs reached the snapshot (exclude %s)", logs)
		}
	}
}

// copyFile's O_NOFOLLOW is the guard for a file swapped for a link between the
// walk classifying it and the open. The walk itself skips links it sees, so only
// a direct call can reach the open with one: it must fail, and the link's
// destination must not be copied.
func TestCopyFileDoesNotFollowASymlink(t *testing.T) {
	secret := write(t, t.TempDir(), "credentials", "aws_secret_access_key=x")
	link := filepath.Join(t.TempDir(), "brief.md")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "brief.md")
	if n, err := copyFile(link, dst, 1<<20); err == nil {
		t.Errorf("copyFile(symlink) = %d, nil; want a refusal to follow it", n)
	}
	if b, err := os.ReadFile(dst); err == nil && strings.Contains(string(b), "aws_secret") {
		t.Errorf("the link's destination was copied: %q", b)
	}
}
