# claude-fable-5 · csharp · run 20260901-161501 (repeat 1)

recall **36/68** · 43 finding(s), 3 unmatched · 35194 tokens · 429s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException on an owner's first link |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve expiry check is inverted: live links are deleted, ex |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice and never validates 'to' |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page clamps end to Count + 1, making GetRange throw on the l |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate uses integer division and reports 0 unless every |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete never returns the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending, returning the least followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring match instead of equality |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links despite the documented contract |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live internal dictionary despite the docu |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic: a mid-batch validation failure leav |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the surviving link count, not the number remov |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-concurrency: Shared Random instance called from concurrent threads corrup |
| N19 | — | parallel: S03 every issued session token is logged in cleartext |  |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never checks ExpiresAt, so sessions never expire |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true for every session |
| N22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken indexes parts[1] without checking, crashing on m |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | YES | CS-ONLY every method locks on `this`, which any external caller can al | review-concurrency: Cache locks on this, a publicly reachable object |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek, Stats, and Utilization read shared state outside |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set evicts an arbitrary entry, not the LRU, and throws when  |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization integer division always yields 0 below full, and |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Failed LINKD_PORT parse zeroes the port default |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Failed LINKD_TIMEOUT_MS parse sets Timeout to zero instead o |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is applied as seconds, not milliseconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile crashes on malformed cache_size and swallows all re |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: ExportCsv and ArchiveAll leak the StreamWriter when a write  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: WriteSnapshot stages in /tmp, so the rename is not atomic ac; review-concurrency: WriteSnapshot uses one fixed temp path shared by all concurr |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void and writes only the header, dis; review-concurrency: ExportCsvAsync is async void: unawaitable and crashes the pr |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-bugs: Metrics.errors is never incremented, so /stats always report; review-concurrency: Metrics.AddCreated uses non-atomic created++ despite documen |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: Janitor timer mutates the store's internal dictionary concur |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before its work finishes despite the docum; review-concurrency: WarmCache fires unawaited Task.Run tasks but documents that  |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks synchronously on ProbeAsync(...).Result |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for owners with no |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| N58 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| N59 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: /stats divides by zero on an empty store |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Unvalidated int.Parse on ttl/offset/size crashes the handler |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report reads back the raw name after ExportCsv defaulted an  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Config.cs:126 — Validate always returns true, so invalid configs pass startup
- (critical) Store.cs:20 — Store's dictionaries are shared mutable state with zero synchronization
- (high) Authenticator.cs:48 — Session dictionary read on every request while Issue/Revoke mutate it, unsynchronized
