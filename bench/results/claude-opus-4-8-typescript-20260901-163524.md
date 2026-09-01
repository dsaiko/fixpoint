# claude-opus-4-8 · typescript · run 20260901-163524 (repeat 1)

recall **35/63** · 41 finding(s), 0 unmatched · 30778 tokens · 332s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: First link per owner sets quota to NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry condition is inverted; review-security: resolve() expiry check is inverted: expired links resolve, v |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source code twice and never validates t |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate collapses to 0 via integer division |
| T06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, returning the least-followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner matches owners by substring; review-security: byOwner uses substring match, matching unrelated owners |
| T09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic despite its contract |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns remaining size instead of removed count |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes generated with Math.random() are guessable |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens generated with Math.random() are predictable |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token written to logs and reflected in error respons |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks token expiry; review-security: verify() never checks token expiry, so sessions never expire |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin uses indexOf as a boolean; review-security: requireAdmin authorization check is inverted, granting admin |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on missing/malformed header |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: Cache timers are never cleared: stale timer evicts a refresh |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: delete() never clears the scheduled timer, leaking timers |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization collapses to 0 via integer division |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for-in over a Map and returns nothing |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Bad LINKD_TIMEOUT_MS is logged but still assigned as NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS multiplied by 1000 |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true |
| T39 | — | parallel: EX01 the report name comes from the query string and is join |  |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: writeSnapshot uses a predictable /tmp path (symlink TOCTOU) |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache resolves before its probes finish; review-concurrency: warmCache uses async forEach: resolves before probes finish  |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets throws instead of reporting the failing target; review-concurrency: checkTargets cannot abandon probes and rejects instead of re |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-concurrency: flushAudit drops the write() promise: no await, no retry, un |
| T49 | — | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning |  |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: deleteLink performs no ownership check (IDOR) |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quotas endpoint authorized by spoofable request header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats divides by zero on an empty store |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: CORS allows any origin together with credentials |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink takes owner from the request body instead of the  |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-security: Path traversal in report name reaches exportCsv/readReport |
