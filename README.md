# fixpoint

An agent-agnostic, automated code-review loop. Reviewer agents inspect a
target in parallel — each through a focused review "lens" (bugs, security,
concurrency, tests, maintainability, design) — a coder agent validates and
fixes the findings, the orchestrator commits each fix round, and the cycle
repeats until reviews come back clean.

The name is the termination condition: the loop iterates review→fix until the
code stops changing — a [fixed point](https://en.wikipedia.org/wiki/Fixed_point_(mathematics)),
reached when a full reviewer panel reports nothing left to fix.

Any agentic CLI works as a reviewer or coder: an agent is just a command that
receives a prompt and prints text to stdout. The shipped configuration mixes
Claude Code, Codex, and local models served via ollama (Gemma, GLM), but
nothing in the code is provider-specific.

## How it works

```
one round:

  review-bugs ─────────► agent A ─┐
  review-security ─────► agent B  │   observations      issues
  review-concurrency ──► agent C  ├──► (raw reports) ──► (deduped) ──► coder ──► verify ──► commit
  review-tests ────────► agent D  │                                    fixes    build/test
  review-maintainability ► ...  ──┘

repeat until a full reviewer panel reports nothing
```

1. **Validate.** Before anything runs, the configuration is statically
   checked: all referenced agents are defined, their binaries exist on PATH,
   prompt files parse and their placeholders resolve, the coder's agent has
   `can_edit: true`, and the lens-assignment strategy is satisfiable. Any
   failure aborts before a single agent is invoked. Optionally
   (`ping_agents: true`, the default) every agent used by the run is also
   pinged with a trivial prompt so expired logins and broken CLIs surface
   before tokens are spent or git is touched.
2. **Review.** Each review lens (a prompt file) is assigned to an agent per
   the configured strategy and all reviewers run in parallel, each reporting
   structured observations (category, severity, file/line, description,
   suggestion).
3. **Aggregate.** Observations are grouped into **issues** — see
   [Observations and issues](#observations-and-issues). Several reviewers
   reporting one problem produce one issue, so agreement between agents raises
   confidence instead of consuming the round's budget twice.
4. **Fix.** The issues are handed to the coder agent, which validates each one:
   it fixes the genuine ones by editing files directly and rejects the rest with
   a reason.
5. **Verify.** fixpoint then runs the project's own configured build, test, and
   static checks *itself* — see `verify` in the config. This is the only signal
   in the loop that no model produced: without it, "fixed" means an agent said
   it fixed something and "converged" means other agents said they saw nothing.
   A round that fails the gate gets one bounded correction attempt from the
   coder; if it still fails, the round is **discarded** (edits stashed, not
   committed), because committing them would put later rounds on a broken base.
   Under the default `no_regressions` policy a check that was already failing
   before the run may keep failing — only newly broken checks block.
6. **Commit.** The round's changes land as one inspectable, individually
   revertable commit whose body lists every fixed and rejected issue.
7. **Repeat.** Each round, reviewers receive the history of prior issues and
   coder verdicts, so a rejected issue is not re-reported forever. The loop ends
   after a configurable number of consecutive clean rounds, when the coder
   rejects everything in a round, or at the iteration cap.

If the coder dies mid-round (timeout, session limit, malformed output) after
editing files, its partial work is put through the same verification gate as a
normal round. If it passes, it is committed as a "partial" round and the loop
continues — the next round re-reviews everything, so the run self-heals instead of
stranding valid edits. If it fails, the edits are stashed for you to inspect
(`git stash pop`) and the run stops rather than building later rounds on a base
that is known to be broken. A coder that died mid-edit is the case most likely to
leave a tree that does not compile, so this is the path that most needs the gate;
reviewers are models reading content, not a substitute for a compiler.

## Target modes

What gets reviewed is controlled by `target.mode`:

| Mode | What is reviewed |
|---|---|
| `directory` | Every file under `target.path` except what `.gitignore` and the `exclude` globs remove. Scope is denylist-only — there is no allowlist option, since one has to be re-derived per language and silently hides whatever it forgets. |
| `git-diff` | Changes relative to `target.base_ref`. The base is resolved to a concrete commit once at run start, so per-round fix commits extend the reviewed diff instead of shrinking it. Empty `base_ref` reviews unstaged working changes. |
| `pr` | A GitHub pull request. The PR branch is checked out locally (`gh pr checkout`) and reviewed against its base, so fixes land in the working tree and later rounds review them too. Pushing fixes back is manual. |

Fix rounds require `target.path` to be a git repository with a clean working
tree at run start (in every mode); `review_only` runs anywhere.

In `directory` mode against a git repository, scope comes from
`git ls-files --cached --others --exclude-standard` — tracked files plus
untracked ones the project doesn't ignore. So build output, caches, local
binaries and coverage profiles drop out per project without a single config
entry, because the project already declared them in `.gitignore`. A non-git
target falls back to a filesystem walk, where only `target.exclude` narrows
scope. What belongs in `target.exclude` is therefore just what's *committed*
but not worth reviewing — vendored dependencies, fixtures, and (defensively) a
committed `.env`.

## Review lenses and assignment strategies

Prompts under [config/prompts/](config/prompts/) are a library you can grow freely; only the
ones referenced in the configuration are used. The shipped lenses:

- [review-bugs.md](config/prompts/review-bugs.md) — correctness
- [review-security.md](config/prompts/review-security.md) — security
- [review-concurrency.md](config/prompts/review-concurrency.md) — concurrency
- [review-tests.md](config/prompts/review-tests.md) — test coverage of what the run changed
- [review-maintainability.md](config/prompts/review-maintainability.md) — smells, simplification, docs (advisory)
- [review-design.md](config/prompts/review-design.md) — architecture (advisory, round 1 only)
- [fix.md](config/prompts/fix.md) — the coder's instructions

Which agent runs which lens is decided by `roles.review.strategy`:

- **`fixed`** — a lens runs only with its pinned agent (every lens must pin one).
- **`rotate`** — unpinned lenses cycle through the agent pool each round, so
  every lens is seen by different models across rounds and fixes get
  re-reviewed by fresh eyes, not by the model that reported the finding.
- **`all`** — every lens runs with every agent, every round: maximum coverage
  at pool-size × the invocations and cost.

Per-lens modifiers:

- **`advisory: true`** — observations are logged as a report but never aggregated
  into issues or handed to the coder, and don't count toward termination. Use it for
  lenses where automated fixing is too risky (design/architecture), and for any
  lens whose findings are open-ended enough that requiring them to reach zero
  would keep the loop from ever converging.
- **`once: true`** — the lens runs in round 1 only. Pairs well with an
  advisory design lens: one report per run instead of a full agent session
  every round.

## Observations and issues

A reviewer's report is an **observation**. What the coder works from is an
**issue**: one distinct problem, with every observation that reported it attached.

The distinction is not bookkeeping. When a finding was simultaneously a reviewer's
report, the unit of work, and the thing tracked across rounds, two agents reporting
the same problem produced two findings — each consuming a slot against
`loop.max_findings_per_round`. Agreement between reviewers therefore *reduced* how
many distinct problems a round could fix. In one real run two lenses reported a
single racy-ordinal defect at the same file and line, one calling it `concurrency`
and the other `tests`, and it cost two of that round's eight slots.

Now the panel's agreement is surfaced as corroboration — to the coder ("reported
independently by 2 agents") and in the run summary — and costs one slot.

Grouping works in two stages, because matching within a round and matching across
rounds are different problems:

- **Fingerprint** — normalized path plus the exact line, or a normalized title when
  no line is given. Nearby lines (within 5) merge only when the titles also agree,
  since three unrelated defects on consecutive lines are three issues: merging them
  would tell the coder to fix one thing when there are three, which is worse than
  leaving a duplicate that merely costs a slot. The category is deliberately **not**
  part of identity — the real duplicate above arrived under two different ones.
- **Across rounds** — a reviewer may set `"issue": "<id>"` on a report to declare it
  is the same problem as an entry in the history it was shown. No lexical rule gets
  from *"Severity vocabulary has two independent declarations"* to *"duplicated
  severity vocabulary"* while the line number moves underneath it, so the reviewer
  that can see both is asked. A nonexistent id falls back to the fingerprint.

An issue carries its own status across rounds, and a re-report means different
things depending on how it was closed. A **rejected** issue stays rejected and is
not handed back — re-submitting a decided question would spend a slot every round.
A **fixed** issue **reopens**: reviewers still seeing it is evidence the fix did not
work, and treating it as closed would let a failed fix end the run as converged.

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
allows writing files (`can_edit: true`) may be assigned as `roles.coder`.

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
requirements you don't know; fixpoint warns at run start when an agent does.

## Getting started

Requirements:

- Linux or macOS. **Windows is not supported yet** — see Platform support below.
- Go 1.26+
- git (and `gh` for `pr` mode)
- at least one agentic CLI installed and authenticated (e.g. `claude`,
  `codex`, `ollama`) — verify the flags in `config/agents/*.yaml` match what your
  installed versions expect

```sh
make build          # compile the fixpoint binary
make check          # static validation of the configuration (no agents invoked)
make check-live     # static validation + ping every configured agent
make review-only    # one review round, coder never invoked (no edits; pr mode still checks out the PR branch)
make run            # the full review->fix cycle (runs tests and vet first)
```

Or directly:

```sh
./fixpoint <config-name> [flags]      # e.g. ./fixpoint review-only
./fixpoint --config path/to/task.yaml [flags]
```

| Flag | Effect |
|---|---|
| `<name>` | Positional: the task config to run, resolved on the bundle search path. A value containing a separator or ending in `.yaml` is used as a path. |
| `--list` | List task configs with the file each resolved from, and exit. |
| `-config path` | Alternative to the positional name. Giving both is an error. |
| `-review-only` | Run exactly one review round; the coder is never invoked (no edits in git-diff/directory mode; pr mode still runs `gh pr checkout`, switching the branch and working tree in Prepare). |
| `-max-iterations n` | Override `loop.max_iterations`. |
| `-trusted-target` | Assert a directory/git-diff target holds only trusted code, permitting fix rounds (fail-closed without it). |
| `-allow-untrusted-fix` | Permit fix rounds in `pr` mode (PR content is untrusted; see Security). |
| `-check` | Validate the configuration and exit. |
| `-check-live` | Validate, ping every agent, and exit. |

Exit codes: `0` converged or review-only completed, `2` hit `max_iterations`
without converging (or a usage error), `3` the coder rejected every issue so
nothing changed — deliberately *not* `0`, since "nobody agreed there was a
problem" is not "the code is clean", `1` any other failure or interruption. `SIGINT`/`SIGTERM` stop the run cleanly.

## Configuration

Configuration is a **bundle**: task configs at the root of a directory plus
`prompts/` and `agents/` beside them, so one bare name resolves in one category.
See [config/README.md](config/README.md) for the search path, `extends` merge
rules, and the security model; [config/defaults.yaml](config/defaults.yaml) is
the commented reference for the settings themselves.

Bundles are searched most-specific first — `<project>/config/`, `~/.fixpoint/`,
the OS per-user config dir, then the package-installed bundle — **per file**, so a
project can shadow one prompt and inherit the rest. There is deliberately no
fallback compiled into the binary: these files decide what agents are told to do
and which trust gates apply, so they must be readable on disk. If nothing
resolves, fixpoint refuses to run and prints every location it searched. Every
run logs the file each name resolved to, and records it in the run summary.

The main sections of a task config:

- **`target`** — what to review: mode, path, exclude globs, base ref
  or PR number.
- **`roles`** — the coder (agent + prompt) and the review lens list with its
  assignment strategy and agent pool.
- **`agents`** — the command templates described above.
- **`loop`** — `max_iterations`, `max_findings_per_round` (caps how many **issues**
  one coder session receives; worst severity goes first and the overflow is
  deferred to later rounds, but every deferral promotes an issue one severity tier,
  so nothing can be starved indefinitely by a steady supply of more-severe ones),
  `review_only`, `commit_message` (placeholders `{round}`, `{fixed}`,
  `{rejected}`), and `clean_rounds_to_stop` (how many consecutive clean rounds end
  the run — `2` pairs well with `strategy: rotate`, so a differently-assigned panel
  must confirm the clean result). The trust gates are deliberately **not** here:
  they are command-line flags only, for the reason given under
  [Security model](#security-model).
- **`verify`** — the deterministic gate fixpoint runs itself between the coder and
  the commit: `commands` (argv, per project, cheapest first), a per-command
  `timeout`, and `policy` — `no_regressions` (the default: a check already failing
  before the run may keep failing, one that passed may not start failing),
  `must_pass`, or `off`. No commands means the gate is off, which is why the shipped
  defaults define none. Commands execute code from the target, so they run only on
  the fix path, which already requires the trust assertion. Every path that can
  produce a commit passes this gate — including the recovery path for a coder that
  failed mid-edit — so a commit later rounds build on, and that convergence can be
  declared over, has always been verified.
- **`logs`** — where and in which formats run artifacts are written.

## Logs

`logs.dir` is a path template, so the layout is yours to choose. It takes
`{timestamp}` (the run's start time — one value for the whole run) and
`{round}`. The default, `.fixpoint/{timestamp}/round-{round}`, produces:

```
.fixpoint/<run-timestamp>/
  round-1/
    review-<agent>-<lens>-<timestamp>.{md,json,raw}   # one set per reviewer step
    fix-<agent>-fix-<timestamp>.{md,json,raw}
    *.prompt                                          # exact prompt, written at invocation start
  round-2/...
  summary-<timestamp>.{md,json}                       # assignments, issues, verdicts, verification, termination
```

The prefix is a hidden, tool-owned name on purpose. fixpoint is meant to run
against projects it doesn't own, and that prefix is used both as a git pathspec
and as a directory-walk skip — so a generic name like `logs` would, in a project
that already has its own `logs/`, silently drop that real directory from review
scope and from round commits. **Add `.fixpoint/` to the target project's
`.gitignore`**: round commits already exclude it, but your own `git add -A`
would not.

Three paths are derived from the template, and validation rejects a template
that would lose artifacts:

- The **literal prefix** before the first placeholder (`logs` above) is the only
  part that never varies, so it's what round commits, clean checks, and
  collected material exclude. A template must have one, or a run would commit
  and then re-review its own logs.
- The **`{timestamp}` segment and its ancestors** form the run directory: the
  summary lives there because it spans all rounds, and it's claimed atomically
  at run start. `{timestamp}` is required and must precede any `{round}`.
- Everything **below** the `{timestamp}` segment is the per-round directory.
  Omit `{round}` (`.fixpoint/{timestamp}`) to put every round in one directory —
  `logs.pattern` must then carry `{round}`, or each round overwrites the last.

The exact prompt sent to each agent is always written at invocation start, so
a slow or killed agent's input is inspectable mid-run. Run directories and
files are owner-only (0700/0600), and persisted artifacts pass through a
best-effort credential redactor — but see below.

## Security model

Read this before pointing the tool at code you did not write.
[config/README.md](config/README.md) and the comments in
[config/defaults.yaml](config/defaults.yaml) cover each point in depth.

- **The coder edits files with permission checks disabled** and is not
  confined to `target.path`. A prompt-injection payload hidden in any reviewed
  file could steer it into writing elsewhere on your machine. Fix rounds are
  therefore **fail-closed**: they are refused unless you affirm the target is
  trusted with `-trusted-target`, and in `pr` mode — where the reviewed code is
  by definition untrusted — they additionally require `-allow-untrusted-fix`.
  Review each round commit before pushing.
- **Trust is asserted on the command line only, never in a config file.**
  fixpoint refuses to load a config that sets `loop.trusted_target` or
  `loop.allow_untrusted_fix`. Bundles are shadowable and `<project>/config` is
  searched *first*, so a config key would let the repository under review declare
  itself trustworthy — one line in a hostile repo's own config, authorizing both
  the execution of the agent definitions it ships and the write-capable coder,
  with no involvement from you. Keeping the assertion in the invocation is also
  what stops it becoming an inherited default that silently applies to the next
  untrusted repository you clone.
- **Reviewers can read anything, even in review-only mode.** Read-only agent
  flags block edits but do not confine reads: a reviewer fed untrusted content
  can be prompt-injected into reading a host secret (`~/.ssh`,
  `~/.aws/credentials`, `.env`) and quoting it into a finding. Point reviewers
  at untrusted content only on a host without sensitive files, or run
  fixpoint inside a container/VM.
- **Logs can contain secrets.** The reviewed material, raw agent output, and
  reviewer-authored text are all persisted; redaction is heuristic, not a
  guarantee. Keep the logs directory out of any sync, backup, or commit (the
  run's own logs dir is always excluded from round commits automatically).
- **`prompt_via: arg` exposes the prompt on the process argument list**,
  readable by other local users via `ps`/`/proc`. Prefer `stdin` on shared
  hosts; fixpoint warns at run start.

## Platform support

Linux and macOS. fixpoint does not currently compile for `GOOS=windows`: it puts
each agent in its own POSIX process group so that a timeout, a cancelled run, or
Ctrl-C reliably kills CLI-spawned children rather than leaking them. Windows has
no direct equivalent — a job object is the analogue — so that lifecycle handling
needs a platform-specific implementation behind build tags before Windows can
work. Two further gaps would need attention at the same time: the owner-only
(0700/0600) permissions on the artifact directory are not enforced on Windows,
which matters because those artifacts can contain secrets, and the test suite's
mock agents are shell scripts.

## Project layout

```
cmd/fixpoint/            CLI entry point: flags, exit codes, signal handling
internal/config/         bundle resolution (search path, extends) + validation
internal/orchestrator/   the review->fix loop: lens assignment, parallel
                         reviews, coder round, per-round commits, convergence
internal/agent/          runs an agent CLI as a subprocess, extracts the JSON
                         envelope, redacts secrets from persisted output
internal/prompt/         renders prompt templates (role placeholders + output contract)
internal/target/         collects review material per mode; git operations
                         (base pinning, clean-tree checks, round commits)
internal/model/          shared data shapes: observations, issues, verdicts, summaries
internal/issue/          groups observations into issues; tracks them across rounds
internal/verify/         runs the deterministic gate (build / test / static checks)
internal/logstore/       per-step logs and the run summary
internal/testfixture/    shared test helpers
config/                  the shipped bundle: task configs, prompts/, agents/
config/defaults.yaml     the commented base every task config extends
```

## Development

```sh
make test         # go test ./...
make test-race    # go test -race ./...
make cover        # tests + per-function coverage (cover-html opens a browser report)
make fmt          # gofmt -l -w .
make vet          # go vet ./...
make lint         # golangci-lint (configured in .golangci.yml)
make staticcheck  # staticcheck
make vulncheck    # govulncheck: known-vulnerability scan
make tidy         # go mod tidy
make audit        # everything CI runs: fmt-check, vet, lint, staticcheck, race tests, vulncheck
make help         # list all targets
```

Analysis tools run via `go run` with pinned versions, so no global installs
are needed and every machine uses identical tool versions. The lint policy
lives in [.golangci.yml](.golangci.yml): an exhaustive curated linter set
where every disabled check and exclusion states its reason.

The only external dependency is `gopkg.in/yaml.v3`.

Dogfooding note: this repository is reviewed by fixpoint itself — see the
`round N (M fixed, K rejected)` commits in the history.
