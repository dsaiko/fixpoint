# beam-ollama-direct · typescript · run 20261028-114604 (repeat 1)

recall **43/63** · 79 finding(s), 6 unmatched · 1506537 tokens · 341s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota increment produces NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve deletes valid (non-expired) links; review-security: resolve() deletes live links and keeps expired ones |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-security: rename() validates from twice instead of to |
| T04 | YES | parallel: C04 page clamps end past the array length; slice silently re | review-bugs: page slices one past end; review-security: page() returns one extra element |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate always 0 for partial usage; review-security: successRate() returns only 0 or 100 |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not return quota slot |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top returns least-followed links; review-security: top() sorts ascending instead of descending |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner uses substring match; review-security: byOwner() matches substrings |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links; review-security: extend() revives expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas returns mutable backing object; review-security: quotas() returns the live internal object |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-security: importAll() is not atomic |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-security: prune() returns remaining count instead of removed count |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Insecure fallback session secret; review-security: Predictable session token generation |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-bugs: Session token written to logs; review-security: Session token written to logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify does not enforce session expiry; review-security: Expired sessions are accepted |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin logic is inverted by indexOf truthiness; review-security: requireAdmin rejects administrators |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on missing header |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-bugs: secretEquals is not constant time; review-security: Non-constant-time secret comparison |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-security: parseInt result is not validated |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction ignores LRU usage; review-security: Eviction is insertion-order, not LRU |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-security: Expiry timers are not cleared |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: delete does not clear scheduled timer |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization percentage is always 0 until full; review-security: Utilization truncates to zero for small caches |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() returns nothing; review-security: keys() always returns an empty array |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-concurrency: loadConfig mutates the shared DEFAULT_CONFIG constant; review-security: loadConfig mutates the shared default object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Bad LINKD_TIMEOUT_MS leaves NaN; review-security: Invalid timeout is assigned as NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: CACHE_TTL_MS multiplied by 1000 although documented as milli; review-security: Cache TTL is multiplied by 1000 contrary to documented milli |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but ignored; review-security: LINKD_FETCH_LIMIT is parsed but ignored |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true; review-security: validate() always reports success |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-bugs: Path traversal via report name; review-security: Path traversal through export name |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-bugs: Report files created world-readable; review-security: Report files are world-readable |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-bugs: Owner-controlled filename in archiveAll |
| T43 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-security: parseReport includes a trailing empty row |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache does not await probes; review-concurrency: warmCache does not await the per-pair work it launches |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets aborts on thrown probe; review-concurrency: checkTargets never cancels the remaining probes on first fai |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-concurrency: flushAudit discards the promise from write() |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner can return NaN; review-security: countForOwner() returns NaN for unknown owners |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-security: Janitor can throw while iterating |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: Delete endpoint does not check ownership; review-security: deleteLink does not enforce ownership |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Open redirect via next parameter; review-security: Open redirect through next parameter |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin endpoint trusts client-supplied x-admin header; review-security: Admin authorization relies on a spoofable header |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: Token leakage in unauthorized response |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink trusts owner from request body |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/cache.ts:45 — Cache timers are never cleared, so stale timers can delete a newer entry
- (medium) src/main.ts:32 — The janitor interval is never cleared
- (medium) src/auth.ts:78 — bearerToken assumes a well-formed header
- (high) src/main.ts:51 — Crash on missing or malformed Authorization header
- (medium) src/store.ts:34 — Quota is never decremented on delete
- (medium) src/worker.ts:70 — flushAudit ignores write errors and retries
