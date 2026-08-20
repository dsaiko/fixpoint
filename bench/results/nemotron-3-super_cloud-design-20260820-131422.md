# nemotron-3-super:cloud · design · run 20260820-131422 (repeat 1)

recall **10/40** · 14 finding(s), 1 unmatched · 338912 tokens · 129s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Unclear transaction semantics for batch edits may lead to pa |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Clock skew can cause incorrect conflict resolution and silen; design-data: Unreliable conflict resolution via client clocks |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-data: Local data loss on auth failure |
| D05 | — | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro |  |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check does not detect database connectivity issues |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |
| D09 | — | §5 removes the server-side request timeout, so a slow or hostile clien |  |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: Server may return large result sets for users with many note |
| D16 | — | the changes query filters on (user_id, modified_at) with no index plan |  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Backup restore procedure has not been tested |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | — | §1 promises support can answer 'why is my note missing' from dashboard |  |
| D24 | YES | the conflict loser is discarded with no copy kept, so a wrong resoluti | design-data: Permanent data loss in conflicts |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: Risk of note ID collision leading to data loss; design-data: Client ID collision risk |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | YES | the outbox stores a full copy of each body beside the note table, with | design-failure: Sync request may be too large if outbox is big; design-failure: Risk of lost edits due to non-atomic update of note table an |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-failure: Account deletion leaves orphaned notes in the database |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) DESIGN.md:32 — Outbox can grow without bound during prolonged offline use
