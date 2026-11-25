# ollama-direct@mimo-v2.6-flash:cloud · java · run 20261125-141910 (repeat 1)

recall **43/69** · 51 finding(s), 2 unmatched · 294382 tokens · 311s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Link defines equals without a matching hashCode |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: First link creation always throws an unboxing NullPointerExc; review-security: Link creation can exhaust memory without quota enforcement |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve deletes live links and serves expired ones; review-security: Expiry check is reversed, so expired links still resolve |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source code twice |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Final page clamps past the list end |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success rate rounds down to zero before applying percent |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Quota counts diverge from stored links |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard sorts from fewest to most hits |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner matches owner substrings instead of exact owners |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode uses reference equality for codes |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic on validation failure |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune can throw ConcurrentModificationException |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | — | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr |  |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store state is mutated without synchronization against the j |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer tokens are disclosed in logs and use non-cryptographi |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify rejects issued tokens and ignores session expiry; review-security: Bearer verification rejects valid tokens using reference equ |
| J22 | — | JAVA-ONLY verify compares the token with == rather than equals, so aut |  |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | — | parallel: S06 bearerToken indexes parts[1] without checking, so a malf |  |
| J25 | — | parallel: S07 the admin check is always true, so requireAdmin rejects  |  |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Session issuance, verification, and revocation race on a Has |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Cache's advertised thread safety omits singleton and statist |
| J30 | — | parallel: CA02 peek reads the map with no synchronization while synchr |  |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Full-cache eviction is not least recently used |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Cache sweep can terminate the janitor with ConcurrentModific |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization is almost always wrong and can divide by z |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond cache TTL is stored as seconds |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Validation reports problems but always accepts the configura |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared SimpleDateFormat can corrupt concurrent export timest |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report name permits path traversal outside the export direct |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: File writers leak when writing fails |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV fields are not escaped; review-concurrency: Concurrent reports using the same name overwrite and read pa |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot move failure is silently ignored |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.created uses an unsynchronized read-modify-write |
| J51 | YES | the errors counter is exposed on /stats and never incremented anywhere | review-bugs: Metrics error count can never increase |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor can terminate while removing expired links |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache returns early and leaks its thread pool; review-concurrency: warmCache returns early and leaks a thread pool on every cal |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: Unknown owner count throws during unboxing |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | YES | parallel: S06 the Authorization header may be absent, and bearerToken  | review-bugs: Missing or malformed Authorization headers crash handlers |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-bugs: Missing or invalid POST fields escape as runtime exceptions |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Authenticated users can delete any owner's link |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Outbound endpoint accepts arbitrary redirect destinations |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is controlled by a spoofable client head |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Stats crashes on an empty store |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect without a code throws a NullPointerException |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exposes every owner's links to any authentic |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report name writes one file and reads another |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:57 — Query values are used without URL decoding
- (medium) src/linkd/Worker.java:74 — checkTargets does not probe target availability
