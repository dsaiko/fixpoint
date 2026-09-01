# qwen/qwen3.8-max · csharp · run 20260901-104542 (repeat 1)

recall **43/68** · 55 finding(s), 4 unmatched · 60045 tokens · 1532s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException for every new owner |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve expiry check is inverted: live links are deleted, ex |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to` |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page clamps end to all.Count + 1, so GetRange throws Argumen |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate multiplies after integer division, so it is almo |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete never gives the owner's quota slot back |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending, returning the n least-followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches by substring instead of exact owner; review-security: ByOwner matches by substring, leaking other owners' links |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives already-expired links despite documented fina |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live dictionary, not the snapshot the doc |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic despite its all-or-nothing contract |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining link count instead of the number |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-concurrency: Shared Random instance used from concurrent handlers corrupt; review-security: Unguessable short codes minted with predictable System.Rando |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens generated with predictable System.Random |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Full bearer token written to the log on issue |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never checks ExpiresAt, so tokens never expire; review-security: Verify never checks ExpiresAt, so session tokens never expir |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true in both branches; review-security: RequireAdmin returns true for everyone |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken indexes parts[1] without a bounds check; crashes; review-security: BearerToken throws on any malformed Authorization header |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals is not constant-time despite its documentation |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Peek and Utilization read the entries dictionary without the |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction picks entries.Keys.First() instead of the least rec |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep removes entries while enumerating entries.Keys; review-concurrency: Cache.Sweep removes entries while enumerating the dictionary |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization truncates due to integer-division order, and div |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Malformed LINKD_TIMEOUT_MS zeroes the timeout despite the 'k |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded by a self-ass |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal in report write and read via user-supplied na |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: ArchiveAll uses attacker-controlled owner names as file path |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Snapshot staged at a fixed predictable path in /tmp |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync writes only the CSV header and ignores the li |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.created++ is a non-atomic increment despite the clas |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes from the store while enumerating its keys; u; review-concurrency: Janitor removes from the store dictionary while enumerating  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache fires Task.Run per link and never awaits, contradi |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for owners with no |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links deletes any link regardless of owner (IDOR) |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out with unvalidated next parameter |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by client-supplied X-Admin header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: /stats divides by store.Count with no zero check |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS combined with credentials on every response |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Unchecked int.Parse on query parameters kills the handler on; review-security: Link owner taken from query string instead of the session |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report returns every owner's links to any authenticated cal |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) Config.cs:126 — Validate always returns true, so an invalid configuration never fails startup
- (critical) Store.cs:20 — All Store state is in unsynchronized Dictionaries shared by every request handler
- (medium) Export.cs:28 — Cross-owner export written world-readable despite 'service user only' comment
- (low) Program.cs:96 — JSON response built by raw interpolation of attacker-controlled values
