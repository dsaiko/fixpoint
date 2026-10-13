# codex@google/gemini-4-argon · design · run 20261013-221900 (repeat 1)

recall **14/34** · 26 finding(s), 0 unmatched · 204610 tokens · 263s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Single-transaction partial-commit contradiction and missing ; design-data: Partial-batch commits on validation error contradict single- |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-write-wins on client wall clock with in-place overwrite; design-data: Last-write-wins with in-place overwrites and client clocks p |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Uncapped outbox storing full note bodies per edit and single; design-failure: Absence of per-user rate limits and payload caps allows a si |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Uncoordinated access to notes.db by desktop and CLI and blan; design-data: Uncoordinated multi-process access to notes.db and blanket o |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Shallow /healthz check and lack of per-device sync telemetry |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys with pre-deploy schema migrations and undrai; design-data: Unversioned protocol, non-concurrent migration assumption, a |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server request timeout combined with client timeout retri |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Missing secondary indexes and unpaginated sync queries cause; design-data: Unindexed changes query and uncompacted full-body outbox sna |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Nightly pg_dump without WAL archiving loses up to 24 hours o |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | YES | the reconnect spike after a regional outage is handled by 'adding inst | design-failure: Deterministic retry schedule and simultaneous post-outage re |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: Omitting sync audit history and per-device state makes Goal  |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Retaining deleted note bodies and deleted accounts' notes fo |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-failure: Client-clock sync cursor and single per-account cursor perma; design-data: Using client wall-clock modified_at as the sync cursor silen |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | YES | attachments are a URL in the note row with no lifecycle: nothing delet | design-data: Implicit deletion and attachment lifecycles retain deleted n |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  |
| D31 | — | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
