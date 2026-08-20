# claude-fable-5 · design · run 20260820-130837 (repeat 1)

recall **17/40** · 27 finding(s), 0 unmatched · 26566 tokens · 328s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: A validation-failing outbox row wedges the device forever; p; design-data: Partial batch application contradicts the single-transaction |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Skewed-clock LWW silently destroys newer edits and the losin; design-data: Wall-clock last-writer-wins discards the losing body with no |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: 401-triggered wipe of notes.db destroys unacknowledged offli; design-data: Sign-out on refresh 401 wipes notes.db including unsynced ou |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Clearing the outbox after acknowledgment drops edits appende; design-failure: Unbounded single-request sync: no body cap, no server timeou |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | — | healthz checks process liveness only; an instance with a dead DB conne |  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploy claim 'old and new never run concurrently' is; design-data: Rolling deploys guarantee old code runs against the migrated |
| D09 | — | §5 removes the server-side request timeout, so a slow or hostile clien |  |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Post-outage reconnect storm: identical backoff schedule, no  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-data: Cursor is client wall-clock modified_at, so late-timestamped |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: Client-wall-clock modified_at cursor makes changes permanent; design-failure: No signal distinguishes a silently broken client from a quie |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-data: Unbounded single-request sync (no pagination, no size cap, n |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query is a full table scan on every sync at target s |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-failure: Unversioned protocol meets arbitrarily old offline clients; design-data: No protocol version despite clients that can lag arbitrarily |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Backup restore silently loses acknowledged edits with no rec; design-data: Restoring a backup silently diverges from clients because cu |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | — | §1 promises support can answer 'why is my note missing' from dashboard |  |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: Random 32-bit client note ids collide at the stated scale, s; design-data: Random 32-bit client note ids will collide at the stated sca |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-failure: Account deletion leaves all note bodies in the database inde; design-data: Account deletion leaves note bodies in place indefinitely |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: One sync cursor per user account makes multi-device sync mis |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | YES | the server holds no per-device state at all, so it cannot tell two dev | design-data: No record of syncs makes the stated supportability goal unan |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-data: Client clears the entire outbox on acknowledgment, deleting  |
