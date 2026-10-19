# minimax-m3.1-flash:cloud · go · run 20261019-170340 (repeat 1)

recall **45/56** · 89 finding(s), 2 unmatched · 384428 tokens · 2464s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve expiry check is inverted: live links are deleted, ex; review-concurrency: Store.Resolve has inverted expiry logic: live links are dele |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the from code twice instead of validating t; review-concurrency: Store.Rename validates `from` twice and never validates `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page computes an end index of len(all)+1, panicking on the l; review-concurrency: Store.Page can slice out of range and panic on a normal requ |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate truncates before multiplying, so every non-zero  |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete never gives the owner's quota slot back, contradictin; review-concurrency: Store.Delete never returns the owner's quota slot |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives an already-expired link, contradicting its ow |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic despite its doc comment, and it overwri; review-concurrency: Store.Import is not atomic despite its doc comment |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: A malformed TTL string is silently parsed as 0, producing li |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the response body of the fetched target |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF: /preview fetches any attacker-supplied URL from the se |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-bugs: deleteLink never checks the link's owner, so any user can de; review-security: deleteLink performs no ownership check, so any authenticated |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-bugs: The admin quotas endpoint is gated by a client-supplied head; review-security: Admin endpoint gated by a client-supplied X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: hits_per_link divides by the number of links, so it is zero  |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets the Content-Type after WriteHeader, so the JS |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: CORS headers allow any origin while advertising credentialed |
| C24 | YES | the create handler decodes an unbounded request body straight into mem | review-security: createLink request body is decoded without a size limit |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded default secret 'dev-secret-do-not-use' is used whe |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-concurrency: Session tokens are generated with math/rand rather than a CS; review-security: Session tokens are generated with math/rand, so they are pre |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Issued session tokens are written to the process log in clea |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-concurrency: Authenticator.Verify never checks Session.ExpiresAt; review-security: Verify does not enforce the session's ExpiresAt, so expired  |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Passwords are hashed with a bare SHA-256, with no salt and n |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin's condition is always true, so it rejects every; review-concurrency: RequireAdmin's role test is a tautology, so it always reject |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-bugs: Unauthorized responses echo the submitted token back to the ; review-security: The rejected bearer token is echoed back in the 401 response |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions is an unsynchronised map read on ever |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: An unparsable LINKD_TIMEOUT_MS still overwrites the default  |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds, 1000x off the ; review-concurrency: A bad LINKD_TIMEOUT_MS silently sets the client timeout to z |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed and then thrown away |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil and never reports the problems i; review-concurrency: Config.Validate always returns nil and never refuses a bad c |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-bugs: The report name from the request is not confined to the expo; review-security: Path traversal in ExportCSV lets a user write files anywhere |
| EX02 | YES | the export holding every owner's links is written world-readable | review-bugs: ExportCSV writes the owner-wide report world-readable, contr; review-security: Export CSV containing every owner's links is written world-r |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-security: CSV export builds rows by raw string concatenation, allowing |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers Close inside a loop, so every file stays o; review-concurrency: ArchiveAll defers f.Close() inside a loop, holding every out |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the rename error and the temp file is ; review-concurrency: WriteSnapshot renames a temp file from /tmp into dir, silent |
| EX07 | YES | nothing fsyncs before the rename the comment calls durable, and the te | review-security: Snapshot is staged in a world-writable shared directory (/tm |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-bugs: Get mutates shared state under a read lock (map delete, e.us; review-concurrency: Cache.Get mutates the map and the hit counter while holding  |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek and Cache.Stats read shared state with no lock at |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Cache.Delete returns without unlocking the mutex |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Set replaces an existing timer without stopping it, leaving ; review-concurrency: Cache.Set leaks an armed timer per overwrite and Delete neve |
| CA06 | YES | Warm's Peek and Set are separately locked, so the janitor and a handle | review-concurrency: Cache.Warm is a check-then-act on an unlocked Peek |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: Raw error text from the store is returned to the client; review-security: Link owner is taken from the request body instead of the aut |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: Unauthenticated /stats and /report leak link metadata and cr |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | YES | WarmCache and CheckTargets take no context, so their in-flight probes  | review-concurrency: Outbound probes and fetches carry no context, so client canc |
| EX08 | YES | two concurrent /report requests with the same name write the same path | review-concurrency: ExportCSV joins a user-supplied report name onto the export  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil request when the URL is malformed, an; review-concurrency: mustHead returns a nil request on error, so client.Do(nil) p |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver, so every increment is thro; review-concurrency: Metrics.AddResolved has a value receiver, so it locks a copy |

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
- (low) worker.go:16 — Metrics.errors is never incremented anywhere
- (low) config.go:23 — DefaultConfig is an exported mutable package-level variable
