# ollama-direct@mimo-v2.6-flash:cloud · python · run 20261125-142937 (repeat 1)

recall **39/66** · 50 finding(s), 5 unmatched · 195692 tokens · 277s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: Default tag list is shared between links |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: First link creation raises a quota KeyError |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry check is inverted in resolve |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename validates the source twice instead of the destination |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: Success rate is floored to only 0% or 100% |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link does not release its quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top returns the least-followed links first |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: Owner lookup matches substrings instead of exact owners |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives expired links |
| Y11 | — | parallel: C14 quotas hands out the store's own dict while the doc call |  |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Import is not atomic when a later link is invalid |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune mutates the store during iteration and returns the wro |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens use non-cryptographic randomness |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer tokens are written verbatim to logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Expired sessions remain valid indefinitely |
| Y21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords are hashed with fast, unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: Missing Authorization header crashes request handling |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Zero-sized cache crashes on first insertion; review-bugs: Eviction is insertion-order rather than least-recently-used |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Cache sweep mutates the dictionary while iterating; review-concurrency: Cache sweep deletes entries during dictionary iteration |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: Cache utilization calculation returns the wrong percentage |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-concurrency: Cache flush deadlocks by reacquiring its non-reentrant lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Malformed cache settings crash configuration loading |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: FETCH_LIMIT environment value is ignored |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: Validation always reports success |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: User-controlled report name escapes the export directory |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-concurrency: Concurrent same-name report exports overwrite one file |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV fields are written without escaping; review-security: CSV export permits formula injection through owner data |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: Snapshot is renamed before its file is flushed and closed |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report reuses rows across calls |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Expiry janitor deletes links during generator iteration; review-concurrency: Janitor deletes store entries while iterating their source d |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: warm_cache returns before its worker threads finish; review-concurrency: stop_janitor neither wakes nor waits for the worker |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets crashes when every target passes |
| Y50 | — | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo |  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics increments can be lost under concurrent requests |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: Malformed create payloads escape as exceptions; review-security: Create endpoint accepts an arbitrary owner instead of the se |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated session can delete any link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: The next parameter creates an open redirect |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quotas gate trusts a spoofable client header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: Stats endpoint divides by zero on an empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exposes every owner's links |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default report reads a different filename than it writes |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) config.py:41 — Invalid timeout is applied instead of keeping the default
- (medium) main.py:78 — Redirect route crashes when code is missing
- (medium) main.py:89 — Pagination query values are parsed without error handling
- (high) store.py:101 — Concurrent delete of the same link raises an unhandled KeyError
- (medium) auth.py:43 — Session verification iterates a dictionary while it is mutated
