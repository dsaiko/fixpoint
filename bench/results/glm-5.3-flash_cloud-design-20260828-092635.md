# glm-5.3-flash:cloud · design · run 20260828-092635 (repeat 1)

recall **14/40** · 23 finding(s), 0 unmatched · 251380 tokens · 162s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-failure: A single invalid row livelocks sync forever, and partial app |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Wall-clock cursor silently drops edits under clock skew and ; design-data: Change feed keyed on client wall-clock modified_at silently  |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Any refresh 401 wipes the device, destroying unsynced offlin; design-data: Refresh-401 wipe destroys undrained outbox edits — unacknowl |
| D05 | — | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro |  |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Two processes share one notes.db with no stated serializatio; design-data: Two processes share one notes.db, one outbox and one cursor  |
| D07 | — | healthz checks process liveness only; an instance with a dead DB conne |  |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys do share the schema; the migration claim is ; design-data: Rolling deploys contradict the claim that old and new versio |
| D09 | — | §5 removes the server-side request timeout, so a slow or hostile clien |  |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Identical backoff with no jitter and no rate limit synchroni |
| D11 | YES | retries repeat until acknowledged with no attempt limit: a batch the s | design-data: A single invalid outbox row wedges the device's sync forever |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | — | the cursor is a millisecond timestamp compared strictly-greater: chang |  |
| D14 | YES | a client that can never sync simply goes quiet; the design names that  | design-failure: No signal distinguishes a broken client fleet from a quiet o |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: Postgres is the unnamed saturating component: unpaginated, u; design-data: No index beyond the primary key makes every sync a full-tabl |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-failure: No protocol version, with a mobile fleet that will skew old; design-data: Versionless wire format assumes all evolution is additive, w |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: 24-hour backup RPO and an unrehearsed restore contradict the; design-data: Nightly backups give a 24-hour RPO that contradicts the 'no  |
| D21 | — | the 30-second sync goal is unreachable with a 15-minute foreground tim |  |
| D22 | — | the reconnect spike after a regional outage is handled by 'adding inst |  |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-data: Stated support goal is unmeetable: no sync/device record exi |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: 32-bit random note id can collide and silently merge two not; design-data: 32-bit random note ids will collide, and collisions are indi |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | — | nothing caps a note body, so one pasted file becomes a row and a reque |  |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | — | tombstones are kept forever in both databases with no GC horizon; the  |  |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-data: Delete and account-deletion lifecycles keep bodies forever,  |
| D32 | — | one cursor per account with several devices: whichever device syncs fi |  |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | — | notes.db is written in the clear with default permissions, while §4 tr |  |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
