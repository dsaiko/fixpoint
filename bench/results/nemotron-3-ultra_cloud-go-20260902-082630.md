# nemotron-3-ultra:cloud · go · run 20260902-082630 (repeat 1)

recall **29/56** · 80 finding(s), 13 unmatched · 555381 tokens · 3041s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve expiry check logic inverted - deletes valid links; review-security: Resolve deletes non-expired links due to inverted expiration |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice, never validates 'to'; review-security: Rename validates 'from' code twice instead of validating 'to |
| C04 | — | Page clamps end to len(all)+1; last page slices out of range |  |
| C05 | — | used/len*100 in integer arithmetic is 0 for every rate under 100% |  |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete doesn't decrement quota - users can't recreate after ; review-concurrency: Store.Delete doesn't decrement quota |
| C12 | — | Extend re-dates an already-expired link, resurrecting a code the comme |  |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import not atomic - partial links stored on validation failu |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns internal map directly, not a snapshot; review-concurrency: Store.Quotas returns internal map reference |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF in preview handler - fetches user-controlled URLs witho |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases code but store uses case-sensitive keys |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-bugs: deleteLink missing ownership check - any user can delete any |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-bugs: adminQuotas checks X-Admin header that is never set; review-security: Admin endpoint protected only by easily-spoofed header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: Division by zero panic in stats when no links exist |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | — | writeJSON calls WriteHeader before setting headers, so Content-Type an |  |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: CORS misconfiguration - wildcard origin with credentials |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded weak default secret for session signing |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand instead of crypto/ra |
| S03 | YES | every issued session token is written to the log in cleartext | review-bugs: Verify does O(n) linear search instead of O(1) map lookup; review-security: Session tokens logged in plaintext |
| S04 | — | Verify never looks at ExpiresAt, so a token works forever despite the  |  |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Password hashing uses unsalted SHA-256 - vulnerable to rainb |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin logic always rejects - condition is always true; review-security: RequireAdmin logic bug - always denies access even for admin |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-bugs: Token leaked in error response; review-security: Token leaked in error response |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map has no synchronization |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | — | the parse-error branch logs 'keeping default' and then assigns the fai |  |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS parsed as seconds not milliseconds; review-security: Cache TTL parsed as seconds but env var is documented as mil |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT parsed but never assigned to config; review-security: LINKD_FETCH_LIMIT parsed but discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil even when validation fails; review-security: Validate() returns nil even when configuration is invalid |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in ExportCSV via user-controlled filename; review-security: Path traversal in ArchiveAll via owner-controlled filename |
| EX02 | YES | the export holding every owner's links is written world-readable | review-bugs: ExportCSV uses world-readable 0666 permissions despite comme; review-security: Exported CSV files created with world-readable permissions ( |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | — | the rename's error is discarded and /tmp is usually another filesystem |  |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-bugs: Get modifies map and counters under read lock (data races); review-concurrency: Cache.Get mutates map and counters under RLock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-bugs: Peek has no locking (concurrent map access race); review-concurrency: Cache.Peek accesses map without lock |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Delete doesn't stop timer, causing timer leak and double-del |
| CA06 | YES | Warm's Peek and Set are separately locked, so the janitor and a handle | review-bugs: Warm has TOCTOU race between Peek and Set |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-bugs: Stats has data race on hits counter; review-concurrency: Cache.Stats reads hits counter without lock |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | — | the link owner is taken from the request body instead of the authentic |  |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns nil request on error causing panic in calle |
| X01 | — | AddResolved has a value receiver; the mutex is copied and the incremen |  |

Retired seeds — still matched so their findings are not counted as noise, but out of the denominator:

| seed | found | note |
|---|---|---|
| C01 | — | s.quota is never initialized in NewStore; first Create panics with nil |
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
- (critical) cache.go:77 — evict calls Get while holding write lock (deadlock)
- (high) export.go:52 — WriteSnapshot doesn't check os.Rename error; cross-device rename fails silently
- (high) store.go:96 — Page has off-by-one error causing panic on last page
- (medium) store.go:113 — SuccessRate integer division order wrong - always returns 0 for partial success
- (medium) store.go:168 — Extend sets expiry from now instead of extending current expiry
- (low) store.go:197 — Prune returns remaining count instead of removed count
- (medium) cache.go:71 — Cache.Set timer can delete wrong entry after code reuse
- (medium) cache.go:89 — Cache.Delete doesn't clean up timer, double-delete risk
- (high) store.go:97 — Store returns internal Link pointers to callers
- (medium) store.go:185 — Store.Import doesn't update quota, no conflict check
- (critical) auth.go:84 — RequireAdmin logic always rejects
- (critical) worker.go:64 — SSRF in WarmCache and CheckTargets - fetches arbitrary user-controlled URLs
- (medium) export.go:52 — Snapshot temp file created in world-writable /tmp - symlink attack risk
