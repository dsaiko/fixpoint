# glm-5.3:cloud · python · run 20260901-123329 (repeat 1)

recall **48/66** · 64 finding(s), 3 unmatched · 606987 tokens · 285s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create() shares one mutable tags list across all default-tag |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: Creating the first link for an owner raises KeyError |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: live links deleted, dead; review-concurrency: Store.resolve check-then-act race: concurrent resolve of an  |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates the source code twice and never validates |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() integer-divides before multiplying, always re |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the least-followed links, not the most |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() matches owners by substring; review-security: by_owner matches owners by substring, leaking other users' l |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives expired links, contradicting 'expiry is fin |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live dict, not a snapshot |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic despite its contract |
| Y13 | YES | PY-ONLY prune deletes from the dict while iterating it, raising Runtim | review-concurrency: Store.prune deletes from self.links while iterating it |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() mutates the links dict during iteration and returns  |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short codes minted from 32 predictable bits |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret used when LINKD_SECRET is unset |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens minted from the Mersenne Twister PRNG |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session tokens printed in plaintext to stdout on every issue |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks session expiry; review-security: verify() never checks expires_at, so sessions are permanent |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin always returns False; review-security: require_admin's condition is unsatisfiable, so the role gate |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: hash_password is unsalted SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token raises IndexError on a missing or empty Authori; review-security: bearer_token crashes on any malformed Authorization header |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals does a plain == compare despite its constant-t |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | — | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  |  |
| Y27 | — | parallel: CA02 peek reads the dict without the lock every other method |  |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction picks the oldest-inserted entry, not the least rece |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() mutates the entries dict while iterating it; review-concurrency: Cache.sweep deletes entries from the dict it is iterating |
| Y31 | — | PY-ONLY utilization floor-divides before scaling so it always reports  |  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks on the non-reentrant lock; review-concurrency: Cache.flush deadlocks by re-acquiring its non-reentrant lock |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond env values are stored raw into fields consumed a |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() reports problems but always returns True |
| Y40 | — | parallel: EX01 the report name comes from the query string; os.path.jo |  |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-security: Report files are world-readable despite the comment claiming |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-bugs: export_csv() writes unescaped fields, corrupting rows on com; review-security: CSV formula injection via owner and target fields |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all builds output filenames from the unvalidated own |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot() renames before the data is flushed, and lea; review-security: write_snapshot uses a fixed /tmp path vulnerable to symlink  |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report() accumulates rows across calls via a mutable d |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes store entries while iterating the find_expir; review-concurrency: Janitor deletes store entries while the find_expired generat |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache() returns before its threads finish, contradictin; review-concurrency: warm_cache starts per-pair threads and never joins them |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() raises IndexError when every target is fine |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner() raises KeyError for an owner with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-concurrency: stop_janitor cannot stop the janitor promptly and never join |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters are incremented without synchronization, lo |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: create_link crashes on a payload missing target or owner |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-bugs: delete_link never checks that the session owner matches the ; review-security: delete_link performs no ownership check despite its docstrin |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Unrestricted open redirect in /out |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by a client-controlled x-admin hea |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: stats() divides by zero on an empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: respond() sets Access-Control-Allow-Origin * with Allow-Cred |
| Y64 | YES | parallel: C29 /report exports every owner's links to any authenticated | review-security: /report streams every owner's links to any authenticated use |
| Y65 | YES | report catches only IOError, so the ValueError and KeyError the same c | review-security: Path traversal via the report name: arbitrary file write and |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-concurrency: Report endpoint write-then-read of a shared file with no syn |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) cache.py:73 — utilization() reports 0% unless the cache is exactly full
- (low) config.py:72 — load_file() leaks the file handle it reads
- (medium) store.py:46 — Quota counter update is a lost-update race under concurrent create
