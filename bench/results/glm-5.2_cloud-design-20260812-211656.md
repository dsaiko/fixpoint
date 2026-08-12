# glm-5.2:cloud · design · run 20260812-211656 (repeat 1)

recall **8/8** · 7 finding(s), 0 unmatched · 106078 tokens · 98s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Server batch handling is self-contradictory: single transact |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Server batch handling is self-contradictory: single transact; design-failure: Client-side commit (apply changes + advance cursor + clear o |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-write-wins on client wall-clock silently discards offli; design-failure: Client-side commit (apply changes + advance cursor + clear o |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: 401 from token corruption wipes notes.db including the un-sy |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded outbox sent as one POST in one transaction — sync ; design-failure: Client-side commit (apply changes + advance cursor + clear o |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: 401 from token corruption wipes notes.db including the un-sy |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores DB reachability, so the LB keeps routing to |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling-deploy claim "old and new never run concurrently" is |
