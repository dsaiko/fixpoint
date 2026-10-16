# muse-spark-open:cloud · design · run 20261016-020051 (repeat 1)

recall **13/34** · 18 finding(s), 0 unmatched · 91892 tokens · 90s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial-commit on validation failure contradicts transaction; design-data: Atomic transaction claim contradicts partial-commit + non-id |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Cursor on client wall-clock loses data and diverges per devi; design-data: Cursor and conflict resolution both trust client wall clock |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: No input or cost caps — unbounded body, outbox, and request  |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Two processes sharing one SQLite file with no concurrency or; design-data: Two local processes open same SQLite file with no concurrenc |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz divorced from DB makes outage invisible to LB and o |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploy assumes no concurrent versions; migration-bef |
| D09 | — | §5 removes the server-side request timeout, so a slow or hostile clien |  |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Identical fixed backoff with no jitter guarantees thundering |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-data: Unbounded bulk sync with no pagination, index, or limit brea |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Unbounded SELECT * with no index, pagination, or timeout wil |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Recovery untested with 24h RPO contradicts durability promis |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: Last-version-only storage with silent discard makes 'never l |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: One cursor per user cannot serve multiple devices |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | YES | a reserved shared_with column is not a sharing design; the migration i | design-data: No protocol versioning or migration story despite rolling de |
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
| D31 | — | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
