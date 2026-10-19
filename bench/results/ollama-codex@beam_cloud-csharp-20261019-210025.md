# ollama-codex@beam:cloud · csharp · run 20261019-210025 (repeat 2)

recall **48/68** · 73 finding(s), 10 unmatched · 1259573 tokens · 159s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException for new owners |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve expiry condition is inverted |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates from code twice and never validates to; review-security: Rename allows an unvalidated target code |
| N04 | — | parallel: C04 page clamps end past the list size, so GetRange throws o |  |
| N05 | — | parallel: C05 integer division before scaling yields 0 for every rate  |  |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not refund owner quota |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns least-followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches by substring rather than equality; review-security: ByOwner uses substring match instead of equality |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links; review-security: Extend resurrects expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live backing dictionary; review-security: Quotas() leaks the mutable quota dictionary |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic; review-security: ImportAll injects links without validation or atomicity |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates dictionary while enumerating and returns wrong; review-concurrency: Store.Prune enumerates Keys while removing entries in the sa |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Short codes generated with predictable System.Random |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback session secret |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens written to the console log |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify ignores session expiry; review-security: Verify ignores token expiry |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true; review-security: RequireAdmin always returns true |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken does not handle malformed headers; review-security: BearerToken crashes on malformed Authorization header |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-bugs: SecretEquals is not constant-time; review-security: SecretEquals is not constant-time |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: Sessions dictionary and shared Random used across request th |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek reads the backing dictionary without holding the  |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Cache.Stats and Utilization read shared state without synchr |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Cache eviction is not LRU |
| N31 | YES | parallel: CA06 Warm's Peek and Set are separately locked, so the janit | review-bugs: Warm prefetch is not synchronized under the cache lock; review-concurrency: Cache.Warm performs unlocked check-then-act (TOCTOU) across  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep mutates dictionary while enumerating; review-concurrency: Cache.Sweep mutates the dictionary while enumerating its Key |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization integer division yields 0 until full |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets a zero timeout |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL treated as seconds despite millisecond units |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | YES | parallel: CF06 Validate logs the problems it finds and returns true, s | review-bugs: Validate always returns true |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile can throw on non-numeric cache_size |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal in ExportCsv via report name |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-security: CSV formula injection in exported reports |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: Owner key used unsanitized as filename in ArchiveAll |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: WriteSnapshot uses a fixed temp path shared by concurrent ca; review-security: WriteSnapshot uses predictable shared /tmp path |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is incomplete and async void; review-concurrency: ExportCsvAsync is async void (fire-and-forget) |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | YES | the errors counter is reported by Snapshot and never assigned anywhere | review-concurrency: Metrics counters are non-atomic and Snapshot reads them with |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor mutates dictionary while enumerating |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache does not await probes; review-concurrency: WarmCache fires Task.Run per link and never awaits completio |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks on ProbeAsync(...).Result |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws on unknown owner |
| N54 | YES | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo | review-bugs: FollowedSummary enumerates the deferred query twice |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-bugs: BearerToken crashes on malformed or missing Authorization he |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown token echoed back in 401 response |
| N57 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via next parameter |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts client-supplied X-Admin header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero on an empty store |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS combined with credentials |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link owner and deletion identity are attacker-controlled |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Store.cs:83 — Page end index clamps to Count+1
- (medium) Store.cs:102 — SuccessRate integer division collapses partial values
- (medium) Store.cs:208 — All exposes the internal links dictionary
- (high) Store.cs:20 — Store links/quota/Random are unsynchronized across HTTP and janitor threads
- (high) Store.cs:208 — Store.All returns the live internal dictionary; Followed returns a deferred query over it
- (high) Authenticator.cs:19 — Session tokens minted with System.Random
- (high) Export.cs:75 — Path traversal in ReadReport via report name
- (medium) Store.cs:208 — All() leaks the internal mutable link dictionary
- (medium) Config.cs:126 — Validate always reports success
- (low) Config.cs:60 — Export directory taken from unvalidated env/config
