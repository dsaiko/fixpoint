# claude-sonnet-5-5 · csharp · run 20260929-100208 (repeat 1)

recall **51/68** · 73 finding(s), 7 unmatched · 27256 tokens · 223s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create always throws KeyNotFoundException: quota[owner] read |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve expiry check is inverted |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates the new co |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page computes end = Count + 1, so GetRange throws on the las |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate truncates to 0 before multiplying |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not give the quota slot back |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending, returning the least followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses Contains instead of equality; review-security: ByOwner matches by substring, not exactly |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links; review-security: Extend revives expired links, contradicting the stated invar |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live internal dictionary rather than a sn; review-security: Quotas() and All() hand out the live internal dictionaries |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic despite its contract |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune removes during enumeration and returns links.Count ins |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-concurrency: Shared System.Random used to mint codes from many threads |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret and public accessor |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-concurrency: Session dictionary and Random used from concurrent requests ; review-security: Session tokens generated with non-cryptographic System.Rando |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens written to logs and echoed in error responses |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never checks session expiry; review-security: Verify never checks session expiry |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true for every session; review-security: RequireAdmin always returns true |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted single-round SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken indexes parts[1] without checking length |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals is not constant-time despite its contract |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | YES | CS-ONLY every method locks on `this`, which any external caller can al | review-concurrency: lock(this) exposes the cache's lock object publicly |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Peek reads the dictionary without the lock, and Warm's check |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction picks an arbitrary key, not the least recently used |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep mutates the dictionary during enumeration |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization uses integer division and divides by zero when l |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: int.TryParse(port, out cfg.Port) zeroes the port on a bad va |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Unparsable LINKD_TIMEOUT_MS sets the timeout to zero despite |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is parsed as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never stored |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | YES | parallel: CF06 Validate logs the problems it finds and returns true, s | review-security: Validate() always returns true, so invalid configuration is  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-bugs: Report name is not confined to dir (path traversal, absolute; review-security: Path traversal / arbitrary file write and read via report na |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-concurrency: Concurrent /report requests race on the same file; review-security: Reports containing every owner's links are created with defa |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV output is not escaped and the report file handle leaks o |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: Owner string used as a filename in ArchiveAll (path traversa |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: WriteSnapshot is not atomic: it writes to /tmp and moves acr; review-concurrency: WriteSnapshot uses one fixed shared temp path |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void, so failures crash the process,; review-concurrency: async void ExportCsvAsync cannot be awaited and crashes the  |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics claims thread safety but created++ and Snapshot are  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes from the dictionary while enumerating it, an; review-concurrency: Janitor mutates the store's live dictionary from a timer thr |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache is fire-and-forget although it documents that it r; review-concurrency: WarmCache fires Task.Run per link and never awaits, contrary |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: Sync-over-async .Result blocks a thread per probe |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for an owner with  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-security: Unhandled exceptions in request handling leave requests hang |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-bugs: DELETE and POST ignore the authenticated session: any caller |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect at /out |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin endpoint is gated by a client-supplied X-Admin header,; review-security: Admin authorization trusts a client-supplied X-Admin header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by store.Count, which is zero on an empty stor |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS combined with Allow-Credentials, over plain HT |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: int.Parse on query parameters throws unhandled FormatExcepti; review-security: Link owner taken from request parameters, and delete has no  |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exposes every owner's links to any authentic |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | YES | parallel: X02 every request runs on a thread-pool thread against a Sto | review-concurrency: Fire-and-forget Task.Run(Handle) swallows exceptions and lea |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Program.cs:20 — Startup ignores Validate, and Validate always returns true
- (high) Export.cs:75 — ReadReport does not apply the 'links' default that ExportCsv uses, so an empty name fails
- (low) Program.cs:96 — JSON response is built by string interpolation without escaping
- (high) Store.cs:40 — Store has no synchronization but is shared by every request thread and the janitor
- (low) Worker.cs:47 — StopJanitor does not wait for an in-flight callback
- (medium) Program.cs:188 — Full exception text returned to clients
- (low) Program.cs:96 — JSON response built by string interpolation without escaping
