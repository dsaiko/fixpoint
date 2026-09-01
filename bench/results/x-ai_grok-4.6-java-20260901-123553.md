# x-ai/grok-4.6 · java · run 20260901-123553 (repeat 1)

recall **38/69** · 44 finding(s), 1 unmatched · 165296 tokens · 1054s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | — | JAVA-ONLY equals is overridden without hashCode, so links break as map |  |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: First create per owner NPEs on quota increment; review-bugs: create() NPEs when target is missing |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() treats live links as expired and serves dead ones; review-security: Expiry check is inverted so dead links still redirect |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() last-page end index is size+1 and throws |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() integer-divides to 0 or 100 |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not return the owner's quota slot |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-hit links, not the most-hit |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches substrings, not the owner |
| J10 | — | JAVA-ONLY findByCode compares strings with == instead of equals, so it |  |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() removes from the map while iterating keySet |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-security: Short codes are short and generated with Random |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store HashMap is shared with the janitor unsynchronized |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens come from java.util.Random; review-security: Session tokens are written to stdout |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never enforces expiresAt; review-security: verify() never enforces session expiry |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares session tokens with == |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() crashes on missing or malformed Authorization |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin() rejects every role, including admin |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | — | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre |  |
| J30 | — | parallel: CA02 peek reads the map with no synchronization while synchr |  |
| J31 | YES | parallel: CA07 stats reads the non-volatile hit counter with no synchr | review-concurrency: Cache reads HashMap and hits without the lock get/set/sweep  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: set() evicts an arbitrary HashMap key, not LRU |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() mutates the map while iterating keySet |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() integer-divides to 0 until the cache is full |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is stored as seconds |
| J39 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s |  |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() logs problems but always returns true |
| J42 | — | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  |  |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: User-controlled report name is a path traversal |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot ignores renameTo failure and writes tmp outsid |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated increments created without synchronizatio |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes from the live key set and throws ConcurrentM |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache returns before work finishes and leaks the pool; review-concurrency: warmCache leaks its pool and does not wait for tasks |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner unboxes a null quota |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: Create takes owner from the query string |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links ignores the authenticated owner |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin API trusts a client-supplied X-Admin header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats divides by store.size() and crashes on an empty store |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect NPEs when code is omitted |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint dumps every owner's links |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report writes links.csv but reads name.csv when name is omit |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Store.java:129 — findByCode() compares codes with ==
