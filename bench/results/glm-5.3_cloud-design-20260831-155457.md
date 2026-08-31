# glm-5.3:cloud · design · run 20260831-155457 (repeat 1)

recall **17/40** · 22 finding(s), 0 unmatched · 239181 tokens · 288s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Retry and partial-application semantics are contradictory an |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-data: Batch atomicity contract is self-contradictory between secti |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Cursor derived from client wall clocks silently drops edits  |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: 401 on refresh wipes notes.db, destroying unacknowledged off; design-data: Auth-failure wipe deletes the outbox, destroying unacknowled |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-data: 'Clears the outbox' as written discards edits appended durin |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Two processes open one SQLite file with no concurrency polic; design-data: Two processes share one SQLite file with no named writer or  |
| D07 | — | healthz checks process liveness only; an instance with a dead DB conne |  |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: Entire outbox in one unbounded POST against a server with no |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Identical retry schedule with no jitter; no rate limit or co |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-data: Timestamp cursor with strict '>' silently and permanently dr |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query is a full-table seq scan per sync at target sc; design-data: Changes scan has no index, no pagination, unbounded rows, an |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Versionless wire protocol and no client version recorded any |
| D19 | YES | in-flight syncs are killed on every rolling deploy and the design lean | design-failure: Rolling-deploy premise is false: two versions do run concurr |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Nightly pg_dump contradicts the no-loss guarantee and restor; design-data: Nightly backup plus unrehearsed restore contradicts the neve |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: The support-dashboard goal is unmeetable: no device identity |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: Random 32-bit client note ids with no collision detection; design-data: 32-bit random note ids collide at the design's own stated sc |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-data: Account deletion orphans note rows and their deletion lifecy |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | YES | the server holds no per-device state at all, so it cannot tell two dev | design-data: No sync record makes the stated support goal unanswerable |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
