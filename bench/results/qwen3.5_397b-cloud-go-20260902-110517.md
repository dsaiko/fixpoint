# qwen3.5:397b-cloud · go · run 20260902-110517 (repeat 1)

recall **17/56** · 31 finding(s), 2 unmatched · 328783 tokens · 131s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve deletes valid links and keeps expired ones |
| C03 | — | Rename validates `from` twice; `to` is never validated |  |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page has off-by-one error causing panic |
| C05 | — | used/len*100 in integer arithmetic is 0 for every rate under 100% |  |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C12 | — | Extend re-dates an already-expired link, resurrecting a code the comme |  |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF vulnerability in preview endpoint |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: Missing owner validation on link deletion |
| C19 | — | the admin endpoint is gated on a request header the caller sets themse |  |
| C20 | — | /stats divides by the link count, panicking whenever the store is empt |  |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | — | writeJSON calls WriteHeader before setting headers, so Content-Type an |  |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded secret key in production |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-concurrency: Authenticator.Issue uses math/rand without synchronization; review-security: Predictable session token generation using math/rand |
| S03 | — | every issued session token is written to the log in cleartext |  |
| S04 | — | Verify never looks at ExpiresAt, so a token works forever despite the  |  |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Weak password hashing with unsalted SHA256 |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin condition always true; review-security: Authorization bypass due to incorrect boolean logic |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | — | the sessions map is read and written from every request goroutine with |  |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | — | the parse-error branch logs 'keeping default' and then assigns the fai |  |
| CF03 | — | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim |  |
| CF04 | — | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Config.Validate never returns an error |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in CSV export filename |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: World-readable/writable export files |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll leaks file handles |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores rename error |
| EX07 | YES | nothing fsyncs before the rename the comment calls durable, and the te | review-concurrency: StartJanitor iterates store.links without synchronization |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get writes to entry.used while holding only RLock |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Cache.Delete has race condition with timer callback |
| CA05 | — | one timer per Set is stored and never stopped, so timers and their gor |  |
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
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: CheckTargets panics on request creation failure |
| X01 | — | AddResolved has a value receiver; the mutex is copied and the incremen |  |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil |
| C10 | — | Top sorts by hits ascending, so the leaderboard shows the least follow |
| C11 | — | ByOwner matches owners by substring, so one owner sees another's links |
| C15 | YES | Prune counts removals but returns the number of links left |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.go:78 — Rename validates wrong variable
- (low) cache.go:95 — Cache.Delete releases lock then calls Delete recursively without protection
