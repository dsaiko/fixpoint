# deepseek-v4.1-flash:cloud · java · run 20260913-222726 (repeat 1)

recall **49/69** · 59 finding(s), 3 unmatched · 142370 tokens · 374s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() is overridden without hashCode(), breaking the docu |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() NPEs for any owner seen for the first time |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() has an inverted expiry test: it deletes live links |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the wrong argument twice, leaving the des |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() computes an out-of-range end index and throws IndexOu |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() uses integer division and always reports 0 |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never gives the owner's quota slot back |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least-followed links as |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses String.contains instead of equals, mixing own |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares codes by reference and never matches |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, contradicting its own contra |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() returns the live map despite promising a snapshot |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic despite its contract |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() removes from the map while iterating its key set |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode() can throw StringIndexOutOfBoundsException and can  |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store is unsynchronized shared mutable state, mutated by the |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret: the service starts with a publicl |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens and short codes are minted with java.util.Ran |
| J20 | — | parallel: S03 every issued session token is printed in cleartext |  |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() compares tokens with == so no session ever verifies; review-concurrency: Session map is iterated and mutated from concurrent requests |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-security: verify() compares tokens by reference, so the session lookup |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing is a single unsalted SHA-256 round |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() throws on a missing or malformed Authorization; review-security: bearerToken throws on a missing or malformed Authorization h |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin() uses || so it always returns false |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: getInstance is an unsynchronized lazy initializer on a non-v |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Cache.peek reads the HashMap without the monitor while get/s |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: set() evicts an arbitrary entry, not the least recently used |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() removes entries while iterating, throwing Concurrent; review-concurrency: sweep() removes from the map while iterating its keySet |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() uses integer division and can divide by zero |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A bad LINKD_TIMEOUT_MS sets the timeout to 0 despite claimin |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is parsed as milliseconds into a seconds  |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so an unusable configuration |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: One shared static SimpleDateFormat is used to stamp reports  |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal: the report name is concatenated onto the exp |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-security: Export files are created world-readable, contradicting the s; review-security: CSV formula injection: unvalidated owner/target values are w |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot ignores renameTo's result and never flushes, s |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Snapshot written through a predictable filename in /tmp, all |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated writes a field that snapshot() reads unde |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes from the map while iterating its key set, ki; review-concurrency: Janitor iterates the live store map and dereferences a key a |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache submits work and returns before any of it runs, le; review-concurrency: warmCache leaks an ExecutorService on every call and returns |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner unboxes a null Integer for an unknown owner |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: The presented authentication token is reflected in the 401 b |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-bugs: POST /links throws instead of returning 400 for a missing ta; review-security: Link ownership is taken from the request instead of the auth |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links deletes any link: no ownership check despite t |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect at /out: ?next= is echoed into Location with n |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is decided by a client-supplied X-Admin  |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: /stats divides by zero when the store is empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() dereferences a null code when the parameter is ab |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Every response carries Access-Control-Allow-Origin: * togeth |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Export.java:85 — Report written under a defaulted name is read back under a null name
- (medium) src/linkd/Store.java:191 — all() and quotas() hand out the live internal maps, defeating the class's own ownership co
- (low) src/linkd/Main.java:182 — Export failures return the filesystem path and the raw exception to the client
