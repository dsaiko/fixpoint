package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/gitenv"
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

// repoIDFor is forge.RepoID behind a variable for the same reason, and it has to be
// one: the check it feeds refuses to publish, so a test that could not answer it
// would be testing the refusal rather than the submission.
var repoIDFor = forge.RepoID

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
//
// The inspected file covers ONE of the two channels out. The line comments are
// separate bytes, taken from the summary, so an operator who read review-body.md
// has not read them -- which is why the directory is checked for having arrived
// with the code under review (committedRun), and why every anchor is named in the
// log before anything is submitted.
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
	// The directory must also be the OPERATOR's own run, not one that arrived with
	// the code under review.
	//
	// Everything from here on is taken from the summary JSON, and only the body is
	// hardened against it (below): sum.PR and sum.Path choose which pull request in
	// which checkout the submission lands on, sum.ReviewedHead is what the poster's
	// head check compares against, sum.Verdict.Outcome is what becomes APPROVE, and
	// sum.ReviewInline carries the full text of every line comment. So the committed
	// lookalike the body read already refuses to trust has a second way to act: point
	// pr at another pull request in the same repository, set reviewed_head to that
	// request's public head so the head check passes, set the verdict to approve --
	// and an approval lands, under the operator's identity, on a change no agent read,
	// which is the exact outcome the head check exists to prevent. The line comments
	// ride along, arbitrary text in a channel review-body.md does not show.
	//
	// It only takes picking the wrong timestamped sibling out of .fixpoint/, which is
	// one shell glob or one mistaken tab-completion away.
	if why := committedRun(ctx, runDir); why != "" {
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
	// Which FORGE is now known; which REPOSITORY on it is a separate question, and
	// sum.Path cannot answer it. Refused before the receipt is claimed, so a run
	// blocked here can still be published from the right checkout afterwards.
	if why := wrongRepository(ctx, sum); why != "" {
		logf("post-run: %s", why)
		return 2
	}
	// The same mapping the run itself would have used, from the same function --
	// a replay that approved what the live path would have commented on would be a
	// different review from the one the operator inspected.
	event := forge.EventFor(sum.Verdict.Outcome, postVerdict)
	inline := make([]forge.InlineComment, 0, len(sum.ReviewInline))
	for _, a := range sum.ReviewInline {
		inline = append(inline, forge.InlineComment{Path: a.Path, Line: a.Line, Body: a.Body})
	}
	// Said out loud before it goes out, because this half was never inspected.
	// review-body.md is the summary comment and nothing else -- RenderInline produced
	// separate bytes for each line comment, and they come off the summary, not the
	// file the operator was invited to read. Naming the destination and every anchor
	// is what lets them notice a submission aimed somewhere they did not expect, or
	// comments on files this review has no business touching.
	logf("post-run: publishing to %s pull request %d at %s as %s, with %d inline comment(s)", p.Kind(), sum.PR, sum.Path, event, len(inline))
	for _, a := range inline {
		logf("post-run: inline comment on %s:%d, from the summary rather than review-body.md", a.Path, a.Line)
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

// committedRun says why a run directory must not be trusted to name its own
// destination, or "" when nothing marks it as content that came in with a
// checkout.
//
// A run directory cannot be authenticated -- it is only files, and every field a
// real one holds can be typed into a fake one. But the shape that carries the
// attack has one property fixpoint's own output never has: it arrived through git,
// so its files are TRACKED. fixpoint writes run artifacts into an ignored
// directory and commits none of them, so a tracked file here means this directory
// is part of some repository's content rather than the record of a run on this
// machine -- and a pull request's content is exactly what must not be allowed to
// choose a pull request, a verdict, and a set of line comments.
//
// Asked of git rather than guessed from the files, because tracked-ness IS the
// question. Permissions, timestamps and plausible contents are all things a
// committed directory can have.
//
// Best-effort in the permissive direction: no git on PATH, no repository, a
// worktree the directory falls outside of all leave the answer empty. They mean
// nothing was learned, and refusing every run whose provenance cannot be
// established would break the mode wherever git is absent -- while the check still
// covers the case that produced the hazard, a directory that came in with the code
// under review, where git is present by construction.
func committedRun(ctx context.Context, runDir string) string {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// The same config pins every other git command fixpoint runs against a checkout
	// it does not trust: this one runs INSIDE that checkout, and core.fsmonitor or
	// core.hooksPath out of its .git/config would otherwise have git execute a program
	// the repository chose, with this process's environment.
	cmd := exec.CommandContext(ctx, "git", append(gitenv.SafeConfigArgs(), "ls-files", "-z", "--", ".")...)
	cmd.Dir = runDir
	// nil is this process's environment, hardened -- so the pins reach the git
	// processes git itself starts, which never see the -c flags above.
	cmd.Env = gitenv.Harden(nil)
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	tracked := string(out)
	if i := strings.IndexByte(tracked, 0); i >= 0 {
		tracked = tracked[:i]
	}
	return fmt.Sprintf("%s is tracked by git (%s is committed), so it came in with a repository's content rather than from a run on this machine. Its summary -- not review-body.md -- chooses the pull request, the verdict and the inline comments that would be published under your identity, so it is refused. Publish your own run's directory, or move this one outside the work tree if it really is yours.", runDir, tracked)
}

// wrongRepository says why the checkout the submission would go through is not the
// repository the run reviewed, or "" when it demonstrably is.
//
// It closes the last thing the summary chooses that nothing checked. sum.Path is a
// PATH, and posterFor and PostReview both resolve the destination repository from
// whatever checkout occupies it now: gh's own {owner}/{repo} placeholders come from
// that directory. Between a run and a replay -- which may be days, the workflow this
// mode exists for -- that directory can be reused for another repository on the same
// forge, have its remotes rewritten, or simply be a copied run directory whose
// recorded path now names somebody else's tree. The review then lands on pull request
// sum.PR of a repository no agent read, and the head check does not catch it: it
// compares commit SHAs, and the reviewed commit is public, so a request proposing it
// can be opened by anyone. With -post-verdict that is an approval, under the
// operator's identity, on a pull request that was never reviewed.
//
// FAILS CLOSED, like requireHead and for the same reason: "which repository does this
// land on?" unanswered is not a license to publish. It costs nothing in practice --
// the identity comes from the same gh that would do the posting, so a call that cannot
// name the repository could not have submitted either.
//
// It is a check and the submission is a separate round trip, so it does not close a
// rewrite of the checkout's remotes in between -- that residue is the same shape as
// requireHead's window, and it needs write access to the operator's own checkout at
// that instant rather than a run directory or a pull request.
func wrongRepository(ctx context.Context, sum *model.RunSummary) string {
	now := repoIDFor(ctx, sum.Path)
	if now == "" {
		return fmt.Sprintf("cannot establish which repository %s is a checkout of, so this review cannot be bound to the one it reviewed (%s); the submission would go wherever that directory points now, so it is refused", sum.Path, sum.ReviewedRepo)
	}
	if !strings.EqualFold(now, sum.ReviewedRepo) {
		return fmt.Sprintf("this run reviewed %s, but %s is now a checkout of %s -- publishing would put the review, and any verdict in it, on pull request %d of a repository nobody reviewed. Replay it from a checkout of %s.", sum.ReviewedRepo, sum.Path, now, sum.PR, sum.ReviewedRepo)
	}
	return ""
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
	// WHICH REPOSITORY that commit is in. A commit does not name a destination, and
	// everything else the replay has -- a path and a number -- describes wherever that
	// path points at publication time. Without this there is nothing to compare the
	// checkout against, so wrongRepository could only ever pass, and the review would
	// go to pull request PR of whatever repository the directory holds by then. Refused
	// here with the same answer as the missing head above, since it is the same
	// situation: a fact the run did not record, which a replay must not invent.
	case sum.ReviewedRepo == "":
		return dir + " does not record which repository it reviewed, so the review cannot be bound to one -- only the path it was run in, which may hold a different repository by now. Runs from before that was recorded cannot be replayed; review again to produce one that can."
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
