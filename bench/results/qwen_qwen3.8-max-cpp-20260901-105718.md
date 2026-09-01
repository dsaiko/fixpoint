# qwen/qwen3.8-max · cpp · run 20260901-105718 (repeat 1)

recall **49/64** · 61 finding(s), 3 unmatched · 68519 tokens · 1884s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Link codes generated from std::rand() are predictable |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: deletes live links, serv |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to` |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() clamps end to size()+1 — one past the end |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() integer-divides before multiplying: always 0  |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() never returns the owner's quota slot |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the LEAST-hit links; also p |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches by substring instead of equality |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, violating the documented inv |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: quotas() returns the live internal map though documented as  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic despite the all-or-nothing contra |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune() erases through the loop iterator — undefined behavio |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining size instead of the removed co |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a string_view into a destroyed local str |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens generated with std::rand() are predictable |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to stdout/log at issue time |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Authenticator::verify() never checks session expiry; review-security: verify() never checks session expiry — tokens are valid fore |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always returns false |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Passwords hashed with unsalted djb2-style checksum |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: bearer_token() copies an arbitrary-length header into a 64-b; review-security: Stack buffer overflow in bearer_token on unauthenticated Aut |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token() crashes on any header without a space — inclu |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator::verify scans sessions_ unsynchronized and ret |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache leaks every Entry: heap objects are never deleted |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: LRU eviction evicts the alphabetically first key, not the le |
| P30 | — | CPP-ONLY sweep erases through the iterator it then increments: undefin |  |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() integer-divides before multiplying: result is  |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS is treated as seconds, and the bad-value br; review-bugs: LINKD_CACHE_TTL_MS is treated as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so a bad config never fails  |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal in export_csv via user-controlled report name |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Confidential export written with default umask permissions |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot() renames before flushing/closing and ignores; review-security: Snapshot staged at fixed predictable path in world-writable  |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns a pointer to a local stack array |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are plain longs incremented and read across |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor thread erases from the map without advancing the ite; review-concurrency: Janitor thread mutates Store and Cache without any synchroni |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache() never joins its threads — std::thread destructo; review-bugs: stop_janitor() never joins or deletes the janitor thread |
| P49 | YES | CPP-ONLY the thread lambda captures the loop variable by reference, so | review-concurrency: warm_cache spawns worker threads and never joins them — dest |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() off-by-one: reads links[links.size()] |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: create_link() lets exceptions escape and abort the request p; review-security: Uncaught exceptions from missing query parameters crash the  |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link ownership taken from query parameter instead of authent |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete any other user's link (IDO |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect at /out — Location taken verbatim from ?next= |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated by client-supplied x-admin header inste |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by store size with no empty-store check; review-security: Division by zero in stats() crashes the service when the sto |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: redirect() lowercases the code before a case-sensitive looku |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: Access-Control-Allow-Origin * with credenti |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented bearer token |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report returns every owner's links to any authenticated cal |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with no name writes links.csv but reads '.csv' |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.cpp:37 — Cache with limit 0: erase(end()) UB in set() and division by zero in utilization()
- (low) store.cpp:26 — create() silently overwrites an existing link on code collision
- (high) export.cpp:63 — Path traversal in read_report via user-controlled name (arbitrary file read)
