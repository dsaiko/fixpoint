# claude-fable-5-2 · csharp · run 20261110-150415 (repeat 1)

recall **54/68** · 86 finding(s), 11 unmatched · 35331 tokens · 367s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | YES | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep | review-bugs: Create throws KeyNotFoundException for any owner without a q |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve has the expiry test inverted: live links are deleted |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to`; review-security: Rename validates `from` twice and never validates the new co |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page off-by-one: end is clamped to Count + 1, so GetRange th |
| N05 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: SuccessRate uses integer division and is 0 unless every link |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete never returns the owner's quota slot |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top sorts ascending, returning the least followed links |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner matches substrings instead of the exact owner; review-security: ByOwner matches by substring and returns other owners' links |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links |
| N10 | YES | parallel: C14 Quotas hands out the store's own dictionary while the do | review-bugs: Quotas returns the live dictionary, not a snapshot; review-concurrency: Quotas() and All() hand the live mutable dictionaries to oth |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll is not atomic and silently overwrites existing cod |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune returns the number of links left, not the number remov |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | YES | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime | review-bugs: Expiry is computed in local time, so it shifts across DST ch |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: NewCode collisions silently overwrite an existing link; review-concurrency: Shared System.Random instance used from concurrent request t |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| N18 | YES | parallel: S02 session tokens come from System.Random, so they are pred | review-concurrency: Token minting uses a shared non-thread-safe Random instance; review-security: Session tokens generated with predictable System.Random |
| N19 | YES | parallel: S03 every issued session token is logged in cleartext | review-concurrency: sessions dictionary is read and written from multiple thread; review-security: Bearer token written to the log in clear text |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify never checks ExpiresAt, so expired sessions stay vali; review-security: Verify never checks session expiry |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin returns true on both branches; review-security: RequireAdmin returns true for every session |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with a single unsalted SHA-256 |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken indexes parts[1] without checking the header sha; review-security: BearerToken crashes on a missing or malformed Authorization  |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-security: SecretEquals is not constant-time |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | YES | CS-ONLY every method locks on `this`, which any external caller can al | review-concurrency: Cache synchronizes on lock(this), a publicly reachable monit |
| N28 | YES | parallel: CA02 Peek reads the dictionary with no lock at all while loc | review-concurrency: Peek reads the entries dictionary without taking the lock |
| N29 | YES | parallel: CA07 Stats reads the hit counter outside the lock while Get  | review-concurrency: Stats returns the hit counter without the lock or an atomic  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Set evicts an arbitrary entry, not the least recently used,  |
| N31 | YES | parallel: CA06 Warm's Peek and Set are separately locked, so the janit | review-concurrency: Warm does check-then-act (Peek, then Set) outside any single |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | YES | parallel: C05 Utilization divides before scaling so it always reports  | review-bugs: Utilization uses integer division and divides by zero when l |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: An unparsable LINKD_PORT overwrites the default port with 0 |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A bad LINKD_TIMEOUT_MS zeroes the timeout while logging 'kee |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then thrown away |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | YES | LoadFile parses cache_size with int.Parse inside the try's successor,  | review-bugs: LoadFile crashes on a bad cache_size and hides every read er |
| N41 | — | parallel: EX01 the report name comes from the query string; Path.Combi |  |
| N42 | YES | CS-ONLY the StreamWriter is closed only on the success path with no `u | review-bugs: StreamWriters are not disposed when a write fails |
| N43 | YES | parallel: EX03 CSV rows are built by interpolation, so a comma or quot | review-bugs: CSV fields are written without quoting or escaping; review-security: ExportCsv does not confine the report name to dir |
| N44 | — | ArchiveAll builds a filename straight from the owner string, so an own |  |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: WriteSnapshot is neither durable nor atomic as documented; review-concurrency: WriteSnapshot uses one fixed temp path, so concurrent snapsh |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void and writes only the header; review-concurrency: ExportCsvAsync is async void: cannot be awaited and its exce |
| N48 | YES | parallel: X01 AddCreated increments with ++ while AddResolved uses Int | review-bugs: The errors metric can never be anything but 0; review-concurrency: Metrics.AddCreated uses a non-atomic created++ |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | YES | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi | review-concurrency: Janitor timer enumerates and mutates the store's live dictio |
| N51 | YES | parallel: X03 WarmCache starts a Task per link and awaits none of them | review-bugs: WarmCache returns before its tasks finish and never probes; review-concurrency: WarmCache starts tasks it never awaits, contrary to its cont |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-concurrency: CheckTargets blocks on ProbeAsync(...).Result (sync-over-asy |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner throws for an owner with no quota entry |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | YES | parallel: S06 an absent Authorization header becomes the empty string  | review-bugs: Every route requires a session, but nothing ever issues one |
| N56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented token back to the client |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links has no ownership check |
| N58 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: /out answers 301, so repeat clicks are cached and not counte; review-security: Open redirect on /out via unvalidated ?next= |
| N59 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorised by a client-supplied X-Admin heade |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by store.Count, which is zero on an empty stor |
| N61 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect lowercases the code, so mixed-case codes can never  |
| N62 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin together with credentials |
| N63 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: Unparsed query integers and no top-level catch leave request; review-security: Link owner taken from the query string instead of the authen |
| N64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Path traversal in /report: user-controlled name writes and r |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | YES | parallel: C26 the accept loop never exits, so StopJanitor is unreachab | review-concurrency: Request tasks are fire-and-forget: exceptions are unobserved |
| N68 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report reads back a different file than the one it just wrot; review-concurrency: Concurrent /report requests with the same name race on one f |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate always returns true, and Main ignores its result anyway
- (low) Cache.cs:53 — Peek reports expired entries as cached, so Warm skips them
- (low) Program.cs:96 — The create response interpolates the target into JSON without escaping
- (low) Program.cs:197 — Respond labels plain-text bodies as application/json
- (high) Store.cs:20 — Store has no synchronization but is shared by every request thread and the janitor
- (low) Cache.cs:126 — Utilization reads entries.Count outside the lock
- (low) Worker.cs:47 — StopJanitor does not wait for an in-flight tick, and StartJanitor overwrites a running tim
- (high) Export.cs:75 — ReadReport reads arbitrary files via path traversal
- (medium) Program.cs:188 — 500 response leaks full exception, stack trace and server paths
- (medium) Program.cs:96 — User-controlled target interpolated into JSON without escaping
- (medium) Export.cs:28 — Report holding every owner's links is created with default permissions
