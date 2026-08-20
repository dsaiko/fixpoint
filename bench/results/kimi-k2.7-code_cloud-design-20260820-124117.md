# kimi-k2.7-code:cloud · design · run 20260820-124117 (repeat 1)

recall **17/40** · 24 finding(s), 0 unmatched · 110867 tokens · 139s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Single-transaction sync is inconsistent with partial-batch a; design-data: Single-transaction sync contradicts partial application |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflict resolution trusts client wall clock and discards lo; design-data: Cursor based on client wall clock drops concurrent edits |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Refresh-token 401 blindly wipes local state including unsync; design-data: Refresh-token revocation wipes unsynced notes |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Outbox and sync request have no size bound; design-data: Outbox has no size cap |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: SQLite file is shared between desktop and CLI without a conc |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health probe checks only process liveness, not database conn |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: Server deliberately has no request timeout |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: All clients share the same fixed retry backoff, causing thun |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | YES | no per-user rate limit and no backpressure; one looping client can con | design-failure: No per-user rate limit |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: A client that stops syncing is invisible to operations |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query is unbounded and has no supporting index; design-data: Changes query is unbounded and unindexed |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-failure: Protocol has no version number |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Nightly backups with unpracticed restore and no point-in-tim |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: No sync audit trail contradicts support goal |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-data: 32-bit random note IDs will collide |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Soft-deleted notes are retained forever with no compaction |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-failure: Account deletion leaves all note content in the database; design-data: Account deletion leaves notes behind |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
