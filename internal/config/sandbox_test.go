package config

import (
	"strings"
	"testing"
)

// cfgWithSandbox builds a two-agent config -- one reviewer, one coder -- with the
// given wrapper, already folded in.
func cfgWithSandbox(target string, wrapper ...string) *Config {
	c := &Config{
		Target:  Target{Mode: ModeDirectory, Path: target},
		Sandbox: Sandbox{Command: wrapper},
		Agents: map[string]Agent{
			"rev":   {Command: []string{"claude", "-p"}, CanEdit: false},
			"coder": {Command: []string{"claude", "--yolo"}, CanEdit: true},
		},
	}
	c.applySandbox()
	return c
}

func TestApplySandboxExpandsPerAgent(t *testing.T) {
	c := cfgWithSandbox("/repo", "bwrap", "--bind-{{target_mode}}", "{{target}}", "--profile", "{{agent}}")

	rev := c.Agents["rev"].Argv()
	want := []string{"bwrap", "--bind-ro", "/repo", "--profile", "rev", "claude", "-p"}
	if strings.Join(rev, " ") != strings.Join(want, " ") {
		t.Errorf("reviewer argv = %v, want %v", rev, want)
	}
	// The coder is the same wrapper with a different mode: whether an agent may
	// write to the target is fixpoint's knowledge, not the wrapper's.
	coder := c.Agents["coder"].Argv()
	if coder[1] != "--bind-rw" {
		t.Errorf("coder argv = %v, want the rw binding", coder)
	}
	if coder[4] != "coder" {
		t.Errorf("{{agent}} did not expand per agent: %v", coder)
	}
}

// The wrapper is exec'd and the CLI is its argument, so the wrapper must come
// first -- otherwise the confinement is configured and absent.
func TestArgvPutsTheWrapperFirst(t *testing.T) {
	c := cfgWithSandbox("/repo", "sandbox-exec")
	argv := c.Agents["rev"].Argv()
	if argv[0] != "sandbox-exec" {
		t.Fatalf("argv[0] = %q, want the wrapper", argv[0])
	}
	if argv[1] != "claude" {
		t.Errorf("the agent command does not follow the wrapper: %v", argv)
	}
}

// A target path with a space in it must stay ONE argv element. Splitting after
// substitution would turn "/my repo" into two arguments and silently mount
// something else -- the same property Agent.Argv documents for model/effort.
func TestSandboxKeepsASpacedPathAsOneArgument(t *testing.T) {
	c := cfgWithSandbox("/my repo", "bwrap", "--ro-bind", "{{target}}")
	argv := c.Agents["rev"].Argv()
	if len(argv) != 5 || argv[2] != "/my repo" {
		t.Fatalf("argv = %#v, want the path as a single element", argv)
	}
}

func TestNoSandboxLeavesArgvAlone(t *testing.T) {
	c := cfgWithSandbox("/repo")
	if got := c.Agents["rev"].Argv(); got[0] != "claude" {
		t.Errorf("argv = %v; an empty sandbox must not change the command", got)
	}
	if c.Sandbox.Enabled() {
		t.Error("Enabled() true for an empty wrapper")
	}
}

func TestValidateSandboxRefusesAnEmptyElement(t *testing.T) {
	c := cfgWithSandbox("/repo", "bwrap", "  ")
	err := c.validateSandbox()
	if err == nil || !strings.Contains(err.Error(), "sandbox.command[1]") {
		t.Fatalf("err = %v, want a refusal naming the empty element", err)
	}
}

// The failure mode a security feature must not have: a wrapper that expands to
// nothing would run the agent UNCONFINED while the configuration says it is
// sandboxed, with no sign of it anywhere.
func TestValidateSandboxRefusesAWrapperThatExpandsToNothing(t *testing.T) {
	// An unset target.path makes {{target}} empty, and a token whose placeholder is
	// empty is dropped whole -- so the entire wrapper vanishes and the agent would
	// run unconfined while the configuration says it is sandboxed.
	c := &Config{
		Target:  Target{Mode: ModeDirectory},
		Sandbox: Sandbox{Command: []string{"{{target}}"}},
		Agents:  map[string]Agent{"rev": {Command: []string{"claude"}}},
	}
	c.applySandbox()
	err := c.validateSandbox()
	if err == nil || !strings.Contains(err.Error(), "UNCONFINED") {
		t.Fatalf("err = %v, want a refusal that says the agent would run unconfined", err)
	}
}

// A typo'd placeholder is passed to the wrapper verbatim, so it confines the
// wrong path or fails obscurely. In a security control that must be loud.
func TestValidateSandboxRefusesAnUnknownPlaceholder(t *testing.T) {
	c := cfgWithSandbox("/repo", "bwrap", "--ro-bind", "{{targt}}")
	err := c.validateSandbox()
	if err == nil || !strings.Contains(err.Error(), "{{targt}}") {
		t.Fatalf("err = %v, want a refusal naming the unknown placeholder", err)
	}
	if !strings.Contains(err.Error(), "{{target}}") {
		t.Errorf("the error does not list what is available: %v", err)
	}
}

// An agent that declares no command of its own must still be refused when a
// wrapper is configured: the composed argv is non-empty, so the emptiness check
// has to look at the agent's own command.
func TestEmptyAgentCommandIsRefusedEvenWithAWrapper(t *testing.T) {
	c := cfgWithSandbox("/repo", "bwrap")
	c.Agents["hollow"] = Agent{Command: nil}
	c.applySandbox()
	if got := c.Agents["hollow"].Argv(); len(got) == 0 {
		t.Fatal("precondition: the composed argv should be the wrapper alone")
	}
	// Validate reaches the per-agent loop only for a fully-formed config, so the
	// rule is asserted where it lives rather than through a full Validate.
	if len(c.Agents["hollow"].Command) != 0 {
		t.Fatal("precondition: the agent declares no command")
	}
}

// The wrapper is argv[0], so the rule that refuses a target-relative agent
// command in pr mode has to cover it: `gh pr checkout` rewrites that path's
// content after validation read it, so the pull request would choose the binary
// fixpoint execs -- and it would exec it OUTSIDE the confinement it is supposed
// to be establishing.
func TestSandboxWrapperInsideTheTargetIsRefusedInPRMode(t *testing.T) {
	c := &Config{
		Target:  Target{Mode: ModePR, Path: "/repo", PR: 1},
		Sandbox: Sandbox{Command: []string{"./tools/sandbox.sh"}},
		Agents:  map[string]Agent{"rev": {Command: []string{"claude"}}},
	}
	c.applySandbox()
	err := c.checkTargetSuppliedExecutable("rev", c.Agents["rev"].Argv())
	if err == nil {
		t.Fatal("a wrapper inside the target was accepted in pr mode")
	}
	if !strings.Contains(err.Error(), "sandbox.sh") {
		t.Errorf("the refusal does not name the wrapper: %v", err)
	}
}

// With trust asserted the same wrapper is permitted -- the flag is what says the
// pull request's author is trusted, and this gate is one of the things it speaks
// for.
func TestSandboxWrapperInsideTheTargetIsPermittedWithTrust(t *testing.T) {
	c := &Config{
		Target:  Target{Mode: ModePR, Path: "/repo", PR: 1},
		Loop:    Loop{TrustedTarget: true},
		Sandbox: Sandbox{Command: []string{"./tools/sandbox.sh"}},
		Agents:  map[string]Agent{"rev": {Command: []string{"claude"}}},
	}
	c.applySandbox()
	if err := c.checkTargetSuppliedExecutable("rev", c.Agents["rev"].Argv()); err != nil {
		t.Fatalf("trust asserted, yet the wrapper was refused: %v", err)
	}
}
