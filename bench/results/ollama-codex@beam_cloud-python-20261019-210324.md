# ollama-codex@beam:cloud · python · run 20261019-210324 (repeat 2)

recall **50/66** · 71 finding(s), 8 unmatched · 826889 tokens · 123s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: mutable default tags argument |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create crashes for new owner quota |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve deletes live links and keeps expired ones |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename validates frm twice and never validates to |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: page end offset off by one |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate is only ever 0 or 100 |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not refund quota |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns least-followed links first |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner matches substrings instead of exact owner |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas exposes internal mutable dict; review-concurrency: quotas returns the internal dict by reference |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic and updates no quotas |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-concurrency: prune mutates the links dict while iterating it |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune mutates dict during iteration and reports wrong count |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-bugs: find_expired returns live generator; review-concurrency: find_expired returns a generator over a live dict |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes minted with predictable randomness |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback session secret |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens generated with non-cryptographic PRNG |
| Y19 | — | parallel: S03 every issued session token is printed in cleartext |  |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify ignores session expiry; review-security: verify() never enforces session expiry |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin always returns False |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with fast unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token crashes on malformed header; review-security: bearer_token crashes on malformed Authorization header |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: Secret comparison uses non-constant-time equality |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke crashes on unknown token |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: peek, stats, and utilization read shared state without the l |
| Y28 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| Y29 | YES | parallel: CA06 warm's peek and set take the lock separately, so the ja | review-concurrency: warm performs a unlocked check-then-act peek/set pair |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep mutates dict during iteration; review-concurrency: Cache.sweep mutates the entries dict while iterating it |
| Y31 | — | PY-ONLY utilization floor-divides before scaling so it always reports  |  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush deadlocks on non-reentrant lock; review-concurrency: Cache.flush deadlocks by re-acquiring a non-reentrant lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | YES | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo | review-bugs: timeout is stored in milliseconds instead of seconds |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: cache TTL stored in milliseconds instead of seconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: fetch_limit environment override is discarded |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate always returns True; review-security: validate() always returns True |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in export filenames |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-security: Unescaped CSV fields enable CSV injection |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot does not durably write before rename; review-security: Predictable shared temp path (TOCTOU/symlink risk) |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report has mutable default argument; review-concurrency: Mutable default argument makes parse_report share one list a |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: janitor deletes while iterating expired codes |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache never joins threads; review-concurrency: warm_cache starts threads but never joins them |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets crashes when no problems |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner crashes for owners with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are mutated and read without synchronizatio |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | YES | parallel: S06 an absent Authorization header becomes the empty string, | review-bugs: handle crashes on missing Authorization header |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown-token error echoes the submitted token |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner taken from request payload instead of session |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-bugs: delete_link does not enforce owner authorization; review-security: Deletion lacks ownership authorization |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated open redirect via ?next= |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated by user-controlled header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats divides by store size without zero check |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS with credentials enabled |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-security: Report export path traversal and error disclosure |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) cache.py:73 — utilization integer division loses percentage
- (medium) cache.py:54 — warm treats expired entries as cached
- (critical) store.py:21 — Store has no synchronization around shared mutable state
- (high) worker.py:36 — Janitor directly deletes store.links without synchronization
- (medium) worker.py:33 — Janitor thread is non-daemon and stop_janitor cannot interrupt its sleep
- (high) auth.py:18 — Authenticator sessions dict is accessed without synchronization
- (high) export.py:58 — Path traversal in report reads
- (medium) export.py:21 — Exported reports created world-readable despite sensitive content
