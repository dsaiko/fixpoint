package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// shippedAgents parses every agent file in the repository's own bundle. The
// files are data, not code, so nothing else in the test suite would notice a
// flag being dropped from one of them.
func shippedAgents(t *testing.T) map[string]Agent {
	t.Helper()
	dir := filepath.Join("..", "..", projectBundleDir, agentsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	agents := make(map[string]Agent)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != configExt {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		var a Agent
		if err := yaml.Unmarshal(b, &a); err != nil {
			t.Fatalf("parsing %s: %v", e.Name(), err)
		}
		agents[strings.TrimSuffix(e.Name(), configExt)] = a
	}
	if len(agents) == 0 {
		t.Fatalf("no agent files found under %s", dir)
	}
	return agents
}

// Every shipped agent that drives the claude CLI must pass
// `--setting-sources user` -- the coder included.
//
// claude runs with its working directory inside the target and by default also
// loads that directory's .claude/settings.json and .claude/settings.local.json.
// A SessionStart hook there is a shell command the CLI executes with the
// operator's full permissions BEFORE the model takes a turn, so it is not
// covered by the reviewer's missing permission-skip flag: a read-only reviewer
// pointed at a hostile PR would be arbitrary execution with its own credential
// in the environment. Review-only mode asserts no trust in the target, so this
// is the untrusted path fixpoint is meant for, not an exotic one.
//
// The coder is not exempt even though it auto-approves every tool request. A
// hook runs with no model in the loop and no tool request to approve, and the
// assertion a fix round requires does not cover it: in pr mode that is only
// -allow-untrusted-fix, which documents prompt-injection risk from an untrusted
// author, while `gh pr checkout` puts that author's settings file inside the
// worktree the coder runs in.
func TestShippedAgentsDoNotLoadTargetSettings(t *testing.T) {
	checked := 0
	for name, a := range shippedAgents(t) {
		argv := a.Argv()
		// Matches both spellings the bundle uses: `claude ...` directly, and the
		// ollama wrappers' `ollama launch claude -- ...`.
		if !slices.Contains(argv, "claude") {
			continue
		}
		checked++
		i := slices.Index(argv, "--setting-sources")
		if i < 0 || i+1 >= len(argv) {
			t.Errorf("agents/%s%s: claude agent without --setting-sources; it would load hooks, MCP servers and instructions from the target's .claude/settings.json", name, configExt)
			continue
		}
		for _, src := range strings.Split(argv[i+1], ",") {
			if src != "user" {
				t.Errorf("agents/%s%s: --setting-sources includes %q; only \"user\" is safe against a target-supplied settings file", name, configExt, src)
			}
		}
	}
	// A rename or an extension change must not turn this into a test that
	// silently inspects nothing.
	//
	// 9: claude, claude-coder, four ollama-routed reviewers (kimi, deepseek, glm,
	// gemma4) and three OpenRouter-routed ones (kimi, glm, qwen). The same model
	// appearing under two routes is deliberate -- the route changes caching and
	// reasoning behavior that no other field records -- and both routes drive the
	// claude CLI, so both must carry --setting-sources.
	if want := 9; checked != want {
		t.Errorf("checked %d claude-backed agents, want %d -- update this test if the bundle gained or lost one", checked, want)
	}
}
