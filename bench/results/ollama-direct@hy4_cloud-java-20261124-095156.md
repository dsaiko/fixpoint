# ollama-direct@hy4:cloud · java · run 20261124-095156 (repeat 1)

recall **56/69** · 92 finding(s), 16 unmatched · 850172 tokens · 1330s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals is overridden without hashCode |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: First link creation always fails with a null quota entry; review-bugs: create dereferences nullable target and owner inputs |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Live links are deleted while expired links are served; review-security: The link expiry comparison is reversed |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source code twice but never the destina |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Paging past the end of the list uses an out-of-range index |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate uses integer division for a percentage |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting or pruning links does not return quota slots |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard sorts ascending and does not clamp negative n |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner returns links whose owner merely contains the reques; review-security: byOwner() matches substrings and leaks other tenants' links |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode compares code strings by identity |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend replaces the remaining lifetime and revives expired l |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: Store exposes its live internal maps despite snapshot and en |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Batch import is not atomic; review-concurrency: importAll publishes a partial batch instead of an atomic bat |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune removes map entries during keySet iteration; review-concurrency: prune removes through the map while iterating its keySet |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: Code generation can throw and does not avoid collisions; review-security: Short codes are predictable and only 32 bits long |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Known fallback secret is used when LINKD_SECRET is unset |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens are generated with a predictable PRNG |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Live bearer tokens are printed to stdout |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Session expiry is never enforced; review-concurrency: Session map is iterated and mutated without synchronization |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Session verification compares token strings by identity |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashes use one unsalted SHA-256 pass |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Bearer token parsing crashes on missing or malformed headers; review-security: Bearer header parsing has no null, format or scheme checks |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin rejects every role |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: Secret comparison is not constant-time |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Process-wide Cache singleton can be initialized more than on |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Unsynchronized Cache reads violate the all-methods-thread-sa |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache eviction is arbitrary rather than least-recently-used |
| J33 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: warm has a check-then-act race that can overwrite a fresh en |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Cache sweep removes entries through the map during keySet it; review-concurrency: Cache.sweep invalidates its own iterator and kills the janit |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Utilization is calculated with integer division before scali |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Invalid timeout configuration sets the timeout to zero; review-bugs: Millisecond TTL environment value is used as seconds |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always reports success |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-bugs: Report timestamps have no fixed time zone; review-concurrency: Shared SimpleDateFormat is used concurrently by report write |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: User-controlled report name permits path traversal and arbit |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: File writers are not closed when a write fails |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV fields are written without quoting or escaping; review-concurrency: Concurrent reports with the same name truncate and read one  |
| J46 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll() lets owner names escape the destination directo |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot rename failure is silently ignored |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-concurrency: writeSnapshot races on a single global temporary filename; review-security: Predictable /tmp snapshot path is vulnerable to symlink and  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated is not synchronized despite the class con |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes store entries while iterating the live keySe; review-concurrency: Janitor removes from the Store map while iterating its keySe |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache leaks executors and returns before its work is don; review-concurrency: warmCache leaks every executor and returns before its tasks  |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner unboxes a missing quota as null |
| J56 | YES | the janitor thread is not a daemon and is never joined, so the JVM can | review-concurrency: stopJanitor cannot cancel or join the sleeping janitor |
| J57 | YES | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin | review-concurrency: HTTP server has no request executor or configured deadline |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-bugs: Unvalidated ttl query parameter escapes the 400 handler; review-security: Link creation trusts a client-supplied owner |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete another owner's link |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out is an unrestricted open redirect |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts the client-controlled X-Admin hea |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Stats handler divides by zero on an empty store; review-concurrency: stats composes separate Store reads into an inconsistent sna |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect dereferences a missing code and lowercases with the |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: All responses use a wildcard CORS policy with credentials en |
| J67 | YES | offset and size are parsed with unguarded parseInt, so any non-numeric | review-bugs: Offset and size query parameters are not validated |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report discloses every owner's links to any authenticated u |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report endpoint reads a different filename than it wrote |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Main.java:57 — Query values are used without URL decoding
- (medium) src/linkd/Main.java:28 — Startup ignores the configuration validation result
- (medium) src/linkd/Authenticator.java:77 — Password hashing depends on the JVM default charset
- (medium) src/linkd/Cache.java:58 — peek treats expired entries as present
- (medium) src/linkd/Config.java:108 — Validation accepts a zero cache size
- (medium) src/linkd/Export.java:85 — Report round-trip uses an unspecified platform charset
- (medium) src/linkd/Main.java:91 — Create response builds JSON without escaping the target
- (medium) src/linkd/Main.java:156 — Admin quotas endpoint reports the number of owners, not quotas
- (low) src/linkd/Main.java:192 — Response stream can leak when writing fails
- (low) src/linkd/Store.java:196 — validateCode throws NPE for a null code
- (critical) src/linkd/Store.java:191 — Live unsynchronized HashMaps are shared with the janitor and handlers
- (medium) src/linkd/Store.java:45 — Hit counter increments and reads are unsynchronized
- (medium) src/linkd/Store.java:53 — rename can lose a link when two callers target the same code
- (medium) src/linkd/Main.java:182 — Export failures disclose internal paths and filesystem errors
- (high) src/linkd/Export.java:84 — readReport() can read CSV files outside the export directory
- (high) src/linkd/Export.java:41 — The all-owner report is created with default file permissions
