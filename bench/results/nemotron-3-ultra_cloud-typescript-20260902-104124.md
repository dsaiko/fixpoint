# nemotron-3-ultra:cloud · typescript · run 20260902-104124 (repeat 2)

recall **43/63** · 65 finding(s), 5 unmatched · 1037274 tokens · 3041s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota becomes NaN for new owners (undefined + 1 = NaN) |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes non-expired links due to inverted expirati; review-concurrency: resolve() TOCTOU between expiry check and delete |
| T03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| T04 | YES | parallel: C04 page clamps end past the array length; slice silently re | review-bugs: page() off-by-one error: end = all.length + 1 |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() uses integer division, returns only 0 or 100 |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() doesn't decrement owner quota |
| T07 | — | parallel: C10 top sorts ascending, and the comparator never returns 0, |  |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring match (includes) instead of exact m; review-security: byOwner uses partial match (includes) instead of exact match |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, violating 'expiry is final'  |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-concurrency: quotas() returns internal mutable object allowing external c |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() not atomic, no validation, no quota update |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns remaining count instead of removed count |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt() crashes with non-null assertion on missing code |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes generated with Math.random() - predictable |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret 'dev-secret-do-not-use' |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens generated with Math.random() - not cryptograp |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens logged in plaintext |
| T18 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| T19 | YES | TS-ONLY verify uses loose equality on the token, so type coercion deci | review-bugs: Loose equality (==) used for token comparison in verify(); review-security: Loose equality (==) in token verification |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin() rejects admin users due to indexOf returning ; review-security: Admin role check inverted - admins are rejected |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses unsalted SHA-256 - vulnerable to rainb |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken() crashes on missing or malformed Authorization ; review-security: bearerToken crashes on missing/malformed Authorization heade |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-bugs: secretEquals() claims constant-time but uses === (timing att; review-security: Timing attack in secretEquals - uses === |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours() returns NaN on invalid input, no valida |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction picks arbitrary first entry, not LRU (used ti |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Timer memory leak: setTimeout timers never cleared on delete; review-concurrency: delete() doesn't clear timer from timers Map |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() integer division always returns 0 until full |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for...in on Map (doesn't iterate values); review-concurrency: keys() uses for...in on Map instead of Map.keys() |
| T31 | YES | parallel: CA06 warm's peek and set are separate operations, so a concu | review-concurrency: warm() has TOCTOU race between peek() and set() |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig() mutates DEFAULT_CONFIG directly (shared referen |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | — | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  |  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: cacheTtlMs multiplied by 1000 incorrectly (ms -> microsecond; review-security: LINKD_CACHE_TTL_MS multiplied by 1000 but comment says alrea |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT parsed but never assigned to config |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when problems exist; review-security: validate() always returns true even when validation fails |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal via user-controlled report name |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV generation doesn't escape values - injection/corruption |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Path traversal via owner name in archiveAll |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: writeSnapshot uses predictable /tmp path - race condition |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() doesn't await async probes - resolves immediatel; review-concurrency: warmCache() fire-and-forget promises never awaited |
| T47 | — | TS-ONLY checkTargets is documented to report the first failure, but pr |  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() doesn't await write or retry (comment says it r; review-concurrency: flushAudit() ignores returned promise from write() |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() returns NaN for unknown owners |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() crashes on missing 'code' query parameter |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink() missing ownership check - any user can delete a; review-security: Missing ownership check in deleteLink - any user can delete  |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: out() open redirect vulnerability - no URL validation; review-security: Open redirect in /out endpoint |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin check via spoofable x-admin header |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: CORS misconfiguration - wildcard origin with credentials |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Token exposed in 401 error response |
| T60 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/store.ts:188 — remove() doesn't check existence or update quota
- (high) src/main.ts:51 — handle() crashes on missing Authorization header via bearerToken()
- (medium) src/store.ts:20 — Quota accounting inconsistent across create/delete/prune/importAll
- (high) src/export.ts:60 — Path traversal in readReport via user-controlled name
- (medium) src/store.ts:149 — importAll accepts unvalidated Link objects
