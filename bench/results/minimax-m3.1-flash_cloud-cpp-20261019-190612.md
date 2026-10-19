# minimax-m3.1-flash:cloud · cpp · run 20261019-190612 (repeat 1)

recall **51/64** · 76 finding(s), 3 unmatched · 543675 tokens · 4846s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-bugs: Link codes are generated from std::rand without seeding and ; review-security: Short-link codes are generated from std::rand, making them g |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Store::resolve inverts the expiry test and erases every live; review-security: Link expiry is inverted in resolve, so live links are destro |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Store::rename validates `from` twice and never validates `to; review-security: rename validates `from` twice and never validates `to`, the  |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Store::page computes the end index as size+1, producing an o |
| P05 | — | parallel: C05 integer division before scaling yields 0, and the int/si |  |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Store::remove does not give the quota slot back |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Store::top sorts ascending and returns the least-hit links |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Store::by_owner matches on substring rather than exact owner; review-security: by_owner matches owners by substring, so one owner can enume |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Store::extend revives an already-expired link, contradicting |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: Store::quotas() returns a mutable reference to the live quot; review-concurrency: Store::quotas() hands out a mutable reference to the live qu |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Store::import_all is not atomic despite its documented guara |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Store::prune returns the remaining link count instead of the |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: Store::last_code returns the lexicographically last code, no |
| P16 | YES | CPP-ONLY create returns a raw pointer into the map; any later erase or | review-bugs: Store::create returns a pointer into the map that a concurre |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: The service falls back to a hardcoded default secret when LI |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated from unseeded std::rand and are |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued session tokens are written to stdout |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Authenticator::verify ignores the session's expiry; review-security: Session expiry is never checked, so a session is valid forev |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin demands the role be both admin and owner, so i; review-security: require_admin ORs two contradictory role checks, so it alway |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: hash_password uses djb2, a fast non-cryptographic hash, with |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: bearer_token overflows a fixed 64-byte buffer and can derefe |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-security: bearer_token strcpy's an unbounded Authorization header into |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-bugs: secret_equals short-circuits on the first mismatch; review-security: secret_equals returns early on a length mismatch, leaking th |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator::verify returns a pointer into the sessions ma |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache leaks every Entry it allocates |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache::set evicts by map order, not by least-recently-used |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache::sweep erases from a std::map while iterating it |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache::utilization divides by limit_ and multiplies by 100 i |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-security: The config file can set export_dir to a relative or attacker |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS is parsed as milliseconds but stored as sec |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so an invalid configuration ; review-security: validate() reports configuration problems but always returns |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: The report filename is built from a user-supplied name with  |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-bugs: The export file is created with default permissions instead ; review-security: Export files containing every owner's links are created worl |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: export_csv does not confine the report name to the export di; review-bugs: export_csv writes link fields into CSV without escaping comm |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all builds per-owner filenames from the owner string |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot renames a fixed /tmp path into the target dir; review-security: write_snapshot uses a fixed path in /tmp, which is a symlink |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to a local buffer that is destr; review-security: format_row returns a pointer to a stack buffer that is out o |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is read but never initialised |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are plain longs incremented from concurrent |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-concurrency: Janitor thread mutates the Store's map with no synchronizati |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache captures the loop variable by reference and never; review-bugs: stop_janitor sets the flag but never joins the thread, and t |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads one element past the end of the links ve; review-concurrency: check_targets reads one element past the end of the links ve |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: create_link reads required query parameters with .at() and t |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-bugs: The ttl query parameter is multiplied by 3600 but interprete; review-security: create_link takes the link owner from an unauthenticated que |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-bugs: delete_link removes any link regardless of who owns it; review-security: delete_link removes any link by code with no ownership check |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The /out endpoint is an open redirect with no host allowlist |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: admin_quotas authenticates with a caller-supplied header ins; review-security: The admin-only endpoint is gated on a client-supplied header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by the store size without guarding against z |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: The API sets Access-Control-Allow-Origin: * together with Ac |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: The unauthorized response echoes the presented token back to; review-security: The 401 response echoes the presented bearer token back to t |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: The /report endpoint exports every owner's links to any auth |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) store.cpp:27 — Store::create increments the owner quota even when it replaces an existing link
- (low) config.cpp:78 — validate() rejects a negative cache size but accepts a zero-sized cache
- (low) linkd.h:20 — The Link struct's expires_at has no default and prune and resolve use opposite senses of t
