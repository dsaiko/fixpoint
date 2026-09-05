# claude-sonnet-5 · csharp · run 20260905-160123 (repeat 1)

recall **35/68** · 43 finding(s), 4 unmatched · 64408 tokens · 620s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create() throws KeyNotFoundException for every owner's first |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve() has inverted expiry check: deletes valid links, se |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename() validates 'from' twice and never validates 'to' |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page() off-by-one crashes GetRange whenever a page runs past |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate() truncates the same way Utilization() does |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete() never restores the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top() returns the least-followed links instead of the most-f |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner() matches owner as a substring instead of exact owne |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend() can revive an already-expired link, contradicting i |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | — | parallel: C13 ImportAll stores as it validates, so a bad code leaves t |  |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune() mutates the links dictionary while iterating its Key |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Short link codes are predictable |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens are generated with a non-cryptographic RNG |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Freshly issued session token is written to the console log |
| N20 | — | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve |  |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true regardless of role |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted single-round SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken crashes on any request without a well-formed 'Sc; review-security: Malformed or missing Authorization header crashes request ha |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | YES | the sessions dictionary is read and written from every request task wi | review-concurrency: Session dictionary accessed without synchronization from con |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Peek() reads the entries dictionary without the lock used by |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | — | parallel: CA08 eviction drops whatever key comes first, not the least  |  |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-concurrency: Sweep() removes entries while enumerating the same Keys coll |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization() truncates to 0% for any non-full cache, and di |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets Timeout to zero instead of keeping |
| N36 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu |  |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal via the user-supplied report name |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Snapshot temp file uses a fixed, predictable path in a world |
| N47 | — | CS-ONLY async void: the caller cannot await it and an exception inside |  |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated increments a shared counter without synch |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes dictionary keys while enumerating them, cras; review-concurrency: Janitor timer removes dictionary entries while enumerating t |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache fires unawaited tasks and its doc comment misstate |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for an owner with  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete any other owner's link (ID |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on a client-supplied header instea |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats() divides by zero when the store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS response combines wildcard origin with allow-credential |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Malformed ttl/offset/size query parameters crash the request; review-security: Link owner is taken from the request, not the authenticated  |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exposes every owner's links to any authenticated cal |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate() always returns true even when it collected problems
- (low) Export.cs:83 — ExportCsvAsync ignores the links parameter and always writes only a header row
- (critical) Store.cs:20 — Store's dictionaries are mutated from multiple threads with no synchronization
- (medium) Program.cs:188 — Export failures leak exception details to the client
