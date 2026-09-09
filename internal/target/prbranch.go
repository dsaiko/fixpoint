package target

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dsaiko/fixpoint/internal/config"
)

// BranchPR is the pull request a checked-out branch resolved to, and the branch
// it was resolved from. The caller logs both: a run that reviews "the PR of this
// branch" must say out loud which pull request that turned out to be, because
// nobody typed the number.
type BranchPR struct {
	Number int
	Branch string
	Base   string
	URL    string
}

// prBranchCandidates bounds the ambiguity probe below. A head branch with more
// open pull requests than this is ambiguous many times over; the list only has
// to prove there is more than one.
const prBranchCandidates = 50

// ResolvePRFromBranch answers "which pull request is this branch?" for a pr-mode
// run that was given no number -- `fixpoint review-pr` with no -pr, from the
// branch the work is on, which is how the command gets used in practice.
//
// The answer comes from `gh pr view` with no argument, deliberately: that is the
// same branch-to-PR resolution `gh pr checkout` acts on, including the parts
// that are not a branch-name match (a fork's push remote decides which owner's
// head counts), so the number resolved here and the tree Prepare checks out
// cannot disagree.
//
// What gh will NOT do is refuse when the answer is ambiguous. A head branch can
// have several open pull requests -- same head, different bases -- and gh picks
// one of them silently; pinning the wrong pull request would put a review, and
// possibly a verdict, on somebody else's work. So uniqueness is proved
// separately, and a branch that names more than one open pull request is
// refused with the numbers, rather than resolved to whichever gh listed first.
//
// env is the environment fixpoint's own git/gh commands run with; callers pass
// agent.EnvWithoutCredentials(cfg.Agents), the same filter the verify gate gets.
// The forge CLI still gets its own token back (see runInput) -- resolving a pull
// request is an authenticated call.
func ResolvePRFromBranch(ctx context.Context, cfg config.Target, env []string) (BranchPR, error) {
	c := New(cfg)
	c.UseGitEnv(env)
	if _, err := c.git(ctx, "rev-parse", "--git-dir"); err != nil {
		return BranchPR{}, fmt.Errorf("target.path %s is not a git repository, so no branch can name a pull request (mode pr): %w", cfg.Path, err)
	}
	branch, err := c.currentBranch(ctx)
	if err != nil {
		return BranchPR{}, err
	}
	view, err := c.viewBranchPR(ctx, branch)
	if err != nil {
		return BranchPR{}, err
	}
	if err := c.proveSolePR(ctx, branch, view); err != nil {
		return BranchPR{}, err
	}
	return BranchPR{Number: view.Number, Branch: branch, Base: view.BaseRefName, URL: view.URL}, nil
}

// currentBranch is the branch name HEAD points at, or an error naming what to do
// instead. symbolic-ref rather than `rev-parse --abbrev-ref HEAD`, which answers
// the literal string "HEAD" on a detached checkout and would send the resolution
// looking for a pull request whose head branch is called HEAD.
func (c *Collector) currentBranch(ctx context.Context) (string, error) {
	out, err := c.git(ctx, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(out)
	if err != nil || branch == "" {
		return "", fmt.Errorf("HEAD is not on a branch in %s (a detached checkout, or a repository with no commits), so there is no branch to resolve a pull request from; pass -pr <number>", c.cfg.Path)
	}
	return branch, nil
}

// branchPRView is the part of `gh pr view` this resolution reads.
type branchPRView struct {
	Number      int    `json:"number"`
	State       string `json:"state"`
	URL         string `json:"url"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
	HeadOwner   struct {
		Login string `json:"login"`
	} `json:"headRepositoryOwner"`
}

// viewBranchPR asks gh for the pull request of the checked-out branch.
//
// A pull request that is not OPEN is refused rather than reviewed. gh answers
// with a merged or closed one when that is all the branch has, and a review of
// merged work -- or worse, a fix run committing onto it -- is never what "review
// the PR of this branch" meant. The number is in the refusal, so an operator who
// does want it can say so explicitly with -pr.
func (c *Collector) viewBranchPR(ctx context.Context, branch string) (branchPRView, error) {
	raw, err := c.run(ctx, "gh", "pr", "view", "--json", "number,state,url,baseRefName,headRefName,headRepositoryOwner")
	if err != nil {
		return branchPRView{}, fmt.Errorf("no pull request found for branch %s; open one, or pass -pr <number>: %w", branch, err)
	}
	var view branchPRView
	if err := json.Unmarshal([]byte(raw), &view); err != nil {
		return branchPRView{}, fmt.Errorf("gh pr view for branch %s returned output this cannot read: %w", branch, err)
	}
	if view.Number <= 0 {
		return branchPRView{}, fmt.Errorf("gh pr view for branch %s named no pull request number; pass -pr <number>", branch)
	}
	if !strings.EqualFold(view.State, "open") {
		return branchPRView{}, fmt.Errorf("branch %s resolves to pull request #%d, which is %s, not open; fixpoint will not review merged or closed work by inference -- pass -pr %d if that is really the target",
			branch, view.Number, strings.ToLower(view.State), view.Number)
	}
	return view, nil
}

// proveSolePR refuses the resolution unless the branch names exactly one open
// pull request. See ResolvePRFromBranch for why gh's own answer is not enough on
// its own.
//
// Candidates are counted per HEAD OWNER, not per branch name. `--head` matches
// the name alone, so in a fork workflow two forks' `feat/x` both come back;
// those are different branches that happen to share a name, and refusing them
// would break the very workflow gh's resolution handles correctly. Two open pull
// requests from the SAME head, differing only in base, are the ambiguity that
// matters.
//
// A failed probe is a refusal, not a shrug: this is the step that decides the
// number is safe to act on, and "could not tell" is not "unique".
func (c *Collector) proveSolePR(ctx context.Context, branch string, view branchPRView) error {
	// The head gh ANSWERED with, not the local branch name, when the two differ:
	// gh can resolve through a branch's push configuration, and asking about a name
	// no pull request has would make this probe vacuous -- it would prove the
	// uniqueness of nothing and pass.
	head := view.HeadRefName
	if head == "" {
		head = branch
	}
	raw, err := c.run(ctx, "gh", "pr", "list", "--head", head, "--state", "open",
		"--limit", strconv.Itoa(prBranchCandidates), "--json", "number,baseRefName,headRepositoryOwner")
	if err != nil {
		return fmt.Errorf("branch %s resolves to pull request #%d, but listing the branch's open pull requests to prove that is the only one failed; pass -pr <number> to say which: %w", branch, view.Number, err)
	}
	var open []branchPRView
	if err := json.Unmarshal([]byte(raw), &open); err != nil {
		return fmt.Errorf("gh pr list for branch %s returned output this cannot read; pass -pr <number>: %w", branch, err)
	}
	var same []branchPRView
	for _, pr := range open {
		if pr.HeadOwner.Login == view.HeadOwner.Login {
			same = append(same, pr)
		}
	}
	if len(same) <= 1 {
		return nil
	}
	var parts []string
	for _, pr := range same {
		parts = append(parts, fmt.Sprintf("#%d into %s", pr.Number, pr.BaseRefName))
	}
	return fmt.Errorf("branch %s has %d open pull requests (%s), so which one to review is ambiguous; pass -pr <number>",
		branch, len(same), strings.Join(parts, ", "))
}
