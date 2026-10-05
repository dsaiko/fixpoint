# vllm@bottlecapai/ThinkingCap-Qwen3.8-27B · go · run 20261005-021657 (repeat 1)

recall **37/56** · 66 finding(s), 0 unmatched · 67460 tokens · 1134s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve expiry condition inverted: live links deleted on acc |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice and never validates 'to' |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page panics with slice out of range when the page overruns t |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate integer division yields only 0 or 100 |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete never returns the owner's quota slot |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives expired links despite 'expiry is final' |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic — earlier links are stored before a lat |
| C14 | — | Quotas hands out the store's own map; a caller mutating it rewrites th |  |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Non-numeric TTL silently becomes 0, creating an instantly-ex |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes resp.Body — connection leak per request |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: preview handler fetches an attacker-controlled target (SSRF) |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code but codes are stored mixed-case |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-bugs: deleteLink never enforces the documented owner-only rule; review-security: GET /links returns every user's links with no owner filter |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-bugs: adminQuotas trusts a client-supplied X-Admin header; review-security: /admin/quotas authorizes on a client-controlled X-Admin head |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero on an empty store |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets headers after WriteHeader, so they are droppe |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: CORS sets a wildcard origin together with credentials |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: session tokens minted with math/rand instead of crypto/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: full session token written to logs |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks ExpiresAt, so sessions never expire |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin condition is always true — rejects even admins; review-security: RequireAdmin condition is always true (always denies) |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions is an unsynchronized map shared acros |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Bad LINKD_TIMEOUT_MS yields Timeout=0 instead of keeping the; review-security: invalid LINKD_TIMEOUT_MS silently sets the timeout to 0 |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is documented in milliseconds but applied; review-security: LINKD_CACHE_TTL_MS is treated as seconds, not milliseconds |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate always returns nil, so it never refuses a bad confi; review-security: Config.Validate always returns nil (and is never called) |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-bugs: ExportCSV does not confine the user-supplied name to dir; review-security: ExportCSV writes to a user-controlled path (traversal) |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: ExportCSV writes the report world-readable (0666) |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the os.Rename error and reports succes; review-security: WriteSnapshot ignores the rename error and skips Sync |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-bugs: Get mutates the shared map and fields under RLock; review-concurrency: Get mutates the map and counters while holding only RLock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Peek reads the map with no lock at all |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-concurrency: Delete returns while still holding the write lock — permanen |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Set leaks and resurrects-deletes timers for re-cached codes; review-concurrency: Set overwrites and Delete orphans entries in c.timers withou |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: Stats reads c.hits with no synchronization while Get increme |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-bugs: createLink trusts owner from the request body instead of the; review-security: createLink takes owner from the request body, not the sessio |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | YES | ArchiveAll discards every WriteString error and still returns nil, so  | review-bugs: ArchiveAll: owner names collide after filepath.Base, truncat |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil *http.Request on error, causing clien |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver — the increment is lost |

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
