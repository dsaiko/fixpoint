# minimax-m3:cloud · java · run 20260901-101942 (repeat 1)

recall **37/69** · 49 finding(s), 8 unmatched · 806480 tokens · 588s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | — | JAVA-ONLY equals is overridden without hashCode, so links break as map |  |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() NPEs for new owners via auto-unboxing |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes live links on every hit |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the wrong variable twice |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() off-by-one can throw IndexOutOfBoundsException |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() always reports 0 or 100 |
| J07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| J08 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses contains() and leaks other users' links; review-security: byOwner uses substring match, leaking adjacent owners |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares codes with `==` |
| J11 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() returns the live internal map, contradicting the do |
| J13 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() mutates the map while iterating its keySet |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-security: Short codes are predictable (java.util.Random) |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret allows token forgery |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens are predictable (java.util.Random) |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued session tokens are written to stdout |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-security: verify() ignores session.expiresAt |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares tokens with `==` |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing is unsalted SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() NPEs on a missing or single-word header; review-security: bearerToken crashes on missing/malformed header |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin rejects admins; review-security: requireAdmin() always rejects (and is never called) |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals is timing-vulnerable |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | — | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre |  |
| J30 | — | parallel: CA02 peek reads the map with no synchronization while synchr |  |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: set() exceeds the cache limit on every full insert |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | — | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C |  |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() reports 0 until the cache is completely full |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS zeroes the timeout, then validate() fai |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS parse error is uncaught |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT bounds check is a no-op |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | — | parallel: CF06 validate logs the problems it finds and returns true, s |  |
| J42 | — | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  |  |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-bugs: exportCsv() allows path traversal through `name`; review-security: Path traversal in exportCsv/readReport/archiveAll |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot() rename is unreliable across filesystems |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: writeSnapshot writes to a hard-coded /tmp path |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-bugs: Metrics.addCreated() is not synchronized |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor mutates Store.all() while iterating its keySet |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache leaks its ExecutorService and returns before work  |
| J55 | — | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null |  |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Failed-auth response echoes the presented token |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links bypasses owner check |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out?next= |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin gate is a client-controlled header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats() divides by store.size() without guarding zero |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() NPEs when ?code= is missing |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows wildcard origin with credentials |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Config.java:117 — validate() always reports success
- (critical) Store.java:? — Unsynchronized HashMap accessed by multiple HTTP handler threads and the janitor
- (critical) Authenticator.java:? — Sessions HashMap accessed from every HTTP thread without synchronization
- (high) Cache.java:57 — `peek` reads `entries` without synchronization while mutators are synchronized
- (high) Worker.java:? — Janitor thread iterates a live, unsynchronized HashMap
- (medium) Cache.java:? — Double-checked locking in `getInstance` is not safe
- (medium) Worker.java:? — Metrics counters are not synchronized between writers and snapshot
- (medium) Worker.java:? — `warmCache` leaks the executor and ignores its Javadoc contract
