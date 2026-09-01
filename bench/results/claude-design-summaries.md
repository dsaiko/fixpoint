# claude · design · run summaries (repeat 1)

recall **20/34** · 37 finding(s), 0 unmatched · 31990 tokens · 412s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Batch apply is described as both atomic and partial, and a r; design-data: Batch application is specified as both atomic and partially  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-writer-wins on client wall clock, with no version histo; design-data: Last-write-wins on client wall clock discards the losing bod |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded outbox holding a full body copy per edit, shipped  |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Two processes share notes.db with no stated locking or journ; design-data: Two processes write notes.db with no owner, no locking, and  |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz cannot distinguish quiet from broken, and the ratio; design-data: /healthz ignores the database, so instances that cannot serv |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys do run two versions at once, and nothing mak; design-data: Rolling deploy runs old and new code against one schema, con |
| D09 | — | §5 removes the server-side request timeout, so a slow or hostile clien |  |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Identical un-jittered backoff plus no rate limit turns the a |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Timestamp cursor with strict '>' silently and permanently sk; design-data: Wall-clock cursor with strictly-newer reads permanently skip |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: Tombstones kept forever make first sync and every scan grow ; design-data: Unbounded batch and unbounded change set can wedge a device  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Unpaginated, unindexed, untimed /sync makes the connection p; design-data: The changes query has no index on its filter and sort column |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: The wire format carries no version, so no incompatible chang |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Restoring from the nightly dump leaves every client permanen; design-data: Nightly unrehearsed pg_dump is the only history behind an ov |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: No sync audit trail directly contradicts the stated support ; design-data: No sync audit trail, so the stated support requirement canno |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | YES | every note has two identities (client id, server BIGSERIAL) with no st | design-data: Retries are not idempotent: no idempotency key and no unique |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Account deletion leaves every note row in place, unbounded a; design-data: Deletion keeps the note body forever on every device and on  |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: Sync progress is per-device state but stored once per user a |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: notes.db written with default permissions on shared-account  |
| D35 | YES | a reserved shared_with column is not a sharing design; the migration i | design-failure: A reserved shared_with column does not avoid the migration i |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | YES | attachments are a URL in the note row with no lifecycle: nothing delet | design-data: Attachment URLs in note rows leave blob lifecycle undefined |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-failure: Step 4 clears the whole outbox, deleting edits made during t; design-data: Sync clears the entire outbox, discarding edits made during  |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
