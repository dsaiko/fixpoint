package implement

import (
	"strings"
	"testing"
)

func rec(task, outcome string) Record { return Record{Task: task, Outcome: outcome, SHA: "abc"} }

func threeTasks() Plan {
	return Plan{Tasks: []Task{{ID: "T01"}, {ID: "T02"}, {ID: "T03"}}}
}

// §5.5's admission rules. A resumed run adopts someone else's work as its own
// and reports it, so each refusal is a case where the repository cannot be
// shown to be the one this plan was built into.
func TestAdmitResume(t *testing.T) {
	const design = "d00d"
	const profile = "p00f"
	sound := History{Bootstrap: "boot", DesignSHA: design, VerifyProfile: profile}

	t.Run("a prefix resumes at the first unrecorded task", func(t *testing.T) {
		h := sound
		h.Records = []Record{rec("T01", "implemented"), rec("T02", "failed")}
		r, err := AdmitResume(h, threeTasks(), design, profile)
		if err != nil {
			t.Fatalf("AdmitResume() = %v", err)
		}
		if r.Index != 2 {
			t.Errorf("Index = %d, want 2", r.Index)
		}
		if r.Carried["T02"].Outcome != "failed" {
			t.Errorf("the carried outcome was lost: %+v", r.Carried["T02"])
		}
	})

	t.Run("a fully recorded plan resumes past the end", func(t *testing.T) {
		h := sound
		h.Records = []Record{rec("T01", "implemented"), rec("T02", "implemented"), rec("T03", "implemented")}
		r, err := AdmitResume(h, threeTasks(), design, profile)
		if err != nil {
			t.Fatalf("AdmitResume() = %v", err)
		}
		if r.Index != 3 {
			t.Errorf("Index = %d, want 3 -- nothing is left to do", r.Index)
		}
	})

	t.Run("out of order is refused", func(t *testing.T) {
		h := sound
		// Tasks run serially in plan order, so this shape means the history and
		// the plan disagree about what was being built.
		h.Records = []Record{rec("T02", "implemented")}
		_, err := AdmitResume(h, threeTasks(), design, profile)
		if err == nil || !strings.Contains(err.Error(), "prefix") {
			t.Errorf("AdmitResume(out of order) = %v, want a prefix refusal", err)
		}
	})

	t.Run("a task the plan does not contain is refused", func(t *testing.T) {
		h := sound
		h.Records = []Record{rec("T99", "implemented")}
		_, err := AdmitResume(h, threeTasks(), design, profile)
		if err == nil || !strings.Contains(err.Error(), "T99") {
			t.Errorf("AdmitResume(unknown task) = %v, want a refusal naming it", err)
		}
	})

	t.Run("a different design is refused", func(t *testing.T) {
		h := sound
		_, err := AdmitResume(h, threeTasks(), "other", profile)
		if err == nil || !strings.Contains(err.Error(), "different document") {
			t.Errorf("AdmitResume(other design) = %v, want a design refusal", err)
		}
	})

	t.Run("a different gate is refused", func(t *testing.T) {
		h := sound
		// A project half-built under one gate must not be finished under another
		// and reported as one thing.
		_, err := AdmitResume(h, threeTasks(), design, "different")
		if err == nil || !strings.Contains(err.Error(), "different gate") {
			t.Errorf("AdmitResume(other gate) = %v, want a gate refusal", err)
		}
	})

	// Carried failed and blocked outcomes are FINAL: re-opening one is a
	// different decision with its own flag (§12.6), and retrying silently would
	// make a resumed run disagree with the report the original produced.
	t.Run("a carried failure is not re-opened", func(t *testing.T) {
		h := sound
		h.Records = []Record{rec("T01", "failed")}
		r, err := AdmitResume(h, threeTasks(), design, profile)
		if err != nil {
			t.Fatal(err)
		}
		if r.Index != 1 {
			t.Errorf("Index = %d: a failed task must be carried, not retried", r.Index)
		}
	})
}

// The reader takes its facts from the commits alone, and says so when the
// history is not one fixpoint wrote.
func TestReadHistoryRejectsAForeignHistory(t *testing.T) {
	gitAvailable(t)
	dir := t.TempDir()
	gitOut(t, dir, "init", "-q")
	write(t, dir, "a.txt", "x\n")
	gitOut(t, dir, "add", "-A")
	gitOut(t, dir, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "not fixpoint")

	if _, err := testGit.ReadHistory(t.Context(), dir); err == nil {
		t.Fatal("a repository fixpoint did not build was accepted")
	} else if !strings.Contains(err.Error(), "not a repository fixpoint built") {
		t.Errorf("the refusal must say what is wrong: %v", err)
	}
}
