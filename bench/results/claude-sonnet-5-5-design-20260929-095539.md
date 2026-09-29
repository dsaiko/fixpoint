# claude-sonnet-5-5 · design · run 20260929-095539 (repeat 1)

recall **18/34** · 31 finding(s), 0 unmatched · 16063 tokens · 149s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Partial-apply semantics contradict the transaction, and the ; design-data: Retried batches have no idempotency key, so a lost response  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-data: Sync atomicity is contradictory and the partial-failure ack  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-write-wins on client wall clock discards data, contradi; design-data: Last-write-wins on client clock discards content and breaks  |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Whole outbox in one request with no size bound, unbounded re; design-data: Unbounded outbox with full body copies and a single all-or-n |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Two processes share notes.db and sync/clear the outbox with ; design-data: Two processes share one SQLite file with no ownership or sch |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Sync failure is invisible to operators: healthz is liveness  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: No protocol/schema version, and rolling deploy claim contrad |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: Synchronized identical backoff with no jitter, no server tim |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Unbounded, unindexed changes query is the first thing to sat; design-data: Unpaginated changes query with no index on (user_id, modifie |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: No protocol or schema version, and old clients will drop new |
| D19 | YES | in-flight syncs are killed on every rolling deploy and the design lean | design-failure: No drain on shutdown, no explicit deploy safety for in-fligh |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Restore from a nightly dump loses acknowledged edits and cli; design-data: Nightly dump gives up to 24h of acknowledged edits, and a re |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | — | §1 promises support can answer 'why is my note missing' from dashboard |  |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | YES | nothing caps a note body, so one pasted file becomes a row and a reque | design-failure: No server-side limits on body size, batch size or per-user s; design-data: Deleted notes and deleted accounts keep their content foreve |
| D29 | YES | the server keeps only the latest body, so nothing can reconstruct a no | design-data: Server keeps only the latest version and no history, so no u |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Tombstones and orphaned notes grow forever; account deletion |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-failure: Single per-user cursor on client-supplied modified_at silent; design-data: Cursor is a client wall-clock timestamp and is stored per ac |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: Remote-wipe protects nothing on an offline stolen device, an; design-data: notes.db is plaintext with default permissions, while the wi |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | YES | the server holds no per-device state at all, so it cannot tell two dev | design-data: No per-device sync state, so the stated support requirement  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  |
| D31 | — | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
