# minimax-m3:cloud · python · run 20260901-105859 (repeat 2)

recall **36/66** · 52 finding(s), 6 unmatched · 1039070 tokens · 704s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError on the very first link any owner ma |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes the link only while it is still valid, nev |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates frm twice and never validates to |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: page() extends `end` one past the list when clamping |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() reports 0 for almost every non-trivial store |
| Y07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| Y08 | — | parallel: C10 top sorts by hits ascending, so the leaderboard shows th |  |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() matches on substring, not equality; review-security: by_owner uses substring match |
| Y10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| Y11 | — | parallel: C14 quotas hands out the store's own dict while the doc call |  |
| Y12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() mutates self.links while iterating it; returns remai; review-bugs: prune() returns len(self.links), not the number of links rem |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes minted with non-cryptographic RNG |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Default secret used when LINKD_SECRET is unset; review-security: Session tokens minted with non-cryptographic RNG |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token logged at issue time |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() ignores the session expiry the docstring promises t |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() can never grant access; the condition is `or; review-security: require_admin always returns True (broken authz check) |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with single-round unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token() raises IndexError on a header without a token; review-security: bearer_token assumes two-space header and lacks validation |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | YES | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a | review-bugs: parse_lifetime() returns 0 for any malformed input, includin |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: peek, stats, and utilization read shared state without the l |
| Y28 | — | parallel: CA08 eviction drops the first key in insertion order, not th |  |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() mutates self.entries while iterating it; review-concurrency: Cache.sweep mutates self.entries during iteration, raising R |
| Y31 | — | PY-ONLY utilization floor-divides before scaling so it always reports  |  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-concurrency: Cache.flush deadlocks by re-entering its own non-reentrant L |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | — | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st |  |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| Y38 | YES | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, | review-bugs: load_file() swallows every exception with a bare `except:` |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() never refuses anything and never raises; review-bugs: load_file() lets a non-numeric cache_size crash the process |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in CSV export name and read_report |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-bugs: Every file open in export.py leaks the handle on exception |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot() is not durable across the rename boundary; review-security: Snapshot write vulnerable to symlink/TOCTOU clobber |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report() uses a mutable default argument; rows accumul |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes links while iterating the generator find_exp; review-concurrency: Janitor thread races with request handlers on store.links an |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache() returns before the threads it spawned finish |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() returns the first problem string instead of  |
| Y50 | — | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo |  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | — | parallel: X01 the counters are incremented from the janitor thread and |  |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the rejected bearer token |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: create_link uses caller-supplied owner instead of session |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: delete_link has no ownership check |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out with no allow-list |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin gate is a forgeable client header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats() divides by zero on an empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin with credentials |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: export report path is built from the query string without sa |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) cache.py:73 — utilization() returns 0 for any cache below `limit` entries because of floor-divide-then-m
- (low) cache.py:80 — flush() calls delete on a sentinel key that is never inserted
- (critical) auth.py:23 — Authenticator.sessions has no lock; every authenticated request races on the dict
- (high) store.py:60 — Store.resolve races on link.hits via non-atomic read-modify-write
- (medium) worker.py:29 — _running flag race and stop_janitor never joins the thread
- (medium) auth.py:44 — Token comparison uses non-constant-time equality
