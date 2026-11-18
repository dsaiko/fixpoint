# gpt-6.2-sol · rust · run 20261118-141536 (repeat 1)

recall **37/64** · 58 finding(s), 15 unmatched · 529428 tokens · 639s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Creating the first link for an owner panics after inserting  |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and accepts invalid destin |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: Requesting a partial page panics |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: Success rate reports zero for every partially followed set |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Owner quotas drift as links are removed or imported |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard returns the least-followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup matches substrings instead of exact owners |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive it |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: Editing the quota snapshot changes live store counts |
| RS11 | — | parallel: C13 import inserts as it validates, so a bad code leaves the |  |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: Prune returns the number of surviving links |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | — | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond |  |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-bugs: Issuing two sessions in one second overwrites the first |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: Missing or incomplete Authorization headers panic; review-security: Missing or malformed Authorization headers panic the request |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: The admin-role helper always rejects administrators |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: A full cache evicts an arbitrary entry even on some replacem |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Cache utilization truncates partial occupancy and panics at  |
| RC04 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Cache hits and flush acquire locks in opposite orders |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A malformed timeout overrides the default with zero |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: Valid fetch-limit overrides are discarded |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: Configuration validation accepts values it explicitly reject |
| RE01 | — | parallel: EX01 the report name comes from the query string and goes in |  |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV fields containing commas, quotes, or line breaks corrupt; review-concurrency: Report readers can observe a file while it is overwritten |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: Owner archival reports success after a failed write |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot writes can report success without publishing a dura; review-concurrency: Concurrent snapshot writers share one temporary file |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Concurrent resolves lose counter increments |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-bugs: Dropping the janitor handle leaves the janitor running; review-concurrency: Dropping the janitor handle leaves its thread running |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: Cache priming returns before its workers finish |
| RW04 | YES | parallel: X04 check_targets returns on the first error; the remaining  | review-concurrency: Target checks keep one thread per input alive until all are  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: Counting links attempts to reacquire the same mutex |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-concurrency: Connections can exhaust request workers before authenticatio; review-security: Unauthenticated requests can exhaust process memory |
| RM02 | — | parallel: C17 delete_link never checks the owner its doc promises to c |  |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: A permanent tracking redirect can bypass later click counts |
| RM04 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: Stats divides by zero when the store is empty; review-concurrency: A panic in stats disables all subsequent store users |
| RM06 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect lookup changes valid mixed-case codes |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | — | parallel: C26 janitor.stop() sits after an infinite accept loop, so it |  |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | — | export_csv defaults an empty name to links.csv but read_report is hand |  |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: Malformed or oversized TTL values create invalid expirations |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Report I/O holds the global store mutex |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | YES | JSON responses are built by string interpolation with no escaping, so  | review-bugs: Link creation can return malformed JSON after saving the lin |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-bugs: Query and form fields are never URL-decoded |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) src/store.rs:47 — A short-code collision silently overwrites an existing link
- (high) src/main.rs:66 — HTTP request handling waits for the client to close its write side
- (high) src/main.rs:93 — No client can authenticate to any route
- (high) src/export.rs:67 — A report requested without a name reads a different file from the one it writes
- (high) src/main.rs:243 — CSV exports and plain-text bodies are labeled as JSON
- (high) src/main.rs:192 — The quotas endpoint omits every owner's quota
- (medium) src/main.rs:237 — Both response writers emit a malformed HTTP status line
- (medium) src/store.rs:35 — A scheme prefix alone passes absolute URL validation
- (medium) src/store.rs:153 — A rejected import leaves the earlier items in the store
- (medium) src/cache.rs:53 — Expired entries prevent cache warming from refreshing them
- (medium) src/worker.rs:100 — Target probes succeed without checking the destination
- (medium) src/cache.rs:105 — Cache sweep can delete a freshly refreshed entry
- (medium) src/cache.rs:87 — Cache warm-up can overwrite a fresher target
- (medium) src/export.rs:61 — Concurrent audit appends can merge records
- (critical) src/main.rs:55 — Unauthenticated connections can exhaust threads and stop the server
