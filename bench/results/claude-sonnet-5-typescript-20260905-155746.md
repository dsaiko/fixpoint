# claude-sonnet-5 · typescript · run 20260905-155746 (repeat 1)

recall **31/63** · 39 finding(s), 2 unmatched · 53095 tokens · 510s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter becomes NaN forever for any new owner |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() condition is inverted: fresh links are deleted, ex; review-security: Link expiry check is inverted: expired links resolve forever |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates the dest |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() has the same divide-then-floor bug, almost alw |
| T06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, returning the least-followed links in |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches owners by substring instead of exact equal |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() ignores the "already-expired links stay expired" in |
| T10 | — | parallel: C14 quotas hands out the store's own object while the doc ca |  |
| T11 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining count instead of the removed c |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short link codes generated with a non-cryptographic PRNG |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens generated with a non-cryptographic PRNG |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token logged in plaintext |
| T18 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin uses indexOf as a boolean, inverting the access; review-security: requireAdmin's indexOf check is inverted: denies real admins |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses unsalted single-round SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on any request without an Authorization ; review-security: Missing Authorization header crashes the request handler ins |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Re-setting the same code leaves a stale timer that can delet; review-concurrency: Stale expiry timer races with a refreshed cache entry and de |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() computes percentage in the wrong order, collap |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map with for...in and always returns an em |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG object instead  |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: LINKD_TIMEOUT_MS assigns NaN even after logging that the def |
| T35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu |  |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied to the config |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when it found problems |
| T39 | — | parallel: EX01 the report name comes from the query string and is join |  |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport() manufactures a bogus trailing row from the fin |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache does not wait for probes despite its documented co; review-concurrency: warmCache resolves before its probes finish and drops reject |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets propagates probe()'s throw instead of reporting |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit neither retries nor observes failures of the audi; review-concurrency: flushAudit fires the write without awaiting, retrying, or ha |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for an owner with no recorded quot |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: DELETE /links has no ownership check — any authenticated use |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out via unvalidated `next` parameter |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated only by a client-supplied header |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Permissive CORS: wildcard origin combined with credentials a |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner comes from the request body, not the authenticate |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/main.ts:32 — startJanitor's stop function is discarded, leaking an uncancellable interval
- (critical) src/main.ts:157 — Path traversal via the /report `name` parameter allows arbitrary file write and read
