# ollama-codex@glm-5.3-flash:cloud#max · cpp · run 20261028-125904 (repeat 1)

recall **47/64** · 68 finding(s), 7 unmatched · 3857225 tokens · 890s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-bugs: generated code collisions silently overwrite existing links; review-security: Short link codes are predictable |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link resolution expiry test is inverted; review-security: Expiry comparison is inverted and destroys active links |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source code twice |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Pagination clamp goes one past the container end |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate performs integer division before scaling |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: delete does not restore quota |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top is sorted in the wrong direction |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner matches a substring instead of exact owner; review-security: Owner lookup uses substring matching |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune erases its map iterator before incrementing it |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune reports the wrong count |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a view into a destroyed local string |
| P16 | YES | CPP-ONLY create returns a raw pointer into the map; any later erase or | review-concurrency: Store::create returns a pointer into a mutable internal map |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated from std::rand |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer tokens are written to logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: expired sessions are never rejected; review-security: Expired sessions remain accepted |
| P21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Passwords use an unsalted non-cryptographic hash |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Stack buffer overflow in Authorization parsing |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token crashes on a missing space and overflows its fi |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Session lookup races issuance/revocation and returns an unpr |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Raw cache Entry pointers leak throughout the cache |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache eviction is not least-recently-used |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: cache sweep erases while incrementing an invalidated iterato; review-concurrency: Cache operations are unsynchronized and sweep invalidates it |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization truncates before scaling |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed but never assigned |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always claims invalid configuration is valid; review-security: validate always claims invalid configuration is valid |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report names allow arbitrary path traversal |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Sensitive export files do not enforce owner-only permissions |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV fields are not escaped; review-security: CSV rows are not field-escaped or formula-neutralized |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: snapshot is renamed before it is flushed or closed; review-concurrency: Concurrent snapshot writers share a fixed predictable tempor |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a dangling pointer |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are non-atomic shared state |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: janitor erases its store iterator before incrementing it; review-concurrency: Janitor races the unsynchronized store map and invalidates i |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache leaves joinable threads to terminate the process; review-concurrency: Janitor lifecycle has no join, wakeup, or safe restart |
| P49 | YES | CPP-ONLY the thread lambda captures the loop variable by reference, so | review-concurrency: warm_cache destroys joinable threads and never waits for the |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads past the end of links |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: count_for_owner mutates the quota map for unknown owners; review-concurrency: count_for_owner mutates the shared quota map during a lookup |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Callers can create links owned by other users |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated token can delete any link |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unrestricted open redirect endpoint |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts a spoofable x-admin header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats divides by zero on an empty store; review-security: Stats endpoint divides by zero on an empty store |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: tolower is called with a potentially negative char value |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: handle has no exception boundary for malformed requests |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Any authenticated user can export all owners' links |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report reads the wrong file when the name is empty |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) cache.cpp:38 — A zero-limit cache erases the end iterator
- (medium) cache.cpp:31 — peek reports expired entries as cached
- (high) config.cpp:20 — millisecond durations are interpreted as seconds
- (medium) auth.cpp:49 — admin authorization condition is always false
- (high) main.cpp:152 — main exits while the janitor still owns live references
- (medium) main.cpp:115 — Concurrent report requests race on the same CSV file
- (high) auth.cpp:49 — Admin authorization check always denies, inviting header-based workarounds
