# qwen/qwen3.8-27b · design · run 20260820-132657 (repeat 1)

recall **18/40** · 31 finding(s), 0 unmatched · 82639 tokens · 1108s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial application on validation failure contradicts the si; design-data: Partial-batch semantics contradict the single-transaction cl |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflict winner is the client wall clock; a skewed clock sil; design-failure: A future-skewed edit poisons the user's cursor; older-timest |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Any 401 wipes notes.db including unsynced offline edits; a r; design-data: 401 on refresh wipes notes.db — including the outbox — on a  |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Uncapped outbox sent whole in one request and one transactio; design-failure: No body cap, no rate limit, no request-size limit: one user  |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Two client processes share one SQLite file with no defined c; design-data: Two named writers on one SQLite file with no concurrency pol |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: No request timeout plus a liveness-only health check: stuck  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys by definition run old and new concurrently, ; design-data: Migrations run before a rolling deploy and in-flight request |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-data: Entire unbounded outbox in one POST, one server transaction, |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Every client uses the identical jitter-free backoff, in the  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query scans the whole notes table per sync; the desi; design-data: Changes query is a sequential scan on a grow-only table at s |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-failure: No protocol version; the compatibility argument covers only ; design-data: No protocol version, and SELECT * makes the table schema the |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: The only backstop against the corruption this design guarant; design-data: A restore silently drops up to 24h of acknowledged edits: th |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: Stuck clients are declared non-alertable and the design has  |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: Random 32-bit note ids collide with certainty at stated scal; design-failure: Sync edits are never stated to be ownership-checked, and the |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | YES | the server keeps only the latest body, so nothing can reconstruct a no | design-data: No version of a note's previous body exists anywhere after a |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-failure: Account deletion leaves the user's notes in place with clean; design-data: Retention is incoherent: account-deletion orphans have no ow |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | YES | attachments are a URL in the note row with no lifecycle: nothing delet | design-data: Attachments: URL in the note row with no owner for the objec |
| D38 | YES | the server holds no per-device state at all, so it cannot tell two dev | design-data: One server-stored cursor per user account contradicts per-de |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
