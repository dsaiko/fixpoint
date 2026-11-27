# Reviewer benchmark: is this model worth a panel seat?

A repeatable measurement of one model's value as a fixpoint reviewer:
**quality per token**, on frozen targets with known, planted defects. It
exists because panel seats are paid for in quota (the ollama weekly limit is
the real price — there is no invoice), and because the panel's history shows
the question is decidable by measurement: gemma4 lost its seat to numbers, and
kimi holds one on numbers.

## What "effective" means

A reviewer is effective when it **finds real defects the panel would
otherwise miss, at a token and wall-clock price the pool can afford, without
breaking the output contract**. Operationalized as four numbers per run, one
disqualifier, and one decision rule.

### The four numbers (per run, written to `results.csv`)

| metric | definition | why it is the right proxy |
|---|---|---|
| **recall** | seeded defects found / seeded (60 code, 40 design) | quality against ground truth — no judge, no taste, no drift between runs |
| **noise** | findings matching no seed | not automatically false (models find real unseeded bugs; skim the per-run report before dismissing) — but a model whose output is mostly unmatched is expensive to triage |
| **cost** | real money, from each route's published per-token rates, plus wall-clock seconds | since 2026-09-01 every route including ollama publishes per-token prices, so cost is comparable across routes in one unit; wall clock is what gates a panel round, which runs at the slowest reviewer's pace |
| **found_per_mtok** | matched seeds per million tokens | the headline: quality per unit of the thing being spent |

The two calibrated targets are one benchmark: **56 go seeds + 34 design seeds
= 90 points**, one point per seed. The score is out of whatever the live pool
is — `report.py` reads each row's own seed total rather than a constant, so
retiring a seed cannot leave the table quoting a scale that no longer exists. `bench/report.py` sums a model's two rows into
that score; a model that has run only one task is reported as INCOMPLETE
rather than scored out of 60.

Nobody scores 100, and that is the design. The previous 20-seed target was
saturated — 10 of 24 code runs and 11 of 21 design runs sat at recall 1.00, so
the metric had stopped ranking the top of the field. A hundred seeds across
~940 lines of Go and a 169-line design document spread the good models out
again; what matters is the distance to the claude baseline, not the distance
to 100.

### The disqualifier

**Contract compliance.** Every output-contract failure in this project's
history came from a non-Anthropic model driving this harness. A session that
produces unparseable output burns a full slot and, in a quorum, can flip the
verdict to INCONCLUSIVE. Any contract failure or timeout across the sweep is
recorded in `results.csv` (`errors`); more than one across three repeats
disqualifies regardless of recall.

### Isolation: the reviewer must not see the answer key

A lens can read anything its user can; its read-only sandbox confines writes,
not reads. Until 2026-11-23 a lens reviewed `bench/testdata/target-<x>` in
place, two directories below its own `manifest-<x>.yaml`, and three rows were
measured by models that read it (hy4 scored go 53/56 that way). Two layers
guard against that now.

**Staging.** `run.sh` copies every run's target to a fresh directory outside the
repository, from git's own file list (ignored build output stays behind), so
nothing sits above the target and its path does not name the repository.

**What this protects, and what it does not.** The goal is BENCH INTEGRITY: a lens
must not read the answer key, past runs, or another lens's findings, and must not
write anywhere but the places it is given. It is not a general defence of the
operator against a hostile model. The lens still holds the provider key its CLI
needs (the CLI's own process environment carries it, and so does every process
between the wrapper and the CLI), it can use the network, and codex's own sandbox is
OFF inside the wrapper (it cannot nest), so what keeps a codex lens from writing is
the wrapper alone.

**A filesystem boundary.** `config/bench-code.yaml` and `config/bench-design.yaml`
set `sandbox.command`, which launches every lens through `bench/sandbox.sh
<target> <ro|rw> -- <agent argv>`: `sandbox-exec` on macOS, `bwrap` on Linux; any
other system, or a missing tool, refuses to run the agent at all. The policy is
what a lens is GIVEN, not everything minus two directories.

| | macOS | Linux |
|---|---|---|
| the checkout, `~/.fixpoint` (resolved if a symlink) | unreadable, unwritable | replaced by an empty tmpfs |
| the target (staged copy) | readable; writable only in `rw` | bound read-only at its own path (`ro`) |
| where it can write | its own scratch directory, nothing else | its own scratch directory, which is `/tmp`; the home directory is a tmpfs that vanishes |
| the staged target's parent | not writable, so it cannot be renamed away | not the host's: `/tmp` is private, so renaming the parent renames nothing |
| `$HOME` | **readable** (`~/.ssh`, `~/.aws`, other repositories) | replaced by an empty tmpfs, with the toolchain directories (`~/.nvm`, `~/.local`, ...) and the directory the CLI is installed in bound back read-only |
| processes | shared | own pid and IPC namespaces (`/proc/<pid>/cwd` of fixpoint is not there) and its own session (no controlling terminal to inject keystrokes into) |
| the operator's terminal (TIOCSTI) | the profile denies the ioctl on `/dev/tty*`, `/dev/pty*` and `/dev/console`, so `/dev/tty` and an inherited tty fd both fail (no `setsid`: fixpoint kills the process group on a timeout and macOS has no die-with-parent, so a session of its own would leak agents) | `--new-session`: no controlling terminal |
| other lenses' scratch directories | unreadable | not there: a `TMPDIR` other than `/tmp` is masked, and `/tmp` is the lens's own |
| `/run`, user D-Bus and system sockets | reachable | masked (only systemd-resolved's directory is bound back, for DNS) |
| network | open | open |

The CLI is taken from `argv[0]` of the agent command; an unknown binary is given no
state at all, and CLI state is **per invocation**:

- **codex** gets a fresh `CODEX_HOME` in the scratch directory. On a route that
  authenticates with the operator's ChatGPT login the fresh home holds a *symlink*
  to the shared copy of the login in `~/.codex/.fixpoint-sandbox/`, the one
  fixpoint's own sandbox uses; that directory is readable and writable, so a token
  refresh saves (codex writes a temp file there and renames it over the login).
  The wrapper syncs the copy with the real `~/.codex/auth.json` under codex's own
  lock (`auth.json.lock`) when the lens starts and when it ends, the newer
  `last_refresh` winning and only for the same account, so parallel lenses and a
  codex outside the bench do not lose each other's rotated refresh token. A wrapper
  killed outright skips the second sync and the next start makes it. Nothing of the
  login is written to scratch. Whether the route is a ChatGPT one is decided by `run.sh`,
  which knows the candidate (a codex agent that names a `model_provider` of its own
  carries its own key), and written to `bench/.sandbox-codex-auth` for the wrapper:
  `ollama-direct@`, `codex@` via OpenRouter and every other route with its own key
  get no `auth.json` at all, and without that file no login is handed over. The live
  `~/.codex` (sessions, sqlite, `config.toml`) is unreadable and unwritable.
- **claude** keeps the **real** config directory, because a fresh
  `CLAUDE_CONFIG_DIR` is "Not logged in" on macOS (the login is a Keychain item
  named from the directory's path; measured 2026-11-24). Inside it the transcripts
  and history (`projects/`, `sessions/`, `file-history/`, `session-env/`,
  `shell-snapshots/`, `todos/`, `backups/`, `debug/`, `plans/`, `paste-cache/`,
  `tasks/`, `teams/`, `usage-data/`, `image-cache/`, `history.jsonl`) cannot be read, and
  NOTHING under it can be written on macOS except `.claude.json`: not
  `settings.json`, `CLAUDE.md`, `commands/`, `skills/`, `plugins/`, hooks, and not
  `projects/` or `shell-snapshots/` either (a memory file planted there is read by the
  operator's next session). A claude lens runs without any of that (checked with a
  real `claude -p` session): it just keeps no transcript. On Linux the state
  directories are empty tmpfs mounts, writable and thrown away, and `history.jsonl`
  is replaced by an empty file. The CLI's shell-tool directory is pointed into the
  lens's own scratch (`CLAUDE_CODE_TMPDIR`), so two lenses on the same target do not
  share `/private/tmp/claude-<uid>/<cwd slug>`. The live codex directory is
  unreadable.

What a lens can still reach, stated plainly:

- On macOS, all of `$HOME` that is not named above, including other copies of this
  repository (they hold the same manifests), and the network. The scorer's
  log check (below) is what still catches a read of the key there.
- The claude CLI's `.claude.json` is **writable** (it rewrites it): it is global
  state that includes user-scope MCP servers, so a prompt-injected lens could plant
  one that runs the next time the operator starts `claude`. It also stays readable.
  The same goes for `.credentials.json` on Linux.
- On Linux the network namespace is shared with the host: abstract unix sockets and
  services on localhost are reachable.
- The lens's own scratch directory and, on Linux, the part of `$HOME` it is given
  back; other users' home directories under their own permissions.

A terminated or timed-out lens: fixpoint stops an agent by SIGKILLing its process
group, which nothing can trap, so the wrapper's scratch directory (no credential in
it) is left in `TMPDIR` and a stray `bench-sandbox.*` there is that. A TERM sent to
the wrapper itself removes it; on macOS the agent also gets the TERM, on Linux
bwrap kills it outright.

`bench/run.sh` runs the wrapper once on a throwaway directory before it sweeps, and
stops with a non-zero status and the wrapper's own message if it does not work (no
`bwrap` or `sandbox-exec`, or a `bwrap` that cannot create a namespace): otherwise every
preflight ping would fail, no row would be written and the sweep would look done. A
sweep in which no requested target produced a score exits non-zero as well.

`bench/run.sh` puts `bench/` on `PATH` (fixpoint finds the wrapper by name: a
relative path would resolve against the staged target) and the wrapper takes that
entry back off before it starts the agent.

**Not supported inside the wrapper: `ollama launch`.** It needs `~/.ollama` (it
saves the selected model there before the harness starts) and neither bench host
has ollama installed. `run.sh` refuses the plain `<tag>:cloud` route,
`ollama-codex@<tag>` and any shipped agent whose command starts with `ollama`, and
points at `ollama-direct@<tag>`, the same harness straight to ollama.com.
`ollama-direct@`, `codex@`, `vllm@`, `claude-*`, OpenRouter ids and the shipped
codex and claude agents work.

**Known limitations**, found in review and left open on purpose (they are outside
the goal above, or cost more than they protect):

- CLI state relocated outside `$HOME` (a symlinked `~/.codex`, an external
  `CLAUDE_CONFIG_DIR`) stays readable on Linux: only paths under the home are replaced.
- The `/run` mask assumes systemd-resolved for DNS; a host whose `/etc/resolv.conf`
  points elsewhere under `/run` has no name resolution inside.
- A custom `CLAUDE_CONFIG_DIR` blocks claude's rewrite of its own config on Linux
  (the directory is bound read-only; only `.claude.json` is bound back).
- A target path ending in a newline is not supported (command substitution strips it).
- A signal that arrives before the agent is spawned is swallowed, and the run goes on.
- Provider keys are readable by the lens: the CLI process, and every process between
  the wrapper and it, carries them in its environment, and `/proc/<pid>/environ` of
  an ancestor is readable by its children. Moving them out of the wrapper's own
  environment would not help while the CLI itself must hold the key; codex's
  `shell_environment_policy.exclude` keeps it out of the shell tool's environment, not
  out of `/proc`.
- On macOS the rest of `$HOME`, including other checkouts of this repository, is
  readable; the scorer's log check is what catches a read of the key there.
- Section (a) of `bench/sandbox_test.sh` (the namespace runs) fails where `bwrap`
  cannot create a user namespace, instead of skipping.
- On macOS a lens can read the environment of same-user processes (the fixpoint
  parent's, through process-argument queries), so a key fixpoint left out of the
  agent's environment is still recoverable.
- On Linux a claude candidate on a third-party route (OpenRouter, vLLM) still sees the
  operator's Anthropic `.credentials.json`.
- The CLI state that stays writable and shared (`.claude.json`, the shared
  `auth.json` copy on a ChatGPT route) is a channel through which colluding lenses could pass findings to
  each other; the scorer does not see it.
- The group SIGKILL fixpoint uses on a timeout is not covered by a test (TERM is).

**Sandboxes do not nest.** The `--sandbox read-only` of a codex-harness agent is
itself seatbelt (macOS) or bubblewrap (Linux) around each shell command, and
inside the wrapper it cannot start: every command a lens ran failed with
`sandbox_apply: Operation not permitted`. So `run.sh` rewrites it to `--sandbox
danger-full-access` in the GENERATED candidate file only, and the wrapper is what
enforces read-only. The shipped `config/agents/*.yaml` are untouched, and a
claude-harness candidate has no such line. fixpoint's agent validation refuses
`can_edit: false` together with `--sandbox danger-full-access` unless the
invocation asserts `-agents-sandboxed`, a command-line flag that needs a
`sandbox.command` and is refused in `pr` mode (`docs/security.md`); `run.sh`
passes it. Run `./fixpoint bench-code` by hand without it and a codex candidate
stops at validation, before any session.

**What was verified, and where.** `bench/sandbox_test.sh` runs real probes through
the sandbox of the host it is on (skipped, loudly, where `bwrap` cannot make a
namespace), each against an unconfined control; the Go test
`TestShippedBenchConfigsReachTheWrapperWithTheStagedTargetReadOnly` loads both
shipped configs through the real entry point and checks every lens reaches the
wrapper with the staged target, `ro` and the candidate's argv. Probes on macOS:
a glob and `find` cannot reach the manifest; a symlinked
`~/.fixpoint` is hidden through; the target and its parent cannot be written,
renamed or replaced; planting a `settings.json` hook, a `CLAUDE.md` or a `commands/`
entry fails; earlier sessions and transcripts cannot be read; real codex
and claude sessions ran through it: `ollama-direct@glm-5.3-flash:cloud` on go scored
39/56 (earlier unconfined go rows: 33 to 40), its three lenses ran 11 shell commands,
all exit 0, none denied; `claude-haiku-5-5` on design authenticated against the real
config directory and scored 21/34 with no permission denial (its raw log carries no
tool calls, so how many it made is not visible).
The Linux branch has been run by the operator on the bench host (earlier version: the
mechanics, 35/56 and no sandbox failure); this version's Linux probe has **not
been run by its author**, only its argument assembly (bwrap stubbed).

**The log check remains as evidence**, not as the boundary: `score.py` writes no
row when any lens's tool call mentions the checkout, `bench/`, `.fixpoint`, a
manifest or the scorer (`key_reads()`, tested by `--leak-selftest` and end to end
in `run_test.sh`). It cannot see a path formed by shell expansion, and the
claude-harness routes' raw stream holds one result envelope and no tool calls, so
`score.py` prints a warning and scores those sessions on trust; the wrapper is
what covers them. A record-breaking cell is still worth a look at its sessions.

### The calibration, measured 2026-08-20

| baseline | go | design | **score** | tokens | wall |
|---|---|---|---|---|---|
| claude (opus-5, effort high) | 43/56 | 20/34 | **63/90** | 123k | 1490s |
| codex | 39/56 | 13/34 | **52/90** | 315k | 941s |

Re-measured on the **2026-09-01 scale** (56 go + 34 design = 90 points) by
re-scoring the original summaries — no re-run. On the retired 100-point scale
the same two runs read 75/100 and 61/100; the drop is the scale getting harder,
not the models getting worse. claude also scores **53/64 (83%)** on the Rust
target.

Zero contract failures either side. That is the readable scale: 75 is what the
best reviewer available does, so a candidate at 50 is doing two thirds of the
job and one at 15 is not reviewing.

Two things the calibration showed that a reader of the numbers needs:

- **The misses move between runs.** claude found the unsalted SHA-256 in one
  run and not in the next, on an unchanged target. Single-repeat recall is a
  sample, not a measurement — which is why a seat needs 3 repeats.
- **Design tops out near 25, not 40.** claude produced 37 findings across the
  two design lenses with *zero* unmatched, and still reached 23 seeds: the two
  lenses report the same flaw, and one finding often covers two or three seeds.
  The 40 design seeds are a pool a reviewer samples from, so *which* ones a
  model finds carries as much information as how many.

### The decision rule

Run **claude and codex through the bench first** — they calibrate it. A seed
neither baseline finds is a bad (or genuinely hard) seed; baseline recall is
the 100% mark that makes candidate recall readable.

A candidate earns a seat when, over ≥3 repeats:

1. zero-or-one contract failures (see above), and
2. median recall ≥ **2/3 of the claude baseline** on the same task (so ≥35/60
   on code, ≥15/40 on design, against the 2026-08-20 calibration), and
3. `found_per_mtok` beats the weakest current seat-holder on the same task —
   a seat is comparative: the question is never "is it good", it is "does it
   beat what the seat currently pays for".

Diversity is a tie-breaker, not a metric: *which* seeds a model finds is in
the per-run report, and a model that reliably finds the seeds the rest of the
panel misses is worth more than its recall says (measured corroboration
across the panel is <4%, so non-overlap is the panel's whole value).

## Protocol

- **Frozen targets.** The same link-shortener service in **seven languages**,
  plus a design document:

  | target | seeds | built by `make bench-check` |
  |---|---|---|
  | `target-go` | 56 scored (11 retired) | `go build` |
  | `target-rust` | 64 | `cargo build --offline` |
  | `target-java` | 68 | `javac` |
  | `target-typescript` | 62 | `tsc -p .` (strict) |
  | `target-csharp` | 67 | `dotnet build` |
  | `target-cpp` | 63 | `clang++ -std=c++20` |
  | `target-python` | 65 | `import` (not just `py_compile`) |
  | `target-design` | 34 scored (6 retired) | — |

  The Go target's 60 original seeds split across `review-bugs`,
  `review-concurrency` and `review-security` (14 security defects — unsalted
  hashes, a predictable token, an open redirect, a path traversal, a
  client-controlled admin header); the design target's 40 split across
  `design-failure` (24) and `design-data` (16). **Never fix the seeded
  bugs** — the target's value is that it does not change between
  measurements. `**/testdata/**` is excluded by the default config, so real
  review runs over this repository never trip on them.
- **One variable.** Every candidate runs the identical harness wrapping
  (`bench/agent-template.yaml` = the kimi-ollama.yaml invocation with only
  the model swapped), identical prompts, identical targets. No judge, no
  refutation — they would measure claude's filtering, not the candidate.
- **5 sessions per candidate per repeat** (3 code lenses + 2 design lenses),
  so a seven-model sweep is ~35 sessions per repeat. That is the price of a
  seed count that discriminates, and it is why the cheap screen matters:
  sweep once at 1 repeat, drop everything that cannot finish a run or scores
  near the floor, then re-run only the finalists at 3 repeats. Single-run
  recall is noisy and the variance itself is information (a model that scores
  61 then 38 is worse than one that scores 49 twice).
- **Pin what you measured.** Floating `:cloud` tags get re-pointed;
  `bench/run.sh` records `ollama show` output at run time, and a dated tag
  (like `deepseek-v4-flash:0731-cloud`) beats a floating one where the
  library offers it. A seat decision quotes the resolved model, not the tag.
- **The prompts are instrument too.** The bench runs the PRODUCTION prompts:
  `config/bench-code.yaml` names `review-bugs`, `review-concurrency` and
  `review-security`, `config/bench-design.yaml` names `design-failure` and
  `design-data`, and `internal/prompt/prompt.go` injects the output contract,
  the mode guidance, the prelude and the untrusted-material envelope into every
  one of them. There are deliberately no bench-private copies — a seat decision
  has to predict the reviewer you actually run, not a frozen ancestor of it.
  The price is that editing any of them changes the scale, and this is the one
  drift the bench cannot repair afterwards: a keyword widening re-scores from
  the kept summaries, but a prompt edit changed what the model was *asked*, so
  those summaries answer a question that no longer exists and the only repair
  is a re-sweep of every model that still needs a comparable row (5 sessions
  per candidate per repeat, 21 models). `score.py` therefore stamps every row
  with an `instrument` hash of those files plus the two lens lists, so the
  boundary is visible instead of silent. Every row in `results.csv` carries
  `993af70a5111`, verified constant from `6e2327a` (the commit these rows start
  at) through HEAD; the archived 2026-08-20 rows predate the column and share
  that value, their break being the seed set, not the prompts. When the hash
  moves, check `git diff` first — it is a whole-file hash, so a comment-only
  edit to `prompt.go` moves it too — and if the change is real, treat it the
  way the seed-set change of 2026-08-20 was treated: archive, re-sweep, and
  never compare across the boundary.
- **Matcher, not judge.** `score.py` counts a seed as found when a finding
  names the right file, lands within ±span lines (code seeds), and mentions
  enough of the seed's keywords (`need`). Deterministic and cheap, at the
  price of occasional false negatives — when a per-run report shows a miss,
  check whether the model found it in words the keywords miss, and widen the
  manifest keywords (widening is safe: it can only make *past* runs
  comparable, never invalid, since all runs are re-scorable from their
  summaries).

  Three properties matter at a hundred seeds and did not at twenty:

  - **One finding, one seed.** A finding is credited to its best match only —
    most keywords, then a line gate beats a file-wide seed, then the closer
    line. With seeds packed a few lines apart, the old any-seed-that-matches
    rule let one vague finding light up three neighbours.
  - **Anchors, not line numbers.** A code seed carries `anchor`, an exact and
    unique substring of its line; `score.py` resolves it against the target at
    scoring time. Hand-counted line numbers rot the moment the target is
    touched, and an anchor that no longer resolves is a hard error instead of
    a seed nobody finds.
  - **A bundled finding earns one point.** Reviewers write one finding where
    the manifest has two seeds — claude reported the logged token and the
    token echoed in the 401 as a single sentence, and covered three design
    seeds with "unpaginated, unindexed, untimed /sync". That is the ceiling
    the design task runs into: 37 findings, every one matched, 23 seeds. The
    rule stays, because itemizing distinct defects is exactly what a panel
    seat is being bought for — but read a miss list with it in mind before
    calling a seed bad.
  - **The manifests are testable.** `make bench-check` runs
    `score.py --check` (every anchor resolves, uniquely) and
    `score.py --selftest` (every seed, handed the most on-the-nose finding it
    could receive, gets that finding back rather than losing it to a
    neighbour). Both must pass before a manifest edit is worth a run.

## Running it

```sh
make bench MODEL=ollama-direct@glm-5.2:cloud                 # the calibrated pair, 1 repeat
make bench MODEL=ollama-direct@kimi-k3:cloud TASK=go N=3     # one target, 3 repeats
make bench MODEL=ollama-direct@kimi-k3:cloud TASK=go,design  # a list of targets
make bench MODEL=claude-haiku-5-5 TASK=design                # a claude-harness candidate
make bench-check                               # validate the manifests, no model
make bench-seeds                               # per-seed hit rate: what still discriminates
python3 bench/report.py                        # scores, plus recall per target
make bench-readme                              # regenerate README.md's table, bench/report.html and bench/results.json
make bench-publish                             # bench-readme, then publish the table to www.saiko.cz/ai-benchmarks
```

**After measuring a new model, run `make bench-publish`.** Its first half,
`make bench-readme`, rewrites three generated surfaces from `results.csv` in one
pass: the results table in the top-level [README.md](../README.md), between two
HTML comment markers; the standings page [report.html](report.html), kept under
source control because it is the file people ask for; and
[results.json](results.json), the same standings as data. Its second half hands
that JSON to the public page at
[www.saiko.cz/ai-benchmarks](https://www.saiko.cz/ai-benchmarks/), a sibling
checkout (`BENCH_SITE_DIR`, default `../www.saiko.cz.ai-benchmarks`) whose
`make publish` builds the page from it and deploys. None of the four updates
itself. A hand-maintained copy would go stale on the first new model, and a
stale leaderboard is worse than no leaderboard: it is the number people quote
without opening the CSV. Commit the three files with the `results.csv` rows they
were generated from.

`TASK` names a **target**, not a task: `go`, `design`, or any language target
added under `bench/testdata/target-<name>` with a matching
`bench/manifest-<name>.yaml` -- `run.sh` resolves those with no edit of its own.
`code` remains an alias for `go`. `all` stays the **calibrated pair** (go +
design) rather than everything on disk, because that pair is what every
published score out of 100 means and what every seat decision on record quotes;
widening it silently would change both the cost and the total of a "comparable"
run.

### One service, many languages

Every language target is **the same link-shortener service** as `target-go`, on
purpose. Holding the domain constant makes the language the only variable, which
is what turns "the field scores 87% on Go and 40% on Rust" into a readable
result rather than a confound about one target simply being a harder program.

Seeds come in two kinds, and the split is the whole point:

- **Parallel** seeds are the same logical defect as a numbered Go seed (marked
  `parallel: Cnn` in the manifest): the inverted expiry check, the off-by-one
  page slice, the missing ownership check on delete, the client-controlled admin
  header. A model that finds these in Go and misses them in Rust is failing at
  the **language**, not at the defect class — and that is a fact no
  single-language bench can produce.
- **Language-only** seeds cannot exist in Go at all. `target-rust` carries
  `unwrap()` on an absent `Option`, an atomic incremented with load-then-store
  instead of `fetch_add`, mutex poisoning propagated by unwrapping every
  `lock()`, a lock-order inversion between two locks, `usize` division that is
  always zero, and `JoinHandle`s dropped without joining.

**No target has dependencies.** Every one builds offline, and a reviewer can
read the whole thing without a package-registry detour. `make bench-check` builds
each with its output pointed OUTSIDE the tree, because the bench reviews these
directories with no excludes and a build directory inside one would be handed to
the reviewer as source.

Three findings the languages produced that a single-language bench could not:

- The always-true admin check (`role != "admin" || role != "owner"`) is a
  **compile error in TypeScript** — control-flow narrowing proves it. It is
  replaced there by the `indexOf` truthiness bug, which rejects admins and
  admits everyone else.
- `quota[owner] = quota[owner] + 1` is a crash in Go, Java, C#, TypeScript and
  Python, and **not a defect at all in C++**, where `std::map::operator[]`
  value-initializes. The same operator IS the defect in that target's
  `count_for_owner`, where a read silently inserts.
- `py_compile` accepts a dataclass with a mutable default that `import`
  rejects, so the Python check imports every module.

### Targets and tasks are different axes

A **target** is a scored artifact with its own seed pool; a **task** is the lens
family run over it. Every language target shares `task: code` and the three code
lenses, so results are keyed on **target** -- keyed on task alone, a second
language's row silently overwrote the first's in the report, which is why the
manifests now declare `target:` explicitly.

### Reading the seed-value report

`make bench-seeds` is the retire/replace instrument, and it costs nothing: it
re-reads the per-run tables in `results/`. A seed found by nearly every model
carries almost no information about the model under test -- it spends tokens and
never moves a score -- and a seed found by nobody is either miscalibrated or
beyond the whole field. **Check a never-found seed against the raw findings
before retiring it.** D33 (no authorization model on `/sync`) read 0/21 and was
not a hard seed at all: one model reported it in as many words, but the seed
needs 2 keywords and that finding hit only `ownership`, while the same seed's
generic entries (`user_id`, `permission`, `tenant`) were firing on 27 unrelated
findings about indexes and file modes. That is a matcher bug wearing a hard
seed's clothes, and retiring it would have hidden the bug and kept the noise.

Candidates for the current sweep are listed in `models.txt`. Model names route
the harness: an existing agent name (`claude`, `codex`) runs through its own
yaml; a `claude-*` id runs claude-opus.yaml with the model swapped; an id with a
slash (`x-ai/grok-4.6`) runs the OpenRouter template (card-billed, cached;
needs `OPENROUTER_API_KEY`); a `codex@`-prefixed OpenRouter id runs the same
route through the CODEX harness instead -- for models whose replies the claude
harness cannot read (OpenRouter's Anthropic translation puts a trailing
signature-only thinking block after some models' text, and the harness's result
comes out empty; gemini-3.7-flash is the measured case). A codex@ number is
comparable to the codex baseline, which shares its harness -- say so when
quoting it. A `vllm@`-prefixed Hugging Face repo (`vllm@bottlecapai/ThinkingCap-Qwen3.8-27B`)
is a model we serve ourselves, for open weights no provider hosts: the claude
harness pointed at our own vLLM's `/v1/messages`, so it shares a harness with
the OpenRouter rows (see "Self-served models" below). An
`ollama-codex@`-prefixed ollama tag (`ollama-codex@beam:cloud`) runs an ollama
model through the CODEX harness (`ollama launch codex`): same model, same plan,
different harness. Effort reaches the model on that route, unlike the claude
one, so the template pins `xhigh`, the deepest level codex can express there;
an ollama-codex@ row differs from the plain ollama row in harness and possibly
thinking depth. An `ollama-direct@` tag runs the same harness against ollama.com
itself, with `OLLAMA_API_KEY` and no local ollama server: the transport is the
variable. Both take a `#<effort>` suffix (`ollama-direct@glm-5.3-flash:cloud#max`;
glm's catalog has no `xhigh`), and the pair is only comparable run the SAME DAY,
interleaved -- on 2026-10-28 beam's launch route scored 109 on three targets that
had given it 144 nine days before. That day's full glm-5.3-flash grid put the two
routes level: panel 408 direct, 406 launched. A bare tag (`glm-5.2:cloud`) runs the
`ollama launch claude` template and `ollama-codex@` the `ollama launch codex` one,
but BOTH launch routes are refused by `run.sh` now that every lens runs inside
`bench/sandbox.sh` (see "Isolation"); the measurements above were taken before that,
and `ollama-direct@` is how the same models are run today. Results append
to `results.csv` (committed — measurements are project knowledge); per-run
seed tables land in `results/`. Re-score an old run without re-paying for it:

```sh
python3 bench/score.py bench/manifest-go.yaml .fixpoint/<ts>/summary-<ts>.json <model>
```

### Self-served models (vllm@)

For open weights that no provider serves, rent a GPU and serve them with vLLM.
Three scripts, in order:

1. **On the GPU box** (e.g. a RunPod pod from the PyTorch template; an 80 GB
   H100 holds a 27B model in BF16, a 141 GB H200 holds it with room for three
   concurrent lens sessions): `VLLM_API_KEY=<secret> sh bench/vllm-serve.sh`.
   It installs vLLM, pins the weights to the current commit, prints the
   revision and the vLLM version, and serves on localhost:8000. `MODEL`,
   `MAX_LEN`, `EFFORT` and `SPEC=1` override the defaults. A gated repo
   (ThinkingCap's are) needs its license accepted on the model page and a
   read token in the box's `~/.cache/huggingface/token` first.
2. **On the bench machine**, tunnel the port (`ssh -N -L 8000:localhost:8000
   root@<pod-ip> -p <port>`) and run `bench/vllm-preflight.sh
   vllm@<repo>` with `VLLM_BASE_URL=http://localhost:8000` and the same
   `VLLM_API_KEY`. It forces one tool call and compares the harness's output
   tokens with the server's own counter: if reasoning tokens are missing from
   the usage the harness reads, a thinking-efficient model would look cheaper
   than it is, and the sweep would measure the wrong thing.
3. `make bench MODEL=vllm@<repo> TASK=go,design,rust,java,typescript,csharp,cpp,python`
   with the same two variables set. `run.sh` refuses to start unless the
   server is reachable and serves exactly that model.

The tunnel, not the provider's https proxy, is the intended path: RunPod's
proxy sits behind a ~100 s gateway timeout that a long prefill can hit before
the first byte. A `vllm@` row's dollar figure is notional -- priced at the
paired baseline's rate card in `report.py`, because the real bill is GPU hours
-- and its wall clock is one rented GPU's, not a provider fleet's. Write both
down in `models.txt`. Terminate the pod afterwards; a stopped pod still bills
for its disk.

## History

The full field was re-swept on 2026-08-20, the day the bench changed: 19
candidates, one repeat each, 95 sessions, zero contract failures except the two
models that have failed before (`nemotron-3-super` 19/100 with one, and
`mistral-large-3` 3/100 with two -- its third measured contract failure, and
5127s for the privilege). What the old 20-seed bench could not see, this one
did: `deepseek-v4-pro` (66) is 14 points ahead of the `deepseek-v4-flash` (52)
holding the seat, `opus-4-8` (52) is 23 behind `opus-5` (75), and
`nemotron-3-ultra` -- which returned nothing at all on the old code target --
scores 63 while spending 1.36M tokens and 31 minutes to do it. Numbers in
`results.csv`, per-seed tables in `results/`.

`glm-5.3-flash:cloud` was added and measured on 2026-08-28, the day it reached
the library (320B total / 18B active, 1M context, one floating `:cloud` tag).
One repeat: **62/100** — code 48/60, design 14/40, zero unmatched on either
task, 849k tokens, 393s, no contract failures. Two things in that row are worth
more than its rank. Its **code** recall is the highest any ollama model has
posted here and third in the whole field, behind only `claude-opus-5` (55) and
the `claude` baseline (52) — ahead of `claude-fable-5`. And it is the largest
marginal contributor measured against the seated panel: over
claude+codex+minimax (85/100) it adds **3** seeds — CA08, EX07, D11 — where
`deepseek-v4-pro` adds 1 and `kimi-k3` adds 1, tying `qwen/qwen3.8-max` at
88/100 for a third of its wall clock. It is not a swap candidate, though:
claude+codex+glm-5.3-flash covers 83, below the 85 minimax holds today, because
minimax finds five seeds this model does not (C26, S05, D09, D24, D29). Recompute
these from the per-seed tables in `results/`; the report does not.

Before any seat talk it needs the 3 repeats the decision rule asks for, and on
one repeat it already fails rule 3 on code — 80.3 found_per_mtok against
minimax's 117.3, since 849k tokens is a lot for 62 points — and lands one seed
under rule 2's design floor (15/40). A fourth-seat argument would have to be
made on marginal coverage against the quota, and against the standing rule that
the panel seats ONE ollama agent so a single dead one cannot take quorum with it.

`glm-5.3:cloud` — the 753B depth tier of the same family, a different
architecture (`glm_dsa_moe`) — was measured on 2026-08-31, one repeat:
**64/100**, code 47/60, design 17/40, 2 unmatched on code, 854k tokens, 572s, no
contract failures. It outscores its own flash tier by 2 points and is the weaker
seat by every measure the panel buys with:

| as the single ollama seat | panel coverage |
|---|---|
| claude + codex (no ollama seat) | 79/100 |
| **+ minimax-m3 (seated today)** | **85/100** |
| + nemotron-3-ultra | 85/100 |
| + qwen/qwen3.8-max | 84/100 |
| + glm-5.3-flash | 83/100 |
| + kimi-k3 | 83/100 |
| + glm-5.3 | 82/100 |
| + deepseek-v4-pro | 82/100 |

As a fourth seat it adds **1** seed (D19), tying deepseek-v4-pro (D12) and kimi-k3
(C24) at the bottom of that list, where its own flash tier adds 3. It also spends
59% more quota than minimax for the privilege (13.3k tokens per point against
9.4k) at the same wall clock, and it is the first GLM row here with unmatched
findings.

So the flash/pro question this bench asked of deepseek gets the same answer from
GLM, and it is worth stating as a pattern rather than a coincidence: **the depth
tier is the better reviewer and the smaller marginal contributor.** deepseek-v4-pro
beats minimax 66 to 57 alone and adds 1 seed where minimax adds 4; glm-5.3 beats
glm-5.3-flash 64 to 62 and adds 1 where flash adds 3. Depth buys seeds the panel
already has. The two GLM tiers share only 54 of their seeds — flash finds 8 the
depth tier misses — so they are genuinely different reviewers, not one model at
two sizes, which is why the smaller one is the better panel member.

No panel change on this measurement. If the ollama seat is ever widened, the
measured candidate is glm-5.3-**flash** at 88/100, not glm-5.3.

### Every scored run is re-scorable, now

`bench/summaries/` holds a re-scoring extract of every run `score.py` has
scored — findings and per-step usage, about a fifth the size of the full
summary, with no prompts, diffs or agent output. Re-score any of them for free:

```sh
python3 bench/score.py bench/manifest-go.yaml bench/summaries/<run>-go.json <model>
```

This exists because its absence already cost a sweep. The 2026-09-01
retire/replace could re-score only **15 of 43** runs onto the new scale — the
other 28 runs' `.fixpoint/` directories had been cleaned up, so 14 models,
**including the seated one**, had nothing left to re-score and had to be
archived on a scale that no longer exists (`results-archive-2026-08-20.csv`, and
the README beside it says which of its columns are still true). Every seed edit
before this was one `rm -rf .fixpoint` away from the same bill.

`results.csv` was emptied on 2026-08-20, when the bench went from 20 seeds to
100. The 46 rows measured against the old target are in git history at
`70b9113` and are **not** comparable: different targets, more lenses, and a
matcher that credits each finding once. Everything is being re-measured.

That reset also fixed a matcher bug worth remembering: the manifest reader kept
the quotation marks on every quoted keyword that was not first in its list, so
those keywords could never match anything. Multi-word keywords were silently
dead in every measurement before the reset, which biased all of them towards
false negatives.

## What this does not measure

- **Fix quality** — the coder role is a different job; this bench seats
  reviewers only.
- **Convergence behaviour over rounds** — a real panel runs review→fix
  cycles; a model that repeats itself round after round costs more than one
  bench run shows. The panel measurements memory tracks that from live runs.
- **Proposal quality for create-design** — a good reviewer is evidence of a
  good proposer, not proof. Seat a model for review first; judge its
  proposals on the first live create run.
