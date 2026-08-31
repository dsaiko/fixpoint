# kimi-k3:cloud · design · run 20260820-123750 (repeat 1)

recall **15/40** · 24 finding(s), 0 unmatched · 124372 tokens · 189s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: 'Single transaction' contradicts partial row application, an; design-data: Client retries a partially-rejected batch forever; single-tr |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: A skewed client clock permanently poisons the sync cursor an; design-data: Sync cursor is derived from client-controlled wall-clock mod |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: A refresh-token 401 wipes notes.db, destroying unacknowledge; design-data: Any 401 — including token corruption — wipes notes.db along  |
| D05 | — | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro |  |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores dependency health, so wedged instances stay |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys do run old and new code concurrently; 'deplo |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No request timeout + uncapped outbox batch + uncapped body + |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Synchronized retry schedule and 15-minute timer guarantee a  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query has no index beyond the primary key — every sy; design-data: Changes query runs per sync with no index beyond the primary |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: No protocol version; only additive change is assumed compati |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Nightly pg_dump gives ~24h RPO for 'never lost' data, and re; design-data: Restore procedure unrehearsed; no point-in-time recovery; RP |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: The central failure class — a client that never syncs — has  |
| D24 | YES | the conflict loser is discarded with no copy kept, so a wrong resoluti | design-failure: Losing conflict copy is discarded immediately with no record |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: 32-bit random note ids collide within a user's lifetime and ; design-data: Note ids are random 32-bit integers generated per device; co |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-failure: Account deletion leaves the user's notes in place indefinite; design-data: Account deletion orphans the user's note rows; deleted-note  |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-failure: One sync cursor per user account cannot serve multiple devic; design-data: One sync cursor is stored per user account, shared by all de |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
