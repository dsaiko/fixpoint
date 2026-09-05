# gpt-6-astra-xhigh · typescript · run 20260905-144718 (repeat 1)

recall **37/63** · 51 finding(s), 7 unmatched · 271148 tokens · 706s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: A new owner's quota becomes NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and accepts invalid destin |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: Success rate reports zero unless every link was followed |
| T06 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: The leaderboard returns the least followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner filtering matches substrings instead of exact owners |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive it |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: The quota snapshot exposes mutable store state |
| T11 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Imports and removals do not maintain owner counts; review-bugs: Prune returns the remaining size instead of the removal coun |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Bearer tokens use noncryptographic randomness |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session issuance writes bearer credentials to logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-security: Expired session tokens remain valid |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | — | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an |  |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: Missing Authorization headers throw instead of returning 401 |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction ignores recent accesses |
| T27 | — | TS-ONLY a timer is stored per set and never cleared -- delete leaves i |  |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: Timer bookkeeping grows beyond the cache limit; review-concurrency: An old expiry timer deletes a replacement cache entry |
| T29 | — | TS-ONLY utilization floors the ratio before multiplying so it always r |  |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache utilization rounds before converting to a percentage; review-bugs: Cache keys always returns an empty array |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: Loading configuration mutates the defaults |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid timeout values overwrite the promised fallback |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Millisecond cache TTL overrides are multiplied by 1000 |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: The fetch limit environment override is discarded |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation never rejects invalid settings |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report names allow writes outside the export directory |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Sensitive reports are created without private file permissio |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are written without escaping; review-security: User-controlled CSV cells can execute spreadsheet formulas |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot writes are not made durable before success; review-concurrency: Concurrent processes share the same snapshot temporary file |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: Parsing an exported report adds a phantom row |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-concurrency: Cache warming discards asynchronous probe failures |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: A rejected probe prevents reporting the failed target |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: Audit flush failures are discarded and never retried; review-concurrency: Audit flushing detaches the write promise |
| T49 | — | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning |  |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed link bodies escape the request error handler |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Invalid TTL values create links that never expire |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: A redirect request without code throws |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Any authenticated user can delete another owner's links; review-security: Listings and reports disclose other owners' links |
| T55 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A caller-controlled header grants administrator access |
| T57 | — | parallel: C20 /stats divides by the link count, producing NaN on an em |  |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: Every supplied session token is rejected |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link creation trusts a forged owner |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Reports without a name read the wrong filename |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/cache.ts:46 — Updating a full cache unnecessarily evicts another entry
- (medium) src/cache.ts:52 — A zero-capacity cache still stores entries
- (medium) src/store.ts:152 — A failed batch import leaves earlier changes applied
- (medium) src/store.ts:33 — A generated code collision overwrites an existing link
- (medium) src/export.ts:45 — Snapshot replacement fails across filesystems
- (medium) src/worker.ts:41 — Target probes report success without checking reachability
- (medium) src/main.ts:32 — The janitor interval has no reachable shutdown path
