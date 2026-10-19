# minimax-m3.1-flash:cloud · csharp · run 20261019-200607 (repeat 3)

recall **34/68** · 44 finding(s), 4 unmatched · 444552 tokens · 2572s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | — | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep |  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry check in Resolve is inverted: live links 404, expired |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates the destin |
| N04 | — | parallel: C04 page clamps end past the list size, so GetRange throws o |  |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate truncates to a whole number before the percentag |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete removes the link but never returns the owner's quota  |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending by hits and returns the least-followed l |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches on substring instead of exact owner |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives an already-expired link, contradicting its do |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | — | parallel: C13 ImportAll stores as it validates, so a bad code leaves t |  |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns links.Count instead of the number of links it  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune removes entries while enumerating links.Keys, throwing |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret is used when LINKD_SECRET is unset |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens are generated from a non-cryptographic Random |
| N19 | — | parallel: S03 every issued session token is logged in cleartext |  |
| N20 | — | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve |  |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin allows every role and always returns true; review-security: RequireAdmin returns true on every path, so the admin role c |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords are hashed with a single unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken indexes parts[1] without checking length, throwi; review-security: BearerToken indexes past the end of the header, throwing on  |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek reads the dictionary without the lock, racing wit |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Cache.Stats and Cache.Utilization read shared state without  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set evicts an arbitrary entry rather than the least recently |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep removes entries while enumerating entries.Keys, throwi; review-concurrency: Cache.Sweep removes entries while enumerating entries.Keys,  |
| N33 | — | parallel: C05 Utilization divides before scaling so it always reports  |  |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is documented as milliseconds but convert |
| N37 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne |  |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Report name from the query string is used unsanitized in a f |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-concurrency: ExportCsvAsync is async void and races with ExportCsv on the |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated uses a plain increment while AddResolved  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: Janitor removes from store.All().Keys while enumerating it,  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache starts one Task.Run per link and never awaits them |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-bugs: CheckTargets blocks on async probes with .Result, risking de; review-concurrency: CheckTargets blocks on ProbeAsync(...).Result, tying up thre |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for an owner with  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Rejected session token is echoed into the 401 response body |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-bugs: DELETE /links does not check that the session owns the link ; review-security: DELETE /links removes any link by code with no ownership che |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out is an unconditional open redirect that echoes any attac |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on a client-supplied X-Admin heade |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by store.Count and throws DivideByZeroExceptio |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS combines a wildcard origin with Allow-Credentials: true |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Link creation takes the owner from the query string instead ; review-bugs: int.Parse on the ttl and paging query parameters throws on m |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | YES | parallel: X02 every request runs on a thread-pool thread against a Sto | review-concurrency: Request handler has no exception guard, so a background-thre |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Store.cs:20 — Store's dictionaries are mutated concurrently from request threads without a lock
- (high) Config.cs:126 — Validate always returns true, so a bad config does not fail at startup
- (critical) Store.cs:20 — Store's dictionaries are shared across request threads and the janitor with no lock
- (high) Authenticator.cs:48 — Authenticator.sessions is mutated and enumerated without synchronization
