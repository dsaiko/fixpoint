# ollama-codex@beam:cloud · typescript · run 20261028-120155 (repeat 1)

recall **48/63** · 64 finding(s), 4 unmatched · 647651 tokens · 163s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: quota counter becomes NaN for a new owner |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve deletes valid links and keeps expired ones; review-security: Expired links are resolved instead of rejected |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates `from` twice and never validates `to` |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | — | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult |  |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not decrement the owner quota |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top sorts ascending instead of descending |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner uses substring matching instead of exact equality; review-security: Owner filter uses substring matching |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas leaks the mutable internal object |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns the remaining size, not the removed count |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt dereferences a missing link |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes generated from non-cryptographic randomness |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing secret |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens use non-cryptographic randomness |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token written to logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify ignores session expiry; review-security: Session expiration is never enforced |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin rejects every admin by using indexOf's zero ret; review-security: Admin authorization check is inverted |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken dereferences possibly missing header; review-security: bearerToken can throw on missing or malformed Authorization |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: Secret comparison is not constant-time |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours can return NaN |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction is not LRU |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: delete does not cancel the expiry timer |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization always reports 0 until completely full |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() always returns an empty array |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: bad timeout value is still assigned |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: cache TTL is scaled by 1000 despite being declared milliseco |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true; review-security: Configuration validation always succeeds |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report file name |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Export files are written with default world-readable permiss |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-security: CSV output is not escaped |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Predictable temporary snapshot path |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport emits a spurious empty trailing row |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache returns before probes/cache writes finish; review-concurrency: warmCache is fire-and-forget: never awaits or surfaces async |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets does not abort and rejects on probe throw; review-concurrency: checkTargets waits for all probes despite claiming early aba |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit neither awaits nor retries writes; review-concurrency: flushAudit discards the write promise and never retries |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for owners with no quota |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: JSON.parse is outside the try block in createLink |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink does not enforce ownership; review-security: Missing ownership check on link deletion |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out endpoint |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: adminQuotas trusts the x-admin header instead of the session; review-security: Admin endpoint trusts a client-supplied header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats divides by zero when the store is empty |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Permissive CORS configuration with credentials |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner is taken from the request body |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exports all owners' links without authorizat |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report uses mismatched filename for empty name |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/main.ts:51 — handle crashes when Authorization header is missing
- (high) src/main.ts:97 — redirect dereferences missing query code
- (medium) src/store.ts:88 — successRate floors before converting to a percentage
- (medium) src/cache.ts:45 — Cache TTL timers are never cleared, so stale timers can delete live entries
