# beam-ollama-direct · python · run 20261124-114222 (repeat 2)

recall **51/66** · 70 finding(s), 4 unmatched · 137447 tokens · 208s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create uses a mutable default tag list; review-concurrency: Store.links and Store.quota are unprotected shared mutable s |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create crashes for a first-time owner |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve deletes live links and keeps expired ones |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename validates the source code twice instead of the destin |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate truncates every non-100 percent result to zero |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not refund the owner quota |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns least-used links first |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner matches owners by substring; review-security: by_owner uses substring match causing data leak |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already expired links; review-security: extend() revives expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas exposes the internal mutable dict; review-security: quotas() exposes mutable internal state |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic and skips quota updates |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune mutates the dictionary while iterating and returns the |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-bugs: find_expired returns a generator over a mutating mapping; review-concurrency: Store.find_expired returns a live view consumed while mutati |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes minted from predictable RNG |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hard-coded fallback session secret |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session token generated with non-cryptographic RNG |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify does not enforce session expiry; review-security: Token expiry is never enforced |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin always returns False |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashed with unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token crashes on missing or malformed headers; review-security: bearer_token crashes on malformed or missing header |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals not actually constant time |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke crashes on unknown tokens |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: Cache peek(), stats(), and utilization() bypass the lock |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: eviction does not pick the LRU entry |
| Y29 | YES | parallel: CA06 warm's peek and set take the lock separately, so the ja | review-concurrency: Cache.warm has an unlocked check-then-act race |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep mutates the cache while iterating it; review-concurrency: Cache.sweep mutates the dict while iterating it |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization is almost always 0 and divides by zero for an em |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush deadlocks by re-entering a non-reentrant lock; review-concurrency: Cache.flush deadlocks by reacquiring its own non-reentrant l |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS is stored as milliseconds, not seconds; review-bugs: LINKD_CACHE_TTL_MS is not converted from milliseconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate always reports success; review-security: validate() always reports success even for bad config |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in report name |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Sensitive reports world-readable |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-security: CSV formula injection in exported reports |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Path traversal via owner in archive_all |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot leaves the handle open and renames before flu; review-concurrency: write_snapshot uses a shared temporary file path |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report has a mutable default and emits a blank trailin |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | — | PY-ONLY the janitor deletes from store.links while consuming the gener |  |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache returns before probe threads finish; review-concurrency: warm_cache returns before its worker threads finish |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets crashes when no targets fail |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner crashes for owners with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are incremented and read without synchroniz |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown token reflected in 401 response |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner taken from request body instead of session |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Deletion has no ownership check |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via next parameter |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by spoofable header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats endpoint divides by zero on an empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS combined with credentials |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-security: Error response leaks internal path and exception |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report reads under the original name after defaulting the wr |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) config.py:72 — load_file leaks the config file handle and can crash on bad values
- (low) export.py:58 — read_report leaks the opened file handle
- (high) auth.py:23 — Authenticator.sessions is unprotected shared mutable state
- (high) worker.py:36 — Janitor thread directly mutates store links and has unsafe global lifecycle
