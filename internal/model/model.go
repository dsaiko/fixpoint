// Package model defines the data shapes exchanged between reviewers, the
// coder, the orchestrator, and the logs.
package model

import (
	"encoding/json"
	"errors"
	"fmt"
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
	Round        int          `json:"round"`
	Assignments  []Assignment `json:"assignments"`
	Findings     []Finding    `json:"findings"`           // non-advisory, with verdicts after fix
	Advisory     []Finding    `json:"advisory,omitempty"` // report-only findings
	ReviewErrors []string     `json:"review_errors,omitempty"`
	Fixed        int          `json:"fixed"`
	Rejected     int          `json:"rejected"`
	CommitSHA    string       `json:"commit_sha,omitempty"`
	// CoderError is set when the coder failed mid-round but its partial edits
	// were salvaged into CommitSHA; the loop then continued.
	CoderError string `json:"coder_error,omitempty"`
	// Verify holds the deterministic gate's results for this round, and
	// VerifyRetried records that the coder was given a correction attempt. Kept on
	// the round record so the summary can show what actually passed -- the one
	// non-model signal in the loop deserves to be persisted, not just logged.
	Verify        []VerifyResult `json:"verify,omitempty"`
	VerifyRetried bool           `json:"verify_retried,omitempty"`
	Steps         []StepStat     `json:"steps,omitempty"` // per-invocation I/O figures
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
	Sources     RunSources    `json:"sources"`
	Mode        string        `json:"mode"`
	Path        string        `json:"path"`
	Strategy    string        `json:"strategy"`
	ReviewOnly  bool          `json:"review_only"`
	Rounds      []RoundRecord `json:"rounds"`
	Termination string        `json:"termination"`
	Error       string        `json:"error,omitempty"`
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
