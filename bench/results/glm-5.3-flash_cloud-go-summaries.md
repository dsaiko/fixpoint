# glm-5.3-flash:cloud · go · run summaries (repeat 1)

recall **40/56** · 61 finding(s), 0 unmatched · 598016 tokens · 231s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve expiry condition is inverted, deleting every valid l; review-security: Expiry check is inverted: live links are destroyed on first  |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never validates t |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page sets end to len(all)+1 and panics on any short page |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate computes integer division before scaling, yieldi |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete discards the owner instead of returning the quota slo |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives expired links and replaces expiry instead of  |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic: a mid-batch validation failure leaves  |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the live internal map instead of a snapshot |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink ignores the strconv error, so a bad ttl yields an |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the upstream response body, leaking con |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF: /preview fetches an attacker-chosen URL from the serve |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code, breaking almost every link |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink never checks ownership: any user can delete anyon |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin endpoint authorizes on a client-controlled header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero when the store is empty; review-security: /stats divides by zero and panics once the store is empty |
| C21 | YES | the export handler returns the server's filesystem path and raw error  | review-security: Internal filesystem paths leaked in the /report error respon |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader, so Content-Type an |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Wildcard CORS combined with Allow-Credentials on every JSON  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret baked into the binary |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens minted from math/rand instead of crypto/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session tokens logged in plaintext |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks ExpiresAt, so expired sessions stay vali; review-security: Verify never checks ExpiresAt: expired sessions stay valid f |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin condition is always true and rejects every role; review-security: RequireAdmin rejects everyone: the condition can never be fa |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map is read and written from request  |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: A malformed LINKD_TIMEOUT_MS still overwrites Timeout with 0 |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is parsed as milliseconds but multiplied  |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded, so the setti |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil, so invalid configuration is nev |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in ExportCSV via the user-supplied report nam |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Export written with 0666 despite the comment promising owner |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: ExportCSV writes fields unescaped, producing a corrupt CSV f |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | — | the rename's error is discarded and /tmp is usually another filesystem |  |
| EX07 | YES | nothing fsyncs before the rename the comment calls durable, and the te | review-bugs: WriteSnapshot creates the temp file in /tmp and ignores the  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates map, entry and hits while holding only an  |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Set never stops or replaces timers, so stale timers delete f; review-concurrency: Per-key AfterFunc timers are never stopped and delete entrie |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | YES | evict drops whatever entry map iteration yields first, not the least r | review-bugs: evict drops an arbitrary entry, not the least recently used  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: createLink takes the owner from the request body, not the se |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: /report exports every owner's links to any authenticated use |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | YES | ArchiveAll discards every WriteString error and still returns nil, so  | review-bugs: ArchiveAll defers f.Close() inside the loop and ignores writ |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil request, which panics in client.Do |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: Metrics.AddResolved has a value receiver, so the resolved co |

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
