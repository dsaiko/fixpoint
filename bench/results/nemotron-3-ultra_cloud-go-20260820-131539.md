# nemotron-3-ultra:cloud · code · run 20260820-131539 (repeat 1)

recall **41/60** · 71 finding(s), 7 unmatched · 989244 tokens · 1178s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: quota map never initialized, panics on first Create |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve logic inverted: deletes valid links, keeps expired o; review-security: Resolve expiry check is backwards - deletes valid links, kee |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice, never validates 'to' |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page has off-by-one: end = len(all) + 1 causes panic |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate integer division truncates to 0 |
| C09 | YES | Delete drops the link but never returns the owner's quota slot the doc | review-bugs: Delete does not decrement quota |
| C10 | YES | Top sorts by hits ascending, so the leaderboard shows the least follow | review-bugs: Top sorts ascending (least hits first), comment says most hi |
| C11 | YES | ByOwner matches owners by substring, so one owner sees another's links | review-bugs: ByOwner uses substring match, not exact match; review-security: ByOwner uses substring match - information leak across users |
| C12 | YES | Extend re-dates an already-expired link, resurrecting a code the comme | review-bugs: Extend allows reviving expired links |
| C13 | YES | Import writes links as it validates, so a bad code leaves the earlier  | review-bugs: Import not atomic: partial insert on validation failure |
| C14 | YES | Quotas hands out the store's own map; a caller mutating it rewrites th | review-bugs: Quotas returns internal map, caller can mutate store state; review-concurrency: Store.Quotas returns internal quota map directly — escape of |
| C15 | YES | Prune counts removals but returns the number of links left | review-bugs: Prune returns remaining count instead of removed count; review-concurrency: Store.Prune returns wrong value (len after deletion, not rem |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C27 | YES | preview fetches any registered target server-side and returns its body | review-security: SSRF in preview endpoint - fetches user-controlled URLs with |
| C16 | YES | redirect lowercases the code before lookup, but codes are stored case- | review-bugs: redirect lowercases code but store keys are case-sensitive |
| C17 | YES | deleteLink never checks the owner its comment promises to check: any c | review-bugs: deleteLink missing ownership check |
| C18 | YES | /out redirects to any address in ?next= -- an open redirect, cached pe | review-bugs: out handler open redirect: redirects to arbitrary user-suppl; review-security: Open redirect in /out endpoint - arbitrary redirect via ?nex |
| C19 | YES | the admin endpoint is gated on a request header the caller sets themse | review-bugs: adminQuotas checks X-Admin header set by client, not authent; review-security: Admin endpoint protected by spoofable X-Admin header instead |
| C20 | YES | /stats divides by the link count, panicking whenever the store is empt | review-bugs: Division by zero in stats when no links exist |
| C21 | — | the export handler returns the server's filesystem path and raw error  |  |
| C22 | — | writeJSON calls WriteHeader before setting headers, so Content-Type an |  |
| C23 | YES | wildcard CORS combined with Allow-Credentials on every JSON response | review-security: CORS misconfiguration - wildcard origin with credentials |
| C24 | — | the create handler decodes an unbounded request body straight into mem |  |
| C25 | — | ListenAndServe with the default server: no read, write or header timeo |  |
| C26 | — | log.Fatal exits the process, so the deferred janitor stop never runs - |  |
| S01 | YES | the signing secret falls back to a hardcoded development value when th | review-security: Hardcoded default secret 'dev-secret-do-not-use' used when L |
| S02 | YES | session tokens come from math/rand, so they are predictable and forgea | review-bugs: Session token uses math/rand (not crypto-secure); review-security: Session tokens generated with math/rand (not cryptographical |
| S03 | — | every issued session token is written to the log in cleartext |  |
| S04 | — | Verify never looks at ExpiresAt, so a token works forever despite the  |  |
| S05 | YES | passwords are stored as a single unsalted SHA-256, crackable offline a | review-security: Password hashing uses unsalted SHA-256 - vulnerable to rainb |
| S06 | YES | BearerToken indexes parts[1] without checking; a request with no Autho | review-bugs: BearerToken panics on missing/malformed Authorization header; review-security: BearerToken panics on missing or malformed Authorization hea |
| S07 | YES | the admin check is always true, so RequireAdmin rejects every session  | review-bugs: RequireAdmin logic always rejects everyone; review-security: RequireAdmin logic bug - always rejects even valid admins |
| S08 | YES | the 401 response echoes the rejected token back to the caller | review-security: Session token leaked in error response |
| S09 | YES | the sessions map is read and written from every request goroutine with | review-concurrency: Authenticator.sessions map accessed concurrently without syn |
| CF01 | — | a malformed LINKD_PORT silently becomes port 0 |  |
| CF02 | — | the parse-error branch logs 'keeping default' and then assigns the fai |  |
| CF03 | YES | LINKD_CACHE_TTL_MS is documented in milliseconds and multiplied by tim | review-bugs: LINKD_CACHE_TTL_MS parsed as seconds not milliseconds |
| CF04 | YES | LINKD_FETCH_LIMIT is parsed and validated, then thrown away; the setti | review-bugs: LINKD_FETCH_LIMIT parsed but discarded |
| CF05 | — | any read error -- not just a missing file -- silently yields an empty  |  |
| CF06 | YES | Validate logs the problems it finds and returns nil, so a bad config s | review-bugs: Validate returns nil even when validation fails; review-security: Config.Validate never returns error - invalid config accepte |
| EX01 | — | the report name comes from the query string and goes into filepath.Joi |  |
| EX02 | YES | the export holding every owner's links is written world-readable | review-bugs: ExportCSV writes world-readable file (0666) despite comment ; review-security: Export files written with 0666 permissions - world readable |
| EX03 | — | CSV rows are built with Sprintf, so a comma or quote in a target break |  |
| EX04 | — | defer inside the loop holds every file open until ArchiveAll returns |  |
| EX06 | YES | the rename's error is discarded and /tmp is usually another filesystem | review-bugs: WriteSnapshot rename across filesystems fails silently |
| EX07 | — | nothing fsyncs before the rename the comment calls durable, and the te |  |
| CA01 | YES | Get deletes from the map and increments hits while holding only a read | review-bugs: Get modifies map and entry under RLock (race condition); review-concurrency: Cache.Get deletes from map while holding read lock |
| CA02 | YES | Peek reads the map with no lock at all while other goroutines are writ | review-bugs: Peek reads map without any lock (race condition); review-concurrency: Cache.Peek reads map without any lock |
| CA03 | — | Delete returns without unlocking when the code is absent, deadlocking  |  |
| CA04 | YES | evict runs under the write lock and calls Get, which takes the read lo | review-bugs: evict calls Get while holding Lock, causes deadlock; review-concurrency: Cache.evict calls Get (which takes RLock) while holding writ |
| CA05 | YES | one timer per Set is stored and never stopped, so timers and their gor | review-bugs: Set creates timer that calls Delete, but Delete doesn't clea; review-concurrency: Cache.Delete leaves timer running and timer map entry orphan |
| CA06 | YES | Warm's Peek and Set are separately locked, so the janitor and a handle | review-concurrency: Cache.Warm has TOCTOU race between Peek and Set |
| CA07 | YES | Stats reads the hits counter with no lock while Get is incrementing it | review-concurrency: Cache.Stats reads hits counter without synchronization |
| CA08 | — | evict drops whatever entry map iteration yields first, not the least r |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns nil request on error, caller panics |
| X01 | — | AddResolved has a value receiver; the mutex is copied and the incremen |  |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-bugs: AddResolved uses value receiver, mutex copied, lock ineffect; review-bugs: StartJanitor accesses store.links without synchronization (d |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: WarmCache wg.Add(1) inside goroutine races with wg.Wait(); review-concurrency: WarmCache calls wg.Add(1) inside the goroutine — WaitGroup r |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-bugs: CheckTargets leaks goroutines on early error return; review-concurrency: CheckTargets leaks goroutines on early error return |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) auth.go:60 — Verify uses O(n) linear search instead of map lookup
- (medium) export.go:36 — ArchiveAll uses filepath.Base on owner, can create invalid filenames
- (medium) store.go:102 — Page returns nil for empty page instead of empty slice
- (high) worker.go:70 — SSRF in WarmCache - makes HEAD requests to user-controlled URLs
- (medium) store.go:39 — URL validation only checks scheme prefix - allows localhost, private IPs, metadata endpoin
- (low) auth.go:26 — Sessions map never cleaned up - memory leak / DoS
- (low) worker.go:90 — CheckTargets SSRF - probes user-controlled URLs, stops on first error
