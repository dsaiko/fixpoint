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
