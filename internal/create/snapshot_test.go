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
