# Review lenses and assignment strategies

What a lens is, how to write one, and how lenses are handed to agents.

[← back to the README](../README.md)

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

Prompts under [config/prompts/](../config/prompts/) are a library you can grow freely; only the
ones referenced in the configuration are used. The shipped lenses:

- [review-bugs.md](../config/prompts/review-bugs.md) — correctness
- [review-security.md](../config/prompts/review-security.md) — security
- [review-concurrency.md](../config/prompts/review-concurrency.md) — concurrency
- [review-tests.md](../config/prompts/review-tests.md) — test coverage of what the run changed
- [review-maintainability.md](../config/prompts/review-maintainability.md) — smells, simplification, docs (advisory; **shipped but not in any panel**)
- [review-design.md](../config/prompts/review-design.md) — architecture (advisory; **shipped but not in any panel**)
- [fix.md](../config/prompts/fix.md) — the coder's instructions

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
