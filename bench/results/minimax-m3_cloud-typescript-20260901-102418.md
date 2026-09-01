# minimax-m3:cloud · typescript · run 20260901-102418 (repeat 1)

recall **31/63** · 39 finding(s), 2 unmatched · 452700 tokens · 585s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | — | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu |  |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes live links on every hit |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice and 'to' never |
| T04 | YES | parallel: C04 page clamps end past the array length; slice silently re | review-bugs: page() returns one row past the end of the list |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() always returns 0 or 100 |
| T06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| T07 | — | parallel: C10 top sorts ascending, and the comparator never returns 0, |  |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() does substring matching; review-security: byOwner uses substring match, leaking other owners' links |
| T09 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| T10 | — | parallel: C14 quotas hands out the store's own object while the doc ca |  |
| T11 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the wrong value |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' used in pr |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-bugs: Session tokens use Math.random; review-security: Session tokens minted with Math.random |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens are written to stdout in plain text |
| T18 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin rejects the admin role; review-security: requireAdmin accepts every role not in the list (indexOf ret |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken dereferences missing header parts |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals is not constant-time despite its doc comment |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction is not LRU; oldest-stored entry is always picked |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: setTimeout for entry expiry is never cleared on overwrite or |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: delete() never clears the expiration timer |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() always reports 0 or 100 |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map as if it were a plain object |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | — | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t |  |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | — | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  |  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT has no effect |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true |
| T39 | — | parallel: EX01 the report name comes from the query string and is join |  |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: writeSnapshot opens /tmp/linkd-snapshot.json for write witho |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache doesn't await probes and has the wrong return type; review-concurrency: warmCache resolves before probes complete |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets doesn't short-circuit; review-concurrency: checkTargets does not abandon probes on first failure |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit never retries and discards its promise; review-concurrency: flushAudit discards the write promise and lies about retries |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for unknown owners |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: DELETE /links does not check link ownership |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect at /out echoes attacker-controlled URL into Lo |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization gate trusts a client-supplied header |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: CORS allows any origin with credentials |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink trusts the request body's owner field instead of  |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-security: Path traversal in /report name param via exportCsv/readRepor |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.ts:159 — report() leaks every owner's links to the caller
- (low) src/export.ts:44 — writeSnapshot writes the temp file to /tmp, not the snapshot dir
