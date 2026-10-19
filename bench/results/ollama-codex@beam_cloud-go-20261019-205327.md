# ollama-codex@beam:cloud · go · run 20261019-205327 (repeat 1)

recall **40/56** · 62 finding(s), 3 unmatched · 953181 tokens · 114s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve deletes live links and keeps expired ones |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates from twice and never validates to |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page slices one past the end |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate integer division yields 0 |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete does not refund quota |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives expired links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the live internal map |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes resp.Body |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Preview endpoint performs server-side fetch of arbitrary tar |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-bugs: deleteLink ignores ownership; review-security: deleteLink performs no ownership check |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin endpoint trusts client-supplied X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero with no links |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Permissive CORS with credentials enabled |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded development secret fallback |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Issued session token written to logs |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never enforces session expiry; review-security: Session expiry is never enforced |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Unsalted SHA-256 password hashing |
| S07 | — | the admin check is always true, so RequireAdmin rejects every session  |  |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Token echoed back in 401 response |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map is shared across goroutines witho |
| CF01 | YES | a malformed LINKD_PORT silently becomes port 0 | review-bugs: LINKD_PORT parse failure overwrites the default with zero |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: LINKD_TIMEOUT_MS parse failure zeroes the timeout |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS interpreted as seconds instead of millise |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in export filename |
| EX02 | YES | the export holding every owner's links is written world-readable | review-bugs: ExportCSV writes reports with mode 0666; review-security: Exported report written world-writable |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-security: CSV field injection via unescaped records |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers file closes inside the loop |
| EX06 | — | the rename's error is discarded and /tmp is usually another filesystem |  |
| EX07 | YES | nothing fsyncs before the rename the comment calls durable, and the te | review-bugs: WriteSnapshot ignores close and rename failures |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates the map and counter while holding only an  |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek reads the map without any lock |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-concurrency: Cache.Delete returns without unlocking on the missing-key pa |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Delete does not stop the scheduled expiry timer; review-concurrency: Cache.Set replaces an entry's timer without stopping the pre |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: Cache.Stats reads c.hits without synchronization |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: createLink trusts attacker-supplied owner |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns nil, causing CheckTargets to panic |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: Metrics.AddResolved uses a value receiver; review-concurrency: Metrics.AddResolved is a value receiver, so its mutex does n |

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
- (high) auth.go:84 — RequireAdmin rejects everyone
- (high) auth.go:99 — Middleware never exposes admin status to handlers
- (medium) main.go:91 — preview does not normalize code case
