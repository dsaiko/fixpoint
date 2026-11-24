# ollama-direct@hy4:cloud · csharp · run 20261124-100912 (repeat 1)

recall **51/68** · 86 finding(s), 19 unmatched · 905391 tokens · 1207s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: First link creation always fails with KeyNotFoundException |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve removes live links and serves expired ones; review-security: Expiry comparison is reversed, so expired links can still re |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never validates t |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page asks GetRange for one item more than the list contains |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate collapses every partial percentage to zero |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not restore the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least-followed links first |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches substrings instead of exact owners; review-security: ByOwner uses substring matching and leaks other owners' link |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend resets expiry from now and revives dead links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live dictionary instead of a snapshot; review-security: Mutable internal quota and link dictionaries are exposed |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining count instead of the removed cou |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-concurrency: Concurrent link creation uses a shared non-thread-safe Rando; review-security: Short codes are predictable |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback shared secret |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-concurrency: Shared Random can mint duplicate authentication tokens; review-security: Session bearer tokens are generated with a non-cryptographic |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session bearer tokens are written to stdout |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never expires sessions; review-security: Verify ignores session expiry |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin accepts every role; review-security: RequireAdmin authorizes every non-admin role |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords are hashed with unsalted fast SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken throws on empty or malformed Authorization heade; review-security: Malformed Authorization input aborts request handling |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: Shared-secret comparison is not constant time |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache methods that claim thread safety read shared state wit |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Cache eviction is not LRU and fails at zero limit |
| N31 | YES | parallel: CA06 Warm's Peek and Set are separately locked, so the janit | review-concurrency: Warm has a check-then-act race that can overwrite a fresh ca |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization reports only 0 or 100 and divides by zero |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Invalid LINKD_PORT replaces the default with zero |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS overwrites the default with zero |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Millisecond cache TTL is interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| N38 | YES | parallel: CF05 LoadFile swallows every exception, so an unreadable con | review-bugs: LoadFile silently ignores all file read errors |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: Invalid cache_size in a config file aborts config loading |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: User-controlled report names escape the export directory |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: Report writers leak handles when a write fails |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV fields are written without escaping; review-concurrency: Concurrent report requests with the same name write and read |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots share one fixed temporary file; review-security: Fixed /tmp snapshot path allows symlink overwrite |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: Async export is unawaitable and writes only a header; review-concurrency: ExportCsvAsync is async void fire-and-forget |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics created counter uses a non-atomic increment and unsy |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | — | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi |  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before cache writes have completed; review-concurrency: WarmCache returns before its parallel work completes |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks on async probes and can deadlock under a |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for owners with no quota entry |
| N54 | YES | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo | review-concurrency: FollowedSummary enumerates a live dictionary-backed query tw |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links ignores the authenticated owner |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated /out Location enables phishing through a trusted |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quota endpoint trusts a client-supplied X-Admin header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero on an empty store; review-concurrency: Stats takes multiple unsynchronized Store views and can divi |
| N61 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect lowercases codes that the store permits |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Malformed ttl query crashes link creation; review-security: POST /links lets the caller spoof another owner |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint discloses every owner's links to any authent |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: Malformed or negative pagination query crashes listing |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | YES | parallel: C26 the accept loop never exits, so StopJanitor is unreachab | review-concurrency: HTTP handler tasks are fire-and-forget with no completion, c |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report writes one file but reads another |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Program.cs:37 — Discarded handler tasks leave requests without any response
- (high) Config.cs:126 — Validate always succeeds and startup ignores its result
- (medium) Program.cs:96 — Link creation response can emit invalid JSON
- (medium) Program.cs:121 — Links listing returns only a count, not the page contents
- (medium) Program.cs:164 — Admin quotas endpoint returns owner count instead of quotas
- (medium) Cache.cs:53 — Peek treats expired entries as present
- (medium) Worker.cs:38 — Janitor bypasses Delete and leaves quota counts inflated
- (medium) Worker.cs:53 — ProbeAsync never probes the target
- (low) Program.cs:197 — All responses are labeled application/json
- (critical) Store.cs:20 — Store dictionaries are mutated by request tasks and the janitor without synchronization
- (high) Store.cs:41 — Store read-modify-write and multi-step operations are not atomic
- (high) Authenticator.cs:37 — Session store is unsynchronized and revocation has a TOCTOU window
- (medium) Worker.cs:31 — Repeated janitor starts leak Timers and allow overlapping callbacks
- (medium) Export.cs:69 — Concurrent audit appends are not coordinated
- (high) Program.cs:30 — Bearer tokens and reports are served over plaintext HTTP on all interfaces
- (medium) Export.cs:69 — Audit log entries can be forged with unescaped newlines
- (medium) Program.cs:96 — Create-link response injects unescaped target into JSON
- (medium) Program.cs:188 — Verbose exception details are returned to clients
- (medium) Config.cs:126 — Invalid runtime configuration is accepted
