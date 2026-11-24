# ollama-direct@hy4:cloud · typescript · run 20261124-100210 (repeat 1)

recall **52/63** · 75 finding(s), 7 unmatched · 315733 tokens · 1103s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: First quota increment produces NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiry condition is inverted; review-security: Link expiry comparison is reversed |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source twice instead of the destinati |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate floors before converting to a percentage |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting or removing a link never restores quota |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() returns least-followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches substrings instead of exact owners; review-security: Owner filter uses substring matching |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() exposes the mutable internal quota object |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining size instead of removed count; review-security: Imported links bypass URL and ownership validation and overw |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt() dereferences missing codes |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-bugs: newCode() can collide or produce invalid short codes |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Public hard-coded fallback secret |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens and short codes use a predictable PRNG |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Full session tokens are written to logs and error responses |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() ignores session expiry; review-security: Session expiry is never checked |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin uses indexOf as a boolean; review-security: Admin role check is inverted |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords are stored as unsalted fast SHA-256 hashes |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken dereferences missing or malformed headers; review-security: Missing Authorization header crashes authentication |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: Secret comparison leaks timing |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours accepts invalid and partial values |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction is FIFO, not least recently used |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache timers are never cleared or removed; review-concurrency: TTL timers are never canceled, so stale timers evict fresh e |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache utilization is always 0 or 100 |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() enumerates a Map with for...in |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the exported defaults object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid timeout is still assigned |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Millisecond cache TTL is multiplied by 1000 |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: Invalid cache sizes disable eviction; review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| T37 | YES | parallel: CF05 loadFile swallows every error, so an unreadable config  | review-bugs: loadFile swallows every read error |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Config validation never fails; review-security: Configuration validation always succeeds and security limits |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: User-controlled report name escapes export directory |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: All-owner CSV reports are created with world-readable defaul |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are written without escaping; review-security: CSV fields are emitted without quoting or formula neutraliza |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Owner-derived archive filenames allow traversal |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot write is not durable or reliably renamable; review-concurrency: Concurrent snapshots share one temporary filename |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport emits bogus rows and undefined fields |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-concurrency: warmCache does not await its parallel probes |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets() rejects instead of reporting failures; review-concurrency: checkTargets launches every probe but cannot cancel remainin |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() does not retry or observe write failures; review-concurrency: flushAudit starts an async write without awaiting or handlin |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() returns NaN for owners with no links |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed create-link bodies escape the error handler |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Missing redirect code crashes the handler; review-bugs: Redirect lowercases mixed-case codes |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: DELETE /links has no owner authorization |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unrestricted open redirect on /out |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts client-controlled X-Admin header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: Stats divides by zero on an empty store |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Wildcard CORS is combined with credentialed responses |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link creation trusts client-supplied owner and unvalidated f |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report discloses every owner's links to any session |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report is written and read under different names |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.ts:25 — Startup ignores the validation result
- (medium) src/cache.ts:46 — set() evicts on updates and ignores nonpositive limits
- (medium) src/cache.ts:38 — peek() reports stale expired entries as cached
- (medium) src/main.ts:106 — Pagination accepts invalid offsets and sizes
- (medium) src/main.ts:160 — Parallel report requests race through a user-chosen shared filename
- (medium) src/main.ts:32 — The janitor's interval stopper is discarded at startup
- (medium) src/export.ts:55 — Audit log entries can be forged with unescaped newlines
