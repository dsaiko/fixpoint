# claude-opus-5 · design · run 20260812-221216 (repeat 1)

recall **8/8** · 12 finding(s), 1 unmatched · 14180 tokens · 185s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Server contradicts itself on batch atomicity, and the client; design-failure: Unbounded outbox sent as a single POST wedges exactly the de |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Server contradicts itself on batch atomicity, and the client; design-failure: Unbounded outbox sent as a single POST wedges exactly the de |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-write-wins on client wall clock silently discards edits |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: A 401 on refresh wipes notes.db, destroying the unsynced out; design-failure: Retry loop has no terminal state, no cap, and no signal to t |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Server contradicts itself on batch atomicity, and the client; design-failure: Unbounded outbox sent as a single POST wedges exactly the de |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: A 401 on refresh wipes notes.db, destroying the unsynced out; design-failure: Two processes share notes.db with no coordination across the |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores every dependency, so a database outage keep |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploy is asserted not to overlap versions, and ther |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) DESIGN.md:19 — Tombstones accumulate forever with no retention policy
