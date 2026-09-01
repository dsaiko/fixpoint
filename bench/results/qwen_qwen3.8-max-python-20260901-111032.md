# qwen/qwen3.8-max · python · run 20260901-111032 (repeat 1)

recall **48/66** · 65 finding(s), 5 unmatched · 55905 tokens · 1255s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create() shares one mutable list across all links via tags=[ |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError for every new owner |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: deletes live links, serv; review-security: resolve() expiry check is inverted: live links are deleted o |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() truncates to 0 (or 100) due to floor-division |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the n LEAST followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() uses substring match instead of equality; review-security: by_owner matches by substring, leaking other owners' links |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links; review-security: extend() revives expired links despite documented finality o |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live dict, not the promised snapshot |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic despite its all-or-nothing contra; review-concurrency: import_all is not atomic despite its all-or-nothing contract |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() deletes while iterating the dict and returns the wro |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Link codes are 32-bit values from the predictable Mersenne T |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens are generated with predictable random.getrand |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are printed to the log at issuance |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks session expiry; review-security: verify() never checks session expiry — tokens are valid fore |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always returns False |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords are hashed with unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token() raises IndexError on empty/malformed header;  |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals claims constant-time comparison but uses == |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke() raises KeyError for unknown tokens |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction is FIFO, not LRU: the recorded 'used' timestamps ar |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() deletes entries while iterating the dict; review-concurrency: Cache.sweep deletes from the dict it is iterating — RuntimeE |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() truncates to 0 due to floor-division order |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks: calls delete() while holding the non-reen; review-concurrency: Cache.flush deadlocks: re-acquires its own non-reentrant loc |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS is used without converting milliseconds; ba; review-bugs: LINKD_CACHE_TTL_MS is neither converted from milliseconds no |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() never rejects an invalid configuration |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in report name allows arbitrary *.csv file wr |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Exports promised to be service-user-only are written with de |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-concurrency: Concurrent exports with the same name write to the same file |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot() renames before the file is flushed/closed; ; review-concurrency: write_snapshot uses a fixed shared tmp path — concurrent sna |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report() uses a mutable default list that accumulates  |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes from store.links while iterating find_expire; review-concurrency: Janitor thread dies: dict mutated while iterating the lazy f |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache() returns before its worker threads finish; review-concurrency: warm_cache starts threads but never joins them, contradictin |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() raises IndexError when every target is fine |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner() raises KeyError for owners with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters incremented without synchronization lose up |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Presented bearer token is echoed back in the 401 response |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner is taken from the request body instead of the aut |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete any link (no ownership che |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out?next= with no validation |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by client-supplied x-admin header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats() divides by zero on an empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS combined with Access-Control-Allow-Credentials |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports every owner's links to any authenticated use |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with default name writes links.csv but reads ".csv" |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) store.py:46 — Unprotected read-modify-write on Store.quota loses increments under concurrent creates
- (medium) auth.py:43 — Authenticator.verify iterates sessions dict while issue/revoke mutate it
- (medium) store.py:98 — Store.delete check-then-act TOCTOU: concurrent deletes raise uncaught KeyError
- (low) worker.py:29 — Janitor stop flag never reset: start_janitor after stop_janitor spawns a thread that exits
- (medium) export.py:58 — read_report path traversal lets a user read arbitrary .csv files on the host
