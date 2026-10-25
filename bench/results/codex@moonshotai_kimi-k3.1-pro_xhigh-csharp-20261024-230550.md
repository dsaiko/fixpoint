# codex@moonshotai/kimi-k3.1-pro#xhigh · csharp · run 20261024-230550 (repeat 1)

recall **41/68** · 60 finding(s), 11 unmatched · 896950 tokens · 569s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Creating a link for a new owner throws after insertion |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve removes live links and serves expired links |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Final or short pages request an out-of-range segment |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success percentage truncates fractional values |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not release its quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns least-followed links first |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner returns substring matches |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend resurrects expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas exposes mutable store state instead of a snapshot |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll leaves partial batches behind |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns remaining links instead of removed count |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: Generated code collisions overwrite existing links |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens use a non-cryptographic PRNG |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Live bearer tokens are written to application logs |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Expired sessions still verify successfully; review-bugs: Expired sessions accumulate indefinitely |
| N21 | — | parallel: S07 RequireAdmin returns true from both branches, so every s |  |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashes use one fast, unsalted SHA-256 operation |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: Malformed or missing Authorization headers throw |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache reads bypass the lock used by its writers |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Cache eviction is not LRU and unnecessarily evicts on replac; review-bugs: A zero-sized cache crashes on the first insertion |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Cache utilization truncates to zero below full capacity |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout input replaces the default with zero |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Millisecond cache TTL is interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is never applied |
| N38 | YES | parallel: CF05 LoadFile swallows every exception, so an unreadable con | review-bugs: Unreadable configuration files are silently treated as missi |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Report filename permits directory traversal and arbitrary CS |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: Export writers leak when writing throws; review-concurrency: Concurrent report requests write the same file and can retur |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV fields are written without escaping; review-security: User-controlled owner values enable CSV formula injection |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots overwrite and move one shared temporary; review-security: Predictable shared temporary snapshot path enables symlink o |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: Asynchronous export cannot be awaited or observed |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Created-link metric loses concurrent increments |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: Janitor enumerates and mutates the live store while request  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before warming completes; review-concurrency: WarmCache returns before its background work finishes |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for owners without links |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete another user's links |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The /out endpoint is an arbitrary open redirect |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Client-supplied X-Admin header bypasses administrative autho |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero when the store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Malformed numeric query values abort request handling |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exports other users' private link data |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default-name report export reads a different file |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Cache.cs:53 — Peek treats expired entries as present
- (high) Config.cs:126 — Configuration validation always succeeds
- (medium) Export.cs:58 — Snapshot move can fail across filesystems
- (medium) Export.cs:83 — Asynchronous export discards all supplied links
- (medium) Program.cs:96 — Creation responses can contain invalid JSON
- (medium) Store.cs:27 — URL validation accepts malformed targets
- (high) Store.cs:40 — Concurrent request handlers mutate the shared store dictionaries without synchronization
- (medium) Authenticator.cs:48 — Session verification races with session issuance and revocation
- (medium) Authenticator.cs:69 — RequireAdmin approves every role
- (medium) Program.cs:30 — Bearer authentication is served over cleartext HTTP
- (low) Program.cs:188 — Export failures disclose internal paths and stack traces
