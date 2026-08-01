package logstore

import (
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/model"
)

// twoAgentRun is a run where both reviewers report one shared issue and one of
// their own, the coder fixes two and rejects one, and a third is deferred by the
// cap -- enough shape to exercise every column.
func twoAgentRun() *model.RunSummary {
	obs := func(agent, lens, issueID, title string) model.Finding {
		return model.Finding{Agent: agent, Lens: lens, IssueID: issueID, Title: title, File: "x.go", Line: 1}
	}
	issue := func(id, verdict string, agents ...string) model.Issue {
		it := model.Issue{ID: id, Title: id, Status: verdict, Verdict: verdict}
		for _, a := range agents {
			it.Observations = append(it.Observations, model.Finding{Agent: a})
		}
		return it
	}
	start := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	return &model.RunSummary{
		StartedAt:           start,
		FinishedAt:          start.Add(95 * time.Minute),
		Sources:             model.RunSources{Config: "/p/config/fix-code.yaml", Extends: "/p/config/defaults.yaml"},
		Mode:                "directory",
		Path:                "/p",
		Strategy:            "rotate",
		MaxIterations:       5,
		MaxFindingsPerRound: 8,
		Overrides:           []string{"trusted_target=true"},
		Coder:               "claude-coder",
		Termination:         model.TermMaxIterations,
		Rounds: []model.RoundRecord{{
			Round: 1,
			Findings: []model.Finding{
				obs("codex", "review-bugs", "i1", "shared defect"),
				obs("claude", "review-security", "i1", "shared defect"),
				obs("codex", "review-bugs", "i2", "codex only"),
				obs("claude", "review-security", "i3", "claude only"),
			},
			Advisory:     []model.Finding{obs("claude", "review-maintainability", "", "advisory note")},
			ReviewErrors: []string{"gemma4 via review-tests: timed out after 15m0s"},
			Issues: []model.Issue{
				issue("i1", model.VerdictFixed, "codex", "claude"),
				issue("i2", model.VerdictRejected, "codex"),
				issue("i3", model.VerdictDeferred, "claude"),
			},
			Fixed: 1, Rejected: 1,
			CommitSHA: "abcdef0123456789",
			Verify: []model.VerifyResult{
				{Name: "test", Passed: true},
				{Name: "lint", Passed: false, Optional: true},
			},
			Steps: []model.StepStat{
				{Role: "review", Agent: "codex", Lens: "review-bugs", DurationMS: 60000},
				{Role: "fix", Agent: "claude-coder", DurationMS: 120000},
			},
		}},
	}
}

// The header carries what was run and how it ended, because a scoreboard nobody can
// attribute to a configuration is not evidence of anything.
func TestRunTableReportsTheRunsIdentityAndOutcome(t *testing.T) {
	got := RenderRunTable(twoAgentRun())
	for _, want := range []string{
		"fix-code",                // the config's bare name, as invoked
		"extends defaults",        // inheritance is part of what ran
		"directory · /p",          // target
		"strategy rotate",         //
		"cap 8 issue(s)/round",    // explains the deferred column
		"max 5 round(s)",          //
		"trusted_target=true",     // the assertion that authorized fix rounds
		"test, lint",              // the gate that ran
		"claude-coder",            // who fixed
		"abcdef012345",            // the round commit, abbreviated
		"max-iterations (exit 2)", // termination AND the process exit it maps to
		"1 fixed",                 //
		"1 rejected",              //
	} {
		if !strings.Contains(got, want) {
			t.Errorf("run table is missing %q:\n%s", want, got)
		}
	}
}

// Attribution is by distinct ISSUE, so a defect two reviewers both reported gives
// both of them credit -- and the TOTAL is the distinct count, not the column sum.
// Getting this wrong would make a panel look more productive the more it repeated
// itself, which is the opposite of what the number is for.
func TestRunTableCreditsCorroborationToBothReviewersButCountsItOnce(t *testing.T) {
	got := RenderRunTable(twoAgentRun())
	rows := map[string]string{}
	for _, line := range strings.Split(got, "\n") {
		if f := strings.Fields(line); len(f) > 1 {
			rows[f[0]] = strings.Join(f[1:], " ")
		}
	}
	// codex reported i1 (shared) and i2; claude reported i1 and i3.
	if !strings.HasPrefix(rows["codex"], "2 ") {
		t.Errorf("codex row = %q, want 2 issues (the shared one plus its own)", rows["codex"])
	}
	if !strings.HasPrefix(rows["claude"], "2 ") {
		t.Errorf("claude row = %q, want 2 issues", rows["claude"])
	}
	// Three distinct issues, not the four the rows sum to.
	if !strings.HasPrefix(rows["TOTAL"], "3 1 1 1") {
		t.Errorf("TOTAL row = %q, want 3 distinct issues: 1 fixed, 1 rejected, 1 deferred", rows["TOTAL"])
	}
	if !strings.Contains(got, "reported by more than one reviewer") {
		t.Error("the table must explain why the rows sum above the total")
	}
}

// A reviewer that timed out contributed nothing but still cost a session, and its
// silence is why a clean round may not mean a clean tree. It has to appear.
func TestRunTableCountsReviewerFailures(t *testing.T) {
	got := RenderRunTable(twoAgentRun())
	var line string
	for _, l := range strings.Split(got, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "gemma4") {
			line = l
		}
	}
	if line == "" {
		t.Fatalf("a reviewer that only failed is missing from the table:\n%s", got)
	}
	if f := strings.Fields(line); f[len(f)-1] != "1" && !strings.Contains(line, " 1 ") {
		t.Errorf("gemma4 row = %q, want its error counted", line)
	}
}

// Advisory findings never reach the coder, so they must not inflate the issue
// counts -- but they are not free either, so they get their own column.
func TestRunTableSeparatesAdvisoryFromIssues(t *testing.T) {
	got := RenderRunTable(twoAgentRun())
	for _, l := range strings.Split(got, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "review-maintainability") {
			f := strings.Fields(l)
			if f[1] != "0" {
				t.Errorf("advisory lens row = %q, want 0 issues: advisory findings are not units of work", l)
			}
			return
		}
	}
	t.Errorf("advisory lens missing from the lens table:\n%s", got)
}

// A review-only run has no coder, and saying "0 fixed" would imply one ran and
// achieved nothing.
func TestRunTableSaysTheCoderNeverRanInAReviewOnlyRun(t *testing.T) {
	sum := twoAgentRun()
	sum.ReviewOnly = true
	sum.Termination = model.TermReviewOnly
	got := RenderRunTable(sum)
	if !strings.Contains(got, "not invoked (review-only run)") {
		t.Errorf("review-only run must say the coder never ran:\n%s", got)
	}
	if !strings.Contains(got, "review-only (exit 0)") {
		t.Errorf("review-only must map to exit 0:\n%s", got)
	}
}

// Under no_regressions a check that was already red before the run fails without
// blocking, and the orchestrator commits the round. Counting failures instead of
// blockers would report "passed in 0/N round(s)" for a run whose gate cleared every
// round -- on exactly the already-red repository the policy exists to support.
func TestRunTableCountsRoundsTheGateClearedNotChecksThatFailed(t *testing.T) {
	sum := twoAgentRun()
	// A pre-existing failure: reported as failed, but nothing blocked the round.
	sum.Rounds[0].Verify = []model.VerifyResult{
		{Name: "test", Passed: false},
		{Name: "lint", Passed: true},
	}
	if got := RenderRunTable(sum); !strings.Contains(got, "passed in 1/1 round(s)") {
		t.Errorf("a round nothing blocked must count as passed:\n%s", got)
	}

	sum.Rounds[0].VerifyBlocking = []string{"test"}
	if got := RenderRunTable(sum); !strings.Contains(got, "passed in 0/1 round(s)") {
		t.Errorf("a round the gate blocked must not count as passed:\n%s", got)
	}
}

// The table is printed to a terminal and embedded in the summary, so no row may
// carry invisible trailing padding.
func TestRunTableHasNoTrailingWhitespace(t *testing.T) {
	for i, line := range strings.Split(RenderRunTable(twoAgentRun()), "\n") {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("line %d has trailing whitespace: %q", i+1, line)
		}
	}
}

// A config resolved from <project>/config is a FILENAME the repository under
// review chose, and so are the extends path and the target path. All three are
// printed in the scoreboard the operator reads at the end of a run -- including a
// FAILED one, where what was actually reviewed is the question -- so raw ESC/CSI
// there could erase or redraw the rows around them, and a bidi override could
// reverse a path. The escaping is applied per FIELD rather than to the whole
// table so the column layout survives, which means the layout has to be asserted
// too: escaping that broke the table would be its own misreport.
func TestRunTableEscapesTerminalControlsInRepoSuppliedPaths(t *testing.T) {
	sum := twoAgentRun()
	// CSI erase-line + cursor-up: enough to overwrite the row printed above.
	sum.Sources.Config = "/p/config/ev\x1b[2K\x1b[Ail.yaml"
	// An OSC that retitles the operator's terminal, plus a stray newline that
	// would otherwise split the row in two.
	sum.Sources.Extends = "/p/config/def\x1b]0;pwned\x07au\nlts.yaml"
	// A bidi override, which needs no ESC at all to make a path read backwards.
	sum.Path = "/p/repo\u202egpj.exe"

	got := RenderRunTable(sum)

	for _, raw := range []string{"\x1b", "\x07", "\u202e"} {
		if strings.Contains(got, raw) {
			t.Errorf("run table still contains the raw control %q:\n%q", raw, got)
		}
	}
	for _, want := range []string{
		`ev\x1b[2K\x1b[Ail`,         // the title's bundle name
		`def\x1b]0;pwned\x07au\x0a`, // the extends path, newline included
		`/p/repo\u202egpj.exe`,      // the target path's bidi override
	} {
		if !strings.Contains(got, want) {
			t.Errorf("run table is missing the escaped form %q:\n%s", want, got)
		}
	}
	// The layout still holds: the escaped fields stay one row each, in their
	// column, and the table has as many lines as the benign one.
	if a, b := len(strings.Split(got, "\n")), len(strings.Split(RenderRunTable(twoAgentRun()), "\n")); a != b {
		t.Errorf("the escaped table has %d lines, the benign one %d: a hostile path broke the layout:\n%s", a, b, got)
	}
	for _, label := range []string{" config     ", " target     "} {
		if !strings.Contains(got, label) {
			t.Errorf("the %q column is no longer aligned:\n%s", strings.TrimSpace(label), got)
		}
	}
	for i, line := range strings.Split(got, "\n") {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("line %d has trailing whitespace: %q", i+1, line)
		}
	}
}

// An empty or failed-before-any-round run still gets a table: that is exactly when
// the operator needs to see what did and did not happen.
func TestRunTableHandlesARunWithNoRounds(t *testing.T) {
	sum := &model.RunSummary{
		Sources:     model.RunSources{Config: "/p/config/fix-code.yaml"},
		Mode:        "directory",
		Path:        "/p",
		Termination: model.TermError,
		Error:       "working tree is dirty\nsecond line",
	}
	got := RenderRunTable(sum)
	if !strings.Contains(got, "error (exit 1)") {
		t.Errorf("want the error termination and its exit code:\n%s", got)
	}
	if !strings.Contains(got, "working tree is dirty") {
		t.Errorf("want the failure reason:\n%s", got)
	}
	if strings.Contains(got, "second line") {
		t.Errorf("the outcome line must stay one line:\n%s", got)
	}
}

// Tokens and cost come from the agents' own CLIs. The table must total them for
// the whole run — coder included — and must not invent a cost for a CLI that
// reports none.
func TestRunTableReportsReportedUsageAndOmitsUnreportedCost(t *testing.T) {
	sum := twoAgentRun()
	steps := &sum.Rounds[0].Steps
	*steps = []model.StepStat{
		{Role: "review", Agent: "codex", Lens: "review-bugs", DurationMS: 1000,
			Usage: model.Usage{InputTokens: 12000, OutputTokens: 900, CacheReadTokens: 140000}},
		{Role: "review", Agent: "claude", Lens: "review-security", DurationMS: 1000,
			Usage: model.Usage{InputTokens: 8000, OutputTokens: 600, CostUSD: 0.42, CostKnown: true}},
		{Role: "fix", Agent: "claude-coder", DurationMS: 1000,
			Usage: model.Usage{InputTokens: 30000, OutputTokens: 5000, CostUSD: 1.08, CostKnown: true}},
	}
	got := RenderRunTable(sum)

	rows := map[string]string{}
	for _, line := range strings.Split(got, "\n") {
		if f := strings.Fields(line); len(f) > 1 {
			rows[f[0]] = strings.Join(f[1:], " ")
		}
	}
	// codex reported tokens but no price: a "-" beats a fabricated $0.00, which
	// would silently understate the run's real spend.
	if !strings.Contains(rows["codex"], "153k") {
		t.Errorf("codex row = %q, want its 152,900 reported tokens", rows["codex"])
	}
	if !strings.Contains(rows["codex"], "-") {
		t.Errorf("codex row = %q, want no cost for a CLI that reports none", rows["codex"])
	}
	if !strings.Contains(rows["claude"], "$0.42") {
		t.Errorf("claude row = %q, want its reported cost", rows["claude"])
	}
	// The total covers the coder too, so it exceeds the reviewer rows.
	if !strings.Contains(rows["TOTAL"], "$1.50") {
		t.Errorf("TOTAL row = %q, want $1.50 including the coder", rows["TOTAL"])
	}
	if !strings.Contains(got, "1.08") || !strings.Contains(got, "35k tok") {
		t.Errorf("coder line must report its own usage:\n%s", got)
	}
}

// A run whose agents report nothing must not grow empty token/cost noise into
// something that looks like measured zero.
func TestRunTableShowsDashesWhenNoUsageIsReported(t *testing.T) {
	got := RenderRunTable(twoAgentRun()) // fixture steps carry no Usage
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "TOTAL") {
			if strings.Contains(line, "$0.00") {
				t.Errorf("TOTAL row = %q, want no cost rather than $0.00", line)
			}
			return
		}
	}
	t.Error("no TOTAL row")
}
