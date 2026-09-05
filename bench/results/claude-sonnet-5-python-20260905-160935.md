# claude-sonnet-5 · python · run 20260905-160935 (repeat 1)

recall **37/66** · 50 finding(s), 2 unmatched · 55503 tokens · 541s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError for any owner's first link |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() has inverted expiry check: valid links are deleted; review-security: Link expiry check is inverted: expired links resolve forever |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() truncates to 0% except when almost every link |
| Y07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-followed links instead of the most-f |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() matches owners by substring instead of exact iden; review-security: by_owner uses substring match instead of exact equality |
| Y10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| Y11 | — | parallel: C14 quotas hands out the store's own dict while the doc call |  |
| Y12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() mutates self.links while iterating it, and returns t |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens generated with a non-cryptographic PRNG |
| Y19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks session expiry |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: revoke() raises KeyError for an already-expired or unknown t; review-security: require_admin's condition always evaluates to false |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token() crashes on a missing or malformed Authorizati; review-security: bearer_token crashes on missing/malformed Authorization head |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals is not constant-time despite its documented pu |
| Y25 | YES | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a | review-bugs: parse_lifetime() swallows all exceptions, not just parsing f |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: peek(), stats(), and utilization() read shared state without |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: set() evicts the oldest-inserted entry, not the least recent |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() mutates self.entries while iterating over it |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() truncates to 0% for any cache below full capac |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks by re-acquiring its own non-reentrant lock; review-concurrency: Cache.flush() self-deadlocks by re-acquiring its own lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is stored in milliseconds but consumed as; review-bugs: LINKD_CACHE_SIZE parsing has no error handling and crashes s |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never stored on the config |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() always returns True even when problems are found |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in export/report file names |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-security: CSV injection via unescaped link fields |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot() renames the temp file without closing or fl; review-concurrency: write_snapshot() uses one fixed temp path shared by every ca |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report() uses a mutable default argument, accumulating |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor loop mutates store.links while iterating a generator; review-concurrency: Janitor thread mutates Store.links directly with no synchron |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache() returns before the spawned threads finish, cont; review-concurrency: warm_cache() returns without waiting for the threads it star |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() raises IndexError when there are no problems |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner() raises KeyError for an owner with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | — | parallel: X01 the counters are incremented from the janitor thread and |  |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: create_link trusts client-supplied owner instead of the auth |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: delete_link performs no ownership check, letting any user de |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unvalidated `next` parameter |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin authorization is a client-controlled header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats() divides by zero when the store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS response combines wildcard origin with allow-credential |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() fails whenever no ?name= is given, even though expo |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) main.py:68 — create_link silently discards an explicit ttl of 0
- (high) auth.py:43 — Authenticator.sessions dict iterated and mutated from concurrent request threads with no l
