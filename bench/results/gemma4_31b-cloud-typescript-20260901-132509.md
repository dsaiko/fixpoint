# gemma4:31b-cloud · typescript · run 20260901-132509 (repeat 1)

recall **17/63** · 23 finding(s), 4 unmatched · 283726 tokens · 175s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter results in NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted link expiry logic |
| T03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | — | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult |  |
| T06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| T07 | — | parallel: C10 top sorts ascending, and the comparator never returns 0, |  |
| T08 | — | parallel: C11 byOwner matches owners by substring, so one owner sees a |  |
| T09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| T10 | — | parallel: C14 quotas hands out the store's own object while the doc ca |  |
| T11 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| T12 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded Default Secret |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Predictable Session Tokens |
| T17 | — | parallel: S03 every issued session token is logged in cleartext |  |
| T18 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: Broken admin authorization check; review-security: Broken Admin Authorization Logic |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: Unsafe null dereference in bearerToken |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: Non-Constant-Time Secret Comparison |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: timer leak leads to premature cache eviction |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | — | TS-ONLY utilization floors the ratio before multiplying so it always r |  |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Incorrect Map iteration in keys() |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: Mutation of global default configuration |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | — | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  |  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: NaN configuration for cache TTL |
| T36 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar |  |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | — | parallel: CF06 validate logs the problems it finds and returns true, s |  |
| T39 | — | parallel: EX01 the report name comes from the query string and is join |  |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: Async callbacks in forEach not awaited; review-concurrency: async callbacks in forEach are not awaited |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-concurrency: failure to abandon probes in checkTargets |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-concurrency: floating promise in flushAudit |
| T49 | — | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning |  |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | — | parallel: C17 deleteLink never checks the owner its doc promises to ch |  |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open Redirect in Out Endpoint |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Trivial Admin Authentication via Header |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: Uncaught JSON parsing error |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/worker.ts:49 — unbounded parallel probes in warmCache
- (critical) src/main.ts:159 — Path Traversal in Report Export
- (high) src/main.ts:64 — Global Link Data Leakage
- (medium) src/store.ts:34 — Potential Prototype Pollution
