# gpt-6.2-sol · cpp · run 20261118-144007 (repeat 1)

recall **48/64** · 78 finding(s), 15 unmatched · 294289 tokens · 906s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Following a live link deletes it; review-security: Resolving a live link deletes it and permits expired redirec |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and skips the destina |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: A partial page copies past the end of the link vector; review-security: Report pagination reads past the end of the link vector |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Success rate reports zero until every link has a hit |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Owner counts drift when links are removed or imported |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard selects the least-followed links and can ret |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup includes substring matches |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive it |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: The quota snapshot exposes mutable store state |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: An invalid link leaves an import partially applied |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns the number of links remaining |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: The latest-code lookup returns a dangling view and selects b |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens come from a predictable random sequence |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session issuance logs reusable bearer tokens |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Sessions remain valid after their recorded expiration; review-security: Expired sessions remain authorized |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: The admin-role helper rejects every session |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: The password storage helper uses a fast unsalted hash |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Authorization parsing can corrupt the stack before authentic |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Missing or oversized authorization headers can crash request |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Session verification races with session revocation |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache churn leaks entries beyond the configured capacity |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache eviction ignores recent use and evicts on replacement |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: The cache sweep increments an invalid iterator after its fir |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: A partially full cache reports zero utilization |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: An invalid timeout overwrites the default despite the fallba |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond settings are interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT never changes the returned configuration |
| P37 | YES | parallel: CF05 load_file never checks whether the stream opened, so an | review-bugs: Configuration file read failures silently return partial set |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation never rejects invalid settings |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report names can escape the export directory |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report files can be readable by other local users |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV exports corrupt fields containing commas, quotes, or new; review-security: Attacker-controlled CSV fields can become spreadsheet formul |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: Archive and audit writers silently drop failed writes |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: Snapshot publication can replace a good snapshot with incomp; review-concurrency: Concurrent audit appends can merge records |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: The terminal row formatter returns a pointer to dead stack s |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Default-constructed metrics can expose an uninitialized erro |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Concurrent requests race on dashboard counters |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: The janitor increments an invalid iterator after removing an; review-concurrency: Janitor shutdown leaves the worker running |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: Warming the cache with any links terminates the process; review-bugs: Stopping the janitor leaks its thread handle and prevents re |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: Target checking reads past the end of the link list |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Missing or invalid request parameters escape as exceptions |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link creation accepts a forged owner |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated caller can delete another owner's link |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: A cached outbound redirect bypasses click counting; review-security: The click-tracking endpoint redirects to arbitrary destinati |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quotas trusts a caller-supplied header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: The stats endpoint divides by zero on an empty store; review-concurrency: Stats can divide by zero after concurrent removal |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: Redirect lookup changes valid mixed-case codes |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: The default report reads a different file from the one it wr; review-concurrency: Requests for the same report can return each other's output |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) main.cpp:152 — The executable exits immediately after announcing that it is listening
- (high) store.cpp:60 — Renaming a link uses it after its map entry is erased
- (high) cache.cpp:37 — A zero-capacity cache has invalid insertion and utilization paths
- (high) main.cpp:44 — TTL parsing can overflow before the duration is constructed
- (high) store.cpp:26 — A generated code collision overwrites an existing link
- (high) export.cpp:29 — Report I/O failures produce successful download responses
- (medium) cache.cpp:31 — Cache peek reports expired entries as available
- (medium) main.cpp:32 — Responses advertise JSON for CSV and plain-text bodies
- (medium) main.cpp:96 — The quotas endpoint omits each owner's link count
- (medium) cache.cpp:36 — Cache workers mutate the same map without synchronization
- (medium) main.cpp:48 — Create response dereferences a link that can be erased
- (medium) export.cpp:48 — Concurrent snapshot writers share one temporary file
- (high) main.cpp:115 — Report downloads expose other owners' records
- (high) store.cpp:27 — Link creation records quotas without enforcing a storage limit
- (medium) main.cpp:62 — Redirect headers accept control characters from request data
