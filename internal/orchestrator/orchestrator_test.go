package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
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
	o, err := New(f.cfg, config.Source{Config: "test.yaml"}, f.t.Logf)
	if err != nil {
		f.t.Fatal(err)
	}
	return o
}

// respond registers the mock agent's n-th response.
func (f *fixture) respond(n int, content string) { testfixture.Respond(f.t, f.respDir, n, content) }

// editRepoOn makes the mock agent's n-th invocation edit the repo before
// answering, simulating a coder fix.
func (f *fixture) editRepoOn(n int) { testfixture.EditRepoOn(f.t, f.respDir, f.repo, n) }

// invocations reports how many times the mock agent has been called.
func (f *fixture) invocations() int { return testfixture.Invocations(f.t, f.respDir) }

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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "patched"}))
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
	msg := gitRun(t, f.repo, "log", "-1", "--format=%B")
	if !strings.Contains(msg, "round 1 (1 fixed, 0 rejected)") {
		t.Errorf("round commit message = %q", msg)
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "rejected", Detail: "by design"}))

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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "rejected", Detail: "by design"}))

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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "d"}))
	f.respond(3, reviewResponse(t, aFinding("bug two")))
	f.editRepoOn(4)
	f.respond(4, fixResponse(t, model.FixResult{ID: "r2.1", Verdict: "fixed", Detail: "d"}))

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
	f.respond(2, "no review block") // reviewer error: streak resets
	f.respond(3, reviewResponse(t)) // clean (streak 1)
	f.respond(4, reviewResponse(t)) // clean (streak 2 -> converged)

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

func TestRunCoderContractViolationFailsRound(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// Coder answers about a finding that does not exist.
	f.respond(2, fixResponse(t, model.FixResult{ID: "r9.9", Verdict: "fixed", Detail: "d"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "unknown finding id") {
		t.Fatalf("Run() err = %v, want unknown finding id error", err)
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
	if _, err := New(f.cfg, config.Source{Config: "test.yaml"}, t.Logf); err == nil || !strings.Contains(err.Error(), "logs.dir") {
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "patched"}))
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
func TestRunRejectsSymlinkedLogsDir(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.cfg.Logs.Dir = filepath.Join(f.repo, "logs") // logs under the target root
	// Redirect the logs dir out of the repo via a symlink; the lexical exclusion
	// ("logs") still hides it from the clean check, so only the symlink guard can
	// catch the redirection.
	if err := os.Symlink(t.TempDir(), filepath.Join(f.repo, "logs")); err != nil {
		t.Fatal(err)
	}
	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Run() = %v, want symlink rejection", err)
	}
	if !strings.Contains(err.Error(), filepath.Join(f.repo, "logs")) {
		t.Errorf("error should name the offending path: %v", err)
	}
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
	if status := gitRun(t, f.repo, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("working tree left dirty after failed commit; want clean (stashed): %q", status)
	}
	if list := gitRun(t, f.repo, "stash", "list"); !strings.Contains(list, "failed round 1") {
		t.Errorf("expected a stash entry for the recovered edits, got %q", list)
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "patched"}))

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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "patched"}))

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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "claimed"}))

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

// An all-rejected verdict sitting on a dirty tree is a hard error: rejecting
// every finding yet editing the repo is contradictory, and the changes must not
// be left uncommitted under a "success" termination.
func TestRunAllRejectedWithEditFails(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t, aFinding("bug")))
	// Verdict "rejected" for every finding, yet the coder dirties the tree.
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "rejected", Detail: "by design"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "rejected every finding yet modified") {
		t.Fatalf("Run() err = %v, want rejected-yet-modified error", err)
	}
	if sum.Termination != model.TermError {
		t.Errorf("termination = %q, want error", sum.Termination)
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
	f.respond(1, reviewResponse(t, low, high, med)) // ids r1.1(low) r1.2(high) r1.3(med)
	f.editRepoOn(2)
	f.respond(2, fixResponse(t,
		model.FixResult{ID: "r1.2", Verdict: "fixed", Detail: "fixed"},
		model.FixResult{ID: "r1.3", Verdict: "fixed", Detail: "fixed"},
	))
	f.respond(3, reviewResponse(t)) // round 2: clean

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
	if byID["r1.1"].Verdict != model.VerdictDeferred {
		t.Errorf("low-severity finding verdict = %q, want deferred", byID["r1.1"].Verdict)
	}
	if byID["r1.2"].Verdict != "fixed" || byID["r1.3"].Verdict != "fixed" {
		t.Errorf("active findings not fixed: %+v", r1.Findings)
	}
	// The coder prompt must carry only the active subset.
	prompts, _ := filepath.Glob(filepath.Join(f.cfg.Logs.StaticBase(), "*", "round-1", "fix-*.prompt"))
	if len(prompts) != 1 {
		t.Fatalf("expected one coder prompt file, got %v", prompts)
	}
	b, err := os.ReadFile(prompts[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "[r1.1]") {
		t.Error("deferred finding leaked into the coder prompt")
	}
	if !strings.Contains(string(b), "[r1.2]") || !strings.Contains(string(b), "[r1.3]") {
		t.Error("active findings missing from the coder prompt")
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "rejected", Detail: "by design"}))
	// Round 2: only the previously-deferred low finding remains; fix it.
	f.respond(3, reviewResponse(t, low)) // r2.1(low)
	f.editRepoOn(4)
	f.respond(4, fixResponse(t, model.FixResult{ID: "r2.1", Verdict: "fixed", Detail: "patched"}))
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

// deferOverCap ranks findings worst-severity-first; a severity not in
// severityRank falls back to len(severityRank) and therefore sorts last, so
// unknown/empty severities are deferred first. validateReviewFindings rejects
// invalid severities upstream, so this fallback is unreachable through the full
// loop and is covered here by calling deferOverCap directly.
func TestDeferOverCapUnknownSeverityRankedLast(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1, MaxFindingsPerRound: 2})
	o := f.orchestrator()
	rec := &model.RoundRecord{Round: 1, Findings: []model.Finding{
		{ID: "r1.1", Severity: "low", Title: "known low"},
		{ID: "r1.2", Severity: "", Title: "empty severity"},
		{ID: "r1.3", Severity: "high", Title: "known high"},
		{ID: "r1.4", Severity: "bogus", Title: "unknown severity"},
	}}
	o.deferOverCap(rec, nil)
	byID := map[string]model.Finding{}
	for _, fd := range rec.Findings {
		byID[fd.ID] = fd
	}
	// Cap 2: the two known severities (high, low) stay active; the two unknown
	// ones (empty, bogus) rank last and are deferred.
	if byID["r1.1"].Verdict == model.VerdictDeferred || byID["r1.3"].Verdict == model.VerdictDeferred {
		t.Errorf("a known-severity finding was deferred over an unknown one: %+v", rec.Findings)
	}
	if byID["r1.2"].Verdict != model.VerdictDeferred || byID["r1.4"].Verdict != model.VerdictDeferred {
		t.Errorf("unknown-severity findings not deferred: %+v", rec.Findings)
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

func TestRunRefusesUntrustedPRFixRounds(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.cfg.Target.Mode = "pr"
	f.cfg.Target.PR = 1
	// trusted_target must not bypass the pr gate: a PR is untrusted regardless.
	f.cfg.Loop.TrustedTarget = true
	_, err := f.orchestrator().Run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "allow_untrusted_fix") {
		t.Fatalf("Run() err = %v, want untrusted-PR refusal naming the opt-in", err)
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
	if err == nil || !strings.Contains(err.Error(), "trusted_target") {
		t.Fatalf("Run() err = %v, want untrusted-target refusal naming trusted_target", err)
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "patched"}))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	logf := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		t.Log(msg)
		// The coder has fully returned (runErr == nil) by the time its fix step
		// logs "done"; canceling here is observed by fix()'s later ctx.Err() check.
		if strings.Contains(msg, "fix: mock done") {
			cancel()
		}
	}
	o, err := New(f.cfg, config.Source{Config: "test.yaml"}, logf)
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "patched"}))

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
	f := newFixture(t, config.Loop{MaxIterations: 3, CleanRoundsToStop: 1})
	f.respond(1, reviewResponse(t,
		model.ReviewFinding{Category: "bugs", Severity: "high", File: "main.go", Line: 1, Title: "real bug"},
		model.ReviewFinding{Category: "style", Severity: "low", File: "main.go", Line: 2, Title: "a nit"},
	))
	f.editRepoOn(2) // the coder makes a real edit for the fixed finding
	f.respond(2, fixResponse(t,
		model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "patched the bug"},
		model.FixResult{ID: "r1.2", Verdict: "rejected", Detail: "intentional by design"},
	))
	f.respond(3, reviewResponse(t)) // round 2: clean -> converge

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
		o, err := New(f.cfg, config.Source{Config: "test.yaml"}, logf)
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
		if _, err := New(f.cfg, config.Source{Config: "test.yaml"}, t.Logf); err == nil {
			t.Fatal("New() = nil, want render error for fix-only .Findings in a review prompt")
		}
	})

	t.Run("review-only placeholder in the coder prompt", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		f.cfg.Roles.Coder.Prompt = writePrompt(t, "{{.Target}}")
		if _, err := New(f.cfg, config.Source{Config: "test.yaml"}, t.Logf); err == nil {
			t.Fatal("New() = nil, want render error for review-only .Target in the coder prompt")
		}
	})

	t.Run("shared prompt is validated for both roles", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		shared := writePrompt(t, "{{.Findings}}") // valid for coder, invalid for review
		f.cfg.Roles.Coder.Prompt = shared
		f.cfg.Roles.Review.Prompts[0].Prompt = shared
		if _, err := New(f.cfg, config.Source{Config: "test.yaml"}, t.Logf); err == nil {
			t.Fatal("New() = nil, want rejection of a fix-only placeholder in a prompt also used for review")
		}
	})

	t.Run("shared role-neutral prompt is accepted", func(t *testing.T) {
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
		shared := writePrompt(t, "Round {{.Round}}\n{{.OutputContract}}")
		f.cfg.Roles.Coder.Prompt = shared
		f.cfg.Roles.Review.Prompts[0].Prompt = shared
		if _, err := New(f.cfg, config.Source{Config: "test.yaml"}, t.Logf); err != nil {
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

func TestApplyVerdicts(t *testing.T) {
	rec := func() *model.RoundRecord {
		return &model.RoundRecord{Findings: []model.Finding{{ID: "r1.1"}, {ID: "r1.2"}}}
	}
	cases := []struct {
		name    string
		results []model.FixResult
		wantErr string
	}{
		{"valid", []model.FixResult{
			{ID: "r1.1", Verdict: "fixed", Detail: "d"},
			{ID: "r1.2", Verdict: "rejected", Detail: "d"},
		}, ""},
		{"unknown id", []model.FixResult{
			{ID: "r1.1", Verdict: "fixed"}, {ID: "r9.9", Verdict: "fixed"},
		}, "unknown finding id"},
		{"duplicate id", []model.FixResult{
			{ID: "r1.1", Verdict: "fixed"}, {ID: "r1.1", Verdict: "rejected"},
		}, "more than once"},
		{"invalid verdict", []model.FixResult{
			{ID: "r1.1", Verdict: "maybe"}, {ID: "r1.2", Verdict: "fixed"},
		}, "unknown verdict"},
		{"missing finding", []model.FixResult{
			{ID: "r1.1", Verdict: "fixed"},
		}, "did not give a verdict"},
		{"empty result set", nil, "did not give a verdict"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := rec()
			err := applyVerdicts(r, tc.results)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("applyVerdicts() = %v", err)
				}
				if r.Fixed != 1 || r.Rejected != 1 {
					t.Errorf("fixed=%d rejected=%d, want 1/1", r.Fixed, r.Rejected)
				}
				if r.Findings[0].Verdict != "fixed" || r.Findings[1].Verdict != "rejected" {
					t.Errorf("verdicts not applied: %+v", r.Findings)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("applyVerdicts() = %v, want error containing %q", err, tc.wantErr)
			}
			// A rejected result set must leave the round untouched: the run
			// summary is always written and must not carry partial verdicts.
			if r.Fixed != 0 || r.Rejected != 0 {
				t.Errorf("failed validation mutated counts: fixed=%d rejected=%d", r.Fixed, r.Rejected)
			}
			for _, f := range r.Findings {
				if f.Verdict != "" || f.VerdictDetail != "" {
					t.Errorf("failed validation mutated finding %s: %+v", f.ID, f)
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

// Aging bounds how long the cap can starve a finding. Reproduces the pathology
// from a real 5-round run: an unbounded generator of medium-severity findings (a
// test-coverage lens can always want more coverage) kept a one-line low-severity
// README error out of every fix round, so it was reported three times and never
// once scheduled. Each deferral promotes a finding one severity tier, so the wait
// is bounded by its distance from the generator's severity -- here one round --
// instead of being unbounded.
func TestDeferOverCapAgesDeferredFindings(t *testing.T) {
	o := &Orchestrator{
		cfg:  &config.Config{Loop: config.Loop{MaxFindingsPerRound: 2}},
		logf: func(string, ...any) {},
	}
	// The starved finding: low severity, same (file, category) every round, and
	// deliberately listed LAST so nothing depends on input order.
	starved := func(round int) model.Finding {
		return model.Finding{
			ID: fmt.Sprintf("r%d.3", round), Severity: "low",
			Category: "maintainability", File: "README.md", Title: "doc error",
		}
	}
	// The generator: two fresh medium findings every round, in new files.
	round := func(n int) model.RoundRecord {
		return model.RoundRecord{Round: n, Findings: []model.Finding{
			{ID: fmt.Sprintf("r%d.1", n), Severity: "medium", Category: "tests", File: fmt.Sprintf("a%d.go", n), Title: "untested"},
			{ID: fmt.Sprintf("r%d.2", n), Severity: "medium", Category: "tests", File: fmt.Sprintf("b%d.go", n), Title: "untested"},
			starved(n),
		}}
	}
	verdict := func(rec model.RoundRecord, id string) string {
		for _, f := range rec.Findings {
			if f.ID == id {
				return f.VerdictOrDefault()
			}
		}
		t.Fatalf("finding %s missing from round record", id)
		return ""
	}

	// Round 1: nothing has waited yet, so severity alone applies and the low
	// finding loses both slots to the fresh mediums.
	r1 := round(1)
	o.deferOverCap(&r1, nil)
	if got := verdict(r1, "r1.3"); got != model.VerdictDeferred {
		t.Fatalf("round 1: low finding verdict = %q, want deferred (severity should decide the first round)", got)
	}

	// Round 2: one deferral promotes low to the mediums' tier, and the age
	// tie-break awards the slot to the finding that already waited. Before aging
	// this finding lost every round forever.
	r2 := round(2)
	o.deferOverCap(&r2, []model.RoundRecord{r1})
	if got := verdict(r2, "r2.3"); got == model.VerdictDeferred {
		t.Error("round 2: the aged finding was deferred again; the cap can starve it indefinitely")
	}
	if verdict(r2, "r2.1") == model.VerdictDeferred || verdict(r2, "r2.2") == model.VerdictDeferred {
		// One generator finding must yield its slot -- the cap is 2 and three
		// findings compete, so exactly one is deferred.
		return
	}
	t.Error("round 2: no generator finding yielded its slot to the aged finding")
}

// Only unresolved findings age. A fixed or rejected finding is closed, so a later
// report of the same (file, category) must start from its own severity rather
// than inheriting priority from work that is already done.
func TestDeferOverCapDoesNotAgeResolvedFindings(t *testing.T) {
	o := &Orchestrator{
		cfg:  &config.Config{Loop: config.Loop{MaxFindingsPerRound: 1}},
		logf: func(string, ...any) {},
	}
	prior := []model.RoundRecord{{Round: 1, Findings: []model.Finding{
		{ID: "r1.1", Severity: "low", Category: "maintainability", File: "README.md", Verdict: model.VerdictFixed},
		{ID: "r1.2", Severity: "low", Category: "tests", File: "a.go", Verdict: model.VerdictRejected},
	}}}
	rec := model.RoundRecord{Round: 2, Findings: []model.Finding{
		{ID: "r2.1", Severity: "low", Category: "maintainability", File: "README.md", Title: "new doc nit"},
		{ID: "r2.2", Severity: "high", Category: "bugs", File: "b.go", Title: "real bug"},
	}}
	o.deferOverCap(&rec, prior)
	byID := map[string]model.Finding{}
	for _, f := range rec.Findings {
		byID[f.ID] = f
	}
	if byID["r2.2"].Verdict == model.VerdictDeferred {
		t.Error("a high-severity finding lost its slot to a low whose predecessor was already fixed")
	}
	if byID["r2.1"].Verdict != model.VerdictDeferred {
		t.Error("expected the low finding to be deferred; fixed/rejected history must not grant aging")
	}
}

// verifyGate points the fixture's verification at a marker file the mock coder
// creates when it "fixes" something: present => the check fails. That models the
// real case (the coder's edits break the build) without needing a real toolchain.
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "done"}))
	// Invocation 3 is the correction attempt: it does not remove the file.
	f.respond(3, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "still done"}))

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

// The coder gets exactly one correction attempt, and a round that passes after it
// commits normally -- so a transient breakage does not throw away the round.
func TestVerifyRetrySucceedsAndCommits(t *testing.T) {
	f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1})
	f.verifyGate(config.VerifyMustPass, "broken.txt")
	before := f.commitCount()

	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.breakBuildOn(2, "broken.txt")
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "done"}))
	// The correction attempt removes the offending file, so the gate passes.
	f.repairBuildOn(3, "broken.txt")
	f.respond(3, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "corrected"}))

	sum, err := f.orchestrator().Run(t.Context())
	if err != nil {
		t.Fatalf("Run() = %v, want the corrected round to commit", err)
	}
	if got := f.commitCount(); got != before+1 {
		t.Errorf("commit count = %d, want %d: the corrected round should commit", got, before+1)
	}
	if len(sum.Rounds) == 0 || !sum.Rounds[0].VerifyRetried {
		t.Error("the round record must note that a correction attempt was made")
	}
	if len(sum.Rounds[0].Verify) == 0 {
		t.Error("verification results must be recorded on the round for the summary")
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "done"}))

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
