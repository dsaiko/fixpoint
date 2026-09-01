# glm-5.2:cloud · cpp · run 20260901-183123 (repeat 1)

recall **40/64** · 51 finding(s), 0 unmatched · 740746 tokens · 530s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: valid links are erased, ; review-security: Inverted expiry check resolves expired links and erases vali |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() clamps end to size()+1, reading one past the vector e |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() uses integer division, so the percentage is a |
| P06 | — | parallel: C09 remove drops the link but never returns the owner's quot |  |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() uses substring match instead of equality; review-security: by_owner uses substring match, leaking links across owners |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() unconditionally revives already-expired links, cont |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic: a validation failure mid-batch l |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() erases from the map then increments an invalidated i |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a string_view dangling into a destroyed ; review-security: last_code returns a string_view dangling into a destroyed lo |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default service secret when LINKD_SECRET is unset |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens generated with std::rand() are predictable an |
| P19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() ignores session expiry and accepts expired tokens; review-security: verify() never checks token expiry, so sessions never expire |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always returns false due to || instead of && |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Passwords hashed with unsalted DJB2, a non-cryptographic has |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Stack buffer overflow and null deref in bearer_token on atta |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token() crashes on a missing space and overflows its  |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals is not constant-time, leaking match position v |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator's sessions_ map is accessed by concurrent requ |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache::set() evicts the lexically-first key, not the least-r |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache::sweep() erases from the map then increments an invali; review-concurrency: Cache is shared between the janitor thread and request handl |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() integer-truncates and divides by zero when lim |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: load_config() overrides the default timeout with 0 when LINK |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded, never assign |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() reports problems but always returns true, so a ba; review-security: validate() always returns true regardless of detected proble |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal via unsanitized report name in export/read en |
| P40 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-security: CSV formula injection via unescaped owner/target fields |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot() renames the temp file before flushing/closi |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns a pointer to a local stack buffer (dang |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ has no initializer, so errors() returns ind |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are plain longs incremented from concurrent |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor loop erases from a std::map then increments the inva; review-concurrency: Janitor thread mutates the store's links_ map with no lock w |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache() never joins its threads, so std::thread destruc; review-concurrency: warm_cache spawns threads it never joins, so std::terminate  |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() loops with i <= links.size(), reading one pa; review-security: check_targets reads one element past the vector (out-of-boun |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: create_link() computes ttl as int*3600, overflowing for larg |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-bugs: delete_link() removes by code without checking the authentic; review-security: delete_link removes any link without owner authorization (ID |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unsanitized ?next= in /out endpoint |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization trusts a client-controlled x-admin heade |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by g_store->size(), crashing on an empty sto |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: Access-Control-Allow-Origin '*' with Allow- |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented token back in the body |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |
