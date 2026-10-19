# ollama-codex@beam:cloud · cpp · run 20261019-210058 (repeat 1)

recall **48/64** · 86 finding(s), 5 unmatched · 2013767 tokens · 196s

| seed | found | note | matched by |
|---|---|---|---|
| P01 | YES | CPP-ONLY (parallel: S02) codes come from std::rand into a fixed buffer | review-security: Short link codes use std::rand |
| P02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes live links and accepts expired ones; review-security: resolve() has inverted expiry logic |
| P03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: rename() validates `from` twice and does not validate `to`; review-security: rename() does not validate the destination code |
| P04 | YES | CPP-ONLY (parallel: C04) page builds an iterator one past the end and  | review-bugs: page() off-by-one end index; review-security: page() computes an out-of-bounds slice end |
| P05 | YES | parallel: C05 integer division before scaling yields 0, and the int/si | review-bugs: success_rate() truncates to 0 due to integer division; review-security: success_rate() reports zero for most stores |
| P06 | YES | parallel: C09 remove drops the link but never returns the owner's quot | review-bugs: remove() does not return the owner's quota slot |
| P07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns least-followed links in possibly oversized vec; review-security: top() sorts ascending and can synthesize links |
| P08 | — | CPP-ONLY top calls resize(n) without clamping, so asking for more link |  |
| P09 | YES | parallel: C11 by_owner matches owners by substring, so one owner sees  | review-bugs: by_owner() matches substrings instead of exact owner; review-security: by_owner() matches substrings instead of exact owners |
| P10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links; review-security: extend() revives expired links |
| P11 | YES | parallel: C14 quotas hands out a non-const reference to the store's ow | review-bugs: quotas() returns mutable reference to internal map; review-security: quotas() returns a mutable internal reference |
| P12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic and overrides duplicates; review-security: import_all is not atomic and can overwrite duplicates |
| P13 | YES | CPP-ONLY prune erases through the iterator and then increments it: the | review-bugs: prune() invalidates iterator and returns wrong count; review-security: prune() invalidates its iterator while erasing |
| P14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| P15 | YES | CPP-ONLY last_code returns a string_view onto a function-local std::st | review-bugs: last_code() returns dangling string_view; review-security: last_code() returns a dangling string_view |
| P16 | — | CPP-ONLY create returns a raw pointer into the map; any later erase or |  |
| P17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret is used when the environment is un |
| P18 | YES | parallel: S02 session tokens come from std::rand, never seeded, writte | review-security: Session tokens are generated from std::rand |
| P19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| P20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks session expiry; review-security: verify() ignores session expiry |
| P21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-security: require_admin() rejects every session |
| P22 | YES | parallel: S05 passwords are hashed with a djb2 accumulator: unsalted,  | review-security: Password storage uses a weak unsalted digest |
| P23 | YES | CPP-ONLY bearer_token strcpy's an attacker-controlled header into a 64 | review-security: bearer_token copies an unbounded header into a fixed buffer |
| P24 | YES | CPP-ONLY (parallel: S06) strchr returns nullptr when the header has no | review-bugs: bearer_token assumes a space and fixed 64-byte buffer |
| P25 | YES | secret_equals is documented as not leaking how much matched and return | review-security: secret_equals leaks timing information |
| P26 | YES | CPP-ONLY verify hands out a pointer into the sessions map; a concurren | review-concurrency: Authenticator sessions map used without synchronization; ver |
| P27 | — | CPP-ONLY entries are heap-allocated raw pointers and nothing ever dele |  |
| P28 | YES | CPP-ONLY the destructor clears the map without deleting the Entry poin | review-bugs: Cache destructor leaks Entry* objects; review-bugs: get() leaks expired Entry* after erase |
| P29 | YES | parallel: CA08 eviction drops the first key in map order, not the leas | review-bugs: Cache::set leaks the previous entry for an existing code; review-bugs: Eviction does not select the least recently used entry |
| P30 | YES | CPP-ONLY sweep erases through the iterator it then increments: undefin | review-bugs: sweep() invalidates its iterator; review-security: sweep() invalidates its iterator while erasing |
| P31 | YES | parallel: C05 utilization divides size_t by size_t before scaling so i | review-bugs: utilization() loses precision via integer division; review-security: utilization() reports zero for non-full caches |
| P32 | — | parallel: CA02 the cache has no mutex at all, while the janitor thread |  |
| P33 | — | parallel: CF01 atoi cannot report failure and returns 0, so a malforme |  |
| P34 | YES | parallel: CF02 the failure branch logs 'keeping default' and then assi | review-bugs: LINKD_TIMEOUT_MS is treated as seconds, not milliseconds |
| P35 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is treated as seconds, not milliseconds; review-security: LINKD_CACHE_TTL_MS is treated as seconds |
| P36 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and discarded; t | review-bugs: LINKD_FETCH_LIMIT is parsed but discarded; review-security: LINKD_FETCH_LIMIT is parsed but discarded |
| P37 | — | parallel: CF05 load_file never checks whether the stream opened, so an |  |
| P38 | YES | parallel: CF06 validate logs the problems it finds and returns true, s | review-bugs: validate() always returns true; review-security: validate() returns success even when validation fails |
| P39 | YES | parallel: EX01 the report name comes from the query string and is conc | review-security: export_csv permits path traversal through the report name |
| P40 | YES | parallel: EX02 the report holding every owner's links is written with  | review-security: Exported reports are created with default world-readable per |
| P41 | — | parallel: EX03 CSV rows are streamed unescaped, so a comma or quote in |  |
| P42 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all permits path traversal through owner names |
| P43 | YES | CPP-ONLY (parallel: EX06/EX07) one fixed /tmp name, the rename's resul | review-bugs: write_snapshot uses fixed /tmp path and cross-filesystem ren; review-security: write_snapshot uses a predictable fixed temporary path |
| P44 | YES | CPP-ONLY format_row returns a pointer to a function-local char array,  | review-bugs: format_row returns dangling pointer to stack buffer; review-security: format_row returns a pointer to a destroyed stack buffer |
| P45 | YES | CPP-ONLY the errors_ counter has no initializer while its siblings do, | review-bugs: Metrics::errors_ is uninitialized; review-security: Metrics::errors_ is uninitialized |
| P46 | YES | parallel: X01 the counters are plain longs incremented from the janito | review-concurrency: Metrics counters are non-atomic and read without synchroniza |
| P47 | YES | CPP-ONLY (parallel: X02) the janitor erases through the iterator it th | review-bugs: Janitor mutates store.all() while invalidating iterator; review-concurrency: Janitor erases store entries while iterating the returned ma |
| P48 | YES | CPP-ONLY warm_cache never joins its threads, so the vector's destructo | review-bugs: warm_cache threads are never joined; review-bugs: stop_janitor does not join the janitor thread |
| P49 | YES | CPP-ONLY the thread lambda captures the loop variable by reference, so | review-concurrency: warm_cache spawns per-link threads without joining them |
| P50 | YES | CPP-ONLY check_targets loops with <= and reads one element past the en | review-bugs: check_targets reads one past the vector end; review-security: check_targets reads one element past the vector end |
| P51 | — | CPP-ONLY count_for_owner uses map::operator[], which INSERTS a zero en |  |
| P52 | — | CPP-ONLY the janitor thread is heap-allocated, never joined and never  |  |
| P53 | YES | CPP-ONLY the handler reads query parameters with .at(), which throws s | review-security: create_link assumes required query parameters exist |
| P54 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| P55 | YES | parallel: C17 delete_link never checks the owner its comment promises  | review-security: delete_link lacks ownership authorization |
| P56 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| P57 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: admin_quotas trusts a spoofable x-admin header |
| P58 | YES | parallel: C20 /stats divides by the link count: division by zero, whic | review-bugs: stats() divides by store size without zero check; review-security: stats() divides by store size without checking zero |
| P59 | — | CPP-ONLY (parallel: C16) the code is lowercased before a case-sensitiv |  |
| P60 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows credentials with a wildcard origin |
| P61 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Authentication errors leak the submitted token |
| P62 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| P63 | — | every service object is heap-allocated with raw new and never deleted, |  |
| P64 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.cpp:49 — require_admin always returns false
- (critical) linkd.h:49 — Store links_ and quota_ mutated concurrently without synchronization
- (high) linkd.h:101 — Cache entries_ and hits_ accessed from multiple threads without a lock
- (medium) config.cpp:21 — LINKD_TIMEOUT_MS is mishandled and treated as seconds
- (medium) linkd.h:43 — all() exposes the private link map by mutable reference
