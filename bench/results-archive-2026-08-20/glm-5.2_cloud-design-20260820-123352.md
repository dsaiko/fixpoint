# glm-5.2:cloud · design · run 20260820-123352 (repeat 1)

recall **16/40** · 19 finding(s), 0 unmatched · 193431 tokens · 204s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Self-contradiction: 'single transaction per request' vs 'row |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Cursor is client wall-clock modified_at with strict > compar; design-data: modified_at as sync cursor silently drops clock-skewed and s |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: A 401 from token corruption wipes notes.db including the un-; design-data: Auth 401 wipes notes.db, destroying unacknowledged offline e |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Outbox is uncapped and stores a full body copy per edit — lo |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-data: Server applies edits with no timestamp guard; retransmission |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores DB health and a never-syncing client is 'no |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys with no drain plus no protocol version — old |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No request timeout, single transaction per request, no pagin; design-failure: Identical retry schedule with no jitter plus no per-user rat |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query has no supporting index, so every sync full-sc |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | — | nightly pg_dump means up to 24h data loss, and the restore procedure h |  |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: No sync audit trail makes the support-engineer goal unanswer |
| D24 | YES | the conflict loser is discarded with no copy kept, so a wrong resoluti | design-data: Only the latest note version is kept; superseded bodies are  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-data: 32-bit random note id will collide at target scale and silen |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-data: Tombstones and orphaned deleted-account notes retained forev |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-data: Account deletion orphans the user's notes with no cascade or |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: One cursor stored per user account cannot serve multiple dev |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | YES | a delete on one device racing an edit on another is resolved by timest | design-failure: Last-write-wins lets a stale edit resurrect a note the user  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
