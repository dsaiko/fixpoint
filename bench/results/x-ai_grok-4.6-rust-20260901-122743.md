# x-ai/grok-4.6 · rust · run 20260901-122743 (repeat 1)

recall **38/64** · 44 finding(s), 3 unmatched · 217815 tokens · 1222s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create panics incrementing an absent quota |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve treats live links as expired |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the old code twice, never the new one |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page slices past the end and panics |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate is 0 or 100 because of integer division |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not restore the owner's quota |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns the least-hit links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner matches substrings |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import is not atomic |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune returns how many links remain, not how many it removed |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are a time-seeded LCG, not unguessable |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Fresh session tokens are printed to stdout |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify never enforces session expiry; review-security: verify() never enforces session expiry |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token panics on missing Authorization |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin rejects every session |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: set evicts an arbitrary key, not the LRU entry |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: cache utilization is 0 until the cache is full |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: flush and get take cache locks in opposite order |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: sweep can delete an entry that was refreshed after the snaps |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: bad LINKD_TIMEOUT_MS zeros the timeout |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is applied as seconds |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | — | parallel: CF06 validate logs the problems it finds and returns Ok, so  |  |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: User-controlled report name is joined onto export_dir withou |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Export files are created with default (world-readable) permi |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot reports success if rename fails; review-concurrency: write_snapshot races on a process-wide temp path |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: add_resolved loses increments under concurrent requests |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Janitor thread is never cancelled on handle drop |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache returns before the probes finish; review-concurrency: warm_cache returns without waiting for spawned setter thread |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner deadlocks on a non-reentrant Mutex |
| RM01 | — | parallel: C24 the handler reads the whole request body with no size li |  |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: Delete does not check the authenticated owner |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on GET /out copies ?next= into Location |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts the client-supplied X-Admin heade |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats divides by zero on an empty store |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Responses send CORS * together with Allow-Credentials: true |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented bearer token |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Store mutex is held across disk and socket I/O |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Create trusts client-supplied owner and ignores the session |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-security: Location header interpolates unsanitized URLs (response spli |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/export.rs:67 — report write and read disagree on the default filename
- (high) src/config.rs:113 — validate never fails
- (high) src/main.rs:216 — Report endpoint dumps every owner's links
