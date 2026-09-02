# kimi-k2.7-code:cloud · cpp · run 20260901-220328 (repeat 1)

recall **30/64** · 43 finding(s), 4 unmatched · 270017 tokens · 314s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted expiration check deletes live links; review-security: Inverted expiry check deletes live links on every resolve |
| P03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page computes one-past-the-end iterator when truncating |
| P05 | — | parallel: C05 integer division before scaling yields 0, and the int/si |  |
| P06 | — | parallel: C09 remove drops the link but never returns the owner's quot |  |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | — | parallel: C11 by_owner matches owners by substring, so one owner sees  |  |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune erases while iterating and returns wrong count |
| P14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a string_view to a local string; review-security: Store::last_code returns a dangling string_view |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated from a weak PRNG |
| P19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify ignores session expiration; review-security: Session verification ignores token expiration |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin rejects every role because of OR instead of AN |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: bearer_token overflows its stack buffer and dereferences nul; review-security: Stack buffer overflow in bearer token parsing |
| P24 | — | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no |  |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator sessions map is unsynchronized and verify retu |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-security: Cache leaks Entry objects, enabling memory exhaustion |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | — | parallel: CA08 eviction drops the first key in map order, not the leas |  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep erases while iterating, invalidating the iterator |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization is always 0 or 100 due to integer division order |
| P32 | YES | parallel: CA02 the cache has no mutex at all, while the janitor thread | review-concurrency: Cache data races and map corruption |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is parsed as seconds, not milliseconds |
| P36 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t |  |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true |
| P39 | — | parallel: EX01 the report name comes from the query string and is conc |  |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | — | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul |  |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a dangling stack pointer; review-security: format_row returns a pointer to a local stack buffer |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are plain longs, causing data races |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-security: Janitor erases store entries while iterating, causing UB/cra |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache spawns threads but never joins them; review-concurrency: warm_cache spawns unjoined threads and races on Cache |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets walks one element past the vector |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-concurrency: count_for_owner mutates the shared quota map unsynchronized |
| P52 | YES | CPP-ONLY the janitor thread is heap-allocated, never joined and never  | review-concurrency: Janitor thread mutates Store and Cache without locks and is  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-bugs: delete_link does not enforce the owner check it documents; review-security: Any authenticated user can delete arbitrary links |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: admin_quotas bypasses the Authenticator role check; review-security: Admin quota endpoint trusts a client-controlled header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-security: Division by zero in stats endpoint when store is empty |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows credentials with wildcard origin |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report reads a different filename than export_csv writes |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) config.cpp:21 — LINKD_TIMEOUT_MS is parsed as seconds, not milliseconds
- (medium) store.cpp:81 — success_rate is always 0 or 100 due to integer division order
- (critical) store.cpp:10 — Store data races and use-after-free under concurrent access
- (high) main.cpp:115 — Path traversal in /report via unsanitized report name
