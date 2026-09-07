# gpt-5.6-terra · design · run 20260907-142958 (repeat 1)

recall **9/34** · 17 finding(s), 2 unmatched · 81995 tokens · 172s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | — | conflict resolution by client wall clock silently discards edits, cont |  |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Partial batch failure has no recoverable acknowledgment prot; design-data: Two independent processes can erase each other's outbox entr |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | — | healthz checks process liveness only; an instance with a dead DB conne |  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deployments lack compatibility and rollback guarante |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: Unbounded sync requests can exhaust service resources |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Regional recovery creates a synchronized retry storm with no |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Timestamp-only cursors permanently skip same-millisecond cha |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: Silent client sync failure is invisible to operators |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Every sync requires an unindexed full-history changes query |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: A database loss can discard up to a day of acknowledged edit |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: The design cannot explain device-specific sync gaps |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
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
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) DESIGN.md:52 — Wall-clock timestamps cannot safely serve as sync cursors
- (high) DESIGN.md:57 — A single invalid edit blocks all later offline edits indefinitely
