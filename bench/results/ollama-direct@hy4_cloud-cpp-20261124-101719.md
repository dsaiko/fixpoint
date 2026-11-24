# ollama-direct@hy4:cloud · cpp · run 20261124-101719 (repeat 1)

recall **54/64** · 81 finding(s), 7 unmatched · 353394 tokens · 1049s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry check is inverted; review-security: Link expiry check is inverted: live links are deleted and de |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename validates the wrong code twice |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page clamps the end iterator past the vector end; review-security: Report pagination constructs a vector from one element past  |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate performs integer division before scaling |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove does not return the owner's quota slot |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top sorts in the wrong direction |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: top pads the result with fake rows |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner uses substring matching; review-security: Per-owner filtering uses substring matching and leaks other  |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links; review-security: Extend operation revokes the finality of link expiration |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: quotas exposes mutable internal state instead of a snapshot |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic and leaves quotas stale; review-security: Imported links bypass target validation and can inject redir |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune increments an erased iterator |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns the remaining count instead of the removed cou |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a view to a destroyed local string; review-bugs: last_code ignores creation time |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Missing LINKD_SECRET silently enables a source-visible unive |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens come from an unseeded, noncryptographic PRNG |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Live session tokens are written to stdout |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Session expiry is never checked; review-security: Session verification ignores expiration |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-security: Admin role check can never succeed |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Password storage uses an unsalted 64-bit DJB2 digest |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: Authorization header is copied into a fixed 64-byte buffer; review-security: Attacker-controlled Authorization header overflows a stack b |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Malformed or missing Authorization header dereferences null |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: Secret comparison leaks length and matching prefix through t |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator shares sessions_ across threads with no lock a |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Re-setting an existing cache key still evicts and leaks |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache Entry objects are leaked |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache eviction ignores LRU order |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep increments an erased map iterator; review-security: Cache sweep uses an iterator after erasing it |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization divides before scaling and divides by zero for a |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS is treated as seconds and bad values are ke; review-bugs: LINKD_CACHE_TTL_MS is treated as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed but discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate always reports success; review-security: Configuration validation always succeeds and dangerous limit |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report names allow path traversal for arbitrary file writes  |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Cross-owner reports are created with default, potentially wo |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV export reports success even when file I/O fails; review-bugs: CSV fields are not escaped |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: Snapshot is renamed before flush and rename failures are ign; review-security: Predictable /tmp snapshot path enables symlink overwrite and |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a dangling stack pointer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: errors_ is uninitialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are plain longs mutated and read across req |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor increments an erased map iterator; review-concurrency: Janitor thread structurally mutates Store/Cache concurrently |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache destroys joinable threads; review-concurrency: stop_janitor signals the flag but never joins/awaits the jan |
| P49 | YES | CPP-ONLY the thread lambda captures the loop variable by reference, so | review-concurrency: warm_cache never joins its worker threads, so destroying joi |
| P50 | — | CPP-ONLY check_targets loops with <= and reads one element past the en |  |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: count_for_owner inserts missing owners into quotas |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Missing query parameters and invalid targets throw uncaught  |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link ownership is taken from an untrusted query parameter |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete another owner's link |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Missing next parameter cannot produce the documented 400; review-security: Unvalidated next parameter creates an open redirect and poss |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts a client-supplied x-admin header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: /stats divides by zero on an empty store; review-security: Stats endpoint can crash on an empty store and serializes un |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: tolower is called with a possibly negative char |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Every response uses wildcard CORS while advertising credenti |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response reflects the raw bearer token |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint discloses every owner's links to any authent |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report is read from the wrong filename |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) cache.cpp:39 — A zero-entry cache erases the end iterator
- (high) store.cpp:59 — rename uses a reference after erasing the map node
- (high) worker.cpp:43 — check_targets reads one element past the vector
- (medium) auth.cpp:49 — require_admin rejects every role
- (medium) store.cpp:26 — create overwrites existing links on code collision
- (high) worker.cpp:37 — warm_cache runs Cache::set concurrently from many threads on one unsynchronized Cache
- (medium) export.cpp:56 — Raw audit lines allow forged log entries
