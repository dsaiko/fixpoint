# gpt-6.2-astra · cpp · run 20261118-155842 (repeat 1)

recall **47/64** · 95 finding(s), 27 unmatched · 157623 tokens · 403s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it on its first visit; review-security: Resolving a live link deletes it, while expired links still  |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source code twice and accepts an invali |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: A short final page reads past the end of the link vector; review-bugs: Negative or overflowing page bounds produce invalid iterator |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: The dashboard reports zero success for every partially used  |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Owner quotas diverge from the links actually stored |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard selects the least-followed links first |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: Requesting more leaderboard entries than exist fabricates li |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup matches substrings instead of exact names; review-security: Owner-scoped lookups return other accounts' records |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten it or revive an expired code |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: The quota snapshot exposes mutable store state |
| P12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: Pruning an expired link invalidates the traversal iterator |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns the number of surviving links |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: The dashboard's last-code view dangles on return |
| P16 | YES | CPP-ONLY create returns a raw pointer into the map; any later erase or | review-concurrency: A newly created link can be erased before the response copie |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Bearer tokens come from a predictable random sequence |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Token issuance logs a reusable bearer credential |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions remain valid and accumulate indefinitely; review-security: Expired sessions continue to authorize requests |
| P21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Stored password hashes are cheap to guess offline |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: Long Authorization headers overflow the parser's local buffe; review-security: Authorization header can overwrite the stack before authenti |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: A missing Authorization header can crash request handling; review-security: A missing Authorization header can crash the handler |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Session revocation can invalidate an in-progress verificatio |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache entries leak on replacement, removal, and destruction; review-bugs: Cache eviction ignores recorded access times |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: A zero cache capacity permits invalid eviction and division  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Sweeping an expired cache entry invalidates the loop iterato |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache utilization displays zero until the cache is full |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: An invalid timeout overwrites the default the loader says it |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond settings are interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: A valid fetch-limit override is discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Invalid configuration never causes startup to fail |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report names allow existing files outside the export directo |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Exported owner data can be readable by other local accounts |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV exports do not preserve fields containing commas, quotes; review-security: User-controlled link fields can inject spreadsheet formulas  |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: Owner archival silently succeeds with missing or partial fil; review-security: Per-owner archive names can escape the archive directory |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: A snapshot replaces the previous copy before its contents ar; review-bugs: Snapshot replacement silently fails across filesystems |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: The row formatter returns a pointer to a destroyed stack arr |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: The error metric has no initialization for default construct |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Concurrent requests race on the dashboard counters |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: The janitor increments an erased map iterator; review-concurrency: The janitor and request handlers access the link map without |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: Warming even one cache entry terminates the process; review-concurrency: Warming any nonempty cache terminates the process |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | — | CPP-ONLY check_targets loops with <= and reads one element past the en |  |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Invalid request parameters escape the dispatcher as exceptio |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link creators can impersonate other owners |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete another owner's link |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Permanent outbound redirects can bypass click counting; review-security: The email click route redirects to any caller-supplied desti |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A caller-controlled header grants access to the admin quota  |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: The stats endpoint divides by zero for an empty store; review-concurrency: The statistics calculation can divide by zero as the last li |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: Redirect lookup changes valid mixed-case codes |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: An unnamed report downloads a different file from the one ju; review-concurrency: Overlapping report requests can return another request's CSV |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) store.cpp:26 — A generated code collision overwrites an existing link
- (high) main.cpp:152 — The service exits immediately after announcing that it is listening
- (high) main.cpp:147 — The running service has no way to establish its first session
- (high) store.cpp:59 — Renaming a link destroys it before copying it to the new key
- (high) main.cpp:115 — CSV downloads silently omit every link after the first 1,000
- (high) export.cpp:29 — A failed export is reported as a successful download
- (high) main.cpp:96 — The quota endpoint omits every owner's link count
- (medium) auth.cpp:49 — The role predicate rejects administrators
- (medium) store.cpp:148 — A rejected import leaves earlier records committed
- (medium) store.cpp:176 — Last-code selection uses lexical order instead of creation time
- (medium) store.cpp:11 — A scheme prefix alone passes absolute-target validation
- (medium) main.cpp:44 — TTL parsing silently accepts bad values and can overflow
- (medium) main.cpp:32 — CSV and plain-text responses are advertised as JSON
- (medium) cache.cpp:31 — Peek reports expired entries as cache hits
- (medium) cache.cpp:37 — Updating an existing key in a full cache can evict an unrelated key
- (medium) worker.cpp:43 — Target checking reads one element past the vector
- (medium) worker.cpp:44 — Target probes classify links by scheme without checking the destination
- (high) cache.cpp:37 — Parallel cache writers can select and erase the same eviction victim
- (high) worker.cpp:29 — Stopping the janitor can return before it uses the store and cache again
- (high) main.cpp:152 — The executable exits immediately after starting its background worker
- (high) store.cpp:27 — Concurrent link creations can lose an owner's quota increment
- (medium) export.cpp:48 — Concurrent snapshot writers share one temporary file across all destinations
- (medium) worker.cpp:14 — Repeated janitor starts lose the worker handle or start an inactive worker
- (high) main.cpp:115 — Ordinary sessions can export other owners' link targets
- (high) export.cpp:63 — Report download can disclose files that the service can read but cannot write
- (high) main.cpp:115 — Repeated report requests can exhaust the export filesystem
- (medium) store.cpp:26 — A short-code collision silently redirects an existing link to a new owner's target
