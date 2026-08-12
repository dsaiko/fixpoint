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
		// Generated per-run by bench/run.sh and gitignored: present whenever a
		// benchmark is running, but never shipped, so it is not this test's to
		// police (a copied baseline would double-count a real agent, and an
		// ollama candidate would demand a budget row per swept model).
		if e.Name() == "bench-candidate"+configExt {
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

// Every shipped agent must carry a prompt_budget, and carry the one its own
// comment justifies.
//
// 0 is the documented "no limit" default, so dropping or emptying the key is
// neither a parse error nor a validation error: it silently restores the
// behavior the budget exists to prevent -- the agent runs a full round of wall
// clock and returns "Prompt is too long", which looks like a crashed agent in the
// summary rather than a prompt fixpoint should never have built. Pinning the
// values (rather than only asserting > 0) makes a widening visible in the diff
// that does it.
func TestShippedAgentsCarryAPromptBudget(t *testing.T) {
	const (
		// In-house CLIs, run against a first-party endpoint: 900 kB against a
		// measured 434 kB maximum on this project. See claude.yaml.
		local = 900_000
		// Agents served over someone else's HTTP endpoint, where a huge prompt is
		// also someone else's bill and rate limit. See kimi-ollama.yaml.
		remote = 400_000
		// prompt_via: arg cannot deliver more than Linux's 128 KiB per-argument
		// limit in the first place. See agy.yaml.
		onArgv = 128_000
	)
	// Keyed by file name, so a new agent file has to be added here -- the same
	// guard the claude-backed count above provides.
	want := map[string]int{
		"claude":          local,
		"claude-coder":    local,
		"codex":           local,
		"agy":             onArgv,
		"deepseek-ollama": remote,
		"gemma4-ollama":   remote,
		"glm-ollama":      remote,
		"kimi-ollama":     remote,
		"glm-openrouter":  remote,
		"kimi-openrouter": remote,
		"qwen-openrouter": remote,
	}
	agents := shippedAgents(t)
	for name, a := range agents {
		w, ok := want[name]
		if !ok {
			t.Errorf("agents/%s%s: no expected prompt_budget for this agent -- add it to this test, and give the file a prompt_budget if it has none", name, configExt)
			continue
		}
		if a.PromptBudget != w {
			t.Errorf("agents/%s%s: prompt_budget is %d, want %d", name, configExt, a.PromptBudget, w)
		}
		if a.PromptBudget <= 0 {
			t.Errorf("agents/%s%s: prompt_budget is %d, which means no limit -- the runaway guard is off for this agent", name, configExt, a.PromptBudget)
		}
	}
	for name := range want {
		if _, ok := agents[name]; !ok {
			t.Errorf("agents/%s%s: expected by this test but not in the bundle -- drop it here if the agent was removed", name, configExt)
		}
	}
}
