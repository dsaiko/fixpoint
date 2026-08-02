package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/testfixture"
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

// dumpedEnv turns an envDumper run's stdout back into name -> value. A duplicate
// name resolves to the LAST entry, matching how os/exec dedups cmd.Env, so an
// injected override is what this reports rather than the value it shadowed.
func dumpedEnv(t *testing.T, out string) map[string]string {
	t.Helper()
	env := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, "=")
		// A value containing a newline continues onto the next line; those
		// continuations are not assignments, and a name never has a space in it.
		if !ok || k == "" || strings.ContainsAny(k, " \t") {
			continue
		}
		env[k] = v
	}
	return env
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

// The agents run their OWN git inside the target -- `git status`/`git diff`/
// `git log` while exploring -- so the environment they get must carry the same
// safe-config pins fixpoint puts on its own git calls. core.fsmonitor is the sharp
// case: a crafted checkout can point it at a script, the preflight cannot refuse on
// it (`core.fsmonitor = true` is the legitimate builtin daemon, so refusing would
// refuse real repos), and gitenv pins it to false instead. Without the pins in the
// agent's environment, a review-only directory run -- which passes no trust gate --
// would execute that script with the credential the agent declared.
func TestRunHardensGitForAgentInvokedGit(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable to run the fsmonitor program")
	}
	// runGitStatus points an agent at a repo whose .git/config names an fsmonitor
	// program, has it run `git status`, and reports whether the program fired.
	runGitStatus := func(t *testing.T, a config.Agent) bool {
		t.Helper()
		repo := testfixture.GitRepo(t)
		sentinel := filepath.Join(repo, "fsmonitor-ran")
		evil := filepath.Join(repo, "evil-fsmonitor.sh")
		// A v1 fsmonitor program prints the paths it considers dirty; the sentinel is
		// the side effect core.fsmonitor=false must prevent.
		if err := os.WriteFile(evil, []byte("#!/bin/sh\ntouch '"+sentinel+"'\nprintf '/\\0'\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		testfixture.GitRun(t, repo, "config", "core.fsmonitor", evil)

		script := filepath.Join(t.TempDir(), "explore.sh")
		if err := os.WriteFile(script, []byte("#!/bin/sh\ngit status --porcelain\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		a.Command = []string{script}
		a.PromptVia = config.PromptViaStdin
		a.Timeout = config.Duration(time.Minute)
		if res := Run(t.Context(), a, "prompt", repo); res.Err != nil {
			t.Fatalf("agent run failed: %v\nstderr: %s", res.Err, res.Stderr)
		}
		_, err := os.Stat(sentinel)
		return err == nil
	}

	t.Run("filtered environment", func(t *testing.T) {
		if runGitStatus(t, config.Agent{}) {
			t.Error("the repo-configured fsmonitor program ran; the agent environment lost gitenv's GIT_CONFIG_* pins")
		}
	})

	// inherit_all is about handing the CLI fixpoint's own variables, not about
	// letting the target run code: buildEnv returns nil there, and Harden must expand
	// that to the parent environment PLUS the pins rather than leave cmd.Env nil.
	t.Run("inherit_all", func(t *testing.T) {
		if runGitStatus(t, config.Agent{Env: config.AgentEnv{InheritAll: true}}) {
			t.Error("the repo-configured fsmonitor program ran under inherit_all; the pins must survive the escape hatch")
		}
	})
}

// The mirror of the test above: that one pins what gitenv.Harden MUST export into
// the agent's environment, this pins what must NOT be exported there. fixpoint's
// own git subprocesses run under LC_ALL=C (target.probeEnv) so that the fatal
// messages notARepository/noWorkTree parse stay untranslated; that pin lives at
// the probe rather than in Harden precisely because Harden's result is also the
// reviewer/coder CLIs' environment. Folding it into Harden is the obvious
// simplification -- every other pin is already there -- and it would leave the
// suite green while silently forcing a C locale on every agent CLI, mangling
// non-ASCII output for any operator not already running one.
func TestRunKeepsOperatorLocaleForAgent(t *testing.T) {
	const locale = "en_US.UTF-8"
	t.Setenv("LC_ALL", locale)
	t.Setenv("LANG", locale)

	assertLocale := func(t *testing.T, a config.Agent) {
		t.Helper()
		env := dumpedEnv(t, runDumper(t, a))
		for _, name := range []string{"LC_ALL", "LANG"} {
			if got := env[name]; got != locale {
				t.Errorf("agent saw %s=%q, want the operator's own %q: the C-locale pin is for fixpoint's own git, and exporting it here changes the CLI's output encoding", name, got, locale)
			}
		}
	}

	t.Run("filtered environment", func(t *testing.T) { assertLocale(t, config.Agent{}) })
	t.Run("inherit_all", func(t *testing.T) {
		assertLocale(t, config.Agent{Env: config.AgentEnv{InheritAll: true}})
	})
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

// The verify gate runs argv the TARGET can supply, so it must not receive the
// agents' credentials -- otherwise a `curl $ANTHROPIC_API_KEY` verify command
// walks around the filtering buildEnv applies to the agents themselves. Both
// sources of "this is a credential" must be honored: the built-in list and
// whatever the configured agents declared.
func TestEnvWithoutCredentialsStripsAgentSecrets(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "builtin-should-be-stripped")
	t.Setenv("GITHUB_TOKEN", "builtin-should-be-stripped")
	t.Setenv("MY_HOUSE_TOKEN", "declared-should-be-stripped")
	t.Setenv("SOME_BUILD_FLAG", "must-survive")

	env := EnvWithoutCredentials(map[string]config.Agent{
		"house": {Env: config.AgentEnv{Pass: []string{"MY_HOUSE_TOKEN"}}},
	})

	names := map[string]bool{}
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		names[k] = true
	}
	for _, gone := range []string{"ANTHROPIC_API_KEY", "GITHUB_TOKEN", "MY_HOUSE_TOKEN"} {
		if names[gone] {
			t.Errorf("%s survived; a verify command must not see an agent credential", gone)
		}
	}
	// A denylist, not an allowlist: a build command's real requirements are
	// project-specific, and dropping what the list forgot would break checks.
	for _, kept := range []string{"PATH", "SOME_BUILD_FLAG"} {
		if !names[kept] {
			t.Errorf("%s was stripped; only credentials may be removed from a verify command's environment", kept)
		}
	}
	// nil means "inherit everything" to exec, so this must never return nil.
	if EnvWithoutCredentials(nil) == nil {
		t.Error("EnvWithoutCredentials must never return nil: exec would inherit the full environment")
	}
}

// verifyEnvNames runs the gate's filter and returns the names that survived.
func verifyEnvNames(t *testing.T, agents map[string]config.Agent) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	for _, kv := range EnvWithoutCredentials(agents) {
		k, _, _ := strings.Cut(kv, "=")
		names[k] = true
	}
	return names
}

// The operator's own secrets are just as exfiltratable as the agents' -- a verify
// command reads the whole environment it is handed -- and fixpoint cannot know the
// name of every vendor's token. So the gate matches the SHAPE of a credential
// name, and this test pins both halves of that: the names it must strip, and the
// ordinary build variables it must not, because a rule that eats GIT_AUTHOR_NAME
// or TOKENIZERS_PARALLELISM breaks checks for no security gain.
func TestEnvWithoutCredentialsStripsCredentialShapedNames(t *testing.T) {
	stripped := []string{
		// The vendor keys the hardcoded roster used to name one by one.
		"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "OPENAI_API_KEY", "CODEX_API_KEY",
		"GOOGLE_API_KEY", "GEMINI_API_KEY", "GITHUB_TOKEN", "GH_TOKEN",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
		// CI/CD credentials nobody enumerated, plus a purely in-house one.
		"NPM_TOKEN", "DOCKER_PASSWORD", "PYPI_TOKEN", "TWINE_PASSWORD", "SONAR_TOKEN",
		"GPG_PASSPHRASE", "GOOGLE_APPLICATION_CREDENTIALS", "AZURE_CLIENT_SECRET",
		"SSH_PRIVATE_KEY", "DB_PASSWORD", "ACME_INTERNAL_TOKEN", "SECRETS_FILE",
		// Exact-match entries: credential material whose name says nothing. The
		// agent sockets are the sharpest of them -- SSH_AUTH_SOCK is a live
		// connection to the operator's keys, so a build recipe the target just
		// edited could `git push` or `ssh deploy@prod` as them without reading a
		// key file. This list is the control for that denylist: dropping an entry
		// from knownCredentialEnv must fail here.
		"KUBECONFIG", "NETRC", "DOCKER_AUTH_CONFIG",
		"SSH_AUTH_SOCK", "SSH_AGENT_PID", "GPG_AGENT_INFO", "GNUPGHOME",
	}
	kept := []string{
		"GIT_AUTHOR_NAME",        // AUTH is not a word boundary match
		"TOKENIZERS_PARALLELISM", // nor is TOKEN inside TOKENIZERS
		"COMPASS_URL",            // nor PASS inside COMPASS
		"SSH_KEY_ALGORITHMS",     // bare KEY is not a credential word
		"GOFLAGS", "JAVA_HOME",   // the ordinary build environment
	}
	for _, name := range append(append([]string(nil), stripped...), kept...) {
		t.Setenv(name, "value")
	}

	names := verifyEnvNames(t, nil)
	for _, gone := range stripped {
		if names[gone] {
			t.Errorf("%s survived; a credential-shaped variable must not reach a target-supplied verify command", gone)
		}
	}
	for _, want := range kept {
		if !names[want] {
			t.Errorf("%s was stripped; the shape rule must match on underscore boundaries, not as a substring", want)
		}
	}
}

// FIXPOINT_STRIP_ENV / FIXPOINT_KEEP_ENV are the operator's adjustments, and they
// are environment variables rather than config keys because a bundle inside the
// target shadows the operator's -- so a keep list in YAML would let the reviewed
// repository hand itself the very secrets this gate withholds. The asymmetry is
// the point: keep can rescue a shape match, never an explicit denial.
func TestEnvWithoutCredentialsOperatorLists(t *testing.T) {
	t.Setenv("HOUSE_BLEND", "bespoke-secret-that-does-not-look-like-one")
	t.Setenv("SPARE_ME_TOKEN", "the-check-really-needs-this")
	t.Setenv("MY_HOUSE_TOKEN", "declared-by-an-agent")
	t.Setenv("KUBECONFIG", "explicitly-denied")
	t.Setenv(stripEnvVar, "HOUSE_BLEND, KUBECONFIG")
	t.Setenv(keepEnvVar, "SPARE_ME_TOKEN MY_HOUSE_TOKEN KUBECONFIG")

	names := verifyEnvNames(t, map[string]config.Agent{
		"house": {Env: config.AgentEnv{Pass: []string{"MY_HOUSE_TOKEN"}}},
	})

	if !names["SPARE_ME_TOKEN"] {
		t.Error("SPARE_ME_TOKEN was stripped; FIXPOINT_KEEP_ENV must spare a name the shape rule matched, else an over-broad match leaves a legitimate check unfixable")
	}
	for _, gone := range []string{"HOUSE_BLEND", "MY_HOUSE_TOKEN", "KUBECONFIG"} {
		if names[gone] {
			t.Errorf("%s survived; FIXPOINT_KEEP_ENV must not override an explicit denial (the strip list, an agent's env.pass, or the built-in names)", gone)
		}
	}
	// fixpoint's own knobs describe the filter; the filtered process has no use
	// for them.
	for _, gone := range []string{stripEnvVar, keepEnvVar} {
		if names[gone] {
			t.Errorf("%s survived; it configures the filter and must not be passed through it", gone)
		}
	}
}
