# fixpoint config bundle

A bundle is a directory holding task configs at its root plus two
subdirectories, so one bare name resolves in one category:

    fix-code.yaml          a task config, referenced as `fix-code`
    prompts/fix.md         a prompt, referenced as `fix`
    agents/claude.yaml     an agent, referenced as `claude`

Run one with `fixpoint <name>`; list what's available with `fixpoint --list`. A
config's optional one-line `description:` appears in that listing and in shell
completion (`fixpoint completion zsh|bash|fish`), so write it for whoever has to
choose between them.

## Where bundles are found

Searched in order, first match wins, **per file** — so a project can shadow one
prompt and inherit everything else:

1. `<project>/config/` — the project root is the git root, or the nearest
   ancestor holding a `config/` directory, walking up from where you ran fixpoint
2. `~/.fixpoint/`
3. the OS per-user config directory (`~/.config/fixpoint` on Linux)
4. the package-installed bundle (`/usr/share/fixpoint`, `/usr/local/share/fixpoint`,
   or `$(brew --prefix)/share/fixpoint`)

There is deliberately **no fallback compiled into the binary**. These files decide
what agents are told to do and which trust gates apply, so "what will this do to
my repository?" must be answerable by reading files on disk. If nothing resolves,
fixpoint refuses to run and prints every location it searched.

Every run logs the file each name resolved to, and records it in the run summary.
Read that first when behavior surprises you — a shadowing copy is the usual cause.

Customize by **copying**, never by editing the installed bundle: package upgrades
replace `/usr/share/fixpoint` wholesale. A copy in `~/.fixpoint/` or the project
shadows it and survives upgrades — but note that it also pins you to the schema of
the fixpoint version you copied from, and a removed field becomes a hard error on
a later upgrade rather than being ignored.

## Shipped configs

Names are `<verb>-<scope>`. The **verb comes first because it is the safety
property**: a `review-` config never invokes the coder and so never modifies a
file, while a `fix-` config edits your working tree and needs an explicit trust
assertion on the command line. Naming that way puts the consequential half of the
name where you read it first, and keeps the two groups apart in `fixpoint --list`
and in shell completion. The scope says what is examined.

| Config | What it does |
|---|---|
| `review-code` | Review a whole project once, no edits. Needs `-trusted-target` if the project ships its own bundle. |
| `review-pr` | Review a GitHub pull request; review-only by default. |
| `fix-code` | Review → fix → verify → commit loop over a whole project. Needs `-trusted-target`. |
| `defaults` | Shared base — not runnable on its own; `fixpoint --list` marks it as such. Inherit it with `extends: defaults`. |

Keep new configs in the scheme: `fix-tests`, `review-design`, `fix-design`. A
`fix-` and a `review-` config over the same scope should share a lens panel, so
the read-only one previews what the fixing one would hand the coder.

`extends` is per-key and **one level deep**: keys the task config sets win, keys
it omits are inherited, and a **list it sets replaces** the inherited list rather
than appending. Replacement is deliberate — appending would make an inherited
entry impossible to remove — which is why the credential patterns that must never
be dropped live in fixpoint's code instead of in `target.exclude`.

## Per-lens modifiers

A `roles.review.prompts` entry may be a bare prompt name or a mapping carrying
`agent`, `advisory: true` (reported for a human, never fixed, never gates
convergence), `once: true` (round 1 only), or `final: true` (held out of the loop
and run in a closing round after it — on every agent unless pinned, with its
findings still fixed, repeating until nothing is left to fix — `review-tests` uses
this). `once` and `final` are mutually exclusive. See the
per-lens modifiers section in the [root README](../README.md) for when to reach for
each.

## Reporting what a run cost

An agent file may declare where its CLI reports token usage and cost, and fixpoint
puts those numbers in the end-of-run scoreboard:

```yaml
command: [claude, -p, --output-format json, --model "{{model}}"]
usage:
  format: json                                   # json | jsonl
  text: result                                   # where the agent's reply lives
  input_tokens: modelUsage.*.inputTokens         # `*` matches every key at that
  output_tokens: modelUsage.*.outputTokens       # level — the map is keyed by
  cache_read_tokens: modelUsage.*.cacheReadInputTokens   # model id, so it cannot
  cache_write_tokens: modelUsage.*.cacheCreationInputTokens  # be written literally
  cost_usd: modelUsage.*.costUSD                 # omit when the CLI reports none
```

`text` is **required** whenever `format` is set: fixpoint swaps the envelope for
the reply before extracting the output contract, so machine-readable mode stays
invisible to everything downstream. Omit it and every round fails to parse;
validation refuses the config rather than letting you find out at runtime.

Paths are dotted, with `*` matching every key at a level. Within one object a
wildcard's matches are **summed** (a session that used two models spent both);
across the lines of a `jsonl` stream the **last** value wins (successive lines are
the same session's running total, so adding them would double-count).

Parsing fails open — a CLI that crashed or changed its output shape yields
unparseable output, and the round's findings matter more than its accounting, so
the raw output passes through and usage is left empty.

This is configuration rather than code because fixpoint is provider-agnostic:
which flag switches a CLI to machine-readable output, and where the numbers sit in
it, is exactly what this file already exists to record. Measuring at fixpoint's own
boundary is not an alternative — the bytes it exchanges miss the agentic session in
between by orders of magnitude.

## Security

Read this before pointing fixpoint at code you didn't write.

**Reviewers can read anything, even in review-only mode.** The read-only flags in
`agents/*.yaml` (`claude -p` without the permission-skip flag, `codex --sandbox
read-only`, `agy --mode plan`) block *edits* but do not confine *reads*: fixpoint
sets only the working directory, with no filesystem sandbox. A reviewer fed
untrusted content can be prompt-injected into reading a host secret
(`~/.ssh/id_rsa`, `~/.aws/credentials`, a `.env`) and quoting it into a finding.
That path has no trust gate. Point reviewers at untrusted content only on a host
without sensitive files, or run fixpoint inside a container or VM.

**The environment is the other read surface**, and it is filtered rather than
inherited — see "The agent environment is filtered" below. What an agent can still
quote into a finding is its own declared credentials (an agent given
`ANTHROPIC_API_KEY` can leak that key), so the filtering bounds which secrets are
reachable, not whether a compromised reviewer can talk. A container does not help
with the remainder: the agents' own API tokens have to be inside it for the CLIs to
work at all.

**fixpoint removes credential-shaped paths from collection unconditionally**, in
code, whatever `target.exclude` says (`.env*`, `*.pem`, `*.key`, `id_rsa`,
`.netrc`, and similar). That bounds the blast radius; it is not a sandbox.

**The agent environment is filtered.** Each agent receives a non-secret baseline —
`PATH`, `HOME`, `TMPDIR`/`TMP`/`TEMP`, `LANG`/`LANGUAGE`/`LC_ALL`/`LC_CTYPE`,
`TERM`, `TZ`, `USER`/`LOGNAME`, the `XDG_*` config paths, TLS trust
(`SSL_CERT_FILE`, `SSL_CERT_DIR`, `NODE_EXTRA_CA_CERTS`, `CURL_CA_BUNDLE`,
`REQUESTS_CA_BUNDLE`), and proxy settings — plus the variables its own
`agents/*.yaml` declares under `env.pass` / `env.set`. Nothing else is present in
the process, so a prompt-injected reviewer cannot quote a secret it cannot see.

The baseline is chosen so that dropping something does not break a CLI in a way
that looks unrelated: without `HOME`, `claude` and `codex` cannot find their
credentials; without the TLS entries, HTTPS fails certificate verification behind a
corporate proxy. One caveat — proxy variables can embed credentials
(`http://user:pass@proxy`), so they are the single baseline entry that may carry a
secret; the redactor masks URI passwords, and an operator who cannot accept that
should unset them for fixpoint's own process.

`env.inherit_all: true` restores full inheritance for one agent, with a run-start
warning. It exists for a CLI whose requirements are unknown, at the cost of
re-exposing every exported secret to that agent.

**`verify` commands do not receive the agents' credentials.** They are argv the
target can supply (a bundle inside the target is searched first), so they inherit
fixpoint's environment *minus* every variable an agent file declares under
`env.pass` / `env.set` and minus a built-in credential list (`ANTHROPIC_API_KEY`,
`OPENAI_API_KEY`, `CODEX_API_KEY`, `GOOGLE_API_KEY`, `GEMINI_API_KEY`,
`GITHUB_TOKEN`, `GH_TOKEN`, and the `AWS_ACCESS_KEY_ID` /
`AWS_SECRET_ACCESS_KEY` / `AWS_SESSION_TOKEN` trio). Otherwise the gate would be a way around the
filtering above: a "build" command that curls a key out is not a model's
misbehavior, it is just argv. This direction is a denylist, because what a build
needs is project-specific and an allowlist would break checks by dropping what it
forgot.

**Fix rounds are fail-closed.** The coder edits files with permission checks
disabled and is not confined to the target, so a prompt-injection payload in any
reviewed file could steer it. Fixes therefore require `-trusted-target` (or
`-allow-untrusted-fix` in `pr` mode) *per invocation* — never an inherited
default. Review each round's commit before pushing.

**A config cannot grant trust.** Setting `loop.trusted_target` or
`loop.allow_untrusted_fix` in any config file is a hard load error, in the task
config and in a base it `extends`. This directory is searched *before* the
operator's own bundles, so it may be shipped by the repository under review: a
trust key here would let reviewed code authorize fixpoint to execute the agent
definitions it supplies and to run the write-capable coder against it — the exact
thing the two gates above exist to prevent.

**Artifacts can contain secrets.** The reviewed material, raw agent output, and
reviewer-authored text are all persisted. Everything passes through a best-effort
credential redactor first, but redaction is heuristic. The run directory being
owner-only (0700/0600), and keeping it out of any sync, backup, or commit, are the
real controls — add `.fixpoint/` to the project's `.gitignore`.

**`prompt_via: arg` exposes the prompt on the process argument list**, readable by
other local users via `ps` or `/proc`. Prefer `stdin`; fixpoint warns at run start
for any agent using `arg`.
