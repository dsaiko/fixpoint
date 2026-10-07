# claude-haiku-5-5 · design · run 20261007-203158 (repeat 1)

recall **21/34** · 35 finding(s), 0 unmatched · 48482 tokens · 242s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Atomicity of /sync is contradictory: one transaction versus ; design-data: Retried batches are not idempotent, and a retry can overlap  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Cursor is a client-supplied timestamp; one skewed clock wedg; design-data: Conflict rule silently discards a losing offline edit, break |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: A row that fails validation stalls the outbox forever; design-failure: Payload size is unbounded in both directions and an oversize |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Clearing the outbox races with a second process writing the ; design-failure: Changes query can miss commits that land after a higher-time |
| D07 | — | healthz checks process liveness only; an instance with a dead DB conne |  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys contradict the claim that old and new versio; design-data: Rolling deploys contradict the no-concurrent-versions rule |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No named saturation point, no timeout, no concurrency bound, |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-data: Failing row is silently acknowledged and its edit dropped |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-data: Outbox is unbounded and stores a full body per edit, so one  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query has no index on user_id, so every sync scans t |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Protocol has no version, so field additions are the only evo |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Restore from nightly dump loses acknowledged notes permanent; design-data: Restore is unrehearsed and loses up to a day of writes while |
| D21 | YES | the 30-second sync goal is unreachable with a 15-minute foreground tim | design-failure: The 30-second sync goal is unreachable under the stated back |
| D22 | YES | the reconnect spike after a regional outage is handled by 'adding inst | design-failure: Identical backoff schedule for every client produces a synch |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: Support cannot answer 'why does this device not have my note |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-data: Tombstones are kept forever in both databases with no compac |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: Cursor is a client-clock value stored per account, so one de |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: Remote-revocation wipe only happens when the device is onlin; design-data: notes.db is plaintext, and the revocation wipe only runs whe |
| D35 | YES | a reserved shared_with column is not a sharing design; the migration i | design-data: shared_with reservation does not cover sharing, because the  |
| D36 | YES | a delete on one device racing an edit on another is resolved by timest | design-data: Delete versus offline edit has no defined winner, so deletio |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | YES | the server holds no per-device state at all, so it cannot tell two dev | design-failure: Operator cannot tell a stuck device, a DB outage, or a bad d |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-data: Client clears the whole outbox after the response, deleting  |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
