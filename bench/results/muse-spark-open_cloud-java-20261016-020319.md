# muse-spark-open:cloud · java · run 20261016-020319 (repeat 1)

recall **35/69** · 37 finding(s), 0 unmatched · 183230 tokens · 147s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Link overrides equals without hashCode breaking hash collect |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: First link for any owner crashes with NPE on quota update |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted expiry check deletes live links and serves expired  |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates source twice and never validates destinatio |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Page clamp off-by-one causes IndexOutOfBoundsException |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate truncated to 0% by integer division |
| J07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns least-popular links instead of most-popular |
| J09 | — | parallel: C11 byOwner matches owners by substring, so one owner sees a |  |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode uses reference equality so lookups miss |
| J11 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| J14 | — | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i |  |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | — | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr |  |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store shared across HTTP handlers and janitor thread with ze |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when env var unset |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Predictable session token via java.util.Random |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to stdout logs |
| J21 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Token verification uses reference equality so valid tokens f; review-security: Token verification uses reference equality (==) not value eq |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken crashes on missing or malformed Authorization he |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: Admin check is a tautology so every caller is rejected; review-security: requireAdmin logic always denies due to OR instead of AND |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Authenticator sessions HashMap and Random accessed concurren |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Cache.getInstance double-checked without synchronization cre |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Cache.peek/stats/utilization unsynchronized while get/set/sw |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | — | parallel: CA08 eviction drops whatever key the iterator yields first,  |  |
| J33 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: Cache.warm check-then-act TOCTOU between peek and set |
| J34 | — | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C |  |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization truncated to 0% by integer division |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS mis-handled: uncaught exception and wrong |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT env value is parsed but never applied |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when config is invalid |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Static SimpleDateFormat shared across threads is not thread- |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in exportCsv allows arbitrary file write |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | — | parallel: EX06 renameTo's boolean result is discarded and it fails sil |  |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated unsynchronized while snapshot/addResolved |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-concurrency: Janitor directly iterates and mutates Store map without sync |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-concurrency: warmCache leaks thread pool and returns before work complete |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner unboxes null for unknown owner |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Missing ownership check on DELETE /links allows IDOR |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= to arbitrary URL |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin check trusts client-controlled X-Admin header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats divides by zero on empty store and truncates hits-per- |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect dereferences null code when query param is absent |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: wildcard origin with credentials |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |
