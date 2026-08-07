package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/forge"
	"github.com/dsaiko/fixpoint/internal/model"
)

// writeRun lays out a finished run directory the way a real one looks.
func writeRun(t *testing.T, sum model.RunSummary, body string) string {
	t.Helper()
	dir := t.TempDir()
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
			// The verdict and the body are written before the round checks whether it
			// was canceled, so an interrupted review leaves a complete-looking APPROVE
			// the run itself refused to post and exited non-zero over.
			"a review the operator stopped part-way",
			model.RunSummary{
				Mode: "pr", PR: 3, ReviewedHead: head,
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
				Mode: "pr", PR: 3, ReviewedHead: head,
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
				Mode: "pr", PR: 3, ReviewedHead: head,
				Termination: model.TermMaxIterations,
				Verdict:     &model.ReviewVerdict{Outcome: model.VerdictApprove},
			},
			"body", "ended as max-iterations",
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
type fakePoster struct {
	calls []postCall
	errs  []error
	url   string
}

func (*fakePoster) Kind() forge.Kind { return forge.GitHub }

func (*fakePoster) Checks(context.Context, string, int) (forge.Checks, error) {
	return forge.Checks{}, nil
}

func (f *fakePoster) PostReview(_ context.Context, dir string, pr int, head, body string, event forge.Event, inline []forge.InlineComment) (string, error) {
	f.calls = append(f.calls, postCall{dir, pr, head, body, event, inline})
	if n := len(f.calls) - 1; n < len(f.errs) && f.errs[n] != nil {
		return "", f.errs[n]
	}
	return f.url, nil
}

// installPoster points -post-run's forge lookup at p for one test.
func installPoster(t *testing.T, p forge.Poster) {
	t.Helper()
	prev := posterFor
	posterFor = func(context.Context, string) forge.Poster { return p }
	t.Cleanup(func() { posterFor = prev })
}

// replayable is a summary -post-run will publish, so a test can vary the one
// field it is about.
func replayable(t *testing.T, outcome string, anchors []model.ReviewAnchor) model.RunSummary {
	t.Helper()
	return model.RunSummary{
		Mode: "pr", PR: 3, Path: t.TempDir(),
		ReviewedHead: head,
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
