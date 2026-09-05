# claude-sonnet-5 · rust · run 20260905-154959 (repeat 1)

recall **29/64** · 30 finding(s), 0 unmatched · 59322 tokens · 575s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() always panics: quota entry for a new owner is never |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() has an inverted expiry check |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() slices past the vector length and panics |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate() truncates via integer division |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least-hit links instead |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches by substring instead of exact owner |
| RS09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | — | parallel: C15 prune computes the before-count, throws it away, and ret |  |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens generated with a predictable LCG seeded from  |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to stdout logs and echoed back to cal |
| RA04 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: Passwords stored with a fast, unsalted, non-cryptographic ha |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token panics on any request without a well-formed Aut; review-security: Every request lacking an Authorization header crashes its ha |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin's condition can never allow access |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Eviction picks an arbitrary key, not the least-recently-used |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() truncates via integer division and can divide  |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Cache::get and Cache::flush take entries/hits locks in oppos |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds, not millisecon |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed but discarded, never applied |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() never actually fails, defeating the startup safet |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal via user-supplied report name |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot silently ignores rename failure and reports s |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Metrics::add_resolved does a non-atomic load+store, losing i |
| RW02 | — | the janitor checks for stop only before a 60s sleep, so shutdown waits |  |
| RW03 | — | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi |  |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner deadlocks by locking the same non-reentrant M |
| RM01 | — | parallel: C24 the handler reads the whole request body with no size li |  |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: Missing ownership check lets any user delete any link |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated only by a client-supplied header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats() divides by zero when the store is empty |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS combined with allow-credentials |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Store mutex held across blocking socket write lets one slow  |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |
