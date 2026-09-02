# nemotron-3-ultra:cloud · python · run 20260902-101831 (repeat 1)

recall **42/66** · 66 finding(s), 10 unmatched · 677283 tokens · 2463s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: KeyError on new owner in create() |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() logic inverted - deletes valid links |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates frm twice, not to |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: page() off-by-one in end calculation |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() returns 0 until all links used |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() doesn't decrement quota |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending not descending |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() uses substring match; review-security: Substring match in by_owner allows authorization bypass |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() replaces instead of extending expiry |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-concurrency: Store.quotas() returns live internal dict reference |
| Y12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-bugs: prune() modifies dict during iteration; review-security: Dictionary modified during iteration in store.prune |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns wrong value; review-concurrency: Store.prune() modifies dict during iteration |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-concurrency: Store.find_expired() returns generator over live dict |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Non-cryptographic random used for short code generation |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded default secret used when LINKD_SECRET not set |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Non-cryptographic random used for session token generation |
| Y19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| Y20 | — | parallel: S04 verify never looks at expires_at, so a token works forev |  |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin always returns False; review-security: Logic bug in require_admin always returns False |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token crashes on missing/malformed header; review-security: IndexError in bearer_token on malformed header |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-bugs: secret_equals vulnerable to timing attacks; review-security: Timing attack in secret comparison |
| Y25 | YES | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a | review-security: Bare except clause catches critical exceptions |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: Cache.peek() reads entries without lock - data race with wri |
| Y28 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Cache.sweep() modifies dict during iteration; review-concurrency: Cache.sweep() modifies dict during iteration |
| Y31 | — | PY-ONLY utilization floor-divides before scaling so it always reports  |  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: Cache.flush() deadlocks on lock re-acquisition; review-concurrency: Cache.flush() deadlocks on non-reentrant lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Timeout/TTL units mismatch (ms vs seconds) |
| Y37 | — | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne |  |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() returns True despite problems; review-security: validate() returns True despite configuration errors |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-bugs: Path traversal in export_csv; review-security: Path traversal in export_csv via name parameter |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Path traversal in archive_all via owner parameter |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-security: Predictable temp file path in write_snapshot |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report mutable default argument |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-concurrency: Janitor thread accesses store.links directly without synchro; review-security: Dictionary modified during iteration in janitor loop |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: warm_cache() starts threads but never joins them - thread le |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() IndexError on empty problems; review-concurrency: check_targets() raises IndexError on empty list |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner() KeyError for unknown owner |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are not thread-safe |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | YES | parallel: S06 an absent Authorization header becomes the empty string, | review-bugs: Missing auth header crashes request handler |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Token leaked in error response and logs |
| Y57 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Missing authorization check in delete_link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: Open redirect in /out endpoint; review-security: Open redirect in /out endpoint |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint protected by spoofable header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: Division by zero in stats() |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Invalid CORS: wildcard origin with credentials |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.py:73 — utilization() returns 0 until cache is full
- (critical) store.py:28 — Store class has no synchronization - data races on all operations
- (critical) auth.py:23 — Authenticator.sessions dict accessed without locks
- (high) worker.py:29 — Janitor _running flag has no memory visibility guarantee
- (medium) auth.py:43 — Authenticator.verify() linear search races with modifications
- (medium) main.py:41 — Global shared state with no synchronization in request handlers
- (critical) export.py:58 — Path traversal in read_report via name parameter
- (critical) store.py:34 — SSRF via insufficient URL validation in store.create
- (high) auth.py:43 — Timing attack in session token verification
- (high) config.py:72 — Path traversal in load_file via user-supplied path
