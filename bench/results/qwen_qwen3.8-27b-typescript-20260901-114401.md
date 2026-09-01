# qwen/qwen3.8-27b · typescript · run 20260901-114401 (repeat 1)

recall **44/63** · 55 finding(s), 0 unmatched · 322366 tokens · 1599s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: quota[owner]! + 1 is NaN on first create, so every quota rea |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check inverted: live links are deleted, exp; review-security: resolve() expiry test inverted: live links are deleted on fi |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to`; review-security: rename validates `from` twice and never validates `to` |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() floors the ratio, reporting 0% unless 100% of  |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never gives the owner their quota slot back |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() comparator is inverted: returns least-followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner matches by substring, not exact owner; review-security: byOwner matches owners by substring |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects already-expired links, contradicting 'ex; review-security: extend() resurrects expired links, contradicting its own con |
| T10 | — | parallel: C14 quotas hands out the store's own object while the doc ca |  |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic: a mid-batch failure leaves a partia |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining link count instead of the numb |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes minted with Math.random |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens generated with Math.random (predictable PRNG) |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Full session token written to the log on every issue |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() ignores expiresAt, so expired sessions verify forev |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin: indexOf truthiness inverts the admin gate; review-security: requireAdmin is inverted: non-admins pass, admins fail |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: hashPassword is unsalted single-iteration SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on a missing Authorization header; review-security: bearerToken throws on a missing Authorization header |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals compares with === despite claiming constant-tim |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction removes the oldest-inserted entry, not the le |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache timers are never cleared: unbounded growth and prematu |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-concurrency: Cache.delete does not clear the entry's TTL timer; a stale t |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() uses Math.floor on the ratio, reporting 0% unt |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for...in on a Map and always returns an empty ar |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates and returns the shared DEFAULT_CONFIG obj |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid LINKD_TIMEOUT_MS logs 'keeping default' but assigns  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS multiplied by 1000 although the doc says  |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied to the config |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so bad config never fails st; review-security: validate() always returns true — the startup gate never reje |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in exportCsv: user-controlled report name esc |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report file written world-readable despite the stated confid |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll interpolates the user-supplied owner directly int |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: writeSnapshot uses a fixed, world-writable /tmp path that fo |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport emits a spurious empty row for the trailing newl |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache resolves immediately because forEach does not awai; review-concurrency: warmCache resolves before any probe completes: forEach never |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets rejects instead of reporting the failing target; review-concurrency: checkTargets rejects instead of reporting the first failing  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit neither retries nor handles the write's promise; review-concurrency: flushAudit fires write() once and drops the promise: no retr |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for an owner with no links |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: createLink: JSON.parse outside the try block crashes on a ma |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect crashes when ?code= is missing |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: DELETE /links has no ownership check — any session deletes a; review-security: GET /links exposes every owner's links to any session |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out follows unvalidated ?next= as the 301 Lo |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: /admin/quotas authorized by client-controlled x-admin header |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: CORS: wildcard origin combined with credentials |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink takes the link owner from the request body instea |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |
