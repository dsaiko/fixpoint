# deepseek-v4-flash:0731-cloud · cpp · run 20260902-090006 (repeat 1)

recall **32/64** · 47 finding(s), 4 unmatched · 857042 tokens · 2404s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry test: live links are erased, ex; review-security: Store::resolve treats 'not yet expired' as expired: it delet |
| P03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() builds an end iterator one past the end of the vector; review-security: Store::page constructs a vector from a one-past-the-end iter |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() divides before multiplying, so the percentage |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() does not refund the owner's quota slot as its comme |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, so it returns the least-followed link |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches a substring, not the exact owner |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives links the code promises are permanently dea |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() erases while iterating (UB) and returns the remainin |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a string_view into a destroyed local |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are minted from std::rand(), a predictable un |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Freshly minted session tokens are written to stdout, and unk |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never enforces session expiry; review-security: verify() never checks expires_at, so session tokens minted w |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() is a tautology and always returns false |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | — | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 |  |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token() dereferences nullptr when the Authorization h; review-security: bearer_token copies an unbounded header into a 64-byte stack |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache leaks every Entry allocation (set, eviction, erase, sw |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | — | parallel: CA08 eviction drops the first key in map order, not the leas |  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep() erases entries while iterating with an invalidated i |
| P31 | — | parallel: C05 utilization divides size_t by size_t before scaling so i |  |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS / LINKD_CACHE_TTL_MS values are treated as  |
| P36 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t |  |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() never fails; bad configurations still boot; review-security: validate() reports config problems but always returns true,  |
| P39 | — | parallel: EX01 the report name comes from the query string and is conc |  |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-security: CSV report fields are written unescaped/unquoted, enabling s |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all writes files named after the attacker-controlled |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-security: Snapshot written to a fixed world-accessible /tmp path: syml |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns a pointer to a local stack buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is never initialized |
| P46 | — | parallel: X01 the counters are plain longs incremented from the janito |  |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor erases from std::map while iterating with the same i; review-concurrency: Janitor thread mutates Store and Cache with no synchronizati |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-concurrency: warm_cache leaks joinable threads, races on the cache, and c; review-concurrency: stop_janitor never joins the janitor thread, so shutdown rac |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() reads one element past the end; review-security: check_targets reads links[size()] out of bounds due to <= in |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Uncaught std::out_of_range on missing query params crashes t; review-security: Missing query parameters throw uncaught std::out_of_range, t |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | — | parallel: C17 delete_link never checks the owner its comment promises  |  |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out is an unvalidated open redirect |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin route is gated only by a client-settable x-admin heade |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: GET /stats divides by zero store size; review-security: stats() divides by zero when the store is empty |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Every response sends Access-Control-Allow-Origin: * together |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with an empty name writes links.csv but reads back . |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) store.cpp:75 — import_all() is documented as atomic but commits partial batches on failure
- (medium) store.cpp:58 — rename() holds a reference into a map node that is then erased (use-after-free)
- (high) main.cpp:115 — report endpoint passes an unvalidated name into filesystem paths, giving path-traversal wr
- (low) store.cpp:11 — create() validates only the URL prefix, allowing CR/LF in the target to smuggle headers vi
