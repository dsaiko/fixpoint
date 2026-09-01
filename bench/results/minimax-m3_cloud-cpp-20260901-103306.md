# minimax-m3:cloud · cpp · run 20260901-103306 (repeat 1)

recall **38/64** · 53 finding(s), 5 unmatched · 653249 tokens · 447s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Store::resolve has inverted expiry check |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Store::rename validates `from` twice and never validates `to; review-security: rename skips validation on the destination code |
| P04 | — | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  |  |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate always returns 0 |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Store::remove does not decrement quota |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner does substring match instead of equality; review-security: by_owner uses substring match, leaking links across owners |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Store::extend revives expired links |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-concurrency: Store::prune: ++it after links_.erase(it) is undefined behav |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Store::prune iterator invalidation and wrong return value |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns dangling string_view |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret enables session forgery |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens generated with non-cryptographic rand() |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Authenticator::issue writes the live token to stdout |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify never checks expires_at; review-security: Authenticator::verify never checks session expiry |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin condition is always true; review-security: require_admin is dead code that always returns true |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: hash_password uses non-cryptographic DJB2 |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Stack buffer overflow and null deref in bearer_token |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token dereferences null on header without space |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-security: Cache::set leaks every entry, enabling memory-exhaustion DoS |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache destructor leaks every Entry |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache::set leaks old Entry on overwrite |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache::sweep iterator invalidation |
| P31 | — | parallel: C05 utilization divides size_t by size_t before scaling so i |  |
| P32 | YES | parallel: CA02 the cache has no mutex at all, while the janitor thread | review-concurrency: Cache has no synchronization; get/set/sweep/erase race on en |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT env var is parsed but discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always returns true |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-bugs: export_csv has no path-traversal protection on name; review-security: Path traversal in export_csv / read_report via user-supplied |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-security: TOCTOU / symlink attack in write_snapshot via /tmp |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns pointer to stack buffer |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | — | parallel: X01 the counters are plain longs incremented from the janito |  |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-concurrency: Janitor thread mutates store.all() while request handlers re; review-concurrency: Janitor loop: ++it after erase(it) is undefined behavior |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache threads are never joined; review-concurrency: warm_cache leaves std::thread objects joinable; std::termina |
| P49 | YES | CPP-ONLY the thread lambda captures the loop variable by reference, so | review-concurrency: warm_cache lambda captures per-iteration loop variable by re |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads past the end of links |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | YES | CPP-ONLY the janitor thread is heap-allocated, never joined and never  | review-bugs: Janitor thread has iterator invalidation and lifetime bug |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-bugs: delete_link has no ownership check; review-security: delete_link ignores owner check documented in the comment |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out with no target validation |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is a spoofable client header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats divides by zero when store is empty; review-security: stats endpoint divides by zero when the store is empty |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS misconfiguration: Allow-Origin=* with Allow-Credentials |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Authentication failure echoes the presented token |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) export.cpp:47 — write_snapshot ignores its dir parameter
- (low) export.cpp:62 — read_report returns empty string when file missing
- (high) auth.cpp:20 — Authenticator.sessions_ is unsynchronized; issue/verify/revoke race
- (medium) config.cpp:43 — LINKD_EXPORT_DIR taken from env without validation
- (medium) store.cpp:145 — import_all silently overwrites existing links
