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

// REGRESSION (review run 20260916-085129, finding i5), and its real fix.
// {{target}} used to be expanded once, from target.path, while create runs invoke
// their agents inside the assignment snapshot and implement's coder inside the
// output directory -- so those runs were refused. The wrapper is now expanded per
// invocation for the directory the agent runs in (Agent.ArgvIn), and both
// pipelines are accepted.
func TestSandboxIsAcceptedForCreateAndImplement(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  func(*Config)
	}{
		{"create", func(c *Config) { c.Create = Create{Propose: "propose"} }},
		{"implement", func(c *Config) { c.Roles.Planner = RoleRef{Agent: "mock", Prompt: "plan"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{
				Target:  Target{Mode: ModeDirectory, Path: "/repo"},
				Sandbox: Sandbox{Command: []string{"/bin/sh", "--"}},
				Agents:  map[string]Agent{"rev": {Command: []string{"claude"}}},
			}
			tc.cfg(c)
			c.applySandbox()
			if err := c.validateSandbox(); err != nil {
				t.Fatalf("a %s with a sandbox was refused: %v", tc.name, err)
			}
		})
	}
}

// The wrapper confines the directory the invocation RUNS in, not target.path.
func TestArgvInExpandsTheTargetForTheInvocationsDirectory(t *testing.T) {
	c := cfgWithSandbox("/repo", "wrap", "{{target}}", "{{target_mode}}", "{{agent}}", "--")
	coder := c.Agents["coder"]
	got := coder.ArgvIn("/out/project")
	want := []string{"wrap", "/out/project", "rw", "coder", "--", "claude", "--yolo"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("ArgvIn = %v, want %v", got, want)
	}
	// For the ordinary invocation, in target.path, it is exactly Argv.
	if strings.Join(coder.ArgvIn("/repo"), "|") != strings.Join(coder.Argv(), "|") {
		t.Errorf("ArgvIn(target.path) = %v, differs from Argv %v", coder.ArgvIn("/repo"), coder.Argv())
	}
	// A relative directory is what exec.Cmd.Dir resolves against fixpoint's cwd;
	// the wrapper, started IN it, must not be handed it relative.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if got := coder.ArgvIn("sub"); got[1] != filepath.Join(wd, "sub") {
		t.Errorf("relative dir reached the wrapper as %q, want %q", got[1], filepath.Join(wd, "sub"))
	}
}

func TestArgvInWithoutASandboxIsArgv(t *testing.T) {
	c := cfgWithSandbox("/repo")
	if got := c.Agents["rev"].ArgvIn("/elsewhere"); strings.Join(got, " ") != "claude -p" {
		t.Errorf("ArgvIn = %v; no wrapper must mean the bare command", got)
	}
}

// codexLike is a read-only agent whose own sandbox has to be switched off inside
// a wrapper, the shape of the shipped codex agents.
func codexLike() Agent {
	return Agent{
		Command:       []string{"codex", "exec", "--sandbox read-only", "-"},
		WhenSandboxed: AgentWhenSandboxed{Replace: map[string]string{"--sandbox read-only": "--sandbox danger-full-access"}},
	}
}

func TestWhenSandboxedSwapsOnlyUnderTheAssertion(t *testing.T) {
	for _, tc := range []struct {
		name     string
		wrapper  []string
		asserted bool
		want     string
	}{
		{"no wrapper", nil, false, "read-only"},
		// A wrapper the operator has not vouched for: the CLI's own sandbox stays,
		// and so does the refusal that reads the argv.
		{"wrapper, not asserted", []string{"wrap", "--"}, false, "read-only"},
		{"wrapper, asserted", []string{"wrap", "--"}, true, "danger-full-access"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{
				Target:  Target{Mode: ModeDirectory, Path: "/repo"},
				Sandbox: Sandbox{Command: tc.wrapper},
				Loop:    Loop{AgentsSandboxed: tc.asserted},
				Agents:  map[string]Agent{"cx": codexLike()},
			}
			c.applySandbox()
			if argv := strings.Join(c.Agents["cx"].Argv(), " "); !strings.Contains(argv, "--sandbox "+tc.want) {
				t.Errorf("argv = %q, want --sandbox %s", argv, tc.want)
			}
		})
	}
}

// The swap must not reach back into a command slice another config shares.
func TestWhenSandboxedDoesNotMutateTheSharedCommand(t *testing.T) {
	a := codexLike()
	c := &Config{
		Target:  Target{Mode: ModeDirectory, Path: "/repo"},
		Sandbox: Sandbox{Command: []string{"wrap", "--"}},
		Loop:    Loop{AgentsSandboxed: true},
		Agents:  map[string]Agent{"cx": a},
	}
	c.applySandbox()
	if a.Command[2] != "--sandbox read-only" {
		t.Errorf("the original command was modified: %v", a.Command)
	}
}

func TestWhenSandboxedRefusesAKeyThatMatchesNothing(t *testing.T) {
	a := codexLike()
	a.WhenSandboxed.Replace = map[string]string{"--sandbox readonly": "--sandbox danger-full-access"}
	err := a.WhenSandboxed.validate("cx", a.Command)
	if err == nil || !strings.Contains(err.Error(), "matches no element") {
		t.Fatalf("err = %v, want a refusal of the stale key", err)
	}
	a.WhenSandboxed.Replace = map[string]string{"--sandbox read-only": " "}
	if err := a.WhenSandboxed.validate("cx", a.Command); err == nil || !strings.Contains(err.Error(), "empty replacement") {
		t.Fatalf("err = %v, want a refusal of the empty replacement", err)
	}
	// Already applied: matched through its replacement.
	applied := []string{"codex", "--sandbox danger-full-access"}
	if err := codexLike().WhenSandboxed.validate("cx", applied); err != nil {
		t.Errorf("an applied swap was refused: %v", err)
	}
}

// fakeBuiltin points the builtin search at a directory holding a wrapper file.
func fakeBuiltin(t *testing.T, dir string, create bool) string {
	t.Helper()
	p := filepath.Join(dir, "sandbox", "fixpoint-sandbox")
	if create {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := builtinSandboxDirs
	builtinSandboxDirs = func() []string { return []string{dir} }
	t.Cleanup(func() { builtinSandboxDirs = old })
	return p
}

func TestSandboxBuiltinUsesTheShippedWrapperAndAsserts(t *testing.T) {
	p := fakeBuiltin(t, t.TempDir(), true)
	c := &Config{
		Target:  Target{Mode: ModeDirectory, Path: "/repo"},
		Sandbox: Sandbox{Command: []string{"bundle-wrapper", "--"}},
		Agents:  map[string]Agent{"cx": codexLike()},
	}
	if err := (Overrides{Sandbox: SandboxBuiltin}).applySandboxChoice(c); err != nil {
		t.Fatal(err)
	}
	want := []string{"/bin/sh", p, "{{target}}", "{{target_mode}}", "--"}
	if strings.Join(c.Sandbox.Command, "|") != strings.Join(want, "|") {
		t.Errorf("sandbox.command = %v, want %v (the bundle's wrapper REPLACED)", c.Sandbox.Command, want)
	}
	if !c.Sandbox.Builtin || !c.Loop.AgentsSandboxed {
		t.Errorf("builtin=%v agents_sandboxed=%v, want both set", c.Sandbox.Builtin, c.Loop.AgentsSandboxed)
	}
	c.applySandbox()
	if argv := strings.Join(c.Agents["cx"].Argv(), " "); !strings.Contains(argv, "danger-full-access") {
		t.Errorf("argv = %q: builtin did not switch codex's own sandbox off", argv)
	}
}

func TestSandboxBuiltinMissingIsAStartupError(t *testing.T) {
	fakeBuiltin(t, t.TempDir(), false)
	c := &Config{Target: Target{Mode: ModeDirectory, Path: "/repo"}}
	err := (Overrides{Sandbox: SandboxBuiltin}).applySandboxChoice(c)
	if err == nil || !strings.Contains(err.Error(), "not found beside this binary") {
		t.Fatalf("err = %v, want a refusal naming where it looked", err)
	}
	if c.Sandbox.Enabled() {
		t.Error("a missing wrapper left a sandbox configured")
	}
}

func TestSandboxBuiltinRefusesAPathWithASpace(t *testing.T) {
	fakeBuiltin(t, filepath.Join(t.TempDir(), "with space"), true)
	err := (Overrides{Sandbox: SandboxBuiltin}).applySandboxChoice(&Config{})
	if err == nil || !strings.Contains(err.Error(), "whitespace") {
		t.Fatalf("err = %v, want a refusal of the spaced path", err)
	}
}

// off overrules the environment; it never removes what a bundle configured.
func TestSandboxOffLeavesTheBundlesWrapper(t *testing.T) {
	c := &Config{Sandbox: Sandbox{Command: []string{"bundle-wrapper", "--"}}}
	if err := (Overrides{Sandbox: SandboxOff}).applySandboxChoice(c); err != nil {
		t.Fatal(err)
	}
	if !c.Sandbox.Enabled() {
		t.Error("-sandbox off removed the bundle's sandbox.command")
	}
}

func TestSandboxChoiceRefusesAnUnknownValue(t *testing.T) {
	err := (Overrides{Sandbox: "on"}).applySandboxChoice(&Config{})
	if err == nil || !strings.Contains(err.Error(), SandboxEnv) {
		t.Fatalf("err = %v, want a refusal that names the variable too", err)
	}
}

// The pr-mode refusal of -agents-sandboxed stands for a wrapper fixpoint did not
// ship, and is lifted for its own.
func TestPRModeAcceptsOnlyTheBuiltinWrapperAsConfining(t *testing.T) {
	for _, builtin := range []bool{false, true} {
		c := &Config{
			Target:  Target{Mode: ModePR, Path: "/repo"},
			Sandbox: Sandbox{Command: []string{"/bin/sh", "--"}, Builtin: builtin},
			Loop:    Loop{AgentsSandboxed: true},
			Agents:  map[string]Agent{"rev": {Command: []string{"claude"}}},
		}
		c.applySandbox()
		err := c.validateSandbox()
		if builtin && err != nil {
			t.Errorf("builtin wrapper refused in pr mode: %v", err)
		}
		if !builtin && (err == nil || !strings.Contains(err.Error(), "mode pr")) {
			t.Errorf("err = %v, want the pr-mode refusal for a wrapper fixpoint did not ship", err)
		}
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

// A wrapper token whose placeholder is empty must drop WHOLE, the way an agent
// command token does. Dropping only the value leaves a dangling flag that
// swallows the next argument -- in a wrapper, the next mount or the agent command
// itself (review run 20260916-085129, finding i13).
func TestSandboxDropsAWholeTokenWhenItsPlaceholderIsEmpty(t *testing.T) {
	c := &Config{
		// No target.path, so {{target}} expands to empty.
		Target:  Target{Mode: ModeDirectory},
		Sandbox: Sandbox{Command: []string{"/bin/sh", "--bind {{target}}", "--keep", "--"}},
		Agents:  map[string]Agent{"rev": {Command: []string{"claude"}}},
	}
	c.applySandbox()
	got := c.Agents["rev"].Sandbox
	for _, tok := range got {
		if tok == "--bind" {
			t.Fatalf("a dangling --bind survived its empty value: %v", got)
		}
	}
	if len(got) != 3 || got[1] != "--keep" {
		t.Errorf("wrapper = %v, want the --bind token dropped whole and the rest intact", got)
	}
}

// sandboxedConfig is validConfig with a wrapper in front of every agent and a
// reviewer whose own argv switches the CLI's sandbox off, the way a codex bench
// candidate has to run inside a wrapper (sandboxes do not nest).
func sandboxedConfig(t *testing.T) *Config {
	t.Helper()
	c := validConfig(t)
	rev := c.Agents["rev"]
	rev.Command = []string{"echo", "exec", "--sandbox", "danger-full-access"}
	c.Agents["rev"] = rev
	c.Sandbox = Sandbox{Command: []string{"echo", "{{target}}", "{{target_mode}}", "--"}}
	c.applySandbox()
	return c
}

// Without the flag nothing changes: a wrapper is not, by itself, a reason to
// believe a reviewer whose argv turns its own sandbox off is still read-only.
func TestAgentsSandboxedIsRequiredToLiftTheBypassRefusal(t *testing.T) {
	c := sandboxedConfig(t)
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "lets the agent write files") {
		t.Fatalf("a read-only reviewer disabling its own sandbox was accepted without -agents-sandboxed: %v", err)
	}
}

func TestAgentsSandboxedWithAWrapperLiftsOnlyThatRefusal(t *testing.T) {
	c := sandboxedConfig(t)
	c.Loop.AgentsSandboxed = true
	if err := c.Validate(); err != nil {
		t.Fatalf("flag + wrapper still refused: %v", err)
	}
	// The refusal is lifted for the same argv in every spelling it guards.
	for _, spelled := range [][]string{
		{"echo", "exec", "--dangerously-bypass-approvals-and-sandbox"},
		{"echo", "exec", "-c", "sandbox_mode=danger-full-access"},
		{"echo", "exec", "--yolo"},
	} {
		c := sandboxedConfig(t)
		c.Loop.AgentsSandboxed = true
		a := c.Agents["rev"]
		a.Command = spelled
		c.Agents["rev"] = a
		c.applySandbox()
		if err := c.Validate(); err != nil {
			t.Errorf("%v: refused although the wrapper is asserted: %v", spelled, err)
		}
	}
}

// Nothing else is relaxed: the flag lifts the one refusal and every other check
// on the composed argv still runs.
func TestAgentsSandboxedRelaxesNothingElse(t *testing.T) {
	t.Run("a missing agent binary", func(t *testing.T) {
		c := sandboxedConfig(t)
		c.Loop.AgentsSandboxed = true
		a := c.Agents["rev"]
		a.Command = []string{"no-such-binary-for-fixpoint-test", "--sandbox", "danger-full-access"}
		c.Agents["rev"] = a
		c.applySandbox()
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "not found on PATH") {
			t.Fatalf("err = %v, want the missing binary refused", err)
		}
	})
	t.Run("a reviewer that can edit", func(t *testing.T) {
		c := sandboxedConfig(t)
		c.Loop.AgentsSandboxed = true
		a := c.Agents["rev"]
		a.CanEdit = true
		c.Agents["rev"] = a
		c.applySandbox()
		if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "must be read-only") {
			t.Fatalf("err = %v, want a can_edit reviewer refused", err)
		}
	})
}

// The flag claims a sandbox: with none configured there is nothing behind the
// claim, and it must be an error rather than a silently lifted check.
func TestAgentsSandboxedWithoutAWrapperIsAnError(t *testing.T) {
	c := sandboxedConfig(t)
	c.Loop.AgentsSandboxed = true
	c.Sandbox = Sandbox{}
	for name, a := range c.Agents {
		a.Sandbox = nil
		c.Agents[name] = a
	}
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "sets no sandbox.command") {
		t.Fatalf("err = %v, want the flag refused for claiming a sandbox the config does not have", err)
	}
}

// The target of a pr run is untrusted content and fixpoint cannot verify that the
// wrapper honors {{target_mode}}, so the barrier it can verify stays.
func TestAgentsSandboxedIsRefusedInPRMode(t *testing.T) {
	c := sandboxedConfig(t)
	c.Loop.AgentsSandboxed = true
	c.Target.Mode = ModePR
	c.Target.PR = 1
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "refused in mode pr") {
		t.Fatalf("err = %v, want -agents-sandboxed refused in pr mode", err)
	}
}

// An agent that can edit is the coder: its command was never subject to the
// bypass refusal, and the flag does not change that either way.
func TestAgentsSandboxedDoesNotChangeAnEditingAgent(t *testing.T) {
	for _, flag := range []bool{false, true} {
		c := sandboxedConfig(t)
		// The wrapper is in front of the reviewer only to keep this test about the
		// coder: the reviewer goes back to an ordinary read-only command.
		rev := c.Agents["rev"]
		rev.Command = []string{"echo"}
		c.Agents["rev"] = rev
		coder := c.Agents["coder"]
		coder.Command = []string{"echo", "exec", "--dangerously-bypass-approvals-and-sandbox"}
		c.Agents["coder"] = coder
		c.applySandbox()
		c.Loop.AgentsSandboxed = flag
		if err := c.Validate(); err != nil {
			t.Errorf("flag=%v: the coder's bypass flag was refused: %v", flag, err)
		}
		// And the coder still gets the wrapper in rw mode, the reviewer in ro.
		if got := c.Agents["coder"].Argv()[2]; got != "rw" {
			t.Errorf("flag=%v: coder wrapper mode = %q, want rw", flag, got)
		}
		if got := c.Agents["rev"].Argv()[2]; got != "ro" {
			t.Errorf("flag=%v: reviewer wrapper mode = %q, want ro", flag, got)
		}
	}
}
