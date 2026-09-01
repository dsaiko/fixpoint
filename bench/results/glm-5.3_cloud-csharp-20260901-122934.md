# glm-5.3:cloud · csharp · run 20260901-122934 (repeat 1)

recall **44/68** · 59 finding(s), 6 unmatched · 477792 tokens · 271s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException for any new owner, so no  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve's expiry condition is inverted: live links are delet |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never validates t |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page sets end = all.Count + 1, so GetRange throws whenever t |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate uses integer division and reports 0 unless every |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete never returns the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending and returns the least-followed links, no |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner uses a substring match instead of equality |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives an already-expired link, contradicting its ow |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live internal dictionary, not a snapshot |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic: an invalid code partway through lea |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune removes during enumeration (crash) and returns the wro |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-concurrency: NewCode uses a shared unsynchronized Random; concurrent call; review-security: Short codes are 32-bit values from time-seeded System.Random |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-security: Session tokens minted from time-seeded System.Random are pre |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-bugs: Expired sessions are never removed, so the sessions dictiona; review-security: Session tokens are written to the console log |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify ignores ExpiresAt, so sessions never expire; review-security: Verify never checks session expiry, so tokens live forever |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true on both branches; the role check n; review-security: RequireAdmin returns true on its reject branch — the role ch |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: HashPassword uses unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken indexes parts[1] of a possibly-empty header, cra; review-security: BearerToken throws on any request without a two-part Authori |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals is a plain comparison contradicting its constan |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Cache.Peek (and the hits/Utilization reads) bypass the lock  |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction picks the first key in insertion order, not the lea |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep removes entries while enumerating entries.Keys; review-concurrency: Cache.Sweep removes entries while enumerating entries.Keys,  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization's integer division yields 0 for anything but a f |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: A bad LINKD_PORT silently becomes port 0 instead of keeping  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS logs 'keeping default' but sets the tim |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds, inflating the  |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then self-assigned; the conf |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: StreamWriter instances are not wrapped in using, leaking han |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-concurrency: Concurrent /report requests write and read the same CSV path |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: WriteSnapshot uses one fixed /tmp path, so concurrent snapsh; review-security: Snapshot staged through a fixed, world-predictable /tmp path |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync writes only the CSV header and discards the l; review-concurrency: ExportCsvAsync is async void: an exception during the write  |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-concurrency: Metrics.AddCreated uses a non-atomic increment while AddReso |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: Janitor removes from the store dictionary while enumerating ; review-concurrency: Janitor removes from store.All()'s live dictionary while enu |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-concurrency: WarmCache fires unawaited Task.Run work and returns, contrad |
| N52 | — | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po |  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws KeyNotFoundException for an owner with  |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links deletes any link with no ownership check (IDOR |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out forwards to any URL in ?next= |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts a client-controlled X-Admin heade |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by store.Count and crashes on an empty store |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Every response carries Access-Control-Allow-Origin: * togeth |
| N63 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | YES | offset and size are parsed with unguarded int.Parse, so any non-numeri | review-bugs: Listing endpoint parses query strings with int.Parse, crashi |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | YES | parallel: C26 the accept loop never exits, so StopJanitor is unreachab | review-concurrency: Request handlers run as unawaited fire-and-forget tasks with |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Config.cs:126 — Validate always returns true, so an invalid configuration is never refused
- (medium) Store.cs:40 — Create overwrites an existing link on a code collision, and can mint codes shorter than th
- (critical) Store.cs:20 — Store's links and quota dictionaries are mutated from multiple threads with no synchroniza
- (medium) Authenticator.cs:48 — Verify enumerates the sessions dictionary while Issue and Revoke can mutate it, with no sy
- (high) Program.cs:183 — Path traversal in /report lets callers write and read files outside the export dir
- (low) Program.cs:96 — JSON response bodies built by string interpolation with user-controlled fields
