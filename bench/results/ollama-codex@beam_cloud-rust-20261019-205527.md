# ollama-codex@beam:cloud · rust · run 20261019-205527 (repeat 1)

recall **46/64** · 58 finding(s), 2 unmatched · 1729409 tokens · 109s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics for every new owner; review-security: create() dereferences missing quota entry (panic / DoS) |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry check |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() can panic on an out-of-bounds slice; review-security: page() can slice past the end of the vector |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate() uses integer division before multiplying |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never releases the owner quota |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending instead of descending |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches substrings, not exact owners; review-security: by_owner uses substring match, leaking other owners' links |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives an expired link |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: quotas() exposes a mutable internal map despite the snapshot |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import() is not atomic and skips duplicate checks |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the remaining count, not the removed count |
| RS13 | YES | RUST-ONLY expires_at unwraps a missing code and panics instead of retu | review-bugs: expires_at() panics on an unknown code |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-security: Short codes derived from nanosecond timestamp (predictable) |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens generated from predictable time-seeded PRNG |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued bearer token written to stdout |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() ignores token expiry; review-security: verify() never checks token expiry |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: Weak unsalted password hash (hash_password) |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token() panics on missing token; review-security: bearer_token indexes parts[1] without bounds check |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin rejects every session |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq is not constant-time |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Entry eviction is not LRU |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() underflows to 0 due to integer division |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Cache::flush lock order is inverted against Cache::get |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: Cache::sweep can delete a freshly re-inserted entry |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to zero |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is parsed as seconds, not milliseconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() always returns Ok |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal via report name in export_csv/read_report |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Export file created with default world-readable permissions |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-security: CSV injection in exported report |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Predictable shared temporary snapshot path |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Metrics::add_resolved loses increments under concurrency |
| RW02 | — | the janitor checks for stop only before a 60s sleep, so shutdown waits |  |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache() returns before spawned threads finish; review-concurrency: warm_cache returns before its spawned threads finish |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner deadlocks on a second lock acquisition; review-concurrency: count_by_owner acquires the same non-reentrant mutex twice |
| RM01 | — | parallel: C24 the handler reads the whole request body with no size li |  |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-bugs: delete_link ignores the authenticated owner; review-security: delete_link performs no ownership or authorization check |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated open redirect in /out |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: admin_quotas trusts a client header instead of the session; review-security: Admin quota endpoint trusts client-supplied x-admin header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats() divides by zero when the store is empty; review-security: stats() divides by zero when store is empty |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS origin combined with credentials |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unauthenticated token value echoed in 401 response |
| RM09 | YES | RUST-ONLY the auth path calls bearer_token on an absent Authorization  | review-bugs: bearer_token() panics on malformed or missing Authorization |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: report holds the store mutex across file I/O |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner taken from request payload instead of session |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/export.rs:49 — write_snapshot ignores rename failure
- (low) src/cache.rs:51 — peek() reports expired entries as cached
