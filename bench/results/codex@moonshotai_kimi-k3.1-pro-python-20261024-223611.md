# codex@moonshotai/kimi-k3.1-pro · python · run 20261024-223611 (repeat 1)

recall **40/66** · 56 finding(s), 9 unmatched · 728262 tokens · 491s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: Default tags list is shared across links |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: Creating the first link for an owner always crashes |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Expiry comparison is inverted |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: Rename validates the source twice and never validates the de |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: Final page end offset exceeds the list length |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: Success rate loses fractional percentages |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Deleting a link never releases its quota |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Leaderboard sorts least-followed links first |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: Owner filtering matches substrings |
| Y10 | — | parallel: C12 extend re-dates an already-expired link, resurrecting a  |  |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: Quota snapshot exposes mutable internal state |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: Batch import is not atomic; review-concurrency: Batch imports are visibly partial to concurrent readers |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: Prune mutates its dictionary during iteration and returns th |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short-link codes are predictable and can collide silently |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens use predictable non-cryptographic randomness |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Live session tokens are written to logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: Expired sessions remain valid; review-security: Expired sessions remain valid indefinitely |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: Admin role check rejects every role |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords use a fast unsalted SHA-256 hash |
| Y23 | — | parallel: S06 bearer_token indexes [1] without checking, so a request  |  |
| Y24 | — | secret_equals is documented as not leaking how much matched and uses = |  |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Updating an existing key can evict another entry |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: Cache sweep crashes after removing an expired entry |
| Y31 | — | PY-ONLY utilization floor-divides before scaling so it always reports  |  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-concurrency: Cache flush deadlocks on its own mutex |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond configuration values are used as seconds |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: Fetch limit environment setting is discarded |
| Y38 | YES | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, | review-bugs: All configuration file read errors are silently treated as m |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: Invalid configuration is reported but still accepted |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Report name permits arbitrary file writes and reads |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV output is malformed for commas, quotes, or newlines; review-security: Unescaped CSV fields enable spreadsheet formula injection |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: Snapshots are neither flushed nor closed before publication; review-concurrency: Fixed snapshot temporary path corrupts concurrent snapshots |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: Parsed report rows leak between calls |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor crashes whenever it finds an expired link; review-concurrency: Janitor consumes a live store iterator while deleting concur |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: Cache warmer returns before background work finishes; review-concurrency: Stopping the janitor does not wait for the thread to stop |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: Successful target checks raise IndexError |
| Y50 | — | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo |  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics increments lose updates under concurrent requests |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | YES | parallel: S06 an absent Authorization header becomes the empty string, | review-bugs: Missing or malformed Authorization headers crash request han |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-security: Link ownership is taken from the request body |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete any link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Arbitrary external open redirect |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Client-controlled header grants access to administrator data |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: Stats endpoint divides by zero when the store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint exports every user's links |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.py:73 — Cache utilization uses integer division before scaling
- (low) export.py:58 — Report reads leak file descriptors
- (high) auth.py:43 — Session verification races with issuance and revocation
- (high) export.py:21 — Concurrent same-name reports overwrite and expose partial output
- (medium) store.py:46 — Concurrent creations lose owner quota updates
- (high) store.py:58 — Resolve and deletion have a check-then-act race
- (medium) store.py:60 — Concurrent redirects lose per-link hit counts
- (high) store.py:67 — Concurrent renames can delete both sources and retain only one link
- (high) store.py:90 — Live dictionary iterations race with request and janitor mutations
