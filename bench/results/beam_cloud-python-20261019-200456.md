# beam:cloud · python · run 20261019-200456 (repeat 2)

recall **50/66** · 68 finding(s), 4 unmatched · 74425 tokens · 78s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create() mutable default tags list is shared; review-concurrency: Store has no synchronization despite being shared across thr |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() crashes on first link for any owner |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes live links and keeps expired ones (inverte; review-security: Expired links still resolve (expiry logic inverted) |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates frm twice, never validates to |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: page() off-by-one upper bound |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() integer division loses fractional results |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not decrement the owner quota |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns least-used links instead of most-used |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() uses substring match, not equality; review-security: by_owner uses substring matching |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links; review-security: extend() revives expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live internal dict |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic and skips quota |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-concurrency: prune mutates self.links while iterating it |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() mutates dict while iterating and returns wrong count |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-bugs: find_expired() returns a live generator over mutating dict |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded weak default secret |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens minted with non-cryptographic RNG |
| Y19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() ignores session expiry; review-security: Session expiry is never enforced |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() can never return True |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token() crashes on missing/malformed header |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: Constant-time comparison not actually constant-time |
| Y25 | YES | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a | review-bugs: parse_lifetime() bare except hides bad input |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke() raises KeyError on unknown token |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-bugs: peek() is not thread-safe and reports stale entries; review-concurrency: peek, warm, stats, and utilization access shared state witho |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: set() evicts the wrong entry (not LRU) |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() mutates the dict while iterating; review-concurrency: sweep mutates self.entries while iterating it |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() integer division truncates to 0 |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks acquiring the lock twice; review-concurrency: flush deadlocks on its own non-reentrant lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: LINKD_TIMEOUT_MS treated as seconds, not milliseconds |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS treated as seconds, not milliseconds; review-bugs: cache_ttl/c size env parsing can crash startup on bad input |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() always returns True |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in export_csv via user-controlled name |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot() leaks handle and skips close/flush; review-concurrency: write_snapshot uses a fixed shared temp path, racing across  |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report() mutable default accumulator; review-concurrency: parse_report's mutable default argument is shared across all |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-concurrency: Janitor deletes from store.links while iterating a generator |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache() returns before threads finish; review-concurrency: warm_cache starts worker threads and returns without waiting |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() raises IndexError when there are no problems |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner() KeyError when owner has no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are mutated and read without synchronizatio |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | YES | parallel: S06 an absent Authorization header becomes the empty string, | review-bugs: handle() crashes when Authorization header is missing |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Token echoed back in 401 response |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Owner is taken from the client payload, enabling impersonati |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-bugs: delete_link() ignores ownership and session; review-security: Any authenticated user can delete any link (IDOR) |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unvalidated ?next= in /out |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: admin_quotas() authorizes via a spoofable header; review-security: Admin endpoint gated on a client-supplied x-admin header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats() divides by zero on empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS with credentials enabled |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() name mismatch between write and read |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) cache.py:56 — warm() crashes when given a mapping
- (low) config.py:72 — load_file() leaks an open file handle
- (medium) auth.py:23 — Authenticator.sessions is mutated and iterated without synchronization
- (low) worker.py:29 — Janitor thread is neither a daemon nor joined on stop
