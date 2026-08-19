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
		h.Records = []Record{rec("T01", OutcomeImplemented), rec("T02", OutcomeFailed)}
		r, err := AdmitResume(h, threeTasks(), design, profile)
		if err != nil {
			t.Fatalf("AdmitResume() = %v", err)
		}
		if r.Index != 2 {
			t.Errorf("Index = %d, want 2", r.Index)
		}
		if r.Carried["T02"].Outcome != OutcomeFailed {
			t.Errorf("the carried outcome was lost: %+v", r.Carried["T02"])
		}
	})

	t.Run("a fully recorded plan resumes past the end", func(t *testing.T) {
		h := sound
		h.Records = []Record{rec("T01", OutcomeImplemented), rec("T02", OutcomeImplemented), rec("T03", OutcomeImplemented)}
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
		h.Records = []Record{rec("T02", OutcomeImplemented)}
		_, err := AdmitResume(h, threeTasks(), design, profile)
		if err == nil || !strings.Contains(err.Error(), "prefix") {
			t.Errorf("AdmitResume(out of order) = %v, want a prefix refusal", err)
		}
	})

	t.Run("a task the plan does not contain is refused", func(t *testing.T) {
		h := sound
		h.Records = []Record{rec("T99", OutcomeImplemented)}
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
		h.Records = []Record{rec("T01", OutcomeFailed)}
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

// ReadHistory's success path: a real bootstrap whose trailers populate DesignSHA
// and VerifyProfile, then task commits whose trailers become Records. Covered
// only through the e2e run before (review run 20260818-234734), so the %x00
// separator, the SHA split, and the bootstrap-vs-record branch all ran unguarded.
func TestReadHistoryReadsBootstrapAndRecords(t *testing.T) {
	gitAvailable(t)
	dir := t.TempDir()
	gitOut(t, dir, "init", "-q")
	commit := func(subject, body string) {
		write(t, dir, "f.txt", subject)
		gitOut(t, dir, "add", "-A")
		gitOut(t, dir, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", subject+"\n\n"+body)
	}
	commit("bootstrap", "Fixpoint-Phase: bootstrap\nDesign-SHA256: dead\nVerify-Profile: beef")
	commit("T01", "Fixpoint-Task: T01\nFixpoint-Outcome: implemented")
	commit("T02", "Fixpoint-Task: T02\nFixpoint-Outcome: failed\nFixpoint-Reason: gate=build")

	h, err := testGit.ReadHistory(t.Context(), dir)
	if err != nil {
		t.Fatalf("ReadHistory() = %v", err)
	}
	if h.DesignSHA != "dead" || h.VerifyProfile != "beef" {
		t.Errorf("bootstrap trailers not read: %+v", h)
	}
	if len(h.Records) != 2 || h.Records[0].Task != "T01" || h.Records[1].Task != "T02" {
		t.Fatalf("records = %+v, want T01 then T02", h.Records)
	}
	if h.Records[1].Outcome != "failed" || h.Records[1].Reason != "gate=build" {
		t.Errorf("record 2 = %+v, want failed/gate=build", h.Records[1])
	}
	if h.Head != h.Records[1].SHA {
		t.Errorf("Head %s is not the last record's SHA %s", h.Head, h.Records[1].SHA)
	}
}

// The mid-history integrity guard -- the case the rule exists for: a run stopped,
// someone committed on top, and the resume must refuse because the recorded
// outcomes can no longer be trusted as complete. Only the foreign-FIRST-commit
// refusal was tested (review run 20260818-234734, two reviewers).
func TestReadHistoryRefusesTamperedTail(t *testing.T) {
	gitAvailable(t)
	build := func(t *testing.T, tailSubject, tailBody string) string {
		t.Helper()
		dir := t.TempDir()
		gitOut(t, dir, "init", "-q")
		write(t, dir, "f.txt", "boot")
		gitOut(t, dir, "add", "-A")
		gitOut(t, dir, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m",
			"bootstrap\n\nFixpoint-Phase: bootstrap\nDesign-SHA256: d\nVerify-Profile: v")
		write(t, dir, "f.txt", "tail")
		gitOut(t, dir, "add", "-A")
		gitOut(t, dir, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", tailSubject+"\n\n"+tailBody)
		return dir
	}
	cases := map[string]struct{ subject, body, want string }{
		"a hand commit with no task trailer": {"fixed it by hand", "", "carries no Fixpoint-Task"},
		"a task with no outcome":             {"T01", "Fixpoint-Task: T01", "which fixpoint never writes"},
		"a task with an invalid outcome":     {"T01", "Fixpoint-Task: T01\nFixpoint-Outcome: donezo", "which fixpoint never writes"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := testGit.ReadHistory(t.Context(), build(t, tc.subject, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("ReadHistory() = %v, want a refusal containing %q", err, tc.want)
			}
		})
	}
}

// trailers() is the parser under §5.5's whole admission. Its branches -- the
// allowlist, the colon requirement, last-wins on duplicates -- were covered only
// by the e2e run (review run 20260818-234734).
func TestTrailers(t *testing.T) {
	body := "Some prose, not a trailer.\n" +
		"Fixpoint-Task: T01\n" +
		"Not-A-Trailer without a colon\n" +
		"Unknown-Key: ignored\n" +
		"Fixpoint-Outcome: failed\n" +
		"Fixpoint-Outcome: implemented\n" // duplicate: last wins
	got := trailers(body)
	if got[trailerTask] != "T01" {
		t.Errorf("task = %q", got[trailerTask])
	}
	if got[trailerOutcome] != "implemented" {
		t.Errorf("outcome = %q, want the last of two", got[trailerOutcome])
	}
	if _, ok := got["Unknown-Key:"]; ok {
		t.Error("an unknown key was kept")
	}
	if len(trailers("")) != 0 {
		t.Error("an empty body produced trailers")
	}
}

// AdmitResume refuses a bootstrap with no Design-SHA256 trailer -- the same
// accept-if-absent rule -plan already refuses, on the other door (review run
// 20260818-234734).
func TestAdmitResumeRefusesMissingDesignTrailer(t *testing.T) {
	h := History{Bootstrap: "boot", DesignSHA: "", VerifyProfile: "p"}
	_, err := AdmitResume(h, threeTasks(), "d", "p")
	if err == nil || !strings.Contains(err.Error(), "not a waiver") {
		t.Errorf("AdmitResume(no design trailer) = %v, want a refusal", err)
	}
}
