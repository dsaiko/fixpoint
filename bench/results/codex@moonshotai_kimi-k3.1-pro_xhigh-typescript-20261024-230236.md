# codex@moonshotai/kimi-k3.1-pro#xhigh · typescript · run 20261024-230236 (repeat 1)

recall **34/63** · 44 finding(s), 3 unmatched · 684754 tokens · 407s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | — | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu |  |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry condition deletes live links and serves expired ones; review-security: Reversed expiry check deletes live links and serves expired  |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | — | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult |  |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting or pruning links never releases owner quota |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: Leaderboard sorts least-followed links first |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner filtering returns substring matches |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend revives expired links |
| T10 | — | parallel: C14 quotas hands out the store's own object while the doc ca |  |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Batch import is not atomic |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns remaining links instead of removed count |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens use a predictable PRNG |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Live bearer tokens are written to logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Expired sessions remain valid; review-security: Expired session tokens remain valid |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-security: requireAdmin authorizes every non-admin role |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: Missing or malformed Authorization headers crash request han |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache evicts insertion-order entries and can exceed its limi |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Stale expiry timers can delete refreshed cache entries; review-concurrency: Stale expiration timers delete refreshed cache entries |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache utilization collapses most occupancy levels to 0 or 10 |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache keys always returns an empty list |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: Configuration loading mutates global defaults and ignores se |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | — | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  |  |
| T35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu |  |
| T36 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar |  |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation always accepts invalid configuratio |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report names allow directory traversal and file overwrite |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Cross-owner reports are created with permissive file modes |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-security: CSV exports permit spreadsheet formula injection |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots share one temporary file |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: Report parser adds a phantom row and cannot round-trip CSV |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: Cache warming finishes before probes and loses failures; review-concurrency: Cache warming detaches all probe promises |
| T47 | — | TS-ONLY checkTargets is documented to report the first failure, but pr |  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: Audit flush neither awaits nor retries failed writes; review-concurrency: Audit flush ignores asynchronous completion and failure |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: Unknown owners produce NaN quota counts |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed JSON escapes the create-link error response; review-security: Malformed JSON escapes the request error handling |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Missing redirect code crashes the request |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Any authenticated user can delete any link; review-security: Link listings and reports expose every user's data |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The out endpoint is an unrestricted open redirect |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A client-controlled header bypasses admin authorization |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: Statistics emit invalid JSON for an empty store |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Clients can forge link ownership |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/store.ts:88 — Success rate rounds whole fractions before converting to percent
- (medium) src/main.ts:32 — Janitor interval's only stop handle is discarded
- (high) src/main.ts:51 — Missing authorization headers cause an unhandled exception
