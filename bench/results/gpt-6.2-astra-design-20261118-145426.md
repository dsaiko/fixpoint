# gpt-6.2-astra · design · run 20261118-145426 (repeat 1)

recall **14/34** · 37 finding(s), 6 unmatched · 69367 tokens · 228s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Concurrent offline edits permanently discard one user's writ; design-failure: Restoring the nightly dump silently loses acknowledged chang |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Acknowledging one batch can erase edits made while it was in; design-failure: A validation failure leaves the client without a safe batch  |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-data: Full-body snapshots can exhaust local storage while a device |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: A stalled database operation can occupy sync capacity indefi; design-failure: An instance isolated from Postgres stays in load-balancer ro |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deployments expose old instances to the migrated sch; design-data: Pre-deploy migrations assume old instances have already stop |
| D09 | — | §5 removes the server-side request timeout, so a slow or hostile clien |  |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: The timestamp cursor permanently skips changes delivered out; design-failure: Endpoint metrics cannot explain why a particular device is m |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: Permanent tombstones retain full bodies and inflate every fi |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-data: Every incremental pull scans the entire multi-user notes tab |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | — | nightly pg_dump means up to 24h data loss, and the restore procedure h |  |
| D21 | YES | the 30-second sync goal is unreachable with a 15-minute foreground tim | design-failure: Reconnection does not trigger the promised 30-second sync |
| D22 | YES | the reconnect spike after a regional outage is handled by 'adding inst | design-failure: Identical retry schedules recreate the regional outage spike |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: The dashboards cannot explain a missing note on a specific d |
| D24 | YES | the conflict loser is discarded with no copy kept, so a wrong resoluti | design-data: Concurrent offline edits permanently discard one user's writ |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | YES | nothing caps a note body, so one pasted file becomes a row and a reque | design-data: Permanent tombstones retain the entire deleted note |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-failure: Even an empty incremental sync scans the shared notes table |
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
- (high) DESIGN.md:54 — An account-wide cursor lets one device acknowledge another device's download
- (high) DESIGN.md:34 — Repeated offline edits can exhaust local storage before reconnection
- (high) DESIGN.md:147 — One account can consume sync capacity needed by other accounts
- (critical) DESIGN.md:99 — A validation response acknowledges edits the server never applied
- (high) DESIGN.md:54 — An account-wide cursor treats one device's download as progress for every device
- (high) DESIGN.md:31 — Saving an offline edit has no atomic boundary with enqueueing it
