# deepseek-v4-pro:cloud · cpp · run 20260901-113537 (repeat 1)

recall **38/64** · 49 finding(s), 1 unmatched · 446346 tokens · 476s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Link codes are predictable 32-bit std::rand() values, enabli |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: valid links are deleted,; review-security: resolve() inverts the expiry check: valid links are deleted, |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() sets end = all.size() + 1, reading one past the end o |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() integer division makes the result always 0 or |
| P06 | — | parallel: C09 remove drops the link but never returns the owner's quot |  |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least-followed links |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() uses substring match instead of exact owner match |
| P10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic despite the doc claim |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() erases the current iterator and returns the wrong va |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a dangling string_view into a destroyed  |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are minted with predictable std::rand() and n |
| P19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks expires_at, so sessions never expire; review-security: verify() never checks expires_at, so sessions never expire |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() condition is always true, so it always retur |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | — | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 |  |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token() dereferences a null pointer when the header h; review-security: bearer_token has a stack buffer overflow and a null-pointer  |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Data race on Authenticator::sessions_ and dangling Session*  |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache leaks every Entry it ever allocates |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: set() evicts the lexicographically-first entry, not the leas |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache::sweep() erases the current iterator and then incremen |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() uses integer division and can divide by zero |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: LINKD_TIMEOUT_MS=0 prints 'keeping default' but still sets t |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so invalid configs never fai |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report name is concatenated into a path unsanitized, allowin |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot() renames the temp file before the stream is ; review-security: write_snapshot uses a fixed, predictable temp path vulnerabl |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns a pointer to a stack buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is never initialized and is read by stats() |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Non-atomic Metrics counters incremented from concurrent requ |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: start_janitor() erases the current map iterator and then inc; review-concurrency: Data race on Store::links_ between janitor thread and reques |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache() never joins its threads, so the vector destruct; review-concurrency: warm_cache spawns threads it never joins, terminating the pr |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() iterates i <= links.size(), reading one past |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: create_link() throws uncaught std::out_of_range on missing q; review-security: Missing query parameters throw uncaught exceptions that cras |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: delete_link performs no ownership check, so any user can del |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: out() throws uncaught std::out_of_range on missing next; review-security: /out is an open redirect with no validation of ?next= |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: admin_quotas is gated by a forgeable x-admin header instead  |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by zero when the store is empty; review-security: stats() divides by store size, crashing on an empty store |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: Access-Control-Allow-Origin * with credenti |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() writes to links.csv but reads back .csv when name i |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) main.cpp:52 — redirect() throws uncaught std::out_of_range on missing code
