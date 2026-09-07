# gpt-5.6-terra · java · run 20260907-143439 (repeat 1)

recall **29/69** · 35 finding(s), 3 unmatched · 282899 tokens · 405s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Link equality violates the hashCode contract |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: Creating a first link for any owner always throws |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolver deletes live links and serves expired links |
| J04 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Listing a short final page throws IndexOutOfBoundsException |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success rate is zero unless every link was followed |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not release its quota count |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard is ordered least-followed first |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup returns links for partial owner names |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: Code lookup uses String identity instead of value equality |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending an expired link revives it |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Batch import is not atomic |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: Pruning expired links can throw ConcurrentModificationExcept |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | — | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr |  |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | — | parallel: S03 every issued session token is printed in cleartext |  |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-security: Expired bearer sessions remain accepted |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Token verification compares String references |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Malformed or absent Authorization headers crash handlers |
| J25 | — | parallel: S07 the admin check is always true, so requireAdmin rejects  |  |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | — | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre |  |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Cache reads its HashMap without the synchronization used by  |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache eviction is not least-recently-used |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Sweeping multiple expired entries throws ConcurrentModificat |
| J35 | — | parallel: C05 utilization divides before scaling so it is always 0, an |  |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond cache TTL is applied as seconds; review-bugs: Invalid cache environment values abort startup |
| J39 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s |  |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | — | parallel: CF06 validate logs the problems it finds and returns true, s |  |
| J42 | — | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  |  |
| J43 | — | parallel: EX01 the report name comes from the query string and is join |  |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot move failure is reported as success |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | — | parallel: X01 addCreated is unsynchronized while addResolved is, so co |  |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor dies when removing expired links during iteration; review-concurrency: Janitor iterates the store HashMap concurrently with request |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: Warm-cache workers are never awaited or shut down; review-concurrency: warmCache leaks executor threads and returns before warming  |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: Counting an owner with no links throws NullPointerException |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete another user's link |
| J62 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is controlled by a client-supplied heade |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Stats crashes when the store has no links |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect without a code throws instead of returning 404 |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Any authenticated user can download every tenant's links |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Unnamed reports write one file and read another |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Cache.java:112 — A configured zero-size cache crashes the stats endpoint
- (high) src/linkd/Main.java:57 — Query values are never URL-decoded
- (high) src/linkd/Main.java:179 — Report endpoint allows traversal outside the export directory
