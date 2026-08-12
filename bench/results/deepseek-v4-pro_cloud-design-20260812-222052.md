# deepseek-v4-pro:cloud · design · run 20260812-222052 (repeat 1)

recall **8/8** · 11 finding(s), 1 unmatched · 103106 tokens · 69s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: No idempotency guarantee for retried sync batches — duplicat |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Transaction semantics contradict partial-application behavio; design-failure: No idempotency guarantee for retried sync batches — duplicat |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Client wall clock as sole conflict-resolution authority — NT |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Wiping notes.db on 401 destroys un-synced offline edits, con |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded outbox with no backpressure — disk exhaustion on l; design-failure: No batch size limit on sync POST — large outbox may exceed H |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Wiping notes.db on 401 destroys un-synced offline edits, con; design-failure: Two processes sharing one SQLite file with no concurrency co |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check doesn't detect database failure — dead instance; design-failure: No sync observability — operator cannot distinguish 'working |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: 'Deploys are quick' is not a schema migration safety guarant |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) DESIGN.md:39 — LWW conflict resolution silently discards user edits, contradicting the 'never lost' goal
