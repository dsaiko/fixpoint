# ollama-direct@hy4:cloud · go · run 20261126-085607 (repeat 1)

recall **46/56** · 78 finding(s), 6 unmatched · 729892 tokens · 1291s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve deletes live links and serves expired links |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never validates t |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page sets slice end past the array length; review-bugs: Page panics on negative offset or size |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate multiplies an integer division by 100 |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete does not return the owner quota slot |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives expired links and shortens far-future links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the store's internal map |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid TTL creates a zero-lifetime link |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the target response body; review-concurrency: Preview never closes the outbound response body |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Preview endpoint permits authenticated SSRF into internal se |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: Redirect lowercases case-sensitive generated codes |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: Any authenticated user can delete another owner's link |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin quotas are protected by a client-supplied header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero when there are no links |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON writes headers after WriteHeader |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | YES | ListenAndServe with the default server: no read, write or header timeo | review-concurrency: HTTP server is started without any connection timeouts |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session bearer tokens are generated with math/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Issued session tokens are written to logs |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never enforces Session.ExpiresAt; review-security: Session expiry is never enforced |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Password hashing uses a single unsalted SHA-256 digest |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin rejects every role |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Unauthorized responses echo the supplied bearer token |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Session map is written and iterated without synchronization |
| CF01 | YES | a malformed LINKD_PORT silently becomes port 0 | review-bugs: Invalid LINKD_PORT overwrites the default with zero |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Invalid LINKD_TIMEOUT_MS disables the HTTP timeout |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: Cache TTL uses seconds for a millisecond setting |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT has no effect |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate never returns an error |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: User-controlled report name escapes the export directory; review-security: Report failures disclose server filesystem details |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: CSV exports containing all users' data are created with mode |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV fields are written without CSV escaping; review-security: CSV cells are not encoded or formula-sanitized |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll keeps every output file open until function exit |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot reports success even when the snapshot is not ; review-bugs: WriteSnapshot leaks its temp file on write failure |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Get mutates shared cache state while holding only an RLock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Peek reads the cache map without taking the mutex |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Cache refreshes leave stale timers and unbounded timer state; review-concurrency: Old TTL timers can delete refreshed cache entries |
| CA06 | YES | Warm's Peek and Set are separately locked, so the janitor and a handle | review-concurrency: Warm's Peek-then-Set check is a TOCTOU race |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: Stats reads the hit counter without synchronization |
| CA08 | YES | evict drops whatever entry map iteration yields first, not the least r | review-bugs: evict deletes an arbitrary entry instead of the LRU entry |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link ownership is taken from untrusted JSON instead of the s |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: Listing and CSV report endpoints expose every owner's links |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | YES | WarmCache and CheckTargets take no context, so their in-flight probes  | review-concurrency: Parallel target probes have no caller-controlled cancellatio; review-concurrency: Janitor stop function cancels but never joins the goroutine |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | YES | ArchiveAll discards every WriteString error and still returns nil, so  | review-bugs: ArchiveAll ignores row write errors |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil request on invalid URLs |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved increments a copy of Metrics |

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
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) main.go:25 — Startup never calls Config.Validate
- (medium) store.go:151 — Top panics for negative n
- (high) store.go:190 — Import overwrites existing links and does not maintain quota counts
- (high) cache.go:91 — Delete leaves the cache mutex locked when the entry is absent
- (medium) main.go:96 — Preview ignores the inbound request context
- (medium) export.go:26 — Concurrent reports write and read the same stable filename
