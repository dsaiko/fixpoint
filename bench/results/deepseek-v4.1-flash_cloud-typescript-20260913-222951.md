# deepseek-v4.1-flash:cloud · typescript · run 20260913-222951 (repeat 1)

recall **46/63** · 57 finding(s), 1 unmatched · 130466 tokens · 325s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: create() records NaN in the per-owner quota for a new owner |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() has the expiry test inverted: live links are kille |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() divides with integer floor, so it only ever re |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never gives the quota slot back, contradicting its  |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, so the leaderboard returns the least- |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches owners by substring, not identity; review-security: byOwner uses substring matching, leaking other owners' links |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, contradicting its documented |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object, not a snapshot |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: a validation failure leaves a pre |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining size, not the number removed |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes generated with Math.random() are predictable |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing secret when LINKD_SECRET is unset |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens are minted from Math.random(), so they are pr |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token written to the log in plaintext |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt, so sessions never expire; review-security: verify() never checks expiresAt, so sessions never expire |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin is inverted — every non-admin role is admitted; review-security: requireAdmin is inverted: every non-admin role is granted ad |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken throws on a missing Authorization header; review-security: bearerToken throws on a missing Authorization header, crashi |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals is a plain comparison, not the constant-time ch |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: eviction is FIFO, not LRU: the tracked `used` timestamp is n |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: Per-entry timers are never cleared: stale timers evict refre |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: delete() leaves the scheduled timer running, and `timers` gr |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors the ratio before scaling, so it reports |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map with for-in and always returns an empt |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates and returns the shared DEFAULT_CONFIG obj |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: A non-numeric LINKD_TIMEOUT_MS is reported as 'keeping defau |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is documented in milliseconds but multipl |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded, so the setting ne |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so a bad configuration is ne; review-security: validate() always returns true, so a rejected configuration  |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in the report name: arbitrary .csv write and  |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report file is world-readable despite the comment claiming o |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot's durability guarantee is not implemented, and; review-security: Snapshot written through a fixed, predictable /tmp path (sym |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache resolves before any probe finishes and drops probe; review-concurrency: warmCache never awaits its probes: returns before work finis |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets rejects instead of reporting the failing target; review-concurrency: checkTargets does not abandon the remaining probes as docume |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit drops the write promise: no retry, and failures b; review-concurrency: flushAudit drops the write promise: no retry despite the con |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for an owner with no recorded quot |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: JSON.parse of the request body sits outside the try/catch; review-security: JSON.parse of the request body is outside the try block, so  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() dereferences a missing ?code query parameter |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: deleteLink performs no ownership check, so any user can dele |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out forwards to any attacker-supplied URL |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: /admin/quotas authorizes on a client-supplied header, not th |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: CORS allows any origin, combined with Allow-Credentials |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response reflects the presented token back to the caller |
| T60 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports every owner's links to any authenticated use |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() reads a different filename than it wrote when ?name |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/main.ts:162 — Error response leaks the filesystem layout to the client
