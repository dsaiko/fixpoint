# gpt-5.6-terra · rust · run 20260907-143202 (repeat 1)

recall **26/64** · 30 finding(s), 2 unmatched · 409473 tokens · 408s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Creating the first link for an owner panics |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolution treats live links as expired |
| RS03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: Partial final pages slice past the end of the link list |
| RS05 | — | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e |  |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| RS08 | — | parallel: C11 by_owner matches owners by substring, so one owner sees  |  |
| RS09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: Import violates its atomicity guarantee |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: Prune returns the remaining count rather than removed count |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are deterministic and predictable |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer tokens are written to application logs |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Expired sessions remain valid indefinitely |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: Missing or malformed Authorization headers panic the handler |
| RA07 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | — | parallel: CA08 eviction drops whatever key iteration yields first, not |  |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Cache utilization is wrong and panics for a zero-sized cache |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Cache flush can deadlock with cache get |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: Configuration validation never rejects invalid values |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Report names allow traversal outside the export directory |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-concurrency: Concurrent reports with the same name race on one file |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots corrupt the shared temporary file |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Resolved metric increments are lost under concurrency |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Dropping the janitor handle leaks its worker thread |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache returns before its spawned work completes |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner self-deadlocks on the store mutex |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: Request handling waits for the client to close the connectio; review-concurrency: Each connection can permanently consume a handler thread |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: Any authenticated user can delete another owner's link |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The out endpoint is an unrestricted open redirect |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is bypassable with a request header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: Stats panics for an empty store |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | YES | export_csv defaults an empty name to links.csv but read_report is hand | review-bugs: Unnamed reports are written under a different name than they |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.rs:40 — The server starts with no usable authentication sessions
- (medium) src/export.rs:49 — Snapshot publication failures are reported as success
