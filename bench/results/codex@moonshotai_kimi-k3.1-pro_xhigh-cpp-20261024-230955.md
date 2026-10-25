# codex@moonshotai/kimi-k3.1-pro#xhigh · cpp · run 20261024-230955 (repeat 1)

recall **44/64** · 63 finding(s), 13 unmatched · 860461 tokens · 458s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-concurrency: Concurrent std::rand calls race and can duplicate identifier; review-security: Predictable unchecked short codes can overwrite existing lin |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiration condition deletes live links and resolves expired |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the source code twice |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Normal final pages construct an out-of-bounds iterator |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Success rate almost always rounds down to zero |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Deleting a link never returns its quota slot |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard sorts least-followed links first |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: Oversized top requests invent default links |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner filtering uses substring matching |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend resurrects expired links |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Failed imports leave partial batches committed |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune increments an invalidated iterator |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns remaining links instead of removals |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a dangling view |
| P16 | YES | CPP-ONLY create returns a raw pointer into the map; any later erase or | review-concurrency: Store maps are accessed concurrently without synchronization |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated with predictable randomness |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer tokens are written to logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions remain valid; review-security: Expired session tokens remain valid indefinitely |
| P21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Password hashing is unsalted and extremely fast |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: Bearer token parsing overflows its stack buffer; review-security: Malformed Authorization headers crash the process |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Malformed Authorization headers cause undefined behavior |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache entry allocations leak on every removal |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Eviction does not select the least recently used entry |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache sweep uses an invalidated iterator; review-concurrency: Cache operations race with workers and the janitor |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache utilization truncates before converting to percent |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: Invalid timeout overwrites the default after warning |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond settings are interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: Fetch limit environment setting is discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation always succeeds |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report names permit path traversal and arbitrary CSV file wr |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | — | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul |  |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to dead stack storage |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Error metric reads an uninitialized value |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics increments and reads are unsynchronized |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor erases through an invalidated iterator; review-security: Janitor races request handlers on the links map |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache destroys joinable threads; review-concurrency: warm_cache destroys joinable threads instead of awaiting the |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads one element past the vector |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Clients can forge link ownership |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete any link |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The outbound endpoint is an open redirect |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A caller-controlled header grants administrator access |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Stats divides by zero on an empty store |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: Redirect corrupts case-sensitive codes |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Authentication errors reflect supplied bearer tokens |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Every authenticated user can export all users' links |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.cpp:49 — Admin authorization rejects every role
- (medium) cache.cpp:31 — peek reports expired entries as cached
- (high) cache.cpp:38 — A zero-sized cache erases from an empty map
- (high) cache.cpp:64 — Zero cache capacity divides by zero
- (high) export.cpp:51 — Snapshot rename failures are silently ignored
- (high) store.cpp:60 — rename modifies a link after erasing it
- (medium) store.cpp:176 — last_code does not select the newest link
- (high) main.cpp:48 — Created link pointer can dangle before the response reads it
- (high) auth.cpp:30 — Session map races with verification and revocation
- (high) worker.cpp:29 — stop_janitor neither waits for nor releases the worker
- (high) main.cpp:115 — Concurrent reports with the same name overwrite each other
- (high) export.cpp:48 — Concurrent snapshots share one temporary pathname
- (high) main.cpp:62 — User-controlled targets permit response header injection
