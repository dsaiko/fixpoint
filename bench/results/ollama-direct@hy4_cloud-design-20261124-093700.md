# ollama-direct@hy4:cloud · design · run 20261124-093700 (repeat 1)

recall **20/34** · 36 finding(s), 0 unmatched · 99929 tokens · 698s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Retry after a lost response is not idempotent |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial validation success has no safe client state transiti; design-failure: Client-side sync completion is not an atomic checkpoint |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Client wall-clock timestamps are the only cursor and conflic; design-data: Client wall-clock modified_at is used as a strict sync high- |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: An unbounded single-batch sync can become permanently undrai; design-data: Unbounded full-body batches and unpaginated scans cannot mee |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Nightly dumps lose up to a day of acknowledged edits; design-data: Client applies changes, stores cursor, and clears outbox wit |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Readiness checks cannot remove instances with broken depende |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling upgrades have no mixed-version or rollback compatibi; design-data: Rolling deploys assume schema versions never coexist |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: The sync boundary has no cost or concurrency cap; design-data: Retries repeat non-idempotent full batches after successful  |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: The change feed performs unbounded full scans and responses |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: The JSON protocol's claim of backward compatibility is unsup |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Restore has no rehearsed procedure or trust verification; design-data: Nightly dumps cannot guarantee an acknowledged edit is never |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | YES | the reconnect spike after a regional outage is handled by 'adding inst | design-failure: Identical client backoff creates a synchronized retry storm |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: Silent sync failures are invisible to operators and support |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | YES | the outbox stores a full copy of each body beside the note table, with | design-data: Note table and outbox duplicate body with no reconciliation |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-data: Tombstone retention has no convergence signal or body-stripp |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: Dashboards cannot answer why a device is missing a note |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: Cleartext local storage defeats offline revocation |
| D35 | YES | a reserved shared_with column is not a sharing design; the migration i | design-data: A single shared_with note column cannot model sharing state |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | YES | attachments are a URL in the note row with no lifecycle: nothing delet | design-data: Attachment URLs have no ownership, state, or deletion lifecy |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-failure: Clearing the entire outbox deletes edits made during an in-f |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
