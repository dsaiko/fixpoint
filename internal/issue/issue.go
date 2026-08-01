// Package issue groups raw reviewer observations into distinct issues and tracks
// each issue's state across rounds.
//
// Why this exists: a finding used to be three things at once -- one reviewer's
// report, the unit of work handed to the coder, and the thing whose state persists
// between rounds. Conflating them produced a real bug. Two agents reporting the
// same problem produced two findings, each consuming a slot against
// loop.max_findings_per_round, so reviewers AGREEING reduced how many distinct
// problems got fixed that round. It also left cross-round identity to an
// approximation -- (file, category) -- which the cap's aging had to rely on.
package issue

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dsaiko/fixpoint/internal/model"
)

// Two reports of the same FILE whose titles agree are one issue, however far apart
// their lines are. The evidence for sameness is the title; the line is not evidence
// at all in a repository this tool is actively editing.
//
// This used to require the lines to be within a few of each other, on the reasoning
// that reviewers point at slightly different lines for one defect -- the
// declaration, the use, the enclosing function. True, but far too narrow: a fix run
// MOVES code, so a deferred issue re-reported next round is typically tens of lines
// from where it was. In one five-round run the same complaint was minted four times
// (orchestrator.go:1343, :1399, :1453, then a different file) as the rounds above it
// grew by ~55 lines each. That is not merely a duplicate costing a slot, which is
// what the conservative stance was priced against: a new id has NO deferral history,
// so severity aging restarted every round and the issue was deferred forever. The
// run never converged.
//
// Merging two distinct problems is still worse than leaving a duplicate, so the
// title agreement below is what carries the decision -- three unrelated defects in
// one file are three issues, because their titles do not agree. Only the line
// window is gone.

// Ledger accumulates issues over a run. It is not safe for concurrent use: the
// parallel part of a round is the reviewers, and aggregation happens after their
// barrier.
type Ledger struct {
	issues []model.Issue
	byID   map[string]int
	seq    int
}

// NewLedger returns an empty ledger.
func NewLedger() *Ledger {
	return &Ledger{byID: map[string]int{}}
}

// Issues returns every issue seen so far, in creation order.
func (l *Ledger) Issues() []model.Issue { return l.issues }

// Absorb folds one round's observations into the ledger and returns the issues
// this round touched, worst-severity ordering left to the caller. Observations are
// tagged in place with the issue they were grouped under.
//
// Matching works in two stages, because within-round and cross-round identity are
// genuinely different problems:
//
//   - An observation whose Issue field names a known issue joins it. Only the
//     reviewer, which can see the history, can recognize its own reworded
//     re-report of a problem whose line has since moved.
//   - Otherwise the fingerprint decides: same file and a nearby line, or -- when
//     no line is given -- same file and the same normalized title.
//
// Note what is NOT part of identity: the category. Two lenses found the same
// racy-ordinal defect in the same file and line under different categories
// (concurrency and tests) with different severities, and keying on category would
// have kept them apart, which is exactly the duplicate that cost a budget slot.
func (l *Ledger) Absorb(round int, observations []model.Finding) []model.Issue {
	touched := map[int]bool{}
	for i := range observations {
		obs := &observations[i]
		idx := l.match(obs)
		if idx < 0 {
			idx = l.create(round, *obs)
		}
		l.attach(idx, round, obs)
		touched[idx] = true
	}
	out := make([]model.Issue, 0, len(touched))
	for _, idx := range sortedKeys(touched) {
		out = append(out, l.forRound(idx, round))
	}
	return out
}

// forRound returns the per-round copy of an issue, with this round's verdict state
// reset so a round never inherits the previous round's verdict, and with its
// observations narrowed to the ones made THIS round.
//
// The ledger keeps every observation for the life of the run, but the round copy
// must not: Issue.Agents() feeds the "reported independently by N agents --
// corroborated" claim in the coder prompt, and under strategy: rotate a lens is
// deliberately reassigned each round, so an issue that survives one round would
// otherwise be presented as corroborated on the strength of one agent seeing it
// per round. It also kept the per-round observation count (len(rec.Findings)) and
// the corroborated count in the same records while counting different sets, which
// could report 3 corroborated issues out of 3 observations. Narrowing here fixes
// both, and stops the accumulated slice from being re-copied into every
// RoundRecord (and so into the summary and the coder prompt) as rounds go by.
//
// A re-report means different things depending on how the issue was closed:
//
//   - REJECTED: the coder judged it not genuine. It stays rejected and is NOT
//     handed back -- re-submitting it would spend a slot every round on something
//     already decided, and the history already tells reviewers not to re-report it.
//     It is still returned, carrying that verdict, so the summary shows it came up
//     again rather than silently dropping a reviewer's work.
//   - FIXED: the coder changed something, yet reviewers still see the problem. That
//     is evidence the fix did not work, so the issue REOPENS. Treating it as closed
//     would let a failed fix end the run as converged.
//   - DEFERRED or open: simply still open.
func (l *Ledger) forRound(idx, round int) model.Issue {
	it := l.issues[idx]
	it.Observations = observationsIn(it.Observations, round)
	if it.Status == model.VerdictRejected {
		it.Verdict = model.VerdictRejected
		it.VerdictDetail = "previously rejected; not re-submitted to the coder (" + it.VerdictDetail + ")"
		return it
	}
	// Reopen: clear the carried verdict so this round's own decision is recorded.
	l.issues[idx].Status = model.StatusOpen
	l.issues[idx].Verdict = ""
	l.issues[idx].VerdictDetail = ""
	it.Status = model.StatusOpen
	it.Verdict = ""
	it.VerdictDetail = ""
	return it
}

// observationsIn returns the observations made in one round, as a fresh slice so
// the round copy never aliases (or appends into) the ledger's own history.
func observationsIn(obs []model.Finding, round int) []model.Finding {
	out := make([]model.Finding, 0, len(obs))
	for _, o := range obs {
		if o.Round == round {
			out = append(out, o)
		}
	}
	return out
}

// match finds an existing issue for an observation, or -1.
func (l *Ledger) match(obs *model.Finding) int {
	// A reviewer-declared reference wins: it is the only signal that survives
	// rewording and code movement.
	if obs.IssueID != "" {
		if idx, ok := l.byID[obs.IssueID]; ok {
			return idx
		}
		// A declared id that does not exist is a model mistake, not a new issue
		// identity -- fall through to the fingerprint rather than trusting it.
	}
	fp := Fingerprint(*obs)
	for idx := range l.issues {
		if l.issues[idx].Fingerprint == fp {
			return idx
		}
	}
	// Same file, agreeing titles: one issue, at any distance. This is what carries a
	// re-report across rounds once a fix has moved the code -- see the note on
	// titlesAgree above for why the line window that used to bound this is gone.
	if obs.File != "" {
		for idx := range l.issues {
			it := l.issues[idx]
			if normalizePath(it.File) != normalizePath(obs.File) {
				continue
			}
			if titlesAgree(it.Title, obs.Title) {
				return idx
			}
		}
	}
	return -1
}

// titlesAgree reports whether two titles describe the same thing, by overlap of
// their distinctive words relative to the shorter one.
//
// The threshold is deliberately above half: "low prio" and "high prio" share
// exactly half their words and are NOT the same issue, while "nil deref on the
// config pointer" and "config pointer can be nil here" share three of four and
// are. This only has to separate neighbors in one file -- an exact line match is
// already handled by the fingerprint, and a genuine cross-round reword is the
// reviewer's job to declare.
func titlesAgree(a, b string) bool {
	ta, tb := titleTokens(a), titleTokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return false
	}
	shared := 0
	for w := range ta {
		if tb[w] {
			shared++
		}
	}
	return float64(shared) > 0.5*float64(min(len(ta), len(tb)))
}

func titleTokens(t string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(t), notAlphanumeric) {
		if !stopwords[w] {
			out[w] = true
		}
	}
	return out
}

func (l *Ledger) create(round int, obs model.Finding) int {
	l.seq++
	iss := model.Issue{
		ID:          fmt.Sprintf("i%d", l.seq),
		Fingerprint: Fingerprint(obs),
		Status:      model.StatusOpen,
		Category:    obs.Category,
		Severity:    obs.Severity,
		File:        obs.File,
		Line:        obs.Line,
		Title:       obs.Title,
		Description: obs.Description,
		Suggestion:  obs.Suggestion,
		Advisory:    obs.Advisory,
		FirstRound:  round,
	}
	l.issues = append(l.issues, iss)
	idx := len(l.issues) - 1
	l.byID[iss.ID] = idx
	return idx
}

// attach records an observation against an issue and lets it raise the issue's
// severity. Severity is the worst any reviewer assigned, not the first or the
// average: it decides scheduling under the per-round cap, and one reviewer
// spotting that a defect is exploitable should not be outvoted by two who did not.
//
// A LATER round's report re-anchors the issue: its location and text describe the
// code as it is now. Without that, a reviewer-declared re-report of a defect that
// moved (the fix shifted it, or an earlier round edited around it) at equal or
// lower severity would leave the canonical file and line pointing at the old
// code -- and FormatIssues prints only that canonical location, so the coder would
// be sent to a line that no longer holds the defect. Severity still keeps the worst
// value ever seen, because that is what drives scheduling.
func (l *Ledger) attach(idx, round int, obs *model.Finding) {
	it := &l.issues[idx]
	obs.IssueID = it.ID
	// Stamp the round on the observation itself: the ledger keeps observations for
	// the whole run, and forRound needs to tell this round's reports from earlier
	// ones to scope the corroboration claim.
	obs.Round = round
	reanchor := round > it.LastRound
	it.Observations = append(it.Observations, *obs)
	it.LastRound = round
	if reanchor {
		// Only the FIRST observation of a new round re-anchors: within one round the
		// worst-severity rule below decides between simultaneous readings, which keeps
		// the result independent of reviewer completion order.
		if obs.File != "" {
			it.File = obs.File
			it.Line = obs.Line
		}
		if obs.Title != "" {
			it.Title = obs.Title
			it.Description = obs.Description
			it.Suggestion = obs.Suggestion
		}
	}
	if model.WorseSeverity(obs.Severity, it.Severity) {
		it.Severity = obs.Severity
		it.Title = obs.Title // keep the title and body from the worst reading
		it.Description = obs.Description
		it.Suggestion = obs.Suggestion
		it.Category = obs.Category
	}
}

// Record applies the coder's verdict to an issue and mirrors it onto every
// observation that reported it, so history and the run summary keep speaking in
// terms reviewers recognize.
func (l *Ledger) Record(id, verdict, detail string) {
	idx, ok := l.byID[id]
	if !ok {
		return
	}
	it := &l.issues[idx]
	it.Verdict = verdict
	it.VerdictDetail = detail
	it.Status = verdict
	if verdict == model.VerdictDeferred {
		it.Deferrals++
	}
}

// Reopen withdraws a verdict already recorded for an issue and returns it to open,
// for the case where the work behind a "fixed" verdict did not survive: the
// verification correction reverted the round's edits, so nothing was committed and
// the issue is still there. Without this the ledger would carry a fixed status no
// commit backs, which the run summary reports and the next round's history repeats.
func (l *Ledger) Reopen(id string) {
	idx, ok := l.byID[id]
	if !ok {
		return
	}
	it := &l.issues[idx]
	it.Status = model.StatusOpen
	it.Verdict = ""
	it.VerdictDetail = ""
}

// Deferrals reports how many rounds an issue has been deferred by the cap. This
// is exact, where the earlier heuristic could only guess from (file, category).
func (l *Ledger) Deferrals(id string) int {
	if idx, ok := l.byID[id]; ok {
		return l.issues[idx].Deferrals
	}
	return 0
}

// Get returns an issue by id.
func (l *Ledger) Get(id string) (model.Issue, bool) {
	if idx, ok := l.byID[id]; ok {
		return l.issues[idx], true
	}
	return model.Issue{}, false
}

// Fingerprint derives a deterministic identity for an observation: the normalized
// path plus either the exact line or, when no line is given, a normalized title.
//
// It is deliberately conservative. Merging two distinct problems is worse than
// leaving a duplicate, because the coder is then told to fix one thing when there
// were two -- whereas a surviving duplicate merely costs a slot, which is the
// status quo this improves on.
func Fingerprint(f model.Finding) string {
	path := normalizePath(f.File)
	if f.Line > 0 {
		return fmt.Sprintf("%s#L%d", path, f.Line)
	}
	return path + "#" + normalizeTitle(f.Title)
}

// normalizePath makes paths comparable across reviewers that spell them
// differently: slashes, a leading ./, and case on the extension.
func normalizePath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "./")
	return strings.Trim(p, "/")
}

// normalizeTitle reduces a title to a comparable key: lowercased, punctuation
// dropped, common filler words removed, remaining words sorted so word order does
// not matter. This only has to catch two reviewers phrasing the SAME sentence
// slightly differently -- recognizing a genuine reword across rounds is the
// reviewer's job via ReviewFinding.Issue, because no lexical rule gets from
// "has two independent declarations" to "duplicated severity vocabulary".
func normalizeTitle(t string) string {
	var words []string
	for _, w := range strings.FieldsFunc(strings.ToLower(t), notAlphanumeric) {
		if !stopwords[w] {
			words = append(words, w)
		}
	}
	sort.Strings(words)
	return strings.Join(words, "-")
}

// notAlphanumeric is the word separator for title tokenizing: anything that is
// not a lowercase letter or digit. Titles arrive with punctuation, backticks, and
// code identifiers, none of which should split a word differently per reviewer.
func notAlphanumeric(r rune) bool {
	return (r < 'a' || r > 'z') && (r < '0' || r > '9')
}

// stopwords are words that carry no identity, so a title differing only in them
// still matches.
var stopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
	"be": true, "but": true, "by": true, "can": true, "for": true, "from": true,
	"in": true, "is": true, "it": true, "its": true, "may": true, "no": true,
	"not": true, "of": true, "on": true, "or": true, "that": true, "the": true,
	"then": true, "this": true, "to": true, "when": true, "which": true,
	"with": true, "would": true,
}

func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
