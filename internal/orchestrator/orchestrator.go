// Package orchestrator runs the review->fix loop described in
// fixpoint.yaml: assign lenses to agents, run reviewers in parallel, hand
// findings to the coder, commit each fix round, and stop on convergence.
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/logstore"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
	"github.com/dsaiko/fixpoint/internal/target"
)

// Orchestrator drives one run: it owns the collector, the log store, and the
// parsed prompt templates, and executes rounds until a termination condition.
type Orchestrator struct {
	cfg       *config.Config
	source    config.Source
	collector *target.Collector
	logs      *logstore.Store
	// templates is keyed by prompt NAME, the same identity carried on assignments
	// and written into log filenames -- not by resolved path, so there is one
	// notion of "which lens is this" everywhere.
	templates map[string]*template.Template
	logf      func(format string, args ...any)
	// gitExclude holds the logs dir as a repo-relative path when it lives
	// inside target.path, so round commits and clean checks never touch the
	// run's own logs.
	gitExclude []string
}

// New sets up the log store, parses every referenced prompt and renders it once
// with the zero value of its role's data type (so a placeholder belonging to the
// wrong role fails now rather than mid-run), and wires the collector and its
// log-directory exclusion. It returns an orchestrator ready to Run.
//
// New does NOT perform the config-level startup validation (agents defined and
// on PATH, strategy satisfiable): that lives in config.Validate, which the
// caller must run beforehand.
func New(cfg *config.Config, source config.Source, logf func(string, ...any)) (*Orchestrator, error) {
	logs, err := logstore.New(cfg.Logs)
	if err != nil {
		return nil, err
	}
	// Parse every referenced prompt up front (part of startup validation) and
	// render it once with the zero value of its role's data type, so a
	// placeholder belonging to the other role fails now instead of mid-run.
	templates := map[string]*template.Template{}
	load := func(name, path string, data any) error {
		// Parsing is cached per name, but the render check runs for every
		// (name, role) pairing: the same file may serve both roles, and
		// skipping the second role would let its placeholders fail mid-run.
		t, ok := templates[name]
		if !ok {
			var err error
			t, err = prompt.Load(path)
			if err != nil {
				return err
			}
		}
		if _, err := prompt.Render(t, data); err != nil {
			return err
		}
		templates[name] = t
		return nil
	}
	if err := load(cfg.Roles.Coder.Prompt, cfg.Roles.Coder.PromptFile(), prompt.FixData{}); err != nil {
		return nil, err
	}
	for _, l := range cfg.Roles.Review.Prompts {
		if err := load(l.Prompt, l.PromptFile(), prompt.ReviewData{}); err != nil {
			return nil, err
		}
	}
	o := &Orchestrator{
		cfg:       cfg,
		source:    source,
		collector: target.New(cfg.Target),
		logs:      logs,
		templates: templates,
		logf:      logf,
	}
	// Exclude the template's literal prefix, not the rendered path: the rendered
	// path changes every run (and every round), so only the static base is a
	// stable pathspec for commits, clean checks, and collection.
	if rel, err := logsDirWithin(cfg.Logs.StaticBase(), cfg.Target.Path); err == nil && rel != "" {
		// "." would mean "exclude the whole repository": GitClean and Commit
		// would then see every tree as clean and silently skip all changes.
		if rel == "." {
			return nil, fmt.Errorf("logs.dir %s: its literal prefix resolves to target.path itself; put the logs under a subdirectory so round commits and clean checks can exclude them", cfg.Logs.Dir)
		}
		o.gitExclude = []string{rel}
		// Also keep the logs out of the material the collector gathers, so later
		// rounds never review the run's own prompts and outputs (directory walks
		// and untracked-file listings), not just the git commits/clean checks.
		o.collector.ExcludeLogs(rel)
	}
	return o, nil
}

// logsDirWithin returns the logs dir as a path relative to the target root if
// it lives inside it, "" otherwise.
func logsDirWithin(logsDir, root string) (string, error) {
	absLogs, err := filepath.Abs(logsDir)
	if err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absLogs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

// checkLogsNotSymlinked verifies that no path component of the in-target logs
// directory is a symlink, so artifacts cannot be redirected outside the lexical
// logs exclusion (o.gitExclude) and swept into a round commit. It is a no-op when
// the logs dir lives outside the target (o.gitExclude unset) or when the logs
// path does not exist yet (it will then be created as a real directory). Called
// after Prepare because a PR checkout can change the path from a plain directory
// into a symlink.
func (o *Orchestrator) checkLogsNotSymlinked() error {
	if len(o.gitExclude) == 0 {
		return nil
	}
	root, err := filepath.Abs(o.cfg.Target.Path)
	if err != nil {
		return err
	}
	cur := root
	for _, comp := range strings.Split(filepath.ToSlash(o.gitExclude[0]), "/") {
		cur = filepath.Join(cur, comp)
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // not yet created; ensureDir will make it a real directory
			}
			return fmt.Errorf("inspect logs path %s: %w", cur, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("logs path component %s is a symlink; the prepared target (e.g. a PR) may have redirected the logs directory to bypass the artifact exclusion and leak prompts/outputs into a commit. Point logs.dir outside the target, or remove the symlink", cur)
		}
	}
	return nil
}

// Run executes the full loop and always writes a run summary before
// returning. The returned summary's Termination field says how it ended.
func (o *Orchestrator) Run(ctx context.Context) (*model.RunSummary, error) {
	sum := &model.RunSummary{
		StartedAt:  time.Now(),
		ConfigPath: o.source.Config,
		Sources: model.RunSources{
			Config:  o.source.Config,
			Extends: o.source.Extends,
			Agents:  o.source.Agents,
			Prompts: o.source.Prompts,
		},
		Mode:       string(o.cfg.Target.Mode),
		Path:       o.cfg.Target.Path,
		Strategy:   string(o.cfg.Roles.Review.Strategy),
		ReviewOnly: o.cfg.Loop.ReviewOnly,
	}
	err := o.run(ctx, sum)
	sum.FinishedAt = time.Now()
	if err != nil && sum.Termination == "" {
		// Cancellation mid-step (e.g. Ctrl-C while the coder runs) surfaces as
		// that step's error -- typically a kill message from agent.Run. Record
		// it as an interruption, like the loop's own ctx checks do, so the
		// summary distinguishes "user stopped it" from "it failed".
		// A dirty tree left behind because the interrupt-time stash failed is a
		// hard failure, not a clean interruption: surface it and record it in the
		// summary even though ctx was canceled, so the user is never told the run
		// stopped cleanly while unreconciled edits sit in the tree.
		if ctx.Err() != nil && !errors.Is(err, errInterruptedTreeDirty) {
			sum.Termination = model.TermInterrupted
			err = nil
		} else {
			sum.Termination = model.TermError
			sum.Error = err.Error()
		}
	}
	if mdPath, werr := o.logs.Summary(sum); werr != nil {
		o.logf("WARNING: failed to write summary: %v", werr)
	} else {
		o.logf("summary written: %s", mdPath)
	}
	return sum, err
}

func (o *Orchestrator) run(ctx context.Context, sum *model.RunSummary) error {
	o.warnArgModePrompts()

	// Enforce the fix-round trust gate; see checkFixTrust for the rationale.
	if err := o.checkFixTrust(); err != nil {
		return err
	}

	if err := o.guardUntrustedGitConfig(ctx); err != nil {
		return err
	}

	// Cheap read-only git checks first. Fix rounds commit; require a clean
	// git tree so every round commit contains exactly the coder's changes and
	// nothing of yours. PR review requires a clean tree too, even review-only:
	// gh pr checkout can preserve unrelated tracked edits and untracked files,
	// which Collect would otherwise fold into the PR diff and misattribute the
	// resulting findings to the PR.
	requireCleanTree := !o.cfg.Loop.ReviewOnly || o.cfg.Target.Mode == config.ModePR
	if !o.cfg.Loop.ReviewOnly {
		if !o.collector.IsGitRepo(ctx) {
			return fmt.Errorf("target.path %s is not a git repository; fix rounds commit each round and require git (use review_only for a report-only run)", o.cfg.Target.Path)
		}
	}
	// The clean check and staging are scoped to target.path ("." pathspec), but a
	// fix round's commit writes the whole repository index and a PR review runs a
	// repo-wide diff over a repo-wide checkout. A subdirectory target would then
	// miss dirt elsewhere in the repo from its target-relative clean check yet
	// still fold those changes into the round commit (fix) or the reviewed diff
	// (pr) -- so require the repository root whenever we enforce a clean tree,
	// including PR review-only runs, not just fix runs.
	if requireCleanTree {
		if atRoot, err := o.collector.AtRepoRoot(ctx); err != nil {
			return err
		} else if !atRoot {
			return fmt.Errorf("target.path %s is a subdirectory of its git repository; fix rounds and PR reviews operate on the whole repository (staging the entire index / diffing the whole checkout), so point target.path at the repository root (or use review_only for a non-pr run)", o.cfg.Target.Path)
		}
	}
	if requireCleanTree {
		if err := o.ensureCleanTree(ctx, "working tree is dirty; commit or stash your changes first so the review sees only the intended target (and, in a fix run, round commits contain only the coder's fixes)"); err != nil {
			return err
		}
	}

	// Then ping (spends a little on each agent), then Prepare (may mutate
	// state -- pr checkout): each stage fails before the next spends more.
	if o.cfg.Ping() {
		o.logf("preflight: pinging agents...")
		if err := o.Ping(ctx); err != nil {
			return err
		}
	}

	if err := o.collector.Prepare(ctx); err != nil {
		return err
	}

	// The logs exclusion (o.gitExclude) is a LEXICAL path fixed before Prepare,
	// but Prepare can check out a PR that turns the in-target logs directory into
	// a symlink pointing back into the repository. Artifacts written through such
	// a symlink land under a different physical path that the lexical exclusion
	// misses, so `git add -A` would stage them and a fix commit push prompts and
	// raw agent output (which can carry secrets past the redactor). Recheck here,
	// after the checkout that could have introduced the symlink and before any
	// artifact is written.
	if err := o.checkLogsNotSymlinked(); err != nil {
		return err
	}

	// The pre-Prepare clean check above protects the checkout itself, but
	// Prepare then switches branches (pr mode runs gh pr checkout) without
	// rechecking. A checkout can leave the new branch dirty -- previously
	// ignored files surface, checkout hooks run -- and those pre-existing
	// changes would then be attributed to the coder and swept into a round
	// commit by git add -A. Re-check for fix runs so a dirty checked-out tree
	// aborts before any coder edits. PR review-only re-checks too, so the
	// checkout's leftovers never reach the collected diff (see requireCleanTree).
	if requireCleanTree {
		if err := o.ensureCleanTree(ctx, "working tree is dirty after preparing the target (e.g. gh pr checkout); commit, stash, or clean the checked-out branch so the review sees only the PR's changes (and round commits contain only the coder's fixes)"); err != nil {
			return err
		}
	}

	cleanStreak := 0
	for round := 1; round <= o.cfg.Loop.MaxIterations; round++ {
		done, err := o.runRound(ctx, round, sum, &cleanStreak)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	sum.Termination = model.TermMaxIterations
	return nil
}

// guardUntrustedGitConfig closes the code-execution path opened by the target's
// own repo-local .git/config: per-name content filters (filter.<n>.clean/
// smudge/process), core.sshCommand, and credential helpers are programs git runs
// itself, and gitSafeConfig's -c overrides cannot neutralize them (their names
// are dynamic). They fire whenever a git command touches the worktree -- git-diff
// mode's `git diff` normalizes files through a clean filter even in a review_only
// run, and every fix round's `git add`/`git status` does the same -- so a target
// that ships its own .git/config (an extracted archive, a crafted checkout) can
// run a repo-controlled program with fixpoint's inherited secrets.
//
// The gate applies wherever such a command runs against a target-controlled repo:
// git-diff mode always (its Collect runs `git diff`), and directory fix rounds
// (their commits run `git add`/`git status`). Directory review-only walks the
// filesystem and runs no worktree-touching git command, so it has no such path;
// pr mode's .git/config is the operator's own -- a PR cannot alter it -- so
// neither is gated. When the operator has asserted no trust we refuse; when trust
// IS asserted we still WARN, because trusted_target/allow_untrusted_fix is
// documented as accepting coder prompt-injection risk and an operator must also
// learn it accepts .git/config-driven code execution (a git-lfs repo they trust,
// or an enforced external sandbox, is the intended use).
func (o *Orchestrator) guardUntrustedGitConfig(ctx context.Context) error {
	touchesGit := o.cfg.Target.Mode == config.ModeGitDiff ||
		(!o.cfg.Loop.ReviewOnly && o.cfg.Target.Mode == config.ModeDirectory)
	if !touchesGit || !o.collector.IsGitRepo(ctx) {
		return nil
	}
	keys, err := o.collector.UnsafeConfig(ctx)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	if !o.cfg.Loop.TrustedTarget && !o.cfg.Loop.AllowUntrustedFix {
		return fmt.Errorf("target %s has repo-local git config that would run repo-controlled programs fixpoint cannot neutralize (%s); git normalizes worktree files through these during diff/add/status, so this is a code-execution path with fixpoint's inherited environment. Review it under an external sandbox (container/VM), or set loop.trusted_target: true (or -trusted-target) if you trust this checkout", o.cfg.Target.Path, strings.Join(keys, ", "))
	}
	o.logf("WARNING: target %s has repo-local git config that runs repo-controlled programs during git diff/add/status (%s) which fixpoint cannot neutralize; loop.trusted_target/allow_untrusted_fix accepts this code-execution path (with fixpoint's inherited environment) in addition to coder prompt-injection. Review untrusted checkouts (extracted archives, crafted .git) under an external sandbox.", o.cfg.Target.Path, strings.Join(keys, ", "))
	return nil
}

// checkFixTrust enforces the fix-round trust gate. Fix rounds run the coder --
// an agent that edits files with permission checks disabled and is NOT confined
// to target.path -- and reviewer findings built from in-scope content are
// injected verbatim into its prompt. A prompt-injection payload hidden in any
// reviewed file can steer it into writing anywhere on the machine, so fix rounds
// require an explicit trust signal in EVERY mode (fail-closed by default), not
// just pr:
//   - pr: the PR author is untrusted by definition -> allow_untrusted_fix.
//   - directory/git-diff: the tree may still hold untrusted material (vendored
//     deps, a fetched base_ref, a cloned third-party project) -> trusted_target
//     (or allow_untrusted_fix).
func (o *Orchestrator) checkFixTrust() error {
	if o.cfg.Loop.ReviewOnly {
		return nil
	}
	switch o.cfg.Target.Mode {
	case config.ModePR:
		if !o.cfg.Loop.AllowUntrustedFix {
			return errors.New("mode pr reviews untrusted code, and fix rounds hand its content to a coder that edits files without permission checks; use review_only, or set loop.allow_untrusted_fix (or -allow-untrusted-fix) if you trust the PR author")
		}
	default:
		if !o.cfg.Loop.TrustedTarget && !o.cfg.Loop.AllowUntrustedFix {
			return fmt.Errorf("fix rounds run a coder that edits files with permission checks disabled and is not confined to target.path, so reviewed content could steer it via prompt injection; set loop.trusted_target: true (or -trusted-target) to assert %q holds only code you trust, or use review_only", o.cfg.Target.Path)
		}
	}
	return nil
}

// ensureCleanTree fails with dirtyMsg when the working tree has changes outside
// the git-exclude set. The pre- and post-Prepare gates share it so the single
// GitClean+error pattern lives in one place.
func (o *Orchestrator) ensureCleanTree(ctx context.Context, dirtyMsg string) error {
	clean, err := o.collector.GitClean(ctx, o.gitExclude...)
	if err != nil {
		return err
	}
	if !clean {
		return errors.New(dirtyMsg)
	}
	return nil
}

// runRound executes one review->fix->commit iteration, appending its record to
// sum and threading the consecutive-clean-round streak through cleanStreak. It
// reports done=true when the loop should stop (setting sum.Termination), and an
// error when the round failed. run() is then just preflight plus this loop.
func (o *Orchestrator) runRound(ctx context.Context, round int, sum *model.RunSummary, cleanStreak *int) (done bool, err error) {
	if ctx.Err() != nil {
		sum.Termination = model.TermInterrupted
		return true, nil //nolint:nilerr // interruption is a normal termination, not a round error
	}
	o.logf("=== round %d/%d ===", round, o.cfg.Loop.MaxIterations)

	material, err := o.collector.Collect(ctx)
	if err != nil {
		return false, err
	}

	rec := model.RoundRecord{Round: round, Assignments: o.assignments(round)}
	// A fix run with no reviewers this round cannot verify anything: an empty
	// review reports no findings and no errors, which would otherwise read as a
	// clean round and let the loop converge on an unverified fix. Config
	// validation already requires a recurring lens for fix runs, so this is a
	// backstop against any future path that leaves a fix round unassigned.
	if !o.cfg.Loop.ReviewOnly && len(rec.Assignments) == 0 {
		return false, fmt.Errorf("round %d has no reviewer assignments to verify the fix; configure at least one recurring (non-once) reviewer lens", round)
	}
	o.review(ctx, &rec, material, sum.Rounds)
	sum.Rounds = append(sum.Rounds, rec)
	recP := &sum.Rounds[len(sum.Rounds)-1]

	o.logf("round %d: %d finding(s), %d advisory, %d reviewer error(s)",
		round, len(recP.Findings), len(recP.Advisory), len(recP.ReviewErrors))

	if ctx.Err() != nil {
		sum.Termination = model.TermInterrupted
		return true, nil //nolint:nilerr // interruption is a normal termination, not a round error
	}

	// Review-only runs exactly one round, and it only counts as a successful
	// review if every reviewer completed: reporting success over a partial
	// round would let automation trust an incomplete review. The round record
	// stays in the summary either way.
	if o.cfg.Loop.ReviewOnly {
		if len(recP.ReviewErrors) > 0 {
			return false, roundReviewErr(recP)
		}
		sum.Termination = model.TermReviewOnly
		return true, nil
	}

	// Termination on clean rounds (advisory findings do not count).
	if len(recP.Findings) == 0 {
		return o.checkCleanStreak(recP, round, sum, cleanStreak), nil
	}
	*cleanStreak = 0

	// recP is the round just appended, so the earlier rounds -- which the cap's
	// aging reads to see what has been waiting -- are everything before it.
	prior := sum.Rounds[:len(sum.Rounds)-1]
	o.deferOverCap(recP, prior)

	salvaged, err := o.fix(ctx, recP, prior)
	if err != nil {
		return false, err
	}
	if salvaged {
		// The coder failed but its partial edits were committed; verdicts are
		// unknown, so skip finalize and let the next round re-review.
		return false, nil
	}
	o.logf("round %d: coder fixed %d, rejected %d", round, recP.Fixed, recP.Rejected)

	return o.finalizeFix(ctx, recP, round, sum)
}

// severities is the one authoritative severity vocabulary, ordered worst-first.
// severityRank (the per-round cap's ordering) and validSeverities (the reviewer
// gate's membership set) are both derived from it below, so adding or renaming a
// severity here updates the ordering and the validation gate together and they
// can never drift apart.
var severities = []string{"critical", "high", "medium", "low"}

// severityRank orders findings worst-first for the per-round cap; unknown
// severities sort last. Derived from severities (rank = index).
var severityRank = func() map[string]int {
	m := make(map[string]int, len(severities))
	for i, s := range severities {
		m[s] = i
	}
	return m
}()

// validSeverities is the closed set a reviewer may report (per ReviewContract),
// derived from severities so the reviewer gate and the ranking share one source.
var validSeverities = func() map[string]bool {
	m := make(map[string]bool, len(severities))
	for _, s := range severities {
		m[s] = true
	}
	return m
}()

// deferralKey identifies a finding across rounds for aging. Reviewers re-report
// an unfixed finding with a fresh id and often reworded prose, so identity cannot
// come from the id or the title: (file, category) is what stays stable when the
// same gap is reported again by a different agent in a later round. Two distinct
// issues in one file and category share a key, which only ever grants the second
// one an earlier turn -- an acceptable trade for bounding starvation.
func deferralKey(f model.Finding) string { return f.File + "\x00" + f.Category }

// priorDeferrals counts, per deferralKey, how many earlier rounds ended with that
// key deferred or unresolved -- i.e. reported but never acted on.
func priorDeferrals(prior []model.RoundRecord) map[string]int {
	n := map[string]int{}
	for _, r := range prior {
		seen := map[string]bool{} // one increment per round, not per duplicate report
		for _, f := range r.Findings {
			switch f.Verdict {
			case model.VerdictFixed, model.VerdictRejected:
				continue // resolved; nothing to age
			}
			if k := deferralKey(f); !seen[k] {
				seen[k] = true
				n[k]++
			}
		}
	}
	return n
}

// deferOverCap enforces loop.max_findings_per_round: the worst maxN findings stay
// active for the coder, the rest are marked deferred. Deferred findings appear in
// history as DEFERRED (still open), so reviewers re-report them and they reach the
// coder in a later round.
//
// Ordering is worst-severity-first with AGING: each earlier round that deferred a
// finding promotes it one severity tier. Without that, an unbounded generator of
// medium-severity findings (a test-coverage lens can always want more coverage)
// starves everything below it forever -- in one 5-round run a one-line README
// error was reported three times and never once scheduled. Aging bounds the wait
// instead: a "low" finding reaches top priority after three skips, so every
// finding is guaranteed a turn while severity still decides the common case.
func (o *Orchestrator) deferOverCap(rec *model.RoundRecord, prior []model.RoundRecord) {
	maxN := o.cfg.Loop.MaxFindingsPerRound
	if maxN <= 0 || len(rec.Findings) <= maxN {
		return
	}
	aged := priorDeferrals(prior)
	idx := make([]int, len(rec.Findings))
	for i := range idx {
		idx[i] = i
	}
	rank := func(i int) int {
		f := rec.Findings[i]
		r, ok := severityRank[strings.ToLower(strings.TrimSpace(f.Severity))]
		if !ok {
			r = len(severityRank) // unknown severities sort last
		}
		r -= aged[deferralKey(f)]
		return max(r, 0)
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ra, rb := rank(idx[a]), rank(idx[b])
		if ra != rb {
			return ra < rb
		}
		// Equal effective rank: whichever has waited longer goes first. The tie
		// would otherwise fall to assignment order -- deterministic, but it lets a
		// fresh finding edge out one already skipped, which is the starvation this
		// aging exists to stop.
		return aged[deferralKey(rec.Findings[idx[a]])] > aged[deferralKey(rec.Findings[idx[b]])]
	})
	for _, i := range idx[maxN:] {
		rec.Findings[i].Verdict = model.VerdictDeferred
		rec.Findings[i].VerdictDetail = fmt.Sprintf(
			"deferred: fix round capped at %d finding(s) by loop.max_findings_per_round (gains priority each round it is deferred)", maxN)
	}
	o.logf("round %d: %d finding(s) exceed the per-round cap of %d; %d deferred to later rounds",
		rec.Round, len(rec.Findings), maxN, len(rec.Findings)-maxN)
}

// activeFindings returns the round's findings that are actually handed to the
// coder (everything not deferred by the cap).
func activeFindings(rec *model.RoundRecord) []model.Finding {
	var out []model.Finding
	for _, f := range rec.Findings {
		if f.Verdict != model.VerdictDeferred {
			out = append(out, f)
		}
	}
	return out
}

// checkCleanStreak advances (or resets) the consecutive-clean-round streak for
// a round that produced no non-advisory findings, and reports done=true once it
// reaches clean_rounds_to_stop (setting TermConverged). A round with reviewer
// errors RESETS the streak: "consecutive clean rounds" means consecutive
// fully-successful clean rounds.
func (o *Orchestrator) checkCleanStreak(rec *model.RoundRecord, round int, sum *model.RunSummary, cleanStreak *int) (done bool) {
	if len(rec.ReviewErrors) == 0 {
		*cleanStreak++
	} else {
		*cleanStreak = 0
		o.logf("round %d had reviewer errors; clean streak reset", round)
	}
	if *cleanStreak >= o.cfg.Loop.CleanRoundsToStop {
		sum.Termination = model.TermConverged
		return true
	}
	o.logf("clean round (%d/%d consecutive needed)", *cleanStreak, o.cfg.Loop.CleanRoundsToStop)
	return false
}

// finalizeFix reconciles the coder's reported verdicts with the actual working
// tree after a fix, then either commits the round or recognizes a genuine
// all-rejected terminal state. Whether to commit is decided from the tree, not
// from the Fixed count alone: a disagreement is a hard error rather than a
// warning, because fixes with no edits mean nothing was really done, and edits
// under an all-rejected verdict would otherwise leave the "all-rejected"
// success termination sitting on top of an uncommitted, dirty tree.
func (o *Orchestrator) finalizeFix(ctx context.Context, rec *model.RoundRecord, round int, sum *model.RunSummary) (done bool, err error) {
	// A cancellation between the coder finishing and this commit must not leave a
	// committed round (or a dirty tree) sitting on top of a stop request. Recheck
	// immediately before touching the tree and route any cancellation through the
	// same stash-and-interrupt reconciliation the fix path uses.
	if ctx.Err() != nil {
		return false, o.reconcileInterrupt(rec.Round, ctx.Err()) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
	}
	clean, err := o.collector.GitClean(ctx, o.gitExclude...)
	if err != nil {
		// Cancellation can land WHILE GitClean runs, after the check above passed
		// (a TOCTOU window). Route the resulting context error through the same
		// interrupt reconciliation, on a fresh context, so a tree left dirty by a
		// canceled coder is stashed (or flagged errInterruptedTreeDirty) rather
		// than returned bare and softened into a clean interruption by Run.
		if ctx.Err() != nil {
			return false, o.reconcileInterrupt(rec.Round, err) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
		}
		return false, err
	}
	if rec.Fixed > 0 && clean {
		return false, fmt.Errorf("round %d: coder reported %d fix(es) but left the working tree unchanged", round, rec.Fixed)
	}
	if rec.Fixed == 0 && !clean {
		return false, fmt.Errorf("round %d: coder rejected every finding yet modified the working tree; refusing to leave the changes uncommitted", round)
	}

	if rec.Fixed > 0 {
		sha, err := o.commit(ctx, rec)
		if err != nil {
			// Same TOCTOU window: cancellation can interrupt the commit's git add /
			// commit after staging the coder's edits. Reconcile so the staged tree
			// is stashed instead of left staged under a softened interruption.
			if ctx.Err() != nil {
				return false, o.reconcileInterrupt(rec.Round, err) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
			}
			// A non-cancellation commit failure (e.g. commit signing failed) leaves
			// the coder's edits staged by Commit's `git add -A`. Left as-is, the
			// dirty/staged tree violates the clean-tree invariant and the next run
			// refuses to start. Stash the edits so the tree is clean again.
			return false, o.reconcileFailedCommit(ctx, rec.Round, err)
		}
		rec.CommitSHA = sha
		if sha != "" {
			o.logf("round %d committed: %s", round, sha[:12])
		}
		return false, nil
	}

	// rec.Fixed == 0 with a clean tree: a genuine all-rejected round. It is a
	// successful terminal state only when the round's review was complete; with
	// reviewer errors it may just mean the failed reviewers' findings never
	// arrived.
	if len(rec.ReviewErrors) > 0 {
		return false, roundReviewErr(rec)
	}
	// Deferred findings never reached the coder, so "everything was rejected"
	// does not hold for the round as a whole -- keep looping so they get
	// their turn.
	if deferred := len(rec.Findings) - len(activeFindings(rec)); deferred > 0 {
		o.logf("round %d: coder rejected all active findings but %d deferred remain; continuing", round, deferred)
		return false, nil
	}
	sum.Termination = model.TermAllRejected
	return true, nil
}

// roundReviewErr aggregates a round's reviewer failures into one error.
func roundReviewErr(rec *model.RoundRecord) error {
	return fmt.Errorf("round %d: %d reviewer(s) failed: %s",
		rec.Round, len(rec.ReviewErrors), strings.Join(rec.ReviewErrors, "; "))
}

// warnArgModePrompts logs a warning for every agent this run will use that is
// configured prompt_via: arg. Such an agent receives the full prompt on its
// process argument list, where the embedded review material (in git-diff/pr
// mode the diff -- which often IS the credential under review) and any secret a
// reviewer quotes into a finding are world-readable via ps / /proc for the
// invocation's lifetime, bypassing the 0600 log permissions and on-disk
// redaction. The exposure is not silent; stdin is the secure default.
func (o *Orchestrator) warnArgModePrompts() {
	for _, n := range o.activeAgentNames() {
		if o.cfg.Agents[n].PromptVia != config.PromptViaArg {
			continue
		}
		o.logf("WARNING: agent %q uses prompt_via: arg -- the full prompt (reviewed material, plus any secret quoted into a finding) is placed on the process argument list and is readable by other local users via ps / /proc for the run's duration, bypassing the 0600 log permissions and on-disk redaction; prefer prompt_via: stdin for material that may contain secrets", n)
	}
}

// activeAgentNames returns the sorted, distinct set of agents this run will
// invoke: the coder (unless review-only) plus every active reviewer agent.
// Ping and warnArgModePrompts share it so "the agents this run uses" has a
// single definition. Sorted so per-agent log lines and failure lists are
// deterministic run-to-run, matching review's index-ordered collection.
func (o *Orchestrator) activeAgentNames() []string {
	seen := map[string]bool{}
	var names []string
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	if !o.cfg.Loop.ReviewOnly {
		add(o.cfg.Roles.Coder.Agent)
	}
	for _, a := range o.cfg.Roles.Review.ActiveAgents() {
		add(a)
	}
	sort.Strings(names)
	return names
}

// pingTimeout caps each preflight ping so a hung CLI cannot stall startup.
const pingTimeout = 3 * time.Minute

// pingTimeoutFor caps an agent's configured timeout to pingTimeout for the
// preflight ping: a hung agent (or one left on the generous default timeout)
// must not stall startup for its full run timeout on each retry. A timeout at or
// below pingTimeout is preserved unchanged.
func pingTimeoutFor(a config.Agent) config.Duration {
	if a.Timeout.Std() > pingTimeout {
		return config.Duration(pingTimeout)
	}
	return a.Timeout
}

// Ping invokes every distinct agent this run will use with a trivial prompt,
// in parallel, and returns an error naming every agent that failed. Success is
// a zero exit with non-empty stdout (models decorate output; content is not
// checked).
func (o *Orchestrator) Ping(ctx context.Context) error {
	// activeAgentNames includes the write-capable coder unless review-only, and
	// pinging it launches that CLI in target.path with permission checks disabled
	// -- where it loads repository instructions (AGENTS.md, CLAUDE.md) that an
	// untrusted target could weaponize. A real run refuses a fix round without a
	// trust signal (checkFixTrust); -check-live must not offer an unguarded path
	// around that gate, so enforce it here before any agent is invoked.
	if err := o.checkFixTrust(); err != nil {
		return err
	}
	names := o.activeAgentNames()

	type ping struct {
		err error
		dur time.Duration
	}
	results := make([]ping, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			a := o.cfg.Agents[name]
			// Cap once, before the retry loop, so both attempts use the bounded
			// timeout rather than the agent's full run timeout.
			a.Timeout = pingTimeoutFor(a)
			// One retry: CLIs occasionally fail transiently on cold start,
			// and a false-negative preflight aborts a wanted run.
			var err error
			var dur time.Duration
			for range 2 {
				res := agent.Run(ctx, a, "Reply with the single word: OK", o.cfg.Target.Path)
				err = res.Err
				dur += res.Duration
				if err == nil && strings.TrimSpace(res.Stdout) == "" {
					err = errors.New("no output")
				}
				if err == nil || ctx.Err() != nil {
					break
				}
			}
			results[i] = ping{err: err, dur: dur}
		}(i, name)
	}
	wg.Wait()

	var failed []string
	for i, name := range names {
		p := results[i]
		desc := agentDesc(o.cfg.Agents[name])
		if p.err != nil {
			o.logf("ping: %s %s FAILED (%s): %v", name, desc, p.dur.Round(time.Second), p.err)
			failed = append(failed, name)
		} else {
			o.logf("ping: %s %s ok (%s)", name, desc, p.dur.Round(time.Second))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("agent(s) not working: %s", strings.Join(failed, ", "))
	}
	return nil
}

// agentDesc renders an agent's model (and effort, when set) for log lines:
// "[fable, high]" or "[gemma4:31b-cloud]".
func agentDesc(a config.Agent) string {
	if a.Effort != "" {
		return fmt.Sprintf("[%s, %s]", a.Model, a.Effort)
	}
	return fmt.Sprintf("[%s]", a.Model)
}

// assignments pairs lenses with agents for one round, per the strategy.
func (o *Orchestrator) assignments(round int) []model.Assignment {
	rv := o.cfg.Roles.Review
	var out []model.Assignment
	unpinned := 0
	for _, l := range rv.Prompts {
		pinned := l.Agent != ""
		// Give every unpinned lens a rotation index fixed by its position in the
		// full configured prompt list, advancing the counter BEFORE the once-only
		// skip below. Otherwise skipping an unpinned once:true lens after round 1
		// would shift the indices of the recurring lenses that follow it,
		// re-selecting the same agent across rounds and defeating rotation (and
		// letting a clean-round confirmation reuse the previous round's reviewer).
		idx := unpinned
		if !pinned {
			unpinned++
		}
		if l.Once && round > 1 {
			continue // once-per-run lens (e.g. design/architecture): round 1 only
		}
		// agents is the shared "who may serve this lens" policy (config.LensAgents):
		// the pinned agent, or the whole pool. The log-collision validator enumerates
		// the same set, so validation cannot drift from what actually runs.
		agents := rv.LensAgents(l)
		switch {
		case pinned: // pinned under every strategy
			out = append(out, model.Assignment{Lens: l.Prompt, Agent: agents[0], Advisory: l.Advisory, Pinned: true})
		case rv.Strategy == config.StrategyAll:
			for _, a := range agents {
				out = append(out, model.Assignment{Lens: l.Prompt, Agent: a, Advisory: l.Advisory})
			}
		default: // rotate
			a := agents[(idx+round-1)%len(agents)]
			out = append(out, model.Assignment{Lens: l.Prompt, Agent: a, Advisory: l.Advisory})
		}
	}
	return out
}

// heartbeatEvery is how often a still-running agent invocation emits a
// progress line, so a long silent stretch is visibly alive rather than
// ambiguous (thinking-heavy coders can run 20+ minutes without output).
const heartbeatEvery = 5 * time.Minute

// runAgent persists the prompt (before invoking, so a killed or hung agent's
// input is still inspectable), invokes the agent with a heartbeat, and
// returns the result. label prefixes the heartbeat lines, e.g.
// "fix: claude-coder".
func (o *Orchestrator) runAgent(ctx context.Context, label, role, agentName, lensName string, round int, text string) agent.Result {
	if err := o.logs.Prompt(role, agentName, lensName, round, text); err != nil {
		o.logf("WARNING: writing %s prompt log: %v", role, err)
	}
	start := time.Now()
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(heartbeatEvery)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				o.logf("%s still running (%s elapsed)", label, time.Since(start).Round(time.Second))
			}
		}
	}()
	res := agent.Run(ctx, o.cfg.Agents[agentName], text, o.cfg.Target.Path)
	close(done)
	return res
}

// stepStat assembles the per-invocation I/O record for the run summary.
func stepStat(role, agentName, lensName string, promptLen int, res agent.Result, failed bool) model.StepStat {
	return model.StepStat{
		Role:        role,
		Agent:       agentName,
		Lens:        lensName,
		PromptBytes: promptLen,
		OutputBytes: len(res.Stdout) + len(res.Stderr),
		DurationMS:  res.Duration.Milliseconds(),
		Failed:      failed,
	}
}

// runReviewAssignment renders, invokes, parses, and logs one reviewer
// assignment synchronously, returning the raw findings it reported.
func (o *Orchestrator) runReviewAssignment(ctx context.Context, asg model.Assignment, round int, material string, history []model.RoundRecord) ([]model.ReviewFinding, model.StepStat, error) {
	lensName := config.LensName(asg.Lens)
	label := fmt.Sprintf("review: %s via %s", asg.Agent, lensName)
	d := prompt.ReviewData{
		Mode:           o.cfg.Target.Mode,
		Path:           o.cfg.Target.Path,
		Round:          round,
		ModeGuidance:   prompt.ModeGuidance(o.cfg.Target.Mode),
		Target:         material,
		History:        prompt.FormatHistory(history),
		OutputContract: prompt.ReviewContract,
	}
	text, err := prompt.Render(o.templates[asg.Lens], d)
	if err != nil {
		// No agent ran, so there is no output/duration to record; route through
		// stepStat anyway so both paths build the record the same way.
		return nil, stepStat("review", asg.Agent, lensName, len(text), agent.Result{}, true), err
	}
	o.logf("%s starting (prompt %s)", label, logstore.SizeDesc(len(text)))
	res := o.runAgent(ctx, label, "review", asg.Agent, lensName, round, text)
	var out model.ReviewOutput
	parseErr := res.Err
	if parseErr == nil {
		parseErr = agent.ExtractJSON(res.Stdout, "review", &out)
	}
	if parseErr == nil {
		parseErr = validateReviewFindings(out.Findings)
	}

	// Per-step logs (IDs are assigned later; md/json here carry raw findings).
	findings := toFindings(out.Findings, asg, lensName, nil, 0)
	md := logstore.RenderReviewMD(asg.Agent, lensName, round, findings, parseErr)
	o.logStep("review", asg.Agent, lensName, round, parseErr == nil, out, md, res)
	o.logf("%s done (%d findings, %s, output %s)", label, len(out.Findings), res.Duration.Round(time.Second), logstore.SizeDesc(len(res.Stdout)+len(res.Stderr)))
	return out.Findings, stepStat("review", asg.Agent, lensName, len(text), res, parseErr != nil), parseErr
}

// logStep writes one agent invocation's step logs, downgrading a log-write
// failure to a WARNING (a run must not abort because a log file could not be
// written). parsed is persisted only when ok, so a failed step never writes a
// JSON payload that would read as a clean result.
func (o *Orchestrator) logStep(role, agentName, promptName string, round int, ok bool, out any, md string, res agent.Result) {
	var parsed any
	if ok {
		parsed = out
	}
	if err := o.logs.Step(role, agentName, promptName, round, parsed, md, res.Raw(o.cfg.Agents[agentName].Argv())); err != nil {
		o.logf("WARNING: writing %s log: %v", role, err)
	}
}

// validateReviewFindings enforces the review contract's required fields before
// findings are accepted. It also closes a subtle failure mode in ExtractJSON:
// when a reviewer echoes its prompt and then emits no valid output, extraction
// can fall back to the contract's own example block -- whose placeholder
// severity ("critical|high|medium|low") is not a real value. Rejecting invalid
// severities turns that silent placeholder into an honest reviewer error
// instead of a fabricated finding handed to the coder.
func validateReviewFindings(findings []model.ReviewFinding) error {
	for _, f := range findings {
		if strings.TrimSpace(f.Title) == "" {
			return errors.New("finding has an empty title")
		}
		if !validSeverities[strings.ToLower(strings.TrimSpace(f.Severity))] {
			return fmt.Errorf("finding %q has invalid severity %q (want critical | high | medium | low)", f.Title, f.Severity)
		}
	}
	return nil
}

// review schedules all assigned reviewers in parallel, collects their results
// deterministically, and updates the round record, assigning finding IDs.
func (o *Orchestrator) review(ctx context.Context, rec *model.RoundRecord, material string, history []model.RoundRecord) {
	type result struct {
		asg      model.Assignment
		findings []model.ReviewFinding
		stat     model.StepStat
		err      error
	}
	results := make([]result, len(rec.Assignments))
	var wg sync.WaitGroup
	for i, asg := range rec.Assignments {
		wg.Add(1)
		go func(i int, asg model.Assignment) {
			defer wg.Done()
			findings, stat, err := o.runReviewAssignment(ctx, asg, rec.Round, material, history)
			results[i] = result{asg: asg, findings: findings, stat: stat, err: err}
		}(i, asg)
	}
	wg.Wait()

	seq := 0
	for _, r := range results {
		rec.Steps = append(rec.Steps, r.stat)
		lensName := config.LensName(r.asg.Lens)
		if r.err != nil {
			rec.ReviewErrors = append(rec.ReviewErrors, fmt.Sprintf("%s via %s: %v", r.asg.Agent, lensName, r.err))
			continue
		}
		fs := toFindings(r.findings, r.asg, lensName, &seq, rec.Round)
		if r.asg.Advisory {
			rec.Advisory = append(rec.Advisory, fs...)
		} else {
			rec.Findings = append(rec.Findings, fs...)
		}
	}
}

// toFindings converts model output to findings; when seq is non-nil, IDs
// r<round>.<n> are assigned.
func toFindings(in []model.ReviewFinding, asg model.Assignment, lensName string, seq *int, round int) []model.Finding {
	out := make([]model.Finding, 0, len(in))
	for _, f := range in {
		nf := model.Finding{
			Agent:       asg.Agent,
			Lens:        lensName,
			Category:    f.Category,
			Severity:    f.Severity,
			File:        f.File,
			Line:        f.Line,
			Title:       f.Title,
			Description: f.Description,
			Suggestion:  f.Suggestion,
			Advisory:    asg.Advisory,
		}
		if seq != nil {
			*seq++
			nf.ID = fmt.Sprintf("r%d.%d", round, *seq)
		}
		out = append(out, nf)
	}
	return out
}

// errInterruptedTreeDirty marks the one cancellation-time error that must NOT
// be softened into a clean interruption by Run: the coder's edits could not be
// stashed, so the working tree is left dirty and the user must see a hard
// failure (with the reason recorded in the summary), consistent with the
// salvage path's own reconcile-failure handling.
var errInterruptedTreeDirty = errors.New("working tree left dirty after interruption")

// fix hands the round's active findings to the coder and applies its
// verdicts. When the coder fails mid-round (timeout, session limit, malformed
// output) but already edited files, the partial work is committed and
// salvaged=true is returned: the loop continues and the next round re-reviews
// everything. This recovery was performed manually four times before being
// automated here.
func (o *Orchestrator) fix(ctx context.Context, rec *model.RoundRecord, history []model.RoundRecord) (salvaged bool, err error) {
	coder := o.cfg.Roles.Coder
	promptName := config.LensName(coder.Prompt)
	label := "fix: " + coder.Agent
	active := activeFindings(rec)
	d := prompt.FixData{
		Mode:           o.cfg.Target.Mode,
		Path:           o.cfg.Target.Path,
		Round:          rec.Round,
		Findings:       prompt.FormatFindings(active),
		History:        prompt.FormatHistory(history),
		OutputContract: prompt.FixContract,
	}
	text, err := prompt.Render(o.templates[coder.Prompt], d)
	if err != nil {
		return false, err
	}
	o.logf("%s starting on %d finding(s) (prompt %s)", label, len(active), logstore.SizeDesc(len(text)))
	res := o.runAgent(ctx, label, "fix", coder.Agent, promptName, rec.Round, text)
	var out model.FixOutput
	runErr := res.Err
	if runErr == nil {
		runErr = agent.ExtractJSON(res.Stdout, "fix", &out)
	}

	// The coder contract requires every finding exactly once with a valid
	// verdict; anything else fails the round rather than being silently
	// miscounted (an empty result set must not read as "all rejected").
	if runErr == nil {
		runErr = applyVerdicts(rec, out.Results)
	}

	rec.Steps = append(rec.Steps, stepStat("fix", coder.Agent, promptName, len(text), res, runErr != nil))
	md := logstore.RenderFixMD(coder.Agent, rec.Round, rec.Findings, out.Notes, runErr)
	o.logStep("fix", coder.Agent, promptName, rec.Round, runErr == nil, out, md, res)
	o.logf("%s done (%s, output %s)", label, res.Duration.Round(time.Second), logstore.SizeDesc(len(res.Stdout)+len(res.Stderr)))
	// A context cancellation (e.g. Ctrl-C) is the user asking to stop, not a
	// salvageable coder failure: committing the coder's edits would defy that
	// request. This holds even when the coder returned valid output (runErr ==
	// nil) but the context was canceled before we could commit -- checking only
	// runErr would miss that race and commit anyway. In every canceled case,
	// reconcile the tree by stashing any edits (kept recoverable via `git
	// stash`) and return the interruption -- run() maps a cancellation-time
	// error to TermInterrupted. A timeout or malformed response (ctx still live)
	// stays on the salvage path below.
	if ctx.Err() != nil {
		interrupted := runErr
		if interrupted == nil {
			interrupted = ctx.Err()
		}
		return false, o.reconcileInterrupt(rec.Round, interrupted) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
	}
	if runErr != nil {
		return o.salvagePartialFix(ctx, rec, active, runErr)
	}
	return false, nil
}

// reconcileInterrupt stashes any working-tree edits left by a canceled coder
// so the stop request never leaves a committed round or a dirty tree behind,
// and returns cause (the interruption) -- or errInterruptedTreeDirty when the
// stash itself fails. It uses a fresh context because the run's context is
// already canceled; target.Collector still bounds each git subprocess with its
// own operation deadline.
func (o *Orchestrator) reconcileInterrupt(round int, cause error) error {
	stashMsg := fmt.Sprintf("fixpoint: interrupted round %d", round)
	if _, serr := o.collector.StashDirty(context.Background(), stashMsg, o.gitExclude...); serr != nil {
		return fmt.Errorf("coder round %d interrupted: %w; and the modified working tree could not be reconciled (it is left dirty): %w: %w", round, cause, serr, errInterruptedTreeDirty)
	}
	return cause
}

// reconcileFailedCommit stashes the edits that Commit's `git add -A` staged
// before the commit itself failed (non-cancellation), so the tree is never left
// staged and dirty -- which would violate the clean-tree invariant and block the
// next run. It mirrors salvagePartialFix's failed-commit handling: return the
// original commit error to surface, or a combined error if the stash also fails.
func (o *Orchestrator) reconcileFailedCommit(ctx context.Context, round int, commitErr error) error {
	stashMsg := fmt.Sprintf("fixpoint: recovered edits from failed commit in round %d", round)
	if stashed, serr := o.collector.StashDirty(ctx, stashMsg, o.gitExclude...); serr != nil {
		return fmt.Errorf("round %d commit failed: %w; and the modified working tree could not be reconciled (it is left dirty): %w", round, commitErr, serr)
	} else if stashed {
		o.logf("round %d: commit failed (%v); edits stashed (recover with `git stash`), clean tree restored", round, commitErr)
	}
	return commitErr
}

// salvagePartialFix handles a coder failure (timeout, session limit,
// malformed output). The coder edits files before it reports, so a failure
// can leave real, per-file-complete work in the tree. That work is committed
// as a clearly-labeled partial round and the loop continues -- the next round
// re-reviews everything, so an incomplete or even broken intermediate state
// is caught by reviewers rather than stranded. Only when nothing can be
// committed (clean tree, or the commit itself fails) does the round become a
// hard error; a failing commit additionally stashes the edits so the tree is
// never left dirty.
func (o *Orchestrator) salvagePartialFix(ctx context.Context, rec *model.RoundRecord, active []model.Finding, runErr error) (salvaged bool, err error) {
	header := fmt.Sprintf("fixpoint: round %d (partial, coder failed)", rec.Round)
	var body strings.Builder
	fmt.Fprintf(&body, "Coder failed before reporting verdicts: %v\n\nFindings it was working on:\n", runErr)
	for _, f := range active {
		fmt.Fprintf(&body, "- [%s] (%s, %s) %s\n", f.ID, f.Category, f.Severity, f.Title)
	}
	// Redact reviewer-authored finding text before it lands in the pushed
	// commit message, mirroring the normal round commit and the logstore.
	sha, cerr := o.collector.Commit(ctx, header, agent.RedactSecrets(body.String()), o.gitExclude...)
	if cerr != nil {
		// A cancellation that lands during the salvage commit is a stop request,
		// not a salvageable failure. Passing the already-canceled ctx to
		// StashDirty below would make its git operations fail instantly and return
		// a reconcile error that lacks errInterruptedTreeDirty, which Run would
		// then soften into a clean interruption even though the tree is dirty.
		// Route it through reconcileInterrupt (fresh context) so the tree is
		// stashed, or the dirty tree is flagged as a hard interruption failure.
		if ctx.Err() != nil {
			return false, o.reconcileInterrupt(rec.Round, runErr) //nolint:contextcheck // deliberate fresh context: ctx is already canceled
		}
		stashMsg := fmt.Sprintf("fixpoint: recovered edits from failed round %d", rec.Round)
		if stashed, serr := o.collector.StashDirty(ctx, stashMsg, o.gitExclude...); serr != nil {
			return false, fmt.Errorf("coder round %d failed: %w; and the modified working tree could not be reconciled (it is left dirty): %w", rec.Round, runErr, serr)
		} else if stashed {
			o.logf("round %d: coder failed and the salvage commit also failed (%v); edits stashed (recover with `git stash`), clean tree restored", rec.Round, cerr)
		}
		return false, fmt.Errorf("coder round %d failed: %w", rec.Round, runErr)
	}
	if sha == "" {
		// Clean tree: the coder did nothing before failing -- a genuine error.
		return false, fmt.Errorf("coder round %d failed: %w", rec.Round, runErr)
	}
	rec.CoderError = runErr.Error()
	rec.CommitSHA = sha
	o.logf("round %d: coder failed (%v) but had modified the tree; partial work committed as %s -- continuing, next round re-reviews", rec.Round, runErr, sha[:12])
	return true, nil
}

// applyVerdicts validates the coder's result set against the round's
// findings -- every ID exactly once, only known IDs, only valid verdicts --
// and applies it. Any violation is an error for the round, and nothing is
// applied unless the whole set is valid: the always-written run summary must
// never carry a partially applied response.
func applyVerdicts(rec *model.RoundRecord, results []model.FixResult) error {
	// Deferred findings were never handed to the coder, so they neither
	// expect nor accept a verdict.
	byID := map[string]*model.Finding{}
	expected := 0
	for i := range rec.Findings {
		if rec.Findings[i].Verdict == model.VerdictDeferred {
			continue
		}
		byID[rec.Findings[i].ID] = &rec.Findings[i]
		expected++
	}
	seen := map[string]bool{}
	for _, r := range results {
		if _, ok := byID[r.ID]; !ok {
			return fmt.Errorf("coder referenced unknown finding id %q", r.ID)
		}
		if seen[r.ID] {
			return fmt.Errorf("coder returned finding id %q more than once", r.ID)
		}
		seen[r.ID] = true
		if r.Verdict != model.VerdictFixed && r.Verdict != model.VerdictRejected {
			return fmt.Errorf("coder gave unknown verdict %q for %s (want fixed | rejected)", r.Verdict, r.ID)
		}
	}
	if len(seen) != expected {
		// Range the ordered findings slice, not the byID map, so the reported
		// ids keep their r<round>.<n> order and the error text is deterministic.
		var missing []string
		for _, f := range rec.Findings {
			if f.Verdict != model.VerdictDeferred && !seen[f.ID] {
				missing = append(missing, f.ID)
			}
		}
		return fmt.Errorf("coder did not give a verdict for finding(s): %s", strings.Join(missing, ", "))
	}
	for _, r := range results {
		f := byID[r.ID]
		f.Verdict = r.Verdict
		f.VerdictDetail = r.Detail
		if r.Verdict == model.VerdictFixed {
			rec.Fixed++
		} else {
			rec.Rejected++
		}
	}
	return nil
}

// commit stages and commits the coder's changes with the configured header
// and a body composed from the coder's own verdicts.
func (o *Orchestrator) commit(ctx context.Context, rec *model.RoundRecord) (string, error) {
	header := strings.NewReplacer(
		"{round}", strconv.Itoa(rec.Round),
		"{fixed}", strconv.Itoa(rec.Fixed),
		"{rejected}", strconv.Itoa(rec.Rejected),
	).Replace(o.cfg.Loop.CommitMessage)

	var body strings.Builder
	writeVerdictSection(&body, "Fixed", rec.Findings, model.VerdictFixed)
	if rec.Rejected > 0 {
		body.WriteString("\n")
		writeVerdictSection(&body, "Rejected", rec.Findings, model.VerdictRejected)
	}
	// The body embeds reviewer-authored titles and coder-authored verdict
	// details, which may quote the very secret under review. Unlike the logs,
	// the commit is meant to be pushed and shared, so redact it too (the
	// logstore path does the same for its on-disk artifacts).
	return o.collector.Commit(ctx, header, agent.RedactSecrets(body.String()), o.gitExclude...)
}

// writeVerdictSection writes a "<title>:" header followed by one line per
// finding carrying the given verdict, so the fixed and rejected sections of a
// commit body share a single formatting site.
func writeVerdictSection(b *strings.Builder, title string, findings []model.Finding, verdict string) {
	b.WriteString(title + ":\n")
	for _, f := range findings {
		if f.Verdict == verdict {
			fmt.Fprintf(b, "- [%s] %s — %s\n", f.Category, f.Title, f.VerdictDetail)
		}
	}
}
