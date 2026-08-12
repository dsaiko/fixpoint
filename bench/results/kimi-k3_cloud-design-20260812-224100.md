# kimi-k3:cloud · design · run 20260812-224100 (repeat 1)

recall **8/8** · 8 finding(s), 0 unmatched · 64031 tokens · 76s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Retry of the full outbox POST is not idempotent; a lost resp |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: §5 contradicts itself: "single transaction per request" vs " |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflict resolution on client wall clock silently discards e |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Any 401 wipes notes.db including the un-synced outbox, destr |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Retry of the full outbox POST is not idempotent; a lost resp; design-failure: §5 contradicts itself: "single transaction per request" vs " |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Any 401 wipes notes.db including the un-synced outbox, destr; design-failure: Two long-lived processes open the same notes.db directly, wi |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores the database, so a full DB outage looks hea |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: "Old and new versions never run concurrently because deploys; design-failure: Two long-lived processes open the same notes.db directly, wi |
