# glm-5.3:cloud · typescript · run 20260901-122732 (repeat 1)

recall **42/63** · 50 finding(s), 1 unmatched · 561412 tokens · 334s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota increment produces NaN on the first create |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted expiry check in resolve() deletes live links and se; review-security: resolve()'s expiry check is inverted: expired links redirect |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | — | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult |  |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, returning the least-followed links fi |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches owners by substring instead of equality |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, contradicting its own contra |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object, not a snapshot |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: a mid-batch failure leaves partia |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining size instead of the number rem |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Auth secret falls back to a hardcoded "dev-secret-do-not-use |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens minted with Math.random() and a timestamp, co |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens are written to the console log on issue |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt, so sessions never expire; review-security: verify() never checks expiresAt, so sessions live forever |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin's indexOf truthiness check is inverted; review-security: requireAdmin logic is inverted: it rejects "admin" and accep |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: hashPassword stores passwords as unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on a missing Authorization header |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals claims constant-time comparison but uses === |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction picks the oldest inserted key, not the least recent |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: Cache timers are never cleared: stale timers evict refreshed |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: delete() never clears the entry's timer; timers map grows wi |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors before multiplying, reporting 0% unless |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map with for-in and always returns an empt |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Bad LINKD_TIMEOUT_MS still assigns NaN after logging 'keepin |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS documented as milliseconds but multiplied |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and then thrown away |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, refusing nothing |
| T39 | — | parallel: EX01 the report name comes from the query string and is join |  |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll builds filenames from unvalidated owner strings |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Snapshot lands at a fixed, predictable /tmp path before rena |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport() emits a bogus all-undefined row for the traili |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() resolves before any probe finishes and drops pro; review-concurrency: warmCache is fire-and-forget: it returns before any probe fi |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets() can never report a failing target; review-concurrency: checkTargets awaits every probe and throws on a failed targe |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() neither retries nor handles the write's rejecti; review-concurrency: flushAudit discards the write promise: no await, no retry, f |
| T49 | — | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning |  |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() dereferences a missing code query parameter |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink never checks the session owner; review-security: deleteLink has no ownership check despite claiming to enforc |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out: ?next is used verbatim as the Locatio |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on a client-controlled request hea |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats() divides by zero and reports NaN on an empty store |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Every JSON response sends Access-Control-Allow-Origin: * wit |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: bearerToken dereferences a missing Authorization header and  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink trusts payload.owner from the request body instea |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.ts:159 — Path traversal in /report: user-controlled name is joined into the export path unsanitized
