# qwen/qwen3.8-27b · code · run 20260820-130703 (repeat 1)

recall **46/60** · 58 finding(s), 2 unmatched · 134497 tokens · 2683s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Create panics on every call: s.quota is a nil map |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve inverts the expiry check: live links are deleted and; review-security: Resolve() expiry check inverted: live links deleted, expired |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice and never validates 'to' |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page returns all[offset:end] with end = len(all)+1 — out-of- |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate truncates before scaling — result is only ever 0 |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete never returns the owner's quota slot |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow | review-bugs: Top sorts ascending — it returns the least-followed links |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links | review-bugs: ByOwner matches by substring, not equality |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives expired links, contradicting 'expiry is final |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic — an invalid link after valid ones leav |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the live internal map, not a snapshot |
| C15 | YES | Prune counts removals but returns the number of links left | review-bugs: Prune returns the remaining link count instead of how many i |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink ignores the TTL parse error — bad TTL creates an ; review-security: createLink takes owner from request body, not the authentica |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes resp.Body — connection and fd leak per  |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF with response readout via /preview |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code but store keys are case-sensiti |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink performs no ownership check (IDOR) |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe | review-security: Open redirect in /out |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin endpoint gated by client-controlled X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats panics with integer division by zero on an empty store |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON calls WriteHeader before setting headers — Content |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Wildcard CORS with credentials on all JSON responses includi |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback HMAC secret |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens minted with math/rand, not a CSPRNG |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Full session token written to the log on issuance |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks ExpiresAt — sessions never expire; review-security: Verify() never checks session expiry |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Password hashing is a single unsalted SHA-256 |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho | review-bugs: BearerToken panics on any request without a well-formed Auth; review-security: BearerToken panics on missing or malformed Authorization hea |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin's condition is a tautology — it rejects every r |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | — | the sessions map is read and written from every request goroutine with |  |
| CF01 | YES | a malformed LINKD_PORT silently becomes port 0 | review-bugs: ignored Atoi error on LINKD_PORT — bad value yields port 0,  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: bad LINKD_TIMEOUT_MS logs 'keeping default' but then sets Ti |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is documented as milliseconds but applied; review-security: Bad LINKD_TIMEOUT_MS silently yields Timeout=0 (no timeout) |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed and validated, then discarded |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil, and main never calls it — bad c; review-security: Config.Validate always returns nil, so bad deploys never fai |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in export name allows arbitrary file write |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Cross-tenant report file written world-readable/writable (06 |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-security: CSV report fields written unescaped: formula injection via o |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the Close and Rename errors — reports  |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Get mutates shared state under a shared RLock: concurrent Ge |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo | review-concurrency: Guaranteed deadlock: Set holds the write lock, evict calls G |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: c.timers is never cleaned up — unbounded map growth and leak; review-concurrency: Cache.timers only grows: evict and Delete never remove entri |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil request on error — client.Do(nil) pan |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver — it increments a copy, so ; review-concurrency: AddResolved has a value receiver: copies a struct containing |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store has no synchronization: janitor and concurrent HTTP ha; review-security: Janitor iterates the links map without a lock; concurrent ha |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-concurrency: WarmCache calls wg.Add inside the spawned goroutine: Wait ca |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-concurrency: CheckTargets leaks one goroutine per unfinished probe: recei |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) export.go:42 — ArchiveAll ignores WriteString errors — silent partial archive, nil error returned
- (medium) export.go:26 — Concurrent /report requests with the same name race on one CSV file: non-atomic write plus
