# qwen/qwen3.8-max · typescript · run 20260901-103658 (repeat 1)

recall **47/63** · 56 finding(s), 0 unmatched · 53441 tokens · 1270s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter becomes NaN on the first link of every owner |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() has the expiry condition inverted: deletes live li; review-security: Inverted expiry check in resolve(): expired links keep resol |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() misplaces the parenthesis: reports 0% until ev |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never decrements the owner's quota |
| T07 | — | parallel: C10 top sorts ascending, and the comparator never returns 0, |  |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches by substring instead of equality |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links despite the stated in |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live quota object, not the promised sna |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: a validation failure mid-batch le; review-concurrency: importAll violates its documented all-or-nothing atomicity |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining link count instead of the numb |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used silently when LINKD_SECRET is |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens generated with Math.random() are predictable |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Live session token written to logs on issue |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks session expiry; sessions never die and; review-security: verify() never checks expiresAt: sessions never expire |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin() truthiness on indexOf: denies "admin", admits; review-security: requireAdmin logic is inverted: non-admin roles pass, 'admin |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with plain unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken() throws TypeError when the Authorization header |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals claims constant-time comparison but uses === |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction is FIFO, not LRU: accesses never reorder entries |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: Cache.set leaks the previous expiry timer on overwrite; stal |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: delete() never clears the entry's timeout; timers map grows ; review-concurrency: Cache.delete never cancels the scheduled expiry timer despit |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() misplaces the parenthesis: reports 0% until fu |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for..in over a Map and always returns an empty a |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig() mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid LINKD_TIMEOUT_MS still overwrites timeoutMs with NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is documented as milliseconds but multipl |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and validated but never applied |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() never rejects a bad configuration; it always retu |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report export: user-controlled name escape |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Export file written world-readable despite 'service user onl |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Predictable temp path in shared /tmp for snapshot write (sym |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport() emits a phantom row of undefined cells for eve |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() resolves immediately: async callbacks inside for; review-concurrency: warmCache fires unawaited async callbacks; resolves before w |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets() rejects instead of reporting the first failin; review-concurrency: checkTargets uses Promise.all, contradicting its first-failu |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() neither retries nor observes the write result; review-concurrency: flushAudit discards the write promise: no retry, silent loss |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() returns NaN for owners with no links |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: createLink() parses the request body outside the try block;  |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Garbage ttl yields NaN expiry |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() crashes when ?code= is missing, and lowercases ca |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: deleteLink has no ownership check: any authenticated user ca; review-security: listLinks returns every user's links to any authenticated ca |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out: unvalidated ?next= becomes the Locati |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated by spoofable X-Admin request header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats() divides by zero when the store is empty |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Permissive CORS: wildcard origin combined with credentials o |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Submitted bearer token echoed back in the 401 response body |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink trusts client-supplied owner instead of the authe |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with no name always returns 500: write and read disa |
