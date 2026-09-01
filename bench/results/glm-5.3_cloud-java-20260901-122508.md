# glm-5.3:cloud · java · run 20260901-122508 (repeat 1)

recall **49/69** · 59 finding(s), 2 unmatched · 840865 tokens · 376s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() is overridden without hashCode(), breaking the docu |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() NPEs on the first link of any owner |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links are deleted,  |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the old code twice and never validates th |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() off-by-one makes subList throw whenever a page extend |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() integer-divides before scaling, so any fractio |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot despite its co |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, so the leaderboard lists the least-fo |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses contains() instead of equals() and NPEs on a  |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares code strings with == and can never fin |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links, contradicting its ow |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() hands out the live internal map, not a snapshot |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic, contradicting its all-or-nothing  |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() removes from links while iterating keySet() — Concur; review-concurrency: prune() removes via map.remove() while iterating keySet(), t |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode() truncates the hex to 8 chars and crashes on short  |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens minted from java.util.Random are predictable |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued session tokens written to stdout logs |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt and expired sessions are nev; review-security: Session expiry is documented but never enforced in verify |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares token strings with == so every session loo; review-security: verify compares token strings with ==, so authentication can |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with a single unsalted SHA-256 pass |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() crashes on a missing or one-part Authorization |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin's condition is inverted — it rejects every sess |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals is not constant-time and NPEs on null, contradi |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: getInstance() is an unsynchronized check-then-act on a non-v |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek(), warm(), stats() and utilization() bypass the lock th |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: set() evicts an arbitrary entry, not the least recently used |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() modifies the map while iterating its keySet; review-concurrency: sweep() removes entries via entries.remove() while iterating |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() integer-divides before scaling, so it reports  |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeoutMillis to 0 while claiming  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: TTL env var is read as milliseconds but stored into a second |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed into a self-assignment and never |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so it never refuses a bad co |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared static SimpleDateFormat is not thread-safe under conc |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in the /report endpoint: user-supplied name e |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: FileWriter handles leak when a write fails |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: exportCsv() does not escape CSV fields; review-security: CSV injection: owner field is written into reports unescaped |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot() ignores renameTo failure and uses a fixed te |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Snapshot written through a fixed, world-visible /tmp path (s |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated() is unsynchronized while the class claim |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes entries from keySet() while iterating, killi; review-concurrency: Janitor removes store entries through the live map while ite |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache() returns before the probes finish and leaks its t; review-concurrency: warmCache() leaks its ExecutorService and returns before any |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() NPEs when the owner is unknown |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links removes any link with no ownership check |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect (and header injection) via /out?next= |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization decided by a client-supplied X-Admin hea |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats handler divides by store.size() and crashes on an empt |
| J65 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: wildcard origin combined with Allow-Credent |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report handler writes links.csv but reads back null.csv when |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Store.java:191 — all() escapes the live unsynchronized HashMap, which the janitor thread mutates concurrent
- (low) src/linkd/Main.java:182 — Verbose 500 response leaks internal exception details and the export directory path
