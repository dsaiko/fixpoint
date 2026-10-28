# beam-ollama-direct · python · run 20261028-114852 (repeat 1)

recall **50/66** · 72 finding(s), 5 unmatched · 586745 tokens · 220s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() crashes for a new owner |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes valid links and keeps expired ones; review-security: Expired links still resolve and active links are deleted |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: page() uses end = len + 1 on overflow |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() always reports 0 unless every link was used |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not return the owner's quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-used links instead of the most-used |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() matches substrings of owners; review-security: Owner matching uses substring instead of equality |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live internal dict; review-security: quotas returns mutable internal quota state |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() mutates the dict while iterating it and returns the  |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-bugs: find_expired() returns a generator over a mutating dict |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Predictable short codes from getrandbits |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback session secret |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens generated with predictable PRNG |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() does not enforce session expiry; review-security: Expired session tokens are still accepted |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() can never return True; review-security: require_admin can never return true |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token() crashes on missing or malformed header; review-security: bearer_token crashes on malformed Authorization header |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: Non-constant-time secret comparison |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke() raises KeyError for unknown tokens |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: Cache.peek, stats and utilization read shared state without  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: LRU eviction picks insertion order, not least recently used |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() mutates the dict while iterating it; review-concurrency: Cache.sweep mutates its dict during iteration |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() truncates to zero for partial caches |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks by re-acquiring a non-reentrant lock; review-concurrency: Cache.flush deadlocks by re-acquiring its own non-reentrant  |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS is stored unconverted as seconds; review-bugs: LINKD_CACHE_TTL_MS is stored unconverted as seconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() always returns True even when problems exist |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Filename path traversal in owner CSV export |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Exported reports world-readable despite sensitive contents |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-security: Unescaped CSV fields allow injection and row spoofing |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot() renames before flushing/closing the handle; review-security: Shared temporary path enables symlink race |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report() uses a mutable default argument; review-concurrency: parse_report uses a mutable default argument shared across c |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: janitor loop deletes while iterating a generator; review-concurrency: Janitor mutates store.links while iterating a generator |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache() does not wait for its threads and assumes pairs; review-concurrency: warm_cache starts threads but never joins them |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() raises IndexError when all targets are healt |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner() raises KeyError for owners without links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters incremented without synchronization |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | YES | parallel: S06 an absent Authorization header becomes the empty string, | review-bugs: handle() crashes on a missing Authorization header |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unhandled exception before authentication for malformed requ |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner can be spoofed by request payload |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-bugs: delete_link() does not verify ownership; review-security: Any authenticated user can delete any link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated redirect in /out endpoint |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Administrator access controlled by spoofable header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats() divides by zero when the store is empty; review-security: Denial of service when store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS with credentials enabled |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-security: Path traversal in report export and read |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) store.py:21 — Store links/quota dicts have no synchronization
- (high) auth.py:23 — Authenticator.sessions mutated and iterated without a lock
- (low) auth.py:43 — Unauthorized response echoes attacker-supplied token
- (medium) main.py:91 — Link listing exposes every user's links
- (medium) config.py:72 — load_file reads arbitrary paths without containment
