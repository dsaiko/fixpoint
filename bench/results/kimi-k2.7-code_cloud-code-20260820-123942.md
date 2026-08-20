# kimi-k2.7-code:cloud · code · run 20260820-123942 (repeat 1)

recall **40/60** · 55 finding(s), 1 unmatched · 214550 tokens · 247s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: quota map not initialized in NewStore; review-security: Create panics because the quota map is never initialized |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve deletes live links and serves expired ones; review-security: Resolve deletes live links |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice instead of `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page slices past the end of the result set; review-security: listLinks can panic on Page bounds |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate integer division truncates to 0% |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow | review-bugs: Top returns least-followed links first |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links | review-bugs: ByOwner matches substrings, not exact owners |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives already-expired links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the live internal map; review-concurrency: Store.Quotas returns the internal mutable quota map |
| C15 | YES | Prune counts removals but returns the number of links left | review-bugs: Prune returns remaining count instead of removed count |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: /preview performs outbound requests to arbitrary URLs |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink does not verify ownership |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe | review-bugs: /out uses a permanent 301 redirect; review-security: Open redirect in /out |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin check relies on a client-controlled header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats handler panics when there are no links |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Permissive CORS with credentials allowed |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand |
| S03 | — | every issued session token is written to the log in cleartext |  |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify ignores session expiration; review-security: Expired sessions are still accepted |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Passwords hashed with unsalted SHA-256 |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho | review-bugs: BearerToken panics on missing or malformed Authorization hea; review-security: BearerToken panics on malformed Authorization header |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin always rejects |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Unauthorized response reflects the token |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator sessions map is not synchronized |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | — | the parse-error branch logs 'keeping default' and then assigns the fai |  |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is parsed as seconds, not milliseconds |
| CF04 | — | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti |  |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Config.Validate always returns nil |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in export filename |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Exported reports are world-readable |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV export does not escape fields; review-security: CSV fields are not escaped |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers file closes inside a loop |
| EX06 | — | the rename's error is discarded and /tmp is usually another filesystem |  |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-bugs: Cache.Get mutates the map under a read lock; review-concurrency: Cache.Get mutates shared state while holding RLock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-bugs: Peek reads the map without holding the lock; review-concurrency: Cache.Peek and Cache.Stats read shared state without locking |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Cache.Delete leaks its lock when the entry is missing |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo | review-bugs: Cache.evict deadlocks by calling Get while holding the write; review-concurrency: Cache.evict deadlocks by reacquiring the mutex |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: Cache.Delete leaks the lock when the entry is missing |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved uses a value receiver, so resolved counter never; review-concurrency: Metrics.AddResolved uses a value receiver, copying the mutex |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store maps accessed without any synchronization |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: WarmCache races on WaitGroup counter; review-concurrency: WarmCache calls wg.Add inside the goroutine |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-bugs: CheckTargets leaks goroutines after the first error; review-concurrency: CheckTargets leaks goroutines on first failure |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) main.go:67 — createLink allows arbitrary owner impersonation
