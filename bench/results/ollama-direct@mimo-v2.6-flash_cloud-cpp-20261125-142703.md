# ollama-direct@mimo-v2.6-flash:cloud · cpp · run 20261125-142703 (repeat 1)

recall **42/64** · 57 finding(s), 4 unmatched · 178037 tokens · 295s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiry check is inverted; review-security: Expiry check deletes valid links and serves expired links |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source twice but never validates the  |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Page can construct an out-of-range vector range |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Success rate integer division loses the percentage |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Removing a link does not return its quota slot |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-hit links and can grow past the stor |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner matches owner substrings instead of exact owners |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune() increments an invalidated iterator |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining size instead of removed count |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a dangling string_view |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated from deterministic std::rand ou |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer session tokens are written to process logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Session expiration is never enforced |
| P21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Passwords are protected with an unsalted non-cryptographic h |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Authorization header parsing overflows a fixed stack buffer |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token dereferences a missing header separator |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator session operations race and return escaping po |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache never frees Entry allocations; review-bugs: Cache evicts in map order instead of least-recent-use order |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache sweep increments an invalidated iterator |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache utilization uses integer division |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Timeout environment values are interpreted as seconds; review-bugs: Cache TTL environment values are interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: Configured fetch limit is discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() reports problems but always succeeds |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report name enables path traversal outside the export direct |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV fields are not escaped; review-concurrency: Concurrent report requests race on the same output path |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: Snapshot is renamed before it is flushed and checked; review-security: Snapshot uses a predictable temporary path vulnerable to sym |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to a local buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics error counter is uninitialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters use unsynchronized read-modify-write operat |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor erases and then increments an invalidated iterator; review-concurrency: Janitor races request threads on the Store map |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-concurrency: warm_cache destroys joinable threads and terminates; review-concurrency: stop_janitor signals without joining the worker thread |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads one element past the vector |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: count_for_owner inserts unknown owners into the quota map |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Handlers dereference missing query parameters with at() |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Create endpoint accepts caller-selected ownership |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Delete endpoint performs no ownership check |
| P56 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts a client-controlled header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Stats divides by an empty store; review-security: Stats endpoint divides by zero on an empty store |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exposes every tenant's links to any session |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report export and read use different filenames |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.cpp:39 — A zero-sized cache erases end() when setting an entry
- (medium) export.cpp:27 — Export write failures are ignored
- (medium) export.cpp:58 — Audit append failures are silently ignored
- (high) main.cpp:152 — Main exits immediately after startup
