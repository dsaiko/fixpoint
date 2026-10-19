# ollama-codex@beam:cloud · csharp · run 20261019-210006 (repeat 1)

recall **51/68** · 66 finding(s), 5 unmatched · 991941 tokens · 106s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException for a new owner's quota |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve treats unexpired links as expired (and vice versa) |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice instead of 'to' |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page over-reads by one when offset+size exceeds count |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate truncates to zero before scaling |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not restore the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending by hits instead of descending |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring match instead of exact equality; review-security: ByOwner uses substring match exposing other owners' data |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives already-expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live internal dictionary |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic and validates incrementally |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining count instead of the removed cou |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates the dictionary while iterating its keys; review-concurrency: Prune mutates links while enumerating its keys |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Short codes are predictable due to System.Random |
| N16 | YES | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l | review-concurrency: Followed() returns a deferred query over the live dictionary |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded weak fallback secret in production |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session bearer tokens written to stdout |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify does not enforce session expiry; review-security: Verify() never enforces session expiry |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true; review-security: RequireAdmin() always returns true |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken throws on missing or malformed Authorization hea |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals is not constant-time |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: Sessions dictionary and Random are shared mutable state acce |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek, Warm, Stats, and Utilization are not synchronize |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set evicts the first dictionary key instead of the LRU entry |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep mutates the dictionary while iterating its keys; review-concurrency: Sweep mutates the entries dictionary while enumerating its k |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization integer division truncates to zero until full |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Invalid LINKD_PORT silently becomes zero |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid LINKD_TIMEOUT_MS produces a zero timeout |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never stored |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | YES | parallel: CF06 Validate logs the problems it finds and returns true, s | review-security: Validate() cannot actually reject bad configuration |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile crashes on a non-integer cache_size value |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal in report/archive filenames |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: ExportCsv does not dispose StreamWriter on failure |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-security: Exported CSV files not restricted to service user; review-security: CSV fields not escaped (injection/corruption) |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: WriteSnapshot uses a cross-filesystem move that may fail; review-concurrency: WriteSnapshot uses a single shared temporary file path |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync writes only a header and is fire-and-forget; review-concurrency: ExportCsvAsync is async void and cannot be awaited |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics created/errors counters use non-atomic increments an |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor mutates the store dictionary while iterating keys |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before spawned tasks complete; review-concurrency: WarmCache fires off Task.Run work without awaiting it |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks on async ProbeAsync with .Result |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for owners with no quota entry |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Verbose error responses disclose token and exception details |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links deletes any link without ownership check |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unvalidated next parameter |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated only by spoofable X-Admin header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero when the store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin with credentials |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: POST /links can throw on a non-integer ttl |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: List query parameters are parsed without validation |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate always returns true
- (critical) Store.cs:20 — Store links/quota dictionaries and Random are accessed concurrently without synchronizatio
- (medium) Store.cs:208 — All() and Quotas() escape internal dictionaries, defeating encapsulation
- (high) Worker.cs:34 — Janitor Timer callback enumerates and mutates store collections concurrently without reent
- (high) Authenticator.cs:19 — Session tokens generated with predictable System.Random
