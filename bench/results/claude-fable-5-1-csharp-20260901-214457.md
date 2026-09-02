# claude-fable-5-1 · csharp · run 20260901-214457 (repeat 1)

recall **51/68** · 75 finding(s), 4 unmatched · 41855 tokens · 475s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException for any owner's first lin; review-concurrency: Store's dictionaries are mutated from concurrent request han |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve has the expiry check inverted: live links are delete |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to` |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page clamps end to Count + 1, so any page that reaches the e |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate uses integer division and reports 0 unless every |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not return the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending and returns the least followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses substring match instead of equality; review-security: ByOwner matches by substring, leaking other owners' links |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links, contrary to its contract |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live dictionary, not a snapshot |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic: a bad code midway leaves a partiall |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining link count instead of the number |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: NewCode does not check for collisions; Create silently overw; review-concurrency: Shared Random instance used from concurrent Create calls can |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens are minted from System.Random; review-security: Session tokens are written to the log |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never checks ExpiresAt, so sessions never expire; review-security: Verify never checks ExpiresAt, so tokens live forever |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true on both branches; review-security: RequireAdmin returns true for every session |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with a single unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken indexes parts[1] without checking there is a sec; review-security: BearerToken throws on any malformed header, leaving the conn |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals is a plain string comparison, not constant-time |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek reads the dictionary without the lock, breaking t |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Stats and Utilization read shared state outside the lock, an |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction removes an arbitrary key, not the least recently us; review-bugs: A cache limit of 0 makes Set throw and Utilization divide by |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization uses integer division and is 0 until the cache i |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Unparseable LINKD_PORT overwrites the default with 0 |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS logs 'keeping default' but then sets Ti |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is documented as milliseconds but applied |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile uses int.Parse, so a malformed cache_size line thro |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Report name is joined into the path unsanitized: arbitrary f |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: StreamWriter handles leak when Write throws |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV rows are built by string interpolation with no quoting; review-concurrency: Concurrent /report requests with the same name race on one f |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: ArchiveAll builds file names from the owner string |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: WriteSnapshot is neither atomic nor durable as documented; review-concurrency: WriteSnapshot uses one fixed temp path, so concurrent snapsh |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void and writes only the header; review-concurrency: ExportCsvAsync is async void: it cannot be awaited and any f |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-bugs: Metrics.errors is never incremented; review-concurrency: Metrics.AddCreated is a plain increment while the class clai |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: Janitor timer mutates the live store dictionary from another |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before any work is done despite promising ; review-concurrency: WarmCache fires Task.Run per link and returns immediately de |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks on .Result inside a loop (sync-over-asyn |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for an owner with no links |
| N54 | YES | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo | review-bugs: FollowedSummary enumerates a lazy query twice |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-bugs: No code path ever calls Authenticator.Issue, so every reques |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links deletes any link with no ownership check |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out is an unvalidated open redirect |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on the client-supplied X-Admin hea |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: /stats divides by store.Count and crashes on an empty store |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS origin combined with Allow-Credentials on an a |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Query integers are parsed with int.Parse and crash the handl; review-security: Link owner is taken from the query string, not the session |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports every owner's links to any authenticated use |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | YES | parallel: C26 the accept loop never exits, so StopJanitor is unreachab | review-bugs: Handle has no exception boundary, so any handler exception l; review-concurrency: Fire-and-forget Task.Run(Handle) swallows exceptions and lea |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report reads back by the raw name, ignoring the path ExportC; review-bugs: Report catches only IOException; permission errors escape |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate collects problems and then returns true regardless
- (medium) Export.cs:28 — Reports created with default permissions despite the owner-only claim
- (medium) Program.cs:188 — 500 response dumps the full exception and server paths
- (low) Program.cs:96 — JSON response assembled by interpolation with unescaped user input
