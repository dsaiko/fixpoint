package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

// A custom logs.dir inside a directory assignment must stay out of the
// snapshot. Only the built-in `.fixpoint` name was excluded, so a `runlogs/`
// logs dir put the run's own snapshot inside the tree being copied: the walk
// reached it and copied the snapshot into itself, and an earlier run's prompts
// rode along as assignment material.
func TestCreateSnapshotExcludesACustomLogsDirInsideTheAssignment(t *testing.T) {
	f, _, _ := createFixture(t, "mock")
	if err := os.WriteFile(filepath.Join(f.repo, "brief.md"), []byte("design a card game\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	prior := filepath.Join(f.repo, "runlogs", "20260101-000000", "round-1", "prompt.md")
	if err := os.MkdirAll(filepath.Dir(prior), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prior, []byte("a previous run's prompt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.cfg.Logs.Dir = filepath.Join(f.repo, "runlogs", "{timestamp}", "round-{round}")
	f.cfg.Create.Out = filepath.Join(t.TempDir(), "DESIGN.md")
	logf, _ := captureLog()
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "t.yaml"}}, logf)
	if err != nil {
		t.Fatal(err)
	}

	_, snap, _, _, err := o.prepareCreate(t.Context(), &model.RunSummary{})
	if err != nil {
		t.Fatalf("prepareCreate() = %v", err)
	}
	if _, err := os.Stat(filepath.Join(snap.Dir, "runlogs")); !os.IsNotExist(err) {
		t.Errorf("the logs dir reached the snapshot (stat err %v); the run would design against its own logs", err)
	}
	if _, err := os.Stat(filepath.Join(snap.Dir, "brief.md")); err != nil {
		t.Errorf("the assignment itself is missing from the snapshot: %v", err)
	}
}

// A logs.dir whose literal prefix IS the assignment directory is refused before
// any agent runs. target.path == logs base is already refused by New; a
// target.document naming a directory is the door that reached the snapshot,
// which dropped the exclusion as "the assignment itself" and copied every
// earlier run's prompts in as material.
func TestCreateRefusesALogsDirThatIsTheAssignment(t *testing.T) {
	f, _, _ := createFixture(t, "mock")
	assignment := filepath.Join(f.repo, "assignment")
	prior := filepath.Join(assignment, "20260101-000000", "round-1", "prompt.md")
	if err := os.MkdirAll(filepath.Dir(prior), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prior, []byte("a previous run's prompt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assignment, "brief.md"), []byte("design a card game\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.cfg.Target.Document = "assignment"
	f.cfg.Logs.Dir = filepath.Join(assignment, "{timestamp}", "round-{round}")
	f.cfg.Create.Out = filepath.Join(t.TempDir(), "DESIGN.md")
	logf, _ := captureLog()
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "t.yaml"}}, logf)
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, _, err = o.prepareCreate(t.Context(), &model.RunSummary{})
	if err == nil || !strings.Contains(err.Error(), "logs.dir") {
		t.Fatalf("prepareCreate() = %v, want the logs-layout refusal", err)
	}
	// The snapshot lands in this run's own logs, under the assignment: the earlier
	// prompt must exist once, where it was written, and nowhere else.
	var copies []string
	_ = filepath.WalkDir(assignment, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "prompt.md" {
			copies = append(copies, p)
		}
		return nil
	})
	if len(copies) != 1 {
		t.Errorf("an earlier run's logs were copied into the snapshot: %v", copies)
	}
}
