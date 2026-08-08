# Logs and artifacts

What a run writes to disk, the end-of-run table, and the run journal.

[← back to the README](../README.md)

## Logs

`logs.dir` is a path template, so the layout is yours to choose. It takes
`{timestamp}` (the run's start time — one value for the whole run) and
`{round}`. The default, `.fixpoint/{timestamp}/round-{round}`, produces:

```
.fixpoint/<run-timestamp>/
  round-1/
    review-<agent>-<lens>-<timestamp>.{md,json,raw}   # one set per reviewer step
    refute-<agent>-<prompt>-<timestamp>.{md,json,raw} # each reviewer's positions, with its evidence
    judge-<agent>-<prompt>-<timestamp>.{md,json,raw}  # what the arbiter kept or dropped, and why
    fix-<agent>-fix-<timestamp>.{md,json,raw}
    *.prompt                                          # exact prompt, written at invocation start
  round-2/...
  journal.jsonl                                       # append-only state transitions, flushed as they happen
  review-body.md                                      # review-only: the document, and the bytes -post-run publishes
  review-posted                                       # written by -post-run: what it published, or that a submission may have gone out
  summary-<timestamp>.{md,json}                       # assignments, issues, verdicts, verification, termination
```

The refutation and judge artifacts are there because those passes produce judgment
*with evidence*, and the aggregate log line ("3 contested") throws the evidence
away — an outvoted refuter would otherwise leave no trace of its argument, and
"who refuted what, on what grounds" could not be answered even from a run sitting
in front of you.

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
 verify     fmt, vet, test, lint · passed in 39/39 gate run(s)

 REVIEWER  issues  fixed  rejected  deferred  advisory  errors  tokens  cache    cost    time
 claude        13     12         1         0        17       0   43.1M   100%  $45.40   1h15m
 codex         14     13         1         0         0       0   34.5M    48%       -  39m25s
 glm            1      0         1         0         0       0   13.5M     0%       -  19m38s
 kimi           8      6         2         0         0       2   36.0M     0%       -  37m42s
 ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
 TOTAL         33     28         5         0                    165.8M    59%  $87.78
 rows sum above the total: 3 issue(s) were reported by more than one reviewer

 LENS                    issues  fixed  rejected  deferred  advisory  errors    time
 review-bugs                 20     15         1         4         0       0  35m21s
 review-security             12     10         0         2         0       1  44m44s
 review-tests                26     12         0        14         0       0  24m37s
 ...

 SEVERITY  issues  fixed  rejected  deferred  open  last round
 high           4      4         0         0     0           0
 medium        19     16         2         0     1           2
 low           10      8         3         0     1           3
 ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
 TOTAL         33     28         5         0     2           5
 last round reported 2 medium, 3 low
 left unresolved: 1 medium, 1 low -- neither fixed nor rejected; they are listed in the summary

 coder      claude-coder · 28 fixed · 5 rejected · 1h15m · 38.7M tok · 100% cached
 commits    39 · f40acbc26fbd a7361e82ecdc d8f4b9d429ea ...  (per_fix)
 exit       max-iterations (exit 2)
──────────────────────────────────────────────────────────────────────────────
```

It answers the question the interleaved per-round log cannot: **which agent and
which lens earned their tokens.** A panel is only worth its cost if the answer
varies between its members, and the table above is what a weak member looks like.
The `review-tests` row — the most reports, the most deferred, the least converted
into fixes — is why that lens is now `final: true`.

**The SEVERITY block is the one that answers "run it again?"** Volume cannot: a
reviewer asked for coverage or style always has more to say, so "33 issues again"
reads the same whether the run found a data race or restated its own
documentation. Severity separates those, and the `last round` column is the part
to read — it counts only what the most recent *reviewing* round reported, so a
final round with nothing above medium means the panel has stopped finding serious
defects in this tree. That is the closest thing to convergence a review loop
offers, and it is stated in words underneath (including "reported nothing", which
is the most informative outcome there is and so is never left as an absent line).

Severity is each issue's **last** reported one, matching how its verdict is
counted — deferral promotes an issue a tier, so its first severity is not the one
its verdict belongs to. A severity outside the vocabulary gets its own row rather
than being folded into a known tier: a reviewer that invents one has said
something, and filing it silently as `low` would be the summary lying about what
was reported. Note that severity is **per-agent uncalibrated** — in one measured
panel claude's 23 `high` findings were rejected 0% of the time and another
reviewer's 4 were rejected 50% — so read the row alongside the REVIEWER table
rather than on its own.

**The `cache` column is the one that explains a bill.** It is the share of an
agent's *input* served from the prompt cache, and it is the difference between two
agents whose token totals look alike. In the run above, `claude` and `kimi` both
processed ~40M tokens, but claude read 100% of its input from cache and kimi 0% —
because an uncached agentic session re-pays for the whole conversation on every
turn, and a reviewer averages tens of turns. Without the column all four rows read
as "large", which is how a cost investigation here first blamed prompt size and
then bought a cached route that fixed the rate and not the bill: the volume was the
defect. Output tokens are excluded, since only input can be cached, and an agent
that reports no usage shows `-` rather than 0%.

**Tokens and cost are what the agent's own CLI reported**, never a fixpoint
estimate. Each agent file says where its CLI puts those numbers (see `usage:` in
[config/README.md](../config/README.md)); an agent that reports nothing shows `-`.
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
verify_baseline    what was already failing before the run touched anything, or
                   `interrupted` when it was stopped before it could say
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
