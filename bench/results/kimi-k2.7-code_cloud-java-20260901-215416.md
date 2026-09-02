# kimi-k2.7-code:cloud · java · run 20260901-215416 (repeat 1)

recall **37/69** · 47 finding(s), 2 unmatched · 197543 tokens · 535s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() is overridden but hashCode() is not |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: Quota lookup crashes for a new owner; review-bugs: create() throws NullPointerException on null target |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve path deletes live links and serves expired ones; review-concurrency: Mutable Link fields are mutated without synchronization |
| J04 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Pagination end index runs one past the list size; review-security: Store.page can throw IndexOutOfBoundsException on large or e |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success rate is always 0 unless every link is used |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not free the owner's quota |
| J08 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner matches owners by substring |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode compares codes by reference |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() can revive an already-expired link |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic |
| J14 | — | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i |  |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode can crash for short hex strings |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store uses unsynchronized HashMaps and Random across threads |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | — | parallel: S03 every issued session token is printed in cleartext |  |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() ignores session expiration |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Session token comparison uses reference equality; review-security: Token verification uses reference equality, rejecting every  |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken crashes on missing or malformed headers; review-security: Missing or malformed Authorization header causes an unhandle |
| J25 | — | parallel: S07 the admin check is always true, so requireAdmin rejects  |  |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Authenticator sessions map and Random are not thread-safe |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Cache singleton is published without synchronization |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Cache reads shared state without synchronization |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | — | parallel: CA08 eviction drops whatever key the iterator yields first,  |  |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | — | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C |  |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization is always 0 or 100 due to integer division |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout value is reset to zero |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL environment value is treated as seconds instead of |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT value is parsed but discarded |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared SimpleDateFormat is not thread-safe |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report export allows arbitrary .csv file w |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | — | parallel: EX06 renameTo's boolean result is discarded and it fails sil |  |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated increments a counter without synchronizat |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-concurrency: Janitor directly mutates Store's internal HashMap |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache creates an executor that is never shut down; review-concurrency: warmCache creates a thread pool and never shuts it down |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner crashes for owners not in the quota map |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: POST /links accepts arbitrary owner, enabling spoofing and q |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links lacks ownership check |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out via next parameter |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quota endpoint trusts a forgeable X-Admin header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats endpoint divides by zero when the store is empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect endpoint dereferences a null code |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS is configured to allow all origins with credentials |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report endpoint writes and reads different files when name i |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Main.java:28 — validate() return value is ignored
- (high) src/linkd/Store.java:191 — Store exposes internal mutable maps to callers
