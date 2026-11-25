package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// Sandbox launches every agent through a wrapper command, so an agent CLI runs
// inside whatever confinement the operator can provide on their platform.
//
// It exists because fixpoint's containment story has one real hole and this is
// the only shape that closes it without fixpoint becoming something else. The
// reviewers are THIRD-PARTY CLIs holding the operator's own credentials: the
// read-only flags they are launched with deny edits and not reads, so a reviewer
// fed a malicious pull request can be prompt-injected into reading ~/.ssh or
// ~/.aws/credentials and quoting it into a finding, and the process-group kill is
// escaped by any descendant that calls setsid. Both are documented in
// docs/security.md, and the documented remedy is "run fixpoint inside a
// container/VM" -- advice an operator has to implement by hand, outside the tool,
// for every invocation.
//
// This makes that advice executable. fixpoint does not implement a sandbox and
// deliberately does not try to: the confinement primitives are per-platform
// (bubblewrap, a podman/docker container, sandbox-exec, a transient systemd
// scope), and which paths must be mounted is a property of the site's credential
// layout rather than of this program. What fixpoint knows and the wrapper does not
// is WHICH target is being reviewed and WHETHER this agent is supposed to be able
// to write to it, so that is exactly what it passes.
//
// What it buys, stated honestly:
//
//   - Host filesystem confinement. Mount the target and the one credential
//     directory the CLI needs, and the reviewer cannot read the operator's other
//     secrets at all -- the exfiltration surface docs/security.md calls
//     unavoidable becomes a mount list.
//   - Descendant containment, on a platform whose sandbox has it. A container or
//     a cgroup scope kills what setsid escapes.
//
// What it does NOT buy, and must never be described as buying: credential
// isolation from the AGENT. The CLI authenticates as the operator, so its
// credential has to be inside the sandbox with it. An agent can still read the
// token it was given. That is a property of driving somebody else's CLI, not
// something a wrapper can fix.
//
// It wraps AGENT invocations only. Verify commands are the operator's own argv and
// git is fixpoint's; putting either inside an agent's confinement would change
// what the gate measures.
type Sandbox struct {
	// Command is the wrapper argv. Empty (the default) means no wrapper: the
	// feature is opt-in, because a wrapper that does not fit the site's credential
	// layout turns every agent into an authentication error.
	//
	// Each element is a template token split on whitespace before substitution,
	// exactly as an agent command is -- so a target path containing a space stays
	// one argv element instead of becoming two arguments.
	//
	// Placeholders:
	//   {{target}}       the absolute path of the directory the invocation works in
	//                    (target.path, or create's snapshot and implement's output
	//                    directory; expanded per invocation, see Agent.ArgvIn)
	//   {{target_mode}}  "rw" for an agent declared can_edit, "ro" otherwise
	//   {{agent}}        the agent's name, for a wrapper that keeps per-agent profiles
	Command []string `yaml:"command"`

	// Builtin records that Command is fixpoint's OWN wrapper, chosen by the
	// operator with -sandbox builtin or FIXPOINT_SANDBOX=builtin, rather than one a
	// bundle file named. Never read from YAML: whether a wrapper is the shipped one
	// is a fact about where it was found, which a config file cannot vouch for.
	//
	// It matters in exactly one place. -agents-sandboxed is refused in pr mode
	// because fixpoint cannot check that an arbitrary wrapper honors
	// {{target_mode}}; the shipped wrapper is fixpoint's own code, found beside the
	// binary and never in a bundle the target can shadow, and its read-only mode is
	// tested -- so for it, and only for it, the assertion holds there too.
	Builtin bool `yaml:"-"`
}

// The operator's choice of wrapper, from -sandbox or FIXPOINT_SANDBOX.
const (
	// SandboxAuto is what the command line uses when neither the flag nor the
	// environment says anything: the bundle's own sandbox.command when it declares
	// one, and the shipped wrapper otherwise. A run is never unconfined by default;
	// that takes an explicit SandboxOff.
	SandboxAuto = "auto"
	// SandboxBuiltin selects the wrapper shipped with this binary, replacing any
	// sandbox.command the bundle set.
	SandboxBuiltin = "builtin"
	// SandboxOff withdraws the shipped wrapper. It never removes a sandbox.command
	// a bundle configured: turning a confinement off is that file's business, not a
	// flag's (see Overrides.apply). With no such command the agents run unconfined.
	SandboxOff = "off"
	// SandboxEnv is the environment variable that sets the choice for every run.
	// The -sandbox flag wins over it.
	SandboxEnv = "FIXPOINT_SANDBOX"
)

// builtinSandboxRel is where the shipped wrapper sits inside a bundle directory.
const builtinSandboxRel = "sandbox/fixpoint-sandbox"

// builtinSandboxDirs lists the directories searched for the shipped wrapper. Only
// the bundle locations tied to the BINARY: a project's config/ or ~/.fixpoint is
// shadowable by what is under review, and a wrapper found there would be the
// reviewed repository choosing its own confinement. A variable so tests can point
// it at a fixture.
var builtinSandboxDirs = binaryBundleDirs

// findBuiltinSandbox returns the shipped wrapper's path, or "" and the places it
// looked.
func findBuiltinSandbox() (string, []string) {
	var searched []string
	for _, dir := range builtinSandboxDirs() {
		p := filepath.Join(dir, filepath.FromSlash(builtinSandboxRel))
		searched = append(searched, p)
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p, searched
		}
	}
	return "", searched
}

// sandboxHostCheck reports whether this host has the program the shipped wrapper
// confines with, so a missing one stops the run at startup instead of at the first
// agent. A variable so tests do not depend on the host.
var sandboxHostCheck = func() error {
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("bwrap"); err != nil {
			return errors.New("bubblewrap (bwrap) is not installed, and it is what confines the agents on Linux; install it (apt install bubblewrap, dnf install bubblewrap)")
		}
	case "darwin":
		if _, err := exec.LookPath("sandbox-exec"); err != nil {
			return errors.New("sandbox-exec was not found, and it is what confines the agents on macOS")
		}
	default:
		return fmt.Errorf("fixpoint's own wrapper supports Linux and macOS, not %s", runtime.GOOS)
	}
	return nil
}

// applySandboxChoice folds -sandbox / FIXPOINT_SANDBOX into the configuration.
//
// builtin REPLACES whatever sandbox.command the bundle set, and asserts
// agents_sandboxed: the shipped wrapper enforces {{target_mode}} itself, which is
// the claim the flag makes, so an operator who picked it has nothing further to
// assert. auto keeps a bundle's own sandbox.command and is builtin otherwise. off
// (and "", which only a caller that is not the command line passes) leaves the
// configuration as the bundle wrote it.
func (o Overrides) applySandboxChoice(c *Config) error {
	why := "-sandbox builtin:"
	switch o.Sandbox {
	case "", SandboxOff:
		return nil
	case SandboxAuto:
		if c.Sandbox.Enabled() {
			return nil
		}
		why = "agents run inside fixpoint's own sandbox by default, but"
	case SandboxBuiltin:
	default:
		return fmt.Errorf("-sandbox %q (or %s): want %q, %q or %q", o.Sandbox, SandboxEnv, SandboxAuto, SandboxBuiltin, SandboxOff)
	}
	if err := sandboxHostCheck(); err != nil {
		return fmt.Errorf("%s %w; or pass -sandbox off (%s=off) to run the agents unconfined", why, err, SandboxEnv)
	}
	p, searched := findBuiltinSandbox()
	if p == "" {
		return fmt.Errorf("%s fixpoint's own wrapper was not found beside this binary (searched %s); the bundle shipped with the binary is incomplete -- reinstall, or pass -sandbox off (%s=off) to run the agents unconfined", why, strings.Join(searched, ", "), SandboxEnv)
	}
	// Command elements are split on whitespace, so a path with a space in it would
	// become two arguments and exec something else. Refused, not quoted: there is no
	// quoting in that syntax to reach for.
	if strings.ContainsAny(p, " \t\n") {
		return fmt.Errorf("%s the wrapper's path %q contains whitespace, which sandbox.command cannot carry as one argument; install fixpoint under a path without spaces", why, p)
	}
	// Run by /bin/sh rather than exec'd, so a packager or an unpacker that dropped
	// the executable bit cannot turn the wrapper into a startup failure.
	c.Sandbox = Sandbox{
		Command: []string{"/bin/sh", p, "{{target}}", "{{target_mode}}", "--"},
		Builtin: true,
	}
	c.Loop.AgentsSandboxed = true
	return nil
}

// Enabled reports whether a wrapper is configured.
func (s Sandbox) Enabled() bool { return len(s.Command) > 0 }

// sandboxPlaceholders are the substitutions available in a wrapper command, for
// one agent.
func sandboxPlaceholders(targetPath, agentName string, canEdit bool) map[string]string {
	mode := "ro"
	if canEdit {
		mode = "rw"
	}
	return map[string]string{
		"{{target}}":      targetPath,
		"{{target_mode}}": mode,
		"{{agent}}":       agentName,
	}
}

// applySandbox folds the configured wrapper into every agent, expanded for that
// agent.
//
// It runs once the target path is FINAL -- after the flag overrides, because
// -target changes what {{target}} means -- and before validation, so the checks
// that guard an agent command (it resolves to a real binary; in pr mode it does
// not resolve inside the target) see the wrapper that will actually be exec'd.
//
// The expansion stored in Agent.Sandbox is against target.path, which is what
// those checks inspect. The template is kept beside it so an invocation that runs
// somewhere else can be expanded for where it runs (Agent.ArgvIn).
func (c *Config) applySandbox() {
	if !c.Sandbox.Enabled() {
		return
	}
	for name, a := range c.Agents {
		a.Sandbox = expandSandbox(c.Sandbox.Command, sandboxPlaceholders(c.Target.Path, name, a.CanEdit))
		a.sandboxTmpl = append([]string(nil), c.Sandbox.Command...)
		a.sandboxName = name
		if c.Loop.AgentsSandboxed && len(a.WhenSandboxed.Replace) > 0 {
			// A copy: the slice may be shared with the agent definition another config
			// in the same process loaded.
			cmd := append([]string(nil), a.Command...)
			for i, tok := range cmd {
				if r, ok := a.WhenSandboxed.Replace[strings.TrimSpace(tok)]; ok {
					cmd[i] = r
				}
			}
			a.Command = cmd
		}
		c.Agents[name] = a
	}
}

// expandSandbox renders a wrapper template for one set of placeholder values.
func expandSandbox(tmpl []string, values map[string]string) []string {
	var argv []string
	for _, tok := range tmpl {
		// Split before substitution, for the reason Agent.Argv documents: a value is
		// then always exactly one argv element however it is spelled, so a target path
		// with a space in it cannot introduce an argument of its own.
		fields := strings.Fields(tok)
		subs := make([]string, 0, len(fields))
		drop := false
		for _, f := range fields {
			sub, keep := expandToken(f, values)
			if !keep {
				drop = true
				break
			}
			subs = append(subs, sub)
		}
		// The WHOLE token goes, exactly as Agent.Argv drops one: "--bind {{target}}"
		// is a flag and its value, and dropping only the value would leave a dangling
		// --bind that swallows whatever argument follows it -- in a sandbox wrapper,
		// the next mount or the agent command itself (review run 20260916-085129,
		// finding i13).
		if drop {
			continue
		}
		argv = append(argv, subs...)
	}
	return argv
}

// validateSandbox rejects a wrapper that cannot work, before a run spends
// anything finding out.
//
// The composed argv is checked by the ordinary agent-command validation, which
// now sees the wrapper as argv[0]: whether it exists, and whether the reviewed
// material could have supplied it. What is left here is the wrapper's own shape.
func (c *Config) validateSandbox() error {
	// -agents-sandboxed claims a wrapper confines the agents; a run without one has
	// nothing behind the claim, and the refusal it lifts would then be lifted for
	// no reason. Fail closed, whichever way the two came apart.
	if c.Loop.AgentsSandboxed {
		if !c.Sandbox.Enabled() {
			return errors.New("-agents-sandboxed claims that sandbox.command confines the agents, but this configuration sets no sandbox.command; drop the flag, or configure the wrapper that enforces read-only")
		}
		// The target of a pr run is the pull request's content, untrusted by
		// definition, and fixpoint cannot check that the wrapper honors {{target_mode}}.
		// Lifting the read-only check for that material would stake it on a wrapper
		// nobody verified, so it is refused rather than allowed with a warning.
		if c.Target.Mode == ModePR && !c.Sandbox.Builtin {
			return errors.New("-agents-sandboxed is refused in mode pr: the target is the pull request's content, and the CLI's own read-only sandbox is the barrier fixpoint can verify; keep it on, use fixpoint's own wrapper (-sandbox builtin), or review the branch in directory or git-diff mode")
		}
	}
	if !c.Sandbox.Enabled() {
		return nil
	}
	for i, tok := range c.Sandbox.Command {
		if strings.TrimSpace(tok) == "" {
			return fmt.Errorf("sandbox.command[%d] is empty; every element must be an argv token", i)
		}
		// A placeholder this build does not know is passed to the sandbox binary
		// VERBATIM -- `{{targt}}` becomes the literal string "{{targt}}" in the mount
		// argument, and the wrapper either fails obscurely or confines the wrong path.
		// A typo in a security control must not be a silent one.
		if ph := unknownPlaceholder(tok); ph != "" {
			return fmt.Errorf("sandbox.command[%d] uses %s, which is not a sandbox placeholder; it would be passed to the wrapper verbatim. Available: %s",
				i, ph, strings.Join(sandboxPlaceholderNames(), ", "))
		}
	}
	// A wrapper whose first element expands to nothing would silently exec the
	// agent CLI directly -- the confinement the operator configured would be absent
	// with no sign of it, which is the one failure mode a security feature must not
	// have. Every agent is checked because the expansion is per-agent.
	for name, a := range c.Agents {
		if len(a.Sandbox) == 0 {
			return fmt.Errorf("agents.%s: sandbox.command expands to nothing for this agent, so it would run UNCONFINED while the configuration says otherwise; check the placeholders in sandbox.command", name)
		}
	}
	return nil
}

// placeholderRE matches a {{name}} token, for the unknown-placeholder check.
var placeholderRE = regexp.MustCompile(`\{\{[^}]*\}\}`)

// unknownPlaceholder returns the first {{...}} in tok that is not a sandbox
// placeholder, or "" when every one of them is known.
func unknownPlaceholder(tok string) string {
	known := sandboxPlaceholders("", "", false)
	for _, ph := range placeholderRE.FindAllString(tok, -1) {
		if _, ok := known[ph]; !ok {
			return ph
		}
	}
	return ""
}

// sandboxPlaceholderNames lists the available placeholders, sorted, for the
// error above. Derived from the same map the expansion uses so a new placeholder
// cannot be added without the message learning about it.
func sandboxPlaceholderNames() []string {
	var out []string
	for ph := range sandboxPlaceholders("", "", false) {
		out = append(out, ph)
	}
	sort.Strings(out)
	return out
}
