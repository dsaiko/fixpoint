# claude-fable-5-2 · python · run 20261110-151014 (repeat 1)

recall **56/66** · 83 finding(s), 10 unmatched · 33591 tokens · 370s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create() uses a mutable default for tags, shared by every li |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError on an owner's first link, after the |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links are deleted,  |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates; review-security: rename validates `frm` twice and never validates the new cod |
| Y05 | YES | parallel: C04 page clamps end past the list length; Python slicing sil | review-bugs: page() has an off-by-one end bound and mishandles negative o |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() uses integer division, so it reports 0 unless |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() does not give the owner's quota slot back |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending and returns the least followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() matches by substring instead of equality; review-security: by_owner matches by substring and returns other owners' link |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links and resets the expiry instead |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live dict instead of a snapshot |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic and silently overwrites existing  |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() deletes while iterating and returns the wrong count |
| Y15 | YES | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja | review-concurrency: Store readers iterate the live links dict while other thread |
| Y16 | — | PY-ONLY (parallel: S02) codes come from the `random` module rather tha |  |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret when LINKD_SECRET is unset |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens are minted with the non-cryptographic `random |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Live session token written to the log |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks expires_at, so expired sessions stay v; review-security: verify() never checks expires_at, so sessions never expire |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() always returns False; review-security: require_admin condition is always true, so the role check ca |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with a single unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token() raises IndexError on a missing or malformed A; review-security: bearer_token crashes on a missing or malformed Authorization |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals uses a non-constant-time comparison |
| Y25 | YES | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a | review-bugs: parse_lifetime() swallows every exception and turns bad inpu |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke() raises KeyError for an unknown or already-revoked t |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: set() evicts the oldest-inserted entry, not the least recent |
| Y29 | YES | parallel: CA06 warm's peek and set take the lock separately, so the ja | review-concurrency: Cache.warm is a check-then-act race, and peek/stats/utilizat |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() deletes from the dict while iterating it; review-concurrency: Cache.sweep deletes from entries while iterating it, killing |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() uses integer division and divides by zero for  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks: it calls delete() while holding the non-r; review-concurrency: Cache.flush self-deadlocks: calls delete() while holding the |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | YES | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo | review-bugs: An invalid LINKD_PORT is silently ignored |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A bad LINKD_TIMEOUT_MS sets the timeout to 0 instead of keep; review-bugs: LINKD_TIMEOUT_MS is stored as milliseconds in a field whose  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_CACHE_TTL_MS is not converted to seconds and crashes o; review-bugs: A non-numeric LINKD_CACHE_SIZE crashes startup with an unhan |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| Y38 | YES | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, | review-bugs: load_file() swallows every read error, leaks the file handle |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() always returns True and never refuses a bad confi |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in export_csv/read_report: user-supplied repo |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Report files are created with default permissions despite th |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: export_csv() writes fields without CSV quoting; review-security: CSV/formula injection: user-controlled fields written unesca |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all builds file paths from the unvalidated owner nam |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot() renames before the data is flushed and fail; review-concurrency: write_snapshot uses one fixed temp path shared by every conc |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report() uses a mutable default argument, so rows accu |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: The janitor deletes from store.links while iterating a lazy ; review-concurrency: Janitor deletes from store.links while lazily iterating it,  |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache() returns without joining its threads; review-concurrency: warm_cache starts threads and never joins them, contrary to  |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() raises IndexError when every target is fine |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner() raises KeyError for an owner with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-bugs: The janitor thread is non-daemon and stop_janitor() cannot s; review-concurrency: Janitor is a non-daemon thread with no prompt cancellation a |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are incremented without synchronization |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: create_link() crashes on malformed bodies instead of returni; review-security: create_link takes the owner from the request body instead of |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: delete_link performs no ownership check (IDOR) |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect in /out via unvalidated ?next= |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorises on a client-supplied x-admin heade |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: /stats divides by zero when the store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows any origin together with credentials |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report exposes every owner's links to any authenticated use |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-security: Error responses leak internal paths, exception text and the  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with no name writes links.csv but reads back .csv; review-concurrency: report() writes then re-reads a shared, name-derived file wi |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.py:40 — set() raises IndexError when the limit is 0
- (medium) config.py:94 — validate() accepts cache_size 0 and does not check cache_ttl or fetch_limit
- (medium) export.py:65 — parse_report() emits a bogus row for the trailing newline
- (low) export.py:58 — read_report() never closes the file it opens
- (medium) main.py:89 — list_links() crashes on non-numeric offset/size and accepts negative values
- (medium) main.py:78 — redirect() and delete_link() raise KeyError when ?code= is missing
- (medium) main.py:116 — out() answers 301, so repeat clicks are cached by the browser and never counted
- (medium) store.py:36 — create() does not check for a code collision and silently overwrites an existing link
- (high) store.py:54 — Store has no lock: check-then-act and read-modify-write races on links and quota
- (high) auth.py:43 — Authenticator.sessions iterated in verify() while issue()/revoke() mutate it unsynchronize
