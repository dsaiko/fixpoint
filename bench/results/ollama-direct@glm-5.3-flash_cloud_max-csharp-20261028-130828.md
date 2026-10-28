# ollama-direct@glm-5.3-flash:cloud#max · csharp · run 20261028-130828 (repeat 1)

recall **48/68** · 67 finding(s), 8 unmatched · 2041171 tokens · 862s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: First link creation crashes for a missing owner quota; review-concurrency: Store state is mutated and read by concurrent request thread |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry check is inverted in Resolve |
| N03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Last page uses an out-of-range count |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success rate is multiplied after integer division |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not release the owner quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner performs substring matching instead of exact ownersh |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend resurrects already expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas exposes mutable internal state; review-concurrency: Quotas returns the live mutable dictionary to concurrent rea |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll applies a partial batch on validation failure |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the wrong count |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates its dictionary while enumerating it |
| N14 | YES | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime | review-bugs: Link lifetime is based on local wall-clock time |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: Short-code collisions replace existing links; review-concurrency: Shared Random is used from concurrent Create calls |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-concurrency: Session dictionary and token Random lack thread synchronizat; review-security: Session tokens are predictable and logged |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Session expiry is never enforced; review-bugs: Expired/unrevoked sessions leak without bound |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always approves every session |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing is unsalted single-pass SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: Malformed Authorization headers throw |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache exposes unsynchronized reads and a non-atomic warm che |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Cache eviction is not least-recently-used |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep modifies the dictionary during enumeration |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Cache utilization is zero and can divide by zero |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Invalid LINKD_PORT silently becomes port 0 |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS overwrites default with zero |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS uses the wrong unit |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT has no effect |
| N38 | YES | parallel: CF05 LoadFile swallows every exception, so an unreadable con | review-bugs: LoadFile treats permission/read errors as missing files |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: Malformed cache_size crashes LoadFile |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: User-supplied report name allows path traversal |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: CSV writer leaks on failed write |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV fields are not escaped; review-security: Exported CSV is not quoted or sanitized |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot uses a fixed global temporary path; review-concurrency: Concurrent snapshots share a fixed temporary file path |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: Async export is fire-and-forget and discards its links; review-concurrency: Background exporter uses async void |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | YES | the errors counter is reported by Snapshot and never assigned anywhere | review-bugs: errors metric can never change |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor modifies the store while enumerating it; review-concurrency: Janitor mutates and enumerates the Store's raw dictionary wh |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache starts unbounded fire-and-forget tasks |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for an owner with no links |
| N54 | YES | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo | review-bugs: FollowedSummary repeats a deferred query |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete another owner's link |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unrestricted open redirect on /out |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts an attacker-controlled header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by the number of links |
| N61 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect mutates case-sensitive code with ToLower |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wild-card CORS is combined with credentials |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Numeric query values throw unhandled exceptions; review-security: Link owner is taken from the query string instead of the ses |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint discloses every owner's links |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report is written and read under different names |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate always reports a successful configuration
- (medium) Program.cs:96 — Target is inserted into JSON without escaping
- (medium) Metrics.cs:17 — Created counter is incremented non-atomically
- (medium) Program.cs:37 — Request handler tasks are fired without exception observation
- (medium) Store.cs:228 — Short codes are only 31 bits from an insecure RNG
- (medium) Store.cs:40 — Link quotas are recorded but never enforced
- (medium) Export.cs:28 — Sensitive exports do not force private permissions
- (medium) Program.cs:30 — Bearer tokens are served over plaintext HTTP
