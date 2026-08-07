package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/model"
)

// postReceipt is the file a -post-run publication leaves in the run directory.
//
// It exists because sum.ReviewPosted only records what the RUN published: a
// summary is written once, when the run ends, and -post-run comes after that and
// never rewrites it. Without a receipt of its own, `-post-run` twice over the same
// directory puts two identical reviews on the pull request -- or two approvals --
// which is exactly the duplicate submission the guarded retry, forge's anchor
// handling and the ReviewPosted refusal all already spend code to avoid.
//
// A separate file rather than a rewrite of the summary because it has to be both
// the record AND the lock. Loading the summary, seeing an empty ReviewPosted and
// writing it back is a read-modify-write: two -post-run processes over one
// directory both read "not posted" and both publish. Creating this file with
// O_EXCL is one syscall that either claims the run or reports that somebody else
// already has, so the claim cannot interleave. It is taken BEFORE the submission,
// not after, for the same reason the retry above is guarded -- a call that failed
// on the client may have been accepted by the forge first, so the outcome is
// written into the receipt afterwards but the claim is never given back.
const postReceipt = "review-posted"

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
func postRun(ctx context.Context, dir string, postVerdict bool, logf func(string, ...any)) int {
	sum, runDir, err := loadRunSummary(dir)
	if err != nil {
		logf("post-run: %v", err)
		return 1
	}
	if why := unreplayable(dir, sum); why != "" {
		logf("post-run: %s", why)
		return 2
	}
	// Reported here, before the body is read and the forge is resolved, so a repeat
	// invocation is answered by the receipt rather than by whatever the network says.
	// The claim below is what actually decides it -- this is the message.
	if why := alreadyPosted(runDir, sum.PR); why != "" {
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

	// Claimed before anything is submitted: after this returns nil nothing else may
	// publish this run, and the claim is not released by a failure. Everything that
	// can refuse locally -- the summary, the body, the forge lookup -- has already
	// run, so a directory only ever gets a receipt for a publish that was really
	// attempted.
	receipt, err := os.OpenFile(filepath.Join(runDir, postReceipt), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		// Lost the race, or the early check above could not read the receipt.
		logf("post-run: %s", alreadyPosted(runDir, sum.PR))
		return 2
	}
	if err != nil {
		// Refused rather than published-and-not-recorded: an unrecordable publish is a
		// publish that can be replayed again tomorrow, which is what this whole file is
		// trying to prevent.
		logf("post-run: cannot record the publication in %s (%v); refusing to publish what could then be published again", runDir, err)
		return 1
	}
	defer func() { _ = receipt.Close() }()

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
		// The receipt stays, and says what is not known: a submission that failed on the
		// client may have been accepted by the forge first -- the case the unretried
		// timeout above exists for -- so the operator, not a second automatic attempt,
		// decides whether the pull request already holds this review.
		recordPublication(receipt, fmt.Sprintf("attempted as %s; the submission to %s failed and may or may not have been accepted: %v", event, p.Kind(), err))
		logf("post-run: publishing to %s failed: %v", p.Kind(), err)
		return 1
	}
	published := fmt.Sprintf("published as %s to %s", event, p.Kind())
	if url != "" {
		published += ": " + url
	}
	recordPublication(receipt, published)
	if url != "" {
		logf("posted %s to %s as %s: %s", filepath.Base(runDir), p.Kind(), event, url)
	} else {
		logf("posted %s to %s as %s", filepath.Base(runDir), p.Kind(), event)
	}
	return 0
}

// recordPublication writes what became of the submission into the receipt already
// claimed for it.
//
// A write failure is deliberately not reported anywhere. The claim -- the file's
// existence -- is what stops the next -post-run, and it is already on disk; the
// text only tells the operator which event went out and whether it was confirmed.
// Failing the run over it would say the publish did not happen, which by this
// point is the one thing that is certainly false.
func recordPublication(receipt *os.File, what string) {
	_, _ = fmt.Fprintln(receipt, what)
}

// alreadyPosted says why a run directory must not be published again, or returns
// "" if -post-run has never claimed it.
//
// The receipt's own text is quoted back, because the two things it can say need
// different actions from a human: a confirmed publication is on the pull request
// and there is nothing to do, while a submission that failed after it was sent may
// or may not be, and only looking can settle it.
func alreadyPosted(runDir string, pr int) string {
	path := filepath.Join(runDir, postReceipt)
	raw, err := os.ReadFile(path)
	if err != nil {
		// Including a permission error: this is only the message, and the O_EXCL claim
		// refuses just as well without being able to read what it collided with.
		return ""
	}
	what := strings.TrimSpace(string(raw))
	if what == "" {
		what = "an earlier -post-run claimed it and recorded no outcome"
	}
	return fmt.Sprintf("%s was %s; posting it again would put a second review on pull request %d. If the forge does not have it, delete %s and try again.", runDir, what, pr, path)
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
	// This run ALREADY published. `review-pr -post` leaves a summary that is
	// otherwise perfectly replayable -- review-only termination, no error -- so
	// without this the inspect-then-publish workflow applied to a run that was
	// already posted puts a second identical review on the pull request under the
	// operator's identity, and with -post-verdict a second approval. That is the
	// same duplicate-submission hazard the guarded retry above and forge's anchor
	// handling both refuse to take; recording the event is what makes it knowable,
	// so it is refused here rather than only documented.
	//
	// Checked before the termination test below so the run whose publish was
	// accepted and then failed -- ReviewPosted set, Error set -- gets told what is
	// already on the pull request instead of the generic "review again".
	case sum.ReviewPosted != "":
		return fmt.Sprintf("%s was already published as %s; posting it again would put a second review on pull request %d. If that is really what you want, the review is in review-body.md.", dir, sum.ReviewPosted, sum.PR)
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
