# glm-5.3-flash:cloud · cpp · run 20260901-135924 (repeat 1)

recall **43/64** · 60 finding(s), 3 unmatched · 733897 tokens · 301s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Link codes are unseeded std::rand() output and collisions si |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Store::resolve expiry check is inverted; review-security: Expiry check in resolve() is inverted: live links are destro |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates `from` twice and never validates the new co |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Store::page off-by-one reads one past the end; negative offs; review-security: Off-by-one in page() reads one element out of bounds |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate truncates to 0 for every store smaller than 100 |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Store::remove never returns the quota slot, contradicting it |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: Store::top returns the least-hit links and can pad with defa |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner matches owners by substring instead of equality |
| P10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic despite its documented guarantee |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Store::prune erases while iterating (UB) and returns the wro |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: Store::last_code returns a dangling string_view to a local s |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens generated from std::rand() are predictable an |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are written to stdout |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify never checks session expiry; review-security: verify() never checks session expiry, so tokens never lapse |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin can never return true |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Password 'hashing' is unsalted 32-bit djb2 |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Stack buffer overflow in bearer_token from attacker-controll |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token overflows a 64-byte stack buffer and dereferenc; review-security: Unauthenticated crash: bearer_token dereferences a null poin |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals early-returns on first mismatch despite claimi |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator::verify returns a raw pointer into sessions_ w |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Eviction picks the lexicographically smallest code, not the  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache::sweep erases from entries_ while iterating it; review-concurrency: Cache is used from the janitor thread, warm_cache workers, a |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache::utilization integer division always yields 0 (or divi |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond env vars are interpreted as whole seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true, so an invalid config is never  |
| P39 | — | parallel: EX01 the report name comes from the query string and is conc |  |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Export report containing all owners' links is written world- |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-security: CSV rows are written without escaping — field and formula in |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot renames the temp file before its data is flus; review-security: Snapshot written through a fixed, predictable /tmp path — sy |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to a dead stack buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is uninitialized; review-security: Metrics::errors_ is never initialized and its value is serve |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are non-atomic longs incremented from concu |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor erases from the store map while iterating it; review-concurrency: Janitor thread mutates the links map concurrently with reque |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache destroys joinable threads (std::terminate) and ca; review-bugs: stop_janitor never joins the janitor thread |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | — | CPP-ONLY check_targets loops with <= and reads one element past the en |  |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: count_for_owner inserts a zero entry into the store's quota ; review-concurrency: count_for_owner inserts into the shared quota map through a  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Missing query parameters throw uncaught std::out_of_range |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: No ownership check on link deletion: any authenticated user  |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out forwards to any URL supplied in ?next= |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is a client-supplied header, not the ses |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by zero on an empty store; review-security: Division by zero in /stats crashes the service when the stor |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: redirect lowercases the code, breaking links created with up |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS combined with Allow-Credentials on every respo |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() reads back a different file than it just wrote when |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) cache.cpp:20 — Every cache erase path leaks its heap-allocated Entry
- (medium) store.cpp:26 — Store::create silently overwrites an existing link on a code collision
- (high) main.cpp:115 — Path traversal in /report allows arbitrary file read and write outside export_dir
