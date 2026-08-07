package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/model"
)

// posterFor is forge.PosterFor behind a variable so a test can observe what would
// be published, exactly as internal/orchestrator does it. Publishing is the half of
// this mode that cannot be checked by reading a log line: the event, the anchors and
// the body all reach the forge in one call, and only a fake poster can see them.
var posterFor = forge.PosterFor

// postRun publishes a review a previous run already produced, without invoking a
// single agent.
//
// It exists because the workflow the documentation described did not work.
// "Review it, read review-body.md, then post it" was two RUNS: the second one
// re-reviewed everything, cost the same again, and published findings the operator
// had never seen -- the panel is not deterministic, and two runs over the same
// pull request produced 34 findings and then 49. So the inspect-before-publish
// control, which the whole posting design rests on, did not exist.
//
// This makes it exist. The bytes posted are the bytes in the file, read off disk;
// the anchors are the ones that run computed. Nothing is recalculated, so nothing
// can differ from what was reviewed.
func postRun(dir string, postVerdict bool, logf func(string, ...any)) int {
	sum, runDir, err := loadRunSummary(dir)
	if err != nil {
		logf("post-run: %v", err)
		return 1
	}
	if why := unreplayable(dir, sum); why != "" {
		logf("post-run: %s", why)
		return 2
	}
	// The body is always the file beside the resolved summary, never the path the
	// summary JSON records. Beside the SUMMARY, not beside the argument: the argument
	// may name the summary file itself, and joining under a file path can only fail.
	//
	// The recorded path is unusable for two separate reasons. It is absolute, so a
	// copied run directory or a moved project makes it wrong. And it is not the
	// operator's: a run directory is only files, so a pull request can commit a
	// lookalike .fixpoint/<run> whose summary points review_body at ~/.aws/credentials
	// while a plausible review-body.md sits beside it for the operator to inspect --
	// and -post-run would publish the credentials to the pull request under their
	// identity. Reading only the canonical name means the bytes published are the
	// bytes in the file the operator was invited to read, which is the entire promise
	// of this mode. Lstat for the same reason: a committed symlink at that name would
	// redirect the read just as well as a path in the JSON.
	bodyPath := filepath.Join(runDir, "review-body.md")
	info, err := os.Lstat(bodyPath)
	if err != nil {
		logf("post-run: cannot read the review body at %s: %v", bodyPath, err)
		return 1
	}
	if !info.Mode().IsRegular() {
		logf("post-run: %s is not a regular file; refusing to publish whatever it resolves to", bodyPath)
		return 1
	}
	body, err := os.ReadFile(bodyPath)
	if err != nil {
		logf("post-run: cannot read the review body at %s: %v", bodyPath, err)
		return 1
	}

	ctx := context.Background()
	p := posterFor(ctx, sum.Path)
	if p == nil {
		logf("post-run: no GitHub or GitLab remote recognized at %s", sum.Path)
		return 1
	}
	// The same mapping the run itself would have used, from the same function --
	// a replay that approved what the live path would have commented on would be a
	// different review from the one the operator inspected.
	event := forge.EventFor(sum.Verdict.Outcome, postVerdict)
	inline := make([]forge.InlineComment, 0, len(sum.ReviewInline))
	for _, a := range sum.ReviewInline {
		inline = append(inline, forge.InlineComment{Path: a.Path, Line: a.Line, Body: a.Body})
	}

	url, err := p.PostReview(ctx, sum.Path, sum.PR, sum.ReviewedHead, string(body), event, inline)
	if forge.AnchorRejection(err) {
		// Only an anchor rejection earns a second submission, exactly as in
		// Orchestrator.postReview. Retrying on ANY error would re-post after an auth
		// failure or -- the case that costs something -- a client-side timeout on a
		// call the forge already accepted, leaving two identical reviews on the pull
		// request and dropping the anchors this mode exists to replay.
		logf("post-run: %s rejected the inline comments (%v); posting the summary without them", p.Kind(), err)
		url, err = p.PostReview(ctx, sum.Path, sum.PR, sum.ReviewedHead, string(body), event, nil)
	}
	if err != nil {
		logf("post-run: publishing to %s failed: %v", p.Kind(), err)
		return 1
	}
	if url != "" {
		logf("posted %s to %s as %s: %s", filepath.Base(runDir), p.Kind(), event, url)
	} else {
		logf("posted %s to %s as %s", filepath.Base(runDir), p.Kind(), event)
	}
	return 0
}

// unreplayable says why a summary cannot be published, or returns "" if it can.
//
// Every answer names dir and ends the same way, because they are all the same
// answer: this run cannot be replayed faithfully, and only reviewing again
// produces one that can. Nothing here is guessed at or worked around -- a replay
// that filled in a fact the run did not record would publish something no panel
// reached, which is the one thing this mode must never do.
func unreplayable(dir string, sum *model.RunSummary) string {
	switch {
	case sum.Verdict == nil:
		return dir + " holds no review verdict -- a fix run has nothing to publish"
	case sum.Mode != "pr":
		return fmt.Sprintf("%s reviewed %s, not a pull request; there is nowhere to post it", dir, sum.Mode)
	case sum.PR <= 0:
		return dir + " records no pull request number. Runs from before that was recorded cannot be replayed; review again to produce one that can."
	// Which commit the review is ABOUT. Refused here rather than left to the poster
	// so the operator gets the same "review again" answer as the missing PR number
	// above: a summary that cannot say what it reviewed cannot be published against
	// anything. The poster refuses too, and additionally refuses when the pull
	// request has moved since -- replaying a verdict onto a commit the panel never
	// read is the reason this is checked at all.
	case sum.ReviewedHead == "":
		return dir + " does not record which commit it reviewed, so the review cannot be bound to one. Runs from before that was recorded cannot be replayed; review again to produce one that can."
	}
	// The run must also have FINISHED the review it is being asked to publish.
	//
	// Checked last because it is the only refusal that can be true of an otherwise
	// complete summary: the verdict and the review body are written by
	// decideVerdict, which runs BEFORE runRound's post-decideVerdict cancellation
	// check (internal/orchestrator, the review-only branch). So a Ctrl-C anywhere in
	// the minutes that refutation, judging and writing take leaves a run directory
	// holding an APPROVE that the run itself deliberately did not post and exited
	// non-zero over -- and without this, -post-run would publish it.
	//
	// An error termination is refused for a second reason. On a review-only run
	// Error is only ever set by recordRunError, and the one error that branch can
	// return is the publish the operator already asked for failing; a publish that
	// failed on the client may well have been accepted by the forge first, which is
	// exactly the duplicate-review hazard postRun's guarded retry refuses to take.
	// Every other termination belongs to a fix run, which has no verdict at all.
	if sum.Termination != model.TermReviewOnly || sum.Error != "" {
		how := sum.Termination
		if how == "" {
			how = "unrecorded"
		}
		if sum.Error != "" {
			how = fmt.Sprintf("%s (%s)", how, sum.Error)
		}
		return fmt.Sprintf("%s ended as %s, not a completed review; a verdict the run itself would not stand behind cannot be published. Review again to produce a run that can be.", dir, how)
	}
	return ""
}

// loadRunSummary reads the summary from a run directory, accepting either the
// directory itself or the summary file.
//
// It returns the run directory it resolved alongside the summary, because both
// input shapes are supported and only one of them is a directory: a caller that
// reused the argument to reach a sibling artifact would build a path underneath
// the summary FILE.
func loadRunSummary(dir string) (*model.RunSummary, string, error) {
	path := dir
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		matches, err := filepath.Glob(filepath.Join(dir, "summary-*.json"))
		if err != nil || len(matches) == 0 {
			return nil, "", fmt.Errorf("%s holds no summary-*.json; is it a run directory?", dir)
		}
		// Newest last by name, since the name carries the timestamp.
		sort.Strings(matches)
		path = matches[len(matches)-1]
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var sum model.RunSummary
	if err := json.Unmarshal(raw, &sum); err != nil {
		return nil, "", fmt.Errorf("parse %s: %w", path, err)
	}
	if sum.ReviewBody == "" {
		return nil, "", errors.New("that run wrote no review body")
	}
	return &sum, filepath.Dir(path), nil
}
