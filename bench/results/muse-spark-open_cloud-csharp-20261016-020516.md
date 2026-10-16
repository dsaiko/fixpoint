# muse-spark-open:cloud · csharp · run 20261016-020516 (repeat 1)

recall **22/68** · 25 finding(s), 3 unmatched · 169144 tokens · 154s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | — | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep |  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-security: Resolve expiry check inverted - expired links stay valid |
| N03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| N04 | — | parallel: C04 page clamps end past the list size, so GetRange throws o |  |
| N05 | — | parallel: C05 integer division before scaling yields 0 for every rate  |  |
| N06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| N07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| N08 | — | parallel: C11 ByOwner matches owners by substring, so one owner sees a |  |
| N09 | — | parallel: C12 Extend re-dates an already-expired link, resurrecting a  |  |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | — | parallel: C13 ImportAll stores as it validates, so a bad code leaves t |  |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET unset |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Predictable session tokens from System.Random |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token written to stdout/logs |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-security: Verify never checks ExpiresAt - expired sessions live foreve |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-security: RequireAdmin always returns true - authorization bypass |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with single SHA256 without salt |
| N23 | — | parallel: S06 BearerToken indexes parts[1] without checking, so a malf |  |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: Timing-unsafe secret comparison |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek reads Dictionary without synchronization |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Cache.Stats and Utilization read shared state without lock |
| N30 | — | parallel: CA08 eviction drops whatever key comes first, not the least  |  |
| N31 | YES | parallel: CA06 Warm's Peek and Set are separately locked, so the janit | review-concurrency: Cache.Warm check-then-act TOCTOU between unsynchronized Peek |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-concurrency: Cache.Sweep removes entries while enumerating Keys under loc |
| N33 | — | parallel: C05 Utilization divides before scaling so it always reports  |  |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| N36 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu |  |
| N37 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne |  |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal file write via user-controlled report name |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Predictable /tmp snapshot path enables symlink/hijack |
| N47 | — | CS-ONLY async void: the caller cannot await it and an exception inside |  |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated uses non-atomic increment, losing updates |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: Janitor Timer mutates Store Dictionary during enumeration wi |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache fire-and-forgets Task.Run and never awaits complet |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | — | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot |  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links missing ownership check (IDOR) |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= attacker URL |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: AdminQuotas authorized by client-controlled X-Admin header |
| N60 | — | parallel: C20 /stats divides by the link count, throwing DivideByZeroE |  |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: wildcard origin with credentials |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link owner spoofing via query param instead of session |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Store.cs:208 — Store.All() exposes mutable internal dictionary
- (high) Export.cs:75 — Path traversal file read via report name
- (medium) Config.cs:126 — Config.Validate always returns true - bad config starts service
