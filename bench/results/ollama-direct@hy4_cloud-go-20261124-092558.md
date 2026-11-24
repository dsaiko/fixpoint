# ollama-direct@hy4:cloud · go · run 20261124-092558 (repeat 1)

recall **42/56** · 70 finding(s), 3 unmatched · 1285099 tokens · 1548s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve has the expiration test inverted; review-security: Expiry comparison is inverted, leaving expired links resolva |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename never validates the destination code |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page computes an out-of-range slice end |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate truncates the percentage |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete does not return the owner's quota slot |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend can revive an expired link |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the store's internal map |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid TTL strings create zero-lifetime links |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: Preview leaks HTTP response bodies; review-concurrency: preview ignores r.Context() and never closes resp.Body, leak |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Preview endpoint is a server-side request forgery sink |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: Redirect lowercases case-sensitive short codes |
| C17 | — | deleteLink never checks the owner its comment promises to check: any c |  |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin endpoint trusts a client-controlled X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: Stats divides by zero on an empty store |
| C21 | YES | the export handler returns the server's filesystem path and raw error  | review-security: Export failures disclose server filesystem paths and OS erro |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after committing the status |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | YES | the create handler decodes an unbounded request body straight into mem | review-security: Link creation accepts an unbounded JSON request body |
| C25 | YES | ListenAndServe with the default server: no read, write or header timeo | review-bugs: The server never makes a session token available |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens are generated with a non-cryptographic PRNG |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Live bearer tokens are written to logs and reflected in erro |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never enforces session expiry; review-security: Session expiry is never enforced |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Passwords are stored as unsalted, single-iteration SHA-256 h |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin can never accept a session |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions is an unsynchronized shared map read  |
| CF01 | YES | a malformed LINKD_PORT silently becomes port 0 | review-bugs: An invalid port is silently applied |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: A bad timeout value disables the client timeout |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: Millisecond cache TTL is converted as seconds |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always reports success |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Report name permits path traversal and arbitrary file overwr |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Sensitive all-owner reports are created with world-readable  |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV fields are written without escaping; review-security: CSV export does not escape untrusted link fields |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll leaks descriptors and hides write failures |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: Snapshot replacement can silently fail; review-bugs: Snapshot data is not synced and close errors are ignored |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates shared state while holding only an RLock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Peek and Stats read shared state with no lock |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-concurrency: Cache.Delete returns while still holding the write lock when |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Cache timers are neither canceled nor removed; review-concurrency: Set never stops old expiry timers and never cleans timers, s |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | YES | evict drops whatever entry map iteration yields first, not the least r | review-bugs: Eviction ignores recency and runs on updates |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link ownership is taken from untrusted request JSON |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: Listing and report endpoints disclose links for every tenant; review-security: Any authenticated user can delete short links belonging to a |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil request for malformed targets |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved uses a value receiver |

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
- (high) main.go:25 — Startup never validates the loaded configuration
- (medium) cache.go:63 — A nonpositive cache limit still caches an entry
- (medium) cache.go:54 — Peek treats expired entries as cached
