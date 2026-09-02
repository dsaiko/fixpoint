# gpt-oss:120b-cloud · csharp · run 20260902-105845 (repeat 1)

recall **38/68** · 65 finding(s), 11 unmatched · 1000642 tokens · 390s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Quota increment may throw KeyNotFoundException; review-concurrency: Unsynchronized mutation of `links` and `quota` dictionaries |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve removes valid links and returns null; review-security: Expired‑link logic inverted |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: `Rename` validates the source code twice; review-security: Rename validates wrong variable and skips new code check |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: `Page` off‑by‑one may cause `ArgumentOutOfRangeException`; review-security: Off‑by‑one error in paging logic |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: `SuccessRate` integer division truncates result |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: `Delete` does not restore owner quota |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: `Top` sorts ascending, returning least‑used links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: `ByOwner` uses substring match |
| N09 | — | parallel: C12 Extend re-dates an already-expired link, resurrecting a  |  |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: `Quotas` exposes internal mutable dictionary; review-concurrency: `Quotas` exposes internal mutable dictionary |
| N11 | — | parallel: C13 ImportAll stores as it validates, so a bad code leaves t |  |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: `Prune` returns remaining count instead of removed count |
| N13 | YES | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  | review-bugs: `Prune` removes entries while iterating over keys; review-concurrency: `Prune` removes entries while iterating over `links.Keys` |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: `NewCode` may generate duplicate short codes |
| N16 | YES | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l | review-concurrency: `Followed` returns a live enumerable over mutable dictionary |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hard‑coded development secret |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-concurrency: Non‑thread‑safe `Random` used for token generation; review-security: Predictable session tokens |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-bugs: Issue can overwrite an existing session token |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify does not check token expiration |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin always returns true; review-security: RequireAdmin always authorises |
| N22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken may throw IndexOutOfRange; review-security: Bearer token parsing can throw |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: Non‑constant‑time secret comparison |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: `Peek` reads dictionary without synchronization |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Eviction does not follow LRU policy |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | YES | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid | review-bugs: Sweep mutates dictionary during enumeration; review-concurrency: Removing entries while iterating over dictionary keys in `Sw |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization uses integer division |
| N34 | — | parallel: CF01 TryParse's result is discarded and it writes 0 into Por |  |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Timeout is set even when parsing fails |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL parsed using wrong time unit; review-security: Cache TTL interpreted as seconds instead of milliseconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: `FETCH_LIMIT` environment variable is ignored |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-security: Path traversal in CSV export |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-concurrency: File writes are not synchronized |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Shared temporary file in `WriteSnapshot` creates a race cond; review-security: Symlink race on temporary snapshot file |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: `ExportCsvAsync` is declared `async void`; review-concurrency: `ExportCsvAsync` is declared `async void` |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-bugs: `StartJanitor` mutates store while enumerating keys; review-concurrency: Janitor timer removes entries from `Store` without synchroni |
| N51 | — | parallel: X03 WarmCache starts a Task per link and awaits none of them |  |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-bugs: `CheckTargets` blocks on async call |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: `CountForOwner` throws if owner not present |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-security: Bearer token extraction used before validation |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Delete endpoint lacks ownership check |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin flag trusted without authentication |
| N60 | — | parallel: C20 /stats divides by the link count, throwing DivideByZeroE |  |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) Cache.cs:53 — Peek reports expired entries as cached
- (high) Config.cs:126 — `Validate` always returns true
- (medium) Export.cs:83 — `ExportCsvAsync` writes only header
- (medium) Metrics.cs:17 — `AddCreated` is not thread‑safe
- (high) Authenticator.cs:37 — Unsynchronized access to shared sessions dictionary
- (medium) Store.cs:228 — `Random` used for code generation is not thread‑safe
- (medium) Store.cs:60 — `link.Hits++` is not atomic
- (medium) Store.cs:108 — `SuccessRate` enumerates `links` without synchronization
- (medium) Metrics.cs:17 — `AddCreated` increments a non‑atomic counter
- (high) Export.cs:75 — Path traversal in CSV read
- (medium) Config.cs:126 — Validate always returns true
