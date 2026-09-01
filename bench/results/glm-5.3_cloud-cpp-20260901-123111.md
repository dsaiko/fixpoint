# glm-5.3:cloud · cpp · run 20260901-123111 (repeat 1)

recall **38/64** · 49 finding(s), 4 unmatched · 701297 tokens · 351s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Link codes come from unseeded rand() and silently overwrite  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry condition is inverted: valid links are dele |
| P03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() clamps end to size()+1 and iterates one past the vect |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() always reports 0 due to integer division orde |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() deletes the link but never gives the quota slot bac |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, so the leaderboard returns the least- |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches owners by substring, not equality; review-security: by_owner matches owners by substring, returning other users' |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links, contradicting its co |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic: a bad link leaves earlier links  |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune() erases from the map while iterating it |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining link count instead of the numb |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a string_view into a destroyed local std |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Auth secret falls back to a hardcoded constant when LINKD_SE |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are minted from unseeded rand() and are predi |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are written to stdout logs on issuance |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks the session's expiry, so expired token; review-security: verify() never checks expires_at despite issuing 12-hour ses |
| P21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: bearer_token copies the attacker-controlled Authorization he |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token() crashes on any header without a space and ove |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals is not constant-time despite its comment, leak |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache leaks every Entry: allocated with new, never deleted |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache::sweep() erases from entries_ while iterating it; review-concurrency: Erase-then-increment iterator invalidation in sweep (and the |
| P31 | — | parallel: C05 utilization divides size_t by size_t before scaling so i |  |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: Bad LINKD_TIMEOUT_MS prints 'keeping default' then overwrite |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond env values are stored as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed and then thrown away |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() returns true even when it found fatal problems |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: User-supplied report name is concatenated into filesystem pa |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | — | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul |  |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns a pointer to a stack buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is never initialized, so /stats prints an i |
| P46 | — | parallel: X01 the counters are plain longs incremented from the janito |  |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor loop erases from store.all() while iterating it; review-concurrency: Janitor thread mutates Store and Cache with no synchronizati |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache() returns before its threads finish and never joi; review-concurrency: warm_cache destroys joinable threads, calling std::terminate |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() loops i <= links.size() and reads out of bou |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: count_for_owner() default-inserts a zero entry into the stor |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: create_link lets exceptions escape handle(): a bad request t |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: delete_link removes any link with no ownership check, contra |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out is an open redirect: Location is set to the raw ?next=  |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes by a client-supplied x-admin heade |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by g_store->size() and crashes on an empty s |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: Access-Control-Allow-Origin: * combined wit |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.cpp:49 — require_admin() always returns false because of the || condition
- (high) store.cpp:59 — rename() copies a Link through a dangling reference after erasing it
- (medium) cache.cpp:64 — utilization() is always ~0 and divides by zero when the limit is 0
- (low) auth.cpp:49 — require_admin is dead code that always returns false
