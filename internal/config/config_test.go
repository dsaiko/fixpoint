package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestArgv(t *testing.T) {
	cases := []struct {
		name  string
		agent Agent
		want  []string
	}{
		{
			"placeholders substituted and split",
			Agent{Model: "fable", Effort: "high", Command: []string{"claude", "-p", "--model {{model}}", "--effort {{effort}}"}},
			[]string{"claude", "-p", "--model", "fable", "--effort", "high"},
		},
		{
			"empty effort drops the whole token",
			Agent{Model: "fable", Command: []string{"claude", "--model {{model}}", "--effort {{effort}}"}},
			[]string{"claude", "--model", "fable"},
		},
		{
			"empty model drops flag and value together",
			Agent{Command: []string{"claude", "--model {{model}}", "-p"}},
			[]string{"claude", "-p"},
		},
		{
			"no placeholders",
			Agent{Command: []string{"ollama", "launch", "claude"}},
			[]string{"ollama", "launch", "claude"},
		},
		{
			"key=value placeholder token",
			Agent{Effort: "high", Command: []string{"codex", "-c model_reasoning_effort={{effort}}"}},
			[]string{"codex", "-c", "model_reasoning_effort=high"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.agent.Argv(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Argv() = %v, want %v", got, tc.want)
			}
		})
	}
}

// writePrompt creates a readable prompt file and returns its path.
func writePrompt(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(p, []byte("review {{.Target}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// validConfig returns a configuration that passes Validate, using binaries
// guaranteed to be on PATH.
func validConfig(t *testing.T) *Config {
	t.Helper()
	p := writePrompt(t)
	return &Config{
		Target: Target{Mode: "directory", Path: "."},
		Roles: Roles{
			Coder: RoleRef{Agent: "coder", Prompt: p},
			Review: Review{
				Strategy: "rotate",
				Agents:   []string{"rev"},
				Prompts:  []ReviewLens{{Prompt: p}},
			},
		},
		Agents: map[string]Agent{
			"coder": {Command: []string{"echo"}, PromptVia: "stdin", CanEdit: true},
			"rev":   {Command: []string{"echo"}, PromptVia: "stdin"},
		},
		Logs: Logs{Formats: []string{"md", "json", "raw"}},
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr string // substring of the expected error; empty means the config is valid
	}{
		{"valid", func(*Config) {}, ""},

		// logs.dir is a template. Each rule below is silent artifact loss (or a
		// committed logs dir) if it is not rejected at startup.
		{"logs.dir default template accepted", func(c *Config) {
			c.Logs.Dir = "logs/{timestamp}/round-{round}"
		}, ""},
		{"logs.dir rounds above timestamp accepted", func(c *Config) {
			c.Logs.Dir = "logs/{timestamp}-run/r{round}"
		}, ""},
		{"logs.dir without round segment needs round in pattern", func(c *Config) {
			c.Logs.Dir = "logs/{timestamp}"
			c.Logs.Pattern = "{role}-{agent}-{prompt}.{ext}"
		}, "each round overwrites the previous"},
		{"logs.dir without round segment ok when pattern has round", func(c *Config) {
			c.Logs.Dir = "logs/{timestamp}"
			c.Logs.Pattern = "{role}-{agent}-{prompt}-{round}.{ext}"
		}, ""},
		{"logs.dir needs a literal prefix", func(c *Config) {
			c.Logs.Dir = "{timestamp}/round-{round}"
		}, "must begin with at least one literal path segment"},
		{"logs.dir run part needs timestamp", func(c *Config) {
			c.Logs.Dir = "logs/run/round-{round}"
		}, "must contain {timestamp}"},
		{"logs.dir timestamp after round does not count", func(c *Config) {
			c.Logs.Dir = "logs/round-{round}/{timestamp}"
		}, "must contain {timestamp}"},
		{"logs.dir rejects unknown placeholder", func(c *Config) {
			c.Logs.Dir = "logs/{tiemstamp}/round-{round}"
		}, "unknown placeholder {tiemstamp}"},
		{"unknown mode", func(c *Config) { c.Target.Mode = "svn" }, "unknown mode"},
		{"pr mode without number", func(c *Config) { c.Target.Mode = "pr" }, "PR number required"},
		{"pr mode with number", func(c *Config) { c.Target.Mode = "pr"; c.Target.PR = 7 }, ""},
		{"no review prompts", func(c *Config) { c.Roles.Review.Prompts = nil }, "at least one review lens"},
		{"unknown strategy", func(c *Config) { c.Roles.Review.Strategy = "random" }, "unknown strategy"},
		{"fixed strategy without pinned agent", func(c *Config) { c.Roles.Review.Strategy = "fixed" }, "requires every lens to pin"},
		{"fixed strategy fully pinned", func(c *Config) {
			c.Roles.Review.Strategy = "fixed"
			c.Roles.Review.Prompts[0].Agent = "rev"
		}, ""},
		{"fixed strategy ignores unused pool entry", func(c *Config) {
			c.Roles.Review.Strategy = "fixed"
			c.Roles.Review.Prompts[0].Agent = "rev"
			c.Roles.Review.Agents = []string{"ghost"} // inert under fixed
		}, ""},
		{"rotate without agent pool", func(c *Config) { c.Roles.Review.Agents = nil }, "non-empty agent pool"},
		{"empty agent pool entry", func(c *Config) {
			// A blank entry is dropped by ActiveAgents, so without the explicit
			// check it would slip past validation yet be selected by assignments.
			c.Roles.Review.Agents = []string{"rev", ""}
		}, "entry 1 is empty"},
		{"all-once lens rejected for fix run", func(c *Config) {
			c.Roles.Review.Prompts[0].Once = true // the only lens becomes once-only
		}, "recurring reviewer lens"},
		{"all-once lens allowed for review-only", func(c *Config) {
			c.Roles.Review.Prompts[0].Once = true
			c.Loop.ReviewOnly = true
		}, ""},
		{"undefined coder agent", func(c *Config) { c.Roles.Coder.Agent = "ghost" }, "not defined"},
		{"missing coder agent name", func(c *Config) { c.Roles.Coder.Agent = "" }, "roles.coder.agent: required"},
		{"undefined review agent", func(c *Config) { c.Roles.Review.Agents = []string{"ghost"} }, "not defined"},
		{"undefined pinned lens agent", func(c *Config) { c.Roles.Review.Prompts[0].Agent = "ghost" }, "not defined"},
		{"empty command", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = nil
			c.Agents["rev"] = a
		}, "empty command"},
		{"binary not on PATH", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"definitely-not-a-binary-xyz"}
			c.Agents["rev"] = a
		}, "not found on PATH"},
		{"invalid prompt_via", func(c *Config) {
			a := c.Agents["rev"]
			a.PromptVia = "env"
			c.Agents["rev"] = a
		}, "prompt_via must be stdin or arg"},
		{"coder cannot edit", func(c *Config) {
			a := c.Agents["coder"]
			a.CanEdit = false
			c.Agents["coder"] = a
		}, "must be able to edit"},
		{"negative max findings", func(c *Config) { c.Loop.MaxFindingsPerRound = -1 }, "must not be negative"},
		{"negative max iterations", func(c *Config) { c.Loop.MaxIterations = -1 }, "must not be negative"},
		{"negative clean rounds", func(c *Config) { c.Loop.CleanRoundsToStop = -1 }, "must not be negative"},
		{"writable reviewer rejected", func(c *Config) {
			// Reviewers run concurrently against the shared tree; a write-capable
			// one could race the others (or the coder), so it must be rejected.
			a := c.Agents["rev"]
			a.CanEdit = true
			c.Agents["rev"] = a
		}, "must be read-only"},
		{"writable pinned reviewer rejected", func(c *Config) {
			c.Roles.Review.Strategy = "fixed"
			c.Roles.Review.Prompts[0].Agent = "rev"
			a := c.Agents["rev"]
			a.CanEdit = true
			c.Agents["rev"] = a
		}, "must be read-only"},
		{"missing coder prompt", func(c *Config) { c.Roles.Coder.Prompt = "" }, "must reference a prompt file"},
		{"unreadable prompt file", func(c *Config) { c.Roles.Coder.Prompt = "/nonexistent/prompt.md" }, "prompt file"},
		{"unknown log format", func(c *Config) { c.Logs.Formats = []string{"xml"} }, "unknown format"},
		{"duplicate agent pool entry", func(c *Config) {
			// A duplicate invokes the same reviewer twice under strategy all,
			// recording (and counting) the same review twice.
			c.Roles.Review.Agents = []string{"rev", "rev"}
		}, "both name"},
		// logs.pattern must render every distinct artifact to a distinct path.
		{"logs.pattern without {ext}", func(c *Config) {
			c.Logs.Pattern = "{role}-{agent}-{prompt}"
		}, "must contain {ext}"},
		{"logs.pattern without {role}", func(c *Config) {
			c.Logs.Pattern = "{agent}-{prompt}.{ext}"
		}, "must contain {role}"},
		{"logs.pattern without {agent}", func(c *Config) {
			c.Logs.Pattern = "{role}-{prompt}.{ext}"
		}, "must contain {agent}"},
		{"logs.pattern without {prompt}", func(c *Config) {
			c.Logs.Pattern = "{role}-{agent}.{ext}"
		}, "must contain {prompt}"},
		{"custom valid logs pattern accepted", func(c *Config) {
			c.Logs.Pattern = "{role}/{agent}/{prompt}-{round}.{ext}"
			c.Logs.SummaryPattern = "run-summary.{ext}"
		}, ""},
		{"summary_pattern without {ext}", func(c *Config) {
			c.Logs.SummaryPattern = "summary.md"
		}, "must contain {ext}"},
		// env.pass names variables to inherit, so a `FOO=bar` entry passes nothing
		// and silently starves the agent of the credential it needed. The error must
		// name the offending VARIABLE and its index -- an earlier version shadowed
		// them with the agent name, pointing readers at a nonexistent agent.
		{"invalid env.pass name", func(c *Config) {
			a := c.Agents["rev"]
			a.Env = AgentEnv{Pass: []string{"HOME", "FOO=bar"}}
			c.Agents["rev"] = a
		}, `env.pass[1] ("FOO=bar")`},
		{"valid env.pass name accepted", func(c *Config) {
			a := c.Agents["rev"]
			a.Env = AgentEnv{Pass: []string{"HOME"}}
			c.Agents["rev"] = a
		}, ""},
		{"invalid env.set name", func(c *Config) {
			a := c.Agents["rev"]
			a.Env = AgentEnv{Set: map[string]string{"NOT A NAME": "x"}}
			c.Agents["rev"] = a
		}, "env.set"},
		// A fix run in git-diff mode with no base_ref diffs only unstaged changes,
		// and fix rounds start from a clean tree -- so that diff is always empty and
		// every round would review nothing.
		{"git-diff fix run without base_ref", func(c *Config) {
			c.Target.Mode = ModeGitDiff
		}, "target.base_ref"},
		{"git-diff fix run with base_ref", func(c *Config) {
			c.Target.Mode = ModeGitDiff
			c.Target.BaseRef = "main"
		}, ""},
		{"git-diff review-only without base_ref", func(c *Config) {
			c.Target.Mode = ModeGitDiff
			c.Loop.ReviewOnly = true
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig(t)
			tc.mutate(cfg)
			err := cfg.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

// Two review lenses whose prompt files share a basename collapse to the same
// {prompt} log token; under strategy all one agent runs both in a round and
// their reviewer goroutines race to os.WriteFile the same step-log path,
// silently losing one durable record. Validate must reject that, and must still
// accept distinct basenames living in different directories.
func TestValidateDuplicateLensBasename(t *testing.T) {
	t.Run("shared basename rejected", func(t *testing.T) {
		// The seenLens check fires before the prompt-file-readable check, so these
		// paths need not exist for the collision to be detected.
		cfg := validConfig(t)
		cfg.Roles.Review.Prompts = []ReviewLens{
			{Prompt: "prompts/a/review.md"},
			{Prompt: "prompts/b/review.md"},
		}
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "both resolve to log name") {
			t.Fatalf("Validate() = %v, want duplicate-basename rejection", err)
		}
	})

	t.Run("distinct basenames accepted", func(t *testing.T) {
		cfg := validConfig(t)
		dir := t.TempDir()
		a := filepath.Join(dir, "review-bugs.md")
		b := filepath.Join(dir, "review-tests.md")
		for _, p := range []string{a, b} {
			if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		cfg.Roles.Review.Prompts = []ReviewLens{{Prompt: a}, {Prompt: b}}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate() = %v, want nil for distinct basenames", err)
		}
	})
}

// A relative command containing a path separator is resolved against
// target.path (agent.Run sets Cmd.Dir there), not fixpoint's launch cwd.
func TestValidateTargetRelativeBinary(t *testing.T) {
	writeExec := func(t *testing.T, dir, name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	withRevCommand := func(t *testing.T, path string, cmd ...string) *Config {
		t.Helper()
		cfg := validConfig(t)
		cfg.Target.Path = path
		rev := cfg.Agents["rev"]
		rev.Command = cmd
		cfg.Agents["rev"] = rev
		return cfg
	}

	t.Run("accepts ./agent.sh under target.path '.'", func(t *testing.T) {
		// The regression: with target.path ".", filepath.Join cleans
		// "./agent.sh" to "agent.sh", which would send LookPath to a PATH search
		// and wrongly reject a valid target-local binary. Chdir so "." is a dir
		// that actually holds the script.
		dir := t.TempDir()
		writeExec(t, dir, "agent.sh")
		t.Chdir(dir)
		if err := withRevCommand(t, ".", "./agent.sh").Validate(); err != nil {
			t.Fatalf("Validate() = %v, want nil for a target-local ./agent.sh", err)
		}
	})

	t.Run("accepts ./agent.sh under an explicit target dir", func(t *testing.T) {
		dir := t.TempDir()
		writeExec(t, dir, "agent.sh")
		if err := withRevCommand(t, dir, "./agent.sh").Validate(); err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
	})

	t.Run("rejects ./agent.sh absent from target.path", func(t *testing.T) {
		if err := withRevCommand(t, t.TempDir(), "./agent.sh").Validate(); err == nil ||
			!strings.Contains(err.Error(), "not found on PATH") {
			t.Fatalf("Validate() = %v, want rejection of a binary absent from target.path", err)
		}
	})
}

func TestValidateRejectsNegativeTimeout(t *testing.T) {
	cfg := validConfig(t)
	rev := cfg.Agents["rev"]
	rev.Timeout = Duration(-1 * time.Second)
	cfg.Agents["rev"] = rev
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "timeout must not be negative") {
		t.Fatalf("Validate() = %v, want negative-timeout rejection", err)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	promptPath := filepath.Join(dir, "p.md")
	if err := os.WriteFile(promptPath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	write := func(t *testing.T, yaml string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "fixpoint.yaml")
		if err := os.WriteFile(p, []byte(yaml), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	valid := `
target: {mode: directory}
roles:
  coder: {agent: coder, prompt: ` + promptPath + `}
  review:
    strategy: rotate
    agents: [rev]
    prompts: [` + promptPath + `]
agents:
  coder: {command: [echo], can_edit: true, timeout: 5m}
  rev: {command: [echo]}
`

	t.Run("valid file with defaults applied", func(t *testing.T) {
		cfg, err := Load(write(t, valid))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Target.Path != "." {
			t.Errorf("Target.Path default = %q, want .", cfg.Target.Path)
		}
		if cfg.Loop.MaxIterations != 5 {
			t.Errorf("MaxIterations default = %d, want 5", cfg.Loop.MaxIterations)
		}
		if cfg.Loop.CleanRoundsToStop != 1 {
			t.Errorf("CleanRoundsToStop default = %d, want 1", cfg.Loop.CleanRoundsToStop)
		}
		if cfg.Agents["rev"].PromptVia != PromptViaStdin {
			t.Errorf("PromptVia default = %q, want stdin", cfg.Agents["rev"].PromptVia)
		}
		// The default logs dir must be hidden and tool-owned. fixpoint runs against
		// projects it does not own, and this prefix becomes both a git pathspec and
		// a walk-skip -- a generic name like "logs" would drop a project's own logs
		// directory from review scope and from round commits.
		if got := cfg.Logs.Dir; got != ".fixpoint/{timestamp}/round-{round}" {
			t.Errorf("Logs.Dir default = %q, want the .fixpoint template", got)
		}
		if base := cfg.Logs.StaticBase(); base != ".fixpoint" {
			t.Errorf("default Logs.StaticBase() = %q, want .fixpoint", base)
		}
		if cfg.Agents["rev"].Timeout.Std() != 10*time.Minute {
			t.Errorf("Timeout default = %s, want 10m", cfg.Agents["rev"].Timeout.Std())
		}
		if cfg.Agents["coder"].Timeout.Std() != 5*time.Minute {
			t.Errorf("explicit timeout = %s, want 5m", cfg.Agents["coder"].Timeout.Std())
		}
		if !cfg.Ping() {
			t.Error("Ping() default = false, want true")
		}
		if cfg.Loop.AllowUntrustedFix {
			t.Error("AllowUntrustedFix default = true, want false")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
			t.Fatal("Load() = nil, want error")
		}
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		if _, err := Load(write(t, valid+"\ntypo_key: true\n")); err == nil {
			t.Fatal("Load() = nil, want unknown-field error")
		}
	})

	t.Run("unknown review lens field rejected", func(t *testing.T) {
		bad := strings.Replace(valid, "prompts: ["+promptPath+"]",
			"prompts: [{prompt: "+promptPath+", agnet: rev}]", 1)
		_, err := Load(write(t, bad))
		if err == nil || !strings.Contains(err.Error(), "unknown review lens field") {
			t.Fatalf("Load() = %v, want unknown lens field error", err)
		}
	})

	t.Run("invalid duration", func(t *testing.T) {
		bad := strings.Replace(valid, "timeout: 5m", "timeout: 5 minutes", 1)
		_, err := Load(write(t, bad))
		if err == nil || !strings.Contains(err.Error(), "invalid duration") {
			t.Fatalf("Load() = %v, want invalid duration error", err)
		}
	})

	// Zero means "unset" and gets the documented default (asserted above), but a
	// negative loop limit is invalid operator input and must be rejected rather
	// than silently masked by that default.
	t.Run("negative loop limits rejected", func(t *testing.T) {
		for _, field := range []string{"max_iterations: -1", "clean_rounds_to_stop: -1"} {
			bad := strings.Replace(valid, "target: {mode: directory}",
				"target: {mode: directory}\nloop: {"+field+"}", 1)
			_, err := Load(write(t, bad))
			if err == nil || !strings.Contains(err.Error(), "must not be negative") {
				t.Fatalf("Load(%s) = %v, want must-not-be-negative error", field, err)
			}
		}
	})

	t.Run("validation failure surfaces", func(t *testing.T) {
		bad := strings.Replace(valid, "mode: directory", "mode: svn", 1)
		_, err := Load(write(t, bad))
		if err == nil || !strings.Contains(err.Error(), "unknown mode") {
			t.Fatalf("Load() = %v, want unknown mode error", err)
		}
	})
}

func TestActiveAgents(t *testing.T) {
	rv := Review{
		Agents: []string{"a1", "a2"},
		Prompts: []ReviewLens{
			{Prompt: "p1.md", Agent: "pinned"},
			{Prompt: "p2.md"},
			{Prompt: "p3.md", Agent: "a1"}, // duplicate of the pool entry
		},
	}
	rv.Strategy = "fixed"
	if got := rv.ActiveAgents(); !reflect.DeepEqual(got, []string{"pinned", "a1"}) {
		t.Errorf("fixed ActiveAgents() = %v, want pinned agents only", got)
	}
	rv.Strategy = "rotate"
	if got := rv.ActiveAgents(); !reflect.DeepEqual(got, []string{"pinned", "a1", "a2"}) {
		t.Errorf("rotate ActiveAgents() = %v, want pinned + pool, deduplicated", got)
	}
}

func TestLensName(t *testing.T) {
	if got := LensName("prompts/review-bugs.md"); got != "review-bugs" {
		t.Errorf("LensName() = %q, want review-bugs", got)
	}
}
