# gpt-6.2-astra · csharp · run 20261118-150611 (repeat 1)

recall **48/68** · 91 finding(s), 27 unmatched · 132165 tokens · 442s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Creating a link for a new owner throws after insertion; review-bugs: A generated code collision silently replaces an existing lin |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it; review-security: Resolving a live link deletes it, while an expired link stil |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and accepts an invalid des |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: A partial final page throws instead of returning the remaini |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: The dashboard reports zero success for every partially follo |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Removing links leaves per-owner quota counts stale |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least-followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup matches substrings instead of exact owners; review-security: The owner lookup returns records from partially matching acc |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive it after |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: The quota snapshot exposes mutable store state |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: A failed batch import leaves earlier records committed; review-concurrency: Readers can observe a partially imported batch |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the number of surviving links |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | YES | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime | review-bugs: Local wall-clock timestamps distort link and cache lifetimes |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens are derived from a noncryptographic random ge |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-concurrency: Concurrent session issuance can assign the same token to two; review-security: Token issuance writes the full bearer credential to stdout |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Expired sessions accumulate indefinitely; review-security: Expired bearer tokens continue to authenticate |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-security: RequireAdmin grants access to every role |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: The password-storage helper produces fast, unsalted hashes |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: A missing Authorization header leaves the request unanswered; review-security: A missing Authorization header leaves the HTTP request unans |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache readers bypass the lock protecting dictionary mutation |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: The hit counter is read outside its writer's lock |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction ignores access history and also runs on replacement |
| N31 | YES | parallel: CA06 Warm's Peek and Set are separately locked, so the janit | review-concurrency: Cache warming can overwrite a value inserted after its prese |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Cache utilization reports zero until the cache is full |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: A malformed port setting erases the default port |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A malformed timeout replaces the default with zero |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: The cache TTL override is interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: The fetch limit override is discarded |
| N38 | YES | parallel: CF05 LoadFile swallows every exception, so an unreadable con | review-bugs: Configuration read failures silently discard file overrides |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: A report name can overwrite CSV files outside the export dir |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-security: All-owner report files can be readable by other local users |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV reports and owner archives lose field boundaries; review-security: Unescaped report fields can inject spreadsheet formulas and  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot replacement can expose a partially copied file; review-concurrency: Concurrent snapshots share one temporary file across all des |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: Asynchronous export failures cannot be observed by the calle |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-bugs: The dashboard error counter can never change; review-concurrency: Concurrent creation counters overwrite each other's incremen |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: A concurrent store change can terminate the process during a; review-security: Concurrent link requests can terminate the service during a  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache returns before its queued writes finish |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: Synchronous probe waits can deadlock a caller's synchronizat |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: Counting links for an unknown owner throws |
| N54 | YES | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo | review-concurrency: The followed count and first link can describe different sto |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-bugs: A freshly started server cannot accept any request |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete another owner's link |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Permanent redirect caching bypasses email click counting; review-security: The email click endpoint redirects to an unverified destinat |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: The quotas endpoint accepts a caller-supplied admin claim |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: The stats endpoint throws when the store is empty; review-concurrency: Removing the last link during a stats request can cause divi |
| N61 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect changes the case of valid stored codes |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Invalid numeric query parameters escape the request handler; review-security: Clients can create records attributed to another owner |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: An unnamed report is read back from the wrong file; review-security: Repeated report downloads can exhaust the export filesystem |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Invalid configuration proceeds into service startup
- (high) Program.cs:197 — CSV and plain-text responses are advertised as JSON
- (medium) Program.cs:96 — Link creation can return malformed JSON
- (medium) Store.cs:27 — URL prefix checks accept unusable targets and reject valid schemes
- (medium) Program.cs:164 — The quota endpoint discards every owner's link count
- (medium) Program.cs:183 — Reports silently omit links after the first 1,000
- (medium) Export.cs:28 — The first export fails when the configured directory is absent
- (medium) Export.cs:83 — The asynchronous exporter overwrites reports with a header only
- (medium) Cache.cs:53 — Expired entries prevent cache warming
- (medium) Store.cs:170 — Successful imports leave owner counts out of sync
- (medium) Store.cs:208 — All lets callers bypass store bookkeeping
- (medium) Worker.cs:54 — Target checks report reachability from the URL scheme alone
- (high) Program.cs:37 — A faulted request task can leave the HTTP response open
- (high) Store.cs:40 — Concurrent creates can lose links in the shared dictionary
- (high) Program.cs:184 — Concurrent reports with the same name can return the wrong CSV
- (medium) Store.cs:228 — Concurrent code generation can overwrite an existing link
- (medium) Authenticator.cs:48 — Issuing a session can interrupt verification of another session
- (medium) Store.cs:41 — Concurrent creations can lose an owner's quota increment
- (medium) Store.cs:60 — Simultaneous resolutions can lose hit counts
- (medium) Store.cs:69 — Two renames can both claim the same destination
- (medium) Worker.cs:47 — Stopping the janitor can return while a callback still mutates the store
- (medium) Export.cs:80 — Callers cannot await asynchronous CSV export or observe its failure
- (medium) Export.cs:69 — Overlapping audit appends can lose a record
- (high) Program.cs:183 — The report endpoint exports other owners' links
- (medium) Program.cs:30 — The listener accepts bearer credentials over cleartext HTTP on every interface
- (medium) Store.cs:40 — A short-code collision silently redirects an existing code to a new target
- (medium) Program.cs:188 — Report errors disclose filesystem paths and exception details
