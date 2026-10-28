# ollama-codex@glm-5.3-flash:cloud#max · csharp · run 20261028-130456 (repeat 1)

recall **41/68** · 59 finding(s), 6 unmatched · 507375 tokens · 492s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | — | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep |  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiration condition is inverted in Resolve; review-security: Link expiry test is inverted |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice, never the destination |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page calculates an out-of-range range for the last page |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate divides before multiplying |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not restore owner quota |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring matching |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend can revive expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live internal dictionary |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic and silently overwrites keys |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates keys while iterating and returns the wrong val |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: Generated codes are not collision-safe; review-security: Short codes are weak, collision-prone and predictable |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens are generated with a non-cryptographic PRNG |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Authentication tokens are written to logs |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Expired sessions are still accepted; review-security: Expired sessions remain valid |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-security: RequireAdmin always authorizes the caller |
| N22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: Missing Authorization header causes an index-out-of-range ex; review-security: Malformed Authorization header causes unhandled exception |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: Session registry is accessed concurrently without synchroniz |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Some cache reads bypass the cache lock |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction is not least-recently-used |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep mutates the dictionary while enumerating it |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization can divide by zero and otherwise reports only 0  |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A bad timeout value replaces the default with zero |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Milliseconds are parsed as seconds for the cache TTL |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: Fetch limit is parsed but never applied |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: User-controlled export filename permits path traversal and f |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV fields are written without escaping; review-security: Unescaped user-controlled CSV allows formula injection |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Predictable world-access temporary snapshot path is unsafe |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: Background export is async void and writes no data; review-concurrency: Fire-and-forget async void export has no completion or excep |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Created metric is not thread-safe despite the documented con |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor mutates the store dictionary while enumerating it; review-concurrency: Janitor mutates the live dictionary while iterating it |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache does not await its work; review-concurrency: WarmCache returns before its background tasks finish |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for any owner without links |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-bugs: Delete ignores authenticated ownership; review-security: Any authenticated user can delete any owner's link |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via user-controlled next target |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin authorization trusts the X-Admin header; review-security: Admin endpoint trusts a client-controlled header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero for an empty store |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Insecure permissive CORS configuration |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Link owner is taken from the query string instead of the ses; review-security: Link owner is taken from the client instead of the session |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint leaks every owner's links to any session |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: Numeric query values are parsed without error handling |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Authenticator.cs:65 — RequireAdmin always allows
- (medium) Config.cs:126 — Validation never refuses invalid configuration
- (low) Export.cs:42 — Archive writers are not disposed on write failure
- (high) Program.cs:55 — Public redirect routes require authentication
- (medium) Program.cs:96 — Link target is interpolated without JSON escaping
- (critical) Store.cs:20 — Link store mutates shared dictionaries and link objects without synchronization
