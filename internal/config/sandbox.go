package config

import (
	"errors"
	"fmt"
	"regexp"
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
	//   {{target}}       the absolute path of the directory under review
	//   {{target_mode}}  "rw" for an agent declared can_edit, "ro" otherwise
	//   {{agent}}        the agent's name, for a wrapper that keeps per-agent profiles
	Command []string `yaml:"command"`
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
func (c *Config) applySandbox() {
	if !c.Sandbox.Enabled() {
		return
	}
	for name, a := range c.Agents {
		values := sandboxPlaceholders(c.Target.Path, name, a.CanEdit)
		var argv []string
		for _, tok := range c.Sandbox.Command {
			// Split before substitution, for the reason Agent.Argv documents: a value is
			// then always exactly one argv element however it is spelled, so a target path
			// with a space in it cannot introduce an argument of its own.
			for _, f := range strings.Fields(tok) {
				sub, keep := expandToken(f, values)
				if !keep {
					continue
				}
				argv = append(argv, sub)
			}
		}
		a.Sandbox = argv
		c.Agents[name] = a
	}
}

// validateSandbox rejects a wrapper that cannot work, before a run spends
// anything finding out.
//
// The composed argv is checked by the ordinary agent-command validation, which
// now sees the wrapper as argv[0]: whether it exists, and whether the reviewed
// material could have supplied it. What is left here is the wrapper's own shape.
func (c *Config) validateSandbox() error {
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
	// {{target}} is expanded ONCE, from target.path, but the create and implement
	// pipelines deliberately run agents somewhere else: create in the assignment
	// snapshot, implement's coder in the output directory. The wrapper would then
	// bind and mode-flag a directory the agent is not in, while the agent works in
	// one the wrapper never mounted -- a confinement that is configured, reported,
	// and aimed at the wrong place. That is the failure mode this file's own
	// doc says a security control must not have, so it is refused rather than
	// approximated (review run 20260916-085129, finding i5). Making the expansion
	// per-invocation is the real fix and is not done here.
	switch {
	case c.IsCreate():
		return errors.New("sandbox.command is set, but a create run invokes its agents inside the assignment snapshot rather than target.path, so {{target}} would name a directory the agent is not working in. Run create without a sandbox, or point target.path at the snapshot")
	case c.IsImplement():
		return errors.New("sandbox.command is set, but an implement run invokes its coder inside the output directory rather than target.path, so {{target}} would name the design's directory while the coder writes somewhere the wrapper never mounted. Run implement without a sandbox")
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
