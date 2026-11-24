# ollama-codex@glm-5.3-flash:cloud#max · cpp · run 20261124-103744 (repeat 1)

recall **44/64** · 59 finding(s), 3 unmatched · 232788 tokens · 337s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-bugs: Random code collisions silently replace links; review-concurrency: Concurrent create calls race on rand's shared PRNG state |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Active links are deleted and expired links remain; review-security: Link expiry check is inverted |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the wrong code and mutates an erased elemen |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Paging constructs a vector with an invalid end iterator |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Success rate is integer-truncated to 0 or 100 |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Deleting a link does not release quota |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns least-used links and may pad with invalid links |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner filtering uses substring matching |
| P10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Batch import is not atomic |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() uses an invalid iterator after erase |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a dangling string view |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens use seeded-free predictable randomness |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are written to logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions remain valid; review-security: Expired sessions remain valid |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Admin role predicate is always false |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Passwords are stored with an unsalted fast hash |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Authorization header parsing is memory-unsafe |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Bearer parsing crashes on missing or malformed header |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator session map races across request threads |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Eviction ignores LRU state; review-bugs: A zero-size cache dereferences end() |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache entries leak on every removal; review-bugs: Cache sweep invalidates its iterator while erasing |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache utilization is truncated to zero |
| P32 | YES | parallel: CA02 the cache has no mutex at all, while the janitor thread | review-concurrency: Cache state is shared without synchronization |
| P33 | YES | parallel: CF01 atoi cannot report failure and returns 0, so a malforme | review-bugs: Invalid timeout overrides the default and milliseconds are t |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: FETCH_LIMIT is parsed but discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Invalid configuration is accepted; review-security: Configuration validation never rejects invalid settings |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Export report name permits path traversal |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV fields are not escaped; review-security: CSV export lacks escaping and formula safeguards |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: Snapshot is renamed before output is flushed; review-concurrency: Concurrent snapshot writes share one temporary file |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a dangling local buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics error counter is not explicitly initialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are modified by concurrent requests without |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-concurrency: Janitor mutates the store during unsynchronized concurrent a; review-concurrency: Janitor invalidates its store iterator after erasing |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-concurrency: warm_cache starts joinable threads then abandons them; review-concurrency: stop_janitor does not join the janitor thread |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads past the end |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: Counting an unknown owner mutates the quota map |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Link deletion has no ownership check |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect through the out endpoint |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Administrator access is granted by a client-controlled heade |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Stats divides by zero on an empty store |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: Authentication is required for public redirects |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report exposes every owner's links to any authenticated user |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report is read from the wrong filename |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) main.cpp:140 — The service never starts listening
- (medium) export.cpp:26 — Concurrent exports to the same report race with readers
- (high) store.cpp:65 — Pagination reads one Link object out of bounds
