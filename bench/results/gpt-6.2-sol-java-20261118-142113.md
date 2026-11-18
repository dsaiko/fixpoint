# gpt-6.2-sol · java · run 20261118-142113 (repeat 1)

recall **49/69** · 83 finding(s), 23 unmatched · 380136 tokens · 814s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Equal links can behave as different keys in hash collections |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: Creating a first link throws after inserting it; review-bugs: Target validation throws for an omitted URL and accepts URLs |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it; review-security: Expired short links still disclose their destinations |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and skips the destination |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: A page that extends past the last link throws; review-bugs: Negative or overflowing pagination arguments throw |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: The success rate is zero until every link has a hit |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Removing links leaves owner counts inflated |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard returns the least-followed links |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup matches substrings |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: Direct code lookup misses equal strings |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can revive it or shorten its lifetime |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: Quota snapshots allow callers to change live counts |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-concurrency: Concurrent readers can observe a partially imported batch |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: Pruning can throw after only one removal |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: Code generation can throw when the hex string has fewer than |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens come from a predictable random generator |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session issuance logs replayable bearer tokens |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Session expiry is ignored and expired sessions accumulate; review-security: Session verification ignores the expiration timestamp |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Verification rejects valid tokens parsed from requests |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Missing or malformed Authorization headers abort requests |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: The admin helper rejects every role |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | — | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre |  |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Cache reads bypass the lock used by writers |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache eviction ignores access order |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Cache sweep can throw while removing expired entries |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization is wrong below capacity and crashes at zer |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout values replace the default with zero |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: The cache TTL treats milliseconds as seconds |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: The configured fetch limit is discarded |
| J40 | YES | parallel: CF05 loadFile swallows every IOException, so an unreadable c | review-bugs: Configuration file read failures silently discard overrides |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Startup validation accepts configurations it flags as invali |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Concurrent exports can mix report timestamp fields |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report names can escape the export directory and overwrite f |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: Export writers remain open when a write throws; review-concurrency: Concurrent CSV writers can leave a mixed report |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV exports break when fields contain delimiters; review-security: Untrusted link fields can become spreadsheet formulas |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot writes can report success without publishing a dura |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-concurrency: Concurrent snapshots share the same temporary file |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Concurrent creation metrics lose increments |
| J51 | YES | the errors counter is exposed on /stats and never incremented anywhere | review-bugs: The dashboard error counter always stays at zero |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Removing an expired link can terminate the janitor; review-concurrency: A concurrent store deletion can kill the janitor |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: Warming the cache leaks executor threads; review-concurrency: Cache warming returns before completion and leaks worker thr |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: Counting links for an unknown owner throws |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | YES | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin | review-concurrency: A failed HTTP bind leaves the JVM running; review-security: An unauthenticated connection can stall the HTTP dispatcher |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Authentication failures copy bearer credentials into respons |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Link creation and deletion ignore authenticated ownership |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Cached outbound redirects bypass click counting; review-security: The email redirect endpoint accepts arbitrary external desti |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: The admin endpoint trusts a caller-supplied role header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Stats divide by zero when the store is empty; review-concurrency: Pruning the last link can make dashboard stats divide by zer |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect lookup fails for missing and mixed-case codes |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: An unnamed report is read from the wrong file |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) src/linkd/Store.java:27 — A generated code collision overwrites an existing link
- (high) src/linkd/Main.java:38 — The server has no way to issue a session
- (high) src/linkd/Main.java:35 — An HTTP bind failure leaves the process running
- (high) src/linkd/Main.java:57 — Query parameters keep their percent encoding
- (medium) src/linkd/Store.java:163 — An invalid link leaves earlier batch entries committed
- (medium) src/linkd/Store.java:164 — Imported links are missing from owner counts
- (medium) src/linkd/Main.java:86 — Malformed numeric parameters bypass the bad-request response
- (medium) src/linkd/Main.java:179 — Reports silently omit links after the first 1,000
- (medium) src/linkd/Main.java:188 — CSV and plain-text responses are mislabeled as JSON
- (medium) src/linkd/Export.java:41 — Export encoding can discard Unicode characters
- (medium) src/linkd/Cache.java:58 — Expired cache entries prevent warming from refreshing them
- (medium) src/linkd/Worker.java:77 — The target checker reports success for unreachable HTTPS targets
- (medium) src/linkd/Worker.java:55 — Stopping the janitor leaves a sweep pending
- (medium) src/linkd/Authenticator.java:49 — Issuing or revoking a session can abort token verification
- (medium) src/linkd/Cache.java:32 — Concurrent initialization can return two process-wide caches
- (medium) src/linkd/Cache.java:90 — Warming can overwrite a newer cache entry
- (medium) src/linkd/Store.java:45 — Concurrent resolutions lose link hits
- (medium) src/linkd/Store.java:53 — Concurrent renames can overwrite a link
- (medium) src/linkd/Export.java:78 — Concurrent audit appends can split a log record
- (high) src/linkd/Authenticator.java:50 — The HTTP service cannot accept a valid bearer credential
- (high) src/linkd/Main.java:37 — The service accepts bearer credentials over plaintext HTTP
- (medium) src/linkd/Main.java:179 — A user report can disclose other owners' links
- (medium) src/linkd/Export.java:41 — Reports can be readable by other local users
