# gpt-6.1-terra · cpp · run 20261029-094659 (repeat 1)

recall **40/64** · 51 finding(s), 7 unmatched · 182769 tokens · 101s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the source twice instead of the destination |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Page reads past the end of the link vector |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: Success percentage truncates before multiplication |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: Removing a link does not decrement its owner's count |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: Leaderboard returns least-followed links and can add empty r |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: Owner lookup matches substrings |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending an expired link revives it |
| P11 | — | parallel: C14 quotas hands out a non-const reference to the store's ow |  |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Failed batch import leaves a partial update |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune invalidates its iterator and returns the wrong count |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: Last-code view refers to a destroyed string |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are predictable |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued bearer tokens are written to logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions remain valid; review-security: Expired sessions remain valid |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Admin role check rejects every role |
| P22 | — | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  |  |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: Authorization header can overflow the stack before authentic |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: Missing or malformed authorization headers cause invalid poi |
| P25 | — | secret_equals is documented as not leaking how much matched and return |  |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache entries are never freed |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache evicts by key instead of least recent use |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache sweep increments an erased iterator |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: Cache utilization truncates before multiplication |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond configuration values are treated as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: Fetch limit environment setting is ignored |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: Configuration validation always succeeds |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Report name permits writes outside the export directory |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Reports can be created with permissions readable by other lo |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-bugs: CSV output does not escape field values |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: Snapshot is renamed before its buffered contents are flushed |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: Formatted row returns a pointer to a local array |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Error metric is uninitialized |
| P46 | — | parallel: X01 the counters are plain longs incremented from the janito |  |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor increments an erased iterator; review-concurrency: Janitor accesses the store concurrently without synchronizat |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: Warming a nonempty cache terminates the process; review-concurrency: Cache warming destroys joinable threads |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: Target checking reads one element beyond the vector |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Invalid requests can throw out of the request handler |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: Link ownership is supplied by the caller |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: Any authenticated user can delete another owner's links |
| P56 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Administrator access is granted by a caller-controlled heade |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: Stats divides by zero for an empty store |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report exposes other users' links |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report reads a different filename from the one writt |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) store.cpp:59 — Rename uses a reference after erasing its link
- (high) main.cpp:152 — Service exits immediately after announcing it is listening
- (high) store.cpp:26 — Generated code collisions overwrite existing links
- (medium) export.cpp:26 — Report write failures are reported as success
- (high) cache.cpp:36 — Concurrent cache operations race on the entries map
- (high) export.cpp:48 — Concurrent snapshots share the same temporary file
- (high) main.cpp:115 — Concurrent reports with the same name overwrite each other
