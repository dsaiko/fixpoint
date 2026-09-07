# gpt-5.6-terra · go · run 20260907-142733 (repeat 1)

recall **28/56** · 43 finding(s), 4 unmatched · 277923 tokens · 336s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolving a live link deletes it while expired links remain  |
| C03 | — | Rename validates `from` twice; `to` is never validated |  |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Partial final pages panic with slice bounds out of range |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate is zero unless every link has been used |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives already expired links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import partially modifies the store on validation failure |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid requested TTL silently becomes zero |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: Preview responses leak HTTP response bodies |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Preview endpoint permits SSRF to internal services |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: Redirects cannot resolve valid codes containing uppercase le |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: Any authenticated user can delete another user's link |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Administrative access is controlled by a client header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: Stats panics for an empty store |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | — | writeJSON calls WriteHeader before setting headers, so Content-Type an |  |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | — | session tokens come from math/rand, so they are predictable and forgea |  |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session bearer tokens are written to logs |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-security: Expired sessions remain valid indefinitely |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | — | the admin check is always true, so RequireAdmin rejects every session  |  |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator sessions map is unsynchronized |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Invalid timeout configuration disables the client timeout |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: Configured fetch limit is discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Configuration validation reports errors but always accepts t |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Report filename allows writes outside the export directory |
| EX02 | — | the export holding every owner's links is written world-readable |  |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV exports corrupt fields containing CSV syntax |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll leaves every output file open until the whole arc |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: Snapshot write reports success when the final rename fails |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache mutates shared state while holding only an RLock |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: Superseded expiry timers can delete a newly cached value |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link creation trusts an attacker-controlled owner |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: Link listing exposes all tenants' link data |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: Resolved metric increments are discarded; review-concurrency: AddResolved copies Metrics' mutex |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow |
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
- (high) cache.go:91 — Deleting a missing cache entry permanently retains the mutex
- (high) store.go:78 — Rename never validates the destination code
- (high) cache.go:91 — Delete leaves the cache mutex locked on a missing key
- (medium) main.go:96 — Preview ignores cancellation of the incoming request
