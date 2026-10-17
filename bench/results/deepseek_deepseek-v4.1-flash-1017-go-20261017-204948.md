# deepseek/deepseek-v4.1-flash-1017 · go · run 20261017-204948 (repeat 1)

recall **37/56** · 60 finding(s), 1 unmatched · 441622 tokens · 1776s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve has an inverted expiry check: live links are deleted |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page clamps the end index to len+1 and panics for any short  |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate uses integer division and reports 0 for every pa |
| C09 | — | Delete drops the link but never returns the owner's quota slot the doc |  |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives already-expired links, contradicting its docu |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import is not atomic despite its contract |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns the live internal map instead of a snapshot |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: A non-numeric ttl is silently accepted as zero |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes resp.Body; review-concurrency: preview ignores the request context, so an abandoned preview |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF: /preview fetches any attacker-supplied URL and returns |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases the code, so nearly every generated code |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: deleteLink deletes any link regardless of owner (IDOR) |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin endpoint authorizes on a client-controlled X-Admin hea |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: stats divides by zero when the store is empty |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: writeJSON sets Content-Type and CORS headers after WriteHead |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded fallback secret silently used when LINKD_SECRET is |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens are generated with math/rand |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Raw session tokens are written to the log |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-bugs: Verify ignores Session.ExpiresAt, so tokens never expire; review-security: Verify never checks ExpiresAt, so session tokens never expir |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin's condition is always true and rejects every ro |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map is mutated and read with no synch |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: A malformed LINKD_TIMEOUT_MS silently sets the timeout to ze |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS is treated as seconds, making the TTL 100 |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Config.Validate logs problems but always returns nil |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Path traversal in the export name gives arbitrary file write |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Export file is written 0666 despite the comment requiring se |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV fields are written without quoting or escaping; review-security: Unescaped CSV fields allow formula injection |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll defers Close inside a loop, leaking one file desc |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot ignores the rename error, so the snapshot is s |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get deletes from the map and increments hits while hol |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Peek and Stats read cache state without the mutex |
| CA03 | YES | Delete returns without unlocking when the code is absent, deadlocking  | review-bugs: Delete returns with the write lock still held when the code ; review-concurrency: Cache.Delete returns without unlocking on the not-found path |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: The timers map grows without bound; Delete never removes ent; review-concurrency: Per-entry timers are never stopped or removed, and a stale t |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | YES | the link owner is taken from the request body instead of the authentic | review-security: Link owner is taken from the request body, not the authentic |
| C29 | — | listing, stats and report all return every owner's links to any authen |  |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil request on error, which client.Do pan |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-concurrency: Metrics.AddResolved uses a value receiver: copies the mutex  |

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
- (medium) auth.go:84 — RequireAdmin's condition is unsatisfiable, so the role check always denies
