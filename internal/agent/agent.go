// Package agent runs a configured agent CLI as a subprocess and extracts the
// tagged JSON envelope from whatever it prints. This is the whole provider
// abstraction: any CLI that accepts a prompt and writes text to stdout works.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
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
	{regexp.MustCompile(`(?i)((?:api[_-]?key|secret|token|password|passwd)\s*[:=]\s*"?)[^\s"'\\]{8,}`), "${1}" + redactionMask},
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
// flags (claude -p without yolo, codex --sandbox read-only, agy --mode plan)
// deny EDITS but still let the agent READ any path on the host. A reviewer fed
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
// the process. See internal/agent/env.go.
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
// setsid escapes that group and keeps running after Run returns. WaitDelay
// bounds the pipe-copy goroutine, not the detached process. Containing such
// descendants requires an OS-level mechanism (a transient cgroup on Linux, a
// job object, or a supervising container); that is not implemented here.
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
	// else. nil means the agent opted into inherit_all, which exec reads as "inherit
	// the parent's environment".
	cmd.Env = buildEnv(a)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return KillProcessGroup(cmd) }
	// Bound how long Wait blocks after cancel/exit: cmd.Stdout/Stderr are
	// non-*os.File writers, so exec copies through a pipe in a goroutine. A
	// CLI-spawned child that inherited the write end keeps that pipe open past
	// the process-group kill, and without WaitDelay cmd.Wait would block on the
	// copy goroutine's EOF forever, hanging Run and leaking the goroutine.
	cmd.WaitDelay = 2 * time.Second
	if a.PromptVia == config.PromptViaStdin {
		cmd.Stdin = strings.NewReader(prompt)
	}
	stdout := NewBoundedBuffer(maxOutput, TruncationMarker(maxOutput))
	stderr := NewBoundedBuffer(maxOutput, TruncationMarker(maxOutput))
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	start := time.Now()
	err := cmd.Run()
	// Own the whole subprocess lifecycle, not just the leader. cmd.Cancel
	// (KillProcessGroup) fires only on ctx cancellation, so a leader that exits
	// SUCCESSFULLY after spawning a detached child leaves that child alive in our
	// process group -- free to edit the repo concurrently with GitClean/commit/a
	// later round (nondeterministic commits, a dirty tree) or to leak forever.
	// SIGKILL the whole group on every exit path; it is a no-op once the group is
	// empty (the common case where the agent spawned nothing).
	_ = KillProcessGroup(cmd)
	// The group kill above cannot rescue the wait it follows: cmd.Run only returns
	// once the pipe-copy goroutines see EOF or WaitDelay expires. So an agent that
	// wrote its whole reply and exited 0 still comes back as ErrWaitDelay whenever
	// some descendant it spawned (an MCP server, a language server, a credential
	// daemon, a plain `child &` in a wrapper script) inherited stdout and outlived
	// it. The leader's exit status is authoritative: report the success and keep the
	// captured reply, instead of discarding a complete review or fix -- which resets
	// the clean streak, or commits the coder's edits as an unverified partial round.
	leakedPipe := ctx.Err() == nil && SucceededDespiteLeakedPipe(cmd, err)
	if leakedPipe {
		err = nil
	}
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("timed out after %s", a.Timeout.Std())
	}
	raw := stdout.String()
	// Unwrap here rather than at the call sites: every consumer of Stdout wants
	// the agent's reply, and only this function knows which agent produced it.
	text, usage := ParseUsage(a.Usage, raw)
	errText := stderr.String()
	if leakedPipe {
		// Say so in the .raw log: the capture is complete in every case seen so far
		// (the leader writes its reply before exiting), but it is cut at the moment
		// exec closed the pipes, so a short reply has an explanation on record.
		errText += "\n[fixpoint: a descendant process held the output pipe open past the agent's successful exit; the capture ends where fixpoint closed the pipe]\n"
	}
	return Result{
		Stdout:    text,
		Stderr:    errText,
		Duration:  time.Since(start),
		Err:       err,
		Usage:     usage,
		rawStdout: raw,
	}
}

// KillProcessGroup SIGKILLs the command's whole process group so CLI-spawned
// children do not linger. It is the cmd.Cancel body, split out so its guards
// are unit-testable and reusable by other packages that launch process-group
// leaders (e.g. target's git/gh subprocesses): ctx may fire before Start
// populates cmd.Process (a nil deref), and a zero pid would make
// syscall.Kill(-0, ...) signal the caller's OWN process group -- killing
// fixpoint itself. Both cases must no-op.
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil || cmd.Process.Pid <= 0 {
		return nil
	}
	// negative pid = the whole process group
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

// SucceededDespiteLeakedPipe reports whether err is cmd.Run's WaitDelay expiry
// for a command whose LEADER exited successfully. It is shared by every runner in
// this program (agent.Run, verify.runOne, target's git/gh runner) because they all
// hit the same interleaving: the leader writes its output and exits 0, a
// descendant that inherited the output pipe keeps it open, no EOF arrives, and
// after WaitDelay exec returns exec.ErrWaitDelay "instead of nil".
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
// It is written by exec.Cmd's background copy goroutine(s) and read via String
// after Wait returns. Those normally do not overlap, but a WaitDelay expiry (a
// leaked child keeping a pipe open) lets Wait return while a copy goroutine is
// still writing, so String would race the Write on strings.Builder. The mutex
// makes both safe under that overlap. Write always reports the full input length
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
