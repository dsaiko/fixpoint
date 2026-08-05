// Package review turns a finished review round into a verdict: APPROVE,
// CHANGES_REQUESTED, or INCONCLUSIVE.
//
// The verdict is computed HERE, in code, from facts the run recorded -- never by
// asking a model. A verdict is an action (it gates a merge, it can be posted to
// someone's pull request), and the panel that would be asked for it is the same
// panel whose findings it is judging. Keeping it deterministic also means two runs
// over the same facts cannot disagree, and that the rule can be argued about by
// reading one function instead of a prompt.
//
// Decide is a pure function on purpose: no clock, no filesystem, no config. Every
// input is something the caller already has, which is what makes the whole rule
// testable as a table.
package review

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dsaiko/fixpoint/internal/model"
)

// Outcome is the verdict. The three values map onto what a forge can express:
// GitHub's APPROVE / REQUEST_CHANGES / COMMENT, and GitLab's approve / unapprove
// plus a plain note.
type Outcome string

const (
	// Approve means the panel reached quorum, nothing above the blocking severity
	// survived, and the target's own checks are not failing.
	Approve Outcome = "approve"
	// ChangesRequested means something blocking survived. It is the only outcome that
	// can be reached WITHOUT quorum, because a real defect found by an incomplete
	// panel is still a real defect -- an incomplete panel is a reason to doubt
	// silence, never a reason to doubt a finding.
	ChangesRequested Outcome = "changes_requested"
	// Inconclusive means nothing blocking was found, but the review was not complete
	// enough for that silence to mean anything. Never posted as an approval.
	Inconclusive Outcome = "inconclusive"
)

// Quorum reports how much of the panel actually finished.
//
// The unit is the AGENT, not the step: under strategy `all` one agent runs every
// lens, so counting steps would let a panel of three agents look two-thirds
// present when in truth one agent had died and taken its whole perspective with
// it. An agent counts as present only if every step assigned to it succeeded --
// a reviewer that answered three lenses and timed out on the fourth has a blind
// spot exactly the size of that lens.
type Quorum struct {
	Panel    int      // agents assigned this round
	Present  int      // agents whose every step succeeded
	Required int      // strict majority of Panel
	Missing  []string // agents that failed at least one step, sorted
}

// Met reports whether enough of the panel finished for silence to be evidence.
func (q Quorum) Met() bool { return q.Panel > 0 && q.Present >= q.Required }

// Majority is the strict majority of n: more than half, rounded up. A panel of
// one needs that one, a panel of four needs three -- not two, which would let a
// tie approve.
func Majority(n int) int {
	if n <= 0 {
		return 0
	}
	return n/2 + 1
}

// CI is what the forge reports about the reviewed head, when it reports anything.
//
// Known is separate from an empty Failing list because "the checks pass" and "we
// could not find out" must not be the same input to a verdict. Unknown CI never
// blocks -- plenty of projects have none -- but it is recorded in the reasons so
// an approval never silently rests on a check nobody ran.
type CI struct {
	Known   bool
	Failing []string
	// Pending are checks still running. They do not block -- that would make the
	// tool unusable while CI runs -- but an approval that ignored them silently
	// would be overstating what it knows, so they are named in the reasons.
	Pending []string
}

// Input is everything Decide is allowed to look at.
type Input struct {
	// Issues are the findings that SURVIVED whatever filtering ran before this:
	// refutation, then the judge. Decide does not re-litigate them; it only reads
	// their severity and status. Advisory observations never reach here -- they are
	// reported for a human and gate nothing.
	Issues []model.Issue
	Quorum Quorum
	CI     CI
	// BlockAt is the severity at or above which a surviving finding forces
	// CHANGES_REQUESTED. Empty means the default, high.
	BlockAt string
	// FilterFailed reports that a configured filter -- today the judge -- did not
	// run to completion. It cannot make a finding appear, so it never forces
	// CHANGES_REQUESTED; it blocks the APPROVAL, because a review whose filter never
	// ran has not been filtered, and approving on that basis trusts a step that did
	// not happen.
	FilterFailed bool
}

// Decision is the verdict plus the whole reason it was reached, in the order the
// rules were applied. The reasons are rendered into the review body and the
// summary: a verdict a human cannot audit is not much better than a model's
// opinion, and "why did this approve?" is the question that gets asked.
type Decision struct {
	Outcome  Outcome
	Reasons  []string
	Blocking []model.Issue
	Quorum   Quorum
}

// DefaultBlockAt is the severity floor that forces CHANGES_REQUESTED.
//
// high, not medium, and the choice is measured rather than tasteful. Across 19
// runs the panel produced 322 issues of which 66 were high or critical: a medium
// floor would block essentially every review, and a gate that always fires is one
// people learn to bypass. It is deliberately permissive on the model's half --
// which is why the deterministic half (failing CI) matters as much as this one.
const DefaultBlockAt = "high"

// Decide applies the rules in a fixed order. The order is the rule:
//
//  1. A blocking finding requires changes, quorum or not. A defect an incomplete
//     panel found is still a defect.
//  2. Failing CI requires changes. This half of the gate costs nothing and no
//     model can talk it out of firing.
//  3. Without quorum, silence proves nothing, so the best available outcome is
//     INCONCLUSIVE -- never an approval.
//  4. Otherwise approve.
func Decide(in Input) Decision {
	blockAt := in.BlockAt
	if blockAt == "" {
		blockAt = DefaultBlockAt
	}
	d := Decision{Quorum: in.Quorum}
	floor := model.SeverityRank(blockAt)

	for _, it := range in.Issues {
		// A decided issue is not outstanding work. In a review-only run nothing is
		// ever fixed, so this only ever drops what the judge or a refutation round
		// already threw out -- but the rule belongs here rather than in every caller.
		switch it.StatusOrDefault() {
		case model.VerdictFixed, model.VerdictRejected:
			continue
		}
		if model.SeverityRank(it.Severity) <= floor {
			d.Blocking = append(d.Blocking, it)
		}
	}
	sort.SliceStable(d.Blocking, func(i, j int) bool {
		return model.SeverityRank(d.Blocking[i].Severity) < model.SeverityRank(d.Blocking[j].Severity)
	})

	if len(d.Blocking) > 0 {
		d.Outcome = ChangesRequested
		d.Reasons = append(d.Reasons, fmt.Sprintf("%d unresolved finding(s) at %s or above: %s",
			len(d.Blocking), blockAt, strings.Join(issueIDs(d.Blocking), ", ")))
	}
	if len(in.CI.Failing) > 0 {
		d.Outcome = ChangesRequested
		d.Reasons = append(d.Reasons, "the target's own checks are failing: "+strings.Join(in.CI.Failing, ", "))
	}
	if d.Outcome == ChangesRequested {
		d.Reasons = append(d.Reasons, quorumNote(in.Quorum))
		return d
	}

	if !in.Quorum.Met() {
		d.Outcome = Inconclusive
		d.Reasons = append(d.Reasons,
			fmt.Sprintf("no finding at %s or above, but the panel did not reach quorum, so that silence is not evidence", blockAt),
			quorumNote(in.Quorum))
		return d
	}
	if in.FilterFailed {
		d.Outcome = Inconclusive
		d.Reasons = append(d.Reasons,
			"nothing blocking survived, but the judge did not finish, so the findings were never filtered",
			quorumNote(in.Quorum))
		return d
	}

	d.Outcome = Approve
	d.Reasons = append(d.Reasons, fmt.Sprintf("no unresolved finding at %s or above", blockAt), quorumNote(in.Quorum))
	if !in.CI.Known {
		d.Reasons = append(d.Reasons, "no CI status was available for the reviewed head; this approval rests on the review alone")
	}
	if len(in.CI.Pending) > 0 {
		d.Reasons = append(d.Reasons, fmt.Sprintf("%d check(s) were still running and are not covered by this approval: %s",
			len(in.CI.Pending), strings.Join(in.CI.Pending, ", ")))
	}
	return d
}

// quorumNote renders the panel's completeness in one line, always -- including
// when quorum was met. An approval that does not say how much of the panel stood
// behind it invites the reader to assume all of it did.
func quorumNote(q Quorum) string {
	if q.Panel == 0 {
		return "no reviewer was assigned"
	}
	s := fmt.Sprintf("%d of %d reviewer(s) completed every lens (quorum %d)", q.Present, q.Panel, q.Required)
	if len(q.Missing) > 0 {
		s += ": " + strings.Join(q.Missing, ", ") + " did not"
	}
	return s
}

func issueIDs(issues []model.Issue) []string {
	out := make([]string, 0, len(issues))
	for _, it := range issues {
		out = append(out, it.ID)
	}
	return out
}

// QuorumFrom counts a round's panel from its assignments and the steps that ran.
//
// failedBy names, per agent, how many of its steps failed; an agent absent from
// that map completed everything it was given. Assignments rather than steps
// decide the panel size, so an agent whose session never started at all still
// counts against the quorum instead of vanishing from the denominator.
func QuorumFrom(assignments []model.Assignment, failedBy map[string]int) Quorum {
	panel := map[string]bool{}
	for _, a := range assignments {
		if a.Advisory {
			continue // reported for a human; gates nothing, so it is not part of the panel
		}
		panel[a.Agent] = true
	}
	q := Quorum{Panel: len(panel)}
	q.Required = Majority(q.Panel)
	for name := range panel {
		if failedBy[name] > 0 {
			q.Missing = append(q.Missing, name)
			continue
		}
		q.Present++
	}
	sort.Strings(q.Missing)
	return q
}

// Summary converts the decision into the form the run summary carries. Issue IDs
// rather than issues: the findings are already recorded on the rounds, and
// duplicating them here would let the two copies disagree about a title.
func (d Decision) Summary() *model.ReviewVerdict {
	return &model.ReviewVerdict{
		Outcome:  string(d.Outcome),
		Reasons:  d.Reasons,
		Blocking: issueIDs(d.Blocking),
		Panel:    d.Quorum.Panel,
		Present:  d.Quorum.Present,
		Required: d.Quorum.Required,
		Missing:  d.Quorum.Missing,
	}
}
