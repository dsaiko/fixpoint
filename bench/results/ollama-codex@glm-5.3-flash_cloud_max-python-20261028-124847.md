# ollama-codex@glm-5.3-flash:cloud#max · python · run 20261028-124847 (repeat 1)

recall **39/66** · 53 finding(s), 4 unmatched · 560879 tokens · 505s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: First new link crashes on quota update |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Link expiry condition is inverted; review-security: Link expiry check is inverted |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename validates the source code twice |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | — | PY-ONLY (parallel: C05) floor division is applied before the multiply, |  |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not release quota |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard returns least-followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: Owner filtering matches substrings |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Expired links can be revived by extend |
| Y11 | — | parallel: C14 quotas hands out the store's own dict while the doc call |  |
| Y12 | — | parallel: C13 import_all stores as it validates, so a bad code leaves  |  |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns the wrong count |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes are only 32 bits and predictable under PRNG stat |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Bearer tokens are minted from noncryptographic randomness |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer credentials are written to logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired session tokens remain valid; review-security: Session expiry is never enforced |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin can never allow a session; review-security: Administrator authorization check can never succeed |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashes are unprotected SHA-256 digests |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: Missing or malformed bearer header crashes request handling; review-security: Malformed Authorization header causes an unhandled index err |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction is not least-recently-used |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Sweeping modifies a dictionary while iterating it |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: Cache utilization is scaled incorrectly and divides by zero |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: Cache flush deadlocks on its non-reentrant lock; review-concurrency: Cache flush deadlocks on a non-reentrant lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL milliseconds are used as seconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and discarded |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: Configuration validation always succeeds |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: User-controlled export name permits path traversal |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Export files do not enforce the stated private permissions |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV fields are written without escaping; review-security: CSV values are not escaped, enabling CSV/spreadsheet injecti |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: Snapshot is not prepared for a durable rename; review-security: Predictable and world-umask snapshot path in /tmp |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report reuses callers' default list across calls |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes a live dictionary behind a generator; review-concurrency: Janitor deletes links while the expiry generator is iteratin |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: Cache warming returns before warming finishes; review-concurrency: warm_cache does not wait for its worker threads |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets crashes when every target is valid |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: Owner count raises for an owner with no known links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters use unsynchronized increments |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: Create-link request parsing can raise unhandled exceptions; review-security: Owner and quota limits are not tied to the authenticated ses |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete another owner's link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on the outbound click endpoint |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint trusts a spoofable header instead of the sess |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: Stats endpoint divides by zero when the store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint discloses every tenant's links |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Report uses the wrong name when no report name is supplied |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) main.py:78 — Redirect without a code raises KeyError
- (medium) store.py:137 — import_all mutates the store before validation completes
- (high) store.py:60 — Store shared state has no synchronization
- (high) auth.py:43 — Session lookup mutates while another thread can modify sessions
