# qwen3.5:397b-cloud · java · run 20260902-110845 (repeat 1)

recall **31/69** · 43 finding(s), 4 unmatched · 334403 tokens · 166s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | — | JAVA-ONLY equals is overridden without hashCode, so links break as map |  |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: NullPointerException when creating first link for an owner; review-security: NullPointerException when creating link for new owner |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() removes non-expired links and returns expired ones |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice, never validates 'to' |
| J05 | — | parallel: C04 page clamps end past the list size, so the last page thr |  |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() returns 0 due to integer division |
| J07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| J08 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses contains() instead of equals(); review-security: Owner comparison uses contains instead of equals |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() uses reference comparison instead of value comp |
| J11 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: ConcurrentModificationException in prune() |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | — | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr |  |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: HashMap accessed concurrently without synchronization |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret enables authentication bypass |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Predictable token generation using java.util.Random |
| J20 | — | parallel: S03 every issued session token is printed in cleartext |  |
| J21 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: String comparison with == instead of equals(); review-concurrency: String comparison uses == instead of equals |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: ArrayIndexOutOfBoundsException in bearerToken(); review-security: Bearer token extraction crashes on malformed Authorization h |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: Logic error in requireAdmin always returns false; review-security: Admin check logic always returns false |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals is not timing-safe despite documentation claim |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Sessions HashMap and Random accessed without synchronization |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Broken double-checked locking in singleton getInstance |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Unsynchronized peek() races with synchronized set() and swee |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | — | parallel: CA08 eviction drops whatever key the iterator yields first,  |  |
| J33 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: warm() method has TOCTOU race between peek() and set() |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: ConcurrentModificationException in sweep() |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() returns 0 due to integer division |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT environment variable ignored |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when problems found |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: SimpleDateFormat is not thread-safe but used as static share |
| J43 | — | parallel: EX01 the report name comes from the query string and is join |  |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: FileWriter resource leak in exportCsv() |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: renameTo() failure silently ignored in writeSnapshot() |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Temporary snapshot file written to world-readable /tmp |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | — | parallel: X01 addCreated is unsynchronized while addResolved is, so co |  |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: ConcurrentModificationException in janitor thread; review-concurrency: Janitor thread iterates store.all() while handlers mutate th |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: ExecutorService resource leak in warmCache(); review-concurrency: Thread pool leak in warmCache - executor never shut down |
| J55 | — | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null |  |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Token value leaked in error response |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| J63 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| J64 | — | parallel: C20 /stats divides by the link count, throwing ArithmeticExc |  |
| J65 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Insecure CORS configuration leaks credentials to any origin |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Config.java:43 — Invalid timeout value on parse error causes validation failure
- (medium) src/linkd/Cache.java:58 — peek() returns true for expired entries
- (medium) src/linkd/Main.java:86 — TTL parsing error: hours vs days mismatch
- (high) src/linkd/Export.java:83 — Path traversal in readReport allows reading arbitrary files
