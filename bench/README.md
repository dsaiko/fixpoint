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
| **recall** | seeded defects found / seeded (12 code, 8 design) | quality against ground truth — no judge, no taste, no drift between runs |
| **noise** | findings matching no seed | not automatically false (models find real unseeded bugs; skim the per-run report before dismissing) — but a model whose output is mostly unmatched is expensive to triage |
| **cost** | input+output tokens (CLI-reported, real) and wall-clock seconds | tokens are the quota drawdown; wall clock is what gates a panel round, which runs at the slowest reviewer's pace |
| **found_per_mtok** | matched seeds per million tokens | the headline: quality per unit of the thing being spent |

### The disqualifier

**Contract compliance.** Every output-contract failure in this project's
history came from a non-Anthropic model driving this harness. A session that
produces unparseable output burns a full slot and, in a quorum, can flip the
verdict to INCONCLUSIVE. Any contract failure or timeout across the sweep is
recorded in `results.csv` (`errors`); more than one across three repeats
disqualifies regardless of recall.

### The decision rule

Run **claude and codex through the bench first** — they calibrate it. A seed
neither baseline finds is a bad (or genuinely hard) seed; baseline recall is
the 100% mark that makes candidate recall readable.

A candidate earns a seat when, over ≥3 repeats:

1. zero-or-one contract failures (see above), and
2. median recall ≥ **2/3 of the claude baseline** on the same task, and
3. `found_per_mtok` beats the weakest current seat-holder on the same task —
   a seat is comparative: the question is never "is it good", it is "does it
   beat what the seat currently pays for".

Diversity is a tie-breaker, not a metric: *which* seeds a model finds is in
the per-run report, and a model that reliably finds the seeds the rest of the
panel misses is worth more than its recall says (measured corroboration
across the panel is <4%, so non-overlap is the panel's whole value).

## Protocol

- **Frozen targets.** `testdata/target-code` (Go link shortener, 12 seeded
  defects: 8 correctness for the `review-bugs` lens, 4 concurrency for
  `review-concurrency`) and `testdata/target-design` (a sync-service design,
  8 seeded failure-mode flaws for `design-failure`). **Never fix the seeded
  bugs** — the target's value is that it does not change between
  measurements. `**/testdata/**` is excluded by the default config, so real
  review runs over this repository never trip on them.
- **One variable.** Every candidate runs the identical harness wrapping
  (`bench/agent-template.yaml` = the kimi-ollama.yaml invocation with only
  the model swapped), identical prompts, identical targets. No judge, no
  refutation — they would measure claude's filtering, not the candidate.
- **3 sessions per candidate per repeat** (2 code lenses + 1 design lens), so
  a seven-model sweep is ~21 sessions per repeat — sized to fit a weekly
  quota alongside real work. Sweep once at 1 repeat, then re-run the
  finalists at 3 repeats before deciding: single-run recall on a dozen seeds
  is noisy, and the variance itself is information (a model that finds 9
  then 4 is worse than one that finds 6 twice).
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

## Running it

```sh
make bench MODEL=glm-5.2:cloud                 # both tasks, 1 repeat
make bench MODEL=kimi-k3:cloud TASK=code N=3   # one task, 3 repeats
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

## What this does not measure

- **Fix quality** — the coder role is a different job; this bench seats
  reviewers only.
- **Convergence behaviour over rounds** — a real panel runs review→fix
  cycles; a model that repeats itself round after round costs more than one
  bench run shows. The panel measurements memory tracks that from live runs.
- **Proposal quality for create-design** — a good reviewer is evidence of a
  good proposer, not proof. Seat a model for review first; judge its
  proposals on the first live create run.
