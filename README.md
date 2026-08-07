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
reason. Every conversation ends with a reply either way. That lets PR comments
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

`-post`/`-post-run` and `-post-verdict` are separate flags because they are
separate acts. The first makes a machine review visible; the second approves
somebody's change or formally blocks it. None of them can be set from a config
file — the first bundle on the search path belongs to the target, so a YAML key
would let reviewed code arrange to have a review posted under your identity.

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
| `-post-verdict` | With `-post` or `-post-run`, let the review carry its verdict — approving, or requesting changes on someone's PR. An inconclusive verdict stays a comment regardless. |
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
- **`agents`** — the command templates described above.
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
