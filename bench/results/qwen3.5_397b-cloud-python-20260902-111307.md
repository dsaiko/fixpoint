# qwen3.5:397b-cloud · python · run 20260902-111307 (repeat 1)

recall **25/66** · 36 finding(s), 7 unmatched · 273545 tokens · 154s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: Mutable default argument in create(); review-concurrency: Store has no synchronization for shared mutable state |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: KeyError when creating link for new owner |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Inverted expiry check deletes valid links |
| Y04 | — | parallel: C03 rename validates `frm` twice; `to` is never validated |  |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() returns 0 for any rate under 100% |
| Y07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns least-hit links instead of most-hit |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner uses substring match instead of equality |
| Y10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| Y11 | — | parallel: C14 quotas hands out the store's own dict while the doc call |  |
| Y12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-bugs: RuntimeError iterating dict while deleting in prune; review-concurrency: RuntimeError from mutating dict during iteration |
| Y14 | — | parallel: C15 prune counts removals into `removed` and then returns th |  |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Predictable short codes using non-CSPRNG |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret key |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Predictable session tokens using non-CSPRNG |
| Y19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| Y20 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin always returns True; review-security: Broken admin role check logic |
| Y22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: IndexError on malformed Authorization header; review-security: Fragile Authorization header parsing |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: Non-constant-time secret comparison |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: RuntimeError iterating dict while deleting |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() returns 0 for non-full cache |
| Y32 | — | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak |  |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: fetch_limit never updated from environment |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate always returns True even with problems |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in CSV export |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | — | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n |  |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: Mutable default argument causes shared state |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor may crash on dict iteration during deletion |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: warm_cache leaks threads - never joins them |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: IndexError when check_targets finds no problems |
| Y50 | — | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo |  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | — | parallel: X01 the counters are incremented from the janitor thread and |  |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Missing authorization check on delete endpoint |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect vulnerability |
| Y60 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| Y61 | — | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr |  |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Invalid CORS configuration |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) export.py:58 — read_report never closes file handle
- (high) store.py:46 — Data race on quota counter increment
- (high) store.py:60 — Data race on hits counter
- (medium) worker.py:37 — Global _running flag accessed without memory barrier
- (medium) auth.py:21 — Authenticator has no synchronization for sessions dict
- (low) cache.py:40 — Check-then-evict race in cache set
- (medium) export.py:58 — Path traversal in report read
