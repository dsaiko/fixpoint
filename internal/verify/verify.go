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
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
	"github.com/dsaiko/fixpoint/internal/prompt"
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
// Order is sequential and as configured; a failure does NOT stop the pass. Two
// callers need the full set: verifyPass, because the coder gets one bounded
// correction attempt and must see every failure at once rather than fixing the
// formatter only to have the test suite fail the retry, and captureVerifyBaseline,
// because Regressions needs a baseline verdict per command. Ordering commands
// cheapest-first is therefore about how the summary reads, not about time saved.
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

	// Config validation rejects an empty run list, so reaching here means the
	// command was built some other way. Record it as a command that could not run
	// rather than indexing c.Run[0] and taking the whole loop down with a panic:
	// one misconfigured gate must fail loudly on its own, not kill the round.
	if len(c.Run) == 0 {
		res.Err = "verify command has no argv configured"
		return res
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, c.Run[0], c.Run[1:]...)
	cmd.Dir = dir
	// Filtered environment: fixpoint's, minus the agents' credentials. See Run.
	cmd.Env = env

	var buf boundedBuffer

	start := time.Now()
	// Same process-group discipline as agents, via the shared supervisor: a build
	// tool spawns children (a compiler, a test binary, a watcher, a test daemon),
	// and a check that exits 0 after backgrounding one must not leave it alive to
	// edit the repository while the next check, the clean-tree check, or the round
	// commit runs -- which is how a verified round turns into a commit nobody
	// verified. agent.Supervise kills the group the instant the leader exits,
	// BEFORE draining the output, so the window in which such a child can still
	// touch the tree does not outlast the command. A nil stderr means one combined
	// capture: a failure's cause is often split across both streams.
	leakedPipe, err := agent.Supervise(cmdCtx, cmd, &buf, nil)
	res.Duration = time.Since(start)
	// Output can quote anything the build printed, including a secret from the
	// environment, and it is persisted and fed back to the coder.
	res.Output = agent.RedactSecrets(buf.String())

	// The leader's own status comes FIRST, exactly as in agent.Run. Supervise
	// returns only after cmd.Wait, the process-group kill and the drain -- and that
	// drain can burn pipeDrainGrace when a descendant escaped the group -- so the
	// deadline can expire in the window after a check exited 0. Reading cmdCtx.Err
	// before err would record such a check as timed out, sending a passing gate back
	// to the coder for a correction it does not need and ultimately discarding valid
	// edits. Reclassify a FAILURE as a timeout, never a success.
	switch {
	case err == nil:
		res.Passed = true
		if leakedPipe {
			// The check itself exited 0, but a descendant that escaped the process group
			// held the output pipe past that exit, so the capture stops where fixpoint
			// stopped draining. Only the capture is affected -- classifying this as
			// "could not run" would block the round under must_pass, and (a baseline entry
			// carrying an Err counts as having no baseline) under no_regressions too, for
			// a check that passed.
			res.Output += "\n[fixpoint: the command exited 0 but a descendant held its output pipe open; the capture ends where fixpoint closed the pipe]\n"
		}
	case cmdCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil:
		res.Err = fmt.Sprintf("timed out after %s", timeout)
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
//
// SECURITY: every part of a Result that reaches the prompt is untrusted. The
// output is whatever the target's own build, test, and lint commands printed while
// running over target content, and the name and argv come from a bundle file the
// target may supply. Rendered raw, a failing test that prints a fence followed by
// its own "## Additional required task" heading would close fixpoint's fence and
// forge prompt structure at the same level as "## Your task" -- and could forge a
// <fix> envelope besides. So it gets exactly what a reviewer's prose gets: the
// "this is data" note, and prompt.Quote, whose per-line "> " marker survives any
// fence and whose defang escapes the contract tags.
func FormatForCoder(blocking []Result) string {
	var sb strings.Builder
	sb.WriteString("## Verification failed\n")
	sb.WriteString("fixpoint ran the project's own checks after your edits. These did not pass.\n")
	sb.WriteString("Fix the cause. Do not disable, skip, or weaken a check to get past it.\n\n")
	sb.WriteString(prompt.UntrustedNote(
		"the output of the project's own check commands, run over code that fixpoint does not trust",
		"diagnostics to act on"))
	for _, r := range blocking {
		fmt.Fprintf(&sb, "### %s — `%s`\n", prompt.Flatten(r.Name), prompt.Flatten(strings.Join(r.Argv, " ")))
		if r.Err != "" {
			fmt.Fprintf(&sb, "could not run: %s\n", prompt.Flatten(r.Err))
		} else {
			fmt.Fprintf(&sb, "exit status %d\n", r.ExitCode)
		}
		if out := prompt.Quote(r.Output); out != "" {
			sb.WriteString("\n" + out + "\n")
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
// stdout and stderr share one instance, and the supervisor gives a combined
// capture a single descriptor and a single copy goroutine, so Write never races
// Write; it also waits for that goroutine before returning, so String() does not
// race a Write either. The mutex stays because neither guarantee is local to
// this file, and strings.Builder under such an overlap garbles output or panics
// outright rather than failing visibly -- the same reason agent.BoundedBuffer
// locks.
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
