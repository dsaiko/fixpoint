# minimax-m3.1-flash:cloud · python · run 20261019-193825 (repeat 1)

recall **55/66** · 89 finding(s), 11 unmatched · 622805 tokens · 4345s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create() takes a mutable default argument for tags |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: First link for any owner raises KeyError instead of creating |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry comparison inverted: live links resolve to None and d |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: page() clamps the end index to len(all_links) + 1 |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() uses floor division, so any rate under 100% r |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never decrements the owner's quota counter |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts by ascending hits and returns the least-followed |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() matches the owner as a substring, so one owner se |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects an expired link, contradicting its own d |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live quota dict, contradicting its docs; review-concurrency: quotas() hands out the store's live quota dict, so a caller  |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic despite its docstring |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the number of links remaining instead of the |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-bugs: The authenticator falls back to a hardcoded shared secret; review-security: Hardcoded fallback secret lets the service start with a know |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-bugs: Session tokens are minted with random.getrandbits, which is ; review-security: Session tokens are minted with the non-cryptographic random  |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-bugs: Issued session tokens are printed to stdout; review-security: Session tokens are written to stdout in plaintext on every i |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() matches on session.token instead of the dict key an |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin's condition is a tautology that always returns; review-security: require_admin denies every role, including admins, because t |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-bugs: hash_password uses a bare SHA-256 with no salt and no work f; review-security: Passwords are hashed with unsalted SHA-256 |
| Y23 | — | parallel: S06 bearer_token indexes [1] without checking, so a request  |  |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-bugs: secret_equals uses == rather than a constant-time comparison; review-security: secret_equals compares with == and leaks match length throug |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke() raises KeyError when the token is unknown |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-bugs: peek() reads the cache without taking the lock |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction always drops the oldest inserted entry, never the l |
| Y29 | YES | parallel: CA06 warm's peek and set take the lock separately, so the ja | review-concurrency: warm()'s peek-then-set is a check-then-act race: the two ste |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() deletes from the dict while iterating it, raising Ru; review-concurrency: sweep() deletes from the dict while iterating it, raising Ru |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() uses floor division, so a half-full cache repo |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() calls delete() while already holding the lock; review-concurrency: flush() self-deadlocks: it holds the non-reentrant Lock and  |
| Y33 | YES | parallel: CA07 stats reads the hit counter outside the lock while get  | review-concurrency: stats() and utilization() read cache state without the lock |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is documented in milliseconds but stored ; review-bugs: LINKD_CACHE_SIZE and LINKD_CACHE_TTL_MS raise ValueError on  |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| Y38 | YES | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, | review-bugs: load_file() uses a bare except that swallows KeyboardInterru |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() reports configuration problems but always returns |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-bugs: export_csv does not confine the report name to the export di; review-security: Path traversal in export_csv lets a user write files outside |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-bugs: Report files are created with default permissions, readable  |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV fields are joined with commas without quoting or escapin |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: archive_all() builds per-owner file names from the owner val; review-security: archive_all writes per-owner files with an unsanitized owner |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot() uses a fixed temp path, so concurrent write; review-concurrency: write_snapshot uses one fixed /tmp path, so concurrent snaps |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report() uses a mutable default argument, so rows accu |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes from store.links while iterating it, raising; review-concurrency: Janitor deletes from store.links while consuming a generator |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache starts its threads but never joins them, so it re; review-concurrency: warm_cache starts threads it never joins, so it returns befo |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets raises IndexError when every target is valid |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner raises KeyError for an owner with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-bugs: stop_janitor() does not stop the janitor loop; review-concurrency: Janitor thread is not a daemon and is never joined, so the p |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are incremented without a lock; += is not a |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | YES | parallel: S06 an absent Authorization header becomes the empty string, | review-bugs: bearer_token() raises IndexError on a request with no Author |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: The 401 body echoes the presented token back to the client; review-security: 401 response echoes the presented bearer token back to the c |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: Malformed JSON body on POST /links raises an unhandled Value; review-security: create_link takes owner from the request body, letting any u |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-bugs: DELETE /links deletes any link by code, ignoring the caller'; review-security: delete_link removes any link without checking that the calle |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out reflects an arbitrary next parameter int |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin authorization for /admin/quotas reads a client-supplie; review-security: Admin quota endpoint authorizes on a client-supplied header  |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats() divides by zero when the store is empty |
| Y62 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: redirect() lowercases the code, so codes that are case-sensi |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-bugs: The CORS header pair is contradictory and blocks credentiale; review-security: CORS sends a wildcard origin together with Allow-Credentials |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-concurrency: Concurrent /report requests with the same name write the sam |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-bugs: The export report reads back a file by name instead of using; review-security: Export failures echo the exception text and absolute path ba |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) worker.py:58 — warm_cache unpacks a dict's keys as pairs, raising ValueError on every call
- (medium) export.py:58 — read_report leaks the file handle and raises on a missing report
- (low) main.py:68 — A ttl of 0 is silently replaced with the default because the check tests truthiness
- (low) main.py:78 — redirect() raises KeyError when the code query parameter is absent
- (low) cache.py:26 — get() treats a TTL of 0 as immediately expired
- (low) config.py:82 — load_file() raises ValueError on a non-numeric cache_size value
- (low) export.py:64 — parse_report() raises IndexError on an empty body
- (high) auth.py:43 — sessions dict is read while it is written, with no lock
- (high) store.py:45 — Store.links is mutated by the janitor thread while request handlers read and write it
- (low) store.py:60 — resolve()'s hit counter is incremented without synchronization
- (critical) export.py:58 — Path traversal in read_report reads and returns arbitrary files to the caller
