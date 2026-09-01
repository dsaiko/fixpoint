# codex · go · run summaries (repeat 1)

recall **39/56** · 58 finding(s), 1 unmatched · 233705 tokens · 599s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Link expiry logic is inverted |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the old code twice |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Partial final pages slice past the end |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate truncates every partial success to zero |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Deleting a link never releases its quota count |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives already-expired links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import mutates the store before the batch is validated |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas exposes the store's mutable internal map |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Malformed TTL values silently become zero |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: Preview leaks HTTP response bodies; review-concurrency: Preview probes outlive cancellation of their incoming reques |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Preview endpoint permits SSRF and returns internal responses |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: Redirect lowercases case-sensitive generated codes |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: Any authenticated user can delete another user's link |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Client-controlled header grants administrator access |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: Stats panics when the store is empty |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: JSON response headers are set after they are committed |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | YES | ListenAndServe with the default server: no read, write or header timeo | review-bugs: The running server cannot have any valid sessions; review-security: HTTP server has no defensive connection timeouts |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens use a non-cryptographic random generator |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Live bearer tokens are written to application logs |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-security: Expired session tokens remain valid indefinitely |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | — | the admin check is always true, so RequireAdmin rejects every session  |  |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator session map is unsynchronized |
| CF01 | YES | a malformed LINKD_PORT silently becomes port 0 | review-bugs: An invalid port silently selects port zero |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Invalid timeout input overwrites the default |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is ignored |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always accepts invalid configurations |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Report name enables path traversal and arbitrary CSV overwri |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: All-owner exports are created with permissive file modes |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV fields are written without CSV escaping; review-security: Unescaped user data permits CSV formula and row injection |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll keeps every output file open until completion |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores failures that leave no snapshot |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache Get mutates state while holding only a read lock |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Deleted cache entries retain their timers forever; review-concurrency: Deleting a missing cache entry permanently retains the mutex |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | — | the link owner is taken from the request body instead of the authentic |  |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: Link listing exposes every tenant's records; review-security: Report endpoint exports every tenant's links |
| C30 | YES | Create never checks whether the generated code is already in use, so a | review-bugs: Generated code collisions overwrite live links |
| X05 | YES | the janitor expires links without touching the quota map, so an owner' | review-bugs: Automatic expiry leaves quota counts stale |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | YES | ArchiveAll discards every WriteString error and still returns nil, so  | review-bugs: ArchiveAll reports success after failed writes |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: Invalid target URLs can produce a nil HTTP request |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: Resolved metrics are incremented only on a copy; review-concurrency: AddResolved copies a live mutex instead of locking shared me |

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
- (medium) export.go:26 — Concurrent reports race through the same output file
