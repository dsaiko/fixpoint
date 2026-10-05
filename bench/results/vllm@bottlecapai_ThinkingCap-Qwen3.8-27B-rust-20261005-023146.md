# vllm@bottlecapai/ThinkingCap-Qwen3.8-27B · rust · run 20261005-023146 (repeat 1)

recall **34/64** · 46 finding(s), 4 unmatched · 51160 tokens · 832s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() panics for every owner: quota map is never seeded; review-security: create() panics via quota.get_mut(owner).unwrap() for every  |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry test: deletes live links, serve; review-security: Inverted expiry check: expired links resolve forever, valid  |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() clamps end to len+1, causing an out-of-bounds slice p; review-security: page() slices offset..len+1 and panics on large ?size= (GET  |
| RS05 | — | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e |  |
| RS06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending and returns the least-followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches by substring, not exact owner |
| RS09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| RS10 | — | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  |  |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the remaining count, not the number removed |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: new_code() derives the code from subsec_nanos, so same-nanos; review-security: Short codes are the clock's subsec-nanos: predictable, leadi |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing secret when LINKD_SECRET is unset |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens from a time-seeded LCG are predictable |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Full session token written to stdout on issuance |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks expires_at, so expired sessions stay v |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: hash_password is an unsalted 32-bit djb2 |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token panics when the Authorization header is missing; review-security: bearer_token panics on any Authorization header that is not  |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() condition is always true, so it rejects ever |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq is not constant-time (early return) and verify use |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Eviction picks an arbitrary entry, not the least recently us |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() divides before multiplying, and panics when li |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | — | RUST-ONLY flush locks hits then entries while get locks entries then h |  |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is read as milliseconds but applied as se |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() always returns Ok, so no bad configuration is eve |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal in report name allows file write/read outside |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | — | parallel: EX03 CSV rows are built with format!, so a comma or quote in |  |
| RE04 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot() swallows the rename error and still returns |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Metrics::add_resolved loses increments: non-atomic load-then |
| RW02 | — | the janitor checks for stop only before a 60s sleep, so shutdown waits |  |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache returns without joining its worker threads despit |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner locks the same non-reentrant Mutex twice and  |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-security: Unbounded request body read enables unauthenticated OOM of t |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-bugs: delete_link does not enforce the documented owner-only restr; review-security: delete_link performs no ownership check; any user can delete |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out: open redirect and CRLF header injection via unvalidate |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: /admin/quotas authorization is a client-controlled x-admin h |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: GET /stats divides by zero on an empty store; review-concurrency: Handler panic while holding the store lock poisons the mutex |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows all origins with credentials |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: ttl * 3600 overflows on large ttl values |
| RM15 | — | every handler writes its HTTP response while still holding the global  |  |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner taken from user input instead of the authenticate |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | — | parse_query never percent-decodes, and the POST body is parsed as a qu |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/cache.rs:45 — Lock-order inversion between Cache::get (entries then hits) and Cache::flush (hits then en
- (low) src/worker.rs:70 — JanitorStop documents drop-to-stop but has no Drop impl; the janitor thread leaks if the h
- (high) src/main.rs:216 — /report leaks every user's links to any authenticated caller
- (medium) src/main.rs:46 — Bearer tokens transmitted in cleartext on 0.0.0.0 with no TLS
