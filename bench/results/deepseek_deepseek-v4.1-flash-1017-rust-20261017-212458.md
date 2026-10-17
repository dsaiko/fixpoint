# deepseek/deepseek-v4.1-flash-1017 · rust · run 20261017-212458 (repeat 1)

recall **45/64** · 59 finding(s), 4 unmatched · 455027 tokens · 2374s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics on every first create: quota map is never po; review-concurrency: A panic under the store lock poisons it and permanently disa |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry test: live links are deleted, e |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() sets end to len+1 and panics on the slice |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate() integer-divides before multiplying, always re |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never releases the owner's quota slot |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() uses substring matching instead of equality |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import() is not atomic: validation and insertion are interle |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the remaining count instead of the removed c |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: new_code() derives codes from subsec_nanos, which repeats ev; review-security: Short codes are predictable and silently overwrite existing  |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hard-coded fallback secret when LINKD_SECRET is unset |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are generated from a time-seeded LCG, not a C |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are written to stdout |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks expires_at; expired sessions stay vali; review-security: Session expiry is never enforced |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: Passwords are hashed with an unsalted, fast, non-cryptograph |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() panics on a missing or malformed Authorizatio; review-security: bearer_token panics on a missing or malformed Authorization  |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() condition is a tautology and rejects everyon; review-security: require_admin can never return Ok |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq is not constant-time and is never used; verify com |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Cache eviction picks an arbitrary entry, not the least recen |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() integer-divides before multiplying and divides |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Lock-order inversion between Cache::get and Cache::flush dea |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS logs 'keeping default' but sets the tim |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is parsed as seconds, not milliseconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() always returns Ok despite collecting problems |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal and absolute-path override in user-supplied r |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Exports containing every owner's links are created world-rea |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: Empty report name writes links.csv but read_report looks for; review-bugs: CSV fields are written without quoting or escaping |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: archive_all discards per-link write errors and reports succe |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot ignores the rename error and never syncs the ; review-security: Snapshot is written through a fixed, predictable path in /tm |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: add_resolved uses a non-atomic load/store and loses incremen |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-bugs: Janitor never stops when its handle is dropped |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache() drops the join handles without joining; review-concurrency: warm_cache detaches its worker threads instead of joining th |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner re-locks the same std::sync::Mutex and deadlo |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: Request handler reads to EOF, so keep-alive clients block fo; review-concurrency: No read timeout and unbounded thread-per-connection allow th |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: Any authenticated user can delete any link; ownership is nev |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin-only endpoint authorizes on a client-supplied X-Admin  |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats() divides by zero on an empty store |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS is wide open and combines wildcard origin with credenti |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: Malformed ttl silently becomes 0 and large ttl overflows |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Global store mutex is held across socket writes and file I/O |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner is taken from the request body, not the session |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/main.rs:40 — No code ever calls Authenticator::issue, so every request is rejected with 401
- (low) src/worker.rs:70 — JanitorStop does not stop the janitor on drop, and stop() is delayed by the sleep
- (medium) src/main.rs:251 — CRLF in a stored link target is injected verbatim into the Location header
- (medium) src/main.rs:216 — Report endpoint exposes every owner's links to any authenticated user
