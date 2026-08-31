# glm-5.2:cloud · code · run 20260820-123136 (repeat 1)

recall **40/60** · 49 finding(s), 1 unmatched · 255255 tokens · 325s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: NewStore never initializes the quota map, so Create panics o; review-bugs: Quotas returns the live internal map, not a snapshot as docu |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve's expiry check is inverted: deletes valid links, ser |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never validates t |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page clamps end to len(all)+1, causing a slice-bounds-out-of |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate uses integer division, rounding every partial fr |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow | review-bugs: Top sorts ascending by hits, returning least-followed first  |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links | review-bugs: ByOwner uses substring match instead of equality, matching u |
| C12 | — | Extend re-dates an already-expired link, resurrecting a code the comme |  |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic: stores links as it validates, leaving  |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C15 | YES | Prune counts removals but returns the number of links left | review-bugs: Prune returns the remaining count instead of the number remo |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the fetched response body, leaking a co |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF: /preview fetches and returns user-controlled URLs |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | — | deleteLink never checks the owner its comment promises to check: any c |  |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe | review-security: Open redirect in /out via the next query parameter |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin authorization trusts a client-controlled X-Admin heade |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by len(links) with no empty-store guard, panic |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON calls WriteHeader before setting Content-Type/CORS |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Permissive CORS: wildcard origin with credentials |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with non-cryptographic math/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Bearer token written to the application log |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks ExpiresAt, so expired sessions remain va; review-security: Verify ignores token expiry, so sessions never expire |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho | review-bugs: BearerToken indexes parts[1] without a bounds check, panicki; review-security: BearerToken panics on a missing or malformed Authorization h |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-security: RequireAdmin condition is tautological and always rejects |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Raw token reflected in the 401 response body |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map mutated concurrently with no lock |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: A malformed LINKD_TIMEOUT_MS zeroes the timeout despite logg |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is parsed as seconds, not milliseconds, c |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed then discarded, so FetchLimit ca |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil even when it detected problems,  |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in /report via unvalidated name parameter |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Export file created world-readable and world-writable (0666) |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers f.Close inside the loop, keeping every fil |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the os.Rename error and leaks the temp |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get writes to map and counter while only holding RLock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek reads the map with no lock |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo | review-bugs: evict calls Get while already holding the write lock, a self; review-concurrency: Cache.evict deadlocks: takes RLock while caller holds the wr |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Cache timers map is never cleaned and timers are never stopp; review-concurrency: Cache.Set overwrites timer without stopping the previous one |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil *http.Request on error, and CheckTarg |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver, so it locks and increments |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store map accessed concurrently with no synchronization; review-concurrency: Link.Hits incremented on shared *Link without synchronizatio |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: WarmCache calls wg.Add inside the goroutine, racing Wait and; review-concurrency: WarmCache calls wg.Add inside the goroutine, racing wg.Wait |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-concurrency: CheckTargets leaks goroutines blocked on send after early re |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.go:84 — RequireAdmin uses || where && is needed, rejecting everyone unconditionally
