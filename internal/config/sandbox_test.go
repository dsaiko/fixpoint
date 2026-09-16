package config

import (
	"os"
	"path/filepath"
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
	// The composed argv is non-empty -- the wrapper occupies it -- which is exactly
	// why the emptiness rule has to look at the agent's OWN command.
	if got := c.Agents["hollow"].Argv(); len(got) == 0 {
		t.Fatal("precondition: the composed argv should be the wrapper alone")
	}
	err := c.validateAgentCommand("hollow", c.Agents["hollow"])
	if err == nil {
		t.Fatal("an agent declaring no command was accepted because a wrapper filled its argv")
	}
	if !strings.Contains(err.Error(), "empty command") {
		t.Errorf("the refusal does not name the problem: %v", err)
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
	err := c.checkTargetSuppliedExecutable("rev", c.Agents["rev"])
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
	if err := c.checkTargetSuppliedExecutable("rev", c.Agents["rev"]); err != nil {
		t.Fatalf("trust asserted, yet the wrapper was refused: %v", err)
	}
}

// REGRESSION (review run 20260916-085129, finding i1). The sandbox made argv[0]
// the wrapper, and the bare-name PATH check read argv[0] -- so the agent's own
// `claude`/`codex` stopped being measured against PATH entries inside the target
// entirely. The reviewed repository could then supply or shadow that executable
// and have it run with the credentials the agent declares; the wrapper confines
// the shim but does not stop it running.
func TestSandboxDoesNotHideTheAgentsOwnBareNameFromThePATHCheck(t *testing.T) {
	root := t.TempDir()
	// A PATH entry inside the target, holding a shim named like the agent CLI.
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "pretend-agent"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	// The wrapper is an absolute path OUTSIDE the target, which is the normal
	// shape and precisely what used to make the check return early.
	c := &Config{
		Target:  Target{Mode: ModeDirectory, Path: root},
		Sandbox: Sandbox{Command: []string{"/bin/sh"}},
		Agents:  map[string]Agent{"rev": {Command: []string{"pretend-agent"}}},
	}
	c.applySandbox()
	err := c.checkTargetSuppliedExecutable("rev", c.Agents["rev"])
	if err == nil {
		t.Fatal("a bare agent name resolving through a PATH entry inside the target was accepted because a wrapper occupied argv[0]")
	}
	if !strings.Contains(err.Error(), "pretend-agent") {
		t.Errorf("the refusal names the wrong executable: %v", err)
	}
}

// ExecHeads reports both executables when a wrapper is configured, and exactly
// one when it is not -- the dedup is what keeps the unwrapped case from checking
// the same name twice and reporting it twice.
func TestExecHeads(t *testing.T) {
	wrapped := cfgWithSandbox("/repo", "bwrap", "--")
	if got := wrapped.Agents["rev"].ExecHeads(); len(got) != 2 || got[0] != "bwrap" || got[1] != "claude" {
		t.Errorf("ExecHeads() = %v, want [bwrap claude]", got)
	}
	bare := cfgWithSandbox("/repo")
	if got := bare.Agents["rev"].ExecHeads(); len(got) != 1 || got[0] != "claude" {
		t.Errorf("ExecHeads() with no wrapper = %v, want [claude]", got)
	}
}

// REGRESSION (review run 20260916-085129, finding i18). applySandbox must run
// AFTER the -target override, and until now that was guaranteed only by a comment
// in load.go. Move the call one line up -- an easy refactor in a 400-line loader
// -- and {{target}} expands to the config's path while agent.Run launches the CLI
// in the overridden one: the wrapper then binds and mode-flags a directory that is
// not the one under review, and the agent runs against the real target outside the
// confinement the operator configured. Nothing else in the suite would fail.
func TestSandboxExpandsTheOverriddenTargetNotTheConfigsOwn(t *testing.T) {
	root := t.TempDir()
	configured := filepath.Join(root, "from-config")
	overridden := filepath.Join(root, "from-flag")
	for _, d := range []string{configured, overridden} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"task": "target:\n  mode: directory\n  path: " + configured +
			"\nsandbox:\n  command: [true, \"--bind\", \"{{target}}\"]\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})

	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{Target: overridden})
	if err != nil {
		t.Fatal(err)
	}
	argv := l.Config.Agents["mock"].Argv()
	// The wrapper must name the directory the run will actually review.
	if len(argv) < 3 || argv[2] != overridden {
		t.Fatalf("wrapper argv = %v, want {{target}} expanded to the overridden path %s", argv, overridden)
	}
	if l.Config.Target.Path != overridden {
		t.Fatalf("precondition: target.path = %s, want the override to have applied", l.Config.Target.Path)
	}
}

// REGRESSION (review run 20260916-085129, finding i12). The wrapper documented in
// defaults.yaml and docs/security.md was refused in pr mode -- the mode a sandbox
// matters most in -- because `--target {{target}}` expands to an element naming
// the target, and the refusal told the operator to pass -trusted-target: drop a
// trust gate in order to turn a confinement on.
func TestDocumentedWrapperIsAcceptedInPRMode(t *testing.T) {
	c := &Config{
		Target:  Target{Mode: ModePR, Path: "/repo", PR: 1},
		Sandbox: Sandbox{Command: []string{"/usr/local/bin/fixpoint-sandbox", "--target", "{{target}}", "--mode", "{{target_mode}}", "--"}},
		Agents:  map[string]Agent{"rev": {Command: []string{"claude"}}},
	}
	c.applySandbox()
	if err := c.checkTargetSuppliedExecutable("rev", c.Agents["rev"]); err != nil {
		t.Fatalf("the documented wrapper is refused in pr mode: %v", err)
	}
}

// The exemption is the target ROOT and nothing below it: a wrapper argument
// naming a file the branch can write is still PR-supplied content, whatever the
// argument is called.
func TestWrapperArgumentDeeperInsideTheTargetIsStillRefused(t *testing.T) {
	c := &Config{
		Target:  Target{Mode: ModePR, Path: "/repo", PR: 1},
		Sandbox: Sandbox{Command: []string{"/bin/sh", "--profile", "{{target}}/sandbox.json", "--"}},
		Agents:  map[string]Agent{"rev": {Command: []string{"claude"}}},
	}
	c.applySandbox()
	err := c.checkTargetSuppliedExecutable("rev", c.Agents["rev"])
	if err == nil {
		t.Fatal("a wrapper argument naming a file inside the target was accepted in pr mode")
	}
	if !strings.Contains(err.Error(), "sandbox.json") {
		t.Errorf("the refusal names the wrong element: %v", err)
	}
}

// REGRESSION (review run 20260916-085129, finding i5). {{target}} is expanded
// once, from target.path, but create runs invoke their agents inside the
// assignment snapshot and implement's coder inside the output directory. The
// wrapper would bind one directory while the agent worked in another: a
// confinement that is configured, reported, and aimed at the wrong place.
func TestSandboxIsRefusedWherePlaceholdersWouldNameTheWrongDirectory(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  func(*Config)
		want string
	}{
		{"create", func(c *Config) { c.Create = Create{Propose: "propose"} }, "create run"},
		{"implement", func(c *Config) { c.Roles.Planner = RoleRef{Agent: "mock", Prompt: "plan"} }, "implement run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{
				Target:  Target{Mode: ModeDirectory, Path: "/repo"},
				Sandbox: Sandbox{Command: []string{"/bin/sh", "--"}},
				Agents:  map[string]Agent{"rev": {Command: []string{"claude"}}},
			}
			tc.cfg(c)
			c.applySandbox()
			err := c.validateSandbox()
			if err == nil {
				t.Fatalf("a %s with a sandbox was accepted; {{target}} names a directory its agents do not run in", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal does not say which pipeline: %v", err)
			}
		})
	}
}

// And the ordinary review/fix pipelines, whose agents DO run in target.path, are
// unaffected.
func TestSandboxIsAcceptedForAReviewRun(t *testing.T) {
	c := cfgWithSandbox("/repo", "/bin/sh", "--")
	if err := c.validateSandbox(); err != nil {
		t.Fatalf("a review run with a sandbox was refused: %v", err)
	}
}
