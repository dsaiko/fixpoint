# claude-opus-5 · design · run 20260820-125505 (repeat 1)

recall **19/40** · 31 finding(s), 0 unmatched · 31766 tokens · 403s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-data: Edits carry no identity, so retried batches replay stale bod |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: §5 contradicts itself on batch atomicity; under partial appl; design-data: Section 5 specifies both a single transaction and partial ap |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Cursor is a client wall clock and is stored once per account; design-failure: Last-writer-wins on an unvalidated client clock discards the |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: A 401 on token refresh wipes the undrained outbox, destroyin; design-data: Sign-out on 401 wipes notes.db including the unsynced outbox |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-data: An uncapped outbox holding a full body copy per edit can exh |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Desktop client and CLI open the same SQLite file with no sta; design-data: Two processes share notes.db with no writer ownership define |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: /healthz ignores database connectivity, so an instance that  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys run two versions at once, which the migratio; design-data: Rolling deploys do run old and new code against one schema,  |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: Unbounded outbox in a single POST, uncapped bodies, and no s |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: One shared un-jittered backoff schedule plus no rate limit r |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | YES | the changes query returns every matching row unpaginated; a first sync | design-data: The changes query has no supporting index and no pagination, |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Unindexed, unpaginated changes query makes Postgres the comp |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: Omitting a protocol version leaves no way to change the wire |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: 24-hour RPO on an unrehearsed restore, and the cursor design; design-data: After a restore, client cursors sit ahead of restored data,  |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: The design states outright that a permanently failing client; design-data: No sync or device record makes the stated support goal unach |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: Random 32-bit note ids collide at the stated scale, silently; design-data: Random 32-bit client note ids collide at the stated scale, m |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-failure: Account deletion leaves every note row in place indefinitely; design-data: Account deletion orphans note rows with no cascade and no er |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: notes.db written in the clear with default permissions is re |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | YES | step 4 clears the whole outbox, discarding edits the user made while t | design-failure: Step 4 clears the whole outbox, discarding edits made while ; design-data: Clearing the whole outbox on acknowledgment deletes edits ma |
