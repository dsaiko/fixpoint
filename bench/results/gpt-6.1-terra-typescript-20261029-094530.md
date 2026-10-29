# gpt-6.1-terra · typescript · run 20261029-094530 (repeat 1)

recall **34/63** · 42 finding(s), 2 unmatched · 149273 tokens · 86s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: New owners receive a NaN quota count |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiration check deletes valid links |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: Success rate rounds before converting to a percentage |
| T06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: Leaderboard sorts least-followed links first |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup matches substrings |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can revive it after expiry |
| T10 | — | parallel: C14 quotas hands out the store's own object while the doc ca |  |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Failed imports leave a partially modified store |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining count instead of the removed cou |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens use predictable randomness |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Bearer tokens are written to logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Expired sessions continue to verify; review-security: Expired sessions remain valid |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | — | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an |  |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | — | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  |  |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction ignores recent accesses |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Old cache timers delete replacement entries; review-concurrency: An earlier expiry timer can delete a replacement entry |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache utilization is zero until the cache is full |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache keys does not iterate map entries |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | — | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t |  |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid timeout replaces the default with NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Cache TTL in milliseconds is multiplied by 1000 |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: Configured fetch limit is ignored |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Invalid configuration is always accepted |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report name permits writing outside the export directory |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Reports containing all owners' links use default file permis |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV values are written without escaping; review-security: Exported CSV permits spreadsheet formula injection |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshot writers share one temporary file |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: Parsing an exported report adds an empty row |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: Cache warming returns before probes finish; review-concurrency: Cache warming returns before its probes finish |
| T47 | — | TS-ONLY checkTargets is documented to report the first failure, but pr |  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: Audit flush neither waits nor retries on failure; review-concurrency: Audit flush discards the asynchronous write |
| T49 | — | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning |  |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed JSON escapes the request error handler |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect without a code throws |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Any authenticated user can delete another owner's link; review-security: Link listing exposes every owner's targets |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Outbound redirect accepts arbitrary destinations |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Administrator access is controlled by a client-supplied head |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link creation trusts a client-supplied owner |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report without a name reads the wrong file |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/auth.ts:78 — Missing authorization header throws instead of returning 401
- (medium) src/export.ts:47 — Snapshot rename fails when the export directory is on another filesystem
