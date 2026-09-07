# gpt-5.6-terra · cpp · run 20260907-144247 (repeat 1)

recall **35/64** · 47 finding(s), 5 unmatched · 397385 tokens · 489s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving an active link deletes it; review-security: Expired links continue to redirect |
| P03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Page construction uses an iterator past vector end; review-security: Report pagination constructs an out-of-bounds vector range |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Success rate is zero unless every link was followed |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Deleting a link does not release its owner's quota |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard is ordered least-followed first |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup includes partial owner-name matches |
| P10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Batch import is not atomic |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Pruning increments an iterator after erasing it |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a view to a destroyed local string |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated with a predictable PRNG |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer session tokens are written to logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Expired sessions remain valid indefinitely |
| P21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Authorization header can overflow a stack buffer |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Missing or malformed Authorization headers cause undefined b; review-security: Missing or malformed Authorization headers can crash the ser |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache entries leak on destruction and removal |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Eviction does not select the least recently used entry |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache sweep increments an invalidated iterator |
| P31 | — | parallel: C05 utilization divides size_t by size_t before scaling so i |  |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond duration configuration is interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Invalid configurations are accepted after being reported |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report names permit path traversal and file overwrite |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: Snapshot is renamed before its stream is flushed or closed; review-concurrency: Parallel snapshots share one fixed temporary file |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to expired stack storage |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics errors counter is uninitialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Request metrics are updated with unsynchronized plain intege |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-concurrency: Janitor concurrently mutates the store map without synchroni |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache terminates the process with joinable threads; review-concurrency: warm_cache destroys joinable worker threads |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: Target checking reads one element past the input |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Missing required query parameters escape as exceptions |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete another owner's link |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Outbound redirect accepts arbitrary destinations |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is controlled by a client header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Stats divides by zero for an empty store; review-security: Authenticated users can trigger a divide-by-zero denial of s |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exposes links belonging to every user |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Unnamed reports read a different filename than they write |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.cpp:37 — A zero-sized cache erases end() on every insertion
- (high) store.cpp:59 — Rename reads a Link after erasing its map element
- (high) main.cpp:152 — The service exits immediately after startup
- (high) cache.cpp:37 — Concurrent cache writers race on entries_
- (medium) export.cpp:26 — Concurrent report requests overwrite one another's file
