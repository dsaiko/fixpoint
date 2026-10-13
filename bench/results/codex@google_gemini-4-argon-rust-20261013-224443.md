# codex@google/gemini-4-argon · rust · run 20261013-224443 (repeat 1)

recall **50/64** · 64 finding(s), 5 unmatched · 1477091 tokens · 501s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Store::create panics with unwrap on None when updating quota |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Store::resolve has an inverted expiration comparison that de |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Store::rename validates `from` twice and never validates `to; review-security: Store::rename validates from twice and never validates the d |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: Store::page sets slice end to all.len() + 1 and panics out o |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: Store::success_rate always returns 0 for partial hit rates d |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Store::delete does not decrement the owner's quota count |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Store::top sorts ascending by hits and returns the least fol |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Store::by_owner matches owners by substring instead of exact; review-security: Store::by_owner uses substring matching instead of exact equ |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Store::extend revives expired links and overwrites expires_a |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: Store::quotas returns a mutable reference to internal state  |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: Store::import is not atomic and partially mutates the store  |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: Store::prune returns the number of remaining links instead o |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-security: new_code generates predictable short link codes from subsec_ |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are generated with a timestamp-seeded LCG ins |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Plaintext bearer session tokens are written to stdout logs |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Authenticator::verify never checks session.expires_at; review-security: Authenticator::verify never checks session expiration and us |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: hash_password uses an unsalted 64-bit DJB2 non-cryptographic |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token panics on empty or single-token Authorization h; review-security: bearer_token panics on missing or malformed Authorization he |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Authenticator::require_admin uses || instead of && and rejec; review-security: Broken role condition in require_admin rejects all sessions |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq returns early on the first mismatched byte, enabli |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Cache::set evicts an arbitrary HashMap key instead of the le |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Cache::utilization truncates to 0 due to integer division an |
| RC04 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: TOCTOU race in Cache::warm between peek and set |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: ABBA lock-ordering deadlock between Cache::get and Cache::fl |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: TOCTOU race in Cache::sweep can delete freshly updated cache |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: load_config sets timeout to 0ms on invalid LINKD_TIMEOUT_MS  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is converted with Duration::from_secs ins |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: load_config discards parsed LINKD_FETCH_LIMIT instead of set |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: Config::validate always returns Ok(()) even when validation  |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal in export_csv and read_report allows arbitrar |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: export_csv creates sensitive multi-owner report files with d |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-security: Unescaped fields in export_csv and archive_all allow CSV and |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot ignores fs::rename errors and renames across ; review-concurrency: Concurrent write_snapshot calls race on fixed temporary file |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Lost updates in Metrics::add_resolved due to non-atomic load |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-bugs: start_janitor leaks the background thread when JanitorStop i; review-concurrency: Janitor thread leaks when JanitorStop is dropped and blocks  |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache detaches spawned threads instead of joining them  |
| RW04 | YES | parallel: X04 check_targets returns on the first error; the remaining  | review-concurrency: check_targets uses zero-capacity channel and retains sender  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: Self-deadlock in count_by_owner from locking non-reentrant M |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: Server::handle calls read_to_end on the TCP stream, hanging ; review-concurrency: Missing socket timeouts and unbounded thread spawning leak h |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: delete_link allows any authenticated user to delete any othe |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in GET /out via unvalidated next query paramet |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization check trusts client-supplied X-Admin req |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: Server::stats divides by zero when the store is empty |
| RM06 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Server::redirect lowercases the link code before resolving |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS origin combined with Access-Control-Allow-Cred |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response reflects the caller's raw bearer token in the r |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Global store Mutex is held across disk and network I/O in re |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: create_link accepts the link owner from the untrusted reques |
| RM17 | YES | JSON responses are built by string interpolation with no escaping, so  | review-security: Unescaped target URL in create_link response enables JSON in |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/export.rs:36 — archive_all silently ignores file write errors
- (high) src/export.rs:67 — read_report does not default an empty report name to "links" like export_csv
- (medium) src/cache.rs:53 — Cache::peek reports expired entries as present
- (high) src/main.rs:251 — CRLF injection / HTTP response splitting in redirect_to Location header
- (low) src/main.rs:223 — Error responses in report leak internal server filesystem paths and OS errors
