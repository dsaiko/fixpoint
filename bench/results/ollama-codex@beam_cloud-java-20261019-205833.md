# ollama-codex@beam:cloud · java · run 20261019-205833 (repeat 2)

recall **51/69** · 71 finding(s), 10 unmatched · 1046454 tokens · 131s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals overridden without matching hashCode |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create NPEs on a new owner |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve treats unexpired links as expired |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates from twice and never validates to |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page end bound is size+1 causing IndexOutOfBoundsException |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate integer division rounds down to 0 |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not decrement owner quota |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top sorts ascending instead of descending |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner uses substring contains instead of exact match |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode uses == on Strings |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links contrary to intent |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas returns the live internal map; review-concurrency: quotas() returns the live quota map despite documenting a sn |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic and partially applies |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune throws ConcurrentModificationException; review-concurrency: prune modifies entries during iteration |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode can throw StringIndexOutOfBoundsException; review-security: Short codes generated with predictable Random |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store's links and quota maps are plain HashMaps used by conc |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to stdout |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify does not enforce session expiry |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify uses reference equality instead of value equality; review-security: Token verification uses reference equality |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted single-pass SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken crashes on null or malformed header |
| J25 | — | parallel: S07 the admin check is always true, so requireAdmin rejects  |  |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals claims constant-time but uses String.equals |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: sessions map is an unsynchronized HashMap accessed concurren |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | — | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre |  |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek reads the backing map without synchronization |
| J31 | YES | parallel: CA07 stats reads the non-volatile hit counter with no synchr | review-concurrency: stats reads hits without synchronization or volatility; review-concurrency: utilization reads entries.size() without synchronization |
| J32 | — | parallel: CA08 eviction drops whatever key the iterator yields first,  |  |
| J33 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: warm performs an unlocked check-then-set on the cache |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep throws ConcurrentModificationException; review-concurrency: sweep mutates entries during iteration |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization always reports 0 until full |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS parsed as seconds and unguarded; review-bugs: LINKD_CACHE_SIZE parse is not guarded |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is read but never applied |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared SimpleDateFormat is not thread-safe |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report name (export/read) |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: FileWriters are not closed on exceptions |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-security: Report files written with default, world-readable permission |
| J46 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: Path traversal via owner names in archiveAll |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot rename may silently fail across filesystems |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-concurrency: writeSnapshot uses a fixed temporary file path; review-security: Predictable temp file enables TOCTOU/symlink attack in write |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-bugs: Metrics.addCreated is not synchronized; review-concurrency: Metrics.addCreated is not synchronized while snapshot is |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: janitor mutates store map while iterating |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache neither awaits completion nor shuts down pool; review-concurrency: warmCache leaks a thread pool and never awaits completion |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner NPEs on owners without quota |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | YES | parallel: S06 the Authorization header may be absent, and bearerToken  | review-bugs: authenticate crashes on missing Authorization header |
| J59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented token |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via ?next= in /out |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated on client-supplied X-Admin header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats divides by zero when store is empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect NPEs when code query param is missing |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS with credentials |
| J67 | YES | offset and size are parsed with unguarded parseInt, so any non-numeric | review-bugs: listing endpoint crashes on bad numeric params |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Authenticator.java:62 — requireAdmin condition is always false
- (medium) src/linkd/Cache.java:57 — peek does not account for expiry
- (medium) src/linkd/Config.java:43 — bad LINKD_TIMEOUT_MS sets timeout to 0 instead of default
- (medium) src/linkd/Config.java:93 — loadFile unguarded parseInt crashes on bad file values
- (high) src/linkd/Cache.java:19 — Cache claimed thread-safe but peek, stats, utilization, warm, and getInstance are unsynchr
- (medium) src/linkd/Cache.java:23 — getInstance uses check-then-act initialization race
- (high) src/linkd/Store.java:191 — all() exposes the live internal links map
- (high) src/linkd/Authenticator.java:15 — Session tokens minted with predictable java.util.Random
- (medium) src/linkd/Main.java:118 — Redirect endpoint forwards stored targets without scheme/host checks
- (medium) src/linkd/Authenticator.java:102 — getSecret() exposes the service secret
