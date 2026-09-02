# kimi-k2.7-code:cloud · python · run 20260901-220537 (repeat 1)

recall **28/66** · 41 finding(s), 6 unmatched · 181447 tokens · 294s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError for new owners |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes live links and serves expired ones; review-security: Inverted expiry check deletes live links on first redirect |
| Y04 | — | parallel: C03 rename validates `frm` twice; `to` is never validated |  |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | — | PY-ONLY (parallel: C05) floor division is applied before the multiply, |  |
| Y07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| Y08 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| Y09 | — | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  |  |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links |
| Y11 | — | parallel: C14 quotas hands out the store's own dict while the doc call |  |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-concurrency: Store.quotas returns the live quota dict instead of a snapsh |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-bugs: prune() mutates dict while iterating over it |
| Y14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-concurrency: Store.find_expired yields a live generator over a shared dic |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens generated with non-cryptographic random |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Newly minted session tokens are written to logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() ignores session expiration; review-security: verify() ignores session expiration |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always returns False |
| Y22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| Y23 | — | parallel: S06 bearer_token indexes [1] without checking, so a request  |  |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-bugs: peek() is not thread-safe despite class contract; review-concurrency: Cache.peek reads the entries dict without its lock |
| Y28 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() mutates dict while iterating over it; review-concurrency: Cache.sweep mutates the dict while iterating it |
| Y31 | — | PY-ONLY utilization floor-divides before scaling so it always reports  |  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks by reacquiring its own lock; review-concurrency: Cache.flush deadlocks by reacquiring its own non-reentrant l |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS not converted to seconds; review-bugs: LINKD_CACHE_TTL_MS not converted to seconds |
| Y37 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne |  |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | — | parallel: CF06 validate logs the problems it finds and returns True, s |  |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in report filename allows arbitrary file read |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | — | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n |  |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: Mutable default argument leaks state across calls |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-concurrency: Janitor deletes store entries directly without synchronizati |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: warm_cache starts threads but never joins them |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() crashes when all targets are valid |
| Y50 | — | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo |  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are incremented without synchronization |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | YES | parallel: S06 an absent Authorization header becomes the empty string, | review-bugs: Missing Authorization header crashes with IndexError; review-security: Missing Authorization header crashes request handling |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: create_link trusts request body for link owner |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: delete_link does not verify link ownership |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out endpoint |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: admin_quotas grants admin access via client-controlled heade |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: Division by zero in stats() when store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS configured with credentials allowed |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() reads back the wrong filename when name is omitted |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) main.py:79 — redirect() crashes when query lacks 'code'
- (high) main.py:101 — delete_link() crashes when query lacks 'code'
- (high) store.py:78 — Store.page iterates shared links dict without locking
- (high) store.py:44 — Store.create races on shared links and quota
- (high) auth.py:43 — Authenticator iterates sessions dict while other threads mutate it
- (medium) store.py:60 — Store.resolve has a lost-update race on link.hits
