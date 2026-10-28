# ollama-codex@glm-5.3-flash:cloud#max · rust · run 20261028-123119 (repeat 1)

recall **41/64** · 56 finding(s), 5 unmatched · 1713233 tokens · 493s

| seed | found | note | matched by |
|---|---|---|---|
| RS01 | YES | RUST-ONLY (parallel: C01) quota has no entry for a new owner, so get_m | review-bugs: First link creation panics on missing quota entry; review-concurrency: A panic while holding the store lock poisons the shared stor |
| RS02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expired-link test is inverted; review-security: Link expiry is implemented backwards |
| RS03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the wrong destination twice |
| RS04 | YES | parallel: C04 page clamps end past the slice length, so the last page  | review-bugs: Pagination slice is out of bounds |
| RS05 | — | RUST-ONLY (parallel: C05) usize division before scaling yields 0 for e |  |
| RS06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete does not restore the owner quota |
| RS07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns least-followed links |
| RS08 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner performs substring matching |
| RS09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Expired links can be revived by extend |
| RS10 | YES | RUST-ONLY (parallel: C14) quotas hands out a mutable reference to the  | review-bugs: Quota snapshot shares mutable store state |
| RS11 | YES | parallel: C13 import inserts as it validates, so a bad code leaves the | review-bugs: Import is not atomic |
| RS12 | YES | parallel: C15 prune computes the before-count, throws it away, and ret | review-bugs: prune reports remaining links instead of removed links |
| RS13 | — | RUST-ONLY expires_at unwraps a missing code and panics instead of retu |  |
| RS14 | YES | RUST-ONLY (parallel: S02) short codes come from the clock's nanosecond | review-bugs: Short codes lack collision retry; review-security: Short codes are predictable sub-second timestamps |
| RA01 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| RA02 | YES | RUST-ONLY (parallel: S02) session tokens come from a hand-rolled LCG s | review-bugs: Two sessions minted in the same second have the same token; review-security: Session tokens are predictable LCG output |
| RA03 | — | parallel: S03 every issued session token is printed in cleartext |  |
| RA04 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions are still accepted; review-security: Expired session tokens remain valid |
| RA05 | — | RUST-ONLY (parallel: S05) passwords are hashed with a djb2-style 64-bi |  |
| RA06 | YES | RUST-ONLY (parallel: S06) bearer_token indexes parts[1] without checki | review-bugs: Missing Authorization header causes indexing panic; review-security: Missing Authorization header crashes the handler |
| RA07 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Admin role check rejects every valid role |
| RA08 | — | RUST-ONLY secret_eq is documented as constant-time and is not: it retu |  |
| RA09 | — | verify scans every session linearly although the map is already keyed  |  |
| RC01 | — | RUST-ONLY get takes the WRITE lock on the read path, so every cache re |  |
| RC02 | YES | parallel: CA08 eviction drops whatever key iteration yields first, not | review-bugs: Eviction is not least-recently-used |
| RC03 | YES | RUST-ONLY utilization divides usizes before scaling so it is always 0, | review-bugs: Cache utilization calculation is wrong |
| RC04 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: Cache maintenance races with concurrent cache writes |
| RC05 | YES | RUST-ONLY flush locks hits then entries while get locks entries then h | review-concurrency: Inverted lock order between Cache::get and Cache::flush can  |
| RC06 | — | sweep collects expired keys under a read lock, drops it, then deletes: |  |
| RC07 | — | RUST-ONLY every lock() is unwrapped, so one panicking thread poisons t |  |
| RF01 | — | parallel: CF01 a malformed LINKD_PORT silently becomes port 0 |  |
| RF02 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| RF03 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: Cache TTL milliseconds are read as seconds |
| RF04 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and validated, then thrown  |  |
| RF05 | — | RUST-ONLY (parallel: CF05) any read error -- not just a missing file - |  |
| RF06 | YES | parallel: CF06 validate logs the problems it finds and returns Ok, so  | review-bugs: Configuration validation always succeeds |
| RE01 | — | parallel: EX01 the report name comes from the query string and goes in |  |
| RE02 | YES | parallel: EX02 the export holding every owner's links is written with  | review-security: Sensitive export files use default permissions |
| RE03 | YES | parallel: EX03 CSV rows are built with format!, so a comma or quote in | review-bugs: CSV fields are not escaped or quoted; review-security: CSV output permits spreadsheet formula injection |
| RE04 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Owner value controls archive filename |
| RE05 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Snapshot uses a predictable /tmp file and ignores rename fai |
| RE06 | — | parallel: C21 the read error carries the server's filesystem path back |  |
| RW01 | YES | RUST-ONLY (parallel: X01) add_resolved does load-then-store instead of | review-concurrency: Resolved metric updates can be lost |
| RW02 | YES | the janitor checks for stop only before a 60s sleep, so shutdown waits | review-concurrency: Dropping JanitorStop can leak the janitor thread |
| RW03 | YES | RUST-ONLY (parallel: X03) warm_cache drops its JoinHandles without joi | review-concurrency: warm_cache returns before spawned workers complete |
| RW04 | — | parallel: X04 check_targets returns on the first error; the remaining  |  |
| RW05 | YES | RUST-ONLY (parallel: CA04) count_by_owner locks the same mutex a secon | review-bugs: Reentrant lock attempt guarantees deadlock; review-concurrency: count_by_owner attempts to lock the same non-recursive mutex |
| RM01 | YES | parallel: C24 the handler reads the whole request body with no size li | review-bugs: HTTP request handling waits for client EOF; review-security: Request bodies are read without any size limit |
| RM02 | YES | parallel: C17 delete_link never checks the owner its doc promises to c | review-security: Deletion has no ownership or privilege check |
| RM03 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out |
| RM04 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts a client-controlled header |
| RM05 | YES | parallel: C20 /stats divides by the link count, panicking whenever the | review-bugs: Stats panic when no links exist |
| RM06 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| RM07 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Credentials-enabled wildcard CORS |
| RM08 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| RM09 | — | RUST-ONLY the auth path calls bearer_token on an absent Authorization  |  |
| RM10 | YES | parallel: C26 janitor.stop() sits after an infinite accept loop, so it | review-concurrency: Connections can leak request threads indefinitely |
| RM11 | — | parallel: C25 no read or write timeout is ever set on an accepted stre |  |
| RM13 | YES | export_csv defaults an empty name to links.csv but read_report is hand | review-bugs: Default report name is written and read inconsistently |
| RM14 | — | a bad ttl parses to 0 and mints an already-expired link, and a large v |  |
| RM15 | YES | every handler writes its HTTP response while still holding the global  | review-concurrency: Store mutex is held across file and network I/O |
| RM16 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner is client-controlled |
| RM17 | — | JSON responses are built by string interpolation with no escaping, so  |  |
| RM12 | YES | parse_query never percent-decodes, and the POST body is parsed as a qu | review-bugs: URL-encoded values are never decoded; review-security: Redirect target allows CRLF response injection |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/cache.rs:60 — Zero-capacity cache can grow without bound
- (medium) src/export.rs:49 — Snapshot rename failures are ignored
- (critical) src/main.rs:213 — Report filename permits path traversal and file overwrite/read
- (high) src/main.rs:216 — Any authenticated user can export every owner's links
- (high) src/main.rs:150 — Link listing ignores ownership and pagination limits
