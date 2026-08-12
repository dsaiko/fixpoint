# nemotron-3-ultra:cloud · design · run 20260812-214256 (repeat 1)

recall **8/8** · 11 finding(s), 1 unmatched · 494311 tokens · 678s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Server partial-apply semantics break retry safety; design-failure: No idempotency keys — duplicate edits on client crash after  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Server partial-apply semantics break retry safety |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Auth 401 wipes local database on any refresh failure; design-failure: Clock-skew conflict resolution loses data silently |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Auth 401 wipes local database on any refresh failure |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded outbox growth causes disk exhaustion; design-failure: Unbounded retries with no circuit breaker or max attempts |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: SQLite concurrent access by desktop client and CLI with no c |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health endpoint ignores database and disk — LB sends traffic |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Deploy assumes quick migrations with no rollback or backward |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) DESIGN.md:28 — Cursor semantics undefined for recovery and partial failure
