# codex@google/gemini-4-argon · typescript · run 20261013-225208 (repeat 1)

recall **49/63** · 62 finding(s), 1 unmatched · 1136472 tokens · 529s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Store.create increments an uninitialized owner quota entry,  |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Store.resolve has an inverted expiration check that deletes ; review-security: Inverted expiration check in Store.resolve serves expired li |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Store.rename validates `from` twice and never validates `to`; review-security: Store.rename validates source code twice and never validates |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: Store.successRate floors the ratio before multiplying by 100 |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Store.delete does not decrement the owner's quota count |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: Store.top sorts links in ascending order instead of most hit |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Store.byOwner uses substring matching instead of exact owner; review-security: Store.byOwner uses substring matching instead of exact equal |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Store.extend revives expired links and overwrites expiry fro |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: Store.quotas returns the internal quota object directly inst |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Store.importAll inserts links before validating the entire b |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Store.prune returns the remaining link count instead of the  |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short link codes generated with predictable Math.random() |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret fallback when LINKD_SECRET is unset |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Predictable session token generation using Math.random() |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Plaintext session bearer tokens written to stdout logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Authenticator.verify never checks whether a session has expi; review-security: Session expiration is never enforced in Authenticator.verify |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: Authenticator.requireAdmin uses indexOf truthiness, rejectin; review-security: Inverted indexOf check in requireAdmin allows privilege esca |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Unsalted single-pass SHA-256 used for password hashing |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken dereferences an undefined Authorization header a; review-security: Missing Authorization header causes uncaught TypeError in be |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals uses non-constant-time === comparison |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache.set evicts in FIFO insertion order instead of least-re |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache.delete and Cache.set never clear scheduled timers, lea; review-concurrency: Cache never cancels per-entry setTimeout timers on delete, u |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache.utilization floors before multiplying by 100, always r |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache.keys uses for...in on a Map, always returning an empty |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates DEFAULT_CONFIG in place |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: loadConfig overwrites cfg.timeoutMs with NaN when LINKD_TIME |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: loadConfig multiplies LINKD_CACHE_TTL_MS by 1000 even though |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: loadConfig discards LINKD_FETCH_LIMIT with `void limit` inst |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true even when configuration validat |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in exportCsv, readReport, and archiveAll via  |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Exported CSV report written without restrictive file permiss |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-security: Unescaped fields in exportCsv and archiveAll allow CSV and f |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: writeSnapshot uses a fixed shared /tmp/linkd-snapshot.json p; review-security: Predictable /tmp/linkd-snapshot.json path enables symlink an |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport does not skip the trailing empty line, adding a  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache uses Array.prototype.forEach with an async callbac; review-concurrency: warmCache does not await async callbacks passed to forEach |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets runs all probes in parallel and rejects on fail; review-concurrency: checkTargets starts all probes concurrently with Promise.all |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit neither awaits nor retries the asynchronous write; review-concurrency: flushAudit fires async write() without awaiting completion o |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN when the owner has no entry in quo |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-bugs: startJanitor removes expired links from the store without re; review-concurrency: startJanitor sweeps expired links from Store without evictin |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: createLink parses JSON outside the try/catch block, crashing |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect crashes when `code` query parameter is missing and  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: deleteLink deletes any link without verifying caller ownersh |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated next query parameter in /out allows open redirec |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: adminQuotas trusts client-supplied x-admin header instead of |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats divides by zero when the store is empty, serializing h |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Wildcard Access-Control-Allow-Origin combined with Access-Co |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink trusts payload.owner from the request body instea |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports all users' links to any authenticated caller |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report fails with ENOENT when the `name` query parameter is  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/export.ts:45 — writeSnapshot writes its temporary file to /tmp, causing renameSync to fail across filesys
