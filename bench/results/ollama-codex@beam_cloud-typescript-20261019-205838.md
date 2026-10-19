# ollama-codex@beam:cloud · typescript · run 20261019-205838 (repeat 1)

recall **50/63** · 72 finding(s), 11 unmatched · 1190609 tokens · 173s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: create() computes quota with undefined base |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes non-expired links and keeps expired ones |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice and never validates 'to' |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() is always 0 or 100 |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not decrement owner quota |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending instead of descending |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring matching instead of exact ownership; review-security: byOwner leaks links via substring matching |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() exposes the internal quota object; review-concurrency: quotas() returns the live internal quota object instead of a |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic and skips quota updates |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns cache size instead of removed count |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt() dereferences possibly missing links |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes use predictable randomness |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing secret |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens use predictable randomness |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Issued session token written to server logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() does not enforce session expiry; review-security: Expired sessions are never rejected |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin() condition is inverted; review-security: Admin authorization check is backwards |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken() dereferences possibly undefined values; review-security: Missing Authorization header crashes request parsing |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: Secret comparison is not constant-time |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours() can return NaN |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction does not select the least recently used entry |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache timers are never cleared |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() is always 0 until the cache is full |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() always returns an empty array |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig() mutates the shared DEFAULT_CONFIG object |
| T33 | YES | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so | review-bugs: Invalid numeric environment values propagate as NaN |
| T34 | — | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  |  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 despite being milli |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal via report name |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report files world-readable despite comment |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-security: CSV fields are not escaped |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Path traversal via owner name in archiveAll |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: writeSnapshot uses a fixed temp path shared across concurren; review-security: Fixed shared temporary file in writeSnapshot |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() does not await its async work; review-concurrency: warmCache does not await concurrent probes before resolving |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets() does not stop on first failure; review-concurrency: checkTargets does not short-circuit or handle thrown probes |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-concurrency: flushAudit fires write() without awaiting or retrying |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() produces NaN for unknown owners |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: JSON.parse is outside the try block |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Non-numeric payload.ttl produces NaN link expiry |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink() does not enforce ownership; review-security: Any authenticated user can delete any link |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quotas endpoint authorized by spoofable header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats() divides by zero when store is empty |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: CORS allows credentials with a wildcard origin |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown token value reflected in 401 response |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner is trusted from the request body |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() reads a different filename than exportCsv() writes  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/cache.ts:37 — peek() reports expired entries as cached
- (medium) src/main.ts:25 — validate() result is ignored at startup
- (high) src/main.ts:51 — Missing/invalid auth header crashes request handling
- (medium) src/main.ts:96 — redirect() assumes query.code is present
- (medium) src/worker.ts:70 — flushAudit() does not retry and ignores errors
- (medium) src/worker.ts:28 — startJanitor() misses links expiring exactly at now
- (high) src/cache.ts:45 — Cache timers are never cleared, so stale timeouts delete newer entries
- (medium) src/auth.ts:62 — getSecret exposes the raw service secret
- (medium) src/main.ts:102 — Redirect targets are not sanitized
- (high) src/store.ts:149 — importAll bypasses validation and quota accounting
- (low) src/config.ts:244 — loadFile reads an arbitrary supplied path
