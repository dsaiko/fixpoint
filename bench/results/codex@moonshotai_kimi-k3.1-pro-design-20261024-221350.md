# codex@moonshotai/kimi-k3.1-pro · design · run 20261024-221350 (repeat 1)

recall **15/34** · 27 finding(s), 2 unmatched · 226028 tokens · 309s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Retries reapply batches without an idempotency mechanism; design-data: Retried sync requests have no idempotency mechanism |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Client wall clocks can silently discard valid edits |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded single-request batches cannot reliably recover fro; design-data: Partial batch commits cannot be reconciled by the client pro |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Liveness-only health checks keep broken instances in rotatio |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deployments are incompatible with the migration assu; design-data: Schema migration assumptions conflict with rolling deploymen |
| D09 | — | §5 removes the server-side request timeout, so a slow or hostile clien |  |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Synchronized retry and polling schedules amplify outage reco |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Timestamp-only cursors skip changes with equal timestamps; design-data: Wall-clock cursors permanently skip valid changes |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: Silent client sync failures are invisible to operations |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Every sync performs an unbounded, unindexed changes query |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Field addition is not a substitute for protocol versioning |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-data: Backup and recovery cannot preserve acknowledged edits |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: Latest-only storage cannot support the required diagnosis |
| D24 | YES | the conflict loser is discarded with no copy kept, so a wrong resoluti | design-data: Client clocks decide which user edit is destroyed |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-failure: A single account-wide cursor breaks multi-device synchroniza; design-data: Synchronization progress is scoped to an account instead of  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | YES | a delete on one device racing an edit on another is resolved by timest | design-data: Outbox acknowledgement can delete edits that were never sent |
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
- (high) DESIGN.md:78 — A validation error can permanently stall the outbox
- (critical) DESIGN.md:133 — Nightly backups cannot satisfy the acknowledged-edit durability guarantee
