# minimax-m3.1-flash:cloud · design · run 20261019-172108 (repeat 1)

recall **13/34** · 28 finding(s), 0 unmatched · 221935 tokens · 1583s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Any response counts as acknowledgment, so a partial failure ; design-data: The response carries no per-edit result while the server app |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Client-wall-clock cursor with strict '>' silently loses edit; design-data: Sync cursor is a client wall clock used as an exclusive lowe |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded outbox with a full body copy per edit, sent as one; design-data: The entire unbounded outbox travels in one request, with no  |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Apply, cursor advance and outbox clear are not stated to be ; design-data: The desktop client and the CLI open the same SQLite file wit |
| D07 | — | healthz checks process liveness only; an instance with a dead DB conne |  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: 'Old and new versions never run concurrently' contradicts th; design-data: 'Deploys are rolling' and 'old and new versions never run co |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server request timeout and no rate limit let one client h |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Identical backoff with no jitter turns recovery into a synch |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: The changes query has no index beyond the primary key, so th; design-data: The change feed has no index and no pagination at the stated |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: 'The protocol carries no version number' is not safe once th |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Backups are a 24-hour-old untested pg_dump with no point-in-; design-data: Nightly dumps with no point-in-time recovery and an unrehear |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: The server keeps no per-device record and no history, making; design-data: The headline support goal - 'why does this device not have m |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Account deletion leaves the notes behind indefinitely, with  |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: 'One cursor is stored per user account' contradicts the prot |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
