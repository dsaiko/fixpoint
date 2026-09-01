# qwen/qwen3.8-27b · java · run 20260901-113347 (repeat 1)

recall **47/69** · 61 finding(s), 3 unmatched · 317832 tokens · 1557s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: Link.equals() overrides equals but not hashCode |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() NPEs on first link per owner: null Integer unbox |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry check: live links deleted, dead; review-security: resolve() inverts expiry check: destroys live links, resurre |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice, never validates 'to'; review-security: rename() validates the source code twice and never validates |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() off-by-one: subList toIndex = size+1 throws IndexOutO |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() integer division: always 0 or 100 |
| J07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, so it returns the least-followed link |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring match and NPEs on null owner; review-security: byOwner uses substring matching, returning other owners' lin |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares codes with == (reference equality) |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives an already-expired link; review-security: extend() resurrects already-expired codes, defeating the fin |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() returns the live map, not the snapshot it documents |
| J13 | — | parallel: C13 importAll stores as it validates, so a bad code leaves t |  |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() removes from a HashMap keySet while iterating -> CME |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | — | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr |  |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store's links/quota HashMaps are shared mutable state with n |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens minted with predictable java.util.Random |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to server log on issuance |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() ignores session expiry despite its contract; review-security: verify() never enforces session expiry despite documenting i |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares tokens with == (reference equality) so it  |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted single-shot SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() NPEs on a missing header and throws AIOOBE on ; review-security: bearerToken NPEs on missing or malformed Authorization heade |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin() is always false: || should be && |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals is not constant-time despite claiming to be |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | — | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre |  |
| J30 | — | parallel: CA02 peek reads the map with no synchronization while synchr |  |
| J31 | YES | parallel: CA07 stats reads the non-volatile hit counter with no synchr | review-concurrency: Cache.readers peek/stats/utilization are not synchronized, b |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: set() evicts an arbitrary entry, not the LRU one the doc pro |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() removes from a HashMap keySet while iterating -> CME |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() integer division: always 0 (or 100); divide-by |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to 0, not the default |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS: milliseconds assigned to a seconds field; review-bugs: LINKD_CACHE_SIZE parsed with no guard: non-numeric value cra |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied (parsed = pars |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so bad configs are never ref; review-security: Config.validate() collects problems but always returns true |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared static SimpleDateFormat (STAMP) is used from concurre |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in report name: arbitrary .csv read and write |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot() ignores renameTo result; /tmp->dir rename is |
| J48 | — | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy |  |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated is not synchronized while its sibling and |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: janitor removes from a HashMap keySet while iterating -> CME; review-concurrency: Janitor thread mutates the store's live map directly, unsync |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache() leaks a thread pool and returns before probes fi; review-concurrency: warmCache leaks a non-daemon thread pool and never awaits it |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() NPEs when the owner is absent from the quota |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: Link owner taken from client query param instead of authenti |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: DELETE /links has no ownership check — any user can delete a |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out via user-controlled ?next= |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated on client-supplied X-Admin header |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: /stats divides by store.size() -> 0/0 when there are no link |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() NPEs when ?code is missing |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Access-Control-Allow-Origin: * combined with Access-Control- |
| J67 | YES | offset and size are parsed with unguarded parseInt, so any non-numeric | review-bugs: GET /links: Integer.parseInt on offset/size is unguarded ->  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports every owner's links to any authenticated use |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() writes 'links.csv' but reads 'null.csv' when ?name  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:86 — POST /links: ttl parsed outside the try -> NFE 500
- (medium) src/linkd/Export.java:41 — All-owners report written world-readable despite claiming service-user-only access
- (low) src/linkd/Main.java:91 — Unescaped user target interpolated into hand-built JSON response
