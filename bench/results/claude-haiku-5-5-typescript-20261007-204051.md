# claude-haiku-5-5 · typescript · run 20261007-204051 (repeat 1)

recall **51/63** · 77 finding(s), 4 unmatched · 84978 tokens · 404s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: First link per owner sets quota to NaN; review-security: Quota is never enforced and its counter is NaN for every own |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry check, deleting live links and ; review-security: resolve() deletes live links and serves expired ones |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates; review-security: rename validates the old code twice and never validates the  |
| T04 | YES | parallel: C04 page clamps end past the array length; slice silently re | review-bugs: page() sets end to all.length + 1, which is misleading altho |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() floors the ratio before multiplying, so it is  |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not give the owner their quota slot back |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, so the leaderboard lists the least fo |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches by substring, so one owner's listing inclu; review-security: byOwner matches owners by substring, leaking other owners' l |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives links that have already expired; review-security: extend revives expired links, contrary to its documented inv |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object although its doc p; review-concurrency: quotas() returns the live internal map although its doc says |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic despite its doc comment; review-security: importAll skips the target scheme check and overwrites exist |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the number of remaining links, not the numbe |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | YES | TS-ONLY (parallel: S02) short codes come from Math.random, which is no | review-security: Link codes come from Math.random and are not checked for col |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret is used when LINKD_SECRET is unset |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens come from Math.random and the clock, not a CS |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Full session bearer token is written to the log on every iss |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() accepts sessions after their expiresAt has passed; review-security: verify() never checks expiresAt, so sessions never expire |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin() is inverted: it allows every role except 'adm; review-security: requireAdmin is inverted: admins are rejected and every othe |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: hashPassword is unsalted SHA-256, which is weak for stored p |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken() throws when the Authorization header is absent; review-security: A missing Authorization header makes bearerToken throw |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals uses === although its doc comment promises cons |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: set() evicts the oldest inserted entry, not the least recent |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Timers map grows without bound because delete() never clears; review-bugs: A stale timer evicts an entry that was set again before its  |
| T28 | — | delete promises to stop the work scheduled for the entry and never tou |  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors the ratio before multiplying, so it is  |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for...in over a Map and always returns an empty  |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig() mutates the shared DEFAULT_CONFIG object; review-concurrency: loadConfig mutates the shared DEFAULT_CONFIG object in place |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: A non-numeric LINKD_TIMEOUT_MS is stored as NaN although the |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 although the doc sa |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so invalid configuration nev |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: User-supplied report name allows path traversal on write and |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Export file with all owners' links is written world-readable |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV rows are written without escaping, so a comma in a field; review-security: CSV export writes owner and target unescaped, allowing row a |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll builds filenames from raw owner strings |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot renames across filesystems from a fixed /tmp p; review-concurrency: writeSnapshot uses a fixed shared temp path, so concurrent w |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport() adds a bogus row for the trailing newline that |
| T45 | YES | the errors counter is reported by snapshot and never incremented anywh | review-bugs: Metrics.errors is reported but never incremented |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() resolves before its probes finish, and unhandled; review-concurrency: warmCache fires probes without awaiting them and loses rejec |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-concurrency: checkTargets fans out unbounded probes, ignores fetchLimit,  |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() does not retry and drops the write's promise; review-concurrency: flushAudit fires the write without awaiting it, so failures  |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() returns NaN for an owner with no links |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-security: JSON.parse on the request body is outside the try block |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: A non-numeric ttl produces a link with NaN expiry that never |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: GET /l without a code parameter throws instead of returning  |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: deleteLink has no ownership check, so any session can delete; review-security: Listing returns every owner's links with no ownership filter |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out redirects to any caller-supplied next URL, including no |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quota view is gated by a client-controlled header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: hits_per_link is NaN on an empty store and serialises as nul |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented token back to the client |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: createLink() parses the request body outside its try block; review-security: Link owner is taken from the request body, so callers can im |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() without a name always returns 500: the write uses ' |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/store.ts:33 — create() silently overwrites an existing link when a random code collides
- (medium) src/worker.ts:37 — probe() never checks reachability, so checkTargets() never reports a failure
- (low) src/main.ts:32 — The janitor interval starts at import time and its stopper is discarded, so it can never b
- (low) src/main.ts:162 — Export failure response discloses the server's filesystem path and raw error
