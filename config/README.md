# fixpoint config bundle

A bundle is a directory holding task configs at its root plus two
subdirectories, so one bare name resolves in one category:

    full-review.yaml       a task config, referenced as `full-review`
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

| Config | What it does |
|---|---|
| `review-only` | One review round, no edits. The safe starting point. |
| `full-review` | The full review → fix → commit loop. Needs `-trusted-target`. |
| `pr-review` | Review a GitHub pull request; review-only by default. |
| `defaults` | Shared base — not runnable on its own; `fixpoint --list` marks it as such. Inherit it with `extends: defaults`. |

`extends` is per-key and **one level deep**: keys the task config sets win, keys
it omits are inherited, and a **list it sets replaces** the inherited list rather
than appending. Replacement is deliberate — appending would make an inherited
entry impossible to remove — which is why the credential patterns that must never
be dropped live in fixpoint's code instead of in `target.exclude`.

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

**The same applies to the environment.** Agents inherit fixpoint's full
environment, so an injected reviewer can quote `ANTHROPIC_API_KEY`,
`GITHUB_TOKEN`, cloud credentials, or a database password into a finding — which
is logged and, in a fix run, echoed into a commit body. A container doesn't help
here: the agents' own API tokens must live in that environment for the CLIs to
work.

**fixpoint removes credential-shaped paths from collection unconditionally**, in
code, whatever `target.exclude` says (`.env*`, `*.pem`, `*.key`, `id_rsa`,
`.netrc`, and similar). That bounds the blast radius; it is not a sandbox.

**Fix rounds are fail-closed.** The coder edits files with permission checks
disabled and is not confined to the target, so a prompt-injection payload in any
reviewed file could steer it. Fixes therefore require `-trusted-target` (or
`-allow-untrusted-fix` in `pr` mode) *per invocation* — never an inherited
default. Review each round's commit before pushing.

**Artifacts can contain secrets.** The reviewed material, raw agent output, and
reviewer-authored text are all persisted. Everything passes through a best-effort
credential redactor first, but redaction is heuristic. The run directory being
owner-only (0700/0600), and keeping it out of any sync, backup, or commit, are the
real controls — add `.fixpoint/` to the project's `.gitignore`.

**`prompt_via: arg` exposes the prompt on the process argument list**, readable by
other local users via `ps` or `/proc`. Prefer `stdin`; fixpoint warns at run start
for any agent using `arg`.
