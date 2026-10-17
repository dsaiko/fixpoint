# deepseek/deepseek-v4.1-flash-1017 · design · run 20261017-211325 (repeat 1)

recall **20/34** · 37 finding(s), 0 unmatched · 188002 tokens · 1081s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-data: A rejected row is acknowledged and then cleared from the out |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-write-wins on client wall clocks permanently destroys t; design-data: Last-writer-wins trusts an untrusted client clock and discar |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Each offline edit keeps a full body copy in an uncapped outb; design-data: Unbounded body, outbox and batch produce requests that can n |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: A permanently failing outbox row blocks that device from eve; design-failure: Desktop and CLI open the same SQLite file with no writer coo |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check cannot distinguish a broken instance from a qui |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: A rolling deploy does run old and new versions concurrently,; design-data: Rolling deploys run old and new code against one schema, wit |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server request timeout lets abandoned and hostile syncs p; design-failure: No per-user or per-device rate limit lets one client degrade |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Identical unjittered backoff and 15-minute timers create a s; design-data: The 30-second sync goal contradicts the 60-second backoff ca |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Cursor is a raw client timestamp, so ties and skewed clocks ; design-data: A client-clock cursor delivered by a strictly-greater query  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: Upload and download sizes are unbounded in a single request  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: The per-sync change query has no supporting index; design-data: The changes query has no usable index and no pagination |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-failure: No protocol version, so old long-lived clients cannot be neg |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Backups are unrehearsed and leave a 24-hour acknowledged-dat; design-data: A backup restore silently diverges every client with no reco |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: Silent client non-convergence and the absence of a sync audi; design-data: No audit trail or device identity, so the stated support goa |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | YES | the outbox stores a full copy of each body beside the note table, with | design-data: Note write and outbox append are a dual write with no stated |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Deletion tombstones are never collected and grow every table |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: One cursor per user account cannot serve multiple devices |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: Notes are written cleartext with default file permissions |
| D35 | YES | a reserved shared_with column is not a sharing design; the migration i | design-data: The reserved shared_with field cannot express sharing and th |
| D36 | YES | a delete on one device racing an edit on another is resolved by timest | design-data: Deletion tombstones never receive a new version, so deletion |
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
