# gpt-6.2-astra · typescript · run 20261118-150254 (repeat 1)

recall **45/63** · 79 finding(s), 15 unmatched · 181745 tokens · 338s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: An owner's first link sets their quota count to NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it; review-security: Resolving a live link deletes it, while an expired link stil |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and accepts an invalid des |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: Partial success rates are reported as zero |
| T06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: The leaderboard returns the least-followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner filtering matches substrings; review-security: Owner lookup crosses account boundaries on partial matches |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can revive it or shorten its remaining life |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: The quota snapshot exposes the store's mutable counters |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: A failed batch import leaves earlier records committed |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Successful imports bypass owner accounting; review-bugs: Prune returns the number of surviving links |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens use predictable randomness |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session issuance writes reusable bearer tokens to logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Expired sessions remain live and accumulate indefinitely; review-security: Expired bearer tokens continue to authenticate |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: The admin-role check rejects an administrator; review-security: The administrative role helper grants access to unknown role |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password storage uses fast, unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: A missing Authorization header throws before the handler can; review-security: An unauthenticated request can throw out of the request hand |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction ignores accesses and can evict on an update; review-bugs: A zero-capacity cache stores entries |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache timers accumulate and can delete replacement entries; review-concurrency: An old expiry timer can delete a refreshed cache entry |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache utilization reports zero at every partial occupancy |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: The cache key listing is empty for a populated cache |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: Loading configuration mutates the defaults and earlier resul |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: A malformed timeout replaces the default with NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: The cache TTL override is multiplied by 1,000 twice in effec |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: A valid fetch limit override is discarded |
| T37 | YES | parallel: CF05 loadFile swallows every error, so an unreadable config  | review-bugs: Configuration file read failures are silently treated as a m |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Invalid configuration passes validation and startup continue |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: A report name can direct writes outside the export directory |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report files can be readable by other local users |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields do not round-trip through export and parsing; review-security: Unescaped report fields allow forged CSV rows and spreadshee |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot writes return before the replacement is durable; review-concurrency: Concurrent snapshot writers share one temporary file |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: Parsing a generated report adds a nonexistent final row |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: Cache warming reports success when a probe fails; review-concurrency: warmCache drops the promises for its per-link work |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: Target checking throws instead of returning the failed targe; review-concurrency: checkTargets starts every probe before observing the first f |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: A transient audit write failure drops the operation; review-concurrency: flushAudit returns while the audit write is pending |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: An owner with no links gets a NaN count |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-bugs: The dashboard error counter never changes; review-bugs: The janitor leaves expired links in the resolver cache |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed create requests escape the request handler |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Invalid TTL input can create a link that never expires; review-security: Malformed lifetimes create links that never expire |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: A redirect request without a code throws; review-bugs: Redirects change the case of valid short codes |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Any authenticated caller can delete another owner's link; review-security: The link listing exposes every owner's destinations and code |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: The permanent outbound redirect can bypass click counting; review-security: The email redirect accepts arbitrary destinations |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: The caller can grant itself access to administrative quotas |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: An empty store produces a null hits-per-link statistic |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link creation lets callers impersonate another owner |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: An unnamed report is read from a different file than the one |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.ts:29 — The request handler has no way to recognize an issued session
- (high) src/store.ts:110 — Removing a link leaves it counted against its owner
- (medium) src/store.ts:33 — A generated code collision silently overwrites an existing link
- (medium) src/store.ts:21 — The absolute URL check accepts targets with no host
- (medium) src/cache.ts:38 — An expired entry can block cache warming
- (medium) src/config.ts:50 — A nonnumeric cache size disables the capacity bound
- (medium) src/export.ts:45 — Snapshot replacement fails across filesystems
- (medium) src/main.ts:159 — Reports silently omit every link after the first 1,000
- (medium) src/worker.ts:41 — The reachability probe succeeds for every HTTPS-prefixed string
- (high) src/main.ts:32 — Importing the request handler starts a janitor that cannot be stopped
- (medium) src/main.ts:160 — A report request can read another worker's export
- (medium) src/main.ts:159 — Report file I/O blocks unrelated requests on the event loop
- (high) src/main.ts:159 — The report endpoint returns other owners' records
- (high) src/main.ts:159 — Unbounded report creation can exhaust the service's disk
- (low) src/main.ts:162 — Export errors disclose server filesystem details
