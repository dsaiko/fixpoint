# codex@google/gemini-4-argon · python · run 20261013-230128 (repeat 1)

recall **48/66** · 60 finding(s), 2 unmatched · 1350877 tokens · 495s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: Store.create uses a mutable default list for tags; review-concurrency: Unsynchronized shared state in Store causes data races, TOCT |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: Store.create raises KeyError on the first link created by an |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Store.resolve has an inverted expiry comparison that deletes |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: Store.rename validates frm twice and never validates to |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: Store.success_rate uses integer floor division before multip |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Store.delete does not decrement the owner's quota count |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Store.top sorts links in ascending order instead of most hit |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: Store.by_owner matches owner by substring instead of exact e; review-security: Substring comparison in Store.by_owner leaks other users' li |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Store.extend revives already-expired links and overwrites re |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: Store.quotas returns the internal dict reference instead of  |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Store.import_all partially mutates the store when a later li |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Store.prune mutates self.links during iteration and returns  |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-bugs: Store.find_expired returns a lazy generator over self.links. |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes are minted with predictable random.getrandbits |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens are generated with non-cryptographic PRNG |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Plaintext session bearer tokens are written to stdout logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Authenticator.verify never checks whether the session has ex; review-security: Authenticator.verify never checks session expiration |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Authenticator.require_admin always returns False due to wron; review-security: Broken boolean condition in require_admin denies all session |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted single-round SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token raises IndexError on missing or malformed Autho |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: Non-constant-time secret comparison leaks timing information |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: Authenticator.revoke raises KeyError when revoking an unknow |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache.set evicts in FIFO order instead of least-recently-use |
| Y29 | YES | parallel: CA06 warm's peek and set take the lock separately, so the ja | review-concurrency: TOCTOU check-then-act race in Cache.warm and unsynchronized  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Cache.sweep mutates self.entries while iterating over it |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: Cache.utilization uses floor division before multiplying by  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: Cache.flush deadlocks by calling self.delete while holding n; review-concurrency: Self-deadlock in Cache.flush when calling Cache.delete under |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: load_config overwrites default timeout with 0 on parse error; review-bugs: load_config assigns LINKD_CACHE_TTL_MS in milliseconds direc |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: load_config assigns parsed to itself instead of updating cfg |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate always returns True even when configuration problem |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in export_csv and archive_all allows writing  |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Sensitive CSV exports are created with default world-readabl |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot renames the temp file before flushing or clos; review-concurrency: Race condition on hardcoded /tmp/linkd-snapshot.json path in |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report uses a mutable default list and appends a bogus |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-concurrency: Janitor thread mutates store.links without synchronization a |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache returns without joining the spawned worker thread; review-concurrency: warm_cache spawns worker threads without joining them before |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets raises IndexError when all link targets are va |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner raises KeyError for owners with no recorded  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Unsynchronized counter increments and snapshots in Metrics |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: create_link trusts client-supplied owner in request body |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Missing ownership verification allows deleting arbitrary lin |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out endpoint via unvalidated next paramete |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts client-controlled x-admin request head |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats raises ZeroDivisionError when the store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Overly permissive CORS headers allow arbitrary origins with  |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-security: Error response in report endpoint leaks internal filesystem  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report passes an empty name to export.read_report after expo; review-concurrency: Concurrent /report requests race on the same output CSV file |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) auth.py:43 — Unsynchronized iteration and mutation of Authenticator.sessions across threads
- (critical) export.py:58 — Path traversal in read_report allows reading arbitrary .csv files
