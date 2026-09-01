# minimax-m3:cloud · python · run 20260901-103651 (repeat 1)

recall **20/66** · 23 finding(s), 3 unmatched · 570719 tokens · 967s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-concurrency: Store.links / Store.quota / Link.hits mutated without any lo |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-concurrency: resolve() bug exposed as a concurrency hazard |
| Y03 | — | parallel: C02 expiry comparison inverted: live links are deleted, expi |  |
| Y04 | — | parallel: C03 rename validates `frm` twice; `to` is never validated |  |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | — | PY-ONLY (parallel: C05) floor division is applied before the multiply, |  |
| Y07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| Y08 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-security: by_owner uses substring match on owner name |
| Y10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| Y11 | — | parallel: C14 quotas hands out the store's own dict while the doc call |  |
| Y12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret `dev-secret-do-not-use` |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session token and short-code generated from non-CSPRNG |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens printed to stdout in plaintext |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Session expiration never checked |
| Y21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords stored as unsalted SHA-256 |
| Y23 | — | parallel: S06 bearer_token indexes [1] without checking, so a request  |  |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: Timing-unsafe secret comparison |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: peek reads cache without the lock, breaking the lock-protect |
| Y28 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-concurrency: sweep() mutates the dict while iterating it |
| Y31 | — | PY-ONLY utilization floor-divides before scaling so it always reports  |  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-concurrency: flush() deadlocks itself by calling delete() while holding t |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| Y37 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne |  |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | — | parallel: CF06 validate logs the problems it finds and returns True, s |  |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in CSV export/read_report |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | — | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n |  |
| Y45 | — | PY-ONLY the default `rows=[]` is shared across calls, so every parse a |  |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-concurrency: Janitor thread races with request handlers and reads unsynch |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: warm_cache leaks threads and returns before they finish |
| Y49 | — | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- |  |
| Y50 | — | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo |  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | — | parallel: X01 the counters are incremented from the janitor thread and |  |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Bearer token echoed in 401 response body |
| Y57 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: delete_link does not enforce ownership |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out?next= |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin gate is a client-supplied header |
| Y61 | — | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr |  |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS advertises wildcard origin with credentials |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.py:23 — Authenticator.sessions mutated without a lock under concurrent verify/issue/revoke
- (low) worker.py:29 — stop_janitor only flips a flag; the loop can sleep past it
- (high) store.py:34 — URL validation only checks scheme prefix — SSRF-friendly
