# nemotron-3-ultra:cloud · rust · run 20260902-085838 (repeat 1)

recall **39/64** · 55 finding(s), 5 unmatched · 665510 tokens · 3159s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics on first link for a new owner; review-security: quota.get_mut(owner).unwrap() panics for new owners |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() logic inverted: removes valid links, keeps expired |
| RS03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-security: page() has off-by-one bug causing panic on large size parame |
| RS05 | — | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e |  |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending (least hits first) but comment says 'm |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() uses substring match instead of exact match; review-security: by_owner uses substring match instead of exact match |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() doesn't enforce 'expiry is final' rule |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: quotas() returns mutable reference to internal state |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns remaining count instead of removed count |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-security: Short code generation uses only nanoseconds - collision pron |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Token generation uses predictable LCG seeded only with times |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() ignores token expiration |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: Password hashing uses insecure custom hash (djb2 variant) |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() panics on malformed/missing Authorization hea; review-security: bearer_token panics on malformed Authorization header |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always rejects due to logic error; review-security: Admin authorization check logic is inverted - always rejects |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | YES | verify scans every session linearly although the map is already keyed  | review-security: Token verification uses linear search instead of HashMap loo |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: set() eviction is arbitrary, not LRU |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() integer division returns 0 for partially fille; review-security: Cache utilization calculation broken by integer division |
| RC04 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: Check-then-act race in Cache.warm() |
| RC05 | — | RUST-ONLY flush locks hits then entries while get locks entries then h |  |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-bugs: sweep() has race condition between read lock drop and delete; review-concurrency: TOCTOU race in Cache.sweep() |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS parsed as seconds, not milliseconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT parsed but ignored |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() always returns Ok(()) even with errors |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-bugs: export_csv() path traversal via user-controlled name; review-security: Path traversal in export_csv via user-controlled filename |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: archive_all() path traversal via owner name |
| RE05 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| RE06 | YES | parallel: C21 the read error carries the server's filesystem path back | review-security: Path traversal in read_report via user-controlled filename |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-bugs: add_resolved() has race condition (load/store instead of fet; review-concurrency: Lost update race in Metrics.add_resolved() |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-bugs: start_janitor() stop signal delayed up to interval; holds lo |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: Thread leak in warm_cache() |
| RW04 | YES | parallel: X04 check_targets returns on the first error; the remaining  | review-bugs: warm_cache() doesn't wait for spawned threads; review-bugs: check_targets() can hang forever if a thread panics |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: Useless double lock in count_by_owner() |
| RM01 | — | parallel: C24 the handler reads the whole request body with no size li |  |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-bugs: delete_link() doesn't verify ownership |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect vulnerability in /out endpoint |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: admin_quotas() bypasses auth system's role check; review-security: Admin endpoint protected only by trivial header check |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats() division by zero when store is empty |
| RM06 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() lowercases code but store keys are case-sensitive |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS misconfiguration: wildcard origin with credentials allo |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Token leaked in error response and logs |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | YES | parallel: C26 janitor.stop() sits after an infinite accept loop, so it | review-concurrency: Janitor stop() unreachable after infinite loop |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: TTL parsing: invalid input becomes 0 (immediate expiry) |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/store.rs:83 — page() off-by-one causes panic on last page
- (medium) src/cache.rs:50 — peek() returns true for expired entries
- (medium) src/main.rs:212 — report() path traversal via user-controlled name
- (high) src/main.rs:49 — Unbounded thread spawning per connection
- (medium) src/config.rs:71 — LINKD_EXPORT_DIR used without path validation
