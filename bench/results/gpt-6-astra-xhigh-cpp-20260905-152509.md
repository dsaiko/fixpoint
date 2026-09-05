# gpt-6-astra-xhigh · cpp · run 20260905-152509 (repeat 1)

recall **45/64** · 66 finding(s), 11 unmatched · 356401 tokens · 869s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a valid link deletes it; review-security: Resolving a live link deletes it regardless of ownership |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename never validates the destination code |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: A short final page reads beyond the vector; review-security: A report request can read beyond the link vector |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Success rate reports zero unless every link was followed |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Link mutations leave owner quotas inconsistent |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard selects the least-followed links |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: Requesting more leaderboard entries creates fictitious links |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup performs substring matching |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Expired links receive a new future expiry |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: The quota snapshot exposes mutable internal state |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: A failed import leaves a partially modified store |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: Pruning increments an erased iterator |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining count instead of the removal cou |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a view into a destroyed local string |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens come from a predictable random sequence |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Authentication logs disclose usable bearer tokens |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Expired session tokens remain valid indefinitely |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: The administrator check rejects every role |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Authorization headers can overflow a stack buffer before aut |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Missing authorization headers cause invalid memory access; review-security: Missing or malformed authorization causes a pre-authenticati |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache entries are never freed |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Eviction uses code ordering instead of last access |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache sweeping increments an erased iterator; review-concurrency: Parallel warm-up mutates the cache map without synchronizati |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache utilization truncates every partial percentage to zero |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: Invalid timeout input replaces the default with zero |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond settings are interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: The fetch limit environment setting is discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Invalid configuration never prevents startup |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report names permit file disclosure and overwrites outside t |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Sensitive reports inherit potentially public filesystem perm |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV fields are written without escaping; review-security: Attacker-controlled CSV cells can execute spreadsheet formul |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: The snapshot is published before its contents are durable |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to a local array |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | — | parallel: X01 the counters are plain longs incremented from the janito |  |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: The janitor advances an invalidated iterator; review-concurrency: Janitor traverses the store while callers can modify it |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-concurrency: Cache warm-up destroys joinable threads and aborts; review-concurrency: Stopping the janitor leaves it accessing borrowed objects |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: Target checking accesses one element past the vector |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Recoverable request errors escape the handler |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link creation trusts a caller-supplied owner |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete another owner's link |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Permanent redirects bypass subsequent click counting |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A client-controlled header bypasses administrator authorizat |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Statistics divide by zero when the store is empty; review-security: Requesting statistics for an empty store triggers division b |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: Redirect lookup changes valid case-sensitive codes |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report downloads expose other users' link records |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: The default report is read from a different filename |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) store.cpp:26 — Generated code collisions overwrite existing links
- (high) store.cpp:60 — Rename uses a link after erasing its owning map entry
- (high) store.cpp:134 — Extending a link can shorten its remaining lifetime
- (medium) cache.cpp:37 — A zero cache limit causes invalid operations
- (medium) cache.cpp:37 — Updating an existing cache entry evicts another entry
- (medium) store.cpp:176 — The latest code is selected lexicographically
- (medium) main.cpp:44 — TTL conversion can overflow before constructing the duration
- (medium) main.cpp:32 — Plain-text and CSV bodies are labeled as JSON
- (medium) main.cpp:96 — The quota endpoint omits all per-owner counts
- (medium) export.cpp:26 — Report I/O failures are reported as successful downloads
- (medium) export.cpp:48 — Snapshot replacement fails across filesystems
