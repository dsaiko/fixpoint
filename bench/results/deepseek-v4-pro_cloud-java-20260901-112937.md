# deepseek-v4-pro:cloud · java · run 20260901-112937 (repeat 1)

recall **42/69** · 53 finding(s), 3 unmatched · 452836 tokens · 307s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() overridden without hashCode() |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() NPEs on the first link for any owner; review-bugs: create() NPEs on a null target |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted, deleting valid links |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() sets end to size+1, so subList throws |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() integer division always reports 0 |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not return the owner's quota slot |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least-followed links |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring match instead of exact owner match |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares codes with == and never matches |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, contradicting its contract |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic despite its contract |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() throws ConcurrentModificationException |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode() can throw StringIndexOutOfBoundsException; review-security: Short link codes use java.util.Random and are predictable |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store's links and quota HashMaps are mutated by concurrent r |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens use java.util.Random and are predictable |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are written to stdout |
| J21 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares tokens with == so auth always fails; review-security: Token verification compares strings with == |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() NPEs on a missing header and AIOOBEs on a malf; review-security: bearerToken throws on missing/malformed Authorization header |
| J25 | — | parallel: S07 the admin check is always true, so requireAdmin rejects  |  |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Authenticator's sessions HashMap is read and mutated concurr |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Cache.getInstance lazy init is an unsynchronized check-then- |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Cache.peek/stats/utilization read shared state without the l |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: set() evicts an arbitrary entry, not the least-recently-used |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() throws ConcurrentModificationException; review-concurrency: Cache.sweep removes entries while iterating the keySet, thro |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() integer division always reports 0 |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: bad LINKD_TIMEOUT_MS sets 0 while claiming to keep the defau |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: load() lets a bad LINKD_CACHE_TTL_MS / LINKD_CACHE_SIZE cras; review-bugs: LINKD_CACHE_TTL_MS is stored in a seconds field and used as  |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true and never refuses a bad confi |
| J42 | — | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  |  |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report export/read allows path traversal via name |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-security: CSV formula injection via unvalidated owner field |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | — | parallel: EX06 renameTo's boolean result is discarded and it fails sil |  |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated is not synchronized, racing with snapshot |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: startJanitor() removes from the map while iterating it; review-concurrency: Janitor iterates the live links map and removes from it, thr |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache() leaks a thread pool and returns before probes fi; review-concurrency: warmCache leaks a fixed thread pool and returns before its t |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() NPEs for an owner with no links |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links does not check link ownership |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out via ?next= |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated by client-supplied X-Admin header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats() divides by zero when the store is empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() NPEs when ?code= is missing |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: * with credentials on authenticated endpoin |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() reads a different filename than it wrote when name  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:86 — links() parses ttl outside the try, so a bad ttl is an unhandled 500
- (medium) src/linkd/Authenticator.java:62 — requireAdmin() is always false due to || instead of &&
- (high) src/linkd/Store.java:45 — link.hits++ is a non-atomic read-modify-write on a shared field
