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

// nearbyLines is how far apart two reports of the same file may be and still be
// treated as one issue. Reviewers point at slightly different lines for the same
// defect -- the declaration, the use, the enclosing function -- and demanding an
// exact match would leave those as separate issues. A few lines is generous enough
// to catch that without merging genuinely unrelated code.
const nearbyLines = 5

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
		out = append(out, l.forRound(idx))
	}
	return out
}

// forRound returns the per-round copy of an issue, with this round's verdict state
// reset so a round never inherits the previous round's verdict.
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
func (l *Ledger) forRound(idx int) model.Issue {
	it := l.issues[idx]
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
	// Line-proximity match, for "the declaration" versus "the use two lines down".
	// Proximity ALONE is not enough: three unrelated defects on consecutive lines
	// of one file are three issues, and merging them would tell the coder to fix
	// one thing when there are three -- strictly worse than leaving a duplicate,
	// which only costs a slot. So a nearby line must also describe the same thing.
	if obs.Line > 0 && obs.File != "" {
		for idx := range l.issues {
			it := l.issues[idx]
			if it.Line == 0 || normalizePath(it.File) != normalizePath(obs.File) {
				continue
			}
			if abs(it.Line-obs.Line) <= nearbyLines && titlesAgree(it.Title, obs.Title) {
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
func (l *Ledger) attach(idx, round int, obs *model.Finding) {
	it := &l.issues[idx]
	obs.IssueID = it.ID
	it.Observations = append(it.Observations, *obs)
	it.LastRound = round
	if worse(obs.Severity, it.Severity) {
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

// severityRank orders severities worst-first; unknown values sort last so a
// reviewer inventing one cannot outrank a real critical.
var severityRank = map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}

func rank(s string) int {
	if r, ok := severityRank[strings.ToLower(strings.TrimSpace(s))]; ok {
		return r
	}
	return len(severityRank)
}

// worse reports whether a is more severe than b.
func worse(a, b string) bool { return rank(a) < rank(b) }

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
