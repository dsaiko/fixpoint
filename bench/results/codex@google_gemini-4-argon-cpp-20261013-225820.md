# codex@google/gemini-4-argon · cpp · run 20261013-225820 (repeat 1)

recall **48/64** · 60 finding(s), 3 unmatched · 1419262 tokens · 531s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Predictable short codes generated with std::rand() in Store: |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted expiration comparison in Store::resolve deletes val |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Store::rename validates `from` twice instead of validating `; review-security: Missing validate_code(to) call in Store::rename() |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Out-of-bounds iterator `all.size() + 1` in Store::page |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Integer division before multiplication in Store::success_rat |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Store::remove does not decrement the owner's quota count |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: Store::top sorts in ascending order and pads with empty link |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Store::by_owner uses substring search instead of exact owner; review-security: Substring comparison in Store::by_owner() leaks other users' |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Store::extend revives already-expired links |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: Store::quotas returns a mutable reference to internal state  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Store::import_all partially mutates the store when validatio |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: Iterator invalidation and wrong return value in Store::prune |
| P14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: Store::last_code returns a dangling std::string_view and use |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Predictable session tokens generated with unseeded std::rand |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Plaintext session bearer tokens written to stdout logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Authenticator::verify does not check session expiration; review-security: Session expiration time is never checked in Authenticator::v |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Impossible condition in Authenticator::require_admin rejects; review-security: Broken boolean logic in Authenticator::require_admin() rejec |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Unsalted non-cryptographic DJB2 hash used for password stora |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Stack buffer overflow and null pointer dereference in bearer |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Null pointer dereference and buffer overflow in bearer_token |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: Non-constant-time comparison in secret_equals() leaks secret |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache leaks heap-allocated Entry objects on destruction, evi |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache::set evicts lexicographically first key instead of lea; review-concurrency: Data race on Cache::entries_ and Cache::hits_ during concurr |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Iterator invalidation in Cache::sweep |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Integer division in Cache::utilization always returns 0 belo |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: load_config overwrites default timeout on invalid input and  |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: load_config parses LINKD_FETCH_LIMIT but discards the value |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate(const Config&) always returns true even when config |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal in export_csv() and read_report() via unsanit |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Sensitive CSV exports created with default world-readable pe |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Path traversal in archive_all() via unvalidated owner name |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot renames temporary file before closing/flushin; review-security: Predictable /tmp file path and TOCTOU race in write_snapshot |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to a stack-allocated local buff |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is uninitialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are updated and read without synchronizatio |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Iterator invalidation in start_janitor link sweep loop; review-concurrency: Unsynchronized concurrent access to Store between janitor th |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache destroys joinable std::thread objects without joi; review-concurrency: warm_cache destroys joinable std::thread objects without joi |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: Off-by-one loop bound `i <= links.size()` in check_targets r |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: count_for_owner inserts unknown owners into the quota map vi |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Account impersonation in create_link() via untrusted owner q |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Missing ownership check in delete_link() allows deleting any |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unvalidated next parameter in out() |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization bypass via client-supplied x-admin reque |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Division by zero in stats() when the store has no links |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS origin combined with Access-Control-Allow-Cred |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Verbose 401 error response reflects raw bearer token without |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Unauthorized disclosure of all owners' links in report() |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) store.cpp:58 — Use-after-free in Store::rename after erasing map iterator
- (medium) cache.cpp:31 — Cache::peek returns true for expired entries
- (high) export.cpp:63 — read_report does not apply the default report name used by export_csv
