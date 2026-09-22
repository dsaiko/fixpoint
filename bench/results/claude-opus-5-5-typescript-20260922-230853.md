# claude-opus-5-5 · typescript · run 20260922-230853 (repeat 1)

recall **53/63** · 70 finding(s), 5 unmatched · 22575 tokens · 238s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter starts at NaN for every new owner |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links are deleted,  |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates `from` twice and never validates `to` |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate floors before scaling, so it is 0 unless every l |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not give the owner's quota slot back |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, returning the least-followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner matches by substring, not exact owner; review-security: byOwner matches by substring, leaking other owners' links |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object, not a snapshot |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining size instead of the number rem |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-bugs: newCode can collide with an existing code and overwrite that; review-security: Short codes generated with Math.random are predictable and e |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing secret |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens are generated with Math.random (predictable) |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session bearer tokens are written to the log |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() ignores session expiry; review-security: verify() ignores session expiry, so tokens never expire |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin uses indexOf as a boolean; review-security: requireAdmin grants admin to every role except 'admin' |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with a single unsalted SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken crashes on a missing or malformed Authorization  |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals is not constant-time despite its contract |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours accepts garbage and returns NaN |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction removes the oldest-inserted entry, not the least re |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: TTL timers are never cleared: stale timers delete fresh entr |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-concurrency: Per-entry TTL timers are never cancelled, so a stale timer d |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors before scaling, so it is 0% until the c |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map with for...in and always returns [] |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: Invalid LINKD_TIMEOUT_MS is logged as 'keeping default' but  |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 despite being docum |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| T37 | YES | parallel: CF05 loadFile swallows every error, so an unreadable config  | review-bugs: loadFile swallows every read error, not only a missing file |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, and main ignores the result |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report name: arbitrary .csv file write and |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report containing every owner's links is written with defaul |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are not escaped, so parseReport misreads rows; review-security: CSV/formula injection in exported reports |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Owner name used unsanitized as archive filename (path traver |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot renames from /tmp, which fails across filesyst; review-security: Snapshot staged at a fixed, predictable /tmp path (symlink/T |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport emits a bogus empty row for the trailing newline |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache does not wait for its probes, and probe rejections; review-concurrency: warmCache resolves before any probe finishes, and probe reje |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets rejects on the first throwing probe instead of ; review-concurrency: checkTargets does not abandon remaining probes; one throwing |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit neither retries nor handles rejection; review-concurrency: flushAudit neither awaits nor retries the write, leaving a f |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for owners with no links |
| T50 | YES | the janitor takes the cache and never sweeps it, so expired cache entr | review-bugs: Janitor never evicts expired links from the cache |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Non-numeric ttl makes a link with expiresAt NaN that never e |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect crashes on missing code and lowercases case-sensiti |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: deleteLink has no ownership check: any user deletes anyone's; review-security: Listing and report endpoints expose every user's links to an |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: /out uses 301, so browsers cache it and repeat clicks are no; review-security: Open redirect via /out?next= |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint is gated by a client-supplied x-admin header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: hits_per_link is NaN when the store is empty |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: Wildcard CORS combined with Allow-Credentials, and internal  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: JSON.parse of the request body is outside the try block; review-security: Link owner is taken from the request body, not the authentic |
| T61 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-bugs: Report silently truncates at 1000 links |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report with no name writes links.csv but reads .csv |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/main.ts:106 — Listing does not validate offset and size
- (low) src/cache.ts:38 — peek() reports expired entries as present, so warm() skips refreshing them
- (low) src/cache.ts:48 — Eviction skips an empty-string key
- (low) src/config.ts:88 — cache_size in the config file is not checked for NaN
- (low) src/main.ts:32 — Janitor stop function is discarded, and the janitor never invalidates the cache
