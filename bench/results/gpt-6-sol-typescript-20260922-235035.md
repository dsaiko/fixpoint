# gpt-6-sol · typescript · run 20260922-235035 (repeat 1)

recall **34/63** · 42 finding(s), 2 unmatched · 269872 tokens · 339s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: First link makes its owner's quota NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Live links are deleted when resolved; review-security: Resolving a live link deletes it |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: Success rate stays zero until every link is used |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not release its quota count |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: Top links are sorted least followed first |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup includes substring matches |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending an expired link revives it |
| T10 | — | parallel: C14 quotas hands out the store's own object while the doc ca |  |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Batch import can leave a partial batch |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns remaining links instead of removed links |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | — | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam |  |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session bearer tokens are written to logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-security: Expired session tokens remain valid |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | — | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an |  |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-security: Missing Authorization header throws before rejection |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction ignores recent accesses |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache timers accumulate and can delete replacement entries; review-concurrency: An old expiry timer can delete a replacement entry |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache utilization reports zero until full |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache keys are always reported as empty |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | — | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t |  |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid timeout replaces the default with NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Configured cache TTL is multiplied by 1000 |
| T36 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar |  |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation never rejects invalid values |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report name permits writes outside the export directory |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are written without escaping; review-security: User-controlled fields can inject spreadsheet formulas |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots share one temporary file |
| T44 | — | TS-ONLY parseReport asserts a header row exists and casts every cell w |  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: Cache warming finishes before its probes; review-concurrency: Cache warming detaches probe failures |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: Target check rejects instead of reporting a failed target |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: Audit flush ignores completion and failures; review-concurrency: Audit writes run without a completion or failure signal |
| T49 | — | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning |  |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed link JSON escapes the request handler |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect without a code throws |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Authenticated users can read other owners' links; review-security: Any authenticated user can delete another owner's link |
| T55 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Client-controlled header grants admin access |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: Empty store produces a null hits-per-link value |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: Missing Authorization header crashes the handler |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link ownership is chosen by the requester |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report without a name always fails |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/main.ts:29 — No request can obtain a usable session
- (medium) src/main.ts:159 — Reports omit links after the first thousand
