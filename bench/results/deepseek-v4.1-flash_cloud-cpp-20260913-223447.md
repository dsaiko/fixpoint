# deepseek-v4.1-flash:cloud · cpp · run 20260913-223447 (repeat 1)

recall **45/64** · 58 finding(s), 2 unmatched · 180248 tokens · 474s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | — | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer |  |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry test in resolve() is inverted: live links are erased,; review-security: Inverted expiry comparison: live links are destroyed and exp |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() sizes its result one element past the end of the vect |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() does integer division before multiplying, so  |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() does not release the owner's quota slot |
| P07 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| P08 | YES | CPP-ONLY top calls resize(n) without clamping, so asking for more link | review-bugs: top() returns the least-followed links and pads the result w |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() uses a substring match instead of equality |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: quotas() hands out a mutable reference to the store's intern |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic as documented |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune() increments an iterator it has just invalidated |
| P14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() reports the remaining size, not the number removed |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns a string_view into a destroyed local str |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens generated with unseeded std::rand(), making t |
| P19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to stdout on every issue |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks token expiry despite its documented co; review-security: verify() ignores expires_at, so sessions never expire |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() can only ever return false |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: hash_password is a single-pass 64-bit djb2, not a password h |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-bugs: bearer_token copies an unbounded header into a 64-byte stack; review-security: bearer_token: stack buffer overflow and null-pointer derefer |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token dereferences a null pointer when the header has |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals returns on the first mismatching byte, contrad |
| P26 | — | CPP-ONLY verify hands out a pointer into the sessions map; a concurren |  |
| P27 | YES | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele | review-bugs: Cache entries are never freed: every erase, sweep and overwr |
| P28 | — | CPP-ONLY the destructor clears the map without deleting the Entry poin |  |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache eviction picks the lowest code, not the least recently |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep() erases entries while iterating the same map; review-concurrency: Cache has no synchronization: hits_, entries_ and Entry::use |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() truncates to 0 in integer division and divides |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | — | parallel: CF02 the failure branch logs 'keeping default' and then assi |  |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS is milliseconds but is stored as seconds, a; review-bugs: LINKD_CACHE_TTL_MS is milliseconds but is stored as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed and thrown away |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true, so startup validation never  |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: Path traversal in the report name: arbitrary .csv write and  |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Report files are created world-readable despite the comment  |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot() renames the file before the stream is flush; review-security: write_snapshot uses a fixed, predictable /tmp path (symlink  |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row() returns a pointer to a stack buffer that no lon |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is the only counter left uninitialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are non-atomic shared state incremented fro |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor loop erases from the store map while iterating it; review-concurrency: Janitor thread mutates store.all() concurrently with request |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache() never joins its threads, so the vector destruct; review-concurrency: warm_cache detaches no worker and joins none, so ~vector<thr |
| P49 | — | CPP-ONLY the thread lambda captures the loop variable by reference, so |  |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets() indexes one element past the end of the vect |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-security: Uncaught std::out_of_range from map::at on missing query par |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: delete_link performs no ownership check: any caller can dele |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: out() throws out_of_range on a missing next instead of retur; review-security: Open redirect: /out forwards to an unvalidated ?next= value |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization decided by a client-supplied x-admin hea |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by the store size with no zero check; review-security: Division by zero in /stats when the store is empty (SIGFPE c |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Presented token reflected unescaped into the 401 response bo |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() writes links.csv for an unnamed report but reads ba |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) linkd.h:43 — Store::all() exports a mutable reference to the internal map, defeating any synchronizatio
- (medium) main.cpp:48 — Store::create returns a pointer into the map, which the janitor thread can erase before th
