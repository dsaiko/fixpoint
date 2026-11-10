# claude-fable-5-2 · design · run 20261110-145257 (repeat 1)

recall **19/34** · 38 finding(s), 0 unmatched · 22352 tokens · 265s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Batch semantics contradict themselves, and a row that fails ; design-data: Batch atomicity is specified three incompatible ways and ret |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Cursor built on client wall-clock modified_at permanently sk; design-failure: Last-writer-wins on unsynchronized device clocks discards us |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Entire unbounded outbox in one request: a device offline lon; design-failure: No per-user rate limit and no server-side size limit: one cl |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Recovery spike is answered by adding stateless instances whi; design-failure: Two processes share notes.db with no stated locking, schema  |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz reports only process liveness, so an instance that  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploy with migrate-first assumes old and new code n; design-data: Rolling deploys run two versions against one schema, which t |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server request timeout and no connection draining: slow s |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: The failures users hit are invisible: stuck clients, rejecte |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query has no supporting index and no pagination; the |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-failure: No protocol version, with clients that cannot be upgraded on; design-data: No protocol or schema version, while notes are replaced whol |
| D19 | YES | in-flight syncs are killed on every rolling deploy and the design lean | design-failure: Blind retry of the same POST with no idempotency key; a lost |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Nightly pg_dump with an unrehearsed restore, and no protocol; design-data: A restore from the nightly dump leaves server and clients pe |
| D21 | YES | the 30-second sync goal is unreachable with a 15-minute foreground tim | design-failure: Sync triggers cannot meet the 30-second goal and align into  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: No record of which device synced what, so the stated support |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | YES | the outbox stores a full copy of each body beside the note table, with | design-data: Outbox duplicates the full note body per edit and must be se |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Tombstones are kept forever in both databases |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-failure: One cursor per user account cannot represent what each devic; design-data: One cursor per user account, but sync position is a per-devi |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | YES | a reserved shared_with column is not a sharing design; the migration i | design-data: Reserved `shared_with` field and URL-in-row attachments comm |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-failure: Client clears the whole outbox on acknowledgment, deleting e; design-data: Client clears the whole outbox on response, dropping edits i |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
