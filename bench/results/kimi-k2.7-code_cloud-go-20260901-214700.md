# kimi-k2.7-code:cloud · go · run 20260901-214700 (repeat 1)

recall **32/56** · 56 finding(s), 0 unmatched · 189048 tokens · 483s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve deletes live links and rejects them as not found; review-security: Inverted expiry check deletes live links and keeps expired o |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the 'from' code twice and skips validating  |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page slices past the end of the result set and panics; review-security: listLinks panics when fewer links than page size exist |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate returns 0% for most ratios due to integer divisi |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend resurrects expired links contrary to its contract |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink ignores TTL parse errors and creates zero-TTL lin |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: /preview fetches arbitrary user-supplied URLs without SSRF p |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code, breaking case-sensitive lookup |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink removes any link regardless of owner |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin endpoint trusts client-supplied X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats handler panics when no links exist |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader, so they are ignore |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with non-cryptographic randomness |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Issued session tokens are logged in plaintext |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks token expiration; review-security: Expired and revoked session tokens remain valid |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin always rejects because its condition is a tauto |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Authentication failures echo the supplied token in the respo |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator sessions map is not concurrency-safe |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | — | the parse-error branch logs 'keeping default' and then assigns the fai |  |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is parsed as milliseconds but stored as s |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed but never assigned to the config |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Config.Validate always returns nil |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-bugs: ExportCSV allows path traversal despite claiming names are c; review-security: CSV export writes to attacker-controlled path with world-wri |
| EX02 | — | the export holding every owner's links is written world-readable |  |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-security: CSV export does not escape user-controlled fields |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the rename error |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-bugs: Cache.Get deletes from the map while holding only a read loc; review-concurrency: Cache.Get mutates shared state while holding only a read loc |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek, Warm and Stats access shared state without locki |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Cache.Delete returns without unlocking on missing entry |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: Cache.Delete leaks the mutex when the entry is already gone; review-concurrency: Cache.Set leaves the old timer running when re-adding a code |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: createLink accepts arbitrary owner from request body |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: Link listing returns every owner's links |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | YES | ArchiveAll discards every WriteString error and still returns nil, so  | review-bugs: ArchiveAll ignores CSV write errors |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved uses a value receiver, so resolved hits are lost |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links |
| C15 | YES | Prune counts removals but returns the number of links left |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered |
