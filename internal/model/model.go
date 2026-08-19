// Package model defines the data shapes exchanged between reviewers, the
// coder, the orchestrator, and the logs.
package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Finding is one issue reported by a reviewer. ID is assigned by the
// orchestrator (not by the model) so the coder can reference findings
// unambiguously.
type Finding struct {
	ID          string `json:"id"`
	Agent       string `json:"agent"` // reviewer agent that reported it
	Lens        string `json:"lens"`  // prompt basename (review-bugs, ...)
	Category    string `json:"category"`
	Severity    string `json:"severity"` // critical | high | medium | low
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
	Advisory    bool   `json:"advisory,omitempty"`
	// Round is the review round this observation was made in. An issue keeps
	// every observation it ever collected, so "which agents reported this" is
	// only a corroboration claim when it is scoped to one round -- under
	// strategy: rotate a lens is deliberately reassigned each round, and without
	// the scope a single agent re-reporting a surviving issue looks like two
	// agents agreeing.
	Round int `json:"round,omitempty"`
	// IssueID is the issue this observation was grouped under. Several
	// observations from different agents and lenses can share one.
	IssueID string `json:"issue_id,omitempty"`

	// Origin records that this finding came from a CONVERSATION on the pull
	// request rather than from the panel -- a comment somebody left, which triage
	// accepted as real work. Zero value means the panel found it.
	//
	// It travels with the finding because two later steps need it: the coder is
	// told which conversation to answer once its fix is committed, and the commit
	// records who commissioned the change. A fix nobody on the panel asked for
	// should say whose request it was.
	Origin Origin `json:"origin,omitempty"`

	// Filled in after the coder round.
	Verdict       string `json:"verdict,omitempty"` // fixed | rejected | deferred
	VerdictDetail string `json:"verdict_detail,omitempty"`
}

// The verdict state space is a closed set of four values. Keeping them as
// named constants (rather than bare literals scattered across packages) gives
// the set one authoritative home and lets the compiler catch typos.
const (
	// VerdictFixed / VerdictRejected are the two verdicts the coder may report
	// per finding (its output contract is "fixed | rejected").
	VerdictFixed    = "fixed"
	VerdictRejected = "rejected"
	// VerdictDeferred marks a finding withheld from the coder because the round
	// was capped (loop.max_findings_per_round). Deferred findings are not
	// terminal: reviewers see them as still open and re-report them until a
	// later round has room.
	VerdictDeferred = "deferred"
	// VerdictUnresolved is the display fallback for a finding the coder never
	// ruled on (see VerdictOrDefault).
	VerdictUnresolved = "unresolved"
)

// Loc renders the finding's location for display: "file:line", or just the
// file when no line is known.
func (f Finding) Loc() string {
	if f.Line > 0 {
		return fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return f.File
}

// VerdictOrDefault returns the verdict, or "unresolved" when the coder never
// ruled on the finding.
func (f Finding) VerdictOrDefault() string {
	if f.Verdict == "" {
		return VerdictUnresolved
	}
	return f.Verdict
}

// ReviewOutput is the JSON envelope a reviewer must emit inside <review> tags.
type ReviewOutput struct {
	Findings []ReviewFinding `json:"findings"`
}

// UnmarshalJSON requires an explicit, non-null findings array. Without it,
// payloads like {} or null would decode as zero findings and a malformed
// review could read as a clean one.
func (r *ReviewOutput) UnmarshalJSON(b []byte) error {
	var raw struct {
		Findings *[]ReviewFinding `json:"findings"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw.Findings == nil {
		return errors.New(`review envelope has no "findings" array (must be [] when there are no findings)`)
	}
	r.Findings = *raw.Findings
	return nil
}

// ReviewFinding is a finding as emitted by the model, before the orchestrator
// assigns IDs and provenance.
type ReviewFinding struct {
	// Issue optionally names an issue id from the History section, declaring that
	// this report is the SAME problem as an earlier one. Cross-round identity
	// cannot be recovered lexically -- the same issue was described as "Severity
	// vocabulary has two independent declarations", then "declared twice
	// (severityRank and validSeverities)", then "duplicated severity vocabulary",
	// while its line number moved as the code around it changed -- so the reviewer
	// that can see both is asked to say so. Empty is fine; the fingerprint below
	// is the fallback.
	Issue       string `json:"issue"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// FixOutput is the JSON envelope the coder must emit inside <fix> tags.
type FixOutput struct {
	Results []FixResult `json:"results"`
	Notes   string      `json:"notes,omitempty"`
	// Replies are answers to the pull request's open conversations, when the coder
	// was shown any. Optional: a fix run over a PR with no threads returns none.
	Replies []FixReply `json:"replies,omitempty"`
}

// FixResult is the coder's verdict on one finding.
type FixResult struct {
	ID      string `json:"id"`
	Verdict string `json:"verdict"` // fixed | rejected
	Detail  string `json:"detail"`
}

// Assignment records which agent ran which lens in a round.
type Assignment struct {
	Lens     string `json:"lens"`
	Agent    string `json:"agent"`
	Advisory bool   `json:"advisory,omitempty"`
	Pinned   bool   `json:"pinned,omitempty"`
}

// RoundRecord is everything that happened in one loop iteration.
type RoundRecord struct {
	Round       int          `json:"round"`
	Assignments []Assignment `json:"assignments"`
	Findings    []Finding    `json:"findings"` // raw observations, with verdicts mirrored after fix
	// Issues is the deduplicated view the coder actually works from: one entry per
	// distinct problem, however many reviewers reported it.
	Issues       []Issue   `json:"issues,omitempty"`
	Advisory     []Finding `json:"advisory,omitempty"` // report-only findings
	ReviewErrors []string  `json:"review_errors,omitempty"`
	Fixed        int       `json:"fixed"`
	Rejected     int       `json:"rejected"`
	// StashedRejects names the issues whose coder session rejected the finding yet
	// left edits in the tree -- typically the probe work that disproved it. The
	// edits belong to no verdict, so each session's were stashed and the round
	// carried on; recorded here so the summary can say where that work went.
	StashedRejects []string `json:"stashed_rejects,omitempty"`
	// CommitSHA is the commit the round RESULTS in: the last of its per-fix commits,
	// or the squash that replaced them. Commits holds every commit the round made,
	// which is what the scoreboard counts -- under commit_policy: per_fix a round of
	// eight fixes is eight commits, and reporting only the last would undercount.
	CommitSHA string   `json:"commit_sha,omitempty"`
	Commits   []string `json:"commits,omitempty"`
	// CoderError is set when the coder failed mid-round but its partial edits
	// were salvaged into CommitSHA; the loop then continued.
	CoderError string `json:"coder_error,omitempty"`
	// Verify holds every run of the deterministic gate in this round, in the order
	// they ran. Kept on the round record so the summary can show what actually
	// passed -- the one non-model signal in the loop deserves to be persisted, not
	// just logged.
	//
	// A slice rather than one set of results because the gate runs per FIX, not per
	// round: under commit_policy: per_fix a round of N fixes runs it N times, plus
	// once more for each correction attempt, plus once on a salvage. Keeping only
	// the last would leave the summary and the scoreboard describing one fix's gate
	// run while claiming to describe the round.
	Verify []VerifyRun `json:"verify,omitempty"`
	Steps  []StepStat  `json:"steps,omitempty"` // per-invocation I/O figures
	// Final marks the closing round that `final: true` lenses run in, after the loop
	// has stopped. It is not part of the convergence story -- it happens once the
	// run's outcome is already decided -- so a reader must be able to tell it apart.
	Final bool `json:"final,omitempty"`
}

// TaskOutcome is one implement task's durable result, mirroring the marker
// trailers (DESIGN.md §5.4): the reason codes are the normalized enum, and a
// task that produced no commit still has a SHA -- its outcome marker's.
type TaskOutcome struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Outcome string `json:"outcome"` // implemented | already_satisfied | blocked | failed | skipped | carried
	// CarriedOutcome is the RECORDED outcome under a carried one -- what the
	// original run concluded. Set only when Outcome is carried. It exists so a
	// resumed run's report and its guards can act on the original verdict: a
	// carried failure makes the run incomplete, and a carried already_satisfied
	// still counts against max_vacuous_frac (review run 20260818-234734).
	CarriedOutcome string `json:"carried_outcome,omitempty"`
	Reason         string `json:"reason,omitempty"`
	SHA            string `json:"sha,omitempty"`
	Attempts       int    `json:"attempts,omitempty"`
	Gate           string `json:"gate,omitempty"` // passed | failed:<check> | ungated | skipped
}

// StepStat records one agent invocation's size and duration figures, so runs
// can be analyzed for cost and slowness (which lens/agent ate the wall clock,
// how big the prompts really were).
type StepStat struct {
	Role        string `json:"role"` // review | fix
	Agent       string `json:"agent"`
	Lens        string `json:"lens"` // prompt basename (review-bugs, fix, ...)
	PromptBytes int    `json:"prompt_bytes"`
	OutputBytes int    `json:"output_bytes"`
	DurationMS  int64  `json:"duration_ms"`
	Failed      bool   `json:"failed,omitempty"`
	// Usage is what the CLI itself reported for this invocation, when it reports
	// anything. It is not derived from the byte counts above and is usually far
	// larger than they suggest -- those measure only what crossed fixpoint's
	// boundary, while the agent's session reads files and calls tools in between.
	Usage Usage `json:"usage,omitempty"`
}

// Usage is one agent invocation's cost, as reported by the agent's own CLI.
//
// Every field is what the CLI said, never a fixpoint estimate. CostKnown
// distinguishes "this CLI reported $0" (a local model) from "this CLI reports no
// cost at all" (subscription auth) -- a zero that means two different things would
// make a run's total silently wrong, so the scoreboard shows nothing rather than
// zero for the second case.
type Usage struct {
	InputTokens      int     `json:"input_tokens,omitempty"`
	OutputTokens     int     `json:"output_tokens,omitempty"`
	CacheReadTokens  int     `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int     `json:"cache_write_tokens,omitempty"`
	CostUSD          float64 `json:"cost_usd,omitempty"`
	CostKnown        bool    `json:"cost_known,omitempty"`
}

// Tokens is every token the agent reported processing, cache included. Cache
// reads are cheaper than fresh input rather than free, and on a warm agentic
// session they dominate the total -- excluding them would understate the work by
// most of it.
func (u Usage) Tokens() int {
	return u.InputTokens + u.OutputTokens + u.CacheReadTokens + u.CacheWriteTokens
}

// Add accumulates another invocation's usage. Cost stays unknown until some
// invocation reports one, so a panel mixing reporting and non-reporting CLIs
// totals what it actually knows.
func (u *Usage) Add(o Usage) {
	u.InputTokens += o.InputTokens
	u.OutputTokens += o.OutputTokens
	u.CacheReadTokens += o.CacheReadTokens
	u.CacheWriteTokens += o.CacheWriteTokens
	u.CostUSD += o.CostUSD
	u.CostKnown = u.CostKnown || o.CostKnown
}

// RunSummary is the artifact written when a run ends, however it ends.
type RunSummary struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	ConfigPath string    `json:"config_path"`
	// Sources records every bundle file the run was built from. Prompts steer
	// agents that edit code and the config holds the trust gates, so which FILE
	// each name resolved to is part of the run's record -- a project-local prompt
	// shadowing the installed one is otherwise invisible after the fact.
	Sources RunSources `json:"sources"`
	Mode    string     `json:"mode"`
	// PR is the pull request this run reviewed, in pr mode. Recorded because the
	// number is part of "what was run" and because publishing a finished run later
	// has to know where it goes.
	PR int `json:"pr,omitempty"`
	// ReviewedHead is the commit the run reviewed, in pr mode: the head `gh pr
	// checkout` left in the tree. A review is a statement about ONE commit, and the
	// pull request can move while the panel runs or between the run and a later
	// -post-run, so publishing compares this against the forge's current head and
	// refuses when they differ -- otherwise an approval lands on code no reviewer
	// read. Empty for a run from before it was recorded, which the posting paths
	// treat as "cannot be bound" and refuse.
	ReviewedHead string `json:"reviewed_head,omitempty"`
	// ReviewedRepo is WHICH repository that commit belongs to, in pr mode: the
	// canonical host/owner/repo gh resolved for the checkout (see forge.RepoID).
	// Path and PR alone cannot say -- a path is not an identity, and the checkout
	// occupying it can be repointed at another repository on the same forge or the
	// directory reused for one, at which point a later -post-run would publish to
	// pull request PR of THAT repository. The reviewed head does not catch it: the
	// commit is public, so anyone may open a request proposing it. So publishing
	// compares this against the repository the checkout resolves to now and refuses
	// when they differ. Empty for a run from before it was recorded, which -post-run
	// treats as "cannot be bound" and refuses.
	ReviewedRepo string `json:"reviewed_repo,omitempty"`
	Path         string `json:"path"`
	Strategy     string `json:"strategy"`
	ReviewOnly   bool   `json:"review_only"`
	// The effective limits and command-line assertions the run used. They are what
	// make the counts readable after the fact: "17 deferred" means nothing without
	// the per-round cap that deferred them, and "fix rounds ran" needs the flag that
	// authorized them, since no config file may grant that.
	MaxIterations       int           `json:"max_iterations,omitempty"`
	MaxFindingsPerRound int           `json:"max_findings_per_round,omitempty"`
	CommitPolicy        string        `json:"commit_policy,omitempty"`
	Overrides           []string      `json:"overrides,omitempty"`
	Coder               string        `json:"coder,omitempty"`
	Rounds              []RoundRecord `json:"rounds"`
	Termination         string        `json:"termination"`
	// LoopTermination preserves how the LOOP ended when the closing round then
	// failed (Termination becomes error) or was interrupted (interrupted). The two
	// facts are different -- the loop can genuinely have converged while the final
	// round's reviewer or coder failed afterwards, or while the operator stopped it
	// -- and collapsing them would either hide how the run really ended or discard
	// the loop's outcome.
	LoopTermination string `json:"loop_termination,omitempty"`
	Error           string `json:"error,omitempty"`
	// ReviewBody is where the rendered review document was written, for a review
	// run. It is the file an operator reads, and on the posting path the exact
	// bytes that were sent.
	ReviewBody string `json:"review_body,omitempty"`
	// Implement marks an implement-design run; Tasks carries its per-task
	// outcomes for the scoreboard, in plan order.
	Implement bool          `json:"implement,omitempty"`
	Tasks     []TaskOutcome `json:"tasks,omitempty"`
	// Create marks a create-design run, whose summary reads in that pipeline's
	// vocabulary rather than the loop's, and Deliverable is where it published
	// its document -- recorded so the summary can answer "where did it go"
	// without re-deriving the default.
	Create      bool   `json:"create,omitempty"`
	Deliverable string `json:"deliverable,omitempty"`
	// ReviewPosted names the event a review was published as ("comment",
	// "approve", "request_changes"), or is empty when nothing reached the forge. An
	// operator reading a summary should not have to infer from a log line whether
	// their -post actually reached the forge.
	//
	// "Published" is not the same as "the run succeeded": it is recorded whenever
	// the forge accepted the submission, including when the call then failed
	// afterwards -- an approval whose head moved between the check and the submit is
	// on the pull request, and a summary that denied it would send the operator
	// looking for a review a human has to dismiss.
	ReviewPosted string `json:"review_posted,omitempty"`
	// ReviewInline are the anchors the run computed for its findings, kept so a
	// later publish sends exactly what this run produced rather than recomputing it
	// against a diff that may have moved.
	ReviewInline []ReviewAnchor `json:"review_inline,omitempty"`
	// Verdict is set for a review-only run: what the review concluded, and why.
	// A fix run has no verdict -- its outcome is the commits it made and the
	// termination above.
	Verdict *ReviewVerdict `json:"verdict,omitempty"`
}

// ReviewVerdict is the serializable form of a review's conclusion. The rule that
// produces it lives in internal/review; this is only how the summary carries it,
// which is why it holds issue IDS rather than issues -- the findings themselves are
// already in the rounds, and duplicating them here would let the two disagree.
type ReviewVerdict struct {
	Outcome  string   `json:"outcome"`
	Reasons  []string `json:"reasons"`
	Blocking []string `json:"blocking,omitempty"`
	Panel    int      `json:"panel"`
	Present  int      `json:"present"`
	Required int      `json:"required"`
	Missing  []string `json:"missing,omitempty"`
}

// ExitCode maps a termination to the process exit status, so the run summary and
// the CLI cannot disagree about what a run meant.
//
// all-rejected is deliberately NOT 0: "the coder rejected every finding" could
// equally mean the reviewers are miscalibrated or the coder was unwilling --
// automation keying on 0 would read a stalled run as a clean one. It does not
// mean the run committed nothing: the verdict covers the LAST loop round, so
// earlier rounds and the closing round may both have landed fixes. Read Rounds
// for that, not the exit status.
func ExitCode(termination string) int {
	switch termination {
	case TermConverged, TermReviewOnly, TermCreated, TermImplemented, TermPlanned:
		return 0
	case TermMaxIterations, TermIncomplete:
		return 2
	case TermAllRejected:
		return 3
	default: // interrupted, error
		return 1
	}
}

// ExitCodeFor is ExitCode with a review run's VERDICT taken into account, and it
// is what both the CLI and the summary must call: a review that requested changes
// terminated perfectly normally, so the termination alone would exit 0 and tell
// automation the branch was fine.
//
// A verdict only ever makes the status WORSE. A review whose loop errored or was
// interrupted keeps that exit code, because an incomplete run's approval is not
// an approval -- and Decide cannot approve without quorum anyway, so the two
// agree rather than compete.
func ExitCodeFor(sum *RunSummary) int {
	if sum == nil {
		return ExitCode("")
	}
	base := ExitCode(sum.Termination)
	if sum.Verdict == nil || base != 0 {
		return base
	}
	switch sum.Verdict.Outcome {
	case VerdictChangesRequested:
		return ExitChangesRequested
	case VerdictInconclusive:
		return ExitInconclusive
	}
	return base
}

// Verdict outcomes, mirroring internal/review's Outcome values. They are declared
// here too because the summary is decoded by tools that must not have to import
// the decision logic to read what it decided.
const (
	VerdictApprove          = "approve"
	VerdictChangesRequested = "changes_requested"
	VerdictInconclusive     = "inconclusive"
)

// Exit codes above the terminations': a review verdict is a different axis from
// how the loop ended, so it gets its own numbers rather than overloading
// all-rejected.
const (
	ExitChangesRequested = 4
	ExitInconclusive     = 5
)

// Termination reasons.
const (
	TermConverged     = "converged"    // clean_rounds_to_stop consecutive clean rounds
	TermAllRejected   = "all-rejected" // coder rejected every finding in a round
	TermReviewOnly    = "review-only"  // single review round requested
	TermMaxIterations = "max-iterations"
	TermInterrupted   = "interrupted"
	TermError         = "error"
	TermCreated       = "created"     // a create run published its deliverable
	TermImplemented   = "implemented" // an implement run built every task
	TermIncomplete    = "incomplete"  // an implement run finished with holes: failed/blocked/skipped tasks, the deadline, the vacuous guard, or a clean-check failure
	TermPlanned       = "planned"     // -plan-only: the plan was made and validated, and nothing was built
)

// RunSources is the provenance of one run's configuration: which file each
// bundle name resolved to on the search path.
type RunSources struct {
	Config  string            `json:"config"`
	Extends string            `json:"extends,omitempty"`
	Agents  map[string]string `json:"agents,omitempty"`
	Prompts map[string]string `json:"prompts,omitempty"`
}

// VerifyResult is one deterministic-gate command's outcome. It lives here rather
// than in the verify package because it is a shape shared by the orchestrator, the
// round record, and the logs -- and keeping it here also stops model from
// depending on a package that runs subprocesses.
type VerifyResult struct {
	Name     string   `json:"name"`
	Argv     []string `json:"argv"`
	Optional bool     `json:"optional,omitempty"`
	// Infra marks the command as environment-dependent (config.VerifyCommand.Infra),
	// so a caller can tell "the registry was down" from "the code is wrong".
	Infra    bool          `json:"infra,omitempty"`
	ExitCode int           `json:"exit_code"`
	Passed   bool          `json:"passed"`
	Output   string        `json:"output,omitempty"` // combined stdout+stderr, capped and redacted
	Duration time.Duration `json:"duration"`
	Err      string        `json:"error,omitempty"` // could not run at all (not a non-zero exit)
}

// VerifyRun is ONE run of the deterministic gate: which fix it gated, on what
// occasion, and what the checks did. The gate runs per fix rather than per round,
// so this -- not the round -- is the granularity at which its verdict is true.
type VerifyRun struct {
	// Issue is the issue whose fix this run gated. Empty on a salvage pass, which
	// gates a failed coder's partial work rather than any one issue's fix.
	Issue string `json:"issue,omitempty"`
	// Attempt is one of VerifyAttempt*: the occasion for this run, so a correction
	// attempt can be told apart from the initial pass it followed.
	Attempt string         `json:"attempt,omitempty"`
	Results []VerifyResult `json:"results,omitempty"`
	// Blocking names the checks that actually blocked under the active policy, which
	// is narrower than "the checks that failed": under no_regressions a check already
	// red in the pre-run baseline fails without blocking. Persisted because Results
	// alone cannot answer "did the gate clear" after the fact -- the baseline it was
	// judged against is gone.
	Blocking []string `json:"blocking,omitempty"`
}

// LastVerify returns the round's most recent gate run -- the one whose verdict the
// round is currently acting on. ok is false when the gate has not run at all.
func (r RoundRecord) LastVerify() (VerifyRun, bool) {
	if len(r.Verify) == 0 {
		return VerifyRun{}, false
	}
	return r.Verify[len(r.Verify)-1], true
}

// Issue statuses. An issue's status is its own, tracked across rounds, and is
// deliberately separate from the observations that reported it: several reviewers
// can corroborate one issue, and it is the ISSUE that gets fixed or rejected.
const (
	StatusOpen = "open"
)

// Issue is one distinct problem, aggregated from every observation that reported
// it. It exists because findings alone conflated three things: a reviewer's raw
// report, the unit of work handed to the coder, and the thing whose state persists
// across rounds.
//
// The practical consequence of that conflation was a bug, not just untidiness:
// two agents reporting the same problem produced two findings, each consuming a
// slot against loop.max_findings_per_round -- so agreement between reviewers
// REDUCED how many distinct problems got fixed in a round. Corroboration is
// signal; it should not cost budget.
type Issue struct {
	ID          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Status      string `json:"status"` // open | fixed | rejected | deferred
	Category    string `json:"category"`
	Severity    string `json:"severity"` // worst severity any observation assigned
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
	Advisory    bool   `json:"advisory,omitempty"`

	// Observations are every raw report grouped under this issue, preserved rather
	// than collapsed: which agent and which lens found a problem is evidence about
	// the panel, and the coder benefits from more than one description of it.
	Observations []Finding `json:"observations,omitempty"`

	// FirstRound is where the issue was first seen and Deferrals counts the rounds
	// it was deferred by the cap. Deferrals drives the cap's aging directly, which
	// replaces the earlier (file, category) approximation of identity.
	FirstRound int `json:"first_round"`
	LastRound  int `json:"last_round"`
	Deferrals  int `json:"deferrals,omitempty"`

	Verdict       string `json:"verdict,omitempty"`
	VerdictDetail string `json:"verdict_detail,omitempty"`
	// Origin is where this issue came from, when it was not the panel, and Also
	// holds the OTHER conversations that turned out to be about the same defect.
	//
	// Two people reporting one problem in two comments is the ordinary case, and the
	// ledger merges them into one issue -- correctly, since it is one fix. But each
	// of those conversations is a person waiting for an answer, so keeping only the
	// first left the second reported as commissioned and never replied to. Every
	// linked conversation is answered when the fix commits. See Finding.Origin.
	Origin Origin   `json:"origin,omitempty"`
	Also   []Origin `json:"also,omitempty"`
	// Contested records that the refutation round disagreed about this finding:
	// somebody who looked at it did not believe it, or nobody could decide. It is
	// kept -- one reviewer still standing behind a defect is enough -- but a reader
	// deciding what to do about it should know the panel split.
	Contested bool `json:"contested,omitempty"`
	// ContestedBy names the refuters whose position produced that doubt, sorted. The
	// boolean alone cannot say WHO doubted the finding, and the judge gate needs
	// exactly that: the same agent is routinely both a panel refuter and the judge,
	// so a drop corroborated only by that agent's own refutation is one agent's word
	// twice, not two agents agreeing. See applyJudgment.
	ContestedBy []string `json:"contested_by,omitempty"`
}

// Agents returns the distinct agents that reported this issue, sorted. Two
// independent agents agreeing is the corroboration signal the coder is shown.
func (i Issue) Agents() []string {
	seen := map[string]bool{}
	var out []string
	for _, o := range i.Observations {
		if o.Agent != "" && !seen[o.Agent] {
			seen[o.Agent] = true
			out = append(out, o.Agent)
		}
	}
	sort.Strings(out)
	return out
}

// Loc renders the issue location for display, like Finding.Loc.
func (i Issue) Loc() string {
	if i.Line > 0 {
		return fmt.Sprintf("%s:%d", i.File, i.Line)
	}
	return i.File
}

// StatusOrDefault reports the issue's status, defaulting to open.
func (i Issue) StatusOrDefault() string {
	if i.Status == "" {
		return StatusOpen
	}
	return i.Status
}

// RefuteOutput is a reviewer's answer in the refutation round: one position on
// every finding it was shown.
type RefuteOutput struct {
	Positions []RefutePosition `json:"positions"`
}

// RefutePosition is one reviewer's stance on one issue.
//
// Evidence is required by the contract for every position, not just a refutation.
// A "maintain" with no evidence is indistinguishable from a reviewer that did not
// look, and the round exists precisely to find out which findings anyone can still
// stand behind after seeing them written down.
type RefutePosition struct {
	Issue    string `json:"issue"`
	Position string `json:"position"`
	Evidence string `json:"evidence"`
}

// The positions a refuter may take. Only Refute removes a finding, and only
// unanimously -- see the orchestrator's applyRefutations.
const (
	PositionMaintain = "maintain"
	PositionRefute   = "refute"
	PositionUnsure   = "unsure"
)

// ValidPosition reports whether p is one a refuter may return.
func ValidPosition(p string) bool {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case PositionMaintain, PositionRefute, PositionUnsure:
		return true
	}
	return false
}

// JudgeOutput is the arbiter's answer: one verdict on every finding it was shown.
type JudgeOutput struct {
	Verdicts []JudgeVerdict `json:"verdicts"`
}

// JudgeVerdict is keep-or-drop on one issue, with the reason recorded.
//
// The reason is not decoration. A dropped finding disappears from the review, and
// the only thing standing between that and an unaccountable filter is a sentence a
// human can read afterwards and disagree with.
type JudgeVerdict struct {
	Issue   string `json:"issue"`
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

// What a judge may decide.
const (
	JudgeKeep = "keep"
	JudgeDrop = "drop"
)

// TriageOutput is the conversation-triage reply.
type TriageOutput struct {
	Decisions []TriageDecision `json:"decisions"`
}

// TriageDecision is accept-or-reject on one pull-request conversation.
//
// An accepted decision carries a whole finding, written by triage rather than
// quoted from the comment: the coder acts on these words, and a comment that says
// "this looks wrong to me" is not something anyone can fix. A rejected one carries
// only the reason, which is posted verbatim as the reply -- so it is addressed to
// the person who commented, not about them.
type TriageDecision struct {
	Thread  string `json:"thread"`
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`

	// Set on accept.
	Title       string `json:"title,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Category    string `json:"category,omitempty"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Description string `json:"description,omitempty"`
}

// What triage may decide.
const (
	TriageAccept = "accept"
	TriageReject = "reject"
)

// ValidTriageVerdict reports whether v is one triage may return.
func ValidTriageVerdict(v string) bool {
	switch NormalizeTriageVerdict(v) {
	case TriageAccept, TriageReject:
		return true
	}
	return false
}

// NormalizeTriageVerdict is the canonical spelling of a triage verdict: what
// ValidTriageVerdict actually checked, and therefore the only form worth STORING.
// The same trim-in-the-validator gap NormalizeSeverity documents applies here, and
// costs more: a stored "accept\n" validates as an acceptance and then fails the
// dispatch comparison, so the work is never commissioned and the acceptance's
// reason is posted to a human as the decline that explains it.
func NormalizeTriageVerdict(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

// ValidJudgeVerdict reports whether v is one the judge may return.
func ValidJudgeVerdict(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case JudgeKeep, JudgeDrop:
		return true
	}
	return false
}

// FixReply is the coder's answer to one open review conversation.
//
// fixpoint does not compose these. A reply appears under a human's comment with
// the operator's identity on it, so the words have to come from the agent that
// actually did the work and can say what it changed -- not from a template that
// claims something happened.
type FixReply struct {
	Thread  string `json:"thread"`
	Message string `json:"message"`
}

// ReviewAnchor is one inline comment as the summary carries it.
type ReviewAnchor struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Body string `json:"body"`
}

// Origin identifies a pull-request conversation a finding was commissioned by.
//
// External says the comment's author is NOT the account fixpoint is authenticated
// as -- so it is neither the operator nor anything fixpoint itself posted. Such a
// request is still acted on (a colleague reviewing your pull request is the normal
// case), but it is labeled everywhere it travels: in the coder's prompt and in
// the commit. Somebody reading the history later should be able to see that a
// change was asked for by a third party, without reconstructing it from the pull
// request.
//
// Unknown authorship counts as external. The check is "does this match the login
// gh reports", and a login it could not read proves nothing.
type Origin struct {
	Thread   string `json:"thread,omitempty"`
	Author   string `json:"author,omitempty"`
	External bool   `json:"external,omitempty"`
}

// FromConversation reports whether the finding was commissioned by a comment.
func (o Origin) FromConversation() bool { return o.Thread != "" }

// Conversations lists every thread waiting on this issue, primary first.
func (i Issue) Conversations() []Origin {
	if !i.Origin.FromConversation() {
		return nil
	}
	return append([]Origin{i.Origin}, i.Also...)
}

// CritiqueOutput is the CRITIQUE phase's reply: one structured judgment per
// proposal the critic was shown.
type CritiqueOutput struct {
	Critiques []Critique `json:"critiques"`
}

// Critique is one critic's judgment of one anonymized proposal. Strengths and
// adopt are what the editor builds from; weaknesses are what it must answer or
// record as dissent.
type Critique struct {
	Proposal   string   `json:"proposal"`
	Strengths  []string `json:"strengths"`
	Weaknesses []string `json:"weaknesses"`
	Adopt      []string `json:"adopt"`
}

// Empty reports a critique carrying no content at all -- indistinguishable from
// the critic not having read the proposal, and refused by the caller for that
// reason.
func (c Critique) Empty() bool {
	return len(c.Strengths) == 0 && len(c.Weaknesses) == 0 && len(c.Adopt) == 0
}

// ObjectionOutput is the OBJECT pass's reply: the blocking defects one panel
// member sees in the editor's draft. Zero objections is a normal answer -- the
// draft may simply hold up.
type ObjectionOutput struct {
	Objections []Objection `json:"objections"`
}

// Objection is one blocking defect in the draft: the passage it names, what is
// wrong with it, and what happens if it ships. "I would have chosen differently"
// is not an objection, and the contract says so; the structure exists so the
// handoff to REVISE is machine-checkable rather than prose the editor may miss.
type Objection struct {
	Passage     string `json:"passage"`
	Defect      string `json:"defect"`
	Consequence string `json:"consequence"`
}

// Substantial reports whether an objection carries enough to act on: a defect at
// minimum. A passage alone is a pointer with no claim, and REVISE cannot address
// a claim that was never made.
func (obj Objection) Substantial() bool { return len([]rune(obj.Defect)) > 0 }
