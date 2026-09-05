# claude-sonnet-5 · java · run 20260905-155340 (repeat 1)

recall **44/69** · 48 finding(s), 0 unmatched · 63639 tokens · 593s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() overridden without hashCode(), breaking the documen |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() throws NPE for any owner's first link; review-bugs: create() dereferences a null target with no validation |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() condition is inverted: live links are deleted, exp |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice and never validates 'to' |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() off-by-one causes IndexOutOfBoundsException |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() truncates to 0 due to integer division order |
| J07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least-followed links in |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring containment instead of exact match |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares codes with == instead of equals() |
| J11 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() returns the live map despite documenting it as a sn |
| J13 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() throws ConcurrentModificationException |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-security: Link codes generated with insecure Random and narrow keyspac |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store's links/quota HashMaps are unsynchronized but shared b |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens minted with java.util.Random are predictable |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to stdout on issuance |
| J21 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares tokens with == instead of equals() |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Unsalted single-round SHA-256 for password storage |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken throws NPE/AIOOBE on missing or malformed Author |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin's condition is always false |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals is not constant-time despite its documented gua |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | — | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre |  |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek() reads the cache map without the lock the other mutato |
| J31 | YES | parallel: CA07 stats reads the non-volatile hit counter with no synchr | review-concurrency: utilization() reads entries.size() outside the lock used by  |
| J32 | — | parallel: CA08 eviction drops whatever key the iterator yields first,  |  |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() throws ConcurrentModificationException; review-concurrency: Cache.sweep() mutates entries while iterating its keySet, gu |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() truncates to 0 due to integer division order |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: cache TTL milliseconds stored directly into a seconds field |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when problems were found |
| J42 | — | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  |  |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal via unsanitized report name |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot silently drops the snapshot if renameTo fails |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Snapshot written to a fixed, predictable path in world-writa |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated() is unsynchronized while sibling methods |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: janitor loop throws ConcurrentModificationException; review-concurrency: Janitor thread mutates Store's map while iterating its keySe |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache returns without waiting for probes to finish and l; review-concurrency: warmCache submits work to a thread pool it never shuts down  |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner unboxes a null Integer into NPE for owners wit |
| J56 | YES | the janitor thread is not a daemon and is never joined, so the JVM can | review-concurrency: stopJanitor only flips a flag; it never interrupts the sleep |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-bugs: malformed ttl query parameter crashes the handler uncaught |
| J61 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by a client-supplied header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats() divides by zero when the store is empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() throws NPE when the code parameter is missing |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS combined with allow-credentials on every respo |
| J67 | YES | offset and size are parsed with unguarded parseInt, so any non-numeric | review-bugs: malformed offset/size query parameters crash the listing han |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report leaks every owner's links to any authenticated calle |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() reads back the wrong file when name is omitted or e |
