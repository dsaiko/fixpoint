# kimi-k3:cloud · design · run 20260812-211845 (repeat 1)

recall **0/8** · 0 finding(s), 0 unmatched · 0 tokens · 0s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | — | conflict resolution by client wall clock silently discards edits, cont |  |
| D04 | — | a failed token refresh wipes notes.db including the unsynced outbox —  |  |
| D05 | — | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro |  |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | — | healthz checks process liveness only; an instance with a dead DB conne |  |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |
