# minimax-m3:cloud · design · run 20260813-002114 (repeat 1)

recall **8/8** · 11 finding(s), 1 unmatched · 70089 tokens · 66s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Retries have no idempotency, so partial batch failure duplic; design-failure: Server is described as both single-transaction and row-by-ro |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Server is described as both single-transaction and row-by-ro |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Auth: a 401 from the server wipes the local notes database; design-failure: Last-writer-wins on client wall clock silently discards edit |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Auth: a 401 from the server wipes the local notes database |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Outbox is uncapped and sync POSTs the entire outbox in one r; design-failure: Postgres is the only stateful component but has no scaling o |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Auth: a 401 from the server wipes the local notes database; design-failure: Desktop client and CLI open the same SQLite file with no con |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Auth: a 401 from the server wipes the local notes database; design-failure: /healthz is dependency-free and is what the load balancer us |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys claimed compatible with the schema, with 'de |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) DESIGN.md:19 — Deletions are a flag with no tombstone handling, so deletes can be undone by a stale clock
