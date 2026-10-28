# ollama-codex@glm-5.3-flash:cloud#max · design · run 20261028-122846 (repeat 1)

recall **15/34** · 24 finding(s), 0 unmatched · 205294 tokens · 270s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-data: Partial batch semantics make synchronization acknowledgment  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial batch success has no reliable client acknowledgment  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Client wall clock is used as a change cursor and conflict au; design-data: Client wall-clock timestamps drive both synchronization and  |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-data: Unbounded outbox and unlimited single request can permanentl |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-data: Concurrent direct writers to one SQLite database are owned b |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Load balancer health cannot detect dependency failure |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling migration and rollback behavior is internally incons |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: Unbounded outbox turns offline growth into an unbounded serv; design-failure: No rate limiting or backpressure for client-generated sync l |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Synchronized retry schedule amplifies recovery load |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: Broken client sync state is invisible until a support ticket |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: Unindexed and unpaged change fan-out will not meet the capac |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-data: Change query lacks the index and bounds needed at stated sca |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Protocol and schema evolution are declared compatible withou |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Recovery has not been validated; design-data: Nightly pg_dump does not meet the no-acknowledged-edit-durab |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: No sync audit trail prevents the promised support diagnosis |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
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
