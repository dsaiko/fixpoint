# glm-5.3-flash:cloud · rust · run 20260901-135000 (repeat 1)

recall **35/64** · 45 finding(s), 1 unmatched · 604511 tokens · 293s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create panics: quota entry is never initialized for the owne; review-security: create() panics for any owner not already in the quota map |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve inverts the expiry test, deleting live links and ser; review-security: Expiry check inverted: expired links keep resolving, live li |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates `from` twice, leaving the destination code  |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page off-by-one clamps end to len + 1, panicking the slice i |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate divides before multiplying, always reporting 0 |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top sorts ascending, returning the least-followed links |
| RS08 | — | parallel: C11 by_owner matches owners by substring, so one owner sees  |  |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links, contradicting the expi |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune returns the surviving count instead of the number remo |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-security: Short codes are predictable (subsecond nanos), so 'private'  |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret silently used when LINKD_SECRET is |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are derived from the issue time via a fixed L |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to stdout |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify never checks expires_at, so expired sessions keep wor; review-security: verify() never checks expires_at, so sessions never expire |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token panics on any request without an Authorization ; review-security: bearer_token panics on missing or malformed Authorization he |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-security: require_admin condition is a tautology and rejects every rol |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq is not constant-time despite its documented guaran |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | — | parallel: CA08 eviction drops whatever key iteration yields first, not |  |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization divides before multiplying and can divide by zer |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Lock-ordering inversion between Cache::get and Cache::flush |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: Cache::sweep check-then-act race deletes freshly refreshed e |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS parsed as milliseconds but converted with |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate always returns Ok, so bad config never fails at sta |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-bugs: report name escapes the export directory, contradicting the ; review-security: Path traversal in report name allows arbitrary file write an |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Exports written world-readable, and snapshot staged in predi |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-security: CSV rows built by string interpolation: record forging and f |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Metrics::add_resolved loses updates: load-then-store instead |
| RW02 | — | the janitor checks for stop only before a 60s sleep, so shutdown waits |  |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache never joins its spawned threads despite promising |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner deadlocks: locks the same non-reentrant Mutex |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: read_to_end blocks until the client closes the connection; review-concurrency: No read timeout and unbounded thread spawn: idle clients pin |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-bugs: delete_link performs no ownership check despite the document; review-security: Any authenticated user can delete any link (no ownership che |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out forwards to any attacker-chosen URL |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated by client-controlled header instead of  |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats divides total hits by store length without a zero chec; review-concurrency: Division by zero while holding the store lock poisons it and |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS combined with Allow-Credentials on every respo |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response reflects the presented token back to the caller |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | YES | JSON responses are built by string interpolation with no escaping, so  | review-security: Hand-built JSON embeds unescaped user input |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/auth.rs:68 — require_admin condition is wrong operator and always rejects
