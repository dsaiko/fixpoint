package orchestrator

import (
	"os"
	"path/filepath"
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
