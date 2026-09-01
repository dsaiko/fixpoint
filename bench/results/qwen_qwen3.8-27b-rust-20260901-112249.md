# qwen/qwen3.8-27b · rust · run 20260901-112249 (repeat 1)

recall **35/64** · 46 finding(s), 3 unmatched · 307782 tokens · 1612s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics on the quota for any owner not yet tracked; review-security: create() panics for any new owner (uninitialized quota) |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() treats valid links as expired and vice-versa; review-security: resolve() expiry check is inverted; expired links stay live |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates from twice and never validates to |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() computes an out-of-bounds end index; review-security: page() off-by-one panics whenever the requested page extends |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate() divides before multiplying, truncating to 0 |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not return the owner's quota slot |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches by substring, not equality |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the remaining count, not the removed count |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are generated from a time-seeded LCG (predict |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token is written to stdout logs |
| RA04 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: Password 'hash' is unsalted djb2 (fast, unkeyed, 64-bit) |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() reads parts[1] without a length check; review-security: bearer_token panics on missing/malformed Authorization heade |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() rejects every role |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq claims constant-time but returns early (timing lea |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | — | parallel: CA08 eviction drops whatever key iteration yields first, not |  |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() can divide by zero and truncates the percentag |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: ABBA deadlock between Cache::get and Cache::flush |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: Cache::sweep can delete a fresh entry: check-then-act TOCTOU |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is applied as seconds, not milliseconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed then discarded |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() collects problems but always returns Ok |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal / arbitrary file write via report ?name= |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot() ignores the rename result |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-bugs: add_resolved() does a non-atomic read-modify-write; review-concurrency: Metrics::add_resolved loses increments: non-atomic load/stor |
| RW02 | — | the janitor checks for stop only before a 60s sleep, so shutdown waits |  |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache() does not wait for its probes to finish; review-concurrency: warm_cache returns before probes finish: JoinHandles dropped |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner() locks the store Mutex twice, deadlocking; review-concurrency: count_by_owner deadlocks: locks the same Mutex twice in one  |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-concurrency: Unbounded thread-per-connection with no read timeout |
| RM02 | — | parallel: C17 delete_link never checks the owner its doc promises to c |  |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out via unvalidated ?next= |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated only on a client-supplied x-admin heade |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats() divides by the link count without a zero check; review-concurrency: Panic under the store lock permanently poisons the mutex and |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS wildcard origin combined with credentials |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Global store mutex held across blocking socket and disk I/O |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner is taken from the request body, not the session |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-security: HTTP response splitting / header injection via CRLF in store |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/worker.rs:75 — Dropping JanitorStop does not stop the janitor thread
- (medium) src/auth.rs:33 — No token-issuance path exists; authentication can never succeed
- (low) src/main.rs:46 — Plaintext HTTP on 0.0.0.0 exposes tokens and data in transit
