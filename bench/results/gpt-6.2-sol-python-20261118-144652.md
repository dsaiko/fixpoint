# gpt-6.2-sol · python · run 20261118-144652 (repeat 1)

recall **44/66** · 71 finding(s), 17 unmatched · 207066 tokens · 723s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: Links created without tags share the same list |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: Creating an owner's first link raises after inserting it |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it; review-security: Resolving a live short code deletes it |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: Rename accepts an invalid destination code |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: The link success rate is zero unless every link has hits |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Removing links leaves owner quotas unchanged |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard returns the least followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: Owner lookup matches partial names |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive it |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: The quota snapshot exposes the store's mutable state |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-concurrency: Readers can observe a partially imported batch |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Pruning raises on expired links and returns the wrong count  |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Observable short codes expose the generator used for session |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session issuance logs reusable bearer credentials |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions remain valid; review-security: Expired session tokens remain valid |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: The admin role check rejects every role |
| Y22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: Missing Authorization header raises before the 401 response |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: A zero-capacity cache raises during insertion and reporting; review-bugs: Cache eviction ignores access recency |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Cache sweep raises after deleting its first expired entry |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: A partially full cache reports zero utilization |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-concurrency: Cache.flush deadlocks by acquiring its own lock twice |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Malformed timeout configuration overwrites the default with  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond durations are stored as seconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: Configured fetch limit is ignored |
| Y38 | YES | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, | review-bugs: An unreadable configuration file is treated as missing |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: Validation accepts configurations it identifies as invalid |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Report names let callers overwrite CSV files outside the exp |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Saved reports can be readable by other local users |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV exports corrupt fields containing commas or line breaks; review-security: User-controlled fields can inject spreadsheet formulas into  |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: Snapshot replacement can publish incomplete data |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: Parsing a report adds its rows to previous results |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Deleting an expired link aborts the janitor sweep; review-concurrency: The janitor deletes from the dictionary it is iterating |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: Cache warming returns while its workers can still write |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: The target checker crashes when no target fails |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: Looking up a quota for a new owner raises KeyError |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Concurrent requests can lose dashboard counter increments |
| Y54 | YES | PY-ONLY importing this module builds the whole service and STARTS A TH | review-concurrency: The janitor can prevent the process from exiting |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: Invalid create payloads escape request error handling; review-security: Callers can attribute new links to another owner |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete another owner’s link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: The permanent outbound redirect can skip click counting; review-security: The tracking endpoint redirects to an arbitrary caller-suppl |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A caller-controlled header grants access to administrator qu |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: The stats endpoint crashes when the store is empty; review-concurrency: Concurrent link changes can crash the stats endpoint |
| Y62 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Uppercase codes accepted by the store cannot be resolved ove |
| Y63 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: A report without a name reads the wrong file; review-concurrency: Concurrent report requests race on the same output file |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) worker.py:39 — The janitor never sweeps while it is running
- (medium) store.py:45 — A generated code collision overwrites the existing link
- (medium) cache.py:35 — Expired cache entries block warming
- (medium) store.py:138 — A failed batch import leaves earlier changes committed
- (medium) store.py:34 — Link creation accepts HTTP targets without a host
- (medium) main.py:78 — Requests without a link code raise KeyError
- (medium) main.py:89 — Pagination accepts negative bounds and crashes on invalid integers
- (medium) main.py:29 — Responses labeled as JSON contain plain text or CSV
- (medium) main.py:140 — Reports silently omit all links after the first 1,000
- (medium) export.py:65 — Parsing a generated report adds a phantom last row
- (high) store.py:58 — Redirect and deletion can race into an uncaught KeyError
- (medium) export.py:43 — Concurrent snapshot writers share one temporary file
- (medium) store.py:67 — Concurrent renames can overwrite a link
- (medium) auth.py:43 — Issuing or revoking a session can interrupt token verification
- (medium) cache.py:57 — Cache warming can overwrite a newer value
- (critical) main.py:140 — Any authenticated user can export other owners’ links
- (high) store.py:45 — Failed creation requests can grow the in-memory store without a limit
