package implement

import (
	"context"
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

// Outcomes indexes the records by task id.
func (h History) Outcomes() map[string]Record {
	out := make(map[string]Record, len(h.Records))
	for _, r := range h.Records {
		out[r.Task] = r
	}
	return out
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
// with no commit, and the outcomes to carry for everything before it.
type Resume struct {
	// Index is the plan position BUILD re-enters at. Equal to len(plan.Tasks)
	// when every task was processed.
	Index int
	// Carried is the recorded outcome per task, by id.
	Carried map[string]Record
}

// AdmitResume applies §5.5's admission rules to a history and a plan, and
// returns where to resume.
//
// The rules exist because a resumed run adopts someone else's work as its own:
// it will report the carried outcomes in its summary and build on the tree they
// left. Each refusal is a case where the repository cannot be shown to be the
// one this plan was built into.
//
//   - The recorded tasks must form a PREFIX of the plan's order. Tasks run
//     serially in plan order, so any other shape means the history and the plan
//     disagree about what was being built -- a plan edited between runs, or a
//     repository built from a different one.
//   - Every recorded task must exist in the plan, for the same reason.
//   - The design must be the one the bootstrap commit names.
//   - The gate must be the one the bootstrap commit names. A project half-built
//     under one gate must not be finished under another and reported as one
//     thing.
//
// Carried outcomes are FINAL: a carried failed or blocked task is not retried
// here. Re-opening one is a different decision with its own flag (§12.6), and
// silently retrying would make a resumed run disagree with the report the
// original produced.
func AdmitResume(h History, pl Plan, designSHA, verifyProfile string) (Resume, error) {
	var r Resume
	if h.DesignSHA != "" && designSHA != "" && h.DesignSHA != designSHA {
		return r, fmt.Errorf("the project was built from design %.12s and the design in it now hashes to %.12s; a half-built project cannot be finished against a different document", h.DesignSHA, designSHA)
	}
	if h.VerifyProfile != verifyProfile {
		return r, fmt.Errorf("the project was built under a different gate (recorded %.12s, configured %.12s): a project half-built under one gate must not be finished under another and reported as one thing -- restore the original verify configuration, or start a fresh run", h.VerifyProfile, verifyProfile)
	}

	position := make(map[string]int, len(pl.Tasks))
	for i, t := range pl.Tasks {
		position[t.ID] = i
	}
	r.Carried = make(map[string]Record, len(h.Records))
	for i, rec := range h.Records {
		pos, known := position[rec.Task]
		switch {
		case !known:
			return r, fmt.Errorf("the history records task %q, which the plan does not contain: the repository and PLAN.json disagree about what was being built", rec.Task)
		case pos != i:
			return r, fmt.Errorf("the history records %q at position %d and the plan puts it at %d: recorded tasks must be a prefix of the plan's order, and this repository's are not", rec.Task, i+1, pos+1)
		}
		r.Carried[rec.Task] = rec
	}
	r.Index = len(h.Records)
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
