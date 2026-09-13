# deepseek-v4.1-flash:cloud · go · run 20260913-222133 (repeat 1)

recall **35/56** · 56 finding(s), 1 unmatched · 186162 tokens · 270s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve's expiry test is inverted, so live links are deleted |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page sets end to len(all)+1, panicking with slice bounds out |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate uses integer division before multiplying, so it  |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete never gives the owner's quota slot back |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives expired links, contradicting its own document |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic and never updates the quota map |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink ignores the Atoi error on ttl, silently creating  |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes resp.Body, leaking a connection per req |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF via /preview: attacker-controlled target fetched server |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lower-cases the code while stored codes are mixed-c |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink performs no ownership check: any user can delete  |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin authorization is a client-supplied header (X-Admin: tr |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by len(links) with no zero guard |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader, so they are discar |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: CORS: wildcard origin combined with Allow-Credentials, and h |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand, not a CSPRNG |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session tokens written to the log in plaintext |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks ExpiresAt, so expired sessions keep work; review-security: Verify never checks ExpiresAt: sessions never expire |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin's condition is a tautology, so it always denies |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions is read on every request and written  |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-security: A malformed LINKD_TIMEOUT_MS silently disables the HTTP clie |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is documented in milliseconds but applied |
| CF04 | — | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate collects problems and then always returns nil |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in /report name: arbitrary file write, then s |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Export file written 0666 despite the comment saying only the |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | — | the rename's error is discarded and /tmp is usually another filesystem |  |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates the map and counters while holding only a  |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek reads the map with no lock, and Warm's Peek-then- |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-concurrency: Cache.Delete returns while still holding the write lock, dea |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Timers are stored but never stopped, and a stale timer delet; review-concurrency: Set overwrites timer handles without stopping the old timer, |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: Cache.Stats reads the hit counter with no lock |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | — | the link owner is taken from the request body instead of the authentic |  |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: listLinks returns every owner's links to any authenticated u |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil *http.Request on error, and the calle |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver, so the resolved counter is; review-concurrency: Metrics.AddResolved has a value receiver, copying the mutex  |

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
- (medium) auth.go:84 — RequireAdmin's condition is unsatisfiable, and no handler calls it
