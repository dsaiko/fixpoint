# gpt-6-astra-xhigh · java · run 20260905-143959 (repeat 1)

recall **34/69** · 49 finding(s), 11 unmatched · 450808 tokens · 740s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Link equality is incompatible with hash collections |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: Creating the first link for an owner throws after inserting  |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolution deletes live links and serves expired links |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Pagination exceeds the list boundary on partial pages |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success percentages truncate before multiplication |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Link mutations leave owner counts inconsistent |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard selects the least-followed links |
| J09 | — | parallel: C11 byOwner matches owners by substring, so one owner sees a |  |
| J10 | — | JAVA-ONLY findByCode compares strings with == instead of equals, so it |  |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extension can shorten live links and revive expired links |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: The quota snapshot exposes mutable store state |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Failed batch imports retain earlier mutations |
| J14 | — | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i |  |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: Short random values crash code generation |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | — | parallel: S03 every issued session token is printed in cleartext |  |
| J21 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| J22 | — | JAVA-ONLY verify compares the token with == rather than equals, so aut |  |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Missing or incomplete Authorization headers abort requests |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: The administrator role check rejects every role |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Make singleton initialization atomic |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Protect cache readers with the writers' monitor |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache eviction ignores least-recently-used ordering |
| J33 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: Make warming's presence check and insertion atomic |
| J34 | — | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C |  |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization truncates partial occupancy to zero |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL milliseconds are interpreted as seconds; review-bugs: Invalid timeout input replaces the default with zero |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: The configured fetch limit is discarded |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation never rejects invalid values |
| J42 | — | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  |  |
| J43 | — | parallel: EX01 the report name comes from the query string and is join |  |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: Export writers leak when writing fails |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV exports corrupt fields containing delimiters |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot publication silently ignores rename failures |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Synchronize creation counter updates |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Expiry sweeps invalidate their own iterators; review-concurrency: Synchronize janitor access with store mutations |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: Cache warming leaks executor threads; review-concurrency: Await cache-warming tasks and shut down their executor |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: Counting links for an unknown owner throws |
| J56 | YES | the janitor thread is not a daemon and is never joined, so the JVM can | review-concurrency: Wake and join the janitor when stopping it |
| J57 | YES | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin | review-security: One incomplete unauthenticated request can stall the entire  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| J62 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| J63 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Statistics divide by zero when the store is empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Missing required query parameters cause null dereferences; review-bugs: Redirect lookup changes valid case-sensitive codes |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Unnamed reports are read from a different filename |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Main.java:32 — The HTTP server never provisions a usable session
- (high) src/linkd/Authenticator.java:50 — Token verification compares String identities
- (high) src/linkd/Main.java:57 — Query parameters are never URL-decoded
- (medium) src/linkd/Main.java:35 — Failed HTTP startup leaves the janitor running
- (medium) src/linkd/Cache.java:66 — A zero cache limit is accepted but cannot function
- (medium) src/linkd/Cache.java:66 — Updating a full cache can evict an unrelated entry
- (medium) src/linkd/Store.java:129 — Code lookup compares String identities
- (medium) src/linkd/Store.java:27 — Generated-code collisions overwrite existing links
- (medium) src/linkd/Main.java:86 — Malformed numeric parameters escape request error handling
- (medium) src/linkd/Main.java:188 — Non-JSON responses are advertised as JSON
- (medium) src/linkd/Main.java:35 — Clean up the janitor when server startup fails
