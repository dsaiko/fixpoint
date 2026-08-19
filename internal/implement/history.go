package implement

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Record is one task's outcome as the repository itself records it -- the
// trailers §5.4 writes on every processed task's commit, read back.
type Record struct {
	Task    string
	Outcome string
	Reason  string
	SHA     string
}

// History is everything a resumed run learns from a repository fixpoint built,
// without any artifact from the original run: §5.5's "the repository is
// self-describing".
type History struct {
	// Bootstrap is the initializing commit, the oldest in the history.
	Bootstrap string
	// DesignSHA and VerifyProfile are the bootstrap commit's assertions about
	// what was implemented and what gate it was implemented under.
	DesignSHA     string
	VerifyProfile string
	// Records are the task outcomes, oldest first.
	Records []Record
	// Head is the tip the replay was read from.
	Head string
}

// trailer field names, kept in one place because they are a DURABLE contract:
// every one of them is already written into repositories that exist, so a
// rename here is a rename of history nobody can rewrite.
const (
	trailerRun       = "Fixpoint-Run:"
	trailerPhase     = "Fixpoint-Phase:"
	trailerTask      = "Fixpoint-Task:"
	trailerOutcome   = "Fixpoint-Outcome:"
	trailerReason    = "Fixpoint-Reason:"
	trailerDesignSHA = "Design-SHA256:"
	trailerProfile   = "Verify-Profile:"
)

// ReadHistory replays a project's own commits into the state a resumed run
// needs. Oldest first, so the records are in the order the tasks were processed
// and the caller can check that order against the plan.
//
// Only what fixpoint wrote is read: a commit with no Fixpoint-Task trailer is
// somebody else's and is reported as such rather than skipped, because §5.5's
// admission rules depend on knowing that the history is entirely this tool's.
func (g Git) ReadHistory(ctx context.Context, dir string) (History, error) {
	var h History
	// %H then the raw body, NUL-terminated per commit. The separator is written
	// as git's own %x00 escape rather than a literal byte: a NUL cannot be passed
	// in argv at all (exec refuses it), and git commit messages cannot contain
	// one, so the record boundary is unambiguous.
	const sep = "\x00"
	out, err := g.run(ctx, dir, "log", "--reverse", "--format=%H%n%B%x00")
	if err != nil {
		return h, fmt.Errorf("read history of %s: %w", dir, err)
	}
	for _, chunk := range strings.Split(out, sep) {
		chunk = strings.TrimLeft(chunk, "\n")
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		sha, body, _ := strings.Cut(chunk, "\n")
		sha = strings.TrimSpace(sha)
		h.Head = sha

		fields := trailers(body)
		if h.Bootstrap == "" {
			// The first commit must be fixpoint's own bootstrap; §5.5 refuses a
			// repository whose history did not start here, and the caller reports
			// it. Recording what was found makes that refusal specific.
			if fields[trailerPhase] == "bootstrap" {
				h.Bootstrap = sha
				h.DesignSHA = fields[trailerDesignSHA]
				h.VerifyProfile = fields[trailerProfile]
				continue
			}
			return h, fmt.Errorf("the first commit of %s is not a fixpoint bootstrap (%.12s); this is not a repository fixpoint built", dir, sha)
		}
		task := fields[trailerTask]
		if task == "" {
			return h, fmt.Errorf("commit %.12s in %s carries no %s trailer: the history has a commit fixpoint did not make, so its recorded outcomes cannot be trusted as complete", sha, dir, strings.TrimSuffix(trailerTask, ":"))
		}
		// The outcome is validated as strictly as the task id, because an
		// unrecognized one would otherwise act as SUCCESS: the dependency check
		// treats any outcome outside failed/blocked/skipped as landed, so a
		// commit carrying `Fixpoint-Outcome:` with a typo -- or nothing -- would
		// release every dependent of a task nobody finished (review run
		// 20260818-234734).
		if out := fields[trailerOutcome]; !recordableOutcome(out) {
			return h, fmt.Errorf("commit %.12s in %s records task %s with outcome %q, which fixpoint never writes: the history has been edited, so its recorded outcomes cannot be trusted", sha, dir, task, out)
		}
		h.Records = append(h.Records, Record{
			Task:    task,
			Outcome: fields[trailerOutcome],
			Reason:  fields[trailerReason],
			SHA:     sha,
		})
	}
	if h.Bootstrap == "" {
		return h, fmt.Errorf("%s has no commits; there is nothing to continue", dir)
	}
	return h, nil
}

// Task outcome vocabulary (§5.4). Strings, because they land in commit
// trailers, journals and the scoreboard verbatim -- and they live HERE, in the
// package that writes and reads the trailers, so the writer and the reader
// cannot drift. internal/orchestrator aliases these rather than restating them.
const (
	OutcomeImplemented = "implemented"
	OutcomeSatisfied   = "already_satisfied"
	OutcomeBlocked     = "blocked"
	OutcomeFailed      = "failed"
	OutcomeSkipped     = "skipped"
	// OutcomeUnreached is for a task the loop never got to -- the deadline, the
	// breaker, the disk bound or Ctrl-C stopped the run first. Not a judgment
	// about the work: it is what §1's "recorded in the run's report as not
	// built, with a reason" means for the tail of an interrupted plan.
	OutcomeUnreached = "unreached"
	// OutcomeCarried is an outcome a run ADOPTED from the repository's own
	// history rather than produced (§5.4). A resumed run reports the whole plan,
	// and the reader has to be able to tell which half it actually did. It is a
	// REPORT label only -- never written to a trailer, which is why
	// recordableOutcome below rejects it.
	OutcomeCarried = "carried"
)

// recordableOutcome reports whether an outcome is one §5.4 ever writes into a
// commit. `skipped` is deliberately absent -- skips are derived and leave no
// commit -- and so is `carried`, which is a report label, never a trailer.
func recordableOutcome(o string) bool {
	switch o {
	case OutcomeImplemented, OutcomeSatisfied, OutcomeBlocked, OutcomeFailed:
		return true
	}
	return false
}

// trailers reads the "Key: value" lines of a commit body. Last wins, which
// matches how git itself resolves a repeated trailer.
func trailers(body string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, " ")
		if !ok || !strings.HasSuffix(key, ":") {
			continue
		}
		switch key {
		case trailerRun, trailerPhase, trailerTask, trailerOutcome,
			trailerReason, trailerDesignSHA, trailerProfile:
			out[key] = strings.TrimSpace(value)
		}
	}
	return out
}

// Resume is where a continued run picks up: the index of the first plan task
// with no recorded or derivable outcome, and the outcomes to carry for
// everything before it.
type Resume struct {
	// Index is the plan position BUILD re-enters at. Equal to len(plan.Tasks)
	// when every task was processed.
	Index int
	// Carried is the recorded (or derived -- see AdmitResume on skips) outcome
	// per task, by id.
	Carried map[string]Record
}

// BlockedDependency returns the first of t's dependencies whose recorded
// outcome means it did not land. Skips are DERIVED, never recorded (§5.4):
// they are a pure function of the plan and the failed/blocked outcomes, which
// is why this lives here -- the build loop and the resume admission must
// derive them identically, and two copies of the rule is how they would drift.
func BlockedDependency(t Task, outcomes map[string]string) string {
	for _, d := range t.DependsOn {
		switch outcomes[d] {
		case OutcomeFailed, OutcomeBlocked, OutcomeSkipped:
			return d
		}
	}
	return ""
}

// AdmitResume applies §5.5's admission rules to a history and a plan, and
// returns where to resume.
//
// The rules exist because a resumed run adopts someone else's work as its own:
// it will report the carried outcomes in its summary and build on the tree they
// left. Each refusal is a case where the repository cannot be shown to be the
// one this plan was built into.
//
// The prefix is over "the recorded tasks -- commits and markers, PLUS THE
// DERIVED SKIPS" (§5.5), and the second half is load-bearing: a skip leaves no
// commit on purpose, so the common interrupted shape -- an early failure, its
// dependents skipped, later independent tasks built before the stop -- leaves
// holes in the recorded sequence. The first version required a gapless prefix
// and therefore refused, forever, exactly the repositories -continue exists to
// recover (review run 20260818-234734, three reviewers independently). The walk
// admits a hole only when the plan task in it is derivable as skipped from the
// outcomes carried so far; a hole nothing explains is still a refusal.
//
//   - every recorded task must exist in the plan, in plan order;
//   - the design must be the one the bootstrap commit names, and a bootstrap
//     with NO design trailer is refused rather than waived -- fixpoint always
//     writes one, so its absence means the history was rewritten. The same
//     accept-if-absent rule was already refused for -plan, and this is the same
//     guard on the other door;
//   - the gate must be the one the bootstrap commit names. A project half-built
//     under one gate must not be finished under another and reported as one
//     thing.
//
// Carried outcomes are FINAL: a carried failed or blocked task is not retried
// here. Re-opening one is a different decision with its own flag (§12.6), and
// silently retrying would make a resumed run disagree with the report the
// original produced.
func AdmitResume(h History, pl Plan, designSHA, verifyProfile string) (Resume, error) {
	var r Resume
	if h.DesignSHA == "" {
		return r, errors.New("the bootstrap commit carries no Design-SHA256 trailer; fixpoint always writes one, so the history has been rewritten -- a missing digest is a refusal, not a waiver")
	}
	if designSHA != "" && h.DesignSHA != designSHA {
		return r, fmt.Errorf("the project was built from design %.12s and the design in it now hashes to %.12s; a half-built project cannot be finished against a different document", h.DesignSHA, designSHA)
	}
	if h.VerifyProfile != verifyProfile {
		return r, fmt.Errorf("the project was built under a different gate (recorded %.12s, configured %.12s): a project half-built under one gate must not be finished under another and reported as one thing -- restore the original verify configuration, or start a fresh run", h.VerifyProfile, verifyProfile)
	}

	position := make(map[string]int, len(pl.Tasks))
	for i, t := range pl.Tasks {
		position[t.ID] = i
	}
	for _, rec := range h.Records {
		if _, known := position[rec.Task]; !known {
			return r, fmt.Errorf("the history records task %q, which the plan does not contain: the repository and PLAN.json disagree about what was being built", rec.Task)
		}
	}

	// Walk the plan and the records together. A record must sit at the cursor;
	// a plan task with no record is admitted as a derived skip only while later
	// records prove the original run went past it.
	r.Carried = make(map[string]Record, len(h.Records))
	outcomes := make(map[string]string, len(h.Records))
	ri := 0
	for idx, t := range pl.Tasks {
		if ri < len(h.Records) && h.Records[ri].Task == t.ID {
			r.Carried[t.ID] = h.Records[ri]
			outcomes[t.ID] = h.Records[ri].Outcome
			ri++
			continue
		}
		if ri >= len(h.Records) {
			// Nothing recorded past here: this is where the original run stopped,
			// and where the resumed one re-enters. The build loop re-derives any
			// skip this task would be, exactly as a fresh run would.
			r.Index = idx
			return r, nil
		}
		// A hole with records beyond it: legitimate only as a derived skip.
		dep := BlockedDependency(t, outcomes)
		if dep == "" {
			return r, fmt.Errorf("the history records %q after a gap at %q, which no failed or blocked dependency explains: recorded tasks plus derived skips must form a prefix of the plan's order, and this repository's do not", h.Records[ri].Task, t.ID)
		}
		r.Carried[t.ID] = Record{Task: t.ID, Outcome: OutcomeSkipped, Reason: "dependency " + dep}
		outcomes[t.ID] = OutcomeSkipped
	}
	r.Index = len(pl.Tasks)
	return r, nil
}

// ShowBlob reads one path out of a commit. The committed bytes, never the
// worktree's: a control artifact is what the bootstrap commit says it is, and a
// resumed run must not be steerable by a file someone edited between runs.
func (g Git) ShowBlob(ctx context.Context, dir, commit, path string) (string, error) {
	out, err := g.run(ctx, dir, "show", commit+":"+path)
	if err != nil {
		return "", fmt.Errorf("read %s from %.12s: %w", path, commit, err)
	}
	return out, nil
}
