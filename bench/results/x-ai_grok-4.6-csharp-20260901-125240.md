# x-ai/grok-4.6 · csharp · run 20260901-125240 (repeat 1)

recall **37/68** · 44 finding(s), 5 unmatched · 179081 tokens · 884s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Creating the first link for an owner throws KeyNotFoundExcep |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve treats live links as expired and serves expired ones |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never the destina |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page always throws when the requested range past the end |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate truncates to 0% or 100% via integer division |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not return the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top returns the least-followed links, ascending |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches substrings instead of the exact owner |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives already-expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live dictionary, not a snapshot |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns remaining size instead of the number removed |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune mutates the links dictionary while enumerating keys |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-security: Short codes are generated with System.Random |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens are minted with System.Random |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Fresh session tokens are written to stdout |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never rejects an expired session; review-security: Verify never enforces session expiry |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true |
| N22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken throws on a missing or non-Bearer Authorization  |
| N24 | — | SecretEquals is documented as not leaking how much matched and uses == |  |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | — | parallel: CA02 Peek reads the dictionary with no lock at all while loc |  |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set evicts insertion order, not the least recently used entr |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep mutates the dictionary while enumerating its keys |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization is 0 until the cache is completely full |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A bad LINKD_TIMEOUT_MS still overwrites the default with zer |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is applied as seconds, not milliseconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: User-controlled export name is not confined to the export di |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-security: CSV export does not escape fields; review-security: Export files are created with default (world-readable) permi |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| N47 | — | CS-ONLY async void: the caller cannot await it and an exception inside |  |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.created is incremented without Interlocked despite c |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes store entries while iterating the live dicti |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before the warmup tasks finish |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | — | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot |  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 responses echo the presented bearer token |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links does not check the authenticated owner |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated open redirect on /out |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quotas gated by a client-supplied X-Admin header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by zero when the store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link owner is taken from the query string, not the session |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report dumps every owner's links to the caller |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate always reports success
- (high) Export.cs:83 — ExportCsvAsync writes a header and drops every link
- (medium) Program.cs:20 — Main ignores Validate and starts anyway
- (high) Store.cs:20 — Store dictionaries are mutated from request threads and the janitor with no lock
- (high) Store.cs:22 — System.Random in Store is used concurrently from Create
