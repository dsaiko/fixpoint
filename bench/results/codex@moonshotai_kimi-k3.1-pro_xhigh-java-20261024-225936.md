# codex@moonshotai/kimi-k3.1-pro#xhigh · java · run 20261024-225936 (repeat 1)

recall **47/69** · 59 finding(s), 5 unmatched · 802704 tokens · 476s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Equality is inconsistent with hash-based collections |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: First link creation for every owner throws NullPointerExcept |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolution removes live links and returns expired targets; review-security: Resolving an active link deletes it |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Final listing page has an invalid end index |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success rate truncates to zero or one hundred |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deletion never releases the owner's quota |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard sorts least-followed links first |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner filtering returns substring matches |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: Code lookup uses reference equality |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extension revives already-expired links |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: Quota snapshot exposes the mutable internal map |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Batch import is not atomic |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: Pruning expired links throws during iteration |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: Generated codes can be shorter than the substring range; review-security: Short codes are predictable and have only 32 bits of entropy |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Request handlers and the janitor concurrently mutate an unpr |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens come from a predictable PRNG |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Live session tokens are written to logs |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Expired sessions are still accepted; review-security: Expired sessions remain valid |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Token verification uses reference equality; review-security: Bearer tokens are compared by object identity |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords use unsalted, single-pass SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Malformed or missing Authorization headers crash authenticat |
| J25 | — | parallel: S07 the admin check is always true, so requireAdmin rejects  |  |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Lazy singleton initialization can create multiple process-wi |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Several Cache methods access synchronized state without sync |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache eviction can discard the entry being updated |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Cache sweep mutates the key set during iteration; review-concurrency: Cache sweep removes directly from the map being iterated |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization divides by zero and truncates partial perc |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout replaces the default with zero |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond TTL is assigned directly to seconds |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: Fetch limit environment setting is discarded |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation always reports success |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared SimpleDateFormat is not thread-safe |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report names allow directory traversal and arbitrary file wr |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: Export writers leak when writing fails; review-concurrency: Concurrent reports with the same name can read partial or wr |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-security: Attacker-controlled fields permit CSV formula injection |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot move failures are silently ignored |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-concurrency: Concurrent snapshots overwrite the same temporary file |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Created-counter increments are not synchronized with snapsho |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes from the store while iterating its key set |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: Cache warming returns early and leaks its executor; review-concurrency: warmCache returns before completion and leaks its executor |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: Unknown owners cause a null-unboxing crash |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: Clients can impersonate another link owner |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete any link |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The /out endpoint is an open redirect |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A client-controlled header grants administrator access |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Statistics crash when the store is empty |
| J65 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Reports disclose every tenant's links |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Authenticator.java:62 — Admin authorization rejects every role
- (medium) src/linkd/Main.java:57 — Query parameters are never URL-decoded
- (high) src/linkd/Authenticator.java:49 — Session verification can race session issuance and revocation
- (medium) src/linkd/Worker.java:55 — Stopping the janitor neither wakes nor awaits the thread
- (medium) src/linkd/Export.java:41 — Sensitive reports are created with ambient filesystem permissions
