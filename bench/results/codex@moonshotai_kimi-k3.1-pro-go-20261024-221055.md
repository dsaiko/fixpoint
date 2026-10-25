# codex@moonshotai/kimi-k3.1-pro · go · run 20261024-221055 (repeat 1)

recall **36/56** · 59 finding(s), 4 unmatched · 683923 tokens · 387s · 11 retired seed(s) not scored

| seed | found | note | matched by |
|---|---|---|---|
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve deletes live links and serves expired links; review-security: Reversed expiry check allows expired links to resolve |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Partial final pages panic |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate truncates nearly every nonzero percentage to zer |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete never restores the owner's quota |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend revives expired links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import partially applies invalid batches |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas exposes mutable store state as a snapshot |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid TTL silently creates an immediately expired link |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: Preview leaks every successful response body; review-security: Preview leaks outbound response bodies |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: Preview provides an authenticated SSRF read primitive |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: Redirect lowercases case-sensitive generated codes |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-security: Any authenticated user can delete another owner's links |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-security: Admin authorization trusts a caller-controlled header |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: Stats panics when no links are returned |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | YES | writeJSON calls WriteHeader before setting headers, so Content-Type an | review-bugs: JSON Content-Type is set after headers are committed |
| C23 | — | wildcard CORS combined with Allow-Credentials on every JSON response |  |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | YES | ListenAndServe with the default server: no read, write or header timeo | review-security: HTTP server has no Slowloris protections |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | — | the signing secret falls back to a hardcoded development value when th |  |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-security: Session tokens use a predictable non-cryptographic PRNG |
| S03 | YES | every issued session token is written to the log in cleartext | review-security: Authentication tokens are written to logs |
| S04 | YES | Verify never looks at ExpiresAt, so a token works forever despite the  | review-security: Expired sessions remain valid indefinitely |
| S05 | — | passwords are stored as a single unsalted SHA-256, crackable offline a |  |
| S07 | — | the admin check is always true, so RequireAdmin rejects every session  |  |
| S08 | — | the 401 response echoes the rejected token back to the caller |  |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Session issuance, verification, and revocation race on a map |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | YES | the parse-error branch logs 'keeping default' and then assigns the fai | review-bugs: Invalid timeout overrides the default with zero |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: Fetch limit configuration is discarded |
| CF06 | — | Validate logs the problems it finds and returns nil, so a bad config s |  |
| EX01 | YES | the report name comes from the query string and goes into filepath.Joi | review-security: Report names permit directory traversal and arbitrary CSV fi |
| EX02 | YES | the export holding every owner's links is written world-readable | review-security: Cross-tenant reports are created with permissive file modes |
| EX03 | YES | CSV rows are built with Sprintf, so a comma or quote in a target break | review-bugs: CSV exports corrupt fields containing special characters; review-security: Unescaped user fields enable spreadsheet formula injection |
| EX04 | YES | defer inside the loop holds every file open until ArchiveAll returns | review-bugs: ArchiveAll leaks files and ignores failed writes |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot reports success even when no snapshot was inst |
| EX07 | YES | nothing fsyncs before the rename the comment calls durable, and the te | review-bugs: Snapshot errors leak temporary files and descriptors |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-concurrency: Cache.Get mutates entries and the map under a read lock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-concurrency: Peek and Stats race with cache mutations |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-concurrency: A stale expiration timer can delete a replacement entry |
| CA06 | — | Warm's Peek and Set are separately locked, so the janitor and a handle |  |
| CA07 | — | Stats reads the hits counter with no lock while Get is incrementing it |  |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C28 | — | the link owner is taken from the request body instead of the authentic |  |
| C29 | YES | listing, stats and report all return every owner's links to any authen | review-security: Listing and reporting expose all tenants' links |
| C30 | — | Create never checks whether the generated code is already in use, so a |  |
| X05 | — | the janitor expires links without touching the quota map, so an owner' |  |
| X06 | — | WarmCache and CheckTargets take no context, so their in-flight probes  |  |
| EX08 | — | two concurrent /report requests with the same name write the same path |  |
| EX09 | — | ArchiveAll discards every WriteString error and still returns nil, so  |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: Invalid target URLs cause a nil request panic |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved locks a copied mutex and loses increments; review-concurrency: AddResolved copies the metrics mutex and discards the increm |

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
- (high) cache.go:91 — Deleting a missing cache entry leaves the mutex locked
- (medium) config.go:116 — Validate accepts every invalid configuration
- (high) cache.go:91 — Cache.Delete leaves the mutex locked when an entry is absent
- (medium) main.go:96 — Preview request outlives its caller cancellation
