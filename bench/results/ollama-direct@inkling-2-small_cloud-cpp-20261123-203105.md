# ollama-direct@inkling-2-small:cloud · cpp · run 20261123-203105 (repeat 1)

recall **51/64** · 78 finding(s), 4 unmatched · 523292 tokens · 427s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Short-link codes from unseeded rand() are predictable |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted — live links are deleted,; review-security: Inverted expiry check in resolve(): live links deleted, dead |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and never validates `to`; review-security: rename() validates 'from' twice and never validates 'to' |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() computes end = all.size() + 1 and constructs an out-o; review-security: page() constructs vector range past end (heap out-of-bounds  |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() integer-divides to 0 (or 100) every time |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() never decrements the owner's quota |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: top() sorts ascending and pads with default-constructed Link |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner uses substring match instead of exact ownership; review-security: by_owner() uses substring match, leaking other owners' links |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links and does not add to expiry; review-security: extend() resurrects expired links |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: quotas() returns the internal map by reference despite docum; review-concurrency: quotas() leaks a mutable reference to the internal quota_ ma |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic despite its contract |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-concurrency: prune() erases map elements while iterating forward |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() erases through the loop iterator then increments it |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a string_view to a destroyed local; review-security: last_code returns string_view to a local that is destroyed |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are predictable (unseeded rand, 16 hex chars) |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to stdout logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks session expiry — expired sessions stay; review-security: verify() never checks expires_at; expired sessions never die |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-security: require_admin uses || so it can never grant admin |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Passwords hashed with unsalted djb2 |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: bearer_token copies unbounded header into 64-byte stack buff; review-security: Stack buffer overflow in bearer_token via Authorization head |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token dereferences nullptr when header has no space |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: verify() returns a pointer into the sessions_ map; concurren |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache leaks every Entry — entries_ holds raw pointers that a |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: cache eviction picks the lexicographically smallest key, not |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep() erases map entries while iterating with ++it; review-concurrency: sweep() erases the current element while iterating |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() integer-divides and divides by zero when limit |
| P32 | YES | parallel: CA02 the cache has no mutex at all, while the janitor thread | review-concurrency: Cache is not thread-safe; janitor and handlers race on entri |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: LINKD_TIMEOUT_MS / LINKD_CACHE_TTL_MS are milliseconds but p |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true — bad configurations are neve; review-bugs: load_file accepts values without applying the same checks as |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal in export name: arbitrary file write and read |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-bugs: File-open and write failures are silently ignored; review-security: Report file created with default (likely world-readable) per |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-security: CSV formula injection (unescaped fields starting with =, +,  |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all writes per-owner files using the attacker-contro |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot renames before the ofstream is flushed/closed; review-security: Snapshot written to predictable fixed /tmp path, never fsync |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to a dead stack buffer; review-security: format_row returns pointer to destroyed stack buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is never initialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are plain long incremented from all request |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor erases from the map while iterating with ++it; review-concurrency: Janitor erases from the map while iterating it (iterator inv |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache never joins its worker threads — std::terminate o; review-bugs: stop_janitor() never joins the thread |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads one past the end of the vector; review-security: check_targets iterates past the end (off-by-one) |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: count_for_owner inserts a zero entry for unknown owners |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Unguarded req.query.at(...) throws out of handlers on missin |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: create_link takes owner from query string, not the session |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: delete_link lets any authenticated user delete any link (IDO |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out via unvalidated next parameter |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by client-supplied 'x-admin: true' |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: /stats divides by zero when the store is empty; review-security: /stats divides by g_store->size() — crash on empty store |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin with credentials |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports every owner's links to any authenticated use |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.cpp:40 — require_admin logic is wrong: admins are always rejected
- (medium) cache.cpp:31 — peek() reports expired entries as cached
- (medium) store.cpp:26 — create() silently overwrites an existing link on code collision
- (medium) config.cpp:100 — validate() always returns true — fail-open configuration gate
