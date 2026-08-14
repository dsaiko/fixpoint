package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ProjectSuppliedPolicy is the control that decides whether fixpoint acts on
// configuration authored by the code under review: argv it executes (agent commands
// and verify commands) and the instructions it hands agents (prompts). It had no
// test at all, and both of the ways it silently failed open -- an `agents:` map
// written INLINE in the task config, and a config that narrows target.path below
// its own bundle -- were live bypasses reaching arbitrary command execution with no
// flag and no model involvement. So every way a run can be built from a file inside
// the project is enumerated here, and each must be reported.
func TestProjectSuppliedPolicy(t *testing.T) {
	// inlineAgent defines the coder/reviewer agent in the task config itself, with no
	// agents/<name>.yaml anywhere -- the shape resolveAgents never records in
	// Source.Agents, and so the shape a content-based gate cannot see.
	const inlineAgent = "agents:\n  mock:\n    command: [true]\n    can_edit: true\n"
	const verifyCmds = "verify:\n  policy: must_pass\n  commands:\n    - {name: build, run: [true]}\n"

	cases := []struct {
		name string
		// setup writes the bundles and returns the resolver search path, the project
		// root, and the config name to load.
		setup func(t *testing.T) (bundles []string, root, cfgName string)
		// wantAny is a substring every acceptable report must contain; empty means the
		// gate must report NOTHING.
		wantAny string
	}{
		{
			name: "agent file inside the project",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target: {mode: directory}\n" + taskBody,
				}, []string{"fix", "review-bugs"}, []string{"mock"})
				return []string{dir}, root, "task"
			},
			wantAny: "agent mock",
		},
		{
			name: "agents defined inline in a project-resolved config",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target: {mode: directory}\n" + inlineAgent + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{dir}, root, "task"
			},
			wantAny: filepath.Join(projectBundleDir, "task"+configExt),
		},
		{
			name: "verify commands in a project-resolved config",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target: {mode: directory}\n" + verifyCmds + inlineAgent + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{dir}, root, "task"
			},
			wantAny: filepath.Join(projectBundleDir, "task"+configExt),
		},
		{
			name: "verify commands in the extends base",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				outside := t.TempDir()
				bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"base": "target: {mode: directory}\n" + verifyCmds,
				}, nil, nil)
				out := bundle(t, outside, map[string]string{
					"task": "extends: base\n" + inlineAgent + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				// The operator's own bundle first, so only the inherited base comes from
				// the project: the base is the file that carries the commands.
				return []string{out, filepath.Join(root, projectBundleDir)}, root, "task"
			},
			wantAny: "extends base",
		},
		{
			// The attack that made target.path the wrong boundary: the hostile config
			// declares a narrow target, which put its own sibling bundle files outside
			// "the target" and so past the gate entirely.
			name: "config narrowing target.path below its own bundle",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
					t.Fatal(err)
				}
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target:\n  mode: directory\n  path: ./src\n" + taskBody,
				}, []string{"fix", "review-bugs"}, []string{"mock"})
				return []string{dir}, root, "task"
			},
			wantAny: "agent mock",
		},
		{
			// A prompt is not executed, but it IS the instruction stream handed verbatim
			// to an agent that can read every file the invoking user can. Shipping
			// prompts/review-bugs.md is how a repository writes the reviewer's orders.
			name: "prompt shadowed inside the project, config from outside",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				outside := t.TempDir()
				bundle(t, filepath.Join(root, projectBundleDir), nil, []string{"review-bugs"}, nil)
				out := bundle(t, outside, map[string]string{
					"task": "target: {mode: directory}\n" + inlineAgent + taskBody,
				}, []string{"fix"}, nil)
				return []string{out, filepath.Join(root, projectBundleDir)}, root, "task"
			},
			wantAny: "prompt review-bugs",
		},
		{
			name: "whole bundle outside the project reports nothing",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				out := bundle(t, t.TempDir(), map[string]string{
					"task": "target: {mode: directory}\n" + verifyCmds + inlineAgent + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{out}, root, "task"
			},
			wantAny: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bundles, root, name := tc.setup(t)
			l, err := LoadBundle(&Resolver{Bundles: bundles}, name, root, Overrides{})
			if err != nil {
				t.Fatal(err)
			}
			got := l.ProjectSuppliedPolicy()
			if tc.wantAny == "" {
				if len(got) != 0 {
					t.Fatalf("ProjectSuppliedPolicy() = %v, want nothing: no file came from the project", got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatalf("ProjectSuppliedPolicy() reported nothing; the run executes or is steered by a file inside %s", root)
			}
			found := false
			for _, s := range got {
				if strings.Contains(s, tc.wantAny) {
					found = true
				}
			}
			if !found {
				t.Errorf("ProjectSuppliedPolicy() = %v, want an entry containing %q", got, tc.wantAny)
			}
		})
	}
}

// The gate reads ProjectRoot, so a run with no project root (nothing to be inside
// of) must report nothing rather than measuring containment against "".
func TestProjectSuppliedPolicyWithoutProjectRoot(t *testing.T) {
	dir := bundle(t, t.TempDir(), map[string]string{
		"task": "target: {mode: directory}\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})
	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", "", Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if got := l.ProjectSuppliedPolicy(); len(got) != 0 {
		t.Errorf("ProjectSuppliedPolicy() = %v, want nothing when there is no project root", got)
	}
}

// env.inherit_all hands an agent every exported secret, including the ones no
// denylist or redactor knows by name -- so a file inside the target must not be
// able to declare it, and no flag may grant it. -trusted-target says the target's
// policy may be executed; it does not say the target may help itself to secrets it
// cannot enumerate. Every shape a target can ship the key in is refused here, and
// the operator's own bundle must still be able to use it.
func TestProjectSuppliedInheritAll(t *testing.T) {
	const inheritAgent = "command: [true]\ncan_edit: true\nenv:\n  inherit_all: true\n"
	const inlineInherit = "agents:\n  mock:\n    command: [true]\n    can_edit: true\n    env: {inherit_all: true}\n"

	// agentFile writes an agents/<name>.yaml with a body of its own, which the shared
	// bundle helper cannot: the whole point here is the env block.
	agentFile := func(t *testing.T, dir, name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, agentsDir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, agentsDir, name+configExt), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name  string
		setup func(t *testing.T) (bundles []string, root, cfgName string)
		// wantRefuse: loading must fail. wantInheritAll: loading must succeed AND the
		// effective agent must still carry inherit_all -- an allowed case that quietly
		// lost the key would pass the refusal check for the wrong reason.
		wantRefuse     bool
		wantInheritAll bool
	}{
		{
			name: "agent file inside the project",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target: {mode: directory}\n" + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				agentFile(t, dir, "mock", inheritAgent)
				return []string{dir}, root, "task"
			},
			wantRefuse: true,
		},
		{
			// No agents/<name>.yaml at all: the declaration rides in the task config's
			// own `agents:` map, which Source.Agents never records.
			name: "agent defined inline in a project-resolved config",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target: {mode: directory}\n" + inlineInherit + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{dir}, root, "task"
			},
			wantRefuse: true,
		},
		{
			// The config comes from the operator's bundle; only the inherited base is
			// the target's, and it is the file carrying the inline agent.
			name: "inline agent in a project-resolved extends base",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"base": "target: {mode: directory}\n" + inlineInherit,
				}, nil, nil)
				out := bundle(t, t.TempDir(), map[string]string{
					"task": "extends: base\n" + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{out, filepath.Join(root, projectBundleDir)}, root, "task"
			},
			wantRefuse: true,
		},
		{
			// The bypass a target.path-only boundary would allow: a narrow target puts
			// the config's own sibling agent file outside "the target".
			name: "config narrowing target.path below its own bundle",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
					t.Fatal(err)
				}
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target:\n  mode: directory\n  path: ./src\n" + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				agentFile(t, dir, "mock", inheritAgent)
				return []string{dir}, root, "task"
			},
			wantRefuse: true,
		},
		{
			// The other direction of the case above, and the one the escape hatch
			// depends on: the operator's own config declares the inline agent and the
			// project base does not mention agents at all. The base being inside the
			// target is not by itself a reason to refuse a key the operator set.
			name: "operator config outside the project may declare it inline over a project base",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"base": "target: {mode: directory}\n",
				}, nil, nil)
				out := bundle(t, t.TempDir(), map[string]string{
					"task": "extends: base\n" + inlineInherit + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{out, filepath.Join(root, projectBundleDir)}, root, "task"
			},
			wantInheritAll: true,
		},
		{
			// The same, with the project base declaring the agent WITHOUT inherit_all:
			// the child's entry replaces the base's whole entry, command included, so
			// nothing about the effective agent came from inside the target.
			name: "operator config outside the project may override a project base's agent inline",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"base": "target: {mode: directory}\nagents:\n  mock:\n    command: [false]\n    can_edit: true\n",
				}, nil, nil)
				out := bundle(t, t.TempDir(), map[string]string{
					"task": "extends: base\n" + inlineInherit + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{out, filepath.Join(root, projectBundleDir)}, root, "task"
			},
			wantInheritAll: true,
		},
		{
			// The direction that makes dropping the base from the candidate paths safe,
			// asserted from the other side: the project's base declares the agent WITH
			// inherit_all (and an env.pass of its own) and the operator's out-of-project
			// config re-declares it inline without either. The child's entry replaces the
			// base's whole entry, so nothing the base said survives and there is no
			// project-set inherit_all left for the gate to catch. Under a merging decoder
			// the key WOULD survive -- into an agent the gate no longer inspects the base
			// of -- so this is the case that fails if that behavior ever shifts.
			name: "an operator's inline agent erases a project base's inherit_all",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"base": "target: {mode: directory}\nagents:\n  mock:\n    command: [false]\n    can_edit: true\n" +
						"    env:\n      inherit_all: true\n      pass: [FIXPOINT_BASE_ONLY]\n",
				}, nil, nil)
				out := bundle(t, t.TempDir(), map[string]string{
					"task": "extends: base\nagents:\n  mock:\n    command: [true]\n    can_edit: true\n" + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{out, filepath.Join(root, projectBundleDir)}, root, "task"
			},
		},
		{
			name: "operator bundle outside the project may declare it",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				out := bundle(t, t.TempDir(), map[string]string{
					"task": "target: {mode: directory}\n" + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				agentFile(t, out, "mock", inheritAgent)
				return []string{out}, root, "task"
			},
			wantInheritAll: true,
		},
		{
			// The inline shape of the case above: the only file that could have declared
			// the agent is the operator's own config, and it is outside the project even
			// though fixpoint is being run from inside one.
			name: "operator bundle outside the project may declare it inline",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				out := bundle(t, t.TempDir(), map[string]string{
					"task": "target: {mode: directory}\n" + inlineInherit + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				t.Chdir(root) // where fixpoint is normally run from: inside the project
				return []string{out}, root, "task"
			},
			wantInheritAll: true,
		},
		{
			// The judge is the role referencedAgents omitted, and an agent that is not
			// in that set skips both resolution and this refusal -- so an inline judge
			// declared by a config inside the target could take fixpoint's entire
			// environment while reading the untrusted code and the findings about it.
			name: "a judge-only agent defined inline in a project-resolved config",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				const inlineJudge = "agents:\n  judge:\n    command: [true]\n    env: {inherit_all: true}\n"
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target: {mode: directory}\n" + inlineJudge + taskBody + "  judge: {agent: judge, prompt: judge}\n",
				}, []string{"fix", "review-bugs", "judge"}, []string{"mock"})
				return []string{dir}, root, "task"
			},
			wantRefuse: true,
		},
		{
			name: "a project-supplied agent without inherit_all is untouched",
			setup: func(t *testing.T) ([]string, string, string) {
				t.Helper()
				root := t.TempDir()
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target: {mode: directory}\n" + taskBody,
				}, []string{"fix", "review-bugs"}, []string{"mock"})
				return []string{dir}, root, "task"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bundles, root, name := tc.setup(t)
			// -trusted-target asserted throughout: the refusal must not be reachable
			// only on the path where the weaker gate already stops the run.
			l, err := LoadBundle(&Resolver{Bundles: bundles}, name, root, Overrides{TrustedTarget: true})
			if !tc.wantRefuse {
				if err != nil {
					t.Fatalf("LoadBundle() = %v, want the operator's own inherit_all declaration to load", err)
				}
				// Otherwise a case could pass by losing the key on the way in, which
				// would test nothing: the gate is only interesting for a run that really
				// does end up with inherit_all set.
				if got := l.Config.Agents["mock"].Env.InheritAll; got != tc.wantInheritAll {
					t.Errorf("agents.mock.env.inherit_all = %v, want %v", got, tc.wantInheritAll)
				}
				// Every allowed case defines mock with command [true]; a base that
				// defines it too uses [false]. So this also pins what makes dropping the
				// base from the candidate paths safe: the child's entry replaces the
				// base's WHOLE entry, command included, rather than merging into it.
				if got := l.Config.Agents["mock"].Command; len(got) != 1 || got[0] != "true" {
					t.Errorf("agents.mock.command = %v, want the declaring file's [true]", got)
				}
				// No allowed case's DECLARING file sets env.pass, so a non-empty one can
				// only have leaked out of a base whose entry the child replaced -- the
				// field the command check cannot see, since the child supplies a command
				// either way. This is what distinguishes replacement from a per-field
				// merge, and the gate stops reading the base on the strength of it.
				if got := l.Config.Agents["mock"].Env.Pass; len(got) != 0 {
					t.Errorf("agents.mock.env.pass = %v, want nothing: only a replaced base declared one", got)
				}
				return
			}
			if err == nil {
				t.Fatalf("LoadBundle() succeeded; the target declared env.inherit_all and would receive every exported secret")
			}
			if !strings.Contains(err.Error(), "inherit_all") || !strings.Contains(err.Error(), "env.pass") {
				t.Errorf("LoadBundle() = %v, want the refusal to name inherit_all and the env.pass alternative", err)
			}
			if l != nil {
				t.Errorf("LoadBundle() returned a configuration alongside the refusal: %v", l.Source)
			}
		})
	}
}

// Every candidate path a provenance check measures may be unset -- a config that
// inherits from nothing has no Source.Extends -- and filepath.Abs("") resolves to
// the working directory, which is normally inside the project being reviewed. An
// unset path measured like a real one would therefore report the operator's own
// files as target-supplied, so the guard is asserted directly rather than left to
// whichever caller happens to reach it.
func TestFromProjectIgnoresUnsetPath(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root) // where fixpoint is normally run from: inside the project
	l := &Loaded{Config: &Config{}, ProjectRoot: root}
	if l.fromProject("") {
		t.Error("fromProject(\"\") = true; an absent file is not a file the project supplied")
	}
	if !l.fromProject(filepath.Join(root, "config", "task.yaml")) {
		t.Error("fromProject() = false for a file inside the project root, want true")
	}
}

// A bundle ROOT is allowed to be a symlink -- ~/.fixpoint pointing into a dotfiles
// checkout is a supported setup, and the resolver deliberately keeps it working. So
// provenance has to be judged by where a file REALLY lives: link the user bundle at
// a directory in the reviewed repository and every resolved path stays lexically
// under ~/.fixpoint while the reviewed commit owns the contents, which a lexical
// test reads as operator-owned. Both gates that ride on it are asserted here: the
// -trusted-target listing, and the inherit_all refusal no flag can grant.
func TestProjectSuppliedPolicyThroughSymlinkedBundleRoot(t *testing.T) {
	root := realDir(t, t.TempDir())
	inside := filepath.Join(root, "vendor", "policy")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	bundle(t, inside, map[string]string{
		"task": "target: {mode: directory}\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})
	// The path an operator's own bundle is searched under, resolving into the project.
	link := filepath.Join(realDir(t, t.TempDir()), "."+appName)
	if err := os.Symlink(inside, link); err != nil {
		t.Fatal(err)
	}

	l, err := LoadBundle(&Resolver{Bundles: []string{link}}, "task", root, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if got := l.ProjectSuppliedPolicy(); len(got) == 0 {
		t.Fatalf("ProjectSuppliedPolicy() reported nothing; %s resolves into the reviewed project at %s, so the reviewed commit owns the agent command the preflight ping runs", link, inside)
	}

	// The same mismatch decides whether the reviewed code may claim the entire parent
	// environment, which -trusted-target explicitly does not grant.
	if err := os.WriteFile(filepath.Join(inside, agentsDir, "mock"+configExt),
		[]byte("command: [true]\ncan_edit: true\nenv:\n  inherit_all: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBundle(&Resolver{Bundles: []string{link}}, "task", root, Overrides{TrustedTarget: true}); err == nil {
		t.Fatalf("LoadBundle() succeeded; an agent file inside %s set env.inherit_all and would receive every exported secret", inside)
	}
}

// A bundle inside target.path but outside the project root is target-supplied too:
// the two boundaries are checked as a union, so neither one being the wrong tree
// lets a file through.
func TestProjectSuppliedPolicyCoversTargetOutsideProjectRoot(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	dir := bundle(t, filepath.Join(elsewhere, projectBundleDir), map[string]string{
		"task": "target:\n  mode: directory\n  path: " + elsewhere + "\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})
	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	got := l.ProjectSuppliedPolicy()
	if len(got) == 0 {
		t.Fatalf("ProjectSuppliedPolicy() reported nothing; the bundle lies inside the reviewed target %s", elsewhere)
	}
}

// TargetSuppliedPolicy is §7.3's narrower control: an implement run must not
// take the commands it EXECUTES from the design's own directory. Its only test
// lived in cmd/fixpoint and built a Loaded literal setting Source.Config alone,
// so of policyFrom's four legs only the config path was ever exercised for the
// design-scoped list -- while the refusal message names agents and prompts
// (review run 20260814-012440). These go through LoadBundle, so the
// Source.Agents/Source.Prompts population that feeds the list is covered too.
func TestTargetSuppliedPolicy(t *testing.T) {
	const inlineAgent = "agents:\n  mock:\n    command: [true]\n    can_edit: true\n"

	cases := []struct {
		name    string
		setup   func(t *testing.T) (bundles []string, root, target, cfgName string)
		wantAny string
	}{
		{
			// The case the cmd-level test never reached: the operator points -config
			// at their own bundle, and the DESIGN's repository shadows one agent file.
			name: "agent file inside the design, config outside it",
			setup: func(t *testing.T) ([]string, string, string, string) {
				t.Helper()
				design := t.TempDir()
				outside := t.TempDir()
				bundle(t, filepath.Join(design, projectBundleDir), nil, nil, []string{"mock"})
				out := bundle(t, outside, map[string]string{
					"task": "target: {mode: directory}\n" + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{out, filepath.Join(design, projectBundleDir)}, outside, design, "task"
			},
			wantAny: "agent mock",
		},
		{
			// A prompt inside the design is the coder's instruction stream, written
			// by the document being implemented.
			name: "prompt inside the design, config outside it",
			setup: func(t *testing.T) ([]string, string, string, string) {
				t.Helper()
				design := t.TempDir()
				outside := t.TempDir()
				bundle(t, filepath.Join(design, projectBundleDir), nil, []string{"review-bugs"}, nil)
				out := bundle(t, outside, map[string]string{
					"task": "target: {mode: directory}\n" + inlineAgent + taskBody,
				}, []string{"fix"}, nil)
				return []string{out, filepath.Join(design, projectBundleDir)}, outside, design, "task"
			},
			wantAny: "prompt review-bugs",
		},
		{
			name: "whole bundle outside the design reports nothing",
			setup: func(t *testing.T) ([]string, string, string, string) {
				t.Helper()
				design := t.TempDir()
				out := bundle(t, t.TempDir(), map[string]string{
					"task": "target: {mode: directory}\n" + inlineAgent + taskBody,
				}, []string{"fix", "review-bugs"}, nil)
				return []string{out}, t.TempDir(), design, "task"
			},
			wantAny: "",
		},
		{
			// Fixpoint's own bundle, resolved from the OPERATOR's checkout with the
			// design somewhere else entirely, is project-supplied but not
			// design-supplied: the two lists differ exactly here, which is why the
			// flags that clear them differ.
			name: "operator's own bundle, design elsewhere, reports nothing",
			setup: func(t *testing.T) ([]string, string, string, string) {
				t.Helper()
				root := t.TempDir()
				design := t.TempDir()
				dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
					"task": "target: {mode: directory}\n" + taskBody,
				}, []string{"fix", "review-bugs"}, []string{"mock"})
				return []string{dir}, root, design, "task"
			},
			wantAny: "",
		},
		{
			// THE CASE THIS GATE EXISTS FOR, and the one an earlier version of this
			// table asserted as intended behavior (review run 20260814-024946).
			// The operator cds into the repository holding the design and runs the
			// implement config. Bundle discovery is anchored on the project root, so
			// <repo>/config wins resolution -- while the design sits in a
			// subdirectory, which is the layout the README documents and fixpoint's
			// own docs/design/DESIGN.md uses. Scoped to the design's directory alone,
			// the gate saw an empty list and let the design's own verify.commands
			// through.
			name: "bundle at the design repository's root, design in a subdirectory",
			setup: func(t *testing.T) ([]string, string, string, string) {
				t.Helper()
				repo := t.TempDir()
				docs := filepath.Join(repo, "docs", "design")
				if err := os.MkdirAll(docs, 0o755); err != nil {
					t.Fatal(err)
				}
				dir := bundle(t, filepath.Join(repo, projectBundleDir), map[string]string{
					"task": "target: {mode: directory}\n" + taskBody,
				}, []string{"fix", "review-bugs"}, []string{"mock"})
				// Project root and design checkout are the same tree: the operator is
				// standing in it.
				return []string{dir}, repo, docs, "task"
			},
			wantAny: "agent mock",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bundles, root, target, name := tc.setup(t)
			l, err := LoadBundle(&Resolver{Bundles: bundles}, name, root, Overrides{})
			if err != nil {
				t.Fatal(err)
			}
			// The design is the target; the pipeline shape is what selects this gate.
			l.Config.Target.Path = target
			got := l.TargetSuppliedPolicy()
			if tc.wantAny == "" {
				if len(got) != 0 {
					t.Fatalf("TargetSuppliedPolicy() = %v, want nothing: no file came from the design", got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatalf("TargetSuppliedPolicy() reported nothing; the run is steered by a file inside %s", target)
			}
			found := false
			for _, s := range got {
				if strings.Contains(s, tc.wantAny) {
					found = true
				}
			}
			if !found {
				t.Errorf("TargetSuppliedPolicy() = %v, want an entry containing %q", got, tc.wantAny)
			}
		})
	}
}
