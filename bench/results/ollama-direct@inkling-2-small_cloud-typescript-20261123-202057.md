# ollama-direct@inkling-2-small:cloud · typescript · run 20261123-202057 (repeat 1)

recall **48/63** · 69 finding(s), 1 unmatched · 313923 tokens · 228s

| seed | found | note | matched by |
|---|---|---|---|
| T01 | YES | TS-ONLY (parallel: C01) a new owner has no quota entry, and the non-nu | review-bugs: Quota counter becomes NaN for every owner |
| T02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry check in resolve() is inverted; review-security: resolve() expiry check is inverted: valid links are deleted, |
| T03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates; review-security: rename() validates 'from' twice and never validates 'to' |
| T04 | — | parallel: C04 page clamps end past the array length; slice silently re |  |
| T05 | YES | TS-ONLY (parallel: C05) Math.floor is applied to the ratio BEFORE mult | review-bugs: successRate() always returns 0 |
| T06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| T07 | YES | parallel: C10 top sorts ascending, and the comparator never returns 0, | review-bugs: top() sorts ascending, returning the least-followed links |
| T08 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches substrings, not exact owners; review-security: byOwner uses substring matching — returns other tenants' lin |
| T09 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects expired links, violating the documented ; review-security: extend() resurrects already-expired links, contradicting 'ex |
| T10 | YES | parallel: C14 quotas hands out the store's own object while the doc ca | review-bugs: quotas() leaks the live internal quota object; review-concurrency: quotas() returns the store's internal quota object, not a sn |
| T11 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic and ignores quotas; review-security: importAll is not atomic and silently overwrites existing cod |
| T12 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining store size instead of the remo |
| T13 | — | TS-ONLY expiresAt asserts non-null on a lookup that can miss, throwing |  |
| T14 | — | TS-ONLY (parallel: S02) short codes come from Math.random, which is no |  |
| T15 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' |
| T16 | YES | TS-ONLY (parallel: S02) session tokens are Math.random plus a timestam | review-security: Session tokens generated with Math.random() — predictable |
| T17 | YES | parallel: S03 every issued session token is logged in cleartext | review-security: Session tokens written to logs in cleartext |
| T18 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks session expiry; expired tokens stay va; review-concurrency: verify() returns the live internal Session object by referen |
| T19 | — | TS-ONLY verify uses loose equality on the token, so type coercion deci |  |
| T20 | YES | TS-ONLY indexOf returns 0 for admin (falsy, so admins are rejected) an | review-bugs: requireAdmin admin check is inverted by indexOf truthiness; review-security: requireAdmin authorization check inverted — non-admins pass, |
| T21 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted single-round SHA-256 |
| T22 | YES | TS-ONLY (parallel: S06) bearerToken asserts the header is present and  | review-bugs: bearerToken throws on a missing Authorization header, and ha; review-security: bearerToken dereferences the header without checking it exis |
| T23 | YES | secretEquals is documented as not leaking how much matched and uses == | review-security: secretEquals is not constant-time despite its contract |
| T24 | YES | TS-ONLY parseInt is called with no radix, so a leading-zero lifetime i | review-bugs: parseLifetimeHours accepts garbage as NaN silently |
| T25 | — | sessions are inserted and never evicted, so the map grows without boun |  |
| T26 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction is insertion-ordered, not LRU, and fires even when  |
| T27 | YES | TS-ONLY a timer is stored per set and never cleared -- delete leaves i | review-bugs: Stale timers delete re-set entries early; delete() never cle; review-security: delete() never clears or removes the entry's TTL timer |
| T28 | YES | delete promises to stop the work scheduled for the entry and never tou | review-concurrency: delete() never clears the entry's TTL timer; stale timers de |
| T29 | YES | TS-ONLY utilization floors the ratio before multiplying so it always r | review-bugs: utilization() is always 0 or 100 |
| T30 | YES | TS-ONLY keys() uses for..in over a Map, which enumerates own enumerabl | review-bugs: keys() uses for..in over a Map and always returns [] |
| T31 | — | parallel: CA06 warm's peek and set are separate operations, so a concu |  |
| T32 | YES | TS-ONLY loadConfig assigns the exported DEFAULT_CONFIG object rather t | review-bugs: loadConfig mutates the shared DEFAULT_CONFIG object |
| T33 | — | parallel: CF01 LINKD_PORT is parsed with no radix and no NaN check, so |  |
| T34 | YES | parallel: CF02 the NaN branch logs 'keeping default' and then assigns  | review-bugs: NaN LINKD_TIMEOUT_MS is applied even after being detected |
| T35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and mu | review-bugs: LINKD_CACHE_TTL_MS is multiplied by 1000 despite being docum |
| T36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated, and then discar | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| T37 | — | parallel: CF05 loadFile swallows every error, so an unreadable config  |  |
| T38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true and lets bad configs start; review-security: validate() always returns true — broken config is never refu |
| T39 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report name — arbitrary .csv read and writ |
| T40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report files written with default permissions despite owner- |
| T41 | YES | parallel: EX03 CSV rows are built by template interpolation, so a comm | review-bugs: CSV output is never escaped; review-security: CSV formula injection via unescaped user-controlled fields |
| T42 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll writes to path derived from unsanitized owner — t |
| T43 | YES | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  | review-bugs: Snapshot write can fail or corrupt: fixed /tmp path, cross-d; review-concurrency: writeSnapshot uses one fixed tmp path; concurrent snapshots  |
| T44 | YES | TS-ONLY parseReport asserts a header row exists and casts every cell w | review-bugs: parseReport crashes on empty input and fabricates a phantom  |
| T45 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| T46 | YES | TS-ONLY warmCache passes an async callback to forEach, which ignores t | review-bugs: warmCache never awaits its probes and drops all failures; review-concurrency: warmCache never awaits the per-pair async work; failures bec |
| T47 | YES | TS-ONLY checkTargets is documented to report the first failure, but pr | review-bugs: checkTargets reports rejections instead of the failing targe; review-concurrency: checkTargets uses Promise.all: failure rejects the whole cal |
| T48 | YES | TS-ONLY flushAudit calls an async function and neither awaits it nor a | review-bugs: flushAudit fires the write once, unawaited, with no retry; review-concurrency: flushAudit fires write() without awaiting, catching, or retr |
| T49 | YES | TS-ONLY countForOwner adds 0 to a possibly-undefined lookup, returning | review-bugs: countForOwner returns NaN for owners with no quota entry |
| T50 | — | the janitor takes the cache and never sweeps it, so expired cache entr |  |
| T51 | — | TS-ONLY the request body is cast to `any`, so target and owner reach t |  |
| T52 | YES | a bad ttl parses to NaN and produces a link whose expiry is NaN, which | review-bugs: Unparseable JSON body throws out of createLink before the tr |
| T53 | YES | parallel: C16 redirect lowercases the code before lookup although code | review-bugs: redirect() crashes when ?code is missing and mangles valid c |
| T54 | YES | parallel: C17 deleteLink never checks the owner its doc promises to ch | review-security: deleteLink performs no ownership or role check — any user ca; review-security: listLinks returns every link of every owner to any authentic |
| T55 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out (?next= is reflected into Location) |
| T56 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin gate is a forgeable x-admin: true header |
| T57 | YES | parallel: C20 /stats divides by the link count, producing NaN on an em | review-bugs: hits_per_link divides by zero on an empty store |
| T58 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every J | review-security: CORS allows any origin together with credentials |
| T59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| T60 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: createLink trusts client-supplied owner instead of the sessi |
| T61 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| T62 | — | the created link is serialized wholesale, returning the owner field an |  |
| T63 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/cache.ts:38 — peek() reports expired entries as cached
