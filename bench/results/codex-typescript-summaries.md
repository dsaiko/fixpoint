# codex · typescript · run summaries (repeat 1)

recall **41/63** · 50 finding(s), 5 unmatched · 195225 tokens · 882s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: First quota increment produces NaN |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiry check is inverted |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: Success rate is rounded to only zero or one hundred |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Removing links does not release owner quota |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: Top links are sorted least-popular first |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup performs substring matching |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link revives expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: Quota snapshot exposes mutable store state |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Batch import is not atomic |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining size instead of removed count |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens use a non-cryptographic random generator |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Bearer credentials are written to application logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-security: Expired session tokens remain valid indefinitely |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | — | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an |  |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: Missing Authorization header crashes request handling |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction is FIFO rather than least-recently-used |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Old cache timers can delete refreshed entries; review-concurrency: Stale expiry timers can delete newer cache entries |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache utilization rounds before converting to percent |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache keys always returns an empty list |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: Loading configuration mutates the shared defaults |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid timeout replaces the default with NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Cache TTL environment value is multiplied by 1000 |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is ignored |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation always succeeds |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report name permits directory traversal and arbitrary CSV wr |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Sensitive reports are created with potentially world-readabl |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are emitted without escaping; review-security: Unescaped CSV fields enable spreadsheet formula injection |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: Parsing a generated report adds a spurious blank row |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-concurrency: warmCache detaches all probe tasks |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: Target checking rejects instead of reporting a failure |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: Audit flush neither waits nor retries; review-concurrency: Audit writes are launched without completion or failure trac |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: Unknown owners have a NaN count |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed JSON escapes the request error handling |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect without a code throws |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Any authenticated user can delete another user's links; review-security: Link listing and reports disclose every tenant's data |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Click-tracking endpoint is an open redirect |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Client-controlled header grants administrator access |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: Empty-store hit average becomes null |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Callers can create records under arbitrary owners |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-bugs: Reports silently omit links after the first 1000 |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Unnamed reports are written and read under different names |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/main.ts:25 — Startup ignores the validation result
- (medium) src/config.ts:100 — NaN numeric settings bypass validation
- (medium) src/cache.ts:46 — Refreshing an existing entry can evict an unrelated entry
- (medium) src/export.ts:47 — Snapshot rename can fail across filesystems
- (medium) src/main.ts:32 — The janitor interval is started without any shutdown path
