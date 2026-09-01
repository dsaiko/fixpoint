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
make bench MODEL=glm-5.2:cloud                 # the calibrated pair, 1 repeat
make bench MODEL=kimi-k3:cloud TASK=go N=3     # one target, 3 repeats
make bench MODEL=kimi-k3:cloud TASK=go,design  # a list of targets
make bench-check                               # validate the manifests, no model
make bench-seeds                               # per-seed hit rate: what still discriminates
python3 bench/report.py                        # scores, plus recall per target
python3 bench/report.py --html out.html        # the same as a shareable page
```

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
yaml; a `claude-*` id runs claude.yaml with the model swapped; an id with a
slash (`x-ai/grok-4.6`) runs the OpenRouter template (card-billed, cached;
needs `OPENROUTER_API_KEY`); a `codex@`-prefixed OpenRouter id runs the same
route through the CODEX harness instead -- for models whose replies the claude
harness cannot read (OpenRouter's Anthropic translation puts a trailing
signature-only thinking block after some models' text, and the harness's result
comes out empty; gemini-3.7-flash is the measured case). A codex@ number is
comparable to the codex baseline, which shares its harness -- say so when
quoting it. Anything else runs the ollama template. Results append
to `results.csv` (committed — measurements are project knowledge); per-run
seed tables land in `results/`. Re-score an old run without re-paying for it:

```sh
python3 bench/score.py bench/manifest-go.yaml .fixpoint/<ts>/summary-<ts>.json <model>
```

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
