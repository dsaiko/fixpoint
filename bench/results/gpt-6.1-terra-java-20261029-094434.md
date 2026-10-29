# gpt-6.1-terra · java · run 20261029-094434 (repeat 1)

recall **39/69** · 49 finding(s), 5 unmatched · 201131 tokens · 113s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | — | JAVA-ONLY equals is overridden without hashCode, so links break as map |  |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: Initialize the owner count before incrementing it |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Correct the expiry comparison when resolving links |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Validate the destination code on rename |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Clamp page end to the list size |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Calculate the success percentage before integer truncation |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Update the owner count when deleting a link |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Sort the leaderboard in descending order |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Match owners exactly |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: Compare link codes by value |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Do not revive expired links when extending them |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Validate the entire import before modifying the store |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: Remove pruned links through the iterator |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: Generate codes safely for short hexadecimal values |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens use predictable pseudorandom values |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued bearer tokens are written to logs |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Reject expired sessions; review-concurrency: Session verification races with issuance and revocation |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Compare session tokens by value |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Handle missing and malformed authorization headers |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: Allow the intended administrative roles |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | — | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre |  |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Cache reads bypass the lock used for mutations |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | — | parallel: CA08 eviction drops whatever key the iterator yields first,  |  |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Remove expired cache entries through the iterator |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Calculate cache utilization before integer truncation |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Preserve the default for an invalid timeout |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Convert the configured cache TTL from milliseconds |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: Apply the configured fetch limit |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Fail validation for invalid configuration |
| J42 | — | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  |  |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report names can traverse outside the export directory |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: Close report writers when a write fails |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: Escape CSV fields; review-security: Untrusted report values can inject spreadsheet formulas |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Check whether the snapshot was moved |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Created counter is updated outside the metrics lock |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Remove expired links through the iterator; review-concurrency: Janitor accesses the store concurrently without synchronizat |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: Shut down the cache-warming executor; review-concurrency: Cache warming leaks its thread pool and returns before work  |
| J55 | — | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null |  |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: Link ownership is taken from request input |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete another owner's link |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Outbound redirect accepts an arbitrary destination |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Administrator access is granted by a caller-controlled heade |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Handle an empty store in stats |
| J65 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Read the report using the resolved default name |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Main.java:57 — Decode query parameters before using them
- (critical) src/linkd/Store.java:27 — Avoid overwriting an existing generated code
- (medium) src/linkd/Link.java:28 — Provide a hash code consistent with link equality
- (high) src/linkd/Main.java:179 — Reports expose all owners' link targets
- (medium) src/linkd/Export.java:41 — Reports containing all owners' links use default file permissions
