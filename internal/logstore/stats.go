package logstore

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/model"
)

// AgentStat is one agent's share of everything that has been run here.
type AgentStat struct {
	Name string
	// Runs is how many runs this agent appeared in, which is the denominator for
	// every other number in the row: an agent seated last week and an agent seated
	// since March are not comparable on totals alone.
	Runs  int
	Steps int
	// Errors are invocations that failed or timed out. It is the number that decides
	// a seat as often as recall does -- a reviewer that dies mid-round burns a slot
	// and, in a quorum, can flip the verdict.
	Errors int
	// Issues is distinct issues reported (not raw observations: two lenses reporting
	// one defect is one issue), with the coder's verdict on them.
	Issues   int
	Fixed    int
	Rejected int
	Advisory int
	Usage    model.Usage
	Dur      time.Duration
}

// RejectRate is the share of this agent's issues the coder rejected, as a
// fraction, and false when it reported nothing to have a rate about.
//
// It is the cheapest signal there is for a miscalibrated reviewer, and it is only
// legible ACROSS runs: within one run the denominator is a handful of issues, so
// the rate swings between 0 and 1 on noise. It is not a quality score on its own
// -- a coder can be wrong, and a reviewer that finds hard true things will be
// argued with -- which is why it is reported beside the volume it is a rate of.
func (a AgentStat) RejectRate() (float64, bool) {
	if a.Issues == 0 {
		return 0, false
	}
	return float64(a.Rejected) / float64(a.Issues), true
}

// Stats is the cross-run aggregate: what this logs directory says about the
// agents that have run here.
type Stats struct {
	Root string
	// Runs counted, and the window they span.
	Runs int
	From time.Time
	To   time.Time
	// Terminations counts how runs ended, keyed by model.Term*.
	Terminations map[string]int
	Agents       []AgentStat
	// Unreadable is how many candidate files could not be parsed as a summary. It
	// is reported rather than silently dropped: a run whose summary is truncated is
	// a run whose cost is missing from every number above, and an operator drawing
	// a seat decision from this table has to know the sample is short.
	Unreadable int
	// Replayed is how many runs were skipped because their replies came from a
	// recording rather than from an agent. Their per-step usage and duration are the
	// RECORDED ones, so counting them would bill the same tokens twice -- inflating
	// the per-agent economics this table exists to report, by an amount that grows
	// with how often the pipeline is regression-tested (review run 20260916-085129,
	// finding i3).
	Replayed int
}

// LoadStats aggregates every run summary under root.
//
// It takes a DIRECTORY and no configuration, which is the point. The question it
// answers -- which agent earns its tokens -- spans runs made under different
// configs, and resolving one config to find the others would make the answer
// depend on which config happened to be named. The same reasoning -post-run
// already follows: what a finished run produced is recorded in the run, so
// reading it back needs nothing else.
//
// Summaries are found by CONTENT, not by filename. logs.summary_pattern is
// site-configurable, so a reader that globbed the default name would silently
// report nothing for an operator who changed it -- and reporting nothing looks
// exactly like having run nothing.
func LoadStats(root string) (*Stats, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory; point stats at a logs root (the parent of the per-run directories)", root)
	}
	s := &Stats{Root: root, Terminations: map[string]int{}}
	agents := map[string]*AgentStat{}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// A directory that cannot be read must not abort the walk: the answer from
			// the runs that ARE readable is still worth having, and Unreadable records
			// that the sample is short.
			s.Unreadable++
			return nil //nolint:nilerr // see above
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		sum, ok := readSummary(path)
		if !ok {
			return nil
		}
		if sum.ReplayedFrom != "" {
			s.Replayed++
			return nil
		}
		s.absorb(sum, agents)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, a := range agents {
		s.Agents = append(s.Agents, *a)
	}
	// Busiest first: the rows that decide a panel are the ones with a sample behind
	// them, and an agent tried once should not head the table by sorting on name.
	sort.Slice(s.Agents, func(i, j int) bool {
		if s.Agents[i].Runs != s.Agents[j].Runs {
			return s.Agents[i].Runs > s.Agents[j].Runs
		}
		return s.Agents[i].Name < s.Agents[j].Name
	})
	return s, nil
}

// readSummary parses one candidate file, reporting whether it is a run summary at
// all. The per-step .json artifacts live under the same root and parse happily
// into a RunSummary full of zero values, so the discriminator is a field only a
// real summary carries.
func readSummary(path string) (*model.RunSummary, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var sum model.RunSummary
	if err := json.Unmarshal(raw, &sum); err != nil {
		return nil, false
	}
	if sum.StartedAt.IsZero() || (sum.Termination == "" && len(sum.Rounds) == 0) {
		return nil, false
	}
	return &sum, true
}

// absorb folds one run into the aggregate.
func (s *Stats) absorb(sum *model.RunSummary, agents map[string]*AgentStat) {
	s.Runs++
	s.Terminations[sum.Termination]++
	if s.From.IsZero() || sum.StartedAt.Before(s.From) {
		s.From = sum.StartedAt
	}
	if sum.StartedAt.After(s.To) {
		s.To = sum.StartedAt
	}

	// The per-run table's own grouping, reused rather than reimplemented: issue
	// identity, the last-word verdict rule and the corroboration handling are
	// subtle enough that a second copy here would drift from the scoreboard and
	// give two different answers to one question.
	st := computeRunStats(sum)

	get := func(name string) *AgentStat {
		if name == "" {
			name = "(unknown)"
		}
		a, ok := agents[name]
		if !ok {
			a = &AgentStat{Name: name}
			agents[name] = a
		}
		return a
	}
	seen := map[string]bool{}
	for _, name := range st.agentOrder {
		c := st.agents[name]
		a := get(name)
		if !seen[name] {
			a.Runs++
			seen[name] = true
		}
		a.Errors += c.errors
		a.Advisory += c.advisory
		a.Issues += len(c.issues)
		a.Dur += c.dur
		a.Usage.Add(c.usage)
		for id := range c.issues {
			switch st.finalVerdict[id] {
			case model.VerdictFixed:
				a.Fixed++
			case model.VerdictRejected:
				a.Rejected++
			}
		}
	}
	absorbCoder(sum, st, get, seen)
	// Steps are counted from the raw record rather than from the grouped rows:
	// every invocation costs a slot, including the coder's and including one that
	// produced no finding at all.
	for _, r := range sum.Rounds {
		for _, step := range r.Steps {
			get(step.Agent).Steps++
		}
	}
}

// absorbCoder adds the coder's share of one run.
//
// It is separate because computeRunStats bills the coder apart from every
// reviewer -- its steps never land in a reviewer row -- so without this the most
// expensive seat in the run would be missing from the table that exists to price
// seats.
func absorbCoder(sum *model.RunSummary, st *runStats, get func(string) *AgentStat, seen map[string]bool) {
	if st.coderUsage.Tokens() == 0 && st.coderDur == 0 {
		return
	}
	name := sum.Coder
	if name == "" {
		name = "(coder)"
	}
	a := get(name)
	if !seen[name] {
		a.Runs++
		seen[name] = true
	}
	a.Dur += st.coderDur
	a.Usage.Add(st.coderUsage)
	// The coder's failures too. computeRunStats credits errors from ReviewErrors,
	// which is a reviewer-only list, so a coder that timed out or died mid-fix
	// showed a clean errors column however often it happened -- hiding the most
	// expensive kind of failure there is (review run 20260916-085129, finding i6).
	for _, r := range sum.Rounds {
		for _, step := range r.Steps {
			if step.Role == "fix" && step.Failed {
				a.Errors++
			}
		}
	}
}

// RenderStats renders the cross-run table.
//
// There is deliberately no single "tokens" column. On some routes a cache read is
// reported INSIDE input_tokens and on others beside it, so one summed figure
// silently double-counts for a subset of the panel -- and the whole purpose of
// this table is comparing agents to each other, which is exactly what a column
// that means different things per row destroys. Input, output and cache are
// therefore shown as what each CLI reported, and nothing adds them up.
func RenderStats(s *Stats) string {
	var b strings.Builder
	rule := strings.Repeat("─", 78)
	title := fmt.Sprintf("fixpoint · %d run(s) · %s", s.Runs, agent.EscapeTerminal(s.Root))
	if !s.From.IsZero() {
		title += fmt.Sprintf(" · %s to %s", s.From.Format("2006-01-02"), s.To.Format("2006-01-02"))
	}
	b.WriteString(rule + "\n " + title + "\n" + rule + "\n")

	if s.Runs == 0 {
		b.WriteString(" no runs found here. fixpoint writes one summary per run under logs.dir;\n")
		b.WriteString(" point stats at that directory (the parent of the per-run directories).\n")
		b.WriteString(rule + "\n")
		return b.String()
	}

	head := []string{
		"AGENT", "runs", "steps", colErrors, colIssues, colFixed, colRejected,
		"rej%", colAdvisory, "in", "out", colCache, colCost, colTime,
	}
	rows := [][]string{head}
	for _, a := range s.Agents {
		rate := "—"
		if r, ok := a.RejectRate(); ok {
			rate = fmt.Sprintf("%.0f%%", r*100)
		}
		rows = append(rows, []string{
			// An agent name comes from a config the repository under review may own,
			// and a cell that can emit ESC/CSI would redraw the rows around it.
			agent.EscapeTerminal(a.Name),
			itoa(a.Runs), itoa(a.Steps), itoa(a.Errors), itoa(a.Issues),
			itoa(a.Fixed), itoa(a.Rejected), rate, itoa(a.Advisory),
			tokenCount(a.Usage.InputTokens), tokenCount(a.Usage.OutputTokens),
			tokenCount(a.Usage.CacheReadTokens + a.Usage.CacheWriteTokens),
			costUSD(a.Usage), humanDuration(a.Dur),
		})
	}
	writeAligned(&b, rows, false)

	b.WriteString("\n")
	for _, kv := range statsFacts(s) {
		fmt.Fprintf(&b, " %-12s %s\n", kv[0], kv[1])
	}
	b.WriteString(rule + "\n")
	return b.String()
}

// statsFacts are the lines under the table: how the runs ended, and the caveats a
// reader must have before drawing a seat decision from the rows.
func statsFacts(s *Stats) [][2]string {
	var out [][2]string
	if len(s.Terminations) > 0 {
		var parts []string
		for _, k := range sortedTerminations(s.Terminations) {
			// The label is derived from the key, never substituted for it: renaming it
			// before the lookup would count every unlabeled run as zero.
			// Escaped like the agent cells above: a summary is a file on disk that a
			// repository can ship, and a termination string carrying ESC/CSI would
			// redraw the table it is printed under (review run 20260916-085129,
			// finding i11).
			label := agent.EscapeTerminal(k)
			if label == "" {
				label = "(none)"
			}
			parts = append(parts, fmt.Sprintf("%s %d", label, s.Terminations[k]))
		}
		out = append(out, [2]string{"outcomes", strings.Join(parts, ", ")})
	}
	// Named every time rather than only when it bites: a reader comparing two rows
	// has no way to know from the table which routes fold cache into input.
	out = append(out,
		[2]string{"tokens", "as each CLI reported them; in/out/cache are NOT summed -- some routes count a cache read inside input_tokens and adding them would double-count those rows"},
		[2]string{colCost, "only from CLIs that report one; a subscription-authenticated route has no per-request price and shows none"},
	)
	if s.Replayed > 0 {
		out = append(out, [2]string{"replays", fmt.Sprintf("%d run(s) were replays and are excluded: their tokens and wall clock are the recorded ones, already counted against the run they came from", s.Replayed)})
	}
	if s.Unreadable > 0 {
		out = append(out, [2]string{"incomplete", fmt.Sprintf("%d path(s) under this root could not be read, so some runs may be missing from every number above", s.Unreadable)})
	}
	return out
}

func sortedTerminations(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
