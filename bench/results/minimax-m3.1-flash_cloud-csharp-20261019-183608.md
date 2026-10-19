# minimax-m3.1-flash:cloud · csharp · run 20261019-183608 (repeat 1)

recall **37/68** · 48 finding(s), 6 unmatched · 226653 tokens · 3427s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Delete() never releases the owner's quota slot, so the per-o |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve() treats live links as expired and expired links as  |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename() validates `from` twice and never validates the dest |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page() computes an end index one past the last element, so G |
| N05 | — | parallel: C05 integer division before scaling yields 0 for every rate  |  |
| N06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top() sorts ascending by hits, so it returns the least-follo |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner() matches owners by substring, so one owner's name c; review-security: ByOwner matches owners by substring, leaking other users' li |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend() resurrects an already-expired link, contradicting i |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | — | parallel: C13 ImportAll stores as it validates, so a bad code leaves t |  |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune() returns the store's remaining size instead of the nu |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: NewCode() mints short codes from a non-cryptographic Random,; review-security: Short link codes are generated from a clock-seeded System.Ra |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing secret |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens are minted from System.Random and are predict |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Issued session tokens are written to stdout |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify() never checks Session.ExpiresAt, so expired sessions |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin() returns true for every role — the allowed-lis; review-security: RequireAdmin returns true on every path, including for non-a |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: HashPassword is a single unsalted SHA-256 digest |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken() indexes parts[1] and throws IndexOutOfRangeExc; review-security: BearerToken indexes parts[1] without a length check |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals uses a non-constant-time string comparison |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | — | parallel: CA02 Peek reads the dictionary with no lock at all while loc |  |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set() evicts the first dictionary key, not the least recentl |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep() removes entries from the dictionary while enumeratin |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization() uses integer division, so it reports 0 until t |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A malformed LINKD_TIMEOUT_MS still overwrites the default wi |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is applied as seconds, so a millisecond s |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded — the value n |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile() throws FormatException on a non-numeric cache_siz |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-security: ArchiveAll builds file paths from owner names without saniti |
| N45 | YES | parallel: EX02 the report holding every owner's links is written with  | review-bugs: ExportCsv() does not confine the user-supplied report name t |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Snapshot temp file uses a fixed, predictable path in /tmp |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync() is `async void`, so its exceptions are unob |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-bugs: Metrics.AddCreated() increments without Interlocked, losing  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes entries from the live store dictionary while |
| N51 | — | parallel: X03 WarmCache starts a Task per link and awaits none of them |  |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | — | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot |  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented token back to the client |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links removes any link regardless of owner |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out echoes an unvalidated URL into the Locat |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is a client-supplied X-Admin request hea |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats() divides by zero when the store is empty |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Access-Control-Allow-Origin: * combined with Access-Control- |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: int.Parse on the ttl/offset/size query parameters throws on ; review-security: Link ownership is taken from the query string instead of the |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate() always returns true — invalid configuration is reported but never rejected
- (low) Store.cs:20 — The store's dictionaries are unsynchronised while Program serves requests on concurrent th
- (high) Program.cs:180 — Path traversal in /report via Path.Combine with an absolute name
- (medium) Program.cs:96 — JSON response body built by interpolation without escaping link targets
- (medium) Program.cs:188 — Exception object interpolated into a 500 response leaks internals
- (medium) Export.cs:28 — Export files are created world-readable although the comment says service-user-only
