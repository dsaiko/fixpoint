# beam:cloud · design · run 20261019-170419 (repeat 1)

recall **17/34** · 29 finding(s), 0 unmatched · 43895 tokens · 90s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial apply on validation failure plus unspecified idempot; design-data: Partial application contradicts the stated single-transactio |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-writer-wins by client clock silently discards edits; design-data: Conflict resolution overwrites and discards a losing acknowl |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Uncapped note body plus full-body outbox rows allow unbounde; design-data: The outbox is unbounded and stores a full body copy per edit |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Desktop and CLI open the same SQLite file without specified ; design-data: Desktop and CLI concurrently open the same SQLite file witho |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check reports liveness only, so load balancers route  |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server request timeout lets syncs hold connections indefi |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | YES | no per-user rate limit and no backpressure; one looping client can con | design-failure: No per-user or per-device rate limit makes tight-loop client |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Cursor keyed on client wall-clock milliseconds misses equal- |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: A client that never syncs is not an alertable signal |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: Changes query has no pagination, index, or ordering tie-brea |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-data: Change fetches have no pagination or supporting index |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | YES | in-flight syncs are killed on every rolling deploy and the design lean | design-failure: Rolling deploy assumes old and new never run together, which |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Backups are unrehearsed and retention is only 14 days |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: No audit trail or provenance is retained despite the support |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Deleted rows retained forever plus orphaned account-deletion; design-data: Deleted notes and tombstones are retained forever |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-failure: A single cursor stored per user account breaks multi-device ; design-data: A single shared account cursor cannot represent multiple dev |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: notes.db is written in the clear with default file permissio |
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
