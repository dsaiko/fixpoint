package model

import "strings"

// Severities is the ONE authoritative severity vocabulary, ordered worst-first.
//
// It lives in model because severity is a scheduling input read by more than one
// package: the orchestrator validates what reviewers report and orders the
// per-round cap by it, and the issue ledger keeps the worst severity any
// observation of an issue assigned. Every consumer must agree on both the
// membership set and the ordering.
//
// This was a real defect, twice. The vocabulary was first declared in two places
// inside the orchestrator (an ordering map and a validation set) which fixpoint's
// own reviewers found -- the example in ReviewFinding.Issue quotes the three
// different ways they described it across rounds. It was then reintroduced when the
// issue ledger grew its own private copy of the same four strings. Keeping the list
// here, with the derived forms below built from it, is what makes a third
// recurrence impossible rather than merely unlikely: there is no second literal to
// drift.
var Severities = []string{"critical", "high", "medium", "low"}

// DefaultBlockAt is the severity floor at or above which a surviving finding
// forces CHANGES_REQUESTED.
//
// high, not medium, and the choice is measured rather than tasteful. Across 19
// runs the panel produced 322 issues of which 66 were high or critical: a medium
// floor would block essentially every review, and a gate that always fires is one
// people learn to bypass. It is deliberately permissive on the model's half --
// which is why the deterministic half (failing CI) matters as much as this one.
//
// DefaultRefuteAt is the floor for the refutation round, and it EQUALS this one on
// purpose: the round earns its cost as the gate that stops a single judge from
// deleting a blocking finding, so what it must cover is exactly what blocks.
//
// Both live here rather than beside their consumers for the reason stated above
// Severities: severity is read by more than one package -- review applies these two
// as fallbacks, the orchestrator selects on them, config validates against them --
// and a floor that differed between the validator and the applier would be a load
// error nobody could act on.
const (
	DefaultBlockAt  = "high"
	DefaultRefuteAt = DefaultBlockAt
)

// severityRank maps a severity to its position in Severities.
var severityRank = func() map[string]int {
	m := make(map[string]int, len(Severities))
	for i, s := range Severities {
		m[s] = i
	}
	return m
}()

// SeverityRank orders severities worst-first (critical = 0). Unknown values sort
// LAST, so a reviewer inventing a severity cannot outrank a real critical and
// jump the per-round cap's queue.
//
// Input is normalized: severity arrives as free text from a model, so "High" and
// " high " have to mean high. ValidSeverity is what rejects a genuinely unknown
// value; this function must still order one sanely if it reaches here.
func SeverityRank(s string) int {
	if r, ok := severityRank[NormalizeSeverity(s)]; ok {
		return r
	}
	return len(Severities)
}

// WorseSeverity reports whether a is more severe than b.
func WorseSeverity(a, b string) bool { return SeverityRank(a) < SeverityRank(b) }

// ValidSeverity reports whether s is in the closed vocabulary reviewers may use
// (the same set stated in the review output contract).
func ValidSeverity(s string) bool {
	_, ok := severityRank[NormalizeSeverity(s)]
	return ok
}

// NormalizeSeverity is the canonical spelling of a severity: what ValidSeverity
// actually checked, and therefore the only form worth STORING.
//
// The raw spelling and the validated one are not the same string, and that gap
// was a defect: ValidSeverity accepts "high\r" or "High\n\n" because it trims
// first, but the untouched value kept flowing into the ledger, the summary and a
// commit body -- where an embedded newline forges message lines and a CR makes
// `git log` render something other than the stored bytes. Reviewers on an
// untrusted tree are prompt-injectable, so severity gets normalized where a
// finding is built and the raw agent text stops there.
func NormalizeSeverity(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
