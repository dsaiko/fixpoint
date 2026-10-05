# vllm@bottlecapai/ThinkingCap-Qwen3.8-27B · java · run 20261005-023717 (repeat 1)

recall **47/69** · 61 finding(s), 4 unmatched · 65273 tokens · 1110s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | YES | JAVA-ONLY equals is overridden without hashCode, so links break as map | review-bugs: equals() defined on code but no hashCode() — breaks use as s |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: create() throws NPE for every new owner: unboxing a null quo |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry check: live links are deleted, ; review-security: resolve() expiry check is inverted: live links deleted on fi |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice and never validates the new  |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: page() off-by-one: sets end = all.size() + 1, so any page ru |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: successRate() integer-divides before multiplying — reports 0 |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot, so quota coun |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, but the contract says most hits first |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: byOwner() matches by substring, not exact owner; review-security: byOwner matches with contains() — substring match leaks othe |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: findByCode() compares codes with ==, so it almost always ret |
| J11 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects already-expired links, contradicting 'ex; review-security: extend() resurrects expired links, violating the 'expiry is  |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: quotas() returns the live internal map, not a snapshot |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: importAll() is not atomic: a mid-batch validation failure le; review-security: importAll validates codes but not targets, bypassing the htt |
| J14 | — | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i |  |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | YES | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr | review-bugs: newCode() can throw StringIndexOutOfBoundsException for shor; review-security: Short codes are 32 bits of java.util.Random — enumerable |
| J17 | — | parallel: X02 the store's HashMap is mutated by request threads and th |  |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Session tokens minted from java.util.Random — predictable be |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to stdout and echoed in 401 responses |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: verify() never checks session expiry, contradicting its cont |
| J22 | YES | JAVA-ONLY verify compares the token with == rather than equals, so aut | review-bugs: verify() compares tokens with ==, so valid tokens never veri |
| J23 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashes are unsalted single-iteration SHA-256 |
| J24 | YES | parallel: S06 bearerToken indexes parts[1] without checking, so a malf | review-bugs: bearerToken() NPEs on a missing header and AIOOBEs on a bare |
| J25 | YES | parallel: S07 the admin check is always true, so requireAdmin rejects  | review-bugs: requireAdmin() condition is always true — it rejects every s |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | — | the sessions map is read and written from every request thread with no |  |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Non-volatile lazy singleton in Cache.getInstance: racing cal |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: peek, stats, utilization are unsynchronized and warm does an |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: set() evicts an arbitrary entry, not the least recently used |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-concurrency: Cache.sweep removes entries during keySet iteration; CME kil |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: utilization() integer-divides before multiplying, so it repo |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to 0 while logging 'keepin |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is read in milliseconds but stored as sec; review-bugs: Uncaught NumberFormatException in load() crashes startup on  |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: LINKD_FETCH_LIMIT is dead: parsed value is self-assigned and |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so it never rejects a bad co |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-bugs: Shared static SimpleDateFormat is not thread-safe; concurren |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Path traversal in exportCsv/readReport: user-controlled name |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | — | parallel: EX03 CSV rows are built by concatenation, so a comma or quot |  |
| J46 | YES | archiveAll builds a filename straight from the owner string, so an own | review-security: archiveAll builds file names from user-controlled owner with |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: writeSnapshot ignores renameTo's result; the snapshot is sil |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-security: Snapshot written to a fixed predictable /tmp path — local sy |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Metrics.addCreated is unsynchronized despite the class claim |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | — | JAVA-ONLY the janitor swallows InterruptedException without restoring  |  |
| J53 | YES | JAVA-ONLY the janitor mutates the store's map while iterating it, from | review-concurrency: Janitor thread reads/removes from the store's unsynchronized; review-concurrency: Janitor thread dies with ConcurrentModificationException on  |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: warmCache() never waits for probes and leaks an 8-thread non; review-concurrency: warmCache never awaits its tasks and never shuts down its po |
| J55 | YES | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null | review-bugs: countForOwner() NPEs for an owner with no quota entry |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | YES | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin | review-concurrency: No setExecutor: all handlers serialize on the single dispatc |
| J58 | — | parallel: S06 the Authorization header may be absent, and bearerToken  |  |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | — | parallel: C28 the link owner comes from the request instead of the aut |  |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-bugs: DELETE /links never enforces the ownership check the comment; review-security: DELETE /links never checks ownership; any authenticated user |
| J62 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by client-controlled X-Admin heade |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: stats() divides by store.size() — /stats 500s on an empty st |
| J65 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: /l without ?code NPEs (code.toLowerCase() on null) instead o |
| J66 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin with credentials on every response |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report serves every owner's links to any authenticated user |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report without ?name writes links.csv but reads null.csv, a |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) src/linkd/Main.java:86 — links() handler lets NumberFormatException escape: bad ttl/offset/size 500 instead of 400
- (medium) src/linkd/Store.java:28 — Quota is tallied but never enforced — unbounded per-user link creation (DoS)
- (low) src/linkd/Main.java:91 — 201 response is hand-built JSON; attacker-controlled target can inject fields
- (low) src/linkd/Main.java:182 — 500 response leaks absolute export path and full exception detail
