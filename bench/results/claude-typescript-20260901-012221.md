# claude · typescript · run 20260901-012221 (repeat 1)

recall **55/63** · 70 finding(s), 4 unmatched · 39499 tokens · 430s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter starts at NaN for every new owner |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: deletes live links, serv; review-security: resolve() inverts the expiry test, so expired links keep wor |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| T04 | YES | parallel: C04 page clamps end past the array length; slice silently re | review-bugs: page() computes an end index one past the array length |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() truncates to 0 or 100 for the same reason |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, returning the LEAST followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner uses substring matching, returning other owners' lin |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object, not the snapshot  |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic: a mid-batch validation failure leav |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the number of surviving links, not the numbe |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt() asserts non-null on an unknown code |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes come from Math.random() and are guessable |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret is used when LINKD_SECRET is unset |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens are generated from Math.random(), so they are |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens are written to stdout on every issue |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks session expiry; review-security: verify() never checks session expiry, so tokens are valid fo |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin uses indexOf result as a boolean, inverting the; review-security: requireAdmin grants access to every role except the admin ro |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords are stored as unsalted single-round SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken throws TypeError on a missing Authorization head; review-security: bearerToken throws on a missing or malformed Authorization h |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals is a short-circuiting comparison, not the const |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours returns NaN and omits the radix |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction is FIFO, not LRU; the recorded `used` timestamp is  |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: Orphaned expiry timer evicts a newer entry before its TTL |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: delete() leaves the scheduled timer behind; the timers map g; review-concurrency: Cache timers are never cleared: unbounded timers map and a h |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() truncates to 0 for any non-full cache |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map with for...in and always returns an em |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid LINKD_TIMEOUT_MS is logged but still assigned as NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 despite being docum |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so a bad configuration still |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in exportCsv/readReport writes and reads arbi |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Export written with default permissions despite the comment  |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are interpolated without escaping, corrupting row |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll writes to a path built from the attacker-controll |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot renames across filesystems, which fails with E; review-security: Snapshot is staged at a fixed, predictable path in world-wri |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport emits a bogus trailing row and undefined cells f |
| T45 | YES | the errors counter is reported by snapshot and never incremented anywh | review-bugs: Metrics.errors is reported but never incremented |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache resolves before any probe completes and drops reje; review-concurrency: warmCache resolves before any probe runs and leaks unhandled |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets throws instead of reporting the failing target; review-concurrency: checkTargets rejects instead of returning the failing target |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit ignores the returned promise: no retry, no error ; review-concurrency: flushAudit discards the write promise: no await, no catch, n |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for an owner with no links |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-bugs: startJanitor never evicts pruned codes from the cache it is  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: JSON.parse of the request body is outside the try block |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() dereferences a possibly-missing ?code query param |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink performs no ownership check despite documenting o; review-security: deleteLink performs no ownership check despite documenting o |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out sends a 301 to any attacker-supplied URL |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: /admin/quotas authorises on a client-supplied header instead |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: GET /stats divides by zero when the store is empty |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Wildcard CORS combined with Allow-Credentials on authenticat |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: The 401 response echoes the presented token back to the clie |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink takes the link owner from the request body, not t |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() writes one filename and reads back another when ?na |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/store.ts:24 — newCode() can collide or produce a code shorter than the 4-char minimum
- (low) src/cache.ts:37 — peek() reports expired entries as present, so warm() skips refreshing them
- (low) src/config.ts:103 — validate accepts a cache size of zero, which breaks the Cache
- (low) src/main.ts:32 — Janitor interval handle discarded, leaving no way to stop the background sweep
