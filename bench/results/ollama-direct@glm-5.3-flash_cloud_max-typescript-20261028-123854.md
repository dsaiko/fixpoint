# ollama-direct@glm-5.3-flash:cloud#max · typescript · run 20261028-123854 (repeat 1)

recall **40/63** · 53 finding(s), 3 unmatched · 506185 tokens · 544s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | — | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu |  |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiry condition is inverted; review-security: Expiry comparison is inverted |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source twice but never the destination |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate is computed as essentially zero |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not return its quota slot |
| T07 | — | parallel: C10 top sorts ascending, and the comparator never returns 0, |  |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner does not select links by exact owner |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() exposes mutable internal state instead of a snapsho |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic and overwrites existing links |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the wrong count |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes use weak randomness |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens are not generated cryptographically |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens are logged |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Expired sessions remain valid; review-security: Expired sessions remain valid |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin rejects admins and accepts non-admins |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hash is fast and unsalted |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: Missing Authorization header crashes request handling |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction uses insertion order, not LRU |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache timers leak and can delete replaced entries; review-concurrency: Cache timers are never cancelled or tracked correctly |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | — | TS-ONLY utilization floors the ratio before multiplying so it always r |  |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache utilization is nearly always zero; review-bugs: Cache keys() always returns an empty array |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the exported default object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid LINKD_TIMEOUT_MS overrides the default with NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Cache TTL environment value is treated as seconds instead of |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() never rejects invalid configuration |
| T39 | — | parallel: EX01 the report name comes from the query string and is join |  |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Sensitive exports are not permission-controlled |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are written without escaping; review-security: CSV output is not escaped |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport manufactures a row from the final blank line |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache returns before priming finishes; review-concurrency: warmCache launches promises but does not await or observe th |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets rejects on the first failing probe; review-concurrency: Concurrent target probes cannot be abandoned and one rejecti |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit neither waits nor retries the audit write; review-concurrency: Audit flush ignores the asynchronous write and performs no r |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: Unknown-owner quota count becomes NaN |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed JSON crashes the request handler; review-security: Malformed JSON crashes create endpoint |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Invalid TTL produces never-expiring links |
| T53 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Any authenticated user can delete any link |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect through arbitrary next parameter |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Administrator authorization uses a client-controlled header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: Stats divide by zero for an empty store |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: Create payload accepts an undefined owner; review-security: Create endpoint trusts client-controlled owner and expiry |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: All users' links exposed to any authenticated session |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report name is written but not read |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.ts:97 — Public redirect endpoint is incorrectly protected
- (critical) src/main.ts:51 — Missing Authorization header crashes authentication handling
- (high) src/main.ts:160 — Report name permits path traversal
