# minimax-m3.1-flash:cloud · typescript · run 20261019-182010 (repeat 1)

recall **41/63** · 69 finding(s), 2 unmatched · 301704 tokens · 2077s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter becomes NaN for every new owner (undefined + 1 |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() has an inverted expiry check: live links 404 and a; review-concurrency: resolve() inverts its expiry test, and the janitor's read-th |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to`; review-concurrency: rename() validates `from` twice and never validates `to` |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() floors before multiplying, so it reports 0% un |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never gives the owner's quota slot back; review-concurrency: delete() never returns the owner's quota slot, and deleteLin |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top()'s sort comparator never returns 0 for equal hit counts |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses a substring match, so it returns other owners |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects an already-expired link despite the docu |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() returns the store's live quota object, not a snapsh; review-concurrency: quotas() returns the live internal quota object, not a snaps |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: validation happens inside the mut |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the number of links remaining, not the numbe; review-concurrency: prune() returns the post-prune size instead of the number of |
| T13 | YES | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing | review-concurrency: expiresAt() uses a non-null assertion on a lookup that the j |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| T16 | — | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam |  |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Every issued session token is written to stdout; review-security: Session tokens minted from Math.random() are predictable |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt, so expired sessions stay va; review-concurrency: verify() ignores the session expiry, so expired tokens keep  |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin admits every role except admin: indexOf returns; review-concurrency: requireAdmin returns true for every role, so the admin check |
| T21 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken throws TypeError on a request with no Authorizat; review-concurrency: bearerToken throws a TypeError when the Authorization header |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-concurrency: secretEquals compares with ===, so it leaks match length thr |
| T24 | — | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i |  |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction picks the oldest inserted entry, not the least rece; review-concurrency: Eviction picks the insertion-order first key, not the least  |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Cache timers are never cleared: a stale timer deletes a refr |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-concurrency: Cache.delete never cancels the TTL timer, so a stale timer d |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() floors before multiplying, so it reports 0% fo |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() iterates a Map with for...in, so it always returns an; review-concurrency: utilization() computes floor(ratio) * 100, yielding only 0 o |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig() mutates the exported DEFAULT_CONFIG object inst; review-concurrency: loadConfig() mutates the shared DEFAULT_CONFIG object instea |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: A bad LINKD_TIMEOUT_MS is logged as "keeping default" but th; review-concurrency: A NaN LINKD_TIMEOUT_MS is assigned even though the code logs |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 although the docume; review-concurrency: LINKD_CACHE_TTL_MS is multiplied by 1000 although it is docu |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed, checked, then discarded: fetchL |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so an invalid configuration  |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-bugs: The user-supplied report name is not confined to the export ; review-concurrency: The report name from the user escapes the export directory ( |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-bugs: The report is written with default 0644 permissions although; review-concurrency: The export is written with the default 0666 mode despite the |
| T41 | — | parallel: EX03 CSV rows are built by template interpolation, so a comm |  |
| T42 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: writeSnapshot uses a fixed temp path in /tmp: writers race, ; review-concurrency: writeSnapshot uses a fixed temp path, so concurrent snapshot |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport returns a spurious empty row for every file expo |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache resolves before the probes finish and leaks unhand; review-concurrency: warmCache resolves before any probe finishes and swallows pr |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets() can never report a failing target: probe() th; review-concurrency: checkTargets waits for every probe instead of abandoning the |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit() never retries and drops the write promise, swal; review-concurrency: flushAudit drops the write promise, so audit failures become |
| T49 | — | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning |  |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | YES | TS-ONLY the request body is cast to `any`, so target and owner reach t | review-bugs: createLink parses JSON outside the try block, so a malformed; review-concurrency: createLink parses the request body outside the try block, so |
| T52 | — | a bad ttl parses to NaN and produces a link whose expiry is NaN, which |  |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() calls .toLowerCase() on an optional query param a |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-bugs: deleteLink performs no ownership check, so any authenticated; review-security: deleteLink deletes any link by code, ignoring the owner's se |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out reflects an arbitrary attacker-chosen UR |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-concurrency: The admin endpoint is gated on a client-supplied header inst; review-security: Administrative endpoint authorized by a client-supplied head |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: hits_per_link is NaN (serialized as null) when the store is  |
| T58 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every J |  |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report without a name parameter exports links.csv but then  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/store.ts:198 — validateCode()'s regex is unanchored, so codes with invalid characters pass validation
- (medium) src/main.ts:32 — The janitor's setInterval handle is discarded, keeping the event loop alive forever
