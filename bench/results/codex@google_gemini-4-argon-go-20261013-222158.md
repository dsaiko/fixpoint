# codex@google/gemini-4-argon · go · run 20261013-222158 (repeat 1)

recall **21/56** · 30 finding(s), 0 unmatched · 677963 tokens · 1478s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | — | expiry comparison inverted: live links are deleted, expired ones resol |  |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-security: Rename validates source code twice and fails to validate des |
| C04 | — | Page clamps end to len(all)+1; last page slices out of range |  |
| C05 | — | used/len*100 in integer arithmetic is 0 for every rate under 100% |  |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C12 | — | Extend re-dates an already-expired link, resurrecting a code the comme |  |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | YES | preview never closes resp.Body; connections leak | review-concurrency: preview handler ignores request context cancellation and lea |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Server-Side Request Forgery (SSRF) in preview handler |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: Missing ownership check in deleteLink allows any user to del |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin authorization bypass via client-supplied X-Admin heade |
| C20 | — | /stats divides by the link count, panicking whenever the store is empt |  |
| C21 | YES | the export handler returns the server's filesystem path and raw error  | review-security: Internal filesystem path and OS error exposed in report erro |
| C22 | — | writeJSON calls WriteHeader before setting headers, so Content-Type an |  |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Overly permissive wildcard CORS with credentials enabled |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Predictable session tokens generated using math/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Plaintext bearer session tokens written to logs and error re |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-security: Verify does not enforce session token expiration |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Unsalted single-round SHA-256 used for password hashing |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-security: Broken boolean condition in RequireAdmin rejects all session |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Unsynchronized concurrent access to Authenticator.sessions m |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | — | the parse-error branch logs 'keeping default' and then assigns the fai |  |
| CF03 | — | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim |  |
| CF04 | — | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti |  |
| CF06 | — | Validate logs the problems it finds and returns nil, so a bad config s |  |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in ExportCSV via unsanitized report name |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: ExportCSV writes multi-tenant report files with world-readab |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-security: CSV and formula injection in ExportCSV |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | — | the rename's error is discarded and /tmp is usually another filesystem |  |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates map, entry timestamp, and hit counter unde |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Unsynchronized reads in Cache.Peek and Cache.Stats |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-concurrency: Cache.Delete leaks c.mu lock on missing key and fails to sto |
| CA05 | — | one timer per Set is stored and never stopped, so timers and their gor |  |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: createLink trusts client-supplied owner in request body inst |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | — | AddResolved has a value receiver; the mutex is copied and the incremen |  |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| C01 | — | s.quota is never initialized in NewStore; first Create panics with nil |
| C10 | — | Top sorts by hits ascending, so the leaderboard shows the least follow |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links |
| C15 | — | Prune counts removals but returns the number of links left |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered |
