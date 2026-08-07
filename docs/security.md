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
  (`./reviewer.sh`, or the script in `[node, ./reviewer.cjs]`) names a file the
  branch `gh pr checkout` writes, *after* validation confirmed the one that was
  there before. That is target-controlled code execution rather than the prompt
  injection the `pr` path is built to contain, and it needs no model's
  cooperation, so fixpoint refuses the combination at startup unless you assert
  `-trusted-target`/`-allow-untrusted-fix` (with the assertion it warns instead).
  Point such a command at a bare name on `PATH` or a path outside the target.
  Every other mode keeps the target-relative form: nothing replaces the file
  between validation and the run.
- **Reviewers can read anything, even in review-only mode.** Read-only agent
  flags block edits but do not confine reads: a reviewer fed untrusted content
  can be prompt-injected into reading a host secret (`~/.ssh`,
  `~/.aws/credentials`, `.env`) and quoting it into a finding. Point reviewers
  at untrusted content only on a host without sensitive files, or run
  fixpoint inside a container/VM.
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
  implemented. Run agent CLIs you do not trust inside a container/VM, and do not
  grant one `env.inherit_all`.
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
- **`prompt_via: arg` exposes the prompt on the process argument list**,
  readable by other local users via `ps`/`/proc`. Prefer `stdin` on shared
  hosts; fixpoint warns at run start.
