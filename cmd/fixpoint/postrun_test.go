package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			if code := postRun(dir, false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code == 0 {
				t.Errorf("postRun() = 0, want a refusal; logs:\n%s", logs.String())
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
	if code := postRun(t.TempDir(), false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code == 0 {
		t.Error("postRun() = 0 for a directory holding no summary")
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
			postRun(arg, false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) })
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
		if code := postRun(dir, false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code == 0 {
			t.Errorf("postRun() = 0 with no review body beside the summary; logs:\n%s", logs.String())
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
		if code := postRun(dir, false, func(f string, a ...any) { fmt.Fprintf(&logs, f+"\n", a...) }); code == 0 {
			t.Errorf("postRun() = 0 for a symlinked review body; logs:\n%s", logs.String())
		}
		if !strings.Contains(logs.String(), "not a regular file") {
			t.Errorf("the symlink was followed rather than refused:\n%s", logs.String())
		}
	})
}
