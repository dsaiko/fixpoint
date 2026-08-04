package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/logstore"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
	"github.com/dsaiko/fixpoint/internal/target"
	"github.com/dsaiko/fixpoint/internal/testfixture"
)

// ---- test fixture -----------------------------------------------------------

// fixture wires a temporary git repository, a scripted mock agent, and a
// config around the orchestrator. The mock-agent protocol and shared helpers
// live in internal/testfixture; this fixture only adds the per-suite
// *config.Config wiring and orchestrator plumbing.
type fixture struct {
	t       *testing.T
	repo    string
	respDir string
	cfg     *config.Config
}

func newFixture(t *testing.T, loop config.Loop) *fixture {
	t.Helper()
	repo := testfixture.GitRepo(t)
	respDir := t.TempDir()
	script := testfixture.WriteMockScript(t, respDir)

	promptDir := t.TempDir()
	reviewPrompt := filepath.Join(promptDir, "review.md")
	fixPrompt := filepath.Join(promptDir, "fix.md")
	if err := os.WriteFile(reviewPrompt, []byte("{{.ModeGuidance}}\n{{.Target}}\n{{.History}}\n{{.OutputContract}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixPrompt, []byte("Round {{.Round}}\n{{.Findings}}\n{{.OutputContract}}"), 0o600); err != nil {
		t.Fatal(err)
	}

	if loop.CommitMessage == "" {
		loop.CommitMessage = "fixpoint: round {round} ({fixed} fixed, {rejected} rejected)"
	}
	// The fixture targets a trusted local repo, so fix rounds may run; the
	// untrusted-target gate is exercised explicitly where it matters.
	loop.TrustedTarget = true
	noPing := false
	cfg := &config.Config{
		Target: config.Target{Mode: "directory", Path: repo},
		Roles: config.Roles{
			Coder: config.RoleRef{Agent: "mock", Prompt: fixPrompt},
			Review: config.Review{
				Strategy: "fixed",
				Prompts:  []config.ReviewLens{{Agent: "mock", Prompt: reviewPrompt}},
			},
		},
		Agents: map[string]config.Agent{
			"mock": {Command: []string{script}, PromptVia: "stdin", Timeout: config.Duration(time.Minute), CanEdit: true},
		},
		Loop: loop,
		Logs: config.Logs{
			Dir:             filepath.Join(t.TempDir(), "logs", "{timestamp}", "round-{round}"),
			Formats:         []string{"md", "json", "raw"},
			Pattern:         "{role}-{agent}-{prompt}-round-{round}.{ext}",
			SummaryPattern:  "summary.{ext}",
			TimestampFormat: "20060102-150405",
		},
		PingAgents: &noPing,
	}
	return &fixture{t: t, repo: repo, respDir: respDir, cfg: cfg}
}

func (f *fixture) orchestrator() *Orchestrator {
	f.t.Helper()
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, f.t.Logf)
	if err != nil {
		f.t.Fatal(err)
	}
	return o
}

// capturingOrchestrator builds f's orchestrator with a logf that tees to the test
// log and records what was printed, so a test can assert on the operator-visible
// output. The returned func reports everything logged so far; the mutex guards
// against the agent heartbeat, which logs from its own goroutine.
func (f *fixture) capturingOrchestrator() (*Orchestrator, func() string) {
	f.t.Helper()
	var mu sync.Mutex
	var logs strings.Builder
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}},
		func(format string, a ...any) {
			f.t.Logf(format, a...)
			mu.Lock()
			defer mu.Unlock()
			fmt.Fprintf(&logs, format+"\n", a...)
		})
	if err != nil {
		f.t.Fatal(err)
	}
	return o, func() string {
		mu.Lock()
		defer mu.Unlock()
		return logs.String()
	}
}

// respond registers the mock agent's n-th response.
func (f *fixture) respond(n int, content string) { testfixture.Respond(f.t, f.respDir, n, content) }

// editRepoOn makes the mock agent's n-th invocation edit the repo before
// answering, simulating a coder fix.
func (f *fixture) editRepoOn(n int) { testfixture.EditRepoOn(f.t, f.respDir, f.repo, n) }

// invocations reports how many times the mock agent has been called.
func (f *fixture) invocations() int { return testfixture.Invocations(f.t, f.respDir) }

// promptLogBytes sums the sizes of every prompt log the run wrote for a role,
// which is the exact number of prompt bytes that role sent to its agent.
func (f *fixture) promptLogBytes(role string) int {
	f.t.Helper()
	dir := strings.NewReplacer("{timestamp}", "*", "{round}", "*").Replace(f.cfg.Logs.Dir)
	names, err := filepath.Glob(filepath.Join(dir, role+"-*.prompt"))
	if err != nil {
		f.t.Fatal(err)
	}
	if len(names) == 0 {
		f.t.Fatalf("no %s prompt logs under %s", role, dir)
	}
	total := 0
	for _, name := range names {
		fi, err := os.Stat(name)
		if err != nil {
			f.t.Fatal(err)
		}
		total += int(fi.Size())
	}
	return total
}

func (f *fixture) commitCount() int {
	f.t.Helper()
	out := gitRun(f.t, f.repo, "rev-list", "--count", "HEAD")
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		f.t.Fatal(err)
	}
	return n
}

func reviewResponse(t *testing.T, findings ...model.ReviewFinding) string {
	t.Helper()
	return testfixture.ReviewResponse(t, findings...)
}

func fixResponse(t *testing.T, results ...model.FixResult) string {
	t.Helper()
	return testfixture.FixResponse(t, results...)
}

func aFinding(title string) model.ReviewFinding { return testfixture.AFinding(title) }

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	return testfixture.GitRun(t, dir, args...)
}

// ---- full-loop integration tests ---------------------------------------------

func TestRunConverges(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("off by one")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, reviewResponse(t)) // round 2: clean

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged", sum.Termination)
	}
	if len(sum.Rounds) != 2 {
		t.Fatalf("rounds = %d, want 2", len(sum.Rounds))
	}
	r1 := sum.Rounds[0]
	if r1.Fixed != 1 || r1.Rejected != 0 {
		t.Errorf("round 1: fixed=%d rejected=%d, want 1/0", r1.Fixed, r1.Rejected)
	}
	if len(r1.Findings) != 1 || r1.Findings[0].ID != "r1.1" || r1.Findings[0].Verdict != "fixed" {
		t.Errorf("round 1 findings = %+v", r1.Findings)
	}
	if r1.CommitSHA == "" {
		t.Error("round 1 has no commit SHA")
	}
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + fix round)", got)
	}
	// Default commit_policy is per_fix, so the commit names the issue it fixed.
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "i1") || !strings.Contains(msg, "off by one") {
		t.Errorf("fix commit message = %q, want it to name the issue and its title", msg)
	}
	if got := f.invocations(); got != 3 {
		t.Errorf("agent invocations = %d, want 3", got)
	}
}

// The per-round commit message embeds reviewer-authored finding titles and
// coder-authored verdict details, which routinely quote the secret under
// review. Unlike the owner-only logs, the commit is meant to be pushed, so a
// credential-shaped string in either must be masked before it reaches git.
func TestRunRedactsSecretsInCommitMessage(t *testing.T) {
	const secret = "sk-ant-api03-ABCDEFGHIJKLMNOPqrstuv"
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("hardcoded key "+secret+" in config.go")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{
		ID: "r1.1", Verdict: "fixed", Detail: "removed the hardcoded key " + secret,
	}))
	f.respond(3, reviewResponse(t)) // round 2: clean -> converge

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if strings.Contains(msg, secret) {
		t.Errorf("commit message leaks the secret:\n%s", msg)
	}
	if !strings.Contains(msg, "[REDACTED]") {
		t.Errorf("commit message not redacted (no mask present):\n%s", msg)
	}
}

// Findings from an advisory lens are report-only: they must land in the round's
// Advisory list (not Findings), never reach the coder, and not block
// convergence. A round whose reviewers report only advisory findings therefore
// counts as clean and the run converges without any coder invocation.
func TestRunAdvisoryFindingsRoutedAndExcluded(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.cfg.Roles.Review.Prompts[0].Advisory = true
	f.respond(1, reviewResponse(t, aFinding("advisory note")))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged (an advisory-only round is clean)", sum.Termination)
	}
	if len(sum.Rounds) != 1 {
		t.Fatalf("rounds = %d, want 1 (converge on the first, advisory-only round)", len(sum.Rounds))
	}
	r1 := sum.Rounds[0]
	if len(r1.Advisory) != 1 || r1.Advisory[0].Title != "advisory note" || !r1.Advisory[0].Advisory {
		t.Errorf("advisory routing wrong: Advisory = %+v", r1.Advisory)
	}
	if len(r1.Findings) != 0 {
		t.Errorf("advisory finding leaked into Findings (would reach the coder / block convergence): %+v", r1.Findings)
	}
	// Only the one reviewer ran; the coder was never invoked for advisory work.
	if got := f.invocations(); got != 1 {
		t.Errorf("agent invocations = %d, want 1 (coder must not run for advisory-only findings)", got)
	}
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (nothing committed)", got)
	}
	// No coder prompt was ever written, so no advisory finding could have been
	// handed to the coder.
	if prompts, _ := filepath.Glob(filepath.Join(f.cfg.Logs.StaticBase(), "*", "round-*", "fix-*.prompt")); len(prompts) != 0 {
		t.Errorf("coder prompt(s) written for an advisory-only run: %v", prompts)
	}
}

func TestRunReviewOnly(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, ReviewOnly: true})
	f.respond(1, reviewResponse(t, aFinding("bug")))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermReviewOnly {
		t.Fatalf("termination = %q, want review-only", sum.Termination)
	}
	if len(sum.Rounds) != 1 {
		t.Fatalf("rounds = %d, want 1", len(sum.Rounds))
	}
	if got := f.invocations(); got != 1 {
		t.Errorf("agent invocations = %d, want 1 (coder must never run)", got)
	}
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (nothing committed)", got)
	}
}

func TestRunAllRejected(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("false positive")))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "by design"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermAllRejected {
		t.Fatalf("termination = %q, want all-rejected", sum.Termination)
	}
	r1 := sum.Rounds[0]
	if r1.Rejected != 1 || r1.Fixed != 0 || r1.CommitSHA != "" {
		t.Errorf("round 1 = %+v, want 1 rejected and no commit", r1)
	}
}

// "All rejected" is only a successful terminal state when the round's review
// was complete; with a failed reviewer the round must fail instead.
func TestRunAllRejectedWithReviewerErrorFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	// A second lens pinned to an agent that always fails, so the round has
	// one finding and one reviewer error.
	f.cfg.Agents["bad"] = config.Agent{Command: []string{"false"}, PromptVia: "stdin", Timeout: config.Duration(time.Minute)}
	f.cfg.Roles.Review.Prompts = append(f.cfg.Roles.Review.Prompts,
		config.ReviewLens{Agent: "bad", Prompt: f.cfg.Roles.Review.Prompts[0].Prompt})
	f.respond(1, reviewResponse(t, aFinding("false positive")))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "by design"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "reviewer(s) failed") {
		t.Fatalf("Run() err = %v, want aggregated reviewer failure", err)
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error (not all-rejected)", sum.Termination)
	}
}

// A review-only run whose only reviewer failed must not exit successfully:
// automation would read "review-only" termination as a completed review.
func TestRunReviewOnlyFailsOnReviewerError(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, ReviewOnly: true})
	f.respond(1, "no review block") // reviewer error, zero findings

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "reviewer(s) failed") {
		t.Fatalf("Run() err = %v, want aggregated reviewer failure", err)
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error", sum.Termination)
	}
	if len(sum.Rounds) != 1 || len(sum.Rounds[0].ReviewErrors) != 1 {
		t.Errorf("partial round record must be kept: %+v", sum.Rounds)
	}
}

func TestRunMaxIterations(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug one")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "d"}))
	f.respond(3, reviewResponse(t, aFinding("bug two")))
	f.editRepoOn(4)
	f.respond(4, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "d"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermMaxIterations {
		t.Fatalf("termination = %q, want max-iterations", sum.Termination)
	}
	if len(sum.Rounds) != 2 {
		t.Fatalf("rounds = %d, want 2", len(sum.Rounds))
	}
	if got := f.commitCount(); got != 3 {
		t.Errorf("repo has %d commits, want 3 (initial + 2 fix rounds)", got)
	}
}

func TestRunReviewerErrorResetsCleanStreak(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 5, CleanRoundsToStop: 2})
	f.respond(1, reviewResponse(t)) // clean (streak 1)
	// A contract failure now gets one reformat attempt, so BOTH invocations of
	// round 2 have to fail for the round to count as a reviewer error -- a single
	// malformed reply is salvaged and the round comes back clean.
	f.respond(2, "no review block") // reviewer error: streak resets
	f.respond(3, "still no review block")
	f.respond(4, reviewResponse(t)) // clean (streak 1)
	f.respond(5, reviewResponse(t)) // clean (streak 2 -> converged)

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged", sum.Termination)
	}
	if len(sum.Rounds) != 4 {
		t.Fatalf("rounds = %d, want 4 (error round must reset the streak)", len(sum.Rounds))
	}
	if len(sum.Rounds[1].ReviewErrors) != 1 {
		t.Errorf("round 2 review errors = %v, want 1", sum.Rounds[1].ReviewErrors)
	}
}

// A reply that broke the FORMAT rather than the review is worth one follow-up
// asking for the block again, because the alternative discards a whole agentic
// session -- tens of turns and millions of tokens -- and counts a reviewer error
// that resets the convergence streak. Three reviews in this project's history died
// that way: a missing block, a bare array where the schema wants an object, a
// string where a line number belongs.
func TestRunSalvagesAReviewerThatBrokeOnlyTheContract(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	// The review happened; only its shape is wrong.
	malformed := "I found an off-by-one in main.go line 1. <review>{not json at all}</review>"
	restated := reviewResponse(t, aFinding("off by one"))
	f.respond(1, malformed)
	f.respond(2, restated)
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v; the reformat should have recovered the round", err)
	}
	r := sum.Rounds[0]
	if len(r.ReviewErrors) != 0 {
		t.Errorf("review errors = %v, want none: a salvaged contract failure is not a failed reviewer", r.ReviewErrors)
	}
	if len(r.Findings) != 1 || r.Findings[0].Title != "off by one" {
		t.Fatalf("findings = %+v, want the one the reformat restated", r.Findings)
	}
	if r.Fixed != 1 {
		t.Errorf("fixed = %d, want 1: the recovered finding must reach the coder", r.Fixed)
	}
	// One step for the lens, not two: the salvage is part of the same review, and a
	// second row would double-count the lens in the scoreboard. (That the step also
	// SUMS both attempts' usage is not asserted here -- the fixture's mock agent
	// declares no usage block, so every Usage in these tests is zero. It is covered
	// by TestParseUsage* in the agent package and by reading reformatReview.)
	var reviewSteps int
	var step model.StepStat
	for _, st := range r.Steps {
		if st.Role == "review" {
			reviewSteps++
			step = st
		}
	}
	if reviewSteps != 1 {
		t.Errorf("review steps = %d, want 1 step covering both attempts", reviewSteps)
	}
	// That one step is billed for BOTH invocations. Reporting only the reformat's
	// figures would let the scoreboard's time and I/O columns understate a salvaged
	// review by an entire agent session -- the expensive half.
	if want := len(malformed) + len(restated); step.OutputBytes != want {
		t.Errorf("step output bytes = %d, want %d (both replies)", step.OutputBytes, want)
	}
	if want := f.promptLogBytes("review"); step.PromptBytes != want {
		t.Errorf("step prompt bytes = %d, want %d (review prompt + reformat prompt)", step.PromptBytes, want)
	}
	// The step writes one .raw, and it is the fidelity record -- so for a salvaged
	// step it has to hold BOTH replies. An operator asking why a review was
	// salvaged is asking about the reply that failed the contract; keeping only the
	// reformat's output would drop the one thing they came to read.
	raw := f.stepRaw("review")
	if !strings.Contains(raw, malformed) {
		t.Errorf("review .raw omits the first attempt's output, the reply whose failure caused the salvage:\n%s", raw)
	}
	if !strings.Contains(raw, restated) {
		t.Errorf("review .raw omits the reformat's output:\n%s", raw)
	}
}

// stepRaw reads the single .raw step log a role wrote: the fidelity record, the
// verbatim record of what an agent printed.
func (f *fixture) stepRaw(role string) string {
	f.t.Helper()
	glob := filepath.Join(f.cfg.Logs.StaticBase(), "*", "round-*", role+"-*.raw")
	matches, err := filepath.Glob(glob)
	if err != nil {
		f.t.Fatal(err)
	}
	if len(matches) != 1 {
		f.t.Fatalf("glob %s matched %d files, want 1", glob, len(matches))
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		f.t.Fatal(err)
	}
	return string(b)
}

// When the reformat fails too, the FIRST error is what the operator sees: it says
// what the agent actually did wrong, and a second failure of the same kind adds
// nothing to read.
func TestRunReportsTheOriginalContractErrorWhenReformatAlsoFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
	f.respond(1, "no review block at all")
	f.respond(2, "<review>[]</review>") // valid JSON, wrong shape: array, not object

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() err = nil, want the reviewer failure to surface after a failed salvage")
	}
	if len(sum.Rounds) != 1 || len(sum.Rounds[0].ReviewErrors) != 1 {
		t.Fatalf("want exactly one reviewer error recorded, got %+v", sum.Rounds)
	}
	got := sum.Rounds[0].ReviewErrors[0]
	if !strings.Contains(got, "no <review> block") {
		t.Errorf("review error = %q, want the FIRST attempt's diagnosis (the missing block), not the second's", got)
	}
}

func TestRunCoderContractViolationFailsRound(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// Coder answers about a finding that does not exist.
	f.respond(2, fixResponse(t, model.FixResult{ID: "r9.9", Verdict: "fixed", Detail: "d"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "unknown issue id") {
		t.Fatalf("Run() err = %v, want unknown issue id error", err)
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error", sum.Termination)
	}
}

func TestRunRefusesDirtyTree(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	if err := os.WriteFile(filepath.Join(f.repo, "dirty.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("Run() err = %v, want dirty-tree refusal", err)
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (must refuse before spending tokens)", got)
	}
}

// The pre-Prepare clean check protects the checkout, but Prepare then switches
// branches (gh pr checkout) which can leave the new branch dirty (checkout
// hooks, previously-ignored files surfacing). Those pre-existing changes would
// be attributed to the coder and swept into a round commit, so a dirty tree
// AFTER Prepare must abort the fix run before any reviewer or coder runs.
func TestRunRefusesDirtyTreeAfterPrepare(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.cfg.Target.Mode = "pr"
	f.cfg.Target.PR = 7
	f.cfg.Loop.AllowUntrustedFix = true // pass the untrusted-PR fix gate

	// A feature branch to check out, plus the base commit oid gh will report.
	gitRun(t, f.repo, "branch", "-M", "main")
	mainSHA := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "HEAD"))
	gitRun(t, f.repo, "checkout", "-q", "-b", "feature")
	if err := os.WriteFile(filepath.Join(f.repo, "main.go"), []byte("package main\n\nfunc pr() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, f.repo, "commit", "-aqm", "pr change")
	gitRun(t, f.repo, "checkout", "-q", "main")

	// Stub gh: checkout switches to the PR branch AND leaves an untracked file
	// behind (standing in for a checkout hook or a pre-existing artifact); view
	// reports the base oid, already reachable so Prepare need not fetch.
	binDir := t.TempDir()
	stub := "#!/bin/sh\n" +
		`case "$1 $2" in` + "\n" +
		`"pr checkout") git checkout -q feature; echo dirt > "` + filepath.Join(f.repo, "leftover.txt") + `" ;;` + "\n" +
		`"pr view") echo ` + mainSHA + " ;;\n" +
		`*) echo "unexpected gh call: $@" >&2; exit 1 ;;` + "\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "dirty after preparing") {
		t.Fatalf("Run() err = %v, want post-prepare dirty-tree error", err)
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (must refuse before running any agent)", got)
	}
}

// logs.dir == target.path would make the git exclusion ".", which excludes
// the whole repository from clean checks and commits.
func TestNewRejectsLogsDirAtTargetRoot(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.cfg.Logs.Dir = f.cfg.Target.Path
	if _, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, t.Logf); err == nil || !strings.Contains(err.Error(), "logs.dir") {
		t.Fatalf("New() = %v, want logs.dir rejection", err)
	}
}

// Fix rounds stage and commit the whole repository index, so target.path must
// be the repo root: a subdirectory target would sweep in unrelated staged
// changes and would miss them from its clean check. The refusal must come
// before any agent runs.
func TestRunRefusesSubdirectoryOfRepo(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	sub := filepath.Join(f.repo, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	f.cfg.Target.Path = sub
	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "subdirectory of its git repository") {
		t.Fatalf("Run() err = %v, want subdirectory-of-repo refusal", err)
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (must refuse before running anything)", got)
	}
}

// When the logs dir lives inside the target repo, the orchestrator must thread
// it as a git exclude so round commits and clean checks never touch the run's
// own logs.
func TestRunExcludesInRepoLogsDir(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.cfg.Logs.Dir = filepath.Join(f.repo, "logs") // logs under the target root
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, reviewResponse(t)) // clean round -> converge

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged", sum.Termination)
	}
	// Logs were actually written under the repo, so the exclusion is exercised.
	if entries, _ := os.ReadDir(f.cfg.Logs.Dir); len(entries) == 0 {
		t.Fatal("expected logs written inside the repo")
	}
	// The fix round committed the coder's edit but not the logs dir.
	shown := gitRun(t, f.repo, "show", "--name-only", "--format=", "HEAD")
	if !strings.Contains(shown, "main.go") {
		t.Errorf("round commit missing the coder's edit:\n%s", shown)
	}
	if strings.Contains(shown, "logs/") {
		t.Errorf("round commit swept in the in-repo logs dir:\n%s", shown)
	}
	// The post-run tree is clean once the logs are excluded.
	if status := gitRun(t, f.repo, "status", "--porcelain", "--", ".", ":(exclude)logs"); strings.TrimSpace(status) != "" {
		t.Errorf("tree not clean (minus logs) after run: %q", status)
	}
}

// checkLogsNotSymlinked (r4.3) rejects an in-target logs path any component of
// which is a symlink, so a prepared target (e.g. a PR checkout) cannot redirect
// artifacts past the lexical logs exclusion and sweep them into a round commit.
// Drive the rejection through Run so the post-Prepare call site is covered, and
// exercise the helper's positive and no-op branches directly.
//
// The refused run must also write NOTHING through the link: Run's closing journal
// and summary writes resolve the symlink at syscall time, so leaving them
// unconditional would create the run directory and its artifacts at the
// checkout-chosen destination -- exactly what the check refuses the run over.
func TestRunRejectsSymlinkedLogsDir(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.cfg.Logs.Dir = filepath.Join(f.repo, "logs") // logs under the target root
	// Redirect the logs dir out of the repo via a symlink; the lexical exclusion
	// ("logs") still hides it from the clean check, so only the symlink guard can
	// catch the redirection.
	dest := t.TempDir()
	if err := os.Symlink(dest, filepath.Join(f.repo, "logs")); err != nil {
		t.Fatal(err)
	}
	o, logged := f.capturingOrchestrator()
	_, err := o.Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Run() = %v, want symlink rejection", err)
	}
	if !strings.Contains(err.Error(), filepath.Join(f.repo, "logs")) {
		t.Errorf("error should name the offending path: %v", err)
	}
	assertRefusalLogged(t, logged(), filepath.Join(f.repo, "logs"))
	entries, rerr := os.ReadDir(dest)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("refused run wrote through the symlink: %v", names)
	}
}

// The symlink suppression must survive a CANCELED context. recordRunError turns
// a refusal raised under cancellation into a clean interruption (nil error), so
// Run recognizes errLogsRedirected BEFORE calling it -- otherwise the marker and
// its text are gone by the time Run decides whether to write, and the closing
// journal and summary resume writing through the very link the check refused the
// run over. Cover it with a review-only directory run so nothing but the symlink
// guard can stop the run: memoize a successful preflight, then cancel, so the
// canceled context reaches the refusal instead of aborting an earlier gate.
func TestRunRejectsSymlinkedLogsDirWhenCanceled(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.cfg.Loop.ReviewOnly = true
	f.cfg.Logs.Dir = filepath.Join(f.repo, "logs") // logs under the target root
	o, logged := f.capturingOrchestrator()
	// Memoize the preflight while the context is still live; a canceled one would
	// fail these git-backed gates first and never reach the symlink check.
	if err := o.PreflightGuards(t.Context()); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := os.Symlink(dest, filepath.Join(f.repo, "logs")); err != nil {
		t.Fatal(err)
	}
	before := dirNames(t, f.repo)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	sum, err := o.Run(ctx)
	// The cancellation softens the refusal away -- that is the precondition this
	// test exists for, not an accident.
	if err != nil {
		t.Fatalf("Run() = %v, want nil (cancellation reported as a clean interruption)", err)
	}
	if sum.Termination != model.TermInterrupted {
		t.Fatalf("termination = %q, want interrupted", sum.Termination)
	}
	if got := dirNames(t, dest); len(got) != 0 {
		t.Errorf("canceled refused run wrote through the symlink: %v", got)
	}
	// Nor beside it: the store sidesteps the existing "logs" name and would drop
	// the summary into a fresh logs-N directory in the target instead.
	if got := dirNames(t, f.repo); !slices.Equal(got, before) {
		t.Errorf("canceled refused run wrote into the target: before %v, after %v", before, got)
	}
	// Suppressing the artifacts is only half of the check: the refusal has to be
	// reported somewhere, and here the log is the ONLY place left. The
	// cancellation softened the error to nil, so an operator who is not told why
	// sees nothing but an interrupted run that happened to write nothing.
	assertRefusalLogged(t, logged(), filepath.Join(f.repo, "logs"))
}

// The suppression must also hold when the run ended BEFORE run() got as far as
// its own symlink check: that call site sits after Prepare and only runs when
// Prepare returned nil, yet the checkout Prepare performs is exactly what can
// plant the link. Any earlier or concurrent failure (here a dirty tree, in pr
// mode a `gh pr view` failure or a Ctrl-C after the checkout) then leaves an
// error that is not errLogsRedirected, and Run's closing journal and summary
// resolve the link at syscall time -- writing the artifacts to the destination
// the checkout chose.
func TestRunRejectsSymlinkedLogsDirWhenRunFailsBeforeCheck(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.cfg.Logs.Dir = filepath.Join(f.repo, "logs") // logs under the target root
	dest := t.TempDir()
	if err := os.Symlink(dest, filepath.Join(f.repo, "logs")); err != nil {
		t.Fatal(err)
	}
	// Fail the run at the pre-Prepare clean-tree gate, so run() returns an error
	// that has nothing to do with the symlink and never reaches its own check.
	if err := os.WriteFile(filepath.Join(f.repo, "dirty.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	o, logged := f.capturingOrchestrator()
	before := dirNames(t, f.repo)

	_, err := o.Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("Run() = %v, want the dirty-tree failure", err)
	}
	if got := dirNames(t, dest); len(got) != 0 {
		t.Errorf("failed run wrote through the symlink: %v", got)
	}
	if got := dirNames(t, f.repo); !slices.Equal(got, before) {
		t.Errorf("failed run wrote into the target: before %v, after %v", before, got)
	}
	assertRefusalLogged(t, logged(), filepath.Join(f.repo, "logs"))
}

// A redirection planted AFTER run()'s own check -- the loop itself succeeded --
// must still fail the run. Run's closing check suppresses the journal and summary,
// so a nil error would exit 0 and print a successful outcome for a run whose
// durable artifacts do not exist (and whose round artifacts went to the planted
// destination). The loop's own verdict survives as LoopTermination.
func TestRunReportsLogsRedirectedAfterSuccessfulLoop(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.cfg.Loop.ReviewOnly = true // nothing but the symlink guard can fail this run
	f.cfg.Logs.Dir = filepath.Join(f.repo, "logs", "{timestamp}")
	dest, moved := t.TempDir(), t.TempDir()
	// The reviewer plants the link mid-round, once run()'s post-Prepare check has
	// already passed: move the claimed run directory out of the target (outside it,
	// so it cannot dirty the tree) and leave a symlink in its place.
	testfixture.WriteSide(t, f.respDir, 1, fmt.Sprintf("#!/bin/sh\nmv '%s' '%s'\nln -s '%s' '%s'\n",
		filepath.Join(f.repo, "logs"), filepath.Join(moved, "logs"), dest, filepath.Join(f.repo, "logs")))
	f.respond(1, reviewResponse(t)) // clean round: the loop itself succeeds
	o, logged := f.capturingOrchestrator()

	sum, err := o.Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Run() = %v, want the symlink refusal reported as the run error", err)
	}
	if sum.Termination != model.TermError || sum.Error == "" {
		t.Errorf("termination = %q, error = %q, want error and a recorded message", sum.Termination, sum.Error)
	}
	if sum.LoopTermination != model.TermReviewOnly {
		t.Errorf("loop termination = %q, want the loop's own verdict kept", sum.LoopTermination)
	}
	assertRefusalLogged(t, logged(), filepath.Join(f.repo, "logs"))
	// Neither closing write may reach the redirected destination.
	if err := filepath.WalkDir(dest, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasPrefix(d.Name(), "summary.") {
			t.Errorf("refused run wrote the summary through the symlink: %s", path)
		}
		if d.Name() == logstore.JournalName {
			b, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			if strings.Contains(string(b), model.EvRunFinished) {
				t.Errorf("refused run journaled the outcome through the symlink: %s", path)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// assertRefusalLogged checks that Run reported a suppressed-artifacts refusal to
// the operator, naming path. Matching the path pins the surviving line to the
// pre-recordRunError refusal text rather than a generic notice.
func assertRefusalLogged(t *testing.T, logs, path string) {
	t.Helper()
	for _, want := range []string{"no journal or summary written", path} {
		if !strings.Contains(logs, want) {
			t.Errorf("refusal log missing %q, got logs:\n%s", want, logs)
		}
	}
}

// dirNames lists the entry names in dir, sorted (os.ReadDir already sorts).
func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// logsDirWithin decides whether the run's own artifacts are kept out of round
// commits, collected material, and clean checks, so it has to recognize the logs
// dir as in-target even when the two paths spell the same directory differently.
func TestLogsDirWithin(t *testing.T) {
	// A symlinked target.path (/tmp -> /private/tmp) against the canonical logs
	// path a relative logs.dir gets anchored to: one physical directory, and the
	// exclusion must survive.
	t.Run("symlinked target path", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "target")
		if err := os.MkdirAll(target, 0o700); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(base, "link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		// The logs dir itself does not exist yet, as on a real first round.
		rel, err := logsDirWithin(filepath.Join(target, ".fixpoint"), link)
		if err != nil {
			t.Fatalf("logsDirWithin() error = %v", err)
		}
		if rel != ".fixpoint" {
			t.Errorf("logsDirWithin() = %q, want %q", rel, ".fixpoint")
		}
	})
	// A logs dir that is lexically inside the target but is itself a symlink must
	// still claim the exclusion: that is what lets checkLogsNotSymlinked see the
	// path and refuse the run instead of writing artifacts through the link.
	t.Run("lexically inside via a symlink still claims the exclusion", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Symlink(t.TempDir(), filepath.Join(root, ".fixpoint")); err != nil {
			t.Fatal(err)
		}
		rel, err := logsDirWithin(filepath.Join(root, ".fixpoint"), root)
		if err != nil {
			t.Fatalf("logsDirWithin() error = %v", err)
		}
		if rel != ".fixpoint" {
			t.Errorf("logsDirWithin() = %q, want %q", rel, ".fixpoint")
		}
	})
	// Genuinely separate trees stay out: logs outside the target is a supported
	// configuration and must not gain a bogus exclusion.
	t.Run("outside the target", func(t *testing.T) {
		rel, err := logsDirWithin(t.TempDir(), t.TempDir())
		if err != nil {
			t.Fatalf("logsDirWithin() error = %v", err)
		}
		if rel != "" {
			t.Errorf("logsDirWithin() = %q, want %q for a logs dir outside the target", rel, "")
		}
	})
}

func TestCheckLogsNotSymlinked(t *testing.T) {
	// A real logs directory inside the target is accepted.
	t.Run("real directory accepted", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1})
		f.cfg.Logs.Dir = filepath.Join(f.repo, "logs")
		if err := os.MkdirAll(f.cfg.Logs.Dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := f.orchestrator().checkLogsNotSymlinked(); err != nil {
			t.Errorf("checkLogsNotSymlinked() = %v, want nil for a real directory", err)
		}
	})
	// A not-yet-created logs path is a no-op (ensureDir will make it a real dir).
	t.Run("uncreated path is a no-op", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1})
		f.cfg.Logs.Dir = filepath.Join(f.repo, "logs")
		if err := f.orchestrator().checkLogsNotSymlinked(); err != nil {
			t.Errorf("checkLogsNotSymlinked() = %v, want nil for an uncreated path", err)
		}
	})
	// Logs outside the target leave o.gitExclude unset, so the check is a no-op.
	t.Run("logs outside target is a no-op", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1}) // default: logs in a separate tempdir
		o := f.orchestrator()
		if len(o.gitExclude) != 0 {
			t.Fatalf("precondition: gitExclude should be empty, got %v", o.gitExclude)
		}
		if err := o.checkLogsNotSymlinked(); err != nil {
			t.Errorf("checkLogsNotSymlinked() = %v, want nil when logs live outside the target", err)
		}
	})
	// A symlinked INTERMEDIATE component (not just the leaf) is rejected too.
	t.Run("intermediate component symlink rejected", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1})
		f.cfg.Logs.Dir = filepath.Join(f.repo, "mid", "logs")
		// Point the "mid" component at a directory outside the repo.
		if err := os.Symlink(t.TempDir(), filepath.Join(f.repo, "mid")); err != nil {
			t.Fatal(err)
		}
		err := f.orchestrator().checkLogsNotSymlinked()
		if err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("checkLogsNotSymlinked() = %v, want symlink rejection", err)
		}
		if !strings.Contains(err.Error(), filepath.Join(f.repo, "mid")) {
			t.Errorf("error should name the symlinked component: %v", err)
		}
	})
}

// A fix round where the coder fails AFTER editing files must not strand the
// work or abort the run: the partial edits are committed as a labeled partial
// round and the loop continues -- the next round re-reviews everything.
func TestRunSalvagesPartialFixWork(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.editRepoOn(2)
	f.respond(2, "I changed files but forgot the <fix> envelope.") // fails parsing
	f.respond(3, reviewResponse(t))                                // round 2: clean

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want salvage + continue", err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged after salvaged round", sum.Termination)
	}
	if sum.Rounds[0].CoderError == "" {
		t.Error("round 1 CoderError not recorded")
	}
	if sum.Rounds[0].CommitSHA == "" {
		t.Error("round 1 salvage commit SHA not recorded")
	}
	msg := gitRun(t, f.repo, "log", "--format=%s%n%b", "-1", sum.Rounds[0].CommitSHA)
	if !strings.Contains(msg, "(partial, coder failed)") || !strings.Contains(msg, "bug") {
		t.Errorf("salvage commit message = %q", msg)
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after salvaged fix: %q", status)
	}
}

// The salvage path must obey the same verification gate as a normal round.
// Otherwise a coder that dies mid-edit -- the case MOST likely to leave a tree
// that does not build -- gets its work committed unverified, later rounds build on
// that base, and the run can end as "converged" on a broken tree. That is exactly
// the outcome the verify gate exists to prevent, so the recovery path must not be a
// way around it.
func TestRunDoesNotCommitUnverifiedPartialWork(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.verifyGate(config.VerifyMustPass, "broken")
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder edits, breaks the build, then fails to report -- the salvage case.
	f.breakBuildOn(2, "broken")
	f.respond(2, "I changed files but forgot the <fix> envelope.")

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatalf("Run() must fail: partial work that does not verify cannot be committed (termination %q)", sum.Termination)
	}
	if !strings.Contains(err.Error(), "does not pass verification") {
		t.Errorf("error should say verification rejected the partial work, got: %v", err)
	}
	if len(sum.Rounds) > 0 && sum.Rounds[0].CommitSHA != "" {
		t.Error("unverified partial work was committed; later rounds would build on a broken base")
	}
	// No round commit at all: HEAD must still be the pre-run commit.
	if subjects := gitRun(t, f.repo, "log", "--format=%s"); strings.Contains(subjects, "partial") {
		t.Errorf("a partial round was committed despite failing verification:\n%s", subjects)
	}
	// The work is preserved, not silently dropped: it may hold the useful part of
	// a fix an operator wants to finish by hand.
	if stashes := gitRun(t, f.repo, "stash", "list"); !strings.Contains(stashes, "unverified partial work") {
		t.Errorf("the rejected edits must be stashed for recovery, got stash list: %q", stashes)
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after rejecting the salvage: %q", status)
	}
	d := f.discarded(model.DiscardSalvageFailed)
	if !d.Stashed || len(d.Checks) == 0 || d.Error == "" {
		t.Errorf("round_discarded = %+v, want the stash, the failing check(s), and the coder error recorded", d)
	}
}

// The converse: partial work that DOES verify is still salvaged and the loop
// continues. The gate must reject broken work, not abandon the recovery behavior.
func TestRunSalvagesVerifiedPartialWork(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.verifyGate(config.VerifyMustPass, "broken") // never created: the edit is sound
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.editRepoOn(2)
	f.respond(2, "I changed files but forgot the <fix> envelope.")
	f.respond(3, reviewResponse(t))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want salvage + continue for work that passes the gate", err)
	}
	if sum.Rounds[0].CommitSHA == "" {
		t.Error("verified partial work must still be committed as a partial round")
	}
	if sum.Termination != model.TermConverged {
		t.Errorf("termination = %q, want converged", sum.Termination)
	}
}

// writeSide installs a custom side-effect script for the coder's invocation
// (call 2: review is call 1). editRepoOn writes a fixed one; this lets a test
// script arbitrary repo mutations, e.g. failing the salvage commit.
func (f *fixture) writeSide(body string) {
	const coderInvocation = 2
	testfixture.WriteSide(f.t, f.respDir, coderInvocation, "#!/bin/sh\n"+body)
}

// When the coder fails after editing but the salvage COMMIT itself fails, the
// edits must be stashed so the tree is never left dirty; the run then surfaces
// the coder error. Here forced commit signing with a failing signer fails the
// commit while the stash still succeeds.
func TestRunSalvageCommitFailsStashSucceeds(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder edits the repo, then forces the salvage commit to fail WITHOUT a
	// git hook (fixpoint disables hooks via core.hooksPath and --no-verify):
	// commit.gpgsign with a signer that always exits non-zero aborts git commit,
	// while git stash never signs its commits, so the fallback stash succeeds.
	f.writeSide(fmt.Sprintf("echo 'partial' >> '%s'\ngit -C '%s' config commit.gpgsign true\ngit -C '%s' config gpg.program /bin/false\n",
		filepath.Join(f.repo, "main.go"), f.repo, f.repo))
	f.respond(2, "edited files but no <fix> envelope") // parse failure -> salvage

	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "coder round 1 failed") {
		t.Fatalf("Run() err = %v, want coder-failure error after stash recovery", err)
	}
	// The commit failure is the OTHER half of the answer: the partial work passed
	// verification, so why it did not land has nothing to do with the coder, and
	// reporting only the coder's malformed output hides the real reason.
	for _, want := range []string{"could not be committed", "git commit"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Run() err = %v, want it to also report the commit failure (%q)", err, want)
		}
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after failed commit; want clean (stashed): %q", status)
	}
	if list := gitRun(t, f.repo, "stash", "list"); !strings.Contains(list, "failed round 1") {
		t.Errorf("expected a stash entry for the recovered edits, got %q", list)
	}
	// The stash exists because of a discard nobody asked for, so the journal has to
	// say which one -- and this reason is distinct from salvage_verify_failed: the
	// work was good.
	d := f.discarded(model.DiscardSalvageCommitFailed)
	if !d.Stashed {
		t.Error("round_discarded.stashed = false, want the recovered edits recorded as stashed")
	}
	if !strings.Contains(d.Error, "could not be committed") {
		t.Errorf("round_discarded.error = %q, want the failed salvage commit recorded", d.Error)
	}
}

// When BOTH the salvage commit and the fallback stash fail, the tree cannot be
// reconciled: the run returns a combined error naming the coder failure and the
// reconcile failure, and the tree is left dirty. Here an index.lock fails the
// commit's `git add` and the stash alike.
func TestRunSalvageCommitFailsStashFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder edits the repo, then plants an index.lock so both the salvage
	// commit's `git add` and the fallback `git stash` fail to lock the index.
	lock := filepath.Join(f.repo, ".git", "index.lock")
	f.writeSide(fmt.Sprintf("echo 'partial' >> '%s'\n: > '%s'\n", filepath.Join(f.repo, "main.go"), lock))
	f.respond(2, "edited files but no <fix> envelope") // parse failure -> salvage

	_, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() err = nil, want combined reconcile failure")
	}
	for _, want := range []string{"coder round 1 failed", "could not be reconciled", "left dirty"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Run() err = %v, want it to contain %q", err, want)
		}
	}
	// Remove the lock so git is usable again, then confirm the tree stayed dirty.
	os.Remove(lock)
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) == "" {
		t.Error("working tree is clean; the unreconcilable edits should have been left in place")
	}
}

// A normal round commit (the coder reported fixes and edited the tree) that
// FAILS after Commit's `git add -A` has staged the edits must not leave the tree
// staged and dirty -- that violates the clean-tree invariant and blocks the next
// run. The edits are stashed (recoverable via `git stash`) and the commit error
// surfaced, mirroring the partial-fix path's failed-commit reconciliation.
func TestRunNormalCommitFailsStashSucceeds(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder edits the repo and reports a valid fix, then forces the round
	// commit to fail: commit.gpgsign with a signer that always exits non-zero
	// aborts git commit, while git stash never signs, so the fallback succeeds.
	f.writeSide(fmt.Sprintf("echo 'fixed' >> '%s'\ngit -C '%s' config commit.gpgsign true\ngit -C '%s' config gpg.program /bin/false\n",
		filepath.Join(f.repo, "main.go"), f.repo, f.repo))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))

	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "git commit") {
		t.Fatalf("Run() err = %v, want a commit-failure error", err)
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after failed commit; want clean (stashed): %q", status)
	}
	if list := gitRun(t, f.repo, "stash", "list"); !strings.Contains(list, "failed commit in round 1") {
		t.Errorf("expected a stash entry for the recovered edits, got %q", list)
	}
	if d := f.discarded(model.DiscardCommitFailed); !d.Stashed || d.Error == "" {
		t.Errorf("round_discarded = %+v, want the failed commit recorded with its stash and cause", d)
	}
}

// When a normal round commit fails AND the fallback stash also fails, the tree
// cannot be reconciled: reconcileFailedCommit returns a combined error naming
// the commit failure and the unreconciled dirty tree, and the coder's edits are
// left in the worktree (recoverable). Here an index.lock fails the commit's
// `git add` and the stash alike.
func TestRunNormalCommitFailsStashFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder edits the repo and reports a valid fix, then plants an index.lock
	// so both the round commit's `git add` and the fallback `git stash` fail.
	lock := filepath.Join(f.repo, ".git", "index.lock")
	f.writeSide(fmt.Sprintf("echo 'fixed' >> '%s'\n: > '%s'\n", filepath.Join(f.repo, "main.go"), lock))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))

	_, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() err = nil, want combined commit+reconcile failure")
	}
	for _, want := range []string{"commit failed", "could not be reconciled", "left dirty"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Run() err = %v, want it to contain %q", err, want)
		}
	}
	// Remove the lock so git is usable again, then confirm the coder's edits stayed
	// in the worktree (recoverable) rather than being lost by a botched reconcile.
	os.Remove(lock)
	if status := gitRun(t, f.repo, "status", "--porcelain"); !strings.Contains(status, "main.go") {
		t.Errorf("coder edits to main.go should remain in the dirty worktree, got status %q", status)
	}
}

// A coder failure with a CLEAN tree (it did nothing) is a genuine hard error,
// not a salvageable round.
func TestRunCoderFailureWithNoEditsFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.respond(2, "no envelope, no edits")

	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "coder round 1 failed") {
		t.Fatalf("Run() err = %v, want hard coder failure", err)
	}
}

// A coder that claims a fix but leaves the tree unchanged is a hard error, not
// a silently-accepted round: nothing was really done, so there is nothing to
// commit and the disagreement must surface.
func TestRunFixedVerdictWithNoEditFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// Verdict "fixed" but no editRepoOn: the tree stays clean.
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "claimed"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "left the working tree unchanged") {
		t.Fatalf("Run() err = %v, want left-tree-unchanged error", err)
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error", sum.Termination)
	}
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (nothing must be committed)", got)
	}
}

// A rejected verdict sitting on a dirty tree is not a contradiction to end the
// run over: rejecting well often means editing -- the probe test that disproves a
// finding's premise. The edits belong to no verdict, so they must not be
// committed; they are stashed (recoverable, named by issue), the round carries
// on, and with nothing else to fix the run ends as an ordinary all-rejected
// round rather than an error.
func TestRunRejectedWithEditsStashesAndContinues(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// Verdict "rejected", yet the coder dirties the tree.
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "by design"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want nil (a rejected session's edits are stashed, not fatal)", err)
	}
	if sum.Termination != model.TermAllRejected {
		t.Errorf("termination = %q, want all-rejected", sum.Termination)
	}
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (a rejected session must not commit)", got)
	}
	if stashes := gitRun(t, f.repo, "stash", "list"); !strings.Contains(stashes, "rejected i1") {
		t.Errorf("the rejected session's edits must be stashed for recovery, named by issue, got stash list: %q", stashes)
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty, so the next run's clean-tree check would refuse to start: %q", status)
	}
	if d := f.sessionDiscarded(model.DiscardRejectedWithEdits); !d.Stashed || d.Issue != "i1" {
		t.Errorf("session_discarded = %+v, want the stash recorded and the issue named", d)
	}
	if got := sum.Rounds[0].StashedRejects; len(got) != 1 || got[0] != "i1" {
		t.Errorf("StashedRejects = %v, want [i1] so the summary can surface the stash", got)
	}
}

// A session that rejects its issue but edits the tree must not cost the round its
// remaining issues: the edits are stashed, the next session runs, and its fix
// lands. This is the shape of the guard's first real trip -- the coder disproved
// a finding with a probe test, and the old round-fatal response ended the run
// with the rest of the round's queue unheard.
func TestRunRejectedSessionEditsDoNotEndTheRound(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t,
		aFinding("bug one"),
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "other.go", Line: 7, Title: "bug two"}))
	// The session for i1 rejects yet edits; the session for i2 fixes.
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "premise false"}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "done"}))
	// Round 2 reviews clean over the committed fix.
	f.respond(4, reviewResponse(t))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	if sum.Termination != model.TermConverged {
		t.Errorf("termination = %q, want converged (the fix landed and round 2 was clean)", sum.Termination)
	}
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + the i2 fix)", got)
	}
	rec := sum.Rounds[0]
	if rec.Fixed != 1 || rec.Rejected != 1 {
		t.Errorf("round 1 fixed/rejected = %d/%d, want 1/1", rec.Fixed, rec.Rejected)
	}
	if len(rec.StashedRejects) != 1 || rec.StashedRejects[0] != "i1" {
		t.Errorf("StashedRejects = %v, want [i1]", rec.StashedRejects)
	}
	// The stash holds ONLY the rejected session's edits; the fix that followed was
	// committed, not swept into it.
	if stash := gitRun(t, f.repo, "stash", "show", "-p", "stash@{0}"); !strings.Contains(stash, "fix 2") || strings.Contains(stash, "fix 3") {
		t.Errorf("stash must hold the rejected session's edits and nothing later, got:\n%s", stash)
	}
	if head := gitRun(t, f.repo, "show", "HEAD"); !strings.Contains(head, "fix 3") {
		t.Errorf("HEAD must carry the i2 fix, got:\n%s", head)
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty: %q", status)
	}
}

// When the rejected session's edits cannot even be stashed, the tree stays
// dirty -- and the run must stop and SAY so, in the error and in the journal,
// rather than carrying on: the next session's verdict reconciliation would blame
// this session's edits on its own coder, and the next run's clean-tree check will
// refuse to start. An index.lock fails the stash.
func TestRunRejectedWithEditStashFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder dirties the tree and plants an index.lock, so the reconciling
	// `git stash` cannot lock the index.
	lock := filepath.Join(f.repo, ".git", "index.lock")
	f.writeSide(fmt.Sprintf("echo 'rejected but edited' >> '%s'\n: > '%s'\n", filepath.Join(f.repo, "main.go"), lock))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "by design"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() err = nil, want combined rejected-with-edits + reconcile failure")
	}
	for _, want := range []string{"rejected i1 yet modified", "could not be reconciled", "left dirty"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Run() err = %v, want it to contain %q", err, want)
		}
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error", sum.Termination)
	}
	if d := f.sessionDiscarded(model.DiscardRejectedWithEdits); d.Stashed || !strings.Contains(d.Error, "tree left dirty") {
		t.Errorf("session_discarded = %+v, want stashed=false and the dirty tree named", d)
	}
	// Remove the lock so git works again, then confirm the edits were left in place
	// rather than lost by a botched reconcile.
	os.Remove(lock)
	if status := gitRun(t, f.repo, "status", "--porcelain"); !strings.Contains(status, "main.go") {
		t.Errorf("the unreconcilable edits should remain in the worktree, got status %q", status)
	}
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (a rejected round must not commit)", got)
	}
}

// Findings over loop.max_findings_per_round are deferred worst-severity-first:
// the coder sees only the cap'd subset, deferred ones stay open, and an
// all-rejected verdict on the active subset does not terminate the run.
func TestRunDefersFindingsOverCap(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, MaxFindingsPerRound: 2})
	low := model.ReviewFinding{Category: "tests", Severity: "low", File: "main.go", Line: 1, Title: "low prio"}
	high := model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 2, Title: "high prio"}
	med := model.ReviewFinding{Category: "bugs", Severity: "medium", File: "main.go", Line: 3, Title: "med prio"}
	// Three distinct issues: i1(low) i2(high) i3(med). The cap of 2 keeps the two
	// worst, so the coder is asked about i2 and i3 only.
	f.respond(1, reviewResponse(t, low, high, med))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "fixed"}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i3", Verdict: "fixed", Detail: "fixed"}))
	f.respond(4, reviewResponse(t)) // round 2: clean

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged", sum.Termination)
	}
	r1 := sum.Rounds[0]
	if r1.Fixed != 2 {
		t.Errorf("round 1 fixed = %d, want 2", r1.Fixed)
	}
	byID := map[string]model.Finding{}
	for _, fd := range r1.Findings {
		byID[fd.ID] = fd
	}
	// Verdicts are mirrored from issues back onto the observations that reported
	// them, so these assertions still speak in reviewer terms.
	if byID["r1.1"].Verdict != model.VerdictDeferred {
		t.Errorf("low-severity finding verdict = %q, want deferred", byID["r1.1"].Verdict)
	}
	if byID["r1.2"].Verdict != "fixed" || byID["r1.3"].Verdict != "fixed" {
		t.Errorf("active findings not fixed: %+v", r1.Findings)
	}
	// One session per active issue, and the deferred one gets none.
	if n := f.coderSessions(1); n != 2 {
		t.Fatalf("round 1 spent %d coder session(s), want 2 (one per active issue)", n)
	}
	prompts := f.coderPrompt(1)
	if strings.Contains(prompts, "[i1]") {
		t.Error("deferred issue leaked into the coder prompt")
	}
	if !strings.Contains(prompts, "[i2]") || !strings.Contains(prompts, "[i3]") {
		t.Error("active issues missing from the coder prompts")
	}
}

// When a round caps findings and the coder rejects every ACTIVE one, the run
// must not terminate as all-rejected while deferred findings remain: they never
// reached the coder, so the loop keeps going and they resurface in a later
// round where they can finally receive a verdict.
func TestRunAllActiveRejectedContinuesForDeferred(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 4, CleanRoundsToStop: 1, MaxFindingsPerRound: 1})
	high := model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 1, Title: "high prio"}
	low := model.ReviewFinding{Category: "tests", Severity: "low", File: "main.go", Line: 2, Title: "low prio"}
	// Round 1: two findings, cap 1 -> r1.1(high) active, r1.2(low) deferred.
	f.respond(1, reviewResponse(t, high, low))
	// Coder rejects the only active finding; the tree stays clean.
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "by design"}))
	// Round 2: only the previously-deferred low issue remains -- it keeps its id
	// (i2) across rounds, which is the point of the ledger.
	f.respond(3, reviewResponse(t, low))
	f.editRepoOn(4)
	f.respond(4, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "patched"}))
	// Round 3: clean -> converge.
	f.respond(5, reviewResponse(t))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	// The all-active-rejected round 1 must NOT have terminated the run.
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged (round 1 must not end as all-rejected)", sum.Termination)
	}
	if len(sum.Rounds) != 3 {
		t.Fatalf("rounds = %d, want 3 (deferred finding must get a later round)", len(sum.Rounds))
	}
	r1 := sum.Rounds[0]
	byID := map[string]model.Finding{}
	for _, fd := range r1.Findings {
		byID[fd.ID] = fd
	}
	if byID["r1.1"].Verdict != "rejected" {
		t.Errorf("round 1 active finding verdict = %q, want rejected", byID["r1.1"].Verdict)
	}
	if byID["r1.2"].Verdict != model.VerdictDeferred {
		t.Errorf("round 1 low finding verdict = %q, want deferred", byID["r1.2"].Verdict)
	}
	// The deferred finding resurfaced and finally received a real verdict.
	r2 := sum.Rounds[1]
	if len(r2.Findings) != 1 || r2.Findings[0].Verdict != "fixed" {
		t.Errorf("round 2 findings = %+v, want the resurfaced finding fixed", r2.Findings)
	}
}

// deferOverCap ranks issues worst-severity-first; a severity not in the known
// vocabulary must sort last rather than winning a slot. Reviewers are told the
// vocabulary but nothing forces them to obey it, so an invented severity must
// never outrank a real critical. Driven through the ledger, which is where issue
// identity and deferral counts live.
func TestDeferOverCapUnknownSeverityRankedLast(t *testing.T) {
	f := newFixture(t, config.Loop{MaxFindingsPerRound: 2})
	o := f.orchestrator()
	rec := &model.RoundRecord{Round: 1, Findings: []model.Finding{
		{Severity: "low", File: "a.go", Line: 1, Title: "known low"},
		{Severity: "", File: "b.go", Line: 1, Title: "empty severity"},
		{Severity: "high", File: "c.go", Line: 1, Title: "known high"},
		{Severity: "bogus", File: "d.go", Line: 1, Title: "unknown severity"},
	}}
	rec.Issues = o.ledger.Absorb(1, rec.Findings)
	o.deferOverCap(rec)

	byTitle := map[string]model.Issue{}
	for _, it := range rec.Issues {
		byTitle[it.Title] = it
	}
	// Cap 2: the two known severities stay active; the two unknown ones rank last.
	for _, title := range []string{"known high", "known low"} {
		if byTitle[title].Verdict == model.VerdictDeferred {
			t.Errorf("%q was deferred; known severities must win slots over unknown ones", title)
		}
	}
	for _, title := range []string{"empty severity", "unknown severity"} {
		if byTitle[title].Verdict != model.VerdictDeferred {
			t.Errorf("%q was not deferred; an unknown severity must sort last", title)
		}
	}
}

func TestRunRefusesNonGitRepoForFixRounds(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.cfg.Target.Path = t.TempDir() // not a git repo
	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("Run() err = %v, want non-git refusal", err)
	}
}

// The mirror of the refusal above: a review-only directory run is the documented
// way to review a tree that is not a repository at all (Collect falls back to a
// filesystem walk), so it must reach the reviewers. Every git probe on this path
// fails soft on "not a repository" for that reason, and nothing a review-only run
// does needs a HEAD -- it never commits, so it never squashes.
func TestRunReviewOnlyAllowsNonGitDirectory(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, ReviewOnly: true})
	f.cfg.Target.Path = t.TempDir() // not a git repo
	if err := os.WriteFile(filepath.Join(f.cfg.Target.Path, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.respond(1, reviewResponse(t, aFinding("bug")))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want a review-only run against a non-git directory to succeed", err)
	}
	if sum.Termination != model.TermReviewOnly {
		t.Fatalf("termination = %q, want review-only", sum.Termination)
	}
	if got := f.invocations(); got != 1 {
		t.Errorf("agent invocations = %d, want 1 (the reviewer must have run)", got)
	}
}

// The repo-supplied-git-config gate applies in EVERY mode that launches an agent
// with its working directory inside the target, and a review-only directory run
// is the shipped review bundle's own shape -- so an UNTRUSTED one over a plain
// folder is the target shape that widening newly probes. Nothing but the
// IsGitRepo short-circuit holds UnsafeConfig/ExternalActivatableConfig back there, and
// no trust is asserted to soften what they report: a probe that errored, or a
// listing that read the operator's own global config as the target's, would kill
// the run before its reviewer ever launched -- and the operator's filters, which
// no git command can fire outside a repository, must not even be warned about.
// The gate is exercised through PreflightGuards (the agent-invoking entry point
// run() and Ping share) rather than the probes alone, because that is where the
// short-circuit lives.
func TestUntrustedReviewOnlyNonGitDirectoryPassesConfigGate(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
	f.cfg.Loop.TrustedTarget = false // undo the fixture's trusted default
	f.cfg.Target.Path = t.TempDir()  // not a git repo

	// The operator config `git lfs install` writes, installed after the fixture so
	// it overrides the empty global scope the fixture pins: outside a repository
	// these filters can never fire, so they must be neither refused nor warned about.
	global := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(global, []byte("[filter \"lfs\"]\n\tclean = git-lfs clean -- %f\n\tsmudge = git-lfs smudge -- %f\n\tprocess = git-lfs filter-process\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)

	if err := os.WriteFile(filepath.Join(f.cfg.Target.Path, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.respond(1, reviewResponse(t, aFinding("bug")))

	o, logged := f.capturingOrchestrator()
	if err := o.PreflightGuards(t.Context()); err != nil {
		t.Fatalf("PreflightGuards() err = %v, want an untrusted review-only run over a non-git directory to proceed", err)
	}
	if strings.Contains(logged(), "filter.lfs") {
		t.Errorf("preflight warned about filters no git command can fire outside a repository:\n%s", logged())
	}
	sum, err := o.Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want the reviewer to launch against a non-git directory", err)
	}
	if sum.Termination != model.TermReviewOnly {
		t.Fatalf("termination = %q, want review-only", sum.Termination)
	}
	if got := f.invocations(); got != 1 {
		t.Errorf("agent invocations = %d, want 1 (the reviewer must have run)", got)
	}
}

func TestRunRefusesUntrustedPRFixRounds(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.cfg.Target.Mode = "pr"
	f.cfg.Target.PR = 1
	// -trusted-target must not bypass the pr gate: a PR is untrusted regardless.
	f.cfg.Loop.TrustedTarget = true
	_, err := f.orchestrator().Run(t.Context())
	// The message must name the FLAG: these fields are no longer settable in YAML,
	// so pointing a reader at loop.allow_untrusted_fix would send them to a config
	// key that now fails to load.
	if err == nil || !strings.Contains(err.Error(), "-allow-untrusted-fix") {
		t.Fatalf("Run() err = %v, want untrusted-PR refusal naming the opt-in flag", err)
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (must refuse before running anything)", got)
	}
}

// Fix rounds against a directory/git-diff target are fail-closed: without an
// explicit trust assertion the coder (which edits files with permission checks
// disabled) must never run.
func TestRunRefusesUntrustedDirectoryFixRounds(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.cfg.Loop.TrustedTarget = false // undo the fixture's trusted default
	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "-trusted-target") {
		t.Fatalf("Run() err = %v, want untrusted-target refusal naming the -trusted-target flag", err)
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (must refuse before running anything)", got)
	}
}

// A review-only git-diff run against a target whose repo-local config defines an
// execution-capable content filter is refused before any git command touches the
// worktree: `git diff` would normalize worktree files through that clean filter,
// running a repo-controlled program with fixpoint's environment. Asserting trust
// via trusted_target is the operator's acknowledgement and lets it proceed.
func TestRunRefusesUntrustedGitDiffConfigFilter(t *testing.T) {
	t.Run("untrusted target with clean filter is refused", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false // undo the fixture's trusted default
		f.cfg.Target.Mode = "git-diff"
		gitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")

		_, err := f.orchestrator().Run(t.Context())
		if err == nil || !strings.Contains(err.Error(), "repo-controlled programs") {
			t.Fatalf("Run() err = %v, want refusal citing repo-controlled programs", err)
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (must refuse before running git)", got)
		}
	})

	t.Run("trusted target with clean filter proceeds", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Target.Mode = "git-diff" // fixture keeps trusted_target: true
		gitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")
		f.respond(1, reviewResponse(t)) // clean review

		if _, err := f.orchestrator().Run(t.Context()); err != nil {
			t.Fatalf("Run() err = %v, want the trust assertion to bypass the config guard", err)
		}
	})
}

// Directory review-only -- the shipped review bundle's own shape -- is gated too,
// because the gate is not only about fixpoint's git commands: every agent is
// launched with its working directory INSIDE the target and runs `git status`/
// `git diff`/`git log` while exploring, which fires a repo-supplied clean filter
// just as fixpoint's own diff would. --check is the exception: it invokes no
// agent, so it keeps the cheap path rather than refusing every git-lfs checkout.
func TestRunRefusesUntrustedDirectoryReviewOnlyConfigFilter(t *testing.T) {
	t.Run("review-only run is refused before any agent", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false // undo the fixture's trusted default
		gitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")

		_, err := f.orchestrator().Run(t.Context())
		if err == nil || !strings.Contains(err.Error(), "repo-controlled programs") {
			t.Fatalf("Run() err = %v, want refusal citing repo-controlled programs", err)
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (must refuse before launching an agent)", got)
		}
	})

	t.Run("check-live is refused before any agent", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		gitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")

		err := f.orchestrator().Ping(t.Context())
		if err == nil || !strings.Contains(err.Error(), "repo-controlled programs") {
			t.Fatalf("Ping() err = %v, want refusal citing repo-controlled programs", err)
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (must refuse before pinging)", got)
		}
	})

	t.Run("check invokes no agent and is not refused", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		gitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")

		if err := f.orchestrator().PreflightGuardsNoAgent(t.Context()); err != nil {
			t.Fatalf("PreflightGuardsNoAgent() err = %v, want --check on a directory review-only target to proceed", err)
		}
	})

	// The preflight answer is memoized, and a --check answer is the WEAK one: it
	// may have skipped the config gate entirely. So a cached no-agent pass must
	// not satisfy an agent-invoking caller on the same orchestrator -- the gate
	// has to re-probe in that direction, or the day a caller does both the agent
	// path runs unguarded.
	t.Run("a cached --check pass does not satisfy an agent-invoking caller", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		gitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")
		o := f.orchestrator()

		if err := o.PreflightGuardsNoAgent(t.Context()); err != nil {
			t.Fatalf("PreflightGuardsNoAgent() err = %v, want --check on a directory review-only target to proceed", err)
		}
		err := o.PreflightGuards(t.Context())
		if err == nil || !strings.Contains(err.Error(), "repo-controlled programs") {
			t.Fatalf("PreflightGuards() after a cached --check pass err = %v, want refusal citing repo-controlled programs", err)
		}
	})

	t.Run("trusted target proceeds", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true}) // fixture keeps trusted_target: true
		gitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")
		f.respond(1, reviewResponse(t)) // clean review

		if _, err := f.orchestrator().Run(t.Context()); err != nil {
			t.Fatalf("Run() err = %v, want the trust assertion to bypass the config guard", err)
		}
	})

	// The other half of the memoization: run() guards and then pings, and Ping
	// guards again, so without it the operator sees the same WARNING twice for
	// one target and starts discounting it.
	t.Run("trusted target warns once across run and ping", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true}) // fixture keeps trusted_target: true
		ping := true
		f.cfg.PingAgents = &ping // undo the fixture's no-ping default: this is the double-guard path
		gitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")
		f.respond(1, "OK")              // ping
		f.respond(2, reviewResponse(t)) // clean review
		o, logged := f.capturingOrchestrator()

		if _, err := o.Run(t.Context()); err != nil {
			t.Fatalf("Run() err = %v, want the trust assertion to bypass the config guard", err)
		}
		if got := strings.Count(logged(), "has repo-supplied git config"); got != 1 {
			t.Errorf("repo-supplied git config warnings = %d, want 1, got logs:\n%s", got, logged())
		}
	})
}

// A repo-defined diff driver is the same class as a content filter, reached from
// the other direction: the agents run their own `git diff`/`git log -p`/`git blame`
// inside the target, and diff.external / diff.<driver>.command / textconv are all
// programs those commands execute. Nothing else stops it -- Collect's
// --no-ext-diff --no-textconv only covers fixpoint's own patch generation, and the
// driver's name is the repository's to choose so no -c override can reach it. So an
// untrusted review-only DIRECTORY run, the shipped review bundle's own shape, must
// refuse before an agent is ever launched.
func TestRunRefusesUntrustedDirectoryReviewOnlyDiffDriver(t *testing.T) {
	for _, key := range []string{"diff.external", "diff.evil.command", "diff.evil.textconv"} {
		t.Run(key, func(t *testing.T) {
			f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
			f.cfg.Loop.TrustedTarget = false // undo the fixture's trusted default
			gitRun(t, f.repo, "config", key, "./payload")

			_, err := f.orchestrator().Run(t.Context())
			if err == nil || !strings.Contains(err.Error(), "repo-controlled programs") {
				t.Fatalf("Run() err = %v, want refusal citing repo-controlled programs", err)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("Run() err = %v, want it to name %s", err, key)
			}
			if got := f.invocations(); got != 0 {
				t.Errorf("agent invocations = %d, want 0 (must refuse before launching an agent)", got)
			}
		})
	}

	// core.fsmonitor is deliberately NOT on that list: `core.fsmonitor = true` is
	// git's own builtin daemon, so refusing the key would refuse legitimate
	// repositories. gitenv pins it to false in every environment fixpoint hands out
	// instead -- the agents' included -- which is what makes proceeding here safe
	// rather than an oversight. TestRunHardensGitForAgentInvokedGit is the other half.
	t.Run("core.fsmonitor is neutralized, not refused", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		gitRun(t, f.repo, "config", "core.fsmonitor", "./payload")
		f.respond(1, reviewResponse(t)) // clean review

		if _, err := f.orchestrator().Run(t.Context()); err != nil {
			t.Fatalf("Run() err = %v, want a pinned setting not to refuse the target", err)
		}
	})
}

// A review-only pr run is gated on the same execution-capable git config. The
// PR cannot write .git/config, but `gh pr checkout` writes the worktree and git
// runs a configured smudge/process filter during checkout -- and the PR supplies
// the .gitattributes that selects it (plus any worktree script it points at). So
// the guard must refuse before Prepare checks anything out, in the DEFAULT
// review-only path where no untrusted-fix opt-in is involved.
func TestRunRefusesUntrustedPRConfigFilter(t *testing.T) {
	t.Run("untrusted PR target with smudge filter is refused", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false // undo the fixture's trusted default
		f.cfg.Target.Mode = "pr"
		f.cfg.Target.PR = 7
		gitRun(t, f.repo, "config", "filter.evil.smudge", "sh -c 'id'")

		_, err := f.orchestrator().Run(t.Context())
		if err == nil || !strings.Contains(err.Error(), "repo-controlled programs") {
			t.Fatalf("Run() err = %v, want refusal citing repo-controlled programs", err)
		}
		// Nothing ran: no agent, and no gh pr checkout (which would have failed
		// against this local repo anyway -- the point is the guard came first).
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (must refuse before checkout)", got)
		}
		if branch := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "--abbrev-ref", "HEAD")); branch != "main" && branch != "master" {
			t.Errorf("HEAD is on %q; the guard must refuse before any checkout", branch)
		}
	})

	t.Run("PR target without unsafe config passes the guard", func(t *testing.T) {
		// The guard must not refuse an ordinary PR review: this run gets past it and
		// fails later, at the gh pr checkout Prepare runs against a local repo.
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		f.cfg.Target.Mode = "pr"
		f.cfg.Target.PR = 7

		_, err := f.orchestrator().Run(t.Context())
		if err == nil {
			t.Fatal("Run() err = nil, want the checkout of a nonexistent PR to fail")
		}
		if strings.Contains(err.Error(), "repo-controlled programs") {
			t.Errorf("Run() err = %v, want the config guard to allow a repo with no unsafe keys", err)
		}
	})
}

// Git config is BRANCH-CONDITIONAL, so the preflight's answer expires at the
// checkout: an `includeIf "onbranch:<pattern>"` in .git/config pulls in a config
// file only while a matching branch is checked out. A crafted checkout can
// therefore park a filter.<name>.clean in a file that is inert on the branch the
// preflight inspects and active on the PR branch `gh pr checkout` switches to --
// after which the post-Prepare `git status`, the `git diff` that collects the PR,
// and every agent's own git commands run that repo-controlled program with
// fixpoint's inherited environment, in a review-only run that asserted no trust.
// So the gates must run AGAIN on the checked-out branch, and their memoized
// pre-checkout answer must not stand in for that.
func TestRunRechecksConfigGuardAfterPRCheckout(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
	f.cfg.Loop.TrustedTarget = false // undo the fixture's trusted default
	f.cfg.Target.Mode = "pr"
	f.cfg.Target.PR = 7

	// A clean feature branch for gh to check out, plus the base oid gh reports.
	gitRun(t, f.repo, "branch", "-M", "main")
	mainSHA := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "HEAD"))
	gitRun(t, f.repo, "checkout", "-q", "-b", "feature")
	if err := os.WriteFile(filepath.Join(f.repo, "main.go"), []byte("package main\n\nfunc pr() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, f.repo, "commit", "-aqm", "pr change")
	gitRun(t, f.repo, "checkout", "-q", "main")

	// The payload: a clean filter that only exists while `feature` is checked out.
	// Written straight into .git (a crafted checkout's own doing -- a PR cannot
	// write there), and left unselected by any .gitattributes so this test never
	// actually runs it.
	evil := filepath.Join(f.repo, ".git", "evil-config")
	if err := os.WriteFile(evil, []byte("[filter \"evil\"]\n\tclean = sh -c 'id'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(f.repo, ".git", "config")
	existing, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, append(existing, []byte("[includeIf \"onbranch:feature\"]\n\tpath = evil-config\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	// The include really is invisible from the branch the preflight sees; without
	// that the test would pass on the pre-checkout gate alone.
	if keys, err := target.New(f.cfg.Target).UnsafeConfig(t.Context()); err != nil || len(keys) != 0 {
		t.Fatalf("UnsafeConfig() = %v, %v on the pre-checkout branch, want none (an onbranch include is inert there)", keys, err)
	}

	// The stub logs each call, so the test can tell WHEN the recheck ran: the rest
	// of Prepare (`gh pr view`, then the base fetch, which would run an activated
	// credential helper or ssh command) must never happen.
	binDir := t.TempDir()
	ghCalls := filepath.Join(t.TempDir(), "gh-calls")
	stub := "#!/bin/sh\n" +
		`echo "$1 $2" >> "` + ghCalls + `"` + "\n" +
		`case "$1 $2" in` + "\n" +
		`"pr checkout") git checkout -q feature ;;` + "\n" +
		`"pr view") echo ` + mainSHA + " ;;\n" +
		`*) echo "unexpected gh call: $@" >&2; exit 1 ;;` + "\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err = f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "repo-controlled programs") {
		t.Fatalf("Run() err = %v, want the post-checkout recheck to refuse, citing repo-controlled programs", err)
	}
	if !strings.Contains(err.Error(), "filter.evil.clean") {
		t.Errorf("Run() err = %v, want it to name the setting the checkout activated", err)
	}
	if branch := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "--abbrev-ref", "HEAD")); branch != "feature" {
		t.Fatalf("HEAD is on %q, want feature: the checkout must have happened, or this tests the pre-checkout gate", branch)
	}
	calls, err := os.ReadFile(ghCalls)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Fields(strings.TrimSpace(string(calls))); len(got) != 2 || got[1] != "checkout" {
		t.Errorf("gh calls = %q, want the checkout alone: the recheck must refuse before the rest of Prepare runs git", got)
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (must refuse before any agent runs in the checked-out tree)", got)
	}
}

// A filter the OPERATOR configured globally is a program the PR can still reach:
// it does not define the filter, it ships the .gitattributes that SELECTS it (and
// the worktree script it may run, and the .lfsconfig that redirects where a
// git-lfs filter talks). All three arrive with the `gh pr checkout` that fires the
// filter, so the preflight has nothing to inspect and pr mode refuses instead of
// warning. Local modes keep the warning: there the attributes are already on disk
// and refusing would refuse every run on a git-lfs host.
func TestRunRefusesActivatableFiltersInPRMode(t *testing.T) {
	// installGlobalFilter gives the test process the operator config `git lfs
	// install` writes, overriding the empty global scope the fixture pins.
	installGlobalFilter := func(t *testing.T) {
		t.Helper()
		global := filepath.Join(t.TempDir(), "gitconfig")
		if err := os.WriteFile(global, []byte("[filter \"lfs\"]\n\tclean = git-lfs clean -- %f\n\tsmudge = git-lfs smudge -- %f\n\tprocess = git-lfs filter-process\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_CONFIG_GLOBAL", global)
	}
	// guardLog runs the preflight with a capturing logger, for the paths that warn
	// rather than refuse.
	guardLog := func(t *testing.T, f *fixture) (string, error) {
		t.Helper()
		var log strings.Builder
		o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, func(format string, args ...any) {
			fmt.Fprintf(&log, format+"\n", args...)
		})
		if err != nil {
			t.Fatal(err)
		}
		err = o.PreflightGuards(t.Context())
		return log.String(), err
	}

	t.Run("untrusted PR run is refused before checkout", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false // undo the fixture's trusted default
		f.cfg.Target.Mode = "pr"
		f.cfg.Target.PR = 7
		installGlobalFilter(t)

		_, err := f.orchestrator().Run(t.Context())
		if err == nil || !strings.Contains(err.Error(), "filter.lfs.clean") {
			t.Fatalf("Run() err = %v, want refusal naming the activatable filter", err)
		}
		if !strings.Contains(err.Error(), "-trusted-target") {
			t.Errorf("Run() err = %v, want it to name the opt-in flag", err)
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (must refuse before checkout)", got)
		}
		if branch := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "--abbrev-ref", "HEAD")); branch != "main" && branch != "master" {
			t.Errorf("HEAD is on %q; the guard must refuse before any checkout", branch)
		}
	})

	t.Run("asserted trust downgrades the PR refusal to a warning", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		f.cfg.Loop.AllowUntrustedFix = true
		f.cfg.Target.Mode = "pr"
		f.cfg.Target.PR = 7
		installGlobalFilter(t)

		log, err := guardLog(t, f)
		if err != nil {
			t.Fatalf("PreflightGuards() err = %v, want the trust assertion to clear the filter gate", err)
		}
		if !strings.Contains(log, "WARNING") || !strings.Contains(log, "filter.lfs.clean") {
			t.Errorf("accepting the path must still name the filters:\n%s", log)
		}
	})

	t.Run("--check on an untrusted PR proceeds", func(t *testing.T) {
		// --check never reaches the checkout the refusal is about: it runs Scope,
		// which reports pr mode as "known only after `gh pr checkout`" without
		// issuing a git command, and exits. Refusing here would refuse the cheap
		// first look at an unfamiliar config on every host with `git lfs install`.
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		f.cfg.Target.Mode = "pr"
		f.cfg.Target.PR = 7
		installGlobalFilter(t)

		var log strings.Builder
		o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, func(format string, args ...any) {
			fmt.Fprintf(&log, format+"\n", args...)
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := o.PreflightGuardsNoAgent(t.Context()); err != nil {
			t.Fatalf("PreflightGuardsNoAgent() err = %v, want --check on a pr target to proceed", err)
		}
		if !strings.Contains(log.String(), "WARNING") || !strings.Contains(log.String(), "filter.lfs.clean") {
			t.Errorf("--check must still name the filters a real run would fire:\n%s", log.String())
		}
	})

	t.Run("untrusted local run only warns", func(t *testing.T) {
		// Not a false refusal: the tree is the one the operator pointed fixpoint at,
		// its .gitattributes is already theirs to read, and the filter is their own
		// program -- refusing here would refuse every git-lfs host.
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		installGlobalFilter(t)

		log, err := guardLog(t, f)
		if err != nil {
			t.Fatalf("PreflightGuards() err = %v, want a directory run to proceed", err)
		}
		if !strings.Contains(log, "WARNING") || !strings.Contains(log, "filter.lfs.clean") {
			t.Errorf("the local path must warn and name the filters:\n%s", log)
		}
	})
}

// A repo-local core.worktree redirects git's work tree away from target.path
// while .git stays put, so target.path still looks like an ordinary checkout.
// git-diff/pr collection is where that bites hardest: Collect runs `git diff` and
// `git ls-files` unconditionally, so the redirected directory's files become the
// material handed to the reviewers -- and the untrusted-config guard ahead of it
// is skipped entirely, because `--is-inside-work-tree` answers false once the work
// tree no longer contains target.path. Unlike the execution-capable config keys,
// -trusted-target does not excuse this: trusting a checkout's CONTENT is not
// consent to review and commit some other directory than the one named.
func TestRunRefusesRedirectedWorktree(t *testing.T) {
	t.Run("git-diff review-only is refused before collection", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		f.cfg.Target.Mode = "git-diff"
		elsewhere := t.TempDir()
		if err := os.WriteFile(filepath.Join(elsewhere, "secret.txt"), []byte("token\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		gitRun(t, f.repo, "config", "core.worktree", elsewhere)

		_, err := f.orchestrator().Run(t.Context())
		if err == nil || !strings.Contains(err.Error(), "work tree") {
			t.Fatalf("Run() err = %v, want refusal citing the redirected work tree", err)
		}
		if !strings.Contains(err.Error(), elsewhere) {
			t.Errorf("Run() err = %v, want it to name the redirected root %s", err, elsewhere)
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (must refuse before collecting)", got)
		}
	})

	// A fix run stops on the redirect too, with the diagnostic that actually names
	// it. Without the guard this shape does still stop -- but incidentally, on
	// "target.path is not a git repository" (IsGitRepo reads the redirect as "no
	// work tree here"), which sends the operator looking for a missing .git that is
	// sitting right there.
	t.Run("trusted directory fix run is refused too", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1}) // fixture keeps trusted_target: true
		before := f.commitCount()
		elsewhere := t.TempDir()
		gitRun(t, f.repo, "config", "core.worktree", elsewhere)

		_, err := f.orchestrator().Run(t.Context())
		if err == nil || !strings.Contains(err.Error(), "work tree") {
			t.Fatalf("Run() err = %v, want the trust assertion NOT to bypass the work-tree check", err)
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (must refuse before the coder runs)", got)
		}
		if got := f.commitCount(); got != before {
			t.Errorf("commits = %d, want %d: nothing may be committed from the redirected tree", got, before)
		}
	})

	// The guard must not refuse ordinary targets: every other test here runs
	// against a plain checkout, and a subdirectory target legitimately reports an
	// ancestor as its work-tree root.
	t.Run("subdirectory target passes the guard", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		sub := filepath.Join(f.repo, "sub")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		f.cfg.Target.Path = sub
		f.respond(1, reviewResponse(t)) // clean review

		if _, err := f.orchestrator().Run(t.Context()); err != nil {
			t.Fatalf("Run() err = %v, want a subdirectory target to pass the work-tree check", err)
		}
	})
}

// Two reviewers that each return a finding in the same round must receive finding
// IDs in assignment (prompt-list) order -- r1.1 to the first lens, r1.2 to the
// second -- regardless of which reviewer's process finishes first. review()
// gathers results into a position-indexed slice and numbers them by iterating it
// in order; a map-based gather would make the IDs non-deterministic.
func TestRunReviewTwoReviewersDeterministicIDs(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})

	// Each lens gets its own agent backed by a fixed response (independent of
	// global invocation order), so a finding's Agent field identifies the
	// assignment that produced it. The first lens's agent sleeps so it finishes
	// LAST, proving IDs track assignment order rather than completion order.
	slow := fixedResponseScript(t, reviewResponse(t, aFinding("from slow")), 250*time.Millisecond)
	fast := fixedResponseScript(t, reviewResponse(t, aFinding("from fast")), 0)
	f.cfg.Agents["slow"] = config.Agent{Command: []string{slow}, PromptVia: "stdin", Timeout: config.Duration(time.Minute)}
	f.cfg.Agents["fast"] = config.Agent{Command: []string{fast}, PromptVia: "stdin", Timeout: config.Duration(time.Minute)}
	f.cfg.Roles.Review.Strategy = "fixed"
	f.cfg.Roles.Review.Prompts = []config.ReviewLens{
		{Agent: "slow", Prompt: f.cfg.Roles.Review.Prompts[0].Prompt},
		{Agent: "fast", Prompt: f.cfg.Roles.Review.Prompts[0].Prompt},
	}

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	fs := sum.Rounds[0].Findings
	if len(fs) != 2 {
		t.Fatalf("got %d findings, want 2: %+v", len(fs), fs)
	}
	if fs[0].ID != "r1.1" || fs[0].Agent != "slow" {
		t.Errorf("finding[0] = {ID:%q Agent:%q}, want r1.1 from slow (first assignment)", fs[0].ID, fs[0].Agent)
	}
	if fs[1].ID != "r1.2" || fs[1].Agent != "fast" {
		t.Errorf("finding[1] = {ID:%q Agent:%q}, want r1.2 from fast (second assignment)", fs[1].ID, fs[1].Agent)
	}
}

// fixedResponseScript writes an agent that drains stdin, optionally sleeps, then
// prints response verbatim -- unlike the count-based mock, its answer does not
// depend on global invocation order, so a per-lens agent is identifiable.
func fixedResponseScript(t *testing.T, response string, delay time.Duration) string {
	t.Helper()
	dir := t.TempDir()
	respFile := filepath.Join(dir, "resp")
	if err := os.WriteFile(respFile, []byte(response), 0o600); err != nil {
		t.Fatal(err)
	}
	sleep := ""
	if delay > 0 {
		sleep = fmt.Sprintf("sleep %.3f\n", delay.Seconds())
	}
	script := filepath.Join(dir, "agent.sh")
	body := "#!/bin/sh\ncat > /dev/null\n" + sleep + "cat '" + respFile + "'\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return script
}

func TestRunInterrupted(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	sum, err := f.orchestrator().Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermInterrupted {
		t.Fatalf("termination = %q, want interrupted", sum.Termination)
	}
}

// Cancellation while the coder is running kills it and surfaces as the fix
// round's error; the summary must still record an interruption, not an error.
func TestRunInterruptedDuringFix(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder invocation signals readiness, then blocks until killed.
	ready := filepath.Join(f.respDir, "coder-started")
	side := fmt.Sprintf("#!/bin/sh\ntouch '%s'\nsleep 60\n", ready)
	if err := os.WriteFile(filepath.Join(f.respDir, "side-2.sh"), []byte(side), 0o700); err != nil {
		t.Fatal(err)
	}
	f.respond(2, "never reached")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()

	sum, err := f.orchestrator().Run(ctx)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil for an interruption", err)
	}
	if sum.Termination != model.TermInterrupted {
		t.Fatalf("termination = %q, want interrupted", sum.Termination)
	}
	if sum.Error != "" {
		t.Errorf("summary error = %q, want empty for an interruption", sum.Error)
	}
}

// Cancellation while the coder is running must NOT commit its partial edits:
// the user asked to stop, so the edits are stashed (leaving a clean tree) and
// the run ends as an interruption rather than a labeled partial commit.
func TestRunInterruptedDuringFixDoesNotCommit(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder edits the repo, signals readiness, then blocks until killed.
	ready := filepath.Join(f.respDir, "coder-started")
	side := fmt.Sprintf("#!/bin/sh\necho 'partial edit' >> '%s'\ntouch '%s'\nsleep 60\n",
		filepath.Join(f.repo, "main.go"), ready)
	if err := os.WriteFile(filepath.Join(f.respDir, "side-2.sh"), []byte(side), 0o700); err != nil {
		t.Fatal(err)
	}
	f.respond(2, "never reached")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()

	sum, err := f.orchestrator().Run(ctx)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil for an interruption", err)
	}
	if sum.Termination != model.TermInterrupted {
		t.Fatalf("termination = %q, want interrupted", sum.Termination)
	}
	// The partial edit must not have been committed as a salvage round.
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (partial edits must not be committed on cancel)", got)
	}
	// The tree is reconciled (stashed), not left dirty.
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after interruption; want clean (stashed): %q", status)
	}
	if list := gitRun(t, f.repo, "stash", "list"); !strings.Contains(list, "interrupted round 1") {
		t.Errorf("expected a stash entry for the interrupted edits, got %q", list)
	}
	if d := f.discarded(model.DiscardInterrupted); !d.Stashed || d.Error == "" {
		t.Errorf("round_discarded = %+v, want the interruption recorded with its stash and cause", d)
	}
}

// A cancellation that lands AFTER the coder returned valid output (runErr ==
// nil) but before the round is committed must still be treated as an
// interruption: the successful coder's edits are stashed, not committed. The
// kill-the-coder interrupt tests always leave runErr set to the kill error, so
// they never reach fix()'s runErr==nil sub-branch (interrupted = ctx.Err());
// this one does, by canceling deterministically at the fix step's "done" log --
// after runAgent has returned success but before fix() checks ctx.Err().
func TestRunInterruptedAfterSuccessfulCoderDoesNotCommit(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.editRepoOn(2) // the coder edits the repo...
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	logf := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		t.Log(msg)
		// The coder has fully returned (runErr == nil) by the time its fix step
		// logs "done"; canceling here is observed by fix()'s later ctx.Err() check.
		// The label names the issue the session was given ("fix: mock on i1 done").
		if strings.HasPrefix(msg, "fix: mock") && strings.Contains(msg, " done") {
			cancel()
		}
	}
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, logf)
	if err != nil {
		t.Fatal(err)
	}

	sum, err := o.Run(ctx)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil for an interruption", err)
	}
	if sum.Termination != model.TermInterrupted {
		t.Fatalf("termination = %q, want interrupted", sum.Termination)
	}
	// The coder ran to completion (one review, one successful coder invocation)...
	if got := f.invocations(); got != 2 {
		t.Errorf("agent invocations = %d, want 2 (review + successful coder)", got)
	}
	// ...but a canceled round must not commit even a successful coder's edits.
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (a canceled round must not commit)", got)
	}
	// The successful edits are reconciled (stashed), leaving a clean tree.
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after interruption; want clean (stashed): %q", status)
	}
	if list := gitRun(t, f.repo, "stash", "list"); !strings.Contains(list, "interrupted round 1") {
		t.Errorf("expected a stash entry for the interrupted edits, got %q", list)
	}
	if d := f.discarded(model.DiscardInterrupted); !d.Stashed || d.Error == "" {
		t.Errorf("round_discarded = %+v, want the interruption recorded with its stash and cause", d)
	}
}

// When cancellation kills the coder after it has edited the tree AND the stash
// that would reconcile those edits fails (index locked), the tree is left
// dirty. That reconciliation failure must NOT be softened into a clean
// interruption: the run surfaces the error distinctly, the summary records it
// as an error, and the dirty tree is left in place for the user to recover.
func TestRunInterruptedDuringFixStashFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder edits the repo, plants an index.lock so the reconciling stash
	// cannot lock the index, signals readiness, then blocks until killed.
	ready := filepath.Join(f.respDir, "coder-started")
	lock := filepath.Join(f.repo, ".git", "index.lock")
	side := fmt.Sprintf("#!/bin/sh\necho 'partial edit' >> '%s'\n: > '%s'\ntouch '%s'\nsleep 60\n",
		filepath.Join(f.repo, "main.go"), lock, ready)
	if err := os.WriteFile(filepath.Join(f.respDir, "side-2.sh"), []byte(side), 0o700); err != nil {
		t.Fatal(err)
	}
	f.respond(2, "never reached")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()

	sum, err := f.orchestrator().Run(ctx)
	if err == nil {
		t.Fatal("Run() err = nil, want the reconcile failure surfaced despite cancellation")
	}
	for _, want := range []string{"interrupted", "could not be reconciled", "left dirty"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Run() err = %v, want it to contain %q", err, want)
		}
	}
	// A dirty tree is a hard failure, not a soft interruption: the summary must
	// record it distinctly rather than reading as a clean stop.
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error (a dirty tree must not read as a clean interruption)", sum.Termination)
	}
	if sum.Error == "" {
		t.Error("summary error empty; the reconcile failure must be recorded")
	}
	// Nothing was committed, and once the lock is cleared the interrupted edits
	// are still in the tree (unreconciled), as the error claims.
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (nothing committed on interruption)", got)
	}
	os.Remove(lock)
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) == "" {
		t.Error("working tree is clean; the interrupted edits should have been left dirty")
	}
}

// A cancellation that lands DURING finalizeFix's commit -- after a successful
// coder's edits have been staged -- must not leave the tree dirty under a
// softened "clean interruption". finalizeFix's pre-commit ctx check is a TOCTOU
// guard; the commit error is routed through the interrupt reconciliation on a
// fresh context, so the outcome is either a stashed clean tree (soft
// interruption) or a hard dirty-tree error -- never a clean interruption over a
// dirty tree. Hooks are disabled, so the commit is wedged via a signer that
// blocks; the leftover index.lock from the killed commit may make the
// reconciling stash fail, which is the allowed hard-error branch.
func TestRunInterruptedDuringFinalizeCommit(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder edits the repo and returns a valid "fixed" verdict, so
	// finalizeFix proceeds to commit; a forced-signing signer that signals
	// readiness then blocks wedges `git commit` after `git add` has staged.
	ready := filepath.Join(f.respDir, "commit-started")
	signer := filepath.Join(f.respDir, "signer.sh")
	if err := os.WriteFile(signer, []byte(fmt.Sprintf("#!/bin/sh\ntouch '%s'\nsleep 60\n", ready)), 0o700); err != nil {
		t.Fatal(err)
	}
	f.writeSide(fmt.Sprintf("echo 'partial edit' >> '%s'\ngit -C '%s' config commit.gpgsign true\ngit -C '%s' config gpg.program '%s'\n",
		filepath.Join(f.repo, "main.go"), f.repo, f.repo, signer))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()

	sum, err := f.orchestrator().Run(ctx)

	// Nothing may have been committed, whichever branch is taken.
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (a canceled commit must not land)", got)
	}
	// A SIGKILLed `git commit` can leave an index.lock behind; clear it so the
	// working-tree state can be inspected.
	os.Remove(filepath.Join(f.repo, ".git", "index.lock"))
	dirty := strings.TrimSpace(gitRun(t, f.repo, "status", "--porcelain")) != ""

	if sum.Termination == model.TermInterrupted {
		// Soft interruption: reconciliation stashed the edits, so err is nil and
		// the tree is clean. A dirty tree here is the exact regression.
		if err != nil {
			t.Errorf("Run() err = %v, want nil for a soft interruption", err)
		}
		if dirty {
			t.Error("tree left dirty under a clean interruption; the commit error bypassed reconciliation")
		}
	} else {
		// The only other allowed outcome: reconciliation failed and surfaced as a
		// hard dirty-tree error rather than a softened interruption.
		if err == nil {
			t.Fatalf("termination = %q with nil error; want interruption or a hard reconcile error", sum.Termination)
		}
		if sum.Termination != model.TermError {
			t.Errorf("termination = %q, want error when the tree could not be reconciled", sum.Termination)
		}
	}
}

// r5.8: salvagePartialFix must route a cancellation that lands DURING the
// salvage commit through reconcileInterrupt (a fresh context) rather than
// StashDirty on the already-canceled ctx, so a dirty tree is never softened into
// a clean interruption. This is the salvage-path twin of
// TestRunInterruptedDuringFinalizeCommit: entered via a parse failure (no <fix>
// envelope, ctx still live) -> salvage, whose commit is then wedged and canceled.
func TestRunInterruptedDuringSalvageCommit(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder edits the repo but returns no <fix> envelope (parse failure ->
	// salvage). A forced-signing signer that signals readiness then blocks wedges
	// the salvage `git commit` after `git add` has staged, so cancellation lands
	// mid-commit.
	ready := filepath.Join(f.respDir, "salvage-commit-started")
	signer := filepath.Join(f.respDir, "signer.sh")
	if err := os.WriteFile(signer, []byte(fmt.Sprintf("#!/bin/sh\ntouch '%s'\nsleep 60\n", ready)), 0o700); err != nil {
		t.Fatal(err)
	}
	f.writeSide(fmt.Sprintf("echo 'partial edit' >> '%s'\ngit -C '%s' config commit.gpgsign true\ngit -C '%s' config gpg.program '%s'\n",
		filepath.Join(f.repo, "main.go"), f.repo, f.repo, signer))
	f.respond(2, "edited files but no <fix> envelope") // parse failure -> salvage

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()

	sum, err := f.orchestrator().Run(ctx)

	// No salvage commit may have landed, whichever branch is taken.
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (a canceled salvage commit must not land)", got)
	}
	// A SIGKILLed `git commit` can leave an index.lock behind; clear it so the
	// working-tree state can be inspected.
	os.Remove(filepath.Join(f.repo, ".git", "index.lock"))
	dirty := strings.TrimSpace(gitRun(t, f.repo, "status", "--porcelain")) != ""

	if sum.Termination == model.TermInterrupted {
		// Soft interruption: reconciliation stashed the edits via a fresh context,
		// so err is nil and the tree is clean. A dirty tree here is the exact
		// regression -- a StashDirty on the canceled ctx softened over dirt.
		if err != nil {
			t.Errorf("Run() err = %v, want nil for a soft interruption", err)
		}
		if dirty {
			t.Error("tree left dirty under a clean interruption; the salvage commit error bypassed reconciliation")
		}
	} else {
		// The only other allowed outcome: reconciliation failed and surfaced as a
		// hard dirty-tree error rather than a softened interruption.
		if err == nil {
			t.Fatalf("termination = %q with nil error; want interruption or a hard reconcile error", sum.Termination)
		}
		if sum.Termination != model.TermError {
			t.Errorf("termination = %q, want error when the tree could not be reconciled", sum.Termination)
		}
	}
}

// Cancellation can land between a caller's ctx.Err() check and the stash that
// follows it, so every reconciliation path goes through stashForReconcile. It
// must stash on a FRESH context (the run's is canceled and would fail each git
// operation instantly, leaving the tree dirty), and must flag a stash failure
// with errInterruptedTreeDirty -- otherwise recordRunError softens it into a
// clean interruption, which in the CLOSING round leaves the loop's success
// termination and its exit code 0 standing over uncommitted edits.
func TestStashForReconcileUnderCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	t.Run("stashes on a fresh context", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		if err := os.WriteFile(filepath.Join(f.repo, "main.go"), []byte("edited\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		stashed, err := f.orchestrator().stashForReconcile(ctx, "fixpoint: test reconcile")
		if err != nil {
			t.Fatalf("stashForReconcile on a canceled ctx = %v, want the stash to run anyway", err)
		}
		if !stashed {
			t.Error("stashed = false; the dirty tree was left unreconciled")
		}
		if got := strings.TrimSpace(gitRun(t, f.repo, "status", "--porcelain")); got != "" {
			t.Errorf("tree still dirty after reconciliation: %q", got)
		}
	})

	t.Run("a failed stash is a hard failure, not a clean interruption", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		f.cfg.Target.Path = t.TempDir() // not a git repository: every stash step fails
		_, err := f.orchestrator().stashForReconcile(ctx, "fixpoint: test reconcile")
		if err == nil {
			t.Fatal("stashForReconcile on a non-repository = nil, want the stash failure")
		}
		// The sentinel is only meaningful through recordRunError: assert the outcome
		// it exists to force, on the closing round's shape (a success already set).
		sum := &model.RunSummary{Termination: model.TermConverged}
		got := recordRunError(ctx, sum, fmt.Errorf("closing round: %w", err))
		if got == nil {
			t.Error("Run would return nil for a canceled round that left the tree dirty")
		}
		if sum.Termination != model.TermError {
			t.Errorf("termination = %q, want %q: a dirty tree must not be reported as a converged run", sum.Termination, model.TermError)
		}
	})
}

// requireCleanTree is true for a PR run even when review-only: gh pr checkout
// can preserve unrelated tracked edits and untracked files, which Collect would
// otherwise fold into the PR diff and misattribute to the PR. A dirty tree must
// therefore be rejected both before and after PR preparation, while a
// directory review-only run keeps its dirty-tree exemption.
func TestRunReviewOnlyCleanTree(t *testing.T) {
	t.Run("pr review-only rejects an initially dirty tree", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Target.Mode = "pr"
		f.cfg.Target.PR = 7
		if err := os.WriteFile(filepath.Join(f.repo, "dirty.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := f.orchestrator().Run(t.Context())
		if err == nil || !strings.Contains(err.Error(), "dirty") {
			t.Fatalf("Run() err = %v, want dirty-tree refusal for a pr review-only run", err)
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (must refuse before spending tokens)", got)
		}
	})

	t.Run("pr review-only rejects a dirty checkout after prepare", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Target.Mode = "pr"
		f.cfg.Target.PR = 7

		// A feature branch to check out, plus the base commit oid gh will report.
		gitRun(t, f.repo, "branch", "-M", "main")
		mainSHA := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "HEAD"))
		gitRun(t, f.repo, "checkout", "-q", "-b", "feature")
		if err := os.WriteFile(filepath.Join(f.repo, "main.go"), []byte("package main\n\nfunc pr() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitRun(t, f.repo, "commit", "-aqm", "pr change")
		gitRun(t, f.repo, "checkout", "-q", "main")

		// Stub gh: checkout switches to the PR branch AND leaves an untracked file
		// behind (standing in for a checkout hook or pre-existing artifact).
		binDir := t.TempDir()
		stub := "#!/bin/sh\n" +
			`case "$1 $2" in` + "\n" +
			`"pr checkout") git checkout -q feature; echo dirt > "` + filepath.Join(f.repo, "leftover.txt") + `" ;;` + "\n" +
			`"pr view") echo ` + mainSHA + " ;;\n" +
			`*) echo "unexpected gh call: $@" >&2; exit 1 ;;` + "\n" +
			"esac\n"
		if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(stub), 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		_, err := f.orchestrator().Run(t.Context())
		if err == nil || !strings.Contains(err.Error(), "dirty after preparing") {
			t.Fatalf("Run() err = %v, want post-prepare dirty-tree error for a pr review-only run", err)
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (must refuse before running any reviewer)", got)
		}
	})

	t.Run("directory review-only keeps its dirty-tree exemption", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, ReviewOnly: true})
		// Directory mode (fixture default) review-only: a dirty tree is allowed.
		if err := os.WriteFile(filepath.Join(f.repo, "dirty.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		f.respond(1, reviewResponse(t, aFinding("bug")))
		sum, err := f.orchestrator().Run(t.Context())
		if err != nil {
			t.Fatalf("Run() err = %v, want a directory review-only run to proceed over a dirty tree", err)
		}
		if sum.Termination != model.TermReviewOnly {
			t.Fatalf("termination = %q, want review-only", sum.Termination)
		}
		if got := f.invocations(); got != 1 {
			t.Errorf("agent invocations = %d, want 1 (the review must run)", got)
		}
	})
}

// The runRound backstop must fail a fix round that ends up with zero reviewer
// assignments, even if config validation is bypassed: an empty review reports
// no findings and no errors, which would otherwise read as a clean round and
// let the loop converge on an unverified fix. Only a once=true lens leaves
// round 2+ unassigned.
func TestRunRoundNoReviewerBackstop(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 2})
	f.cfg.Roles.Review.Prompts[0].Once = true // runs round 1 only; round 2 is unassigned
	f.respond(1, reviewResponse(t))           // round 1: clean (streak 1/2), does not converge

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "no reviewer assignments") {
		t.Fatalf("Run() err = %v, want the no-reviewer backstop error in round 2", err)
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error (must not advance the clean streak or converge)", sum.Termination)
	}
	// Round 1 was recorded; round 2 hits the backstop before its record is
	// appended, so it never counts toward the clean streak.
	if len(sum.Rounds) != 1 {
		t.Fatalf("rounds = %d, want 1 (round 2 aborts before recording)", len(sum.Rounds))
	}
}

// A mixed round (some findings fixed, some rejected) must write BOTH audit
// sections into the commit body: an all-rejected round makes no commit and an
// all-fixed round omits the Rejected section, so only a mixed round exercises
// the Rejected-section formatting.
func TestCommitBodyIncludesFixedAndRejectedSections(t *testing.T) {
	// The Fixed/Rejected body describes a ROUND, so it is what a squashing policy
	// produces. Under per_fix each commit is one issue and needs no sections.
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerRound})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 1, Title: "real bug"},
		model.ReviewFinding{Category: "style", Severity: "low", File: "main.go", Line: 2, Title: "a nit"},
	))
	f.editRepoOn(2) // the coder makes a real edit for the fixed finding
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched the bug"}))
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "rejected", Detail: "intentional by design"}))
	f.respond(4, reviewResponse(t)) // round 2: clean -> converge

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	r1 := sum.Rounds[0]
	if r1.Fixed != 1 || r1.Rejected != 1 {
		t.Fatalf("round 1 fixed=%d rejected=%d, want 1/1", r1.Fixed, r1.Rejected)
	}
	body := gitRun(t, f.repo, "log", "-1", "--format=%b", r1.CommitSHA)
	for _, want := range []string{"Fixed:", "real bug", "patched the bug", "Rejected:", "a nit", "intentional by design"} {
		if !strings.Contains(body, want) {
			t.Errorf("commit body missing %q:\n%s", want, body)
		}
	}
	if fi, ri := strings.Index(body, "Fixed:"), strings.Index(body, "Rejected:"); fi < 0 || ri < 0 || fi > ri {
		t.Errorf("Fixed section must precede Rejected section:\n%s", body)
	}
}

// warnArgModePrompts is the runtime mitigation for process-list prompt
// exposure. It must warn once per distinct active arg-mode agent, name both the
// exposure and the stdin alternative, exclude stdin agents and unused pool
// entries, and (in review-only mode) exclude the coder.
func TestWarnArgModePrompts(t *testing.T) {
	argMode := func() config.Agent {
		return config.Agent{Command: []string{"x"}, PromptVia: "arg", Timeout: config.Duration(time.Minute), CanEdit: true}
	}
	// collect builds an orchestrator from f with a capturing logf, runs
	// warnArgModePrompts, and returns the set of agents named in arg-mode
	// warnings plus the number of such warnings.
	collect := func(t *testing.T, f *fixture) (map[string]bool, int) {
		t.Helper()
		var lines []string
		logf := func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) }
		o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, logf)
		if err != nil {
			t.Fatal(err)
		}
		o.warnArgModePrompts()
		warned := map[string]bool{}
		n := 0
		for _, l := range lines {
			if !strings.Contains(l, "uses prompt_via: arg") {
				continue
			}
			n++
			if !strings.Contains(l, "ps / /proc") || !strings.Contains(l, "prompt_via: stdin") {
				t.Errorf("arg-mode warning missing exposure/alternative wording: %q", l)
			}
			for name := range f.cfg.Agents {
				if strings.Contains(l, `"`+name+`"`) {
					warned[name] = true
				}
			}
		}
		return warned, n
	}

	t.Run("fix run warns active arg agents once each, excluding stdin and unused", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		reviewPrompt := f.cfg.Roles.Review.Prompts[0].Prompt
		f.cfg.Agents = map[string]config.Agent{
			"coder":     argMode(),
			"rev-arg1":  argMode(),
			"rev-arg2":  argMode(),
			"rev-stdin": {Command: []string{"x"}, PromptVia: "stdin", Timeout: config.Duration(time.Minute), CanEdit: true},
			"unused":    argMode(), // inert under fixed strategy
		}
		f.cfg.Roles.Coder.Agent = "coder"
		f.cfg.Roles.Review.Strategy = "fixed"
		f.cfg.Roles.Review.Agents = []string{"unused"}
		f.cfg.Roles.Review.Prompts = []config.ReviewLens{
			{Agent: "rev-arg1", Prompt: reviewPrompt},
			{Agent: "rev-arg2", Prompt: reviewPrompt},
			{Agent: "rev-stdin", Prompt: reviewPrompt},
		}

		warned, n := collect(t, f)
		if n != 3 {
			t.Errorf("arg-mode warnings = %d, want 3 (one per distinct active arg agent)", n)
		}
		want := map[string]bool{"coder": true, "rev-arg1": true, "rev-arg2": true}
		if fmt.Sprint(warned) != fmt.Sprint(want) {
			t.Errorf("warned agents = %v, want exactly %v (stdin and unused agents excluded)", warned, want)
		}
	})

	t.Run("review-only excludes the coder", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		reviewPrompt := f.cfg.Roles.Review.Prompts[0].Prompt
		f.cfg.Agents = map[string]config.Agent{
			"coder":   argMode(),
			"rev-arg": argMode(),
		}
		f.cfg.Roles.Coder.Agent = "coder"
		f.cfg.Roles.Review.Strategy = "fixed"
		f.cfg.Roles.Review.Prompts = []config.ReviewLens{{Agent: "rev-arg", Prompt: reviewPrompt}}

		warned, n := collect(t, f)
		if n != 1 {
			t.Errorf("arg-mode warnings = %d, want 1 (only the active reviewer)", n)
		}
		if warned["coder"] {
			t.Error("coder warned in review-only mode; it never runs there")
		}
		if !warned["rev-arg"] {
			t.Error("active arg-mode reviewer was not warned")
		}
	})
}

// config.Validate refuses an agent command that resolves into a pr-mode target;
// with trust asserted the run proceeds, and warnTargetSuppliedCommand is the only
// place the operator learns the assertion also accepted PR-authored code running
// AS the agent process. It must fire only in pr mode, and only for a command
// element inside the target.
func TestWarnTargetSuppliedCommand(t *testing.T) {
	collect := func(t *testing.T, f *fixture) []string {
		t.Helper()
		var lines []string
		logf := func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) }
		o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, logf)
		if err != nil {
			t.Fatal(err)
		}
		o.warnTargetSuppliedCommand()
		var warnings []string
		for _, l := range lines {
			if strings.Contains(l, "inside target") {
				warnings = append(warnings, l)
			}
		}
		return warnings
	}
	// targetLocal points the mock agent at a script inside the target, spelled
	// relative to it -- the form `gh pr checkout` gets to replace.
	targetLocal := func(t *testing.T, f *fixture) {
		t.Helper()
		script := testfixture.WriteMockScript(t, f.respDir)
		data, err := os.ReadFile(script)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.repo, "reviewer.sh"), data, 0o700); err != nil {
			t.Fatal(err)
		}
		a := f.cfg.Agents["mock"]
		a.Command = []string{"./reviewer.sh"}
		f.cfg.Agents["mock"] = a
	}

	t.Run("pr mode warns for a target-supplied command", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Target.Mode = config.ModePR
		f.cfg.Target.PR = 7
		targetLocal(t, f)
		warnings := collect(t, f)
		if len(warnings) != 1 {
			t.Fatalf("warnings = %v, want exactly one", warnings)
		}
		for _, want := range []string{`"mock"`, "./reviewer.sh", "gh pr checkout", "-trusted-target"} {
			if !strings.Contains(warnings[0], want) {
				t.Errorf("warning missing %q: %q", want, warnings[0])
			}
		}
	})

	t.Run("directory mode does not warn", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		targetLocal(t, f)
		if warnings := collect(t, f); len(warnings) != 0 {
			t.Errorf("warnings = %v, want none outside mode pr (no checkout swaps the file)", warnings)
		}
	})

	t.Run("pr mode with a command outside the target does not warn", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Target.Mode = config.ModePR
		f.cfg.Target.PR = 7
		if warnings := collect(t, f); len(warnings) != 0 {
			t.Errorf("warnings = %v, want none for a command outside the target", warnings)
		}
	})
}

// Startup validation must check every (prompt, role) pairing: a file shared by
// both roles gets rendered with both data types, so a fix-only placeholder in
// a shared template fails at New, not mid-run in the review round.
func TestNewValidatesCrossRolePlaceholders(t *testing.T) {
	writePrompt := func(t *testing.T, content string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "prompt.md")
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("fix-only placeholder in a review prompt", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		f.cfg.Roles.Review.Prompts[0].Prompt = writePrompt(t, "{{.Findings}}")
		if _, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, t.Logf); err == nil {
			t.Fatal("New() = nil, want render error for fix-only .Findings in a review prompt")
		}
	})

	t.Run("review-only placeholder in the coder prompt", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		f.cfg.Roles.Coder.Prompt = writePrompt(t, "{{.Target}}")
		if _, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, t.Logf); err == nil {
			t.Fatal("New() = nil, want render error for review-only .Target in the coder prompt")
		}
	})

	t.Run("shared prompt is validated for both roles", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		shared := writePrompt(t, "{{.Findings}}") // valid for coder, invalid for review
		f.cfg.Roles.Coder.Prompt = shared
		f.cfg.Roles.Review.Prompts[0].Prompt = shared
		if _, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, t.Logf); err == nil {
			t.Fatal("New() = nil, want rejection of a fix-only placeholder in a prompt also used for review")
		}
	})

	t.Run("shared role-neutral prompt is accepted", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		shared := writePrompt(t, "Round {{.Round}}\n{{.OutputContract}}")
		f.cfg.Roles.Coder.Prompt = shared
		f.cfg.Roles.Review.Prompts[0].Prompt = shared
		if _, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}}, t.Logf); err != nil {
			t.Fatalf("New() = %v, want nil for a prompt valid in both roles", err)
		}
	})
}

// ---- Ping ---------------------------------------------------------------------

func TestPing(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.respond(1, "OK")
	if err := f.orchestrator().Ping(t.Context()); err != nil {
		t.Fatalf("Ping() = %v, want nil", err)
	}
}

// pingTimeoutFor must cap only the timeouts that exceed pingTimeout, so a hung
// or default-timeout agent cannot stall preflight for its full run timeout on
// each retry, while an already-tight timeout is preserved unchanged.
func TestPingTimeoutFor(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want time.Duration
	}{
		{"above cap is capped", pingTimeout + time.Minute, pingTimeout},
		{"default 10m is capped", 10 * time.Minute, pingTimeout},
		{"equal to cap is preserved", pingTimeout, pingTimeout},
		{"below cap is preserved", time.Minute, time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pingTimeoutFor(config.Agent{Timeout: config.Duration(tc.in)}).Std()
			if got != tc.want {
				t.Errorf("pingTimeoutFor(%s) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestPingFailure(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	// No responses registered: the mock exits non-zero on both attempts.
	err := f.orchestrator().Ping(t.Context())
	if err == nil || !strings.Contains(err.Error(), "mock") {
		t.Fatalf("Ping() = %v, want failure naming the agent", err)
	}
	if got := f.invocations(); got != 2 {
		t.Errorf("agent invocations = %d, want 2 (one retry)", got)
	}
}

// The retry exists so a transient cold-start failure does not abort a wanted
// run: the first attempt fails (empty output), the second succeeds, and Ping
// must then return nil after exactly two invocations.
func TestPingRetrySucceeds(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.respond(1, "")   // first attempt: empty output -> transient failure
	f.respond(2, "OK") // retry: succeeds
	if err := f.orchestrator().Ping(t.Context()); err != nil {
		t.Fatalf("Ping() = %v, want nil (the retry must succeed)", err)
	}
	if got := f.invocations(); got != 2 {
		t.Errorf("agent invocations = %d, want 2 (first fails, retry succeeds)", got)
	}
}

// An agent that exits 0 but prints nothing is a broken/hung CLI, not a healthy
// agent: Ping must convert the empty stdout into a failure and retry, matching
// the non-zero-exit failure contract.
func TestPingEmptyOutputFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	// Both attempts exit 0 with empty stdout, so only the empty-output guard
	// (not a non-zero exit) can turn this into a failure.
	f.respond(1, "")
	f.respond(2, "")
	err := f.orchestrator().Ping(t.Context())
	if err == nil || !strings.Contains(err.Error(), "mock") {
		t.Fatalf("Ping() = %v, want failure naming the agent for empty output", err)
	}
	if got := f.invocations(); got != 2 {
		t.Errorf("agent invocations = %d, want 2 (one retry)", got)
	}
}

// Ping must target exactly the agents the run will use: pinned reviewers, the
// pool only under rotate/all, and the coder only when fix rounds will happen.
func TestPingAgentSelection(t *testing.T) {
	// probe registers a healthy agent that records being pinged by touching a
	// per-name marker file.
	probe := func(t *testing.T, f *fixture, dir, name string) {
		t.Helper()
		p := filepath.Join(dir, name+".sh")
		body := fmt.Sprintf("#!/bin/sh\ncat > /dev/null\ntouch '%s'\necho OK\n", filepath.Join(dir, name+".pinged"))
		if err := os.WriteFile(p, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
		f.cfg.Agents[name] = config.Agent{Command: []string{p}, PromptVia: "stdin", Timeout: config.Duration(time.Minute), CanEdit: true}
	}
	pinged := func(dir, name string) bool {
		_, err := os.Stat(filepath.Join(dir, name+".pinged"))
		return err == nil
	}

	t.Run("fixed strategy ignores an unhealthy unused pool entry", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		dir := t.TempDir()
		probe(t, f, dir, "coder")
		probe(t, f, dir, "reviewer")
		f.cfg.Agents["broken"] = config.Agent{Command: []string{"false"}, PromptVia: "stdin", Timeout: config.Duration(time.Minute)}
		f.cfg.Roles.Coder.Agent = "coder"
		f.cfg.Roles.Review.Strategy = "fixed"
		f.cfg.Roles.Review.Agents = []string{"broken"} // inert under fixed
		f.cfg.Roles.Review.Prompts[0].Agent = "reviewer"
		if err := f.orchestrator().Ping(t.Context()); err != nil {
			t.Fatalf("Ping() = %v, want nil (the unused pool entry must not be pinged)", err)
		}
		if !pinged(dir, "coder") || !pinged(dir, "reviewer") {
			t.Errorf("coder pinged=%v reviewer pinged=%v, want both", pinged(dir, "coder"), pinged(dir, "reviewer"))
		}
	})

	t.Run("rotate strategy pings the pool", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		dir := t.TempDir()
		probe(t, f, dir, "coder")
		probe(t, f, dir, "pool")
		f.cfg.Roles.Coder.Agent = "coder"
		f.cfg.Roles.Review.Strategy = "rotate"
		f.cfg.Roles.Review.Agents = []string{"pool"}
		f.cfg.Roles.Review.Prompts[0].Agent = "" // unpinned: served by the pool
		if err := f.orchestrator().Ping(t.Context()); err != nil {
			t.Fatalf("Ping() = %v, want nil", err)
		}
		if !pinged(dir, "pool") {
			t.Error("pool agent was not pinged under rotate")
		}
		if !pinged(dir, "coder") {
			t.Error("coder was not pinged for a fix run")
		}
	})

	t.Run("review-only does not ping the coder", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		dir := t.TempDir()
		probe(t, f, dir, "coder")
		probe(t, f, dir, "reviewer")
		f.cfg.Roles.Coder.Agent = "coder"
		f.cfg.Roles.Review.Prompts[0].Agent = "reviewer"
		if err := f.orchestrator().Ping(t.Context()); err != nil {
			t.Fatalf("Ping() = %v, want nil", err)
		}
		if pinged(dir, "coder") {
			t.Error("coder was pinged in review-only mode")
		}
		if !pinged(dir, "reviewer") {
			t.Error("reviewer was not pinged")
		}
	})
}

// ---- unit tests for the loop's pure pieces -------------------------------------

// applyVerdicts validates the coder's result set against the round's ISSUES and
// applies nothing unless the whole set is valid -- the run summary is always
// written and must never carry a partially applied response.
func TestApplyVerdicts(t *testing.T) {
	setup := func(t *testing.T) (*Orchestrator, *model.RoundRecord, string, string) {
		t.Helper()
		f := newFixture(t, config.Loop{})
		o := f.orchestrator()
		rec := &model.RoundRecord{Round: 1, Findings: []model.Finding{
			{ID: "r1.1", Severity: "high", File: "a.go", Line: 1, Title: "one"},
			{ID: "r1.2", Severity: "low", File: "b.go", Line: 1, Title: "two"},
		}}
		rec.Issues = o.ledger.Absorb(1, rec.Findings)
		if len(rec.Issues) != 2 {
			t.Fatalf("fixture produced %d issues, want 2", len(rec.Issues))
		}
		return o, rec, rec.Issues[0].ID, rec.Issues[1].ID
	}

	t.Run("valid set applies and mirrors onto observations", func(t *testing.T) {
		o, rec, id1, id2 := setup(t)
		if err := o.applyVerdicts(rec, []model.FixResult{
			{ID: id1, Verdict: "fixed", Detail: "d"},
			{ID: id2, Verdict: "rejected", Detail: "d"},
		}, rec.Issues); err != nil {
			t.Fatalf("applyVerdicts() = %v", err)
		}
		if rec.Fixed != 1 || rec.Rejected != 1 {
			t.Errorf("fixed=%d rejected=%d, want 1/1", rec.Fixed, rec.Rejected)
		}
		// The verdict must reach the observations too, so history and the summary
		// keep speaking in the terms the reviewers used.
		for _, f := range rec.Findings {
			if f.Verdict == "" {
				t.Errorf("verdict was not mirrored onto observation %s", f.ID)
			}
		}
	})

	cases := []struct {
		name    string
		results func(id1, id2 string) []model.FixResult
		wantErr string
	}{
		{"unknown id", func(id1, _ string) []model.FixResult {
			return []model.FixResult{{ID: id1, Verdict: "fixed"}, {ID: "i999", Verdict: "fixed"}}
		}, "unknown issue id"},
		{"duplicate id", func(id1, _ string) []model.FixResult {
			return []model.FixResult{{ID: id1, Verdict: "fixed"}, {ID: id1, Verdict: "rejected"}}
		}, "more than once"},
		{"invalid verdict", func(id1, id2 string) []model.FixResult {
			return []model.FixResult{{ID: id1, Verdict: "maybe"}, {ID: id2, Verdict: "fixed"}}
		}, "unknown verdict"},
		{"missing issue", func(id1, _ string) []model.FixResult {
			return []model.FixResult{{ID: id1, Verdict: "fixed"}}
		}, "did not give a verdict"},
		{"empty result set", func(string, string) []model.FixResult { return nil }, "did not give a verdict"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, rec, id1, id2 := setup(t)
			err := o.applyVerdicts(rec, tc.results(id1, id2), rec.Issues)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("applyVerdicts() = %v, want error containing %q", err, tc.wantErr)
			}
			if rec.Fixed != 0 || rec.Rejected != 0 {
				t.Errorf("failed validation mutated counts: fixed=%d rejected=%d", rec.Fixed, rec.Rejected)
			}
			for _, it := range rec.Issues {
				if it.Verdict != "" {
					t.Errorf("failed validation mutated issue %s: %+v", it.ID, it)
				}
			}
		})
	}
}

func TestAssignments(t *testing.T) {
	newO := func(strategy config.Strategy) *Orchestrator {
		return &Orchestrator{cfg: &config.Config{Roles: config.Roles{Review: config.Review{
			Strategy: strategy,
			Agents:   []string{"a1", "a2"},
			Prompts: []config.ReviewLens{
				{Prompt: "pinned.md", Agent: "x", Advisory: true},
				{Prompt: "b.md"},
				{Prompt: "c.md"},
			},
		}}}}
	}

	t.Run("rotate cycles unpinned lenses across rounds", func(t *testing.T) {
		o := newO("rotate")
		r1 := o.assignments(1)
		want1 := []model.Assignment{
			{Lens: "pinned.md", Agent: "x", Advisory: true, Pinned: true},
			{Lens: "b.md", Agent: "a1"},
			{Lens: "c.md", Agent: "a2"},
		}
		if fmt.Sprint(r1) != fmt.Sprint(want1) {
			t.Errorf("round 1 = %v, want %v", r1, want1)
		}
		r2 := o.assignments(2)
		if r2[1].Agent != "a2" || r2[2].Agent != "a1" {
			t.Errorf("round 2 should rotate: %v", r2)
		}
		if r2[0].Agent != "x" {
			t.Errorf("pinned lens must not rotate: %v", r2[0])
		}
	})

	t.Run("once lenses run in round 1 only", func(t *testing.T) {
		o := newO("rotate")
		o.cfg.Roles.Review.Prompts[0].Once = true // the pinned advisory lens
		r1 := o.assignments(1)
		if len(r1) != 3 || r1[0].Lens != "pinned.md" {
			t.Fatalf("round 1 must include the once lens: %v", r1)
		}
		r2 := o.assignments(2)
		if len(r2) != 2 {
			t.Fatalf("round 2 must skip the once lens: %v", r2)
		}
		for _, a := range r2 {
			if a.Lens == "pinned.md" {
				t.Errorf("once lens ran again in round 2: %v", r2)
			}
		}
	})

	t.Run("skipping an unpinned once lens does not shift later lenses' rotation", func(t *testing.T) {
		// An UNPINNED once:true lens ahead of recurring unpinned lenses. Its
		// rotation index must stay reserved after round 1 so the recurring lenses
		// keep rotating; otherwise they'd re-select round 1's agent, defeating
		// rotation (and letting a clean-round confirmation reuse the reviewer).
		o := &Orchestrator{cfg: &config.Config{Roles: config.Roles{Review: config.Review{
			Strategy: "rotate",
			Agents:   []string{"a1", "a2"},
			Prompts: []config.ReviewLens{
				{Prompt: "once.md", Once: true},
				{Prompt: "b.md"},
				{Prompt: "c.md"},
			},
		}}}}
		r1 := o.assignments(1)
		r2 := o.assignments(2)
		agentFor := func(as []model.Assignment, lens string) string {
			for _, a := range as {
				if a.Lens == lens {
					return a.Agent
				}
			}
			return ""
		}
		if agentFor(r2, "once.md") != "" {
			t.Errorf("once lens must not run in round 2: %v", r2)
		}
		for _, lens := range []string{"b.md", "c.md"} {
			if got1, got2 := agentFor(r1, lens), agentFor(r2, lens); got1 == got2 {
				t.Errorf("%s must rotate to a new agent in round 2 (round1=%s, round2=%s); the skipped once lens shifted its index", lens, got1, got2)
			}
		}
	})

	t.Run("all fans every unpinned lens across the pool", func(t *testing.T) {
		got := newO("all").assignments(1)
		// Assert the exact composition, not just the count: each unpinned lens
		// must pair with every pool agent, plus the pinned lens on its agent.
		set := map[string]bool{}
		for _, a := range got {
			set[a.Lens+"->"+a.Agent] = true
		}
		want := []string{
			"pinned.md->x", // pinned lens keeps its agent
			"b.md->a1", "b.md->a2",
			"c.md->a1", "c.md->a2",
		}
		if len(got) != len(want) {
			t.Fatalf("assignments = %d, want %d: %v", len(got), len(want), got)
		}
		for _, w := range want {
			if !set[w] {
				t.Errorf("missing assignment %q; got %v", w, got)
			}
		}
	})

	// A review-only run runs its final lenses in this one round (there is no closing
	// round to hold them). They must fan out exactly as the closing round would:
	// an unpinned final lens on every eligible agent regardless of the review
	// strategy, a pinned one still on its own agent. Letting rotate pick a single
	// agent makes a review-only config a subset of its fix sibling, not a preview.
	t.Run("review-only fans an unpinned final lens across the pool under rotate", func(t *testing.T) {
		o := newO("rotate")
		o.cfg.Loop.ReviewOnly = true
		o.cfg.Roles.Review.Prompts = append(o.cfg.Roles.Review.Prompts,
			config.ReviewLens{Prompt: "final.md", Final: true},
			config.ReviewLens{Prompt: "pinnedfinal.md", Agent: "a2", Final: true},
		)
		got := o.assignments(1)
		set := map[string]bool{}
		for _, a := range got {
			set[a.Lens+"->"+a.Agent] = true
		}
		want := []string{
			"pinned.md->x",
			"b.md->a1", "c.md->a2",
			"final.md->a1", "final.md->a2", // unpinned final: the whole pool
			"pinnedfinal.md->a2", // pinned final: its agent only
		}
		if len(got) != len(want) {
			t.Fatalf("assignments = %d, want %d: %v", len(got), len(want), got)
		}
		for _, w := range want {
			if !set[w] {
				t.Errorf("missing assignment %q; got %v", w, got)
			}
		}
	})
}

func TestValidateReviewFindings(t *testing.T) {
	cases := []struct {
		name    string
		in      []model.ReviewFinding
		wantErr string
	}{
		{"valid", []model.ReviewFinding{{Title: "t", Severity: "high"}}, ""},
		{"mixed case severity", []model.ReviewFinding{{Title: "t", Severity: "CRITICAL"}}, ""},
		{"none", nil, ""},
		{"empty title", []model.ReviewFinding{{Severity: "low"}}, "empty title"},
		{"bad severity", []model.ReviewFinding{{Title: "t", Severity: "moderate"}}, "invalid severity"},
		// The contract's placeholder severity must never read as a real value.
		{"placeholder severity", []model.ReviewFinding{{Title: "t", Severity: "critical|high|medium|low"}}, "invalid severity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateReviewFindings(tc.in)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("validateReviewFindings() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("validateReviewFindings() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

// If a reviewer echoes its prompt and emits no valid tagged output of its own,
// the injected contract example must not survive as a fabricated finding. The
// extraction layer catches it: the contract's own explanatory text follows its
// <review> example, so the block is not the last thing printed and ExtractJSON
// refuses it rather than accepting the echoed example (validation's placeholder-
// severity check remains a backstop, covered by validateReviewFindings tests).
func TestEchoedContractExampleRejected(t *testing.T) {
	echoed := "here is the format:\n" + prompt.ReviewContract + "\n(no real output followed)"
	var out model.ReviewOutput
	err := agent.ExtractJSON(echoed, "review", &out)
	if err == nil {
		t.Fatal("ExtractJSON accepted an echoed contract example with trailing prompt text; it must reject it as non-final")
	}
	if !strings.Contains(err.Error(), "not the last output") {
		t.Fatalf("ExtractJSON error = %v, want a non-final-block rejection", err)
	}
}

// Aging bounds how long the cap can starve an issue. Reproduces the pathology from
// a real five-round run: an unbounded generator of medium-severity findings (a
// coverage lens can always want more coverage) kept a one-line low-severity README
// error out of every fix round, so it was reported three times and never once
// scheduled. Each deferral promotes an issue one severity tier, so the wait is
// bounded by its distance from the generator's severity -- one round here.
//
// The deferral count is now exact, read from the ledger, where it used to be
// approximated from (file, category).
func TestDeferOverCapAgesDeferredIssues(t *testing.T) {
	f := newFixture(t, config.Loop{MaxFindingsPerRound: 2})
	o := f.orchestrator()

	// The starved issue is listed LAST so nothing depends on input order.
	round := func(n int) *model.RoundRecord {
		rec := &model.RoundRecord{Round: n, Findings: []model.Finding{
			{Severity: "medium", Category: "tests", File: fmt.Sprintf("a%d.go", n), Line: 1, Title: "untested"},
			{Severity: "medium", Category: "tests", File: fmt.Sprintf("b%d.go", n), Line: 1, Title: "untested"},
			{Severity: "low", Category: "maintainability", File: "README.md", Line: 7, Title: "doc error"},
		}}
		rec.Issues = o.ledger.Absorb(n, rec.Findings)
		return rec
	}
	deferredDoc := func(rec *model.RoundRecord) bool {
		for _, it := range rec.Issues {
			if it.File == "README.md" {
				return it.Verdict == model.VerdictDeferred
			}
		}
		t.Fatal("the README issue is missing from the round")
		return false
	}

	// Round 1: nothing has waited yet, so severity alone decides and the low issue
	// loses both slots to the fresh mediums.
	r1 := round(1)
	o.deferOverCap(r1)
	if !deferredDoc(r1) {
		t.Fatal("round 1: severity should decide the first round, deferring the low issue")
	}

	// Round 2: one deferral promotes it to the mediums' tier and the age tie-break
	// awards the slot to what already waited. Before aging it lost every round.
	r2 := round(2)
	o.deferOverCap(r2)
	if deferredDoc(r2) {
		t.Error("round 2: the aged issue was deferred again; the cap can starve it indefinitely")
	}
}

// Only unresolved issues age. A fixed or rejected issue is closed, so a later
// report of the same location starts from its own severity rather than inheriting
// priority from work already done.
func TestDeferOverCapDoesNotAgeResolvedIssues(t *testing.T) {
	f := newFixture(t, config.Loop{MaxFindingsPerRound: 1})
	o := f.orchestrator()

	r1 := &model.RoundRecord{Round: 1, Findings: []model.Finding{
		{Severity: "low", Category: "maintainability", File: "README.md", Line: 7, Title: "doc nit"},
	}}
	r1.Issues = o.ledger.Absorb(1, r1.Findings)
	// Resolved, not deferred: no aging credit.
	o.setIssueVerdict(r1, 0, model.VerdictFixed, "done")

	r2 := &model.RoundRecord{Round: 2, Findings: []model.Finding{
		{Severity: "low", Category: "maintainability", File: "README.md", Line: 7, Title: "doc nit again"},
		{Severity: "high", Category: "bugs", File: "b.go", Line: 1, Title: "real bug"},
	}}
	r2.Issues = o.ledger.Absorb(2, r2.Findings)
	o.deferOverCap(r2)

	for _, it := range r2.Issues {
		switch it.File {
		case "b.go":
			if it.Verdict == model.VerdictDeferred {
				t.Error("a high-severity issue lost its slot to a low whose predecessor was already fixed")
			}
		case "README.md":
			if it.Verdict != model.VerdictDeferred {
				t.Error("expected the low issue to be deferred; fixed history must not grant aging")
			}
		}
	}
}

func (f *fixture) verifyGate(policy config.VerifyPolicy, failWhenPresent string) {
	f.t.Helper()
	script := filepath.Join(f.t.TempDir(), "check.sh")
	body := "#!/bin/sh\nif [ -e " + filepath.Join(f.repo, failWhenPresent) + " ]; then echo 'check failed: build is broken'; exit 1; fi\nexit 0\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		f.t.Fatal(err)
	}
	f.cfg.Verify = config.Verify{
		Policy:   policy,
		Timeout:  config.Duration(time.Minute),
		Commands: []config.VerifyCommand{{Name: "build", Run: []string{script}}},
	}
}

// breakBuildOn makes the mock agent's n-th invocation create the marker file that
// the configured check fails on -- i.e. a coder whose "fix" breaks the build.
// Which invocation is the coder's depends on the test (the closing round's coder
// runs after an extra reviewer), so it is named rather than assumed.
func (f *fixture) breakBuildOn(n int, path string) {
	f.t.Helper()
	// A real edit AND the breakage, which is the realistic shape: the coder does
	// useful work and also breaks something. A correction that only removes the
	// breakage must therefore still leave something to commit.
	testfixture.WriteSide(f.t, f.respDir, n, fmt.Sprintf("#!/bin/sh\necho 'fix %d' >> '%s'\necho broken > '%s'\n",
		n, filepath.Join(f.repo, "main.go"), filepath.Join(f.repo, path)))
}

// repairBuildOn makes the n-th invocation remove that marker, i.e. a successful
// correction attempt.
func (f *fixture) repairBuildOn(n int, path string) {
	f.t.Helper()
	testfixture.WriteSide(f.t, f.respDir, n, fmt.Sprintf("#!/bin/sh\nrm -f '%s'\n", filepath.Join(f.repo, path)))
}

// A round whose edits fail the gate must NOT be committed, even though the coder
// reported a successful fix. Committing it would put later rounds on a broken
// base and let "converged" mean "converged on something that does not build".
func TestVerifyFailureDiscardsRound(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1})
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	before := f.commitCount()

	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The coder creates the file that makes the check fail, and claims success.
	f.breakBuildOn(2, "broken.txt")
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	// Invocation 3 is the correction attempt: it does not remove the file.
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "still done"}))

	_, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() = nil, want an error: a round failing verification must not be reported as success")
	}
	if !strings.Contains(err.Error(), "verification failed") {
		t.Errorf("error should say verification failed: %v", err)
	}
	if got := f.commitCount(); got != before {
		t.Errorf("commit count = %d, want %d: an unverified round must not be committed", got, before)
	}
	// The coder's work is stashed, not silently deleted.
	if out := gitRun(t, f.repo, "stash", "list"); !strings.Contains(out, "round 1") {
		t.Errorf("the discarded round's edits should be recoverable from the stash, got:\n%s", out)
	}
	if clean := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(clean) != "" {
		t.Errorf("the working tree must be restored, got:\n%s", clean)
	}
}

// The verdicts of a round the gate discarded must go with it. The coder reports
// "fixed" before the gate runs, so a round that never commits still leaves that
// verdict, its ledger status, its mirrored observations and the Fixed counter
// standing -- and the summary is written on the error path too, so the durable
// record and the scoreboard would both claim a fix that no commit contains while
// the defect sits untouched in the stash.
func TestVerifyFailureWithdrawsTheDiscardedFixVerdict(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1})
	f.verifyGate(config.VerifyMustPass, "broken.txt")

	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.breakBuildOn(2, "broken.txt")
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	// The correction does not remove the breakage, so the round is discarded.
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "still done"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() = nil, want an error: a round failing verification must not be reported as success")
	}
	if len(sum.Rounds) == 0 {
		t.Fatal("the summary must carry the failed round")
	}
	r1 := sum.Rounds[0]
	if r1.Fixed != 0 {
		t.Errorf("round 1 fixed = %d, want 0: the discarded fix is in the stash, not in a commit", r1.Fixed)
	}
	for _, it := range r1.Issues {
		if it.Verdict == model.VerdictFixed || it.StatusOrDefault() != model.StatusOpen {
			t.Errorf("issue %s = %q/%q, want it reopened", it.ID, it.Verdict, it.StatusOrDefault())
		}
	}
	for _, fnd := range r1.Findings {
		if fnd.Verdict == model.VerdictFixed {
			t.Errorf("observation %s still claims FIXED for a fix that was discarded", fnd.ID)
		}
	}
}

// The coder gets exactly one correction attempt, and a round that passes after it
// commits normally -- so a transient breakage does not throw away the round.
func TestVerifyRetrySucceedsAndCommits(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	before := f.commitCount()

	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.breakBuildOn(2, "broken.txt")
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	// The correction attempt removes the offending file, so the gate passes.
	f.repairBuildOn(3, "broken.txt")
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "corrected"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() = %v, want the corrected round to commit", err)
	}
	if got := f.commitCount(); got != before+1 {
		t.Errorf("commit count = %d, want %d: the corrected round should commit", got, before+1)
	}
	if len(sum.Rounds) == 0 || len(sum.Rounds[0].Verify) == 0 {
		t.Fatal("verification results must be recorded on the round for the summary")
	}
	// Both gate runs, each naming the fix and the occasion: the initial pass that
	// blocked and the one after the correction. Keeping only the last would leave the
	// summary unable to say which fix needed correcting.
	got := make([]string, 0, len(sum.Rounds[0].Verify))
	for _, run := range sum.Rounds[0].Verify {
		got = append(got, run.Issue+"/"+run.Attempt)
		if len(run.Results) == 0 {
			t.Errorf("gate run %+v recorded no check results", run)
		}
	}
	want := []string{"i1/" + model.VerifyAttemptInitial, "i1/" + model.VerifyAttemptCorrection}
	if !slices.Equal(got, want) {
		t.Errorf("recorded gate runs = %v, want %v", got, want)
	}
}

// A correction attempt that reverts the round's edits entirely leaves nothing to
// commit -- so the "fixed" verdicts the coder already reported claim work that no
// commit contains. They must be withdrawn: otherwise the summary, and the history
// the next round's reviewers read, both say FIXED (which tells them to verify the
// fix rather than re-report it) for a defect still sitting in the tree untouched.
func TestVerifyCorrectionRevertingEverythingReopensTheIssues(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1})
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	before := f.commitCount()

	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.breakBuildOn(2, "broken.txt")
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	// The correction throws away everything the round did -- the breakage and the
	// edit -- so the gate passes over a tree identical to the round's starting point.
	testfixture.WriteSide(f.t, f.respDir, 3, fmt.Sprintf("#!/bin/sh\nrm -f '%s'\ngit -C '%s' checkout -- .\n",
		filepath.Join(f.repo, "broken.txt"), f.repo))
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "reverted it all"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() = %v, want the reverted round to end the run without an error", err)
	}
	if got := f.commitCount(); got != before {
		t.Errorf("commit count = %d, want %d: a reverted round has nothing to commit", got, before)
	}
	r1 := sum.Rounds[0]
	if r1.Fixed != 0 {
		t.Errorf("round 1 fixed = %d, want 0: no fix landed", r1.Fixed)
	}
	for _, it := range r1.Issues {
		if it.Verdict == model.VerdictFixed || it.StatusOrDefault() != model.StatusOpen {
			t.Errorf("issue %s = %q/%q, want it reopened", it.ID, it.Verdict, it.StatusOrDefault())
		}
	}
	// The observations carry the verdict into the reviewers' history, so they have to
	// be withdrawn too -- an empty verdict renders as UNRESOLVED, which is what asks
	// for the re-report.
	for _, fnd := range r1.Findings {
		if fnd.Verdict == model.VerdictFixed {
			t.Errorf("observation %s still claims FIXED in the history handed to reviewers", fnd.ID)
		}
	}
}

// Reverting one fix must not withdraw the verdicts of fixes already committed in
// the same round. Each fix is its own commit, so an earlier issue's work is in the
// history; reopening it because a LATER issue's correction reverted itself would
// tell the ledger, the summary, and the next round's reviewers that a defect is
// still open while the commit that fixed it stands.
func TestVerifyCorrectionRevertingOneFixKeepsEarlierFixes(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1})
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	before := f.commitCount()

	second := aFinding("a different bug elsewhere")
	second.File, second.Line = "other.go", 42 // distinct, or grouping folds it into i1
	f.respond(1, reviewResponse(t, aFinding("first bug"), second))
	// i1: a real edit that passes the gate, so it commits on its own.
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	// i2: breaks the gate, and its correction throws away everything still
	// uncommitted -- which is i2's work only, since i1 is already in the history.
	f.breakBuildOn(3, "broken.txt")
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "done"}))
	testfixture.WriteSide(f.t, f.respDir, 4, fmt.Sprintf("#!/bin/sh\nrm -f '%s'\ngit -C '%s' checkout -- .\n",
		filepath.Join(f.repo, "broken.txt"), f.repo))
	f.respond(4, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "reverted it all"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() = %v, want the round to end without an error", err)
	}
	if got := f.commitCount(); got != before+1 {
		t.Errorf("commit count = %d, want %d: i1 committed, i2 reverted", got, before+1)
	}
	r1 := sum.Rounds[0]
	if r1.Fixed != 1 {
		t.Errorf("round 1 fixed = %d, want 1: only i2's fix was withdrawn", r1.Fixed)
	}
	for _, it := range r1.Issues {
		switch it.ID {
		case "i1":
			if it.Verdict != model.VerdictFixed {
				t.Errorf("issue i1 = %q, want it still FIXED: its commit stands", it.Verdict)
			}
		case "i2":
			if it.Verdict == model.VerdictFixed || it.StatusOrDefault() != model.StatusOpen {
				t.Errorf("issue i2 = %q/%q, want it reopened", it.Verdict, it.StatusOrDefault())
			}
		}
	}
	for _, fnd := range r1.Findings {
		if fnd.IssueID == "i1" && fnd.Verdict != model.VerdictFixed {
			t.Errorf("observation %s = %q, want the history to keep i1's FIXED", fnd.ID, fnd.Verdict)
		}
		if fnd.IssueID == "i2" && fnd.Verdict == model.VerdictFixed {
			t.Errorf("observation %s still claims FIXED for the reverted i2", fnd.ID)
		}
	}
}

// A round whose only fix was reverted by its verification correction commits
// nothing -- but that is not the coder rejecting the finding. Stopping there would
// report all-rejected (a decision nobody made) and leave the defect in the tree, so
// the reopened issue has to get another review round.
func TestRevertedFixContinuesInsteadOfReportingAllRejected(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	before := f.commitCount()

	f.respond(1, reviewResponse(t, aFinding("nil map write")))
	// Round 1's fix breaks the gate, and its correction throws away everything --
	// the breakage and the edit -- so nothing is left to commit and i1 is reopened.
	f.breakBuildOn(2, "broken.txt")
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	testfixture.WriteSide(f.t, f.respDir, 3, fmt.Sprintf("#!/bin/sh\nrm -f '%s'\ngit -C '%s' checkout -- .\n",
		filepath.Join(f.repo, "broken.txt"), f.repo))
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "reverted it all"}))
	// Round 2: the reviewer re-reports the still-open issue and this time the fix
	// survives, which is exactly the progress the premature stop would have lost.
	f.respond(4, reviewResponse(t, aFinding("nil map write")))
	f.editRepoOn(5)
	f.respond(5, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded the write"}))
	f.respond(6, reviewResponse(t)) // round 3: clean

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want %q: a reverted fix is not an all-rejected round", sum.Termination, model.TermConverged)
	}
	if len(sum.Rounds) != 3 {
		t.Fatalf("rounds = %d, want 3: the reopened issue must get another review round", len(sum.Rounds))
	}
	if got := f.commitCount(); got != before+1 {
		t.Errorf("commit count = %d, want %d: round 2's fix must land", got, before+1)
	}
	if got := issueByID(sum.Rounds[1])["i1"].Verdict; got != model.VerdictFixed {
		t.Errorf("round 2 issue i1 verdict = %q, want it fixed on the second attempt", got)
	}
}

// A SHA shorter than the abbreviation is logged whole rather than sliced: an
// unguarded [:12] would panic while reporting a commit that just landed, turning a
// successful round into a lost run.
func TestShortSHA(t *testing.T) {
	cases := []struct{ sha, want string }{
		{"", ""},
		{"abc", "abc"},
		{"0123456789ab", "0123456789ab"}, // exactly the abbreviation length
		{"0123456789abcdef", "0123456789ab"},
	}
	for _, tc := range cases {
		if got := shortSHA(tc.sha); got != tc.want {
			t.Errorf("shortSHA(%q) = %q, want %q", tc.sha, got, tc.want)
		}
	}
}

// artifact reads one per-step artifact the run wrote, by its rendered filename
// (which carries its own round, so the round-<n>/ directory need not be named).
// Reading the file from disk is the point: it is both what an operator inspects
// afterwards and, for a prompt, the only record of what an agent was actually told.
func (f *fixture) artifact(name string) string {
	f.t.Helper()
	glob := filepath.Join(f.cfg.Logs.StaticBase(), "*", "round-*", name)
	matches, err := filepath.Glob(glob)
	if err != nil {
		f.t.Fatal(err)
	}
	if len(matches) != 1 {
		f.t.Fatalf("glob %s matched %d files, want 1", glob, len(matches))
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		f.t.Fatal(err)
	}
	return string(b)
}

// The correction attempt is the coder's one chance to repair a round the gate
// blocked, and this prompt is the only place it learns WHICH check failed and what
// that check printed. Rendering it wrong (an empty verification block, the round's
// issues lost, the contract missing) would still produce a plausible-looking run:
// the coder would be re-invoked, answer something, and the round would be
// discarded as unverifiable. So assert the prompt's content, not just that a
// correction happened.
func TestVerifyCorrectionPromptCarriesFailures(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	// The fixture's coder template omits {{.Verification}}; the correction render is
	// the only one that fills it, so this test needs a template that uses it.
	fixPrompt := filepath.Join(t.TempDir(), "fix.md")
	if err := os.WriteFile(fixPrompt, []byte("Round {{.Round}}\n{{.Findings}}\n{{.Verification}}\n{{.OutputContract}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.cfg.Roles.Coder.Prompt = fixPrompt
	f.verifyGate(config.VerifyMustPass, "broken.txt")

	f.respond(1, reviewResponse(t, aFinding("off by one")))
	f.breakBuildOn(2, "broken.txt")
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	f.repairBuildOn(3, "broken.txt")
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "corrected"}))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatalf("Run() = %v, want the corrected round to commit", err)
	}
	// The correction artifact is qualified by the issue whose fix is being corrected:
	// the gate runs per fix, so a round can produce several corrections.
	got := f.artifact("fix-mock-fix-verify-i1-round-1.prompt")
	for _, want := range []string{
		"## Verification failed",          // the block's header
		"### build",                       // which check failed, by its configured name
		"exit status 1",                   // how it failed
		"check failed: build is broken",   // and what it printed, so the coder can act
		"Do not disable, skip, or weaken", // the instruction that keeps a "fix" honest
		"off by one",                      // the issue whose fix is in the tree
		"Round 1",                         // the template's own data
		"<fix>",                           // the output contract
	} {
		if !strings.Contains(got, want) {
			t.Errorf("fix-verify prompt is missing %q:\n%s", want, got)
		}
	}
	// The .md is the human-readable record of the attempt an operator reads first.
	// An empty one is indistinguishable from a step that produced nothing at all.
	md := f.artifact("fix-mock-fix-verify-i1-round-1.md")
	for _, want := range []string{
		"# Fix — mock (round 1)", // the step it belongs to
		"off by one",             // the issue under correction, in the reviewer's terms
	} {
		if !strings.Contains(md, want) {
			t.Errorf("fix-verify md is missing %q:\n%s", want, md)
		}
	}
}

// no_regressions is what makes fixpoint usable on a repository that is already
// red: a check failing before the run must not block a round that does not make
// it worse.
func TestVerifyNoRegressionsToleratesPreExistingFailure(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	// The marker exists from the start, so the baseline is already failing.
	if err := os.WriteFile(filepath.Join(f.repo, "broken.txt"), []byte("pre-existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitRun(t, f.repo, "add", "-A")
	gitRun(t, f.repo, "commit", "-q", "-m", "pre-existing breakage")
	f.verifyGate(config.VerifyNoRegressions, "broken.txt")
	before := f.commitCount()

	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.editRepoOn(2) // an unrelated edit; the check still fails, as it did before
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatalf("Run() = %v, want a pre-existing failure to be tolerated under no_regressions", err)
	}
	if got := f.commitCount(); got != before+1 {
		t.Errorf("commit count = %d, want %d: the round should commit", got, before+1)
	}
	if f.invocations() != 2 {
		t.Errorf("invocations = %d, want 2: no correction attempt should have been needed", f.invocations())
	}
}

// The baseline is the longest step before round 1 -- a whole build and test suite
// over an untouched tree -- so a Ctrl-C landing inside it is ordinary. What it must
// not produce is a record claiming the project was already red: the checks it lists
// were killed by the run, and the ones it never started are simply absent.
func TestVerifyBaselineInterruptedIsNotRecordedAsPreExisting(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	// The check signals readiness, then blocks until the teardown kills it.
	ready := filepath.Join(f.t.TempDir(), "baseline-started")
	script := filepath.Join(f.t.TempDir(), "check.sh")
	if err := os.WriteFile(script, []byte(fmt.Sprintf("#!/bin/sh\ntouch '%s'\nsleep 60\n", ready)), 0o700); err != nil {
		t.Fatal(err)
	}
	f.cfg.Verify = config.Verify{
		Policy:   config.VerifyNoRegressions,
		Timeout:  config.Duration(time.Minute),
		Commands: []config.VerifyCommand{{Name: "build", Run: []string{script}}},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()

	o, logs := f.capturingOrchestrator()
	sum, err := o.Run(ctx)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil for an interruption", err)
	}
	if sum.Termination != model.TermInterrupted {
		t.Fatalf("termination = %q, want interrupted", sum.Termination)
	}
	if got := logs(); strings.Contains(got, "pre-existing") {
		t.Errorf("operator log calls the interrupted baseline's checks pre-existing:\n%s", got)
	}

	var base model.JournalVerifyFinished
	payload(t, f.journal(), model.EvVerifyBaseline, &base)
	if !base.Interrupted || base.Passed {
		t.Errorf("verify_baseline = %+v, want it marked interrupted with no verdict", base)
	}
}

// The bug this aggregation exists to fix, end to end: two agents reporting the
// same problem must cost ONE slot against the per-round cap, not two. Before, a
// cap of 1 with two agreeing reviewers meant one report was deferred and the
// panel's agreement actively reduced how much got fixed.
func TestCorroboratedReportsCostOneCapSlot(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, MaxFindingsPerRound: 1})
	// Two lenses, two agents, the same defect at the same location -- as happened
	// in a real run where one lens called it concurrency and the other tests.
	f.cfg.Roles.Review.Prompts = []config.ReviewLens{
		{Agent: "mock", Prompt: f.cfg.Roles.Review.Prompts[0].Prompt},
		{Agent: "mock2", Prompt: f.cfg.Roles.Review.Prompts[0].Prompt},
	}
	f.cfg.Agents["mock2"] = f.cfg.Agents["mock"]

	same := model.ReviewFinding{Category: "concurrency", Severity: "high", File: "main.go", Line: 33, Title: "racy ordinal allocation"}
	other := model.ReviewFinding{Category: "tests", Severity: "high", File: "main.go", Line: 33, Title: "ordinals allocated racily"}
	f.respond(1, reviewResponse(t, same))
	f.respond(2, reviewResponse(t, other))
	f.editRepoOn(3)
	// ONE issue, so one verdict -- and it fits the cap of 1.
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "made it atomic"}))
	f.respond(4, reviewResponse(t))
	f.respond(5, reviewResponse(t))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	r1 := sum.Rounds[0]
	if len(r1.Findings) != 2 {
		t.Fatalf("got %d observations, want both preserved", len(r1.Findings))
	}
	if len(r1.Issues) != 1 {
		t.Fatalf("got %d issues, want 1: the same defect reported twice is one unit of work", len(r1.Issues))
	}
	// Nothing was deferred: agreement no longer consumes budget.
	for _, it := range r1.Issues {
		if it.Verdict == model.VerdictDeferred {
			t.Errorf("issue %s was deferred despite a cap of 1 and only one distinct problem", it.ID)
		}
	}
	if r1.Fixed != 1 {
		t.Errorf("round 1 fixed = %d, want 1", r1.Fixed)
	}
	if agents := r1.Issues[0].Agents(); len(agents) != 2 {
		t.Errorf("Agents() = %v, want the corroboration recorded", agents)
	}
}

// coderPrompt returns the coder prompt the run wrote for a round, or "" when the
// coder was never invoked in it. The prompt file is the only artifact that shows
// what was actually HANDED to the coder, as opposed to what the round recorded.
// coderSessions counts the coder invocations a round made -- one per issue it was
// given work for, so 0 proves the round never spent a session at all.
func (f *fixture) coderSessions(round int) int {
	f.t.Helper()
	return len(f.coderPromptFiles(round))
}

func (f *fixture) coderPromptFiles(round int) []string {
	f.t.Helper()
	prompts, err := filepath.Glob(filepath.Join(f.cfg.Logs.StaticBase(), "*", fmt.Sprintf("round-%d", round), "fix-*.prompt"))
	if err != nil {
		f.t.Fatal(err)
	}
	return prompts
}

// coderPrompt returns every coder prompt the round wrote, concatenated. The round
// now spends one session per issue, so a caller asking "was i1 handed back?" wants
// the round's whole workload, not one session's.
func (f *fixture) coderPrompt(round int) string {
	f.t.Helper()
	prompts := f.coderPromptFiles(round)
	if len(prompts) == 0 {
		return ""
	}
	var all strings.Builder
	for _, name := range prompts {
		b, err := os.ReadFile(name)
		if err != nil {
			f.t.Fatal(err)
		}
		all.Write(b)
	}
	return all.String()
}

// issueByID indexes a round's issues, so an assertion can name the issue it means
// rather than depending on ordering.
func issueByID(rec model.RoundRecord) map[string]model.Issue {
	out := map[string]model.Issue{}
	for _, it := range rec.Issues {
		out[it.ID] = it
	}
	return out
}

// A reviewer's own cross-round issue reference is the only signal that identifies a
// finding whose wording AND line have both moved -- the ledger's fingerprint cannot.
// It must therefore survive the review output into the observation and reach the
// ledger. When it was dropped, a reworded re-report became a second issue: the fix
// that had already been attempted looked new, deferral aging restarted, and a coder
// answering about the id it was shown failed the round with "unknown issue id".
func TestRunHonorsReviewerDeclaredIssueID(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, model.ReviewFinding{
		Category: "bugs", Severity: "high", File: "main.go", Line: 1, Title: "off by one in the loop bound",
	}))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	// Round 2: the same defect, still there after the fix, but reported by a
	// different lens in different words at a different line. Nothing lexical or
	// positional links it to i1 -- only the reviewer's declaration does.
	f.respond(3, reviewResponse(t, model.ReviewFinding{
		Issue: "i1", Category: "bugs", Severity: "high", File: "other.go", Line: 42,
		Title: "the loop still walks one element past the end",
	}))
	f.editRepoOn(4)
	f.respond(4, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "really patched"}))
	f.respond(5, reviewResponse(t)) // round 3: clean -> converge

	o := f.orchestrator()
	sum, err := o.Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged", sum.Termination)
	}
	if len(sum.Rounds) < 2 {
		t.Fatalf("rounds = %d, want at least 2", len(sum.Rounds))
	}
	r2 := sum.Rounds[1]
	if len(r2.Issues) != 1 || r2.Issues[0].ID != "i1" {
		t.Fatalf("round 2 issues = %+v, want the declared i1 rather than a new issue", r2.Issues)
	}
	if len(r2.Findings) != 1 || r2.Findings[0].IssueID != "i1" {
		t.Errorf("round 2 observation was not grouped under i1: %+v", r2.Findings)
	}
	if got := len(o.ledger.Issues()); got != 1 {
		t.Errorf("ledger holds %d issues, want 1: the reword must not create a second identity", got)
	}
	if got := o.ledger.Issues()[0].FirstRound; got != 1 {
		t.Errorf("issue first seen in round %d, want 1: cross-round identity was lost", got)
	}
}

// A previously rejected issue is re-reported by reviewers who have not been
// convinced, and the ledger deliberately returns it carrying that verdict so the
// summary shows it came up again. It must NOT be handed back to the coder: the
// decision is made, and resubmitting it would spend a coder verdict -- and one of
// the round's limited issue slots -- on it in every remaining round.
func TestRunDoesNotResubmitRejectedIssues(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 4, CleanRoundsToStop: 1, MaxFindingsPerRound: 2})
	rejected := model.ReviewFinding{Category: "design", Severity: "high", File: "main.go", Line: 1, Title: "unexported field should be exported"}
	fixable := model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 10, Title: "nil map write"}
	// Round 1: two issues, both within the cap. The coder rejects i1 and fixes i2,
	// so the round commits and the loop continues.
	f.respond(1, reviewResponse(t, rejected, fixable))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "deliberate: the field is internal"}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "guarded the write"}))
	// Round 2: the rejected issue comes back unchanged, plus two genuinely new ones.
	// With a cap of 2 the new pair must BOTH be active: the decided issue is not
	// work, so it cannot displace one of them.
	f.respond(4, reviewResponse(t, rejected,
		model.ReviewFinding{Category: "tests", Severity: "medium", File: "main.go", Line: 20, Title: "no coverage for the error path"},
		model.ReviewFinding{Category: "docs", Severity: "low", File: "main.go", Line: 30, Title: "stale comment on the exported helper"},
	))
	f.editRepoOn(5)
	f.respond(5, fixResponse(t, model.FixResult{ID: "i3", Verdict: "fixed", Detail: "added a case"}))
	f.editRepoOn(6)
	f.respond(6, fixResponse(t, model.FixResult{ID: "i4", Verdict: "fixed", Detail: "rewrote the comment"}))
	f.respond(7, reviewResponse(t)) // round 3: clean -> converge

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged", sum.Termination)
	}
	if len(sum.Rounds) != 3 {
		t.Fatalf("rounds = %d, want 3", len(sum.Rounds))
	}
	r2 := issueByID(sum.Rounds[1])
	if got := r2["i1"].Verdict; got != model.VerdictRejected {
		t.Errorf("round 2 issue i1 verdict = %q, want it preserved as rejected", got)
	}
	if !strings.Contains(r2["i1"].VerdictDetail, "previously rejected") {
		t.Errorf("round 2 issue i1 detail = %q, want it to say the rejection is carried over", r2["i1"].VerdictDetail)
	}
	// The cap of 2 was spent on the two new issues, not on the decided one.
	for _, id := range []string{"i3", "i4"} {
		if got := r2[id].Verdict; got != model.VerdictFixed {
			t.Errorf("round 2 issue %s verdict = %q, want fixed: a rejected issue must not consume a cap slot", id, got)
		}
	}
	prompt := f.coderPrompt(2)
	if prompt == "" {
		t.Fatal("round 2 wrote no coder prompt")
	}
	if strings.Contains(prompt, "[i1]") {
		t.Errorf("the rejected issue was handed back to the coder:\n%s", prompt)
	}
	if !strings.Contains(prompt, "[i3]") || !strings.Contains(prompt, "[i4]") {
		t.Errorf("round 2 coder prompt is missing the new issues:\n%s", prompt)
	}
	// The carried verdict must reach the round's OBSERVATIONS too, not just the
	// issue: history is rendered from findings, and one with an empty verdict prints
	// as UNRESOLVED -- which the history preamble tells reviewers to report again.
	// The run would then solicit a re-report of a decided issue every round.
	for _, f := range sum.Rounds[1].Findings {
		if f.IssueID != "i1" {
			continue
		}
		if f.Verdict != model.VerdictRejected {
			t.Errorf("round 2 observation of i1 has verdict %q, want it mirrored as rejected (it renders as %s in history)",
				f.Verdict, strings.ToUpper(f.VerdictOrDefault()))
		}
	}
	// "— UNRESOLVED:" is the rendered per-finding form; the word alone also appears
	// in the section preamble, which is not what this is about.
	if h := f.reviewPrompt(3); strings.Contains(h, "— UNRESOLVED") {
		t.Errorf("round 3 reviewer history shows a decided issue as UNRESOLVED, which asks for it to be re-reported:\n%s", h)
	}
}

// reviewPrompt returns a reviewer prompt the run wrote for a round -- the artifact
// that shows the history reviewers were actually shown.
func (f *fixture) reviewPrompt(round int) string {
	f.t.Helper()
	prompts, err := filepath.Glob(filepath.Join(f.cfg.Logs.StaticBase(), "*", fmt.Sprintf("round-%d", round), "review-*.prompt"))
	if err != nil {
		f.t.Fatal(err)
	}
	if len(prompts) == 0 {
		return ""
	}
	b, err := os.ReadFile(prompts[0])
	if err != nil {
		f.t.Fatal(err)
	}
	return string(b)
}

// A refused declaration -- a reviewer citing a REJECTED issue's id while
// describing a different defect -- is only ever visible to an operator through the
// line logLedgerConflicts prints. The suppression itself is prevented by the
// ledger either way, so a wiring mistake (draining the conflicts before Absorb, or
// losing one of the two call sites in a refactor) costs no correctness and fails
// no test: it just makes the event disappear. Both call sites are covered here,
// since they are separate lines.
func TestRunLogsRefusedDeclarationOfRejectedIssue(t *testing.T) {
	// loggedRun runs f's orchestrator with a capturing logf and returns the summary
	// and everything it logged.
	loggedRun := func(t *testing.T, f *fixture) (*model.RunSummary, string) {
		t.Helper()
		var logs strings.Builder
		o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}},
			func(format string, a ...any) { fmt.Fprintf(&logs, format+"\n", a...) })
		if err != nil {
			t.Fatal(err)
		}
		sum, err := o.Run(t.Context())
		if err != nil {
			t.Fatalf("Run() err = %v", err)
		}
		return sum, logs.String()
	}
	// The defect the reviewer describes while citing i1: another file, another
	// line, a title that agrees with nothing already known -- so the declaration is
	// the ONLY thing tying it to the rejected issue, and the ledger refuses it.
	hijack := model.ReviewFinding{
		Issue: "i1", Category: "bugs", Severity: "high", File: "other.go", Line: 7,
		Title: "the deadline context is never canceled",
	}
	// assertConflict checks the operator-visible line and that the observation was
	// filed under a new issue that the round actually worked on.
	assertConflict := func(t *testing.T, logs string, round model.RoundRecord) {
		t.Helper()
		for _, want := range []string{"declared issue i1", "already rejected", "recorded as i3"} {
			if !strings.Contains(logs, want) {
				t.Errorf("conflict log missing %q, got logs:\n%s", want, logs)
			}
		}
		if len(round.Findings) != 1 || round.Findings[0].IssueID != "i3" {
			t.Fatalf("observations = %+v, want the hijacked report filed under a new issue", round.Findings)
		}
		byID := issueByID(round)
		if _, buried := byID["i1"]; buried {
			t.Errorf("the report was attached to the rejected i1: %+v", round.Issues)
		}
		if got := byID["i3"]; got.Title != hijack.Title || got.Verdict != model.VerdictFixed {
			t.Errorf("issue i3 = %+v, want the described defect, fixed rather than suppressed", got)
		}
	}
	// Round 1 is shared: two issues, i1 rejected (so its id becomes one that
	// silences a finding) and i2 fixed (so the round commits and the run goes on).
	setupRound1 := func(f *fixture) {
		f.respond(1, reviewResponse(t,
			model.ReviewFinding{Category: "design", Severity: "high", File: "main.go", Line: 1, Title: "unexported field should be exported"},
			model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 10, Title: "nil map write"},
		))
		f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "deliberate: the field is internal"}))
		f.editRepoOn(3)
		f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "guarded the write"}))
	}

	t.Run("in a loop round", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 4, CleanRoundsToStop: 1, MaxFindingsPerRound: 2})
		setupRound1(f)
		f.respond(4, reviewResponse(t, hijack)) // round 2: the refused declaration
		f.editRepoOn(5)
		f.respond(5, fixResponse(t, model.FixResult{ID: "i3", Verdict: "fixed", Detail: "canceled it"}))
		f.respond(6, reviewResponse(t)) // round 3: clean -> converge

		sum, logs := loggedRun(t, f)
		if len(sum.Rounds) != 3 {
			t.Fatalf("rounds = %d, want 3", len(sum.Rounds))
		}
		if !strings.Contains(logs, "round 2: mock declared issue i1") {
			t.Errorf("conflict must name the round and the reviewer, got logs:\n%s", logs)
		}
		assertConflict(t, logs, sum.Rounds[1])
	})

	t.Run("in the closing round", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, MaxFindingsPerRound: 2})
		f.finalLens()
		setupRound1(f)
		f.respond(4, reviewResponse(t))         // round 2: clean -> converged
		f.respond(5, reviewResponse(t, hijack)) // closing pass 1: the refused declaration
		f.editRepoOn(6)
		f.respond(6, fixResponse(t, model.FixResult{ID: "i3", Verdict: "fixed", Detail: "canceled it"}))
		f.respond(7, reviewResponse(t)) // closing pass 2: clean -> the closing round is done

		sum, logs := loggedRun(t, f)
		closing := sum.Rounds[2]
		if !closing.Final {
			t.Fatalf("round 3 = %+v, want the closing pass", closing)
		}
		if !strings.Contains(logs, "round 3: mock2 declared issue i1") {
			t.Errorf("the closing-pass call site logged no conflict, got logs:\n%s", logs)
		}
		assertConflict(t, logs, closing)
	})
}

// A round in which every reported issue was already rejected has no work in it.
// The coder must not be invoked on an empty list (a wasted session), and the loop
// must not keep re-reviewing the same decided issues until max_iterations: it is the
// same terminal state as a round the coder rejected outright, which exits non-zero
// because nothing changed.
func TestRunTerminatesWhenOnlyRejectedIssuesRemain(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 4, CleanRoundsToStop: 1})
	rejected := model.ReviewFinding{Category: "design", Severity: "high", File: "main.go", Line: 1, Title: "unexported field should be exported"}
	fixable := model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 10, Title: "nil map write"}
	f.respond(1, reviewResponse(t, rejected, fixable))
	// One coder session per issue, in worst-first order: i1 then i2.
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "deliberate: the field is internal"}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "guarded the write"}))
	f.respond(4, reviewResponse(t, rejected)) // round 2: only the decided issue is left

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if sum.Termination != model.TermAllRejected {
		t.Fatalf("termination = %q, want %q", sum.Termination, model.TermAllRejected)
	}
	if len(sum.Rounds) != 2 {
		t.Fatalf("rounds = %d, want 2", len(sum.Rounds))
	}
	if got := f.invocations(); got != 4 {
		t.Errorf("agent invocations = %d, want 4 (two reviews and one fix session per issue; round 2 must not invoke the coder)", got)
	}
	if n := f.coderSessions(2); n != 0 {
		t.Errorf("round 2 invoked the coder %d time(s) with no work to do", n)
	}
	if got := issueByID(sum.Rounds[1])["i1"].Verdict; got != model.VerdictRejected {
		t.Errorf("round 2 issue i1 verdict = %q, want the rejection reported", got)
	}
}

// The same early all-rejected termination, but with a reviewer that FAILED: the
// round's review is then incomplete, so the issues that reviewer would have
// reported were never seen. Reporting the success-shaped all_rejected termination
// there would tell automation a complete decision was reached over a partial
// round, so this route needs its own reviewer-error guard -- the older
// all-rejected-with-reviewer-error test reaches finalizeFix through the coder and
// cannot protect it.
func TestRunTerminatesWhenOnlyRejectedIssuesRemainWithReviewerErrorFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 4, CleanRoundsToStop: 1})
	// A second lens pinned to an agent that always fails, so every round carries a
	// reviewer error alongside the working lens's findings.
	f.cfg.Agents["bad"] = config.Agent{Command: []string{"false"}, PromptVia: "stdin", Timeout: config.Duration(time.Minute)}
	f.cfg.Roles.Review.Prompts = append(f.cfg.Roles.Review.Prompts,
		config.ReviewLens{Agent: "bad", Prompt: f.cfg.Roles.Review.Prompts[0].Prompt})
	rejected := model.ReviewFinding{Category: "design", Severity: "high", File: "main.go", Line: 1, Title: "unexported field should be exported"}
	fixable := model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 10, Title: "nil map write"}
	f.respond(1, reviewResponse(t, rejected, fixable))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "deliberate: the field is internal"}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "guarded the write"}))
	f.respond(4, reviewResponse(t, rejected)) // round 2: only the decided issue is left

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "reviewer(s) failed") {
		t.Fatalf("Run() err = %v, want aggregated reviewer failure", err)
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error (not %q): a partial round is not a decision", sum.Termination, model.TermAllRejected)
	}
	if len(sum.Rounds) != 2 || len(sum.Rounds[1].ReviewErrors) != 1 {
		t.Errorf("the partial round record must be kept with its reviewer error: %+v", sum.Rounds)
	}
	// Still no wasted coder session: the guard fires instead of the termination, and
	// neither invokes the coder on an empty workload.
	if n := f.coderSessions(2); n != 0 {
		t.Errorf("round 2 invoked the coder %d time(s) with no work to do", n)
	}
}

// Two fixpoint runs on one repository invalidate every snapshot the loop's
// decisions rest on: both see a clean tree, both launch coders into the same
// files, and edits one run never verified land in the other's commit. The second
// run must be refused at preflight -- before the clean check, before any agent is
// paid for, and before anything is committed.
func TestRunRefusesWhileAnotherRunHoldsTheRepository(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))

	release, err := target.New(config.Target{Mode: config.ModeDirectory, Path: f.repo}).LockRepo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "another fixpoint run") {
		t.Fatalf("Run() err = %v, want a refusal naming the run that owns the repository", err)
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error", sum.Termination)
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0: the refusal must precede any agent work", got)
	}
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want 1 (nothing may be committed)", got)
	}
}

// ---- closing round for final: true lenses ------------------------------------

// finalLens adds a `final: true` lens on a second agent, so the closing round is
// distinguishable from the loop's own reviewer.
func (f *fixture) finalLens() {
	f.t.Helper()
	f.cfg.Agents["mock2"] = f.cfg.Agents["mock"]
	f.cfg.Roles.Review.Prompts = append(f.cfg.Roles.Review.Prompts, config.ReviewLens{
		Agent:  "mock2",
		Prompt: f.cfg.Roles.Review.Prompts[0].Prompt,
		Final:  true,
	})
}

// A final lens must not run during the loop -- that is the whole point, since its
// subject is the finished code -- and must run after it, with its findings fixed.
// The phase then repeats while it keeps fixing, so it ends on the pass that finds
// nothing rather than after a fixed number of rounds.
// A round's reviewers all read ONE snapshot, then its per-fix sessions commit into
// the tree one after another -- so the second session opens a finding written
// against a tree that has since moved. Observed in a real run as three findings
// rejected with "already handled at HEAD, in the most recent commit 390cfcb", a
// commit made minutes earlier by an earlier session of the same pass. The later
// session is told which of its files moved, so it reads before it edits.
func TestFixSessionIsWarnedWhenItsFilesMovedEarlierInTheRound(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	// The fixture's coder template omits {{.Stale}} (see newFixture); this test is
	// about what reaches the prompt, so it needs the placeholder the shipped
	// config/prompts/fix.md carries.
	fixPrompt := f.cfg.Roles.Coder.PromptFile()
	if err := os.WriteFile(fixPrompt, []byte("Round {{.Round}}\n{{.Findings}}\n{{.Stale}}\n{{.OutputContract}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Two issues in ONE round, both naming main.go -- which is the file editRepoOn
	// writes, so session 1's commit moves the ground under session 2's finding.
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 1, Title: "first"},
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 90, Title: "second"}))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "patched too"}))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	prompts := f.coderPromptFiles(1)
	if len(prompts) != 2 {
		t.Fatalf("got %d coder prompts, want 2", len(prompts))
	}
	// The FIRST session runs against the tree its reviewers read, so it must not be
	// told anything moved -- a warning on every session would be ignored on all of them.
	first := f.artifact(filepath.Base(prompts[0]))
	if strings.Contains(first, "may already be out of date") {
		t.Errorf("the first session of a round was warned about a tree that had not moved yet:\n%s", first)
	}
	// The SECOND session opens its finding after session 1 committed to main.go.
	second := f.artifact(filepath.Base(prompts[1]))
	for _, want := range []string{"may already be out of date", "main.go", "Read the CURRENT contents"} {
		if !strings.Contains(second, want) {
			t.Errorf("the second session's prompt is missing %q:\n%s", want, second)
		}
	}
}

// The closing phase repeats until it has nothing left to fix, which for a coverage
// lens is never: each pass reviews the tests the previous pass just wrote. It used
// to borrow loop.max_iterations, which conflated two unrelated budgets -- one
// measured run spent 51 minutes there and pass 2 still surfaced four NEW issues.
// max_final_passes bounds it on its own, and the run still ends cleanly with what
// did not fit reported rather than dropped.
func TestFinalPhaseStopsAtMaxFinalPasses(t *testing.T) {
	// max_iterations deliberately far larger, to prove the phase no longer reads it.
	f := newFixture(t, config.Loop{MaxIterations: 9, MaxFinalPasses: 2, CleanRoundsToStop: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t)) // loop round 1: clean -> converged

	// Every closing pass finds a NEW gap and fixes it, so the phase never runs out
	// of work on its own and only the cap can end it.
	for i, n := range []int{2, 4, 6, 8} {
		f.respond(n, reviewResponse(t, model.ReviewFinding{
			Category: "tests", Severity: "medium",
			File: fmt.Sprintf("gap%d.go", i), Line: 10 + i,
			Title: fmt.Sprintf("gap %d has no test", i),
		}))
		f.editRepoOn(n + 1)
		f.respond(n+1, fixResponse(t, model.FixResult{
			ID: fmt.Sprintf("i%d", i+1), Verdict: "fixed", Detail: "test added",
		}))
	}

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermConverged {
		t.Errorf("termination = %q, want the loop's converged preserved", sum.Termination)
	}
	passes := 0
	for _, r := range sum.Rounds {
		if r.Final {
			passes++
		}
	}
	if passes != 2 {
		t.Errorf("closing passes = %d, want exactly 2 (max_final_passes), not %d (max_iterations)", passes, f.cfg.Loop.MaxIterations)
	}
	// Bounded, not silently truncated: the cap stops the phase but the commits it
	// did make still stand.
	if sum.Rounds[len(sum.Rounds)-1].Fixed != 1 {
		t.Errorf("last closing pass fixed %d, want 1: the capped phase still commits its work", sum.Rounds[len(sum.Rounds)-1].Fixed)
	}
}

// A closing pass whose verification correction reverts the whole fix commits
// nothing and reopens the issue -- the defect is still in the tree, and nobody
// rejected it. The phase used to read "nothing committed" as "nothing left to ask",
// so that reopened issue ended the run unresolved with no pass left to look at it.
// The loop's finalizeRound already keeps going in exactly this case; the closing
// path has to as well.
func TestClosingPassContinuesWhenACorrectionReopensTheOnlyFix(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 9, MaxFinalPasses: 2, CleanRoundsToStop: 1})
	f.finalLens()
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	before := f.commitCount()

	f.respond(1, reviewResponse(t)) // loop round 1: clean -> converged
	// Closing pass 1 reports a finding, and the coder's edits fail the gate.
	f.respond(2, reviewResponse(t, aFinding("bug")))
	f.breakBuildOn(3, "broken.txt")
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	// The correction throws away everything the pass did, so the gate passes over a
	// tree identical to the pass's starting point and there is nothing to commit.
	testfixture.WriteSide(t, f.respDir, 4, fmt.Sprintf("#!/bin/sh\nrm -f '%s'\ngit -C '%s' checkout -- .\n",
		filepath.Join(f.repo, "broken.txt"), f.repo))
	f.respond(4, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "reverted it all"}))
	// Pass 2 is the only thing that can still resolve i1, so it must run.
	f.respond(5, reviewResponse(t))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() = %v, want the reverted closing pass to continue without an error", err)
	}
	if got := f.commitCount(); got != before {
		t.Errorf("commit count = %d, want %d: a reverted pass has nothing to commit", got, before)
	}
	passes := 0
	for _, r := range sum.Rounds {
		if r.Final {
			passes++
		}
	}
	if passes != 2 {
		t.Errorf("closing passes = %d, want 2: the reopened issue must get another pass", passes)
	}
	for _, it := range sum.Rounds[1].Issues {
		if it.Verdict == model.VerdictFixed || it.StatusOrDefault() != model.StatusOpen {
			t.Errorf("issue %s = %q/%q, want it reopened", it.ID, it.Verdict, it.StatusOrDefault())
		}
	}
}

// warnFinalPhaseCapped only runs when the last allowed pass asked for another one,
// so every call is a real exhaustion. It used to say nothing when that pass had
// fixed everything it reported -- the very case where the closing round committed
// edits and then ran out of passes to review them, leaving a run that reads as a
// clean finish over a tree nobody looked at last. The two loose ends it reports are
// independent: a rejected issue is decided rather than unfixed, and a pass can both
// commit fixes and leave issues behind.
func TestWarnFinalPhaseCappedAlwaysReportsExhaustion(t *testing.T) {
	issue := func(id, status string) model.Issue {
		return model.Issue{ID: id, Status: status}
	}
	tests := []struct {
		name     string
		last     model.RoundRecord
		wants    []string
		notWants []string
	}{
		{
			name:  "issues still open",
			last:  model.RoundRecord{Round: 3, Final: true, Issues: []model.Issue{issue("i1", model.VerdictDeferred)}},
			wants: []string{"WARNING", "1 issue(s) still open", "NOT fixed"},
		},
		{
			name:  "everything fixed, so the fixes themselves went unreviewed",
			last:  model.RoundRecord{Round: 3, Final: true, Issues: []model.Issue{issue("i1", model.VerdictFixed)}},
			wants: []string{"WARNING", "NOT re-reviewed"},
		},
		{
			name: "a rejected issue is decided, not unfixed work",
			last: model.RoundRecord{Round: 3, Final: true, Issues: []model.Issue{
				issue("i1", model.VerdictFixed), issue("i2", model.VerdictRejected),
			}},
			wants:    []string{"WARNING", "NOT re-reviewed"},
			notWants: []string{"still open"},
		},
		{
			name: "fixes committed AND issues left over: both are reported",
			last: model.RoundRecord{Round: 3, Final: true, Issues: []model.Issue{
				issue("i1", model.VerdictFixed), issue("i2", model.VerdictDeferred), issue("i3", model.VerdictRejected),
			}, Commits: []string{"abc1234"}},
			wants: []string{"WARNING", "1 issue(s) still open", "NOT fixed", "NOT re-reviewed"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var log strings.Builder
			o := &Orchestrator{logf: func(format string, args ...any) {
				fmt.Fprintf(&log, format+"\n", args...)
			}}
			o.warnFinalPhaseCapped(&model.RunSummary{Rounds: []model.RoundRecord{tc.last}}, 2)
			for _, want := range tc.wants {
				if !strings.Contains(log.String(), want) {
					t.Errorf("warning is missing %q:\n%s", want, log.String())
				}
			}
			for _, no := range tc.notWants {
				if strings.Contains(log.String(), no) {
					t.Errorf("warning should not say %q:\n%s", no, log.String())
				}
			}
		})
	}
}

func TestFinalLensRunsAfterTheLoopAndStopsWhenClean(t *testing.T) {
	// MaxFinalPasses set explicitly: the default is 1, and "stops when clean" is a
	// claim about the phase ending ON ITS OWN -- which only a second pass can show,
	// since a single-pass phase would end at the cap either way.
	f := newFixture(t, config.Loop{MaxIterations: 3, MaxFinalPasses: 2, CleanRoundsToStop: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t, aFinding("off by one"))) // round 1 reviewer
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, reviewResponse(t)) // round 2: clean -> converged
	// A distinct location, or it would match the round-1 issue by fingerprint.
	f.respond(4, reviewResponse(t, model.ReviewFinding{
		Category: "tests", Severity: "medium", File: "helper.go", Line: 42, Title: "no test here",
	}))
	f.editRepoOn(5)
	f.respond(5, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "test added"}))
	f.respond(6, reviewResponse(t)) // closing pass 2: nothing left -> phase ends

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// The loop's verdict survives: the closing phase is extra work on an already
	// decided run, not a new verdict on it.
	if sum.Termination != model.TermConverged {
		t.Errorf("termination = %q, want the loop's converged to be preserved", sum.Termination)
	}
	if len(sum.Rounds) != 4 {
		t.Fatalf("got %d rounds, want 4 (2 loop + 2 closing passes)", len(sum.Rounds))
	}
	// The loop rounds saw ONLY the recurring lens; the final lens was held back.
	for i, r := range sum.Rounds[:2] {
		for _, a := range r.Assignments {
			if a.Agent == "mock2" {
				t.Errorf("round %d ran the final lens (%s) inside the loop", i+1, a.Agent)
			}
		}
		if r.Final {
			t.Errorf("round %d is marked final", i+1)
		}
	}
	for i, r := range sum.Rounds[2:] {
		if !r.Final {
			t.Errorf("closing pass %d is not marked Final; a reader could not tell it from a loop round", i+1)
		}
		if len(r.Assignments) != 1 || r.Assignments[0].Agent != "mock2" {
			t.Errorf("closing pass %d assignments = %+v, want only the final lens", i+1, r.Assignments)
		}
	}
	if first := sum.Rounds[2]; first.Fixed != 1 {
		t.Errorf("first closing pass fixed = %d, want 1: a final lens's findings are fixed, not just reported", first.Fixed)
	} else if first.CommitSHA == "" {
		t.Error("the first closing pass's fix was not committed")
	}
	if last := sum.Rounds[3]; last.Fixed != 0 || last.CommitSHA != "" {
		t.Errorf("the clean closing pass committed something: fixed=%d sha=%q", last.Fixed, last.CommitSHA)
	}
}

// The point of iterating: max_findings_per_round caps ONE CODER SESSION, and with no
// round following the closing one, a single capped pass would fix the cap's worth
// and silently drop the rest. Two gaps arrive under a cap of one, and both get fixed.
func TestFinalPhaseProcessesMoreThanTheCapAcrossPasses(t *testing.T) {
	// MaxFinalPasses set explicitly: the default is 1, and what this test exercises
	// is the REPEAT -- pass 2 is the subject, not the default.
	f := newFixture(t, config.Loop{MaxIterations: 4, MaxFinalPasses: 2, CleanRoundsToStop: 1, MaxFindingsPerRound: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t)) // loop round 1: clean -> converged immediately

	gapA := model.ReviewFinding{Category: "tests", Severity: "high", File: "a.go", Line: 10, Title: "a has no test"}
	gapB := model.ReviewFinding{Category: "tests", Severity: "low", File: "b.go", Line: 20, Title: "b has no test"}
	// Closing pass 1: two distinct gaps, but the cap admits one.
	f.respond(2, reviewResponse(t, gapA, gapB))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "test for a"}))
	// Pass 2: the deferred gap is still reported, and now fits.
	f.respond(4, reviewResponse(t, gapB))
	f.editRepoOn(5)
	f.respond(5, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "test for b"}))
	f.respond(6, reviewResponse(t)) // pass 3: clean -> phase ends

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	fixed, passes := 0, 0
	for _, r := range sum.Rounds {
		if r.Final {
			fixed += r.Fixed
			passes++
		}
	}
	if fixed != 2 {
		t.Errorf("closing phase fixed %d issue(s), want 2: a cap of 1 must not mean 1 of 2 gaps gets fixed", fixed)
	}
	if passes < 2 {
		t.Errorf("closing passes = %d, want at least 2: one pass cannot exceed the cap", passes)
	}
}

// A capped closing pass hands the coder only its active issues. If the coder
// rejects every one of them, nothing is committed -- but the deferred remainder
// never reached the coder at all, so the phase is not done: without another pass
// those findings are dropped without even the cap warning, which is exactly the
// silent partial closing round this phase exists to prevent.
func TestFinalPhaseContinuesForDeferredWhenActiveAllRejected(t *testing.T) {
	// MaxFinalPasses set explicitly: the default is 1, and what this test exercises
	// is the REPEAT -- pass 2 is the subject, not the default.
	f := newFixture(t, config.Loop{MaxIterations: 4, MaxFinalPasses: 2, CleanRoundsToStop: 1, MaxFindingsPerRound: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t)) // loop round 1: clean -> converged immediately

	gapA := model.ReviewFinding{Category: "tests", Severity: "high", File: "a.go", Line: 10, Title: "a has no test"}
	gapB := model.ReviewFinding{Category: "tests", Severity: "low", File: "b.go", Line: 20, Title: "b has no test"}
	// Closing pass 1: two gaps, cap 1 -> gapA active, gapB deferred.
	f.respond(2, reviewResponse(t, gapA, gapB))
	// The one active issue is rejected, so the pass commits nothing.
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "covered elsewhere"}))
	// Pass 2 must happen: the deferred gap is reported again and now fits.
	f.respond(4, reviewResponse(t, gapB))
	f.editRepoOn(5)
	f.respond(5, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "test for b"}))
	f.respond(6, reviewResponse(t)) // pass 3: clean -> phase ends

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if sum.Termination != model.TermConverged {
		t.Errorf("termination = %q, want the loop's converged to be preserved", sum.Termination)
	}
	fixed, passes := 0, 0
	for _, r := range sum.Rounds {
		if r.Final {
			fixed += r.Fixed
			passes++
		}
	}
	if passes < 2 {
		t.Fatalf("closing passes = %d, want at least 2: the deferred gap never reached the coder in pass 1", passes)
	}
	if fixed != 1 {
		t.Errorf("closing phase fixed %d issue(s), want 1: rejecting the active issue must not drop the deferred one", fixed)
	}
	first := sum.Rounds[1]
	if !first.Final || first.Rejected != 1 || first.CommitSHA != "" {
		t.Fatalf("first closing pass = %+v, want a final pass that rejected 1 and committed nothing", first)
	}
	if got := deferredFindings(&first); got != 1 {
		t.Errorf("first closing pass deferred %d finding(s), want 1", got)
	}
}

// An issue the cap deferred never reached the coder, and it only gets there if a
// later round's reviewers report it again. When they do not, the round reads as
// clean and the loop converges -- exit 0 over work fixpoint itself withheld. The
// streak is decided from one round's observations, so only the ledger can see it,
// and the run must at least say so out loud.
func TestConvergenceWarnsAboutIssuesTheCapNeverHandedOver(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, MaxFindingsPerRound: 1})
	bug := model.ReviewFinding{Category: "correctness", Severity: "high", File: "a.go", Line: 1, Title: "off by one"}
	starved := model.ReviewFinding{Category: "docs", Severity: "low", File: "README.md", Line: 3, Title: "stale flag name"}
	f.respond(1, reviewResponse(t, bug, starved)) // cap 1 -> i1 active, i2 deferred
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, reviewResponse(t)) // round 2: clean, and i2 is never mentioned again

	o, logs := f.capturingOrchestrator()
	sum, err := o.Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged", sum.Termination)
	}
	log := logs()
	if !strings.Contains(log, "WARNING") || !strings.Contains(log, "i2") || !strings.Contains(log, "deferred") {
		t.Errorf("converged run said nothing about the deferred issue it withheld; log:\n%s", log)
	}
}

// The counterpart: a run whose every issue was decided converges quietly. A
// warning on a clean finish would train operators to ignore it.
func TestConvergenceQuietWhenEveryIssueWasDecided(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	bug := model.ReviewFinding{Category: "correctness", Severity: "high", File: "a.go", Line: 1, Title: "off by one"}
	naming := model.ReviewFinding{Category: "style", Severity: "low", File: "b.go", Line: 2, Title: "bad name"}
	f.respond(1, reviewResponse(t, bug, naming))
	// One session per issue: the first fixes, the second rejects.
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "rejected", Detail: "intentional"}))
	f.respond(4, reviewResponse(t)) // round 2: clean

	o, logs := f.capturingOrchestrator()
	sum, err := o.Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged", sum.Termination)
	}
	if log := logs(); strings.Contains(log, "never resolved") {
		t.Errorf("warned about unresolved work on a run where every issue was fixed or rejected; log:\n%s", log)
	}
}

// The closing round runs after max-iterations too: the loop is equally done
// editing, whether it converged or ran out of rounds.
func TestFinalLensRunsAfterMaxIterations(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t, aFinding("still broken")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, reviewResponse(t)) // closing round: nothing to report

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermMaxIterations {
		t.Errorf("termination = %q, want max-iterations preserved", sum.Termination)
	}
	if len(sum.Rounds) != 2 || !sum.Rounds[1].Final {
		t.Fatalf("want a closing round after max-iterations, got %d round(s)", len(sum.Rounds))
	}
}

// A failed run must not start new work: the tree is in a state nobody vouched for.
func TestFinalLensSkippedWhenTheLoopFailed(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1})
	f.finalLens()
	// The coder claims a fix but leaves the tree untouched, which fails the round.
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "lied"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("expected the round to fail")
	}
	for _, r := range sum.Rounds {
		if r.Final {
			t.Error("a closing round ran after the loop failed; the tree is unverified at that point")
		}
	}
}

// A closing reviewer that fails is a failed closing round. Nothing follows to
// catch what it never looked at, so its silence must not be read as "no findings"
// -- and the run must not report the loop's success while the last look over the
// finished tree did not happen.
func TestFinalRoundReviewerFailureFailsTheRun(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t)) // round 1: clean -> converged
	f.respond(2, "I looked around but forgot the <review> envelope.")

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() succeeded although the only closing reviewer failed")
	}
	if !strings.Contains(err.Error(), "reviewer(s) failed") {
		t.Errorf("error should name the reviewer failure, got: %v", err)
	}
	// The durable artifacts must agree with the exit status: a summary still
	// saying "converged" with no error is a claim of success for work that failed.
	if sum.Termination != model.TermError || sum.Error == "" {
		t.Errorf("termination = %q, error = %q; want error recorded in the summary", sum.Termination, sum.Error)
	}
	if sum.LoopTermination != model.TermConverged {
		t.Errorf("loop termination = %q, want the loop's own converged preserved alongside the failure", sum.LoopTermination)
	}
	var fin model.JournalRunFinished
	payload(t, f.journal(), model.EvRunFinished, &fin)
	if fin.Termination != model.TermError || fin.Error == "" {
		t.Errorf("run_finished = %+v, want the closing-round failure recorded", fin)
	}
}

// The same failure in the ADVISORY report pass must NOT fail the run. That pass
// hands nothing to the coder -- its findings land in rec.Advisory and rec.Findings
// stays empty -- so there is no partial list for anything downstream to act on, and
// config.ReviewLens.Advisory promises those lenses are excluded from the
// termination condition. Failing here turned a run whose loop converged, and whose
// every fix was verified and committed, into exit 1 because one pinned report lens
// (the shipped fix-code pins maintainability and design) had a bad minute. The
// incomplete report is still worth a warning, and the errors stay on the round.
func TestFinalRoundAdvisoryReviewerFailureDoesNotFailTheRun(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.finalLens() // actionable, on mock2
	// The report lens, pinned to an agent that always fails.
	f.cfg.Agents["bad"] = config.Agent{Command: []string{"false"}, PromptVia: "stdin", Timeout: config.Duration(time.Minute)}
	f.cfg.Roles.Review.Prompts = append(f.cfg.Roles.Review.Prompts, config.ReviewLens{
		Agent: "bad", Prompt: f.cfg.Roles.Review.Prompts[0].Prompt, Final: true, Advisory: true,
	})
	f.respond(1, reviewResponse(t))                  // loop round 1: clean -> converged
	f.respond(2, reviewResponse(t, aFinding("bug"))) // closing fix pass 1
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded it"}))
	f.respond(4, reviewResponse(t)) // closing fix pass 2: clean -> the fix half is done
	// The report then runs on "bad" and fails; it consumes no mock response.

	var logs strings.Builder
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "test.yaml"}},
		func(format string, a ...any) { fmt.Fprintf(&logs, format+"\n", a...) })
	if err != nil {
		t.Fatal(err)
	}
	sum, err := o.Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want success: a failed advisory report is not a failed run", err)
	}
	if sum.Termination != model.TermConverged || sum.Error != "" {
		t.Errorf("termination = %q, error = %q; want converged with no error", sum.Termination, sum.Error)
	}
	// The fix half really did its work, which is what makes the old exit 1 a lie.
	if n := f.commitCount(); n < 2 {
		t.Errorf("repo has %d commit(s), want the closing fix committed on top of the initial one", n)
	}
	last := sum.Rounds[len(sum.Rounds)-1]
	if !last.Final || len(last.ReviewErrors) != 1 {
		t.Fatalf("last round should be the report pass carrying its reviewer error: %+v", last)
	}
	if !strings.Contains(logs.String(), "advisory reviewer(s) failed") {
		t.Errorf("an incomplete report must be warned about, got logs:\n%s", logs.String())
	}
}

// An interruption of the CLOSING phase is an interruption of the run. The loop has
// already recorded its own outcome by then, so a Ctrl-C there used to leave
// "converged" standing: the CLI printed a clean finish and exited 0 (its contract
// is 1 for any interruption) while configured closing work was left undone. The
// loop's verdict is still preserved -- in LoopTermination, the same place a failed
// closing round puts it.
func TestClosingPhaseInterruptionIsReportedAsAnInterruption(t *testing.T) {
	// Ctrl-C while the closing reviewer is running: the agent is killed, so the
	// cancellation reaches Run as that step's error.
	t.Run("during the closing round", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
		f.finalLens()
		f.respond(1, reviewResponse(t)) // round 1: clean -> converged
		// The closing reviewer signals that it started, then blocks until it is killed.
		ready := filepath.Join(f.respDir, "closing-started")
		testfixture.WriteSide(t, f.respDir, 2, fmt.Sprintf("#!/bin/sh\ntouch '%s'\nsleep 30\n", ready))
		f.respond(2, reviewResponse(t))

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go func() {
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				if _, err := os.Stat(ready); err == nil {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			cancel()
		}()

		sum, err := f.orchestrator().Run(ctx)
		if err != nil {
			t.Fatalf("Run() err = %v, want nil: an interruption is a clean stop, not a run failure", err)
		}
		assertInterruptedAfterConverging(t, sum)
	})

	// The same when the cancellation lands before the phase starts: the work is
	// skipped rather than cut off, which is equally not the finish the loop reached.
	t.Run("before the closing round starts", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
		f.finalLens()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		sum := &model.RunSummary{Termination: model.TermConverged}
		if err := f.orchestrator().runFinalPhase(ctx, sum, ""); err != nil {
			t.Fatalf("runFinalPhase() err = %v, want nil: skipping the phase is not a run failure", err)
		}
		if f.invocations() != 0 {
			t.Errorf("the closing phase ran %d agent invocation(s) on a canceled context", f.invocations())
		}
		assertInterruptedAfterConverging(t, sum)
	})
}

// assertInterruptedAfterConverging checks the outcome both halves of the test
// above want: the RUN is an interruption (exit 1), the LOOP's converged is kept.
func assertInterruptedAfterConverging(t *testing.T, sum *model.RunSummary) {
	t.Helper()
	if sum.Termination != model.TermInterrupted {
		t.Errorf("termination = %q, want %q: closing work was left undone", sum.Termination, model.TermInterrupted)
	}
	if sum.LoopTermination != model.TermConverged {
		t.Errorf("loop termination = %q, want the loop's own converged preserved", sum.LoopTermination)
	}
	if got := model.ExitCode(sum.Termination); got != 1 {
		t.Errorf("exit code = %d, want 1: the CLI reports every interruption as a failure", got)
	}
}

// The same for anything else the closing round does: an error raised after the
// loop set its termination must still reach the summary and the journal.
func TestFinalRoundFailureIsRecordedOverTheLoopTermination(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.finalLens()
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	f.respond(1, reviewResponse(t)) // the loop's only round: clean -> converged
	f.respond(2, reviewResponse(t, model.ReviewFinding{
		Category: "tests", Severity: "medium", File: "helper.go", Line: 42, Title: "no test here",
	}))
	// The closing coder's edits break the gate, and its one correction attempt
	// does not repair them: the closing round cannot be committed.
	f.breakBuildOn(3, "broken.txt")
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "done"}))
	f.respond(4, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "still done"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() succeeded although the closing round failed")
	}
	if sum.Termination != model.TermError || sum.Error == "" {
		t.Errorf("termination = %q, error = %q; want the failure recorded", sum.Termination, sum.Error)
	}
	if sum.LoopTermination != model.TermConverged {
		t.Errorf("loop termination = %q, want the loop's converged preserved", sum.LoopTermination)
	}
}

// salvagePartialFix commits a failed coder's partial edits because the NEXT round
// re-reviews them. The closing round has no next round, so the same commit would
// leave work no reviewer ever saw in the repository under a run still reporting
// the loop's success. Stash it and fail instead.
func TestFinalRoundDoesNotSalvageAFailedCoder(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t)) // round 1: clean -> converged
	f.respond(2, reviewResponse(t, model.ReviewFinding{
		Category: "tests", Severity: "medium", File: "helper.go", Line: 42, Title: "no test here",
	}))
	f.editRepoOn(3)
	f.respond(3, "I changed files but forgot the <fix> envelope.") // fails parsing

	before := f.commitCount()
	sum, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() succeeded although the closing round's coder failed after editing")
	}
	if got := f.commitCount(); got != before {
		t.Errorf("repo has %d commits, want %d: partial closing-round work must not be committed unreviewed", got, before)
	}
	if subjects := gitRun(t, f.repo, "log", "--format=%s"); strings.Contains(subjects, "partial") {
		t.Errorf("a partial closing round was committed:\n%s", subjects)
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error", sum.Termination)
	}
	// The work is preserved and the tree restored, like every other discard path.
	if stashes := gitRun(t, f.repo, "stash", "list"); !strings.Contains(stashes, "closing round") {
		t.Errorf("the closing round's edits must be stashed for recovery, got stash list: %q", stashes)
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after discarding the closing round: %q", status)
	}
	if d := f.discarded(model.DiscardFinalCoderFailed); !d.Stashed || d.Error == "" {
		t.Errorf("round_discarded = %+v, want the stash and the coder error recorded", d)
	}
}

// The discard's stash is the part that can itself fail, and that is the outcome
// that matters most: the closing round's edits are still sitting in the worktree,
// under a run whose loop had already converged. The operator has to be told --
// the next run's clean-tree check will refuse to start over dirt they never made.
// An index.lock fails the stash.
func TestFinalRoundFailedCoderStashFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t)) // round 1: clean -> converged
	f.respond(2, reviewResponse(t, model.ReviewFinding{
		Category: "tests", Severity: "medium", File: "helper.go", Line: 42, Title: "no test here",
	}))
	// The closing round's coder dirties the tree and plants an index.lock, so the
	// discarding `git stash` cannot lock the index.
	lock := filepath.Join(f.repo, ".git", "index.lock")
	testfixture.WriteSide(t, f.respDir, 3, fmt.Sprintf("#!/bin/sh\necho 'closing edit' >> '%s'\n: > '%s'\n",
		filepath.Join(f.repo, "main.go"), lock))
	f.respond(3, "I changed files but forgot the <fix> envelope.") // fails parsing

	before := f.commitCount()
	sum, err := f.orchestrator().Run(t.Context())
	if err == nil {
		t.Fatal("Run() err = nil, want the coder failure combined with the reconcile failure")
	}
	for _, want := range []string{"closing round", "coder failed", "could not be reconciled", "left dirty"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Run() err = %v, want it to contain %q", err, want)
		}
	}
	// The tidy outcome must NOT be claimed: nothing was stashed.
	if strings.Contains(err.Error(), "git stash pop") {
		t.Errorf("Run() err = %v, but nothing was stashed; offering recovery would misdirect the operator", err)
	}
	if got := f.commitCount(); got != before {
		t.Errorf("repo has %d commits, want %d: an unreconcilable closing round must still not be committed", got, before)
	}
	if sum.Termination != model.TermError || sum.Error == "" {
		t.Errorf("termination = %q, error = %q; want the failure recorded", sum.Termination, sum.Error)
	}
	if d := f.discarded(model.DiscardFinalCoderFailed); d.Stashed || !strings.Contains(d.Error, "tree left dirty") {
		t.Errorf("round_discarded = %+v, want stashed=false and the dirty tree named", d)
	}
	// Remove the lock so git works again, then confirm the edits are still there
	// rather than lost by a botched reconcile.
	os.Remove(lock)
	if status := gitRun(t, f.repo, "status", "--porcelain"); !strings.Contains(status, "main.go") {
		t.Errorf("the unreconcilable closing-round edits should remain in the worktree, got status %q", status)
	}
}

// The closing round's coder is the same write-capable agent as any other round's,
// so it can reject its issue after having edited files -- often the probe work
// that disproved it. Those edits are accounted for by no verdict, so they must not
// be committed; but a session that did its job must not turn a converged run into
// an error either. The edits are stashed, the tree comes back clean, and the
// loop's own termination stands.
func TestFinalRoundStashesEditsLeftByARejectingCoder(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t)) // round 1: clean -> converged
	f.respond(2, reviewResponse(t, model.ReviewFinding{
		Category: "tests", Severity: "medium", File: "helper.go", Line: 42, Title: "no test here",
	}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "not worth it"}))

	before := f.commitCount()
	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want nil (a rejected session's edits are stashed, not fatal)", err)
	}
	if got := f.commitCount(); got != before {
		t.Errorf("repo has %d commits, want %d: edits under a rejected verdict must not be committed", got, before)
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after the closing round: %q", status)
	}
	if stashes := gitRun(t, f.repo, "stash", "list"); !strings.Contains(stashes, "rejected i1") {
		t.Errorf("the closing round's edits must be stashed for recovery, got stash list: %q", stashes)
	}
	if d := f.sessionDiscarded(model.DiscardRejectedWithEdits); !d.Stashed || d.Issue != "i1" {
		t.Errorf("session_discarded = %+v, want the stash recorded and the issue named", d)
	}
	if sum.Termination != model.TermConverged {
		t.Errorf("termination = %q, want converged: the loop's outcome stands", sum.Termination)
	}
	final := sum.Rounds[len(sum.Rounds)-1]
	if len(final.StashedRejects) != 1 || final.StashedRejects[0] != "i1" {
		t.Errorf("closing round StashedRejects = %v, want [i1]", final.StashedRejects)
	}
}

// A closing round whose findings were all decided in earlier rounds has nothing to
// hand over. Invoking the coder on an empty list would spend a session asking about
// nothing, and a stray verdict for an id it was never given would fail the round --
// turning a run that genuinely reached its terminal state into an error.
func TestFinalRoundSkipsTheCoderWhenEveryIssueWasAlreadyDecided(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.finalLens()
	f.respond(1, reviewResponse(t, aFinding("off by one")))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "works as intended"}))
	// The closing reviewer re-reports the same problem, which the ledger recognizes
	// as the issue round 1 already rejected.
	f.respond(3, reviewResponse(t, aFinding("off by one")))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermAllRejected {
		t.Errorf("termination = %q, want the loop's all-rejected preserved", sum.Termination)
	}
	if got := f.invocations(); got != 3 {
		t.Errorf("mock was invoked %d times, want 3: the closing coder must not run with no work", got)
	}
	last := sum.Rounds[len(sum.Rounds)-1]
	if !last.Final || last.Fixed != 0 {
		t.Fatalf("last round = %+v, want an unmodified closing round", last)
	}
	// The verdict the issue carried must be mirrored onto this round's observations,
	// or the summary's findings block says UNRESOLVED where its issues block says
	// REJECTED, about the same problem.
	if len(last.Findings) != 1 || last.Findings[0].Verdict != model.VerdictRejected {
		t.Errorf("closing round findings = %+v, want the carried rejected verdict mirrored", last.Findings)
	}
}

// In a review-only run there is no closing round -- no coder to hand findings to --
// so a final lens runs in the single round instead. Without that, a review-only
// config would silently drop the lens and stop previewing its fix sibling.
func TestFinalLensRunsInlineForAReviewOnlyRun(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, ReviewOnly: true})
	f.finalLens()
	f.respond(1, reviewResponse(t))
	f.respond(2, reviewResponse(t))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Rounds) != 1 {
		t.Fatalf("got %d rounds, want 1: a review-only run has no closing round", len(sum.Rounds))
	}
	var sawFinalAgent bool
	for _, a := range sum.Rounds[0].Assignments {
		if a.Agent == "mock2" {
			sawFinalAgent = true
		}
	}
	if !sawFinalAgent {
		t.Errorf("the final lens must run inline in a review-only run, got %+v", sum.Rounds[0].Assignments)
	}
}

// ---- commit policy -----------------------------------------------------------

// Every fix gets its own coder session under every policy, so the round's work is
// made and verified one issue at a time. Under per_fix those commits are also what
// lands: one commit per issue, each naming the issue it fixed, so `git revert` can
// undo a single bad fix and a bisect points at one change rather than a batch.
func TestCommitPolicyPerFixCommitsEachIssueSeparately(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerFix})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1, Title: "nil deref"},
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 9, Title: "off by one"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded"}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "bounded"}))
	f.respond(4, reviewResponse(t))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if n := f.coderSessions(1); n != 2 {
		t.Errorf("round 1 spent %d coder session(s), want one per issue", n)
	}
	if got := f.commitCount(); got != 3 {
		t.Errorf("repo has %d commits, want 3 (initial + one per fix)", got)
	}
	// Each commit names its own issue, and neither claims the other's.
	log := gitRun(t, f.repo, "log", "-2", "--format=%s")
	for _, want := range []string{"i1", "nil deref", "i2", "off by one"} {
		if !strings.Contains(log, want) {
			t.Errorf("commit subjects missing %q:\n%s", want, log)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
		if strings.Contains(line, "i1") && strings.Contains(line, "i2") {
			t.Errorf("a per-fix commit subject names both issues: %q", line)
		}
	}
	// Every one of those commits went through the gate on its own, so the summary's
	// round SHA is the last of them rather than a batch nobody verified as a whole.
	if sum.Rounds[0].CommitSHA == "" {
		t.Error("round 1 has no commit SHA")
	}
}

// A per-fix commit body must record what the coder changed, not just where the
// issue was: the location line is already in the subject's title, so a body
// without the verdict detail leaves the one commit that IS the fix saying nothing
// about it. The detail is written onto the round record after the session reports,
// so reading it off the pre-session snapshot silently yields an empty body.
func TestPerFixCommitBodyRecordsTheCodersDetail(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerFix})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 12, Title: "off by one"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{
		ID: "i1", Verdict: "fixed", Detail: "clamped the loop bound to len(xs)",
	}))
	f.respond(3, reviewResponse(t))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	body := gitRun(t, f.repo, "log", "-1", "--format=%b")
	if !strings.Contains(body, "clamped the loop bound to len(xs)") {
		t.Errorf("per-fix commit body is missing the coder's verdict detail:\n%s", body)
	}
	if !strings.Contains(body, "main.go:12") {
		t.Errorf("per-fix commit body is missing the issue location:\n%s", body)
	}
}

// The per-fix subject renders a reviewer-authored title, so it needs the same
// handling the body gets. A title quoting a credential the reviewer found while
// exploring must be masked -- the commit is the one artifact meant to be pushed --
// and a title carrying newlines must not be able to append lines of its own to the
// message, since `git commit -m` takes its argument literally.
func TestPerFixCommitSubjectRedactsAndFlattensTheTitle(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerFix})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1,
			Title: "leaked AKIA1234567890ABCDEF\nSigned-off-by: Someone <nobody@example.com>"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "rotated"}))
	f.respond(3, reviewResponse(t))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if strings.Contains(msg, "AKIA1234567890ABCDEF") {
		t.Errorf("commit message leaks the credential quoted in the title:\n%s", msg)
	}
	if !strings.Contains(msg, "[REDACTED]") {
		t.Errorf("commit message missing the mask (elided instead of masked?):\n%s", msg)
	}
	subject := strings.TrimSpace(gitRun(t, f.repo, "log", "-1", "--format=%s"))
	if !strings.Contains(subject, "Signed-off-by") {
		t.Errorf("subject = %q, want the title flattened onto one line, trailer text and all", subject)
	}
	for _, line := range strings.Split(msg, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Signed-off-by:") {
			t.Errorf("the title forged a line of its own into the commit message:\n%s", msg)
		}
	}
}

// The coder's verdict detail is the LAST line of the per-fix commit body, and a
// lone `token: value` line at the end of a message is what git calls a trailer:
// forging one attributes the commit to someone who never made it, or auto-closes
// an unrelated issue on push. Flattening the detail onto one line does not stop
// that -- one line is already the shape -- so the detail must be rendered so git
// cannot parse it as a trailer, while still saying what the coder did.
func TestPerFixCommitBodyCannotForgeATrailerFromTheDetail(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerFix})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 12, Title: "off by one"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{
		ID: "i1", Verdict: "fixed", Detail: "Signed-off-by: Someone <nobody@example.com>",
	}))
	f.respond(3, reviewResponse(t))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if trailers := strings.TrimSpace(gitRun(t, f.repo, "log", "-1", "--format=%(trailers)")); trailers != "" {
		t.Errorf("the verdict detail forged a git trailer: %q", trailers)
	}
	// Defanged, not dropped: the commit still has to record what the coder reported.
	if body := gitRun(t, f.repo, "log", "-1", "--format=%b"); !strings.Contains(body, "nobody@example.com") {
		t.Errorf("per-fix commit body lost the verdict detail:\n%s", body)
	}
}

// A commit is the most-read thing fixpoint writes -- it is pushed, and every later
// `git log`/`git show` renders it in someone's terminal -- so the agent-authored
// text in it must be defanged the way the prompts and logs are. Collapsing
// whitespace does not do it: ESC and the bidi overrides are not unicode.IsSpace,
// git stores the message verbatim, and OSC 52 would then write the reader's
// clipboard while a bidi override rewrites what the finding appears to say.
func TestPerFixCommitMessageDefangsTerminalControlCharacters(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerFix})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 12,
			Title: "off by one\x1b]52;c;cGF5bG9hZA==\x07"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{
		ID: "i1", Verdict: "fixed", Detail: "clamped \x1b[2Jthe bound\u202egnorw si siht",
	}))
	f.respond(3, reviewResponse(t))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if strings.ContainsAny(msg, "\x1b\x07") || strings.ContainsRune(msg, '\u202e') {
		t.Errorf("commit message carries raw terminal control characters:\n%q", msg)
	}
	// Escaped, not dropped: the commit still records what the agents reported.
	for _, want := range []string{"off by one", "clamped", "the bound", `\x1b`, `\u202e`} {
		if !strings.Contains(msg, want) {
			t.Errorf("commit message is missing %q:\n%s", want, msg)
		}
	}
}

// per_round regroups the round's per-fix commits into the single commit fixpoint
// has always produced. The squash is `reset --soft`, so the tree must be identical
// to what the per-fix commits already verified -- and the body describes the round.
func TestCommitPolicyPerRoundSquashesTheRound(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerRound})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1, Title: "nil deref"},
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 9, Title: "off by one"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded"}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "bounded"}))
	f.respond(4, reviewResponse(t))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	// Still one session per issue: the policy groups commits, it does not batch work.
	if n := f.coderSessions(1); n != 2 {
		t.Errorf("round 1 spent %d coder session(s), want one per issue", n)
	}
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + one squashed round)", got)
	}
	// A round-shaped header, not one of the per-fix ones it replaced.
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "round 1 (2 fixed, 0 rejected)") {
		t.Errorf("squashed commit message = %q, want a round-shaped header", msg)
	}
	for _, want := range []string{"nil deref", "off by one"} {
		if !strings.Contains(msg, want) {
			t.Errorf("squashed commit body missing %q:\n%s", want, msg)
		}
	}
	// The summary must cite the squash, not a per-fix commit the squash removed.
	head := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "HEAD"))
	if sum.Rounds[0].CommitSHA != head {
		t.Errorf("round 1 CommitSHA = %q, want the squash %q", sum.Rounds[0].CommitSHA, head)
	}
	// Squashing must not change content: a `reset --soft` cannot, and a clean tree
	// afterwards is what proves the commit holds exactly what was verified.
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("tree left dirty by the squash: %q", status)
	}
}

// A round that ends in a salvage still obeys per_round: its earlier issues were
// each committed before the later coder session failed, so without the squash the
// round leaves those per-fix commits and the salvage commit standing separately --
// exactly the layout the policy says not to produce. The squashed message must
// still say the round is partial, since part of what it holds has no verdict.
func TestCommitPolicyPerRoundSquashesASalvagedRound(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerRound})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1, Title: "nil deref"},
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 9, Title: "off by one"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded"}))
	// The second session edits and then dies without reporting: its work is salvaged.
	f.editRepoOn(3)
	f.respond(3, "I changed files but forgot the <fix> envelope.")
	f.respond(4, reviewResponse(t)) // round 2: clean

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want salvage + continue", err)
	}
	if sum.Termination != model.TermConverged {
		t.Fatalf("termination = %q, want converged after the salvaged round", sum.Termination)
	}
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + one squashed round)", got)
	}
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "(partial, coder failed)") {
		t.Errorf("squashed commit message = %q, want it still labeled partial", msg)
	}
	// Both halves of what the commit holds: the verdict that landed, and the issue
	// left undecided when the coder failed.
	for _, want := range []string{"nil deref", "off by one", "Coder failed before reporting verdicts"} {
		if !strings.Contains(msg, want) {
			t.Errorf("squashed commit message missing %q:\n%s", want, msg)
		}
	}
	assertNoVerdictSection(t, msg, []string{"off by one"}, []string{"nil deref"})
	head := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "HEAD"))
	if sum.Rounds[0].CommitSHA != head || len(sum.Rounds[0].Commits) != 1 {
		t.Errorf("round 1 CommitSHA = %q, Commits = %v; want just the squash %q",
			sum.Rounds[0].CommitSHA, sum.Rounds[0].Commits, head)
	}
	if sum.Rounds[0].CoderError == "" {
		t.Error("round 1 CoderError not recorded; the squash must not lose the coder failure")
	}
	// A squash is a `reset --soft`: it cannot change content, and a clean tree is
	// what proves the commit holds exactly what the per-fix commits verified.
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("tree left dirty by the squash: %q", status)
	}
}

// The other half of per_round's salvage rule: a round whose ONLY commit is the
// salvage commit has nothing to regroup, and re-squashing it would rewrite a commit
// that already carries exactly the right message -- adding an empty "Fixed:" section
// to a round where, by definition, nobody reported a verdict.
func TestCommitPolicyPerRoundLeavesALoneSalvageCommitAlone(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerRound})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// The round's single session edits and then dies without reporting.
	f.editRepoOn(2)
	f.respond(2, "I changed files but forgot the <fix> envelope.")
	f.respond(3, reviewResponse(t)) // round 2: clean

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want salvage + continue", err)
	}
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + the salvage commit)", got)
	}
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "(partial, coder failed)") || !strings.Contains(msg, "bug") {
		t.Errorf("salvage commit message = %q", msg)
	}
	if strings.Contains(msg, "Fixed:") {
		t.Errorf("the lone salvage commit was re-squashed and now claims a verdict section:\n%s", msg)
	}
	if len(sum.Rounds[0].Commits) != 1 {
		t.Errorf("round 1 Commits = %v, want just the salvage commit", sum.Rounds[0].Commits)
	}
}

// The DEFAULT per_fix policy must leave a salvaged round's commits standing apart.
// This is the round shape salvage exists for -- a coder that fixes one issue, commits,
// then dies on the next -- so squashing here would be the common case, not an edge: it
// would drop the per-fix granularity the policy is asked for AND relabel a verified fix
// as "(partial, coder failed)", work nobody reported a verdict for.
func TestCommitPolicyPerFixKeepsASalvagedRoundsCommitsApart(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerFix})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1, Title: "nil deref"},
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 9, Title: "off by one"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded"}))
	// The second session edits and then dies without reporting: its work is salvaged.
	f.editRepoOn(3)
	f.respond(3, "I changed files but forgot the <fix> envelope.")
	f.respond(4, reviewResponse(t)) // round 2: clean

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want salvage + continue", err)
	}
	if got := f.commitCount(); got != 3 {
		t.Errorf("repo has %d commits, want 3 (initial + the per-fix commit + the salvage commit)", got)
	}
	if got := sum.Rounds[0].Commits; len(got) != 2 {
		t.Errorf("round 1 Commits = %v, want the per-fix commit and the salvage commit kept separate", got)
	}
	// The per-fix commit keeps its OWN message: it names the issue it fixed, and no
	// squash has folded it under the salvage header.
	subjects := gitRun(t, f.repo, "log", "--format=%s")
	if !strings.Contains(subjects, "fixpoint: i1 — nil deref") {
		t.Errorf("the verified per-fix commit lost its own message:\n%s", subjects)
	}
	// HEAD is the salvage commit, and it accounts only for the issue left undecided.
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "(partial, coder failed)") {
		t.Errorf("HEAD message = %q, want the salvage commit on top", msg)
	}
	assertNoVerdictSection(t, msg, []string{"off by one"}, []string{"nil deref"})
	head := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "HEAD"))
	if sum.Rounds[0].CommitSHA != head {
		t.Errorf("round 1 CommitSHA = %q, want the salvage commit %q", sum.Rounds[0].CommitSHA, head)
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after the salvaged round: %q", status)
	}
}

// assertNoVerdictSection checks the "Issues left without a verdict" section of a
// salvage commit message names exactly the undecided issues: an issue an earlier
// session of the same round fixed appears in the Fixed section, and listing it
// here too would make one commit message contradict itself.
func assertNoVerdictSection(t *testing.T, msg string, want, unwanted []string) {
	t.Helper()
	const header = "Issues left without a verdict:"
	_, section, ok := strings.Cut(msg, header)
	if !ok {
		t.Fatalf("commit message has no %q section:\n%s", header, msg)
	}
	// A per_run squash concatenates one section per salvaged round; stop at the next.
	if next, _, cut := strings.Cut(section, "\nRound "); cut {
		section = next
	}
	for _, w := range want {
		if !strings.Contains(section, w) {
			t.Errorf("no-verdict section missing undecided issue %q:\n%s", w, section)
		}
	}
	for _, u := range unwanted {
		if strings.Contains(section, u) {
			t.Errorf("no-verdict section names %q, which already has a verdict:\n%s", u, section)
		}
	}
}

// per_run collapses the whole run, loop rounds and closing passes together, into
// one commit -- and only at the very end, so the closing round still reviews the
// per-fix history it builds on.
func TestCommitPolicyPerRunSquashesTheWholeRun(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerRun})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	// Round 2 finds a second problem, so the run produces commits in two rounds. Its
	// title must share no tokens with round 1's, or the ledger matches the two as one
	// issue (same file + titles agree) and round 2 has nothing new to fix.
	f.respond(3, reviewResponse(t, model.ReviewFinding{
		Category: "concurrency", Severity: "high", File: "other.go", Line: 42, Title: "unsynchronized map write",
	}))
	f.editRepoOn(4)
	f.respond(4, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "took the lock"}))
	f.respond(5, reviewResponse(t))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if len(sum.Rounds) < 2 {
		t.Fatalf("rounds = %d, want at least 2 so the run has commits to collapse", len(sum.Rounds))
	}
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + one for the whole run)", got)
	}
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	// Both rounds' work is described by the one commit that replaced them.
	for _, want := range []string{"bug", "unsynchronized map write"} {
		if !strings.Contains(msg, want) {
			t.Errorf("run commit body missing %q:\n%s", want, msg)
		}
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("tree left dirty by the squash: %q", status)
	}
	// Exactly one round cites the squash and no round still cites a per-fix commit the
	// squash removed: every SHA the summary names must be reachable from HEAD, or the
	// scoreboard's commits row -- which falls back to CommitSHA when Commits is empty,
	// so it counts these too -- reports one commit per round that committed, each
	// naming an object `git log` cannot show.
	head := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "HEAD"))
	reachable := map[string]bool{}
	for _, sha := range strings.Fields(gitRun(t, f.repo, "rev-list", "HEAD")) {
		reachable[sha] = true
	}
	var cited []int
	for i, r := range sum.Rounds {
		for _, sha := range r.Commits {
			if !reachable[sha] {
				t.Errorf("round %d lists commit %q, which the squash removed from history", r.Round, sha)
			}
		}
		if r.CommitSHA != "" && !reachable[r.CommitSHA] {
			t.Errorf("round %d cites CommitSHA %q, which the squash removed from history", r.Round, r.CommitSHA)
		}
		if r.CommitSHA != "" {
			cited = append(cited, i)
			if r.CommitSHA != head {
				t.Errorf("round %d CommitSHA = %q, want the squash %q", r.Round, r.CommitSHA, head)
			}
			// The squash belongs to a round whose work is in it, not to the trailing
			// clean round that committed nothing.
			if r.Fixed == 0 {
				t.Errorf("round %d committed nothing yet names the squash %q", r.Round, r.CommitSHA)
			}
		}
	}
	if len(cited) != 1 {
		t.Errorf("rounds naming a commit = %v, want exactly one (the squash)", cited)
	}
	if _, err := os.Stat(filepath.Join(f.repo, ".git", "HEAD")); err != nil {
		t.Errorf("repository damaged by the squash: %v", err)
	}
}

// per_run collapses a salvaged round into the run's ONE commit, so it is the shape
// where the fact that part of that commit has no verdict behind it is lost for good
// if the message does not say so. The aggregated Fixed/Rejected sections cannot say
// it: the failed coder reported no verdict at all.
func TestCommitPolicyPerRunKeepsASalvagedRoundVisible(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerRun})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1, Title: "nil deref"},
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 9, Title: "off by one"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded"}))
	// The second session edits and then dies without reporting: its work is salvaged.
	f.editRepoOn(3)
	f.respond(3, "I changed files but forgot the <fix> envelope.")
	f.respond(4, reviewResponse(t)) // round 2: clean

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want salvage + continue", err)
	}
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + one for the whole run)", got)
	}
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "(partial, coder failed)") {
		t.Errorf("run commit message = %q, want it labeled partial", msg)
	}
	// Both halves of what the commit holds: the verdict that landed, and the issue
	// the failed coder left undecided, named by the round it was left in.
	for _, want := range []string{"nil deref", "off by one", "Coder failed before reporting verdicts", "Round 1:"} {
		if !strings.Contains(msg, want) {
			t.Errorf("run commit message missing %q:\n%s", want, msg)
		}
	}
	assertNoVerdictSection(t, msg, []string{"off by one"}, []string{"nil deref"})
	if sum.Rounds[0].CoderError == "" {
		t.Error("round 1 CoderError not recorded; the squash must not lose the coder failure")
	}
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("tree left dirty by the squash: %q", status)
	}
}

// A run can salvage more than once, and per_run collapses every one of those rounds
// into the SAME commit. The coder error and the issues left undecided differ per
// round, so the one message must carry a section per salvaged round: a reader given
// only the first (or only the last) would believe the run's other rounds were fully
// accounted for by the Fixed/Rejected sections.
func TestCommitPolicyPerRunKeepsEverySalvagedRoundVisible(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerRun})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1, Title: "nil deref"},
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 9, Title: "off by one"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded"}))
	// Round 1's second session edits and then dies without reporting: salvage #1.
	f.editRepoOn(3)
	f.respond(3, "I changed files but forgot the <fix> envelope.")
	// Round 2 reports something new -- a title sharing no tokens with round 1's, or the
	// ledger matches it to an issue already decided -- and its coder dies too: salvage #2.
	f.respond(4, reviewResponse(t, model.ReviewFinding{
		Category: "concurrency", Severity: "high", File: "other.go", Line: 42, Title: "unsynchronized map write",
	}))
	f.editRepoOn(5)
	f.respond(5, "I changed files but forgot the <fix> envelope.")
	f.respond(6, reviewResponse(t)) // round 3: clean

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() err = %v, want two salvages + continue", err)
	}
	if len(sum.Rounds) < 3 {
		t.Fatalf("rounds = %d, want 3 so the run salvages twice before converging", len(sum.Rounds))
	}
	for _, r := range []int{0, 1} {
		if sum.Rounds[r].CoderError == "" {
			t.Errorf("round %d CoderError not recorded; both rounds must count as salvaged", r+1)
		}
	}
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + one for the whole run)", got)
	}
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "(partial, coder failed)") {
		t.Errorf("run commit message = %q, want it labeled partial", msg)
	}
	// One section per salvaged round, each carrying its own coder error.
	for _, want := range []string{"Round 1:", "Round 2:"} {
		if !strings.Contains(msg, want) {
			t.Errorf("run commit message missing the %q salvage section:\n%s", want, msg)
		}
	}
	if n := strings.Count(msg, "Coder failed before reporting verdicts"); n != 2 {
		t.Errorf("run commit message reports %d coder failure(s), want one per salvaged round:\n%s", n, msg)
	}
	// Each section names only ITS round's undecided issue: round 1 left "off by one"
	// (its "nil deref" was fixed and verified), round 2 left its own finding.
	assertNoVerdictSection(t, msg, []string{"off by one"}, []string{"nil deref", "unsynchronized map write"})
	_, second, ok := strings.Cut(msg, "\nRound 2:")
	if !ok {
		t.Fatalf("run commit message has no second salvage section:\n%s", msg)
	}
	assertNoVerdictSection(t, second, []string{"unsynchronized map write"}, []string{"nil deref", "off by one"})
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("tree left dirty by the squash: %q", status)
	}
}

// A salvaged round owes its commit message every issue it left undecided: the one
// the coder died on AND the ones no session was ever handed. A round that dies on
// its LAST issue cannot tell those two apart -- one issue answers either rule -- so
// this round dies in the MIDDLE: i1 is fixed and committed, i2's session dies, and
// i3 is never reached at all.
//
// The two commit shapes legitimately disagree about i3, and both sides are pinned
// here. A squash (per_round, per_run) covers the whole round, so its no-verdict
// section must name i2 AND i3. The per_fix salvage commit stands alone and holds
// only the edits of the session that died, so it names only i2.
func TestSalvageCommitNamesTheIssuesTheRoundNeverReached(t *testing.T) {
	for _, tc := range []struct {
		name     string
		policy   string
		commits  int
		want     []string
		unwanted []string
	}{
		{
			name: "per_round", policy: config.CommitPerRound, commits: 2,
			want:     []string{"off by one", "unsynchronized map write"},
			unwanted: []string{"nil deref"},
		},
		{
			name: "per_run", policy: config.CommitPerRun, commits: 2,
			want:     []string{"off by one", "unsynchronized map write"},
			unwanted: []string{"nil deref"},
		},
		{
			// initial + the verified per-fix commit + the salvage commit.
			name: "per_fix", policy: config.CommitPerFix, commits: 3,
			want:     []string{"off by one"},
			unwanted: []string{"nil deref", "unsynchronized map write"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1, CommitPolicy: tc.policy})
			// Three findings, so the round has an issue left BEHIND the one it dies on.
			// The titles share no tokens, or the ledger would fold them into one issue.
			f.respond(1, reviewResponse(t,
				model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1, Title: "nil deref"},
				model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 9, Title: "off by one"},
				model.ReviewFinding{Category: "concurrency", Severity: "medium", File: "other.go", Line: 42, Title: "unsynchronized map write"},
			))
			f.editRepoOn(2)
			f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded"}))
			// i2's session edits and then dies without reporting. The round stops there,
			// so i3 never reaches a coder session.
			f.editRepoOn(3)
			f.respond(3, "I changed files but forgot the <fix> envelope.")
			f.respond(4, reviewResponse(t)) // round 2: clean

			sum, err := f.orchestrator().Run(t.Context())
			if err != nil {
				t.Fatalf("Run() err = %v, want salvage + continue", err)
			}
			if got := f.invocations(); got != 4 {
				t.Fatalf("mock invoked %d time(s), want 4: i3 must never be handed to a session", got)
			}
			if got := f.commitCount(); got != tc.commits {
				t.Errorf("repo has %d commits, want %d", got, tc.commits)
			}
			msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
			if !strings.Contains(msg, "(partial, coder failed)") {
				t.Errorf("commit message = %q, want it labeled partial", msg)
			}
			assertNoVerdictSection(t, msg, tc.want, tc.unwanted)
			if sum.Rounds[0].CoderError == "" {
				t.Error("round 1 CoderError not recorded; the salvage must not lose the coder failure")
			}
			if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
				t.Errorf("tree left dirty after the salvaged round: %q", status)
			}
		})
	}
}

// A review-only run never commits, so per_run has nothing to collapse -- and its
// run base is deliberately left unresolved. Squashing to that empty base would build
// a ROOT commit from the current index and reset the branch onto it, cutting the
// repository's history off from the branch in a run the operator asked to be
// read-only. `-review-only` over a config that sets per_run reaches this with no YAML
// edit at all.
func TestCommitPolicyPerRunLeavesTheBranchAloneInAReviewOnlyRun(t *testing.T) {
	f := newFixture(t, config.Loop{
		MaxIterations: 3, CleanRoundsToStop: 1, ReviewOnly: true, CommitPolicy: config.CommitPerRun,
	})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	before := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "HEAD"))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Termination != model.TermReviewOnly {
		t.Fatalf("termination = %q, want review-only", sum.Termination)
	}
	// The decisive check: a root-commit rewrite leaves the same commit COUNT, so only
	// HEAD itself shows whether the branch was rebuilt.
	if after := strings.TrimSpace(gitRun(t, f.repo, "rev-parse", "HEAD")); after != before {
		t.Errorf("HEAD moved from %s to %s: a review-only run must not rewrite the branch", before, after)
	}
	if got := f.commitCount(); got != 1 {
		t.Errorf("repo has %d commits, want the 1 it started with", got)
	}
}

// An INTERRUPTION of the closing phase must keep them for the same reason -- and it
// is the one unvouched-for tree that reaches the squash's caller, because a
// cancellation there is deliberately reported as nil rather than as a failure. What
// holds the line is finishRun's own ctx check, not the squash's first git call
// happening to fail on the canceled context.
func TestCommitPolicyPerRunKeepsCommitsWhenTheClosingPhaseIsInterrupted(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerRun})
	f.finalLens()
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, reviewResponse(t)) // round 2: clean -> converged
	// The closing reviewer signals that it started, then blocks until it is killed.
	ready := filepath.Join(f.respDir, "closing-started")
	testfixture.WriteSide(t, f.respDir, 4, fmt.Sprintf("#!/bin/sh\ntouch '%s'\nsleep 30\n", ready))
	f.respond(4, reviewResponse(t))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()

	sum, err := f.orchestrator().Run(ctx)
	if err != nil {
		t.Fatalf("Run() err = %v, want nil: an interruption is a clean stop, not a run failure", err)
	}
	if sum.Termination != model.TermInterrupted {
		t.Fatalf("termination = %q, want %q", sum.Termination, model.TermInterrupted)
	}
	// A squash of this run would also leave two commits, so only the SUBJECT says
	// which history survived: the per-fix header carries the issue id, the squashed
	// one is round-shaped (see roundCommitMessage).
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + the one per-fix commit)", got)
	}
	if subject := gitRun(t, f.repo, "log", "-1", "--format=%s"); !strings.Contains(subject, "i1") {
		t.Errorf("HEAD subject = %q, want the per-fix commit kept intact", subject)
	}
}

// A run that fails must keep its per-fix commits whatever the policy says. They are
// how an operator sees how far it got, and rewriting history over a tree nobody
// vouched for would destroy exactly that.
func TestCommitPolicyPerRunKeepsCommitsWhenTheRunFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, CommitPolicy: config.CommitPerRun})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1, Title: "real bug"},
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 9, Title: "claimed but not done"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	// The second session claims a fix without touching the tree, which fails the run
	// after the first fix has already been committed.
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "lied"}))

	if _, err := f.orchestrator().Run(t.Context()); err == nil {
		t.Fatal("expected the run to fail on the fabricated fix")
	}
	if got := f.commitCount(); got != 2 {
		t.Errorf("repo has %d commits, want 2 (initial + the one fix that really happened)", got)
	}
	msg := gitRun(t, f.repo, "log", "-1", "--format=%s")
	if !strings.Contains(msg, "i1") {
		t.Errorf("the surviving commit should be the real fix, got %q", msg)
	}
}

// logs.pattern renders one artifact path per (role, agent, prompt, round,
// timestamp), and timestamp_format is second-granularity -- so several fix sessions
// in one round would write the same files and silently overwrite each other. The
// coder's artifact identity carries the issue for exactly that reason: losing the
// prompt and raw output of a fix means losing the only record of what was asked.
func TestPerFixSessionsWriteDistinctArtifacts(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 2, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "critical", File: "main.go", Line: 1, Title: "nil deref"},
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 9, Title: "off by one"},
	))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "guarded"}))
	f.editRepoOn(3)
	f.respond(3, fixResponse(t, model.FixResult{ID: "i2", Verdict: "fixed", Detail: "bounded"}))
	f.respond(4, reviewResponse(t))

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	files := f.coderPromptFiles(1)
	if len(files) != 2 {
		t.Fatalf("round 1 wrote %d coder prompt files, want one per fix session: %v", len(files), files)
	}
	// Distinct paths, and each names the issue its session was given.
	seen := map[string]bool{}
	for _, name := range files {
		base := filepath.Base(name)
		if seen[base] {
			t.Errorf("two fix sessions wrote the same artifact %q", base)
		}
		seen[base] = true
	}
	got := make([]string, 0, len(seen))
	for base := range seen {
		got = append(got, base)
	}
	joined := strings.Join(got, " ")
	for _, id := range []string{"i1", "i2"} {
		if !strings.Contains(joined, id) {
			t.Errorf("no fix artifact names issue %s: %v", id, got)
		}
	}
	// The prompts must differ too: each session is asked about its own issue only.
	for _, name := range files {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "[i1]") && strings.Contains(string(b), "[i2]") {
			t.Errorf("%s asks about both issues; a session must get exactly one", filepath.Base(name))
		}
	}
}

// gatedScript writes an agent that drains stdin and then blocks until gate
// exists, so a test owns the exact moment the invocation ends.
func gatedScript(t *testing.T, gate string) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "gated.sh")
	body := "#!/bin/sh\ncat > /dev/null\nwhile [ ! -f '" + gate + "' ]; do sleep 0.01; done\necho done\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return script
}

// A tick queued at the moment the agent finishes leaves both of the heartbeat's
// channels ready, and select picks between ready cases at random -- so without
// the inner done re-check the tick branch wins about half the time and prints
// "still running" for an agent that has already returned. One iteration cannot
// tell the two apart; the loop makes a missing re-check a near-certain failure.
func TestHeartbeatSkipsTickAfterDone(t *testing.T) {
	for i := range 200 {
		done := make(chan struct{})
		close(done)
		ticks := make(chan time.Time, 1)
		ticks <- time.Now()

		logged := 0
		heartbeat(done, ticks, func() { logged++ })
		if logged != 0 {
			t.Fatalf("iteration %d: logged %d progress lines after done closed, want 0", i, logged)
		}
	}
}

// A heartbeat tick already inside logf when the agent exits must finish BEFORE
// runAgent returns: otherwise the "still running" line lands after the caller
// has moved on, interleaved with whatever it logs next -- or with the
// end-of-run table, whose column alignment it shreds. hb.Wait is the only thing
// enforcing that, and the 5-minute default interval puts the tick out of reach
// of any test, so the field seam is what makes the join observable at all.
// Deleting hb.Wait turns the ordering below around.
func TestRunAgentWaitsForInFlightHeartbeat(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
	gate := filepath.Join(t.TempDir(), "gate")
	f.cfg.Agents["gated"] = config.Agent{
		Command: []string{gatedScript(t, gate)}, PromptVia: "stdin", Timeout: config.Duration(time.Minute),
	}

	// The recorded transcript: heartbeat lines as they complete, plus a marker
	// appended the instant runAgent returns. Its own mutex, so -race sees any
	// unsynchronized write.
	var mu sync.Mutex
	var lines []string
	record := func(s string) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, s)
	}

	entered := make(chan struct{}) // the first heartbeat has reached logf...
	release := make(chan struct{}) // ...and is held there until the test says so
	var hold sync.Once
	o := f.orchestrator()
	o.heartbeatEvery = 10 * time.Millisecond
	o.logf = func(format string, args ...any) {
		hold.Do(func() {
			close(entered)
			<-release
		})
		record(fmt.Sprintf(format, args...))
	}

	const marker = "<runAgent returned>"
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		o.runAgent(t.Context(), "review: gated via lens", "review", "gated", "lens", 1, "prompt")
		record(marker)
	}()

	<-entered
	// The tick is now blocked inside logf. Let the agent exit: close(done) and the
	// return are free to happen immediately, and only the join holds them back.
	if err := os.WriteFile(gate, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond) // ample room for an unjoined return to record first
	close(release)
	<-returned

	mu.Lock()
	got := append([]string(nil), lines...)
	mu.Unlock()

	if len(got) < 2 {
		t.Fatalf("transcript = %q, want at least one heartbeat line and the return marker", got)
	}
	if got[len(got)-1] != marker {
		t.Errorf("transcript = %q, want %q last: runAgent returned while a heartbeat was still printing", got, marker)
	}
	for _, l := range got[:len(got)-1] {
		if !strings.Contains(l, "still running") || !strings.Contains(l, "review: gated via lens") {
			t.Errorf("transcript line %q is not a labeled heartbeat: %q", l, got)
		}
	}
}

// A red baseline means opposite things under the two policies -- tolerated under
// no_regressions, fatal to every round under must_pass, because Report.Blocking
// reads the baseline only on the first path. The narration is the operator's only
// warning before round 1, so it is pinned here per policy.
func TestVerifyBaselineNarrationMatchesThePolicy(t *testing.T) {
	tests := []struct {
		policy   config.VerifyPolicy
		wants    []string
		notWants []string
	}{
		{
			policy:   config.VerifyNoRegressions,
			wants:    []string{"pre-existing", "permits them to keep failing"},
			notWants: []string{"WILL block every round"},
		},
		{
			policy:   config.VerifyMustPass,
			wants:    []string{"pre-existing", "must_pass ignores the baseline", "WILL block every round"},
			notWants: []string{"permits them to keep failing"},
		},
	}
	for _, tc := range tests {
		t.Run(string(tc.policy), func(t *testing.T) {
			var log strings.Builder
			o := &Orchestrator{
				cfg: &config.Config{Verify: config.Verify{
					Policy:   tc.policy,
					Timeout:  config.Duration(30 * time.Second),
					Commands: []config.VerifyCommand{{Name: "red", Run: []string{"false"}}},
				}},
				logf:         func(format string, args ...any) { fmt.Fprintf(&log, format+"\n", args...) },
				journalWrite: func(string, int, any) error { return nil },
			}
			o.captureVerifyBaseline(t.Context())

			if o.verifyBaseline.Passed() {
				t.Fatal("baseline passed: the test needs a failing command to narrate")
			}
			for _, want := range tc.wants {
				if !strings.Contains(log.String(), want) {
					t.Errorf("baseline narration is missing %q:\n%s", want, log.String())
				}
			}
			for _, no := range tc.notWants {
				if strings.Contains(log.String(), no) {
					t.Errorf("baseline narration under %s must not say %q:\n%s", tc.policy, no, log.String())
				}
			}
		})
	}
}

// End to end: the closing round is not shown the test files the run itself wrote.
//
// This is the wiring that makes loop.final_skip_run_edits mean anything -- the glob
// has to be matched against THIS RUN's commits, computed at the point the loop stops
// editing, and applied to the material the closing reviewers actually receive. The
// unit test in the target package proves the collector narrows; only a whole run
// proves the orchestrator hands it the right paths at the right moment.
//
// Everything else stays in scope on purpose: the run's non-test edits (where a
// regression a closing fix introduced would show up) and pre-existing test files.
func TestClosingRoundIsNotShownTheTestFilesThisRunWrote(t *testing.T) {
	f := newFixture(t, config.Loop{
		MaxIterations:     3,
		CleanRoundsToStop: 1,
		FinalSkipRunEdits: []string{"**/*_test.go"},
	})
	f.finalLens()
	// A test file that was already there when the run started: it matches the glob,
	// but the run did not write it, so the closing round must still see it.
	if err := os.WriteFile(filepath.Join(f.repo, "old_test.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitRun(t, f.repo, "add", ".")
	gitRun(t, f.repo, "commit", "-q", "-m", "pre-existing test")

	f.respond(1, reviewResponse(t, aFinding("off by one")))
	// The fix writes both a test file and a source file, exactly as a real coder
	// asked for a regression test would.
	testfixture.WriteSide(t, f.respDir, 2, fmt.Sprintf(
		"#!/bin/sh\necho 'package main' > '%s'\necho '// fixed' >> '%s'\n",
		filepath.Join(f.repo, "new_test.go"), filepath.Join(f.repo, "main.go")))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "fixed with a test"}))
	f.respond(3, reviewResponse(t)) // round 2: clean -> converged
	f.respond(4, reviewResponse(t)) // closing pass: nothing left

	if _, err := f.orchestrator().Run(t.Context()); err != nil {
		t.Fatal(err)
	}

	closing := f.reviewPrompt(3)
	if closing == "" {
		t.Fatal("no closing-round reviewer prompt was written")
	}
	if strings.Contains(closing, "new_test.go") {
		t.Errorf("the closing round was shown a test file this run wrote:\n%s", closing)
	}
	for _, want := range []string{"main.go", "old_test.go"} {
		if !strings.Contains(closing, want) {
			t.Errorf("the closing round lost %q, which final_skip_run_edits must not hide:\n%s", want, closing)
		}
	}
	// The loop's own rounds keep full scope: reviewing what an earlier round changed
	// is how a fix's own bug gets caught, and narrowing that was never the point.
	if r2 := f.reviewPrompt(2); r2 != "" && !strings.Contains(r2, "new_test.go") {
		t.Errorf("a LOOP round lost the run's own test file; only the closing round narrows:\n%s", r2)
	}
}
