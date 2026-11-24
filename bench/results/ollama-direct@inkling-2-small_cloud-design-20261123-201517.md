# ollama-direct@inkling-2-small:cloud · design · run 20261123-201517 (repeat 1)

recall **19/34** · 32 finding(s), 0 unmatched · 88185 tokens · 150s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Whole-batch resend is not idempotent; a lost ack can duplica |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial batch application contradicts 'single transaction' a; design-data: 'Single transaction' contradicts partial application, and ou |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-write-wins on client wall clock silently destroys concu; design-data: Last-write-wins on unsynchronized client wall clocks silentl |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Outbox is unbounded and always sent whole: after long offlin; design-data: Outbox stores full body copies, is uncapped, and is POSTed w |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-data: notes.db opened directly by two processes with no coordinati |
| D07 | — | healthz checks process liveness only; an instance with a dead DB conne |  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys guarantee old-and-new coexistence; migration; design-data: 'Old and new never run against the same schema because deplo |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server timeout anywhere: a stuck query holds a connection |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Identical retry schedule on every client; no jitter, no cap, |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | YES | no per-user rate limit and no backpressure; one looping client can con | design-failure: No per-user rate limit and no input caps: one account can ex |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-data: Cursor is client modified_at with strictly-greater compariso |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: Operators cannot tell 'quiet' from 'broken': DB down is invi |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: No index on (user_id, modified_at); SELECT * with no bound — |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-data: Changes query has no supporting index or pagination; every s |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Wire format carries no version and no migration path; 'addin |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Restore untested + overwrite-in-place storage + no audit tra; design-data: Nightly pg_dump with unrehearsed restore cannot honor 'no ac |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: No sync audit trail makes the stated support goal unachievab; design-data: No audit trail makes the stated support goal unanswerable an |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-data: Account deletion orphans note rows (personal data) in place ; design-data: Tombstones retained forever with no growth bound or GC |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-failure: Sync cursor loses changes: per-account cursor, no tie-breake; design-data: One cursor per user account cannot represent per-device sync |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: notes.db written cleartext with default permissions; stolen  |
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
