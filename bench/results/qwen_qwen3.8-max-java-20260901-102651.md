# qwen/qwen3.8-max · java · run 20260901-102651 (repeat 1)

recall **44/69** · 52 finding(s), 3 unmatched · 59833 tokens · 1560s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() overridden without hashCode(), breaking hash-based  |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() throws NullPointerException for every owner's first |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links are deleted, ; review-security: Expiry check in resolve() is inverted — expired links stay r |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() clamps end to all.size() + 1 — subList always throws  |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() does integer division before multiplying — onl |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never decrements the owner's quota despite the docu |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least-followed links |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches with String.contains, returning other owne |
| J10 | — | JAVA-ONLY findByCode compares strings with == instead of equals, so it |  |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links despite the documente |
| J12 | — | parallel: C14 quotas hands out the store's own map while the doc calls |  |
| J13 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| J14 | — | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i |  |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-security: Short codes minted with java.util.Random truncated to 8 hex  |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Unsynchronized HashMaps shared by handler threads and the ja |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are generated with java.util.Random and are p; review-security: Bearer session tokens are written to the log on issue |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt despite the documented lifet; review-security: verify() never checks expiresAt — session tokens never expir |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares tokens with == instead of equals(), so no  |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256; error path yields an |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() throws on missing or malformed Authorization h; review-security: Malformed or missing Authorization header crashes every auth |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin() can never return true: !admin || !owner is al |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals claims constant-time comparison but uses String |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: getInstance is an unsynchronized check-then-act lazy singlet |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek/stats/utilization bypass the lock every other Cache met |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Eviction removes an arbitrary HashMap entry, not the least r |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() removes from entries while iterating its keySet — Co |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() does integer division before multiplying — alw |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS zeroes the timeout instead of keeping t |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is milliseconds but is assigned directly  |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied — parsed = par |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, even when problems are found |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Process-wide static SimpleDateFormat shared across request h |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in CSV report name allows writing files outsi |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot() ignores renameTo's result across a filesyste |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Snapshot written to a predictable /tmp path — local symlink  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated is unsynchronized while addResolved/snaps |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes from store.all() while iterating its keySet ; review-concurrency: Janitor removes from the map while iterating it and races ha |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-concurrency: warmCache leaks an 8-thread pool and returns before the work |
| J55 | — | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null |  |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: Link creation takes the owner from a query parameter, not th |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links performs no ownership check — any user can del |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out sends the caller to any unvalidated ?nex |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated by client-controlled X-Admin header ins |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats() divides by store.size() — ArithmeticException on an  |
| J65 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS combined with Access-Control-Allow-Credentials |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports and returns every owner's links to any authe |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() reads back a different filename than exportCsv wrot |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:89 — POST /links crashes with 500 on missing target or non-numeric ttl
- (low) src/linkd/Authenticator.java:49 — sessions HashMap iterated in verify() while issue()/revoke() mutate it unsynchronized
- (medium) src/linkd/Export.java:41 — All-tenant report written with default umask despite 'service user only' requirement
