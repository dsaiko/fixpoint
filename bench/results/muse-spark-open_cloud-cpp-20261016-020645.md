# muse-spark-open:cloud · cpp · run 20261016-020645 (repeat 1)

recall **17/64** · 18 finding(s), 0 unmatched · 183059 tokens · 89s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted — deletes valid links |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice, never validates 'to' |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() end computed as size+1 causes out-of-range iterator |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() integer division always yields 0 or 100 |
| P06 | — | parallel: C09 remove drops the link but never returns the owner's quot |  |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | — | parallel: C11 by_owner matches owners by substring, so one owner sees  |  |
| P10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() iterator invalidation and returns wrong value |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns dangling string_view to local |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | — | parallel: S02 session tokens come from std::rand, never seeded, writte |  |
| P19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| P20 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() uses || instead of && — always denies |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | — | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 |  |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token() null-dereferences when header has no space |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | — | parallel: CA08 eviction drops the first key in map order, not the leas |  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep() iterator invalidation on erase |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() integer division always yields 0 or 100 |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS treated as seconds — 1000x error; review-bugs: LINKD_CACHE_TTL_MS treated as seconds — 1000x error |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT parsed but never stored |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when problems found |
| P39 | — | parallel: EX01 the report name comes from the query string and is conc |  |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | — | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul |  |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns dangling pointer to stack array |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ never initialized |
| P46 | — | parallel: X01 the counters are plain longs incremented from the janito |  |
| P47 | — | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th |  |
| P48 | — | CPP-ONLY warm_cache never joins its threads, so the vector's destructo |  |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() off-by-one reads past vector end |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | — | parallel: C17 delete_link never checks the owner its comment promises  |  |
| P56 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| P57 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by zero when store is empty |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |
