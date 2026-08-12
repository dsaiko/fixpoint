# kimi-k2.7-code:cloud · design · run 20260812-211942 (repeat 1)

recall **8/8** · 9 finding(s), 0 unmatched · 60939 tokens · 87s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Retry replays edits because there is no idempotency key |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial batch application contradicts single-transaction syn |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Retry replays edits because there is no idempotency key; design-failure: Conflict resolution trusts unvalidated client wall-clock tim |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Auth 401 response wipes unsynced local notes |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Outbox has no size cap and is sent as a single batch; design-failure: A single failing edit can block the entire outbox forever |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Desktop client and CLI share the SQLite file without a concu |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz is liveness-only and ignores the database |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Deployment assumes old and new code never overlap and lacks  |
