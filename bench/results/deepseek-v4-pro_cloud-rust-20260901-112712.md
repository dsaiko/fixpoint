# deepseek-v4-pro:cloud · rust · run 20260901-112712 (repeat 1)

recall **33/64** · 42 finding(s), 2 unmatched · 504471 tokens · 353s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics on the first link: quota map is never popula; review-security: Store::create panics on quota lookup for a new owner |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() slices out of bounds when size exceeds remaining link; review-security: Store::page off-by-one panics on short listings |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate() truncates to 0 or 100 |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() uses substring match instead of equality |
| RS09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import() is not atomic despite its contract |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the remaining count, not the removed count |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: new_code() is not unique and silently overwrites links; review-security: Short codes derived from wall-clock nanoseconds are guessabl |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | — | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s |  |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks session expiry |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() panics on a missing or malformed Authorizatio; review-security: bearer_token panics on missing/malformed Authorization heade |
| RA07 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: set() evicts an arbitrary key, not the least recently used |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() truncates to 0/100 and panics when limit is 0 |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Lock-ordering inversion between Cache::get and Cache::flush |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: sweep deletes keys after releasing the lock (TOCTOU) |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to 0 instead of keeping th |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() never returns an error |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal in report export via user-controlled name |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: report with no ?name= writes links.csv but reads .csv |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot() ignores the rename result |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: add_resolved uses load+store, losing concurrent increments |
| RW02 | — | the janitor checks for stop only before a 60s sleep, so shutdown waits |  |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache() returns before its threads finish; review-concurrency: warm_cache drops JoinHandles without joining |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner() deadlocks by locking the store twice; review-concurrency: count_by_owner deadlocks by locking the same Mutex twice |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-security: No read timeout allows slowloris connection exhaustion |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: delete_link does not check link ownership |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated by client-supplied x-admin header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats endpoint divides by zero when the store is empty; review-security: stats endpoint divides by zero on an empty store |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: * with credentials |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-security: Stored CRLF injection in redirect Location header |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/auth.rs:68 — require_admin() rejects every role
- (high) src/main.rs:216 — Report endpoint exports every owner's links
