// Package agent runs a configured agent CLI as a subprocess and extracts the
// tagged JSON envelope from whatever it prints. This is the whole provider
// abstraction: any CLI that accepts a prompt and writes text to stdout works.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/gitenv"
	"github.com/dsaiko/fixpoint/internal/model"
)

// Result is one agent invocation's outcome.
type Result struct {
	// Stdout is the agent's reply. When the agent reports usage, this is the reply
	// UNWRAPPED from its machine-readable envelope, so callers extracting the
	// output contract never see the envelope -- see config.AgentUsage.
	Stdout   string
	Stderr   string
	Duration time.Duration
	Err      error
	// Usage is what the CLI reported spending, zero when it reports nothing.
	Usage model.Usage
	// ProviderStatus is the status the CLI's own envelope reported when its
	// PROVIDER refused the call (429 rate limit, 5xx outage), zero otherwise. It
	// separates "this agent is broken" from "the API said no", which are the same
	// exit code and want opposite responses. See config.AgentUsage.ErrorStatus.
	ProviderStatus int
	// rawStdout is the untouched process output, kept for the .raw log: the
	// envelope carries the token counts and turn structure, which is exactly what
	// someone reading a run back wants to see.
	rawStdout string
}

// Raw renders the invocation for the .raw log: the exact command, then both
// streams.
func (r Result) Raw(argv []string) string {
	var sb strings.Builder
	sb.WriteString("$ " + strings.Join(argv, " ") + "\n")
	fmt.Fprintf(&sb, "duration: %s\n", r.Duration.Round(time.Millisecond))
	if r.Err != nil {
		fmt.Fprintf(&sb, "error: %v\n", r.Err)
	}
	// The untouched process output, envelope and all: its token counts and turn
	// structure are the most useful thing in this file when reading a run back.
	out := r.rawStdout
	if out == "" {
		out = r.Stdout
	}
	sb.WriteString("\n--- stdout ---\n" + out)
	if r.Stderr != "" {
		sb.WriteString("\n--- stderr ---\n" + r.Stderr)
	}
	return redactSecrets(sb.String())
}

const redactionMask = "[REDACTED]"

// redactRules mask credential-shaped substrings before raw output is persisted.
// The .raw log keeps the subprocess's verbatim stdout AND stderr, and the agent
// CLIs are third-party tools that commonly print auth diagnostics (an expired
// token, an OAuth refresh failure) to stderr -- so a live secret can otherwise
// become a durable on-disk artifact. This is best-effort defense in depth, not
// a guarantee; the operator guidance in fixpoint.yaml (keep the logs dir out
// of any sync/backup/commit, or drop `raw`) still applies. Rules with a capture
// group keep that visible prefix and mask only the value.
var redactRules = []struct {
	re   *regexp.Regexp
	repl string
}{
	// Authorization headers and bearer tokens, keeping the scheme prefix.
	{regexp.MustCompile(`(?i)(authorization\s*[:=]\s*(?:bearer|basic|token)?\s*)[A-Za-z0-9._~+/=-]{8,}`), "${1}" + redactionMask},
	{regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]{16,}`), "${1}" + redactionMask},
	// Well-known provider token formats.
	{regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{16,}`), redactionMask},
	// OpenAI project-scoped keys (sk-proj-...) carry a `-` right after the
	// prefix, which the class below excludes, so they need their own rule; it
	// runs after sk-ant- so Anthropic keys still match their dedicated rule first.
	{regexp.MustCompile(`sk-proj-[A-Za-z0-9_-]{16,}`), redactionMask},
	// Other OpenAI-style keys. The class includes `-`/`_` so a key whose value
	// contains those separators (as sk-proj- keys do) is not cut short.
	{regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`), redactionMask},
	{regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`), redactionMask},
	{regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`), redactionMask},
	{regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`), redactionMask},
	{regexp.MustCompile(`AKIA[0-9A-Z]{16}`), redactionMask},
	{regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`), redactionMask},
	// Stripe secret/restricted keys (sk_live_/sk_test_ and rk_ variants). The
	// value after the prefix is alphanumeric, so the sk-/sk_proj rules above (which
	// key off `sk-`) never reach these underscore-delimited keys.
	{regexp.MustCompile(`[sr]k_(?:live|test)_[0-9A-Za-z]{10,}`), redactionMask},
	// JSON Web Tokens: three base64url segments joined by dots, the header
	// beginning with the `eyJ` that base64-encodes `{"`. Masks the whole token
	// (a signed JWT is itself a bearer credential).
	{regexp.MustCompile(`eyJ[A-Za-z0-9_-]{6,}\.[A-Za-z0-9_-]{6,}\.[A-Za-z0-9_-]{6,}`), redactionMask},
	// PEM private key blocks (RSA/EC/OPENSSH/PKCS8 etc.). (?s) lets `.` span the
	// newlines between the BEGIN/END markers so the entire key body is masked.
	{regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`), redactionMask},
	// Passwords embedded in URI connection strings (scheme://user:pass@host).
	// The user is optional (scheme://:pass@host); the host part after @ is kept.
	{regexp.MustCompile(`(?i)([a-z][a-z0-9+.\-]*://[^\s:/@]*:)[^\s:/@]+@`), "${1}" + redactionMask + "@"},
	// Generic key/token/secret/password assignments with a longish value. The
	// value class excludes the backslash so this rule never consumes a JSON
	// string escape: when redaction runs over already-serialized JSON (the .json
	// step log and the summary), a masked value must not eat the "\" of a
	// trailing \" and leave the surrounding string quote unbalanced.
	//
	// The keyword and the value may each be wrapped in a quote, optionally
	// backslash-escaped, so the JSON shapes are covered too: bare `password=v`,
	// `"password": "v"` (a config file inside a reviewed diff), and `\"password\":
	// \"v\"` (that same JSON once embedded in a JSON string). Without the quote
	// after the keyword the delimiter would have to follow it directly and every
	// quoted-key form -- the common one for an opaque password or internal token
	// that matches none of the shape-based rules above -- would escape redaction.
	// Only the value is masked; the captured prefix (quotes and all) is kept, so
	// admitting the quotes here cannot unbalance anything either.
	{regexp.MustCompile(`(?i)((?:\\?["'])?(?:api[_-]?key|secret|token|password|passwd)(?:\\?["'])?\s*[:=]\s*(?:\\?["'])?)[^\s"'\\]{8,}`), "${1}" + redactionMask},
}

// redactSecrets applies redactRules in order; later rules see already-masked
// text, which only ever over-redacts (acceptable for a debug artifact).
func redactSecrets(s string) string {
	for _, r := range redactRules {
		s = r.re.ReplaceAllString(s, r.repl)
	}
	return s
}

// RedactSecrets masks credential-shaped substrings before text is persisted to
// any on-disk log. It is the exported entry point so every persisted format
// (raw, prompt, md, json) shares one redaction pass, not just raw output. Same
// best-effort caveat as redactRules -- keeping the logs dir private is the real
// control. Applying it twice is harmless (masked text does not re-match).
func RedactSecrets(s string) string { return redactSecrets(s) }

// Run executes the agent with the prompt, in dir, honoring the configured
// timeout. The process group is killed on timeout/cancel so CLI-spawned
// children do not linger.
//
// SECURITY (reads are NOT confined to dir): Run only sets cmd.Dir and kills the
// process group; it applies no filesystem sandbox. The per-agent read-only
// flags (claude -p without yolo, codex --sandbox read-only) deny EDITS but
// still let the agent READ any path on the host. A reviewer fed
// untrusted content (e.g. a malicious PR) can therefore be prompt-injected into
// reading a host secret (~/.ssh/id_rsa, ~/.aws/credentials, .env) and quoting
// it into a finding, even in a review-only run. Point reviewers at untrusted
// content only on a host without sensitive files, or run under an external
// sandbox (container/VM). Full in-process confinement (landlock/chroot) is not
// implemented and is platform-specific; this is a documented limitation.
//
// SECURITY (the process ENVIRONMENT is filtered, not inherited): Run sets cmd.Env
// to a baseline of non-secret variables plus whatever the agent's own file
// declares, so an exported secret the agent did not ask for is simply absent from
// the process. See internal/agent/env.go. It also adds internal/gitenv's
// GIT_CONFIG_* pins, so the git commands the CLI runs inside the target do not
// honor a repo-supplied setting that would execute a target-controlled program.
//
// This closes what was the one exfiltration surface a container could not: the
// agents' credentials must be inside the container for the CLIs to work, so an
// env-based secret used to leak even inside the recommended sandbox. A reviewer
// runs in a mode that denies EDITS, not READS -- on Linux a process can read its
// own /proc/self/environ, and any CLI with a shell tool can run `env` -- and
// redaction only masks fixed-shape tokens, so a bare database password would pass
// through unmasked.
//
// What it does NOT close: an agent authenticating via an environment variable must
// be given that variable, so its own credential stays reachable by the process
// that needs it. The win is everything else. An agent that declares
// `env.inherit_all: true` opts back out entirely, and fixpoint warns at run start.
//
// SECURITY (descendants that escape the process group): KillProcessGroup below
// SIGKILLs the command's process group, but a descendant that calls setpgrp or
// setsid escapes that group and keeps running after Run returns. Supervise's
// drain grace bounds the pipe-copy goroutine, not the detached process.
// Containing such
// descendants requires an OS-level mechanism (a transient cgroup on Linux, a
// job object, or a supervising container); that is not implemented here. Until
// it is, the residual risk is operational and documented under "Security model"
// in the README: run an agent CLI you do not trust under an external
// container/VM, and do not grant it env.inherit_all -- an escaped descendant
// keeps whatever environment its agent was given.
func Run(ctx context.Context, a config.Agent, prompt, dir string) Result {
	argv := a.Argv()
	// Defensive: config validation rejects an empty command, but an agent
	// reaching Run with no argv (e.g. a blank pool entry that slipped past
	// validation) would otherwise index argv[0] and panic the whole process.
	// Fail this one invocation instead.
	if len(argv) == 0 {
		return Result{Err: errors.New("agent has no command configured (empty argv)")}
	}
	// Defensive: config.Load defaults "" to "stdin" and Validate rejects any
	// other value, but Run is exported and called with hand-built configs. Only
	// "arg" and "stdin" actually deliver the prompt below; any other value would
	// silently start the agent with NO prompt. Fail closed like the empty-argv
	// guard rather than run an agent that received nothing.
	if a.PromptVia != config.PromptViaArg && a.PromptVia != config.PromptViaStdin {
		return Result{Err: fmt.Errorf("agent prompt_via %q is not supported (want \"arg\" or \"stdin\"); refusing to run with no prompt delivered", a.PromptVia)}
	}
	if a.PromptVia == config.PromptViaArg {
		argv = append(argv, prompt)
	}

	ctx, cancel := context.WithTimeout(ctx, a.Timeout.Std())
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	// Filtered environment: the baseline plus what this agent declared, and nothing
	// else. buildEnv returns nil for inherit_all, which gitenv.Harden expands to
	// fixpoint's own environment rather than leaving cmd.Env nil -- the git pins below
	// must hold on that path too.
	//
	// gitenv.Harden adds the GIT_CONFIG_* overrides that neutralize the target's own
	// execution-capable git settings (core.fsmonitor, core.hooksPath, ...). The CLI is
	// launched with its working directory inside the target and runs its own
	// `git status`/`git diff`/`git log` while exploring, so without these it would
	// spawn a target-supplied program with the credential this agent declared -- the
	// same hole the -c overrides close for fixpoint's own git calls. The keys with
	// dynamic names, which no override can reach, are refused by the preflight
	// instead (target.UnsafeConfig).
	cmd.Env = gitenv.Harden(buildEnv(a))
	if a.PromptVia == config.PromptViaStdin {
		cmd.Stdin = strings.NewReader(prompt)
	}
	stdout := NewBoundedBuffer(maxOutput, TruncationMarker(maxOutput))
	stderr := NewBoundedBuffer(maxOutput, TruncationMarker(maxOutput))

	start := time.Now()
	// Supervise, not cmd.Run: it owns the whole subprocess lifecycle rather than
	// just the leader, and kills the process group the instant the leader exits.
	// An agent that exits SUCCESSFULLY after spawning a child (an MCP server, a
	// language server, a credential daemon, a plain `child &` in a wrapper script)
	// would otherwise leave it alive -- free to edit the repo concurrently with
	// GitClean/commit/a later round, or to leak forever -- and, if that child
	// inherited stdout, would hold cmd.Run for the whole pipe drain first.
	//
	// Supervise also reports the LEADER's status when only a drain timed out: an
	// agent that wrote its whole reply and exited 0 must not come back as a
	// failure, which discards a complete review or fix -- resetting the clean
	// streak, or committing the coder's edits as an unverified partial round.
	leakedPipe, err := Supervise(ctx, cmd, stdout, stderr)
	// Reclassify a FAILURE as a timeout, never a success. Supervise returns only
	// after cmd.Wait, the process-group kill and drainAll -- and drainAll can burn
	// pipeDrainGrace when a descendant escaped the group -- so the deadline can
	// expire in the window after a leader exited 0 with its whole reply captured.
	// Overwriting a nil err there would discard a complete review or fix for a run
	// that actually succeeded, exactly what SucceededDespiteLeakedPipe prevents.
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("timed out after %s", a.Timeout.Std())
	}
	raw := stdout.String()
	// Unwrap here rather than at the call sites: every consumer of Stdout wants
	// the agent's reply, and only this function knows which agent produced it.
	text, usage, providerStatus := parseEnvelope(a.Usage, raw)
	// A failed agent that still produced an envelope explained itself in it, and
	// that explanation is worth more than the exit code: `exit status 1` reads as a
	// broken coder, while the reply it came with says "You've hit your session
	// limit · resets 8:20pm". Only the first line, and only when it is not the raw
	// output (ParseUsage falls back to raw when it cannot decode), so an
	// unparseable dump is not spliced into the error.
	if err != nil && text != "" && text != raw {
		if msg := firstLine(text); msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
	}
	if err != nil && providerStatus > 0 {
		err = fmt.Errorf("%w (provider returned %d -- the API refused the call, the agent did not fail)", err, providerStatus)
	}
	errText := stderr.String()
	if leakedPipe {
		// Say so in the .raw log: a descendant that escaped the process group kept
		// writing past the agent's exit, so the capture is cut at the moment fixpoint
		// stopped draining and a short reply has an explanation on record.
		errText += "\n[fixpoint: a descendant process held the output pipe open past the agent's exit; the capture ends where fixpoint closed the pipe]\n"
	}
	return Result{
		Stdout:         text,
		Stderr:         errText,
		Duration:       time.Since(start),
		Err:            err,
		Usage:          usage,
		ProviderStatus: providerStatus,
		rawStdout:      raw,
	}
}

// firstLine is the first non-empty line of s, trimmed. Agent replies are wrapped
// into one-line errors, and a CLI's failure message routinely carries a multi-line
// body after its headline.
func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}

// groupKill signals pid's whole PROCESS GROUP -- the negative-pid form of kill(2)
// that both the containment kill in KillProcessGroup and the state probe in
// epermMeansGroupGone go through. It is the single seam for both because the rule
// they implement together is a composition of the two calls, and that composition
// is the containment: an EPERM from the kill may only be downgraded to
// already-finished when the probe then says the group is empty. The reply that
// makes it matter needs a group member running under other credentials, which a
// test cannot create, so the seam is what lets the composition be driven end to
// end. Nothing in production reassigns it.
var groupKill = func(pid int, sig syscall.Signal) error {
	return syscall.Kill(-pid, sig)
}

// KillProcessGroup SIGKILLs the command's whole process group so CLI-spawned
// children do not linger. It is the cmd.Cancel body, split out so its guards
// are unit-testable and reusable by other packages that launch process-group
// leaders (e.g. target's git/gh subprocesses): ctx may fire before Start
// populates cmd.Process (a nil deref), and a zero pid would make
// syscall.Kill(-0, ...) signal the caller's OWN process group -- killing
// fixpoint itself. Both cases must no-op.
//
// A group that is already gone reports os.ErrProcessDone, which is what os/exec
// requires of a cmd.Cancel: see the ESRCH comment below.
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil || cmd.Process.Pid <= 0 {
		return nil
	}
	err := groupKill(cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		// The group is empty: the leader exited AND was reaped, and no descendant is
		// left to keep the group alive -- there is nothing here that still needs
		// killing. cmd.Wait does that reaping, so it races the context watcher that
		// calls this as cmd.Cancel: a command that completes just as its deadline
		// expires lands here. os/exec only treats a cancel error as the benign
		// already-finished case when it wraps os.ErrProcessDone; hand it the raw
		// errno instead and it replaces the leader's own successful result with
		// `exec: canceling Cmd: no such process` -- a passing check reported as
		// unrunnable, a complete review thrown away, a commit that landed reported
		// as failed.
		return os.ErrProcessDone
	}
	// EPERM is where the kernels disagree about what a vanished group answers, and
	// the disagreement is not cosmetic: off darwin it is the one reply that PROVES
	// the group still holds something this process cannot kill, so it may only be
	// downgraded to already-finished after the state is re-checked. See the two
	// epermMeansGroupGone implementations.
	if errors.Is(err, syscall.EPERM) && epermMeansGroupGone(cmd.Process.Pid) {
		return os.ErrProcessDone
	}
	return err
}

// SucceededDespiteLeakedPipe reports whether err is exec's WaitDelay expiry for
// a command whose LEADER exited successfully. Supervise applies it on behalf of
// every runner in this program (agent.Run, verify.runOne, target's git/gh
// runner) because they all hit the same interleaving: the leader writes its
// output and exits 0, a descendant that inherited a pipe exec owns keeps it
// open, no EOF arrives, and after WaitDelay exec returns exec.ErrWaitDelay
// "instead of nil".
//
// That error says the pipe drain timed out, NOT that the command failed -- the
// process state carries the leader's real exit status. Treating the two the same
// turns a successful invocation into a reported failure, which is a far more
// damaging outcome here than a possibly-short capture: a passing check reads as
// unrunnable, a complete review is thrown away, a commit that landed is reported
// as failed. Callers keep the captured output and note that it may be cut.
func SucceededDespiteLeakedPipe(cmd *exec.Cmd, err error) bool {
	return errors.Is(err, exec.ErrWaitDelay) && cmd.ProcessState != nil && cmd.ProcessState.Success()
}

// maxOutput caps captured process output so a misbehaving agent cannot
// exhaust memory; excess is dropped with a marker.
const maxOutput = 10 << 20 // 10 MB per stream

// TruncationMarker is the suffix a BoundedBuffer appends when it drops output
// past a cap of n bytes. Deriving the size text from the cap keeps the marker
// honest: change the cap constant and the message it prints follows, instead of
// silently lying to the operator about where the output was cut.
func TruncationMarker(n int) string {
	return fmt.Sprintf("\n[... output truncated at %s ...]", HumanSize(n))
}

// HumanSize renders a byte count as a compact MB/KB figure for truncation
// markers. It is deliberately coarse (the exact byte cap is an implementation
// detail); its only contract is that the number it prints tracks the constant
// it is derived from so the two can never drift apart.
func HumanSize(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%d MB", n/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%d KB", n/1_000)
	default:
		return fmt.Sprintf("%d bytes", n)
	}
}

// BoundedBuffer accumulates writes up to max bytes, then drops the rest and
// appends marker to String's result to record that it truncated. It is the one
// shared implementation for capping subprocess output: agent.Run caps each
// stream here, and target's git/gh runner reuses it with its own (smaller) cap
// and marker, so the short-write and truncation semantics cannot drift apart
// between two copies.
//
// It is written by a background copy goroutine and read via String once the
// command is done. Supervise waits for its own copy goroutines before returning,
// so those two no longer overlap for the runners in this program -- but the
// mutex stays: the guarantee lives in another function, one instance may serve
// two streams (verify's combined capture), and strings.Builder under a
// concurrent Write/String garbles output or panics outright rather than failing
// visibly. Write always reports the full input length
// so os/exec's io.Copy never fails with io.ErrShortWrite once the cap is hit --
// oversized output is truncated, not turned into a write error.
type BoundedBuffer struct {
	limit  int
	marker string

	mu        sync.Mutex
	b         strings.Builder
	truncated bool
}

// NewBoundedBuffer returns a buffer that stores at most limit bytes and appends
// marker to String's result once any input has been dropped.
func NewBoundedBuffer(limit int, marker string) *BoundedBuffer {
	return &BoundedBuffer{limit: limit, marker: marker}
}

func (b *BoundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p) // report the full write below so the child never sees a short-write error
	if room := b.limit - b.b.Len(); room > 0 {
		if len(p) > room {
			p = p[:room]
			b.truncated = true
		}
		b.b.Write(p)
	} else {
		b.truncated = true
	}
	return n, nil
}

func (b *BoundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.truncated {
		return b.b.String() + b.marker
	}
	return b.b.String()
}

// maxExtractAttempts bounds how many candidate tag pairs ExtractJSON parses.
// Pairing is quadratic in the number of tags, and the reviewed content can
// steer an agent into emitting pathological tag sequences; unbounded pairing
// over a 10 MB capture could stall the orchestrator. Legitimate output needs
// a handful of attempts at most.
const maxExtractAttempts = 100

// ExtractJSON finds the agent's FINAL <tag>...</tag> block in the output and
// unmarshals its JSON payload into out.
//
// Agent output is hostile territory for naive matching: the echoed prompt
// contains the contract's example block, the model may quote literal tags
// inside finding descriptions (JSON strings), and weaker models wrap the
// payload in markdown fences. The contract requires the tagged block to be the
// LAST thing the agent prints, so extraction anchors on the final closing tag
// and tries its candidate openings from nearest to furthest back; the first
// that parses as valid JSON wins.
//
// Crucially, if none of the final block's candidates parse, this returns an
// error rather than falling back to an EARLIER closing tag. Scanning older
// blocks would let a reviewer that echoes or quotes a valid <tag>{...}</tag>
// (from the prompt or the reviewed source) and then emits a malformed final
// response be accepted via the earlier echoed block -- an embedded
// {"findings":[]} could turn a failed reviewer into a clean review and cause
// false convergence. Failing closed treats a malformed final block as the
// failure it is.
func ExtractJSON(output, tag string, out any) error {
	// Every candidate is unmarshaled into a FRESH value and copied into out
	// only on full success: the JSON decoder populates fields as it goes, so a
	// candidate that sets early fields before failing on a later one would
	// otherwise leave that partial state behind for a subsequent candidate to
	// inherit (e.g. a malformed final block populates Results, then an earlier
	// empty object parses cleanly yet keeps those stale verdicts).
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("ExtractJSON: out must be a non-nil pointer, got %T", out)
	}
	elem := rv.Elem().Type()

	openTag, closeTag := "<"+tag+">", "</"+tag+">"
	// Anchor on the final closing tag only. A literal </tag> quoted inside the
	// payload's JSON strings is not the last one (the real closing tag follows
	// it), so LastIndex still lands on the true block end; the opening scan below
	// then steps back past any literal <tag> quoted inside the payload.
	c := strings.LastIndex(output, closeTag)
	if c < 0 {
		return fmt.Errorf("no <%s> block found in agent output", tag)
	}
	// The contract requires the tagged block to be the LAST thing printed, so
	// only whitespace may follow its closing tag. Without this check, an agent
	// that QUOTES an earlier valid block (e.g. <review>{"findings":[]}</review>)
	// and then emits its real -- untagged -- response would have that quoted
	// block accepted as a clean final review, turning a failed reviewer into a
	// false convergence. Trailing non-whitespace means the true final response
	// was not tagged: fail closed rather than fall back to the quoted block.
	if strings.TrimSpace(output[c+len(closeTag):]) != "" {
		return fmt.Errorf("<%s> block is not the last output (the contract requires the tagged block to be final; untagged trailing output follows the last </%s>)", tag, tag)
	}
	var lastErr error
	seen := false
	attempts := 0
	for o := strings.LastIndex(output[:c], openTag); o >= 0; o = strings.LastIndex(output[:o], openTag) {
		seen = true
		if attempts >= maxExtractAttempts {
			return fmt.Errorf("no valid JSON in the last %d <%s> candidate blocks: %w", maxExtractAttempts, tag, lastErr)
		}
		attempts++
		payload := strings.TrimSpace(output[o+len(openTag) : c])
		payload = strings.TrimPrefix(payload, "```json")
		payload = strings.TrimPrefix(payload, "```")
		payload = strings.TrimSuffix(payload, "```")
		payload = strings.TrimSpace(payload)
		fresh := reflect.New(elem)
		err := json.Unmarshal([]byte(payload), fresh.Interface())
		if err == nil {
			rv.Elem().Set(fresh.Elem())
			return nil
		}
		lastErr = err
	}
	if !seen {
		return fmt.Errorf("no <%s> block found in agent output", tag)
	}
	return fmt.Errorf("invalid JSON in <%s> block: %w", tag, lastErr)
}
