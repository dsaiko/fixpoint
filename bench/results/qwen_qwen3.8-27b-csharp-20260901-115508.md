# qwen/qwen3.8-27b · csharp · run 20260901-115508 (repeat 1)

recall **43/68** · 59 finding(s), 6 unmatched · 226137 tokens · 1616s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | — | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep |  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve inverts the expiry check: valid links deleted, expir; review-security: Expiry check inverted: expired links resolve forever, live l |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice and never validates 'to' |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page clamps end to count+1, so any last-page read throws |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate truncates to 0% via integer division |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete never returns the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending, returning the least-followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches by substring |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend resurrects already-expired links |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining count, not the number removed |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune removes entries while enumerating the dictionary |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Link codes from System.Random: 31-bit, predictable, no colli |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens generated with System.Random: predictable, fo |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens are printed to stdout on every issue |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never checks session expiry |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true for every role; review-security: RequireAdmin unconditionally returns true |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: HashPassword is unsalted, single-round SHA-256 — not a passw |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken indexes parts[1] without checking header shape; review-security: BearerToken dereferences parts[1] unguarded: every request w |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals compares secrets with == despite claiming const |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | — | parallel: CA02 Peek reads the dictionary with no lock at all while loc |  |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Cache.Utilization (and Peek) read entries without the monito |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set evicts by insertion order, not least-recently-used |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep removes entries while enumerating the cache |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization truncates to 0% via integer division; divides by; review-security: Utilization divides by limit, which Validate() permits to be |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Failed LINKD_PORT parse clobbers the default with 0 |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS zeroes the timeout while logging 'keepi |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is read as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is a no-op |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-bugs: User-supplied report name is not confined to dir; review-security: User-controlled report name is used in a path with no confin |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-security: CSV rows built by raw interpolation: row forging and spreads |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: WriteSnapshot uses a fixed, predictable /tmp path and world- |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync writes only the header and ignores links |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated uses a non-atomic created++ and Snapshot  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes links while enumerating the live dictionary,; review-concurrency: Janitor Timer mutates the raw store dictionary on a backgrou |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before the cache is primed; review-concurrency: WarmCache fires unawaited Task.Run per link and returns imme |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | — | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot |  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: Every endpoint, including the public redirect /l, requires a |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-bugs: DELETE /links never checks the owning session; review-security: DELETE /links has no ownership check: any token can delete a |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out with a cached 301 |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by a client-controlled X-Admin hea |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero when the store is empty; review-security: /stats divides by store.Count: DivideByZeroException on an e |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS wildcard origin combined with Access-Control-Allow-Cred |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Malformed ttl/offset/size fault the handler outside any catc |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report streams every owner's links to any valid token |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Config.cs:126 — Validate always returns true
- (critical) Store.cs:20 — Store.links (and quota) is an unsynchronized Dictionary shared by all request handlers and
- (medium) Export.cs:28 — Report files are created world-readable despite the comment claiming service-user-only acc
- (medium) Config.cs:126 — Validate() always returns true and its result is discarded, so bad deploys never fail at s
- (medium) Store.cs:208 — Store exposes its live internal dictionaries (All(), Quotas()), violating its own encapsul
- (low) Program.cs:188 — 500 response body interpolates the full exception, leaking internals to the client
