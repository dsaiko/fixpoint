# kimi-k2.7-code:cloud · rust · run 20260901-215232 (repeat 1)

recall **33/64** · 45 finding(s), 3 unmatched · 238081 tokens · 249s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics because the quota map has no entry for the o; review-security: create() panics on first link for any owner, causing denial  |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry check; review-security: Link expiry check is inverted so expired links keep resolvin |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-security: rename() validates the source code twice and never validates |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() panics when offset+size exceeds the number of links |
| RS05 | — | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e |  |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-security: by_owner() uses substring match, leaking other owners' links |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | — | parallel: C15 prune computes the before-count, throws it away, and ret |  |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-security: Short link codes are derived from the current nanosecond and |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing secret when LINKD_SECRET is missi |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are generated from a predictable LCG seeded w |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued bearer tokens are logged to stdout |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks whether a session has expired; review-security: Token verification never checks expiration |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: Custom 64-bit password hash is unsalted, uniterated, and rev |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() panics on a missing or malformed Authorizatio; review-security: bearer_token() panics on malformed Authorization headers |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() rejects every role because its condition is  |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq() is not constant-time despite its documented purp |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | — | parallel: CA08 eviction drops whatever key iteration yields first, not |  |
| RC03 | — | RUST-ONLY utilization divides usizes before scaling so it is always 0, |  |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | — | RUST-ONLY flush locks hits then entries while get locks entries then h |  |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is parsed as seconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is read but never assigned |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-security: Config validate() always returns Ok, defeating startup check |
| RE01 | — | parallel: EX01 the report name comes from the query string and goes in |  |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-security: Exported CSV file is created with default world-readable per; review-security: CSV values are not escaped, enabling CSV injection and forma |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Read-modify-write race on resolved counter |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Janitor holds store lock during cache sweep |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache abandons spawned threads |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner() deadlocks by locking the same mutex twice; review-concurrency: Double lock of store mutex deadlocks |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-concurrency: Handler read has no timeout, threads can hang forever |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-bugs: delete_link() never checks link ownership; review-security: Delete link endpoint does not enforce ownership |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quotas endpoint relies on a client header instead of t |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats() divides by zero when the store is empty; review-security: Division by zero in stats endpoint when store is empty |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows wildcard origin with credentials |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | YES | export_csv defaults an empty name to links.csv but read_report is hand | review-security: Path traversal in /report via the name parameter |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Store lock held during report export I/O |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Create link endpoint lets caller forge the owner |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-security: Unescaped Location header enables HTTP response splitting an |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/config.rs:113 — Config::validate() always succeeds even when it finds problems
- (medium) src/cache.rs:125 — Cache::utilization() integer-division loses all precision
- (medium) src/cache.rs:32 — Lock-ordering inversion between cache entries and hit counter
