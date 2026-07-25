package agent

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
)

// envDumper writes an agent script that prints its own environment, so these
// tests assert on what the process ACTUALLY receives rather than on what buildEnv
// returns. The distinction matters: a bug in how cmd.Env is wired would be
// invisible to a test of buildEnv alone.
func envDumper(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "dump.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\nenv\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return p
}

func runDumper(t *testing.T, a config.Agent) string {
	t.Helper()
	a.Command = []string{envDumper(t)}
	a.PromptVia = config.PromptViaStdin
	a.Timeout = config.Duration(time.Minute)
	res := Run(t.Context(), a, "prompt", t.TempDir())
	if res.Err != nil {
		t.Fatalf("agent run failed: %v\nstderr: %s", res.Err, res.Stderr)
	}
	return res.Stdout
}

// The point of the whole feature: a secret the agent did not ask for must not be
// in its process. The environment is the one exfiltration surface a container does
// not close, because the agents' own credentials have to be inside the container.
func TestRunWithholdsUndeclaredSecrets(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp-should-not-be-visible")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws-should-not-be-visible")
	t.Setenv("MY_DATABASE_PASSWORD", "db-should-not-be-visible")

	out := runDumper(t, config.Agent{})

	for _, name := range []string{"GITHUB_TOKEN", "AWS_SECRET_ACCESS_KEY", "MY_DATABASE_PASSWORD"} {
		if strings.Contains(out, name) {
			t.Errorf("%s reached the agent process; an undeclared secret must be absent:\n%s", name, out)
		}
	}
}

// ...but the baseline must be present, because dropping it breaks CLIs in ways
// that look like unrelated failures. HOME especially: it is how claude and codex
// find their credentials, so losing it turns a working agent into an auth error.
func TestRunPassesBaselineEnv(t *testing.T) {
	out := runDumper(t, config.Agent{})
	for _, name := range []string{"PATH", "HOME"} {
		if !strings.Contains(out, name+"=") {
			t.Errorf("baseline variable %s is missing; CLIs cannot work without it:\n%s", name, out)
		}
	}
}

// A declared name is inherited when set.
func TestRunPassesDeclaredEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "declared-and-needed")
	t.Setenv("GITHUB_TOKEN", "not-declared")

	out := runDumper(t, config.Agent{Env: config.AgentEnv{Pass: []string{"ANTHROPIC_API_KEY"}}})

	if !strings.Contains(out, "ANTHROPIC_API_KEY=declared-and-needed") {
		t.Errorf("a declared variable must be passed:\n%s", out)
	}
	if strings.Contains(out, "GITHUB_TOKEN") {
		t.Errorf("declaring one variable must not pass others:\n%s", out)
	}
}

// A declared name that is not set in fixpoint's environment is absent, not an
// error: agents commonly accept either an env var or a config file for
// credentials, so requiring it to exist would break the config-file case.
func TestRunToleratesUnsetDeclaredEnv(t *testing.T) {
	os.Unsetenv("SOME_UNSET_CREDENTIAL")
	out := runDumper(t, config.Agent{Env: config.AgentEnv{Pass: []string{"SOME_UNSET_CREDENTIAL"}}})
	if strings.Contains(out, "SOME_UNSET_CREDENTIAL") {
		t.Errorf("an unset declared variable must not appear at all:\n%s", out)
	}
}

// Literal values, and precedence over an inherited value of the same name.
func TestRunSetOverridesInherited(t *testing.T) {
	t.Setenv("NO_COLOR", "from-fixpoint")
	out := runDumper(t, config.Agent{Env: config.AgentEnv{
		Pass: []string{"NO_COLOR"},
		Set:  map[string]string{"NO_COLOR": "from-config"},
	}})
	if !strings.Contains(out, "NO_COLOR=from-config") {
		t.Errorf("env.set must override an inherited value:\n%s", out)
	}
	if strings.Contains(out, "NO_COLOR=from-fixpoint") {
		t.Errorf("the inherited value must not survive alongside the override:\n%s", out)
	}
}

// The escape hatch restores the old behavior for a CLI whose requirements are
// unknown. It is warned about at run start rather than silently allowed.
func TestRunInheritAllPassesEverything(t *testing.T) {
	t.Setenv("SOME_UNRELATED_SECRET", "visible-by-request")
	out := runDumper(t, config.Agent{Env: config.AgentEnv{InheritAll: true}})
	if !strings.Contains(out, "SOME_UNRELATED_SECRET=visible-by-request") {
		t.Errorf("inherit_all must pass the whole environment:\n%s", out)
	}
}

// buildEnv returns nil only for inherit_all, because exec reads nil as "inherit
// the parent's environment". An empty non-nil slice means "empty environment", and
// confusing the two would silently restore full inheritance.
func TestBuildEnvNilOnlyMeansInherit(t *testing.T) {
	if got := buildEnv(config.Agent{Env: config.AgentEnv{InheritAll: true}}); got != nil {
		t.Errorf("inherit_all must yield nil (exec's inherit signal), got %v", got)
	}
	if got := buildEnv(config.Agent{}); got == nil {
		t.Error("the filtered case must never yield nil, or exec would inherit everything")
	}
}

// The baseline is documentation as much as code: it is listed in config/README.md,
// so it must not drift silently. These are the entries whose absence breaks things
// non-obviously.
func TestBaselineCoversEssentials(t *testing.T) {
	names := BaselineEnvNames()
	for _, want := range []string{"PATH", "HOME", "TMPDIR", "LANG", "SSL_CERT_FILE", "HTTPS_PROXY", "XDG_CONFIG_HOME"} {
		if !slices.Contains(names, want) {
			t.Errorf("baseline is missing %s; its absence breaks CLIs in ways that look unrelated", want)
		}
	}
	// The baseline must not itself carry obvious credentials.
	for _, unwanted := range []string{"GITHUB_TOKEN", "ANTHROPIC_API_KEY", "AWS_SECRET_ACCESS_KEY"} {
		if slices.Contains(names, unwanted) {
			t.Errorf("baseline must not include the credential %s", unwanted)
		}
	}
}
