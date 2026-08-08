package orchestrator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/worktree"
)

// batchByFile groups issues so that no two in one batch name the same file, and
// no batch is larger than n.
//
// Two coder sessions editing one file from a shared base produce patches that do
// not compose: each is a diff against the same original text, so applying the
// second after the first either conflicts or silently reverts part of it. Keeping
// a file to one session per batch removes that case by construction rather than
// detecting it afterwards -- there is no merge to get wrong.
//
// The consequence is that the speedup is bounded by the BUSIEST file, not by n.
// Measured on this project's own rounds: 9 to 11 issues spread over 4 to 6 files,
// with the worst file holding 4 -- so a round of nine runs in four batches however
// high n is set. That is a real limit and worth knowing before setting the number.
//
// Order is preserved within and across batches: issues arrive worst-severity
// first, and a batch boundary must not promote a low finding ahead of a high one.
// An issue with no file goes in a batch of its own, since "which files does it
// touch" is unanswerable and assuming none would let it collide with anything.
func batchByFile(issues []model.Issue, n int) [][]model.Issue {
	if n < 1 {
		n = 1
	}
	var batches [][]model.Issue
	// files[i] is the set of files batch i already covers.
	var files []map[string]bool
	for _, it := range issues {
		placed := false
		if it.File != "" && n > 1 {
			for i := range batches {
				if len(batches[i]) >= n || files[i][it.File] {
					continue
				}
				batches[i] = append(batches[i], it)
				files[i][it.File] = true
				placed = true
				break
			}
		}
		if !placed {
			batches = append(batches, []model.Issue{it})
			files = append(files, map[string]bool{it.File: true})
		}
	}
	return batches
}

// session is one coder invocation's product, held until the serial half of the
// round can act on it.
type session struct {
	issue model.Issue
	// patch is what the session changed, as a diff against the round's base. Empty
	// means it changed nothing, which is what a rejection looks like.
	patch string
	// scratch is the record the session wrote its verdict and step into. It is
	// merged into the round's real record serially, so nothing here is written from
	// two goroutines.
	scratch *model.RoundRecord
	replies []model.FixReply
	// salvaged reports that the coder died mid-edit and its partial work was
	// committed as a salvage round.
	salvaged bool
	err      error
}

// runBatch runs one batch of issues concurrently, each in its own worktree of the
// round's base commit, and returns their products in the order the issues were
// given.
//
// Nothing here touches the main working tree. That is the division the whole
// feature rests on: sessions are network-bound and parallelize almost for free,
// while the verify gate is CPU-bound, scales at about 1.6x when run four ways, and
// -- decisively -- cannot tell which of two fixes broke a build when they are
// verified together. So this half produces patches and the caller applies them one
// at a time through the gate that already exists.
func (o *Orchestrator) runBatch(ctx context.Context, rec *model.RoundRecord, history []model.RoundRecord, base string, batch []model.Issue) []session {
	out := make([]session, len(batch))
	parent, err := os.MkdirTemp("", "fixpoint-worktrees-")
	if err != nil {
		// Without somewhere to put them there is no parallel path; report it once and
		// let the caller fall back to running this batch in the main tree.
		for i, it := range batch {
			out[i] = session{issue: it, err: fmt.Errorf("worktree parent: %w", err)}
		}
		return out
	}
	defer func() { _ = os.RemoveAll(parent) }()

	var wg sync.WaitGroup
	for i, it := range batch {
		wg.Add(1)
		// i and it are passed in rather than captured, matching every other fan-out
		// here: the slot a goroutine writes decides which issue's verdict is recorded
		// where.
		go func(i int, it model.Issue) {
			defer wg.Done()
			out[i] = o.runInWorktree(ctx, rec, history, base, parent, it)
		}(i, it)
	}
	wg.Wait()
	return out
}

// runInWorktree gives one issue its own checkout, runs the coder there, and reads
// back what it changed.
func (o *Orchestrator) runInWorktree(ctx context.Context, rec *model.RoundRecord, history []model.RoundRecord, base, parent string, it model.Issue) session {
	s := session{issue: it}
	w, err := worktree.Add(ctx, o.cfg.Target.Path, parent, base, it.ID)
	if err != nil {
		s.err = err
		return s
	}
	defer func() {
		if cerr := w.Close(ctx); cerr != nil {
			// Never fatal: the patch has already been read out, so a leaked directory
			// costs disk and nothing else. Said out loud because it is under /tmp with
			// the run's edits in it.
			o.logf("WARNING: could not remove the worktree for %s (%v); it is at %s", it.ID, cerr, w.Dir)
		}
	}()

	// A private record, so two sessions never write one struct. It carries only what
	// fix() reads and writes for a single issue; the caller merges it back in issue
	// order once the batch has joined.
	s.scratch = &model.RoundRecord{
		Round:       rec.Round,
		Assignments: rec.Assignments,
		Issues:      []model.Issue{it},
		Findings:    issueFindings(rec, it.ID),
	}
	// allowSalvage is false in a worktree: the salvage path commits partial work to
	// rescue a dead coder's edits, and a commit in a throwaway checkout rescues
	// nothing -- the patch below captures the same edits whether the session
	// finished or died.
	_, replies, err := o.fixIn(ctx, w.Dir, s.scratch, history, false, []model.Issue{it}, nil)
	s.replies = replies
	if err != nil {
		s.err = err
		return s
	}
	patch, err := w.Patch(ctx, o.gitExclude...)
	if err != nil {
		s.err = err
		return s
	}
	s.patch = patch
	return s
}

// applyPatch writes a session's diff into the main working tree.
//
// `git apply` rather than a merge: the patch was taken against the commit this
// round started from, and every fix committed since is by construction in other
// files -- so a failure here means the tree moved under an assumption the batch
// was built on, and the honest answer is to leave the issue for the next round
// rather than to force it.
func (o *Orchestrator) applyPatch(ctx context.Context, patch string) error {
	if patch == "" {
		return nil
	}
	f, err := os.CreateTemp("", "fixpoint-patch-*.diff")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.WriteString(patch); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return o.collector.Apply(ctx, f.Name())
}

// mergeSession folds a finished session's record into the round's, under the
// caller's serial ordering.
//
// Steps are billed whatever happened -- a session that failed still spent tokens
// -- and the verdict is copied only for the issue that session was given, so a
// scratch record cannot carry a claim about anybody else's issue.
func mergeSession(rec *model.RoundRecord, s session) {
	if s.scratch == nil {
		return
	}
	rec.Steps = append(rec.Steps, s.scratch.Steps...)
	rec.Findings = mergeFindings(rec.Findings, s.scratch.Findings)
	for _, si := range s.scratch.Issues {
		if si.ID != s.issue.ID {
			continue
		}
		for i := range rec.Issues {
			if rec.Issues[i].ID != si.ID {
				continue
			}
			rec.Issues[i].Status = si.Status
			rec.Issues[i].Verdict = si.Verdict
			rec.Issues[i].VerdictDetail = si.VerdictDetail
		}
	}
	rec.Fixed += s.scratch.Fixed
	rec.Rejected += s.scratch.Rejected
}

// mergeFindings copies back the verdicts a session recorded on its own
// observations, which is where issueFindings reads them from for the artifact.
func mergeFindings(into, from []model.Finding) []model.Finding {
	byID := map[string]int{}
	for i, f := range into {
		byID[f.IssueID+"\x00"+f.Agent+"\x00"+f.Lens] = i
	}
	for _, f := range from {
		if i, ok := byID[f.IssueID+"\x00"+f.Agent+"\x00"+f.Lens]; ok {
			into[i].Verdict = f.Verdict
			into[i].VerdictDetail = f.VerdictDetail
		}
	}
	return into
}

// parallelFixes is how many coder sessions this run may have in flight, resolved
// once so the loop and the log line cannot disagree.
//
// A directory target has no repository to make worktrees of. Validation already
// refuses that combination, so reaching here with one means a hand-built config;
// running sequentially is the safe reading.
func (o *Orchestrator) parallelFixes() int {
	if o.cfg.Loop.ParallelFixes < 2 || o.cfg.Target.Mode == config.ModeDirectory {
		return 1
	}
	return o.cfg.Loop.ParallelFixes
}

// issueIDsOf names a batch for the log, so an operator watching a parallel round
// can tell which findings are in flight together.
func issueIDsOf(issues []model.Issue) string {
	ids := make([]string, 0, len(issues))
	for _, it := range issues {
		ids = append(ids, it.ID)
	}
	return strings.Join(ids, ", ")
}
