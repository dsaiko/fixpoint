package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/model"
)

// writeRun lays out a finished run directory the way a real one looks.
func writeRun(t *testing.T, sum model.RunSummary, body string) string {
	t.Helper()
	return writeRunAt(t, t.TempDir(), sum, body)
}

// writeRunAt is writeRun in a directory the caller chose, for a test that cares
// WHERE the run directory sits -- inside a git work tree, say.
func writeRunAt(t *testing.T, dir string, sum model.RunSummary, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if body != "" {
		path := filepath.Join(dir, "review-body.md")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if sum.ReviewBody == "" {
			sum.ReviewBody = path
		}
	}
	raw, err := json.Marshal(sum)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "summary-20260806-120000.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The commit a replayable summary says it reviewed.
const head = "0123456789abcdef0123456789abcdef01234567"

// And the repository it reviewed it in. Publishing compares this against the
// repository the checkout at sum.Path resolves to, so a test that means to reach the
// forge has to agree with it -- see installPoster.
const reviewedRepo = "github.com/o/r"

// Publishing a finished run must refuse anything it cannot faithfully replay,
// and say which. The whole point of the mode is that what goes out is what was
// reviewed, so guessing at a missing fact would defeat it.
func TestPostRunRefusesWhatItCannotReplay(t *testing.T) {
	for _, tc := range []struct {
		name string
		sum  model.RunSummary
		body string
		want string
	}{
		{
			"a fix run has no verdict to publish",
			model.RunSummary{Mode: "pr", PR: 3},
			"body", "holds no review verdict",
		},
		{
			"a directory review has nowhere to go",
			model.RunSummary{Mode: "directory", Verdict: &model.ReviewVerdict{Outcome: model.VerdictApprove}},
			"body", "not a pull request",
		},
		{
			"a run from before the PR number was recorded",
			model.RunSummary{Mode: "pr", Verdict: &model.ReviewVerdict{Outcome: model.VerdictApprove}},
			"body", "records no pull request number",
		},
		{
			// Without it the review cannot be bound to a commit, so publishing could
			// attach the verdict to a head nobody reviewed.
			"a run that does not say which commit it reviewed",
			model.RunSummary{Mode: "pr", PR: 3, Verdict: &model.ReviewVerdict{Outcome: model.VerdictApprove}},
			"body", "does not record which commit it reviewed",
		},
		{
			// The commit says what was reviewed, not where it belongs. Without the
			// repository there is nothing to compare the checkout against, and the review
			// would go to pull request 3 of whatever repository sum.Path holds by then.
			"a run that does not say which repository it reviewed",
			model.RunSummary{
				Mode: "pr", PR: 3, ReviewedHead: head,
				Termination: model.TermReviewOnly,
				Verdict:     &model.ReviewVerdict{Outcome: model.VerdictApprove},
			},
			"body", "does not record which repository it reviewed",
		},
		{
			// The verdict and the body are written before the round checks whether it
			// was canceled, so an interrupted review leaves a complete-looking APPROVE
			// the run itself refused to post and exited non-zero over.
			"a review the operator stopped part-way",
			model.RunSummary{
				Mode: "pr", PR: 3, ReviewedHead: head, ReviewedRepo: reviewedRepo,
				Termination: model.TermInterrupted,
				Verdict:     &model.ReviewVerdict{Outcome: model.VerdictApprove},
			},
			"body", "ended as interrupted",
		},
		{
			// On a review-only run the recorded error is the publish that failed --
			// which the forge may have accepted before the client gave up.
			"a review whose own publish failed",
			model.RunSummary{
				Mode: "pr", PR: 3, ReviewedHead: head, ReviewedRepo: reviewedRepo,
				Termination: model.TermError, LoopTermination: model.TermReviewOnly,
				Error:   "post review: 502 from github",
				Verdict: &model.ReviewVerdict{Outcome: model.VerdictApprove},
			},
			"body", "ended as error (post review: 502 from github)",
		},
		{
			// Nothing but a review-only round computes a verdict, so any fix termination
			// carrying one is a summary that did not come from a completed review.
			"a fix termination carrying a verdict",
			model.RunSummary{
				Mode: "pr", PR: 3, ReviewedHead: head, ReviewedRepo: reviewedRepo,
				Termination: model.TermMaxIterations,
				Verdict:     &model.ReviewVerdict{Outcome: model.VerdictApprove},
			},
			"body", "ended as max-iterations",
		},
		{
			// The one refusal an otherwise flawless summary earns: a run that posted
			// with -post is review-only, error-free and fully recorded, so nothing else
			// here stops -post-run from putting the same review on the pull request a
			// second time under the operator's identity.
			"a review the run itself already published",
			model.RunSummary{
				Mode: "pr", PR: 3, ReviewedHead: head, ReviewedRepo: reviewedRepo,
				Termination:  model.TermReviewOnly,
				ReviewPosted: "comment",
				Verdict:      &model.ReviewVerdict{Outcome: model.VerdictApprove},
			},
			"body", "was already published as comment",
		},
		{
			// Accepted by the forge and then failed on the client -- the case
			// ReviewPosted is recorded from the URL for. The refusal must name what is
			// already on the pull request, not send the operator off to review again.
			"a publish the forge accepted before the call failed",
			model.RunSummary{
				Mode: "pr", PR: 3, ReviewedHead: head, ReviewedRepo: reviewedRepo,
				Termination: model.TermError, LoopTermination: model.TermReviewOnly,
				Error:        "confirm approval: head moved",
				ReviewPosted: "approve",
				Verdict:      &model.ReviewVerdict{Outcome: model.VerdictApprove},
			},
			"body", "was already published as approve",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs strings.Builder
			dir := writeRun(t, tc.sum, tc.body)
			if code := postRun(t.Context(), dir, false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code == 0 {
				t.Errorf("postRun(t.Context(), ) = 0, want a refusal; logs:\n%s", logs.String())
			}
			if !strings.Contains(logs.String(), tc.want) {
				t.Errorf("logs should explain the refusal (%q):\n%s", tc.want, logs.String())
			}
		})
	}
}

// A directory that is not a run at all must say so rather than fail obscurely.
func TestPostRunRejectsANonRunDirectory(t *testing.T) {
	var logs strings.Builder
	if code := postRun(t.Context(), t.TempDir(), false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code == 0 {
		t.Error("postRun(t.Context(), ) = 0 for a directory holding no summary")
	}
	if !strings.Contains(logs.String(), "is it a run directory") {
		t.Errorf("unhelpful message:\n%s", logs.String())
	}
}

// The body published is the one beside the summary, whatever the summary's own
// review_body says: the recorded path is absolute, so a copied run directory or a
// moved project makes it wrong. Both documented input shapes must find it -- the
// summary file names no directory to join under, so the body has to be resolved
// against the summary that was loaded, not the argument.
func TestPostRunReadsTheBodyBesideTheSummary(t *testing.T) {
	for _, shape := range []string{"the run directory", "the summary file"} {
		t.Run(shape, func(t *testing.T) {
			dir := writeRun(t, model.RunSummary{
				Mode: "pr", PR: 3, Path: t.TempDir(),
				ReviewedHead: head,
				ReviewedRepo: reviewedRepo,
				Termination:  model.TermReviewOnly,
				ReviewBody:   "/gone/review-body.md",
				Verdict:      &model.ReviewVerdict{Outcome: model.VerdictApprove},
			}, "the review that was actually produced")
			arg := dir
			if shape == "the summary file" {
				arg = filepath.Join(dir, "summary-20260806-120000.json")
			}

			var logs strings.Builder
			// No GitHub remote in that temp path, so this stops at the poster -- which is
			// past the body read, which is what this test is about.
			postRun(t.Context(), arg, false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) })
			if strings.Contains(logs.String(), "cannot read the review body") {
				t.Errorf("the fallback to the body beside the summary did not happen:\n%s", logs.String())
			}
			if !strings.Contains(logs.String(), "no GitHub or GitLab remote") {
				t.Errorf("expected to get as far as resolving the forge:\n%s", logs.String())
			}
		})
	}
}

// A run directory is only files, so a pull request can commit one: a summary whose
// review_body names a local secret, with a plausible review-body.md beside it for
// the operator to read before publishing. Neither the recorded path nor a symlink
// standing in for the canonical name may decide what goes to the pull request.
func TestPostRunPublishesNothingButTheFileBesideTheSummary(t *testing.T) {
	secret := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(secret, []byte("aws_secret_access_key = hunter2"), 0o600); err != nil {
		t.Fatal(err)
	}
	base := model.RunSummary{
		Mode: "pr", PR: 3, Path: t.TempDir(),
		ReviewedHead: head,
		ReviewedRepo: reviewedRepo,
		Termination:  model.TermReviewOnly,
		ReviewBody:   secret,
		Verdict:      &model.ReviewVerdict{Outcome: model.VerdictApprove},
	}

	t.Run("the path in the summary is not read", func(t *testing.T) {
		// No review-body.md beside the summary, and a readable file named by the
		// summary: reading it would be the only way to get past this point.
		dir := writeRun(t, base, "")
		var logs strings.Builder
		if code := postRun(t.Context(), dir, false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code == 0 {
			t.Errorf("postRun(t.Context(), ) = 0 with no review body beside the summary; logs:\n%s", logs.String())
		}
		if !strings.Contains(logs.String(), "cannot read the review body") {
			t.Errorf("the recorded path was read instead of refusing:\n%s", logs.String())
		}
	})

	t.Run("a symlink at the canonical name is refused", func(t *testing.T) {
		dir := writeRun(t, base, "")
		if err := os.Symlink(secret, filepath.Join(dir, "review-body.md")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		var logs strings.Builder
		if code := postRun(t.Context(), dir, false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code == 0 {
			t.Errorf("postRun(t.Context(), ) = 0 for a symlinked review body; logs:\n%s", logs.String())
		}
		if !strings.Contains(logs.String(), "not a regular file") {
			t.Errorf("the symlink was followed rather than refused:\n%s", logs.String())
		}
	})
}

// The same committed lookalike has a second way to act, and the body read does
// not touch it: everything else -post-run does comes out of the summary JSON. A
// directory that arrived with the code under review can aim the submission at
// another pull request in the repository, name that request's public head so the
// head check passes, ask for an approval, and supply its own line comments -- and
// the operator only has to pick the wrong timestamped sibling out of .fixpoint/.
//
// Nothing can authenticate a run directory, but git can say whether this one is a
// repository's content. fixpoint commits no run artifacts, so a tracked file here
// means it is not the record of a run on this machine.
func TestPostRunRefusesARunDirectoryThatCameInWithTheCheckout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git("init")
	// Committed by the pull request, so it sits exactly where the operator's own runs
	// do and its timestamp is one they might mistake for theirs.
	planted := filepath.Join(repo, ".fixpoint", "20260807-153512")
	sum := replayable(t, model.VerdictApprove, []model.ReviewAnchor{
		{Path: "internal/forge/forge.go", Line: 42, Body: "text nobody read in review-body.md"},
	})
	sum.PR = 99 // a pull request nothing on this machine reviewed
	dir := writeRunAt(t, planted, sum, "a plausible review for the operator to read")
	git("add", "-f", ".fixpoint")

	p := &fakePoster{url: "https://github.com/o/r/pull/99#pullrequestreview-1"}
	installPoster(t, p)
	var logs strings.Builder
	if code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code != 2 {
		t.Errorf("postRun(t.Context(), ) = %d for a tracked run directory, want the refusal 2; logs:\n%s", code, logs.String())
	}
	if len(p.calls) != 0 {
		t.Errorf("PostReview called %d times, want 0: an approval was published from a committed directory", len(p.calls))
	}
	if !strings.Contains(logs.String(), "tracked by git") {
		t.Errorf("the refusal should say why the directory is not trusted:\n%s", logs.String())
	}
	// And the operator's own run in the same work tree still publishes: the check is
	// about tracked-ness, not about living inside a checkout, which is where every
	// run directory lives.
	mine := writeRunAt(t, filepath.Join(repo, ".fixpoint", "20260807-160000"), replayable(t, model.VerdictApprove, nil), "the review this machine produced")
	logs.Reset()
	if code := postRun(t.Context(), mine, true, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code != 0 {
		t.Fatalf("postRun(t.Context(), ) = %d for an untracked run directory in a work tree, want 0; logs:\n%s", code, logs.String())
	}
	if len(p.calls) != 1 {
		t.Errorf("PostReview called %d times for the operator's own run, want 1", len(p.calls))
	}
}

// postCall is one submission a fake poster received.
type postCall struct {
	dir    string
	pr     int
	head   string
	body   string
	event  forge.Event
	inline []forge.InlineComment
}

// fakePoster records what would have gone to the forge, and fails the calls it is
// told to: errs[i] is returned from submission i, so an anchor rejection followed
// by a success is expressible, and so is a failure that must NOT be retried.
//
// mu guards calls, because one test drives two -post-run publications at once and
// a poster that lost a concurrent submission to a data race would report the
// duplicate it is there to catch as a single call.
type fakePoster struct {
	mu    sync.Mutex
	calls []postCall
	errs  []error
	url   string
}

func (*fakePoster) Kind() forge.Kind { return forge.GitHub }

func (*fakePoster) Checks(context.Context, string, int, string) (forge.Checks, error) {
	return forge.Checks{}, nil
}

func (f *fakePoster) PostReview(_ context.Context, dir string, pr int, head, body string, event forge.Event, inline []forge.InlineComment) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, postCall{dir, pr, head, body, event, inline})
	if n := len(f.calls) - 1; n < len(f.errs) && f.errs[n] != nil {
		return "", f.errs[n]
	}
	return f.url, nil
}

// installPoster points -post-run's forge lookup at p for one test.
//
// It answers the repository lookup too, with the identity replayable records. Both
// ask the same question of the same checkout -- "what is at sum.Path?" -- and sum.Path
// in these tests is an empty temp directory, so a test that left the second one real
// would exercise the refusal in front of the submission rather than the submission.
// A test about that refusal calls installRepoID afterwards to disagree on purpose.
func installPoster(t *testing.T, p forge.Poster) {
	t.Helper()
	prev := posterFor
	posterFor = func(context.Context, string) forge.Poster { return p }
	t.Cleanup(func() { posterFor = prev })
	installRepoID(t, reviewedRepo)
}

// installRepoID makes the checkout at sum.Path resolve to id for one test, as
// forge.RepoID would have resolved it from a real one.
func installRepoID(t *testing.T, id string) {
	t.Helper()
	prev := repoIDFor
	repoIDFor = func(context.Context, string) string { return id }
	t.Cleanup(func() { repoIDFor = prev })
}

// replayable is a summary -post-run will publish, so a test can vary the one
// field it is about.
func replayable(t *testing.T, outcome string, anchors []model.ReviewAnchor) model.RunSummary {
	t.Helper()
	return model.RunSummary{
		Mode: "pr", PR: 3, Path: t.TempDir(),
		ReviewedHead: head,
		ReviewedRepo: reviewedRepo,
		Termination:  model.TermReviewOnly,
		ReviewInline: anchors,
		Verdict:      &model.ReviewVerdict{Outcome: outcome},
	}
}

// What reaches the forge is what the run produced: the bytes on disk, the commit
// it reviewed, its own anchors, and the event its outcome maps to. This is the
// promise the mode is FOR, and it is made in the one call no log line can show.
//
// The event mapping is the consequential half. Approving a change the panel did
// not clear, or blocking one it did, is a social act taken under the operator's
// identity -- and both are one swapped case away.
func TestPostRunPublishesTheRunItReplays(t *testing.T) {
	anchors := []model.ReviewAnchor{
		{Path: "internal/forge/forge.go", Line: 42, Body: "the anchor the run computed"},
		{Path: "cmd/fixpoint/main.go", Line: 7, Body: "and the second one"},
	}
	for _, tc := range []struct {
		name        string
		outcome     string
		postVerdict bool
		want        forge.Event
	}{
		{"approve alone is only a comment", model.VerdictApprove, false, forge.Comment},
		{"changes-requested alone is only a comment", model.VerdictChangesRequested, false, forge.Comment},
		{"approve with -post-verdict approves", model.VerdictApprove, true, forge.EventApprove},
		{"changes-requested with -post-verdict blocks", model.VerdictChangesRequested, true, forge.EventRequestChanges},
		{"inconclusive is a comment under every flag", model.VerdictInconclusive, true, forge.Comment},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const body = "the review that was actually produced"
			sum := replayable(t, tc.outcome, anchors)
			dir := writeRun(t, sum, body)
			p := &fakePoster{url: "https://github.com/o/r/pull/3#pullrequestreview-1"}
			installPoster(t, p)

			var logs strings.Builder
			code := postRun(t.Context(), dir, tc.postVerdict, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) })
			if code != 0 {
				t.Fatalf("postRun(t.Context(), ) = %d, want 0; logs:\n%s", code, logs.String())
			}
			if len(p.calls) != 1 {
				t.Fatalf("PostReview called %d times, want 1", len(p.calls))
			}
			got := p.calls[0]
			if got.event != tc.want {
				t.Errorf("event = %q, want %q", got.event, tc.want)
			}
			if got.body != body {
				t.Errorf("body = %q, want the bytes on disk %q", got.body, body)
			}
			if got.head != head {
				t.Errorf("head = %q, want the reviewed head %q", got.head, head)
			}
			if got.pr != sum.PR || got.dir != sum.Path {
				t.Errorf("posted to (%s, #%d), want (%s, #%d)", got.dir, got.pr, sum.Path, sum.PR)
			}
			want := []forge.InlineComment{
				{Path: "internal/forge/forge.go", Line: 42, Body: "the anchor the run computed"},
				{Path: "cmd/fixpoint/main.go", Line: 7, Body: "and the second one"},
			}
			if !reflect.DeepEqual(got.inline, want) {
				t.Errorf("inline comments = %+v, want the run's own anchors %+v", got.inline, want)
			}
			if !strings.Contains(logs.String(), string(tc.want)) || !strings.Contains(logs.String(), p.url) {
				t.Errorf("the log should name the event and the URL:\n%s", logs.String())
			}
		})
	}
}

// The destination is the repository that was REVIEWED, not whatever repository the
// recorded path holds when the replay runs.
//
// sum.Path is a path, and both the poster lookup and the submission resolve their
// repository from the checkout occupying it: gh's {owner}/{repo} come from that
// directory. Days can pass between a run and its -post-run -- that delay is the whole
// point of the mode -- and in them the directory can be reused for another repository
// on the same forge, have its remotes rewritten, or be a copied run directory whose
// absolute path now names somebody else's tree. The head check does not notice, since
// the reviewed commit is public and anyone may open a pull request proposing it: with
// -post-verdict that is an approval on a pull request nobody reviewed, under the
// operator's identity.
func TestPostRunPublishesOnlyToTheRepositoryItReviewed(t *testing.T) {
	for _, tc := range []struct {
		name  string
		now   string
		calls int
		want  string
	}{
		{
			"another repository on the same forge",
			"github.com/o/other", 0, "is now a checkout of github.com/o/other",
		},
		{
			// Fails closed like requireHead: unanswered is not a license to publish, and
			// the identity comes from the same gh that would have done the posting.
			"a checkout whose repository cannot be established",
			"", 0, "cannot establish which repository",
		},
		{
			// And it must not false-alarm on the ordinary case, which would break the
			// mode for everybody: GitHub reads an owner and a name case-insensitively.
			"the same repository, spelled differently",
			"GitHub.com/O/R", 1, "publishing to github",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeRun(t, replayable(t, model.VerdictApprove, nil), "the review that was actually produced")
			p := &fakePoster{url: "https://github.com/o/r/pull/3#pullrequestreview-1"}
			installPoster(t, p)
			installRepoID(t, tc.now)

			var logs strings.Builder
			code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) })
			if tc.calls == 0 && code != 2 {
				t.Errorf("postRun(t.Context(), ) = %d, want the refusal 2; logs:\n%s", code, logs.String())
			}
			if tc.calls == 1 && code != 0 {
				t.Errorf("postRun(t.Context(), ) = %d, want 0; logs:\n%s", code, logs.String())
			}
			if len(p.calls) != tc.calls {
				t.Errorf("PostReview called %d times, want %d", len(p.calls), tc.calls)
			}
			if !strings.Contains(logs.String(), tc.want) {
				t.Errorf("logs should say %q:\n%s", tc.want, logs.String())
			}
			// A refusal that published nothing must leave the run publishable from the
			// right checkout: the receipt is a claim on a submission that was attempted.
			if _, err := os.Stat(filepath.Join(dir, postReceipt)); tc.calls == 0 && err == nil {
				t.Error("the run was claimed by a replay that never submitted anything")
			}
		})
	}
}

// -post-run is the mode an operator runs by hand, from a shell, over a directory
// that stays on disk for as long as the logs do. Nothing in the summary changes
// when it publishes -- the summary was written when the run ended -- so without a
// receipt of its own, running it twice is a second identical review on somebody's
// pull request, or a second approval, under the operator's identity. That is the
// same duplicate submission the guarded retry and the ReviewPosted refusal already
// exist to prevent, arrived at by the most ordinary route there is: pressing up
// and enter.
func TestPostRunPublishesARunOnlyOnce(t *testing.T) {
	dir := writeRun(t, replayable(t, model.VerdictApprove, nil), "the review that was actually produced")
	p := &fakePoster{url: "https://github.com/o/r/pull/3#pullrequestreview-1"}
	installPoster(t, p)

	var first strings.Builder
	if code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&first, f+"\n", a...) }); code != 0 {
		t.Fatalf("first postRun(t.Context(), ) = %d, want 0; logs:\n%s", code, first.String())
	}

	var second strings.Builder
	code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&second, f+"\n", a...) })
	if code == 0 {
		t.Errorf("second postRun(t.Context(), ) = 0, want a refusal; logs:\n%s", second.String())
	}
	if len(p.calls) != 1 {
		t.Errorf("PostReview called %d times, want 1: the review reached the forge twice", len(p.calls))
	}
	// The refusal has to name the event and the URL that are already on the pull
	// request: "already published" alone sends the operator looking for it.
	if !strings.Contains(second.String(), string(forge.EventApprove)) || !strings.Contains(second.String(), p.url) {
		t.Errorf("the refusal should say what is already on the pull request:\n%s", second.String())
	}
}

// The sequential test above proves the receipt REFUSES a replay; it does not prove
// the claim is indivisible. An implementation that read the receipt and then created
// it would pass that test just as well, and lose to two -post-run invocations that
// overlap -- pressing up and enter in two terminals, or the same command in two panes
// of a tmux session -- putting two reviews, or two approvals, on one pull request.
// That is why the claim is a single O_WRONLY|O_CREATE|O_EXCL open and not a check
// followed by a create.
//
// The rendezvous is repoIDFor, the last hook postRun calls before the claim: holding
// every replay there until all of them arrive puts them at the open together, past
// the early alreadyPosted check that would otherwise settle the losers before the
// winner had claimed anything. It SPINS rather than parking on a channel, because the
// window a non-atomic claim leaves open is two syscalls wide and the latency of
// waking a parked goroutine is enough to miss it. For the same reason the race is run
// several times over fresh directories: one round caught a deliberately reverted
// check-then-create about half the time, and rounds compound.
//
// Every wait here is bounded. A replay that refuses ahead of the barrier never
// reaches it, and a test that hangs reports nothing at all.
func TestPostRunPublishesARunOnlyOnceUnderSimultaneousReplays(t *testing.T) {
	const (
		replays = 3
		rounds  = 8
	)
	for round := range rounds {
		sum := replayable(t, model.VerdictApprove, nil)
		dir := writeRun(t, sum, "the review that was actually produced")
		p := &fakePoster{url: "https://github.com/o/r/pull/3#pullrequestreview-1"}
		installPoster(t, p)

		var arrived atomic.Int32
		prevRepoID := repoIDFor
		repoIDFor = func(context.Context, string) string {
			arrived.Add(1)
			for deadline := time.Now().Add(10 * time.Second); arrived.Load() < replays; {
				if time.Now().After(deadline) {
					break
				}
				runtime.Gosched()
			}
			return reviewedRepo
		}

		codes := make([]int, replays)
		logs := make([]strings.Builder, replays)
		var wg sync.WaitGroup
		for i := range replays {
			wg.Add(1)
			go func() {
				defer wg.Done()
				codes[i] = postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&logs[i], f+"\n", a...) })
			}()
		}
		wg.Wait()
		repoIDFor = prevRepoID

		// Exactly one replay may reach the forge, and every other one must say the run is
		// already claimed rather than report a success it did not have.
		var published, refused int
		var transcript strings.Builder
		for i, code := range codes {
			fmt.Fprintf(&transcript, "replay %d exited %d:\n%s\n", i, code, logs[i].String())
			switch code {
			case 0:
				published++
			case 2:
				refused++
				// The loser reads a receipt the winner may not have filled in yet, so the
				// wording varies; which directory and which pull request do not, and they are
				// the whole of what the operator can act on.
				for _, want := range []string{dir, strconv.Itoa(sum.PR)} {
					if !strings.Contains(logs[i].String(), want) {
						t.Errorf("round %d: the refusal should name %q:\n%s", round, want, logs[i].String())
					}
				}
			default:
				t.Errorf("round %d: postRun(t.Context(), ) = %d, want 0 or the refusal 2; logs:\n%s", round, code, logs[i].String())
			}
		}
		if published != 1 || refused != replays-1 {
			t.Errorf("round %d: %d replays published and %d refused, want 1 and %d:\n%s",
				round, published, refused, replays-1, transcript.String())
		}
		if n := len(p.calls); n != 1 {
			t.Fatalf("round %d: PostReview called %d times, want 1: the claim let simultaneous replays through:\n%s",
				round, n, transcript.String())
		}
	}
}

// A submission that failed on the client may have been ACCEPTED by the forge
// first -- the timeout case the guarded retry refuses to retry within one run. The
// receipt has to survive that, or the very next -post-run becomes exactly the
// retry that was just refused, only with a human's finger on it.
func TestPostRunDoesNotReplayAFailedSubmission(t *testing.T) {
	dir := writeRun(t, replayable(t, model.VerdictApprove, nil), "the review that was actually produced")
	p := &fakePoster{errs: []error{context.DeadlineExceeded}}
	installPoster(t, p)

	var first strings.Builder
	if code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&first, f+"\n", a...) }); code != 1 {
		t.Fatalf("first postRun(t.Context(), ) = %d, want 1; logs:\n%s", code, first.String())
	}

	var second strings.Builder
	if code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&second, f+"\n", a...) }); code == 0 {
		t.Errorf("second postRun(t.Context(), ) = 0 after a submission that may have been accepted; logs:\n%s", second.String())
	}
	if len(p.calls) != 1 {
		t.Errorf("PostReview called %d times, want 1", len(p.calls))
	}
	// Not "published": the operator has to be told the outcome is unknown, since
	// what they do about it -- go and look -- is different from doing nothing.
	if !strings.Contains(second.String(), "may or may not have been accepted") {
		t.Errorf("the refusal should say the outcome is unknown:\n%s", second.String())
	}
}

// The receipt is both the record and the lock, and the two roles can disagree: the
// O_EXCL claim collides with a receipt whose TEXT cannot be read. alreadyPosted
// answers "" there -- it only ever produces the message, and it treats unreadable
// as nothing-to-quote -- so without the fallback the collision is reported as a
// bare "post-run: " and the operator is told nothing at all, about a pull request
// that may already carry the review.
//
// A directory at the receipt's name is that state exactly, and it needs no chmod
// (which root would defeat): os.ReadFile fails on it while open with
// O_CREATE|O_EXCL still reports ErrExist.
func TestPostRunExplainsAnUnreadableReceipt(t *testing.T) {
	sum := replayable(t, model.VerdictApprove, nil)
	dir := writeRun(t, sum, "the review that was actually produced")
	receipt := filepath.Join(dir, postReceipt)
	if err := os.Mkdir(receipt, 0o700); err != nil {
		t.Fatal(err)
	}
	p := &fakePoster{url: "https://github.com/o/r/pull/3#pullrequestreview-1"}
	installPoster(t, p)

	var logs strings.Builder
	code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) })
	if code != 2 {
		t.Errorf("postRun(t.Context(), ) = %d, want the refusal 2; logs:\n%s", code, logs.String())
	}
	if len(p.calls) != 0 {
		t.Errorf("PostReview called %d times over a run that may already have been published: %+v", len(p.calls), p.calls)
	}
	if strings.Contains(logs.String(), "post-run: \n") {
		t.Errorf("the collision was reported with no explanation at all:\n%s", logs.String())
	}
	// Which directory, which file to delete, and which pull request to go and look
	// at: those three are the whole of what the operator can act on.
	for _, want := range []string{dir, receipt, strconv.Itoa(sum.PR)} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("the refusal should name %q:\n%s", want, logs.String())
		}
	}
}

// Nothing that refuses before the forge is reached may leave the run unpublishable:
// a missing body or an unrecognized remote created nothing anywhere, and a
// directory blocked by one of them would have to be re-reviewed to be posted at all.
func TestPostRunClaimsNothingWhenItNeverSubmits(t *testing.T) {
	sum := replayable(t, model.VerdictApprove, nil)
	dir := writeRun(t, sum, "the review that was actually produced")
	installPoster(t, nil)
	var logs strings.Builder
	if code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code == 0 {
		t.Fatalf("postRun(t.Context(), ) = 0 with no forge; logs:\n%s", logs.String())
	}

	p := &fakePoster{url: "https://github.com/o/r/pull/3#pullrequestreview-1"}
	installPoster(t, p)
	logs.Reset()
	if code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code != 0 {
		t.Fatalf("postRun(t.Context(), ) = %d after a refusal that published nothing, want 0; logs:\n%s", code, logs.String())
	}
	if len(p.calls) != 1 {
		t.Errorf("PostReview called %d times, want 1", len(p.calls))
	}
}

// A second submission is earned only by an anchor rejection, which created
// nothing on the forge. Every other failure may have been ACCEPTED before the
// client saw it, so retrying is how a pull request collects two identical reviews
// under the operator's identity. Nothing else in the suite protects this copy of
// the rule: the orchestrator's own test covers the orchestrator's copy.
func TestPostRunRetriesOnlyWhenTheAnchorsWereRejected(t *testing.T) {
	for _, tc := range []struct {
		name  string
		err   error
		calls int
		code  int
		want  string
	}{
		{
			"a rejected anchor is retried without the inline comments",
			forge.RejectedAnchors(errors.New("line 42 is not part of the diff")),
			2, 0, "rejected the inline comments",
		},
		{
			"an auth failure is reported, not retried",
			errors.New("gh: HTTP 401 Bad credentials"),
			1, 1, "publishing to github failed",
		},
		{
			// The case that costs something: the forge may already hold the review.
			"a timeout is reported, not retried",
			context.DeadlineExceeded,
			1, 1, "publishing to github failed",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeRun(t, replayable(t, model.VerdictApprove, []model.ReviewAnchor{
				{Path: "internal/forge/forge.go", Line: 42, Body: "an anchor the forge may not accept"},
			}), "the review that was actually produced")
			p := &fakePoster{errs: []error{tc.err}}
			installPoster(t, p)

			var logs strings.Builder
			code := postRun(t.Context(), dir, true, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) })
			if code != tc.code {
				t.Errorf("postRun(t.Context(), ) = %d, want %d; logs:\n%s", code, tc.code, logs.String())
			}
			if len(p.calls) != tc.calls {
				t.Fatalf("PostReview called %d times, want %d", len(p.calls), tc.calls)
			}
			if tc.calls == 2 {
				if p.calls[1].inline != nil {
					t.Errorf("the retry carried anchors again: %+v", p.calls[1].inline)
				}
				if p.calls[1].body != p.calls[0].body || p.calls[1].event != p.calls[0].event {
					t.Errorf("the retry changed the review: %+v then %+v", p.calls[0], p.calls[1])
				}
			}
			if !strings.Contains(logs.String(), tc.want) {
				t.Errorf("logs should say %q:\n%s", tc.want, logs.String())
			}
		})
	}
}
