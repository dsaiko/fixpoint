# gpt-6-astra-xhigh · python · run 20260905-153357 (repeat 1)

recall **42/66** · 56 finding(s), 10 unmatched · 298609 tokens · 823s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: The first creation for every owner raises KeyError |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it; review-security: Resolving a live link deletes it without ownership authoriza |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: Rename never validates the destination code |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: Success rate truncates partial success to zero |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Removing links never releases their owner counts |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard returns the least-followed links first |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: Owner lookup uses substring matching instead of equality |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive an expir |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: The quota snapshot exposes mutable internal state |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Batch import leaves partial changes after validation fails |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-bugs: Pruning fails after deleting its first expired link |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns the remaining count instead of the removed cou |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens can be predicted from exposed link codes |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Application logs contain usable session credentials |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-security: Expired session tokens remain authorized indefinitely |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: The role check rejects every session |
| Y22 | — | parallel: S05 passwords are stored as a single unsalted SHA-256, crack |  |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: Missing authorization headers crash request handling |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction removes the oldest insertion instead of the least r |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Sweeping an expired cache entry raises RuntimeError |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: Cache utilization reports zero until the cache is full |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-concurrency: flush deadlocks by acquiring its own non-reentrant lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: An invalid timeout replaces the default with zero |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL milliseconds are interpreted as seconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is ignored |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: Validation permits configurations that crash cache operation |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Report names allow overwriting files outside the export dire |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Sensitive reports are created without private file permissio |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV fields are written without escaping; review-security: Exported user input can execute spreadsheet formulas |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: Snapshots are published before buffered data is made durable |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: Report parsing retains rows from previous calls |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: The first expired link terminates the janitor; review-concurrency: Concurrent store changes can permanently terminate the janit |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: warm_cache returns before its worker threads finish |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: Target checking crashes when no failures are found |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: Counting links for a new owner raises KeyError |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-concurrency: The janitor has no completed shutdown lifecycle |
| Y53 | — | parallel: X01 the counters are incremented from the janitor thread and |  |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: Malformed create payloads escape the client-error handler; review-security: Link creation lets users impersonate another owner |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete another user's links |
| Y59 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: A caller-controlled header grants administrative access |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: The stats endpoint crashes when the store is empty |
| Y62 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect lookup changes valid case-sensitive codes |
| Y63 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: The report endpoint exposes other users' link records |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: The default report request reads the wrong filename |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) store.py:45 — Generated code collisions silently overwrite existing links
- (medium) main.py:29 — Responses declare JSON while returning unencoded text or CSV
- (medium) main.py:89 — Invalid pagination values raise uncaught exceptions
- (medium) cache.py:40 — Updating a full cache unnecessarily evicts another entry
- (medium) export.py:43 — Snapshot replacement fails across filesystems
- (medium) export.py:66 — Every generated report produces a spurious parsed row
- (medium) store.py:32 — Links created without tags share the same mutable list
- (high) export.py:21 — Concurrent reports overwrite files before responses read them
- (medium) auth.py:43 — Session mutation can interrupt token verification
- (medium) export.py:43 — Concurrent snapshots share one temporary file
