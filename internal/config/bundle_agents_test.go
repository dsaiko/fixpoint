package config

import (
	"maps"
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
	// 10: claude, claude-coder, five ollama-routed reviewers (minimax, deepseek,
	// kimi, glm, gemma4) and three OpenRouter-routed ones (kimi, glm, qwen). The
	// same model appearing under two routes is deliberate -- the route changes
	// caching and reasoning behavior that no other field records -- and both
	// routes drive the claude CLI, so both must carry --setting-sources. Agents
	// that lost their panel seat stay in the bundle: the seat is decided in
	// defaults.yaml, and a measured alternative is worth keeping ready.
	if want := 10; checked != want {
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
		"minimax-ollama":  remote,
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

// Every shipped agent's environment is pinned by name, so widening one is a
// deliberate edit to this table rather than a quiet change to a YAML file.
//
// env.pass IS the credential boundary: everything outside it -- the operator's
// GitHub token, cloud credentials, database passwords -- is absent from the
// process, so a prompt-injected reviewer cannot quote what it cannot see. The
// existing shipped-agent tests covered prompt budgets and the claude
// settings-sources flag and never touched this, so making minimax-ollama an
// active reviewer relied on an allowlist nothing asserted (review run
// 20260814-012440).
func TestShippedAgentsPinTheirEnvironment(t *testing.T) {
	// The ollama route needs only the server address: the wrapped claude harness
	// authenticates to ollama, not to Anthropic, so an ANTHROPIC_API_KEY here
	// would be a credential handed to a third party for no reason.
	ollama := []string{"OLLAMA_HOST"}
	// The OpenRouter agents drive the claude CLI at a different base URL, which
	// is the one legitimate use of env.set here: a value fixpoint chooses, not a
	// credential it forwards. ANTHROPIC_API_KEY is deliberately absent from all
	// three -- it would take precedence over the auth token and silently bill
	// Anthropic for a model served by someone else.
	const openRouter = "https://openrouter.ai/api"
	want := map[string]struct {
		pass []string
		set  map[string]string
	}{
		"claude":          {pass: []string{"ANTHROPIC_API_KEY"}},
		"claude-coder":    {pass: []string{"ANTHROPIC_API_KEY"}},
		"codex":           {pass: []string{"OPENAI_API_KEY", "CODEX_API_KEY"}},
		"agy":             {pass: []string{"GOOGLE_API_KEY", "GEMINI_API_KEY", "GOOGLE_APPLICATION_CREDENTIALS"}},
		"deepseek-ollama": {pass: ollama},
		"gemma4-ollama":   {pass: ollama},
		"glm-ollama":      {pass: ollama},
		"kimi-ollama":     {pass: ollama},
		"minimax-ollama":  {pass: ollama},
		"glm-openrouter":  {pass: []string{"ANTHROPIC_AUTH_TOKEN"}, set: map[string]string{"ANTHROPIC_BASE_URL": openRouter}},
		"kimi-openrouter": {pass: []string{"ANTHROPIC_AUTH_TOKEN"}, set: map[string]string{"ANTHROPIC_BASE_URL": openRouter}},
		"qwen-openrouter": {pass: []string{"ANTHROPIC_AUTH_TOKEN"}, set: map[string]string{"ANTHROPIC_BASE_URL": openRouter}},
	}
	for name, a := range shippedAgents(t) {
		w, ok := want[name]
		if !ok {
			t.Errorf("agents/%s%s: no expected environment for this agent -- add it here, and think about what it is allowed to see", name, configExt)
			continue
		}
		if a.Env.InheritAll {
			t.Errorf("agents/%s%s: env.inherit_all is set, which hands this agent fixpoint's whole environment", name, configExt)
		}
		if !maps.Equal(a.Env.Set, w.set) {
			t.Errorf("agents/%s%s: env.set = %v, want %v", name, configExt, a.Env.Set, w.set)
		}
		if !slices.Equal(a.Env.Pass, w.pass) {
			t.Errorf("agents/%s%s: env.pass = %v, want exactly %v", name, configExt, a.Env.Pass, w.pass)
		}
		// No agent may be handed a credential for a provider it does not use.
		for _, v := range a.Env.Pass {
			if v == "ANTHROPIC_API_KEY" && !strings.HasPrefix(name, "claude") {
				t.Errorf("agents/%s%s: passes ANTHROPIC_API_KEY to a non-Anthropic route", name, configExt)
			}
		}
	}
}
