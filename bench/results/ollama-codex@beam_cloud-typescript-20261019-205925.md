# ollama-codex@beam:cloud · typescript · run 20261019-205925 (repeat 2)

recall **51/63** · 69 finding(s), 4 unmatched · 1176163 tokens · 148s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: create computes quota as NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve deletes non-expired links and never returns them; review-security: Expired link expiry comparison is inverted |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates from twice and never validates to; review-security: rename() does not validate the destination code |
| T04 | YES | parallel: C04 page clamps end past the array length; slice silently re | review-bugs: page end index is off by one |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate is truncated to 0 unless all links used |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not refund the owner's quota |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top sorts ascending despite docs |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner uses substring match instead of exact owner; review-security: byOwner() matches owners by substring |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links; review-security: extend() can revive an expired link |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas returns mutable internal state; review-security: quotas() exposes the mutable internal object |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune reports size instead of removed count |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt dereferences possibly missing link |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes use Math.random |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback authentication secret |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session token generated with Math.random |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session token logged in full |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify never checks session expiry; review-security: verify() never enforces session expiration |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin rejects admins; review-security: requireAdmin authorization check uses indexOf incorrectly |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with fast unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on missing/short header; review-security: bearerToken assumes a present Authorization header |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals leaks timing information |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-security: parseLifetimeHours does not validate numeric input |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction is FIFO, not LRU |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: delete leaks the scheduled expiry timer; review-concurrency: Expired/stale timers outlive their cache entries and delete  |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization percentage is almost always zero |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() returns an empty array |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid timeout is assigned after being logged; review-security: Invalid timeout value is accepted despite logged validation  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Cache TTL multiplied as if seconds |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is ignored |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal through export report name |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Export reports written with default file permissions |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Path traversal through owner name in archiveAll |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-security: Predictable temporary snapshot path enables overwrite race |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport emits a trailing empty row |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache does not await all probes; review-concurrency: warmCache fire-and-forgets its async probes, so it resolves  |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets runs every probe despite doc; review-concurrency: checkTargets does not abandon remaining probes after a failu |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit does not retry until success; review-concurrency: flushAudit drops the write promise and never retries |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner can return NaN |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: createLink does not handle invalid JSON |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect crashes on missing code query |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Any authenticated user can delete another user's link; review-security: Link listing is not scoped to an owner |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect through next query parameter |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint guarded only by spoofable header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats divides by zero on empty store |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Permissive CORS combined with credentialed responses |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Client-controlled owner allows ownership spoofing |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exports all owners' data without access scop |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report reads a different file than it writes |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/main.ts:32 — Janitor interval handle is discarded, so the timer is never cleared
- (high) src/export.ts:60 — Path traversal when reading reports
- (medium) src/export.ts:65 — Unescaped CSV output allows formula injection
- (high) src/store.ts:34 — Plain object quotas keyed by untrusted owner risk prototype mutation
