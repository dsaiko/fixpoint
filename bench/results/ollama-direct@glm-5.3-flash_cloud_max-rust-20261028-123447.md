# ollama-direct@glm-5.3-flash:cloud#max · rust · run 20261028-123447 (repeat 1)

recall **37/64** · 47 finding(s), 4 unmatched · 962188 tokens · 479s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Link creation always panics because the owner quota entry do; review-security: Creating a link panics and poisons the global store lock |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiry check is inverted |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-security: List pagination panic poisons the store lock |
| RS05 | — | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e |  |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not restore its owner quota |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard returns the least followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner filtering uses substring matching instead of equality |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Expired links can be extended back to life |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: Import is not atomic when validation fails |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: Prune reports the wrong count |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-security: Short-link codes are predictable |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are predictable and logged |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions are accepted; review-security: Expired sessions are still accepted |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: Malformed Authorization header causes an indexing panic; review-security: Missing Authorization header causes request-handler panic |
| RA07 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Cache eviction does not evict the least recently used entry |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Zero cache size causes utilization division by zero and unbo |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Cache flush and get acquire locks in opposite order |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: Configuration validation never rejects invalid configuration |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Report name allows path traversal and arbitrary file writes |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Sensitive export files do not enforce private permissions |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV report is not escaped; review-security: CSV output is unescaped and vulnerable to formula injection |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot write silently reports success when rename fails |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Non-atomic metric increment loses updates |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-bugs: Dropping JanitorStop does not stop the janitor thread |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: Cache warming is detached and unawaited |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: Self-deadlock by re-locking a non-reentrant Mutex |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: HTTP request reading hangs on ordinary keep-alive clients; review-concurrency: Unbounded request threads with blocking reads |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: Any authenticated user can delete any link |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated arbitrary redirect endpoint |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts a client-controlled header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: GET /stats divides by zero when there are no links |
| RM06 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect lowercases case-sensitive short codes |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: Invalid TTL creates an immediately expired link |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Store mutex held across file and network I/O |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner is taken from the request instead of the session |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-security: Redirect targets permit HTTP header injection |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/store.rs:83 — Paging panics for ordinary list requests
- (medium) src/auth.rs:68 — Admin authorization condition rejects every role
- (medium) src/cache.rs:53 — Cache warming skips expired entries and leaves stale data
- (high) src/main.rs:216 — Report endpoint exposes all users' links without authorization
