# nemotron-3-ultra:cloud · design · run 20260820-132500 (repeat 1)

recall **22/40** · 34 finding(s), 1 unmatched · 374951 tokens · 670s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Partial batch apply with no idempotency keys causes duplicat; design-data: Partial batch application with unclear retry semantics |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Last-writer-wins on wall-clock modified_at silently loses da; design-failure: No alerting on clients that permanently fail to sync |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Auth 401 wipes notes.db including unsynced work |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded note body and outbox sizes enable DoS/OOM; design-data: Unbounded outbox growth with full-body duplication |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: No audit trail contradicts debuggability goal; design-data: Auth failure wipes local notes including offline edits |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health checks don't verify DB connectivity; design-data: Health check does not verify database connectivity |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Migrations block deploy with no backward-compatibility strat |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server request timeout + health checks that don't verify ; design-data: No server request timeout ties up connections indefinitely |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | YES | no per-user rate limit and no backpressure; one looping client can con | design-data: No per-user rate limit enables DoS by misbehaving client |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Wall-clock cursor breaks on NTP corrections |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: Single-request sync with no pagination hits HTTP/memory limi; design-data: No pagination on changes query risks OOM on first sync |
| D16 | — | the changes query filters on (user_id, modified_at) with no index plan |  |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Protocol has no version number; only additive changes are sa |
| D19 | YES | in-flight syncs are killed on every rolling deploy and the design lean | design-failure: No graceful shutdown draining in-flight requests |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Backup restore never rehearsed; design-data: Backup restore untested; no RTO/RPO defined |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: No audit trail makes 'why does this device not have my note' |
| D24 | YES | the conflict loser is discarded with no copy kept, so a wrong resoluti | design-data: Conflict resolution discards losing copy with no recovery |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: 32-bit random note IDs will collide at target scale; design-data: 32-bit random note IDs will collide |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | YES | the server keeps only the latest body, so nothing can reconstruct a no | design-data: Server overwrites note bodies in place; no history or undo |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Deleted notes retained forever with no cleanup plan |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-data: Account deletion leaves orphaned note rows with no cleanup p |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: Single cursor per user breaks multi-device sync |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-data: Device-level encryption assumption lacks verification |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) DESIGN.md:1 — No per-user rate limit allows single-user DoS
