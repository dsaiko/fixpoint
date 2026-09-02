# nemotron-3-ultra:cloud · design · run 20260902-084724 (repeat 1)

recall **16/34** · 29 finding(s), 2 unmatched · 289792 tokens · 1109s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Partial batch application with client retry duplicates edits |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Clock skew in conflict resolution causes silent data loss; design-data: Clock skew causes silent data loss in conflict resolution |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded note body length enables memory exhaustion and DoS; design-data: Unbounded outbox growth can fill device storage |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: SQLite file shared by multiple processes without locking pro |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check does not verify database connectivity |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server request timeout with unbounded outbox enables conn; design-failure: No per-user rate limit enables abuse and noisy neighbors |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Fixed backoff schedule creates thundering herd after outage |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-failure: No pagination on changes query returns unbounded result sets |
| D16 | — | the changes query filters on (user_id, modified_at) with no index plan |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Protocol has no version field |
| D19 | YES | in-flight syncs are killed on every rolling deploy and the design lean | design-failure: No in-flight request draining on deploy causes avoidable 5xx |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Database restore procedure has never been rehearsed; design-data: Backup restore procedure untested |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: No audit trail of sync activity hinders debugging and securi; design-data: No audit trail contradicts operational goal |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | YES | the outbox stores a full copy of each body beside the note table, with | design-data: Outbox duplicates note bodies with no consistency check |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | YES | the server keeps only the latest body, so nothing can reconstruct a no | design-data: Server overwrites note bodies, losing all history |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-data: Deleted notes retained forever with no cleanup |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: Single cursor per user breaks multi-device sync |
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
- (high) DESIGN.md:83 — Unpaginated changes query risks OOM on first sync
- (medium) DESIGN.md:147 — No per-user rate limit allows DoS
