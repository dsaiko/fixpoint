package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bundle writes a minimal, valid bundle into dir and returns dir.
func bundle(t *testing.T, dir string, configs map[string]string, prompts, agents []string) string {
	t.Helper()
	for _, sub := range []string{promptsDir, agentsDir} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range configs {
		if err := os.WriteFile(filepath.Join(dir, name+configExt), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range prompts {
		if err := os.WriteFile(filepath.Join(dir, promptsDir, p+promptExt), []byte("{{.OutputContract}}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range agents {
		body := "command: [true]\ncan_edit: true\n"
		if err := os.WriteFile(filepath.Join(dir, agentsDir, a+configExt), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// The project root anchors every relative path in a run, so it must be found by
// walking up -- otherwise the same command reviews a different subtree depending
// on which directory it was invoked from.
func TestProjectRootWalksUp(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "internal", "target", "sub")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	// ".git" as a FILE, the shape git uses inside a worktree or submodule. A
	// directory-only check would walk straight past the root of every worktree.
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /elsewhere\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, from := range []string{root, filepath.Join(root, "internal"), deep} {
		got, err := ProjectRoot(from)
		if err != nil {
			t.Fatal(err)
		}
		if got != root {
			t.Errorf("ProjectRoot(%s) = %s, want %s", from, got, root)
		}
	}
}

// The bundle directory name must NOT be a root marker: "config" is an extremely
// common package name (this repository has internal/config), and treating it as a
// marker detected internal/ as the project root -- so a run from internal/target
// reviewed only that subtree and could not find any config.
func TestProjectRootIgnoresNestedConfigDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "internal", "config")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ProjectRoot(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("ProjectRoot(%s) = %s, want the repository root %s; a nested config package must not look like a project root", nested, got, root)
	}
}

// A missing name must name every location searched: without that list a user
// cannot tell a typo from a missing bundle from a shadowing surprise.
func TestResolverNotFoundListsSearchPath(t *testing.T) {
	r := &Resolver{Bundles: []string{"/nowhere/a", "/nowhere/b"}}
	_, err := r.Config("ghost")
	if err == nil {
		t.Fatal("expected an error for a missing config")
	}
	for _, want := range []string{"ghost", "/nowhere/a", "/nowhere/b"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q:\n%s", want, err)
		}
	}
}

// Resolution is per file, so a project can shadow one prompt and inherit the
// rest. This is the whole point of a search path -- and the reason every resolved
// path is logged.
func TestResolverShadowsPerFile(t *testing.T) {
	high := bundle(t, filepath.Join(t.TempDir(), "project"), nil, []string{"review-bugs"}, nil)
	low := bundle(t, filepath.Join(t.TempDir(), "system"), nil, []string{"review-bugs", "fix"}, nil)
	r := &Resolver{Bundles: []string{high, low}}

	got, err := r.Prompt("review-bugs")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(high, promptsDir, "review-bugs.md"); got != want {
		t.Errorf("Prompt(review-bugs) = %s, want the shadowing copy %s", got, want)
	}
	// Not shadowed: falls through to the lower bundle.
	got, err = r.Prompt("fix")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(low, promptsDir, "fix.md"); got != want {
		t.Errorf("Prompt(fix) = %s, want the inherited %s", got, want)
	}
}

// ListConfigs backs both --list and shell completion, so it must report the file
// a run would actually use, not every copy on the path.
func TestListConfigsOmitsShadowedCopies(t *testing.T) {
	high := bundle(t, filepath.Join(t.TempDir(), "project"), map[string]string{"full": "target: {mode: directory}\n"}, nil, nil)
	low := bundle(t, filepath.Join(t.TempDir(), "system"), map[string]string{
		"full": "target: {mode: directory}\n",
		"pr":   "target: {mode: pr, pr: 1}\n",
	}, nil, nil)
	r := &Resolver{Bundles: []string{high, low, "/nonexistent"}}
	got, err := r.ListConfigs()
	if err != nil {
		t.Fatalf("a missing bundle on the path must not be an error: %v", err)
	}
	if want := filepath.Join(high, "full.yaml"); got["full"] != want {
		t.Errorf("full resolved to %s, want the shadowing %s", got["full"], want)
	}
	if want := filepath.Join(low, "pr.yaml"); got["pr"] != want {
		t.Errorf("pr resolved to %s, want %s", got["pr"], want)
	}
	if len(got) != 2 {
		t.Errorf("got %d configs, want 2 (shadowed copies must not be listed twice)", len(got))
	}
}

const taskBody = `roles:
  coder: {agent: mock, prompt: fix}
  review:
    strategy: fixed
    prompts:
      - {agent: mock, prompt: review-bugs}
`

// extends must override per key and leave untouched keys inherited, and a list the
// child sets must REPLACE the inherited one -- appending would make an inherited
// entry impossible to remove.
func TestLoadBundleExtends(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"defaults": "target:\n  mode: directory\n  exclude: [\"**/vendor/**\", \"**/testdata/**\"]\nloop:\n  max_iterations: 9\n  clean_rounds_to_stop: 3\n",
		"task":     "extends: defaults\nloop:\n  max_iterations: 2\ntarget:\n  exclude: [\"only/**\"]\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})

	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root)
	if err != nil {
		t.Fatal(err)
	}
	cfg := l.Config
	if cfg.Loop.MaxIterations != 2 {
		t.Errorf("MaxIterations = %d, want the child's 2", cfg.Loop.MaxIterations)
	}
	if cfg.Loop.CleanRoundsToStop != 3 {
		t.Errorf("CleanRoundsToStop = %d, want the inherited 3", cfg.Loop.CleanRoundsToStop)
	}
	if cfg.Target.Mode != ModeDirectory {
		t.Errorf("Mode = %q, want the inherited directory", cfg.Target.Mode)
	}
	if len(cfg.Target.Exclude) != 1 || cfg.Target.Exclude[0] != "only/**" {
		t.Errorf("Exclude = %v, want the child's list to replace the base's", cfg.Target.Exclude)
	}
	if l.Source.Extends == "" {
		t.Error("Source.Extends must record the base file for provenance")
	}
}

// Inheritance is one level deep: a chain would mean the effective value of a
// field requires reading N files, defeating the point of naming the base.
func TestLoadBundleRejectsExtendsChain(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"a": "extends: b\n" + taskBody,
		"b": "extends: c\n",
		"c": "target: {mode: directory}\n",
	}, []string{"fix", "review-bugs"}, []string{"mock"})
	_, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "a", root)
	if err == nil || !strings.Contains(err.Error(), "one level deep") {
		t.Fatalf("expected a one-level-deep error, got %v", err)
	}
}

// Every relative path resolves against the project root, so a run from a
// subdirectory reviews the whole project and writes artifacts at the top -- not
// into whatever directory it happened to start in.
func TestLoadBundleAnchorsPathsToProjectRoot(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"task": "target:\n  mode: directory\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})

	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root)
	if err != nil {
		t.Fatal(err)
	}
	if l.Config.Target.Path != root {
		t.Errorf("Target.Path = %s, want the project root %s", l.Config.Target.Path, root)
	}
	// The logs default is applied before anchoring; anchoring after is what keeps
	// the artifact directory at the project root rather than the working directory.
	if !strings.HasPrefix(l.Config.Logs.Dir, root) {
		t.Errorf("Logs.Dir = %s, want it anchored under %s", l.Config.Logs.Dir, root)
	}
}

// Agents load from their own files so a task config says which agents it uses
// without restating how to invoke them.
func TestLoadBundleResolvesAgentsFromFiles(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"task": "target: {mode: directory}\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})

	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := l.Config.Agents["mock"]
	if !ok {
		t.Fatal("agent mock was not loaded from agents/mock.yaml")
	}
	if a.PromptVia != PromptViaStdin {
		t.Errorf("PromptVia = %q, want the default %q applied to file-loaded agents too", a.PromptVia, PromptViaStdin)
	}
	if a.Timeout.Std() == 0 {
		t.Error("Timeout default was not applied to a file-loaded agent")
	}
	if l.Source.Agents["mock"] == "" {
		t.Error("Source.Agents must record where each agent resolved from")
	}
}

// A name that looks like a path is used directly, so an ad-hoc config outside any
// bundle still runs.
func TestResolverAcceptsPathLikeNames(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "adhoc.yaml")
	if err := os.WriteFile(p, []byte("target: {mode: directory}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &Resolver{Bundles: []string{"/nowhere"}}
	got, err := r.Config(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Errorf("Config(%s) = %s, want the path itself", p, got)
	}
}
