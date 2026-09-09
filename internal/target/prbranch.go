package target

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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
	// CrossRepository is true when the head is a FORK's branch rather than one in
	// the repository being reviewed. It decides a refusal rather than a log line;
	// see Orchestrator.resolvePR.
	CrossRepository bool
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
// It is a method rather than a free function so it runs the commands through the
// collector the caller already owns -- its pinned tools, its hardened
// environment, its credential filter. The forge CLI still gets its own token
// back (see runInput): resolving a pull request is an authenticated call.
//
// It must not run before the target-integrity preflight. Four commands here
// execute git and an authenticated gh inside the target, and this feature's own
// premise is that the PR's branch is ALREADY checked out -- so the content those
// guards exist for is in the tree, unlike the pr mode they were written for,
// where nothing arrives until Prepare's checkout. The orchestrator therefore
// calls this after PreflightGuards and behind the repository lock; see
// Orchestrator.resolvePR (review run 20260909-213147, findings i1 and i8).
func (c *Collector) ResolvePRFromBranch(ctx context.Context) (BranchPR, error) {
	if _, err := c.git(ctx, "rev-parse", "--git-dir"); err != nil {
		return BranchPR{}, fmt.Errorf("target.path %s is not a git repository, so no branch can name a pull request (mode pr): %w", c.cfg.Path, err)
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
	// The branch is read once at the start and gh consults the checkout again on
	// its own, so the answer is only about THIS branch if the branch did not move
	// in between. It can: another fixpoint run switching branches under the same
	// repository, or the operator doing it by hand. The number is pinned from here
	// on and `gh pr checkout` is deterministic, so all that is needed is to notice
	// -- a run that resolved from a branch nobody is standing on any more must not
	// review, and certainly not post to, the pull request it happened to catch
	// (review run 20260909-213147, finding i10).
	after, err := c.currentBranch(ctx)
	if err != nil {
		return BranchPR{}, fmt.Errorf("branch %s resolved to pull request #%d, but the checkout no longer names a branch: %w", branch, view.Number, err)
	}
	if after != branch {
		return BranchPR{}, fmt.Errorf("the checkout moved from branch %s to %s while its pull request was being resolved, so #%d is the answer to a question about a branch nobody is on any more; re-run, or pass -pr <number>",
			branch, after, view.Number)
	}
	// The collector keeps its own copy of the target description, taken when it was
	// built, so it has to learn the number here: the caller updating the
	// configuration alone would leave Prepare checking out pull request 0 and Scope
	// reporting it (caught by the --check scope assertion in
	// TestRunResolvesPRFromTheCheckedOutBranch). One resolution, one number, both
	// halves holding it.
	c.cfg.PR = view.Number
	return BranchPR{Number: view.Number, Branch: branch, Base: view.BaseRefName, URL: view.URL, CrossRepository: view.CrossRepo}, nil
}

// currentBranch is the branch name HEAD points at, or an error naming what to do
// instead. symbolic-ref rather than `rev-parse --abbrev-ref HEAD`, which answers
// the literal string "HEAD" on a detached checkout and would send the resolution
// looking for a pull request whose head branch is called HEAD.
func (c *Collector) currentBranch(ctx context.Context) (string, error) {
	out, err := c.git(ctx, "symbolic-ref", "--quiet", "--short", "HEAD")
	branch := strings.TrimSpace(out)
	if err == nil && branch != "" {
		return branch, nil
	}
	// Only --quiet's own "HEAD is not symbolic" exit means detached. A timeout, a
	// cancellation, a corrupt repository or a permission failure exit differently,
	// and reporting those as a detached checkout sends the operator to -pr, which
	// addresses none of them (review run 20260909-224407).
	if err != nil && !detachedHEAD(err) {
		return "", fmt.Errorf("read the checked-out branch of %s: %w", c.cfg.Path, err)
	}
	return "", fmt.Errorf("HEAD is not on a branch in %s (a detached checkout, or a repository with no commits), so there is no branch to resolve a pull request from; pass -pr <number>", c.cfg.Path)
}

// detachedHEAD reports whether a `symbolic-ref --quiet` failure is the answer
// "HEAD does not name a ref" rather than a failure to ask. With --quiet that is
// exit 1 and no output; git's other failures use 128, and a canceled or
// timed-out command is not an exit status at all.
func detachedHEAD(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var exit *exec.ExitError
	return errors.As(err, &exit) && exit.ExitCode() == 1
}

// branchPRView is the part of `gh pr view` this resolution reads.
type branchPRView struct {
	Number      int    `json:"number"`
	State       string `json:"state"`
	URL         string `json:"url"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
	CrossRepo   bool   `json:"isCrossRepository"`
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
	view, err := c.viewBranchPRAny(ctx, branch)
	if err != nil {
		return branchPRView{}, err
	}
	if !strings.EqualFold(view.State, "open") {
		return branchPRView{}, fmt.Errorf("branch %s resolves to pull request #%d, which is %s, not open; fixpoint will not review merged or closed work by inference -- pass -pr %d if that is really the target",
			branch, view.Number, strings.ToLower(view.State), view.Number)
	}
	return view, nil
}

// prViewFields is the projection both readers ask for, in one place: the fork
// check and the number resolution must not drift apart about what they know, and
// dropping a field here (isCrossRepository especially) would silently decode as
// its zero value and disarm a gate. Pinned by TestResolvePRFromBranchPinsTheViewQuery.
const prViewFields = "number,state,url,baseRefName,headRefName,headRepositoryOwner,isCrossRepository"

// viewBranchPRAny is `gh pr view` for the checked-out branch WHATEVER state its
// pull request is in. The fork check needs that: a merged pull request's content
// sits in the tree exactly as an open one's does.
func (c *Collector) viewBranchPRAny(ctx context.Context, branch string) (branchPRView, error) {
	raw, err := c.run(ctx, "gh", "pr", "view", "--json", prViewFields)
	if err != nil {
		// Deliberately not "no pull request found": that exit status also covers a
		// lapsed token, an unreachable network, and a checkout whose base repository
		// gh cannot pick -- and telling an operator whose gh auth expired that their
		// pull request does not exist invites them to open a second one (review run
		// 20260909-224407). gh's own stderr is appended by runInput, so the cause is
		// in the line; the wording here only stops asserting the wrong one.
		return branchPRView{}, fmt.Errorf("could not resolve a pull request for branch %s -- if the branch has no pull request open one, otherwise check `gh auth status`; or pass -pr <number>: %w", branch, err)
	}
	var view branchPRView
	if err := json.Unmarshal([]byte(raw), &view); err != nil {
		return branchPRView{}, fmt.Errorf("gh pr view for branch %s returned output this cannot read: %w", branch, err)
	}
	if view.Number <= 0 {
		return branchPRView{}, fmt.Errorf("gh pr view for branch %s named no pull request number; pass -pr <number>", branch)
	}
	return view, nil
}

// CheckedOutFork answers the question the fork refusal is really about: is the
// tree at target.path the content of a pull request from ANOTHER repository?
//
// It is separate from ResolvePRFromBranch, and that separation is the fix for a
// real bypass. Gating the refusal on "no number was found anywhere" made it
// skippable by supplying one: `pr:` is an ordinary config key, the bundle is
// searched in the project's own config/ FIRST, and on a pull request's branch
// that file is the pull request's -- so a fork could set `pr:` and turn off the
// gate that exists to refuse it, while its `agents:` still named the commands
// fixpoint runs. The property is about the TREE, so it is now checked for every
// pr-mode run however the number arrived (review run 20260909-224407, one
// critical and two highs).
//
// Two signals, refusing on either, because neither alone is both precise and
// reliable:
//
//   - `gh pr view` with no argument is authoritative about THIS branch (it reads
//     the branch's push configuration, so it tells a fork's `feat/x` from ours),
//     but its failure cannot be told apart from "no pull request here".
//   - `gh pr list --head` matches the branch NAME across every fork, so it can
//     name a pull request that is not this branch's -- but it exits cleanly with
//     an empty list, so a failure really is a failure.
//
// A listing that cannot be read is therefore an error, not a shrug: this is a
// gate, and "could not tell" must not mean "allowed".
func (c *Collector) CheckedOutFork(ctx context.Context) (BranchPR, bool, error) {
	branch, err := c.currentBranch(ctx)
	if err != nil {
		// No branch names the checkout (detached, or an empty repository), so neither
		// signal applies. A detached checkout of a fork's commit is not caught here;
		// the number resolution refuses a detached HEAD outright, and with an explicit
		// -pr the operator is naming what they are reviewing.
		return BranchPR{}, false, nil //nolint:nilerr // "HEAD names no branch" is an answer to this question, not a failure of it
	}
	if view, verr := c.viewBranchPRAny(ctx, branch); verr == nil && view.CrossRepo {
		return BranchPR{Number: view.Number, Branch: branch, Base: view.BaseRefName, URL: view.URL, CrossRepository: true}, true, nil
	}
	raw, err := c.run(ctx, "gh", "pr", "list", "--head", branch, "--state", "all",
		"--limit", strconv.Itoa(prBranchCandidates), "--json", "number,isCrossRepository,headRepositoryOwner")
	if err != nil {
		return BranchPR{}, false, fmt.Errorf("could not determine whether branch %s is the head of a pull request from another repository, and a run whose target may be a fork's checkout is not started on a guess; pass -trusted-target if this checkout is yours: %w", branch, err)
	}
	var open []branchPRView
	if err := json.Unmarshal([]byte(raw), &open); err != nil {
		return BranchPR{}, false, fmt.Errorf("gh pr list for branch %s returned output this cannot read, so whether it is a fork's branch is unknown; pass -trusted-target if this checkout is yours: %w", branch, err)
	}
	for _, pr := range open {
		if pr.CrossRepo {
			return BranchPR{Number: pr.Number, Branch: branch, CrossRepository: true}, true, nil
		}
	}
	return BranchPR{}, false, nil
}

// proveSolePR refuses the resolution unless the branch names exactly one open
// pull request, and that one is the pull request gh answered with. See
// ResolvePRFromBranch for why gh's own answer is not enough on its own.
//
// Candidates are counted per HEAD OWNER, not per branch name. `--head` matches
// the name alone, so in a fork workflow two forks' `feat/x` both come back;
// those are different branches that happen to share a name, and refusing them
// would break the very workflow gh's resolution handles correctly. Two open pull
// requests from the SAME head, differing only in base, are the ambiguity that
// matters.
//
// A failed probe is a refusal, not a shrug: this is the step that decides the
// number is safe to act on, and "could not tell" is not "unique". Three ways it
// can fail to tell, each of which used to pass by counting alone (review run
// 20260909-213147, findings i17, i19 and the truncation medium):
//
//   - The listing does not contain the viewed number. Then it is a listing of
//     something else -- a different base repository, a race, an inconsistent API
//     answer -- and it proves nothing about #N.
//   - The listing came back empty. Same thing, in its starkest form: the head gh
//     answered with must have at least gh's own pull request on it.
//   - The listing hit the limit. The cap applies BEFORE the head-owner filter, so
//     other forks' same-named branches can fill it and hide a sibling of ours.
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
	if len(open) >= prBranchCandidates {
		return fmt.Errorf("branch %s (head %s) has at least %d open pull requests, so the listing that would prove #%d is the only one from this head is truncated and proves nothing; pass -pr <number>",
			branch, head, prBranchCandidates, view.Number)
	}
	var same []branchPRView
	found := 0
	for _, pr := range open {
		if pr.HeadOwner.Login != view.HeadOwner.Login {
			continue
		}
		same = append(same, pr)
		if pr.Number == view.Number {
			found++
		}
	}
	if found != 1 {
		return fmt.Errorf("listing the open pull requests on head %s returned %d of them, and pull request #%d -- the one gh resolved for branch %s -- appears %d times among those from the same owner; the listing therefore cannot prove #%d is the only one, so it is not acted on. Pass -pr <number>",
			head, len(open), view.Number, branch, found, view.Number)
	}
	if len(same) == 1 {
		return nil
	}
	var parts []string
	for _, pr := range same {
		parts = append(parts, fmt.Sprintf("#%d into %s", pr.Number, pr.BaseRefName))
	}
	return fmt.Errorf("branch %s has %d open pull requests (%s), so which one to review is ambiguous; pass -pr <number>",
		branch, len(same), strings.Join(parts, ", "))
}
