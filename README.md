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
Claude Code, Codex, models served via ollama (Kimi, GLM), and — in an agent file
you can enable — anything on OpenRouter, but nothing in the code is
provider-specific.

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
6. **Commit.** Each fix lands as its own commit, naming the issue it closed.
   `loop.commit_policy` decides whether those commits stay separate (`per_fix`,
   the default) or are squashed per round or per run.
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
| `git-diff` | Changes relative to `target.base_ref`. The base is resolved to a concrete commit once at run start, so per-round fix commits extend the reviewed diff instead of shrinking it. A trailing `...` (`origin/main...`) pins the **merge base** with HEAD instead of the ref's tip — what this branch added, which is what `fix-branch` uses. Without it, any commit the base branch has and yours does not appears in the diff *reversed*, and the panel reviews someone else's work as deletions you made. Empty `base_ref` reviews unstaged working changes, so it is review-only: a fix run needs a base ref (it starts from a clean tree, which makes the unstaged diff empty) and is rejected at validation without one. |
| `pr` | A GitHub pull request. The PR branch is checked out locally (`gh pr checkout`) and reviewed against its base, so fixes land in the working tree and later rounds review them too. Pushing fixes back is manual. |

Fix rounds require `target.path` to be a git repository with a clean working
tree at run start (in every mode); `review_only` runs anywhere.

`target.exclude` and the credential patterns fixpoint enforces on top of it apply
in **every** mode: in `git-diff` and `pr` they become git exclude pathspecs, so an
excluded file never contributes its diff or its name to the collected material.
That matters most there, since a diff carries file *content* into every reviewer's
prompt, where a directory listing only carries paths.

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
- **`once: true`** — the lens runs in round 1 only: one report per run instead of
  a session every round. Consider `final: true` instead — it costs the same single
  session and reports on the code the run *produced* rather than the code it
  started from, which for a lens nobody acts on mid-run is the only state that
  matters. `once` earns its keep only when you specifically want the "before"
  picture.
- **`final: true`** — the lens is held out of the loop and runs once at the end,
  on **every** agent in the pool, in a closing round whose findings are fixed like
  any other. For a lens whose subject is the *finished* code. `review-tests` is the
  case: asked inside the loop it assesses coverage of work later rounds rewrite, so
  it demands tests for intermediate states and re-reports the gap every time the
  code moves — in one five-round run it produced 30 of 68 reports, had 18 deferred
  (more than every other lens combined), and 65% of everything that run wrote was
  test code. Asked once, at the end, by the whole panel, the question is answered
  about code that has stopped changing and nothing follows it to starve.

  **Actionable final lenses repeat until nothing is left to fix.** A single pass is
  not enough whenever anything bounds what one pass hands over — and inside the loop
  that never mattered, because the next round picked up whatever was deferred. Here
  there is no next round, so a pass that stopped early would let a run read as
  complete with coverage gaps still open. Re-reviewing between passes isn't waste
  either:
  pass 2 sees the tests pass 1 wrote, so it reports what's genuinely still missing
  instead of working from a list computed before the code changed — which is also
  what makes the phase stop on its own. `loop.max_final_passes` (default 2) bounds
  it as a last resort, and running out is always said loudly rather than dropped
  silently — either issues are still open, or the last pass fixed everything it
  reported and there was no pass left to review those fixes.

  Two, and its own knob rather than `max_iterations`, because this phase is where a
  measured run spent 51 minutes and still had pass 2 producing four *new* issues:
  each pass reviews the tests the previous pass just wrote. Every repeated issue id
  in that run came from here — a real bug fixed in the loop, re-opened as "the test
  for that fix is flaky", then as "the test for the test" — while the loop's own
  rounds did not repeat themselves at all.

  **Advisory final lenses run exactly once, after that** — they're reports, and a
  report should describe the code that actually shipped, which isn't known until the
  fixing stops. They never share a round with the fix passes, so a three-pass
  closing round still produces one design report, not three.

  It runs after **every** normal termination — converged, all-rejected, and
  max-iterations alike — since the loop is done editing in all three, but not after
  an error or interruption, when the tree is in a state nobody vouched for. It does
  not change the run's termination: it is extra work on an already-decided run.
  `final` and `once` are mutually exclusive, and a review-only run has no closing
  round (there is no coder), so a final lens simply runs in its single round.

  **Pin a final lens whose findings are advisory.** Unpinned means the whole panel,
  which is right when the findings get fixed — nothing follows to catch what one
  model missed. For a report a human reads, it means four overlapping documents and
  corroboration that buys nothing, since nothing gets scheduled. The shipped
  `fix-code` uses both shapes: `review-tests` unpinned and fixed, `review-design`
  and `review-maintainability` pinned and reported.

## Observations and issues

A reviewer's report is an **observation**. What the coder works from is an
**issue**: one distinct problem, with every observation that reported it attached.

The distinction is not bookkeeping. When a finding was simultaneously a reviewer's
report, the unit of work, and the thing tracked across rounds, two agents reporting
the same problem produced two findings — and so two units of work. Agreement between
reviewers therefore *cost* more, rather than telling the run more. In one real run
two lenses reported a single racy-ordinal defect at the same file and line, one
calling it `concurrency` and the other `tests`, and it was scheduled twice — twice
against `loop.max_findings_per_round`, which was 8 at the time, and now twice as a
coder session.

Now the panel's agreement is surfaced as corroboration — to the coder ("reported
independently by 2 agents") and in the run summary — and is one issue, one session,
one commit.

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

## One fix, one commit

The coder is handed **exactly one issue per session**, under every commit policy.
`loop.commit_policy` then decides how those commits are grouped:

| Policy | Result |
|---|---|
| `per_fix` *(default)* | One commit per issue, each verified on its own. |
| `per_round` | Each round's commits squashed into one. |
| `per_run` | The whole run squashed into one, after the closing round. |

One issue per session is not a policy choice, because nothing downstream can undo
batching. A session handed eight issues edits files for all eight at once, and its
report says only *id, verdict, detail* — nothing that attributes a change to an
issue. So a commit built from that session cannot honestly claim to contain one fix,
however the commits are later grouped. Working one at a time buys three things a
batch cannot:

- **The verify gate names the fix that broke the build**, not the round. A round of
  eight fixes that fails the gate is discarded entirely today; the same eight as
  eight gated fixes lose only the one that failed.
- **`git revert` undoes a single bad fix**, and `git bisect` lands on one change.
- **A rejection is checkable.** A session that rejected its one issue should have
  left the tree untouched, and a session that claimed a fix should have changed
  something. With one issue per session both are verifiable claims rather than
  aggregate ones — a fabricated fix now fails the run instead of hiding behind seven
  real ones.

The cost is that the verify gate runs once per fix rather than once per round, so a
round of eight fixes runs the project's checks eight times. That is the price of
knowing which fix broke them, and it buys back the eight-fix round that used to be
discarded whole. Keep `verify.commands` cheapest-first, as the config already
advises — the gate stops at the first blocking failure.

Squashing re-commits the index as it stands, so a squashed commit's content is
byte-for-byte what the per-fix commits already verified one at a time. The
replacement commit is built before the branch moves, so a squash that fails leaves
the per-fix commits exactly where they were. It only ever removes
information, which is why `per_fix` is the default. A failed or interrupted run is
never squashed: its per-fix commits are how you see how far it got.

`commit_message` is per-fix shaped by default (`{issue}`, `{title}`). A squashing
policy renders a round-shaped header instead (`{round}`, `{fixed}`, `{rejected}`)
with a body listing every fixed and rejected issue; set `commit_message` to a
round-shaped template to control that wording yourself.

### Why there is no per-round cap by default

`loop.max_findings_per_round` used to default to 8. That number was one coder
*session's* capacity — 17 issues blew the 30m timeout, ~8 fit — and with one issue
per session it no longer bounds anything real, so it now defaults to `0`
(unlimited).

It was never free. In one five-round run a cap of 8 deferred 18 issues, and in the
closing round — where no later round follows — it dropped 5 coverage gaps outright.
Set it only as a **cost lever**, to bound what a single round spends, knowing the
overflow waits for a later round (worst severity first, and every deferral promotes
an issue one tier so the tail cannot be starved). Note that `commit_policy` is not
a way to get the old behavior back: `per_round` groups commits, it does not batch
work.

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

### Borrowing an agentic harness for a model that has no CLI

A reviewer has to *explore* the repository, not just answer from a prompt, so a
raw chat endpoint is not enough on its own. Most models don't ship a CLI of their
own — but any Anthropic-compatible endpoint can borrow Claude Code's harness,
which is how the shipped ollama agents work (`ollama launch claude`) and how
`config/agents/qwen.yaml` reaches OpenRouter with nothing but a base URL:

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

`codex` is not a second route to OpenRouter: codex 0.146 dropped
`wire_api = "chat"` in favour of the Responses API, which OpenRouter does not
serve.

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

The `verify` commands are filtered too, from the other direction. They are argv the
*target* can supply, so running them with fixpoint's whole environment would hand a
`curl $ANTHROPIC_API_KEY` "build" command every credential the agents deliberately
do not share. They inherit fixpoint's environment **minus** every variable that
carries an agent credential: the names your agent files declare (`env.pass`,
`env.set`) plus a built-in list (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`,
`CODEX_API_KEY`, `GOOGLE_API_KEY`, `GEMINI_API_KEY`, `GITHUB_TOKEN`, `GH_TOKEN`,
and the three `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`/`AWS_SESSION_TOKEN`
names). That direction is a denylist rather than an allowlist on purpose: what a
build actually needs is language- and project-specific (`GOFLAGS`, `JAVA_HOME`,
`CARGO_HOME`, `VIRTUAL_ENV`, ...), and an allowlist would silently break checks by
dropping what it forgot.

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
make review-code    # one review round, coder never invoked (no edits; pr mode still checks out the PR branch)
make run            # the full review->fix cycle (runs tests and vet first)
```

Or directly:

```sh
./fixpoint <config-name> [flags]      # e.g. ./fixpoint review-code
./fixpoint --config path/to/task.yaml [flags]
```

| Flag | Effect |
|---|---|
| `<name>` | Positional: the task config to run, resolved on the bundle search path. A value containing a separator or ending in `.yaml` is used as a path. |
| `--list` | List task configs with the file each resolved from, and exit. |
| `-config path` | Alternative to the positional name. Giving both is an error. |
| `-review-only` | Run exactly one review round; the coder is never invoked (no edits in git-diff/directory mode; pr mode still runs `gh pr checkout`, switching the branch and working tree in Prepare). |
| `-max-iterations n` | Override `loop.max_iterations`. |
| `-base-ref ref` | Override `target.base_ref` in git-diff mode; a trailing `...` means the merge base with HEAD. For `fix-branch` on a branch with no upstream: `-base-ref 'origin/main...'`. |
| `-trusted-target` | Assert a directory/git-diff target holds only trusted code, permitting fix rounds (fail-closed without it). |
| `-allow-untrusted-fix` | Permit fix rounds in `pr` mode (PR content is untrusted; see Security). |
| `-check` | Validate the configuration and exit. |
| `-check-live` | Validate, ping every agent, and exit. |

Exit codes: `0` converged or review-only completed, `2` hit `max_iterations`
without converging (or a usage error), `3` the coder rejected every issue so
nothing changed — deliberately *not* `0`, since "nobody agreed there was a
problem" is not "the code is clean", `1` any other failure or interruption. `SIGINT`/`SIGTERM` stop the run cleanly: the current step is abandoned and any edits
in the tree are stashed, so nothing half-finished is left behind. Signal a second time to quit
immediately without that reconciliation — which can leave the working tree dirty.

## Configuration

Configuration is a **bundle**: task configs at the root of a directory plus
`prompts/` and `agents/` beside them, so one bare name resolves in one category.
See [config/README.md](config/README.md) for the search path, `extends` merge
rules, and the security model; [config/defaults.yaml](config/defaults.yaml) is
the commented reference for the settings themselves.

Shipped configs are named `<verb>-<scope>`, with the verb first because it is the
safety property: a `review-` config never invokes the coder and cannot modify a
file, a `fix-` config edits your working tree and needs a trust assertion on the
command line.

| Config | What it does |
|---|---|
| [review-code](config/review-code.yaml) | Review a whole project once, no edits. Needs `-trusted-target` if the project ships its own bundle. |
| [review-pr](config/review-pr.yaml) | Review a GitHub pull request; review-only by default. |
| [fix-code](config/fix-code.yaml) | Review → fix → verify → commit loop over a whole project. Needs `-trusted-target`. |
| [fix-branch](config/fix-branch.yaml) | The same loop over only what this branch changed — git-diff against the merge base with `@{upstream}`. Needs `-trusted-target`. |
| [defaults](config/defaults.yaml) | Shared base the others extend; not runnable on its own. |

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
- **`loop`** — `max_iterations`, `max_final_passes` (how many times the closing
  round may repeat, default 2), `commit_policy` (see
  [One fix, one commit](#one-fix-one-commit)), `max_findings_per_round` (caps how
  many **issues** a round hands over, `0` = unlimited and the default; worst
  severity goes first and the overflow is deferred to later rounds, but every
  deferral promotes an issue one severity tier, so nothing can be starved
  indefinitely by a steady supply of more-severe ones), `review_only`,
  `commit_message` (placeholders `{issue}`, `{title}` for a single fix; `{round}`,
  `{fixed}`, `{rejected}` for a squashed one), and `clean_rounds_to_stop` (how many
  consecutive clean rounds end the run — `2` pairs well with `strategy: rotate`, so
  a differently-assigned panel must confirm the clean result). The trust gates are
  deliberately **not** here: they are command-line flags only, for the reason given
  under [Security model](#security-model).
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
  journal.jsonl                                       # append-only state transitions, flushed as they happen
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
best-effort credential redactor — but see below. The `.md`, `.prompt` and
summary artifacts are also terminal-escaped, so paging one cannot let injected
agent output or reviewed content drive your terminal; `.raw` deliberately keeps
its bytes as the fidelity record, so pipe it through `cat -v` or a pager that
does not interpret escapes.

### The end-of-run table

Every run prints a scoreboard when it finishes, and the same text opens the
`summary-*.md`. It prints even when the run failed — a partial run still spent
tokens and may have committed rounds, which is exactly when you need to see what
landed.

```
──────────────────────────────────────────────────────────────────────────────
 fixpoint · fix-code · max-iterations after 5 round(s) · 2h02m
──────────────────────────────────────────────────────────────────────────────
 config     config/fix-code.yaml  (extends defaults)
 target     directory · /home/coder/fixpoint
 settings   review+fix · strategy rotate · no per-round cap · max 5 round(s)
 flags      trusted_target=true
 verify     fmt, vet, test, lint · passed in 5/5 round(s)

 REVIEWER  issues  fixed  rejected  deferred  advisory  errors    time
 claude        27     18         0         9        23       1  48m47s
 codex         23     17         1         5        11       0  44m08s
 gemma4         9      7         0         2         1       1  36m34s
 glm            8      4         0         4         6       0  25m38s
 ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
 TOTAL         60     39         1        20
 rows sum above the total: 7 issue(s) were reported by more than one reviewer

 LENS                    issues  fixed  rejected  deferred  advisory  errors    time
 review-bugs                 20     15         1         4         0       0  35m21s
 review-security             12     10         0         2         0       1  44m44s
 review-tests                26     12         0        14         0       0  24m37s
 ...

 coder      claude-coder · 39 fixed · 1 rejected · 1h15m
 commits    39 · f40acbc26fbd a7361e82ecdc d8f4b9d429ea ...  (per_fix)
 exit       max-iterations (exit 2)
──────────────────────────────────────────────────────────────────────────────
```

It answers the question the interleaved per-round log cannot: **which agent and
which lens earned their tokens.** A panel is only worth its cost if the answer
varies between its members, and the table above is what a weak member looks like.
The `review-tests` row — the most reports, the most deferred, the least converted
into fixes — is why that lens is now `final: true`.

**Tokens and cost are what the agent's own CLI reported**, never a fixpoint
estimate. Each agent file says where its CLI puts those numbers (see `usage:` in
[config/README.md](config/README.md)); an agent that reports nothing shows `-`.
Cost is shown only where the CLI computes it — `codex` on ChatGPT-account auth
reports tokens but has no per-request price, and inventing one from a published
rate card would be a guess printed as an audit figure. There is no pricing API to
consult, and a rate card knows nothing about how much of a prompt was served from
cache.

Do not substitute the byte counts for this. The prompt fixpoint sends and the text
it gets back are the two ends of a session that reads files and calls tools in
between, none of which crosses this process: a one-word probe whose boundary I/O
was ~30 bytes reported **21,072 tokens and $0.06**. `tokens` therefore counts cache
reads and writes too — on a warm agentic session they are most of the total.

`commits` counts every commit the run made, which under the default
`commit_policy: per_fix` is one per fix — the number you can reconcile against
`git log`. The policy is printed beside it because "39 commits" and "5 commits" can
describe the same 39 fixes, and only the policy says which you are looking at.

Attribution is by **issue**, not by raw observation, so two agents reporting one
defect each get credit for that one issue. The per-agent rows therefore sum to more
than the distinct TOTAL whenever the panel agreed, and the line under the total
accounts for the difference.

### The run journal

`journal.jsonl` records each state transition as one JSON object, flushed to disk
as it happens. The summary is a single write at the *end* of a run, which makes it
useless for the two questions that matter when a run goes wrong: **what order did
things happen in**, and **what was true at the moment it died**. A run killed
mid-round has a journal up to its last transition and a summary that describes a
run which never finished — or no summary at all.

Every record carries `v` (schema version), `seq` (authoritative ordering — the wall
clock can repeat within a timestamp interval and can move backwards), `at`, `type`,
an optional `round`, and a per-`type` `data` payload:

```
run_started        config, mode, path, strategy, review_only, max_iterations,
                   and the flag overrides that authorized the run
verify_baseline    what was already failing before the run touched anything
round_started      the lens→agent assignment (differs per round under `rotate`)
review_finished    observations, advisory, reviewer errors
issues_aggregated  observations → issues, and how many were corroborated
issues_deferred    the per-round cap biting, naming which issue ids waited
fix_finished       the coder's self-report: issues handed over, fixed, rejected
verify_finished    one gate run: attempt (initial|correction|salvage), per-check
                   results, and which failures actually block under the policy
round_committed    sha, and whether it was salvaged partial work
round_discarded    reason (verify_failed | salvage_verify_failed | interrupted |
                   commit_failed | rejected_with_edits) and whether the work was
                   stashed
round_clean        the convergence streak, and whether reviewer errors reset it
run_finished       termination, rounds, error
```

Two properties are deliberate. **No transition record is written before the
symlink check** that authorizes artifact writes, because every round after it
commits; a run refused by a gate therefore has a journal containing only
`run_finished`, naming the refusal. And **a journal failure never fails the run** —
it is an audit artifact, and losing it must not discard fixes that already passed
verification and were committed. You get one warning on stderr and the run
continues.

It reads well with `jq`:

```sh
jq -c 'select(.type=="verify_finished") | {round, a:.data.attempt, b:.data.blocking}' \
  .fixpoint/*/journal.jsonl
```

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
- **A bundle resolved from inside the target needs `-trusted-target` too, even
  for a review-only run.** `<project>/config` is searched first, so a repository
  can ship the very files a run is built from: agent commands and verify commands
  are argv fixpoint executes, and a prompt is the instruction stream handed
  verbatim to a reviewer that can read anything you can. None of that needs a
  model's cooperation to exploit. fixpoint therefore lists every bundle file that
  came from inside the target and refuses until you assert trust — or point
  `-config` at a bundle outside it. The check is per *file*, not per key, so it
  cannot go stale as configuration grows new surface.
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
internal/model/          shared data shapes: observations, issues, verdicts, summaries, journal events
internal/issue/          groups observations into issues; tracks them across rounds
internal/verify/         runs the deterministic gate (build / test / static checks)
internal/logstore/       per-step logs, the run journal, and the run summary
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
