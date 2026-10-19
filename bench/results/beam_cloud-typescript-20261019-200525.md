# beam:cloud · typescript · run 20261019-200525 (repeat 3)

recall **50/63** · 66 finding(s), 6 unmatched · 100854 tokens · 76s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: create() increments quota from undefined, yielding NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() has inverted expiry logic; review-security: resolve() serves expired links |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() floors before multiplying by 100 |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not refund the owner's quota slot |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending instead of descending |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring match instead of exact owner equali; review-security: byOwner uses substring match leaking other owners |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() exposes the internal quota map (mutable alias); review-security: quotas() leaks mutable internal state |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns size instead of number removed |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt() non-null assertion can throw |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes generated with Math.random() |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing/application secret |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens minted with Math.random() |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token written to stdout |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks session expiry; review-security: verify() never enforces expiry |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin uses indexOf result as a boolean; review-security: requireAdmin rejects admins and admits owners |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashed with unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken dereferences a possibly undefined header; review-security: bearerToken non-null dereference can throw |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals uses non-constant-time comparison |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours returns NaN for bad input |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction picks insertion order, not least recently used |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: delete() does not clear the scheduled timer (resource leak); review-concurrency: Stale expiry timers can delete a freshly cached entry |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors before multiplying, always 0 or 100 |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() always returns an empty array |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid LINKD_TIMEOUT_MS still overwrites timeoutMs with NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 though documented a |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report name |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Sensitive report world-readable |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll writes files named by owner |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Predictable /tmp snapshot path enables race |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport produces a trailing empty row |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() does not await probes and never resolves when do; review-concurrency: warmCache resolves before the parallel probes finish |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets() calls probe() which throws on insecure target; review-concurrency: checkTargets does not abandon remaining probes on failure |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() ignores the returned promise and retries nothin; review-concurrency: flushAudit neither awaits nor retries the write |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() adds 0 to possibly undefined quota |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-concurrency: Janitor interval is never cleared |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: createLink() parses JSON outside the try block |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: DELETE /links has no ownership check |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via ?next= in /out |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated on spoofable header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats() divides by store.size(), which can be zero |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Permissive CORS with credentials enabled |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown-token error echoes the presented token |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink trusts owner from request body |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() reads `${name}.csv` but export writes `links.csv` f |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/cache.ts:27 — Expired entries are never evicted by get()'s TTL check... actually expiry check is correct
- (medium) src/cache.ts:37 — peek() reports expired entries as present
- (medium) src/main.ts:51 — handle() calls bearerToken() with a possibly missing header
- (medium) src/auth.ts:62 — getSecret() exposes the signing secret
- (medium) src/main.ts:162 — Verbose internal error and path disclosure
- (medium) src/store.ts:149 — importAll accepts unvalidated link objects
