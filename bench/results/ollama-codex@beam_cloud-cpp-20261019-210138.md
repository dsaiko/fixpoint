# ollama-codex@beam:cloud · cpp · run 20261019-210138 (repeat 2)

recall **49/64** · 88 finding(s), 11 unmatched · 2198918 tokens · 189s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: create() uses rand() for codes with no uniqueness check |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve deletes live links and accepts expired ones; review-security: resolve() inverts the expiry check |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source code twice; review-security: rename() uses a dangling reference after erase |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page computes an out-of-range end index |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate truncates before scaling |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove does not refund the owner quota; review-security: remove() neither refunds quota nor cleans cache |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top sorts ascending and may append default links; review-security: top() sorts ascending and pads the result with default-const |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner matches owners by substring; review-security: by_owner() uses substring matching instead of equality |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links; review-security: extend() revives expired links |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-security: quotas() returns a mutable reference to internal state |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic; review-security: import_all is not atomic |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune invalidates its iterator and returns the wrong count; review-security: prune() invalidates its iterator and returns the wrong count |
| P14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a dangling view and picks the wrong link |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Falls back to a hard-coded, publicly known default secret |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated with std::rand |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued session tokens are logged in plaintext |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify does not honor session expiry; review-security: verify() does not check session expiration |
| P21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Password hashing uses a weak, non-collision-resistant hash |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: bearer_token can overflow or dereference null; review-security: bearer_token overflows a fixed stack buffer and dereferences |
| P24 | — | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no |  |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals has a (theoretical) timing side channel |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator sessions map unprotected and verify returns da |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache leaks Entry objects; review-security: Cache destructor leaks Entry objects and expired entries |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache eviction does not implement LRU; review-security: set() does not evict the LRU entry and leaks on overwrite |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep invalidates its iterator while erasing; review-security: sweep() erases while iterating, causing undefined behavior |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization uses integer division and can divide by zero; review-security: utilization() divides before multiplying and may divide by z |
| P32 | YES | parallel: CA02 the cache has no mutex at all, while the janitor thread | review-concurrency: Cache entries map and hits counter lack synchronization |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: bad LINKD_TIMEOUT_MS is still applied |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: cache TTL milliseconds are treated as seconds; review-security: LINKD_CACHE_TTL_MS (milliseconds) is treated as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed but ignored; review-security: LINKD_FETCH_LIMIT is parsed but never applied |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always reports success; review-security: validate() returns true even when problems are found |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal in report name (export_csv/read_report) |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Export file is world-readable despite the comment |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all builds paths from owner names |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot renames before flushing output; review-security: write_snapshot uses a predictable /tmp path (TOCTOU/symlink) |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a dangling pointer; review-security: format_row returns a dangling pointer to a stack buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is uninitialized; review-security: Metrics::errors_ is uninitialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters incremented and read without atomicity or i |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: janitor loop invalidates its iterator while erasing; review-concurrency: Janitor thread mutates store.all() without locking and inval |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache returns before workers finish; review-concurrency: warm_cache spawns std::thread workers that are never joined |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads past the end of the vector; review-security: check_targets iterates one past the end |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-security: count_for_owner mutates the quota map when an owner is absen |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: request handlers dereference missing query values |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: delete_link does not enforce ownership (IDOR) |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via user-controlled next parameter |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quota check trusts a client-supplied header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats divides by the store size without checking for zero; review-security: stats() divides by zero when the store is empty |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows credentials with a wildcard origin |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown token is echoed back in the 401 response |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-security: report() uses a different name for the empty case |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.cpp:49 — require_admin always returns false
- (high) export.cpp:116 — report reads the wrong file name
- (medium) linkd.h:38 — quotas exposes a mutable store reference
- (high) store.cpp:10 — Store maps are accessed concurrently without synchronization
- (high) auth.cpp:49 — require_admin() always returns false
- (low) cache.cpp:30 — peek() reports expired entries as cached
- (medium) config.cpp:20 — LINKD_TIMEOUT_MS (milliseconds) is treated as seconds
- (high) store.cpp:65 — page() computes an out-of-bounds end index
- (medium) store.cpp:81 — success_rate() truncates to zero via integer division
- (medium) store.cpp:173 — last_code() returns the map-last key, not the most recent link
- (high) linkd.h:43 — Store::all() exposes the internal map for external mutation
