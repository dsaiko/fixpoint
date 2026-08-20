# mistral-large-3:675b-cloud · code · run 20260820-133057 (repeat 1)

recall **2/60** · 4 finding(s), 2 unmatched · 57265 tokens · 4055s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | — | s.quota is never initialized in NewStore; first Create panics with nil |  |
| C02 | — | expiry comparison inverted: live links are deleted, expired ones resol |  |
| C03 | — | Rename validates `from` twice; `to` is never validated |  |
| C04 | — | Page clamps end to len(all)+1; last page slices out of range |  |
| C05 | — | used/len*100 in integer arithmetic is 0 for every rate under 100% |  |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C10 | — | Top sorts by hits ascending, so the leaderboard shows the least follow |  |
| C11 | — | ByOwner matches owners by substring, so one owner sees another's links |  |
| C12 | — | Extend re-dates an already-expired link, resurrecting a code the comme |  |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C15 | — | Prune counts removals but returns the number of links left |  |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C27 | — | preview fetches any registered target server-side and returns its body |  |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | — | deleteLink never checks the owner its comment promises to check: any c |  |
| C18 | — | /out redirects to any address in ?next= -- an open redirect, cached pe |  |
| C19 | — | the admin endpoint is gated on a request header the caller sets themse |  |
| C20 | — | /stats divides by the link count, panicking whenever the store is empt |  |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | — | writeJSON calls WriteHeader before setting headers, so Content-Type an |  |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded JWT secret in production code |
| S02 | — | session tokens come from math/rand, so they are predictable and forgea |  |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Weak JWT signing algorithm |
| S04 | — | Verify never looks at ExpiresAt, so a token works forever despite the  |  |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S06 | — | BearerToken indexes parts[1] without checking; a request with no Autho |  |
| S07 | — | the admin check is always true, so RequireAdmin rejects every session  |  |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | — | the sessions map is read and written from every request goroutine with |  |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | — | the parse-error branch logs 'keeping default' and then assigns the fai |  |
| CF03 | — | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim |  |
| CF04 | — | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti |  |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | — | Validate logs the problems it finds and returns nil, so a bad config s |  |
| EX01 | — | the report name comes from the query string and goes into filepath.Joi |  |
| EX02 | — | the export holding every owner's links is written world-readable |  |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | — | the rename's error is discarded and /tmp is usually another filesystem |  |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | — | Get deletes from the map and increments hits while holding only a read |  |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA04 | — | evict runs under the write lock and calls Get, which takes the read lo |  |
| CA05 | — | one timer per Set is stored and never stopped, so timers and their gor |  |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | — | AddResolved has a value receiver; the mutex is copied and the incremen |  |
| X02 | — | janitor iterates and deletes s.links concurrently with HTTP handlers;  |  |
| X03 | — | wg.Add inside the goroutine races wg.Wait; WarmCache can return before |  |
| X04 | — | CheckTargets returns on first error; remaining sends to the unbuffered |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) auth.go:42 — No validation of JWT claims
- (low) auth.go:37 — Missing error handling for empty JWT secret
