# gpt-6.2-sol · csharp · run 20261118-143537 (repeat 1)

recall **36/68** · 66 finding(s), 25 unmatched · 254306 tokens · 606s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: A new owner's first create leaves a link behind and throws |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and accepts invalid destin |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: A partial final page throws an out-of-range exception |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: The follow percentage reports zero until every link has been |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Owner quotas drift when links are removed or imported |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup matches substrings |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive it |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: The quota snapshot exposes mutable store state |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: A failed batch import leaves earlier changes applied; review-concurrency: Readers can observe a partially imported batch |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the number of surviving links |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | YES | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime | review-bugs: Local wall-clock changes alter link and cache lifetimes |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | — | parallel: S03 every issued session token is logged in cleartext |  |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Issued sessions never age out |
| N21 | — | parallel: S07 RequireAdmin returns true from both branches, so every s |  |
| N22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: A missing or one-part Authorization header aborts the reques |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache inspection bypasses the lock used by writers |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Cache eviction ignores recency and can evict on replacement |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Partial cache utilization is always reported as zero |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Malformed numeric environment values overwrite defaults with |
| N35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL is interpreted in seconds instead of milliseconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: The fetch-limit override has no effect |
| N38 | YES | parallel: CF05 LoadFile swallows every exception, so an unreadable con | review-bugs: Configuration file read failures are silently ignored |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: Failed CSV writes leave file handles open |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV exports corrupt fields containing commas, quotes, or lin |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot replacement can overwrite the last good copy with a; review-concurrency: Concurrent snapshot writes share one temporary file |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: An asynchronous export write failure can terminate the proce; review-concurrency: Asynchronous exports expose neither completion nor write fai |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-bugs: The dashboard error counter always reports zero |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | — | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi |  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: Cache warming returns before the scheduled writes finish |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: Target checks can deadlock on a single-threaded synchronizat |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: Counting links for an unknown owner throws |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-bugs: The running service has no way to obtain a valid session |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Permanent outbound redirects can bypass click counting |
| N59 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: The stats handler divides by zero on an empty store; review-concurrency: Statistics can divide by zero when a concurrent operation re |
| N61 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Lowercasing a code breaks links stored with uppercase letter |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Invalid numeric query values leave requests unanswered |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: An unnamed report is written and then read from a different ; review-concurrency: Requests for the same report name can return each other's CS |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) Store.cs:40 — A generated code collision silently replaces an existing link
- (high) Config.cs:126 — Startup continues after configuration validation finds errors
- (medium) Store.cs:27 — A scheme prefix is accepted as a complete link target
- (medium) Program.cs:96 — The create response interpolates unescaped JSON values
- (medium) Program.cs:197 — CSV and plain-text responses are labeled as JSON
- (medium) Export.cs:28 — Report generation fails when the export directory does not exist
- (medium) Program.cs:183 — Reports silently omit links after the first 1,000
- (medium) Export.cs:83 — The asynchronous export drops every supplied link
- (medium) Cache.cs:64 — A zero-sized cache crashes on insert and stats
- (medium) Cache.cs:53 — Expired entries prevent warming until the next sweep
- (medium) Program.cs:128 — The resolver never uses the configured cache
- (medium) Worker.cs:54 — Target checks classify URLs by scheme without checking reachability
- (high) Program.cs:37 — Faulted request tasks leave the HTTP response open
- (medium) Worker.cs:36 — Deleting a link during a janitor tick can terminate the process
- (medium) Store.cs:199 — Store readers enumerate dictionaries while other callers mutate them
- (medium) Store.cs:41 — Parallel creates can lose an owner's quota increment
- (medium) Store.cs:228 — Parallel code generation can overwrite a link
- (medium) Store.cs:60 — Parallel resolutions lose per-link hits
- (medium) Store.cs:69 — Concurrent renames to the same code can discard a link
- (medium) Authenticator.cs:48 — Session verification races with session issuance
- (medium) Cache.cs:95 — Cache warming can replace a value inserted by another caller
- (medium) Worker.cs:17 — Concurrent metric updates lose creation counts
- (medium) Worker.cs:47 — Stopping the janitor can leave a sweep running
- (medium) Export.cs:69 — Concurrent audit appends can reject one of the records
- (critical) Authenticator.cs:94 — Unauthenticated requests can exhaust listener connections
