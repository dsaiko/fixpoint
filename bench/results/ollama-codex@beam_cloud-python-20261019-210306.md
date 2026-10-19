# ollama-codex@beam:cloud · python · run 20261019-210306 (repeat 1)

recall **49/66** · 67 finding(s), 6 unmatched · 1067713 tokens · 127s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-concurrency: Store has no synchronization despite being shared across req |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create crashes for a new owner |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve deletes live links and keeps expired ones; review-security: Expired links are kept resolvable and non-expired links are  |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename validates frm twice and never validates to |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: page clamps end one past the list |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate always 0 unless 100% used |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not restore owner quota |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns least-followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner matches substrings instead of exact owner; review-security: by_owner uses substring matching, leaking other owners' link |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas returns the live internal dict |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-concurrency: Store.prune mutates links while iterating over it |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune mutates dict during iteration and reports wrong count |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-bugs: find_expired returns a live view the janitor deletes over; review-concurrency: Store.find_expired returns a live view over a mutating dict |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes generated with the insecure random module |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing secret when LINKD_SECRET is unset |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens are minted with the insecure random module |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to the log in plaintext |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify ignores token lifetime; review-security: verify() never checks expires_at, so sessions never expire |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin can never accept anything; review-security: require_admin predicate is contradictory and never authorize |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token crashes on missing/malformed header; review-security: bearer_token crashes on a missing or malformed Authorization |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals is a plain comparison, not constant-time |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke raises KeyError for unknown token |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-bugs: peek is not thread-safe; review-concurrency: Cache reports thread-safety but peek/stats/utilization read  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction is FIFO, not LRU |
| Y29 | YES | parallel: CA06 warm's peek and set take the lock separately, so the ja | review-concurrency: Cache.warm performs unlocked check-then-act between threads |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep mutates dict during iteration; review-concurrency: Cache.sweep mutates the entries dict while iterating it |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization integer division truncates to 0 |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush deadlocks by re-acquiring its own lock; review-concurrency: Cache.flush deadlocks by reacquiring a non-reentrant lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS treated as seconds; review-bugs: LINKD_CACHE_TTL_MS treated as seconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate always returns True; review-security: validate() always returns True, so a rejected configuration  |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in report/export names |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-security: Snapshot written through a fixed, predictable /tmp path |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report shares mutable default list across calls |
| Y46 | YES | the errors counter is reported by snapshot and never incremented anywh | review-concurrency: Metrics counters are incremented without synchronization |
| Y47 | — | PY-ONLY the janitor deletes from store.links while consuming the gener |  |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache never joins worker threads; review-concurrency: warm_cache starts threads but never joins them |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets crashes when there are no problems |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner crashes for unused owners |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | — | parallel: X01 the counters are incremented from the janitor thread and |  |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response reflects the presented token back to the caller |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: JSON parsing and input validation are outside the guarded pa |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: delete_link performs no ownership check |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect: /out forwards to any attacker-supplied URL |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: /admin/quotas authorizes on a client-supplied header instead |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: /stats crashes on empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin together with Allow-Credentials |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports every owner's links to any authenticated use |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) export.py:43 — write_snapshot leaves file handle open
- (medium) main.py:78 — redirect raises KeyError on missing code
- (high) auth.py:23 — Authenticator mutates shared sessions dict without synchronization
- (high) worker.py:33 — Janitor runs in a non-daemon thread that mutates the store unlocked
- (high) export.py:21 — Export files are created world-readable despite the comment claiming otherwise
- (low) main.py:142 — Error response leaks filesystem layout to the client
