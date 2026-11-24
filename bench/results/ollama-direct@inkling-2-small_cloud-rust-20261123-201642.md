# ollama-direct@inkling-2-small:cloud · rust · run 20261123-201642 (repeat 1)

recall **50/64** · 72 finding(s), 0 unmatched · 544247 tokens · 339s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: create() unwraps quota entry that does not exist for new own; review-concurrency: create() panics on empty quota map while holding the store m |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverted expiry check deletes valid links and serv; review-security: resolve() expiry check is inverted — fresh links are deleted |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to`; review-security: rename() validates `from` twice and never validates `to` |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: page() sets end = len+1 and panics on slice out of bounds; review-concurrency: page() sets end = len + 1 and slices out of bounds, panickin |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: success_rate divides before multiplying, always yields 0 |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending and truncates, returning the least-fol |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner matches substrings instead of exact owner; review-security: by_owner matches substrings — returns other owners' links |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects expired links despite the documented fin |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: quotas() returns &mut to the live internal map, not a snapsh |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: import() is not atomic on validation failure |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune() returns the post-prune size, not the number removed |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: new_code is time-derived and create() overwrites colliding c; review-security: Short codes minted from sub-second nanoseconds are predictab |
| RA01 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' when LINKD |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-security: Session tokens are generated from a time-seeded LCG — predic |
| RA03 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to logs (and echoed in 401 responses) |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks session expiry; review-security: verify() never checks session expiry — issued tokens live fo |
| RA05 | YES | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi | review-security: Passwords hashed with an unsalted 64-bit djb2-style function |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: bearer_token panics on any Authorization header without a sp; review-security: bearer_token panics on missing/malformed Authorization heade |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin's role check is always true, rejecting everyon; review-security: require_admin's condition can never pass — and even the inte |
| RA08 | YES | RUST-ONLY secret_eq is documented as constant-time and is not: it retu | review-security: secret_eq is not constant-time; verify() doesn't use it anyw |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Eviction picks an arbitrary key, not the least recently used |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: utilization() truncates to 0/100 and divides by zero when li |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: flush() takes hits then entries; get() takes entries then hi |
| RC06 | YES | sweep collects expired keys under a read lock, drops it, then deletes: | review-concurrency: sweep() deletes by code without re-checking expiry — can evi |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to 0 despite 'keeping defa |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds, not millisecon |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: LINKD_FETCH_LIMIT is parsed and then thrown away |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: validate() collects problems but always returns Ok(()); review-security: Config::validate() prints problems but always returns Ok(()) |
| RE01 | YES | parallel: EX01 the report name comes from the query string and goes in | review-security: Path traversal in /report via unsanitized name (arbitrary fi |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Export reports containing all owners' data written with defa |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV fields are written without escaping; review-security: CSV injection / no field escaping in exports |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all path traversal via owner value |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: write_snapshot discards the rename result and never syncs; review-concurrency: write_snapshot uses a fixed shared temp path and ignores ren |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Metrics::add_resolved uses a non-atomic load-then-store, los |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Janitor shutdown is ineffective: stop() is unreachable, lags |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-bugs: warm_cache never joins its threads — returns before any prob; review-concurrency: warm_cache drops its JoinHandles and returns before any prob |
| RW04 | YES | parallel: X04 check_targets returns on the first error; the remaining  | review-bugs: check_targets leaks threads blocked on send after early retu |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: count_by_owner locks the same non-reentrant Mutex twice and ; review-concurrency: count_by_owner locks the same non-reentrant Mutex twice and  |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-security: Unbounded read_to_end on request body — memory-exhaustion Do |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-bugs: delete_link ignores the authenticated session and owner chec; review-security: delete_link performs no ownership or session check — any aut |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out?next= (no target validation) |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: /admin/quotas trusts a client-supplied x-admin header instea |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: stats endpoint divides by guard.len() with no zero check; review-concurrency: stats() divides by guard.len() with no zero check, panicking |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Access-Control-Allow-Origin: * combined with Access-Control- |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: Invalid ttl silently becomes 0 (instant expiry), and large v |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-bugs: handle() reads until EOF, deadlocking keep-alive clients; review-concurrency: Store mutex is held across blocking writes to the client soc |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: create_link takes owner from the request body instead of the |
| RM17 | YES | JSON responses are built by string interpolation with no escaping, so  | review-security: Unescaped user input interpolated into JSON response body |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-bugs: parse_query does not percent-decode keys/values; review-security: Stored CRLF injection in redirect target allows HTTP respons |
