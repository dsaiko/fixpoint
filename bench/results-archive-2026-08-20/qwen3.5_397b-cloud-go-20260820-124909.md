# qwen3.5:397b-cloud · code · run 20260820-124909 (repeat 1)

recall **30/60** · 39 finding(s), 3 unmatched · 205501 tokens · 149s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Store.quota map never initialized causes panic |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve expiry check has inverted logic; review-security: Inverted expiry check deletes valid links |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice, never validates 'to' |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page end index calculation off-by-one |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate always returns 0 due to integer division |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete does not decrement owner quota |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow | review-bugs: Top sorts ascending instead of descending |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links | review-bugs: ByOwner uses substring match instead of exact match |
| C12 | — | Extend re-dates an already-expired link, resurrecting a code the comme |  |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C15 | YES | Prune counts removals but returns the number of links left | review-bugs: Prune returns remaining count instead of removed count |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF vulnerability in /preview endpoint |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | — | deleteLink never checks the owner its comment promises to check: any c |  |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe | review-security: Open redirect vulnerability at /out |
| C19 | — | the admin endpoint is gated on a request header the caller sets themse |  |
| C20 | — | /stats divides by the link count, panicking whenever the store is empt |  |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | — | writeJSON calls WriteHeader before setting headers, so Content-Type an |  |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: CORS misconfiguration with credentials |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded default secret |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Predictable session tokens using math/rand |
| S03 | — | every issued session token is written to the log in cleartext |  |
| S04 | — | Verify never looks at ExpiresAt, so a token works forever despite the  |  |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho | review-bugs: BearerToken panics on missing Authorization header; review-security: Panic on malformed Authorization header |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin condition always returns error; review-security: RequireAdmin logic always fails |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Session token leakage in error responses |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Data race on Authenticator.sessions map |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | — | the parse-error branch logs 'keeping default' and then assigns the fai |  |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS uses seconds instead of milliseconds |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT parsed but never assigned |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate logs problems but always returns nil |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in CSV export |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: World-readable/writable CSV export files |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll leaks file handles |
| EX06 | — | the rename's error is discarded and /tmp is usually another filesystem |  |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-bugs: Cache.Get modifies data while holding RLock; review-concurrency: Get() modifies map while holding RLock |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo | review-bugs: Cache.evict deadlocks calling Get while locked |
| CA05 | — | one timer per Set is stored and never stopped, so timers and their gor |  |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: Stats() reads hits without synchronization |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | — | AddResolved has a value receiver; the mutex is copied and the incremen |  |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store.quota map accessed without synchronization |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: WarmCache has WaitGroup race condition; review-concurrency: WaitGroup.Add called after goroutine start |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-bugs: CheckTargets leaks goroutines on early return; review-concurrency: CheckTargets goroutines leak on first error |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) export.go:75 — AppendAudit leaks file handle on write success
- (high) main.go:33 — Janitor goroutine accesses store.links without synchronization
- (medium) main.go:20 — Metrics.AddResolved has value receiver preventing proper mutex usage
