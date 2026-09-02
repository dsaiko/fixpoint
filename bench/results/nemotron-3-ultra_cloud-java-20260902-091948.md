# nemotron-3-ultra:cloud · java · run 20260902-091948 (repeat 1)

recall **47/69** · 80 finding(s), 7 unmatched · 563076 tokens · 1909s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() can throw NPE if other.code is null; review-bugs: equals() overridden but hashCode() not overridden |
| J02 | — | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox |  |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check inverted - removes valid links; review-security: Logic error in resolve() removes valid links |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' code twice instead of 'to' |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() off-by-one error in end calculation |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() integer division produces 0 for any non-100% r |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-concurrency: delete() removes link but doesn't decrement quota - quota le |
| J08 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses contains() instead of equals(); review-security: byOwner() uses contains() allowing cross-owner data leakage |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() uses reference equality (==) instead of .equals; review-concurrency: findByCode() uses '==' for String comparison |
| J11 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-concurrency: quotas() and all() return internal mutable maps - encapsulat |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() not atomic - partial imports on failure; review-security: importAll() bypasses code validation and lacks atomicity |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() ConcurrentModificationException when removing during; review-concurrency: prune() causes ConcurrentModificationException by modifying  |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | — | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr |  |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: HashMap 'links' accessed concurrently without synchronizatio |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret in production |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-bugs: nextLong() can produce negative values, toHexString includes |
| J21 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: String comparison uses reference equality (==) instead of .e; review-concurrency: verify() uses '==' for String comparison instead of .equals( |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses unsalted SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() throws ArrayIndexOutOfBoundsException on malfo; review-security: bearerToken() throws ArrayIndexOutOfBoundsException on malfo |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin logic inverted - always returns false; review-concurrency: requireAdmin() logic always returns false - admin/owner chec |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals() uses non-constant-time comparison |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: HashMap 'sessions' accessed concurrently without synchroniza |
| J28 | YES | JAVA-ONLY hashPassword returns the empty string when the digest is una | review-bugs: hashPassword() returns empty string on NoSuchAlgorithmExcept |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Broken double-checked locking in getInstance() - instance no |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek() reads HashMap unsynchronized while other methods writ |
| J31 | YES | parallel: CA07 stats reads the non-volatile hit counter with no synchr | review-concurrency: stats() reads 'hits' counter unsynchronized while get() incr; review-concurrency: utilization() reads entries.size() and limit unsynchronized |
| J32 | — | parallel: CA08 eviction drops whatever key the iterator yields first,  |  |
| J33 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: warm() has TOCTOU race between peek() and set(); review-security: TOCTOU race in warm() |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() ConcurrentModificationException when removing during; review-concurrency: sweep() causes ConcurrentModificationException by modifying  |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() integer division always returns 0 |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-security: Unit confusion: LINKD_CACHE_TTL_MS parsed as seconds |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT parsed but never assigned to cfg.fetchLimi; review-security: LINKD_FETCH_LIMIT parsed but never assigned |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when validation fails; review-security: validate() always returns true even with errors |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-bugs: SimpleDateFormat is not thread-safe; review-concurrency: Static SimpleDateFormat STAMP is not thread-safe |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-bugs: exportCsv() path traversal vulnerability via name parameter |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: FileWriter not closed on exception - resource leak |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot() renameTo() return value ignored; review-security: Non-atomic rename in writeSnapshot() |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Predictable temporary file path in writeSnapshot() |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-bugs: Metrics.addCreated() not synchronized - race condition; review-concurrency: Metrics.addCreated() not synchronized while addResolved() an |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-concurrency: Janitor thread iterates and mutates Store's internal map dir |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache() leaks ExecutorService - never shuts down; review-bugs: warmCache() returns before tasks complete |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() throws NPE when owner not in quotas |
| J56 | YES | the janitor thread is not a daemon and is never joined, so the JVM can | review-concurrency: stopJanitor() sets running=false but doesn't interrupt the t |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | YES | parallel: S06 the Authorization header may be absent, and bearerToken  | review-bugs: authenticate() NPE when Authorization header missing; review-security: NullPointerException in authenticate() on missing Authorizat |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin bypass via X-Admin header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats() division by zero when store empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() NPE when code parameter missing |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS misconfiguration: Allow-Origin * with Allow-Credentials |
| J67 | YES | offset and size are parsed with unguarded parseInt, so any non-numeric | review-security: NumberFormatException on invalid offset/size parameters |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/linkd/Authenticator.java:15 — Random instance shared across threads - contention under high load
- (low) src/linkd/Store.java:18 — Random instance shared across threads - contention under high load
- (critical) src/linkd/Authenticator.java:15 — Weak session token generation using java.util.Random
- (critical) src/linkd/Store.java:22 — SSRF via insufficient target URL validation
- (high) src/linkd/Main.java:179 — Path traversal in exportCsv via user-controlled name parameter
- (high) src/linkd/Main.java:180 — Path traversal in readReport via user-controlled name parameter
- (medium) src/linkd/Store.java:18 — Predictable short codes using java.util.Random
