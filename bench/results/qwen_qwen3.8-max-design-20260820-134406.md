# qwen/qwen3.8-max · design · run 20260820-134406 (repeat 1)

recall **22/40** · 29 finding(s), 0 unmatched · 41917 tokens · 862s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | — | retrying the same POST after a timeout re-applies edits the server alr |  |
| D02 | YES | §5 claims one transaction per request AND that rows 1..N-1 survive a f | design-data: Partial-batch validation failure leaves the rejected edit an |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflict winner is chosen by client wall clocks and the lose; design-data: LWW on client wall clocks with immediate discard of the lose |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Any 401 on refresh wipes notes.db, destroying unsynced offli |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: The entire unbounded outbox ships as one request and one tra; design-data: Unbounded outbox of full-body copies is drained as one reque |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: A row that fails validation blocks the outbox forever; design-data: Two processes share notes.db with no named owner for its sch |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Liveness-only healthz keeps dead-end instances in rotation d |
| D08 | YES | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  | design-failure: Rolling deploys contradict the claim that old and new versio; design-data: Rolling deploys contradict the claim that old and new versio |
| D09 | YES | §5 removes the server-side request timeout, so a slow or hostile clien | design-failure: No server request timeout, on a service whose own mobile mod |
| D10 | YES | every client shares one backoff schedule with no jitter; after an outa | design-failure: Identical backoff schedule with no jitter turns the post-out |
| D11 | — | retries repeat until acknowledged with no attempt limit: a batch the s |  |
| D12 | — | no per-user rate limit and no backpressure; one looping client can con |  |
| D13 | YES | the cursor is a millisecond timestamp compared strictly-greater: chang | design-failure: A wall-clock cursor with strict-greater queries permanently  |
| D14 | — | a client that can never sync simply goes quiet; the design names that  |  |
| D15 | — | the changes query returns every matching row unpaginated; a first sync |  |
| D16 | YES | the changes query filters on (user_id, modified_at) with no index plan | design-failure: No index on (user_id, modified_at): every sync is a full-tab |
| D17 | — | when Postgres is down every sync 500s with no degraded mode, and the r |  |
| D18 | YES | the wire format carries no version, so any change that is not purely a | design-data: No protocol version field forecloses the breaking changes th |
| D19 | — | in-flight syncs are killed on every rolling deploy and the design lean |  |
| D20 | YES | nightly pg_dump means up to 24h data loss, and the restore procedure h | design-failure: 'No acknowledged edit is ever lost' versus nightly dumps wit |
| D21 | YES | the 30-second sync goal is unreachable with a 15-minute foreground tim | design-failure: The 30-second sync goal is unattainable with the stated trig |
| D22 | YES | the reconnect spike after a regional outage is handled by 'adding inst | design-data: Deterministic, jitter-free retry schedule synchronizes the e |
| D23 | YES | §1 promises support can answer 'why is my note missing' from dashboard | design-failure: The support goal is unattainable with the stated data model; design-data: Stated goal 'answer why a device lacks a note from dashboard |
| D24 | — | the conflict loser is discarded with no copy kept, so a wrong resoluti |  |
| D25 | YES | note ids are random 32-bit integers minted per device: collisions are  | design-failure: Random 32-bit client note ids will collide at the stated sca; design-data: 32-bit random client note ids will collide and silently merg |
| D26 | — | every note has two identities (client id, server BIGSERIAL) with no st |  |
| D27 | — | the outbox stores a full copy of each body beside the note table, with |  |
| D28 | YES | nothing caps a note body, so one pasted file becomes a row and a reque | design-failure: No rate limits, request caps, or body caps on the shared wri |
| D29 | — | the server keeps only the latest body, so nothing can reconstruct a no |  |
| D30 | YES | tombstones are kept forever in both databases with no GC horizon; the  | design-failure: Tombstones and deleted-account rows grow without bound |
| D31 | YES | deleting an account leaves every note row in place, orphaned and undel | design-data: Account deletion orphans all note rows forever with no named |
| D32 | YES | one cursor per account with several devices: whichever device syncs fi | design-failure: One cursor per account cannot represent multiple devices; design-data: One cursor per account silently strands multi-device edits b |
| D33 | — | nothing states who may write a given note: /sync applies whatever edit |  |
| D34 | YES | notes.db is written in the clear with default permissions, while §4 tr | design-failure: Plaintext notes.db with default permissions undermines the s |
| D35 | — | a reserved shared_with column is not a sharing design; the migration i |  |
| D36 | — | a delete on one device racing an edit on another is resolved by timest |  |
| D37 | — | attachments are a URL in the note row with no lifecycle: nothing delet |  |
| D38 | — | the server holds no per-device state at all, so it cannot tell two dev |  |
| D39 | — | modified_at is client-supplied and stored as the truth; the server nev |  |
| D40 | — | step 4 clears the whole outbox, discarding edits the user made while t |  |
