# claude-opus-5 · typescript · run 20260901-142445 (repeat 1)

recall **55/63** · 73 finding(s), 5 unmatched · 43237 tokens · 488s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter starts at NaN for every new owner |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links are deleted, ; review-security: resolve inverts its expiry test: expired links resolve forev |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| T04 | YES | parallel: C04 page clamps end past the array length; slice silently re | review-bugs: page() computes an end index one past the array length |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() integer-divides and can only return 0 or 100 |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending and uses an unstable non-numeric compa |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() substring-matches the owner, returning other owner; review-security: byOwner matches owner as a substring, leaking other owners'  |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object, not the documente |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: a bad code in the batch leaves ea |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the surviving link count instead of the numb |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes are generated with Math.random and are enumerabl |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret ships in the binary and silently a |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens are minted from Math.random and are guessable |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens are written to the application log at issue t |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt, so sessions never expire; review-security: verify never checks expiresAt, so sessions never expire |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin() uses indexOf truthiness, admitting every non-; review-security: requireAdmin is inverted: it rejects admins and admits every |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: hashPassword is unsalted, single-round SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken() dereferences an undefined header and returns a; review-security: bearerToken dereferences a missing Authorization header, cra |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals uses === and is not constant-time, contradictin |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours returns NaN and accepts trailing garbage |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction picks the oldest inserted entry, not the least rece |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: Cache TTL timers are never cleared: stale timer evicts a ref |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: delete() and re-set() leak timers; a stale timer evicts a fr |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() integer-divides and always reports 0 or 100 |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map with for...in and always returns an em |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig() mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: A non-numeric LINKD_TIMEOUT_MS is logged as bad and then ass |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 despite being docum |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed, tested, and then thrown away |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so an invalid config never f |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in exportCsv/readReport writes and reads arbi |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Cross-owner export is written with default permissions despi |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV rows are built without quoting or escaping, corrupting t; review-security: CSV export does not escape or neutralize cell content (formu |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll builds filenames from the owner string, allowing  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot renames across filesystems and never fsyncs, s; review-concurrency: writeSnapshot uses one hard-coded temp path, so concurrent s |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport emits a bogus trailing row with undefined cell v |
| T45 | YES | the errors counter is reported by snapshot and never incremented anywh | review-bugs: Metrics.errors is reported but never incremented |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache resolves before any probe completes and drops prob; review-concurrency: warmCache resolves before any probe runs; rejections are unh |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets rejects on the first bad target instead of repo; review-concurrency: checkTargets rejects instead of reporting the first failing  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit ignores the returned promise: no retry, and rejec; review-concurrency: flushAudit drops the write promise: no retry, no error path, |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for an owner with no links |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-bugs: The janitor sweeps the store but never the cache, leaving de |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: JSON.parse sits outside the try block, so a malformed body t |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: A non-numeric ttl in the create payload produces a NaN expir |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() dereferences a missing ?code query parameter |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink() performs no ownership check despite documenting; review-security: deleteLink performs no ownership check, so any user can dele |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out is an unvalidated open redirect |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: /admin/quotas authorizes on a client-controlled X-Admin head |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: /stats divides by zero when the store is empty |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: CORS allows any origin together with credentials |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: The 401 response echoes the presented bearer token back to t |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink takes the link owner from the request body instea |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report reads back a different filename than it wrote when n |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/cache.ts:37 — peek() ignores expiry, so warm() refuses to refresh expired entries
- (medium) src/store.ts:24 — create() overwrites an existing link on short-code collision
- (medium) src/main.ts:32 — Janitor interval handle discarded, so the sweep can never be stopped
- (low) src/worker.ts:37 — No deadline on any probe: cfg.timeoutMs is configured but never applied
- (low) src/main.ts:162 — Export failures return the server filesystem path and raw exception to the client
