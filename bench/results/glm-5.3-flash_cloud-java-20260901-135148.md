# glm-5.3-flash:cloud · java · run 20260901-135148 (repeat 1)

recall **44/69** · 57 finding(s), 4 unmatched · 598568 tokens · 290s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() is overridden without hashCode(), breaking Link as  |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() throws NullPointerException on the first link of an |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry condition is inverted: valid links are dele; review-security: Expiry check in resolve() is inverted: live links are delete |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() off-by-one throws IndexOutOfBoundsException instead o |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() does integer division before multiplying, so i |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot, contradicting |
| J08 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| J09 | — | parallel: C11 byOwner matches owners by substring, so one owner sees a |  |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares codes with == so it never matches |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links, contradicting its do |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() throws ConcurrentModificationException on the first  |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode() can produce codes shorter than the valid minimum a; review-security: Short codes are 32 bits of a predictable java.util.Random ou |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store's links/quota HashMaps are mutated by concurrent HTTP ; review-security: Store's plain HashMap is mutated concurrently by HTTP handle |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens minted from java.util.Random are predictable, |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session bearer tokens are written to stdout in plaintext |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-security: verify() never enforces the session expiry it documents |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares tokens with == so authentication always fa; review-security: verify() compares tokens with == so no token can ever authen |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() crashes on a missing or malformed Authorizatio; review-security: bearerToken() crashes on a missing or malformed Authorizatio |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin() always returns false due to inverted role log |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Cache.getInstance is an unsynchronized lazy singleton |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek() (and therefore warm()) reads the map without synchron |
| J31 | YES | parallel: CA07 stats reads the non-volatile hit counter with no synchr | review-concurrency: utilization() and stats() read shared state without synchron |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: set() evicts an arbitrary entry, not the least recently used |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() removes from entries while iterating the key set; review-concurrency: Cache.sweep() removes from the map while iterating its key s |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() integer division makes it report 0 until the c |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is parsed as milliseconds but stored as s |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded via a self-as |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so invalid configuration sti |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Static shared SimpleDateFormat is used from concurrent reque |
| J43 | — | parallel: EX01 the report name comes from the query string and is join |  |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-security: CSV export does not escape fields, enabling CSV/formula inje |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot() ignores the renameTo() result and hardcodes  |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: writeSnapshot uses a fixed, predictable path in world-writab |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated is not synchronized while the rest of the |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes from the store map while iterating it, killi; review-concurrency: Janitor removes from the live store map while iterating its  |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache() returns before warming and leaks its thread pool; review-concurrency: warmCache leaks its executor and returns before the probes f |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() unboxes a null Integer when the owner has no |
| J56 | YES | the janitor thread is not a daemon and is never joined, so the JVM can | review-concurrency: stopJanitor signals only a flag and never interrupts or join |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented bearer token back to the c |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links performs no ownership check — any user can del |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out forwards to any attacker-chosen ?next= U |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is a client-controlled X-Admin header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats() divides by zero when the store is empty; review-security: Unvalidated numeric inputs and empty-store state produce 500 |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() dereferences a null code when the query parameter |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Every response sends Access-Control-Allow-Origin: * together |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report without a name parameter reads back the wrong file a |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:86 — Non-numeric or negative ttl parameter throws uncaught NumberFormatException/IllegalArgumen
- (medium) src/linkd/Authenticator.java:49 — Authenticator's session map is an unsynchronized HashMap scanned by verify()
- (high) src/linkd/Main.java:179 — Path traversal in /report: ?name= escapes the export directory for both read and write
- (low) src/linkd/Export.java:41 — Export file containing every owner's links is created world-readable
