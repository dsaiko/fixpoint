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
	return writeNamedPrompt(t, "prompt.md")
}

// writeNamedPrompt is writePrompt with control over the basename, which is what
// LensName derives the {prompt} log token from.
func writeNamedPrompt(t *testing.T, base string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), base)
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
		Loop:   Loop{CommitPolicy: CommitPerFix},
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
		// commit_policy decides whether history is rewritten, so a typo must not
		// silently fall back to a default that squashes (or does not).
		{"unknown commit policy", func(c *Config) { c.Loop.CommitPolicy = "per-fix" }, "unknown policy"},
		{"commit policy per_round accepted", func(c *Config) { c.Loop.CommitPolicy = CommitPerRound }, ""},
		{"commit policy per_run accepted", func(c *Config) { c.Loop.CommitPolicy = CommitPerRun }, ""},
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
		// A reviewer's can_edit: false is the only barrier between a prompt
		// injection in the reviewed code and the write tools, so the command must
		// not contradict it. Each spelling below is one way to switch the
		// permission system off while still declaring read-only.
		{"read-only reviewer passing a permission-skip flag rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--dangerously-skip-permissions", "-p"}
			c.Agents["rev"] = a
		}, "--dangerously-skip-permissions"},
		{"read-only reviewer passing codex's bypass flag rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--dangerously-bypass-approvals-and-sandbox"}
			c.Agents["rev"] = a
		}, "lets the agent write files"},
		{"read-only reviewer passing --yolo rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--yolo"}
			c.Agents["rev"] = a
		}, "--yolo"},
		// A bypass switch spelled =true is the same grant written differently, while
		// =false is an opt-out no parser reads as on -- rejecting that one would
		// refuse a config for asking for the safe thing.
		{"read-only reviewer passing --yolo=true rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--yolo=true"}
			c.Agents["rev"] = a
		}, "--yolo"},
		{"read-only reviewer passing --yolo=false accepted", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--yolo=false"}
			c.Agents["rev"] = a
		}, ""},
		{"read-only reviewer passing a permission-skip flag =false accepted", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--dangerously-skip-permissions=FALSE", "-p"}
			c.Agents["rev"] = a
		}, ""},
		{"read-only reviewer passing bypassPermissions as a mode value rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--permission-mode bypassPermissions"}
			c.Agents["rev"] = a
		}, "--permission-mode bypassPermissions"},
		{"read-only reviewer passing bypassPermissions with = rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--permission-mode=bypassPermissions"}
			c.Agents["rev"] = a
		}, "--permission-mode bypassPermissions"},
		// A write-granting mode need not be the full bypass: acceptEdits auto-approves
		// the edit tools alone, which is the whole of what can_edit: false forbids.
		{"read-only reviewer passing acceptEdits as a mode value rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--permission-mode acceptEdits", "-p"}
			c.Agents["rev"] = a
		}, "--permission-mode acceptEdits"},
		{"read-only reviewer passing acceptEdits with = rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--permission-mode=acceptEdits"}
			c.Agents["rev"] = a
		}, "--permission-mode acceptEdits"},
		{"read-only reviewer passing codex --full-auto rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "--full-auto"}
			c.Agents["rev"] = a
		}, "--full-auto"},
		{"read-only reviewer passing codex --sandbox workspace-write rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "--sandbox workspace-write"}
			c.Agents["rev"] = a
		}, "--sandbox workspace-write"},
		{"read-only reviewer passing codex --sandbox danger-full-access rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "--sandbox=danger-full-access"}
			c.Agents["rev"] = a
		}, "--sandbox danger-full-access"},
		{"read-only reviewer passing codex's short sandbox spelling rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "-s workspace-write"}
			c.Agents["rev"] = a
		}, "-s workspace-write"},
		// codex spells every flag as a settings override too, and the override wins
		// over --sandbox read-only, so each -c/--config spelling of sandbox_mode has
		// to be refused as well -- otherwise the check is one argument away from
		// being bypassed while still declaring can_edit: false.
		{"read-only reviewer passing codex -c sandbox_mode rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "-c sandbox_mode=workspace-write", "-"}
			c.Agents["rev"] = a
		}, "sandbox_mode=workspace-write"},
		{"read-only reviewer passing codex --config sandbox_mode rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "--config sandbox_mode=danger-full-access", "-"}
			c.Agents["rev"] = a
		}, "sandbox_mode=danger-full-access"},
		{"read-only reviewer passing codex --config=sandbox_mode rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "--config=sandbox_mode=workspace-write"}
			c.Agents["rev"] = a
		}, "sandbox_mode=workspace-write"},
		{"read-only reviewer passing codex -c with the value attached rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "-csandbox_mode=workspace-write"}
			c.Agents["rev"] = a
		}, "sandbox_mode=workspace-write"},
		// The value is TOML, so a string may be quoted; the quotes must not hide it.
		{"read-only reviewer passing a quoted codex sandbox_mode rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", `-c sandbox_mode="workspace-write"`, "-"}
			c.Agents["rev"] = a
		}, "sandbox_mode=workspace-write"},
		// An override that grants nothing is ordinary configuration: config/agents/codex.yaml
		// itself passes -c model_reasoning_effort, and read-only is the value it earns
		// its can_edit: false with.
		{"read-only reviewer passing an unrelated codex override accepted", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "--sandbox read-only", "-c model_reasoning_effort=high"}
			c.Agents["rev"] = a
		}, ""},
		{"read-only reviewer passing codex -c sandbox_mode=read-only accepted", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", `-c sandbox_mode="read-only"`, "-"}
			c.Agents["rev"] = a
		}, ""},
		{"read-only reviewer passing agy --mode accept-edits rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--mode accept-edits"}
			c.Agents["rev"] = a
		}, "--mode accept-edits"},
		{"read-only reviewer passing gemini --approval-mode auto_edit rejected", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--approval-mode auto_edit"}
			c.Agents["rev"] = a
		}, "--approval-mode auto_edit"},
		// A mode value that is not write-granting is ordinary configuration, and a
		// bypass flag on the write-capable coder is exactly what it is there for.
		{"read-only reviewer in a non-bypass permission mode accepted", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--permission-mode plan"}
			c.Agents["rev"] = a
		}, ""},
		{"read-only reviewer in codex's read-only sandbox accepted", func(c *Config) {
			// What config/agents/codex.yaml ships: the enforced no-write sandbox is
			// how a codex reviewer earns its can_edit: false.
			a := c.Agents["rev"]
			a.Command = []string{"echo", "exec", "--sandbox read-only"}
			c.Agents["rev"] = a
		}, ""},
		{"read-only reviewer in agy plan mode accepted", func(c *Config) {
			a := c.Agents["rev"]
			a.Command = []string{"echo", "--mode plan"}
			c.Agents["rev"] = a
		}, ""},
		{"coder passing a permission-skip flag accepted", func(c *Config) {
			a := c.Agents["coder"]
			a.Command = []string{"echo", "--dangerously-skip-permissions"}
			c.Agents["coder"] = a
		}, ""},
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
		// The logstore joins the rendered pattern onto the round directory, and
		// filepath.Join cleans it -- so identifiers carrying path syntax, and
		// patterns whose own literals do, must be rejected rather than compared as
		// raw strings that look distinct but name one file.
		{"agent name with a path separator rejected", func(c *Config) {
			// Renders "review-x/../rev-..." , which cleans onto plain "rev"'s path.
			c.Agents["x/../rev"] = c.Agents["rev"]
			c.Roles.Review.Agents = []string{"rev", "x/../rev"}
		}, "must not contain a path separator"},
		{"agent name that is a dot segment rejected", func(c *Config) {
			c.Agents[".."] = c.Agents["coder"]
			c.Roles.Coder.Agent = ".."
		}, "must not be a dot segment"},
		{"lens name that is a dot segment rejected", func(c *Config) {
			c.Roles.Review.Prompts = []ReviewLens{{Prompt: writeNamedPrompt(t, "...md")}} // LensName -> ".."
		}, "must not be a dot segment"},
		{"empty lens name rejected", func(c *Config) {
			c.Roles.Review.Prompts = []ReviewLens{{Prompt: writeNamedPrompt(t, ".md")}} // LensName -> ""
		}, "name is empty"},
		{"logs.pattern collision only visible after path cleaning", func(c *Config) {
			// Both reviewers render ".../review/<agent>/../prompt.<ext>", which
			// cleans to one file even though the raw renderings differ.
			c.Logs.Pattern = "{role}/{agent}/../{prompt}.{ext}"
			c.Agents["rev2"] = c.Agents["rev"]
			c.Roles.Review.Strategy = "all"
			c.Roles.Review.Agents = []string{"rev", "rev2"}
		}, "same path"},
		{"logs.pattern climbing out of the round directory rejected", func(c *Config) {
			c.Logs.Pattern = "../{role}-{agent}-{prompt}.{ext}"
		}, "outside its round directory"},
		{"summary_pattern without {ext}", func(c *Config) {
			c.Logs.SummaryPattern = "summary.md"
		}, "must contain {ext}"},
		// logs.redact is the operator's escape hatch from the shape-based built-in
		// redaction rules, so a pattern that cannot do its job must be rejected at
		// startup: otherwise the failure surfaces on the write that was supposed to
		// mask a secret, or (for an empty-matching pattern) only by reading the
		// destroyed artifacts of a finished run.
		{"logs.redact valid patterns accepted", func(c *Config) {
			c.Logs.Redact = []string{`(?i)(x-internal-auth\s*:\s*)\S+`, `ACME-[A-Z0-9]{24}`}
		}, ""},
		{"logs.redact uncompilable pattern rejected", func(c *Config) {
			c.Logs.Redact = []string{`ACME-[A-Z`}
		}, "does not compile"},
		{"logs.redact empty entry rejected", func(c *Config) {
			c.Logs.Redact = []string{"  "}
		}, "logs.redact[0] is empty"},
		{"logs.redact empty-matching pattern rejected", func(c *Config) {
			c.Logs.Redact = []string{`[A-Z0-9]*`}
		}, "matches the empty string"},
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
		// A usage path without a format reads as configured and is never parsed,
		// since every read of it sits behind AgentUsage.Enabled. error_status is the
		// easy one to leave out of the check: it is the only path that feeds the
		// provider-refusal signal rather than the token counters.
		{"usage.error_status without format", func(c *Config) {
			a := c.Agents["rev"]
			a.Usage = AgentUsage{ErrorStatus: "error.status"}
			c.Agents["rev"] = a
		}, "usage.format is empty"},
		{"usage.input_tokens without format", func(c *Config) {
			a := c.Agents["rev"]
			a.Usage = AgentUsage{InputTokens: "usage.input"}
			c.Agents["rev"] = a
		}, "usage.format is empty"},
		{"usage.format without text", func(c *Config) {
			a := c.Agents["rev"]
			a.Usage = AgentUsage{Format: UsageFormatJSON}
			c.Agents["rev"] = a
		}, "usage.text is not"},
		{"usage block with format and text accepted", func(c *Config) {
			a := c.Agents["rev"]
			a.Usage = AgentUsage{Format: UsageFormatJSON, Text: "result", ErrorStatus: "error.status"}
			c.Agents["rev"] = a
		}, ""},
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

// The mandatory credential excludes are the reason directory-mode collection never
// lists a key file to a reviewer, and they live in code precisely so no config can
// drop them ("a safety property must not be something a config can forget"). This
// asserts the property that makes that true: whatever target.exclude says --
// nothing, or a duplicate of one of them -- every mandatory pattern comes back.
func TestEffectiveExcludesAlwaysCarriesTheMandatoryPatterns(t *testing.T) {
	has := func(globs []string, want string) bool {
		for _, g := range globs {
			if g == want {
				return true
			}
		}
		return false
	}
	for _, tc := range []struct {
		name    string
		exclude []string
	}{
		{"empty", nil},
		{"configured", []string{"vendor/**", "*.md"}},
		{"duplicating a mandatory pattern", []string{"**/*.pem", "vendor/**"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Target{Exclude: tc.exclude}.EffectiveExcludes()
			for _, want := range mandatoryExcludes {
				if !has(got, want) {
					t.Errorf("EffectiveExcludes() = %v, missing mandatory %q", got, want)
				}
			}
			for _, want := range tc.exclude {
				if !has(got, want) {
					t.Errorf("EffectiveExcludes() = %v, dropped configured %q", got, want)
				}
			}
			seen := map[string]bool{}
			for _, g := range got {
				if seen[g] {
					t.Errorf("EffectiveExcludes() = %v, repeats %q", got, g)
				}
				seen[g] = true
			}
		})
	}
	// The list itself is the control, so an empty one is a broken control.
	if len(mandatoryExcludes) == 0 {
		t.Fatal("mandatoryExcludes is empty: nothing keeps credential files out of directory-mode collection")
	}
}
