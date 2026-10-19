# minimax-m3.1-flash:cloud · java · run 20261019-175923 (repeat 1)

recall **53/69** · 72 finding(s), 5 unmatched · 455293 tokens · 3087s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | — | JAVA-ONLY equals is overridden without hashCode, so links break as map |  |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() unboxes a null Integer from the quota map |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry test: live links are deleted an |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() clamps `end` to size+1 and throws IndexOutOfBoundsExc |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() truncates the ratio to an integer before scali |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not return the owner's quota slot |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least followed links |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches owners by substring instead of equality |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares codes with == instead of equals |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects an already-expired link |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() hands out the live quota map instead of a snapshot |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: a bad code leaves earlier links i |
| J14 | — | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i |  |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode() can produce a code shorter than four characters an; review-security: Short link codes are generated with java.util.Random, so oth |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store's maps are unsynchronized while the janitor and reques |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued bearer tokens are written to stdout in cleartext; review-security: Session tokens are drawn from java.util.Random, making them  |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-security: Session expiry is never enforced, so a leaked token works fo |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares session tokens with == (reference equality; review-security: Token is compared with == instead of equals, so the auth che |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords are hashed with a single unsalted SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() throws on a missing or malformed Authorization |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-security: requireAdmin can never return true, so the admin role check  |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals claims constant-time comparison but uses String |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Authenticator's session map is unsynchronized while being re |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Cache.getInstance is an unsynchronized lazy initialization o |
| J30 | — | parallel: CA02 peek reads the map with no synchronization while synchr |  |
| J31 | YES | parallel: CA07 stats reads the non-volatile hit counter with no synchr | review-concurrency: Cache.peek() and Cache.stats() read shared state without syn |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: set() evicts an arbitrary entry, not the least recently used |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() removes entries while iterating the keySet; review-concurrency: Cache.sweep() removes from a HashMap while iterating its key |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() truncates before scaling, and divides by zero  |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A malformed LINKD_TIMEOUT_MS sets the timeout to 0 while log |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is assigned to a field measured in second; review-bugs: Unguarded Integer/Long parsing crashes startup on a malforme |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed and then thrown away |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so the startup guard never r |
| J42 | — | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  |  |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-bugs: The user-supplied report name escapes the export directory (; review-security: Report name from the query string is used as a file path, al |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: FileWriter is not closed when the write throws |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV fields are not escaped, so a comma or newline in a targe; review-security: CSV export writes unescaped attacker-controlled fields, enab |
| J46 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll builds a filename directly from the owner name |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot() ignores renameTo's result and renames across |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Snapshot is written through a fixed, predictable filename in |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated() is not synchronized, so the created cou |
| J51 | YES | the errors counter is exposed on /stats and never incremented anywhere | review-bugs: The errors counter is never incremented, so /stats always re |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes from the store's map while iterating its key; review-concurrency: Janitor mutates the Store's HashMap while iterating it, corr |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache() returns before the probes finish and leaks its t; review-concurrency: warmCache submits tasks to a fixed pool but never waits for  |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() unboxes a null Integer and throws NPE |
| J56 | YES | the janitor thread is not a daemon and is never joined, so the JVM can | review-concurrency: stopJanitor() does not interrupt the sleeping janitor thread |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | YES | parallel: S06 the Authorization header may be absent, and bearerToken  | review-security: Unhandled exceptions on user input turn ordinary requests in |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-bugs: The authenticated session is never used for ownership: owner; review-security: Link owner is taken from the request body instead of the aut |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links removes any link by code with no ownership che |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: GET /out redirects to any caller-supplied URL (open redirect |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin authorization is a client-supplied X-Admin header; review-security: Admin-only endpoint authorizes on a client-supplied X-Admin  |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: /stats divides by the store size and 500s on an empty store |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() dereferences a null code when ?code is missing |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Every response carries a wildcard CORS origin together with  |
| J67 | YES | offset and size are parsed with unguarded parseInt, so any non-numeric | review-bugs: The listing endpoint parses offset/size without validation |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:54 — Query values are never URL-decoded and an & in the target truncates it
- (medium) src/linkd/Main.java:86 — The ttl parameter is parsed outside the try block, so a bad ttl yields a 500
- (medium) src/linkd/Authenticator.java:62 — requireAdmin() always returns false
- (low) src/linkd/Main.java:188 — Every response is labelled application/json, including the non-JSON bodies
- (medium) src/linkd/Main.java:91 — User-controlled target is interpolated into a JSON response without escaping
