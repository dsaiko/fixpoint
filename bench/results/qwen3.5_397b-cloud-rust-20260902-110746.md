# qwen3.5:397b-cloud · rust · run 20260902-110746 (repeat 1)

recall **26/64** · 36 finding(s), 5 unmatched · 285763 tokens · 128s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Panic on first link creation due to uninitialized quota map; review-security: Panic on first link creation per owner |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted expiry check deletes valid links and serves expired |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice, never validates 'to' |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() can panic with out-of-bounds slice when offset + size |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate() always returns 0 due to integer division orde |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns least-followed links due to wrong sort order |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches partial owner names instead of exact; review-security: Owner enumeration via substring matching |
| RS09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import() is not atomic - partial imports on validation failu |
| RS12 | — | parallel: C15 prune computes the before-count, throws it away, and ret |  |
| RS13 | YES | RUST-ONLY expires_at unwraps a missing code and panics instead of retu | review-bugs: expires_at() panics on unknown code instead of handling grac |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-security: Predictable short code generation |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret enables token forgery |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Predictable token generation from timestamp seed |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token logged to stdout |
| RA04 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() panics on malformed authorization header; review-security: bearer_token panics on malformed Authorization header |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always rejects due to tautological condition; review-security: Admin check always passes due to logic error |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | YES | verify scans every session linearly although the map is already keyed  | review-concurrency: Authenticator::verify iterates HashMap without synchronizati |
| RC01 | YES | RUST-ONLY get takes the WRITE lock on the read path, so every cache re | review-concurrency: Cache::get holds write lock for read operations, causing unn |
| RC02 | — | parallel: CA08 eviction drops whatever key iteration yields first, not |  |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() always returns 0 due to integer division order |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | — | RUST-ONLY flush locks hits then entries while get locks entries then h |  |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: Cache::sweep has TOCTOU race between read and write locks |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu |  |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | — | parallel: CF06 validate logs the problems it finds and returns Ok, so  |  |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal via user-controlled export name |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | — | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of |  |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Janitor thread handle is dropped, causing unbounded thread l |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache drops thread handles, threads may not complete |
| RW04 | YES | parallel: X04 check_targets returns on the first error; the remaining  | review-concurrency: check_targets leaks sender clones and may deadlock on channe |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner() deadlocks due to double lock acquisition; review-concurrency: count_by_owner acquires same lock twice causing potential de |
| RM01 | — | parallel: C24 the handler reads the whole request body with no size li |  |
| RM02 | — | parallel: C17 delete_link never checks the owner its doc promises to c |  |
| RM03 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| RM04 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats() divides by zero when no links exist |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS with credentials enables credential theft |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.rs:92 — handle() panics on missing authorization header
- (high) src/cache.rs:105 — sweep() deadlocks trying to acquire write lock while holding read lock
- (high) src/store.rs:48 — Race condition in add_resolved uses non-atomic read-modify-write
- (medium) src/main.rs:44 — Janitor stop called before accepting connections, race with handler threads
- (low) src/main.rs:92 — Missing Authorization header causes panic
