# claude-sonnet-5-5 · python · run 20260929-100550 (repeat 1)

recall **51/66** · 71 finding(s), 5 unmatched · 22691 tokens · 181s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: Mutable default `tags=[]` is shared between links; review-concurrency: create() uses a mutable default tags=[] that every link shar |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError for an owner's first link; review-concurrency: Store mutates shared dicts with no lock (check-then-act and  |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename validates the old code twice and never validates the  |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate uses integer division before multiplying |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not return the owner's quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner matches substrings, not the exact owner; review-security: by_owner matches owners by substring |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live dict, not a snapshot |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() mutates the dict while iterating and returns the wro |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Falls back to a hardcoded default secret when LINKD_SECRET i |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens and link codes come from the non-cryptographi |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to stdout logs in the clear |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() ignores session expiry; review-security: Expired sessions are still accepted |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin is always False |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with a single unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token raises IndexError on a missing or malformed hea |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals is not constant-time despite its docstring |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke raises KeyError for an unknown token |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: peek(), stats() and utilization() read shared state without  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction removes the oldest-inserted entry, not the least re |
| Y29 | YES | parallel: CA06 warm's peek and set take the lock separately, so the ja | review-concurrency: warm() does check-then-act across two separately locked call |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() deletes from the dict while iterating it |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() uses integer division before multiplying |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks by re-acquiring a non-reentrant lock; review-concurrency: flush() re-acquires the non-reentrant lock it already holds  |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid LINKD_TIMEOUT_MS sets timeout to 0 instead of keepin |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is stored unconverted, but the cache trea |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-security: validate() always returns True, so a bad configuration never |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in report name allows arbitrary .csv write an |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Sensitive exports created with default permissions |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV rows are not quoted or escaped |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot never flushes or closes the file before renam; review-concurrency: write_snapshot uses a fixed shared temp path and renames it  |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report mutable default accumulates rows across calls; review-concurrency: parse_report uses a mutable default argument shared by all c |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes from store.links while consuming a generator; review-concurrency: Janitor deletes from the dict it is lazily iterating, unsync |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache returns without joining its threads; review-concurrency: warm_cache never joins its threads despite documenting that  |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets raises IndexError when there are no problems |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner raises KeyError for an owner with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-bugs: Janitor thread is non-daemon and stop_janitor cannot wake it; review-concurrency: Janitor shutdown is a bare global flag and the thread is nev |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are incremented without synchronization |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the submitted token; bearer_token crashe |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner taken from the request body, not the authenticate |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-bugs: delete_link does not enforce ownership; review-security: delete_link has no ownership check, so any user can delete a |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect and Location header injection in /out |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin check trusts a client-supplied header instead of the s; review-security: Admin endpoint authorized by a client-supplied x-admin heade |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats() divides by zero when the store is empty; review-concurrency: stats() computes hits_per_link from two separate unlocked re |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS with credentials enabled on every response |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports every owner's links to any authenticated use |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() returns a different file than the one it wrote when; review-concurrency: Concurrent /report requests with the same name race on one f |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) config.py:99 — validate() never refuses a bad configuration
- (medium) store.py:36 — New codes can collide and silently overwrite an existing link
- (low) export.py:58 — read_report leaks the file handle
- (medium) auth.py:43 — sessions dict is iterated in verify() while issue() and revoke() mutate it, with no lock
- (low) main.py:90 — Unbounded, unvalidated pagination parameters
