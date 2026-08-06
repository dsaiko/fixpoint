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
	if sum.Verdict == nil {
		logf("post-run: %s holds no review verdict -- a fix run has nothing to publish", dir)
		return 2
	}
	if sum.Mode != "pr" {
		logf("post-run: %s reviewed %s, not a pull request; there is nowhere to post it", dir, sum.Mode)
		return 2
	}
	if sum.PR <= 0 {
		logf("post-run: %s records no pull request number. Runs from before that was recorded cannot be replayed; review again to produce one that can.", dir)
		return 2
	}
	// Which commit the review is ABOUT. Refused here rather than left to the poster
	// so the operator gets the same "review again" answer as the missing PR number
	// above: a summary that cannot say what it reviewed cannot be published against
	// anything. The poster refuses too, and additionally refuses when the pull
	// request has moved since -- replaying a verdict onto a commit the panel never
	// read is the reason this is checked at all.
	if sum.ReviewedHead == "" {
		logf("post-run: %s does not record which commit it reviewed, so the review cannot be bound to one. Runs from before that was recorded cannot be replayed; review again to produce one that can.", dir)
		return 2
	}
	body, err := os.ReadFile(sum.ReviewBody)
	if err != nil {
		// The path is recorded as absolute, but a run directory can be copied or the
		// project moved, so fall back to the file beside the summary. Beside the
		// SUMMARY, not beside the argument: the argument may name the summary file
		// itself, and joining under a file path can only fail.
		alt := filepath.Join(runDir, "review-body.md")
		body, err = os.ReadFile(alt)
		if err != nil {
			logf("post-run: cannot read the review body (%s or %s): %v", sum.ReviewBody, alt, err)
			return 1
		}
	}

	ctx := context.Background()
	p := forge.PosterFor(ctx, sum.Path)
	if p == nil {
		logf("post-run: no GitHub or GitLab remote recognized at %s", sum.Path)
		return 1
	}
	event := forge.Comment
	if postVerdict {
		switch sum.Verdict.Outcome {
		case model.VerdictApprove:
			event = forge.EventApprove
		case model.VerdictChangesRequested:
			event = forge.EventRequestChanges
		}
	}
	inline := make([]forge.InlineComment, 0, len(sum.ReviewInline))
	for _, a := range sum.ReviewInline {
		inline = append(inline, forge.InlineComment{Path: a.Path, Line: a.Line, Body: a.Body})
	}

	url, err := p.PostReview(ctx, sum.Path, sum.PR, sum.ReviewedHead, string(body), event, inline)
	if err != nil && len(inline) > 0 && strings.HasPrefix(err.Error(), "inline:") {
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
