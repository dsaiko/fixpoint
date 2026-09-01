# minimax-m3:cloud · rust · run 20260901-101516 (repeat 1)

recall **28/64** · 37 finding(s), 3 unmatched · 742483 tokens · 551s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics on the first link from any owner; review-security: Quota increment panics on unknown owner |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expired check and deletes valid links |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the wrong code twice |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() panics when the page overflows the store |
| RS05 | — | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e |  |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() ignores the quota decrement the docstring promises |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-followed links, not the most |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches owner names by substring |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, contradicting the docstring |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: quotas() exposes a mutable handle to the internal counter ma |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the remaining count, not the removed count |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback HMAC/session secret |
| RA02 | — | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s |  |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: hash_password uses DJB2, not a cryptographic hash |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-security: bearer_token panics on a malformed Authorization header |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() rejects every session, including admins; review-security: require_admin role check is logically broken |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | — | parallel: CA08 eviction drops whatever key iteration yields first, not |  |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() is always zero because of integer division ord |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | — | RUST-ONLY flush locks hits then entries while get locks entries then h |  |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is parsed as seconds, not milliseconds |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() accepts every configuration, including invalid on |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal via user-supplied ?name= on /report |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot() reports success when the atomic move failed; review-security: Predictable temp path /tmp/linkd-snapshot.json enables symli |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-bugs: Metrics::add_resolved is racy and loses updates |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: JanitorStop does not actually stop the sweep on drop |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache() returns before the warm threads finish; review-concurrency: warm_cache returns before its worker threads finish |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner() deadlocks by holding one lock and taking an; review-concurrency: count_by_owner deadlocks on a non-reentrant Mutex |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-security: Unbounded read_to_end on TCP stream causes memory exhaustion; review-security: HTTP parser only handles CRLF line endings |
| RM02 | — | parallel: C17 delete_link never checks the owner its doc promises to c |  |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= with no allowlist |
| RM04 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| RM05 | — | parallel: C20 /stats divides by the link count, panicking whenever the |  |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Contradictory CORS: wildcard origin with credentials |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response reflects the user-supplied token |
| RM09 | YES | RUST-ONLY the auth path calls bearer_token on an absent Authorization  | review-bugs: bearer_token() panics on requests without an Authorization h |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/main.rs:223 — /report conflates export and read errors in the response body
- (medium) src/cache.rs:33 — Cache::get and Cache::flush take entries/hits in opposite orders
- (medium) src/store.rs:34 — Target URL validation is prefix-only, allows private/internal hosts
