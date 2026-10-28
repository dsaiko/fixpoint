# ollama-codex@glm-5.3-flash:cloud#max · java · run 20261028-132110 (repeat 1)

recall **49/69** · 61 finding(s), 4 unmatched · 648202 tokens · 724s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Link.equals does not define hashCode |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: First link for an owner causes NullPointerException |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry check is inverted and deletes active links; review-security: Link expiration check is inverted |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source code twice |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Paging end index is one beyond the list |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success rate is truncated by premature integer division |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not reclaim its quota |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard sorting is ascending |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner performs substring matching |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode compares codes by reference |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas exposes the internal mutable map |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll is not atomic |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune mutates the map while iterating it; review-concurrency: prune removes map entries during iteration |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-security: Short link codes are generated predictably |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Shared link and quota maps have no synchronization |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens use predictable java.util.Random |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Full bearer token is written to logs |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Expired sessions are accepted and retained |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Session tokens are compared by reference; review-security: Session verification uses reference equality and ignores exp |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashes are unsalted SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: Missing Authorization header causes a NullPointerException |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin can never return true |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Session map is accessed unsynchronized |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Singleton initialization has a check-then-act race |
| J30 | — | parallel: CA02 peek reads the map with no synchronization while synchr |  |
| J31 | YES | parallel: CA07 stats reads the non-volatile hit counter with no synchr | review-concurrency: Cache read paths bypass synchronization |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache eviction is not least-recently-used |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Cache sweep throws ConcurrentModificationException; review-concurrency: Cache sweep removes entries while iterating the key set |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization calculation is wrong and can divide by zer |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad timeout value is replaced with zero, not the default |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Milliseconds are interpreted as cache TTL seconds |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always reports success |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared SimpleDateFormat is not thread-safe |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: User-controlled report name allows path traversal and arbitr |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV fields are not escaped; review-security: CSV fields are neither quoted nor neutralized |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Failed snapshot rename is ignored |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics partial synchronization permits lost updates |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes entries while iterating the key set; review-concurrency: Janitor mutates and races Store's exposed map |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache leaks its executor and does not wait; review-concurrency: warmCache leaks its thread pool and does not wait |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner throws for an unknown owner |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | YES | parallel: S06 the Authorization header may be absent, and bearerToken  | review-security: Unauthenticated requests crash the authentication helper |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-bugs: Unvalidated query fields cause uncaught runtime exceptions; review-security: Authenticated users can create links owned by arbitrary iden |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any session holder can delete another user's link |
| J62 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts a spoofable request header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Statistics divide by zero when no links exist |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect mishandles missing and uppercase codes |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Every authenticated user can export all owners' links |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report reads a different file than it writes when name is mi |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:54 — Query parameters are not URL-decoded
- (medium) src/linkd/Main.java:91 — Create response is not JSON-escaped
- (medium) src/linkd/Store.java:209 — Generated codes are not checked for collisions
- (medium) src/linkd/Export.java:41 — Sensitive export files are created with inherited permissions
