# claude-opus-5 · csharp · run 20260901-160240 (repeat 1)

recall **52/68** · 71 finding(s), 6 unmatched · 47364 tokens · 548s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException on an owner's first link; review-concurrency: Store's dictionaries are mutated from concurrent request and |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve has its expiry condition inverted: live links are de |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never validates t |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page clamps the end index one past the list, so GetRange thr |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate is integer division and is always 0 or 100 |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete never returns the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least-followed links, not the most |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches owners by substring instead of equality; review-security: ByOwner matches owners by substring, leaking other owners' l |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives already-expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live internal dictionary, not the documen |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic despite its documented all-or-nothin |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates while iterating and returns the wrong count |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: NewCode can collide and silently overwrite an existing link; review-security: Short codes generated from non-cryptographic Random are enum |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret silently used when LINKD_SECRET is |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens minted from non-cryptographic System.Random |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-concurrency: Shared Random for token minting can hand two users the same ; review-security: Freshly issued session token written to stdout |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never checks ExpiresAt, so tokens never expire; review-security: Verify never checks session expiry, so tokens are valid fore |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true on both branches, admitting every ; review-security: RequireAdmin returns true for every role, granting admin to  |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords stored as unsalted single-round SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken dereferences parts[1] without checking the split; review-security: BearerToken indexes parts[1] unconditionally, crashing every |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals compares secrets with a non-constant-time opera |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | YES | CS-ONLY every method locks on `this`, which any external caller can al | review-concurrency: Cache synchronizes on `this`, exposing its lock to any exter |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Peek reads the cache dictionary outside the lock that every  |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Stats and Utilization read cache state outside the lock, ret |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction picks an arbitrary key, not the least recently used |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep removes entries while enumerating entries.Keys |
| N33 | — | parallel: C05 Utilization divides before scaling so it always reports  |  |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Failed LINKD_PORT parse overwrites the default with 0 |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS logs 'keeping default' and then sets th |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds, inflating the  |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and thrown away (parsed = parsed |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile throws FormatException on a malformed cache_size li |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Arbitrary file write via unvalidated report name in Path.Com |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: StreamWriter leaked when the write throws |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV rows are written without quoting or escaping, corrupting |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: ArchiveAll builds output paths from unvalidated owner names |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot temp file is a fixed path in /tmp, defeating the do; review-security: Snapshot written through a predictable /tmp path, enabling a |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void: failures are unobservable and ; review-concurrency: ExportCsvAsync is async void, so failures crash the process  |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated uses a non-atomic increment despite the c |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes from the store dictionary while enumerating ; review-concurrency: Janitor timer callback removes from the store dictionary whi |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before the work it launched has finished; review-concurrency: WarmCache returns before the tasks it starts have run, contr |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks on .Result, serializing the probes and r |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for an owner with  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response body echoes the presented bearer token |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-bugs: DELETE /links ignores the session owner it documents; review-security: DELETE /links has no ownership check, letting any session de |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out: next parameter is sent as Location un |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by a client-supplied X-Admin heade |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: /stats divides by store.Count and crashes on an empty store |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS combined with Allow-Credentials on every respo |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Unvalidated int.Parse on query parameters crashes the reques; review-security: Link owner taken from the query string instead of the authen |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report returns every owner's links to any authenticated cal |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | YES | parallel: C26 the accept loop never exits, so StopJanitor is unreachab | review-concurrency: Request handler tasks are never observed, so a throwing hand |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with no name writes links.csv but reads .csv back |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate always returns true, so an invalid config starts the service
- (high) Authenticator.cs:48 — Session dictionary is enumerated on every request while other threads insert and remove
- (high) Store.cs:228 — Shared Random used from concurrent handlers can mint duplicate codes and overwrite live li
- (high) Export.cs:75 — ReadReport allows reading arbitrary .csv files off the filesystem
- (medium) Program.cs:188 — Export failures return the full exception, including stack trace, to the client
- (medium) Export.cs:28 — Reports containing every owner's data are created with default permissions
