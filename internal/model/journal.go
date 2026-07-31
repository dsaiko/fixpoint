package model

import "time"

// JournalVersion is the schema version stamped on every record. A reader that
// does not recognize it must refuse the file rather than guess: the journal is
// meant to become the input to a resume, and silently misreading a run's state is
// worse than declining to resume it.
const JournalVersion = 1

// JournalEvent is one state transition, written as a single line of
// journal.jsonl at the run root as it happens.
//
// It exists because the run summary is written ONCE, at the end. That makes the
// summary useless for the two questions that matter when a run goes wrong: what
// order did things happen in, and what was true at the moment it died. A run
// killed mid-round leaves a summary that describes a run that never finished, or
// no summary at all if the process was killed outright -- while the journal has
// already recorded every transition up to the last one.
//
// Records are append-only and never rewritten. Seq is authoritative for ordering,
// because At comes from the wall clock, which can repeat within a timestamp
// interval and can move backwards.
//
// Data carries a per-type payload (the Journal* types below), rather than one wide
// struct shared by every event: with a shared struct an absent count would mean
// both "zero" and "not part of this event", and "the coder fixed 0 issues" is a
// fact an audit log must be able to state.
type JournalEvent struct {
	V     int       `json:"v"`
	Seq   int       `json:"seq"`
	At    time.Time `json:"at"`
	Type  string    `json:"type"`
	Round int       `json:"round,omitempty"` // absent for run-level events
	Data  any       `json:"data,omitempty"`
}

// Journal event types. Each names a transition the loop actually makes, so the
// sequence in a journal reconstructs the state machine rather than narrating it.
const (
	EvRunStarted       = "run_started"
	EvVerifyBaseline   = "verify_baseline"
	EvRoundStarted     = "round_started"
	EvReviewFinished   = "review_finished"
	EvIssuesAggregated = "issues_aggregated"
	EvIssuesDeferred   = "issues_deferred"
	EvFixFinished      = "fix_finished"
	EvVerifyFinished   = "verify_finished"
	EvRoundCommitted   = "round_committed"
	EvRoundDiscarded   = "round_discarded"
	EvRoundClean       = "round_clean"
	EvRunFinished      = "run_finished"
)

// The occasions the deterministic gate runs. Named because "verification failed"
// means something different each time: the coder still gets a correction attempt
// after Initial, gets none after Correction, and after Salvage the run stops.
const (
	VerifyAttemptInitial    = "initial"
	VerifyAttemptCorrection = "correction"
	VerifyAttemptSalvage    = "salvage"
)

// Reasons a round's work did not become a commit. Every one of these leaves the
// working tree changed and then restores it, so which one happened is the first
// thing anyone asks when a run ends with a stash they did not expect.
const (
	DiscardVerifyFailed  = "verify_failed"
	DiscardSalvageFailed = "salvage_verify_failed"
	DiscardInterrupted   = "interrupted"
	DiscardCommitFailed  = "commit_failed"
	// DiscardRejectedWithEdits is the round where the coder rejected every issue
	// yet edited files: no verdict claims the edits, so they are stashed instead
	// of committed.
	DiscardRejectedWithEdits = "rejected_with_edits"
)

// JournalRunStarted records what the run was configured to do. It repeats values
// the summary also carries, deliberately: a journal has to be readable on its own
// when the summary was never written.
type JournalRunStarted struct {
	Config        string `json:"config"`
	Mode          string `json:"mode"`
	Path          string `json:"path"`
	Strategy      string `json:"strategy"`
	ReviewOnly    bool   `json:"review_only"`
	MaxIterations int    `json:"max_iterations"`
	// Overrides names the flag assertions that shaped this run. A trust gate is a
	// per-invocation flag and never a config key, so without this the journal
	// cannot answer why fix rounds were permitted at all.
	Overrides []string `json:"overrides,omitempty"`
}

// JournalRunFinished is the last record of a completed run. Its absence is itself
// information: the process died before it could finish.
type JournalRunFinished struct {
	Termination string `json:"termination"`
	Rounds      int    `json:"rounds"`
	Error       string `json:"error,omitempty"`
}

// JournalRoundStarted records the lens-to-agent assignment, which under
// strategy: rotate differs every round and decides who confirmed a clean result.
type JournalRoundStarted struct {
	Assignments []string `json:"assignments"` // "lens->agent", advisory ones marked
}

// JournalReviewFinished separates the three outcomes a review round has: findings
// that reach the coder, advisory ones that never do, and reviewers that failed --
// which is why a clean round is not automatically a converging one.
type JournalReviewFinished struct {
	Observations int      `json:"observations"`
	Advisory     int      `json:"advisory"`
	Errors       []string `json:"errors,omitempty"`
}

// JournalIssuesAggregated records corroboration. Two agents reporting one problem
// is the strongest evidence a panel produces, and it is invisible in the raw
// observation count.
type JournalIssuesAggregated struct {
	Observations int `json:"observations"`
	Issues       int `json:"issues"`
	Corroborated int `json:"corroborated"`
}

// JournalIssuesDeferred records the per-round cap biting. Deferral is a scheduling
// decision with a history of starving the tail, so which issues waited -- and how
// many times -- needs to be recoverable after the fact.
type JournalIssuesDeferred struct {
	Cap      int      `json:"cap"`
	Active   int      `json:"active"`
	Deferred int      `json:"deferred"`
	IDs      []string `json:"ids,omitempty"` // the deferred issues, worst-first
}

// JournalFixFinished is the coder's self-report: a claim, not evidence. The
// verify_finished record that follows is the part no model produced.
type JournalFixFinished struct {
	Agent    string `json:"agent"`
	Issues   int    `json:"issues"` // handed to the coder this round
	Fixed    int    `json:"fixed"`
	Rejected int    `json:"rejected"`
	Error    string `json:"error,omitempty"`
}

// JournalVerifyFinished is one run of the deterministic gate. Blocking is narrower
// than the failing checks: under no_regressions a check that was already failing
// at the baseline fails without blocking the round.
type JournalVerifyFinished struct {
	Attempt  string         `json:"attempt,omitempty"` // absent on the baseline
	Policy   string         `json:"policy"`
	Passed   bool           `json:"passed"`
	Checks   []JournalCheck `json:"checks,omitempty"`
	Blocking []string       `json:"blocking,omitempty"`
}

// JournalCheck is one gate command's outcome, without its output: the journal is
// an index of what happened, and the full output is already in the round's logs.
type JournalCheck struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Optional bool   `json:"optional,omitempty"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"` // could not run at all
}

// JournalRoundCommitted records a round becoming a commit. Partial marks work
// salvaged from a coder that died before reporting verdicts, which is committed
// with unknown verdicts and re-reviewed next round.
type JournalRoundCommitted struct {
	SHA     string `json:"sha"`
	Partial bool   `json:"partial,omitempty"`
	Fixed   int    `json:"fixed"`
}

// JournalRoundDiscarded records work that was NOT committed and the tree restored.
// Stashed says whether it is recoverable with `git stash pop` or was never there
// to begin with.
type JournalRoundDiscarded struct {
	Reason  string   `json:"reason"`
	Stashed bool     `json:"stashed"`
	Checks  []string `json:"checks,omitempty"` // the blocking checks, when that is the reason
	Error   string   `json:"error,omitempty"`
}

// JournalRoundClean records a zero-finding round and the convergence streak it
// contributes to. Reviewer errors reset the streak, so a clean round with errors
// is recorded as clean but not as progress.
type JournalRoundClean struct {
	Streak       int `json:"streak"`
	Needed       int `json:"needed"`
	ReviewErrors int `json:"review_errors,omitempty"`
}
