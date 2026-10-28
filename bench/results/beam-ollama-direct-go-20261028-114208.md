# beam-ollama-direct · go · run 20261028-114208 (repeat 1)

recall **38/56** · 74 finding(s), 5 unmatched · 528489 tokens · 177s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve deletes valid links and keeps expired ones; review-security: Expired links resolve successfully (expiry check inverted) |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and never the target |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page slice end index can exceed slice length; review-security: Page slice index out of range (DoS) |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate integer division truncates to 0 |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete does not decrement quota |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives expired links |
| C13 | — | Import writes links as it validates, so a bad code leaves the earlier  |  |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the internal map |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink ignores TTL parse errors |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview leaks resp.Body |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Server-side request forgery in preview endpoint |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-bugs: deleteLink ignores ownership; review-security: deleteLink performs no ownership check |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-bugs: adminQuotas always forbidden because X-Admin is never set; review-security: Admin endpoint trusts client-supplied X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero when there are no links; review-security: Division by zero when no links exist |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Permissive CORS with credentials enabled |
| C24 | YES | the create handler decodes an unbounded request body straight into mem | review-security: Link owner taken from request body, enabling impersonation |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback signing/session secret |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session token written to logs |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify does not enforce ExpiresAt; review-security: Verify ignores session expiry |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Password hashing uses unsalted SHA-256 |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin condition is always true; review-security: RequireAdmin always rejects |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Unauthorized response echoes the supplied token |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator sessions map unprotected across goroutines |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Timeout parse failure silently zeroes the timeout |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: CacheTTL interpreted as seconds despite millisecond contract |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in export filename |
| EX02 | YES | the export holding every owner's links is written world-readable | review-bugs: ExportCSV writes reports world-readable; review-security: Export file written world-readable/writable |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers Close inside the loop |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot rename can fail across filesystems and error i |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-bugs: Get mutates the map under a read lock and has a data race; review-concurrency: Cache.Get mutates entries and counters under RLock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek reads the map without any lock |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Delete does not stop the scheduled timer; review-concurrency: Cache.Delete returns without unlocking on miss, and never st |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: Cache.Stats reads c.hits without synchronization |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | — | the link owner is taken from the request body instead of the authentic |  |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns nil request and callers panic; review-security: Server-side requests to attacker-controlled targets |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: Metrics.AddResolved uses a value receiver |

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
- (medium) store.go:148 — Top panics for negative n
- (medium) cache.go:79 — evict deadlocks by calling Get while holding the lock
- (medium) main.go:91 — preview does not lowercase code unlike redirect
- (low) cache.go:53 — Peek reports expired entries as present
- (medium) worker.go:74 — WarmCache fetches attacker-controlled URLs
