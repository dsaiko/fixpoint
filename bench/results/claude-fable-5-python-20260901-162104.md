# claude-fable-5 · python · run 20260901-162104 (repeat 1)

recall **48/66** · 62 finding(s), 3 unmatched · 40837 tokens · 503s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: Store.create shares one mutable default tags list across lin |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError on an owner's first link |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry check is inverted: deletes live links, serv |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates the source code twice and the destination |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate uses floor division — returns 0 unless every li |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never returns the owner's quota slot |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() returns the n LEAST followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner uses substring match instead of equality; review-security: by_owner matches owner as a substring instead of equality |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() revives already-expired links, contradicting its ow |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live internal dict, not a snapshot |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all is not atomic: a mid-batch validation failure lea |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() mutates the dict during iteration and returns the wr |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Short-link codes generated with predictable 32-bit PRNG |
| Y17 | YES | parallel: S01 the signing secret falls back to a hardcoded development | review-security: Hardcoded fallback secret |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens generated with non-cryptographic PRNG |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Session token written to logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks expires_at, so expired sessions authen; review-security: verify() never checks token expiry |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin rejects every session, including admins |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Passwords hashed with unsalted single-round SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token crashes with IndexError on missing/malformed Au; review-security: bearer_token crashes on missing/malformed Authorization head |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals uses non-constant-time comparison |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke() raises KeyError for an unknown or already-revoked t |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: peek() reads entries without the lock, making warm()'s check |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: Eviction ignores the 'used' timestamp — not LRU as documente |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() deletes from the dict it is iterating — RuntimeError; review-concurrency: sweep() deletes entries while iterating the dict, raising Ru |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() floor-divides — reports 0% until the cache is  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks: calls delete() while already holding the ; review-concurrency: flush() self-deadlocks by calling delete() while holding the |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: Bad LINKD_TIMEOUT_MS sets timeout to 0 despite printing 'kee |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: Millisecond env vars are stored into second-based fields wit |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but never applied |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() reports problems but always returns True; review-bugs: load_file crashes on a non-numeric cache_size line |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal in report name allows arbitrary file write an |
| Y41 | — | parallel: EX02 the report is written with default permissions and with |  |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-security: CSV formula injection via unsanitized fields |
| Y43 | — | archive_all builds a filename straight from the owner string, so an ow |  |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot renames the temp file before closing/flushing; review-concurrency: write_snapshot uses one fixed /tmp path shared by all concur |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report uses a mutable default argument — rows leak acr |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes store entries while iterating a generator ov; review-concurrency: Janitor deletes from store.links while iterating a live gene |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache returns before its threads finish, despite promis; review-concurrency: warm_cache starts threads but never joins them despite promi |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets raises IndexError when there are no problems |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner raises KeyError for owners with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | YES | the janitor thread is not a daemon and is never joined, so the interpr | review-bugs: Janitor thread is non-daemon, so the process cannot exit cle; review-concurrency: Janitor thread is non-daemon and never stopped, so the proce |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters use non-atomic += from concurrent request t |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: create_link crashes on malformed body instead of returning 4 |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: delete_link performs no ownership check (IDOR) |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unchecked ?next= parameter |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint authorized by client-controlled header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: /stats raises ZeroDivisionError when the store is empty |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: Permissive CORS: wildcard origin with credentials |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | YES | HARVESTED from calibration (2026-09-01, reported by both baselines). e | review-bugs: /report with no name writes links.csv but reads back ".csv" |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) export.py:65 — parse_report turns the trailing newline into a bogus row
- (high) store.py:60 — Store shares mutable dicts across request threads and the janitor with no lock
- (medium) auth.py:43 — Authenticator.sessions is read and mutated by concurrent request threads with no lock
