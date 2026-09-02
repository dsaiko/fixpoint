# qwen3.5:397b-cloud · csharp · run 20260902-111056 (repeat 1)

recall **26/68** · 33 finding(s), 6 unmatched · 280878 tokens · 140s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Quota increment throws for new owners |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve removes valid links and returns null for expired one |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' parameter twice, never validates 'to |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page method off-by-one error causes ArgumentOutOfRangeExcept |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate returns 0 due to integer division |
| N06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| N07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring match instead of equality |
| N09 | — | parallel: C12 Extend re-dates an already-expired link, resurrecting a  |  |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | — | parallel: C13 ImportAll stores as it validates, so a bad code leaves t |  |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns remaining count instead of removed count |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Predictable short code generation |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret key |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Predictable session token generation |
| N19 | — | parallel: S03 every issued session token is logged in cleartext |  |
| N20 | — | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve |  |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true bypassing authorization; review-security: Admin authorization check always passes |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Insecure password hashing |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken crashes on malformed Authorization header |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: Timing-vulnerable secret comparison |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: System.Random instance is not thread-safe |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | — | parallel: CA02 Peek reads the dictionary with no lock at all while loc |  |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: LRU eviction picks arbitrary key instead of least recently u |
| N31 | YES | parallel: CA06 Warm's Peek and Set are separately locked, so the janit | review-concurrency: Cache.Warm() has check-then-act race condition |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization returns 0 due to integer division order |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL uses wrong time unit |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: FetchLimit never set from environment variable |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: ExportCsv may leak file handle on exception |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| N47 | — | CS-ONLY async void: the caller cannot await it and an exception inside |  |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: Janitor timer callback modifies collection while iterating |
| N51 | — | parallel: X03 WarmCache starts a Task per link and awaits none of them |  |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: Blocking on async with .Result can deadlock |
| N53 | — | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot |  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect vulnerability in /out endpoint |
| N59 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| N60 | — | parallel: C20 /stats divides by the link count, throwing DivideByZeroE |  |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Invalid CORS configuration |
| N63 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | YES | parallel: C26 the accept loop never exits, so StopJanitor is unreachab | review-concurrency: Request handling loop has no cancellation or graceful shutdo |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Config.cs:126 — Validate always returns true even when problems detected
- (low) Export.cs:42 — ArchiveAll may leak file handles on exception
- (critical) Store.cs:208 — Store.All() exposes internal dictionary without synchronization
- (critical) Store.cs:20 — Store class has no synchronization on shared mutable state
- (low) Metrics.cs:17 — AddCreated uses plain increment instead of Interlocked
- (high) Export.cs:75 — Path traversal in ReadReport
