# ollama-codex@beam:cloud · design · run 20261019-205426 (repeat 1)

recall **17/34** · 30 finding(s), 0 unmatched · 571386 tokens · 83s · 6 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-data: Row-by-row partial commit combined with whole-batch retries  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Row-by-row partial application plus whole-batch retries are  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflict resolution trusts client wall clock and permanently; design-data: Conflict resolution discards a losing edit, contradicting th |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded body size and unbounded outbox with full body copi |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Desktop and CLI share one SQLite file with no stated concurr; design-data: Desktop client and CLI open the same SQLite file without con |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores dependencies so broken-but-alive instances  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys assume no concurrent versions and no compati |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server request timeout lets one slow sync hold resources  |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: No rate limit and identical backoff make a single client a c |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: Cursor is a wall-clock timestamp with strict greater-than; c; design-data: Timestamp cursor with strict inequality drops tied or equal- |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query is an unindexed full scan that returns every r; design-data: Tombstones are never pruned and the changes query has no pag |
| D18 | — | the wire format carries no version, so any change that is not purely a |  |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Nightly unrehearsed backup leaves 24h RPO and no verified re |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: A device that cannot sync is invisible; Goal 5 is unachievab; design-data: No per-device sync audit trail makes the support goal unachi |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | YES | nothing caps a note body, so one pasted file becomes a row and a reque | design-data: Uncapped body and outbox have no bounded resource behavior |
| D29 | YES | the server keeps only the latest body, so nothing can reconstruct a no | design-data: Server keeps only the latest version, discarding history nee |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-data: A single cursor per user account breaks multi-device sync |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: notes.db stored in the clear; security delegated entirely to; design-data: notes.db is written in the clear relying solely on platform  |
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
