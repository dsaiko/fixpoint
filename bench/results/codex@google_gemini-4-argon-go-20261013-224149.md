# codex@google/gemini-4-argon · go · run 20261013-224149 (repeat 2)

recall **42/56** · 63 finding(s), 2 unmatched · 895758 tokens · 438s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Inverted expiry check in Store.Resolve deletes active links  |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Store.Rename validates 'from' twice and never validates 'to'; review-security: Rename validates source code twice and leaves destination co |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Off-by-one slice bound in Store.Page causes out-of-range pan |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: Store.SuccessRate performs integer division before multiplyi |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Store.Delete does not decrement the owner's quota count |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Store.Extend revives already-expired links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Store.Import mutates the store before validating the full ba |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Store.Quotas returns internal map reference instead of a sna |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink ignores strconv.Atoi error on TTL, creating immed |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview leaks HTTP response body on every successful upstrea |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Server-Side Request Forgery (SSRF) in link preview endpoint |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases short codes before lookup, breaking mixe |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: Missing ownership check in deleteLink allows deleting any us |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Privilege escalation via client-controlled X-Admin request h |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats handler panics with division by zero when store is emp |
| C21 | YES | the export handler returns the server's filesystem path and raw error  | review-security: Report endpoint leaks internal filesystem paths and exports  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON calls WriteHeader before setting response headers |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Overly permissive wildcard CORS with credentials in writeJSO |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with predictable math/rand PRNG |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Plaintext bearer session tokens written to application logs |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Authenticator.Verify never checks session expiration; review-security: Verify ignores session expiration and uses non-constant-time |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Unsalted single-round SHA-256 used for password hashing |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin uses || instead of &&, rejecting all sessions |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Unauthorized response reflects raw bearer token and internal |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Data race and concurrent map panic on Authenticator.sessions |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: LoadConfig overwrites Timeout with 0 when LINKD_TIMEOUT_MS i |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is multiplied by time.Second instead of t |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LoadConfig discards parsed LINKD_FETCH_LIMIT instead of upda |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Config.Validate always returns nil even when validation prob |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-concurrency: Concurrent /report requests race on the same output file pat; review-security: Path traversal via user-controlled report name in ExportCSV |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Sensitive multi-owner CSV export written with world-readable |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-security: CSV injection and unescaped fields in ExportCSV |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers file Close inside a loop and ignores write |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores close/rename errors and creates temp f |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates map, entry timestamp, and hit counter unde |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Unsynchronized reads in Cache.Peek and Cache.Stats and TOCTO |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Cache timers are never stopped or removed, leaking timers an; review-concurrency: Cache.Delete returns without unlocking mutex on missing key  |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | YES | evict drops whatever entry map iteration yields first, not the least r | review-bugs: Cache.evict deletes an arbitrary map entry instead of the le |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner taken from untrusted request body instead of auth |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns nil Request on invalid URL, causing CheckTa |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: Metrics.AddResolved uses a value receiver so resolved count  |

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

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) main.go:96 — server.preview does not propagate request context to outbound HTTP call
- (high) auth.go:84 — Broken boolean condition in RequireAdmin rejects all sessions
