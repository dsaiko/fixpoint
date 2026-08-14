package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

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
//
// Names here are matched EXACTLY, and every entry is one the shape rule below
// would not catch on its own: the products whose credential variable is not
// spelled with a credential word (KUBECONFIG, NETRC, DOCKER_AUTH_CONFIG all name a
// file or a blob of auth material). The vendor keys that do say TOKEN, SECRET or
// API_KEY are deliberately NOT enumerated here -- credentialNameRE covers those,
// and a hand-maintained roster of vendor names is exactly the list that is one
// release behind whatever the operator actually has exported.
//
// The agent sockets belong here for the same reason, and are the sharpest case:
// SSH_AUTH_SOCK is not a key, it is a live connection to one. A verify command
// that inherits it authenticates as the operator to every host the agent holds a
// key for (`ssh deploy@prod`, `git push`) without reading a key file at all --
// exactly the capability this filter is documented as removing -- and the recipe
// behind the operator's `make lint` is the target's Makefile, which a fix run has
// by definition just changed. GPG_AGENT_INFO is the signing equivalent.
//
// GNUPGHOME is deliberately ABSENT, and the difference from SSH_AUTH_SOCK is the
// rule for the whole class. SSH_AUTH_SOCK is the ONLY handle to the agent, so
// deleting the name deletes the capability. GNUPGHOME is a REDIRECT away from
// ~/.gnupg, which HOME -- a baseline variable, kept -- leaves reachable either
// way: deleting it does not take a keyring away, it points gpg back at the
// operator's real one, live agent socket and cached passphrases included. An
// operator who isolated fixpoint with GNUPGHOME=/tmp/scratch-gnupg would get the
// exact inverse of what they asked for, so this list leaves their redirect alone.
//
// KUBECONFIG and NETRC are redirects too, and stay only because the balance runs
// the other way for them: ~/.kube/config and ~/.netrc are commonly absent, so
// dropping the pointer usually does remove reach, whereas ~/.gnupg is populated on
// any machine that has ever run gpg. Neither entry should be read as DENYING
// access to the default file -- a verify command runs as the operator and can open
// it by path regardless. Only a sandbox stops that, and this filter is not one; it
// removes what the ENVIRONMENT carries.
var knownCredentialEnv = []string{
	"KUBECONFIG", "NETRC", "DOCKER_AUTH_CONFIG",
	"SSH_AUTH_SOCK", "SSH_AGENT_PID", "GPG_AGENT_INFO",
}

// credentialNameRE matches a variable name that is credential-SHAPED, whichever
// product it belongs to: it is what makes the gate cover NPM_TOKEN,
// DOCKER_PASSWORD, PYPI_TOKEN, TWINE_PASSWORD, SONAR_TOKEN, GPG_PASSPHRASE,
// GOOGLE_APPLICATION_CREDENTIALS, AZURE_CLIENT_SECRET and an in-house
// ACME_INTERNAL_TOKEN without knowing any of them by name. It also covers every
// agent credential the old hardcoded roster listed (ANTHROPIC_API_KEY,
// GITHUB_TOKEN, AWS_SECRET_ACCESS_KEY, ...), which is the evidence that the shape
// rather than the vendor is the thing worth matching.
//
// Words match on underscore boundaries, NOT as substrings. A substring rule
// looks equivalent and is not: `AUTH` occurs inside GIT_AUTHOR_NAME, `PASS`
// inside COMPASS_URL, `TOKEN` inside TOKENIZERS_PARALLELISM -- stripping those
// breaks commits and builds for no security gain. For the same reason bare KEY is
// absent (SSH_KEY_ALGORITHMS, KEYCHAIN, ...) while the compounds that only ever
// name a secret are present. AUTH is absent entirely: AUTH_TOKEN already matches
// on TOKEN, and the standalone word appears in too much non-secret configuration.
//
// The second branch is the exception the first one needs, and it exists because
// PGPASSWORD does not have an underscore in it. libpq (psql, pg_dump, every
// Postgres client) reads PGPASSWORD and PGPASSFILE, MySQL reads MYSQL_PWD, and a
// leading-boundary rule matches none of them while stripping DOCKER_PASSWORD from
// the same environment -- a gap that is invisible precisely because every name an
// operator thinks to check is underscore-separated. So PASSWORD, PASSPHRASE and
// PASSFILE match with a vendor prefix run straight into them: they are long enough
// that no ordinary variable contains one by accident, which is exactly what is not
// true of the short words above. PWD is the counter-example that keeps this branch
// honest -- it needs the LEADING underscore, or the filter would eat PWD and
// OLDPWD, the working directory every shell exports.
//
// This over-strips by design where the two conflict -- a check that fails because
// its credential is gone says so loudly, whereas a leaked credential says nothing
// at all -- and FIXPOINT_KEEP_ENV is the operator's escape hatch for the cases
// where a check legitimately needs one (a private-registry NPM_TOKEN, say).
var credentialNameRE = regexp.MustCompile(`(?i)(?:` +
	`(?:^|_)(?:api_?keys?|access_keys?|secret_keys?|private_keys?|signing_keys?|tokens?|secrets?|passwd|credentials?)(?:$|_)` +
	`|(?:passwords?|passphrases?|passfiles?)(?:$|_)` +
	`|_pwd(?:$|_)` +
	`)`)

// credentialValueRE matches a VALUE that carries a credential whatever its
// variable is called, and it exists because the largest class of secret-bearing
// variables is named after the service rather than the secret: DATABASE_URL,
// MONGODB_URI, REDIS_URL, CELERY_BROKER_URL, SENTRY_DSN. credentialNameRE cannot
// see those -- `postgres://user:pass@db` is a password in a variable whose name
// says "url" -- so without this rule a target-supplied verify command inherits
// every connection string the operator (or their CI job) exported.
//
// It matches the value's SHAPE, not the name's, and that is what makes it safe to
// be this broad. The obvious alternative -- adding *_URL / *_URI / *_DSN to
// credentialNameRE -- strips by name and so eats SONAR_HOST_URL, CI_API_V4_URL and
// COMPASS_URL, ordinary endpoint configuration that carries nothing and whose
// removal breaks a check for no gain. Keying on the value instead splits the class
// exactly where the risk is: DATABASE_URL=postgres://db/app survives,
// DATABASE_URL=postgres://user:pass@db does not.
//
// Two shapes, because a connection string has two spellings:
//
//   - URI userinfo (scheme://user:pass@host). The user is optional and the
//     password must be non-empty, so `redis://cache:6379` -- host and PORT, not
//     user and password -- does not match: a port follows the HOST, with no `@`
//     after it. This is the same shape redactSecrets masks in agent output.
//   - A `password=` / `pwd=` keyword inside the value, which is how libpq
//     ("host=db user=u password=s"), JDBC ("...?user=u&password=s") and ODBC
//     ("...;Uid=u;Pwd=s;") spell a DSN. Only the long unambiguous words plus ODBC's
//     PWD, and each needs a non-space value after it, so a variable that merely
//     mentions the word is left alone.
//
// A value match is rescueable with FIXPOINT_KEEP_ENV exactly like a name match --
// a test suite that genuinely needs its DATABASE_URL is the expected case, and it
// fails loudly with the variable absent. The one match an operator is likely to
// hit without having exported a secret on purpose is an authenticated proxy
// (HTTPS_PROXY=http://user:pass@proxy): it is a credential by the same reading, so
// a verify command loses network unless they keep it back deliberately.
var credentialValueRE = regexp.MustCompile(`(?i)(?:` +
	`[a-z][a-z0-9+.\-]*://[^\s:/@]*:[^\s:/@]+@` +
	`|(?:^|[^a-z0-9_])(?:password|passwd|pwd)\s*=\s*\S` +
	`)`)

// stripEnvVar and keepEnvVar let the OPERATOR adjust the gate for their own
// environment: FIXPOINT_STRIP_ENV names extra variables to remove (a bespoke
// credential whose name says nothing about being one), FIXPOINT_KEEP_ENV names
// variables to spare from credentialNameRE (a check that genuinely needs one).
// Both take a list separated by commas or whitespace.
//
// They are environment variables and NOT config keys, for the same reason
// loop.trusted_target is flag-only: the FIRST bundle search location is inside the
// target, so a `verify.strip_env` key would be one the reviewed repository could
// shadow -- and a keep list in YAML would be strictly worse, letting a hostile
// bundle write `keep_env: [ANTHROPIC_API_KEY]` and hand itself every secret this
// function exists to withhold. The invoking environment is a channel the target
// cannot write to.
//
// Neither list can rescue an EXPLICIT denial (knownCredentialEnv, an agent's
// env.pass/env.set, or FIXPOINT_STRIP_ENV itself). Keeping the hard guarantee --
// a verify command never sees a credential fixpoint knows an agent authenticates
// with -- unconditional means a typo in an operator's keep list cannot quietly
// reopen the exfiltration path this gate closes.
const (
	stripEnvVar = "FIXPOINT_STRIP_ENV"
	keepEnvVar  = "FIXPOINT_KEEP_ENV"
)

// envNameList splits one of the operator's lists on commas or whitespace. Empty
// entries are dropped, so a trailing comma is not a variable named "".
func envNameList(v string) []string {
	return strings.FieldsFunc(v, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
}

// EnvWithoutCredentials returns fixpoint's own environment minus every variable
// that carries a credential: the names the configured agents declare (env.pass
// and env.set), knownCredentialEnv, the operator's FIXPOINT_STRIP_ENV list, any
// name whose SHAPE says credential (credentialNameRE) and any VALUE whose shape
// does (credentialValueRE, the connection strings whose name says only which
// service they point at), except the names the operator spared in
// FIXPOINT_KEEP_ENV.
//
// It exists because the verify gate runs argv the TARGET supplies (a bundle file
// inside the target shadows the operator's), and inheriting fixpoint's whole
// environment there would hand those commands exactly the secrets buildEnv keeps
// away from the agents themselves -- a `curl $ANTHROPIC_API_KEY` verify command
// would exfiltrate every agent credential before a coder edits anything. The same
// command can read any OTHER secret in that environment just as easily, which is
// why the filter is not limited to the credentials fixpoint itself uses: an
// operator running fixpoint from CI has the release and registry tokens of that
// job exported too, and none of them are anything a build check needs to see.
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
	// fixpoint's own knobs go too: they configure this filter, and a verify
	// command has no use for the list of what was withheld from it.
	deny[stripEnvVar] = true
	deny[keepEnvVar] = true
	for _, name := range envNameList(os.Getenv(stripEnvVar)) {
		deny[name] = true
	}
	keep := map[string]bool{}
	for _, name := range envNameList(os.Getenv(keepEnvVar)) {
		keep[name] = true
	}

	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		// An entry with no "=" cannot be attributed to a name; pass it through
		// rather than guess, exactly as before.
		if ok && (deny[k] || (!keep[k] && (credentialNameRE.MatchString(k) || credentialValueRE.MatchString(v)))) {
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
// reporting which prompt drove it. The GIT_CONFIG_* pins Run adds on top
// (internal/gitenv) are deliberately absent: they are fixpoint's own hardening, not
// something the agent's declaration exposed, and listing four of them per run would
// bury the names this log exists to show.
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

// forgeCredentialEnv is what a forge CLI needs in order to be a forge CLI: the
// token it authenticates with, and the host/endpoint settings that say where.
//
// EnvWithoutCredentials strips these along with every other credential-shaped
// name, which is right for a git plumbing probe and wrong for `gh` -- a `gh pr
// checkout` with no token cannot check anything out. On a machine where gh
// authenticates from the environment rather than from ~/.config/gh (a container,
// CI, or any setup that exports GITHUB_TOKEN), stripping it turned every pr-mode
// run into "To get started with GitHub CLI, please run: gh auth login".
var forgeCredentialEnv = []string{
	"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN",
	"GH_HOST", "GH_CONFIG_DIR",
	"GITLAB_TOKEN", "GL_TOKEN", "GITLAB_HOST", "GLAB_CONFIG_DIR",
}

// WithForgeCredentials puts the forge CLI's own credentials back into an
// environment EnvWithoutCredentials stripped, for the narrow case of running
// that CLI.
//
// Deliberately narrow. The credential a tool needs to do its job is not the same
// thing as the credentials it must not be handed: `gh` gets the GitHub token
// because every `gh` command is an authenticated GitHub call, and it still does
// not get ANTHROPIC_API_KEY, AWS keys, or the agent tokens -- which is the whole
// point of running it under a filtered environment rather than the process's own
// (see forge.run, which inherits everything and should not).
//
// Values come from the process environment, so a name absent there stays absent
// here; nothing is invented.
func WithForgeCredentials(env []string) []string {
	have := make(map[string]bool, len(env))
	for _, kv := range env {
		if k, _, ok := strings.Cut(kv, "="); ok {
			have[k] = true
		}
	}
	out := env
	for _, name := range forgeCredentialEnv {
		if have[name] {
			continue
		}
		if v, ok := os.LookupEnv(name); ok {
			out = append(out, name+"="+v)
		}
	}
	return out
}

// IsForgeCLI reports whether a command name is a forge client whose own
// credentials WithForgeCredentials must restore.
func IsForgeCLI(name string) bool {
	switch filepath.Base(name) {
	case "gh", "glab":
		return true
	}
	return false
}
