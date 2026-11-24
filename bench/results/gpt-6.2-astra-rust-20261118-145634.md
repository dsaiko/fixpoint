# gpt-6.2-astra · rust · run 20261118-145634 (repeat 1)

recall **42/64** · 62 finding(s), 12 unmatched · 277011 tokens · 284s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: Creating the first link for an owner panics after inserting  |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Renaming a link validates the source twice and accepts an in |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: A partial final page slices past the end of the link list |
| RS05 | YES | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e | review-bugs: The success rate reports zero for every partially used store |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Per-owner counts drift as links are deleted, expired, or imp |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard selects the least-followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup returns substring matches |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive an expir |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: The quota snapshot allows callers to modify live store count |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: A rejected import leaves earlier batch entries in the store |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: Pruning returns the number of survivors |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: Generated codes can overwrite an existing link |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | — | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s |  |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: A missing Authorization header crashes the request handler; review-security: A missing Authorization header triggers a worker panic |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: The admin role check rejects every administrator |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Cache eviction ignores the recorded access times |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Cache utilization truncates partial occupancy and panics at  |
| RC04 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: Cache warming can overwrite a newer target |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Cache hits and flushes acquire the same locks in opposite or |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A malformed timeout replaces the documented default with zer |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL configuration interprets milliseconds as seconds |
| RF04 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  | review-bugs: A valid fetch-limit override is discarded |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: Configuration validation reports success after finding error |
| RE01 | — | parallel: EX01 the report name comes from the query string and goes in |  |
| RE02 | — | parallel: EX02 the export holding every owner's links is written with  |  |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV exports lose column boundaries for ordinary field values |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: Owner archives can report success after a write failure |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot publication reports success even when rename fails; review-bugs: Snapshot success does not guarantee crash durability |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Concurrent redirects lose resolved-counter increments |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-bugs: Dropping the janitor handle leaves the worker running indefi; review-concurrency: Dropping the janitor handle leaves the worker running |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: Cache warming returns before its worker threads finish |
| RW04 | YES | parallel: X04 check_targets returns on the first error; the remaining  | review-concurrency: Target checks require one live OS thread per target |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-concurrency: count_by_owner locks the store twice on the same thread |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: The server waits for the client to close before answering an; review-concurrency: Request handlers wait indefinitely for clients to close thei |
| RM02 | — | parallel: C17 delete_link never checks the owner its doc promises to c |  |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Permanent redirects allow repeat clicks to bypass counting |
| RM04 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: Reading stats on an empty store divides by zero; review-concurrency: An empty-store stats request poisons the mutex for subsequen |
| RM06 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect lookup changes the case of valid stored codes |
| RM07 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| RM08 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Authentication failures copy bearer credentials into respons |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | YES | parallel: C26 janitor.stop() sits after an infinite accept loop, so it | review-concurrency: The accept loop creates an unbounded number of handler threa |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | YES | export_csv defaults an empty name to links.csv but read_report is hand | review-bugs: Downloading the default report reads a different file from t; review-concurrency: Concurrent service instances can return each other's CSV rep |
| RM14 | YES | a bad ttl parses to 0 and mints an already-expired link, and a large v | review-bugs: Invalid TTL values become zero or overflow during conversion |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Report and response I/O hold the global store mutex |
| RM16 | — | the link owner is taken from the request body instead of the authentic |  |
| RM17 | YES | JSON responses are built by string interpolation with no escaping, so  | review-bugs: Link creation can return malformed JSON |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-bugs: Encoded query and form values are stored literally |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.rs:93 — No request can authenticate after startup
- (high) src/main.rs:192 — The quota endpoint drops every owner's link count
- (high) src/main.rs:158 — The link listing returns only the page length
- (high) src/main.rs:243 — The server labels CSV and plain-text responses as JSON
- (medium) src/store.rs:213 — The generator can issue codes rejected by the store's own validator
- (medium) src/store.rs:35 — Target validation accepts URLs with no host
- (medium) src/main.rs:216 — A successful report silently omits links beyond the first 1,000
- (medium) src/cache.rs:60 — Updating a cached key can evict an unrelated entry
- (medium) src/cache.rs:53 — Expired cache entries prevent warming their replacements
- (medium) src/main.rs:52 — Persistent accept errors make the listener spin
- (medium) src/cache.rs:105 — A sweep can delete an entry that another thread just refreshed
- (critical) src/main.rs:55 — Idle connections can exhaust threads and terminate the listener
