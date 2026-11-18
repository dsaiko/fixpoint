# gpt-6.2-sol · go · run 20261118-140500 (repeat 1)

recall **41/56** · 80 finding(s), 14 unmatched · 321004 tokens · 863s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolving a live link deletes it; review-security: Link resolution accepts expired links until cleanup |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename accepts invalid destination codes |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Requesting a partial final page panics |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: Success rate is zero until every link has been followed |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Quota counts drift as links are removed or imported |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extending a link can shorten its lifetime or revive it |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: A failed import leaves earlier rows in the store |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the store's mutable map |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid link TTLs are silently accepted |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: Preview leaves upstream response bodies open; review-concurrency: Preview can leave an HTTP transport goroutine waiting on an  |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Link previews can fetch internal services and return their r |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: Redirect lowercases case-sensitive link codes |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: Any authenticated user can delete another owner's link; review-security: Link listings disclose records across owners |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Quota endpoint trusts a caller-supplied admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: Stats divides by zero when the store is empty; review-bugs: The errors metric always reports zero |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: JSON response headers are set after the status is committed; review-concurrency: Concurrent reports can truncate the file another response is |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | YES | ListenAndServe with the default server: no read, write or header timeo | review-bugs: The running server has no way to issue a session; review-concurrency: Incomplete HTTP headers can hold server goroutines indefinit |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | — | session tokens come from math/rand, so they are predictable and forgea |  |
| S03 | YES | every issued session token is written to the log in cleartext | review-bugs: Expired sessions remain in memory indefinitely; review-security: Session issuance logs reusable bearer credentials |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-security: Session verification accepts expired credentials |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin rejects every role |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Session verification races with issuance and revocation |
| CF01 | YES | a malformed LINKD_PORT silently becomes port 0 | review-bugs: A malformed port silently selects an ephemeral listener |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: A malformed timeout disables the HTTP client timeout |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: The configured fetch limit is discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Invalid configuration passes validation |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Report names can overwrite CSV files outside the export dire |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: CSV exports can be readable by other local users |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV exports corrupt fields containing commas, quotes, or new; review-security: CSV exports allow spreadsheet formula injection |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll keeps every owner's file open until the batch fin |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: Snapshot publication can fail while WriteSnapshot reports su |
| EX07 | YES | nothing fsyncs before the rename the comment calls durable, and the te | review-bugs: Snapshot data is not synchronized before publication |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Concurrent cache reads mutate entries under a shared lock |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | — | one timer per Set is stored and never stopped, so timers and their gor |  |
| CA06 | YES | Warm's Peek and Set are separately locked, so the janitor and a handle | review-concurrency: Cache warming can race with insertion and overwrite a newer  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | YES | evict drops whatever entry map iteration yields first, not the least r | review-bugs: Eviction does not select the least recently used entry |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link creation allows owner impersonation |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | YES | Create never checks whether the generated code is already in use, so a | review-bugs: A generated code collision overwrites an existing link |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | YES | WarmCache and CheckTargets take no context, so their in-flight probes  | review-concurrency: Target probes can wait indefinitely with a zero-timeout HTTP |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | YES | ArchiveAll discards every WriteString error and still returns nil, so  | review-bugs: ArchiveAll reports success after a file write fails |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: A malformed target can crash the process during a probe |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-concurrency: AddResolved copies the mutex and can block forever |

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
- (medium) cache.go:71 — Cache timers outlive their entries and can evict replacements
- (medium) cache.go:63 — A zero-capacity cache still stores an entry
- (medium) store.go:39 — Link creation accepts targets without a host
- (medium) main.go:91 — Preview records a link as followed
- (medium) main.go:165 — Stats and CSV reports silently omit links above 1000
- (medium) main.go:182 — Reports fail when the export directory is absent
- (medium) store.go:98 — Offset pagination can skip and repeat links
- (medium) store.go:160 — ByOwner exposes mutable stored links
- (medium) export.go:36 — Owners with the same basename overwrite each other's archives
- (medium) worker.go:20 — Resolution metrics are incremented on a copy
- (medium) cache.go:79 — Eviction deadlocks when the cache fills
- (medium) cache.go:92 — Deleting a missing cache key leaves the cache locked
- (medium) cache.go:71 — An old expiry callback can delete a replacement entry
- (medium) main.go:96 — Disconnecting a preview caller leaves its outbound fetch running
