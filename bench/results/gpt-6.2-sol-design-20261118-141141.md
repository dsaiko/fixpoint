# gpt-6.2-sol · design · run 20261118-141141 (repeat 1)

recall **16/34** · 31 finding(s), 3 unmatched · 91114 tokens · 366s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflict resolution permanently discards offline edits; design-data: Conflict resolution permanently discards offline edits |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Clearing the outbox can drop edits made during sync; design-data: Partial batch commits have no matching acknowledgment |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: One account can consume shared sync capacity |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: A worker can pass health checks while unable to serve sync |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys overlap old workers with schema migrations; design-data: Rolling deploys expose active instances to migrated schemas |
| D09 | — | §5 removes the server-side request timeout, so a slow or hostile clien |  |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Wall-clock cursors permanently skip valid changes; design-data: Client timestamps cause sync to skip committed changes |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: A large backlog can become impossible to sync |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Incremental polls scan the shared notes table |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-data: Backup restore loses acknowledged edits made since the last  |
| D21 | YES | the 30-second sync goal is unreachable with a 15-minute foreground tim | design-failure: An open client can miss the reconnect sync target |
| D22 | YES | the reconnect spike after a regional outage is handled by 'adding inst | design-failure: Fixed retry timing sustains recovery traffic spikes |
| D23 | — | §1 promises support can answer 'why is my note missing' from dashboard |  |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Deleted notes make first sync grow with account history |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: An account-wide cursor conflates independent replicas |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: Remote revocation cannot protect the plaintext cache while o |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | YES | the server holds no per-device state at all, so it cannot tell two dev | design-failure: Endpoint metrics cannot diagnose a missing note on a device; design-data: Sync dashboards cannot explain a missing note on a device |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-data: Clearing the outbox can erase edits made during sync |

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
- (critical) DESIGN.md:133 — Restoring a nightly dump can lose acknowledged edits
- (high) DESIGN.md:80 — Partial batch commits have no safe acknowledgment protocol
- (high) DESIGN.md:32 — Offline autosaves can exhaust local storage
