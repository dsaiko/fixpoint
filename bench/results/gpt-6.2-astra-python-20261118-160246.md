# gpt-6.2-astra · python · run 20261118-160246 (repeat 1)

recall **46/66** · 89 finding(s), 24 unmatched · 125030 tokens · 318s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: Creating a first link raises KeyError after storing it |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolving a live link deletes it; review-security: Resolving a live link deletes it, while expired links reveal |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: Rename validates the source twice and accepts an invalid des |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: The dashboard reports zero success for every partially used  |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Removing links leaves their owners' quota counts unchanged |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: The leaderboard selects the least-followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: Owner lookup includes substring matches; review-security: Owner-scoped lookups include records belonging to other user |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: Extending a link can shorten its lifetime or revive an expir |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: Mutating a quota snapshot changes the live store |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: A rejected batch import leaves earlier records installed |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-bugs: Pruning stops after deleting the first expired link; review-concurrency: Readers can observe a partially imported batch |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Successful imports omit the imported links from owner counts; review-bugs: Prune reports the remaining store size as the removal count |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-concurrency: Dashboard calculations traverse links while requests can cha |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens use the same predictable generator as visible |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Token issuance writes usable bearer credentials to stdout |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions accumulate indefinitely; review-security: Expired bearer tokens continue to authorize requests |
| Y21 | — | parallel: S07 the admin check is always true, so require_admin rejects |  |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Stored password hashes permit fast offline guessing |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: Requests without an Authorization header crash before the 40 |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Cache eviction ignores recent accesses; review-bugs: Updating an existing key can evict an unrelated cache entry |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Sweeping an expired cache entry raises RuntimeError; review-concurrency: An expired cache entry terminates the background sweep loop |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: Partially occupied caches report zero utilization |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-concurrency: Flushing the cache deadlocks every subsequent locked cache o |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: An invalid timeout overwrites the default with zero |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Configured cache lifetimes are 1,000 times too long |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: A valid fetch-limit override is discarded |
| Y38 | YES | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, | review-bugs: Unreadable configuration files silently fall back to default |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: Invalid configuration passes startup validation |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Report names allow overwrites outside the export directory |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Aggregate reports can be readable by other local users |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV reports do not round-trip fields containing delimiters; review-security: User-controlled CSV fields can inject spreadsheet formulas |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: Archive filenames allow owner identifiers to escape the arch |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: The snapshot is published before its contents are durable; review-bugs: Snapshots cannot be installed across filesystem boundaries |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-security: Report parsing retains previous callers' rows |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: The first expired link terminates the janitor; review-concurrency: Deleting the first expired link terminates the janitor |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-concurrency: Stopping the janitor returns before its final sweep and prev; review-concurrency: Cache warming returns while workers are still writing |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: Checking healthy targets raises IndexError |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: Counting links for a new owner raises KeyError |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-concurrency: Importing the HTTP module can prevent process exit |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Concurrent requests can lose dashboard counter increments |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: Malformed create requests escape request error handling; review-security: Callers can create links attributed to another owner |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete another owner's link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-bugs: A permanent redirect lets repeat clicks bypass the click cou; review-security: The email click endpoint redirects to an unrestricted destin |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: The quota endpoint trusts a caller-supplied admin flag |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: The stats endpoint crashes on an empty store |
| Y62 | YES | parallel: C16 redirect lowercases the code before lookup, but codes ar | review-bugs: Redirect cannot resolve valid mixed-case codes |
| Y63 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: Downloading a report without a name reads the wrong file; review-concurrency: Concurrent same-name report requests can return the wrong CS |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (critical) store.py:45 — A generated code collision overwrites an existing link
- (high) main.py:29 — The HTTP response declares JSON for CSV and plain-text bodies
- (medium) store.py:32 — Links created without tags share one mutable list
- (medium) store.py:34 — URL validation accepts targets with no host
- (medium) cache.py:35 — An expired entry prevents cache warming
- (medium) auth.py:50 — The admin-role check rejects administrators
- (medium) export.py:61 — Parsing a report returns rows from previous calls
- (medium) export.py:65 — Every generated report parses with a phantom final row
- (medium) main.py:68 — A numeric zero TTL silently becomes thirty days
- (medium) main.py:78 — Missing code parameters crash link lookup and deletion
- (medium) main.py:89 — Invalid pagination values crash or return unintended pages
- (medium) main.py:140 — CSV downloads silently omit links after the first thousand
- (medium) worker.py:67 — The target probe classifies links solely by URL scheme
- (high) store.py:58 — Concurrent removals can turn a missing link into an unhandled KeyError
- (high) auth.py:43 — Session changes can interrupt verification of an unrelated request
- (medium) store.py:67 — Concurrent renames can silently discard a link
- (medium) cache.py:57 — Cache warming can overwrite a newer target
- (medium) cache.py:35 — Unlocked cache readers can observe a replacement halfway through
- (medium) store.py:60 — Concurrent resolutions can undercount hits on the same link
- (medium) store.py:46 — Concurrent creates can lose an owner's quota increment
- (medium) export.py:61 — Concurrent report parsers append into each other's results
- (critical) main.py:140 — Any authenticated user can export other owners' links
- (high) main.py:140 — Repeated report requests can exhaust the export filesystem
- (high) store.py:45 — Short-code collisions silently replace another owner's link
