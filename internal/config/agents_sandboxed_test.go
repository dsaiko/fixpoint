package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// -agents-sandboxed reaches the compiled configuration, and loading it into a
// config with no sandbox.command is refused by Validate: the flag claims a
// sandbox the file does not have.
func TestLoadBundleCarriesAgentsSandboxed(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"task": "target: {mode: directory}\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})
	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{AgentsSandboxed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !l.Config.Loop.AgentsSandboxed {
		t.Fatal("Overrides.AgentsSandboxed did not reach Loop.AgentsSandboxed")
	}
	if err := l.Validate(); err == nil || !strings.Contains(err.Error(), "sets no sandbox.command") {
		t.Errorf("Validate() = %v, want the flag refused for a config with no sandbox.command", err)
	}
	l, err = LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if l.Config.Loop.AgentsSandboxed {
		t.Error("Loop.AgentsSandboxed is set without the flag")
	}
}
