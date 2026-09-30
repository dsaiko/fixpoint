# gpt-6.1-sol · java · run 20260930-100214 (repeat 1)

recall **37/69** · 51 finding(s), 9 unmatched · 394265 tokens · 1095s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Code-based equality lacks a matching hashCode |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: Creating a link fails for every new owner; review-bugs: A missing target bypasses the client-error response |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a valid link deletes it |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Renaming validates the source code twice |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Partial pages use an invalid exclusive end index |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success percentages truncate to zero |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Owner counts are not maintained across link mutations |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard returns the least-followed links |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner lookup matches substrings instead of exact owners |
| J10 | — | JAVA-ONLY findByCode compares strings with == instead of equals, so it |  |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten or revive its lifetime |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: Quota snapshots expose the live counters |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Failed batch imports leave partial changes |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: Pruning invalidates its own iterator |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: Short hexadecimal values make code generation throw |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | — | parallel: S03 every issued session token is printed in cleartext |  |
| J21 | — | parallel: S04 verify never looks at expiresAt, so a token works foreve |  |
| J22 | — | JAVA-ONLY verify compares the token with == rather than equals, so aut |  |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Missing or malformed Authorization headers crash handlers |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: The admin-role condition rejects every role |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Make singleton initialization atomic |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Protect cache readers with the cache monitor |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Eviction ignores the recorded access times |
| J33 | YES | parallel: CA06 warm's peek and set are separately locked, so the janit | review-concurrency: Make cache warming's absence check and insertion atomic |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Cache sweeping invalidates its own iterator |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization loses fractional occupancy |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout input discards the working default |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: The configured fetch limit is never assigned |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Validation accepts configurations it identifies as invalid |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Avoid concurrent use of the shared SimpleDateFormat |
| J43 | — | parallel: EX01 the report name comes from the query string and is join |  |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: Write failures leak report and audit file handles |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV fields are written without escaping |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot publication failures are silently ignored; review-concurrency: Give each snapshot write its own temporary file |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Synchronize updates to the created counter |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Expiry cleanup can terminate the janitor; review-concurrency: Synchronize janitor access with store operations |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: Cache warming leaks executor threads; review-concurrency: Shut down and await the cache-warming executor |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: Counting an owner with no quota entry throws |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| J62 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| J63 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Statistics crash when the store is empty |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect requests without a code crash; review-bugs: Redirects change case-sensitive stored codes |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Reports with the default name read the wrong file |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Main.java:32 — The server never creates an authenticatable session
- (high) src/linkd/Authenticator.java:50 — Token verification compares String identities
- (high) src/linkd/Main.java:57 — Query parameters are never URL-decoded
- (medium) src/linkd/Cache.java:66 — Zero-capacity caches still store entries and crash statistics
- (medium) src/linkd/Main.java:86 — Malformed numeric parameters escape request error handling
- (medium) src/linkd/Main.java:188 — CSV and text responses are labeled as JSON
- (medium) src/linkd/Store.java:129 — Code lookup compares String identities
- (medium) src/linkd/Store.java:27 — Generated code collisions overwrite existing links
- (medium) src/linkd/Main.java:35 — Avoid leaking the janitor when server initialization fails
