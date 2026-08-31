# deepseek-v4-pro:cloud · design · run 20260820-124527 (repeat 1)

recall **19/40** · 28 finding(s), 0 unmatched · 194701 tokens · 217s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Transaction semantics are self-contradictory, and a permanen; design-data: Single transaction contradicts 'rows 1..N-1 stay applied' on |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Client wall clock is the cursor and conflict arbiter; skew s; design-failure: An edit is two writes (note row + outbox row) with no stated |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Any refresh 401 wipes notes.db, destroying unsynced offline ; design-data: 401 wipes notes.db including the outbox, destroying unacknow |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-data: Unbounded body + full-copy outbox + single-request sync mean |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores the database, so the load balancer keeps ro |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: No protocol version, and 'old and new never run concurrently |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: Unbounded note body, outbox, and single-POST batch with no s |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Identical backoff schedule on every client synchronizes the  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | YES | no per-user rate limit and no backpressure; one looping client can con | design-failure: No per-user rate limit and no audit trail: a misbehaving cli |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-data: Cursor is a single modified_at with strict '>' — same-millis |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-data: deleted rows kept forever with no pagination makes first syn |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: No index on user_id: every sync is a full table scan; design-data: No index on (user_id, modified_at) forces a full table scan  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Protocol carries no version number; 'adding a field is backw |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Backups are never restored; RPO/RTO are unstated and the res; design-data: Backups are nightly pg_dump but restore is unrehearsed |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: 'A client that stops syncing is not alertable' contradicts t; design-data: No audit trail of syncs makes goal 5 ('answer from dashboard |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: 32-bit random note id can collide across a user's devices; design-data: 32-bit random note id will collide and silently merge notes |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | YES | the server keeps only the latest body, so nothing can reconstruct a no | design-data: Conflict resolution discards the losing copy, violating 'no  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-failure: Account deletion leaves the user's notes in clear text forev; design-data: Account deletion leaves note rows orphaned, with no owner an |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | YES | a reserved shared_with column is not a sharing design; the migration i | design-data: Reserved shared_with field gives false confidence; sharing b |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
