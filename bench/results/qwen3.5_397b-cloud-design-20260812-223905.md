# qwen3.5:397b-cloud · design · run 20260812-223905 (repeat 1)

recall **7/8** · 9 finding(s), 0 unmatched · 64631 tokens · 28s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Partial batch failure leaves client and server inconsistent |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Clock skew causes silent data loss in conflict resolution |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Auth revocation wipes local notes permanently |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Entire outbox in one request has no size bound; design-failure: Partial batch failure leaves client and server inconsistent |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Concurrent SQLite access from multiple processes is unsafe; design-failure: Crash mid-write can corrupt SQLite database |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check does not detect database unavailability |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Schema migration assumes zero-downtime deploys |
