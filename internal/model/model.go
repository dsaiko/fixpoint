// Package model defines the data shapes exchanged between reviewers, the
// coder, the orchestrator, and the logs.
package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
	// CommitSHA is the commit the round RESULTS in: the last of its per-fix commits,
	// or the squash that replaced them. Commits holds every commit the round made,
	// which is what the scoreboard counts -- under commit_policy: per_fix a round of
	// eight fixes is eight commits, and reporting only the last would undercount.
	CommitSHA string   `json:"commit_sha,omitempty"`
	Commits   []string `json:"commits,omitempty"`
	// CoderError is set when the coder failed mid-round but its partial edits
	// were salvaged into CommitSHA; the loop then continued.
	CoderError string `json:"coder_error,omitempty"`
	// Verify holds the deterministic gate's results for this round, and
	// VerifyRetried records that the coder was given a correction attempt. Kept on
	// the round record so the summary can show what actually passed -- the one
	// non-model signal in the loop deserves to be persisted, not just logged.
	Verify []VerifyResult `json:"verify,omitempty"`
	// VerifyBlocking names the checks that actually blocked the round under the
	// active policy, which is narrower than "the checks that failed": under
	// no_regressions a check already red in the pre-run baseline fails without
	// blocking. Persisted because Verify alone cannot answer "did the gate clear
	// this round" after the fact -- the baseline it was judged against is gone.
	VerifyBlocking []string   `json:"verify_blocking,omitempty"`
	VerifyRetried  bool       `json:"verify_retried,omitempty"`
	Steps          []StepStat `json:"steps,omitempty"` // per-invocation I/O figures
	// Final marks the closing round that `final: true` lenses run in, after the loop
	// has stopped. It is not part of the convergence story -- it happens once the
	// run's outcome is already decided -- so a reader must be able to tell it apart.
	Final bool `json:"final,omitempty"`
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
	Sources    RunSources `json:"sources"`
	Mode       string     `json:"mode"`
	Path       string     `json:"path"`
	Strategy   string     `json:"strategy"`
	ReviewOnly bool       `json:"review_only"`
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
	// failed and rewrote Termination to error. The two facts are different -- the
	// loop can genuinely have converged while the final round's reviewer or coder
	// failed afterwards -- and collapsing them would either hide the failure or
	// discard the loop's outcome.
	LoopTermination string `json:"loop_termination,omitempty"`
	Error           string `json:"error,omitempty"`
}

// ExitCode maps a termination to the process exit status, so the run summary and
// the CLI cannot disagree about what a run meant.
//
// all-rejected is deliberately NOT 0: "the coder rejected every finding" could
// equally mean the reviewers are miscalibrated or the coder was unwilling, and
// nothing changed -- automation keying on 0 would read a no-op as a clean run.
func ExitCode(termination string) int {
	switch termination {
	case TermConverged, TermReviewOnly:
		return 0
	case TermMaxIterations:
		return 2
	case TermAllRejected:
		return 3
	default: // interrupted, error
		return 1
	}
}

// Termination reasons.
const (
	TermConverged     = "converged"    // clean_rounds_to_stop consecutive clean rounds
	TermAllRejected   = "all-rejected" // coder rejected every finding in a round
	TermReviewOnly    = "review-only"  // single review round requested
	TermMaxIterations = "max-iterations"
	TermInterrupted   = "interrupted"
	TermError         = "error"
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
	Name     string        `json:"name"`
	Argv     []string      `json:"argv"`
	Optional bool          `json:"optional,omitempty"`
	ExitCode int           `json:"exit_code"`
	Passed   bool          `json:"passed"`
	Output   string        `json:"output,omitempty"` // combined stdout+stderr, capped and redacted
	Duration time.Duration `json:"duration"`
	Err      string        `json:"error,omitempty"` // could not run at all (not a non-zero exit)
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
