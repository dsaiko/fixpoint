# claude-sonnet-5-5 · typescript · run 20260929-100041 (repeat 1)

recall **49/63** · 64 finding(s), 2 unmatched · 20379 tokens · 166s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter becomes NaN on a user's first link |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates `from` twice and never validates `to` |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate truncates the fraction before scaling |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending and with an inconsistent comparator |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner matches by substring; review-security: byOwner uses substring matching instead of exact owner match |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() exposes the live internal object |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic; review-security: importAll is not atomic and overwrites existing links |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns the remaining size instead of the number remov |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes are Math.random-based but documented as unguessa |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens are generated with Math.random and a timestam |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token written to the log |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks session expiry; review-security: verify() never enforces session expiry |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin tests indexOf result as a boolean; review-security: requireAdmin is inverted: everyone except admins passes |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on a missing or malformed Authorization  |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals is not constant-time; password hashing is unsal |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction removes the oldest-inserted entry, not the least re |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Timers are never cleared, so they leak and can delete refres; review-concurrency: Per-entry TTL timers are never cleared, so a stale timer evi |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors the ratio before multiplying |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for...in on a Map |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: A bad LINKD_TIMEOUT_MS is logged as ignored but then applied |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true and misses NaN values; review-security: validate() always returns true, so invalid configuration is  |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal via report name in exportCsv and readReport |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report and owner files are written world-readable, contrary  |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are not escaped; review-security: CSV export does not escape fields (CSV/formula injection) |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll uses the owner string as a filename |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot is neither durable nor safe; review-security: Snapshot staged at a fixed, world-writable /tmp path |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport turns the trailing newline into a bogus row |
| T45 | YES | the errors counter is reported by snapshot and never incremented anywh | review-bugs: Metrics.errors is never incremented |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache does not wait for its probes and lets rejections e; review-concurrency: warmCache uses forEach(async) so no probe is awaited and rej |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets rejects instead of reporting the first failing  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit neither awaits nor retries; review-concurrency: flushAudit fires the write without awaiting it, so failures  |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for an owner with no quota entry |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: createLink and redirect throw on malformed input |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect lowercases the code but codes with uppercase are ac |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink performs no ownership check; review-security: deleteLink and createLink ignore the authenticated identity |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect at /out |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin endpoint trusts a client header and ignores requireAdm; review-security: Admin endpoint trusts a client-supplied X-Admin header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: hits_per_link divides by zero on an empty store |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Wildcard CORS combined with Access-Control-Allow-Credentials |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: 401 response echoes the token; review-security: 401 response echoes the presented token |
| T60 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report and /links expose every owner's links to any authent |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() reads back a different file than it wrote when name |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/store.ts:24 — create() can overwrite an existing link on a code collision
- (low) src/main.ts:162 — Verbose error reveals the server export path and raw exception text
