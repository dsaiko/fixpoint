package logstore

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
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

	// The bundle name comes from a filename that may live inside the target, so it
	// is escaped here as it is in the row below and in `--list`.
	title := fmt.Sprintf("fixpoint · %s · %s", agent.EscapeTerminal(configName(sum)), sum.Termination)
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
	usage    model.Usage // what the agent's own CLI reported spending
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
	coderUsage   model.Usage
	// total is every invocation's reported usage, reviewers and coder alike. It
	// is summed independently rather than added up from the rows, so the run
	// total stays right even for a step no row claims.
	total        model.Usage
	commits      []string
	verifyNames  []string
	verifyPassed int
	verifyRuns   int
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
		get(st.lenses, &st.lensOrder, f.Lens).issues[id] = true
	}
	for _, f := range r.Advisory {
		get(st.agents, &st.agentOrder, f.Agent).advisory++
		get(st.lenses, &st.lensOrder, f.Lens).advisory++
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
		st.total.Add(s.Usage)
		if s.Role == "fix" {
			st.coderDur += d
			st.coderUsage.Add(s.Usage)
			continue
		}
		a := get(st.agents, &st.agentOrder, s.Agent)
		a.dur += d
		a.usage.Add(s.Usage)
		l := get(st.lenses, &st.lensOrder, s.Lens)
		l.dur += d
		l.usage.Add(s.Usage)
	}
	st.coderFixed += r.Fixed
	st.coderReject += r.Rejected
	if r.Final {
		st.finalRounds++
	}
	// Every commit the round made, not just the one it ended on: under
	// commit_policy: per_fix a round of eight fixes is eight commits.
	for _, sha := range r.Commits {
		st.commits = append(st.commits, shortSHA(sha))
	}
	if len(r.Commits) == 0 && r.CommitSHA != "" {
		st.commits = append(st.commits, shortSHA(r.CommitSHA))
	}
}

// absorbVerify counts the gate runs the deterministic gate cleared. Runs, not
// rounds: the gate runs once per fix (plus once more for a correction attempt, or
// once on a salvage), so counting per round would report one of a round's N runs and
// silently discard the rest.
//
// What counts as cleared is what BLOCKED under the active policy, not what failed:
// an optional check never blocks, and under no_regressions neither does a check that
// was already red in the pre-run baseline. Counting failures instead would report
// "passed in 0/N" for a run that committed every fix, on exactly the already-red
// repository no_regressions exists to support.
func (st *runStats) absorbVerify(r model.RoundRecord) {
	for _, run := range r.Verify {
		st.verifyRuns++
		if len(run.Blocking) == 0 {
			st.verifyPassed++
		}
		if len(st.verifyNames) == 0 {
			for _, v := range run.Results {
				st.verifyNames = append(st.verifyNames, v.Name)
			}
		}
	}
}

// writeContributorTable renders one attribution table. totals adds the TOTAL row,
// which is the DISTINCT issue count rather than the column sum; pass nil to omit it.
func writeContributorTable(b *strings.Builder, heading string, order []string, m map[string]*contributor, verdicts map[string]string, st *runStats) {
	head := []string{heading, "issues", "fixed", "rejected", "deferred", "advisory", "errors", "tokens", "cache", "cost", "time"}
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
		// Escaped for the same reason the config and target rows are: an agent or
		// lens name comes from a config the repository under review may own, and a
		// cell that can emit ESC/CSI would redraw the rows around it.
		rows = append(rows, []string{
			agent.EscapeTerminal(name), itoa(len(c.issues)), itoa(fixed), itoa(rejected), itoa(deferred),
			itoa(c.advisory), itoa(c.errors), tokenCount(c.usage.Tokens()), cacheRate(c.usage), costUSD(c.usage), humanDuration(c.dur),
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
		// The token and cost totals cover the WHOLE run, coder included, so they
		// are the run's bill rather than the sum of the reviewer rows above.
		rows = append(rows, []string{
			"TOTAL", itoa(len(st.finalVerdict)), itoa(fixed), itoa(rejected), itoa(deferred), "", "",
			tokenCount(st.total.Tokens()), cacheRate(st.total), costUSD(st.total), "",
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
	// Escaped: both are paths, and a config resolved from <project>/config is a
	// FILENAME the repository under review chose, so it can carry ESC/CSI or a bidi
	// override. The table is printed to the operator's terminal at the end of a run
	// (and after a FAILED one, where what was actually reviewed is the question),
	// and a row that can redraw the rows around it misreports exactly that.
	out = append(out,
		[2]string{"config", agent.EscapeTerminal(cfg)},
		[2]string{"target", agent.EscapeTerminal(strings.TrimSpace(sum.Mode + " · " + sum.Path))},
	)

	mode := "review+fix"
	if sum.ReviewOnly {
		mode = "review only (the coder never runs)"
	}
	strategy := mode + " · strategy " + sum.Strategy
	// The cap defaults to unlimited, and "17 deferred" is unreadable without knowing
	// whether a cap deferred them -- so say which of the two applied, rather than
	// leaving its absence to be inferred from a missing field.
	if sum.MaxFindingsPerRound > 0 {
		strategy += fmt.Sprintf(" · cap %d issue(s)/round", sum.MaxFindingsPerRound)
	} else if !sum.ReviewOnly {
		strategy += " · no per-round cap"
	}
	if sum.MaxIterations > 0 {
		strategy += fmt.Sprintf(" · max %d round(s)", sum.MaxIterations)
	}
	out = append(out, [2]string{"settings", strategy})

	if len(sum.Overrides) > 0 {
		out = append(out, [2]string{"flags", strings.Join(sum.Overrides, ", ")})
	}
	if st.verifyRuns > 0 {
		// Escaped for the same reason the rows above are: a check's name is
		// verify.commands[].name from a config the repository under review may own,
		// and a cell that can emit ESC/CSI would redraw the rows printed after it --
		// including the commits row and the exit row.
		names := make([]string, 0, len(st.verifyNames))
		for _, n := range st.verifyNames {
			names = append(names, agent.EscapeTerminal(n))
		}
		// Gate run(s), not round(s): under commit_policy: per_fix one round gates each
		// fix in turn, so "rounds" would name a unit this counter never measured.
		out = append(out, [2]string{"verify", fmt.Sprintf("%s · passed in %d/%d gate run(s)",
			strings.Join(names, ", "), st.verifyPassed, st.verifyRuns)})
	}
	return out
}

// runOutcome is the "what happened" block that closes the table.
func runOutcome(sum *model.RunSummary, st *runStats) [][2]string {
	out := [][2]string{}
	// Escaped like the reviewer rows: the coder's name is roles.coder.agent from a
	// config the repository under review may own, and this row is followed by the
	// commits and exit rows a CSI sequence here could overwrite.
	coder := agent.EscapeTerminal(sum.Coder)
	if coder == "" {
		coder = "coder"
	}
	if sum.ReviewOnly {
		out = append(out, [2]string{"coder", "not invoked (review-only run)"})
	} else {
		line := fmt.Sprintf("%s · %d fixed · %d rejected · %s",
			coder, st.coderFixed, st.coderReject, humanDuration(st.coderDur))
		if n := st.coderUsage.Tokens(); n > 0 {
			// The coder is the run's most cache-dependent agent -- one measured run had
			// it at 100% against reviewers at 0% -- so its rate belongs here for the
			// same reason the reviewer column exists.
			line += " · " + tokenCount(n) + " tok · " + cacheRate(st.coderUsage) + " cached"
		}
		if st.coderUsage.CostKnown {
			line += " · " + costUSD(st.coderUsage)
		}
		out = append(out, [2]string{"coder", line})
	}
	if len(st.commits) > 0 {
		// Under per_fix a long run makes dozens of commits, and listing them all would
		// wrap the row and wreck the table. The count is the number that matters; the
		// SHAs are a convenience for a short run, and `git log` has the rest.
		const maxSHAs = 6
		shown, extra := st.commits, 0
		if len(shown) > maxSHAs {
			shown, extra = shown[:maxSHAs], len(st.commits)-maxSHAs
		}
		line := fmt.Sprintf("%d · %s", len(st.commits), strings.Join(shown, " "))
		if extra > 0 {
			line += fmt.Sprintf(" +%d more", extra)
		}
		// Name the policy beside the count: "3 commits" for 3 fixes and "1 commit"
		// for the same 3 fixes are both right, and only the policy says which.
		if !sum.ReviewOnly && sum.CommitPolicy != "" {
			line += "  (" + sum.CommitPolicy + ")"
		}
		out = append(out, [2]string{"commits", line})
	}
	// Sessions that rejected their issue but edited the tree left a stash each; a
	// stash the table never mentions is one the operator discovers as a preflight
	// surprise two runs later.
	var stashed []string
	for _, r := range sum.Rounds {
		stashed = append(stashed, r.StashedRejects...)
	}
	if len(stashed) > 0 {
		out = append(out, [2]string{"stashed", fmt.Sprintf(
			"%d rejected session(s) left edits: %s · recover with `git stash pop`",
			len(stashed), strings.Join(stashed, " "))})
	}
	exit := fmt.Sprintf("%s (exit %d)", sum.Termination, model.ExitCode(sum.Termination))
	if sum.Error != "" {
		// A run error routinely wraps a subprocess's stderr verbatim -- including,
		// in pr mode, `remote:` lines whose bytes an attacker-controlled server
		// picks. Truncate to the first line first, so escaping cannot hide a
		// newline behind a "\x0a" and spill the rest of the output into the table,
		// then escape what is left for the same reason the config row is escaped.
		exit += " · " + agent.EscapeTerminal(firstLine(sum.Error))
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

// tokenCount abbreviates a token total so a column of them stays scannable:
// 1.2M, 340k, 812. A dash for zero, which here means the CLI reported nothing
// rather than that it did no work.
func tokenCount(n int) string {
	switch {
	case n <= 0:
		return "-"
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.0fk", float64(n)/1e3)
	default:
		return itoa(n)
	}
}

// costUSD renders what a CLI reported spending, and NOTHING when it reports no
// cost at all. Printing $0.00 there would be a claim fixpoint cannot support: a
// subscription-authenticated CLI has a real cost, it just is not per-request, and
// showing zero would quietly understate a run's total.
func costUSD(u model.Usage) string {
	if !u.CostKnown {
		return "-"
	}
	if u.CostUSD > 0 && u.CostUSD < 0.01 {
		return "<$0.01"
	}
	return fmt.Sprintf("$%.2f", u.CostUSD)
}

// cacheRate is the share of an agent's INPUT that came from the prompt cache,
// which is the single number that explains a token bill this project kept
// misreading.
//
// Measured here: two agents on one run spent 41.6M fresh input tokens across nine
// sessions at a 0% cache rate while a third spent 0.4M across five at 99%, because
// an uncached agentic session re-pays for the whole conversation every turn. The
// totals column showed all three as merely "large", so the diagnosis took a day and
// went the wrong way twice -- first blaming prompt size, then buying a cached route
// that fixed the rate and not the bill. A rate in the table would have said which.
//
// Only INPUT counts. Output is never cached, and folding it in would drag every rate
// toward zero by an amount that varies with how talkative an agent is, which is not
// what this column is asking. "-" when an agent reports no input at all: a rate over
// nothing is not 0%, and printing 0% there would accuse a CLI that simply does not
// report usage.
func cacheRate(u model.Usage) string {
	in := u.InputTokens + u.CacheReadTokens
	if in == 0 {
		return "-"
	}
	return fmt.Sprintf("%d%%", int(math.Round(100*float64(u.CacheReadTokens)/float64(in))))
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
