# minimax-m3:cloud · code · run 20260820-122557 (repeat 1)

recall **40/60** · 49 finding(s), 1 unmatched · 341108 tokens · 248s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Create writes to a nil quota map and panics |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve deletes live links and returns ErrNotFound for them |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice, never validates `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page off-by-one causes slice-bounds panic on full-page reque |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate divides before multiplying and is always 0 or 10 |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete never decrements quota, so the per-owner counter grow |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow | review-bugs: Top sorts ascending, leaderboard is reversed |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links | review-bugs: ByOwner uses substring match instead of equality; review-security: ByOwner uses substring match instead of equality, leaks cros |
| C12 | — | Extend re-dates an already-expired link, resurrecting a code the comme |  |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas hands out the internal map, not a snapshot |
| C15 | YES | Prune counts removals but returns the number of links left | review-bugs: Prune returns the post-prune size, not the number removed |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes resp.Body |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF in /preview: server fetches any URL a creator registers |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code but preview does not |
| C17 | — | deleteLink never checks the owner its comment promises to check: any c |  |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe | review-security: Open redirect on GET /out |
| C19 | — | the admin endpoint is gated on a request header the caller sets themse |  |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero when no links exist |
| C21 | YES | the export handler returns the server's filesystem path and raw error  | review-security: Internal export directory leaked in error response |
| C22 | — | writeJSON calls WriteHeader before setting headers, so Content-Type an |  |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: CORS misconfiguration: Access-Control-Allow-Origin: * with A |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | YES | log.Fatal exits the process, so the deferred janitor stop never runs - | review-concurrency: Stop is deferred but cancel never propagates to in-flight re |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens minted with math/rand (predictable) |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session tokens written to logs in plaintext |
| S04 | — | Verify never looks at ExpiresAt, so a token works forever despite the  |  |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Passwords stored as unsalted single-round SHA-256 |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho | review-bugs: BearerToken panics on a missing or malformed Authorization h; review-security: BearerToken panics on missing Authorization header (DoS) |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin always rejects, including admins |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Bearer token echoed back to the client in 401 response |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions is a concurrent map with no synchroni |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Invalid LINKD_TIMEOUT_MS silently sets timeout to 0 |
| CF03 | — | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim |  |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed but never assigned |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate collects problems and returns nil anyway |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-bugs: ExportCSV path is controllable by the caller and escapes dir; review-security: Path traversal in ExportCSV via user-controlled name |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Export file written world-writable (0666) |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll ignores WriteString errors and accumulates defers |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores os.Rename error and leaks the temp fil |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-bugs: Get deletes from the map while holding only the read lock; review-concurrency: Get mutates the map and an entry under a read lock |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo | review-bugs: evict calls Get while holding the write lock and deadlocks |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: Delete leaks its mutex and deadlocks the cache; review-concurrency: Stale TTL timer deletes the replacement entry |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-security: mustHead returns nil on error, panics in client.Do |
| X01 | — | AddResolved has a value receiver; the mutex is copied and the incremen |  |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Janitor mutates Store.links without any lock while handlers ; review-concurrency: l.Hits++ is a data race |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: wg.Add inside the goroutine lets WarmCache return before any; review-concurrency: wg.Add called after goroutine launch races with wg.Wait |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-bugs: CheckTargets leaks a goroutine on every early-return error; review-concurrency: CheckTargets leaks goroutines on the first failure |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) worker.go:86 — SSRF via CheckTargets / WarmCache against attacker-controlled URLs
