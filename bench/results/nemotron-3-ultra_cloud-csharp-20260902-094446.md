# nemotron-3-ultra:cloud · csharp · run 20260902-094446 (repeat 1)

recall **44/68** · 69 finding(s), 14 unmatched · 485679 tokens · 1522s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | — | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep |  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve expiration logic is inverted - removes valid links |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' code twice instead of 'from' and 'to |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page has off-by-one error in range calculation |
| N05 | — | parallel: C05 integer division before scaling yields 0 for every rate  |  |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not decrement owner quota |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns least-followed links instead of most-followed |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring match instead of exact match; review-security: ByOwner uses Contains instead of equality enabling partial m |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links and extends from now not curren |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns internal dictionary allowing external mutatio; review-concurrency: Quotas() and All() expose internal dictionaries — allows ext |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll doesn't validate duplicates or update quotas |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns remaining count instead of removed count |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune modifies dictionary during enumeration |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-bugs: Token generation uses non-cryptographic Random |
| N19 | — | parallel: S03 every issued session token is logged in cleartext |  |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify does not check session expiration |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true, bypassing authorization; review-security: RequireAdmin always returns true, bypassing admin authorizat |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Unsalted SHA-256 for password hashing vulnerable to rainbow  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken vulnerable to IndexOutOfRangeException |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-bugs: SecretEquals uses non-constant-time comparison; review-security: Non-constant-time secret comparison in SecretEquals |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: Sessions dictionary accessed without synchronization from mu |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Peek() reads entries dictionary without locking while other  |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Stats() reads hits field without volatile/Interlocked while  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set eviction picks arbitrary key, not LRU |
| N31 | YES | parallel: CA06 Warm's Peek and Set are separately locked, so the janit | review-concurrency: Warm() has TOCTOU race between Peek() (unlocked) and Set() ( |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep modifies dictionary during enumeration; review-security: Sweep modifies dictionary during enumeration causing crash |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization uses integer division, always returns 0 |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS parsed as seconds instead of milliseconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT parsed but never applied |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | YES | parallel: CF06 Validate logs the problems it finds and returns true, s | review-security: Validate() always returns true even when validation fails |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal in ExportCsv via user-controlled 'name' param |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: ExportCsv StreamWriter not disposed on exception; review-concurrency: StreamWriter not disposed — resource leak under concurrent e |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: WriteSnapshot uses hardcoded /tmp path; review-security: Predictable temporary file path in /tmp enables TOCTOU race  |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync returns void, loses exceptions and doesn't wr; review-concurrency: ExportCsvAsync is async void — fire-and-forget with unobserv |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-bugs: AddCreated not thread-safe while AddResolved is |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: StartJanitor modifies store dictionary during enumeration; review-security: Janitor modifies dictionary during enumeration causing crash |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache fires Task.Run for each link but doesn't await — f |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-bugs: CheckTargets uses .Result on async method causing potential ; review-concurrency: CheckTargets uses .Result on async method — potential deadlo |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for unknown owners |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-bugs: BearerToken throws IndexOutOfRangeException on missing/malfo |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: BearerToken crashes on malformed header and leaks token in e |
| N57 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint via user-controlled 'next' pa |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: AdminQuotas uses client-controlled X-Admin header instead of |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Division by zero in Stats when store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: TTL parsing throws on invalid input without error handling |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: Offset/size parsing throws on invalid input |
| N66 | YES | parallel: X02 every request runs on a thread-pool thread against a Sto | review-concurrency: No coordination between HTTP handlers and janitor timer — sh |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) Config.cs:126 — Validate always returns true even with configuration errors
- (high) Store.cs:102 — SuccessRate uses integer division, always returns 0 or 100
- (medium) Cache.cs:51 — Peek doesn't check expiration, inconsistent with Get
- (medium) Worker.cs:31 — WarmCache fires tasks but doesn't await them
- (low) Store.cs:208 — All() returns internal dictionary breaking encapsulation
- (critical) Store.cs:20 — No synchronization on shared dictionaries accessed from HTTP handlers and janitor timer
- (critical) Worker.cs:34 — Janitor directly mutates store's internal dictionary via store.All() concurrently with han
- (high) Metrics.cs:17 — AddCreated() uses non-atomic increment while AddResolved() uses Interlocked
- (medium) Cache.cs:124 — Utilization() reads entries.Count and limit without locking
- (medium) Worker.cs:86 — CountForOwner accesses store.Quotas() dictionary directly without synchronization
- (high) Store.cs:27 — SSRF risk: target URL validation only checks http/https prefix
- (high) Authenticator.cs:19 — Cryptographically insecure session token generation using System.Random
- (medium) Program.cs:192 — Invalid CORS configuration: Allow-Origin * with Allow-Credentials true
- (medium) Store.cs:208 — All() exposes internal dictionary allowing external mutation
