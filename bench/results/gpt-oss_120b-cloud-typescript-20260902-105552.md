# gpt-oss:120b-cloud · typescript · run 20260902-105552 (repeat 1)

recall **31/63** · 44 finding(s), 4 unmatched · 1269574 tokens · 406s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota count becomes NaN for first link of an owner; review-security: Quota counter can become NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiration check inverted; review-security: Expiration logic inverted in resolve() |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: `rename` validates the source code twice, never validates th |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: `successRate` always rounds to 0% unless every link is used |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: `delete` does not adjust owner quota; review-security: Delete does not update per‑owner quota |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: `top` returns lowest‑hit links instead of highest |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: `byOwner` matches owners by substring |
| T09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: `quotas` returns a mutable reference |
| T11 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: `prune` returns the remaining size instead of removed count |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Predictable session tokens |
| T17 | — | parallel: S03 every issued session token is logged in cleartext |  |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Session verification ignores expiration; review-security: Token verification ignores expiration |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin always returns true for non‑admin roles; review-security: Incorrect admin role check |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on missing or malformed header |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: Timing‑attack‑prone secret comparison |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction does not follow LRU policy |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: `Cache.delete` leaves TTL timer running |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: `utilization` integer division yields 0 until cache is full |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: `keys` iterates over Map with `for…in` and returns an empty  |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: `loadConfig` mutates the shared DEFAULT_CONFIG object |
| T33 | YES | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so | review-security: Invalid timeout environment variable results in NaN |
| T34 | — | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  |  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Cache TTL environment variable multiplied twice |
| T36 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar |  |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: `validate` always returns true even on errors; review-security: Configuration validation always returns true |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in CSV export |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Path traversal in per‑owner archive |
| T43 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: `warmCache` does not await async operations; review-concurrency: `warmCache` returns before async work completes |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: `checkTargets` fails to handle probe rejections; review-concurrency: `checkTargets` does not short‑circuit and mishandles probe f |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-concurrency: `flushAudit` ignores the asynchronous write operation |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: `countForOwner` returns NaN for owners without a quota entry |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Delete endpoint lacks ownership verification |
| T55 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin quota endpoint trusts only an `x-admin` header; review-security: Admin endpoint trusts client header without authentication |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report endpoint reads a different file than it writes |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/store.ts:134 — `extend` resets expiry instead of extending it
- (high) src/main.ts:51 — `bearerToken` called without checking Authorization header
- (medium) src/main.ts:97 — Redirect handler assumes `code` query param exists
- (critical) src/export.ts:60 — Path traversal when reading exported reports
