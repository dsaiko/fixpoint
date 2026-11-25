# ollama-direct@mimo-v2.6-flash:cloud · rust · run 20261125-141611 (repeat 1)

recall **43/64** · 53 finding(s), 3 unmatched · 542312 tokens · 383s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Link creation panics after inserting the link |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry condition is inverted during resolution |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice instead of the destin |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: Partial pages produce an out-of-bounds slice |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: Success-rate percentage rounds almost every value to zero |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not return its quota slot |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top-links sorting is in ascending hit order |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner filtering uses substring matching |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives links that have already expired |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: Quota accessor exposes mutable store state despite snapshot  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: Batch import is neither atomic nor collision-safe |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: Prune reports remaining links instead of removed links |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-security: Short-link codes are predictable from subsecond time |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are deterministic from wall-clock time |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued bearer tokens are written to logs |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Expired sessions remain valid indefinitely |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: Malformed Authorization headers panic; review-security: Malformed Authorization headers panic the request handler |
| RA07 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Cache eviction does not select the least recently used entry |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Cache utilization divides by zero and truncates percentages |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Cache flush and get acquire shared locks in opposite orders |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: Sweep deletes refreshed entries after a check-then-act race |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL environment value uses the wrong unit |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: Configured fetch limit is parsed and then discarded |
| RF05 | YES | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - | review-bugs: Configuration file read failures are indistinguishable from  |
| RF06 | — | parallel: CF06 validate logs the problems it finds and returns Ok, so  |  |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Report name permits path traversal and arbitrary file overwr |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Cross-owner reports are created with default file permission |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV records are written without field escaping; review-security: CSV export is vulnerable to formula and row injection |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: Archive write errors are silently discarded |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot rename failure is reported as success; review-concurrency: Concurrent snapshots race through one fixed temporary path |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Resolved metric uses a non-atomic read-modify-write |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-bugs: Dropping the janitor handle does not stop the thread; review-concurrency: Janitor stop handle neither guarantees termination nor stops |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache drops JoinHandles instead of awaiting its worker  |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner self-deadlocks by locking the same mutex twic |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: HTTP request parsing waits for connection EOF; review-concurrency: Unbounded handler threads block indefinitely on request read |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: Delete endpoint ignores link ownership |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect through the /out endpoint |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts a client-controlled header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: Stats divides by zero on an empty store |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | YES | export_csv defaults an empty name to links.csv but read_report is hand | review-bugs: Default report writes and reads different filenames |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: TTL parsing accepts invalid input and can overflow |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Report holds the global store lock across disk and network I |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | YES | JSON responses are built by string interpolation with no escaping, so  | review-bugs: Create response interpolates unescaped JSON strings |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-bugs: Query and form values are not URL-decoded; review-security: Unvalidated redirect targets allow HTTP response header inje |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/config.rs:113 — Configuration validation always returns success
- (medium) src/main.rs:192 — Quota endpoint returns only the number of owners
- (high) src/main.rs:216 — Report endpoint exposes links belonging to other owners
