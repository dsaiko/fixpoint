# codex@google/gemini-4-argon · csharp · run 20261013-225523 (repeat 1)

recall **44/68** · 56 finding(s), 4 unmatched · 1041296 tokens · 444s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Store.Create throws KeyNotFoundException on an owner's first |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted expiration check in Store.Resolve deletes active li |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Store.Rename validates from twice and does not validate to; review-security: Rename validates source code twice and skips validating dest |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Off-by-one clamping in Store.Page throws ArgumentException w |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Integer division in Store.SuccessRate truncates partial perc |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Store.Delete does not return the deleted link's quota slot t |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Store.Top sorts links in ascending order instead of most hit |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: Store.ByOwner matches owner substrings instead of exact owne; review-security: ByOwner uses substring Contains check instead of exact equal |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Store.Extend revives expired links and overwrites expiry fro |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Store.Quotas returns the internal dictionary instead of a sn |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: Store.ImportAll is not atomic when a later link in the batch |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Store.Prune returns the remaining link count instead of the  |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Short codes generated with predictable System.Random PRNG |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens generated with predictable System.Random PRNG |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Plaintext bearer session tokens written to standard output l |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Authenticator.Verify ignores Session.ExpiresAt and returns e; review-security: Verify never checks session expiration timestamp |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: Authenticator.RequireAdmin unconditionally returns true and ; review-security: RequireAdmin always returns true and permits non-admin owner |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Unsalted single-pass SHA-256 used for password hashing |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: Authenticator.BearerToken throws IndexOutOfRangeException on; review-security: BearerToken indexes split Authorization header without bound |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: Non-constant-time secret comparison in SecretEquals leaks ti |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: Authenticator sessions dictionary and Random instance are no |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek, Warm, Stats, and Utilization access shared state |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Cache.Set evicts the first dictionary key instead of the lea |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Integer division in Cache.Utilization truncates usage below  |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid LINKD_TIMEOUT_MS and LINKD_PORT environment values o |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is converted as seconds instead of millis |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT assigns parsed value to itself instead of  |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal via unsanitized report and owner names in CSV |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-security: Sensitive CSV report written with default world-readable per |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: WriteSnapshot uses a fixed shared temporary file path that r; review-security: Predictable temporary file in /tmp in WriteSnapshot enables  |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: Export.ExportCsvAsync is async void and never writes the lin; review-concurrency: ExportCsvAsync is async void, preventing callers from awaiti |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated and Snapshot have data races despite thre |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: StartJanitor mutates Store's dictionary without synchronizat |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: Worker.WarmCache returns before background tasks finish; review-concurrency: WarmCache spawns fire-and-forget tasks without waiting for c |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks synchronously on ProbeAsync(...).Result |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: Worker.CountForOwner throws KeyNotFoundException when owner  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unescaped reflection of untrusted bearer token in 401 respon |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Missing ownership check on DELETE /links allows deleting arb |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unvalidated next query parameter in /out |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts client-supplied X-Admin header instead |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Program.Stats divides by zero when the store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS origin combined with Access-Control-Allow-Cred |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: POST /links takes owner from untrusted query parameter and i |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Config.Validate always returns true and Program.Main ignores its return value
- (medium) Export.cs:75 — Export.ReadReport fails on empty report name while ExportCsv defaults it to links
- (critical) Store.cs:20 — Store state, Random instance, and Link hit counters are accessed without synchronization
- (medium) Program.cs:188 — Verbose 500 error response leaks internal filesystem path and full exception stack trace
