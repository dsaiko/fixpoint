# deepseek/deepseek-v4.1-flash-1017 · java · run 20261017-214100 (repeat 1)

recall **52/69** · 65 finding(s), 3 unmatched · 324894 tokens · 1991s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() is overridden without hashCode() |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: Creating the first link for an owner throws NullPointerExcep |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry check is inverted: live links are deleted and expired; review-security: Expiry check is inverted: live links die, expired links reso |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() uses size+1 as the sublist end, throwing IndexOutOfBo |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() uses integer division and reports 0 for partia |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never releases the owner's quota slot |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least-followed links |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches owner by substring instead of exact equali |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode compares codes with == and always returns null fo |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links, contrary to its cont |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() returns the live internal map instead of a snapshot |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic despite its contract |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() removes from the keySet while iterating it |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode() can throw when toHexString produces fewer than 8 c; review-security: Short codes use java.util.Random and only 32 bits |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store's HashMaps are mutated from the janitor and HTTP threa |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret |
| J19 | — | parallel: S02 session tokens come from java.util.Random, so they are p |  |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to stdout; review-security: Session tokens minted with java.util.Random |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks expiresAt and never purges expired ses; review-security: Session expiry is never enforced |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Token comparison uses reference equality, so issued tokens n; review-security: Token comparison uses reference equality |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken throws on a missing or malformed Authorization h |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin rejects every role, including admin |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals is not constant-time despite its contract |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: sessions HashMap is shared across request threads without sy |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Cache.getInstance is an unsynchronized lazy singleton on a n |
| J30 | — | parallel: CA02 peek reads the map with no synchronization while synchr |  |
| J31 | YES | parallel: CA07 stats reads the non-volatile hit counter with no synchr | review-concurrency: utilization()/stats()/peek() read shared state without the c |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Eviction removes an arbitrary HashMap key instead of the lea |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() removes entries while iterating the keySet |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() truncates to zero and divides by zero for a ze |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets the timeout to 0 despite logging t |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is documented as milliseconds but stored  |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so invalid configuration is  |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared static SimpleDateFormat is not thread-safe |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in export filename (arbitrary .csv write) |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-security: Exports are written with default, often world-readable, perm; review-security: CSV formula and row injection from attacker-controlled field |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot ignores renameTo's result, silently losing the |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Snapshot uses a fixed, predictable /tmp path (symlink attack |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated increments a shared counter without the m |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes links from the map while iterating it; review-concurrency: Janitor iterates and removes from the live links map, killin |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache() leaks its thread pool and returns before the pro; review-concurrency: warmCache leaks a fixed thread pool and returns before the p |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() unboxes a null count for an unknown owner |
| J56 | YES | the janitor thread is not a daemon and is never joined, so the JVM can | review-concurrency: stopJanitor cannot interrupt the sleeping janitor and does n |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-bugs: Missing target/owner parameters reach Store.create and throw; review-security: Link owner is taken from the request instead of the session |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links deletes any owner's link (missing ownership ch |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on a client-supplied X-Admin heade |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats() divides by store.size() without guarding the empty s |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect dereferences a missing code parameter |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report returns every owner's links to any authenticated use |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report with no name writes links.csv but reads null.csv |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:57 — Query parameters are never URL-decoded
- (medium) src/linkd/Main.java:86 — Malformed ttl crashes the request instead of returning 400
- (low) src/linkd/Main.java:182 — Verbose export errors leak filesystem layout
