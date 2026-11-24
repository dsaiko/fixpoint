# ollama-direct@hy4:cloud · python · run 20261124-102438 (repeat 1)

recall **55/66** · 77 finding(s), 8 unmatched · 285615 tokens · 967s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: All links created without tags share one list |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: Creating the first link for an owner raises KeyError |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Live links are deleted on resolve while expired links keep w; review-security: Inverted expiry check destroys live links and keeps expired  |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename validates the source code twice and never validates t |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: Pagination accepts negative offsets and sizes |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: Success rate reports zero unless every link was followed |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not return the owner's quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns the least-followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner performs substring matching instead of exact owner ; review-security: Owner filtering uses substring matching and leaks other tena |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas returns the live internal dict |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-bugs: prune deletes while iterating the live links dict |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns the remaining link count instead of the remove |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-bugs: Expired-link generator is consumed while the worker deletes  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes are generated from the same predictable global P |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hard-coded fallback secret is shipped in source |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens are generated with a predictable non-cryptogr |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-bugs: Issued sessions are never cleaned up; review-security: Every session token is written to application logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Session expiry is never enforced; review-security: Expired sessions are never rejected |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Admin check is an impossible condition |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords are stored as unsalted single-round SHA-256 hashes |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: Missing Authorization header crashes request handling; review-security: Missing or malformed Authorization header crashes request ha |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: Secret comparison helper is not constant-time |
| Y25 | YES | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a | review-bugs: parse_lifetime swallows all exceptions as a zero lifetime |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: Revoking an unknown or already revoked token raises KeyError |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction is insertion-order and crashes at zero capaci |
| Y29 | YES | parallel: CA06 warm's peek and set take the lock separately, so the ja | review-concurrency: Cache.warm() is a check-then-act race and peek/stats/utiliza |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Cache sweep deletes entries while iterating the live dict |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: Cache utilization uses floor division and can divide by zero |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-concurrency: flush() self-deadlocks on the non-reentrant cache lock; review-security: Admin cache flush deadlocks while holding the global cache l |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Invalid timeout environment value overwrites the default wit |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond configuration values are interpreted as seconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| Y38 | YES | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, | review-bugs: Config file loader silently swallows every error |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: Validation never rejects invalid configuration; review-security: Configuration validation always succeeds and allows unsafe s |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Unvalidated report name allows writing files outside the exp |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-bugs: Report writers leak handles when a write raises; review-security: Reports containing all owners' data are created with default |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV reports are written without escaping fields; review-security: CSV fields are written without quoting or formula neutraliza |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Owner-derived archive filenames permit path traversal |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: Snapshot is renamed before being flushed, fsynced, or closed; review-concurrency: write_snapshot uses a fixed shared temp path, so concurrent  |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report shares its default row list across calls; review-security: Mutable default argument leaks rows between report parses |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-concurrency: Janitor mutates store.links without a lock and dies permanen |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: warm_cache spawns unbounded threads and never joins any of t; review-concurrency: stop_janitor never wakes or joins the sleeping thread; cance |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets crashes when there are no failures |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner raises KeyError for owners with no quota ent |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters use non-atomic +=, losing increments under  |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link owner is taken from the request body instead of the aut |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete any owner's link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unvalidated next parameter creates an open redirect and head |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quota endpoint trusts a client-supplied header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: Stats endpoint divides by zero on an empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS is combined with credentialed responses on eve |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-concurrency: Concurrent /report requests clobber the same CSV file and re; review-security: Report endpoint discloses every owner's links to any session |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report name is not reused when reading the generated |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) export.py:58 — Report reads leave file closure dependent on garbage collection
- (medium) cache.py:35 — Cache peek ignores entry expiration
- (medium) export.py:65 — parse_report turns trailing and blank lines into bogus rows
- (low) config.py:72 — Config file is opened without a context manager
- (high) store.py:78 — Store's shared dicts are read and written by concurrent request threads with no lock
- (medium) store.py:60 — link.hits += 1 and the quota update are unsynchronized read-modify-writes
- (medium) auth.py:43 — Authenticator.sessions is iterated and mutated by concurrent threads without a lock
- (high) store.py:34 — URL validation allows CRLF injection into redirect Location headers
