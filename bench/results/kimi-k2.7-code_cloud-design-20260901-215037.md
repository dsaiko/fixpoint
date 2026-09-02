# kimi-k2.7-code:cloud · design · run 20260901-215037 (repeat 1)

recall **15/34** · 22 finding(s), 0 unmatched · 104599 tokens · 194s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Single-transaction sync contradicts partial-row failure sema; design-data: Partial success acknowledges edits it did not apply |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflict resolution relies on client wall clocks; design-data: Client wall-clock timestamps can permanently hide updates |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: No cap on note or outbox size; design-data: Outbox grows unbounded with full-body copies |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Multiple processes write the same SQLite database directly; design-data: Desktop and CLI concurrent access to notes.db is unmodeled |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz does not reflect database availability |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: Sync sends the whole outbox in a single unbounded request |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: All clients share the same retry schedule with no jitter |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | YES | no per-user rate limit and no backpressure; one looping client can con | design-failure: No rate limits; abuse is handled by support tickets |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-data: Changes query lacks an index and pagination |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query is unbounded and unindexed |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Disaster recovery is documented but untested |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: Support cannot diagnose missing-note questions |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Deleted notes and account orphans are never cleaned up |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: Local note database relies on platform encryption and defaul |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | YES | the server holds no per-device state at all, so it cannot tell two dev | design-data: Stated support goal cannot be met from dashboards |
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
