# muse-spark-open:cloud · typescript · run 20261016-020423 (repeat 1)

recall **5/63** · 6 finding(s), 0 unmatched · 221714 tokens · 85s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | — | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu |  |
| T02 | — | parallel: C02 expiry comparison inverted: live links are deleted, expi |  |
| T03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | — | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult |  |
| T06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| T07 | — | parallel: C10 top sorts ascending, and the comparator never returns 0, |  |
| T08 | — | parallel: C11 byOwner matches owners by substring, so one owner sees a |  |
| T09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-concurrency: quotas() returns internal mutable object allowing unsynchron |
| T11 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| T12 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | — | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam |  |
| T17 | — | parallel: S03 every issued session token is logged in cleartext |  |
| T18 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | — | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an |  |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | — | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  |  |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: Cache.set overwrites entry without cancelling prior timer ca; review-concurrency: Cache.delete does not clearTimeout leaving dangling timer an |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | — | TS-ONLY utilization floors the ratio before multiplying so it always r |  |
| T30 | — | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl |  |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | — | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t |  |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | — | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  |  |
| T35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu |  |
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
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-concurrency: warmCache uses forEach with async callback and never awaits  |
| T47 | — | TS-ONLY checkTargets is documented to report the first failure, but pr |  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-concurrency: flushAudit fire-and-forget promise leak with no retry or err |
| T49 | — | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning |  |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-concurrency: startJanitor TOCTOU crash and never-stopped interval leak |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
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
