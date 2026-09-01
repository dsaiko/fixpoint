# qwen/qwen3.8-27b · cpp · run 20260901-120618 (repeat 1)

recall **50/64** · 67 finding(s), 2 unmatched · 354053 tokens · 1442s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Link codes generated with unseeded std::rand(): all codes en |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry test inverted: live links are 404'd and del; review-security: Inverted expiry check: live links deleted on resolve, expire |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice and never validates 'to' |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() builds a range one element past the end of the vector; review-security: page() off-by-one: deterministic out-of-bounds heap read on  |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() integer-divides before multiplying, reporting |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() never returns the owner's quota slot |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-followed links instead of the most |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches by substring, not exact owner; review-security: by_owner matches by substring, leaking other users' links |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects already-expired links; review-security: extend() resurrects already-expired links, defeating final e |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: quotas() is documented as a snapshot but returns a mutable r |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic: a bad code mid-batch leaves earl |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune() erases from the map and then increments the invalida |
| P14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a dangling string_view and picks the wro |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens generated with unseeded std::rand(): predicta |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Live session token written to stdout on every issue |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never expires sessions; review-security: verify() never enforces session expiry: tokens valid forever |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() condition is a tautology: always rejects; review-security: require_admin predicate is always true: admin check rejects  |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Password storage hash is unsalted 32-bit djb2 |
| P23 | — | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 |  |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token() overflows a 64-byte stack buffer and derefere; review-security: Pre-auth stack buffer overflow and null-pointer crash in bea |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals is not constant-time despite claiming to be |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: verify() returns a pointer into sessions_ while issue/revoke |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache leaks every Entry: destructor, eviction, and overwrite |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache eviction picks the alphabetically first entry, not the |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache::sweep() erases from the map and then increments the i |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() integer-divides (always 0% until full) and div |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: 'keeping default' branch actually overwrites the timeout wit; review-security: LINKD_TIMEOUT_MS=0 logs 'keeping default' but applies a zero |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Milliseconds env vars installed as Seconds (1000x error) |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed then discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so bad deploys never fail at; review-security: validate() always returns true: dangerous configs start the  |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal via unsanitized report name: arbitrary CSV re |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-bugs: Export/report file I/O failures are silently swallowed |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all writes files named after unsanitized user-contro |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot() renames before flush, ignores rename failur; review-security: Snapshot staged at fixed, world-readable /tmp path: info lea |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns a pointer to a local array (dangling) |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ has no default initializer |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are non-atomic longs mutated by every reque |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor erases from the store's map and then increments the ; review-concurrency: Janitor erases from Store::links_ while request handlers mut |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache() never joins its worker threads; returning calls; review-concurrency: warm_cache destroys joinable std::thread objects, terminatin |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() reads one element past the end of the vector |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | YES | CPP-ONLY the janitor thread is heap-allocated, never joined and never  | review-concurrency: Janitor thread is never joined, detached, or deleted; stop_j |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Uncaught exceptions kill the service on malformed requests |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-bugs: delete_link() never checks ownership; review-security: delete_link has no ownership check: any session deletes any  |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out sends users to an arbitrary attacker URL |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated on client-controlled x-admin header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: GET /stats divides by zero when the store is empty |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS: wildcard origin combined with Allow-Credentials on eve |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response body echoes the presented token |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report serves every owner's link data to any authenticated  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: GET /report without ?name always returns an empty body; review-concurrency: /report writes then reads back a shared file, racing concurr |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) main.cpp:44 — ttl from ?ttl= is parsed with unchecked atoi
- (medium) store.cpp:27 — Per-owner quota is tracked but never enforced: unbounded creation leads to memory exhausti
