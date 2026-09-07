# gpt-5.6-terra · csharp · run 20260907-144036 (repeat 1)

recall **27/68** · 36 finding(s), 5 unmatched · 310968 tokens · 325s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Creating the first link for an owner throws after inserting  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve removes live links and serves expired ones |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename never validates the destination code |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page requests past the end construct an out-of-range slice |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success rate loses all partial percentages |
| N06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns least-followed links first |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup includes other owners with matching substrings |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune throws while removing expired links |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-concurrency: Session issuance, revocation, and verification race on the s; review-security: Session bearer tokens are written to logs |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-security: Expired sessions remain valid indefinitely |
| N21 | — | parallel: S07 RequireAdmin returns true from both branches, so every s |  |
| N22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: Malformed or absent Authorization headers crash request hand |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache observers read Dictionary state outside the cache lock |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction does not select the least recently used cache entry |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Cache sweep mutates the dictionary during enumeration; review-concurrency: Cache sweep invalidates its own enumeration |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Cache utilization is always zero below full capacity and can |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL environment value uses seconds instead of document |
| N37 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne |  |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: User-controlled report name permits path traversal and file  |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: All snapshots share one temporary filename |
| N47 | — | CS-ONLY async void: the caller cannot await it and an exception inside |  |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor throws when it encounters an expired link; review-concurrency: Janitor mutates the dictionary it is enumerating |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before cache warming has completed; review-concurrency: WarmCache returns before its background cache work completes |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | — | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot |  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete another user's link |
| N58 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts an attacker-controlled request header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats endpoint crashes when the store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Invalid ttl query values escape the request error handling |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exposes every user's link data |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Config.cs:126 — Configuration validation accepts every invalid configuration
- (medium) Export.cs:83 — Asynchronous CSV export discards all supplied links
- (high) Store.cs:20 — Request handlers and janitor access the link dictionary without synchronization
- (medium) Metrics.cs:17 — Created counter loses concurrent increments
- (medium) Export.cs:28 — Concurrent reports with the same name race on one output file
