# qwen3.5:397b-cloud · cpp · run 20260902-111154 (repeat 1)

recall **28/64** · 38 finding(s), 4 unmatched · 332378 tokens · 173s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Predictable short code generation |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted expiry check deletes valid links; review-security: Logic error deletes valid links on access |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() does not validate the new code |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Off-by-one error in page() end index |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Integer division truncates success_rate() |
| P06 | — | parallel: C09 remove drops the link but never returns the owner's quot |  |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns least popular links instead of most |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | — | parallel: C11 by_owner matches owners by substring, so one owner sees  |  |
| P10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic and skips quota |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: Iterator invalidation in prune() loop |
| P14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: Dangling string_view returned from last_code() |
| P16 | YES | CPP-ONLY create returns a raw pointer into the map; any later erase or | review-concurrency: Data race on Store::links_ map between HTTP handlers and jan |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded secret key in fallback |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Predictable session token generation |
| P19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| P20 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Logic error in require_admin() always rejects; review-security: Authentication bypass in require_admin |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Weak password hashing algorithm |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Buffer overflow in bearer_token |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Null pointer dereference in bearer_token() |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache evicts by key order, not LRU |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Iterator invalidation in sweep() loop |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Integer division breaks utilization() |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t |  |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | — | parallel: CF06 validate logs the problems it finds and returns true, s |  |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal in CSV export |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-concurrency: Race condition on export file writes; review-security: Hardcoded temporary file path |
| P44 | — | CPP-ONLY format_row returns a pointer to a function-local char array,  |  |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Data race on Metrics counters from concurrent request handle |
| P47 | — | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th |  |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache() returns before workers complete; review-concurrency: Thread leak in warm_cache() - threads never joined |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: Off-by-one out-of-bounds access in check_targets(); review-security: Out-of-bounds array access in check_targets |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | — | parallel: C17 delete_link never checks the owner its comment promises  |  |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| P57 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Division by zero in stats() at startup |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Insecure CORS configuration with credentials |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Session token leaked in error response |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) cache.cpp:36 — Data race on Cache::entries_ map from concurrent cache.set() calls
- (high) auth.cpp:20 — Data race on Authenticator::sessions_ map
- (high) worker.cpp:10 — Non-atomic access to g_janitor pointer
- (critical) export.cpp:63 — Path traversal in report download
