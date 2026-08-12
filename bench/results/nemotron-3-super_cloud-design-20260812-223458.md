# nemotron-3-super:cloud · design · run 20260812-223458 (repeat 1)

recall **5/8** · 4 finding(s), 0 unmatched · 213427 tokens · 166s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial transaction apply risks data inconsistency |
| D03 | — | conflict resolution by client wall clock silently discards edits, cont |  |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Auth failure causes irreversible data loss |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded outbox growth risks resource exhaustion |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Auth failure causes irreversible data loss |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check ignores critical dependencies |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |
