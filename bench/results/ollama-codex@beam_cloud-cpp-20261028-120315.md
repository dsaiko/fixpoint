# ollama-codex@beam:cloud · cpp · run 20261028-120315 (repeat 1)

recall **22/64** · 24 finding(s), 1 unmatched · 1733378 tokens · 175s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-concurrency: std::rand() used without synchronization in Store::create an |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-security: Expired links remain resolvable |
| P03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| P04 | — | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  |  |
| P05 | — | parallel: C05 integer division before scaling yields 0, and the int/si |  |
| P06 | — | parallel: C09 remove drops the link but never returns the owner's quot |  |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-security: by_owner uses substring match and leaks nearby owners |
| P10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| P15 | — | CPP-ONLY last_code returns a string_view onto a function-local std::st |  |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET unset |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens minted with predictable std::rand |
| P19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Expired sessions are not rejected |
| P21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Weak unsalted password hash |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Stack buffer overflow in bearer_token |
| P24 | — | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no |  |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator mutates sessions_ without synchronization; ver |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | — | parallel: CA08 eviction drops the first key in map order, not the leas |  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-concurrency: Cache is not thread-safe: concurrent get/set/erase/sweep/pee |
| P31 | — | parallel: C05 utilization divides size_t by size_t before scaling so i |  |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| P36 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t |  |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | — | parallel: CF06 validate logs the problems it finds and returns true, s |  |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal via report name |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Export files rely on umask for confidentiality |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-security: Predictable /tmp snapshot path TOCTOU |
| P44 | — | CPP-ONLY format_row returns a pointer to a function-local char array,  |  |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are non-atomic shared across request thread |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-concurrency: Janitor erases from the links map while iterating it, concur |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-concurrency: warm_cache spawns threads and never joins them (terminates p; review-concurrency: stop_janitor only flips a flag: the janitor thread is never  |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | — | CPP-ONLY check_targets loops with <= and reads one element past the en |  |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Owner taken from query instead of session |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Delete endpoint has no ownership or auth check |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via ?next= |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts client-controlled X-Admin header |
| P58 | — | parallel: C20 /stats divides by the link count: division by zero, whic |  |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS with credentials |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown token value leaked in 401 response |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) store.cpp:10 — Store has no synchronization while shared by janitor, handlers, and worker threads
