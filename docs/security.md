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
- **Trust is asserted on the command line only, never in a config file.**
  fixpoint refuses to load a config that sets `loop.trusted_target`,
  `loop.trusted_bundle` or `loop.allow_untrusted_fix`. Bundles are shadowable and `<project>/config` is
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
  at untrusted content only on a host without sensitive files, or launch them
  through [`sandbox.command`](#confining-agents-with-sandboxcommand) — which is
  this advice made executable rather than left to the operator.
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
| `{{target}}` | The absolute path of the directory under review. |
| `{{target_mode}}` | `rw` for an agent declared `can_edit`, `ro` for every other — so a reviewer and the coder beside it get the same wrapper with different access. |
| `{{agent}}` | The agent's name, for a wrapper that keeps per-agent profiles. |

A placeholder fixpoint does not recognize is refused at startup rather than passed
through verbatim: a typo in a mount argument would confine the wrong path. A
wrapper that expands to nothing at all is refused for the sharper version of the
same reason — the run would be unconfined while the configuration said otherwise.

**fixpoint implements no sandbox and does not try to.** Which primitive fits
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

And it is **refused for create and implement runs**. Those pipelines invoke their
agents in the assignment snapshot and the output directory rather than in
`target.path`, so `{{target}}` — expanded once, from the config — would name a
directory the agent is not working in: the wrapper would bind one tree while the
CLI wrote to another. A confinement aimed at the wrong place is the failure mode
this feature exists to prevent, so it fails closed instead of approximating.
Expanding the placeholders per invocation is the real fix and is not implemented.

One deliberate exemption is worth knowing about: an argument naming the target
**root** (`--ro-bind {{target}}`) does not count as a path the reviewed material
supplied, because it is the checkout the run is about rather than a file the
branch can write. Anything *deeper* inside the target still does, and is still
refused in `pr` mode.

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
