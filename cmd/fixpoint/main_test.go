package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/runlog"
	"github.com/dsaiko/fixpoint/internal/testfixture"
)

// fixture wires a temporary git repository, a scripted mock agent, prompt
// files, and a config file so run() can be driven end to end. The mock-agent
// protocol and shared helpers live in internal/testfixture; this fixture only
// adds the per-suite config-file wiring.
type fixture struct {
	t            *testing.T
	repo         string
	respDir      string
	script       string
	reviewPrompt string
	fixPrompt    string
	logsDir      string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{t: t, repo: testfixture.GitRepo(t), respDir: t.TempDir(), logsDir: filepath.Join(t.TempDir(), "logs", "{timestamp}", "round-{round}")}
	f.script = testfixture.WriteMockScript(t, f.respDir)

	// Prompts resolve by bare name from a bundle on the search path, so the fixture
	// writes one -- into the HOME bundle (~/.fixpoint), with HOME redirected at a
	// temporary directory. Deliberately NOT <repo>/config: a bundle file inside the
	// project is target-supplied policy and the trust gate refuses it without
	// -trusted-target, so putting the fixture's prompts there would make almost
	// every test below assert its own subject through a provenance refusal instead.
	// The tests that mean to exercise that gate plant a bundle in the repo
	// themselves (see TestRunRefusesTargetSuppliedBundle).
	//
	// Chdir keeps these tests from resolving against fixpoint's own bundle, which
	// would silently exercise the shipped prompts instead of the fixture's.
	home := t.TempDir()
	t.Setenv("HOME", home)
	promptDir := filepath.Join(home, ".fixpoint", "prompts")
	if err := os.MkdirAll(promptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f.reviewPrompt = "review"
	f.fixPrompt = "fix"
	if err := os.WriteFile(filepath.Join(promptDir, "review.md"), []byte("{{.Target}}\n{{.OutputContract}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "fix.md"), []byte("{{.Findings}}\n{{.OutputContract}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(f.repo)
	return f
}

// configFile writes a config with the given target mode, extra target lines,
// and loop body (the last two may be empty) and returns its path.
func (f *fixture) configFile(mode, targetExtra, loop string) string {
	f.t.Helper()
	if loop == "" {
		loop = "  max_iterations: 3"
	}
	content := fmt.Sprintf(`target:
  mode: %s
  path: %q
%sroles:
  coder:
    agent: mock
    prompt: %q
  review:
    strategy: fixed
    prompts:
      - {agent: mock-rev, prompt: %q}
agents:
  mock:
    command: [%q]
    prompt_via: stdin
    timeout: 1m
    can_edit: true
  mock-rev:
    command: [%q]
    prompt_via: stdin
    timeout: 1m
    can_edit: false
loop:
%s
logs:
  dir: %q
ping_agents: false
`, mode, f.repo, targetExtra, f.fixPrompt, f.reviewPrompt, f.script, f.script, loop, f.logsDir)
	p := filepath.Join(f.t.TempDir(), "fixpoint.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		f.t.Fatal(err)
	}
	return p
}

// respond registers the mock agent's n-th response.
func (f *fixture) respond(n int, content string) { testfixture.Respond(f.t, f.respDir, n, content) }

// editRepoOn makes the mock agent's n-th invocation edit the repo before
// answering, simulating a coder fix.
func (f *fixture) editRepoOn(n int) { testfixture.EditRepoOn(f.t, f.respDir, f.repo, n) }

// invocations reports how many times the mock agent has been called.
func (f *fixture) invocations() int { return testfixture.Invocations(f.t, f.respDir) }

func reviewResponse(t *testing.T, findings ...model.ReviewFinding) string {
	t.Helper()
	return testfixture.ReviewResponse(t, findings...)
}

func fixResponse(t *testing.T, results ...model.FixResult) string {
	t.Helper()
	return testfixture.FixResponse(t, results...)
}

func aFinding(title string) model.ReviewFinding { return testfixture.AFinding(title) }

// ---- exit-status and flag-override contract -----------------------------------

func TestRunBadFlagExits2(t *testing.T) {
	var buf bytes.Buffer
	if got := run([]string{"-no-such-flag"}, &buf, &buf); got != 2 {
		t.Errorf("run() = %d, want 2 for a usage error", got)
	}
}

// -h is a served request, not a usage error, so it must not look like a failure
// to shells and automation.
func TestRunHelpExits0(t *testing.T) {
	for _, flag := range []string{"-h", "-help", "--help"} {
		t.Run(flag, func(t *testing.T) {
			var buf bytes.Buffer
			if got := run([]string{flag}, &buf, &buf); got != 0 {
				t.Errorf("run(%s) = %d, want 0; stderr:\n%s", flag, got, buf.String())
			}
			if !strings.Contains(buf.String(), "Usage:") {
				t.Errorf("run(%s) printed no usage text:\n%s", flag, buf.String())
			}
		})
	}
}

func TestRunMissingConfigExits1(t *testing.T) {
	var buf bytes.Buffer
	if got := run([]string{"-config", filepath.Join(t.TempDir(), "missing.yaml")}, &buf, &buf); got != 1 {
		t.Errorf("run() = %d, want 1 for a missing config", got)
	}
}

func TestRunCheck(t *testing.T) {
	f := newFixture(t)
	var buf bytes.Buffer
	if got := run([]string{"-config", f.configFile("directory", "", ""), "-check"}, &buf, &buf); got != 0 {
		t.Fatalf("run(-check) = %d, want 0; stderr:\n%s", got, buf.String())
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (-check must not run agents)", got)
	}
	if !strings.Contains(buf.String(), "configuration OK") {
		t.Errorf("stderr missing confirmation:\n%s", buf.String())
	}
	// The scope line is why --check is worth running against a real target: it
	// answers "how much am I about to pay to review" before any agent is invoked.
	if !strings.Contains(buf.String(), "scope:") || !strings.Contains(buf.String(), "file(s) in scope") {
		t.Errorf("stderr missing the scope estimate:\n%s", buf.String())
	}
}

// -gate is the documented interface of the verify gate, and every test of it in
// internal/config drives LoadBundle with Overrides{Gate} directly. This pins the
// two things only this layer can: that the flag's value reaches the Overrides
// literal in run (a field written at the call site but never read back is a
// silently empty string, not a compile error -- the drift TestAbsPathFlags
// records for the path flags), and that -check prints the provenance line both
// READMEs promise, naming the gate from the FLAG rather than the one in the file.
func TestRunGateFlagReachesConfigAndCheckNamesIt(t *testing.T) {
	f := newFixture(t)
	gates := filepath.Join(os.Getenv("HOME"), ".fixpoint", "gates")
	if err := os.MkdirAll(gates, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"alpha", "beta"} {
		body := "commands:\n  - {name: check, run: [true]}\nskip_run_edits: ['**/*_" + name + ".x']\n"
		if err := os.WriteFile(filepath.Join(gates, name+".yaml"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := f.configFile("directory", "", "")
	body, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, append(body, []byte("verify:\n  gate: alpha\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if got := run([]string{"-config", cfg, "-gate", "beta", "-check", "-trusted-target"}, &buf, &buf); got != 0 {
		t.Fatalf("run(-gate beta -check) = %d, want 0; stderr:\n%s", got, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "gate=beta") {
		t.Errorf("applied overrides do not record gate=beta; the flag did not reach Overrides:\n%s", out)
	}
	want := "gate beta: " + filepath.Join(gates, "beta.yaml")
	if !strings.Contains(out, want) {
		t.Errorf("provenance line missing %q -- -check must name the gate the FLAG chose:\n%s", want, out)
	}
	if strings.Contains(out, "gate alpha:") {
		t.Errorf("the config's own gate must not be listed once the flag replaced it:\n%s", out)
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (-check must not run agents)", got)
	}
}

// A base_ref that resolves to the WRONG commit is a valid configuration, so
// validation alone cannot catch it -- the run would simply review the wrong diff
// and bill for it. --check therefore prints the commit the base resolved to and
// the size of the diff it selects, and fails outright when it resolves to nothing.
func TestRunCheckReportsGitDiffScope(t *testing.T) {
	f := newFixture(t)
	// GitRepo commits once; HEAD~1 needs a second commit to point at.
	if err := os.WriteFile(filepath.Join(f.repo, "second.go"), []byte("package main\n\nfunc second() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	testfixture.GitRun(t, f.repo, "add", "-A")
	testfixture.GitRun(t, f.repo, "commit", "-qm", "second")
	cfg := f.configFile("git-diff", "  base_ref: HEAD~1\n", "")

	var buf bytes.Buffer
	if got := run([]string{"-config", cfg, "-check", "-trusted-target"}, &buf, &buf); got != 0 {
		t.Fatalf("run(-check) = %d, want 0; stderr:\n%s", got, buf.String())
	}
	for _, want := range []string{"scope:", `base_ref "HEAD~1"`, "changed", "insertions(+)"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("stderr missing %q:\n%s", want, buf.String())
		}
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (-check must not run agents)", got)
	}

	// An unresolvable base fails here rather than at round 1, after the first
	// review has already been paid for.
	buf.Reset()
	bad := f.configFile("git-diff", "  base_ref: no-such-ref...\n", "")
	if got := run([]string{"-config", bad, "-check", "-trusted-target"}, &buf, &buf); got != 1 {
		t.Fatalf("run(-check) = %d, want 1 for an unresolvable base; stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "merge base") {
		t.Errorf("stderr does not name the merge-base failure:\n%s", buf.String())
	}
}

// --check points git at the target -- `git diff` in git-diff mode, `git ls-files`
// in directory mode -- so it must clear the same target-integrity gates a real
// run does, even though it invokes no agent. It is the command an operator is
// told to run FIRST against an unfamiliar checkout, and a repo-supplied
// filter.<name>.clean is a program git runs itself while normalizing the worktree
// for that diff: gitenv.SafeConfigArgs cannot neutralize it (the name is dynamic), so a
// --check exempt from the gate would execute repo-controlled code with fixpoint's
// inherited environment before any lock, trust gate, or agent.
func TestRunCheckAppliesTargetGuards(t *testing.T) {
	// gitDiff returns a fixture and a git-diff config whose base_ref resolves:
	// GitRepo commits once, and HEAD~1 needs a second commit to point at.
	gitDiff := func(t *testing.T) (*fixture, string) {
		t.Helper()
		f := newFixture(t)
		if err := os.WriteFile(filepath.Join(f.repo, "second.go"), []byte("package main\n\nfunc second() {}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		testfixture.GitRun(t, f.repo, "add", "-A")
		testfixture.GitRun(t, f.repo, "commit", "-qm", "second")
		cfg := f.configFile("git-diff", "  base_ref: HEAD~1\n", "")
		return f, cfg
	}

	t.Run("repo-supplied git config refused without -trusted-target", func(t *testing.T) {
		f, cfg := gitDiff(t)
		testfixture.GitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")
		var buf bytes.Buffer
		if got := run([]string{"-config", cfg, "-check"}, &buf, &buf); got != 1 {
			t.Fatalf("run(-check) = %d, want 1 for a target with repo-supplied git config; stderr:\n%s", got, buf.String())
		}
		for _, want := range []string{"filter.evil.clean", "-trusted-target"} {
			if !strings.Contains(buf.String(), want) {
				t.Errorf("stderr missing %q:\n%s", want, buf.String())
			}
		}
		// The gate is only worth anything if it precedes the estimate: the estimate
		// is what runs git diff over the worktree.
		if strings.Contains(buf.String(), "scope:") {
			t.Errorf("the refusal must come before the scope estimate:\n%s", buf.String())
		}
	})
	t.Run("repo-supplied git config warns with -trusted-target", func(t *testing.T) {
		f, cfg := gitDiff(t)
		testfixture.GitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")
		var buf bytes.Buffer
		if got := run([]string{"-config", cfg, "-check", "-trusted-target"}, &buf, &buf); got != 0 {
			t.Fatalf("run(-check -trusted-target) = %d, want 0; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "WARNING") || !strings.Contains(buf.String(), "filter.evil.clean") {
			t.Errorf("asserted trust must still warn which repo-supplied config git would run:\n%s", buf.String())
		}
		if !strings.Contains(buf.String(), "scope:") {
			t.Errorf("the estimate must still be reported once trust is asserted:\n%s", buf.String())
		}
	})
	// The redirect guard is not trust-gated: -trusted-target says "I trust this
	// checkout's content", never "report on a different directory than the one I
	// named".
	t.Run("redirected work tree refused even with -trusted-target", func(t *testing.T) {
		f := newFixture(t)
		elsewhere := t.TempDir()
		testfixture.GitRun(t, f.repo, "config", "core.worktree", elsewhere)
		var buf bytes.Buffer
		cfg := f.configFile("directory", "", "  review_only: true")
		if got := run([]string{"-config", cfg, "-check", "-trusted-target"}, &buf, &buf); got != 1 {
			t.Fatalf("run(-check) = %d, want 1 for a redirected work tree; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "core.worktree") || !strings.Contains(buf.String(), elsewhere) {
			t.Errorf("the refusal must name the redirect and the tree it points at (%s):\n%s", elsewhere, buf.String())
		}
	})
}

func TestRunCheckLive(t *testing.T) {
	// -check-live pings the write-capable coder, so it must clear the same
	// fix-round trust gate a real run does; -trusted-target satisfies it for a
	// directory target (it is a flag, not a config key: a config cannot grant trust). The coder and reviewer are pinged in parallel and
	// share one mock response counter, so register an OK for each invocation index
	// either ordering can land on.
	t.Run("responding agents exit 0", func(t *testing.T) {
		f := newFixture(t)
		f.respond(1, "OK")
		f.respond(2, "OK")
		var buf bytes.Buffer
		if got := run([]string{"-config", f.configFile("directory", "", ""), "-trusted-target", "-check-live"}, &buf, &buf); got != 0 {
			t.Fatalf("run(-check-live) = %d, want 0; stderr:\n%s", got, buf.String())
		}
	})
	t.Run("failing agent exits 1", func(t *testing.T) {
		f := newFixture(t) // no responses: the mock exits non-zero
		var buf bytes.Buffer
		if got := run([]string{"-config", f.configFile("directory", "", ""), "-trusted-target", "-check-live"}, &buf, &buf); got != 1 {
			t.Fatalf("run(-check-live) = %d, want 1; stderr:\n%s", got, buf.String())
		}
	})
	// An untrusted target must not get an unguarded path to launch the coder: the
	// trust gate refuses -check-live before any agent is pinged.
	t.Run("untrusted target refused before pinging", func(t *testing.T) {
		f := newFixture(t)
		f.respond(1, "OK")
		var buf bytes.Buffer
		if got := run([]string{"-config", f.configFile("directory", "", ""), "-check-live"}, &buf, &buf); got != 1 {
			t.Fatalf("run(-check-live) = %d, want 1; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "-trusted-target") {
			t.Errorf("stderr must name the opt-in flag:\n%s", buf.String())
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (refusal comes before any ping)", got)
		}
	})
	// A review-only target clears the trust gate, but -check-live still starts the
	// agent CLIs with their working directory inside the checkout, and those CLIs
	// run git themselves -- so the target-integrity gates apply here exactly as they
	// do to -check, which invokes no agent at all and is therefore the LESS invasive
	// of the two. git-diff mode is what an untrusted-checkout preflight uses; it
	// touches git in every trust mode.
	t.Run("repo-supplied git config refused before pinging", func(t *testing.T) {
		f := newFixture(t)
		if err := os.WriteFile(filepath.Join(f.repo, "second.go"), []byte("package main\n\nfunc second() {}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		testfixture.GitRun(t, f.repo, "add", "-A")
		testfixture.GitRun(t, f.repo, "commit", "-qm", "second")
		testfixture.GitRun(t, f.repo, "config", "filter.evil.clean", "sh -c 'id'")
		f.respond(1, "OK")
		var buf bytes.Buffer
		cfg := f.configFile("git-diff", "  base_ref: HEAD~1\n", "  review_only: true")
		if got := run([]string{"-config", cfg, "-check-live"}, &buf, &buf); got != 1 {
			t.Fatalf("run(-check-live) = %d, want 1 for a target with repo-supplied git config; stderr:\n%s", got, buf.String())
		}
		for _, want := range []string{"filter.evil.clean", "-trusted-target"} {
			if !strings.Contains(buf.String(), want) {
				t.Errorf("stderr missing %q:\n%s", want, buf.String())
			}
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (refusal comes before any ping)", got)
		}
	})
	// The redirect guard is not trust-gated, and a ping runs each CLI with its
	// working directory in the target: a redirected work tree means those CLIs
	// explore a tree the operator did not name.
	t.Run("redirected work tree refused even with -trusted-target", func(t *testing.T) {
		f := newFixture(t)
		elsewhere := t.TempDir()
		testfixture.GitRun(t, f.repo, "config", "core.worktree", elsewhere)
		f.respond(1, "OK")
		f.respond(2, "OK")
		var buf bytes.Buffer
		cfg := f.configFile("directory", "", "  max_iterations: 3")
		if got := run([]string{"-config", cfg, "-trusted-target", "-check-live"}, &buf, &buf); got != 1 {
			t.Fatalf("run(-check-live) = %d, want 1 for a redirected work tree; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "core.worktree") || !strings.Contains(buf.String(), elsewhere) {
			t.Errorf("the refusal must name the redirect and the tree it points at (%s):\n%s", elsewhere, buf.String())
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (refusal comes before any ping)", got)
		}
	})
	// review-only clears the gate (no coder) so a read-only preflight still works.
	t.Run("review-only untrusted target pings reviewers", func(t *testing.T) {
		f := newFixture(t)
		f.respond(1, "OK")
		var buf bytes.Buffer
		if got := run([]string{"-config", f.configFile("directory", "", "  review_only: true"), "-check-live"}, &buf, &buf); got != 0 {
			t.Fatalf("run(-check-live -review-only) = %d, want 0; stderr:\n%s", got, buf.String())
		}
		if got := f.invocations(); got != 1 {
			t.Errorf("agent invocations = %d, want 1 (reviewer ping only)", got)
		}
	})
}

func TestRunReviewOnlySuccessExits0(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t))
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  review_only: true")
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 0 {
		t.Fatalf("run() = %d, want 0; stderr:\n%s", got, buf.String())
	}
	if got := f.invocations(); got != 1 {
		t.Errorf("agent invocations = %d, want 1 (coder must never run)", got)
	}
}

// A fix run whose coder rejects every finding terminates as TermAllRejected with
// a nil error, which run()'s exit switch must map to 0 alongside converged and
// review-only. The orchestrator's TestRunAllRejected checks the enum but never
// drives main's switch, so this is the only end-to-end coverage of that arm --
// a regression mapping it to 1/2 would slip through otherwise.
func TestRunAllRejectedExits3(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t, aFinding("false positive")))
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "rejected", Detail: "by design"}))
	var buf bytes.Buffer
	// A directory fix run must clear the trust gate, which only the flag can do.
	cfg := f.configFile("directory", "", "  max_iterations: 3\n  clean_rounds_to_stop: 1")
	// Exit 3, not 0: nothing changed, and automation keying on 0 must not read a
	// no-op as a converged run.
	if got := run([]string{"-config", cfg, "-trusted-target"}, &buf, &buf); got != 3 {
		t.Fatalf("run() = %d, want 3 for all-rejected; stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "done: "+model.TermAllRejected) {
		t.Errorf("stderr missing all-rejected termination line:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "no changes were made") {
		t.Errorf("a run that committed nothing must say so:\n%s", buf.String())
	}
}

// all-rejected is a verdict on the LAST loop round, so a run that committed
// fixes in an earlier round (or in the closing round, which commits after the
// outcome is decided) still ends there -- and must NOT claim "no changes were
// made". Automation reading that line would skip pushing commits already in
// history. Round 1 fixes an issue and commits it; round 2 reports a different
// issue and the coder rejects it, which ends the loop as all-rejected.
func TestRunAllRejectedAfterCommitDoesNotClaimNoChanges(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t, aFinding("first bug")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	f.respond(3, reviewResponse(t, aFinding("second bug")))
	f.respond(4, fixResponse(t, model.FixResult{ID: "i2", Verdict: "rejected", Detail: "by design"}))
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  max_iterations: 3\n  clean_rounds_to_stop: 1")
	// Still exit 3: the closing verdict is unchanged, only the claim about what
	// landed is.
	if got := run([]string{"-config", cfg, "-trusted-target"}, &buf, &buf); got != 3 {
		t.Fatalf("run() = %d, want 3 for all-rejected; stderr:\n%s", got, buf.String())
	}
	if strings.Contains(buf.String(), "no changes were made") {
		t.Errorf("run committed a fix in round 1, so the outcome line must not claim nothing changed:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "1 commit(s)") {
		t.Errorf("the outcome line must report the commits that remain in history:\n%s", buf.String())
	}
}

// An interrupted run ends as TermInterrupted with a NIL error from o.Run; run()
// must map that to exit 1 via the exit-code switch's default branch. Every other
// exit-1 test reaches the earlier err != nil path instead, so this is the only
// coverage of the interrupted -> 1 default. A SIGTERM is delivered while the
// reviewer blocks (run() has installed its signal handler by then); the canceled
// review round terminates as an interruption.
func TestRunInterruptedExits1(t *testing.T) {
	f := newFixture(t)
	// The reviewer signals readiness, then blocks until the run is canceled.
	ready := filepath.Join(f.respDir, "review-started")
	side := fmt.Sprintf("#!/bin/sh\ntouch '%s'\nsleep 60\n", ready)
	if err := os.WriteFile(filepath.Join(f.respDir, "side-1.sh"), []byte(side), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := f.configFile("directory", "", "  review_only: true")

	go func() {
		deadline := time.Now().Add(10 * time.Second)
		seen := false
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				seen = true
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		// Only signal on the path that observed readiness. A process-directed
		// SIGTERM is safe solely while run()'s notifySignals handler is
		// installed (run() blocked in the reviewer). If the reviewer never started,
		// run() may have already returned and removed the handler, so a blind
		// SIGTERM would hit the default disposition and kill the whole test binary
		// -- taking down every other test with an ambiguous signal-kill instead of
		// a clean assertion failure. Bail and let the main assertion time out.
		if !seen {
			return
		}
		// run() has registered its notifySignals handler well before the
		// reviewer started, so this cancels the run's context rather than killing
		// the test binary.
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	var buf bytes.Buffer
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1 for an interrupted run; stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "interrupted") {
		t.Errorf("stderr should report the interruption termination:\n%s", buf.String())
	}
}

// The first interrupt cancels the run; a SECOND one must still be acted on.
// signal.NotifyContext's relay goroutine returns after canceling while its
// registration stays installed, which disables the default die-on-SIGINT
// disposition and silently swallows every later SIGINT and SIGTERM -- exactly
// during interruption reconciliation, which runs on a fresh context and can take
// a while. notifySignals keeps the handler live so the second signal quits.
func TestNotifySignalsSecondSignalForceQuits(t *testing.T) {
	quit := make(chan struct{})
	orig := forceQuit
	forceQuit = func() { close(quit) }
	t.Cleanup(func() { forceQuit = orig })

	// The handler is installed synchronously by notifySignals, so these
	// process-directed signals reach it rather than the default disposition (which
	// would kill the test binary).
	ctx, stop := notifySignals(func(string, ...any) {})
	defer stop()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("the first signal did not cancel the run context")
	}

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-quit:
	case <-time.After(10 * time.Second):
		t.Fatal("the second signal was swallowed; an operator waiting on reconciliation has no way to stop the run")
	}
}

// A signal that arrives as the run is finishing must die with the handler. stop()
// unregisters delivery and cancels the returned context, so a handler that decided
// "first or second interrupt?" by reading that context would see a canceled one and
// force-quit -- turning a successful run into exit 1 for a signal the operator sent
// once, or never (SIGTERM can arrive during ordinary shutdown).
//
// This drives watchSignals directly rather than sending real signals: the window
// only exists while the watch is pinned mid-handler with teardown already begun,
// and close(done) lives inside notifySignals' stop func, so from the outside the
// two can only be raced -- a test that loses the race asserts nothing and still
// passes.
//
// Both arms of the watch's outer select are ready when it wakes, and Go picks
// among ready cases at random, so a single scenario reaches the tie-break -- the
// only place the guard under test runs -- about half the time; the other half
// every assertion holds vacuously. So retry until the queued-signal arm is
// actually taken and fail if it never is: the test cannot pass without visiting
// the window it names.
func TestWatchSignalsTeardownIgnoresQueuedSignal(t *testing.T) {
	quit := make(chan struct{}, 1)
	orig := forceQuit
	forceQuit = func() { quit <- struct{}{} }
	t.Cleanup(func() { forceQuit = orig })

	// attempt runs the scenario once and reports whether the watch took its
	// queued-signal arm: that arm is the one that receives from ch, so a drained ch
	// means the tie-break ran, while a still-buffered signal means the watch
	// returned straight out of its teardown arm and pinned nothing.
	attempt := func() bool {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		ch := make(chan os.Signal, 2)
		done := make(chan struct{})

		// Pin the watch inside the first signal's log call so the second one has to wait
		// in the buffered channel -- the window the race lives in.
		entered := make(chan struct{})
		release := make(chan struct{})
		var once sync.Once
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			watchSignals(ch, done, func(string, ...any) {
				once.Do(func() {
					close(entered)
					<-release
				})
			}, cancel)
		}()

		ch <- syscall.SIGTERM
		select {
		case <-entered:
		case <-time.After(10 * time.Second):
			t.Fatal("the watch never took the first signal; nothing can queue behind an unpinned handler")
		}

		// Queue the second signal and begin teardown while the watch is still parked, so
		// when it wakes both arms of its select are ready -- the tie teardown must win.
		ch <- syscall.SIGTERM
		close(done)
		close(release)

		select {
		case <-stopped:
		case <-time.After(10 * time.Second):
			t.Fatal("watchSignals never returned; teardown must not deadlock on a queued signal")
		}
		select {
		case <-quit:
			t.Fatal("a signal queued at teardown force-quit the process; a completed run would exit 1")
		default:
		}
		if ctx.Err() == nil {
			t.Error("the first interrupt should have canceled the run context")
		}
		return len(ch) == 0
	}

	// 100 coin flips: missing the arm every time is not something a run can do.
	for range 100 {
		if attempt() {
			return
		}
	}
	t.Fatal("the watch never took its queued-signal arm, so the teardown tie-break this test exists to pin never ran")
}

// Neither interrupt may be gated on stderr. The announcements go through the log
// mutex to a writer that can block indefinitely -- a piped stderr whose consumer
// stopped reading, or a reviewer goroutine already parked inside such a write
// while holding the lock. Announcing inline would mean the first signal never
// cancels, the second sits unread in the buffered channel instead of reaching
// forceQuit, and stop()'s join hangs on the stalled watch: the escape would fail
// exactly when the process is already wedged, which is when it is used.
func TestWatchSignalsSurvivesBlockedLogging(t *testing.T) {
	quit := make(chan struct{}, 1)
	orig := forceQuit
	forceQuit = func() { quit <- struct{}{} }
	t.Cleanup(func() { forceQuit = orig })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ch := make(chan os.Signal, 2)
	done := make(chan struct{})

	// A log call that never returns, released only once the test is over.
	blocked := make(chan struct{})
	t.Cleanup(func() { close(blocked) })
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		watchSignals(ch, done, func(string, ...any) { <-blocked }, cancel)
	}()

	ch <- syscall.SIGTERM
	select {
	case <-ctx.Done():
	case <-time.After(announceGrace / 2):
		t.Fatal("the first interrupt did not cancel the run while stderr was blocked; cancellation must not wait on the announcement")
	}

	ch <- syscall.SIGTERM
	select {
	case <-quit:
	case <-time.After(4 * announceGrace):
		t.Fatal("the second interrupt never reached forceQuit while stderr was blocked; the escape must not wait on a writer that may never drain")
	}

	// And teardown still joins: a watch that cannot be left waiting on stderr is
	// also a watch stop() cannot deadlock behind.
	close(done)
	select {
	case <-stopped:
	case <-time.After(4 * announceGrace):
		t.Fatal("watchSignals never returned while stderr was blocked; stop() would deadlock on its join")
	}
}

// The wiring around watchSignals: stop() must return rather than deadlock on its
// join with the watch goroutine, and must cancel the context it handed out.
func TestNotifySignalsStopCancels(t *testing.T) {
	ctx, stop := notifySignals(func(string, ...any) {})
	stop()
	if ctx.Err() == nil {
		t.Error("stop() should cancel the returned context")
	}
}

// run() tears the handler down as soon as o.Run returns and still defers the same
// stop func, so stop must be idempotent: a second close(done) inside it would
// panic and take the process down after a run that had already succeeded.
func TestNotifySignalsStopIsIdempotent(t *testing.T) {
	ctx, stop := notifySignals(func(string, ...any) {})
	stop()
	stop()
	if ctx.Err() == nil {
		t.Error("stop() should cancel the returned context")
	}
}

// The post-run window: once o.Run has returned there is no step to stop and no
// tree to reconcile, so the handler must already be gone while the scoreboard and
// the closing log lines print -- otherwise an interrupt there announces a pause
// that never happens, and a second one force-quits a run that already succeeded
// with exit 1. The assertion has to run inside the window, so it rides the logRaw
// wrapper (the scoreboard is the first thing printed after the run returns) and
// reads a stop() flag recorded through the installSignals seam.
func TestRunTearsDownSignalHandlerBeforeScoreboard(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t))

	var stopped, sawScoreboard, stillLive bool
	origInstall := installSignals
	installSignals = func(logf func(string, ...any)) (context.Context, func()) {
		ctx, stop := origInstall(logf)
		return ctx, func() {
			stop()
			stopped = true
		}
	}
	// The scoreboard is recognized by the horizontal rule that opens it, which
	// appears in no timestamped line. Observed at the WRITER rather than around the
	// log's method, because what matters is when the bytes land relative to the
	// handler teardown.
	origLogger := newRunLogger
	newRunLogger = func(stderr io.Writer) *runlog.Log {
		return origLogger(&watchWriter{w: stderr, mark: strings.Repeat("\u2500", 10), seen: func() {
			if !sawScoreboard {
				sawScoreboard, stillLive = true, !stopped
			}
		}})
	}
	t.Cleanup(func() {
		installSignals, newRunLogger = origInstall, origLogger
	})

	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  review_only: true")
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 0 {
		t.Fatalf("run() = %d, want 0; stderr:\n%s", got, buf.String())
	}
	if !sawScoreboard {
		t.Fatal("the scoreboard never went through the wrapped writer, so the post-run window was never observed")
	}
	if stillLive {
		t.Error("the interrupt handler was still installed while the scoreboard printed; a signal there would be mistaken for an in-run interrupt")
	}
}

func TestRunReviewerFailureExitsNonZeroAsInconclusive(t *testing.T) {
	f := newFixture(t)
	f.respond(1, "no review block") // reviewer contract violation
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  review_only: true")
	// The panel is one agent, so losing it leaves nothing: no quorum, therefore no
	// approval. It exits 5 (inconclusive) rather than 1 (error) because the run did
	// not fail -- the REVIEW was incomplete, which is a different fact and one the
	// verdict can name. What must not change is that automation cannot read it as
	// success.
	got := run([]string{"-config", cfg}, &buf, &buf)
	if got == 0 {
		t.Fatalf("run() = 0; an incomplete review must never exit success; stderr:\n%s", buf.String())
	}
	if got != model.ExitInconclusive {
		t.Fatalf("run() = %d, want %d (inconclusive); stderr:\n%s", got, model.ExitInconclusive, buf.String())
	}
	if !strings.Contains(buf.String(), "INCONCLUSIVE") {
		t.Errorf("the verdict should be stated on stderr:\n%s", buf.String())
	}
}

// A prompt-injected reviewer can smuggle a credential into a field that flows
// verbatim into a logged error -- here an invalid severity, which
// validateReviewFindings echoes back into the reviewer error that main logs as
// "run failed". The central logf must redact stderr the same way persisted logs
// are masked, so the value never reaches a retained CI console log.
func TestRunRedactsSecretsInLoggedErrors(t *testing.T) {
	const secret = "sk-ant-abcdef0123456789ABCDEF"
	f := newFixture(t)
	f.respond(1, reviewResponse(t, model.ReviewFinding{
		Category: "bugs", File: "main.go", Line: 1, Title: "bug",
		Severity: secret, // invalid severity: the value is quoted into the error
	}))
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  review_only: true")
	if got := run([]string{"-config", cfg}, &buf, &buf); got != model.ExitInconclusive {
		t.Fatalf("run() = %d, want %d for a reviewer contract violation; stderr:\n%s", got, model.ExitInconclusive, buf.String())
	}
	if strings.Contains(buf.String(), secret) {
		t.Errorf("stderr leaked the credential-shaped value:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "[REDACTED]") {
		t.Errorf("stderr missing the redaction mask (error was not redacted):\n%s", buf.String())
	}
}

// logs.redact is the operator's answer to the built-in redactor being
// shape-based: a site's own opaque token matches none of the shipped rules and
// would otherwise be written verbatim into the artifacts. The patterns are
// installed process-wide from main, so this asserts the wiring where it actually
// has to hold -- the persisted artifacts and the console -- not just the redactor
// unit. ACME-ABCD1234 is a synthetic value in nobody's rule set: without the
// config key it survives every existing masking pass.
func TestRunAppliesConfiguredRedactionPatterns(t *testing.T) {
	const siteToken = "ACME-ABCD1234"
	f := newFixture(t)
	cfgYAML := fmt.Sprintf(`target:
  mode: directory
  path: %q
roles:
  coder:
    agent: mock
    prompt: %q
  review:
    strategy: fixed
    prompts:
      - {agent: mock-rev, prompt: %q}
agents:
  mock:
    command: [%q]
    prompt_via: stdin
    timeout: 1m
    can_edit: true
  mock-rev:
    command: [%q]
    prompt_via: stdin
    timeout: 1m
    can_edit: false
loop:
  review_only: true
logs:
  dir: %q
  redact: ['ACME-[A-Z0-9]{8}']
ping_agents: false
`, f.repo, f.fixPrompt, f.reviewPrompt, f.script, f.script, f.logsDir)
	cfgPath := filepath.Join(t.TempDir(), "fixpoint.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	// The reviewer quotes the token into a finding, the way a prompt-injected one
	// quotes a value it read: it then reaches the .md/.json step logs, the journal,
	// and the summary.
	f.respond(1, reviewResponse(t, model.ReviewFinding{
		Category: "bugs", File: "main.go", Line: 1, Severity: "low",
		Title: "leaked " + siteToken, Description: "value " + siteToken,
	}))
	var buf bytes.Buffer
	if got := run([]string{"-config", cfgPath}, &buf, &buf); got != 0 {
		t.Fatalf("run() = %d, want 0; stderr:\n%s", got, buf.String())
	}
	if strings.Contains(buf.String(), siteToken) {
		t.Errorf("stderr leaked the site-specific token:\n%s", buf.String())
	}
	// Walk every artifact the run wrote: the point of the key is that it covers
	// each persisted format, so a format that bypassed the pass must fail here.
	base := strings.Split(filepath.ToSlash(f.logsDir), "/{")[0]
	masked := false
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), siteToken) {
			t.Errorf("artifact %s leaked the site-specific token:\n%s", path, b)
		}
		if strings.Contains(string(b), "[REDACTED]") {
			masked = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !masked {
		t.Error("no artifact contains the redaction mask; the configured pattern elided instead of masking, or nothing was written")
	}
}

// -review-only must force a single review round even when the config enables
// fix rounds: without the override the coder would run and fail (no resp-2).
func TestRunReviewOnlyFlagOverride(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t, aFinding("bug")))
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  max_iterations: 3")
	// The fixture finding is a high, so the review's VERDICT is changes-requested
	// and the exit status says so. That is incidental to what this test is about --
	// the flag suppressing the coder -- but it has to be asserted rather than
	// ignored, or a future change to the verdict rule would go unnoticed here.
	if got := run([]string{"-config", cfg, "-review-only"}, &buf, &buf); got != model.ExitChangesRequested {
		t.Fatalf("run(-review-only) = %d, want %d; stderr:\n%s", got, model.ExitChangesRequested, buf.String())
	}
	if got := f.invocations(); got != 1 {
		t.Errorf("agent invocations = %d, want 1 (flag must suppress the coder)", got)
	}
}

// CLI overrides must be applied BEFORE the effective config is validated. A
// config whose only review lens is once:true is invalid for a fix run (no
// recurring lens verifies rounds after the first) but valid for a review-only
// run (a single round). So -review-only must flip it from rejected to accepted:
// validating the raw config first would reject it before the override lands.
func TestRunReviewOnlyOverrideAppliedBeforeValidation(t *testing.T) {
	f := newFixture(t)
	cfgYAML := fmt.Sprintf(`target:
  mode: directory
  path: %q
roles:
  coder:
    agent: mock
    prompt: %q
  review:
    strategy: fixed
    prompts:
      - {agent: mock-rev, prompt: %q, once: true}
agents:
  mock:
    command: [%q]
    prompt_via: stdin
    timeout: 1m
    can_edit: true
  mock-rev:
    command: [%q]
    prompt_via: stdin
    timeout: 1m
    can_edit: false
loop:
  max_iterations: 3
logs:
  dir: %q
ping_agents: false
`, f.repo, f.fixPrompt, f.reviewPrompt, f.script, f.script, f.logsDir)
	cfgPath := filepath.Join(t.TempDir(), "fixpoint.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	// Without the flag the all-once lens list is an invalid fix config.
	var buf bytes.Buffer
	if got := run([]string{"-config", cfgPath}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1 (all-once lens list is an invalid fix config); stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "recurring reviewer lens") {
		t.Errorf("stderr missing the recurring-lens validation error:\n%s", buf.String())
	}

	// With -review-only the same config validates and runs exactly one review round.
	f.respond(1, reviewResponse(t))
	buf.Reset()
	if got := run([]string{"-config", cfgPath, "-review-only"}, &buf, &buf); got != 0 {
		t.Fatalf("run(-review-only) = %d, want 0; stderr:\n%s", got, buf.String())
	}
	if got := f.invocations(); got != 1 {
		t.Errorf("agent invocations = %d, want 1 (one review round)", got)
	}
}

// -max-iterations must override the config: with the config's 3 rounds the
// missing resp-3 would fail the run, so exit 2 proves the flag capped it at 1.
func TestRunMaxIterationsFlagExits2(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t, aFinding("bug")))
	f.editRepoOn(2)
	f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  max_iterations: 3")
	if got := run([]string{"-config", cfg, "-max-iterations", "1", "-trusted-target"}, &buf, &buf); got != 2 {
		t.Fatalf("run(-max-iterations 1) = %d, want 2; stderr:\n%s", got, buf.String())
	}
}

// A negative -max-iterations is invalid operator input: the override must reach
// Config.Validate (not be silently ignored as a non-positive value), so the run
// aborts with exit 1 and the must-not-be-negative diagnostic before any agent.
func TestRunNegativeMaxIterationsFlagExits1(t *testing.T) {
	f := newFixture(t)
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  max_iterations: 3")
	if got := run([]string{"-config", cfg, "-max-iterations", "-1"}, &buf, &buf); got != 1 {
		t.Fatalf("run(-max-iterations -1) = %d, want 1; stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "must not be negative") {
		t.Errorf("stderr must carry the negative-value diagnostic:\n%s", buf.String())
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (validation fails before any agent)", got)
	}
}

// The -trusted-target flag must clear the directory/git-diff fix-round gate that a
// bare config fails closed on. The flag is the ONLY way to clear it: trust is not a
// config key, because a config can come from the repository under review.
func TestRunTrustedTargetFlag(t *testing.T) {
	t.Run("directory fix rounds refused without the flag", func(t *testing.T) {
		f := newFixture(t)
		var buf bytes.Buffer
		p := f.configFile("directory", "", "")
		if got := run([]string{"-config", p}, &buf, &buf); got != 1 {
			t.Fatalf("run() = %d, want 1; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "-trusted-target") {
			t.Errorf("stderr must name the opt-in flag:\n%s", buf.String())
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (refusal comes first)", got)
		}
	})
	t.Run("flag clears the gate and the run proceeds", func(t *testing.T) {
		f := newFixture(t)
		f.respond(1, reviewResponse(t, aFinding("bug")))
		f.editRepoOn(2)
		f.respond(2, fixResponse(t, model.FixResult{ID: "i1", Verdict: "fixed", Detail: "patched"}))
		f.respond(3, reviewResponse(t)) // clean round -> converge, exit 0
		var buf bytes.Buffer
		p := f.configFile("directory", "", "")
		if got := run([]string{"-config", p, "-trusted-target"}, &buf, &buf); got != 0 {
			t.Fatalf("run(-trusted-target) = %d, want 0; stderr:\n%s", got, buf.String())
		}
		if strings.Contains(buf.String(), "pass -trusted-target") {
			t.Errorf("-trusted-target did not suppress the refusal:\n%s", buf.String())
		}
	})
	// -trusted-bundle is not a shortcut to it. That flag says the bundle files may
	// be run, nothing about the target's content -- so the fix-round gate, which is
	// entirely about content a prompt injection could hide in, stays closed.
	t.Run("-trusted-bundle does not clear the fix-round gate", func(t *testing.T) {
		f := newFixture(t)
		var buf bytes.Buffer
		p := f.configFile("directory", "", "")
		if got := run([]string{"-config", p, "-trusted-bundle"}, &buf, &buf); got != 1 {
			t.Fatalf("run(-trusted-bundle) = %d, want 1: bundle trust must not authorize edits; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "-trusted-target") {
			t.Errorf("the refusal must still name the flag that does clear it:\n%s", buf.String())
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (refusal comes first)", got)
		}
	})
	// The attack this closes, end to end at the CLI: a config asserting its own
	// trust must not run. Bundles resolve from <project>/config first, so this file
	// is one a hostile repository can ship -- and honoring it would authorize both
	// executing the agent definitions that repository supplies and running the
	// write-capable coder against it. No agent may be invoked.
	t.Run("a config cannot grant its own trust", func(t *testing.T) {
		f := newFixture(t)
		f.respond(1, reviewResponse(t))
		var buf bytes.Buffer
		p := f.configFile("directory", "", "  trusted_target: true")
		if got := run([]string{"-config", p}, &buf, &buf); got != 1 {
			t.Fatalf("run() = %d, want 1: a config that grants itself trust must not load; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "cannot be set in a configuration file") {
			t.Errorf("stderr must explain why the key is refused:\n%s", buf.String())
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0: refusal must precede every process launch", got)
		}
	})
}

// The machine-readable listing feeds shell completion, so a config name is
// untrusted input on its way to a shell: names are FILENAMES from a bundle
// directory and the first one searched is <project>/config, inside the repository
// under review. A name that is not a bare identifier is dropped rather than emitted,
// because an installed completion script is generated once and never regenerated --
// the binary is the only place that can still stop it.
func TestListPorcelainDropsUnsafeConfigNames(t *testing.T) {
	dir := t.TempDir()
	const body = "roles:\n  review:\n    prompts: [review-bugs]\n"
	for _, name := range []string{
		"review-code.yaml",
		"$(touch pwned).yaml", // command substitution: what compgen -W would run
		"`touch pwned2`.yaml",
		"a b.yaml",   // word splitting
		"x\ny.yaml",  // a second listing line
		"we*rd.yaml", // pathname expansion
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var out, errOut bytes.Buffer
	if code := listPorcelain(&config.Resolver{Bundles: []string{dir}}, &out, &errOut); code != 0 {
		t.Fatalf("listPorcelain() = %d, want 0; stderr:\n%s", code, errOut.String())
	}
	if got := out.String(); got != "review-code\trunnable\t\n" {
		t.Errorf("porcelain listing =\n%q\nwant only the safely-named config", got)
	}
	// Silence would make a config vanish from completion with no explanation.
	for _, want := range []string{"pwned", "a b", "we*rd"} {
		if !strings.Contains(errOut.String(), want) {
			t.Errorf("stderr must report dropping %q:\n%s", want, errOut.String())
		}
	}
}

// Both listings walk every bundle on the search path, and the first is
// <project>/config, inside the repository under review. One directory nobody can
// read must not blank the whole listing: the failure is a warning and the configs
// that did resolve are still printed. Completion parses the porcelain form with
// stderr discarded, so treating that error as fatal would leave TAB offering
// nothing, with no explanation anywhere the operator could see.
func TestListDegradesWhenOneBundleIsUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a directory whatever its mode, so there is nothing to deny here")
	}
	root := t.TempDir()
	closed, open := filepath.Join(root, "closed"), filepath.Join(root, "open")
	for _, dir := range []string{closed, open} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	const body = "roles:\n  review:\n    prompts: [review-bugs]\n"
	if err := os.WriteFile(filepath.Join(open, "review-code.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(closed, 0o000); err != nil {
		t.Fatal(err)
	}
	// Restored before TempDir's own cleanup, which cannot remove a directory it is
	// not allowed to read.
	t.Cleanup(func() { _ = os.Chmod(closed, 0o700) })
	newResolver := func() *config.Resolver { return &config.Resolver{Bundles: []string{closed, open}} }

	t.Run("human listing", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if code := listConfigs(newResolver(), open, &out, &errOut); code != 0 {
			t.Fatalf("listConfigs() = %d, want 0: the readable bundle still has a config; stderr:\n%s", code, errOut.String())
		}
		if !strings.Contains(out.String(), "review-code") {
			t.Errorf("the config that resolved must still be listed:\n%s", out.String())
		}
		if !strings.Contains(errOut.String(), "warning") {
			t.Errorf("the unreadable bundle must be reported as a warning:\n%s", errOut.String())
		}
	})

	t.Run("porcelain listing", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if code := listPorcelain(newResolver(), &out, &errOut); code != 0 {
			t.Fatalf("listPorcelain() = %d, want 0; stderr:\n%s", code, errOut.String())
		}
		if got := out.String(); got != "review-code\trunnable\t\n" {
			t.Errorf("porcelain listing =\n%q\nwant the config completion can still offer", got)
		}
	})
}

// A description and a filename are repository-controlled: bundles resolve from
// <project>/config FIRST, and listing runs before any provenance or trust gate. So
// merely asking a hostile clone what it offers must not let it drive the terminal
// -- via OSC 52 (clipboard write), CSI cursor controls (redraw the listing to
// misattribute a config), BEL, or a bidi override (make text read backwards).
// Both consumers are covered: `--list` prints to the operator's terminal, and the
// porcelain description is what zsh and fish display on TAB.
func TestListEscapesTerminalControlsInProjectMetadata(t *testing.T) {
	// YAML double-quoted escapes, so the file on disk really holds ESC, BEL, and the
	// bidi overrides -- not their textual spelling.
	const hostileDesc = `description: "\x1b]52;c;cGF5bG9hZA==\x07 \x1b[2K\x1b[A \u202Egnidaer \u2066 desc"`
	const body = hostileDesc + "\nroles:\n  review:\n    prompts: [review-bugs]\n"
	// Each character that must never reach the terminal, with the visible escape the
	// output has to carry instead: dropping them silently would hide the payload
	// from the operator reading the listing.
	controls := []struct{ raw, escaped string }{
		{"\x1b", `\x1b`},
		{"\x07", `\x07`},
		{"\u202e", `\u202e`},
		{"\u2066", `\u2066`},
	}
	assertEscaped := func(t *testing.T, what, got string) {
		t.Helper()
		for _, c := range controls {
			if strings.Contains(got, c.raw) {
				t.Errorf("%s carries the raw control %q to the terminal:\n%q", what, c.raw, got)
			}
			if !strings.Contains(got, c.escaped) {
				t.Errorf("%s dropped %s instead of escaping it visibly:\n%q", what, c.escaped, got)
			}
		}
	}

	t.Run("human listing", func(t *testing.T) {
		dir := t.TempDir()
		// The FILENAME is repository-controlled too, and the human listing prints it
		// as both the name and the path (porcelain drops it via bundleNameRE).
		// Assembled here rather than inline: a "\x1b" literal inside a filepath.Join
		// call reads to gocritic as a Windows path separator.
		hostileName := "ev" + "\x1b" + "il.yaml"
		if err := os.WriteFile(filepath.Join(dir, hostileName), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		var out, errOut bytes.Buffer
		if code := listConfigs(&config.Resolver{Bundles: []string{dir}}, dir, &out, &errOut); code != 0 {
			t.Fatalf("listConfigs() = %d, want 0; stderr:\n%s", code, errOut.String())
		}
		got := out.String()
		assertEscaped(t, "the human listing", got)
		if !strings.Contains(got, "desc") {
			t.Errorf("the printable part of the description must survive:\n%q", got)
		}
	})

	t.Run("porcelain listing", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "review-code.yaml"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		var out, errOut bytes.Buffer
		if code := listPorcelain(&config.Resolver{Bundles: []string{dir}}, &out, &errOut); code != 0 {
			t.Fatalf("listPorcelain() = %d, want 0; stderr:\n%s", code, errOut.String())
		}
		got := out.String()
		assertEscaped(t, "the porcelain description completion displays", got)
		// One line, three fields: the escaping must not have broken the format.
		if lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n"); len(lines) != 1 {
			t.Fatalf("porcelain listing = %q, want exactly one line", got)
		} else if fields := strings.Split(lines[0], "\t"); len(fields) != 3 {
			t.Errorf("porcelain line = %q, want three tab-separated fields", lines[0])
		}
	})
}

// The log stream is a decision surface too, and one the repository under review
// helps draw: agent and prompt names come from a task config it can ship, and the
// paths they resolve to are FILENAMES inside <project>/config. Both are printed
// before the trust gate -- as the provenance listing, and again inside the refusal
// that asks the operator to read those files and pass -trusted-target. A name
// carrying CSI could scroll the other entries of that listing away, so the
// operator would assert trust over a listing the target drew.
func TestRunEscapesTargetSuppliedNamesInLogs(t *testing.T) {
	f := newFixture(t)
	// Erase-line plus cursor-up: the pair that rewrites what is already on screen.
	// Built here rather than inside the filepath.Join call below, where a "\x1b"
	// literal reads to gocritic as a Windows path separator.
	hostile := "ev" + "\x1b" + "[2K" + "\x1b" + "[Ail"
	planted := f.planted(filepath.Join("agents", hostile+".yaml"),
		fmt.Sprintf("command: [%q]\nprompt_via: stdin\ntimeout: 1m\ncan_edit: false\n", f.script))

	// The task config itself lives OUTSIDE the repository, so the only
	// target-supplied file -- and the only source of the escape sequence -- is the
	// agent the repository shipped.
	content := fmt.Sprintf(`target:
  mode: directory
  path: %q
roles:
  coder:
    agent: mock
    prompt: %q
  review:
    strategy: fixed
    prompts:
      - {agent: %q, prompt: %q}
agents:
  mock:
    command: [%q]
    prompt_via: stdin
    timeout: 1m
    can_edit: true
loop:
  review_only: true
  max_iterations: 1
logs:
  dir: %q
ping_agents: false
`, f.repo, f.fixPrompt, hostile, f.reviewPrompt, f.script, f.logsDir)
	cfg := filepath.Join(t.TempDir(), "fixpoint.yaml")
	if err := os.WriteFile(cfg, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1: the target-supplied agent file must be refused; stderr:\n%s", got, buf.String())
	}
	got := buf.String()
	if strings.Contains(got, "\x1b") {
		t.Errorf("stderr carries a raw ESC from a target-supplied agent name to the terminal:\n%q", got)
	}
	// Escaped, not dropped: the operator has to be able to see WHY the name looks odd.
	if !strings.Contains(got, `\x1b`) {
		t.Errorf("the control character was dropped instead of escaped visibly:\n%q", got)
	}
	// Both printers are covered by this one run -- the provenance line names the
	// agent, the refusal names the file it resolved to.
	if !strings.Contains(got, "agent ev") {
		t.Errorf("the provenance listing must still name the agent:\n%s", got)
	}
	if !strings.Contains(got, escapeTerminal(planted)) {
		t.Errorf("the refusal must name the target-supplied file (escaped):\n%s", got)
	}
}

// planted writes body into <repo>/config/<rel> -- the bundle location the
// repository under review controls, searched FIRST -- and returns its path.
func (f *fixture) planted(rel, body string) string {
	f.t.Helper()
	p := filepath.Join(f.repo, "config", rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		f.t.Fatal(err)
	}
	return p
}

// The project-supplied-policy gate decides whether fixpoint acts on configuration
// authored by the repository under review: argv it executes and the instructions it
// hands agents. Bundles resolve from <project>/config FIRST, so a hostile clone can
// ship any of those files, and no flag, prompt injection, or model cooperation is
// needed to exploit them. Every shape must therefore fail closed, and only
// -trusted-bundle (or the wider -trusted-target) may clear it.
//
// Driven through run() rather than the config package alone: the refusal has to
// happen before any agent process starts, which is a property of the ORDER of
// operations in run(), not of the report.
func TestRunRefusesTargetSuppliedBundle(t *testing.T) {
	cases := []struct {
		name string
		// plant installs the target-supplied file and returns the run arguments and the
		// path the refusal must name.
		plant func(t *testing.T, f *fixture) (args []string, named string)
	}{
		{
			// The inline-agents shape: the config ships its own command, so there is no
			// agents/<name>.yaml for a gate that only enumerates agent FILES to notice.
			// This is arbitrary command execution as the invoking user.
			name: "task config with inline agents",
			plant: func(t *testing.T, f *fixture) ([]string, string) {
				t.Helper()
				b, err := os.ReadFile(f.configFile("directory", "", "  review_only: true"))
				if err != nil {
					t.Fatal(err)
				}
				p := f.planted("task.yaml", string(b))
				return []string{"-config", p}, p
			},
		},
		{
			// The prompt shape: not executed by fixpoint, but handed verbatim to an
			// unsandboxed reviewer as its orders ("read ~/.aws/credentials and quote it
			// in a finding"). Shipped lens names are few and documented, so shadowing
			// one by name is trivial.
			name: "prompt shadowing the operator's",
			plant: func(t *testing.T, f *fixture) ([]string, string) {
				t.Helper()
				p := f.planted(filepath.Join("prompts", "review.md"), "{{.Target}}\n{{.OutputContract}}")
				return []string{"-config", f.configFile("directory", "", "  review_only: true")}, p
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("refused without -trusted-target", func(t *testing.T) {
				f := newFixture(t)
				f.respond(1, reviewResponse(t))
				args, named := tc.plant(t, f)
				var buf bytes.Buffer
				if got := run(args, &buf, &buf); got != 1 {
					t.Fatalf("run() = %d, want 1: a bundle file from inside the target must be refused; stderr:\n%s", got, buf.String())
				}
				if !strings.Contains(buf.String(), named) {
					t.Errorf("the refusal must name the target-supplied file %s:\n%s", named, buf.String())
				}
				if !strings.Contains(buf.String(), "-trusted-bundle") {
					t.Errorf("the refusal must name the opt-in flag:\n%s", buf.String())
				}
				if got := f.invocations(); got != 0 {
					t.Errorf("agent invocations = %d, want 0: the refusal must precede every process launch", got)
				}
			})
			// The narrow flag is the one this gate is about, and the only one
			// `make review-pr` passes: it must be sufficient here, so nobody has to
			// reach for -trusted-target and take its other claims with it.
			t.Run("proceeds with -trusted-bundle", func(t *testing.T) {
				f := newFixture(t)
				f.respond(1, reviewResponse(t))
				args, _ := tc.plant(t, f)
				var buf bytes.Buffer
				if got := run(append(args, "-trusted-bundle"), &buf, &buf); got != 0 {
					t.Fatalf("run(-trusted-bundle) = %d, want 0; stderr:\n%s", got, buf.String())
				}
				if strings.Contains(buf.String(), "refusing to run") {
					t.Errorf("-trusted-bundle did not clear the gate:\n%s", buf.String())
				}
				if got := f.invocations(); got != 1 {
					t.Errorf("agent invocations = %d, want 1 (the review round ran)", got)
				}
			})
			t.Run("proceeds with -trusted-target", func(t *testing.T) {
				f := newFixture(t)
				f.respond(1, reviewResponse(t))
				args, _ := tc.plant(t, f)
				var buf bytes.Buffer
				if got := run(append(args, "-trusted-target"), &buf, &buf); got != 0 {
					t.Fatalf("run(-trusted-target) = %d, want 0; stderr:\n%s", got, buf.String())
				}
				if strings.Contains(buf.String(), "refusing to run") {
					t.Errorf("-trusted-target did not clear the gate:\n%s", buf.String())
				}
				if got := f.invocations(); got != 1 {
					t.Errorf("agent invocations = %d, want 1 (the review round ran)", got)
				}
			})
		})
	}
}

func TestRunAllowUntrustedFixFlag(t *testing.T) {
	prTarget := "  pr: 1\n"
	t.Run("pr fix rounds refused without the flag", func(t *testing.T) {
		f := newFixture(t)
		var buf bytes.Buffer
		p := f.configFile("pr", prTarget, "")
		if got := run([]string{"-config", p}, &buf, &buf); got != 1 {
			t.Fatalf("run() = %d, want 1; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "-allow-untrusted-fix") {
			t.Errorf("stderr must name the opt-in flag:\n%s", buf.String())
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (refusal comes first)", got)
		}
	})
	t.Run("flag moves past the refusal into PR preparation", func(t *testing.T) {
		f := newFixture(t)
		// Install a gh stub on PATH that fails loudly. It runs only from
		// target.Prepare (gh pr checkout), so its marker in the run's error proves
		// execution advanced past the trust gate into PR preparation -- a plain
		// "refusal absent" check would also pass if the flag were removed, since a
		// dropped flag makes flag.Parse exit 2 with the hyphenated unknown-flag
		// message, never printing the underscore-form refusal either.
		const marker = "GH_STUB_REACHED_PR_PREP"
		ghDir := t.TempDir()
		stub := "#!/bin/sh\necho " + marker + " >&2\nexit 1\n"
		if err := os.WriteFile(filepath.Join(ghDir, "gh"), []byte(stub), 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", ghDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		var buf bytes.Buffer
		p := f.configFile("pr", prTarget, "")
		// The run still fails (the gh stub aborts PR checkout), but it must fail
		// inside PR preparation, not at the trust gate: exit 1, marker present,
		// refusal absent.
		if got := run([]string{"-config", p, "-allow-untrusted-fix"}, &buf, &buf); got != 1 {
			t.Fatalf("run(-allow-untrusted-fix) = %d, want 1; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), marker) {
			t.Errorf("run did not reach PR preparation (gh stub never ran):\n%s", buf.String())
		}
		if strings.Contains(buf.String(), "-allow-untrusted-fix if you trust") {
			t.Errorf("-allow-untrusted-fix did not suppress the refusal:\n%s", buf.String())
		}
	})
}

// gateWriter records each Write in arrival order and holds the one carrying
// mark open for hold, modeling a stderr that does not write atomically. A
// second writer entering that window would have its output land inside the
// held one on a real terminal.
type gateWriter struct {
	mark    string
	hold    time.Duration
	started chan struct{} // closed as the FIRST marked Write begins

	once   sync.Once
	mu     sync.Mutex
	writes []string
}

func (w *gateWriter) Write(p []byte) (int, error) {
	s := string(p)
	if strings.Contains(s, w.mark) {
		// Only the first marked write opens the window. A regression that sends the
		// table through logf instead of Raw arrives one line per Write (runlog.emit
		// splits on "\n"), and the table's two horizontal rules both carry the mark:
		// closing unguarded would panic in run()'s goroutine and take the whole
		// package down instead of failing with the count below.
		w.once.Do(func() { close(w.started) })
		time.Sleep(w.hold)
	}
	w.mu.Lock()
	w.writes = append(w.writes, s)
	w.mu.Unlock()
	return len(p), nil
}

func (w *gateWriter) recorded() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.writes...)
}

// The end-of-run scoreboard goes out through logRaw while the signal handler
// (still installed until run returns) can log at any moment. logRaw must
// therefore take the SAME lock as logf, or a timestamped line lands inside the
// table and shreds its column alignment. Dropping the lock from logRaw makes
// the concurrent line arrive first here.
func TestLogRawSerializesAgainstLogLines(t *testing.T) {
	w := &gateWriter{mark: "TABLE", hold: 200 * time.Millisecond, started: make(chan struct{})}
	l := newLogger(w)
	logf, logRaw := l.Logf(), l.Raw

	done := make(chan struct{})
	go func() {
		defer close(done)
		logRaw("\nTABLE row one\nTABLE row two\n")
	}()

	<-w.started        // the table write is in progress...
	logf("concurrent") // ...so this must not be written until it finishes
	<-done

	got := w.recorded()
	if len(got) != 2 {
		t.Fatalf("writes = %q, want the table and the log line", got)
	}
	if !strings.Contains(got[0], "TABLE") {
		t.Errorf("writes = %q, want the table first: logRaw did not hold the log lock", got)
	}
	if !strings.Contains(got[1], "concurrent") {
		t.Errorf("writes = %q, want the concurrent line second", got)
	}
}

// The test above pins newLogger's shared lock; this pins run() actually SENDING
// the scoreboard through it. The two are separate regressions: a table reverted
// to a bare fmt.Fprint(stderr, ...) keeps every logRaw guarantee intact and still
// lets the signal handler -- installed until run returns -- split the table.
// Wrapping the run's own writers is the only way a full-CLI test can open that
// window on demand, and a scoreboard that bypassed them would never arrive here.
func TestRunScoreboardWritesThroughLockedWriter(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t))

	// The horizontal rule opens the table and appears in no timestamped line.
	w := &gateWriter{mark: strings.Repeat("─", 10), hold: 200 * time.Millisecond, started: make(chan struct{})}
	orig := newRunLogger
	racer := make(chan struct{})
	// Closed when the test is done, so the racer never outlives it: a scoreboard
	// that bypassed the run's writers never closes w.started, and the goroutine
	// would otherwise park forever and hang the package.
	testDone := make(chan struct{})
	newRunLogger = func(io.Writer) *runlog.Log {
		l := newLogger(w)
		// Race a log line against the table write: the gate closes w.started as the
		// marked write begins and then holds it open, so this line is offered to the
		// log while the table is half-written. The lock must hold it back until the
		// table is whole; without it the line lands first, as it would mid-table.
		go func() {
			defer close(racer)
			select {
			case <-w.started:
				l.Printf("concurrent")
			case <-testDone:
			}
		}()
		return l
	}
	t.Cleanup(func() { close(testDone) })
	t.Cleanup(func() { newRunLogger = orig })

	var buf bytes.Buffer
	if got := run([]string{"-config", f.configFile("directory", "", "  review_only: true")}, &buf, &buf); got != 0 {
		t.Fatalf("run() = %d, want 0; log:\n%s", got, strings.Join(w.recorded(), ""))
	}
	// Counted before the join: the table is whole by the time run() returns, and a
	// scoreboard that bypassed the run's writers would leave the racer blocked, so
	// joining first would hang here instead of reporting tables = 0.
	got := w.recorded()
	tables := 0
	for _, s := range got {
		if strings.Contains(s, w.mark) {
			tables++
		}
	}
	if tables != 1 {
		t.Fatalf("the scoreboard reached the locked writer %d times, want 1: run() is not printing the table through Raw", tables)
	}
	// The table arrived, so the racer was released by it; join before measuring
	// order, since the line only reaches the writer once the lock is free.
	<-racer

	got = w.recorded()
	table, concurrent := -1, -1
	for i, s := range got {
		if strings.Contains(s, w.mark) && table < 0 {
			table = i
		}
		if strings.Contains(s, "concurrent") && concurrent < 0 {
			concurrent = i
		}
	}
	if table < 0 || concurrent < 0 {
		t.Fatalf("writes = %q, want both the table and the concurrent line", got)
	}
	if concurrent < table {
		t.Errorf("concurrent line at %d precedes the table at %d: the scoreboard did not hold the log lock\nwrites = %q",
			concurrent, table, got)
	}
}

// watchWriter calls seen the first time a write carries mark, then passes it
// through untouched. It exists so a test can observe WHEN a particular piece of
// output lands without wrapping the logger's own methods.
type watchWriter struct {
	w    io.Writer
	mark string
	once sync.Once
	seen func()
}

func (w *watchWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), w.mark) {
		w.once.Do(w.seen)
	}
	return w.w.Write(p)
}

// §7.3 rules that an implement run must not take its policy from the design's
// own directory, and nothing enforced it: bundle discovery is cwd-anchored, so
// running the config from inside the design's repository made that repository
// the first bundle searched -- and -trusted-target, which §7.3 makes MANDATORY
// for every implement run, cleared the general guard that would have caught it
// (review run 20260813-222753). verify.commands is argv fixpoint executes
// itself, so the design would have chosen how it was checked.
func TestDesignSuppliedPolicyNeedsTheNarrowFlag(t *testing.T) {
	design := t.TempDir()
	inside := filepath.Join(design, "config", "implement.yaml")
	outside := filepath.Join(t.TempDir(), "bundle", "implement.yaml")
	// Both files must EXIST: withinTree deliberately fails closed when a path
	// cannot be canonicalized, so a fixture of bare strings would report every
	// bundle as design-resident and the "outside" case would pass for the wrong
	// reason.
	for _, f := range []string{inside, outside} {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("description: t\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	loaded := func(configPath string, trustTarget, trustBundle bool) *config.Loaded {
		return &config.Loaded{
			Config: &config.Config{
				Target: config.Target{Mode: config.ModeDirectory, Path: design, Document: "DESIGN.md"},
				Roles: config.Roles{
					Planner: config.RoleRef{Agent: "claude", Prompt: "implement-plan"},
					Coder:   config.RoleRef{Agent: "claude-coder", Prompt: "implement-task"},
				},
				Loop: config.Loop{TrustedTarget: trustTarget, TrustedBundle: trustBundle},
			},
			Source:      config.Source{Config: configPath},
			ProjectRoot: design,
		}
	}
	discard := func(string, ...any) {}

	cases := []struct {
		name       string
		configPath string
		target     bool
		bundle     bool
		want       bool
	}{
		{"design-resident bundle refused with the mandatory flag alone", inside, true, false, false},
		{"design-resident bundle allowed by the narrow flag", inside, true, true, true},
		{"a bundle outside the design is never in scope", outside, true, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allowDesignSuppliedPolicy(loaded(tc.configPath, tc.target, tc.bundle), discard); got != tc.want {
				t.Fatalf("allowDesignSuppliedPolicy() = %v, want %v", got, tc.want)
			}
		})
	}

	// The refusal has to say which file and which flag, or the operator cannot act.
	var lines []string
	allowDesignSuppliedPolicy(loaded(inside, true, false), func(f string, a ...any) {
		lines = append(lines, fmt.Sprintf(f, a...))
	})
	msg := strings.Join(lines, "\n")
	for _, want := range []string{inside, "-trusted-bundle", "verify.commands"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not mention %q:\n%s", want, msg)
		}
	}

	// A review run is untouched: this gate is the implement pipeline's alone.
	l := loaded(inside, true, false)
	l.Config.Roles.Planner = config.RoleRef{}
	if !allowDesignSuppliedPolicy(l, discard) {
		t.Error("a non-implement run was gated by the implement-only rule")
	}
}

// The REAL wiring: -plan and -continue given relative values must reach the
// configuration as absolute paths. TestAbsPathFlags below pins the helper, but
// it builds its own correctly-keyed map, so a typo at the call site
// (paths["plans"]) would still pass it -- precisely the silent empty-plan path
// that would fall through to a planner session and build a decomposition the
// operator never approved (review run 20260819-104919). The run log's
// "overridden by flags" line is printed from config.Overrides.Applied() during
// load, before any validation refusal, so it shows what actually arrived.
func TestPlanAndContinueFlagsReachConfigAsAbsolutePaths(t *testing.T) {
	for _, tc := range []struct{ flag, rel, key string }{
		{"-plan", "rel-plan.json", "plan"},
		{"-continue", "rel-project", "continue"},
	} {
		t.Run(tc.flag, func(t *testing.T) {
			f := newFixture(t)
			var buf bytes.Buffer
			p := f.configFile("directory", "", "")
			run([]string{"-config", p, tc.flag, tc.rel, "-check"}, &buf, &buf)
			// The value must be present, absolute, and end in what was typed. The
			// exact prefix is the fixture's working directory, which is not this
			// test's -- asserting the property rather than a literal is also what
			// keeps this from becoming a test that restates its own setup.
			re := regexp.MustCompile(tc.key + `=(\S+)`)
			m := re.FindStringSubmatch(buf.String())
			if m == nil {
				t.Fatalf("no %s= in the applied overrides -- the flag never reached the config:\n%s", tc.key, buf.String())
			}
			if !filepath.IsAbs(m[1]) || filepath.Base(m[1]) != tc.rel {
				t.Errorf("%s = %q, want an absolute path ending in %q", tc.key, m[1], tc.rel)
			}
		})
	}
}

// absPathFlags keys on string literals now, so a key written at the call site
// but not read back -- or vice versa -- is a silently empty path, not a compile
// error (review run 20260818-234734). This pins the helper; the test above pins
// the call site.
func TestAbsPathFlags(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	out, err := absPathFlags(map[string]string{
		"target": filepath.Join("docs", "DESIGN.md"), "out": "", "plan": filepath.Join("rel", "plan.json"), "continue": "/abs/proj",
	})
	if err != nil {
		t.Fatalf("absPathFlags() = %v", err)
	}
	// Every key the run() call site passes must come back present.
	for _, k := range []string{"target", "out", "plan", "continue"} {
		if _, ok := out[k]; !ok {
			t.Errorf("key %q was dropped -- a call-site/read mismatch is a silent empty path", k)
		}
	}
	if want := filepath.Join(cwd, "rel", "plan.json"); out["plan"] != want {
		t.Errorf("plan = %q, want the absolute form %q", out["plan"], want)
	}
	if out["out"] != "" {
		t.Errorf("empty stays empty (use config), got %q", out["out"])
	}
	if out["continue"] != "/abs/proj" {
		t.Errorf("an absolute path is unchanged, got %q", out["continue"])
	}
}

// stubGHForPR puts a `gh` on PATH answering the two reads the branch resolution
// makes, and returns the file it records its argv into. PATH is prepended so git
// stays real -- the branch half of the answer comes from the fixture repository.
func stubGHForPR(t *testing.T, number int) string {
	t.Helper()
	bin, calls := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	n := strconv.Itoa(number)
	view := `{"number":` + n + `,"state":"OPEN","url":"https://example.test/pull/` + n +
		`","baseRefName":"main","headRefName":"feat/x","headRepositoryOwner":{"login":"o"}}`
	list := `[{"number":` + n + `,"baseRefName":"main","headRepositoryOwner":{"login":"o"}}]`
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + calls + "\n" +
		"if [ \"$1\" = pr ] && [ \"$2\" = view ]; then printf '%s' '" + view + "'; exit 0; fi\n" +
		"if [ \"$1\" = pr ] && [ \"$2\" = list ]; then printf '%s' '" + list + "'; exit 0; fi\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return calls
}

func ghCalls(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatal(err)
	}
	return string(b)
}

// A pr-mode run given no number takes the pull request of the checked-out
// branch. Driven through run() rather than the resolver, because the wiring is
// what can break: the resolution has to land BEFORE Validate, whose rule is that
// mode pr needs a positive target.pr, and the number it produces has to be the
// one the rest of the run acts on -- which --check prints as its scope.
func TestRunResolvesPRFromTheCheckedOutBranch(t *testing.T) {
	f := newFixture(t)
	testfixture.GitRun(t, f.repo, "checkout", "-q", "-b", "feat/x")
	calls := stubGHForPR(t, 170)

	var buf bytes.Buffer
	if got := run([]string{"-config", f.configFile("pr", "", ""), "-review-only", "-check"}, &buf, &buf); got != 0 {
		t.Fatalf("run(pr -check with no -pr) = %d, want 0; stderr:\n%s", got, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "resolved from the checked-out branch feat/x") {
		t.Errorf("the run does not say where the number came from; nobody typed it:\n%s", out)
	}
	if !strings.Contains(out, "pull request #170") {
		t.Errorf("the resolved number is not reported:\n%s", out)
	}
	// --check's own scope line reads target.pr, so this proves the resolved number
	// reached the configuration rather than only the log.
	if !strings.Contains(out, "pr #170") {
		t.Errorf("the resolved number did not reach the effective config:\n%s", out)
	}
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 (-check must not run agents)", got)
	}
	// Both reads have to happen: the second is what proves the branch names only
	// one open pull request.
	for _, want := range []string{"pr view", "pr list"} {
		if c := ghCalls(t, calls); !strings.Contains(c, want) {
			t.Errorf("gh was not asked to %q; calls were %q", want, c)
		}
	}
}

// An explicit -pr is a choice, and the resolution must not second-guess it: the
// branch may be anything at all when the number was typed, and a `gh pr view`
// here would be a network round trip whose answer nobody asked for.
func TestRunExplicitPRSkipsBranchResolution(t *testing.T) {
	f := newFixture(t)
	testfixture.GitRun(t, f.repo, "checkout", "-q", "-b", "feat/x")
	calls := stubGHForPR(t, 170)

	var buf bytes.Buffer
	if got := run([]string{"-config", f.configFile("pr", "", ""), "-pr", "42", "-review-only", "-check"}, &buf, &buf); got != 0 {
		t.Fatalf("run(-pr 42 -check) = %d, want 0; stderr:\n%s", got, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "pr #42") {
		t.Errorf("-pr 42 did not decide the target:\n%s", out)
	}
	if strings.Contains(out, "resolved from the checked-out branch") {
		t.Errorf("the branch resolution ran over an explicit number:\n%s", out)
	}
	// The fork question is about the TREE and is asked however the number arrived,
	// so gh IS called; what must not happen is the number resolution, whose
	// listing is the one restricted to open pull requests.
	if c := ghCalls(t, calls); strings.Contains(c, "--state open") {
		t.Errorf("the uniqueness probe ran for a number the operator supplied: %q", c)
	}
}

// A config that names a real pull request is also a choice already made, and the
// placeholder zero is the only hole the resolution fills.
func TestRunConfiguredPRSkipsBranchResolution(t *testing.T) {
	f := newFixture(t)
	testfixture.GitRun(t, f.repo, "checkout", "-q", "-b", "feat/x")
	calls := stubGHForPR(t, 170)

	var buf bytes.Buffer
	cfg := f.configFile("pr", "  pr: 99\n", "")
	if got := run([]string{"-config", cfg, "-review-only", "-check"}, &buf, &buf); got != 0 {
		t.Fatalf("run(config pr: 99 -check) = %d, want 0; stderr:\n%s", got, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "pr #99") {
		t.Errorf("the config's own number did not decide the target:\n%s", out)
	}
	if c := ghCalls(t, calls); strings.Contains(c, "--state open") {
		t.Errorf("the uniqueness probe ran although the config named a pull request: %q", c)
	}
}

// The refusal has to reach the operator as a run failure, not as the validation
// error the placeholder zero would otherwise produce -- "PR number required for
// mode pr" says nothing about the branch it just failed to resolve from.
func TestRunReportsAFailedBranchResolution(t *testing.T) {
	f := newFixture(t)
	testfixture.GitRun(t, f.repo, "checkout", "-q", "-b", "feat/x")
	bin := t.TempDir()
	// gh works: its listing exits cleanly and empty, which is how "no pull request
	// here" is spelled, while `pr view` fails the way gh fails for a branch that
	// has none. A gh that failed EVERY call would be the broken-tooling case, and
	// the fork gate refuses that one first (see TestCheckedOutForkFailsClosed).
	script := "#!/bin/sh\n" +
		"if [ \"$2\" = list ]; then printf '%s' '[]'; exit 0; fi\n" +
		"echo 'no pull requests found for branch \"feat/x\"' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	var buf bytes.Buffer
	if got := run([]string{"-config", f.configFile("pr", "", ""), "-review-only", "-check"}, &buf, &buf); got != 1 {
		t.Fatalf("run = %d, want 1 for a branch with no pull request; stderr:\n%s", got, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "could not resolve a pull request for branch feat/x") {
		t.Errorf("the failure does not name the branch it could not resolve:\n%s", out)
	}
	// Not "no pull request found": that exit status also covers a lapsed token, and
	// telling an operator their pull request does not exist invites a duplicate.
	if !strings.Contains(out, "gh auth status") {
		t.Errorf("the failure asserts a cause it cannot know instead of naming both:\n%s", out)
	}
	if !strings.Contains(out, "-pr") {
		t.Errorf("the failure does not tell the operator what to pass instead:\n%s", out)
	}
}

// The resolution runs git and an authenticated gh INSIDE the target, so it must
// not happen before the target-integrity preflight. That ordering matters more
// here than in the pr mode those guards were written for: nothing of a pull
// request's is in the tree until Prepare checks it out, whereas resolving from
// the branch presumes the branch is ALREADY checked out -- so a PATH entry inside
// the reviewed checkout can supply the `gh` that fixpoint would run, with the
// forge token restored into its environment, before any guard had a chance to
// refuse. review-pr asserts only -trusted-bundle, so the refusal is armed; it
// simply used to fire too late (review run 20260909-213147, findings i1 and i8).
//
// Proved by execution, not by message: the planted gh records that it ran, and
// the assertion is that it never did.
func TestRunResolvesPRBehindTheTargetGuards(t *testing.T) {
	f := newFixture(t)
	testfixture.GitRun(t, f.repo, "checkout", "-q", "-b", "feat/x")

	// The habit this models is ordinary: a repo-local bin directory on PATH.
	inside := filepath.Join(f.repo, "bin")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	ran := filepath.Join(t.TempDir(), "planted-gh-ran")
	script := "#!/bin/sh\ntouch " + ran + "\n" +
		"printf '%s' '{\"number\":666,\"state\":\"OPEN\",\"url\":\"u\",\"baseRefName\":\"main\"," +
		"\"headRefName\":\"feat/x\",\"headRepositoryOwner\":{\"login\":\"o\"}}'\n"
	if err := os.WriteFile(filepath.Join(inside, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", inside+string(os.PathListSeparator)+os.Getenv("PATH"))

	var buf bytes.Buffer
	// -check is the cheapest path that reaches the resolution, and it is gated by
	// the no-agent variant of the same preflight.
	code := run([]string{"-config", f.configFile("pr", "", ""), "-review-only", "-check"}, &buf, &buf)
	if code != 1 {
		t.Fatalf("run = %d, want 1: a gh inside the target must be refused; stderr:\n%s", code, buf.String())
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("the target's own gh was executed; the resolution ran ahead of the guard that exists to refuse it")
	}
	if out := buf.String(); !strings.Contains(out, "inside target") {
		t.Errorf("the refusal is not the pinned-helper guard's:\n%s", out)
	}
}

// --check-live LAUNCHES every configured agent command, and the relaxed
// validation (a pr-mode config may now reach it with no number) is what let this
// path skip the target questions entirely -- including whether the checkout is a
// fork's pull request, whose own bundle would be naming those commands (review
// run 20260909-224407).
func TestCheckLiveResolvesBeforePinging(t *testing.T) {
	f := newFixture(t)
	testfixture.GitRun(t, f.repo, "checkout", "-q", "-b", "feat/x")
	// A gh that reports the branch as a FORK's pull request head.
	bin := t.TempDir()
	view := `{"number":170,"state":"OPEN","url":"u","baseRefName":"main","headRefName":"feat/x",` +
		`"headRepositoryOwner":{"login":"them"},"isCrossRepository":true}`
	script := "#!/bin/sh\n" +
		"if [ \"$2\" = view ]; then printf '%s' '" + view + "'; exit 0; fi\n" +
		"if [ \"$2\" = list ]; then printf '%s' '[]'; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	var buf bytes.Buffer
	code := run([]string{"-config", f.configFile("pr", "", ""), "-review-only", "-check-live"}, &buf, &buf)
	if code != 1 {
		t.Fatalf("run(-check-live) = %d, want 1 on a fork's checkout; stderr:\n%s", code, buf.String())
	}
	if out := buf.String(); !strings.Contains(out, "ANOTHER repository") {
		t.Errorf("the fork refusal did not reach --check-live:\n%s", out)
	}
	// The point of the ordering: no agent was launched.
	if got := f.invocations(); got != 0 {
		t.Errorf("agent invocations = %d, want 0 -- the ping ran before the target was authorized", got)
	}
}
