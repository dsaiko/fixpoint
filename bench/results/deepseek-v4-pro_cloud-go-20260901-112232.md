# deepseek-v4-pro:cloud · go · run 20260901-112232 (repeat 1)

recall **29/56** · 44 finding(s), 0 unmatched · 692020 tokens · 496s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve expiry check is inverted |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page panics with slice bounds out of range when the page ext |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate uses integer division and always returns 0 or 10 |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C12 | — | Extend re-dates an already-expired link, resurrecting a code the comme |  |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the response body |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF: /preview fetches an arbitrary user-controlled URL serv |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code, breaking codes that contain up |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: IDOR: deleteLink deletes any link without checking ownership |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Authorization bypass: adminQuotas trusts a client-supplied X |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero when the store is empty |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader, so they are droppe |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Permissive CORS: wildcard origin with credentials flag |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens minted with math/rand are predictable |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session token written to the log in plaintext |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks ExpiresAt, so sessions never expire; review-security: Verify never checks ExpiresAt, so sessions never expire |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin condition is always true; review-security: RequireAdmin always rejects due to || instead of && |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: sessions map has no mutex; Issue/Revoke race with Verify |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Bad LINKD_TIMEOUT_MS sets Timeout to 0 instead of keeping th |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is multiplied by time.Second, not millise |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| CF06 | — | Validate logs the problems it finds and returns nil, so a bad config s |  |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal: export filename escapes the export directory |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Export written world-readable (0666) despite 'service user o |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers Close inside the loop, leaking file descri |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the rename error and reports success |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Get writes to the map and entry while holding only a read lo |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Peek and Stats read shared state without the lock |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Cache.Delete returns without unlocking when the code is abse; review-concurrency: Delete returns without unlocking when the key is absent, dea |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: Set never stops the previous timer, so a stale timer deletes |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | — | the link owner is taken from the request body instead of the authentic |  |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-concurrency: AddResolved copies the mutex and counter, so resolved never  |

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
