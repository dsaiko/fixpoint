# codex@moonshotai/kimi-k3.1-pro · java · run 20261024-222106 (repeat 1)

recall **42/69** · 51 finding(s), 6 unmatched · 841891 tokens · 380s

| seed | found | note | matched by |
|---|---|---|---|
| J01 | — | JAVA-ONLY equals is overridden without hashCode, so links break as map |  |
| J02 | YES | JAVA-ONLY (parallel: C01) quota has no entry for a new owner, so unbox | review-bugs: First link for an owner always fails quota accounting |
| J03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolution deletes live links and serves expired ones |
| J04 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| J05 | YES | parallel: C04 page clamps end past the list size, so the last page thr | review-bugs: Final partial page computes an out-of-bounds end index |
| J06 | YES | parallel: C05 integer division before scaling yields 0 for every rate  | review-bugs: Success percentage uses integer division before multiplying |
| J07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting links never releases their quota |
| J08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard sorts least-followed links first |
| J09 | YES | parallel: C11 byOwner matches owners by substring, so one owner sees a | review-bugs: Owner filtering returns substring matches |
| J10 | YES | JAVA-ONLY findByCode compares strings with == instead of equals, so it | review-bugs: Code lookup uses reference equality |
| J11 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| J12 | YES | parallel: C14 quotas hands out the store's own map while the doc calls | review-bugs: Quota snapshot exposes the mutable internal map |
| J13 | YES | parallel: C13 importAll stores as it validates, so a bad code leaves t | review-bugs: Batch import is not atomic |
| J14 | YES | JAVA-ONLY (parallel: C15) prune removes from the map while iterating i | review-bugs: Pruning an expired link corrupts iteration |
| J15 | — | JAVA-ONLY page returns a subList VIEW backed by the temporary list rat |  |
| J16 | — | JAVA-ONLY (parallel: S02) codes come from java.util.Random, and substr |  |
| J17 | YES | parallel: X02 the store's HashMap is mutated by request threads and th | review-concurrency: Store maps are accessed concurrently without synchronization |
| J18 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| J19 | YES | parallel: S02 session tokens come from java.util.Random, so they are p | review-security: Predictable PRNG allows session-token prediction |
| J20 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Live bearer tokens are written to application logs |
| J21 | YES | parallel: S04 verify never looks at expiresAt, so a token works foreve | review-bugs: Session tokens are compared by object identity; review-security: Expired sessions remain valid indefinitely |
| J22 | — | JAVA-ONLY verify compares the token with == rather than equals, so aut |  |
| J23 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| J24 | — | parallel: S06 bearerToken indexes parts[1] without checking, so a malf |  |
| J25 | — | parallel: S07 the admin check is always true, so requireAdmin rejects  |  |
| J26 | — | secretEquals is documented as not leaking how much matched and uses St |  |
| J27 | YES | the sessions map is read and written from every request thread with no | review-concurrency: Session registry is unsafe for concurrent authentication ope |
| J28 | — | JAVA-ONLY hashPassword returns the empty string when the digest is una |  |
| J29 | YES | JAVA-ONLY getInstance is an unsynchronized lazy singleton, so concurre | review-concurrency: Lazy singleton initialization can return different cache ins |
| J30 | YES | parallel: CA02 peek reads the map with no synchronization while synchr | review-concurrency: Unsynchronized cache reads violate the thread-safety contrac |
| J31 | — | parallel: CA07 stats reads the non-volatile hit counter with no synchr |  |
| J32 | YES | parallel: CA08 eviction drops whatever key the iterator yields first,  | review-bugs: Cache eviction removes through the map instead of the iterat |
| J33 | — | parallel: CA06 warm's peek and set are separately locked, so the janit |  |
| J34 | YES | JAVA-ONLY sweep removes entries while iterating the keySet, throwing C | review-bugs: Sweeping an expired entry corrupts iteration |
| J35 | YES | parallel: C05 utilization divides before scaling so it is always 0, an | review-bugs: Cache utilization truncates partial usage to zero |
| J36 | — | parallel: CF01 a malformed LINKD_PORT is swallowed by an empty catch b |  |
| J37 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Malformed timeout replaces the default with zero |
| J38 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| J39 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed and self-assigned, so the s | review-bugs: Fetch limit environment setting is discarded |
| J40 | — | parallel: CF05 loadFile swallows every IOException, so an unreadable c |  |
| J41 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation always accepts invalid settings |
| J42 | YES | JAVA-ONLY a static SimpleDateFormat is shared across request threads;  | review-concurrency: Shared SimpleDateFormat races across report requests |
| J43 | YES | parallel: EX01 the report name comes from the query string and is join | review-security: Report name permits path traversal and file overwrite |
| J44 | — | JAVA-ONLY the FileWriter is closed only on the success path; any excep |  |
| J45 | YES | parallel: EX03 CSV rows are built by concatenation, so a comma or quot | review-bugs: CSV fields are emitted without escaping; review-security: Attacker-controlled fields permit CSV formula injection |
| J46 | — | archiveAll builds a filename straight from the owner string, so an own |  |
| J47 | YES | parallel: EX06 renameTo's boolean result is discarded and it fails sil | review-bugs: Snapshot move failure is silently ignored |
| J48 | YES | parallel: EX07 the snapshot temp file has one fixed name in /tmp: a sy | review-concurrency: Concurrent snapshots share one temporary file |
| J49 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| J50 | YES | parallel: X01 addCreated is unsynchronized while addResolved is, so co | review-concurrency: Created metric has data races and lost updates |
| J51 | — | the errors counter is exposed on /stats and never incremented anywhere |  |
| J52 | YES | JAVA-ONLY the janitor swallows InterruptedException without restoring  | review-concurrency: Janitor stop and restart lifecycle is broken |
| J53 | — | JAVA-ONLY the janitor mutates the store's map while iterating it, from |  |
| J54 | YES | JAVA-ONLY (parallel: X03) warmCache never shuts down its ExecutorServi | review-bugs: Cache warmer returns before work finishes and leaks its pool; review-concurrency: warmCache returns early and leaks its executor |
| J55 | — | JAVA-ONLY countForOwner unboxes a possibly-null Integer, throwing Null |  |
| J56 | — | the janitor thread is not a daemon and is never joined, so the JVM can |  |
| J57 | — | JAVA-ONLY no executor is set on HttpServer, so it uses the default sin |  |
| J58 | YES | parallel: S06 the Authorization header may be absent, and bearerToken  | review-bugs: Missing or malformed Authorization headers crash every prote |
| J59 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| J60 | YES | parallel: C28 the link owner comes from the request instead of the aut | review-security: Caller can spoof link ownership and consume another user's q |
| J61 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-security: Any authenticated user can delete any owner's link |
| J62 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated next parameter creates an open redirect |
| J63 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Client-controlled X-Admin header bypasses admin authorizatio |
| J64 | YES | parallel: C20 /stats divides by the link count, throwing ArithmeticExc | review-bugs: Stats endpoint divides by zero on an empty store |
| J65 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| J66 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| J67 | — | offset and size are parsed with unguarded parseInt, so any non-numeric |  |
| J68 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exports every tenant's private link data |
| J69 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report name is applied only when writing |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) src/linkd/Authenticator.java:62 — Admin authorization rejects every role
- (medium) src/linkd/Main.java:57 — Query parameters are never URL-decoded
- (high) src/linkd/Store.java:28 — Concurrent creates lose quota updates
- (medium) src/linkd/Store.java:45 — Link hit increments race with resolves and readers
- (medium) src/linkd/Export.java:77 — Concurrent audit writes can interleave records
- (medium) src/linkd/Export.java:41 — Sensitive reports are created without restrictive permissions
