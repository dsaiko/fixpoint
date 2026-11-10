# claude-fable-5-2 · typescript · run 20261110-150125 (repeat 1)

recall **57/63** · 79 finding(s), 5 unmatched · 34006 tokens · 370s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter becomes NaN on an owner's first link |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links are deleted,  |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to`; review-security: rename validates `from` twice and never validates the new co |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() floors before multiplying, so it is 0 unless e |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never gives the quota slot back |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending and uses an inconsistent comparator |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses a substring match instead of equality; review-security: byOwner uses substring matching and returns other owners' li |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links and can shorten a live link's |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object instead of a snaps |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic and silently overwrites existing l; review-security: importAll overwrites existing codes and skips target validat |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the number of remaining links instead of the |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt() dereferences a missing link |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-bugs: newCode() can return short or empty codes and create() overw; review-security: Short codes generated with Math.random are predictable |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens generated with Math.random and a timestamp |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token written to the log in clear text |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt, so expired sessions stay va; review-security: verify never checks expiresAt: expired sessions stay valid f |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin() uses indexOf() as a boolean: admins are rejec; review-security: requireAdmin treats indexOf result as a boolean: non-admin r |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with a single unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken() throws on a missing Authorization header and r; review-security: bearerToken crashes on a missing Authorization header and ig |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals is not constant time despite its contract |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours() accepts garbage and can return NaN; review-security: parseLifetimeHours accepts any client-requested session life |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction removes the oldest inserted entry, not the least re |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: set() never cancels the previous timer, so a stale timer del; review-concurrency: Cache expiry timers are never cancelled, so a stale timer de |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: delete() does not stop the scheduled timer or drop it from t |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors before multiplying and divides by a pos |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map with for...in and always returns an em |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig() mutates the shared DEFAULT_CONFIG object |
| T33 | YES | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so | review-bugs: LINKD_PORT and LINKD_CACHE_SIZE are parsed without validatio |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid LINKD_TIMEOUT_MS is logged as 'keeping default' but  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 although it is alre |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| T37 | YES | parallel: CF05 loadFile swallows every error, so an unreadable config  | review-bugs: loadFile() swallows every read error, not just a missing fil |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, and main ignores the result  |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in exportCsv: user-controlled report name esc |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report written with default permissions although it must be  |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are written without quoting or escaping; review-security: CSV/formula injection and row forgery in exported reports |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Path traversal in archiveAll through the owner name |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot() stages in /tmp under a fixed name, so the re; review-concurrency: writeSnapshot uses one fixed temp path shared by every write |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport() emits a bogus row for the trailing newline and |
| T45 | YES | the errors counter is reported by snapshot and never incremented anywh | review-bugs: Metrics.errors is reported but can never be incremented |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() resolves before any probe finishes and leaks rej; review-concurrency: warmCache fires async callbacks through forEach and never aw |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets() rejects instead of reporting the first failin; review-concurrency: checkTargets rejects on the first failing probe instead of r |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() drops the promise: no retry, and failures becom; review-concurrency: flushAudit drops the write promise: no await, no retry, unha |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() returns NaN for an owner with no links |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-bugs: Janitor ignores the cache and removes links without releasin |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: createLink parses the body outside the try and accepts NaN/n; review-security: Client-supplied TTL is not validated |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() crashes on a missing code and lowercases codes th |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: deleteLink has no ownership check: any authenticated user ca; review-security: GET /links returns every owner's links to any authenticated  |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out via unvalidated ?next= |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorised by a client-supplied X-Admin heade |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: /stats divides by zero links and reports NaN |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Wildcard CORS origin combined with Allow-Credentials on ever |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented token |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink takes the owner from the request body instead of  |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports all owners' links to any authenticated user |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with no name writes links.csv but reads back '.csv' |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/main.ts:134 — /out answers with a permanent 301, so repeat clicks are never counted
- (medium) src/main.ts:106 — listLinks passes unvalidated offset/size to page()
- (low) src/main.ts:32 — Janitor interval started at module load and its stopper discarded
- (high) src/export.ts:60 — Path traversal in readReport: arbitrary .csv file read via the download path
- (low) src/main.ts:162 — Verbose error responses disclose filesystem paths and internal errors
