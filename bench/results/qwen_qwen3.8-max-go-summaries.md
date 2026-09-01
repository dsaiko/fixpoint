# qwen/qwen3.8-max · go · run summaries (repeat 1)

recall **37/56** · 54 finding(s), 0 unmatched · 51609 tokens · 1015s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve's expiry condition is inverted |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page clamps end to len(all)+1, slicing past capacity — panic |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate computes used/len before *100 — integer division |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete never decrements the owner's quota |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives expired links, contradicting its own document |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import writes links while validating, leaving a partial batc |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-concurrency: Quotas hands out the live internal quota map, which JSON enc |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: TTL parse error ignored — invalid TTL silently becomes a zer |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the fetched response body |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Full-read SSRF via /preview/{code} |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code, but codes are mixed-case base6 |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink has no ownership check — any authenticated user d; review-security: GET /links exposes every owner's links to any authenticated  |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin endpoint gated on client-supplied X-Admin header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by len(links), which is zero on an empty store |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON calls WriteHeader before setting headers, so they  |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: Wildcard CORS with credentials declared in writeJSON (inert  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens generated with math/rand, a predictable PRNG |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Session bearer tokens written to the log in plaintext |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify never checks ExpiresAt — expired sessions stay valid ; review-security: Verify never checks Session.ExpiresAt — tokens never expire |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: HashPassword uses unsalted, single-iteration SHA-256 |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin's condition is always true — it rejects everyon; review-security: RequireAdmin condition is always true — it rejects every ses |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map has no lock: Issue/Revoke writes  |
| CF01 | YES | a malformed LINKD_PORT silently becomes port 0 | review-bugs: LINKD_PORT parse error ignored — garbage value makes the ser |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Bad LINKD_TIMEOUT_MS zeroes the timeout despite 'keeping def |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is documented as milliseconds but applied |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed and validated, then discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate logs problems but always returns nil |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in /report: arbitrary file write plus full li |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Cross-owner export written world-readable/writable (0666) |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers Close inside the loop and ignores all writ |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the rename error and uses /tmp, so the |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get writes under RLock: map delete, e.used, and hits++ |
| CA02 | — | Peek reads the map with no lock at all while other goroutines are writ |  |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Delete returns while still holding the mutex when the key is |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: Delete returns without unlocking when the key is absent — pe |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: createLink trusts the owner field from the request body |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver — the resolved counter neve; review-concurrency: AddResolved has a value receiver: copies the mutex and drops |

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
