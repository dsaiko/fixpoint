package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/model"
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

	// Prompts resolve by bare name from <projectRoot>/config/prompts, and the
	// project root is discovered by walking up from the working directory -- so the
	// fixture writes a real bundle into the repo and runs from there. Chdir also
	// keeps these tests from resolving against fixpoint's own bundle, which would
	// silently exercise the shipped prompts instead of the fixture's.
	promptDir := filepath.Join(f.repo, "config", "prompts")
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
	// Commit the bundle: fix rounds require a clean tree at run start, and a
	// project's config bundle is committed in real use anyway.
	testfixture.GitRun(t, f.repo, "add", "-A")
	testfixture.GitRun(t, f.repo, "commit", "-q", "-m", "add config bundle")
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
}

func TestRunCheckLive(t *testing.T) {
	// -check-live pings the write-capable coder, so it must clear the same
	// fix-round trust gate a real run does; the trusted_target opt-in satisfies it
	// for a directory target. The coder and reviewer are pinged in parallel and
	// share one mock response counter, so register an OK for each invocation index
	// either ordering can land on.
	t.Run("responding agents exit 0", func(t *testing.T) {
		f := newFixture(t)
		f.respond(1, "OK")
		f.respond(2, "OK")
		var buf bytes.Buffer
		if got := run([]string{"-config", f.configFile("directory", "", "  trusted_target: true"), "-check-live"}, &buf, &buf); got != 0 {
			t.Fatalf("run(-check-live) = %d, want 0; stderr:\n%s", got, buf.String())
		}
	})
	t.Run("failing agent exits 1", func(t *testing.T) {
		f := newFixture(t) // no responses: the mock exits non-zero
		var buf bytes.Buffer
		if got := run([]string{"-config", f.configFile("directory", "", "  trusted_target: true"), "-check-live"}, &buf, &buf); got != 1 {
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
		if !strings.Contains(buf.String(), "trusted_target") {
			t.Errorf("stderr must name the opt-in:\n%s", buf.String())
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "rejected", Detail: "by design"}))
	var buf bytes.Buffer
	// A directory fix run must clear the trust gate; trusted_target satisfies it.
	cfg := f.configFile("directory", "", "  max_iterations: 3\n  clean_rounds_to_stop: 1\n  trusted_target: true")
	// Exit 3, not 0: nothing changed, and automation keying on 0 must not read a
	// no-op as a converged run.
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 3 {
		t.Fatalf("run() = %d, want 3 for all-rejected; stderr:\n%s", got, buf.String())
	}
	if !strings.Contains(buf.String(), "done: "+model.TermAllRejected) {
		t.Errorf("stderr missing all-rejected termination line:\n%s", buf.String())
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
		// SIGTERM is safe solely while run()'s signal.NotifyContext handler is
		// installed (run() blocked in the reviewer). If the reviewer never started,
		// run() may have already returned and removed the handler, so a blind
		// SIGTERM would hit the default disposition and kill the whole test binary
		// -- taking down every other test with an ambiguous signal-kill instead of
		// a clean assertion failure. Bail and let the main assertion time out.
		if !seen {
			return
		}
		// run() has registered its signal.NotifyContext handler well before the
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

func TestRunReviewerFailureExits1(t *testing.T) {
	f := newFixture(t)
	f.respond(1, "no review block") // reviewer contract violation
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  review_only: true")
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1; stderr:\n%s", got, buf.String())
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
	if got := run([]string{"-config", cfg}, &buf, &buf); got != 1 {
		t.Fatalf("run() = %d, want 1 for a reviewer contract violation; stderr:\n%s", got, buf.String())
	}
	if strings.Contains(buf.String(), secret) {
		t.Errorf("stderr leaked the credential-shaped value:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "[REDACTED]") {
		t.Errorf("stderr missing the redaction mask (error was not redacted):\n%s", buf.String())
	}
}

// -review-only must force a single review round even when the config enables
// fix rounds: without the override the coder would run and fail (no resp-2).
func TestRunReviewOnlyFlagOverride(t *testing.T) {
	f := newFixture(t)
	f.respond(1, reviewResponse(t, aFinding("bug")))
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  max_iterations: 3")
	if got := run([]string{"-config", cfg, "-review-only"}, &buf, &buf); got != 0 {
		t.Fatalf("run(-review-only) = %d, want 0; stderr:\n%s", got, buf.String())
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
	f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "patched"}))
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  max_iterations: 3\n  trusted_target: true")
	if got := run([]string{"-config", cfg, "-max-iterations", "1"}, &buf, &buf); got != 2 {
		t.Fatalf("run(-max-iterations 1) = %d, want 2; stderr:\n%s", got, buf.String())
	}
}

// A negative -max-iterations is invalid operator input: the override must reach
// Config.Validate (not be silently ignored as a non-positive value), so the run
// aborts with exit 1 and the must-not-be-negative diagnostic before any agent.
func TestRunNegativeMaxIterationsFlagExits1(t *testing.T) {
	f := newFixture(t)
	var buf bytes.Buffer
	cfg := f.configFile("directory", "", "  max_iterations: 3\n  trusted_target: true")
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

// The -trusted-target flag must clear the directory/git-diff fix-round gate
// that the bare config (trusted_target absent) fails closed on.
func TestRunTrustedTargetFlag(t *testing.T) {
	t.Run("directory fix rounds refused without the flag", func(t *testing.T) {
		f := newFixture(t)
		var buf bytes.Buffer
		p := f.configFile("directory", "", "")
		if got := run([]string{"-config", p}, &buf, &buf); got != 1 {
			t.Fatalf("run() = %d, want 1; stderr:\n%s", got, buf.String())
		}
		if !strings.Contains(buf.String(), "trusted_target") {
			t.Errorf("stderr must name the opt-in:\n%s", buf.String())
		}
		if got := f.invocations(); got != 0 {
			t.Errorf("agent invocations = %d, want 0 (refusal comes first)", got)
		}
	})
	t.Run("flag clears the gate and the run proceeds", func(t *testing.T) {
		f := newFixture(t)
		f.respond(1, reviewResponse(t, aFinding("bug")))
		f.editRepoOn(2)
		f.respond(2, fixResponse(t, model.FixResult{ID: "r1.1", Verdict: "fixed", Detail: "patched"}))
		f.respond(3, reviewResponse(t)) // clean round -> converge, exit 0
		var buf bytes.Buffer
		p := f.configFile("directory", "", "")
		if got := run([]string{"-config", p, "-trusted-target"}, &buf, &buf); got != 0 {
			t.Fatalf("run(-trusted-target) = %d, want 0; stderr:\n%s", got, buf.String())
		}
		if strings.Contains(buf.String(), "set loop.trusted_target") {
			t.Errorf("-trusted-target did not suppress the refusal:\n%s", buf.String())
		}
	})
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
		if !strings.Contains(buf.String(), "allow_untrusted_fix") {
			t.Errorf("stderr must name the opt-in:\n%s", buf.String())
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
		if strings.Contains(buf.String(), "allow_untrusted_fix") {
			t.Errorf("-allow-untrusted-fix did not suppress the refusal:\n%s", buf.String())
		}
	})
}
