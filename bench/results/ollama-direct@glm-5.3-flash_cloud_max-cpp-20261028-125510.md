# ollama-direct@glm-5.3-flash:cloud#max · cpp · run 20261028-125510 (repeat 1)

recall **35/64** · 46 finding(s), 1 unmatched · 1097058 tokens · 565s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolver deletes live links and resolves expired ones; review-security: Link expiry test is inverted |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the old code twice |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Page construction forms a vector iterator past end; review-security: Report paging reads memory out of bounds |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate truncates most percentages to zero |
| P06 | — | parallel: C09 remove drops the link but never returns the owner's quot |  |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns least followed links |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner performs substring matching |
| P10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune invalidates its map iterator and returns the wrong cou |
| P15 | — | CPP-ONLY last_code returns a string_view onto a function-local std::st |  |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens use predictable rand() |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-concurrency: Authenticator sessions race and verify returns an interior p; review-security: Active session token is written to logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions still authenticate; review-security: Expired sessions are accepted |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin rejects every role |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: bearer_token crashes on absent or malformed Authorization he; review-security: Authorization header causes stack buffer overflow |
| P24 | — | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no |  |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache leaks every heap-allocated entry it removes |
| P29 | — | parallel: CA08 eviction drops the first key in map order, not the leas |  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache sweep invalidates the iterator it increments; review-concurrency: Cache state is exposed to concurrent mutation without a mute |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache utilization performs integer division before scaling |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond settings are treated as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: FETCH_LIMIT is parsed but discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always reports success |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: User-controlled report name allows path traversal |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-bugs: Export and report I/O failures are silently ignored; review-security: Sensitive export files do not get owner-only permissions |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV fields are not escaped; review-security: CSV data is emitted without separator or formula escaping |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | — | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul |  |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a dangling stack pointer |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters use unsynchronized long increments |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor invalidates its map iterator while erasing; review-concurrency: Janitor mutates the store through an unsynchronized shared m |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache destroys joinable threads; review-concurrency: warm_cache returns while its spawned threads remain joinable |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads one element past the end |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Create request errors escape uncaught |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link creation trusts client-supplied owner identity |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete another user's link |
| P56 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is replaced by spoofable x-admin header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Stats divide by zero when the store is empty |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: Redirect lowercases case-sensitive link codes |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exposes all links without administrator chec |
| P63 | YES | every service object is heap-allocated with raw new and never deleted, | review-concurrency: Janitor thread is neither stopped nor joined on shutdown |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Unspecified report name reads the wrong file |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) store.cpp:26 — Code collisions overwrite existing links and inflate quotas
