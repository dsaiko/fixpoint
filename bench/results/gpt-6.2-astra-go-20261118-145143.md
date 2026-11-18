# gpt-6.2-astra · go · run 20261118-145143 (repeat 1)

recall **31/56** · 67 finding(s), 18 unmatched · 154252 tokens · 301s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolving a live link deletes it |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and accepts invalid destin |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: A partial final page panics instead of returning the remaini |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate reports zero for every partially followed collec |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Removing or importing links leaves owner counts incorrect |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend can revive expired links or shorten live ones |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the store's mutable map |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid TTL input creates a link with an unintended expiry |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: Preview leaves the upstream HTTP response body open; review-concurrency: Preview leaves HTTP transport work active after reading the  |
| C27 | — | preview fetches any registered target server-side and returns its body |  |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: Redirect changes case-sensitive short codes before lookup |
| C17 | — | deleteLink never checks the owner its comment promises to check: any c |  |
| C19 | — | the admin endpoint is gated on a request header the caller sets themse |  |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: The stats handler divides by zero on an empty store |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: JSON responses commit their headers before setting them |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | YES | ListenAndServe with the default server: no read, write or header timeo | review-bugs: The running server has no way to accept a valid session; review-concurrency: Incomplete HTTP headers can hold connection goroutines indef |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | — | session tokens come from math/rand, so they are predictable and forgea |  |
| S03 | — | every issued session token is written to the log in cleartext |  |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Expired sessions accumulate indefinitely |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | — | the admin check is always true, so RequireAdmin rejects every session  |  |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Session verification races with issuance and revocation |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: An invalid timeout silently disables the HTTP client deadlin |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: Configured cache TTLs are 1,000 times too long |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: A valid fetch-limit override is discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Invalid configuration never stops startup |
| EX01 | — | the report name comes from the query string and goes into filepath.Joi |  |
| EX02 | — | the export holding every owner's links is written world-readable |  |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV fields containing commas, quotes, or newlines corrupt re |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: Archiving keeps one file descriptor open per owner |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: Snapshot installation can fail while returning success |
| EX07 | YES | nothing fsyncs before the rename the comment calls durable, and the te | review-bugs: A successful snapshot return does not establish durability |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Get mutates the cache under a shared read lock |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | — | one timer per Set is stored and never stopped, so timers and their gor |  |
| CA06 | YES | Warm's Peek and Set are separately locked, so the janitor and a handle | review-concurrency: Warm can overwrite an entry inserted after its presence chec |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: The cache hit counter has unsynchronized readers and writers |
| CA08 | YES | evict drops whatever entry map iteration yields first, not the least r | review-bugs: Eviction never selects the least recently used entry |
| C28 | — | the link owner is taken from the request body instead of the authentic |  |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | YES | Create never checks whether the generated code is already in use, so a | review-bugs: A generated code collision overwrites an existing link |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | YES | ArchiveAll discards every WriteString error and still returns nil, so  | review-bugs: ArchiveAll reports success after an archive write fails |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: An invalid probe URL can panic the process |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: Successful resolutions never increment the dashboard counter; review-concurrency: AddResolved copies the metrics mutex and discards every incr |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links |
| C15 | YES | Prune counts removals but returns the number of links left |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho |
| CF05 | YES | any read error -- not just a missing file -- silently yields an empty  |
| CA04 | — | evict runs under the write lock and calls Get, which takes the read lo |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) main.go:182 — CSV reports silently omit links beyond the first 1,000
- (medium) store.go:39 — The URL check accepts targets without a host
- (medium) store.go:98 — Offset pagination can repeat or skip links in an unchanged store
- (medium) store.go:151 — A negative leaderboard size panics
- (medium) store.go:160 — ByOwner exposes mutable store records
- (medium) auth.go:84 — RequireAdmin rejects every role
- (medium) main.go:165 — Dashboard averages exclude links beyond an arbitrary 1,000
- (medium) worker.go:35 — The dashboard error counter can never increase
- (medium) cache.go:63 — Set requests eviction even when no new capacity is needed
- (medium) cache.go:71 — An old expiry timer can delete a refreshed cache entry
- (medium) cache.go:94 — Timer records outlive the cache entries they track
- (medium) export.go:36 — Distinct owners can overwrite each other's archive files
- (high) export.go:26 — Concurrent reports can overwrite a response while it is being served
- (medium) cache.go:79 — Eviction deadlocks on the cache's own write lock
- (medium) cache.go:92 — Deleting an absent cache entry permanently locks the cache
- (medium) cache.go:54 — Peek reads the cache map without synchronization
- (medium) cache.go:71 — An old expiration timer can delete a replacement entry
- (medium) main.go:96 — Preview fetches continue after the requesting client disconnects
