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

## Where to read what

| | |
|---|---|
| **[Getting started](#getting-started)** | Install, the make targets, and the exact commands for reviewing or fixing a pull request. |
| **[How it works](#how-it-works)** | One round, end to end, in a diagram. |
| **[What a run actually does](docs/concepts.md)** | Target modes, the review round in detail, how observations become issues, one fix per commit. |
| **[Review lenses](docs/lenses.md)** | What a lens is, how to write one, and how they are assigned to agents. |
| **[Agents](docs/agents.md)** | Adding an agent, borrowing a harness for a model with no CLI, why the route matters, environment filtering. |
| **[Logs and artifacts](docs/logs.md)** | What a run writes, the end-of-run table, the run journal. |
| **[Security model](docs/security.md)** | What is trusted, what is not, and why the trust gates are flags rather than config keys. |
| **[Configuration](#configuration)** | Bundles and the shipped configs. The reference for every setting is [config/README.md](config/README.md) and the comments in [config/defaults.yaml](config/defaults.yaml). |

## How it works

```
one round:

  review-bugs ─────────► agent A ─┐
  review-security ─────► agent B  │   observations      issues
  review-concurrency ──► agent C  ├──► (raw reports) ──► (deduped) ──► coder ──► verify ──► commit
  review-tests ────────► agent D  ┘                                    fixes    build/test

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
   [Observations and issues](docs/concepts.md#observations-and-issues). Several reviewers
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

**Reviewers are told not to build or test.** They were doing it — `go test ./...`
inside a *review* — and every line of that output returns as input tokens on the
next turn of a session whose turn count is already what drives the bill. It buys
nothing either: the gate above is fixpoint's own run, and a reviewer's private one
does not feed it. The rule lives in code (`prompt.ReviewWorkingRules`) so a new lens
inherits it, and it rides in the shared prelude so it costs nothing per lens.

**A reply that breaks the format gets one chance to restate it.** When an agent
exits cleanly but its `<review>` block is missing or malformed, fixpoint asks it to
emit the findings again in the required shape — without re-sending the material and
without letting it redo the review, so the second pass cannot quietly report
different findings. The alternative is what used to happen: a whole agentic session
discarded over its punctuation, plus a reviewer error that resets the convergence
streak and denies the run a clean round it had earned. Both attempts' usage is
billed to the one step. A crashed, timed-out or rate-limited agent is *not* retried
this way — it has nothing to restate.


**Better still, let the CLI enforce the shape.** An agent whose command carries a
schema placeholder (`--json-schema {{schema}}` on Claude Code, `--output-schema
{{schema_file}}` on Codex) has its output validated by the provider, and then
returns one bare JSON value with no envelope to break. That switches the prompt
contract and the extractor along with it — see
[config/README.md](config/README.md) for the mechanics. The shipped Claude agents
use it; agents whose CLI has no such flag keep the `<review>` envelope and the
salvage pass above, which is why both paths still exist.

## Target modes

What gets reviewed is controlled by `target.mode`:

| Mode | What is reviewed |
|---|---|
| `directory` | Every file under `target.path` except what `.gitignore` and the `exclude` globs remove. Scope is denylist-only — there is no allowlist option, since one has to be re-derived per language and silently hides whatever it forgets. |
| `git-diff` | Changes relative to `target.base_ref`. The base is resolved to a concrete commit once at run start, so per-round fix commits extend the reviewed diff instead of shrinking it. A trailing `...` (`origin/main...`) pins the **merge base** with HEAD instead of the ref's tip — what this branch added, which is what `fix-branch` uses. Without it, any commit the base branch has and yours does not appears in the diff *reversed*, and the panel reviews someone else's work as deletions you made. Empty `base_ref` reviews unstaged working changes, so it is review-only: a fix run needs a base ref (it starts from a clean tree, which makes the unstaged diff empty) and is rejected at validation without one. |
| `pr` | A GitHub pull request, named by `target.pr` or per run with `-pr`. The PR branch is checked out locally (`gh pr checkout`) and reviewed against its base, so fixes land in the working tree and later rounds review them too. Pushing fixes back is manual. |

Fix rounds require `target.path` to be a git repository with a clean working
tree at run start (in every mode); `review_only` runs anywhere.

### Choosing a base in git-diff mode

`base_ref` is handed straight to git, so anything git resolves works — in the
config, or per run with `-base-ref`:

| `base_ref` | What gets reviewed |
|---|---|
| `@{upstream}...` | This branch against the branch it tracks. The `fix-branch` default. |
| `origin/HEAD...` | This branch against the remote's default branch — no upstream needed and no branch name hard-coded, so it is the best generic choice. Needs the local ref, which `git remote set-head origin -a` creates. |
| `origin/develop...` | An explicitly named trunk. |
| `@{push}...` | Triangular workflows, where you push somewhere other than you fetch from. |
| `HEAD~3` | The last three commits. |
| `v1.4.0` | Everything since a release tag. |
| `ORIG_HEAD` | Whatever the last rebase, merge, or reset moved past. |
| `HEAD@{yesterday}` | What you have done since yesterday. Read from the reflog, so it is per-clone and approximate. |

**When the `...` matters.** Only when the base can hold commits your HEAD does
not. For a **branch name it is essential** — without it you diff against that
branch's tip, and every commit it has that you lack shows up *reversed*, so the
panel spends a round reviewing someone else's work as deletions you made. With
two machines pushing to one repository that is hours, not months. For anything
already behind you — `HEAD~3`, a merged tag, a SHA on your own history —
`merge-base(X, HEAD)` is just `X`, so the dots are a harmless no-op.

Short version: **a branch name takes the dots; anything already in your history
does not.**

Getting this wrong is silent — both spellings resolve, and only the diff says
which one you meant — so `--check` prints the commit the base resolved to and
the size of the diff it selects, before any agent runs:

```
$ fixpoint fix-branch --check --trusted-target
scope: base_ref "@{upstream}..." -> d122ea0dd503; 9 files changed, 313 insertions(+), 12 deletions(-)
configuration OK: 5 review lens(es), coder claude-coder, strategy rotate
```

A base that resolves to nothing fails there rather than after the first round
has been paid for. `--check` reports scope in `directory` mode too (`77 file(s)
in scope`); in `pr` mode it cannot, because finding out would mean running
`gh pr checkout` and switching your branch.

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

An entry with no wildcard at all names a *directory* as well as a file:
`config/secrets` — or `config/secrets/`, the two are the same entry — excludes
everything beneath it, the way a git pathspec prefix does. One carrying a wildcard
anywhere (`*.env`, `**/node_modules`) describes a file shape and matches names
only; to take the contents of every matching directory too, say so:
`**/node_modules/**`.

Your entries match case-sensitively, as written. The credential patterns fixpoint
enforces on top of them do not: `PRODUCTION.ENV` and `ID_RSA` are dropped the
same way their lowercase spellings are.

## Review lenses and assignment strategies

### Write a lens prelude-first, or the round pays per lens

Every shipped lens template opens with `{{.Prelude}}` and puts its own
instructions *after* it. That ordering is load-bearing rather than stylistic.

Anthropic's prompt cache matches on an exact **leading prefix**, so a template that
opens with its own role line ("You are an expert security reviewer…") diverges from
its siblings at byte one, and the round pays for the material once per reviewer.
`{{.Prelude}}` is every part that is identical for all lenses in a round — target
header, mode guidance, working rules, material, history — rendered from one place
so that prefix is shared. Measured through the harness with a ~47k-token prompt:
two calls sharing only a prefix and differing in their tail, and the second read
39,552 tokens from cache. In `git-diff` mode the material alone runs to 220 KB.

It also puts the instructions after the document, which is what Anthropic
recommends for long inputs anyway.

The individual fields (`{{.Target}}`, `{{.History}}`, …) stay available if you want
to lay a prompt out yourself — at the cost of that sharing.
`TestShippedLensesShareARenderedPrefix` renders every shipped lens and fails if one
emits anything before the prelude, because the only symptom otherwise is a bill.

Prompts under [config/prompts/](config/prompts/) are a library you can grow freely; only the
ones referenced in the configuration are used. The shipped lenses:

- [review-bugs.md](config/prompts/review-bugs.md) — correctness
- [review-security.md](config/prompts/review-security.md) — security
- [review-concurrency.md](config/prompts/review-concurrency.md) — concurrency
- [review-tests.md](config/prompts/review-tests.md) — test coverage of what the run changed
- [review-maintainability.md](config/prompts/review-maintainability.md) — smells, simplification, docs (advisory; **shipped but not in any panel**)
- [review-design.md](config/prompts/review-design.md) — architecture (advisory; **shipped but not in any panel**)
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
  what makes the phase stop on its own. `loop.max_final_passes` (default 1) bounds
  it as a last resort, and running out is always said loudly rather than dropped
  silently — either issues are still open, or the last pass fixed everything it
  reported and there was no pass left to review those fixes.

  Its own knob rather than `max_iterations`, because this phase is where a measured
  run spent 51 minutes and still had pass 2 producing four *new* issues: each pass
  reviews the tests the previous pass just wrote. Every repeated issue id in that
  run came from here — a real bug fixed in the loop, re-opened as "the test for that
  fix is flaky", then as "the test for the test" — while the loop's own rounds did
  not repeat themselves at all.

  **The default is 1**, lowered from 2 on a second measurement. Pass 2's premise was
  "confirm the fix did not open something new"; what it actually did, over a
  seven-round run, was file 4 critiques of the tests pass 1 had just written, 2
  restatements of what pass 1 already reported, and 1 real regression — which pass
  1's own fix had introduced. A pass whose main yield is repairing the previous pass
  is not converging on the code, and what reliably catches a broken fix is the
  verify gate, which runs per fix. Raise it to 2 when your closing lenses are
  list-shaped (a fixed set of gaps to work through) rather than opinion-shaped, and
  pair that with `loop.final_skip_run_edits` below.

  **`loop.final_skip_run_edits`** keeps the closing round from reviewing its own
  output. It is a glob list, matched against the paths *this run's own commits*
  changed; a match is hidden from the closing round's material only. Empty by
  default; the shipped Go configs set `["**/*_test.go"]`.

  It is not a general "don't review your own work" rule — inside the loop that
  review is productive, and it is how a fix's own bug gets caught. It targets one
  feedback loop that has no fixed point: `review-tests` asks for a test, the coder
  writes one, and the next look reviews *that test* ("the assertion claims more than
  it proves", "the timing is wall-clock noise", "only one branch is exercised"). In
  the run above, 7 of the closing phase's 14 findings were exactly that, every one
  against a test file the run had committed minutes earlier. Hiding *everything* the
  run touched was measured and rejected: the same run changed 27 of 57 source files,
  so the blunt rule blinds the closing round to half the tree — including the newest
  code, which is the code most worth a coverage review.

  **Advisory final lenses run exactly once, after that** — they're reports, and a
  report should describe the code that actually shipped, which isn't known until the
  fixing stops. They never share a round with the fix passes, so a three-pass
  closing round still produces one design report, not three.

  It runs after **every** normal termination — converged, all-rejected, and
  max-iterations alike — since the loop is done editing in all three, but not after
  an error or interruption, when the tree is in a state nobody vouched for. It does
  not change the run's termination: it is extra work on an already-decided run. The
  exception is being interrupted itself — closing work is then left undone, so the
  run ends as `interrupted` (exit 1) with the loop's own outcome kept in the
  summary's `loop termination` line, exactly as a failed closing round does.
  `final` and `once` are mutually exclusive, and a review-only run has no closing
  round (there is no coder), so a final lens simply runs in its single round.

  **Pin a final lens whose findings are advisory.** Unpinned means the whole panel,
  which is right when the findings get fixed — nothing follows to catch what one
  model missed. For a report a human reads, it means four overlapping documents and
  corroboration that buys nothing, since nothing gets scheduled.

  The shipped configs now use only the first shape — `review-tests`, unpinned and
  fixed. `review-design` and `review-maintainability` were pinned and reported, and
  were removed from every panel because nobody read them: an unread report still
  costs a full agent session per run. Both prompts remain in the bundle, so the
  advisory shape is one line away if that changes.

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
  no line is given. A shared location is where two reports *may* be about one
  defect; agreeing titles are what say they are, so reports in one file merge at any
  distance when their titles agree and stay apart when they do not — even on the
  same line, since one statement routinely holds two defects and one issue carries
  one verdict. Merging them would tell the coder to fix one thing when there are
  two, which is worse than leaving a duplicate that merely costs a slot. The
  category is deliberately **not** part of identity — the real duplicate above
  arrived under two different ones.
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

### Rejection is also a value judgment

The shipped `fix` prompt authorizes the coder to reject a finding that is *correct
but not worth making*, not only one that is wrong. Because rejection is durable,
this is the loop's one damper: a fix costs a commit, a verify run, and whatever the
next round writes about the code it added, so a change that buys less than that is
a net loss. The prompt names the recurring shapes — a finding that restates a
decision the code already documents, a request for a test of a test, a speculative
defect with no input that triggers it, a refactor riskier than the edge case it
removes.

Two limits keep it from becoming a way out of work, and both are in the prompt: a
`high` or `critical` finding may be rejected only for being *wrong*, never for
being expensive, and every rejection reason is recorded in the run summary for a
human to read. Rejecting on value is a judgment the operator can audit afterwards,
which is the only reason it is safe to delegate.

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
discarded whole. Each pass runs every configured command rather than stopping at the
first failure: the coder gets one bounded correction attempt, so it has to see every
failure at once — fixing the formatter and only then discovering on the retry that
the test suite is red too would waste the round. Ordering `verify.commands`
cheapest-first therefore saves no time, but it still reads best: the log summary and
the failure block handed to the coder follow the configured order.

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
[effort (claude)](config/agents/_README.md) — so `effort` on an ollama agent
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
asserted on the command line. `-trusted-target` says the target's policy may be
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
make help           # every target, including one per shipped config
```

There is one target per shipped config, so the command says which one it is
going to spend money on. (There is no `make run`: it took its config from a
variable, which made the two-hour run and the five-minute one look identical.)

```sh
make review-code            # review the whole project, no edits
make review-branch          # review only what this branch changed, no edits
make review-pr PR=170       # review a pull request
make fix-code               # review -> fix -> verify -> commit, whole project
make fix-branch             # the same cycle over this branch's changes
```

Or directly:

```sh
./fixpoint <config-name> [flags]      # e.g. ./fixpoint review-code
./fixpoint --config path/to/task.yaml [flags]
```

### Fixing a pull request

```sh
./fixpoint review-pr -pr 170                        # read it first
./fixpoint fix-pr -pr 170 --allow-untrusted-fix     # then let it edit
./fixpoint fix-pr -pr 170 --allow-untrusted-fix -post   # …and answer the threads
```

`fix-pr` also **triages the pull request's open comments**: a read-only agent
decides each one before any fixing starts, accepted ones become ordinary issues
with their own session, gate and commit, and declined ones are answered with the
reason. Every conversation gets a decision: a declined one is answered straight
away, an accepted one once its fix commits — and if the coder then rejects that
issue or its gate fails, nothing is claimed and the thread is left for a later
run to decide again. That lets PR comments
direct work, which is a real widening of what untrusted text can ask for — see
[config/README.md](config/README.md#letting-the-comments-commission-work) for what
bounds it.

Conversations are read **whole**, replies included, and one whose last word is
already this tool's answer is left alone — otherwise every later run would answer
the same comment again, since replying does not resolve a thread. When a person
writes back, the thread is live again and the agent sees the entire exchange,
including what it said last time.

`fix-pr` is the most dangerous config in the bundle and its flag says so: it edits
a working tree holding **externally authored** code, with an agent whose
permission checks are disabled. A payload in the diff, in a commit message, or in
a review comment reaches the coder. It is also shown the pull request's
unresolved conversations and may answer the ones its work addressed — see
[config/README.md](config/README.md#answering-a-pull-requests-conversations). A
reply is posted only once that session's fix has been **committed**: it goes out
under your identity as a claim that the work landed, and a session whose gate failed
or whose issue was rejected has its edits withdrawn or stashed, so a reply about it
would be a claim about work that does not exist.

### Reviewing a pull request

```sh
# 1. See what it would review, and what it would cost, before spending anything.
./fixpoint review-pr -pr 170 --check

# 2. Review it. The result is written to .fixpoint/<run>/review-body.md and
#    nothing leaves this machine.
./fixpoint review-pr -pr 170

# 3. Read that file. Then publish THOSE bytes as a comment -- no approval, no
#    block, no second review, and no agent invoked:
./fixpoint -post-run .fixpoint/<run-timestamp>

# 4. Or let the published review carry its verdict, approving or requesting
#    changes:
./fixpoint -post-run .fixpoint/<run-timestamp> -post-verdict
```

Step 3 is `-post-run` and not a second `review-pr -pr 170 -post` because the two
are not the same act. `-post` publishes the review the run in front of it just
produced, so on its own it publishes a review nobody has read; re-running to post
what you read would re-review everything, cost the same again, and publish a
*different* review — the panel is not deterministic, and two runs over one pull
request here produced 34 findings and then 49. `-post-run` takes the run directory
and posts the body off disk, with the anchors that run computed. That is what makes
reading the file first mean anything. It refuses a run that has no verdict (a fix
run), one that did not review a pull request, one from before the PR number was
recorded, one that did not finish — the verdict is written before the round
checks whether it was interrupted, so a review stopped part-way leaves an approval
on disk that the run itself refused to post and exited non-zero over — and one
that has *already* been published, so `-post` followed by `-post-run` cannot leave
two identical reviews on the pull request. Publishing also leaves a `review-posted`
receipt in the run directory, so a second `-post-run` over the same run refuses
too — including after a submission that failed *after* it was sent, which the forge
may well have accepted; the receipt says so, and only you can decide whether the
review is there.

It also refuses a run directory that is **tracked by git**. A run directory is only
files, so a pull request can commit a lookalike `.fixpoint/<run>` next to your own:
a plausible `review-body.md` to read, and a summary whose fields choose a different
pull request, an approval, and inline comments of its own — none of which are in the
file you inspected. fixpoint commits no run artifacts, so a tracked file there means
the directory came in with the code under review rather than from a run on this
machine, and nothing is published. Every anchor's file and line is also logged
before the submission goes out, because the line comments are separate bytes from
`review-body.md` and reading that file does not show them.

Either way, the review is bound to the **commit it was about**. A run records the
head it reviewed, and posting reads the pull request's current head first: if the
author has pushed since — while the panel ran, or in the days between a run and its
`-post-run` — nothing is published. A forge applies a review to whatever the pull
request points at now, so without that check an approval could clear code no
reviewer ever read: push something clean, collect the approval, push the payload.
On GitHub the submission also names that commit (`commit_id`), so the review is
recorded against it and its comments are marked outdated if the branch moves
afterwards.

The **approval itself is not**, and no client can make it so. A forge counts an
approval toward its merge requirements until something clears it, so once fixpoint
has exited an author can push and merge on an approval given for the commit before
— exactly what the check above refuses *during* the run. Only the repository can
close that: on GitHub enable *Dismiss stale pull request approvals when new commits
are pushed*, on GitLab *Remove all approvals when commits are added to the source
branch*. Turn it on before you let anything approve with `-post-verdict`, machine
or human — every approval fixpoint publishes says so in the log, naming the commit
it was for.

It is bound to the **repository** it was about too. A commit is not a destination:
the run directory records the path it ran in, and `-post-run` submits through
whatever checkout is at that path when you run it — which, days later, may have been
reused for another repository on the same forge or had its remotes rewritten. Pull
request 170 of *that* repository would then receive the review, and the head check
would not notice, because the reviewed commit is public and anyone can open a pull
request proposing it. So a run also records the repository `gh` resolved for the
checkout, and `-post-run` refuses unless the checkout still resolves to it — or
cannot say what it resolves to at all.

`-post`/`-post-run` and `-post-verdict` are separate flags because they are
separate acts. The first makes a machine review visible; the second approves
somebody's change or formally blocks it. None of them can be set from a config
file — the first bundle on the search path belongs to the target, so a YAML key
would let reviewed code arrange to have a review posted under your identity.

Reviewing the same pull request twice is fine: findings it already carries are
recognized and not repeated, the new ones are published, and the body says how
many it left out. Recognition is bound to the commit each finding was published
about, so one whose code has been pushed to since is published again in full rather
than counted as old news — see
[config/README.md](config/README.md#reviewing-the-same-pull-request-twice).

Findings that name a file and a line are also posted as **inline comments**, so a
reader meets each one beside the code instead of in a list at the bottom. Each one
carries the same signature as the body, because it is read in the Files tab with no
sight of the review it belongs to — unsigned, it would be an unattributed assertion
sitting on somebody's code. A forge only accepts an anchor on a line the pull
request actually touches; if it refuses any of them it refuses the whole
submission, so fixpoint retries with the summary alone and says so. Nothing is lost
either way — every finding is in the body.

> **GitLab is not usable today.** There is a `glab`-based provider — it posts a
> note, adds inline discussions, approves and unapproves — but nothing can reach
> it: `pr` mode's `Prepare` unconditionally runs `gh pr checkout`, so a run against
> a GitLab remote fails before any provider is chosen. The code is written and
> tested against a fake CLI; what is missing is a checkout path that is not `gh`.
> Treat every mention of GitLab below as describing code that has never run
> against a real merge request.

**GitHub refuses an approval or a change request on your OWN pull request**, so
`-post-verdict` only does anything when the token belongs to somebody other than
the PR's author — a bot account, or a reviewer running it on a colleague's branch.
`-post` works either way, which is part of why the comment is the default: on your
own PR it is the only thing that can land. A refusal is reported, not swallowed,
and the review is still on disk.

| Flag | Effect |
|---|---|
| `<name>` | Positional: the task config to run, resolved on the bundle search path. A value containing a separator or ending in `.yaml` is used as a path. |
| `--list` | List task configs with the file each resolved from, and exit. |
| `-config path` | Alternative to the positional name. Giving both is an error. |
| `-review-only` | Run exactly one review round; the coder is never invoked (no edits in git-diff/directory mode; pr mode still runs `gh pr checkout`, switching the branch and working tree in Prepare). |
| `-max-iterations n` | Override `loop.max_iterations`. |
| `-base-ref ref` | Override `target.base_ref` in git-diff mode; a trailing `...` means the merge base with HEAD. For `fix-branch` on a branch with no upstream: `-base-ref 'origin/main...'`. |
| `-pr n` | Override `target.pr` in pr mode. `review-pr` ships with no number, so this is how you say which PR: `fixpoint review-pr -pr 1234`. |
| `-trusted-target` | Assert a directory/git-diff target holds only trusted code, permitting fix rounds (fail-closed without it). |
| `-trusted-bundle` | Assert **only** that the bundle files resolved from inside the target may be executed and sent to agents. Permits no fix round and trusts no other target content — this is the flag to use when the config is yours but the code is not, as `review-pr` on a fork's branch is. |
| `-allow-untrusted-fix` | Permit fix rounds in `pr` mode (PR content is untrusted; see Security). |
| `-post` | Publish the review on the pull request as a **comment**: findings become visible, no verdict is acted on. Publishes what the run just produced, so nobody has read it yet — prefer `-post-run`. A publish that was asked for and did not happen fails the run (exit `1`), so an approval can never exit `0` over a review that never reached the pull request; the review is still in `review-body.md`. |
| `-post-run dir` | Publish the review a **finished** run already produced, from its `.fixpoint/<run>` directory (or its `summary-*.json`). Invokes no agent and reviews nothing: the bytes posted are the bytes in `review-body.md` and the inline anchors are the ones that run computed. Resolves no configuration at all — everything it acts on is in that run's summary — so every flag but `-post-verdict` is ignored. |
| `-post-verdict` | With `-post` or `-post-run`, let the review carry its verdict — approving, or requesting changes on someone's PR. An inconclusive verdict stays a comment regardless. An approval is bound to the reviewed commit only until the run ends: unless the repository dismisses stale approvals on push, it keeps counting after one (see above). |
| `-check` | Validate the configuration, report how much material the run would review, and exit. No agent is invoked. See [Choosing a base](docs/concepts.md#choosing-a-base-in-git-diff-mode). |
| `-check-live` | Validate, ping every agent, and exit. |

Exit codes: `0` converged, or a review that **approved**; `2` hit
`max_iterations` without converging (or a usage error); `3` the coder rejected
every issue so nothing changed — deliberately *not* `0`, since "nobody agreed
there was a problem" is not "the code is clean"; `4` the review **requested
changes**; `5` the review was **inconclusive** (nothing blocking was found, but
the panel did not reach quorum or the judge did not finish, so that silence is
not evidence); `1` any other failure or interruption. A verdict only ever makes
the status worse, so an errored or interrupted run keeps its own code. `SIGINT`/`SIGTERM` stop the run cleanly: the current step is abandoned and any edits
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
| [review-code](config/review-code.yaml) | Review a whole project once, no edits. Needs `-trusted-bundle` (or `-trusted-target`) if the project ships its own bundle. |
| [review-branch](config/review-branch.yaml) | Review only what this branch changed, no edits. The review-only twin of `fix-branch`. |
| [review-pr](config/review-pr.yaml) | Review a GitHub pull request; review-only by default. |
| [fix-pr](config/fix-pr.yaml) | Fix a pull request's changes and triage its open conversations; the replies are posted only with `-post`. Needs `-allow-untrusted-fix`, and `-trusted-bundle` as well when the target ships the bundle being used. |
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
  assignment strategy. The reviewer **pool** lives in `defaults.yaml` and is
  inherited, so changing who reviews is one edit rather than one per config; a
  config that sets `roles.review.agents` replaces that pool instead of adding to
  it.
- **`agents`** — the command templates described above, each optionally carrying
  `prompt_budget` (bytes; over it the invocation is refused before the process
  starts, and the step is recorded as failed rather than sent and rejected by the
  provider — see [config/README.md](config/README.md)).
- **`loop`** — `max_iterations`, `max_final_passes` (how many times the closing
  round may repeat, default 1), `final_skip_run_edits` (globs the closing round is
  not shown when this run wrote the file), `commit_policy` (see
  [One fix, one commit](docs/concepts.md#one-fix-one-commit)), `max_findings_per_round` (caps how
  many **issues** a round hands over, `0` = unlimited and the default; worst
  severity goes first and the overflow is deferred to later rounds, but every
  deferral promotes an issue one severity tier, so nothing can be starved
  indefinitely by a steady supply of more-severe ones), `review_only`,
  `commit_message` (placeholders `{issue}`, `{title}` for a single fix; `{round}`,
  `{fixed}`, `{rejected}` for a squashed one), and `clean_rounds_to_stop` (how many
  consecutive clean rounds end the run — `2` pairs well with `strategy: rotate`, so
  a differently-assigned panel must confirm the clean result). The trust gates are
  deliberately **not** here: they are command-line flags only, for the reason given
  under [Security model](docs/security.md).
- **`review`** — what a review run concludes and how it says so: `block_at` (the
  severity that forces CHANGES_REQUESTED, default `high`), `refute_at` (the floor
  for the refutation round, default `high` — it may be looser than `block_at` but
  never stricter), `refute` (the prompt, or empty to skip the round), and the two
  signature templates, `signature` and `reply_signature`. Publishing is **not**
  here: `-post`, `-post-run` and `-post-verdict` are flags only, for the reason
  given under [Security model](docs/security.md).
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
internal/review/         the verdict rules and the review document (body, inline
                         comments, signature)
internal/forge/          reads a pull request's checks and conversations, and
                         publishes the review back through the vendor's CLI (`gh`)
internal/logstore/       per-step logs, the run journal, and the run summary
internal/runlog/         renders the run's progress: phase blocks, color on a tty
internal/testfixture/    shared test helpers
config/                  the shipped bundle: task configs, prompts/, agents/
config/defaults.yaml     the commented base every task config extends
docs/                    the long-form documentation this README links to
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
