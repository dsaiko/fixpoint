# codex@moonshotai/kimi-k3.1-pro · typescript · run 20261024-222328 (repeat 1)

recall **44/63** · 52 finding(s), 2 unmatched · 835039 tokens · 514s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: A new owner's quota becomes NaN; review-bugs: Generated-code collisions silently overwrite existing links |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiration check is inverted |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source twice and never validates the de |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate rounds the fraction before converting to a perce |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting links does not release owner quota |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: Leaderboard sorting is ascending instead of descending |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner returns substring matches instead of exact ownership |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives links that have already expired |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas exposes mutable internal state as a snapshot |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Failed batch imports leave partial writes behind |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns remaining links instead of removed count |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens use predictable randomness |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Bearer tokens are written to logs |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-security: Expired session tokens remain valid |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | — | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an |  |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashes are fast and unsalted |
| T22 | — | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  |  |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction ignores recency and unnecessary evicts on upd |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Stale expiry timers can delete refreshed entries; review-concurrency: Stale expiration timers delete refreshed entries |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: Cache utilization remains zero until the cache is full |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: Cache keys are always reported as empty |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig permanently mutates the defaults |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid timeout values overwrite the default with NaN |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: Millisecond cache TTLs are multiplied by 1000 |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| T37 | YES | parallel: CF05 loadFile swallows every error, so an unreadable config  | review-bugs: All configuration-file read failures are treated as missing  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation can never stop an invalid startup |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report name permits writes outside the export directory |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV exports corrupt rows containing commas, quotes, or newli; review-security: CSV export allows spreadsheet formula injection |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-concurrency: Concurrent snapshots share one temporary file |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: Every exported report parses with a phantom final row |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache resolves before any asynchronous warm-up finishes; review-concurrency: warmCache returns before its probes finish |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: Probe exceptions reject checkTargets instead of identifying  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: Audit flush neither awaits nor retries failed writes; review-concurrency: flushAudit discards the pending write |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: Unknown owners produce NaN instead of a zero count |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-bugs: The exported error metric can never increase |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: Malformed JSON escapes the create-link error handler |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: A redirect without a code throws |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: Any authenticated user can delete any link; review-security: Link listing exposes every owner's links |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Authenticated open redirect enables trusted-domain phishing |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts a client-supplied header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: Empty-store statistics serialize hits_per_link as null |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: Missing Authorization headers crash request handling |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Clients can impersonate arbitrary link owners |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-bugs: Reports silently omit links after the first 1000 |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report requests write and read different files |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/export.ts:47 — Snapshot rename fails when /tmp and the destination are on different filesystems
- (medium) src/main.ts:32 — Janitor interval is started without a lifecycle owner
