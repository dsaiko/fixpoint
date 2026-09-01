# codex · rust · run summaries (repeat 1)

recall **36/64** · 44 finding(s), 4 unmatched · 261131 tokens · 817s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Creating a link for any new owner panics |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiration logic is inverted |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice instead of validating |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: Partial final pages panic with an out-of-bounds slice |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: Success rate truncates before converting to a percentage |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting links never returns their quota slots |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top links are returned in ascending hit order |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup matches substrings instead of exact owners |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link revives already-expired links |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: Batch import is not atomic |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: Prune returns the number of links remaining instead of remov |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are predictable from their issuance time |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Live access tokens are written to application logs |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions continue to verify successfully; review-security: Expired access tokens remain valid indefinitely |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: Missing or malformed Authorization headers panic; review-security: Malformed Authorization headers remotely panic request handl |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Admin-role check rejects every possible role |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Cache eviction is arbitrary rather than least-recently-used |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Cache utilization is miscalculated and can panic |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | — | RUST-ONLY flush locks hits then entries while get locks entries then h |  |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: Cache sweep can delete a concurrently refreshed entry |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: Configuration validation always accepts invalid settings |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Report names permit writes outside the export directory |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-security: Unescaped owner values enable CSV formula injection |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: Archive writes silently ignore I/O failures |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots race through one shared temporary file |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Concurrent resolved-counter increments are lost |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Janitor stop can return before another sweep and dropping it |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache returns before its worker threads finish |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner deadlocks by relocking the same mutex |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: HTTP handling waits for the client to close the connection; review-concurrency: Persistent connections consume unbounded handler threads |
| RM02 | — | parallel: C17 delete_link never checks the owner its doc promises to c |  |
| RM03 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Client-controlled header grants administrator access |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: Stats endpoint divides by zero for an empty store |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Authenticated identity is discarded, eliminating tenant isol |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | YES | export_csv defaults an empty name to links.csv but read_report is hand | review-bugs: A report with the default name is written and then read unde |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: Invalid or overflowing TTL input is silently accepted |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Handlers retain the global store mutex during blocking I/O |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.rs:40 — The running server has no way to create a valid session
- (medium) src/export.rs:49 — Snapshot installation failures are reported as success
- (medium) src/cache.rs:39 — Cache get and flush acquire locks in opposite orders
- (high) src/main.rs:144 — Stored link targets allow HTTP response splitting
