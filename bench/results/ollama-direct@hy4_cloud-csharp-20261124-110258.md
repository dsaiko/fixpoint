# ollama-direct@hy4:cloud · csharp · run 20261124-110258 (repeat 1)

recall **48/68** · 77 finding(s), 13 unmatched · 811956 tokens · 1170s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Creating the first link for an owner throws after inserting ; review-concurrency: Store's dictionaries are written and enumerated by concurren |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve deletes live links and serves expired ones; review-security: Store.Resolve has the expiry comparison inverted |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page requests extending past the end request one element too; review-bugs: Page does not guard negative or overflowing pagination argum |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate is zero unless every link has been followed |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not return the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least-followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner returns partial owner matches; review-security: ByOwner leaks links belonging to other owners |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links and can shorten live ones; review-security: Extend can revive an expired link |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live quota dictionary instead of a snapsh; review-security: Quotas exposes the live mutable dictionary |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining link count instead of the number |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-concurrency: Shared System.Random in NewCode is invoked concurrently, cor; review-security: Short codes use a predictable PRNG |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret is used when LINKD_SECRET is absent |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens use a non-cryptographic PRNG |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Full session tokens are written to stdout |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify accepts expired sessions forever; review-security: Verify ignores session expiry |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin approves every role; review-security: RequireAdmin always returns true |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken crashes on requests without a two-part Authoriza; review-security: BearerToken crashes on malformed Authorization headers |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals is not constant-time |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek (and Warm's check-then-act) read entries without  |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Cache eviction ignores recency and evicts on updates |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization is truncated to 0 or 100 and can divide by zero |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Invalid LINKD_PORT overwrites the default with port 0 |
| N35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Invalid LINKD_TIMEOUT_MS replaces the default with zero; review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never assigned |
| N38 | YES | parallel: CF05 LoadFile swallows every exception, so an unreadable con | review-bugs: LoadFile silently swallows every file read error |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: A malformed cache_size in the config file crashes LoadFile |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Unvalidated report name allows path traversal and arbitrary  |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: Export writers leak file handles when Write fails; review-security: Report files are created with default permissions |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV fields are not escaped; review-concurrency: Export writes/reads a file keyed only by the user name with  |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: ArchiveAll uses owner names directly as file names |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: WriteSnapshot uses a predictable /tmp file and follows symli |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: Asynchronous CSV export writes only a header |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated uses a non-atomic increment while the cla |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: Janitor enumerates and removes from the raw store dictionary |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before its work has completed; review-concurrency: WarmCache starts a Task.Run per link but never awaits or obs |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for an owner with no links |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links lets any user delete any link |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The /out endpoint is an open redirect |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts a client-supplied X-Admin header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero on an empty store |
| N61 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect lowercases codes that the store treats case-sensiti |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS with credentials is sent on every response |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Invalid ttl query throws outside the request error handler; review-security: Link owner is taken from user input instead of the authentic |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: The /report endpoint exposes every owner's links |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: Links GET parses pagination without validation |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Program.cs:24 — The HTTP server has no path that can mint a session
- (high) Config.cs:126 — Validate reports success for every invalid configuration
- (high) Export.cs:75 — ReadReport cannot read a report written with the default name
- (medium) Config.cs:114 — Validate accepts CacheSize zero
- (medium) Program.cs:20 — Main ignores the result of config validation
- (medium) Store.cs:208 — All exposes the internal links dictionary
- (medium) Cache.cs:23 — Cache accepts nonpositive limits
- (medium) Cache.cs:53 — Peek treats expired entries as present
- (medium) Worker.cs:38 — Janitor removes expired links without updating quota
- (medium) Cache.cs:126 — Cache.Utilization/Stats read shared fields without the lock or a memory barrier
- (high) Program.cs:96 — JSON response is built by string interpolation
- (medium) Program.cs:188 — IOException details are returned to clients
- (medium) Store.cs:27 — Target URLs are not actually validated as URLs
