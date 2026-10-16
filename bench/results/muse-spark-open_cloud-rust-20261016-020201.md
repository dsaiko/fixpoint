# muse-spark-open:cloud · rust · run 20261016-020201 (repeat 1)

recall **27/64** · 29 finding(s), 0 unmatched · 158652 tokens · 194s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: panic on first link for any new owner |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: inverted expiry check in resolve() |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates `from` twice, never validates `to` |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: off-by-one in page() causes panic |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate always 0% unless every link was used |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| RS08 | — | parallel: C11 by_owner matches owners by substring, so one owner sees  |  |
| RS09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune returns remaining count instead of removed count |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret and missing token expiry enforcemen |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are predictable from wall-clock time |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: Passwords hashed with reversible DJB2, not a password KDF |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token panics on malformed Authorization header; review-security: Malformed Authorization header panics the handler thread (Do |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin always denies due to || instead of &&; review-security: require_admin logic error denies all admins |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: cache eviction picks arbitrary entry, not LRU |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization always reports 0% until cache is full |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Lock-order inversion between Cache::get and Cache::flush cau |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS treated as seconds |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() never returns an error |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal via report name allows arbitrary file read/wr |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Metrics::add_resolved lost updates via non-atomic RMW |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Janitor stop signal delayed up to full interval |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache spawns threads but never joins them |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner deadlocks on second Mutex lock |
| RM01 | — | parallel: C24 the handler reads the whole request body with no size li |  |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: IDOR: any authenticated user can delete any link |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out enables phishing and OAuth token theft |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization bypassed via client-controlled X-Admin h |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: division by zero in stats() when store is empty |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: wildcard origin with credentials allowed |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Store Mutex held across blocking file I/O in report() |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |
