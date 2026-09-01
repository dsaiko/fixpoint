# minimax-m3:cloud · design · run 20260901-101125 (repeat 1)

recall **13/34** · 21 finding(s), 0 unmatched · 180870 tokens · 397s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Retries have no idempotency key, so partial applies double-w |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-data: Batch atomicity is broken by design: partial application con |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-Writer-Wins on client wall clock silently loses edits; design-data: Cursor is a client wall-clock value; the design assumes shar |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Outbox stores the full body of every edit, with no cap |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | — | healthz checks process liveness only; an instance with a dead DB conne |  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys contradict 'old and new versions never run c |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server timeout, no rate limit, no body cap — a single cli; design-data: Unbounded body, unbounded outbox, no server timeout, no rate |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Sync query has no index and returns every column; design-data: Changes query has no pagination and no (user_id, modified_at |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-failure: Wire format 'no version number' is forward-only |
| D19 | YES | in-flight syncs are killed on every rolling deploy and the design lean | design-data: Wire format has no version number; rolling deploys do not dr |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Restore procedure documented but never rehearsed |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: No audit trail contradicts the support-engineer goal; design-data: Conflict resolution discards the losing body and there is no |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-data: Tombstones and orphan user-deleted notes are kept 'forever'  |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-failure: Clearing the outbox on ack loses edits made during an in-fli |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  |
| D31 | — | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
