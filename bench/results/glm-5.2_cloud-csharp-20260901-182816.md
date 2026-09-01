# glm-5.2:cloud · csharp · run 20260901-182816 (repeat 1)

recall **38/68** · 52 finding(s), 5 unmatched · 293466 tokens · 369s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-concurrency: Store's links/quota dictionaries and Random are mutated by c |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve has inverted expiry check: removes valid links, serv |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never validates t |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page sets end = all.Count + 1, crashing GetRange when size e |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate uses integer division, always returning 0 or 100 |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete never decrements the owner's quota, leaking the slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending then takes the first n, returning the LE |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring containment instead of equality |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives already-expired links, violating the stated i |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live internal dictionary, not a snapshot |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic: partial batch is committed on valid |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates links.Keys during iteration and returns remain |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-concurrency: Store.NewCode uses a shared System.Random from concurrent Cr; review-security: Short codes minted with non-cryptographic System.Random — en |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret dev-secret-do-not-use |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens use non-cryptographic System.Random — forgeab |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token written to stdout/logs on issue |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never enforces session expiry; review-security: Verify never checks session expiry — sessions live forever |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true for every role; review-security: RequireAdmin always returns true — every role is admin |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken throws IndexOutOfRangeException when no Authoriz; review-security: BearerToken throws on any request lacking a Bearer header |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals uses plain == despite constant-time claim |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | — | parallel: CA02 Peek reads the dictionary with no lock at all while loc |  |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set evicts the first-inserted key, not the least recently us |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep modifies entries.Keys while iterating, throwing Invali; review-concurrency: Cache.Sweep mutates the entries dictionary while enumerating |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization integer division always yields 0 unless the cach |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to zero instead of keeping |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS parsed as seconds, not milliseconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied to cfg.FetchLi |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal in ExportCsv/ReadReport via user-supplied nam |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: ArchiveAll path traversal via owner key |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: WriteSnapshot uses a predictable fixed temp path — symlink a |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync writes only the CSV header and ignores the li |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: StartJanitor mutates store.All().Keys during iteration, thro; review-concurrency: Janitor mutates the store dictionary while enumerating its o |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache fires Task.Run without awaiting, contradicting 're |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | — | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot |  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-bugs: Links DELETE performs no ownership check despite the comment; review-security: DELETE /links has no ownership check — any user deletes any  |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via ?next= in /out endpoint |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: AdminQuotas gates on a client-controlled X-Admin header inst; review-security: Admin endpoint gated on a spoofable X-Admin HTTP header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by store.Count, throwing DivideByZeroException; review-concurrency: Stats computes hits_per_link from two unsynchronized store r |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows all origins with credentials |
| N63 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) Config.cs:126 — Validate always returns true, never rejecting an invalid configuration
- (medium) Authenticator.cs:48 — Authenticator.sessions is read by Verify concurrently with Issue/Revoke writes, unsynchron
- (medium) Metrics.cs:17 — Metrics.AddCreated uses non-atomic increment, losing counts under concurrent writes
- (medium) Cache.cs:126 — Cache.Peek and Utilization read the entries dictionary outside the lock that protects it
- (medium) Program.cs:96 — JSON injection via unescaped user input in responses
