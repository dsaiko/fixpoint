# vllm@bottlecapai/ThinkingCap-Qwen3.8-27B · cpp · run 20261005-025630 (repeat 1)

recall **50/64** · 73 finding(s), 2 unmatched · 62873 tokens · 1014s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Short-link codes generated with unseeded std::rand() - predi |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() inverts the expiry check: live links deleted on fi; review-security: resolve() deletes still-valid links: inverted expiry compari |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates 'from' twice and never validates 'to'; review-security: rename() validates the 'from' code twice and never validates |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() clamps end to all.size() + 1, constructing a vector o; review-security: page() returns an iterator range one element past the end (u |
| P05 | — | parallel: C05 integer division before scaling yields 0, and the int/si |  |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() never returns the owner's quota slot, despite its d; review-security: remove() never returns the owner's quota slot, contrary to i |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending by hits, returning the n LEAST followe |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches by substring, so one owner's listing incl; review-security: by_owner uses substring match - owner 'bob' retrieves 'bobsm |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, contradicting its 'expiry is |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: quotas() returns the live quota map by mutable reference; co |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic: a mid-batch validation throw lea |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the remaining link count instead of how many |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a string_view into a destroyed local (da; review-security: last_code returns a string_view into a local string (danglin |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret 'dev-secret-do-not-use' when LINKD |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens generated with unseeded std::rand() - determi |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued session tokens written to stdout (secrets in logs) |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks expires_at — expired tokens remain val; review-security: verify() never enforces session expiry - the documented 12-h |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() condition is a tautology — always returns fa; review-security: require_admin is a tautology that rejects every session |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Password 'hash' is unsalted djb2 truncated to 32 bits |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: bearer_token() strcpy into 64-byte stack buffer overflows on; review-security: Stack buffer overflow in bearer_token from attacker-controll |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token() dereferences null when the header has no spac; review-security: Null dereference in bearer_token: every request without a sp |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals is not constant-time despite its documented pu |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: verify() returns a raw Session* into sessions_ that concurre |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache leaks every Entry: eviction, erase, sweep, and destruc |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache eviction picks the lexicographically smallest key, not |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache::sweep() erases while iterating: ++it on an invalidate |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() integer-divides before multiplying (always 0%  |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: LINKD_TIMEOUT_MS: prints 'keeping default' but applies the p |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded — the knob ha |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true — startup validation is a no-; review-security: validate() always returns true - invalid configurations are  |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal via report name: writes the full store dump t |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Multi-tenant CSV exports created world-readable despite the  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot() renames into place before the ofstream flus; review-security: Snapshot written to a fixed world-writable /tmp path: symlin |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns a pointer to a stack buffer; review-security: format_row returns a pointer to a local stack buffer (dangli |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is uninitialized and has no setter — errors; review-security: Metrics::errors_ is an uninitialized member - /stats reads i |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are plain longs incremented from concurrent |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor erases from the store map while iterating: ++it on a; review-concurrency: Janitor thread mutates Store::links_ with no lock while requ |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-concurrency: warm_cache spawns threads it never joins: std::terminate on ; review-concurrency: stop_janitor never joins the janitor thread: torn-down state |
| P49 | YES | CPP-ONLY the thread lambda captures the loop variable by reference, so | review-bugs: warm_cache() destroys joinable threads without joining (std: |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() off-by-one: i <= links.size() reads one elem; review-security: check_targets off-by-one: out-of-bounds read of links[i] at  |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-security: Uncaught std::out_of_range from req.query.at() terminates th |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link owner taken from the query string instead of the authen |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: DELETE /links has no ownership check: any session can delete |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out?next= sends a permanent 301 to any attac |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by a client-supplied x-admin heade |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by g_store->size() — SIGFPE crash on an empt; review-security: Integer division by zero in /stats when the store is empty ( |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Access-Control-Allow-Origin: * combined with Access-Control- |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response reflects the presented token back to the client |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report returns every owner's links to any authenticated use |
| P63 | YES | every service object is heap-allocated with raw new and never deleted, | review-bugs: main() prints 'listening on port ...' and exits — no server  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with no name writes 'links.csv' but reads back '.csv |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) cache.cpp:31 — peek() reports expired entries as cached, disagreeing with get()
- (medium) store.cpp:11 — Targets accepted with only a prefix check - CRLF and other control bytes can reach the Loc
