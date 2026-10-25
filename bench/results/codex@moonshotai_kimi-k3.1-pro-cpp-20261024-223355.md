# codex@moonshotai/kimi-k3.1-pro · cpp · run 20261024-223355 (repeat 1)

recall **39/64** · 53 finding(s), 6 unmatched · 594547 tokens · 298s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiration check is inverted; review-security: Expired links are erased before use and nonexpired links der |
| P03 | — | parallel: C03 rename validates `from` twice; `to` is never validated |  |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Pagination can construct an out-of-range vector |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Success rate truncates to zero below full success |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Deleting and pruning links never updates owner quotas |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard sorting and limiting are reversed |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner filtering uses substring matching |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can resurrect an expired link |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Batch import is not atomic |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Pruning invalidates its iterator and returns the wrong count |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: Last-code view refers to a destroyed string |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Production authentication silently falls back to a hardcoded |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated with predictable rand output |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer session tokens are written to application logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Session verification never checks expiration; review-security: Expired sessions remain valid indefinitely |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Admin authorization rejects every role; review-security: Administrative role check rejects every session |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Passwords use a fast unsalted non-cryptographic hash |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Authorization parsing enables denial of service through memo |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Malformed Authorization headers cause out-of-bounds access |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache eviction and replacement leak entries and do not imple |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache sweep iterates invalidated iterators |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache utilization is always zero for a nonfull cache |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond configuration values are interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: Fetch limit environment setting is discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation always reports success |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report name permits arbitrary file write and traversal reads |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Sensitive reports are created with default file permissions |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-concurrency: Concurrent exports of the same report can serve a partial fi; review-security: Unescaped report fields allow CSV formula injection |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: Snapshot writes ignore all I/O and rename failures; review-security: Predictable shared snapshot file permits symlink attacks |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: Report formatter returns a dangling pointer |
| P45 | — | CPP-ONLY the errors_ counter has no initializer while its siblings do, |  |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Request metrics lose updates and race dashboard reads |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor cleanup uses invalid iterators |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: Cache warming destroys joinable threads before they finish; review-concurrency: Cache warm-up destroys joinable worker threads and never joi |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | — | CPP-ONLY check_targets loops with <= and reads one element past the en |  |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | — | CPP-ONLY the handler reads query parameters with .at(), which throws s |  |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete any link |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unrestricted next parameter creates an open redirect |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Client-controlled header bypasses administrator authorizatio |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Statistics on an empty store divide by zero |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: Redirect lowercases case-sensitive short codes |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Authentication failures reflect bearer tokens in responses |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exports every owner's private links |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) store.cpp:58 — Rename reads a link after erasing it
- (high) worker.cpp:43 — Target checking reads one element past the vector
- (high) linkd.h:49 — Store has no synchronization despite concurrent request and janitor access
- (high) linkd.h:101 — Cache operations race with the janitor and parallel warm-up
- (high) linkd.h:75 — Session verification returns pointers into an unsynchronized mutable map
- (high) export.cpp:48 — Concurrent snapshots share one temporary pathname
