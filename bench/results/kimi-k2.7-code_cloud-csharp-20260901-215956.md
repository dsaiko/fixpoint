# kimi-k2.7-code:cloud · csharp · run 20260901-215956 (repeat 1)

recall **34/68** · 49 finding(s), 7 unmatched · 185453 tokens · 486s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: First link for a new owner throws KeyNotFoundException |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve treats active links as expired; review-security: Resolve deletes live links and keeps expired ones |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice and never validates 'to'; review-security: Rename validates source code twice and never the destination |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page throws when the result set is smaller than the requeste |
| N05 | — | parallel: C05 integer division before scaling yields 0 for every rate  |  |
| N06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns least-followed links instead of most-followed |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring match instead of exact match; review-security: ByOwner leaks links across owners |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend can resurrect expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-concurrency: Quotas() returns the live dictionary instead of a snapshot |
| N11 | — | parallel: C13 ImportAll stores as it validates, so a bad code leaves t |  |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-concurrency: Store.Prune removes entries while enumerating keys |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Short codes are predictable and small |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens minted with non-cryptographic Random |
| N19 | — | parallel: S03 every issued session token is logged in cleartext |  |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Expired session tokens are still accepted; review-security: Token expiry is never enforced |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true; review-security: RequireAdmin always returns true |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with fast, unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken crashes on missing or malformed Authorization he; review-security: BearerToken throws on missing or malformed header |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals is not constant-time |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: Authenticator mutates sessions without synchronization |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek reads the dictionary without locking |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Cache.Stats and Utilization read mutable state without locki |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Cache eviction does not use least-recently-used order |
| N31 | YES | parallel: CA06 Warm's Peek and Set are separately locked, so the janit | review-concurrency: Cache.Warm has a check-then-act race |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Cache sweep mutates dictionary during enumeration; review-concurrency: Cache.Sweep removes entries while enumerating keys |
| N33 | — | parallel: C05 Utilization divides before scaling so it always reports  |  |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to zero instead of default |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| N37 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne |  |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-security: CSV formula injection in exports |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-concurrency: ExportCsvAsync is fire-and-forget async void |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor mutates store dictionary while enumerating it; review-concurrency: Janitor removes expired links directly from the shared dicti |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache spawns unawaited tasks |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks synchronously on async probes |
| N53 | — | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot |  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links does not check link ownership |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out endpoint is an open redirect |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint protected only by a spoofable header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats endpoint divides by zero when store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Unparsed TTL query value causes unhandled FormatException |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) Store.cs:20 — Store mutates shared dictionaries without synchronization
- (high) Store.cs:208 — Store.All() returns the mutable internal links dictionary
- (high) Store.cs:61 — Resolve increments Hits without synchronization
- (medium) Metrics.cs:17 — Metrics.AddCreated increments without interlocking
- (medium) Export.cs:12 — ExportCsv has a file-level race on the report name
- (high) Program.cs:183 — Report name parameter allows path traversal
- (medium) Program.cs:188 — Internal exception details leaked to clients
