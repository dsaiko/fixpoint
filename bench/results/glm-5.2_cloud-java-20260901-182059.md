# glm-5.2:cloud · java · run 20260901-182059 (repeat 1)

recall **42/69** · 50 finding(s), 2 unmatched · 488850 tokens · 615s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | — | JAVA-ONLY equals is overridden without hashCode, so links break as map |  |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() NPEs on a new owner because quota.get(owner) is nul |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry condition is inverted: valid links are kill; review-security: Expiry check is inverted: expired links keep resolving, live |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() sets end = all.size() + 1, throwing IndexOutOfBoundsE |
| J06 | — | parallel: C05 integer division before scaling yields 0 for every rate  |  |
| J07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending then takes the first n, returning the  |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring contains() instead of equals, match |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() uses == reference equality and never matches |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() unconditionally revives expired links, violating th |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() returns the live quota map, not a snapshot |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: a failed validation midway leaves |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() throws ConcurrentModificationException by removing d |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-security: Short codes generated with non-cryptographic java.util.Rando |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store's shared HashMaps are mutated from concurrent handler  |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens generated with non-cryptographic java.util.Ra |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to stdout on issue |
| J21 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares tokens with == (reference equality), so ev; review-security: Token verification uses reference equality, so every session |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses unsalted SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() crashes on a missing/malformed Authorization h; review-security: bearerToken crashes on missing or malformed Authorization he |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin() always returns false due to || instead of &&; review-security: requireAdmin logic is always false (dead admin gate) |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals is not constant-time despite its doc comment |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Cache.getInstance has a check-then-act race on the static si |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Cache's peek/stats/utilization/warm access shared state with |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | — | parallel: CA08 eviction drops whatever key the iterator yields first,  |  |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() throws ConcurrentModificationException by removing d |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() integer-divides size by limit, yielding 0 unti |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeoutMillis to 0 instead of keep |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is parsed as milliseconds but stored into |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied to cfg.fetchLi |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when problems were colle |
| J42 | — | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  |  |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal via user-controlled report name in exportCsv/ |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll writes files named by owner with no path sanitiza |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot() ignores renameTo() failure and uses a fixed  |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Snapshot written to a fixed, predictable temp path (symlink  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated is unsynchronized while sibling counters  |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor loop throws ConcurrentModificationException removing; review-concurrency: Janitor iterates and mutates the live store map concurrently |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache() never shuts down its ExecutorService and returns; review-concurrency: warmCache leaks its thread pool and returns before the submi |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() NPEs on unboxing when the owner is unknown |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: Link owner taken from query parameter, not the authenticated |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links has no ownership check |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unvalidated next parameter on /out |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin gate trusts a client-supplied X-Admin header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats endpoint divides hits by store.size(), crashing on an  |
| J65 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin with credentials |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Authenticator.java:48 — Authenticator's sessions map is iterated and mutated concurrently without synchronization
- (medium) src/linkd/Main.java:91 — JSON response built by string concatenation, allowing JSON injection
