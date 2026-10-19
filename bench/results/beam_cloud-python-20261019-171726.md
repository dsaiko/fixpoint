# beam:cloud · python · run 20261019-171726 (repeat 1)

recall **52/66** · 80 finding(s), 9 unmatched · 77874 tokens · 82s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create uses a mutable default argument; review-concurrency: Store shares mutable state across request threads with no sy |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create crashes on first link for a new owner |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve deletes unexpired links and keeps expired ones |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename validates the source code twice |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: page has off-by-one in clamp |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate truncates to zero |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not restore quota |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns least-hit links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner matches substrings; review-security: by_owner uses substring match |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas returns internal dict; review-concurrency: quotas() returns the live internal dict, breaking its snapsh |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic and mutates during validation |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-concurrency: prune mutates self.links while iterating it |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune mutates dict while iterating and returns wrong count |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-bugs: find_expired returns a generator consumed with deletes; review-concurrency: find_expired returns a live dict view iterated while being m |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes are predictable |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback session secret |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-concurrency: Authenticator.sessions is mutated and iterated without synch; review-security: Session tokens generated with non-cryptographic PRNG |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued session token written to stdout |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Session expiry is never checked; review-security: Token expiry is never enforced |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin can never return True; review-security: require_admin can never return True |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing is fast and unsalted |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token crashes on malformed authorization header |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: Secret comparison is not constant-time |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke raises KeyError for unknown tokens |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-bugs: peek bypasses the lock despite thread-safe contract; review-concurrency: peek, stats, and utilization read shared state without the l |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction ignores LRU timestamps |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep mutates dict while iterating; review-concurrency: sweep() mutates self.entries while iterating it |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization returns zero percent and divides by zero risk |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush deadlocks by reacquiring its own lock; review-concurrency: flush() deadlocks by re-acquiring its own non-reentrant lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS overwrites default with 0 |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is not converted from milliseconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| Y38 | YES | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, | review-bugs: load_file leaks the file handle and crashes on bad integers |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate always returns True |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal via report name |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Export files created world-readable contrary to intent |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Path traversal via owner filename |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot skips closing and fsync despite durability cl; review-concurrency: write_snapshot uses a fixed shared /tmp path, racing concurr |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report has mutable default and emits an empty trailing; review-concurrency: parse_report's mutable default list is shared across calls |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes while iterating expired codes; review-concurrency: Janitor directly mutates store.links and races the request p |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache does not join threads; review-concurrency: warm_cache spawns threads but never joins them |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets crashes when all targets pass |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner crashes for unknown owners |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are incremented and read without synchroniz |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | YES | parallel: S06 an absent Authorization header becomes the empty string, | review-bugs: Unhandled crash on missing authorization header; review-security: Malformed Authorization header crashes request handler |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Unknown token reflected in 401 response |
| Y57 | — | parallel: C28 the link owner is taken from the request body instead of |  |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-bugs: delete_link does not enforce ownership; review-security: delete_link ignores ownership |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via ?next= parameter |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin authorization uses a spoofable header; review-security: Admin quota endpoint gated by client-controlled header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats divides by zero when store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS with credentials enabled |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.py:54 — warm returns before probes complete
- (medium) store.py:149 — total_hits() and size() are separate unsynchronized reads
- (medium) worker.py:29 — Module-global _running/_janitor make repeated start/stop leak and racy
- (medium) main.py:128 — stats() performs non-atomic reads across store and cache
- (low) export.py:21 — export_csv/read_report have no protection against concurrent writers of the same name
- (high) main.py:84 — Attacker-controlled redirect target enables HTTP response splitting
- (medium) main.py:91 — Link listing is not scoped to the caller
- (high) export.py:58 — Path traversal on report read
- (medium) main.py:68 — Unvalidated TTL can cause extreme expiry / resource abuse
