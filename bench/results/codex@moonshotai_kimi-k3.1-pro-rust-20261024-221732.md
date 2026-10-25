# codex@moonshotai/kimi-k3.1-pro · rust · run 20261024-221732 (repeat 1)

recall **37/64** · 45 finding(s), 4 unmatched · 1070544 tokens · 509s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Creating the first link always panics |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolution removes live links and serves expired links |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: Final pages slice past the vector boundary |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: Success rate uses integer division before scaling |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not release its quota |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top links are sorted least-followed first |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner filtering returns substring matches |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link revives expired entries |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: Quota snapshots expose live mutable state |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: Failed imports leave partial batches committed |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: Prune returns the remaining count instead of removals |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are predictable from issuance time |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer tokens are written to logs |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Expired sessions remain valid indefinitely |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: Malformed or missing Authorization headers panic |
| RA07 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Cache eviction does not implement LRU |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Cache utilization is incorrect and can divide by zero |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Opposite cache lock order can deadlock |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: Sweep can delete an entry refreshed after its scan |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: Configuration validation always succeeds |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Report names permit path traversal and arbitrary CSV file ac |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV fields are written without escaping; review-security: CSV exports permit spreadsheet-formula injection |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: Archive write errors are silently discarded |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots share and corrupt one temporary file |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Resolved metric loses concurrent increments |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Janitor handle neither stops on drop nor waits for shutdown |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: Cache warming returns before its workers finish; review-concurrency: warm_cache returns without joining its workers |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner deadlocks by locking the same mutex twice |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: HTTP handler waits for EOF before responding; review-concurrency: Idle clients can retain request threads indefinitely |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: Any authenticated user can delete any link |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Arbitrary external redirects create an open-redirect endpoin |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is bypassed with a request header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: Stats panics while the store is empty |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Report requests hold the store lock across file and socket I |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link ownership is controlled by the request body |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/cache.rs:53 — peek reports expired entries as cached
- (medium) src/export.rs:49 — Snapshot rename failures are reported as success
- (high) src/main.rs:216 — Report endpoint exports every user's links without authorization
- (high) src/main.rs:251 — Unvalidated redirect values allow response-header injection
