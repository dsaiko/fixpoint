# ollama-direct@glm-5.3-flash:cloud#max · python · run 20261028-125225 (repeat 1)

recall **47/66** · 65 finding(s), 5 unmatched · 614798 tokens · 390s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: Default tags list is shared between links; review-concurrency: Store state is shared without synchronization |
| Y02 | — | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s |  |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Valid links are deleted instead of expired links; review-security: Resolution condition is inverted, deleting live links and se |
| Y04 | — | parallel: C03 rename validates `frm` twice; `to` is never validated |  |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: Page end calculation is misleading |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: Success rate uses integer floor division incorrectly |
| Y07 | — | parallel: C09 delete drops the link but never returns the owner's quot |  |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard is sorted from fewest to most hits |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: Owner matching uses substring containment |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend resurrects already expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: Quotas expose internal mutable state and changes are not mai |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Batch import is not atomic and can overwrite existing links |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-bugs: Prune mutates the dictionary while iterating it |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune returns the wrong count |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens use insecure random generation |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Full bearer session tokens are written to logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Session expiry is never checked; review-security: Expired session tokens remain valid |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Administrator authorization can never succeed |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing uses unsalted fast SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: Missing Authorization header crashes request handling; review-security: Malformed Authorization header causes an unhandled exception |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-bugs: Secret comparison leaks timing information |
| Y25 | YES | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a | review-bugs: Bare except hides all error classes |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: Revocation fails ungracefully for unknown or expired tokens |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-bugs: Peek reads shared cache state without locking; review-concurrency: peek bypasses the cache lock |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction is insertion order, not LRU |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-concurrency: Cache sweep deletes while iterating the cache |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: Cache utilization is calculated as a floor rather than a per |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: Cache flush deadlocks on its own lock; review-bugs: Flush cleanup call is semantically dead |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Cache TTL unit mismatch causes incorrect expiry |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: Fetch limit is parsed but never assigned |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: Configuration file parsing crashes on malformed integer valu; review-bugs: Validation is not actually applied |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Report names allow path traversal and arbitrary path writes |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Sensitive reports are not created with restrictive permissio |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV writing does not escape delimiter-sensitive values; review-concurrency: Concurrent exports to the same name can corrupt reports |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-bugs: Per-owner archive filenames permit path traversal |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: Snapshot is not durable and uses a shared fixed temp path; review-security: Snapshot uses a predictable, attacker-controlled symlink pat |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: Report parser uses shared mutable state and creates an empty |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-concurrency: Janitor mutates the store while iterating a live view |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: warm_cache leaks work and claims the threads are done |
| Y49 | — | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- |  |
| Y50 | — | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo |  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-concurrency: Janitor thread has no safe shutdown or restart lifecycle |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are read-modify-write races |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: Authentication accepts an unknown token role without authori |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: Create endpoint crashes on malformed or missing JSON; review-security: Links can be created under another user's owner identity |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete any owner's link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect from arbitrary next parameter |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-bugs: Admin quota check uses a custom header rather than the verif; review-security: Admin authorization is controlled by a client-supplied heade |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: Stats divide by zero when the store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exposes every owner's links |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-bugs: Export path accepts traversal from a user-controlled name |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Default export name is read from the wrong filename |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) cache.py:40 — Zero-size cache is accepted and then crashes
- (medium) config.py:96 — Configuration validation accepts invalid cache size and timeout
- (medium) export.py:58 — Read report leaks an open file handle on success or exception
- (medium) main.py:88 — List endpoint discards its own links data
- (medium) auth.py:43 — Session map is scanned without synchronization
