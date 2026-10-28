# ollama-direct@glm-5.3-flash:cloud#max · java · run 20261028-131535 (repeat 1)

recall **42/69** · 61 finding(s), 12 unmatched · 1047294 tokens · 673s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() is overridden without hashCode() |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: New owners trigger NullPointerException and partially create |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiry condition is inverted |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() sets an out-of-bounds end index |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() performs integer division before scaling |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link never restores its quota |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard sorted ascending instead of most hits first |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() performs substring matching instead of exact owner |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares codes by reference |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already expired links |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is advertised as atomic but is not |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() mutates map while iterating and throws; review-concurrency: prune removes entries while iterating the same map |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: Generated code can be shorter than eight hex digits and cras |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store maps are shared without synchronization |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are predictable and logged |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Expired sessions are never rejected; review-security: Expired sessions are still accepted |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Session tokens are compared by reference, not value; review-security: Bearer token verification uses reference equality |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Missing or malformed Authorization header crashes request ha |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: Admin authorization rejects every role |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Session map is not safe for concurrent access |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Cache singleton initialization has a check-then-act race |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek, stats, and utilization bypass Cache synchronization |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache eviction does not select the least recently used item |
| J33 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: warm uses a non-atomic peek-then-set race |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() mutates map while iterating and throws; review-concurrency: sweep removes map entries during iteration |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization is truncated to zero and divides by zero |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout is set to zero despite a log saying the defa |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL is interpreted as seconds when configured in milli; review-bugs: Several environment values are parsed without error handling |
| J39 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s |  |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always accepts invalid configuration |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared SimpleDateFormat is not thread-safe |
| J43 | — | parallel: EX01 the report name comes from the query string and is join |  |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: File writers leak if writing fails |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot rename result is ignored |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.created is incremented without synchronization |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor mutates store map while iterating and dies; review-concurrency: Janitor mutates a live HashMap during iteration and dies on  |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache leaks worker threads and returns before work compl; review-concurrency: warmCache submits work without waiting or shutting down the  |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() unboxes a nullable Integer |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-bugs: POST /links accepts null request fields and crashes |
| J61 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| J62 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| J63 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Stats divide by zero when the store is empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect request without code throws |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | YES | offset and size are parsed with unguarded parseInt, so any non-numeric | review-bugs: Listing pagination accepts malformed offset/size and crashes |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default CSV export name is not used consistently |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Main.java:57 — Query parameters are not URL-decoded
- (high) src/linkd/Main.java:86 — Malformed TTL crashes link creation
- (high) src/linkd/Store.java:191 — Internal mutable maps escape Store
- (high) src/linkd/Main.java:619 — Admin authorization trusts a client-supplied header
- (high) src/linkd/Main.java:550 — Link creation uses caller-controlled owner
- (high) src/linkd/Main.java:566 — Any session can delete anyone's link
- (high) src/linkd/Main.java:647 — Report endpoint exports all users' links for any session
- (high) src/linkd/Export.java:369 — User-controlled report name permits path traversal
- (high) src/linkd/Store.java:705 — Expiry check is inverted, deleting live links and serving expired ones
- (high) src/linkd/Store.java:873 — Predictable short codes can collide and overwrite links
- (medium) src/linkd/Export.java:383 — Sensitive multi-owner CSV is not forced to owner-only permissions
- (medium) src/linkd/Export.java:374 — CSV fields are not encoded
