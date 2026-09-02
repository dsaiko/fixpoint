# gpt-oss:120b-cloud · python · run 20260902-110251 (repeat 1)

recall **28/66** · 38 finding(s), 4 unmatched · 1009933 tokens · 256s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: Quota counter key error for new owners |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiration logic reversed; review-security: Expiration logic reversed in link resolution |
| Y04 | — | parallel: C03 rename validates `frm` twice; `to` is never validated |  |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: `success_rate` uses integer division |
| Y07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: `top` returns links in ascending hit order |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: `by_owner` uses substring match |
| Y10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: `quotas` returns internal mutable dict |
| Y12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-bugs: Deleting entries while iterating in `prune`; review-concurrency: Prune removes links while iterating over the dict |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: `prune` returns wrong value |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Predictable session tokens |
| Y19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Session expiration not verified; review-concurrency: Session store modified without any locking |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Incorrect admin check logic |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Unsalted SHA‑256 password hashing |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: Bearer token parsing can raise IndexError |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: `revoke` may raise KeyError |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Deleting entries while iterating over the dict; review-concurrency: Deleting entries while iterating over cache dict |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: Cache utilization uses integer division and can divide by ze |
| Y32 | — | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak |  |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: FETCH_LIMIT environment variable parsed but never applied |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: `validate` always returns True |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Directory traversal in CSV export |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-concurrency: Concurrent CSV export writes can clobber each other |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-security: Symlink race on snapshot write |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: Mutable default argument in `parse_report` |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | — | PY-ONLY the janitor deletes from store.links while consuming the gener |  |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: `warm_cache` does not wait for threads to finish |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: `check_targets` may raise IndexError on empty problem list |
| Y50 | — | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo |  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | — | parallel: X01 the counters are incremented from the janitor thread and |  |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| Y58 | — | parallel: C17 delete_link never checks the owner its docstring promise |  |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in `/out` endpoint |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quotas endpoint lacks authentication |
| Y61 | — | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr |  |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS wildcard with credentials |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) auth.py:43 — Iterating over sessions dict without synchronization
- (high) store.py:58 — Unsynchronized deletion of links in resolve()
- (high) worker.py:41 — Janitor thread mutates store.links without lock
- (high) bench/testdata/target-python/export.py:58 — Arbitrary file read via export report
