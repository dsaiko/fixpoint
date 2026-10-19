# beam:cloud · csharp · run 20261019-200310 (repeat 2)

recall **45/68** · 59 finding(s), 6 unmatched · 115409 tokens · 113s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create crashes on unknown owner quota |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve treats live links as expired |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates from twice, not to |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page reads one element past the end |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate integer division loses remainder |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not restore owner quota |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending instead of descending |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring match |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns internal dictionary reference; review-concurrency: Quotas() returns the live internal dictionary despite snapsh |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates dictionary during iteration and returns wrong  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Short codes minted with predictable System.Random |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens written to console log |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify ignores session expiry; review-security: Verify ignores session expiry |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true; review-security: RequireAdmin always returns true, bypassing admin-only check |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses fast unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken crashes on malformed header |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals uses non-constant-time comparison |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: sessions dictionary and Random shared without synchronizatio |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | — | parallel: CA02 Peek reads the dictionary with no lock at all while loc |  |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction does not pick least recently used entry |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep mutates dictionary during enumeration; review-concurrency: Sweep mutates the dictionary while foreach iterates its Keys |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization always truncates to zero |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Bad LINKD_PORT overwrites default with zero |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS still overwrites default |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL treated as seconds, not milliseconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | YES | parallel: CF06 Validate logs the problems it finds and returns true, s | review-bugs: Validate always returns true |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile throws on non-integer cache_size |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal via user-controlled report name |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: WriteSnapshot uses a hard-coded shared temp file path; review-security: Snapshot uses a shared predictable /tmp path (TOCTOU/symlink |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync ignores input and is async void; review-concurrency: ExportCsvAsync is async void with unobserved exceptions |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor mutates dictionary during iteration; review-concurrency: Janitor callback enumerates and mutates store.All() unsynchr |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache does not wait for probes; review-concurrency: WarmCache is fire-and-forget and returns before work complet |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks on async work with .Result |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for unknown owner |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Presented (invalid) token reflected in 401 response |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links does not enforce ownership (IDOR) |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out via unvalidated next parameter |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: AdminQuotas trusts a spoofable X-Admin header instead of the |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Divide by zero when store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS with credentials enabled |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Unguarded integer parse of ttl query |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: Unguarded integer parse of paging query |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Store.cs:208 — All exposes internal links dictionary
- (high) Cache.cs:16 — Thread-safety claim is false: Peek, Warm, Stats and Utilization bypass the lock
- (medium) Metrics.cs:15 — AddCreated is non-atomic and Snapshot reads unsynchronized long fields
- (high) Store.cs:20 — Store has no synchronization despite concurrent callers and a background janitor
- (high) Authenticator.cs:19 — Session tokens generated with insecure System.Random
- (medium) Export.cs:28 — Reports written with default (world-readable) file permissions
