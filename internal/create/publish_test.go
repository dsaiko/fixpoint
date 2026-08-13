package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/model"
)

// The provenance is fixpoint's voice: degradations a reader must see, stamped by
// the tool -- a degradation notice the degraded model writes about itself is not
// a notice.
func TestDeliverableStampsProvenanceAndAppendix(t *testing.T) {
	got := Deliverable("# The Design\n\n## Decisions and dissent\n\nnoted.\n",
		Provenance{RunID: "20260809-1", Editor: "claude", Pool: 2, Proposals: 1, Critiques: 0,
			SingleModel: true, Uncritiqued: true, Unrevised: true, SkippedLinks: 2},
		[]model.Objection{{Passage: "retries", Defect: "unbounded", Consequence: "wedge"}})

	for _, want := range []string{
		"stamped by the tool",
		"SINGLE-MODEL",
		"UNCRITIQUED",
		"UNREVISED",
		"2 symlink(s)",
		"# The Design",
		"Appendix: unapplied objections",
		"**retries** — unbounded (wedge)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("deliverable missing %q:\n%s", want, got)
		}
	}
	// The header precedes the document; the appendix follows it.
	if strings.Index(got, "stamped by the tool") > strings.Index(got, "# The Design") {
		t.Error("the provenance header must open the deliverable")
	}
	if strings.Index(got, "Appendix") < strings.Index(got, "# The Design") {
		t.Error("the appendix must follow the document")
	}
}

// A clean run stamps no degradation flags: a header crying wolf on every healthy
// deliverable would teach readers to skip it.
func TestDeliverableCleanRunHasNoFlags(t *testing.T) {
	got := Deliverable("# D\n", Provenance{RunID: "r", Pool: 2, Proposals: 2, Critiques: 2}, nil)
	for _, absent := range []string{"SINGLE-MODEL", "UNCRITIQUED", "UNREVISED", "Appendix"} {
		if strings.Contains(got, absent) {
			t.Errorf("a clean deliverable carries %q:\n%s", absent, got)
		}
	}
}

// Publish must never replace an existing file, atomically -- link(2) fails on an
// existing target, which makes the refusal and the write one operation. And a
// failed publish must not leave a truncated file squatting on the protected name.
func TestPublishRefusesToOverwriteAndLeavesNoDebris(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "DESIGN.md")
	if err := Publish(out, "v1"); err != nil {
		t.Fatalf("Publish() = %v", err)
	}
	if err := Publish(out, "v2"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Publish() over an existing file = %v, want the refusal", err)
	}
	b, err := os.ReadFile(out)
	if err != nil || string(b) != "v1" {
		t.Fatalf("the original deliverable was disturbed: %q, %v", b, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("debris left beside the deliverable: %v", entries)
	}
}

// The default -out sits beside the assignment, whichever shape it has.
func TestDefaultOut(t *testing.T) {
	if got := DefaultOut("/a/b/brief.md", false); got != "/a/b/DESIGN.md" {
		t.Errorf("file assignment: %q", got)
	}
	if got := DefaultOut("/a/b", true); got != "/a/b/DESIGN.md" {
		t.Errorf("dir assignment: %q", got)
	}
}

// A pre-placed temp path must never be written through: the deliverable lands
// beside an assignment whose directory an untrusted target may own, so a
// symlink squatting on the temp name would otherwise truncate its destination
// (review run 20260813-124710, the run's one critical finding).
func TestPublishDoesNotFollowAPrePlacedTempSymlink(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("precious\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "DESIGN.md")
	// The old fixed name, and a couple of nearby guesses.
	for _, bait := range []string{out + ".fixpoint-tmp", out + ".0.fixpoint-tmp"} {
		if err := os.Symlink(victim, bait); err != nil {
			t.Fatal(err)
		}
	}
	if err := Publish(out, "# design\n"); err != nil {
		t.Fatalf("Publish() = %v", err)
	}
	got, err := os.ReadFile(victim)
	if err != nil || string(got) != "precious\n" {
		t.Fatalf("the victim file was written through: %q, %v", got, err)
	}
	if b, err := os.ReadFile(out); err != nil || string(b) != "# design\n" {
		t.Fatalf("deliverable = %q, %v", b, err)
	}
}

// The temp file is owner-only regardless of what was there before, so a
// pre-created world-readable file cannot donate its permissions to the
// deliverable.
func TestPublishDeliverableIsOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "DESIGN.md")
	if err := Publish(out, "x"); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if perm := st.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("deliverable mode = %v, want owner-only", perm)
	}
}
