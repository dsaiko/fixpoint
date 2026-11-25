# Security model

What is trusted, what is not, and which gates are invocation-only.

[← back to the README](../README.md)

## Security model

Read this before pointing the tool at code you did not write.
[config/README.md](../config/README.md) and the comments in
[config/defaults.yaml](../config/defaults.yaml) cover each point in depth.

- **The coder edits files with permission checks disabled** and is not
  confined to `target.path`. A prompt-injection payload hidden in any reviewed
  file could steer it into writing elsewhere on your machine. Fix rounds are
  therefore **fail-closed**: they are refused unless you affirm trust in the
  invocation — `-trusted-target` for a directory or git-diff target, and in `pr`
  mode, where the reviewed code is by definition untrusted,
  `-allow-untrusted-fix`. Review each round commit before pushing.
- **No agent loads the target's `.claude/settings.json`.** Every shipped
  `claude`-backed agent — the coder included — passes `--setting-sources user`.
  A hook in that file is a shell command the CLI runs before the model takes a
  turn, so neither a reviewer's read-only flags nor the trust assertion a fix
  round makes covers it: it is execution with no model in the loop, and in `pr`
  mode the branch `gh pr checkout` writes into the worktree is the thing that
  supplied it. The same flag keeps the target's MCP servers, skills and
  `CLAUDE.md` out of the session. Agent commands you write yourself get no such
  treatment automatically.
- **No codex reviewer gets your codex MCP servers or ChatGPT apps.** Shipped
  codex agents pass `--ignore-user-config` and `-c features.apps=false`: the
  read-only sandbox confines the shell, not MCP tools, and the desktop app's
  `~/.codex/config.toml` can hand a session Computer Use, while the login itself
  brings apps that deploy Sites and share pages. The `ollama launch` agents can
  take only the second flag.
- **Trust is asserted on the command line only, never in a config file.**
  fixpoint refuses to load a config that sets `loop.trusted_target`,
  `loop.trusted_bundle`, `loop.allow_untrusted_fix` or `loop.agents_sandboxed`. Bundles are shadowable and `<project>/config` is
  searched *first*, so a config key would let the repository under review declare
  itself trustworthy — one line in a hostile repo's own config, authorizing both
  the execution of the agent definitions it ships and the write-capable coder,
  with no involvement from you. Keeping the assertion in the invocation is also
  what stops it becoming an inherited default that silently applies to the next
  untrusted repository you clone.
- **A bundle resolved from inside the target needs `-trusted-bundle`, even for a
  review-only run.** `<project>/config` is searched first, so a repository
  can ship the very files a run is built from: agent commands and verify commands
  are argv fixpoint executes, and a prompt is the instruction stream handed
  verbatim to a reviewer that can read anything you can. None of that needs a
  model's cooperation to exploit. fixpoint therefore lists every bundle file that
  came from inside the target and refuses until you assert trust — or point
  `-config` at a bundle outside it. The check is per *file*, not per key, so it
  cannot go stale as configuration grows new surface.
- **`-trusted-bundle` is deliberately narrower than `-trusted-target`.** Bundle
  files are resolved, and every prompt parsed, before anything replaces the
  worktree — in `pr` mode that means before `gh pr checkout`, so they are *your*
  commit's files. `-trusted-target` also clears that gate, but it says more: it
  permits fix rounds and, in `pr` mode, downgrades to warnings the two refusals
  below that cover content the checkout brings. So when the configuration is yours
  and the code is not — reviewing a fork's pull request with your own bundle, which
  is what `make review-pr` does — assert `-trusted-bundle` and nothing wider.
- **In `pr` mode an agent command may not resolve into the target.** An agent's
  `command` is argv fixpoint execs with that agent's declared credentials, and it
  runs with its working directory inside `target.path` — so a relative element
  (`./reviewer.sh`, the script in `[node, ./reviewer.cjs]`, or a path packed into
  an option as in `--require=./hook.js`) names a file the branch `gh pr checkout`
  writes, *after* validation confirmed the one that was there before. That is
  target-controlled code execution rather than the prompt
  injection the `pr` path is built to contain, and it needs no model's
  cooperation, so fixpoint refuses the combination at startup unless you assert
  `-trusted-target`/`-allow-untrusted-fix` (with the assertion it warns instead).
  Point such a command at a path outside the target.
  Every other mode keeps the target-relative form: nothing replaces the file
  between validation and the run.
- **In `pr` mode a bare command name may not be resolvable from inside the
  target.** A bare `command` names no path, so which file it runs is decided by
  `PATH` — re-read at every invocation, after the checkout. If any absolute `PATH`
  entry lies inside `target.path` (a repo-local `bin/` shim), the pull request can
  ship that executable, or shadow one found further down `PATH` by adding a file of
  the same name, and it runs as the agent process with the agent's credentials.
  Same gate as above: refused in `pr` mode, downgraded to a warning by
  `-trusted-target`/`-allow-untrusted-fix`. Drop the entry from `PATH`, or give the
  command an absolute path outside the target. Relative `PATH` entries (including
  the trailing-colon empty one) do not count — Go refuses to run a command resolved
  through one.
- **Reviewers can read anything, even in review-only mode.** Read-only agent
  flags block edits but do not confine reads: a reviewer fed untrusted content
  can be prompt-injected into reading a host secret (`~/.ssh`,
  `~/.aws/credentials`, `.env`) and quoting it into a finding. Point reviewers
  at untrusted content only on a host without sensitive files, or keep them
  inside [fixpoint's own wrapper](#fixpoints-own-wrapper--sandbox-builtin) — the
  default — or a [`sandbox.command`](#confining-agents-with-sandboxcommand) of
  yours, which is this advice made executable rather than left to the operator.
  This bullet describes a run with `-sandbox off`.
- **The process-group kill is the only containment, and a descendant can escape
  it.** Every agent, verify command and git subprocess runs as a process-group
  leader, and fixpoint SIGKILLs the whole group the moment the leader exits — so an
  MCP server, a test daemon or a plain `child &` in a wrapper script dies with the
  command rather than editing the repository during a later check. A child that
  calls `setsid`/`setpgrp` first is no longer *in* that group: it outlives the run
  with the working directory still inside the target and whatever environment its
  agent was given, free to keep reading files or writing to the repository after
  fixpoint believes the run has ended. fixpoint records it in the `.raw` log when
  such a descendant holds an output pipe open past the agent's exit, but one that
  closes its own descriptors first leaves no trace. Nothing underneath enforces
  termination — containing a deliberately daemonizing process needs an OS-level
  mechanism (a transient cgroup, a job object, a supervising container) that is not
  implemented *by fixpoint* — [`sandbox.command`](#confining-agents-with-sandboxcommand)
  is how you supply one. Run agent CLIs you do not trust inside a container/VM, and
  do not grant one `env.inherit_all`.
- **Logs can contain secrets.** The reviewed material, raw agent output, and
  reviewer-authored text are all persisted; redaction is heuristic, not a
  guarantee. Keep the logs directory out of any sync, backup, or commit (the
  run's own logs dir is always excluded from round commits automatically) — the
  owner-only permissions on it are the real access control, not the redactor. The
  built-in rules match credential *shapes* (`sk-ant-…`, `ghp_…`, JWTs,
  `password: …`, PEM blocks), so a site's own opaque token — an
  `x-internal-auth` header, a bearer string in a vendor format, a connection URL
  under a name that looks like nothing — matches none of them and is written
  verbatim. Add your own patterns under `logs.redact`:

  ```yaml
  logs:
    redact:
      - '(?i)(x-internal-auth\s*:\s*)\S+'  # capture group kept, rest masked
      - 'ACME-[A-Z0-9]{24}'                # no group: whole match masked
  ```

  They apply to every artifact and every log line, after the built-in rules, and
  are compiled at startup so a bad pattern fails the run rather than the write
  that was supposed to mask something.
- **One run owns the repository.** A fix run (and any `pr` run, which checks out a
  branch) takes an exclusive `flock` on a file in the target's git directory for its
  whole duration, and a second run on the same checkout is refused at preflight
  rather than queued. Every mutating decision the loop makes rests on a snapshot of
  the worktree — it is clean, it verifies, `git add -A` commits what the coder wrote
  — and two concurrent runs invalidate all three silently, producing commits nobody
  verified. Run concurrent fixpoints against separate checkouts (or worktrees).
  Review-only directory runs neither take the lock nor are blocked by one.
- **An approval outlives the commit it was given for.** Posting refuses a pull
  request that moved since it was reviewed, and binds the review to that commit
  (`commit_id` on GitHub, the `sha` on GitLab), so nothing is published about code
  nobody read. The *approval* is a statement about the pull request, though, not
  about a commit: both forges go on counting it toward the merge requirements after
  a push, unless the repository is configured to clear approvals on one. So an
  author can wait for `-post-verdict` to exit `0`, push, and merge on it. No client
  can close that — enable *Dismiss stale pull request approvals when new commits
  are pushed* (GitHub) or *Remove all approvals when commits are added to the source
  branch* (GitLab) before letting anything approve. fixpoint prints the same warning
  on every approval it publishes.
- **`prompt_via: arg` exposes the prompt on the process argument list**,
  readable by other local users via `ps`/`/proc`. Prefer `stdin` on shared
  hosts; fixpoint warns at run start.

## Confining agents with `sandbox.command`

Everything above is the model as it stands with agents launched directly. The one
switch that changes it is `sandbox.command`, which launches every agent through a
wrapper of your choosing:

```yaml
sandbox:
  command: [/usr/local/bin/fixpoint-sandbox, "--target", "{{target}}", "--mode", "{{target_mode}}", "--"]
```

The wrapper is prepended to the agent's argv, so it is what fixpoint execs and the
agent CLI is its argument. Three placeholders are expanded, per agent:

| Placeholder | Expands to |
|---|---|
| `{{target}}` | The absolute path of the directory the invocation works in: the target, or for create the assignment snapshot and for implement's coder the output directory. Expanded per invocation. |
| `{{target_mode}}` | `rw` for an agent declared `can_edit`, `ro` for every other — so a reviewer and the coder beside it get the same wrapper with different access. |
| `{{agent}}` | The agent's name, for a wrapper that keeps per-agent profiles. |

A placeholder fixpoint does not recognize is refused at startup rather than passed
through verbatim: a typo in a mount argument would confine the wrong path. A
wrapper that expands to nothing at all is refused for the sharper version of the
same reason — the run would be unconfined while the configuration said otherwise.

**fixpoint implements no sandbox primitive and does not try to.** Which primitive fits
(bubblewrap, a podman/docker container, `sandbox-exec`, a transient systemd scope)
and which paths must be mounted are properties of your machine, not of this
program. What fixpoint knows and your wrapper cannot is which target is under
review and whether this agent is supposed to be able to write to it, so that is
exactly what it passes.

### What it closes

- **The read surface.** Mount the target and the one credential directory the CLI
  needs, and nothing else: the prompt-injected reviewer two bullets above cannot
  read `~/.ssh` because `~/.ssh` is not there. The exfiltration hole this document
  calls unavoidable becomes a mount list.
- **The escaped descendant**, on a platform whose sandbox contains one. A
  container or a cgroup scope kills what `setsid` escapes from the process group.

### What it does not close

**It does not isolate the credential from the agent.** The agent CLI
authenticates as you, so its token has to be inside the sandbox with it. A
reviewer can still read the credential it was given; what it can no longer read is
every *other* secret on the host. That limit is a property of driving somebody
else's CLI, and no wrapper can lift it.

It also covers agents only. Verify commands are argv you wrote and git is
fixpoint's own; confining either would change what the deterministic gate
measures, so neither goes through the wrapper.

Create and implement runs are covered too. Their agents work outside
`target.path` — create's in the assignment snapshot, implement's coder in the
output directory — so `{{target}}` is expanded for each invocation from the
directory it actually runs in, rather than once from the config. (Until it was,
both pipelines were refused outright: a wrapper bound to one directory while the
agent wrote in another is a confinement configured, reported and aimed at the
wrong place.)

One deliberate exemption is worth knowing about: an argument naming the target
**root** (`--ro-bind {{target}}`) does not count as a path the reviewed material
supplied, because it is the checkout the run is about rather than a file the
branch can write. Anything *deeper* inside the target still does, and is still
refused in `pr` mode.

### fixpoint's own wrapper: `-sandbox builtin`

Writing a wrapper is the part of the above most operators will not do, so one
ships with fixpoint: `config/sandbox/fixpoint-sandbox`, installed in the bundle
beside the binary, and **it is on by default**. `-sandbox` (or
`FIXPOINT_SANDBOX` in your environment; the flag wins) picks one of:

| Value | Agents run inside |
|---|---|
| `auto` (default) | the bundle's own `sandbox.command` if it sets one, otherwise this wrapper |
| `builtin` | this wrapper, replacing any `sandbox.command` the bundle set |
| `off` | the bundle's `sandbox.command` if it sets one, otherwise **nothing** |

So a run is unconfined only when you said `off`. When the wrapper cannot run —
it is missing beside the binary, or the host has no `bwrap` — fixpoint refuses to
start rather than fall back, and says how to opt out. Every run logs one
`sandbox:` line naming what confines its agents, `NONE` included.

The choice is read from the invocation, never from a config file — a config can
be the reviewed repository's, and so can a bundle: the wrapper is looked up only
in the bundle locations tied to the binary (`<binary dir>/config`,
`<prefix>/share/fixpoint`), never in a project's `config/` or `~/.fixpoint`, which
the material under review can shadow.

What it does, per platform:

- **Linux, bubblewrap.** The root read-only; `/run` masked (the user bus and the
  Docker socket live there); `$HOME` an empty tmpfs with only the toolchain
  directories a CLI is installed in bound back; `/tmp` replaced by a private
  scratch directory; the working directory bound read-only, or writable for a
  `can_edit` agent; own PID and IPC namespaces, its own session, killed with the
  wrapper. Needs unprivileged user namespaces.
- **macOS, sandbox-exec.** File *contents* under `$HOME` unreadable except the
  same allowlist (names stay visible, which keeps path resolution working); writes
  denied everywhere except the scratch directory, a writable working directory and
  the CLI's state files; terminal ioctls denied.

**Trees outside `$HOME` are hidden the same way** when they are yours: the target
of every symlink in `$HOME`, down to depth 2, that leads out of it (`~/oddin ->
/tank/enc/oddin` puts every repository there out of reach, the reviewed one
excepted), and every path in `FIXPOINT_SANDBOX_HIDE` (colon-separated, from your
environment, never from a config) for what no symlink leads to. The working
directory, the toolchains, the agent's install directory and the CLI state are
bound back on top. Not hidden: a symlink target under a system root (`/usr`,
`/etc`, `/opt`, `/var`, ...), which holds nothing of yours and which the agent
needs. A `FIXPOINT_SANDBOX_HIDE` entry that does not exist, or that is or holds
the working directory, stops the agent before it starts.

CLI state is decided from the agent's binary name. **codex** gets a fresh
`CODEX_HOME` holding only a link to the real `auth.json`, and none at all on a
route with a `model_provider` of its own; when codex refreshes the login during
the session, the wrapper copies it back afterwards, but only if it names the same
`account_id` — a lens that wrote a login of its own choosing into its scratch must
not be able to sign you into that account. **claude** gets its real config
directory read-only with every transcript store emptied, and `.claude.json` and
`.credentials.json` writable because the CLI rewrites them. Any other CLI gets no
state; `FIXPOINT_SANDBOX_RO` / `FIXPOINT_SANDBOX_RW` (colon-separated paths from
your environment) bind what it needs.

It implies `-agents-sandboxed`, and unlike any other wrapper it is accepted in `pr`
mode: the refusal below exists because fixpoint cannot check that a wrapper honors
`{{target_mode}}`, and this one is fixpoint's own code, found where the reviewed
material cannot put a file, with its read-only mode under test
(`internal/config/builtin_wrapper_test.go` runs it through the host's real sandbox).
The shipped codex agents declare `when_sandboxed`, so their own sandbox is
switched off only inside it.

What it leaves open, beyond what every wrapper leaves open: the network; what is
readable outside `$HOME` that no symlink in `$HOME` leads to and
`FIXPOINT_SANDBOX_HIDE` does not name (on Linux what your user can read, read-only;
on macOS the same, plus nothing writable);
`.claude.json`, which can declare user-scope MCP servers, stays writable (the CLI
rewrites it on every start), so a lens could plant one that runs the next time you
start `claude`; and on Linux a custom `CLAUDE_CONFIG_DIR` puts `.claude.json` inside the
read-only config directory, which blocks claude's rewrite of it (the bench
wrapper, which shares this layout, has the same limitation). The coder is
confined like the reviewers, so a `go test` it runs builds with a cold cache
(`GOCACHE` points into the scratch directory).

A mount below `/` can stay writable inside the Linux sandbox, and it is not a
hole. `--ro-bind / /` remounts every submount read-only, but bubblewrap skips a
submount it gets `EACCES` for — one your user cannot reach because a directory
on the way to it is not searchable — on the reasoning that it is out of reach
either way. `/proc/self/mountinfo` inside then still lists it `rw`. The agent
runs as your user with your groups, so it cannot reach it either: what protects
those mounts is the same directory permission that protected them before, and
the sandbox adds nothing there. (On mediabox: the backup datasets under
`/tank/enc/backups`, and Docker's overlays under `/var/lib/docker`, root-only.)
A directory you can reach is remounted read-only. Check yours with
`bwrap --ro-bind / / --dev /dev --proc /proc cat /proc/self/mountinfo`, and
confirm that each `rw` entry outside `/dev`, `/proc` and `/tmp` is one
`ls` cannot open.

### Agents that switch their own sandbox off: `-agents-sandboxed`

Sandboxes do not nest. A CLI that confines its own shell commands — codex's
`--sandbox read-only` is seatbelt on macOS and bubblewrap on Linux — cannot start
that confinement inside a wrapper: on both platforms every command it runs fails.
The only way to run such an agent inside `sandbox.command` is to switch its own
sandbox off (`--sandbox danger-full-access`) and let the wrapper be what keeps it
read-only.

fixpoint refuses that by default, because for a `can_edit: false` agent the
command's own argv is the one barrier it can read (see [agents](agents.md)).
`-agents-sandboxed` lifts that one refusal for an invocation: you are asserting
that `sandbox.command` enforces read-only for the agents it wraps — and fixpoint
passes it `{{target_mode}}` = `ro` for exactly these. Nothing else is relaxed.
An agent file states the swap itself with `when_sandboxed.replace`, applied only
under this assertion, so one file serves both a confined and an unconfined run.
The flag needs a `sandbox.command` to point at, and without one the load fails
with a message that says so, rather than quietly lifting a check for a sandbox
that is not there. Every other check on the composed argv still runs: the binaries
must resolve, a reviewer may not declare `can_edit: true`, and the pr-mode rules
about a command that resolves inside the target are unchanged.

It is a flag, like `-trusted-target`, and for the same reason: a config that could
declare its own agents sandboxed would be one line in the repository under review
that removes the write barrier from the reviewers reading it, on the strength of a
wrapper the same repository chose. A config setting `loop.agents_sandboxed`, in the
task config or in any base it extends, fails to load.

**It is refused in `pr` mode**, for every wrapper but fixpoint's own (above). The
target there is the pull request's content, the one input fixpoint treats as
hostile, and it cannot check that your wrapper honors `{{target_mode}}`. The CLI's own read-only sandbox is the barrier it can verify, so
it stays on; review the branch in directory or git-diff mode if you need the
confinement a wrapper gives. The flag appears in the run's applied overrides as
`agents_sandboxed=true`, beside the trust flags.

The bench is the one shipped user: `bench/run.sh` passes it, with
`bench/sandbox.sh` as the wrapper, so a codex candidate can run with the answer key
unreadable. The wrapper's goal is bench integrity (the answer key, past runs and
other lenses' findings stay unreadable, and nothing is written outside the places the
lens is given), not a general defence of the operator against a hostile model: the
lens still holds the provider key its CLI needs and can use the network. Inside it
codex's own sandbox is off, so writes are confined by the wrapper alone. That wrapper is a worked example of the "mount what the CLI needs"
shape, and of its limits: it hides the checkout and gives codex a fresh state
directory, but on macOS the rest of `$HOME` stays readable, a claude lens keeps the
operator's real config directory (writable `.claude.json`: a user-scope MCP server
planted there runs the next time you start `claude`), and the network is open on both
platforms. `bench/README.md` lists exactly what a lens can still reach, and the
known limitations review found and left open: state relocated outside `$HOME` stays
readable on Linux; the `/run` mask assumes systemd-resolved; a custom
`CLAUDE_CONFIG_DIR` blocks claude's config rewrite on Linux; a target path ending in
a newline is unsupported; a signal before the agent is spawned is swallowed; provider
keys are readable by the lens through `/proc/<pid>/environ` of its ancestors; the
test's namespace runs fail instead of skipping where `bwrap` has no user namespace; a
ChatGPT token refresh cannot persist from inside the wrapper (codex renames a temp file
over `auth.json`); on macOS a lens can read same-user processes' environments; on Linux
a claude candidate on a third-party route still sees the Anthropic `.credentials.json`;
the writable CLI state (`.claude.json`, `auth.json`) is a channel colluding lenses could
use, unseen by the scorer; and the timeout's group SIGKILL path is untested.

### Trust

Two separate rules apply, and they are about different things.

`sandbox.command` is a value in a bundle file, so if the *file declaring it* was
resolved from inside the target, it is target-supplied policy and needs
`-trusted-bundle` like every other executable thing a bundle can carry. That rule
is unchanged; the wrapper is simply one more argv the repository under review must
not get to choose.

Separately, the wrapper's own *path* is now `argv[0]`. Every agent-command check
reads the composed argv, so a wrapper that resolves inside the target is refused
in `pr` mode by the same rule that refuses a target-relative agent command — and
for the same reason, since `gh pr checkout` rewrites that path's content after
validation read it.
