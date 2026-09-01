# qwen/qwen3.8-max · rust · run 20260901-101312 (repeat 1)

recall **44/64** · 57 finding(s), 3 unmatched · 74446 tokens · 1888s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() unwraps a quota entry that is never inserted — ever; review-security: Every link creation panics: quota.get_mut(owner).unwrap() on |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: deletes valid links, ser; review-security: Inverted expiry test: expired links resolve forever, live li |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() clamps end to all.len() + 1, so the slice panics; review-security: page() off-by-one clamps end to len+1, panicking on slice |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate() integer division truncates to 0 in almost all |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending and returns the LEAST followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() uses substring match instead of exact owner equal |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links despite the documente |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: quotas() returns &mut HashMap — callers can mutate live stor |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import() inserts as it validates — a mid-batch failure leave |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the remaining link count instead of the numb |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: new_code() derives codes solely from sub-second nanos — coll; review-security: Link codes derived only from sub-second nanos are enumerable |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are deterministic from the issue second (LCG  |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are written to stdout/logs on issue |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks expires_at — sessions never expire; review-security: verify() never enforces session expiry |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() indexes parts[1] unconditionally — panics on ; review-security: bearer_token panics on missing/malformed Authorization heade |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() condition is always true — every caller is r |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Eviction picks an arbitrary HashMap key instead of the LRU e |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() computes len/limit*100 — integer truncation yi |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Lock-order inversion between Cache::get (entries→hits) and C |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to 0 instead of keeping th |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is documented as milliseconds but parsed  |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded — the overrid |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() collects problems but always returns Ok — startup |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal via ?name= in report export and download |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Exports written world-readable despite 'service user only' c |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV export performs no field escaping |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: archive_all() ignores per-row write errors and reports succe |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot() ignores the rename result and never fsyncs  |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: add_resolved does a non-atomic load-then-store on an AtomicI |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Dropping JanitorStop disconnects the channel and makes the j |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache() drops the join handles — returns before probes ; review-concurrency: warm_cache spawns threads but never joins them, violating it |
| RW04 | YES | parallel: X04 check_targets returns on the first error; the remaining  | review-bugs: check_targets() early return leaves sender threads blocked f |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner locks the same non-reentrant Mutex twice — gu |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: handle() uses read_to_end — request is not processed until t; review-security: Unbounded request read with no timeout (memory/slowloris DoS |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: DELETE /links deletes any link regardless of owner (IDOR) |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on GET /out with no scheme/host validation |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated by client-supplied x-admin header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats() divides by the number of links — panics on an empty ; review-concurrency: Panic while holding Mutex<Store> poisons the lock; every han |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: ttl hours-to-secrets conversion can overflow u64 |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link ownership taken from request body, not the authenticate |
| RM17 | YES | JSON responses are built by string interpolation with no escaping, so  | review-security: JSON injection in create response: target echoed without esc |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-security: HTTP response splitting: stored link target written verbatim |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.rs:55 — Unbounded thread-per-connection with no socket timeout lets idle clients pin handler threa
- (low) src/worker.rs:65 — Janitor holds the global store lock while running cache.sweep()
- (high) src/main.rs:216 — /report returns every owner's links to any authenticated user
