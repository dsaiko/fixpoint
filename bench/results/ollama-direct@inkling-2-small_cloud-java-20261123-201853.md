# ollama-direct@inkling-2-small:cloud · java · run 20261123-201853 (repeat 1)

recall **50/69** · 82 finding(s), 20 unmatched · 353432 tokens · 242s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() overridden without hashCode() |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: First link for any owner throws NPE (quota.get unboxing) |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry check inverted: live links are deleted, expired links; review-concurrency: link.hits++ and resolve() check-then-act are not atomic |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice and never validates 'to' |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() sets end = size+1, causing IndexOutOfBoundsException |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() divides before multiplying — returns 0 unless  |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() doesn't refund the owner's quota slot as documented |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the LEAST followed links |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() uses substring match instead of equality |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares strings with == |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects already-expired links, contradicting its |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() exposes the live internal map, not the promised sna; review-concurrency: quotas() returns the live internal map despite documenting i |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic despite the documented all-or-noth; review-concurrency: importAll() documented as atomic but applies a partial batch |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: prune() iterates the live keySet while removing — Concurrent; review-concurrency: prune() removes from the map while iterating its keySet |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | — | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr |  |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: links/quota maps mutated from all handler threads and the ja |
| J18 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens minted from java.util.Random |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to stdout logs |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks session expiry; review-security: verify() never checks session expiry |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: Token compared with == instead of equals; review-security: verify() compares tokens with == (reference equality) |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted plain SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken crashes on missing or malformed Authorization he; review-security: bearerToken() crashes on malformed Authorization header |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin can never return true; review-security: requireAdmin() condition can never pass |
| J26 | YES | secretEquals is documented as not leaking how much matched and uses St | review-security: secretEquals() uses String.equals, leaks prefix length via t |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: sessions HashMap shared across request threads with no synch |
| J28 | YES | JAVA-ONLY hashPassword returns the empty string when the digest is una | review-bugs: hashPassword() returns empty string on algorithm failure |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Unsynchronized lazy singleton in getInstance() with non-vola |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek()/stats()/utilization() read cache state without synchr |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Eviction is not LRU and can evict an unrelated entry on upda |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: sweep() removes from entries while iterating its keySet — Co; review-concurrency: sweep() removes entries from the map while iterating its key |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() truncates to 0 unless full (and divides by zer |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to 0 instead of keeping th |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS (milliseconds) loaded straight into a sec; review-bugs: Unparseable LINKD_CACHE_SIZE / LINKD_FETCH_LIMIT / LINKD_CAC |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true; bad deploys are never refuse |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared static SimpleDateFormat used from multiple threads |
| J43 | — | parallel: EX01 the report name comes from the query string and is join |  |
| J44 | YES | JAVA-ONLY the FileWriter is closed only on the success path; any excep | review-bugs: FileWriter not closed if write() throws |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV fields are written without escaping |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot rename result ignored; cross-filesystem /tmp→target |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-concurrency: writeSnapshot() uses a fixed shared temp path in /tmp |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated() is not synchronized unlike the rest of  |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | YES | JAVA-ONLY the janitor swallows InterruptedException without restoring  | review-concurrency: Janitor thread is non-daemon and stopJanitor() neither inter |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-bugs: Janitor removes from the map while iterating it — thread die; review-concurrency: Janitor thread iterates and removes from store.all()'s live  |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache returns before the work finishes and leaks the poo; review-concurrency: warmCache() never shuts down or awaits its thread pool despi |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner unboxes a possibly-null Integer |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | — | parallel: C17 the delete branch never checks the owner its comment pro |  |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unsanitized target interpolated into JSON response |
| J63 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Stats endpoint divides by store.size() — ArithmeticException |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Missing ?code in /l redirect throws NullPointerException |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | YES | offset and size are parsed with unguarded parseInt, so any non-numeric | review-bugs: Unvalidated offset/size query params crash the listing handl |
| J68 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report reads a different filename than exportCsv wrote when |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:57 — Query parameter values are never percent-decoded
- (low) src/linkd/Cache.java:57 — peek() reports expired entries as present, defeating warm()
- (high) src/linkd/Main.java:175 — Open redirect in /out via ?next=
- (high) src/linkd/Main.java:139 — DELETE /links deletes any link — no ownership check (IDOR)
- (high) src/linkd/Main.java:122 — Link owner taken from query parameter, not the session
- (medium) src/linkd/Main.java:190 — Admin endpoint authorized by client-supplied X-Admin: true header
- (high) src/linkd/Export.java:369 — Path traversal via report name (write and read)
- (low) src/linkd/Main.java:228 — Access-Control-Allow-Origin: * combined with Access-Control-Allow-Credentials: true
- (low) src/linkd/Main.java:107 — 401 response echoes the presented token
- (low) src/linkd/Main.java:221 — 500 response leaks internal filesystem path
- (high) src/linkd/Store.java:276 — resolve() expiry check is inverted — live links deleted, expired links served
- (medium) src/linkd/Store.java:380 — extend() resurrects already-expired links
- (medium) src/linkd/Store.java:354 — byOwner() uses substring match — leaks other owners' links
- (medium) src/linkd/Store.java:444 — Short-code generation uses java.util.Random and 8 hex chars
- (medium) src/linkd/Export.java:383 — Report file written with default permissions despite 'service-user only' contract
- (medium) src/linkd/Export.java:374 — CSV injection / formula injection in exported report
- (medium) src/linkd/Export.java:409 — Snapshot written to predictable shared /tmp path — symlink/TOCTOU
- (medium) src/linkd/Export.java:395 — archiveAll() builds filenames from unvalidated owner — path traversal
- (low) src/linkd/Config.java:340 — validate() always returns true — invalid config accepted
- (low) src/linkd/Config.java:272 — LINKD_CACHE_TTL_MS documented as milliseconds but stored/used as seconds
