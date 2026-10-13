# codex@google/gemini-4-argon · java · run 20261013-224803 (repeat 1)

recall **53/69** · 64 finding(s), 5 unmatched · 1179247 tokens · 500s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Link overrides equals without overriding hashCode |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: NullPointerException when creating the first link for an own |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted expiration check in Store.resolve deletes valid lin |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Store.rename validates source code twice and never validates; review-security: Store.rename validates from twice and leaves target code to  |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Off-by-one clamping in Store.page throws IndexOutOfBoundsExc |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Integer division in Store.successRate truncates all partial  |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Store.delete does not decrement the owner's quota count |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Store.top sorts ascending instead of descending by hits |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Store.byOwner matches owner substrings instead of exact equa; review-security: Store.byOwner uses substring match instead of exact owner eq |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: Store.findByCode compares Strings with == instead of equals |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Store.extend revives already-expired links and resets expiry |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: Store.quotas exposes the mutable internal map instead of a s |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Store.importAll partially mutates the store when a later lin |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: ConcurrentModificationException in Store.prune when removing |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: StringIndexOutOfBoundsException in Store.newCode when random; review-security: Predictable short link codes generated with java.util.Random |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Unsynchronized concurrent access to Store maps and Link fiel |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Predictable session tokens generated with java.util.Random |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Plaintext session bearer tokens written to stdout logs |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Authenticator.verify never checks session expiration; review-security: Session verification ignores token expiration and uses refer |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Token verification compares Strings with == instead of equal |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Unsalted single-pass SHA-256 used for password hashing |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Uncaught NullPointerException and ArrayIndexOutOfBoundsExcep |
| J25 | — | parallel: S07 the admin check is always true, so requireAdmin rejects  |  |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: Non-constant-time secret comparison in secretEquals |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Unsynchronized HashMap and Random in Authenticator |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Unsynchronized lazy initialization race in Cache.getInstance |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Unsynchronized reads in peek, stats, and utilization and TOC |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache.set evicts an arbitrary HashMap entry instead of the l |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: ConcurrentModificationException in Cache.sweep when removing |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Integer division in Cache.utilization always returns 0 below |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Invalid LINKD_TIMEOUT_MS sets timeoutMillis to 0 instead of ; review-bugs: LINKD_CACHE_TTL_MS is assigned to cacheTtlSeconds without co |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed into a local variable and never  |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Config.validate always returns true even when validation pro |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared static SimpleDateFormat used concurrently without syn |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in CSV report export and readback |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: Unclosed FileWriter streams on IOException in Export methods |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-security: Multi-owner CSV report written with default world-readable p |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Export.writeSnapshot ignores File.renameTo failure and renam |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Predictable temporary file in /tmp enables symlink and TOCTO |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Data race on Metrics.created in unsynchronized addCreated() |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | YES | JAVA-ONLY the janitor swallows InterruptedException without restoring  | review-concurrency: Janitor thread lifecycle races and TOCTOU during expiration  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: ConcurrentModificationException in janitor thread when remov |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: Worker.warmCache leaks thread pool and returns before warmin; review-concurrency: warmCache leaks ExecutorService threads and returns before t |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: NullPointerException in Worker.countForOwner when owner has  |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Reflected bearer token in 401 response and unvalidated Autho |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: POST /links trusts client-supplied owner parameter instead o |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Missing ownership check in DELETE /links allows deleting any |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out via unvalidated next query parameter |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization bypass on /admin/quotas via client-suppl |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Division by zero in Main.stats when store is empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: NullPointerException on missing code parameter and broken up |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Overly permissive CORS configuration allowing wildcard origi |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exposes all owners' links to any authenticated user  |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Authenticator.java:62 — Always-true condition in requireAdmin rejects all sessions
- (high) src/linkd/Export.java:84 — Export.readReport does not default null or empty report name to "links"
- (high) src/linkd/Main.java:57 — Query parameter parser does not URL-decode keys or values
- (high) src/linkd/Authenticator.java:62 — Broken admin authorization condition in requireAdmin
- (medium) src/linkd/Main.java:91 — Unescaped target URL concatenated directly into JSON response
