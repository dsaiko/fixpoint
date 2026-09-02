# kimi-k2.7-code:cloud · typescript · run 20260901-215745 (repeat 1)

recall **33/63** · 45 finding(s), 3 unmatched · 195306 tokens · 321s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: New-link quota counter becomes NaN for a new owner |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve deletes live links instead of expired ones; review-security: Active links are deleted on first access while expired links |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source code twice and never validates t |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate always reports 0% unless every link was used |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not decrement the owner's quota |
| T07 | — | parallel: C10 top sorts ascending, and the comparator never returns 0, |  |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner matches owners by substring |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend can revive an already-expired link |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live quota object, not a snapshot; review-concurrency: quotas() returns the live internal quota object |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns the remaining size, not the removed count |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt crashes on an unknown code |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens are minted with a non-cryptographic PRNG |
| T17 | — | parallel: S03 every issued session token is logged in cleartext |  |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify never checks session expiry; review-security: verify() ignores session expiration |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin rejects admins and accepts everyone else |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes when the Authorization header is missing; review-security: Bearer-token parser crashes on missing Authorization header |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache timeouts are never cleared, so stale TTLs can evict ne; review-concurrency: Stale setTimeout timers evict freshly cached entries |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache utilization is always 0% until the cache is full |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache keys() always returns an empty array |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Bad LINKD_TIMEOUT_MS is logged but still assigned as NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is treated as seconds, not milliseconds |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is read but never stored |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true, even when config is invalid |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in CSV report export and download |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache resolves before probes finish; review-concurrency: warmCache resolves before probes finish |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets waits for all probes instead of abandoning on f |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-concurrency: flushAudit discards the async write and never retries |
| T49 | — | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning |  |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect crashes when the code query parameter is missing |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink does not enforce owner authorization; review-security: Link deletion does not verify ownership |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quotas endpoint trusts a client-controlled X-Admin hea |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats divides by zero when the store is empty |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: createLink parses JSON outside its try/catch; review-security: Link creation accepts an arbitrary owner from the request bo |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/main.ts:25 — The result of validate(cfg) is ignored at startup
- (low) src/worker.ts:71 — flushAudit does not retry
- (medium) src/main.ts:32 — Janitor interval stop handle is discarded
