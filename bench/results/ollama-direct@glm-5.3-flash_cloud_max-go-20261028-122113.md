# ollama-direct@glm-5.3-flash:cloud#max · go · run 20261028-122113 (repeat 1)

recall **33/56** · 56 finding(s), 2 unmatched · 778803 tokens · 735s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Link expiry condition is inverted; review-security: Expiration check is reversed |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice, not the destination |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page slices past the end of the slice |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate performs the operations in the wrong order |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete does not reclaim its quota slot |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend can revive an expired link |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the store map without a snapshot |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid or non-positive TTL is accepted silently |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: Preview leaks target response bodies |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Preview fetch enables server-side request forgery |
| C16 | — | redirect lowercases the code before lookup, but codes are stored case- |  |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: Any authenticated user can delete another owner's link |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin authorization is client-controlled |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: Stats divides by zero when the store is empty |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: Response headers are set after WriteHeader |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens use non-cryptographic randomness |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Bearer session token is written to logs |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Expired sessions continue to authenticate; review-security: Expired session tokens remain valid |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin rejects every role |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator sessions map is unsynchronized |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Bad timeout configuration removes the default instead of kee |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| CF04 | — | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate never returns a validation failure |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: CSV export filename permits path traversal |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Sensitive export is world-group/world-readable |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: ExportCSV does not encode CSV delimiters or newlines; review-security: CSV is generated without escaping or formula protection |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: Snapshot rename and write errors are ignored |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache mutates state while holding only a read lock |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Replacing a cache entry leaves its old TTL timer armed; review-concurrency: Cache.Delete returns without unlocking when the entry is abs |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-concurrency: Preview fetch is not request-bound and its body is never clo; review-security: Created links can be attributed to arbitrary owners |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: Listing and report endpoints expose all owners' links |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | YES | ArchiveAll discards every WriteString error and still returns nil, so  | review-bugs: ArchiveAll defers file closes inside a loop and ignores writ |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved uses a value receiver and never updates shared m |

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
- (medium) cache.go:91 — Cache.Delete keeps the mutex locked when the entry is absent
- (medium) export.go:26 — Concurrent report requests race on the same CSV path
