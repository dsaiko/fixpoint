# claude-haiku-5-5 · csharp · run 20261007-204342 (repeat 1)

recall **47/68** · 79 finding(s), 15 unmatched · 103361 tokens · 496s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create always throws after storing the link, so POST /links ; review-security: Quota is never initialised, so every create inserts a link a |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve has an inverted expiry check, deleting live links an; review-security: Resolve has an inverted expiry check: live links are deleted |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never validates t; review-security: Rename never validates the new code, and it has no owner che |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page overruns with end = Count + 1, so GetRange throws whene |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate truncates to 0 unless every link has been follow |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete and Prune never release quota, so per-owner counts on |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending, so it returns the least followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches owners by substring, returning other owners'; review-security: ByOwner matches by substring, and an empty owner matches eve |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives an expired link, which the doc says must neve; review-security: Extend revives expired links, contradicting the documented r |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-concurrency: Store hands out live internal collections despite its docume |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic: a failing code leaves earlier links; review-concurrency: ImportAll documents an atomic all-or-nothing batch but inser |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining count instead of the number remo |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-concurrency: Store shares one System.Random instance across concurrent Cr; review-security: Short link codes come from System.Random, so they are predic |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: LINKD_SECRET is never used, and a hardcoded fallback secret  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens come from System.Random, which is predictable |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Full bearer tokens are written to the console log |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify ignores session expiry, and expired sessions are neve; review-security: Verify never checks session expiry, so sessions stay valid f |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true on both branches and admits every ; review-security: RequireAdmin returns true on both branches, so the admin gat |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: HashPassword uses unsalted, fast SHA-256 for values stored i |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken throws IndexOutOfRange for a missing or malforme; review-security: BearerToken throws IndexOutOfRangeException on an empty or u |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals uses ordinary string equality, not a constant-t |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | YES | CS-ONLY every method locks on `this`, which any external caller can al | review-concurrency: Cache locks on its own public instance, so any code holding  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Peek and Stats read shared state without the lock that guard |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction is not LRU, and Set evicts an unrelated entry when  |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization truncates to 0 for any partially full cache, and |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: An unparseable LINKD_PORT becomes 0, and Validate failures n |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: An unparseable LINKD_TIMEOUT_MS sets the timeout to zero, no |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is parsed as milliseconds but applied as  |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is read and then ignored because the value |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile calls int.Parse on cache_size, so a malformed value |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Report name is joined into the output path unchecked, allowi |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: StreamWriters are closed only on the success path, so an exc |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV rows are written without quoting, so commas or newlines ; review-security: Owner and target values are written to the CSV without escap |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: ArchiveAll uses the owner name as a file name, allowing path |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: WriteSnapshot is neither atomic nor durable: it moves across; review-concurrency: WriteSnapshot uses one fixed /tmp path, so concurrent snapsh |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync ignores the links argument and is async void; review-concurrency: ExportCsvAsync is async void, so its failures cannot be obse |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | YES | the errors counter is reported by Snapshot and never assigned anywhere | review-bugs: The errors counter is never incremented, so /stats always re |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: Janitor timer enumerates and mutates the store with no synch |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before the cache is primed, and its task e; review-concurrency: WarmCache says it returns after every probe finishes, but it |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for an owner with  |
| N54 | YES | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo | review-concurrency: FollowedSummary enumerates a lazy view of the live store twi |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links has no ownership check, so any authenticated u |
| N58 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: /admin/quotas is gated only by a client-set X-Admin header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: /stats divides by zero when the store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Unparsed numeric query parameters throw outside the try bloc; review-security: Link owner is taken from the client-supplied query string, n |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report returns every owner's links to any authenticated use |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | YES | parallel: C26 the accept loop never exits, so StopJanitor is unreachab | review-concurrency: Fire-and-forget Task.Run swallows handler exceptions and lea |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Store.cs:40 — Generated codes are not checked for collisions, so a new link silently overwrites an exist
- (medium) Program.cs:96 — The create response interpolates target and code into JSON without escaping
- (medium) Worker.cs:54 — ProbeAsync does not probe and rejects every http:// target that Store accepts
- (critical) Store.cs:20 — Store dictionaries mutated from concurrent handler threads without any lock
- (high) Program.cs:183 — Report writes a shared file and reads it back, so concurrent requests with the same name r
- (medium) Metrics.cs:17 — AddCreated increments a shared long without Interlocked, despite the class doc claiming th
- (medium) Authenticator.cs:48 — Verify enumerates the session dictionary while Issue and Revoke mutate it, with no synchro
- (high) Export.cs:75 — ReadReport allows reading any file ending in .csv through /report
- (high) Export.cs:28 — The full all-owner link report is written with default permissions, despite the comment sa
- (high) Store.cs:40 — Create overwrites any existing link that has the same random code, hijacking another owner
- (medium) Store.cs:20 — The link store is a plain Dictionary shared by request tasks and the janitor with no locki
- (medium) Config.cs:126 — Validate always returns true, so the startup check never refuses a bad configuration
- (medium) Program.cs:30 — The listener is plain HTTP on all interfaces, so bearer tokens cross the network in cleart
- (medium) Program.cs:37 — Handle has no error handling, so any exception leaves the request unanswered
- (low) Program.cs:188 — Export errors return the server path and full exception text to the client
