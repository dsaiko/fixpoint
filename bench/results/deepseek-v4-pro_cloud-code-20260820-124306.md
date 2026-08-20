# deepseek-v4-pro:cloud · code · run 20260820-124306 (repeat 1)

recall **47/60** · 59 finding(s), 1 unmatched · 305246 tokens · 359s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Create panics: s.quota is a nil map; review-security: Store.Create panics on nil quota map |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve expiry check is inverted; review-security: Resolve expiry check is inverted: expired links are served |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page sets end to len(all)+1, causing slice-out-of-range pani; review-security: Store.Page panics with slice bounds out of range (DoS) |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate does integer division before multiplying |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow | review-bugs: Top sorts ascending, returning the least-followed links |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links | review-bugs: ByOwner uses substring matching instead of exact owner match |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives already-expired links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic despite its contract |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the internal map, not a snapshot |
| C15 | YES | Prune counts removals but returns the number of links left | review-bugs: Prune returns the remaining count, not the number removed |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink ignores a bad TTL and creates an already-expired  |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the response body |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF in /preview: server fetches arbitrary stored targets |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases a case-sensitive code |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink has no owner check (IDOR) |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe | review-security: Open redirect in /out via unvalidated ?next= |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: adminQuotas trusts a client-supplied X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero when the store is empty |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader, so they are droppe |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Permissive CORS: * with credentials |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand (predictable) |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session token written to logs in plaintext |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks ExpiresAt; review-security: Verify never checks ExpiresAt, so sessions never expire |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho | review-bugs: BearerToken panics on a missing or malformed Authorization h; review-security: BearerToken panics on missing/malformed Authorization header |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin condition is a tautology |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map is unsynchronized across Issue/Ve |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Bad LINKD_TIMEOUT_MS sets Timeout to 0 instead of keeping th |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds, not millisecon |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil even when problems are found |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in CSV export writes files outside the export |
| EX02 | — | the export holding every owner's links is written world-readable |  |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: ExportCSV does not escape fields, producing malformed CSV |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers f.Close inside the loop, leaking file desc |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the rename error and can fail across f |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates the map, entry, and hit counter while hold |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek reads the map without any lock |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Delete returns without unlocking when the code is absent; review-concurrency: Cache.Delete returns while still holding the mutex, leaking  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo | review-bugs: evict calls Get while holding the write lock, deadlocking |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: Set -> evict -> Get deadlocks when the cache is full; review-concurrency: Cache.Set overwrites a code's timer without stopping the old |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: Cache.Stats reads the hit counter without synchronization |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil request on error, which client.Do wil |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved uses a value receiver, so the counter never incr; review-concurrency: Metrics.AddResolved uses a value receiver, copying the mutex |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store has no synchronization; every handler and the janitor ; review-concurrency: Store.Quotas returns the internal quota map by reference, no |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: WarmCache calls wg.Add inside the goroutine, so Wait returns; review-concurrency: WarmCache calls wg.Add(1) inside the goroutine, so Wait can  |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-bugs: CheckTargets leaks goroutines on the first failure; review-concurrency: CheckTargets leaks goroutines blocked on an unbuffered chann |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) main.go:67 — createLink takes Owner from the request body, not the session
