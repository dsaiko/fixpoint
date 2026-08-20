# qwen3.5:397b-cloud · design · run 20260820-125004 (repeat 1)

recall **14/40** · 17 finding(s), 0 unmatched · 153371 tokens · 113s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Sync protocol lacks idempotency for retry after partial fail |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Clock-skew causes silent data loss with no recovery; design-data: LWW conflict resolution loses data with clock skew |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Auth revocation wipe loses unsynced offline edits |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Uncapped outbox becomes unrecoverable at scale |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check does not detect database failures |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No request timeout enables connection exhaustion |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | YES | no per-user rate limit and no backpressure; one looping client can con | design-failure: No rate limiting exposes server to abusive clients |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: Changes query has no pagination for large result sets |
| D16 | — | the changes query filters on (user_id, modified_at) with no index plan |  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | YES | in-flight syncs are killed on every rolling deploy and the design lean | design-failure: Deploy drops in-flight requests risking partial sync |
| D20 | — | nightly pg_dump means up to 24h data loss, and the restore procedure h |  |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: No audit trail contradicts Goal 5 |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: 32-bit random note IDs will collide at stated scale; design-data: 32-bit random note IDs will collide at scale |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | YES | the server keeps only the latest body, so nothing can reconstruct a no | design-data: No version history prevents undo and audit |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Tombstones accumulate forever causing unbounded growth; design-data: Tombstones accumulate with no cleanup strategy |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-data: Account deletion leaves orphaned notes |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
