# claude-fable-5-1 · python · run 20260902-081433 (repeat 1)

recall **52/66** · 70 finding(s), 4 unmatched · 36670 tokens · 411s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create() shares one tags list between every link created wit |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError on an owner's first link |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links are deleted,  |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates frm twice and never validates to; review-concurrency: rename() check-then-act on the destination code |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() uses floor division and can only return 0 or  |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() does substring matching, not equality; review-security: `by_owner` uses substring matching and returns other owners' |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live dict, not a snapshot |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic and silently overwrites existing  |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() mutates the dict during iteration and returns the wr |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret silently used when LINKD_SECRET is |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens come from `random` (Mersenne Twister) and are |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens are written to stdout/logs in plaintext |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks expires_at, so tokens live forever; review-security: `verify` never checks `expires_at`; sessions are valid forev |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always returns False; review-bugs: revoke() raises KeyError for an unknown or already-revoked t |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with a single unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token() raises IndexError on a missing or malformed A; review-security: `bearer_token` raises on a missing or malformed Authorizatio |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: `secret_equals` is a plain `==`, not the constant-time compa |
| Y25 | YES | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a | review-bugs: parse_lifetime() swallows every exception and treats garbage |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: peek()/warm() bypass the lock the class promises every metho |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction is insertion-order, not LRU, and crashes when limit |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() deletes from the dict it is iterating and raises Run; review-concurrency: sweep() deletes from entries while iterating it, crashing th |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() floors to 0 and divides by zero when limit is  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks: calls delete() while holding the non-reen; review-concurrency: flush() re-acquires the non-reentrant cache lock via delete( |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS logs 'keeping default' but sets timeout |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is documented as milliseconds but stored  |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| Y38 | YES | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, | review-bugs: load_file() swallows every error and crashes on a non-numeri |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() always returns True, so a bad configuration is ne |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in report name: arbitrary file write and read |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Reports are created world-readable despite the comment claim |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV rows are built by string concatenation with no quoting; review-security: Unescaped user-controlled fields in CSV (column breakage and |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: `archive_all` builds file names from owner strings without s |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot renames the temp file before its bytes are fl; review-concurrency: write_snapshot uses one fixed temp path for all callers and  |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report accumulates rows across calls and emits a garba |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes from store.links while consuming a generator; review-concurrency: Janitor mutates store.links while iterating it and shares th |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache() returns before its threads finish; review-concurrency: warm_cache() starts threads but never joins them despite pro |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() raises IndexError when every target is fine |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner() raises KeyError for owners with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-concurrency: Janitor is non-daemon, sleeps before checking the stop flag, |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are unsynchronized read-modify-write across |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: create_link crashes on malformed input instead of returning ; review-security: Link owner is taken from the request body, not the authentic |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete any link (no ownership che |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: `/out?next=` forwards to any URL |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint is gated by a client-supplied `x-admin: true` |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: /stats raises ZeroDivisionError when the store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows every origin with credentials on an authenticate |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: `/report` hands every owner's links to any authenticated use |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-security: Error response leaks the server-side export path and raw exc |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with no ?name writes links.csv but reads back .csv; review-concurrency: report() writes and then re-reads a shared file by name, rac |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) store.py:36 — new_code() collisions silently overwrite an existing link
- (high) auth.py:43 — verify() iterates the sessions dict without a lock while issue()/revoke() mutate it
- (medium) store.py:58 — resolve() check-then-act on links and unsynchronized hits += 1
- (medium) store.py:34 — Target validation is prefix-only; CR/LF in a target enables response header injection
