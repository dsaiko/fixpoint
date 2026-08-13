# x-ai/grok-4.6 · design · run 20260813-091551 (repeat 1)

recall **8/8** · 7 finding(s), 0 unmatched · 45798 tokens · 180s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Partial batch apply cannot be retried or acknowledged |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial batch apply cannot be retried or acknowledged |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-write-wins on client clocks silently discards the real  |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Any refresh 401 wipes the outbox, including unsynced notes |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded outbox is sent as one POST with no backpressure; design-failure: Partial batch apply cannot be retried or acknowledged |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Any refresh 401 wipes the outbox, including unsynced notes; design-failure: Partial batch apply cannot be retried or acknowledged |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz stays 200 when the process cannot serve |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys are treated as if mixed versions never run |
