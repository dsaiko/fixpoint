# deepseek-v4-flash:0731-cloud · design · run 20260812-211248 (repeat 1)

recall **8/8** · 8 finding(s), 0 unmatched · 86661 tokens · 145s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: One permanently failing row makes the batch unacknowledgable; design-failure: Retry-until-acknowledged assumes an idempotent apply the pro |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: One permanently failing row makes the batch unacknowledgable |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-write-wins on client wall clocks silently discards offl |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Any refresh 401 — including password change or token corrupt; design-failure: Whole outbox shipped as one unbounded POST with no chunking, |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: One permanently failing row makes the batch unacknowledgable; design-failure: Whole outbox shipped as one unbounded POST with no chunking, |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Any refresh 401 — including password change or token corrupt; design-failure: Desktop and CLI write one SQLite file directly with no locki |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores dependencies, so a dependency-dead instance |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys run old and new versions against one migrate |
