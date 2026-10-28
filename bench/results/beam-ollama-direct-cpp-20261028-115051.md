# beam-ollama-direct · cpp · run 20261028-115051 (repeat 1)

recall **29/64** · 36 finding(s), 3 unmatched · 1107726 tokens · 226s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve has inverted expiry condition and uses an erased ite |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates `from` twice instead of validating `to` |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() clamps end to size()+1, reading past the vector end |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate integer division truncates to 0 unless all link |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() does not return the owner's quota slot |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending and resizes without guarding n > size |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches substrings instead of exact owners |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links despite contract |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: quotas() returns a mutable reference to internal state; review-concurrency: Store::quotas() exposes the internal map by reference |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic and partially commits on failure |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() erases during iteration and returns the wrong value |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a dangling view and is not the newest |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | — | parallel: S02 session tokens come from std::rand, never seeded, writte |  |
| P19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify does not enforce session expiry |
| P21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | — | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 |  |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token overflows fixed buffer and dereferences possibl |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator::sessions_ map is accessed without synchroniza |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache leaks heap-allocated Entry objects |
| P29 | — | parallel: CA08 eviction drops the first key in map order, not the leas |  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep erases during iteration, invalidating iterators |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization integer division always reports 0 until full |
| P32 | YES | parallel: CA02 the cache has no mutex at all, while the janitor thread | review-concurrency: Cache shared state (entries_ and hits_) is accessed without  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: LINKD_TIMEOUT_MS and LINKD_CACHE_TTL_MS are treated as secon |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() returns true even when problems are found |
| P39 | — | parallel: EX01 the report name comes from the query string and is conc |  |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | — | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul |  |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns dangling pointer to stack buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is uninitialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are incremented non-atomically from multipl |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: janitor erases from the store map while iterating it; review-concurrency: Janitor thread operates on store/cache without synchronizati |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache spawns joinable threads but never joins them; review-concurrency: stop_janitor never joins the janitor thread |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads one past the end of links |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | — | parallel: C17 delete_link never checks the owner its comment promises  |  |
| P56 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| P57 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by zero when the store is empty |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.cpp:49 — require_admin always rejects everyone
- (high) store.cpp:10 — Store maps links_ and quota_ are mutated without synchronization
- (high) linkd.h:43 — Store::all() leaks a mutable reference to the internal links_ map
