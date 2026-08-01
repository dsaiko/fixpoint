// Package verify runs the deterministic quality gate: the configured build, test,
// and static-check commands that fixpoint executes itself between the coder and
// the round commit.
//
// This is the only part of the loop that produces a signal no model authored.
// Everything else is one agent's judgment reviewed by another agent's judgment, so
// without this a "fixed" finding means only that a model said it fixed something,
// and a "converged" run means only that other models said they saw nothing.
package verify

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

// maxOutput caps what is retained per command. Enough to diagnose a failure and
// hand it back to the coder; bounded so a runaway test suite cannot exhaust memory
// or blow the coder's context.
const maxOutput = 64 << 10

// Result is one command's outcome. Aliased to the shared shape in model so the
// round record, the summary, and this package cannot drift apart.
type Result = model.VerifyResult

// Report is one verification pass.
type Report struct {
	Results []Result `json:"results"`
}

// Passed reports whether every non-optional command succeeded.
func (r Report) Passed() bool {
	for _, res := range r.Results {
		if !res.Optional && !res.Passed {
			return false
		}
	}
	return true
}

// Failures returns the non-optional commands that did not pass.
func (r Report) Failures() []Result {
	var out []Result
	for _, res := range r.Results {
		if !res.Optional && !res.Passed {
			out = append(out, res)
		}
	}
	return out
}

// Regressions returns the non-optional commands that pass in the baseline but
// fail in r. A command already failing before fixpoint touched anything is not
// this run's fault, and treating it as one would refuse to work on any repository
// that starts red. A command missing from the baseline counts as a regression:
// absent evidence that it ever passed, the safe reading is that this run broke it.
//
// A baseline entry that could not RUN -- a missing binary, a bad working
// directory, a timeout -- is treated as missing rather than as a pre-existing
// failure. Such an entry says nothing about the project: it says the gate itself
// never executed. Tolerating its repeat failures would let a misconfigured or
// permanently timing-out command be silently exempt for the whole run, so the
// configured check would never once run successfully and no round would ever be
// blocked by it. Absent evidence, block.
func (r Report) Regressions(baseline Report) []Result {
	was := make(map[string]bool, len(baseline.Results))
	for _, b := range baseline.Results {
		if b.Err != "" {
			continue // unusable baseline for this command; see above
		}
		was[b.Name] = b.Passed
	}
	var out []Result
	for _, res := range r.Results {
		if res.Optional || res.Passed {
			continue
		}
		if passed, known := was[res.Name]; !known || passed {
			out = append(out, res)
		}
	}
	return out
}

// Unrunnable returns the non-optional commands that did not execute at all: a
// missing binary, a bad working directory, a timeout. In a baseline these are the
// entries Regressions refuses to read as pre-existing failures, and the caller
// reports them so an operator is not left believing a check they configured is
// merely "already red".
func (r Report) Unrunnable() []Result {
	var out []Result
	for _, res := range r.Results {
		if !res.Optional && res.Err != "" {
			out = append(out, res)
		}
	}
	return out
}

// Blocking returns the results that must stop the round under the given policy,
// or nil when the round may proceed.
func (r Report) Blocking(policy config.VerifyPolicy, baseline Report) []Result {
	switch policy {
	case config.VerifyMustPass:
		return r.Failures()
	case config.VerifyNoRegressions:
		return r.Regressions(baseline)
	default: // VerifyOff -- Run is not called, but be explicit rather than clever
		return nil
	}
}

// Run executes every configured command in order, in dir, and returns the report.
// Order is sequential and as configured, so a cheap build check fails before an
// expensive test suite runs.
//
// A command that cannot be started, exits non-zero, or exceeds its timeout is a
// failure; ctx cancellation stops the pass and is reported as such. The context
// error is the caller's cue to distinguish "the run was interrupted" from "the
// checks failed", which are very different outcomes for a round.
//
// SECURITY: env is the environment every command receives, as exec.Cmd.Env -- the
// caller passes agent.EnvWithoutCredentials so a verify command cannot read the
// agents' credentials. These commands are argv the TARGET can supply (a bundle
// file inside the target shadows the operator's), so they are the one
// target-controlled execution path in the loop; inheriting fixpoint's whole
// environment here would defeat the filtering buildEnv applies to agents. A nil
// env means "inherit fixpoint's environment" (exec's own convention) and is for
// tests that assert nothing about the environment.
func Run(ctx context.Context, cfg config.Verify, dir string, env []string) Report {
	rep := Report{Results: make([]Result, 0, len(cfg.Commands))}
	for _, c := range cfg.Commands {
		rep.Results = append(rep.Results, runOne(ctx, c, cfg.Timeout.Std(), dir, env))
		if ctx.Err() != nil {
			break // interrupted: do not start further commands
		}
	}
	return rep
}

func runOne(ctx context.Context, c config.VerifyCommand, timeout time.Duration, dir string, env []string) Result {
	res := Result{Name: c.Name, Argv: c.Run, Optional: c.Optional}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, c.Run[0], c.Run[1:]...)
	cmd.Dir = dir
	// Filtered environment: fixpoint's, minus the agents' credentials. See Run.
	cmd.Env = env
	// Same process-group discipline as agents: a build tool spawns children (a
	// compiler, a test binary, a watch process), and killing only the leader on
	// timeout would leave them running and holding the output pipe open.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return agent.KillProcessGroup(cmd) }
	cmd.WaitDelay = 2 * time.Second

	var buf boundedBuffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf // combined: a failure's cause is often split across both

	start := time.Now()
	err := cmd.Run()
	// Own the whole subprocess lifecycle, not just the leader, exactly as
	// agent.Run does. cmd.Cancel (KillProcessGroup) fires only on cancellation, so
	// a build tool that exits SUCCESSFULLY after backgrounding a child (a watcher,
	// a test daemon) leaves that child alive in our process group -- free to edit
	// the repository concurrently with the next check, the clean-tree check, or the
	// round commit, which is how a verified round turns into a commit nobody
	// verified. SIGKILL the whole group on every exit path; it is a no-op once the
	// group is empty, which is the common case.
	_ = agent.KillProcessGroup(cmd)
	res.Duration = time.Since(start)
	// Output can quote anything the build printed, including a secret from the
	// environment, and it is persisted and fed back to the coder.
	res.Output = agent.RedactSecrets(buf.String())

	switch {
	case cmdCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil:
		res.Err = fmt.Sprintf("timed out after %s", timeout)
	case err == nil:
		res.Passed = true
	case ctx.Err() == nil && agent.SucceededDespiteLeakedPipe(cmd, err):
		// The check itself exited 0; a descendant it left behind (a test daemon, a
		// build watcher, a language server) held the output pipe past that exit, so
		// exec reported the drain timeout rather than the success. Classifying that as
		// "could not run" would block the round under must_pass, and -- because a
		// baseline entry carrying an Err counts as having no baseline -- under
		// no_regressions too, for a check that passed.
		res.Passed = true
		res.Output += "\n[fixpoint: the command exited 0 but a descendant held its output pipe open; the capture ends where fixpoint closed the pipe]\n"
	default:
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
		} else {
			// Could not start at all: a missing binary, a bad working directory. A
			// misconfigured gate must fail loudly, never be treated as passing.
			res.Err = err.Error()
		}
	}
	return res
}

// FormatForCoder renders failures as the block handed back to the coder for its
// correction attempt. It leads with the command and exit status, then the captured
// output, because the coder needs to know what to run to reproduce.
func FormatForCoder(blocking []Result) string {
	var sb strings.Builder
	sb.WriteString("## Verification failed\n")
	sb.WriteString("fixpoint ran the project's own checks after your edits. These did not pass.\n")
	sb.WriteString("Fix the cause. Do not disable, skip, or weaken a check to get past it.\n\n")
	for _, r := range blocking {
		fmt.Fprintf(&sb, "### %s — `%s`\n", r.Name, strings.Join(r.Argv, " "))
		if r.Err != "" {
			fmt.Fprintf(&sb, "could not run: %s\n", r.Err)
		} else {
			fmt.Fprintf(&sb, "exit status %d\n", r.ExitCode)
		}
		if out := strings.TrimSpace(r.Output); out != "" {
			fmt.Fprintf(&sb, "\n```\n%s\n```\n", out)
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// Summary renders a one-line-per-command digest for the run log.
func (r Report) Summary() string {
	parts := make([]string, 0, len(r.Results))
	for _, res := range r.Results {
		status := "ok"
		switch {
		case res.Err != "":
			status = res.Err
		case !res.Passed:
			status = fmt.Sprintf("exit %d", res.ExitCode)
		}
		if res.Optional && !res.Passed {
			status += " (optional)"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", res.Name, status))
	}
	return strings.Join(parts, ", ")
}

// boundedBuffer accumulates at most maxOutput bytes and notes the truncation, so
// a command producing gigabytes of output cannot exhaust memory.
//
// stdout and stderr share one instance, and exec dedups a shared writer to a
// single copy goroutine, so Write never races Write. The mutex guards a
// different overlap: cmd.WaitDelay lets Run return while that copy goroutine is
// still draining a pipe a leaked grandchild holds open, so the String() below
// can run concurrently with a Write. strings.Builder is not safe for that --
// it can produce garbled output or panic outright -- which is the same reason
// agent.BoundedBuffer locks.
type boundedBuffer struct {
	mu       sync.Mutex
	b        strings.Builder
	dropped  bool
	overflow int
}

func (w *boundedBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if room := maxOutput - w.b.Len(); room > 0 {
		if len(p) <= room {
			w.b.Write(p)
			return len(p), nil
		}
		w.b.Write(p[:room])
		w.dropped = true
		w.overflow += len(p) - room
		return len(p), nil
	}
	w.dropped = true
	w.overflow += len(p)
	return len(p), nil
}

func (w *boundedBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.dropped {
		return w.b.String()
	}
	return w.b.String() + fmt.Sprintf("\n[... %s of further output dropped; re-run the command to see it all ...]",
		agent.HumanSize(w.overflow))
}
