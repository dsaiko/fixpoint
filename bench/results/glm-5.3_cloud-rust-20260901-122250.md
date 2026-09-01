# glm-5.3:cloud · rust · run 20260901-122250 (repeat 1)

recall **39/64** · 50 finding(s), 1 unmatched · 664447 tokens · 351s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics for every request: quota map is never popula; review-concurrency: Store::create unwraps a missing quota entry under the store  |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: fresh links are deleted  |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() clamps end to len()+1 and panics whenever offset+size; review-concurrency: Store::page slice panics while holding the store mutex, pois |
| RS05 | — | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e |  |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot despite its co |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending and returns the LEAST followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches by substring, not equality; review-security: by_owner matches owners by substring, leaking other owners'  |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, contradicting its own contra |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import() is not atomic despite the all-or-nothing contract |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the surviving count, not the number removed |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: new_code() can collide, silently overwriting an existing lin; review-security: Short codes are just subsecond nanoseconds and are guessable |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are generated from a wall-clock-seeded LCG an |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued session tokens are written to stdout logs |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() ignores expires_at: sessions never expire; review-security: verify() never checks expires_at, so sessions never expire |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: hash_password is an unsalted, trivially-invertible hash pres |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() panics on any request without an Authorizatio; review-security: bearer_token panics on any Authorization header without a sp |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always rejects everyone; review-security: require_admin's condition is a tautology and always denies e |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | YES | RUST-ONLY get takes the WRITE lock on the read path, so every cache re | review-concurrency: Cache::get takes an exclusive lock for reads, serializing al |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: set() evicts an arbitrary entry, not the least recently used |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() divides before multiplying — result is only 0  |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Lock-order inversion between Cache::get and Cache::flush cau |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: Cache::sweep TOCTOU: a freshly re-cached entry can be delete |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | YES | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 | review-bugs: Unparsable LINKD_PORT becomes port 0 instead of keeping the  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds, not millisecon |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed and then thrown away |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() always returns Ok, so a bad configuration never f |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal in the report name allows arbitrary file writ |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Export report is written with default permissions despite th |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Snapshot uses a fixed predictable /tmp path, enabling a syml |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Metrics::add_resolved uses load+store instead of fetch_add,  |
| RW02 | — | the janitor checks for stop only before a 60s sleep, so shutdown waits |  |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache() discards the join handles and returns before th; review-concurrency: warm_cache detaches its probe threads and returns before the |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner() locks the same std Mutex twice and deadlock; review-concurrency: count_by_owner locks the same non-reentrant mutex twice, sel |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: handle() reads the request to EOF, so a keep-alive client ne; review-security: Unbounded read_to_end buffers the entire request before auth |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: delete_link performs no ownership check; any session can del |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out is an open redirect via the next parameter |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is a client-controlled X-Admin header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats endpoint divides by guard.len() and panics on an empty |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Every response carries Access-Control-Allow-Origin: * with A |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: Malformed ttl query parameter silently creates an immediatel |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/export.rs:49 — write_snapshot() ignores the rename result and reports success without a snapshot
