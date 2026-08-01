package logstore

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

// RenderRunTable renders the end-of-run scoreboard: what was run, what each
// reviewer contributed, what the coder did with it, and how the run ended.
//
// It answers the question the per-round log cannot, because the log is interleaved
// across parallel reviewers and spread over hours: which lens and which agent
// actually earned their tokens. A panel is only worth its cost if the answer varies
// between agents, and before this there was no way to see that except by reading a
// summary by hand.
//
// Attribution is by ISSUE, not by raw observation. Two agents reporting one defect
// both get credit for that one issue, so the per-agent numbers sum to more than the
// distinct total whenever the panel agreed -- the TOTAL row is the distinct count,
// and the corroboration line below it accounts for the difference.
func RenderRunTable(sum *model.RunSummary) string {
	var b strings.Builder
	st := computeRunStats(sum)

	title := fmt.Sprintf("fixpoint · %s · %s", configName(sum), sum.Termination)
	if n := len(sum.Rounds) - st.finalRounds; n > 0 {
		title += fmt.Sprintf(" after %d round(s)", n)
	}
	// The closing round is named separately: it runs after the outcome is decided,
	// so folding it into the round count would misreport how long convergence took.
	if st.finalRounds > 0 {
		title += " + closing round"
	}
	if d := sum.FinishedAt.Sub(sum.StartedAt); d > 0 {
		title += " · " + humanDuration(d)
	}

	rule := strings.Repeat("─", 78)
	b.WriteString(rule + "\n " + title + "\n" + rule + "\n")

	for _, kv := range runFacts(sum, st) {
		fmt.Fprintf(&b, " %-10s %s\n", kv[0], kv[1])
	}

	if len(st.agents) > 0 {
		b.WriteString("\n")
		writeContributorTable(&b, "REVIEWER", st.agentOrder, st.agents, st.finalVerdict, st)
	}
	// Under `rotate` an agent runs a different lens each round, so the two tables
	// answer different questions: which MODEL is worth its tokens, and which LENS is.
	if len(st.lenses) > 1 {
		b.WriteString("\n")
		writeContributorTable(&b, "LENS", st.lensOrder, st.lenses, st.finalVerdict, nil)
	}

	b.WriteString("\n")
	for _, kv := range runOutcome(sum, st) {
		fmt.Fprintf(&b, " %-10s %s\n", kv[0], kv[1])
	}
	b.WriteString(rule + "\n")
	return b.String()
}

// contributor is one reviewer's or one lens's share of the run, counted in
// distinct issues rather than raw reports.
type contributor struct {
	name     string
	issues   map[string]bool // distinct issue ids this contributor reported
	advisory int             // advisory observations, which never reach the coder
	errors   int             // invocations that failed or timed out
	dur      time.Duration
}

type runStats struct {
	agents     map[string]*contributor
	lenses     map[string]*contributor
	agentOrder []string
	lensOrder  []string

	// finalVerdict is each issue's LAST verdict across the run: an issue deferred
	// three times and then fixed counts once, as fixed.
	finalVerdict map[string]string
	corroborated int
	finalRounds  int
	coderFixed   int
	coderReject  int
	coderDur     time.Duration
	commits      []string
	verifyNames  []string
	verifyPassed int
	verifyRounds int
}

func computeRunStats(sum *model.RunSummary) *runStats {
	st := &runStats{
		agents:       map[string]*contributor{},
		lenses:       map[string]*contributor{},
		finalVerdict: map[string]string{},
	}
	get := func(m map[string]*contributor, order *[]string, name string) *contributor {
		if name == "" {
			name = "(unknown)"
		}
		c, ok := m[name]
		if !ok {
			c = &contributor{name: name, issues: map[string]bool{}}
			m[name] = c
			*order = append(*order, name)
		}
		return c
	}

	corroborated := map[string]bool{}
	for _, r := range sum.Rounds {
		st.absorbIssues(r, corroborated)
		st.absorbAttribution(r, get)
		st.absorbCosts(r, get)
		st.absorbVerify(r)
	}
	st.corroborated = len(corroborated)
	sort.Strings(st.agentOrder)
	sort.Strings(st.lensOrder)
	return st
}

// getFn hands out (creating on first sight) the contributor row for a name.
type getFn func(m map[string]*contributor, order *[]string, name string) *contributor

// absorbIssues records each issue's outcome and whether the panel agreed on it.
func (st *runStats) absorbIssues(r model.RoundRecord, corroborated map[string]bool) {
	for _, it := range r.Issues {
		// Later rounds overwrite: the last word on an issue is its outcome, so an
		// issue deferred three times and then fixed counts once, as fixed.
		if v := it.StatusOrDefault(); v != "" {
			st.finalVerdict[it.ID] = v
		}
		if len(it.Agents()) > 1 {
			corroborated[it.ID] = true
		}
	}
}

// absorbAttribution credits reports, advisory notes and failures to whoever made
// them, by agent and by lens.
func (st *runStats) absorbAttribution(r model.RoundRecord, get getFn) {
	for _, f := range r.Findings {
		id := f.IssueID
		if id == "" {
			id = f.ID // ungrouped: still one unit of work
		}
		get(st.agents, &st.agentOrder, f.Agent).issues[id] = true
		get(st.lenses, &st.lensOrder, config.LensName(f.Lens)).issues[id] = true
	}
	for _, f := range r.Advisory {
		get(st.agents, &st.agentOrder, f.Agent).advisory++
		get(st.lenses, &st.lensOrder, config.LensName(f.Lens)).advisory++
	}
	// A reviewer that failed contributed nothing this round but still cost a
	// session, and its silence is why a clean round may not mean a clean tree.
	for _, e := range r.ReviewErrors {
		if agent, lens, ok := parseReviewError(e); ok {
			get(st.agents, &st.agentOrder, agent).errors++
			get(st.lenses, &st.lensOrder, lens).errors++
		}
	}
}

// absorbCosts accumulates wall-clock per contributor, the coder's tally, and the
// round commits.
func (st *runStats) absorbCosts(r model.RoundRecord, get getFn) {
	for _, s := range r.Steps {
		d := time.Duration(s.DurationMS) * time.Millisecond
		if s.Role == "fix" {
			st.coderDur += d
			continue
		}
		get(st.agents, &st.agentOrder, s.Agent).dur += d
		get(st.lenses, &st.lensOrder, config.LensName(s.Lens)).dur += d
	}
	st.coderFixed += r.Fixed
	st.coderReject += r.Rejected
	if r.Final {
		st.finalRounds++
	}
	if r.CommitSHA != "" {
		st.commits = append(st.commits, shortSHA(r.CommitSHA))
	}
}

// absorbVerify counts the rounds the deterministic gate cleared. An optional check
// failing does not block, so it does not count against the round.
func (st *runStats) absorbVerify(r model.RoundRecord) {
	if len(r.Verify) == 0 {
		return
	}
	st.verifyRounds++
	blocked := false
	for _, v := range r.Verify {
		if !v.Optional && !v.Passed {
			blocked = true
		}
	}
	if !blocked {
		st.verifyPassed++
	}
	if len(st.verifyNames) == 0 {
		for _, v := range r.Verify {
			st.verifyNames = append(st.verifyNames, v.Name)
		}
	}
}

// writeContributorTable renders one attribution table. totals adds the TOTAL row,
// which is the DISTINCT issue count rather than the column sum; pass nil to omit it.
func writeContributorTable(b *strings.Builder, heading string, order []string, m map[string]*contributor, verdicts map[string]string, st *runStats) {
	head := []string{heading, "issues", "fixed", "rejected", "deferred", "advisory", "errors", "time"}
	rows := [][]string{head}
	for _, name := range order {
		c := m[name]
		var fixed, rejected, deferred int
		for id := range c.issues {
			switch verdicts[id] {
			case model.VerdictFixed:
				fixed++
			case model.VerdictRejected:
				rejected++
			case model.VerdictDeferred:
				deferred++
			}
		}
		rows = append(rows, []string{
			name, itoa(len(c.issues)), itoa(fixed), itoa(rejected), itoa(deferred),
			itoa(c.advisory), itoa(c.errors), humanDuration(c.dur),
		})
	}
	if st != nil {
		var fixed, rejected, deferred int
		for _, v := range st.finalVerdict {
			switch v {
			case model.VerdictFixed:
				fixed++
			case model.VerdictRejected:
				rejected++
			case model.VerdictDeferred:
				deferred++
			}
		}
		rows = append(rows, []string{
			"TOTAL", itoa(len(st.finalVerdict)), itoa(fixed), itoa(rejected), itoa(deferred), "", "", "",
		})
	}
	writeAligned(b, rows, st != nil)
	if st != nil && st.corroborated > 0 {
		fmt.Fprintf(b, " %s\n", fmt.Sprintf("rows sum above the total: %d issue(s) were reported by more than one reviewer", st.corroborated))
	}
}

// writeAligned prints rows in aligned columns: the first column left-justified
// (names vary in length), the rest right-justified so digits line up.
func writeAligned(b *strings.Builder, rows [][]string, withTotalRule bool) {
	if len(rows) == 0 {
		return
	}
	width := make([]int, len(rows[0]))
	for _, r := range rows {
		for i, cell := range r {
			if len([]rune(cell)) > width[i] {
				width[i] = len([]rune(cell))
			}
		}
	}
	total := -2
	for _, w := range width {
		total += w + 2
	}
	for i, r := range rows {
		if withTotalRule && i == len(rows)-1 {
			fmt.Fprintf(b, " %s\n", strings.Repeat("╌", total))
		}
		var line strings.Builder
		for j, cell := range r {
			pad := width[j] - len([]rune(cell))
			if j == 0 {
				line.WriteString(cell + strings.Repeat(" ", pad))
			} else {
				line.WriteString(strings.Repeat(" ", pad) + cell)
			}
			if j < len(r)-1 {
				line.WriteString("  ")
			}
		}
		// Trim: an empty trailing cell (the TOTAL row has no advisory/errors/time)
		// would otherwise leave invisible padding on the line.
		b.WriteString(" " + strings.TrimRight(line.String(), " ") + "\n")
	}
}

// runFacts is the "what was run" block: enough to reproduce the run, and the
// assertions that authorized it.
func runFacts(sum *model.RunSummary, st *runStats) [][2]string {
	out := [][2]string{}
	cfg := sum.Sources.Config
	if cfg == "" {
		cfg = sum.ConfigPath
	}
	if sum.Sources.Extends != "" {
		cfg += "  (extends " + configBase(sum.Sources.Extends) + ")"
	}
	out = append(out,
		[2]string{"config", cfg},
		[2]string{"target", strings.TrimSpace(sum.Mode + " · " + sum.Path)},
	)

	mode := "review+fix"
	if sum.ReviewOnly {
		mode = "review only (the coder never runs)"
	}
	strategy := mode + " · strategy " + sum.Strategy
	if sum.MaxFindingsPerRound > 0 {
		strategy += fmt.Sprintf(" · cap %d issue(s)/round", sum.MaxFindingsPerRound)
	}
	if sum.MaxIterations > 0 {
		strategy += fmt.Sprintf(" · max %d round(s)", sum.MaxIterations)
	}
	out = append(out, [2]string{"settings", strategy})

	if len(sum.Overrides) > 0 {
		out = append(out, [2]string{"flags", strings.Join(sum.Overrides, ", ")})
	}
	if st.verifyRounds > 0 {
		out = append(out, [2]string{"verify", fmt.Sprintf("%s · passed in %d/%d round(s)",
			strings.Join(st.verifyNames, ", "), st.verifyPassed, st.verifyRounds)})
	}
	return out
}

// runOutcome is the "what happened" block that closes the table.
func runOutcome(sum *model.RunSummary, st *runStats) [][2]string {
	out := [][2]string{}
	coder := sum.Coder
	if coder == "" {
		coder = "coder"
	}
	if sum.ReviewOnly {
		out = append(out, [2]string{"coder", "not invoked (review-only run)"})
	} else {
		out = append(out, [2]string{"coder", fmt.Sprintf("%s · %d fixed · %d rejected · %s",
			coder, st.coderFixed, st.coderReject, humanDuration(st.coderDur))})
	}
	if len(st.commits) > 0 {
		out = append(out, [2]string{"commits", fmt.Sprintf("%d · %s", len(st.commits), strings.Join(st.commits, " "))})
	}
	exit := fmt.Sprintf("%s (exit %d)", sum.Termination, model.ExitCode(sum.Termination))
	if sum.Error != "" {
		exit += " · " + firstLine(sum.Error)
	}
	out = append(out, [2]string{"exit", exit})
	return out
}

// parseReviewError pulls the agent and lens out of a round's reviewer-error string,
// which the orchestrator formats as "<agent> via <lens>: <reason>".
func parseReviewError(e string) (agent, lens string, ok bool) {
	rest, _, found := strings.Cut(e, ":")
	if !found {
		return "", "", false
	}
	agent, lens, found = strings.Cut(rest, " via ")
	if !found {
		return "", "", false
	}
	return strings.TrimSpace(agent), strings.TrimSpace(lens), true
}

func configName(sum *model.RunSummary) string {
	if n := configBase(sum.Sources.Config); n != "" {
		return n
	}
	return configBase(sum.ConfigPath)
}

// configBase is a config's bare bundle name, which is how the operator invoked it.
func configBase(path string) string {
	if path == "" {
		return ""
	}
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		path = path[i+1:]
	}
	return strings.TrimSuffix(path, ".yaml")
}

// shortSHA abbreviates a commit for display without ever slicing past the end: a
// partial git call can record something shorter than 12 characters, and a summary
// must not panic on it.
func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// humanDuration renders a duration at one useful unit, so a column of them lines up
// and reads at a glance: 3s, 4m12s, 2h3m.
func humanDuration(d time.Duration) string {
	switch {
	case d <= 0:
		return "-"
	case d < time.Minute:
		return d.Round(time.Second).String()
	case d < time.Hour:
		d = d.Round(time.Second)
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		d = d.Round(time.Minute)
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func itoa(n int) string { return strconv.Itoa(n) }
