# minimax-m3:cloud · design · run 20260820-122757 (repeat 1)

recall **17/40** · 23 finding(s), 0 unmatched · 194793 tokens · 322s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-data: Retried POSTs are not idempotent — duplicate applies |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: Partial-batch handling drops rows after the failing row with |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Cursor is the client wall clock; a forward-skewed client sil; design-failure: No protocol version field means semantic changes to existing |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: 401 wipes local state, conflating revocation with any auth h; design-data: Refresh-token 401 wipes the local database — token corruptio |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-data: Outbox stores a full body copy per edit — unbounded growth a |
| D06 | — | desktop client and CLI open the same SQLite file with no locking story |  |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz does not gate on the database, hiding full outages |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys and 'old and new versions never run concurre |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No request timeout, no body cap, no rate limit — one user ca |
| D10 | — | every client shares one backoff schedule with no jitter; after an outa |  |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: Silent client-side failures are not alertable signals |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Changes query has no supporting index at the planned 50k-use; design-data: Changes query has no index, no pagination, no request timeou |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Wire format has no version field and no migration story |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: Restore has not been rehearsed; nightly pg_dump + 14-day ret |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | — | §1 promises support can answer 'why is my note missing' from dashboard |  |
| D24 | YES | the conflict loser is discarded with no copy kept, so a wrong resoluti | design-data: Last-writer-wins with discard loses data on tombstone races |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: 32-bit client-side note IDs collide across offline creators; design-data: 32-bit random note IDs will collide in production |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | YES | the server keeps only the latest body, so nothing can reconstruct a no | design-failure: Server keeps only the latest version; conflict rule leaves n |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-failure: Account deletion leaves the user's note rows in place until ; design-data: Account deletion leaves orphan notes and has no defined owne |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | YES | the server holds no per-device state at all, so it cannot tell two dev | design-data: Stated 'support engineer can answer from dashboards' goal is |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
