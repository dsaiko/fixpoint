# ollama-direct@inkling-2-small:cloud · go · run 20261123-200657 (repeat 1)

recall **44/56** · 81 finding(s), 0 unmatched · 462002 tokens · 1070s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve treats live links as expired and deletes them; review-security: Expiry check inverted: live links are deleted on resolve, ex |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to`; review-security: Rename validates 'from' twice and never validates 'to' |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page() slice-out-of-range panic on partial last page; review-security: Page() slices with end = len+1; any page past the end panics |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate integer division always yields 0 or 100 |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete never releases the owner's quota slot |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend resurrects expired links, violating the finality of e; review-security: Extend revives expired links, contradicting the 'expiry is f |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic despite its contract; review-security: Import claims atomicity and validates codes but not targets, |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the store's internal map, not a snapshot; review-concurrency: Quotas returns the live internal quota map, not a snapshot |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink ignores TTL parse errors |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes resp.Body; review-security: SSRF via /preview: server fetches arbitrary stored URLs and  |
| C27 | — | preview fetches any registered target server-side and returns its body |  |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code, so mixed-case codes never reso |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink performs no ownership check; any authenticated us |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin authorization decided by client-controlled X-Admin hea |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by len(links), panicking on an empty store; review-security: Integer division by zero panics in /stats when no links exis |
| C21 | YES | the export handler returns the server's filesystem path and raw error  | review-security: Error response leaks internal export directory and filesyste |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sends headers after WriteHeader |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: writeJSON sets headers after WriteHeader; CORS allows any or |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' when LINKD |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand, not crypto/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Full session token written to the log on every issue |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks ExpiresAt, so tokens never expire; review-security: Verify never checks ExpiresAt; expired tokens remain valid f |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Passwords hashed with unsalted SHA-256 |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin rejects every role, including admin |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map is accessed without any synchroni |
| CF01 | YES | a malformed LINKD_PORT silently becomes port 0 | review-bugs: Invalid LINKD_PORT is silently accepted as port 0 |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Invalid LINKD_TIMEOUT_MS zeroes the timeout instead of keepi |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is interpreted as seconds, not millisecon |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate never returns an error; review-security: Validate() always returns nil; bad config never fails startu |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in ExportCSV writes outside the export dir; r |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Report file created world-readable/writable (0666) despite c |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV output is never escaped, so fields containing , or newli; review-security: CSV injection and no field escaping in exports |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the rename error and cross-device temp; review-security: WriteSnapshot ignores os.Rename failure (and uses /tmp + ren |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Get mutates shared state under a read lock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Peek reads the cache map with no lock at all |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Cache.Delete returns without unlocking when the key is missi; review-concurrency: Cache.Delete returns without unlocking when the key is absen |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Cache eviction deadlocks the entire cache; review-bugs: Stale timers are never stopped and outlive replaced entries |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: Cache.hits is incremented under RLock and read with no lock |
| CA08 | YES | evict drops whatever entry map iteration yields first, not the least r | review-bugs: evict does not evict the least recently used entry |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner is taken from the request body, not the session |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: GET /links returns every owner's links to any authenticated  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | YES | ArchiveAll discards every WriteString error and still returns nil, so  | review-bugs: ArchiveAll defers Close inside the loop and ignores write er |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead can pass a nil *http.Request to client.Do, panickin; review-security: mustHead returns nil request on parse failure; CheckTargets  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver; the counter never changes |

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
