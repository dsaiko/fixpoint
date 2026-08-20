# deepseek-v4-flash:0731-cloud · code · run 20260820-122214 (repeat 1)

recall **40/60** · 48 finding(s), 0 unmatched · 410462 tokens · 318s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Create panics: quota map is never initialized; review-security: Create panics on a nil quota map |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve expiry check is inverted: live links deleted, expire; review-security: Resolve deletes live links: expiry check is inverted |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates from twice and never validates to |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page computes a slice end of len(all)+1 and panics on the la; review-security: Page off-by-one slices out of bounds and stats divides by ze |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate computes the percentage in the wrong order |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow | review-bugs: Top sorts ascending despite claiming 'most hits first' |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links | review-bugs: ByOwner uses substring matching instead of exact owner equal |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend resurrects already-expired links despite the document |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C15 | YES | Prune counts removals but returns the number of links left | review-bugs: Prune returns the remaining count instead of the removed cou |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink silently ignores a non-numeric TTL; review-security: createLink trusts the client-supplied owner instead of the s |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the target response body |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: preview is an SSRF proxy over any http(s) target |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: Redirect lowercases the code but codes are case-sensitive |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink has no ownership check: any user deletes any link |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe | review-security: Open redirect in /out via unvalidated next parameter |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin endpoint gated only by a spoofable X-Admin request hea |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero when there are no links |
| C21 | YES | the export handler returns the server's filesystem path and raw error  | review-security: CSV report leaks every owner's links to any authenticated us |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader, so they are discar |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Permissive CORS: wildcard origin with allow-credentials on a |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand, not crypto/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session bearer tokens written to logs in plaintext |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks session expiry; review-security: Session expiry is never enforced by Verify |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho | review-bugs: BearerToken panics on a missing or malformed Authorization h; review-security: BearerToken panics when the Authorization header is missing  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin can never succeed |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map is read and written with no lock |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | — | the parse-error branch logs 'keeping default' and then assigns the fai |  |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | — | Validate logs the problems it finds and returns nil, so a bad config s |  |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: report name allows path traversal out of ExportDir |
| EX02 | — | the export holding every owner's links is written world-readable |  |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the os.Rename error and reports succes |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates the map and counters while holding only RL |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek and Cache.Stats read the map/hits without any loc |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-concurrency: Cache.Delete returns early without unlocking, permanently we |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo | review-bugs: Cache.Set deadlocks when the cache is full; review-concurrency: Cache.evict self-deadlocks: called under write Lock, it call |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: Set doesn't stop the previous TTL timer, so a stale timer de |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved uses a value receiver, so the resolved counter n; review-concurrency: Metrics.AddResolved uses a value receiver, so resolved click |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store has no synchronization; handlers and the janitor race  |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-concurrency: WarmCache calls wg.Add(1) inside the goroutine after go, so  |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-concurrency: CheckTargets leaks goroutines when it returns early on the f |
