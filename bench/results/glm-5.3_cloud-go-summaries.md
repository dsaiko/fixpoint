# glm-5.3:cloud · go · run summaries (repeat 1)

recall **39/56** · 58 finding(s), 0 unmatched · 614935 tokens · 284s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve's expiry check is inverted: live links are deleted,  |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page clamps end to len(all)+1, causing an out-of-range slice |
| C05 | — | used/len*100 in integer arithmetic is 0 for every rate under 100% |  |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete never returns the quota slot despite the documented p |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend resurrects expired links, contradicting its documente |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic: a validation failure mid-batch leaves  |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas hands out the live internal map, not a snapshot |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the fetched response body, leaking conn |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF: /preview fetches attacker-chosen URLs server-side and  |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code, which is case-sensitive base64 |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink never checks ownership despite the documented own |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin endpoint gated on a client-controlled X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero on an empty store |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader, so Content-Type an |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Intended CORS policy is Access-Control-Allow-Origin * with c |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand instead of crypto/ra |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session tokens written to the log in plaintext |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks session expiry, and does a linear scan i; review-security: Session expiry is never enforced in Verify |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Password hashing is a single unsalted SHA-256 |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin's condition is always true, so it rejects every; review-security: RequireAdmin's condition is inverted — the control rejects e |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions is an unguarded map written by Issue/ |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Bad LINKD_TIMEOUT_MS zeroes the timeout despite the log sayi |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds, inflating the  |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed, validated, then discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil, so invalid configurations are a |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-bugs: ExportCSV does not confine the user-supplied name to dir; review-security: Path traversal in CSV export lets any user write an arbitrar |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Export file created world-readable and world-writable (0666) |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-security: CSV fields written unescaped: formula injection and row forg |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers Close inside the loop, holding every file  |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the Rename error and leaks the temp fi |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates map, entry, and hit counter under RLock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Cache.Peek and Cache.Stats read shared state with no lock at |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Cache.Delete returns early while holding the write lock, dea; review-concurrency: Cache.Delete early-returns without unlocking, permanently we |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Set's timers are never stopped or removed: unbounded growth ; review-concurrency: Set's per-entry timers are never cancelled: stale timer dele |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner taken from the request body, not the session |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: /report exports every owner's links to any authenticated use |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns nil on error and CheckTargets passes it to  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver, so the resolved counter ne; review-concurrency: Metrics.AddResolved has a value receiver: increments a copy  |

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
