# x-ai/grok-4.6 · code · run 20260820-122245 (repeat 1)

recall **39/60** · 44 finding(s), 2 unmatched · 216940 tokens · 909s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Create panics on nil quota map |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve treats live links as expired and serves dead ones |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates from twice and never validates to |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page panics on a partial last page |
| C05 | — | used/len*100 in integer arithmetic is 0 for every rate under 100% |  |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete does not restore the owner's quota slot |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow | review-bugs: Top returns the least-hit links, not the most |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links | review-bugs: ByOwner matches substrings, not the owner |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives already-expired links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C15 | YES | Prune counts removals but returns the number of links left | review-bugs: Prune returns remaining size instead of removed count |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid TTL silently becomes a zero duration |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the upstream response body |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Preview handler SSRF-fetches the target and returns the body |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: Redirect lowercases codes that were stored mixed-case |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink ignores the session owner; any caller can delete ; review-security: List and report endpoints return every tenant's links |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe | review-security: Open redirect on GET /out via unvalidated next parameter |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin quotas authorized by a client-supplied X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: /stats divides by zero when the store is empty |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets Content-Type after WriteHeader |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens are minted with math/rand, not a CSPRNG |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Live session tokens are logged and echoed on 401 |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never rejects an expired session; review-security: Verify never checks ExpiresAt, so sessions cannot time out |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho | review-bugs: BearerToken panics when Authorization is missing or not two  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin is always true and always rejects |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | — | the sessions map is read and written from every request goroutine with |  |
| CF01 | YES | a malformed LINKD_PORT silently becomes port 0 | review-bugs: LINKD_PORT parse errors become port 0 |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Bad LINKD_TIMEOUT_MS still overwrites the default |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is applied as seconds |
| CF04 | — | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti |  |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate logs problems then always returns nil |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: User-controlled export name allows path-traversal writes |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Export CSV is written mode 0666 despite holding every owner' |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores rename failure and skips fsync |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates the map, entry, and hit counter under RLoc |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek reads the map with no lock |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo | review-concurrency: evict deadlocks by calling Get while the write lock is held |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: Cache.Delete leaves the mutex locked when the key is absent |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved uses a value receiver so the counter never moves; review-concurrency: AddResolved uses a value receiver and copies the mutex |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store maps are shared across HTTP and janitor with no lock |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-concurrency: WarmCache calls WaitGroup.Add inside the started goroutine |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-concurrency: CheckTargets leaks workers on the first error |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) store.go:123 — SuccessRate integer-divides away the percentage
- (medium) export.go:42 — ArchiveAll reports success after ignored write errors
