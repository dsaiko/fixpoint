# gpt-6.2-sol · typescript · run 20261118-142740 (repeat 1)

recall **44/63** · 74 finding(s), 15 unmatched · 245300 tokens · 839s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: The first link sets its owner's quota count to NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it; review-security: Resolving an active code deletes it and expired codes can st |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Renaming validates the source twice and accepts invalid dest |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: The success rate rounds every partial result to zero |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Removing or importing links leaves quota counts out of sync |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: The leaderboard returns the least-followed links first |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup includes partial name matches |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending expiry can shorten a live link or revive an expire |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: Editing a quota snapshot changes the store's counters |
| T11 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Pruning reports the number of survivors instead of removals |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Bearer tokens use a noncryptographic random generator |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session issuance logs usable bearer credentials |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Sessions remain valid and retained past their expiry; review-security: Expired bearer tokens remain valid |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: The admin role fails the administrator check; review-security: The admin authorization helper grants access to unlisted rol |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing allows cheap offline guessing |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: A missing Authorization header throws before the 401 respons; review-security: Requests without an Authorization header throw before authen |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction ignores recent accesses and evicts on updates |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache timers accumulate and can delete refreshed entries; review-concurrency: An old expiry timer can delete a replacement cache entry |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Partial cache utilization is always reported as zero |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: The cache key list is empty even when entries exist |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: Loading configuration permanently changes the defaults |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: A malformed timeout replaces the default with NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Cache TTL overrides are multiplied by 1,000 |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT never changes the configured limit |
| T37 | YES | parallel: CF05 loadFile swallows every error, so an unreadable config  | review-bugs: Unreadable configuration files silently fall back to default |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Invalid configuration passes startup validation |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report names allow writes outside the export directory |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Generated reports can be readable by other local users |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV exports corrupt fields containing commas or line breaks; review-security: User-supplied CSV cells can execute spreadsheet formulas |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot writes fail across filesystem boundaries; review-bugs: Snapshot writes return before the replacement is durable |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: Parsing a generated CSV produces an extra empty row |
| T45 | YES | the errors counter is reported by snapshot and never incremented anywh | review-bugs: The dashboard error count never increases |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: Cache warmup loses probe failures; review-concurrency: Cache warming detaches probe failures from its returned prom |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: Target checking throws instead of identifying a failed targe; review-concurrency: A rejected probe bypasses the batch's failure result |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: Audit flushing drops failed writes without retrying; review-concurrency: Audit flush discards the in-flight write promise |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: An owner without links is counted as NaN |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-bugs: Janitor cleanup leaves expired link targets in the cache |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Invalid TTL input can create a link the janitor never expire; review-security: Invalid TTLs let callers retain links indefinitely |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: A redirect without a code throws |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Any authenticated caller can delete another owner's link; review-security: Listing and reporting disclose other owners' links |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Cached outbound redirects can skip click counting |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A client-controlled header grants access to admin quotas |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: An invalid owner can leave a link stored after a 400 respons; review-security: Link creators can impersonate another owner |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: An unnamed report is read from the wrong file |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.ts:52 — The handler has no way to accept a valid session
- (high) src/store.ts:152 — A rejected import leaves earlier changes in the store
- (medium) src/cache.ts:48 — Zero capacity and an empty-string victim break the cache bound
- (medium) src/cache.ts:38 — Cache warming skips entries that have already expired
- (medium) src/main.ts:85 — Malformed JSON and null bodies escape link creation error handling
- (medium) src/main.ts:107 — Negative page sizes can return nearly the entire store
- (medium) src/main.ts:159 — Reports silently omit links after the first 1,000
- (medium) src/main.ts:97 — The redirect handler never uses the resolver cache
- (medium) src/main.ts:32 — Importing the request handler starts a timer that cannot be stopped
- (medium) src/store.ts:21 — The store accepts targets that are not absolute URLs
- (medium) src/store.ts:33 — A generated code collision silently replaces an existing link
- (medium) src/worker.ts:41 — Target probes report unreachable HTTPS addresses as reachable
- (medium) src/main.ts:159 — Parallel report requests can return another request's CSV
- (medium) src/main.ts:159 — Report file I/O blocks unrelated requests and timers
- (medium) src/main.ts:32 — The janitor interval keeps the process alive after shutdown
