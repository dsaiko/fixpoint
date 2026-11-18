# gpt-6.2-astra · java · run 20261118-145853 (repeat 1)

recall **47/69** · 90 finding(s), 29 unmatched · 185937 tokens · 414s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Equal links cannot be reliably looked up in hash collections |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: Creating the first link for an owner fails after inserting i; review-bugs: Target validation crashes on a missing target and accepts ho |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link permanently deletes it; review-security: Resolving a link permits expired destinations and destroys l |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and accepts an invalid des |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: A partial last page throws instead of returning the remainin |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: The dashboard reports zero success until every link has been |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Owner counts drift when links are removed or imported |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard selects the least-followed links first |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Filtering by owner matches partial names; review-security: Owner-scoped lookup returns records belonging to other ident |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode misses equal codes held in different String objec |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can revive an expired code or shorten a liv |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: The advertised quota snapshot allows callers to overwrite st |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: A failed batch import leaves earlier writes committed; review-concurrency: Readers can observe a partially imported batch |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: Pruning can fail after removing only part of the expired set |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: Short hexadecimal random values crash code generation |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens come from a predictable random generator |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session issuance writes the complete bearer token to stdout |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Expired sessions accumulate indefinitely and remain eligible; review-security: Session verification never enforces the twelve-hour expiry |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Valid token text from an HTTP header fails verification |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: An absent or incomplete Authorization header aborts the requ |
| J25 | — | parallel: S07 the admin check is always true, so requireAdmin rejects  |  |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | — | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre |  |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Cache readers bypass the lock used by writers |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache eviction ignores recency and can discard an entry duri |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Sweeping expired cache entries can abort the janitor |
| J35 | — | parallel: C05 utilization divides before scaling so it is always 0, an |  |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL overrides last 1,000 times too long; review-bugs: Invalid timeout input replaces the working default with zero |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: The fetch-limit environment override has no effect |
| J40 | YES | parallel: CF05 loadFile swallows every IOException, so an unreadable c | review-bugs: An unreadable configuration file silently falls back to defa |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Startup proceeds with configurations that validation rejects |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Concurrent exports race on the shared timestamp formatter |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: A report name can direct a truncating write outside the expo |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: Failed report, archive, and audit writes can leak file descr; review-concurrency: Concurrent reports with the same name can return another cal |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV exports change field boundaries for ordinary URL and own; review-security: User-controlled CSV cells can be interpreted as spreadsheet  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot publication can fail silently |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-concurrency: Concurrent snapshots can publish another caller's payload; review-security: Snapshot writes trust a predictable path in shared temporary |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Concurrent creation counters lose increments |
| J51 | YES | the errors counter is exposed on /stats and never incremented anywhere | review-bugs: The dashboard error counter always reports zero |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Removing an expired link can terminate the janitor permanent |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: Cache warming returns before completion and leaks its worker; review-concurrency: Each nonempty cache warm-up leaks executor threads |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: Counting links for an unknown owner throws |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | YES | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin | review-concurrency: A failed port bind leaves the process running without a serv; review-security: A slow client can block the HTTP server before authenticatio |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Authentication failures copy the bearer credential into the  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: Link creation trusts the owner supplied by the requester |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Deleting a link requires its code but never checks its owner |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Caching the permanent outbound redirect skips later click co; review-security: The outbound redirect accepts an arbitrary destination |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: The caller supplies its own administrator assertion |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: The stats endpoint divides by zero on an empty store; review-concurrency: The janitor can change the denominator after stats counts hi |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect lookup dereferences a missing code and changes vali |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Requesting the default report reads a different file from th |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) src/linkd/Store.java:27 — A generated code collision overwrites an existing link
- (high) src/linkd/Main.java:32 — The running server has no way to obtain its first valid session
- (high) src/linkd/Main.java:57 — Query values remain percent-encoded
- (high) src/linkd/Main.java:188 — CSV reports and plain-text responses are advertised as JSON
- (medium) src/linkd/Authenticator.java:62 — requireAdmin rejects every administrator
- (medium) src/linkd/Main.java:86 — Malformed numeric request parameters abort the handler
- (medium) src/linkd/Main.java:105 — Unsupported methods on /links execute the listing handler
- (medium) src/linkd/Main.java:156 — The quotas endpoint discards every per-owner count
- (medium) src/linkd/Main.java:179 — Reports silently omit links beyond the first 1,000
- (medium) src/linkd/Main.java:35 — A port bind failure leaves an unusable service process running
- (medium) src/linkd/Store.java:192 — The backing links map escapes the store
- (medium) src/linkd/Cache.java:58 — Expired cache entries prevent warming
- (medium) src/linkd/Cache.java:112 — Partial cache utilization is reported as zero
- (medium) src/linkd/Config.java:108 — A zero cache limit is accepted but breaks cache operations
- (medium) src/linkd/Export.java:68 — Snapshot writes do not provide the promised crash durability
- (medium) src/linkd/Worker.java:56 — A stopped janitor cannot be restarted
- (medium) src/linkd/Worker.java:77 — Target checks report unreachable HTTPS links as healthy
- (high) src/linkd/Worker.java:44 — A store mutation can terminate the expiry janitor
- (medium) src/linkd/Worker.java:56 — Stopping the janitor permits another sweep and prevents restart
- (medium) src/linkd/Cache.java:32 — Concurrent first callers can receive different process-wide caches
- (medium) src/linkd/Cache.java:90 — Cache warming can overwrite a target installed by another caller
- (medium) src/linkd/Store.java:53 — Concurrent renames can silently discard a link
- (medium) src/linkd/Store.java:28 — Concurrent creates can lose an owner's quota increment
- (medium) src/linkd/Store.java:45 — Concurrent resolutions lose hit increments
- (medium) src/linkd/Authenticator.java:49 — Session issuance or revocation can abort a concurrent verification
- (high) src/linkd/Main.java:37 — The wildcard listener accepts bearer credentials over plaintext HTTP
- (medium) src/linkd/Main.java:179 — User reports include other owners' links
- (medium) src/linkd/Export.java:41 — Reports containing all owners' links inherit filesystem permissions
- (low) src/linkd/Main.java:182 — Report errors disclose deployment paths and filesystem details
