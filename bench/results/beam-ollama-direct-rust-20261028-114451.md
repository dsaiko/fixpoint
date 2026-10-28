# beam-ollama-direct · rust · run 20261028-114451 (repeat 1)

recall **38/64** · 55 finding(s), 7 unmatched · 804345 tokens · 166s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics for a new owner |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expires live links and keeps expired ones; review-security: Expired links keep resolving; valid links are evicted |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice instead of 'to'; review-security: rename validates 'from' twice and never validates 'to' |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() slices past the end and panics |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate() truncates before scaling |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the quota slot |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns least-followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches substrings; review-security: by_owner uses substring match, leaking other owners' links |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links; review-security: extend() revives expired links |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: quotas() leaks a mutable handle to internal state |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import() is not atomic |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() reports post-count instead of removed count |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Predictable session tokens from time-seeded LCG |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens printed to stdout |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() does not enforce token expiry; review-security: Token expiry is never enforced |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token panics on malformed Authorization header; review-security: bearer_token can panic on malformed Authorization header |
| RA07 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq leaks timing and length |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: set() evicts an arbitrary entry instead of LRU |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() always rounds to zero |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | — | RUST-ONLY flush locks hits then entries while get locks entries then h |  |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS resets timeout to zero |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS treated as seconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() returns Ok even with problems |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal via user-controlled export name |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Path traversal via owner-controlled filename in archive_all |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: write_snapshot races on a shared fixed temporary path |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-bugs: add_resolved() is non-atomic; review-concurrency: Metrics::add_resolved uses non-atomic load/store |
| RW02 | — | the janitor checks for stop only before a 60s sleep, so shutdown waits |  |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache() returns before workers finish; review-concurrency: warm_cache discards thread handles and returns immediately |
| RW04 | YES | parallel: X04 check_targets returns on the first error; the remaining  | review-bugs: check_targets() can hang on early failure |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner() deadlocks on re-locking the mutex; review-concurrency: count_by_owner locks the same non-reentrant Mutex twice |
| RM01 | — | parallel: C24 the handler reads the whole request body with no size li |  |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-bugs: delete_link() does not enforce ownership; review-security: delete_link does not enforce ownership |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unvalidated ?next= parameter |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by spoofable x-admin header instea |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats() divides by zero on empty store |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS with credentials enabled |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown token echoed back in 401 response |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner taken from request body instead of authenticated  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/auth.rs:68 — require_admin() rejects every role
- (low) src/cache.rs:51 — peek() reports expired entries as present
- (medium) src/export.rs:36 — archive_all ignores write errors
- (high) src/export.rs:49 — write_snapshot ignores rename failure
- (critical) src/cache.rs:45 — Cache::get and Cache::flush acquire locks in opposite order
- (low) src/worker.rs:70 — JanitorStop has no Drop implementation
- (high) src/main.rs:248 — Header injection through redirect Location value
