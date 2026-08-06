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

// The summary path is absolute, but a run directory can be copied or the project
// moved; the body beside the summary is then the right file. Both documented input
// shapes must find it -- the summary file names no directory to join under, so the
// fallback has to be resolved against the summary that was loaded, not the
// argument.
func TestPostRunFallsBackToTheBodyBesideTheSummary(t *testing.T) {
	for _, shape := range []string{"the run directory", "the summary file"} {
		t.Run(shape, func(t *testing.T) {
			dir := writeRun(t, model.RunSummary{
				Mode: "pr", PR: 3, Path: t.TempDir(),
				ReviewedHead: "0123456789abcdef0123456789abcdef01234567",
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
