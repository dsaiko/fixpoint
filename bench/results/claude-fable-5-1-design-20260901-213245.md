# claude-fable-5-1 · design · run 20260901-213245 (repeat 1)

recall **15/34** · 27 finding(s), 0 unmatched · 23583 tokens · 305s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial-apply semantics contradict the single transaction, a; design-data: Partial batch application contradicts the single transaction |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Cursor is a client wall-clock modified_at, so offline edits ; design-failure: Last-writer-wins on client wall clocks discards a user's wri |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Uncapped single-request batch with no server timeout: long-o |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-data: Two processes share notes.db and the outbox with no owner; c |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores database connectivity so a broken instance  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys do run both versions concurrently, migration; design-data: Rolling deploy with migrations run first means old code runs |
| D09 | — | §5 removes the server-side request timeout, so a slow or hostile clien |  |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Post-outage thundering herd: identical un-jittered backoff,  |
| D11 | YES | retries repeat until acknowledged with no attempt limit: a batch the s | design-failure: Deleted notes retain their full body forever and account del |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query has no index and no pagination at 50k users sy |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: No protocol or local schema version, though the design alrea |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: A backup restore rewinds the server but no client can detect; design-data: Restoring from backup leaves client cursors ahead of the ser |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: No signal distinguishes a device that cannot sync from a qui; design-data: No device identity or sync record exists, so the support goa |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | YES | the outbox stores a full copy of each body beside the note table, with | design-data: Uncapped outbox of full-body copies is sent as one POST; a b |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: Sync cursor is a client wall-clock timestamp, so changes are |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-failure: Outbox is cleared wholesale on ack, deleting edits appended  |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |
