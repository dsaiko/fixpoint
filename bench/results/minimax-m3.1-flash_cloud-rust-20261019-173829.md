# minimax-m3.1-flash:cloud · rust · run 20261019-173829 (repeat 1)

recall **38/64** · 55 finding(s), 6 unmatched · 470227 tokens · 3242s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics on every call: quota.get_mut(owner).unwrap() |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() treats a live link as expired and deletes it on th |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates the dest |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() clamps end to all.len() + 1 and slices out of bounds; review-concurrency: page() builds an end index of len+1 and slices out of bounds |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate() always returns 0 because the integer division |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending and returns the least-followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches on a substring instead of the exact owner |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives an expired link, contradicting its document |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the number of links remaining, not the numbe |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: new_code() derives the code from subsec_nanos only, so codes |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-bugs: The token generator is a seeded LCG, not an unguessable rand |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks the session's expiry |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() panics with an index-out-of-bounds on a missi |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always returns Err: the role check is a taut |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: The cache evicts an arbitrary entry rather than the least re |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() divides by zero when the cache limit is zero |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | — | RUST-ONLY flush locks hits then entries while get locks entries then h |  |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: sweep() reads a snapshot then deletes by key, so a concurren |
| RC07 | YES | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t | review-concurrency: utilization() divides by the cache limit, panicking under th |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A malformed LINKD_TIMEOUT_MS sets the timeout to zero instea |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is applied as seconds |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() always returns Ok, so cfg.validate().unwrap() can |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-bugs: The report name is not confined to export_dir, so ?name= can; review-security: Report filename comes from the query string and escapes the  |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: archive_all() swallows write errors and omits the CSV header |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot() uses a fixed temp path, so concurrent write |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-bugs: add_resolved() uses load-then-store instead of fetch_add and; review-concurrency: add_resolved does a non-atomic load-then-store, losing incre |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-bugs: The janitor checks its stop signal only before sleeping, so  |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache drops the JoinHandles, so it returns before the c |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner() deadlocks by locking the store mutex twice; review-concurrency: count_by_owner locks the same non-reentrant mutex twice and  |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: read_to_end has no socket timeout and no size cap; review-concurrency: Connections are read to EOF with no read deadline, so a stal |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-bugs: delete_link() does not enforce the ownership rule its own do; review-security: DELETE /links deletes any user's link; the owner is never ch |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out echoes the unvalidated next parameter in |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: admin_quotas() authorizes on a client-supplied header instea; review-security: Admin authorization is decided by a client-supplied X-Admin  |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: hits_per_link divides by zero and panics on an empty store; review-concurrency: stats() divides by the link count, so an empty store panics  |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: The 401 response echoes the presented token back to the clie; review-security: Rejected bearer token is reflected back to the client in the |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: The ttl request field is interpreted as hours while the defa |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: A panic under the store mutex poisons it, and every later re; review-concurrency: The /report handler holds the global store lock across file  |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-bugs: create_link() takes the link owner from the request body ins |
| RM17 | YES | JSON responses are built by string interpolation with no escaping, so  | review-bugs: The create_link response interpolates the target URL raw, pr |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/worker.rs:61 — The janitor deletes links that have not expired
- (medium) src/export.rs:49 — write_snapshot() discards the rename error and reports success when the snapshot was lost
- (low) src/main.rs:84 — The request body is rejoined with CRLF, so a bare-LF body is mis-parsed
- (high) src/cache.rs:45 — Lock-order inversion between Cache::get and Cache::flush can deadlock the cache
- (medium) src/worker.rs:59 — The janitor holds the store mutex for its whole O(n) sweep and again across cache.sweep()
- (medium) src/worker.rs:76 — Dropping JanitorStop never stops the janitor, despite the documented contract
