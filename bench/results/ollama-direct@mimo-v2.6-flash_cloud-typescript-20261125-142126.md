# ollama-direct@mimo-v2.6-flash:cloud · typescript · run 20261125-142126 (repeat 1)

recall **46/63** · 59 finding(s), 3 unmatched · 240039 tokens · 383s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: First create produces NaN owner quotas |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Valid links are deleted while expired links still resolve; review-security: Link expiry check is inverted |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the old code instead of the new code |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate reports zero except when every link was used |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting or pruning links never decrements owner quotas |
| T07 | — | parallel: C10 top sorts ascending, and the comparator never returns 0, |  |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner uses substring matching instead of owner equality |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas returns the live internal object; review-concurrency: quotas exposes the live mutable quota state as a snapshot |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic when validation fails |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns remaining size instead of removal count |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Bearer tokens use non-cryptographic randomness |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Bearer tokens are written to logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Expired sessions are never removed from memory; review-security: Expired bearer sessions remain valid |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-security: Administrator role check is logically inverted |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords use unsalted SHA-256 |
| T22 | — | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  |  |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction uses insertion order rather than least-recent use |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-concurrency: Cache deletion never cancels or retires TTL timers |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-bugs: Cache deletion leaves stale timers that can remove refreshed |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache utilization rounds down before calculating percent |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache keys always returns an empty array |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared default configuration |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid timeout logs fallback but stores NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Millisecond cache TTL is multiplied by 1000 |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| T37 | YES | parallel: CF05 loadFile swallows every error, so an unreadable config  | review-bugs: All config-file read failures are treated as a missing file |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation always succeeds |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: User-controlled report name escapes the export directory |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Export permissions do not enforce owner-only access |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are written without escaping; review-security: CSV export permits formula injection |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Owner value controls archive filename path |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Snapshot writes race through one fixed temporary path; review-security: Fixed snapshot path is vulnerable to symlink attacks |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport adds a bogus row for the trailing newline |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: Warm-cache failures are detached from the returned promise; review-concurrency: warmCache resolves before its asynchronous work completes |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets does not return or stop at the first failure; review-concurrency: checkTargets launches every probe and cannot abandon failure |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit ignores write completion and failures; review-concurrency: flushAudit detaches the asynchronous write |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for an owner with no links |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed create-link JSON escapes the error handler |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Nonnumeric TTL creates a permanently unprunable link; review-bugs: Handled creation errors never update the error metric |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: Redirect query handling crashes or misses stored codes |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Authenticated link listing and report expose all owners; review-security: Any authenticated user can delete any link |
| T55 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts a spoofable client header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: Empty-store stats serialize hits_per_link as null |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: Requests without an Authorization header throw |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Client can spoof the link owner |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-bugs: CSV report silently omits links after the first 1000 |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report export reads the wrong filename |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/cache.ts:46 — A zero-sized cache still stores one entry
- (medium) src/worker.ts:41 — probe reports unreachable HTTPS targets as reachable
- (low) src/main.ts:162 — Export failures disclose filesystem details
