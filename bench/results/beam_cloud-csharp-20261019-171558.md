# beam:cloud · csharp · run 20261019-171558 (repeat 1)

recall **44/68** · 57 finding(s), 5 unmatched · 62286 tokens · 116s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws for any owner without an existing quota |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve removes valid links and keeps expired ones |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the wrong code twice and skips the target |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page end index is off by one |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate integer division yields 0 unless fully used |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not return the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top() returns least-used links first |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches substrings; review-security: ByOwner uses substring match, leaking other owners' links |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live internal dictionary; review-security: Quotas returns internal mutable dictionary |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic and partially applies |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune returns the wrong count |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Short codes generated with insecure Random (predictable) |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens written to stdout |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify ignores session expiry; review-security: Session expiry never enforced |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true; review-security: RequireAdmin always returns true, bypassing admin checks |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashed with unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken crashes on malformed header |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals compares with == (timing side channel) |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: sessions Dictionary and Random are shared across request thr |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache exposes unlocked reads and counters alongside a locked |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction does not pick least-recently-used entry |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep mutates dictionary during enumeration; review-concurrency: Sweep mutates the dictionary while enumerating its keys |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization always reports 0 |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS zeroes the timeout |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | YES | parallel: CF06 Validate logs the problems it finds and returns true, s | review-bugs: Validate always returns true; review-security: Validate() always returns true |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile int.Parse can throw on bad value |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal via user-controlled report name |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Fixed shared temp file enables symlink/TOCTOU attack on snap |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void and writes only a header |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor mutates the store dictionary during enumeration; review-concurrency: Janitor timer callback mutates store through All() and can o |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache does not wait for probes; review-concurrency: WarmCache starts tasks but never awaits or observes them |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for unknown owners |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-security: BearerToken can throw on malformed Authorization header (DoS |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown token echoed back in 401 response |
| N57 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized solely by client-supplied X-Admin  |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero when the store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin while enabling credentials |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Owner taken from query string instead of authenticated sessi |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: Links query endpoints crash on non-integer offset/size |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) Store.cs:20 — Store has no synchronization; All() leaks its backing dictionary and Link.Hits races
- (medium) Metrics.cs:16 — created and errors are not atomic and Snapshot can tear
- (high) Authenticator.cs:19 — Session tokens generated with insecure System.Random
- (medium) Store.cs:208 — All() exposes the live links dictionary
- (low) Export.cs:27 — Exported reports written with default world-readable permissions
