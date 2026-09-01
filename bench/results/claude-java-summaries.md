# claude · java · run summaries (repeat 1)

recall **52/69** · 67 finding(s), 4 unmatched · 62202 tokens · 576s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() is overridden without hashCode(), breaking the docu |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() unboxes a null quota count, NPE on an owner's first |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() has the expiry test inverted: it deletes live link |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() clamps the end index to size()+1 and throws IndexOutO |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() truncates to 0 with integer division |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending and returns the least-followed links |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses contains() and returns other owners' links; review-security: byOwner matches owners by substring, leaking other owners' l |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode compares codes with == instead of equals |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links, which the Javadoc fo |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() returns the live internal map, not the documented s |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic despite its documented all-or-nothin |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() removes from the map it is iterating (ConcurrentModi |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode() can throw StringIndexOutOfBoundsException and does |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret silently used when LINKD_SECRET is |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens minted from java.util.Random are predictable |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to stdout and echoed in 401 responses |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-security: verify() never checks expiresAt, so sessions never expire |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares tokens with == instead of equals; review-security: verify() compares tokens with reference equality |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted single-round SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken NPEs on a missing Authorization header and throw; review-security: bearerToken NPEs or throws on a missing or malformed Authori |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin's condition is always true, so it denies every ; review-security: requireAdmin's condition is unsatisfiable, so the admin chec |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals does not do what its contract promises: non-con |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: getInstance() is an unsynchronized lazy singleton with an un |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek(), stats() and utilization() read shared state without  |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Eviction picks an arbitrary entry, not the least recently us |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() removes from the map it is iterating (ConcurrentModi; review-concurrency: sweep() removes entries while iterating its own keySet, thro |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() truncates to 0 with integer division |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A bad LINKD_TIMEOUT_MS sets the timeout to 0 while logging " |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is stored into a seconds field, and an un |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed and thrown away (parsed = parsed |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, and the caller ignores it an |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Public static SimpleDateFormat is not thread-safe |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in exportCsv/readReport gives arbitrary file  |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: FileWriters leak on an I/O failure |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV fields are concatenated without quoting or escaping; review-security: Reports containing every owner's data are created with defau |
| J46 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll builds filenames from unvalidated owner names |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot ignores renameTo's return value and renames ac; review-security: Snapshot staged through a fixed, world-writable /tmp path |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated() is unsynchronized while addResolved() a |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor loop mutates the store's map while iterating it, kil; review-concurrency: Janitor thread mutates Store's unsynchronized HashMap while  |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache returns before the probes finish and leaks the thr; review-concurrency: warmCache leaks eight non-daemon threads per call and return |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner unboxes a null Integer for an unknown owner |
| J56 | YES | the janitor thread is not a daemon and is never joined, so the JVM can | review-concurrency: stopJanitor cannot actually stop the janitor: no interrupt,  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-bugs: Unvalidated Integer.parseInt on query parameters throws out ; review-security: Link owner is taken from the query string instead of the ses |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links deletes any link regardless of owner |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out is an unrestricted open redirect |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: /admin/quotas grants admin on a client-supplied X-Admin head |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: /stats divides by store.size() without guarding zero |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() dereferences a missing ?code parameter |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS with credentials on every response, including  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report reads back a different filename than it wrote when ? |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) src/linkd/Main.java:191 — 204 and 302 responses are sent with a chunked body
- (medium) src/linkd/Store.java:28 — Quota increment is a non-atomic read-modify-write on a shared map
- (medium) src/linkd/Main.java:91 — Response JSON built by concatenation of unescaped user input
- (low) src/linkd/Main.java:182 — Export failures return the server path and raw exception to the client
