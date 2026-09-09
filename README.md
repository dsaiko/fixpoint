# fixpoint

An agent-agnostic automation loop for writing, reviewing and fixing code with
AI panels — where every claim a model makes is checked by something that is not
a model.

A panel of reviewer agents inspects a target in parallel, each through one
focused lens (bugs, security, concurrency, tests, design). Their findings are
deduplicated, contested by a refutation round, and filtered by a judge. A coder
agent then fixes what survives, and fixpoint runs your project's own build and
tests **itself** before anything is committed. That gate is the point: without
it, "fixed" means an agent said so, and "clean" means other agents said they saw
nothing.

The name is the termination condition: the loop iterates review→fix until the
code stops changing — a [fixed point](https://en.wikipedia.org/wiki/Fixed_point_(mathematics)),
reached when a full panel reports nothing left to fix.

Any agentic CLI can be a reviewer or a coder: an agent is just a command that
receives a prompt and prints text to stdout. The shipped panel mixes Claude
Code, Codex and models served through ollama, and nothing in the code is
provider-specific.

---

## What you can use it for

Three things, in increasing scope. Every command below is real and ships in the
box.

### 1. Review code that already exists

Nothing is modified. The panel reads, the findings land in
`.fixpoint/<run>/review-body.md`, and the run's exit code carries the verdict.

```sh
make review-code                      # the whole project
make review-branch                    # only what this branch changed
make review-pr PR=170                 # a GitHub pull request
make review-pr                        # …or your own PR, from its branch
make review-design TARGET=docs/DESIGN.md   # a design document, or an architecture
```

Use it when you want a second opinion and nothing else: before a release,
after a big merge, or on somebody else's branch. `review-design` is the odd one
out — it reads a *document* (or a whole project as an architecture) and asks
about structure, data ownership and failure modes rather than about lines.

Cost first, always:

```sh
./fixpoint review-code --check        # what it would review, and how much material
./fixpoint review-code --check-live   # …and ping every agent, so a dead login surfaces first
```

### 2. Let it fix what it finds

The autonomous cycle: review → fix → verify → commit, repeated until the panel
comes back clean or the iteration cap is reached. Each fix is its own commit,
naming the issue it closed, and **every commit has passed your gate**.

```sh
make fix-code                         # the whole project
make fix-branch                       # only this branch's changes
make fix-pr PR=170                    # a pull request, conversations included
make fix-pr                           # …or your own PR, from its branch
```

These edit your working tree with an agent whose permission checks are
disabled, so they refuse to start without an explicit trust assertion on the
command line — `--trusted-target`, or `--allow-untrusted-fix` for a pull
request, whose content is somebody else's. The assertion is a flag and never a
config key, because the first bundle on the search path belongs to the target:
see [Security model](docs/security.md).

A round that fails the gate is **discarded**, not committed — edits stashed for
you to inspect — because committing them would put every later round on a
broken base.

### 3. Design → review → build a whole project

The full cycle, from a paragraph of intent to a repository with a gated commit
per task. Each stage is a separate command, so you read the product of each one
before paying for the next.

```
   assignment.md
        │
        ▼
   create-design ──────► DESIGN.md          a panel proposes independently,
        │                                   critiques anonymously, one editor
        │                                   synthesizes, dissent recorded
        ▼
   review-design ─────► findings            structure, data, failure modes
        │                                   — read them, revise the design
        │  (repeat until the design holds)
        ▼
   implement-go ──────► a new repository    one planner session decomposes it,
   implement-node                           the coder builds one task per
   implement-web                            gated commit
        │
        ▼
   review-code ───────► findings            now review the code that was built
   fix-code   ────────► fixes               …and let it fix them
        │
        ▼
   review-pr / fix-pr ► a reviewed PR       when the work goes out for merge
```

**Building an application** — a browser game, no build step, gate asserted off:

```sh
make create-design TARGET=assignment.md OUT=docs/DESIGN.md
make review-design TARGET=docs/DESIGN.md          # read the findings, revise, repeat
make implement-web TARGET=docs/DESIGN.md OUT=~/src/prsi

cd ~/src/prsi && ~/src/fixpoint/fixpoint review-code --trusted-target
```

**Building a library or a tool** — a Go module with a real gate:

```sh
make create-design TARGET=assignment.md OUT=docs/DESIGN.md
make review-design TARGET=docs/DESIGN.md
make implement-go TARGET=docs/DESIGN.md OUT=~/src/newtool

cd ~/src/newtool && ~/src/fixpoint/fixpoint fix-code --trusted-target
```

The difference between the two is the **gate**, and it is the only difference
that matters. `implement-go` runs `go build`, `go vet` and `go test` after every
task, so a task commits only if the project still builds. `implement-web` is
plain HTML and JavaScript with no build step and therefore no gate — which is a
statement the config has to make out loud (`verify.policy: off`), because on a
project that does not exist yet, silence is far more likely to be a
half-finished config than a decision. Quality for an ungated stack comes
afterwards, from `review-code` over the produced repository.

The last step of each example runs the **binary by path**, because the make
targets live in this repository and the project you just built does not have
them. Putting the binary on your PATH is enough to run it from anywhere — a symlink
into this checkout works, because the search path includes the bundle beside the
resolved binary. Copy `config/` to `~/.fixpoint/` when you want your own edits
to win over it; see [config/README.md](config/README.md) for the full search
path.

### Reading the plan before you pay for it

The plan decides how up to forty coder sessions are spent, so it is worth a
minute of your own. `-plan-only` produces one and stops — no directory is
claimed, no coder runs:

```sh
fixpoint implement-go -target docs/DESIGN.md -plan-only
$EDITOR .fixpoint/<run>/plan.json
fixpoint implement-go -target docs/DESIGN.md -out ~/src/prsi \
    -plan .fixpoint/<run>/plan.json --trusted-target
```

A plan handed back this way is held to every rule the planner's answer is held
to, and it must carry `provenance.design_sha256` matching the design you point
at — a plan for one document must never be implemented against another. The
refusal prints the line to paste. It composes with the panel too:
`-plan-only` → `review-design -target PLAN.md` → `-plan`.

`-no-coverage-check` waives the rule that every design section is accounted for,
for documents whose heading structure fixpoint cannot read. Every report then
says `coverage: unchecked` rather than implying a check that never ran.

### Finishing a run that stopped

A run that stops early — deadline, provider outage, Ctrl-C — leaves every
committed task standing and gated, and **can be resumed**:

```sh
fixpoint implement-go -continue ~/src/prsi --trusted-target
```

The project is self-describing: the plan, the design and every finished task's
outcome are read out of its own commits, not from the artifacts of the run that
stopped. It refuses if the tree is dirty, if the design or the gate is not the
one the project was built under, or if the history is not entirely fixpoint's —
each of those means the repository cannot be shown to be the one the plan was
built into. Finished tasks are carried, never rebuilt, and a task recorded as
failed stays failed. Specification: [docs/design/DESIGN.md](docs/design/DESIGN.md).

---

## Where to read what

| | |
|---|---|
| **[Getting started](#getting-started)** | Requirements, install, and the make targets. |
| **[How a round works](#how-a-round-works)** | One review→fix round, end to end. |
| **[What a run actually does](docs/concepts.md)** | Target modes, the review round in detail, how observations become issues, one fix per commit. |
| **[Pull requests](docs/pull-requests.md)** | Reviewing and fixing a PR, and everything publishing a review under your identity is bound by. |
| **[Review lenses](docs/lenses.md)** | What a lens is, how to write one, and how they are assigned to agents. |
| **[Agents](docs/agents.md)** | Adding an agent, borrowing a harness for a model with no CLI, why the route matters, environment filtering. |
| **[Logs and artifacts](docs/logs.md)** | What a run writes, the end-of-run table, the run journal. |
| **[Security model](docs/security.md)** | What is trusted, what is not, and why the trust gates are flags rather than config keys. |
| **[Which model is worth a panel seat](#which-model-is-worth-a-panel-seat)** | The reviewer benchmark: every candidate model scored against planted defects in seven languages and a design doc. |
| **[Configuration](#configuration)** | Bundles and the shipped configs. The reference for every setting is [config/README.md](config/README.md) and the comments in [config/defaults.yaml](config/defaults.yaml). |

## Getting started

### Install a release

Releases carry a tar.gz per platform plus deb, rpm, apk and Arch packages, all
built from a `v*` tag by [GoReleaser](.goreleaser.yaml). Every archive contains
the **config bundle** as well as the binary, because fixpoint compiles no
fallback configuration into the executable: the prompts drive an agent that
edits files with permission checks disabled and the configs hold the trust
gates, so a binary on its own refuses to run and prints where it looked.

The repository is private, so the assets need an authenticated download rather
than a bare `curl`, and there is no Homebrew tap — a formula fetches by URL and
brew cannot authenticate here.

No tag in the commands below: `gh release download` takes the latest release
when none is given, so these stay correct across releases rather than pinning a
version that goes stale the next time one ships. Name a tag to pin one.

```sh
# macOS (Apple silicon); swap Darwin_arm64 for Linux_x86_64, Linux_arm64, Darwin_x86_64
gh release download --repo dsaiko/fixpoint --pattern '*Darwin_arm64.tar.gz'
tar xzf fixpoint_*_Darwin_arm64.tar.gz
./fixpoint -version
./fixpoint --list
```

```sh
# Debian/Ubuntu
gh release download --repo dsaiko/fixpoint --pattern '*linux_amd64.deb'
sudo dpkg -i fixpoint_*_linux_amd64.deb    # /usr/bin/fixpoint + /usr/share/fixpoint
```

Keep `config/` beside the binary, or copy it to `~/.fixpoint/` to edit your own
copy — the search path finds both, and yours wins. Packages install the bundle
to `/usr/share/fixpoint`, which the same search path already knows.

Windows is not built. See [Platform support](#platform-support).

### Build from source

Requirements:

- Linux or macOS. **Windows is not supported yet** — see [Platform support](#platform-support).
- Go 1.26+
- git (and `gh` for pull-request mode)
- at least one agentic CLI installed and authenticated (e.g. `claude`, `codex`,
  `ollama`) — verify the flags in `config/agents/*.yaml` match what your
  installed versions expect

```sh
make build          # compile the fixpoint binary
make check          # static validation of the configuration (no agents invoked)
make check-live     # static validation + ping every configured agent
make help           # every target, including one per shipped config
```

There is one make target per shipped config, so the command says which one it is
going to spend money on. (There is deliberately no `make run`: it took its config
from a variable, which made the two-hour run and the five-minute one look
identical.)

Or call the binary directly, which is how you reach the flags the make targets
do not expose:

```sh
./fixpoint <config-name> [flags]      # e.g. ./fixpoint review-code
./fixpoint --config path/to/task.yaml [flags]
```

## How a round works

```
one round:

  review-bugs ─────────► agent A ─┐
  review-security ─────► agent B  │   observations      issues
  review-concurrency ──► agent C  ├──► (raw reports) ──► (deduped) ──► coder ──► verify ──► commit
  review-tests ────────► agent D  ┘                                    fixes    build/test

repeat until a full reviewer panel reports nothing
```

1. **Validate.** Before anything runs, the configuration is statically checked:
   every referenced agent is defined, its binary exists on PATH, prompt files
   parse and their placeholders resolve, the coder's agent has `can_edit: true`,
   and the lens-assignment strategy is satisfiable. Any failure aborts before a
   single agent is invoked. With `ping_agents` (the default) every agent the run
   will use is also pinged with a trivial prompt, so expired logins and broken
   CLIs surface before tokens are spent or git is touched.
2. **Review.** Each lens is assigned to an agent per the configured strategy and
   all reviewers run in parallel, each reporting structured observations
   (category, severity, file/line, description, suggestion).
3. **Aggregate.** Observations are grouped into **issues** — see
   [Observations and issues](docs/concepts.md#observations-and-issues). Several
   reviewers reporting one problem produce one issue, so agreement between
   agents raises confidence instead of consuming the round's budget twice.
4. **Fix.** The issues go to the coder agent, which validates each one: it fixes
   the genuine ones by editing files directly and rejects the rest with a reason.
5. **Verify.** fixpoint runs the project's own build, test and static checks
   *itself* — see `verify` in the config. This is the only signal in the loop
   that no model produced. A round that fails the gate gets one bounded
   correction attempt from the coder; if it still fails, the round is discarded
   (edits stashed, not committed). Under the default `no_regressions` policy a
   check that was already failing before the run may keep failing — only newly
   broken checks block.
6. **Commit.** Each fix lands as its own commit, naming the issue it closed.
   `loop.commit_policy` decides whether those stay separate (`per_fix`, the
   default) or are squashed per round or per run.
7. **Repeat.** Each round, reviewers receive the history of prior issues and
   coder verdicts, so a rejected issue is not re-reported forever. The loop ends
   after a configurable number of consecutive clean rounds, when the coder
   rejects everything in a round, or at the iteration cap.

Three behaviours are worth knowing before you read a run:

**A coder that dies mid-round** (timeout, session limit, malformed output) after
editing files has its partial work put through the same gate as a normal round.
If it passes, it is committed as a "partial" round and the loop continues — the
next round re-reviews everything, so the run self-heals instead of stranding
valid edits. If it fails, the edits are stashed for you to inspect (`git stash
pop`) and the run stops rather than building later rounds on a broken base.

**Reviewers are told not to build or test.** They were doing it — `go test ./...`
inside a *review* — and every line of that output returns as input tokens on the
next turn of a session whose turn count is what drives the bill. It buys nothing
either: the gate above is fixpoint's own run, and a reviewer's private one does
not feed it. The rule lives in code (`prompt.ReviewWorkingRules`) so a new lens
inherits it.

**A reply that breaks the format gets one chance to restate it.** When an agent
exits cleanly but its `<review>` block is missing or malformed, fixpoint asks it
to emit the findings again in the required shape — without re-sending the
material and without letting it redo the review, so the second pass cannot
quietly report different findings. The alternative is a whole agentic session
discarded over its punctuation, plus a reviewer error that resets the
convergence streak. Both attempts' usage is billed to the one step. A crashed,
timed-out or rate-limited agent is *not* retried this way — it has nothing to
restate.

## Which model is worth a panel seat

Every candidate reviews the **same frozen targets** — one link-shortener service
written seven times, once per language, plus a design document — and is scored
against defects planted in them. No judge and no taste: a deterministic matcher
credits each finding to at most one seed, so the number is repeatable and two
models are compared on the same ground. Nobody scores full marks, and that is
the point; what matters is the distance to the `claude` baseline and what the
points cost. Protocol, calibration and the decision rule are in
[bench/README.md](bench/README.md).

Read the per-target matrix, not just the total. A seat is also not bought with
recall alone — **wall clock** matters, because a round runs at its slowest
reviewer's pace, and a **contract failure** disqualifies regardless of score.

<!-- BENCH TABLE: generated by `make bench-readme`, do not edit -->

25 candidates, scored on 8 targets (go, design, cpp, csharp, java, python, rust, typescript).

| # | candidate | route | score | recall | total $ | per point | wall s | fails |
|--:|---|---|--:|--:|--:|--:|--:|--:|
| 1 | `claude` | cli | 376/484 | 78% | ~$8.61 | ~$0.023 | 3958 | 0 |
| 2 | `claude-opus-5` | anthropic | 373/484 | 77% | ~$8.84 | ~$0.024 | 4152 | 0 |
| 3 | `claude-fable-5-1` | anthropic | 365/484 | 75% | ~$13.65 | ~$0.037 | 3504 | 0 |
| 4 | `qwen/qwen3.8-max` | openrouter | 331/484 | 68% | $6.35 | $0.019 | 11266 | 0 |
| 5 | `kimi-k3:cloud` | ollama | 319/484 | 66% | ~$7.58 | ~$0.024 | 2179 | 0 |
| 6 | `qwen/qwen3.8-27b` | openrouter | 314/484 | 65% | $2.48 | $0.008 | 12930 | 0 |
| 7 | `glm-5.3:cloud` | ollama | 313/484 | 65% | ~$7.31 | ~$0.023 | 2540 | 0 |
| 8 | `nemotron-3-ultra:cloud` | ollama | 310/484 | 64% | ~$1.66 | ~$0.005 | 20316 | 1 |
| 9 | `glm-5.3-flash:cloud` | ollama | 304/484 | 63% | ~$0.78 | ~$0.003 | 2209 | 0 |
| 10 | `codex` | cli | 298/484 | 62% | ~$8.36 | ~$0.028 | 6810 | 0 |
| 11 | `glm-5.2:cloud` | ollama | 278/484 | 57% | ~$5.72 | ~$0.021 | 3948 | 0 |
| 12 | `claude-fable-5` | anthropic | 277/484 | 57% | ~$13.39 | ~$0.048 | 3653 | 0 |
| 13 | `x-ai/grok-4.6` | openrouter | 277/484 | 57% | $6.00 | $0.022 | 7955 | 0 |
| 14 | `deepseek-v4-pro:cloud` | ollama | 270/484 | 56% | ~$2.83 | ~$0.010 | 2551 | 0 |
| 15 | `gpt-6-astra-xhigh` | cli | 270/484 | 56% | ~$14.94 | ~$0.055 | 5617 | 0 |
| 16 | `claude-opus-4-8` | anthropic | 268/484 | 55% | ~$6.04 | ~$0.023 | 2912 | 0 |
| 17 | `gpt-6-astra` | cli | 265/484 | 55% | ~$10.58 | ~$0.040 | 3600 | 0 |
| 18 | `minimax-m3:cloud` | ollama | 251/484 | 52% | ~$4.56 | ~$0.018 | 6003 | 2 |
| 19 | `deepseek-v4-flash:0731-cloud` | ollama | 249/484 | 51% | ~$1.19 | ~$0.005 | 6800 | 0 |
| 20 | `claude-sonnet-5` | anthropic | 245/484 | 51% | ~$4.89 | ~$0.020 | 4313 | 0 |
| 21 | `kimi-k2.7-code:cloud` | ollama | 242/484 | 50% | ~$2.54 | ~$0.010 | 2876 | 0 |
| 22 | `gpt-oss:120b-cloud` | ollama | 236/484 | 49% | ~$1.41 | ~$0.006 | 3533 | 0 |
| 23 | `gpt-5.6-terra` | cli | 221/484 | 46% | ~$3.04 | ~$0.014 | 2896 | 0 |
| 24 | `qwen3.5:397b-cloud` | ollama | 186/484 | 38% | ~$1.66 | ~$0.009 | 1187 | 0 |
| 25 | `gemma4:31b-cloud` | ollama | 156/484 | 32% | ~$0.66 | ~$0.004 | 1598 | 0 |

Recall per target, which is where a model that is strong in one language and weak in another shows up:

| candidate | go | design | cpp | csharp | java | python | rust | typescript |
|---|--:|--:|--:|--:|--:|--:|--:|--:|
| `claude` | 43/56 | 20/34 | 51/64 | 51/68 | 52/69 | 51/66 | 53/64 | 55/63 |
| `claude-opus-5` | 42/56 | 16/34 | 51/64 | 52/68 | 55/69 | 49/66 | 53/64 | 55/63 |
| `claude-fable-5-1` | 40/56 | 15/34 | 49/64 | 51/68 | 53/69 | 52/66 | 52/64 | 53/63 |
| `qwen/qwen3.8-max` | 37/56 | 19/34 | 49/64 | 43/68 | 44/69 | 48/66 | 44/64 | 47/63 |
| `kimi-k3:cloud` | 41/56 | 14/34 | 46/64 | 48/68 | 44/69 | 39/66 | 41/64 | 46/63 |
| `qwen/qwen3.8-27b` | 38/56 | 15/34 | 50/64 | 43/68 | 47/69 | 42/66 | 35/64 | 44/63 |
| `glm-5.3:cloud` | 39/56 | 14/34 | 38/64 | 44/68 | 49/69 | 48/66 | 39/64 | 42/63 |
| `nemotron-3-ultra:cloud` | 32/56 | 16/34 | 47/64 | 44/68 | 47/69 | 42/66 | 39/64 | 43/63 |
| `glm-5.3-flash:cloud` | 40/56 | 11/34 | 43/64 | 45/68 | 44/69 | 45/66 | 35/64 | 41/63 |
| `codex` | 39/56 | 13/34 | 42/64 | 44/68 | 44/69 | 39/66 | 36/64 | 41/63 |
| `glm-5.2:cloud` | 33/56 | 14/34 | 40/64 | 38/68 | 42/69 | 39/66 | 35/64 | 37/63 |
| `claude-fable-5` | 41/56 | 12/34 | 33/64 | 36/68 | 38/69 | 48/66 | 33/64 | 36/63 |
| `x-ai/grok-4.6` | 32/56 | 11/34 | 39/64 | 37/68 | 38/69 | 41/66 | 38/64 | 41/63 |
| `deepseek-v4-pro:cloud` | 29/56 | 15/34 | 38/64 | 35/68 | 42/69 | 38/66 | 33/64 | 40/63 |
| `gpt-6-astra-xhigh` | 32/56 | 14/34 | 45/64 | 31/68 | 34/69 | 42/66 | 35/64 | 37/63 |
| `claude-opus-4-8` | 31/56 | 12/34 | 37/64 | 37/68 | 44/69 | 40/66 | 32/64 | 35/63 |
| `gpt-6-astra` | 35/56 | 14/34 | 42/64 | 28/68 | 40/69 | 40/66 | 29/64 | 37/63 |
| `minimax-m3:cloud` | 37/56 | 13/34 | 38/64 | 31/68 | 37/69 | 36/66 | 28/64 | 31/63 |
| `deepseek-v4-flash:0731-cloud` | 28/56 | 9/34 | 32/64 | 35/68 | 33/69 | 41/66 | 28/64 | 43/63 |
| `claude-sonnet-5` | 28/56 | 8/34 | 33/64 | 35/68 | 44/69 | 37/66 | 29/64 | 31/63 |
| `kimi-k2.7-code:cloud` | 32/56 | 15/34 | 30/64 | 34/68 | 37/69 | 28/66 | 33/64 | 33/63 |
| `gpt-oss:120b-cloud` | 23/56 | 9/34 | 35/64 | 38/68 | 41/69 | 28/66 | 31/64 | 31/63 |
| `gpt-5.6-terra` | 28/56 | 9/34 | 35/64 | 27/68 | 29/69 | 32/66 | 26/64 | 35/63 |
| `qwen3.5:397b-cloud` | 17/56 | 12/34 | 28/64 | 26/68 | 31/69 | 25/66 | 26/64 | 21/63 |
| `gemma4:31b-cloud` | 17/56 | 10/34 | 24/64 | 20/68 | 30/69 | 19/66 | 19/64 | 17/63 |

`!` marks a cell that lost a lens to a contract failure: its recall is deflated, not low. `~$` is notional &mdash; the route bills a plan, not the tokens (claude, codex and ollama, which still meters session and weekly quotas); a bare `$` is money actually billed to a card, which is OpenRouter only.

<!-- END BENCH TABLE -->

## Configuration

Configuration is a **bundle**: task configs at the root of a directory plus
`prompts/`, `agents/` and `gates/` beside them, so one bare name resolves in one category.
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
| [review-pr](config/review-pr.yaml) | Review a GitHub pull request; review-only by default. See [Pull requests](docs/pull-requests.md). |
| [review-design](config/review-design.yaml) | Review a design document (`-target docs/DESIGN.md`) or a project's architecture (`-target <dir>`), no edits. Structure, data and failure modes — not code defects. |
| [create-design](config/create-design.yaml) | Draft a design from an assignment (`-target assignment.md`): the pool proposes independently, critiques anonymously, an editor synthesizes with dissent recorded. Writes `DESIGN.md` beside the assignment (or `-out`), never overwriting. |
| [implement-go](config/implement-go.yaml) / [implement-node](config/implement-node.yaml) / [implement-java](config/implement-java.yaml) / [implement-web](config/implement-web.yaml) | Build a reviewed design into a **new project** (`-target DESIGN.md -out <fresh-dir>`): one planner session decomposes it, the coder builds one task per gated commit, and every task's outcome lands in the history as durable trailers. `implement-web` is the asserted-ungated stack. Spec: [docs/design/DESIGN.md](docs/design/DESIGN.md). |
| [fix-code](config/fix-code.yaml) | Review → fix → verify → commit loop over a whole project. Needs `-trusted-target`; the gate is Go's unless `-gate <lang>` says otherwise. |
| [fix-branch](config/fix-branch.yaml) | The same loop over only what this branch changed — git-diff against the merge base with `@{upstream}`. Needs `-trusted-target`. |
| [fix-pr](config/fix-pr.yaml) | Fix a pull request's changes and triage its open conversations; replies are posted only with `-post`. Needs `-allow-untrusted-fix`. |
| [defaults](config/defaults.yaml) | Shared base the others extend; not runnable on its own. |

Bundles are searched most-specific first — `<project>/config/`, `~/.fixpoint/`,
the OS per-user config dir, the bundle beside the binary itself, then the
package-installed one — **per file**, so a project can shadow one prompt and
inherit the rest. The binary-adjacent entry resolves symlinks first, so a link
on your PATH pointing into a checkout finds that checkout's bundle; it is what
lets `fixpoint` run from a directory with no bundle of its own. There is deliberately no
fallback compiled into the binary: these files decide what agents are told to do
and which trust gates apply, so they must be readable on disk. If nothing
resolves, fixpoint refuses to run and prints every location it searched. Every
run logs the file each name resolved to, and records it in the run summary.

The main sections of a task config:

- **`target`** — what to review: mode, path, exclude globs, base ref or PR number.
- **`roles`** — the coder (agent + prompt) and the review lens list with its
  assignment strategy. The reviewer **pool** lives in `defaults.yaml` and is
  inherited, so changing who reviews is one edit rather than one per config; a
  config that sets `roles.review.agents` replaces that pool instead of adding to
  it.
- **`agents`** — the command templates, each optionally carrying `prompt_budget`
  (bytes; over it the invocation is refused before the process starts, and the
  step is recorded as failed rather than sent and rejected by the provider).
- **`loop`** — `max_iterations`, `max_final_passes` (how many times the closing
  round may repeat, default 1), `final_skip_run_edits` (globs the closing round is
  not shown when this run wrote the file), `commit_policy` (see
  [One fix, one commit](docs/concepts.md#one-fix-one-commit)),
  `max_findings_per_round` (caps how many **issues** a round hands over, `0` =
  unlimited and the default; worst severity goes first and the overflow is
  deferred to later rounds, but every deferral promotes an issue one severity
  tier, so nothing can be starved indefinitely), `review_only`, `commit_message`
  (placeholders `{issue}`, `{title}` for a single fix; `{round}`, `{fixed}`,
  `{rejected}` for a squashed one), and `clean_rounds_to_stop` (how many
  consecutive clean rounds end the run — `2` pairs well with `strategy: rotate`,
  so a differently-assigned panel must confirm the clean result). The trust gates
  are deliberately **not** here: they are command-line flags only, for the reason
  given under [Security model](docs/security.md).
- **`review`** — what a review run concludes and how it says so: `block_at` (the
  severity that forces CHANGES_REQUESTED, default `high`), `refute_at` (the floor
  for the refutation round, default `high` — it may be looser than `block_at` but
  never stricter), `refute` (the prompt, or empty to skip the round), and the two
  signature templates. Publishing is **not** here: `-post`, `-post-run` and
  `-post-verdict` are flags only.
- **`verify`** — the deterministic gate fixpoint runs itself between the coder and
  the commit: `commands` (argv, per project, cheapest first), a per-command
  `timeout`, and `policy` — `no_regressions` (the default), `must_pass`, or `off`.
  A command may also be marked `infra: true` to say its failure is a fact about
  the environment (a registry, a proxy) rather than about the code, so an outage
  does not become a permanent verdict. No commands means the gate is off, which is
  why the shipped defaults define none. Commands execute code from the target, so
  they run only on the fix path, which already requires the trust assertion.
- **`implement`** — the build pipeline's bounds: `max_tasks`, `max_task_attempts`,
  `max_run_duration`, `max_infra_tries`, `max_task_bytes`, `min_free_disk`,
  `clean_check`, and the stack's `gitignore_seed` / `gate_generated` lists.
- **`logs`** — where and in which formats run artifacts are written.

## Flags

The publishing flags (`-post`, `-post-run`, `-post-verdict`) are the ones worth
reading twice, and [Pull requests](docs/pull-requests.md) is where they are
explained: each one acts under your identity on somebody else's branch.

| Flag | Effect |
|---|---|
| `<name>` | Positional: the task config to run, resolved on the bundle search path. A value containing a separator or ending in `.yaml` is used as a path. |
| `--list` | List task configs with the file each resolved from, and exit. |
| `-config path` | Alternative to the positional name. Giving both is an error. |
| `-review-only` | Run exactly one review round; the coder is never invoked (no edits in git-diff/directory mode; pr mode still runs `gh pr checkout`, switching the branch and working tree in Prepare). |
| `-max-iterations n` | Override `loop.max_iterations`. |
| `-base-ref ref` | Override `target.base_ref` in git-diff mode; a trailing `...` means the merge base with HEAD. For `fix-branch` on a branch with no upstream: `-base-ref 'origin/main...'`. |
| `-gate name` | Override `verify.gate`: which language's toolchain the deterministic gate runs, by name under `<bundle>/gates/` (`go`, `rust`, `node`, `java`, `python`, `dotnet`, `cpp`). The shipped `fix-*` configs name `go`; on another project: `fixpoint fix-branch -gate node -trusted-target`. Replaces the config's commands. |
| `-target path` | Point a directory-mode run at a file or a directory. A file is reviewed as a **document**, shown to the panel in full; a directory is collected as a listing. How `review-design` is aimed. |
| `-out path` | Where a create run writes its deliverable (default: `DESIGN.md` beside the assignment). An existing file is never overwritten. |
| `-pr n` | Override `target.pr` in pr mode. `review-pr` ships with no number, so this is how you say which PR: `fixpoint review-pr -pr 1234`. Omit it on your own pull request's branch and fixpoint resolves the number from the checkout, refusing rather than guessing when the branch has no pull request, several open ones, only a merged one, or one from a fork — see [Pull requests](docs/pull-requests.md#which-pull-request). |
| `-trusted-target` | Assert a directory/git-diff target holds only trusted code, permitting fix rounds (fail-closed without it). |
| `-trusted-bundle` | Assert **only** that the bundle files resolved from inside the target may be executed and sent to agents. Permits no fix round and trusts no other target content — this is the flag to use when the config is yours but the code is not, as `review-pr` on a fork's branch is. |
| `-allow-untrusted-fix` | Permit fix rounds in `pr` mode (PR content is untrusted; see [Security model](docs/security.md)). |
| `-post` | Publish the review on the pull request as a **comment**: findings become visible, no verdict is acted on. Publishes what the run just produced, so nobody has read it yet — prefer [`-post-run`](docs/pull-requests.md). A publish that was asked for and did not happen fails the run (exit `1`), so an approval can never exit `0` over a review that never reached the pull request; the review is still in `review-body.md`. |
| `-post-run dir` | Publish the review a **finished** run already produced, from its `.fixpoint/<run>` directory (or its `summary-*.json`). Invokes no agent and reviews nothing: the bytes posted are the bytes in `review-body.md` and the inline anchors are the ones that run computed. Resolves no configuration at all — everything it acts on is in that run's summary — so every flag but `-post-verdict` is ignored. |
| `-post-verdict` | With `-post` or `-post-run`, let the review carry its verdict — approving, or requesting changes on someone's PR. An inconclusive verdict stays a comment regardless. An approval is bound to the reviewed commit only until the run ends: unless the repository dismisses stale approvals on push, it keeps counting after one ([why](docs/pull-requests.md)). |
| `-plan-only` | **implement**: plan and validate, write `plan.json` and the rendered plan into the run's artifacts, and stop. Claims no `-out` and spends no coder session; terminates `planned` and exits 0. |
| `-plan file` | **implement**: run this validated plan instead of a planner session. It is held to every rule in §4.2 and must carry `provenance.design_sha256` matching `-target`; a missing digest is a refusal, not a waiver. The plan is stamped `operator-supplied` with the file's own digest. |
| `-no-coverage-check` | **implement**: proceed with a design whose heading outline the coverage rule cannot read. Every report then says `coverage: unchecked`. |
| `-continue dir` | **implement**: resume a project fixpoint built, replaying finished tasks from its own commits. Mutually exclusive with `-out`, `-plan` and `-plan-only`. Still needs `-trusted-target`. |
| `-version` | Print the version, commit, build date, platform and Go version, and exit. Answers before any bundle is resolved. |
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

## Platform support

Linux and macOS. fixpoint does not currently compile for `GOOS=windows`: it puts
each agent in its own POSIX process group so that a timeout, a canceled run, or
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
internal/create/         the create-design pipeline: propose, critique, synthesize,
                         object, revise, publish atomically
internal/implement/      the implement-design mechanics: plan validation, the
                         design outline, censuses, repository invariants, scaffold
internal/verify/         runs the deterministic gate (build / test / static checks)
internal/review/         the verdict rules and the review document (body, inline
                         comments, signature)
internal/forge/          reads a pull request's checks and conversations, and
                         publishes the review back through the vendor's CLI (`gh`)
internal/logstore/       per-step logs, the run journal, and the run summary
internal/runlog/         renders the run's progress: phase blocks, color on a tty
internal/gitenv/         the git hardening every subprocess against a target carries
internal/testfixture/    shared test helpers
config/                  the shipped bundle: task configs, prompts/, agents/, gates/
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

```sh
make release-check      # validate .goreleaser.yaml
make release-snapshot   # build archives + Linux packages into dist/, no tag, no publish
make release-verify     # …then unpack one and prove it runs
```

Cutting a release is a tag: `git tag -a v0.2.0 -m ... && git push origin v0.2.0`
runs the audit gate on Linux and macOS, builds every artifact, and publishes the
GitHub Release. The workflow then unpacks an archive and installs the deb and
asserts both resolve their bundle — an archive whose bundle is mis-shaped
installs a tool that refuses to run, and both packagers flattened it on the
first attempt. `make release-verify` is the same check locally, and is what a
packaging change should be tested with before a tag exists, since a tag that
produces a broken archive cannot be taken back.

Analysis tools run via `go run` with pinned versions, so no global installs
are needed and every machine uses identical tool versions. The lint policy
lives in [.golangci.yml](.golangci.yml): an exhaustive curated linter set
where every disabled check and exclusion states its reason.

The only external dependency is `gopkg.in/yaml.v3`.

Dogfooding note: this repository is reviewed by fixpoint itself — see the
`round N (M fixed, K rejected)` commits in the history.

## License

MIT — see [LICENSE](LICENSE). The same terms cover the shipped config bundle:
the prompts and task configs are as much the tool as the Go code is, and a
license that stopped at the binary would leave the part you are most likely to
copy and edit unlicensed.
