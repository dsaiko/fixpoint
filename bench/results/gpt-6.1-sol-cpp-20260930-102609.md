# gpt-6.1-sol · cpp · run 20260930-102609 (repeat 1)

recall **42/64** · 65 finding(s), 12 unmatched · 263484 tokens · 744s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it; review-security: Resolving a live link deletes another owner's data |
| P03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Partial pages read beyond the vector; review-security: Report requests trigger an out-of-bounds Link copy |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Success rate reports zero for every partial success |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Owner counts are not maintained when links change |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard returns the least-followed links first |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: Leaderboard invents records when fewer than n links exist |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup uses substring matching instead of equality |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive an expir |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: Quota snapshots expose the live count map |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Batch import leaves partial changes after validation failure |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: Pruning invalidates the loop iterator |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining count instead of the removed cou |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a dangling string view |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens come from a predictable random sequence |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Authentication logs contain usable session credentials |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Expired session tokens remain authorized indefinitely |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: The administrator check rejects every role |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: Long authorization headers overflow the stack buffer; review-security: Authorization parsing permits unauthenticated stack corrupti |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Missing authorization delimiters cause undefined behavior |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache entries leak on replacement and removal |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: A zero-capacity cache erases its end iterator; review-bugs: Eviction ignores access recency |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache sweeping increments an erased iterator |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache utilization truncates before calculating the percentag |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: Invalid timeout input overwrites the default with zero |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond settings are interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: The fetch-limit environment setting is discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Invalid configurations are never rejected |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: User-controlled report names escape the export directory |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-bugs: Report export reports success after file I/O failures; review-security: Sensitive reports are created with ambient file permissions |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV fields are written without quoting or escaping; review-security: CSV exports allow attacker-controlled spreadsheet formulas |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: Snapshot replacement occurs before the data is durable |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to a destroyed stack array |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | — | parallel: X01 the counters are plain longs incremented from the janito |  |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor increments an erased iterator; review-concurrency: Janitor accesses the store without synchronization |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-concurrency: Cache warming destroys joinable threads; review-concurrency: Stopping the janitor does not wait for it to release referen |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: Target checking reads past the input vector |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: Counting an unknown owner creates a phantom owner |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link creation trusts a caller-supplied owner |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete another owner's link |
| P56 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A client-controlled header bypasses administrator authorizat |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Statistics divide by zero for an empty store; review-security: Requesting statistics for an empty store divides by zero |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: Redirects alter case-sensitive stored codes |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Unnamed reports read a different filename than they write |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) main.cpp:152 — The service exits immediately after startup
- (high) main.cpp:128 — Recoverable request errors escape the handler
- (medium) store.cpp:60 — Rename accesses a link after destroying it
- (medium) store.cpp:50 — Rename never validates the destination code
- (medium) store.cpp:176 — last_code selects the largest code instead of the newest link
- (medium) cache.cpp:37 — Updating an existing cache key unnecessarily evicts another entry
- (medium) cache.cpp:31 — peek reports expired entries as cached
- (medium) export.cpp:51 — Snapshot rename failures are silently ignored
- (medium) store.cpp:26 — Generated code collisions overwrite existing links
- (medium) main.cpp:44 — TTL conversion overflows before constructing the duration
- (medium) cache.cpp:45 — Parallel cache warming races on the entries map
- (high) main.cpp:115 — The report endpoint exposes links belonging to every owner
