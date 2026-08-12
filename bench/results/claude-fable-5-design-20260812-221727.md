# claude-fable-5 · design · run 20260812-221727 (repeat 1)

recall **8/8** · 8 finding(s), 0 unmatched · 9891 tokens · 126s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: A validation-failing row wedges sync permanently; partial ap |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: A validation-failing row wedges sync permanently; partial ap |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflict resolution by client wall clock silently discards e |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: 401 on token refresh wipes notes.db including unsynced outbo |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Client clears the whole outbox on ack, dropping edits made d; design-failure: Unbounded outbox sent as one POST: the longest-offline devic |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: A validation-failing row wedges sync permanently; partial ap; design-failure: Desktop client and CLI share notes.db directly with no state |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores database connectivity, so a DB-dead instanc |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploy runs old code against migrated schema; 'deplo |
