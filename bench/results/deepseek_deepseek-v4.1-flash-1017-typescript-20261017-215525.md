# deepseek/deepseek-v4.1-flash-1017 · typescript · run 20261017-215525 (repeat 1)

recall **49/63** · 64 finding(s), 1 unmatched · 139880 tokens · 363s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter becomes NaN the first time an owner creates a  |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry check in Store.resolve is inverted: live links are de |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() always reports 0 unless every link was followe |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not give the owner's quota slot back |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, returning the least-followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring matching and returns other owners' ; review-security: byOwner matches substrings, leaking other owners' links |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links, contrary to its docu |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object rather than a snap |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: invalid entries leave earlier ent |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining link count instead of the numb |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-bugs: expiresAt() dereferences a missing link with a non-null asse |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Short codes are generated with Math.random and are predictab |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret is used when LINKD_SECRET is unset |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens are minted with Math.random, not a CSPRNG |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Full session bearer token written to the log |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Authenticator.verify never checks expiresAt, so sessions nev; review-security: verify() never checks expiresAt, so sessions never expire |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin treats indexOf's return value as a boolean, all; review-security: requireAdmin logic is inverted: admins are denied and everyo |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken dereferences a possibly-undefined header, crashi; review-security: bearerToken throws on a missing Authorization header |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals is not constant-time despite its contract |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction is FIFO, not LRU; the maintained `used` times |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache.delete never clears the timer; timers accumulate and a; review-concurrency: Stale TTL timers are never cancelled and delete re-set entri |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors before multiplying, so it always report |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for..in over a Map, so it always returns an empt |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG object instead  |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Bad LINKD_TIMEOUT_MS logs "keeping default" but assigns NaN  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is treated as seconds although it is docu |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded, so the setting ha |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so bad configuration is neve |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report name allows arbitrary .csv read/wri |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report files are world-readable despite the comment claiming |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are written without quoting or escaping; review-security: CSV formula injection in exported reports |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot uses a fixed cross-process temp path and renam; review-concurrency: writeSnapshot races concurrent writers through a fixed /tmp  |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport turns the trailing newline into a bogus row and  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache resolves before any probe finishes and leaks rejec; review-concurrency: warmCache returns before probes finish and leaks rejected pr |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets can never report a failure and rejects instead ; review-concurrency: checkTargets has no cancellation or timeout, and rejecting p |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit does not await or retry the write, silently dropp; review-concurrency: flushAudit fires the write and drops the promise; no retry,  |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for an owner with no links |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-concurrency: Janitor takes the cache but never invalidates it, so store e |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: createLink does not validate the requested TTL, producing Na |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() dereferences a missing code query parameter |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: DELETE /links deletes any owner's link; ownership check docu |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on a client-supplied X-Admin heade |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats() divides by zero on an empty store, emitting null for |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Wildcard CORS origin combined with credentials |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: JSON.parse in createLink runs outside the try, so malformed ; review-security: Link owner is taken from the request body instead of the ses |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports and returns every owner's links to any authe |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() exports under the default name "links" but reads ba |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/main.ts:32 — Janitor interval is started at module load and its stopper is discarded
