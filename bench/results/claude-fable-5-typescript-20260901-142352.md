# claude-fable-5 · typescript · run 20260901-142352 (repeat 1)

recall **36/63** · 41 finding(s), 1 unmatched · 39726 tokens · 479s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota increment on first link is undefined + 1 = NaN, perman |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links are deleted,  |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice and never validates 'to' |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() floors before multiplying, so it only ever ret |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, returning the LEAST followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches by substring, returning other owners' link |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links despite the documented 'expir |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the live internal object, not a snapshot |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: a mid-batch validation failure le |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the surviving link count, not the number rem |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | — | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam |  |
| T17 | — | parallel: S03 every issued session token is logged in cleartext |  |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt, so sessions live forever |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin() truthiness on indexOf inverts the role check |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken() crashes on a missing Authorization header |
| T23 | — | secretEquals is documented as not leaking how much matched and uses == |  |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction removes the oldest-inserted entry, not the least re |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Stale TTL timer deletes a re-set entry early; timers are nev |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-concurrency: Cache never clears its expiry timers: stale timer deletes a  |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors before multiplying, reporting 0 until t |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map with for…in and always returns an empt |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig() mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: NaN LINKD_TIMEOUT_MS is assigned despite the 'keeping defaul |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 despite being docum |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() collects problems but always returns true |
| T39 | — | parallel: EX01 the report name comes from the query string and is join |  |
| T40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV fields are not escaped, so commas in targets shift colum |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot temp file is written to /tmp, so renameSync can fai; review-concurrency: writeSnapshot funnels all writers through one fixed /tmp pat |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport() turns the trailing newline into a garbage row  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache() resolves immediately and leaks unhandled rejecti; review-concurrency: warmCache fires unawaited async callbacks: resolves early an |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets() rejects on probe failure instead of reporting; review-concurrency: checkTargets rejects on the first bad target instead of repo |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() neither retries nor handles rejection despite i; review-concurrency: flushAudit drops the write promise: no await, no retry, unha |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner() returns NaN for owners with no quota entry |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() crashes when the code query parameter is missing |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink() never checks ownership despite the documented c |
| T55 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| T56 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: stats() divides by zero on an empty store |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: createLink() parses the request body outside the try block |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() writes with a default name but reads back with the  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/main.ts:32 — Janitor's stop function is discarded, so the interval can never be cancelled
