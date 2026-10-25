# codex@moonshotai/kimi-k3.1-pro#xhigh · python · run 20261024-231303 (repeat 1)

recall **43/66** · 60 finding(s), 9 unmatched · 663813 tokens · 541s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: New links share one tags list |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: First link creation for every owner fails |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry condition is inverted; review-security: Expired links continue resolving after their expiry |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: Rename validates the source twice instead of the destination |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: Success rate only reports 0 or 100 |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting links never releases owner quota |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard returns least-followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: Owner filtering uses substring matching |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can revive it or shorten its lifetime |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: Quota snapshot exposes mutable internal state |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Batch import is not atomic |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Pruning expired links raises and returns the wrong count |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-concurrency: Janitor deletes links while consuming a live dictionary gene |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens use a predictable non-cryptographic PRNG |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Live session tokens are written to logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions remain valid; review-security: Expired sessions remain valid indefinitely |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Admin check rejects every role |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords use a fast unsalted SHA-256 hash |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: Malformed or missing Authorization headers crash routing |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction is not LRU and evicts on replacement |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Sweeping an expired cache entry raises RuntimeError; review-concurrency: An expired cache entry terminates the janitor thread |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: Cache utilization truncates to 0 or 100 |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: Flushing the cache deadlocks; review-concurrency: Cache.flush deadlocks on its own lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond settings are interpreted as seconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is discarded |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: Invalid configurations are never rejected |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Report names permit directory traversal and arbitrary CSV ov |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Cross-owner reports may be created world-readable |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-concurrency: Concurrent reports overwrite and expose each other's output; review-security: Owner-controlled CSV cells permit spreadsheet formula inject |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: Snapshot is renamed before its data is flushed or closed; review-security: Predictable shared temporary file enables symlink overwrite  |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: Parsed rows leak between calls |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor crashes while deleting expired links |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: Cache warming returns before workers finish; review-concurrency: warm_cache returns before its worker threads finish |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: Successful target checks raise IndexError |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: Owners with no links crash the quota count |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-concurrency: Janitor shutdown and restart lifecycle is broken |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Dashboard metrics lose concurrent updates |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Clients can assign links to arbitrary owners |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete any link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The outbound redirect accepts arbitrary destinations |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A client-supplied header bypasses administrator authorizatio |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: Stats crashes while the store is empty; review-concurrency: Statistics numerator and denominator are read non-atomically |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Reports disclose every user's links |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) config.py:39 — Invalid timeout overwrites the default despite saying otherwise
- (medium) export.py:65 — Normal trailing newlines produce phantom report rows
- (medium) main.py:67 — Malformed create requests escape the 400 handler
- (high) auth.py:43 — Session verification races with issuance and revocation
- (high) store.py:78 — Request-side store iterations race with concurrent mutations
- (medium) store.py:46 — Store counters lose concurrent increments
- (medium) store.py:67 — Concurrent renames can overwrite and lose a link
- (medium) store.py:101 — Concurrent deletion raises an uncaught KeyError
- (medium) export.py:43 — Concurrent snapshots share a temporary pathname
