# deepseek/deepseek-v4.1-flash-1017 · python · run 20261017-220331 (repeat 1)

recall **48/66** · 67 finding(s), 5 unmatched · 157732 tokens · 549s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create's default tags list is shared by every link created w |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create raises KeyError for any owner creating their first li; review-concurrency: Unguarded read-modify-write on Store.quota and Link.hits los |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve has its expiry test inverted, deleting live links an; review-security: Expiry check is inverted, so expired links keep resolving an |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename validates the source twice and never validates the de |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate uses integer division and almost always reports |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete does not return the owner's quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top sorts ascending, returning the least-followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner matches owner names as substrings; review-security: by_owner uses substring matching, leaking other owners' link |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend revives already-expired links, contradicting its cont |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas returns the live dict instead of a snapshot |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-bugs: prune deletes from the dict while iterating it |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune returns the remaining size, not the number removed |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short-link codes are predictable and drawn from the shared M |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Known hardcoded fallback secret when LINKD_SECRET is unset |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens come from a non-cryptographic PRNG (Mersenne  |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens written to stdout/logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify never checks expires_at, so tokens never expire; review-security: verify() ignores expires_at, so sessions never expire |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin can never return True |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted, single-round SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token raises IndexError when the Authorization header; review-security: bearer_token crashes on missing/malformed Authorization and  |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals is not constant-time despite its docstring |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke raises KeyError for an unknown or already-revoked tok |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: set evicts the oldest-inserted key, not the LRU entry, and e |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep deletes entries while iterating the dict; review-concurrency: Cache.sweep deletes entries while iterating the dict |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization performs integer division first, so it is 0 for  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush deadlocks on its own non-reentrant lock; review-concurrency: Cache.flush self-deadlocks by re-acquiring a non-reentrant l |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is stored as seconds without converting f |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never assigned to cfg.fetch_ |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate collects problems but always returns True |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal: user-supplied report name escapes the export |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-concurrency: Concurrent exports with the same report name race on one fil; review-security: CSV injection through attacker-controlled owner/target field |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot renames an unflushed, unclosed file and never; review-concurrency: write_snapshot uses one fixed /tmp path, so concurrent write |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report has a mutable default argument that accumulates; review-security: parse_report uses a mutable default argument, leaking rows b |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes from the dict while iterating find_expired's; review-concurrency: Janitor deletes from the dict it is iterating, killing the j |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache returns before its threads finish; review-concurrency: warm_cache starts threads and never joins them |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets raises IndexError when there are no problems |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner raises KeyError for owners with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-concurrency: Non-daemon janitor thread is never stopped or joined, so the |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters use unsynchronized += and lose increments |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: create_link lets malformed request bodies escape as unhandle; review-security: Link owner is taken from the request body instead of the ses |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: delete_link performs no ownership check |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via /out?next= |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorizes on a client-supplied X-Admin heade |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats divides by zero when the store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin together with credentials |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exports every owner's links to any authenticated use |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: report reads back a different filename than export_csv wrote |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) export.py:64 — parse_report crashes on empty input and appends a phantom row for trailing newlines
- (high) store.py:78 — page()/top()/success_rate() iterate the shared dict while other threads mutate it
- (medium) store.py:58 — resolve() check-then-act delete raises KeyError under concurrent access
- (low) auth.py:43 — Authenticator.verify iterates sessions while issue/revoke mutate it
- (medium) export.py:21 — Reports containing every owner's data are created world-readable
