# gpt-6-astra-xhigh · design · run 20260905-142924 (repeat 1)

recall **14/34** · 25 finding(s), 1 unmatched · 133076 tokens · 412s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-data: Client timestamps cannot serve as reliable change cursors |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Partial batch commits have no safe client recovery transitio; design-failure: Unbounded synchronization payloads can prevent recovery inde |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Authentication failures erase unsynchronized notes; design-failure: Clearing the entire outbox loses edits made during sync |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: An instance unable to reach Postgres remains in rotation |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deployment exposes old binaries to the migrated sche |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: Timed-out clients leave unbounded server work behind |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Backup restoration cannot reconcile already-acknowledged cli; design-failure: The retained telemetry cannot explain missing notes |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Client timestamps cannot identify all newly committed change; design-failure: Every incremental sync scans the global notes table |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-data: Backup recovery loses acknowledged writes without reconcilia |
| D21 | YES | the 30-second sync goal is unreachable with a 15-minute foreground tim | design-failure: The scheduling policy cannot meet the reconnect latency goal |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: Discarded sync history makes the support requirement impossi |
| D24 | YES | the conflict loser is discarded with no copy kept, so a wrong resoluti | design-failure: Ordinary offline conflicts permanently destroy user edits; design-data: Conflict resolution permanently discards offline writing |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | YES | attachments are a URL in the note row with no lifecycle: nothing delet | design-data: Account deletion leaves note data without a deletion lifecyc |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-data: Clearing the outbox deletes edits created during sync |

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
- (high) DESIGN.md:147 — Unrestricted sync traffic can monopolize the shared database
