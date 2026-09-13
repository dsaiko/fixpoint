# deepseek-v4.1-flash:cloud · csharp · run 20260913-223242 (repeat 1)

recall **41/68** · 51 finding(s), 5 unmatched · 169896 tokens · 275s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | — | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep |  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve() has its expiry test inverted: live links are delet |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename() validates `from` twice and never validates `to` |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page() clamps the end index one past the list, so GetRange t |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate() does integer division and reports 0 or 100, ne |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete() never returns the owner's quota slot, so the docume |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top() sorts ascending and returns the least-followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner() matches a substring instead of the owner, leaking  |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend() revives expired links, contradicting its own stated |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-concurrency: Quotas() hands out the live quota dictionary while Create mu |
| N11 | — | parallel: C13 ImportAll stores as it validates, so a bad code leaves t |  |
| N12 | — | parallel: C15 Prune counts removals into `removed` and then returns th |  |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: Prune() mutates the dictionary while enumerating it, and ret |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | — | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  |  |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback credential when LINKD_SECRET is unset |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens minted from System.Random are predictable |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens written to the process log |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify() never checks ExpiresAt, so sessions never expire; review-security: Verify() never checks ExpiresAt, so sessions never expire |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-security: RequireAdmin returns true unconditionally — the authorizatio |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords stored as unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken() indexes parts[1] unguarded, so a missing Autho; review-security: BearerToken indexes parts[1] unguarded, crashing every unaut |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals uses == despite a constant-time contract |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek reads the dictionary outside the lock that guards |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Cache.Stats and Utilization read lock-guarded state without  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set() evicts the first-inserted entry, not the least recentl |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep() removes entries while enumerating entries.Keys; review-concurrency: Cache.Sweep removes entries while enumerating entries.Keys,  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization() integer-divides and can divide by zero when th |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A malformed duration is logged as 'keeping default' but then |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is converted as seconds, 1000x too long |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded by a self-ass |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal in export name: arbitrary file write and read |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: WriteSnapshot uses one fixed temp path for all callers; review-security: Snapshot written through a fixed, predictable /tmp path |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is `async void`, so a failed write crashes th |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated is a non-atomic increment despite the cla |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes from the dictionary while enumerating its Ke; review-concurrency: Janitor enumerates the store's live dictionary and removes f |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache starts tasks and never awaits them, contradicting  |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks a thread-pool thread on .Result inside a |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for any owner with |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: User-controlled values interpolated into response bodies wit |
| N57 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unvalidated ?next= in /out |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization decided by a client-supplied header |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by store.Count with no empty-store guard |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS set to wildcard together with Allow-Credentials |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Unvalidated query parameters parsed with int.Parse abort req |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-concurrency: /report writes and reads the same user-named file, so concur |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Config.cs:126 — Validate() always returns true, so a bad configuration never fails startup
- (critical) Store.cs:20 — Store's dictionaries are shared across thread-pool threads with no synchronization at all
- (medium) Authenticator.cs:48 — Authenticator.Verify enumerates the sessions dictionary with no lock
- (high) Store.cs:116 — Delete() performs no ownership check (IDOR)
- (low) Program.cs:188 — Verbose exception detail and filesystem paths returned to the client
