# claude-opus-5 · python · run 20260901-161128 (repeat 1)

recall **49/66** · 70 finding(s), 1 unmatched · 43292 tokens · 490s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create()'s mutable default tags list is shared by every link; review-concurrency: Store.links is shared between the janitor thread and request |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError on an owner's first link |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() has the expiry comparison inverted, deleting live ; review-security: Inverted expiry comparison in resolve(): expired links resol |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() floor-divides before scaling, so it is always |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-followed links instead of the most-f |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() substring-matches owners and returns other users'; review-security: by_owner matches owners by substring, returning other owners |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects already-expired links |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live internal dict, not a snapshot |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic despite promising all-or-nothing |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() mutates self.links while iterating it, and returns t |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes are predictable and enumerable (32-bit, non-cryp |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret silently used when LINKD_SECRET is |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens generated from the predictable `random` modul |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to stdout and echoed in 401 responses |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks session expiry; review-security: verify() never checks session expiry, so tokens are valid fo |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always returns False; review-bugs: revoke() raises KeyError for an unknown or already-revoked t |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with a single unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token() raises IndexError on a missing or malformed h; review-security: bearer_token raises IndexError on any request without a well |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals uses a non-constant-time comparison despite pr |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: peek bypasses the cache lock and warm() builds a check-then- |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction picks the oldest inserted entry, not the least rece |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() mutates self.entries while iterating it; review-concurrency: Cache.sweep mutates self.entries while iterating it, killing |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() floor-divides and always reports 0% |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks by calling delete() while holding the lock; review-concurrency: Cache.flush self-deadlocks by re-acquiring a non-reentrant l |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Timeout is assigned in milliseconds and forced to 0 on a par |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is stored as seconds without converting f |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() always returns True, so a bad config never fails ; review-security: validate() reports problems and then accepts the configurati |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in export_csv/read_report via user-supplied r |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Reports containing every owner's data are created world-read |
| Y42 | — | parallel: EX03 CSV rows are built by string formatting, so a comma or  |  |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all writes to a path built from an attacker-controll |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot renames an unclosed, unflushed file and can f; review-security: Snapshot written through a fixed predictable /tmp path (syml |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report's mutable default argument accumulates rows acr; review-bugs: read_report leaks the file handle on every download |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes from store.links while iterating find_expire; review-concurrency: Janitor thread dies on its first tick with an expired link,  |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache never joins the threads it starts; review-concurrency: warm_cache never joins the threads it starts, so it returns  |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets raises IndexError when every target is healthy |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner raises KeyError for an owner with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-bugs: Janitor sleeps before checking the stop flag and blocks inte; review-concurrency: stop_janitor does not join the thread and makes the janitor  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are incremented without synchronization and |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: create_link crashes on malformed or incomplete JSON bodies; review-security: create_link takes the owner from the request body instead of |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-bugs: delete_link ignores the session owner, letting anyone delete; review-security: delete_link performs no ownership check |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect on /out?next= |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on a client-supplied X-Admin heade |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: /stats divides by zero on an empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS origin combined with Allow-Credentials on ever |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report returns every owner's links to any authenticated cal |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report writes one filename and reads back a different one |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.py:40 — set() raises IndexError when limit is 0
