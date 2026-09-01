# minimax-m3:cloud · csharp · run 20260901-102841 (repeat 1)

recall **31/68** · 42 finding(s), 8 unmatched · 865182 tokens · 663s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException on a fresh owner |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve expiry check is inverted |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice and never validates 'to' |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page overshoots end by one and can throw on overflow pages |
| N05 | — | parallel: C05 integer division before scaling yields 0 for every rate  |  |
| N06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| N07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| N08 | — | parallel: C11 ByOwner matches owners by substring, so one owner sees a |  |
| N09 | — | parallel: C12 Extend re-dates an already-expired link, resurrecting a  |  |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | — | parallel: C13 ImportAll stores as it validates, so a bad code leaves t |  |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates dictionary during iteration and returns wrong  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback when `LINKD_SECRET` is unset; the secret  |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | — | parallel: S03 every issued session token is logged in cleartext |  |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify ignores ExpiresAt |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true |
| N22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken throws on malformed Authorization header; review-security: `BearerToken` throws `IndexOutOfRangeException` on an Author |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: Authenticator.sessions is mutated and iterated from multiple |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek, Utilization, and Stats read entries/hits outside |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set eviction picks an arbitrary entry, not the least-recentl |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep mutates entries while iterating entries.Keys |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization uses integer division in the wrong order |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS applied as seconds, not milliseconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT parsed but discarded |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal: user-controlled `name` flows into Path.Combi |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Snapshot write to a predictable `/tmp/linkd-snapshot.json` p |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void; review-concurrency: ExportCsvAsync is async void and never awaited |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | YES | the errors counter is reported by Snapshot and never assigned anywhere | review-concurrency: Metrics.AddCreated uses non-atomic ++ on a long field; Snaps |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor mutates store.All() while iterating store.All().Keys; review-concurrency: Janitor mutates Store.All() while handlers mutate Store conc |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache fires unbounded Task.Run with no awaiting or backp |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks on ProbeAsync(...).Result — sync-over-as |
| N53 | — | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot |  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Bearer token echoed back in the 401 response body |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: `DELETE /links?code=...` does not check the link's owner aga |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: `/out` redirects to any URL from an attacker- |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: `/admin/quotas` is gated by a client-controlled header, not  |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by store.Count unconditionally |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: `Access-Control-Allow-Origin: *` paired with `Access-Control |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Uncaught `int.Parse` on user-supplied `ttl`, `offset`, and ` |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | YES | parallel: X02 every request runs on a thread-pool thread against a Sto | review-concurrency: Main loop spawns unbounded Task.Run per request with no conc |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate always returns true
- (high) Store.cs:31 — Create does not retry on code collision
- (critical) Store.cs:20 — Store is shared mutable state with no synchronization; every handler races on it
- (high) Store.cs:22 — Random is shared and not thread-safe; NewCode can return collisions
- (low) Program.cs:192 — HttpListenerResponse is not closed in a finally; partial writes leak sockets
- (high) Authenticator.cs:19 — Session tokens and short codes are generated with non-cryptographic `System.Random`
- (medium) Authenticator.cs:62 — `RequireAdmin` returns `true` for every role, including unknown ones
- (medium) Program.cs:81 — `POST /links` accepts a target URL with no host or destination validation
