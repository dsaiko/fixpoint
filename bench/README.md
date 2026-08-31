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
| **cost** | input+output tokens (CLI-reported, real) and wall-clock seconds | tokens are the quota drawdown; wall clock is what gates a panel round, which runs at the slowest reviewer's pace |
| **found_per_mtok** | matched seeds per million tokens | the headline: quality per unit of the thing being spent |

The two tasks are one benchmark: **60 code seeds + 40 design seeds = 100
points**, one point per seed. `bench/report.py` sums a model's two rows into
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

| baseline | code | design | **score** | tokens | wall |
|---|---|---|---|---|---|
| claude (opus-5, effort high) | 52/60 | 23/40 | **75/100** | 76k | 901s |
| codex | 45/60 | 16/40 | **61/100** | 315k | 941s |

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

- **Frozen targets.** `testdata/target-code` (a Go link shortener in seven
  files, 60 seeded defects: correctness for `review-bugs`, concurrency for
  `review-concurrency`, and 14 security defects — unsalted hashes, a
  predictable token, an open redirect, a path traversal, a client-controlled
  admin header — for `review-security`) and `testdata/target-design` (a
  sync-service design, 40 seeded flaws: 24 failure-and-operation for
  `design-failure`, 16 data-model for `design-data`). **Never fix the seeded
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
make bench MODEL=glm-5.2:cloud                 # both tasks, 1 repeat
make bench MODEL=kimi-k3:cloud TASK=code N=3   # one task, 3 repeats
make bench-check                               # validate the manifests, no model
python3 bench/report.py                        # every model's score out of 100
```

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
python3 bench/score.py bench/manifest-code.yaml .fixpoint/<ts>/summary-<ts>.json <model>
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
