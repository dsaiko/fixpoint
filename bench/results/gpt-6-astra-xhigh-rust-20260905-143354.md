# gpt-6-astra-xhigh · rust · run 20260905-143354 (repeat 1)

recall **35/64** · 50 finding(s), 11 unmatched · 370423 tokens · 590s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Creating the first link for an owner always panics |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving an unexpired link deletes it |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never the destination |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: Pages extending beyond the store panic |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: Success percentages truncate before multiplication |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Removing links leaves owner counts unchanged |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard selects the least-followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup matches substrings instead of exact owners |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten or revive it |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: Quota snapshots expose the mutable internal map |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: Failed imports leave partial changes behind |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: Prune returns the number retained instead of removed |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | — | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s |  |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: Missing Authorization headers panic during token parsing |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: The admin-role condition rejects every role |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Cache eviction ignores access recency |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: A zero cache size causes division by zero; review-bugs: Partial cache utilization is always reported as zero |
| RC04 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: Cache warming can overwrite a concurrent insertion |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Cache flush and get acquire locks in opposite orders |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout input replaces the default with zero |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: The configured fetch limit is discarded |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: Configuration validation succeeds despite detected errors |
| RE01 | — | parallel: EX01 the report name comes from the query string and goes in |  |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV fields are written without escaping |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot success does not provide the promised durability; review-concurrency: Concurrent snapshots share and truncate one temporary file |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Resolved counter increments lose concurrent updates |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Dropping the janitor handle leaves its thread running |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache returns before its workers finish |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner locks the same mutex twice |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: Request processing waits for connection EOF; review-concurrency: Request threads wait indefinitely for connection EOF |
| RM02 | — | parallel: C17 delete_link never checks the owner its doc promises to c |  |
| RM03 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| RM04 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: Statistics panic when the store is empty |
| RM06 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect lookup changes case-sensitive stored codes |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: Unbounded TTL arithmetic can panic or wrap |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Report I/O holds the global store mutex |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | YES | JSON responses are built by string interpolation with no escaping, so  | review-bugs: Accepted targets can produce invalid creation JSON |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-bugs: Query and form values are never decoded |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) src/store.rs:47 — Generated code collisions silently overwrite existing links
- (high) src/main.rs:40 — The server has no way to obtain a valid session
- (high) src/main.rs:243 — Non-JSON responses are advertised as JSON
- (medium) src/main.rs:216 — Reports silently omit links after the first 1000
- (medium) src/export.rs:67 — Reading a report uses a different default filename
- (medium) src/cache.rs:60 — Updating a full cache can remove an unrelated entry
- (medium) src/cache.rs:86 — Cache warming skips expired entries
- (medium) src/export.rs:36 — Archive writes can fail while the function reports success
- (medium) src/export.rs:49 — Snapshot installation failures are ignored
- (medium) src/cache.rs:105 — Cache sweeping can delete a concurrently refreshed entry
- (critical) src/main.rs:55 — Idle connections can exhaust threads and terminate the server
