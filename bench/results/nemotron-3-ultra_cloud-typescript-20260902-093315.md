# nemotron-3-ultra:cloud · typescript · run 20260902-093315 (repeat 1)

recall **25/63** · 32 finding(s), 3 unmatched · 630951 tokens · 1712s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | — | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu |  |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes valid (non-expired) links due to inverted  |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` code twice instead of validating ` |
| T04 | YES | parallel: C04 page clamps end past the array length; slice silently re | review-bugs: page() off-by-one: end = all.length + 1 includes extra eleme |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() returns 0 or 100 due to integer division befor |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() doesn't decrement owner quota despite comment promi |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending (fewest hits first) but comment says ' |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring match (includes) instead of exact e |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links despite comment saying 'expir |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns internal mutable object, not a snapshot |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() not atomic: partial failure leaves store modifie |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns remaining count instead of removed count |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt() crashes on missing code with non-null assertion |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | — | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam |  |
| T17 | — | parallel: S03 every issued session token is logged in cleartext |  |
| T18 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin incorrectly rejects admins due to indexOf misus |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on missing or malformed Authorization he |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: LRU eviction uses insertion order, not access order |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Timer leak: setTimeout timers never cleared on manual delete; review-concurrency: Cache timer leak and race condition on repeated set() |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() returns 0 for any cache size below limit due t |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for..in on Map, returning prototype properties n; review-concurrency: keys() uses for...in on Map, iterates prototype not entries |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | — | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t |  |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | — | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  |  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS multiplied by 1000 despite comment saying |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT parsed but never assigned to config |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when validation fails |
| T39 | — | parallel: EX01 the report name comes from the query string and is join |  |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() doesn't await async probes, returns immediately; review-concurrency: warmCache() doesn't await async probe/cache operations |
| T47 | — | TS-ONLY checkTargets is documented to report the first failure, but pr |  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() doesn't await write() or implement retry logic; review-concurrency: flushAudit() ignores returned promise from async write |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() returns NaN for unknown owners |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() crashes on missing 'code' query parameter |
| T54 | — | parallel: C17 deleteLink never checks the owner its doc promises to ch |  |
| T55 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| T56 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.ts:51 — handle() crashes on missing Authorization header via bearerToken
- (high) src/main.ts:117 — deleteLink() crashes on missing 'code' query parameter
- (high) src/worker.ts:27 — Janitor race: expiresAt() throws if link deleted between codes() and expiresAt()
