# claude-fable-5-1 · typescript · run 20260901-214213 (repeat 1)

recall **53/63** · 66 finding(s), 2 unmatched · 34563 tokens · 378s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter starts from undefined, so every owner's count  |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links are deleted,  |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() floors before scaling, so it reports 0 unless  |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending and its comparator is inconsistent for |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches by substring, returning other owners' link; review-security: byOwner matches by substring, leaking other owners' links |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links despite the documented 'expir |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object, not a snapshot |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: a bad code midway leaves a partia |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining size instead of the number rem |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-bugs: newCode() never checks for collisions and can be shorter tha |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens are minted from Math.random() |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens written to the log |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt, so expired sessions stay va; review-security: verify() never checks token expiry |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin() is inverted: admins are rejected, everyone el; review-security: requireAdmin is inverted: admin denied, unknown roles allowe |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted single-round SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken() crashes on a missing Authorization header; review-security: Missing Authorization header crashes the handler before auth |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals is not constant-time despite its contract |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours() returns NaN for non-numeric input and u |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction takes the oldest inserted key, not the LRU one, and |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: Stale expiry timers are never cleared and delete newer entri |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors before scaling and reports 0 until the  |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for...in over a Map and always returns an empty  |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig() mutates the exported DEFAULT_CONFIG instead of  |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid LINKD_TIMEOUT_MS is logged as 'keeping default' but  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 although it is docu |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so a bad config never refuse |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report name (write and read) |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report containing all owners' data is written with default p |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are not escaped, so targets containing commas cor; review-security: CSV injection from unescaped user-controlled fields |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Owner name used as a filename without sanitization |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot() renames from /tmp into dir, which fails acro; review-concurrency: writeSnapshot shares one fixed /tmp path between all writers |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport() emits a bogus row for the trailing newline |
| T45 | YES | the errors counter is reported by snapshot and never incremented anywh | review-bugs: Metrics.errors is never incremented |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() resolves immediately and turns probe failures in; review-concurrency: warmCache returns before any probe finishes and leaks unhand |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets() can never report a failure because probe() th; review-concurrency: checkTargets rejects on the first failing probe instead of r |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() neither awaits nor retries the write, and a rej; review-concurrency: flushAudit fires write() and forgets it: no retry, unhandled |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() returns NaN for owners without links |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: JSON.parse of the request body is outside the try, so a malf; review-security: Unvalidated request inputs throw out of the handler |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Non-numeric ttl produces NaN expiry: the link never expires  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect()/deleteLink() crash when ?code is absent, and lowe |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: DELETE /links lets any user delete any link (IDOR) |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on a client-supplied x-admin heade |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: hits_per_link divides by zero on an empty store |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: CORS allows any origin together with credentials |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Rejected bearer token echoed into the 401 body |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner is taken from the request body, not the session |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exposes every owner's links to any authenticated use |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with no ?name writes links.csv but reads .csv, so th |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/main.ts:32 — Janitor stopper is discarded, so the interval can never be cancelled
- (low) src/main.ts:162 — Error response leaks filesystem path and raw exception
