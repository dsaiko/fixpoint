# ollama-direct@mimo-v2.6-flash:cloud · csharp · run 20261125-142428 (repeat 1)

recall **34/68** · 51 finding(s), 10 unmatched · 234465 tokens · 316s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | — | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep |  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiry condition is inverted; review-security: Link expiry check is inverted, deleting live links and retai |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page clamping creates an out-of-range range |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success rate is calculated with integer truncation |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not restore its quota count |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least-followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring matching |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates during enumeration and reports the wrong count |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Raw bearer tokens are written to operational logs |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Expired sessions are never rejected; review-security: Session expiration is never enforced by Verify |
| N21 | — | parallel: S07 RequireAdmin returns true from both branches, so every s |  |
| N22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: Bearer-token parsing assumes two header tokens; review-security: Malformed or absent Authorization headers fault the request  |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache readers bypass the lock used by writers |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction ignores the recorded least-recent-use time |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Cache sweep mutates during enumeration |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization truncates to zero and can divide by zero |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout overwrites the documented default |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL environment value is interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: Configured fetch limit is never applied |
| N38 | YES | parallel: CF05 LoadFile swallows every exception, so an unreadable con | review-bugs: LoadFile swallows every file-read failure |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: User-controlled report names allow path traversal and file o |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: CSV writers are not disposed on write failure; review-concurrency: Concurrent report requests race on the same output file |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV fields are written without escaping |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| N47 | — | CS-ONLY async void: the caller cannot await it and an exception inside |  |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-bugs: Dashboard error count can never change |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes dictionary entries while enumerating them |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before warming finishes; review-concurrency: WarmCache starts unobserved work and returns before warming  |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for an unknown owner |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Link deletion ignores the authenticated owner |
| N58 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts a client-controlled X-Admin heade |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by an empty store |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Malformed query parameters escape request handling; review-security: Authenticated callers can spoof the owner assigned to new li |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-bugs: Report generation silently truncates large stores; review-security: Report endpoint exposes every tenant's links to any authenti |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report name is not normalized before reading |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Config.cs:126 — Validation never rejects invalid configuration
- (medium) Export.cs:83 — Async export writes only the header
- (medium) Worker.cs:54 — Target probing rejects valid HTTP links
- (medium) Program.cs:128 — The configured resolver cache is never consulted
- (medium) Store.cs:40 — Generated codes can overwrite existing links
- (medium) Program.cs:186 — Report handler does not catch permission failures
- (critical) Store.cs:20 — Store dictionaries and link counters are mutated without synchronization
- (medium) Metrics.cs:17 — Concreated counter increments can be lost
- (medium) Authenticator.cs:37 — Session issuance, verification, and revocation race on an unprotected dictionary
- (high) Program.cs:30 — Bearer tokens are transported over cleartext HTTP on all interfaces
