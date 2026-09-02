# nemotron-3-ultra:cloud · cpp · run 20260902-095440 (repeat 1)

recall **47/64** · 72 finding(s), 6 unmatched · 1032490 tokens · 2360s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Link codes generated with std::rand (predictable) |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes valid links and returns expired ones; review-security: Logic bug in resolve deletes valid links, keeps expired ones |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-security: validate_code called twice on 'from', not called on 'to' |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() off-by-one in end calculation |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() returns 0 due to integer division |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() does not decrement owner quota |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending but should be descending |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() uses substring match instead of exact match; review-security: Authorization bypass in by_owner via substring match |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links contrary to comment; review-security: extend() allows reviving expired links |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-security: quotas() returns non-const reference to internal map |
| P12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-concurrency: Store::prune() has iterator invalidation when erasing during; review-security: Iterator invalidation and wrong return value in Store::prune |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() has iterator invalidation and returns wrong count |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns dangling string_view to local variable; review-concurrency: Store::last_code() returns dangling string_view to local var |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret 'dev-secret-do-not-use' |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens generated with std::rand (predictable) |
| P19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() does not check session expiration |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin logic always returns false; review-security: require_admin logic bug always denies admin access |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Password hashing uses insecure djb2 algorithm |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Buffer overflow in bearer_token |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token() crashes on malformed/missing Authorization he |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals is not constant-time (timing side-channel) |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: set() leaks memory on eviction |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Destructor leaks all Entry objects |
| P29 | — | parallel: CA08 eviction drops the first key in map order, not the leas |  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep() has iterator invalidation; review-security: Iterator invalidation in Cache::sweep |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() returns 0 due to integer division order; review-security: Integer division bug in Cache::utilization |
| P32 | YES | parallel: CA02 the cache has no mutex at all, while the janitor thread | review-concurrency: Cache class has no synchronization for concurrent access to  |
| P33 | YES | parallel: CF01 atoi cannot report failure and returns 0, so a malforme | review-security: atoi error handling ambiguity for timeout/ttl/size |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT parsed but never stored; review-security: LINKD_FETCH_LIMIT parsed but ignored |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true even when validation fails; review-security: validate() returns true even when config is invalid |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal in export_csv via user-controlled name |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Path traversal in archive_all via owner name |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-security: Predictable temp file path in write_snapshot (TOCTOU risk) |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns dangling pointer to stack array |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is uninitialized; review-concurrency: Metrics class counters are not atomic and have no synchroniz |
| P46 | — | parallel: X01 the counters are plain longs incremented from the janito |  |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor thread has iterator invalidation and data race; review-concurrency: Janitor thread has iterator invalidation and data race on st |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache() destroys joinable threads causing std::terminat; review-concurrency: warm_cache() spawns threads but never joins them — thread le |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() off-by-one reads out of bounds; review-security: Out-of-bounds access in check_targets |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-security: count_for_owner modifies quotas map via operator[] |
| P52 | YES | CPP-ONLY the janitor thread is heap-allocated, never joined and never  | review-concurrency: Janitor thread is never joined — thread leak on shutdown |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-bugs: delete_link() does not verify ownership; review-security: Missing authorization check in delete_link |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Trivial admin bypass via x-admin header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Division by zero in stats() when store is empty; review-security: Division by zero in stats endpoint |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-security: Case-insensitive code lookup mismatch in redirect |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Invalid CORS configuration allows credentials with wildcard  |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Token leaked in error response |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Missing Authorization header crashes bearer_token() |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) store.cpp:49 — Store class has no synchronization for concurrent access to links_ and quota_ maps
- (high) auth.cpp:75 — Authenticator class has no synchronization for concurrent access to sessions_ map
- (critical) export.cpp:63 — Path traversal in read_report via user-controlled name
- (medium) store.cpp:145 — import_all lacks validation and duplicate checking
- (medium) linkd.h:43 — Store::all() returns non-const reference to internal map
- (medium) auth.cpp:37 — Linear search in Authenticator::verify enables DoS
