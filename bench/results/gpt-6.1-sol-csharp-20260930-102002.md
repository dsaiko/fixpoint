# gpt-6.1-sol · csharp · run 20260930-102002 (repeat 1)

recall **29/68** · 41 finding(s), 9 unmatched · 254042 tokens · 588s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Creating an owner's first link throws after inserting it; review-concurrency: Parallel handlers and the janitor mutate an unsynchronized s |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Partial final pages request one element beyond the list |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success percentages truncate before multiplication |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting links leaves owner counts unchanged |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least-followed links |
| N08 | — | parallel: C11 ByOwner matches owners by substring, so one owner sees a |  |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive it |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: Failed imports leave a partially modified store |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining count instead of the removed cou |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: Generated code collisions overwrite existing links |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | — | parallel: S03 every issued session token is logged in cleartext |  |
| N20 | — | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve |  |
| N21 | — | parallel: S07 RequireAdmin returns true from both branches, so every s |  |
| N22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: Missing authorization headers throw instead of returning 401; review-security: Malformed Authorization headers leave HTTP requests unclosed |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | — | parallel: CA02 Peek reads the dictionary with no lock at all while loc |  |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction ignores the recorded access order |
| N31 | YES | parallel: CA06 Warm's Peek and Set are separately locked, so the janit | review-concurrency: Cache warming checks and inserts in separate critical sectio |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Cache utilization reports zero until completely full |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout values replace the default with zero |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: The fetch limit environment setting is ignored |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: Export writers leak handles when writing fails |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV fields are written without escaping |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots share one temporary file |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: Asynchronous export failures cannot be handled by callers; review-concurrency: Async void export cannot be awaited or have failures observe |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | — | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi |  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache returns before its background tasks finish |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: Counting links for an absent owner throws |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-bugs: No request can obtain a valid session |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| N58 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| N59 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Statistics fail when the store is empty |
| N61 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirects cannot resolve supported uppercase codes |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Invalid numeric query parameters leave requests unanswered |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Reports without a name read a different file than they write |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Program.cs:96 — Creation responses do not escape JSON strings
- (medium) Program.cs:197 — Text and CSV responses are labeled as JSON
- (medium) Config.cs:114 — Zero cache capacity passes validation but causes exceptions
- (medium) Config.cs:126 — Validation never rejects invalid configuration
- (medium) Export.cs:83 — Asynchronous exports discard every link
- (medium) Worker.cs:54 — Target checks do not test reachability
- (medium) Authenticator.cs:48 — Session verification races with session issuance
- (medium) Worker.cs:17 — Concurrent creation counter updates are lost
- (high) Export.cs:28 — Concurrent reports compete for the same output file
