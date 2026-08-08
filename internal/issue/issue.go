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
	"unicode"

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
//
// Nor does an EXACT line get to skip that agreement: two defects sharing a
// statement is the ordinary case, not a corner one, and one issue can carry only
// one verdict.

// Ledger accumulates issues over a run. It is not safe for concurrent use: the
// parallel part of a round is the reviewers, and aggregation happens after their
// barrier.
type Ledger struct {
	issues    []model.Issue
	byID      map[string]int
	seq       int
	conflicts []string
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
//     re-report of a problem whose line has since moved. The one exception is a
//     declaration naming an already REJECTED issue, which would suppress the
//     observation outright -- see declarationHolds.
//   - Otherwise the same file plus agreeing titles decide, at any distance. A
//     shared location is where two reports MAY be about one defect; the titles
//     are what say they are.
//
// Note what is NOT part of identity: the category. Two lenses found the same
// racy-ordinal defect in the same file and line under different categories
// (concurrency and tests) with different severities, and keying on category would
// have kept them apart, which is exactly the duplicate that cost a budget slot.
func (l *Ledger) Absorb(round int, observations []model.Finding) []model.Issue {
	touched := map[int]bool{}
	for i := range observations {
		obs := &observations[i]
		declared := obs.IssueID
		idx := l.match(obs)
		if idx < 0 {
			idx = l.create(round, *obs)
		}
		l.attach(idx, round, obs)
		l.noteRefusedDeclaration(round, declared, idx, obs)
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
	// rewording and code movement. Except when the issue it names is already
	// rejected -- see declarationHolds.
	if obs.IssueID != "" {
		if idx, ok := l.byID[obs.IssueID]; ok && l.declarationHolds(idx, obs) {
			return idx
		}
		// A declared id that does not exist is a model mistake, not a new issue
		// identity -- fall through to the fingerprint rather than trusting it.
	}
	for idx := range l.issues {
		if fingerprintMatch(l.issues[idx], *obs) {
			return idx
		}
	}
	// Same file, agreeing titles: one issue, at any distance. This is what carries a
	// re-report across rounds once a fix has moved the code -- see the note on
	// titlesAgree above for why the line window that used to bound this is gone.
	for idx := range l.issues {
		if fileTitleMatch(l.issues[idx], *obs) {
			return idx
		}
	}
	return -1
}

// fingerprintMatch reports whether an observation shares an issue's fingerprint
// AND, when that fingerprint is a location, agrees with its title.
//
// An exact location is where two reports MAY be about one defect, not proof that
// they are. One statement routinely holds two: the nil deref and the unchecked
// error it came from, the racy read and the test that never asserts it. Both
// reviewers cite that line, and merging them hands the coder ONE title,
// description and suggestion for two problems -- then one verdict closes both, so
// fixing or rejecting the defect the issue happens to describe silently buries the
// other for the rest of the run. That is the failure this package prices as
// strictly worse than a surviving duplicate, so a located observation must clear
// the same title agreement fileTitleMatch demands. A reviewer that can see past
// the wording says so with Issue.
func fingerprintMatch(it model.Issue, obs model.Finding) bool {
	if it.Fingerprint != Fingerprint(obs) {
		return false
	}
	return !locationKeyed(obs) || titlesAgree(it.Title, obs.Title)
}

// fileTitleMatch reports whether an observation names the same file as an issue
// with an agreeing title, at any distance.
func fileTitleMatch(it model.Issue, obs model.Finding) bool {
	if obs.File == "" || normalizePath(it.File) != normalizePath(obs.File) {
		return false
	}
	return titlesAgree(it.Title, obs.Title)
}

// declarationHolds reports whether a reviewer's "this is issue X" may be taken at
// face value.
//
// It may, for every issue this round will still hand to the coder: absorbing a
// re-report is the whole point of the declaration, and only the reviewer can
// recognize a reword whose wording AND file have both moved -- no lexical rule
// reaches that, which is why the declaration is taken on its own here.
//
// A wrong id is not free, and costs more than one cap slot spent on two problems:
// attach re-anchors the issue onto the declaring observation's location and text, so
// the coder is handed THAT reading and the verdict is recorded against it. What it
// cannot do is bury the defect the issue was about. Once the verdict lands, the
// canonical title is no longer that defect's, so an honest re-report of it agrees
// with neither the fingerprint's title nor the file's -- the evidence both match
// paths require -- and is minted as its own issue; and an honest declaration citing
// the now rejected id is refused by the rule below. A redirect therefore costs a
// coder session and restarted aging, not the finding.
//
// It may NOT when the named issue was REJECTED in an earlier round, because that is
// the one state where attaching an observation SUPPRESSES it: forRound carries the
// rejection forward and coderWork then drops the issue, so the reported problem
// never reaches the coder -- this round or any later one, since the issue stays
// rejected. Every review prompt replays the history with issue ids, so a reviewer
// that is prompt-injected, or merely confused about which id it is re-reporting,
// holds the full list of ids that silence a finding. A rejected issue therefore has
// to earn the match on the same evidence an undeclared observation would: the
// fingerprint, or the file with an agreeing title. Anything else becomes a new
// issue -- a slot spent, but seen -- and Absorb records the conflict.
func (l *Ledger) declarationHolds(idx int, obs *model.Finding) bool {
	it := l.issues[idx]
	if it.Status != model.VerdictRejected {
		return true
	}
	return fingerprintMatch(it, *obs) || fileTitleMatch(it, *obs)
}

// noteRefusedDeclaration records that an observation's declared issue id was not
// honored, so a refusal is reported rather than silently rewriting what a reviewer
// claimed. Only a declaration naming a KNOWN issue is a conflict: an unknown id is
// an ordinary model slip, already covered by match's fallback.
func (l *Ledger) noteRefusedDeclaration(round int, declared string, idx int, obs *model.Finding) {
	if declared == "" || declared == l.issues[idx].ID {
		return
	}
	if _, known := l.byID[declared]; !known {
		return
	}
	l.conflicts = append(l.conflicts, fmt.Sprintf(
		"round %d: %s declared issue %s, which is already rejected and does not match the report (%s:%d %q) -- recorded as %s instead",
		round, obs.Agent, declared, obs.File, obs.Line, obs.Title, l.issues[idx].ID))
}

// TakeConflicts returns the declaration conflicts recorded since the last call and
// clears them, so the caller reports each one once.
func (l *Ledger) TakeConflicts() []string {
	out := l.conflicts
	l.conflicts = nil
	return out
}

// titlesAgree reports whether two titles describe the same thing, by overlap of
// their distinctive words -- measured against BOTH titles, not only the shorter.
//
// The bar on the shorter title is deliberately above half: "low prio" and "high
// prio" share exactly half their words and are NOT the same issue, while "nil
// deref on the config pointer" and "config pointer can be nil here" share three of
// four and are. This decides every merge -- neighbors in one file, and two lenses
// naming the same statement -- so a genuine cross-round reword that clears no
// lexical bar at all remains the reviewer's job to declare.
//
// That bar alone is one the SHORTER title sets, though, and a terse enough title
// sets it at nothing: a one-word title is cleared by a SINGLE shared word. Since
// fileTitleMatch asks nothing of the line, "Deadlock" at scheduler.go:40 and
// "Unbounded goroutine growth risks deadlock" at scheduler.go:120 then became one
// issue -- one title, description and suggestion for two defects, and one verdict
// closing both, which is the outcome the file header prices as strictly worse than
// a surviving duplicate. So the overlap must also be a real fraction of the LONGER
// title, and must reach minSharedWords: one distinctive word is a topic, not
// evidence of one defect.
//
// The same words on BOTH sides is the exception, whatever the count. That is one
// title, not an overlap, and refusing it would mint a fresh issue every round for a
// defect whose short title never changes but whose line moves -- the aging-restarts
// failure this file exists to prevent.
//
// Counting runs over the SHORTER title, so one word cannot be matched twice and
// push the overlap past the length it is measured against.
func titlesAgree(a, b string) bool {
	short, long := titleTokens(a), titleTokens(b)
	if len(short) == 0 || len(long) == 0 {
		return false
	}
	if len(long) < len(short) {
		short, long = long, short
	}
	shared := 0
	for w := range short {
		if long[w] || hasInflection(long, w) {
			shared++
		}
	}
	if shared == len(short) && len(short) == len(long) {
		return true
	}
	if shared < minSharedWords {
		return false
	}
	return float64(shared) > shortTitleShare*float64(len(short)) &&
		float64(shared) > longTitleShare*float64(len(long))
}

const (
	// minSharedWords is the fewest distinctive words whose overlap is evidence that
	// two titles name one defect. Two reports of one file sharing a single word
	// share a subject ("deadlock", "config"), which one file is expected to have
	// several defects about.
	minSharedWords = 2
	// shortTitleShare is the share of the shorter title the overlap must exceed:
	// above half, so a title that agrees on exactly half its words does not match.
	shortTitleShare = 0.5
	// longTitleShare is the share of the longer title the overlap must exceed. It is
	// far below half on purpose -- reviewers do write one defect up in four words and
	// in fourteen -- and only rules out the case where the longer title is mostly
	// words the shorter one never mentions.
	longTitleShare = 0.25
)

// hasInflection reports whether a set holds a word that is the same word as w in
// another form.
//
// Reviewers describe one defect in whatever voice the sentence wants -- "racy
// ordinal allocation" from one lens, "ordinals allocated racily" from another, for
// the same read-modify-write. Comparing tokens exactly reads those as sharing no
// vocabulary whatsoever, which would split the very duplicate this package exists
// to merge. Matching on a common prefix instead folds ordinal/ordinals,
// allocated/allocation and deref/dereferenced together.
func hasInflection(set map[string]bool, w string) bool {
	for other := range set {
		if sameStem(w, other) {
			return true
		}
	}
	return false
}

// sameStem reports whether two words are one word in two forms.
//
// Two words that BOTH continue past their shared opening have diverged, and a
// short agreement is then coincidence rather than evidence: severity and several
// share five letters and are unrelated words. Such a pair has to reach
// stemPrefix, which still folds allocated/allocation.
//
// A word that IS the whole shared opening is the other case -- the longer word
// is the shorter one plus a suffix, as in deref/dereferenced and
// config/configuration -- and nothing has diverged, so stemRoot is enough.
func sameStem(a, b string) bool {
	n := commonPrefixLen(a, b)
	if n == len(a) || n == len(b) {
		return n >= stemRoot
	}
	return n >= stemPrefix
}

const (
	// stemRoot is the shortest word that carries enough meaning to be a stem of
	// its own rather than a syllable two words happen to open with.
	stemRoot = 5
	// stemPrefix is the shortest shared opening that is evidence of one word
	// when the two words then disagree.
	stemPrefix = 6
)

// commonPrefixLen counts the leading bytes two words share. Tokens are lowercase
// letters and digits by construction -- notWordRune drops everything else -- so
// for the ASCII titles this stemming rule is tuned on, bytes are characters. A
// multi-byte token can share a partial rune's bytes with another, which UTF-8
// makes possible only between runes that already agree on their opening bytes;
// the stem thresholds below are a heuristic either way, and exact tokens are
// compared before it is consulted.
func commonPrefixLen(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

func titleTokens(t string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(t), notWordRune) {
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
		Origin:      obs.Origin,
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
// The CANONICAL text and location come from the worst reading of the LATEST round,
// chosen independently of that lifetime severity, and adopted together:
//
//   - Latest round, because its location and text describe the code as it is now.
//     Without that, a reviewer-declared re-report of a defect that moved (the fix
//     shifted it, or an earlier round edited around it) at equal or lower severity
//     would leave the canonical file and line pointing at the old code -- and
//     FormatIssues prints only that canonical location, so the coder would be sent
//     to a line that no longer holds the defect.
//   - Worst within that round, so simultaneous readings are decided by how bad the
//     defect is rather than by reviewer completion order.
//   - Independently of the lifetime severity, because comparing against it instead
//     silently drops the round's worst reading whenever an EARLIER round already
//     recorded that severity: the strict comparison then fails and the first, milder
//     observation of the round keeps the headline.
//   - Together, because a location from one observation under the title, body and
//     category of another is a headline no reviewer wrote.
func (l *Ledger) attach(idx, round int, obs *model.Finding) {
	it := &l.issues[idx]
	obs.IssueID = it.ID
	// Stamp the round on the observation itself: the ledger keeps observations for
	// the whole run, and forRound needs to tell this round's reports from earlier
	// ones to scope the corroboration claim.
	obs.Round = round
	firstOfRound := round > it.LastRound
	// The round's worst reading so far, taken BEFORE this observation joins it.
	roundWorst := worstSeverityIn(it.Observations, round)
	it.Observations = append(it.Observations, *obs)
	it.LastRound = round
	if firstOfRound || model.WorseSeverity(obs.Severity, roundWorst) {
		adopt(it, obs)
	}
	if model.WorseSeverity(obs.Severity, it.Severity) {
		it.Severity = obs.Severity
	}
	// A conversation origin survives merging, whichever observation carried it. If
	// the panel independently reports what a comment already asked for, the issue is
	// still one somebody is waiting for an answer to -- and dropping the origin here
	// would silently unanswer that conversation.
	if obs.Origin.Thread != "" {
		switch {
		case it.Origin.Thread == "":
			it.Origin = obs.Origin
		case it.Origin.Thread != obs.Origin.Thread && !hasThread(it.Also, obs.Origin.Thread):
			// A second conversation about the same defect. One issue, one fix -- but two
			// people waiting, and dropping the second leaves a comment marked as
			// commissioned that nothing ever answers.
			it.Also = append(it.Also, obs.Origin)
		}
	}
}

// worstSeverityIn returns the worst severity any observation of one round carries,
// or "" when the round has none -- which SeverityRank sorts last, so any severity
// beats it.
func worstSeverityIn(obs []model.Finding, round int) string {
	worst := ""
	for _, o := range obs {
		if o.Round == round && model.WorseSeverity(o.Severity, worst) {
			worst = o.Severity
		}
	}
	return worst
}

// adopt makes one observation the issue's canonical reading: its location, text
// and category, field by field, keeping whatever the issue already has wherever
// the observation left a field blank. Only the title is validated as required when
// findings are ingested, so an observation that omits the description, the
// suggestion or the file must not erase the detail that explains the defect to the
// coder -- or the location that sends the coder to it.
func adopt(it *model.Issue, obs *model.Finding) {
	if obs.File != "" {
		it.File = obs.File
		it.Line = obs.Line
	}
	if obs.Title != "" {
		it.Title = obs.Title
	}
	if obs.Description != "" {
		it.Description = obs.Description
	}
	if obs.Suggestion != "" {
		it.Suggestion = obs.Suggestion
	}
	if obs.Category != "" {
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
// path plus either the exact line or, when there is no path or no line, a
// normalized title.
//
// It is deliberately conservative. Merging two distinct problems is worse than
// leaving a duplicate, because the coder is then told to fix one thing when there
// were two -- whereas a surviving duplicate merely costs a slot, which is the
// status quo this improves on. Hence a line only carries identity alongside a
// path: two reviewers that both omit the file and both happen to name line 42 are
// not looking at the same code, and keying on the bare line would fold them into
// one issue whose title, description and suggestion come from a single reading,
// hiding the other problem from the coder entirely.
//
// For the same reason a location-keyed fingerprint is necessary but NOT sufficient
// for identity -- two defects can share a statement. Ledger.match pairs it with
// title agreement, so several issues may legitimately carry one fingerprint.
func Fingerprint(f model.Finding) string {
	path := normalizePath(f.File)
	if locationKeyed(f) {
		return fmt.Sprintf("%s#L%d", path, f.Line)
	}
	title := NormalizeTitle(f.Title)
	if title == "" {
		// Nothing distinctive survived normalization -- a title of only filler
		// words or punctuation. Fall back to the raw text so two differently
		// worded findings still get separate identities.
		title = strings.ToLower(strings.TrimSpace(f.Title))
	}
	return path + "#" + title
}

// locationKeyed reports whether an observation names a place precisely enough for
// the fingerprint to be built from it rather than from the title. It is the one
// definition of "this fingerprint means a location", so match knows when an equal
// fingerprint still needs the titles to agree.
func locationKeyed(f model.Finding) bool {
	return normalizePath(f.File) != "" && f.Line > 0
}

// normalizePath makes paths comparable across reviewers that spell them
// differently: slashes, a leading ./, and case on the extension.
func normalizePath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "./")
	return strings.Trim(p, "/")
}

// NormalizeTitle reduces a title to a comparable key: lowercased, punctuation
// dropped, common filler words removed, remaining words sorted so word order does
// not matter. This only has to catch two reviewers phrasing the SAME sentence
// slightly differently -- recognizing a genuine reword across rounds is the
// reviewer's job via ReviewFinding.Issue, because no lexical rule gets from
// "has two independent declarations" to "duplicated severity vocabulary".
//
// Exported because the identity published on a pull request has to mean what this
// package means by "the same defect": a location alone does not, so review.FindingID
// pairs the fingerprint with this key.
func NormalizeTitle(t string) string {
	var words []string
	for _, w := range strings.FieldsFunc(strings.ToLower(t), notWordRune) {
		if !stopwords[w] {
			words = append(words, w)
		}
	}
	sort.Strings(words)
	return strings.Join(words, "-")
}

// notWordRune is the word separator for title tokenizing: anything that is not a
// letter or a digit. Titles arrive with punctuation, backticks, and code
// identifiers, none of which should split a word differently per reviewer.
//
// unicode, not ASCII. An ASCII-only rule treated every other script as
// punctuation, so a title written in Chinese or Cyrillic tokenized to NOTHING and
// two unrelated defects reduced to the same empty key -- which review.FindingID
// hashes into the identity that decides whether a finding is withheld from a pull
// request. Keeping the letters costs nothing for ASCII titles, whose tokens are
// unchanged, and keeps two findings distinct in the scripts most of the world
// reviews in.
func notWordRune(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
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

// hasThread reports whether a conversation is already linked to an issue.
func hasThread(origins []model.Origin, thread string) bool {
	for _, o := range origins {
		if o.Thread == thread {
			return true
		}
	}
	return false
}
