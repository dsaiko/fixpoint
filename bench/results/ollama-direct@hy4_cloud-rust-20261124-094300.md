# ollama-direct@hy4:cloud · rust · run 20261124-094300 (repeat 1)

recall **57/64** · 82 finding(s), 4 unmatched · 613592 tokens · 1227s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: First link creation for an owner panics on an absent quota e; review-concurrency: Time-only short codes can collide and overwrite links under  |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expired-link check is inverted and deletes live links; review-security: Expiry condition is inverted, deleting live links and servin |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source twice and never validates the de |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page uses len plus one as the end bound and panics |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate loses all non-100-percent values to integer div |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not return the owner's quota slot |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top sorts by ascending hits |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner uses substring matching instead of exact ownership; review-security: by_owner uses substring matching instead of equality |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend resets expiry from now and revives expired links |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: quotas returns a mutable reference instead of a snapshot |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import applies partial batches before validating the whole b |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune returns the remaining link count instead of the number |
| RS13 | YES | RUST-ONLY expires_at unwraps a missing code and panics instead of retu | review-bugs: expires_at panics on an unknown code |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: new_code can produce short or colliding codes; review-security: Short codes are derived only from the current nanosecond cou |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Missing LINKD_SECRET silently falls back to a public hardcod |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are deterministic from the issuance second |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer tokens are printed to stdout |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify never enforces session expiration; review-security: verify never enforces session expiry |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: Password hashing uses an unsalted 64-bit non-cryptographic c |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token panics on missing or single-token Authorization; review-security: Missing or malformed Authorization header panics the request |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin rejects every role because its condition is al |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq is not constant time and verify does not use it |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | YES | RUST-ONLY get takes the WRITE lock on the read path, so every cache re | review-concurrency: Every cache read takes an exclusive write lock |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: set evicts an arbitrary key instead of the least recently us |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Cache utilization truncates percentages and divides by zero |
| RC04 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: Cache warm has a check-then-act race that can overwrite fres |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Cache flush has a lock-order inversion against cache reads |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: Cache sweep can delete an entry refreshed after expiry was c |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | YES | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 | review-bugs: An invalid LINKD_PORT silently becomes port zero |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A bad LINKD_TIMEOUT_MS sets timeout to zero despite saying d |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| RF05 | YES | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - | review-bugs: load_file treats every read failure as an empty config file |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate collects configuration problems but always returns ; review-security: validate reports invalid configuration but always succeeds |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: User-controlled report name escapes the export directory |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Exports containing every owner's links are created with defa |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV export does not escape or quote link fields; review-security: CSV fields are emitted without quoting or formula neutraliza |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: archive_all discards CSV write errors; review-security: archive_all uses attacker-controlled owner strings as filena |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot ignores rename failure and reports success; review-bugs: Snapshot writes use a fixed /tmp temporary path |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Resolved metric loses increments under concurrent requests |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-bugs: Dropping JanitorStop does not stop or exit the janitor threa; review-concurrency: Janitor stop cannot interrupt sleep or sweep, and disconnect |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache returns before its spawned set operations are gua; review-concurrency: warm_cache returns before spawned cache writers finish |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner deadlocks by locking the same mutex twice |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: Request handling waits for connection EOF before responding; review-concurrency: Idle connections park detached OS threads indefinitely |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: Any authenticated user can delete another owner's link |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated next parameter creates an open redirect |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts a client-supplied x-admin header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: GET /stats divides by zero on an empty store; review-concurrency: A single panicking stats request poisons the global store mu |
| RM06 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect lowercases codes that the store treats as case-sens |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: All responses advertise wildcard CORS with credential suppor |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Error responses reflect tokens and internal filesystem detai |
| RM09 | YES | RUST-ONLY the auth path calls bearer_token on an absent Authorization  | review-bugs: The HTTP server has no way to create a valid session |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | YES | export_csv defaults an empty name to links.csv but read_report is hand | review-bugs: The report endpoint writes links.csv but reads a different . |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: TTL parsing maps invalid values to zero and can overflow |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Slow client writes hold the global store mutex; review-concurrency: Report performs blocking filesystem I/O while holding the gl |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner is taken from the request body instead of the aut |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-bugs: parse_query leaves URL-encoded keys and values encoded; review-security: Stored link target is written unescaped into the Location he |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/auth.rs:35 — Tokens issued in the same second collide and overwrite sessions
- (medium) src/cache.rs:53 — peek reports expired entries as cached
- (medium) src/worker.rs:65 — Janitor holds the store lock throughout cache sweeping
- (low) src/export.rs:56 — Audit log is opened with default permissions and accepts unsanitized lines
