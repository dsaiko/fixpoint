package agent

import (
	"os"
	"sort"
	"strings"

	"github.com/dsaiko/fixpoint/internal/config"
)

// baselineEnv are the variables every agent gets regardless of what it declares.
// They are not secrets, and omitting them breaks CLIs in ways that look like
// unrelated failures — which is the trap this list exists to avoid.
//
// Each group is here for a reason:
//
//   - PATH, HOME: without these nothing resolves. HOME in particular is how
//     `claude` and `codex` find their credential and config files, so dropping it
//     turns a working agent into an authentication error.
//   - TMPDIR/TMP/TEMP: CLIs write scratch files; without a temp dir they fail
//     late and obscurely.
//   - LANG/LC_*/TERM/TZ: locale and terminal handling. Missing locale corrupts
//     non-ASCII output, which matters when the reviewed material is not English.
//   - XDG_*: where a CLI looks for config on Linux. If the operator has set these
//     and we drop them, the CLI silently reads a different (or no) config.
//   - TLS trust: SSL_CERT_FILE, SSL_CERT_DIR, NODE_EXTRA_CA_CERTS, CURL_CA_BUNDLE,
//     REQUESTS_CA_BUNDLE. Behind a corporate MITM proxy, dropping these makes
//     every HTTPS call fail certificate verification.
//   - Proxy settings: without them there is no network access at all in a
//     proxied environment. NOTE these can embed credentials
//     (http://user:pass@proxy), so they are the one baseline entry that may carry
//     a secret; the redactor masks URI passwords, and an operator who cannot
//     accept that should unset them for fixpoint's own process.
var baselineEnv = []string{
	"PATH", "HOME",
	"TMPDIR", "TMP", "TEMP",
	"LANG", "LANGUAGE", "LC_ALL", "LC_CTYPE", "TERM", "TZ",
	"USER", "LOGNAME",
	"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR",
	"SSL_CERT_FILE", "SSL_CERT_DIR", "NODE_EXTRA_CA_CERTS", "CURL_CA_BUNDLE", "REQUESTS_CA_BUNDLE",
	"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY",
	"http_proxy", "https_proxy", "all_proxy", "no_proxy",
}

// BaselineEnvNames returns the always-passed variable names, for documentation and
// tests. Sorted, so output is stable.
func BaselineEnvNames() []string {
	out := append([]string(nil), baselineEnv...)
	sort.Strings(out)
	return out
}

// knownCredentialEnv are variables that carry a credential and belong only in the
// process that authenticates with them. It backs EnvWithoutCredentials, which
// strips them from the subprocesses fixpoint runs that are NOT agents (the verify
// gate's project-supplied build/test commands).
//
// The list is a DENYLIST on purpose. A build command's real requirements are
// language- and project-specific (GOFLAGS, JAVA_HOME, CARGO_HOME, npm_config_*,
// VIRTUAL_ENV, ...), so an allowlist like baselineEnv would have to be re-derived
// per ecosystem and would silently break checks by dropping what it forgot -- the
// same reasoning that makes fixpoint's scope filtering denylist-only. The
// agents' own declarations (env.pass / env.set) extend this at run time, so an
// operator who authenticates an agent with some other variable is covered without
// this list having to know about it.
var knownCredentialEnv = []string{
	"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN",
	"OPENAI_API_KEY", "CODEX_API_KEY",
	"GOOGLE_API_KEY", "GEMINI_API_KEY",
	"GITHUB_TOKEN", "GH_TOKEN",
	"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
}

// EnvWithoutCredentials returns fixpoint's own environment minus every variable
// that carries an agent credential: the names the configured agents declare
// (env.pass and env.set) plus knownCredentialEnv.
//
// It exists because the verify gate runs argv the TARGET supplies (a bundle file
// inside the target shadows the operator's), and inheriting fixpoint's whole
// environment there would hand those commands exactly the secrets buildEnv keeps
// away from the agents themselves -- a `curl $ANTHROPIC_API_KEY` verify command
// would exfiltrate every agent credential before a coder edits anything.
//
// Unlike buildEnv this never returns nil: an empty result must mean "an empty
// environment", not "inherit the parent's".
func EnvWithoutCredentials(agents map[string]config.Agent) []string {
	deny := make(map[string]bool, len(knownCredentialEnv))
	for _, name := range knownCredentialEnv {
		deny[name] = true
	}
	for _, a := range agents {
		for _, name := range a.Env.Pass {
			deny[name] = true
		}
		for name := range a.Env.Set {
			deny[name] = true
		}
	}
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if k, _, ok := strings.Cut(kv, "="); ok && deny[k] {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// buildEnv assembles the environment for one agent invocation: the baseline, plus
// the names the agent declared, plus its literal values. Anything else in
// fixpoint's environment is absent from the process.
//
// Returns nil when the agent opted into inherit_all, which makes exec fall back to
// the parent's environment — the pre-allowlist behavior, kept as an escape hatch
// for a CLI whose requirements are unknown.
func buildEnv(a config.Agent) []string {
	if a.Env.InheritAll {
		return nil
	}
	// Later entries win in exec's handling, but be explicit rather than relying on
	// that: build a map so `set` unambiguously overrides an inherited value.
	vals := map[string]string{}
	for _, name := range baselineEnv {
		if v, ok := os.LookupEnv(name); ok {
			vals[name] = v
		}
	}
	for _, name := range a.Env.Pass {
		if v, ok := os.LookupEnv(name); ok {
			vals[name] = v
		}
		// Not set in fixpoint's environment: absent, not an error. Agents commonly
		// accept either an env var or a config file for credentials, so requiring the
		// variable to exist would break the config-file case.
	}
	for k, v := range a.Env.Set {
		vals[k] = v
	}

	out := make([]string, 0, len(vals))
	for _, k := range sortedEnvKeys(vals) {
		out = append(out, k+"="+vals[k])
	}
	// A non-nil empty slice would mean "empty environment"; nil means "inherit".
	// Every agent gets at least PATH in practice, but be explicit about the
	// distinction so a stripped test environment cannot silently inherit.
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func sortedEnvKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// EnvNames returns the variable names an agent will actually receive, for the
// run's provenance log. Reporting what an agent CAN see is worth as much as
// reporting which prompt drove it.
func EnvNames(a config.Agent) []string {
	if a.Env.InheritAll {
		return nil
	}
	env := buildEnv(a)
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if k, _, ok := strings.Cut(kv, "="); ok {
			out = append(out, k)
		}
	}
	return out
}
