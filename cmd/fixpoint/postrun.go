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
	sum, err := loadRunSummary(dir)
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
	body, err := os.ReadFile(sum.ReviewBody)
	if err != nil {
		// The path is recorded as absolute, but a run directory can be copied or the
		// project moved, so fall back to the file beside the summary.
		alt := filepath.Join(dir, "review-body.md")
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

	url, err := p.PostReview(ctx, sum.Path, sum.PR, string(body), event, inline)
	if err != nil && len(inline) > 0 {
		logf("post-run: %s rejected the inline comments (%v); posting the summary without them", p.Kind(), err)
		url, err = p.PostReview(ctx, sum.Path, sum.PR, string(body), event, nil)
	}
	if err != nil {
		logf("post-run: publishing to %s failed: %v", p.Kind(), err)
		return 1
	}
	if url != "" {
		logf("posted %s to %s as %s: %s", filepath.Base(dir), p.Kind(), event, url)
	} else {
		logf("posted %s to %s as %s", filepath.Base(dir), p.Kind(), event)
	}
	return 0
}

// loadRunSummary reads the summary from a run directory, accepting either the
// directory itself or the summary file.
func loadRunSummary(dir string) (*model.RunSummary, error) {
	path := dir
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		matches, err := filepath.Glob(filepath.Join(dir, "summary-*.json"))
		if err != nil || len(matches) == 0 {
			return nil, fmt.Errorf("%s holds no summary-*.json; is it a run directory?", dir)
		}
		// Newest last by name, since the name carries the timestamp.
		sort.Strings(matches)
		path = matches[len(matches)-1]
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sum model.RunSummary
	if err := json.Unmarshal(raw, &sum); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if sum.ReviewBody == "" {
		return nil, errors.New("that run wrote no review body")
	}
	return &sum, nil
}
