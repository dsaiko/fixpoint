# claude-haiku-5-5 · python · run 20261007-205107 (repeat 1)

recall **51/66** · 78 finding(s), 8 unmatched · 82686 tokens · 391s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | — | PY-ONLY the default `tags=[]` is created once at definition time and s |  |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError on the first link of every owner |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() deletes live links and resolves expired ones (inve; review-security: Redirect expiry check is inverted: live links are deleted, e |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates the old code twice and never validates th; review-security: rename validates the old code twice and never validates the  |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() integer-divides before multiplying |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not return the owner's quota slot, contrary to |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending, returning the least-followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner uses substring matching and returns other owners' l; review-security: by_owner uses substring match, leaking other owners' links |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives a link that has already expired; review-security: extend() revives expired links despite its docstring |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-concurrency: quotas() returns the live internal dict, contradicting its o |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic, so a bad link leaves earlier links |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-concurrency: prune() deletes from the links dict while iterating it, with |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() deletes during iteration and returns the remaining c |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback signing secret when LINKD_SECRET is unset |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens come from the global Mersenne Twister and are |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Bearer token printed to stdout on every issue |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks session expiry; review-security: Session expiry is never enforced |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin always returns False because its rejection con; review-security: require_admin rejects every admin and every owner |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted, fast SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token raises IndexError when the Authorization header |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals uses == although its docstring promises consta |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: eviction drops the first-inserted entry, not the least recen |
| Y29 | YES | parallel: CA06 warm's peek and set take the lock separately, so the ja | review-concurrency: warm() checks membership outside the lock, so concurrent war |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() deletes from the dict it is iterating; review-concurrency: sweep() deletes entries while iterating self.entries, raisin |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() integer-divides before multiplying and divides |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() calls delete() while holding the non-reentrant lock; review-concurrency: flush() re-acquires the non-reentrant lock via delete() and  |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: LINKD_TIMEOUT_MS is stored unconverted, and a parse failure  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is stored unconverted into a field the ca |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded; review-security: LINKD_FETCH_LIMIT is parsed and then discarded |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() reports problems but always returns True |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in export_csv and read_report via the name pa |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Report claimed to be owner-only is written with default perm |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: CSV rows are written without quoting, so commas in a field s; review-security: CSV export does not neutralize spreadsheet formulas |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all builds file paths from owner names |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot renames the temp file without closing or flus; review-concurrency: write_snapshot() uses one fixed temp path, so concurrent sna |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report's default rows list is shared across calls; review-concurrency: parse_report() has a mutable default list shared by every ca |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: janitor deletes links while iterating the live dict view fro; review-concurrency: Janitor deletes from the live links dict while iterating it, |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache starts probe threads and returns without joining ; review-concurrency: warm_cache() returns before its probe threads finish; the do |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets raises IndexError when every target passes |
| Y50 | — | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo |  |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-concurrency: Janitor thread is non-daemon and never stopped, and the _run |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are unsynchronised read-modify-write increm |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | YES | parallel: S06 an absent Authorization header becomes the empty string, | review-security: Malformed Authorization header raises IndexError instead of  |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-security: 401 response echoes the presented token |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: create_link crashes on malformed JSON or missing fields inst; review-security: Link owner is taken from the request body, not the session |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: Any authenticated user can delete any link |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: /out redirects to any ?next= value (open redirect, header in |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin quota endpoint trusts a client-supplied header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats() raises ZeroDivisionError on an empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Wildcard CORS with credentials on every response |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: Report endpoint returns every owner's links to any caller |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-security: Export error exposes the server's filesystem path |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report() without a name always returns 500; review-concurrency: report() writes to a name-derived shared path and reads it b |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.py:41 — set() raises IndexError when the cache limit is 0
- (medium) store.py:45 — create() silently overwrites an existing link when a 32-bit code collides
- (high) auth.py:43 — verify() iterates the live sessions dict while issue() and revoke() mutate it from other t
- (medium) store.py:101 — delete() check-then-act race surfaces a KeyError that main.py does not catch
- (medium) store.py:67 — rename() check-then-act lets two renames silently overwrite one link
- (medium) store.py:46 — Per-owner quota is a non-atomic read-modify-write on a shared dict
- (medium) store.py:90 — success_rate, total_hits, and by_owner iterate the live links dict from request threads
- (low) store.py:60 — resolve() increments link.hits without synchronisation, losing hits under concurrent redir
