# glm-5.3-flash:cloud · csharp · run 20260901-135618 (repeat 1)

recall **45/68** · 60 finding(s), 3 unmatched · 698458 tokens · 352s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException on an owner's first link; review-concurrency: Store's links/quota dictionaries are mutated from many reque |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve expiry condition is inverted: live links are deleted; review-security: Expiry check in Resolve is inverted: expired links keep redi |
| N03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page off-by-one makes GetRange throw on the last page |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate integer division always yields 0 |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete never returns the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least-followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring matching instead of equality |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives already-expired links |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic despite its contract |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates links.Keys during iteration and returns the wr |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: NewCode collisions silently overwrite an existing link; review-concurrency: Shared Random used from concurrent request threads can colla |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret silently used when LINKD_SECRET is |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens minted from System.Random are predictable |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens written to stdout and echoed in 401 responses |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never checks session expiry; review-security: Verify never checks ExpiresAt, so sessions never expire |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true; review-security: RequireAdmin always returns true |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken crashes on a missing or malformed Authorization ; review-security: BearerToken throws on missing/malformed Authorization header |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals is a plain == despite claiming constant-time co |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Peek and Utilization read the cache dictionary without takin |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction picks an arbitrary entry, not the least recently us |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep removes entries while iterating entries.Keys; review-concurrency: Sweep removes entries while enumerating entries.Keys |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization integer division always returns 0 (and divides b |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A malformed LINKD_TIMEOUT_MS still overwrites the timeout wi |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile throws on a malformed cache_size value |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal: report name escapes ExportDir for read and w |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: StreamWriter is not disposed if writing throws |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-security: CSV rows are not escaped, enabling formula injection |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: WriteSnapshot moves /tmp file across filesystems and uses a ; review-security: Snapshot staged at a predictable /tmp path |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void and ignores its payload; review-concurrency: ExportCsvAsync is async void: unawaitable, and its failures  |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated increments non-atomically and Snapshot re |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes from the dictionary while enumerating its ke; review-concurrency: Janitor thread removes from the live store dictionary while  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before any work is done; review-concurrency: WarmCache returns before its work finishes and the tasks are |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for an unknown own |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: Token requirement also gates the public redirect endpoints |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links has no ownership check; any user deletes anyon |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out forwards to any attacker-supplied next U |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by spoofable X-Admin request heade |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero on an empty store |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin together with credentials |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: POST /links takes owner from the query string, not the sessi |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports every owner's links with no role check |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: Unvalidated int.Parse on query parameters kills the request |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Config.cs:126 — Validate always returns true and its result is ignored
- (high) Authenticator.cs:48 — Session dictionary is read by Verify on every request while Issue/Revoke mutate it on othe
- (low) Export.cs:28 — Export report written with default permissions despite single-user claim
