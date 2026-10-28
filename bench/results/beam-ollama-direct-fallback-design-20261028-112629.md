# beam-ollama-direct-fallback · design · run 20261028-112629 (repeat 1)

recall **19/34** · 32 finding(s), 0 unmatched · 305942 tokens · 98s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-data: Outbox retries lack idempotency keys on edits |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial failure is neither atomic nor idempotent; design-data: Claimed single transaction yet rows 1..N-1 persist on failur |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflicts discard a possibly acknowledged copy immediately |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded outbox with no drain batching |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Two processes share one SQLite file with no coordination; design-data: Desktop and CLI contend on one SQLite file with no coordinat |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check ignores dependencies |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploy assumed to have no version overlap |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server-side request timeout |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Post-outage sync stampede has no mitigation |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | YES | no per-user rate limit and no backpressure; one looping client can con | design-failure: No per-user rate limit on sync |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Cursor uses strict inequality on clientModified timestamp; design-data: Strict inequality cursor plus millisecond timestamps drops e |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query has no index and no pagination |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Backups are nightly and restore is unrehearsed |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: A client that never syncs is not alertable; design-failure: No audit trail of syncs |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | YES | nothing caps a note body, so one pasted file becomes a row and a reque | design-failure: Unbounded note body on client and server |
| D29 | YES | the server keeps only the latest body, so nothing can reconstruct a no | design-failure: Server keeps only the latest note version; design-data: Losing conflict body is discarded, contradicting 'no acknowl |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Deleted rows are retained forever; design-data: Deleted rows retained forever with no pruning policy |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: Single cursor stored per user account breaks multi-device sy |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: notes.db stored in the clear with default permissions |
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
