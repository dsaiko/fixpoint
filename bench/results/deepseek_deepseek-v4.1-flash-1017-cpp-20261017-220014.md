# deepseek/deepseek-v4.1-flash-1017 · cpp · run 20261017-220014 (repeat 1)

recall **54/64** · 73 finding(s), 3 unmatched · 160251 tokens · 544s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Short-link codes come from unseeded rand() and silently over |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry test in Store::resolve is inverted |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Store::rename validates the source code twice and never vali; review-security: rename validates the source code twice and never validates t |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: Store::page builds a range past end of vector |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate performs integer division before multiplying |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() does not give the quota slot back |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: Store::top returns the least-hit links and pads the result w |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner matches owners by substring; review-security: by_owner matches owners by substring, leaking other owners'  |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: quotas() exposes the live map despite documenting a snapshot |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic despite its contract |
| P13 | — | CPP-ONLY prune erases through the iterator and then increments it: the |  |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Store::prune erases from a map while iterating and returns t |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code returns a string_view to a local string |
| P16 | YES | CPP-ONLY create returns a raw pointer into the map; any later erase or | review-concurrency: Store::create returns a raw Link* into a map the janitor can |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated from unseeded std::rand() and a |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are written to stdout and end up in logs |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify ignores session expiry; review-security: Authenticator::verify never checks expires_at, so sessions n |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin can never return true |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Passwords are stored using unsalted DJB2 instead of a passwo |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: bearer_token copies an unbounded header into a 64-byte stack; review-security: Unbounded strcpy of the Authorization header overflows a 64- |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token dereferences strchr result without a null check; review-security: Null-pointer dereference in bearer_token on requests without |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals is not constant-time despite its contract |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator::verify returns a pointer into sessions_ that  |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache never deletes its Entry objects |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache evicts entries_.begin() instead of the least recently  |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: Cache::sweep erases the current element and then advances th; review-concurrency: Cache::sweep mutates entries_ on the janitor thread with no  |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization divides integers and can divide by zero |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: Bad LINKD_TIMEOUT_MS logs that the default is kept but overw |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond environment variables are interpreted as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() reports problems but always returns true |
| P39 | — | parallel: EX01 the report name comes from the query string and is conc |  |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Reports containing every owner's links are written world-rea |
| P41 | YES | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in | review-security: CSV formula/row injection through unescaped owner and target |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Owner string is used as a filename in archive_all without sa |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot ignores write and rename failures; review-security: Snapshot is written through a fixed, predictable /tmp path ( |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns a pointer to a stack buffer; review-security: format_row returns a pointer to a local stack buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is never initialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are plain longs incremented from concurrent |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor thread erases from the store map while iterating it; review-concurrency: janitor walks and erases Store's std::map while request hand |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache destroys joinable std::thread objects; review-bugs: stop_janitor does not join or delete the janitor thread |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets loops with i <= links.size(); review-security: Off-by-one out-of-bounds read in check_targets |
| P51 | YES | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en | review-bugs: count_for_owner inserts into the quota map as a side effect; review-concurrency: count_for_owner mutates the shared quota map on a read path  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-bugs: Missing query parameters throw uncaught exceptions |
| P54 | YES | parallel: C28 the link owner comes from the query string instead of th | review-security: create_link trusts the client-supplied owner instead of the  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: delete_link lets any authenticated user delete any owner's l |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out via unvalidated next parameter |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on a client-supplied x-admin heade |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats divides by zero on an empty store; review-concurrency: stats() divides by a size read that the janitor can make zer |
| P59 | YES | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv | review-bugs: std::tolower is called on a possibly negative char |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin with credentials enabled |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response reflects the presented token back to the caller |
| P62 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: CSV export returns every owner's links to any authenticated  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report endpoint exports the default name but reads a differe; review-security: Path traversal in the report name: arbitrary .csv write and  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) store.cpp:58 — Store::rename uses a reference to an element after erasing it
- (medium) config.cpp:78 — validate accepts cache_size == 0
- (low) export.cpp:26 — export_csv does not check that the output file opened or was written
