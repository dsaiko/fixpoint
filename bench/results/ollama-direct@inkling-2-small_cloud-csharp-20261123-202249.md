# ollama-direct@inkling-2-small:cloud · csharp · run 20261123-202249 (repeat 1)

recall **49/68** · 82 finding(s), 12 unmatched · 337086 tokens · 883s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create() throws KeyNotFoundException for every first-time ow |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve() expiry check is inverted and deletes valid links; review-security: Resolve() expiry check is inverted; live links 404, dead lin |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename() validates the source code twice and never validates; review-security: Rename validates 'from' twice and never validates 'to' |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page() off-by-one overflows GetRange and throws |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate() is integer-truncated to 0 for anything below 1 |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete() never returns the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top() returns the LEAST followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner() matches by substring, not equality; review-security: ByOwner uses substring match, leaking other owners' links |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend() resurrects expired links; review-security: Extend() resurrects expired links, defeating expiry |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas() returns the live internal dictionary, not a snapsho |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll() is not atomic despite its contract |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune() returns the total count instead of the number remove |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune() removes from the dictionary during enumeration; review-concurrency: Prune() modifies the dictionary during enumeration |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: 32-bit short codes with no collision check silently overwrit |
| N16 | YES | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l | review-concurrency: Followed() returns a deferred query over live state; callers |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens and short codes use non-cryptographic System. |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens written to logs in plaintext |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify() never checks token expiry; review-bugs: Sessions are never evicted, so the session table grows witho |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin() returns true for every role; review-security: RequireAdmin() returns true on every path |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted, fast SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken() crashes on any missing/malformed Authorization; review-security: BearerToken and query-string int.Parse throw unhandled excep |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals uses non-constant-time == comparison |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: sessions dictionary accessed without synchronization on the  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | YES | CS-ONLY every method locks on `this`, which any external caller can al | review-concurrency: Locking on `this` exposes the monitor to outside code |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache is documented as thread-safe but Peek, Stats and Utili |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction ignores recency entirely (and punishes cache hits) |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep() removes entries while enumerating the dictionary; review-concurrency: Sweep() removes entries while enumerating entries.Keys, thro |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization() truncates to 0 and divides by zero when limit  |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Unparseable LINKD_TIMEOUT_MS zeroes the timeout instead of k |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds; review-security: LINKD_CACHE_TTL_MS documented as milliseconds but applied as |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile() crashes on a malformed cache_size value |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV output does not escape field values; review-security: CSV formula injection in exported reports |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: ArchiveAll path traversal via owner-controlled file name |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: WriteSnapshot uses a fixed /tmp path and assumes a same-file; review-concurrency: WriteSnapshot uses a fixed shared temp path, so concurrent s |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void and writes no data rows; review-concurrency: ExportCsvAsync is `async void` and fire-and-forget |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes from the live dictionary during enumeration; review-concurrency: Janitor timer callback enumerates the live store dictionary  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before any probe/set finishes; review-concurrency: WarmCache fires unobserved Task.Run calls and returns before |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks on ProbeAsync().Result (sync-over-async) |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for unknown owners; review-concurrency: Quotas() hands out the live internal dictionary despite clai |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response reflects the presented token |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links deletes any link; ownership never checked (IDO |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect at /out?next= |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by client-supplied X-Admin header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats endpoint divides by zero on an empty store; review-security: Division by zero in /stats (and Cache.Utilization) |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Access-Control-Allow-Origin: * combined with Access-Control- |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Query strings are parsed with int.Parse that throws on bad i |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | YES | parallel: X02 every request runs on a thread-pool thread against a Sto | review-concurrency: Fire-and-forget Task.Run per request with no observation, li |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate() always returns true even when the config is rejected
- (low) Cache.cs:53 — Peek() reports expired entries as cached
- (medium) Store.cs:31 — New codes are not checked for collisions with existing links
- (high) Store.cs:20 — Store dictionaries and Link objects are mutated with no synchronization while handlers run
- (low) Metrics.cs:17 — created++ and Snapshot() are not atomic despite the class claiming full thread safety
- (high) Program.cs:183 — Path traversal in /report: arbitrary file write and read
- (medium) Config.cs:126 — Config.Validate() is fail-open and returns true unconditionally
- (medium) Export.cs:28 — Multi-owner report written with default permissions despite 'service user only' contract
- (medium) Program.cs:30 — Service speaks plain HTTP; bearer tokens travel in cleartext
- (low) Program.cs:96 — Unescaped user input interpolated into JSON response
- (low) Program.cs:188 — Error responses leak internal paths and exception details
- (low) Store.cs:41 — Per-owner quota is counted but never enforced
