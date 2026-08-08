# Agents

What an agent is to fixpoint, how to add one, why the route matters, and what environment it runs with.

[← back to the README](../README.md)

## Agents

An agent definition is a provider-agnostic command template:

```yaml
agents:
  claude:
    model: opus
    effort: high
    command:
      - claude
      - -p                    # read-only: no permission-skip flag, edits are denied
      - --model {{model}}
      - --effort {{effort}}
    prompt_via: stdin         # stdin | arg
    timeout: 10m
    can_edit: false
```

The orchestrator wraps the prompt with role instructions plus a required JSON
output schema, then extracts the tagged JSON envelope from whatever the agent
prints — which is why any CLI (`claude`, `codex`, `ollama launch`, ...) works
without special-casing. `{{model}}` and `{{effort}}` are injected into the
command; a command token whose placeholder resolves to empty is dropped whole,
so `effort` simply disappears for providers that don't support it.

`can_edit` must reflect what the command actually permits: reviewer agents run
in enforced read-only modes (`can_edit: false`); only agents whose command
allows writing files (`can_edit: true`) may be assigned as `roles.coder`. Part
of that is machine-checked: `can_edit: false` is rejected outright for a command
carrying a flag that grants the write tools — either the full permission bypass
(`--dangerously-skip-permissions`,
`--dangerously-bypass-approvals-and-sandbox`, `--yolo`, `--full-auto`,
`--permission-mode bypassPermissions`) or a mode that auto-approves the edits
alone (`--permission-mode acceptEdits`, `--sandbox workspace-write`,
`--sandbox danger-full-access`, `--mode accept-edits`,
`--approval-mode auto_edit`) — because a reviewer's read-only claim is the only
barrier left when the code under review turns out to be prompt-injected.

### Borrowing an agentic harness for a model that has no CLI

A reviewer has to *explore* the repository, not just answer from a prompt, so a
raw chat endpoint is not enough on its own. Most models don't ship a CLI of their
own — but any Anthropic-compatible endpoint can borrow Claude Code's harness,
which is how the shipped ollama agents work (`ollama launch claude`) and how
`config/agents/qwen-openrouter.yaml` reaches OpenRouter with nothing but a base URL:

```yaml
command: [claude, --model {{model}}, --output-format json, -p]
env:
  pass: [ANTHROPIC_AUTH_TOKEN]                  # must hold your OpenRouter key
  set:  {ANTHROPIC_BASE_URL: https://openrouter.ai/api}
```

Three things that cost an afternoon to learn:

- **The base URL must not end in `/v1`.** The harness appends `/v1/messages`
  itself, so `.../api/v1` becomes `.../api/v1/v1/messages` and every call 404s
  with *"the selected model may not exist or you may not have access to it"* —
  which reads like a model or entitlement problem and is neither.
- **The credential goes in `ANTHROPIC_AUTH_TOKEN`,** because `env.set` takes
  literal values and cannot forward one variable into another. `ANTHROPIC_API_KEY`
  must *not* be passed: it takes precedence, and the run then quietly bills
  Anthropic for a model you meant to buy elsewhere. Not declaring it is enough —
  the environment filter above keeps it out of the process.
- **Don't read the cost these agents report.** The harness prices every model it
  serves against Anthropic's rate card and labels the provider `firstParty`,
  so the number is a rate-card calculation rather than a bill. Measured on the
  probe that added the OpenRouter agent: the harness claimed $0.599 for two
  sessions that cost $0.0064 on the OpenRouter key, overstating by ~93×. Leave
  `cost_usd` unset and the scoreboard prints `-` instead of money nobody was
  charged; tokens are real either way and still counted.

`codex` is a second route to OpenRouter, and a working one. It needs
`wire_api = "responses"` — 0.146 dropped `"chat"` — plus a provider block:

```
-c model_provider=openrouter
-c 'model_providers.openrouter.base_url="https://openrouter.ai/api/v1"'
-c 'model_providers.openrouter.env_key="OPENROUTER_API_KEY"'
-c 'model_providers.openrouter.wire_api="responses"'
```

Verified: it explores with tools, answers correctly, and reports
`cached_input_tokens`, so caching survives that route too.

*(An earlier version of this file said OpenRouter does not serve the Responses
API. That was wrong — the probe behind it used an expired key, and the resulting
401 "User not found" was read as a missing endpoint. With a valid key
`POST /api/v1/responses` returns 200.)*

Which harness suits a non-Anthropic model is a separate question from which route
reaches it, and OpenRouter's own docs note that Claude Code is tuned for Anthropic
models. Measured here over every run so far, the gap is narrow but real — and it
shows up in output-contract compliance rather than in error rate:

| agent | sessions | errors | error rate | of which schema |
|---|---|---|---|---|
| kimi | 24 | 4 | 17% | **3** |
| claude | 43 | 7 | 16% | 0 |
| codex | 38 | 5 | 13% | 0 |
| glm | 33 | 4 | 12% | 1 |

Every schema failure in the project's history — a missing `<review>` block, a
string where the contract wants an int — came from a non-Anthropic model driving
the claude harness. Yield is a different matter: kimi was the panel's top producer
in one run. Adding an agent is one file, so comparing harnesses is a config
experiment, not a migration.

### The route is part of the agent's identity

Two endpoints can both be Anthropic-compatible, accept the same request, return a
valid answer — and behave so differently that the same model is a different agent
through each. That is why the shipped bundle names agents by route
(`kimi-ollama`, `kimi-openrouter`) rather than by model alone.

**ollama silently drops `cache_control`.** In an agentic loop that is the dominant
cost, because every turn re-pays for the whole conversation so far and a reviewer
averages ~47 turns. Measured over one `fix-branch` run:

| agent | sessions | fresh input | cache read | cache rate |
|---|---|---|---|---|
| kimi (ollama) | 5 | 29.0M | 0.0M | **0%** |
| glm (ollama) | 4 | 12.6M | 0.0M | **0%** |
| claude | 5 | 0.4M | 31.1M | 99% |

Two agents, a third of the sessions, **81% of the run's fresh input**.

It is not a client misconfiguration. The same body carrying the same
`cache_control`, sent to each proxy twice:

```
ollama      1st: in=5418  cacheWrite=-     cacheRead=-
            2nd: in=5418  cacheWrite=-     cacheRead=-

OpenRouter  1st: in=16    cacheWrite=5604  cacheRead=0
            2nd: in=16    cacheWrite=0     cacheRead=5604
```

Verified end to end through the harness afterwards: with the cache warm,
`kimi-openrouter` went from 25,593 fresh input tokens to **687** on the second
call (99.1% cached).

**And it did not fix the bill, which is the more useful half of the result.** At a
98% hit rate one kimi session still cost ~$5, because it took 168 turns to
claude's 37 on the same round: caching cut the price per token 4.9×, and the
volume — the thing actually wrong — was untouched. So the shipped panel runs
`kimi-ollama` after all. Prepaid quota absorbs that volume where a card bills it,
and the quota ceiling is the price of the choice; it ended two runs early, which
is why glm left the panel rather than moving to a cheaper route. Diagnose the
resource before optimising it: this looked like a caching problem for a day and
was a convergence problem all along.

**ollama ignores `thinking.budget_tokens` the same way** — see
[effort (claude)](../config/agents/_README.md) — so `effort` on an ollama agent
validates, runs, and does nothing.

OpenRouter is not the answer to that one either, which is why no shipped agent
outside `claude`/`codex` sets `effort`. It advertises reasoning support for both
models, and the numbers do move — but not as a cap and not in one direction: a
1024 budget produced ~2327 thinking tokens from `kimi-k2.7-code` and ~3121 from
`glm-5.2`, and a 6000 budget produced *less* than a 1024 one. The likely cause is
translation: OpenRouter's native control is an OpenAI-style `reasoning` field,
while the harness sends Anthropic's `thinking`. A knob that moves unpredictably is
worse than none, so the bundle leaves it unset on both routes.

Practically: for one-shot prompts none of this matters. For anything agentic it
decides the bill, and for a prepaid endpoint it decides whether a long run
finishes at all — uncached volume is what exhausted a session limit mid-run twice
here.

The panel nevertheless runs the **`-ollama`** files, and the `-openrouter` ones are
the fallback rather than the other way round: caching cut the price per token 4.9×
and left the bill unchanged, because the volume is the defect. One 98%-cached kimi
session still ran ~$5. Prepaid quota absorbs that volume where a card bills it, so
the cheap route for a model that will not converge is the prepaid one, and its
ceiling is the price of the choice.

> **This is measured behaviour as of August 2026, not a documented contract.**
> Both proxies are free to change: ollama may add cache and budget support, and
> OpenRouter's caching varies by upstream provider. Re-measure before relying on
> either — POST the same request twice with a `cache_control` breakpoint on a
> large system block and compare `usage`, which is the whole experiment. If a run's
> scoreboard shows a 0% cache rate for an agent that should be caching, the route
> stopped working.

### The agent environment is filtered

An agent process gets a **non-secret baseline** — `PATH`, `HOME`, temp dir, locale,
TLS trust settings, proxy settings — plus only the variables its own file declares:

```yaml
env:
  pass: [ANTHROPIC_API_KEY]   # inherited from fixpoint's environment, if set
  set:  {NO_COLOR: "1"}       # literal values; override anything inherited
```

Everything else in fixpoint's environment — your `GITHUB_TOKEN`, cloud
credentials, database passwords — is **absent from the process**. That matters
because the environment is the one exfiltration surface a container does *not*
close: the agents' own credentials have to be inside the container for the CLIs to
work at all. A reviewer runs in a mode that denies *edits*, not *reads*; on Linux a
process can read its own `/proc/self/environ`, and any CLI with a shell tool can
just run `env`. Redaction only masks fixed-shape tokens like `ghp_…`, so a bare
database password would otherwise pass straight through into a finding, the logs,
and — in a fix run — the commit body.

fixpoint doesn't need to know what each CLI requires, which is what made this
tractable despite being provider-agnostic: the agent's file declares it, and
whoever wrote its `command` is exactly who knows. A name that isn't set in
fixpoint's environment is simply absent, not an error, since `claude` and `codex`
read credentials from `~/.claude` and `~/.codex` when you've logged in
interactively — in that case a reviewer runs with no credential in its environment
at all.

What this does **not** fix: an agent authenticating *via* an environment variable
must still be given it, so its own credential stays reachable by the process that
needs it. The win is everything else — your GitHub token is no longer inside the
code reviewer. `env.inherit_all: true` opts back out entirely for a CLI whose
requirements you don't know; fixpoint warns at run start when an agent does — and
**refuses to run** when the agent was declared inside the target, whatever you
asserted on the command line. `-trusted-bundle`/`-trusted-target` say the target's
policy may be
executed, not that it may help itself to secrets it cannot even name; declare
those under `env.pass`, or keep the agent in a bundle outside the target.

The `verify` commands are filtered too, from the other direction. They are argv the
*target* can supply, so running them with fixpoint's whole environment would hand a
`curl $ANTHROPIC_API_KEY` "build" command every credential the agents deliberately
do not share. They inherit fixpoint's environment **minus** every variable that
carries a credential — not just the agents'. That means the names your agent files
declare (`env.pass`, `env.set`), a few exact names whose value is auth material or
a live connection to it (`KUBECONFIG`, `NETRC`, `DOCKER_AUTH_CONFIG`,
`SSH_AUTH_SOCK`, `SSH_AGENT_PID`, `GPG_AGENT_INFO` — an inherited
ssh-agent would let a build command authenticate as you without ever reading a
key), and every variable whose name is
credential-*shaped*: one whose underscore-separated words include `TOKEN`,
`SECRET`, `PASSWORD`, `PASSWD`, `PASSPHRASE`, `CREDENTIAL(S)`, `API_KEY`,
`ACCESS_KEY`, `SECRET_KEY`, `PRIVATE_KEY` or `SIGNING_KEY`. Matching the shape
rather than a roster of vendor names is what covers `ANTHROPIC_API_KEY` and the
release tokens of the CI job you ran fixpoint from (`NPM_TOKEN`,
`DOCKER_PASSWORD`, `PYPI_TOKEN`, `SONAR_TOKEN`, `GPG_PASSPHRASE`,
`GOOGLE_APPLICATION_CREDENTIALS`, ...) and your own `ACME_INTERNAL_TOKEN`, none of
which a build check needs to see. Words match on underscore boundaries, so
`GIT_AUTHOR_NAME` and `TOKENIZERS_PARALLELISM` are untouched. `PASSWORD`,
`PASSPHRASE` and `PASSFILE` are the exception: they also match with a vendor
prefix run straight into them, because `PGPASSWORD`, `PGPASSFILE` and `MYSQL_PWD`
are how the database clients spell it and a boundary rule would miss all three.

A variable is also removed when its **value** is a credential-carrying connection
string — `scheme://user:pass@host`, or a `password=` / `Pwd=` keyword inside a
libpq, JDBC or ODBC DSN. Connection strings are named after the service rather than
the secret (`DATABASE_URL`, `MONGODB_URI`, `CELERY_BROKER_URL`, `SENTRY_DSN`), so no
name rule can see them, and stripping `*_URL` by name instead would take
`SONAR_HOST_URL` and every other endpoint setting with it. Reading the value splits
the class where the risk actually is: `DATABASE_URL=postgres://db/app` survives,
`DATABASE_URL=postgres://user:pass@db` does not. An authenticated proxy
(`HTTPS_PROXY=http://user:pass@proxy`) is a credential by the same reading and goes
too, which costs a verify command its network access unless you keep it back
deliberately.

What this removes is what the *environment* carries. A verify command still runs
as you, so a credential file it can name by path — `~/.netrc`, `~/.gnupg`,
`~/.aws/credentials` — stays readable; `KUBECONFIG` and `NETRC` are pointers, and
dropping a pointer is not the same as revoking access. `GNUPGHOME` is deliberately
*not* on the list for exactly that reason: stripping it would send `gpg` from
whatever scratch directory you isolated fixpoint with back to your real `~/.gnupg`,
live agent included — the opposite of what you asked for. Isolation at that level
wants a container, not an environment filter.

Set `FIXPOINT_STRIP_ENV` to name extra variables to remove, and `FIXPOINT_KEEP_ENV`
to spare one the shape rule caught but a check really needs (both take a comma- or
space-separated list). They are environment variables and deliberately not config
keys: a bundle inside the target shadows yours, so a keep list in YAML would let
the reviewed repository hand itself these secrets — which is also why
`FIXPOINT_KEEP_ENV` cannot override an explicitly denied name.

That direction is a denylist rather than an allowlist on purpose: what a
build actually needs is language- and project-specific (`GOFLAGS`, `JAVA_HOME`,
`CARGO_HOME`, `VIRTUAL_ENV`, ...), and an allowlist would silently break checks by
dropping what it forgot.
