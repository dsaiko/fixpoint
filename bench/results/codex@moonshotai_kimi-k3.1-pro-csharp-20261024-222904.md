# codex@moonshotai/kimi-k3.1-pro · csharp · run 20261024-222904 (repeat 1)

recall **40/68** · 60 finding(s), 11 unmatched · 851506 tokens · 598s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Creating a link for a new owner throws; review-concurrency: Concurrent requests mutate and enumerate the links dictionar |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve deletes live links and resolves expired ones; review-security: Reversed expiry check deletes live links and serves expired  |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Pages extending past the final item request an out-of-range  |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success rate computes zero for partially followed stores |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link never releases its quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns least-followed links first |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner returns substring matches instead of exact ownership |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas exposes mutable internal state as a snapshot |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll partially applies invalid batches |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns remaining links instead of removed count |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens use predictable randomness |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Live bearer tokens are written to logs |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify accepts sessions after their expiry; review-security: Session expiration is never enforced |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-security: RequireAdmin approves every role |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords use a fast unsalted hash |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: Malformed or missing Authorization headers crash request han; review-security: Malformed Authorization headers cause unhandled request fail |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Unlocked cache readers race locked dictionary writers |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Updating an existing cache entry evicts another entry; review-bugs: Cache eviction does not select the least recently used entry |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Zero-sized cache causes division by zero; review-bugs: Cache utilization truncates nearly all partial percentages t |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout input overwrites the default with zero |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL environment value uses the wrong units |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: Fetch limit setting is parsed but never applied |
| N38 | YES | parallel: CF05 LoadFile swallows every exception, so an unreadable con | review-bugs: All configuration file read errors are silently treated as m |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV fields are written without escaping; review-security: Attacker-controlled CSV fields permit formula injection |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots share one temporary file |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-concurrency: Async void exporter hides completion and failures |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Created metric loses concurrent increments |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | — | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi |  |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before cache warming finishes; review-concurrency: WarmCache returns while its fire-and-forget work is still ru |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: Synchronous Result can deadlock ProbeAsync |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for owners with no links |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete any link |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The /out endpoint is an open redirect |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Client-supplied header bypasses admin authorization |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero when the store is empty; review-concurrency: Stats divides by a store count from a different snapshot |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link creation trusts an attacker-selected owner |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Every user can export every owner's links |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: Malformed numeric query values escape as unhandled exception |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report export writes and reads different filenames |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Config.cs:126 — Configuration validation always reports success
- (medium) Export.cs:28 — Default report directory is never created
- (medium) Export.cs:83 — Asynchronous export discards all supplied links
- (high) Store.cs:60 — Concurrent redirects lose hit increments
- (high) Store.cs:41 — Quota updates and reads race across requests
- (medium) Authenticator.cs:48 — Session verification enumerates a concurrently mutable dictionary
- (high) Program.cs:183 — Concurrent reports with the same name overwrite one another
- (medium) Export.cs:69 — Concurrent audit appends can fail or interfere
- (critical) Program.cs:180 — Report name permits arbitrary file read and overwrite
- (medium) Program.cs:30 — Bearer authentication is exposed over cleartext HTTP
- (low) Program.cs:188 — Export errors disclose local paths and internals
