# deepseek-v4-pro:cloud · design · run 20260901-112551 (repeat 1)

recall **15/34** · 25 finding(s), 0 unmatched · 246019 tokens · 143s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-data: Retry-until-acknowledged is at-least-once delivery, but crea |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Transactionality is internally inconsistent, leaving partial; design-data: Single-transaction apply and partial application are asserte |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Wall-clock cursor plus LWW makes clock skew silently lose da; design-data: Last-write-wins on client wall clock plus a wall-clock curso |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-data: Unbounded body, unbounded outbox, full-copy-per-edit, and a  |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz does not gate on the database, so the LB keeps rout |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys run old and new concurrently, but the protoc; design-data: Rolling deploys run old and new versions concurrently, contr |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No request timeout, no body cap, and no rate limit leave res |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: A silently failing client is invisible, contradicting the da |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query has no supporting index, so every sync is a fu; design-data: The changes query filters on user_id and modified_at but no  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Protocol carries no version number on the assumption all fut |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Restore is unrehearsed with no RPO/RTO, so recovery is unpro; design-data: Nightly pg_dump with an unrehearsed restore leaves up to 24h |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: No audit trail of syncs makes the stated support goal unansw |
| D24 | YES | the conflict loser is discarded with no copy kept, so a wrong resoluti | design-data: 'No acknowledged edit is ever lost' is contradicted by disca |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Tombstones and orphaned rows grow without bound |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: Clear-text notes.db contradicts the 'stolen device cannot re |
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
