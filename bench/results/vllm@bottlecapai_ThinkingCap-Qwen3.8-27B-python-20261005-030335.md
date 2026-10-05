# vllm@bottlecapai/ThinkingCap-Qwen3.8-27B · python · run 20261005-030335 (repeat 1)

recall **47/66** · 65 finding(s), 5 unmatched · 62611 tokens · 1017s

| seed | found | note | matched by |
|---|---|---|---|
| Y01 | YES | PY-ONLY the default `tags=[]` is created once at definition time and s | review-bugs: create() has a mutable default tags=[] shared by all links |
| Y02 | YES | PY-ONLY (parallel: C01) the quota dict has no entry for a new owner, s | review-bugs: create() raises KeyError for any owner not already in the qu |
| Y03 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: resolve() expiry test is inverted: valid links are deleted, ; review-security: resolve() comparison inverted: expired links keep resolving |
| Y04 | YES | parallel: C03 rename validates `frm` twice; `to` is never validated | review-bugs: rename() validates frm twice and never validates the destina |
| Y05 | — | parallel: C04 page clamps end past the list length; Python slicing sil |  |
| Y06 | YES | PY-ONLY (parallel: C05) floor division is applied before the multiply, | review-bugs: success_rate() uses integer division — reports 0% unless lit |
| Y07 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: delete() never gives the owner's quota slot back |
| Y08 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: top() sorts ascending — returns the least-followed links |
| Y09 | YES | parallel: C11 by_owner uses `in`, a substring test, so one owner sees  | review-bugs: by_owner() matches by substring, not equality; review-security: by_owner matches by substring, leaking other owners' links |
| Y10 | YES | parallel: C12 extend re-dates an already-expired link, resurrecting a  | review-bugs: extend() resurrects already-expired links; review-security: extend() revives already-expired links, contradicting 'expir |
| Y11 | YES | parallel: C14 quotas hands out the store's own dict while the doc call | review-bugs: quotas() returns the live dict, not the snapshot it promises |
| Y12 | YES | parallel: C13 import_all stores as it validates, so a bad code leaves  | review-bugs: import_all() is not atomic — a bad code mid-batch leaves a p |
| Y13 | — | PY-ONLY prune deletes from the dict while iterating it, raising Runtim |  |
| Y14 | YES | parallel: C15 prune counts removals into `removed` and then returns th | review-bugs: prune() returns the number of remaining links instead of rem |
| Y15 | — | PY-ONLY find_expired returns a GENERATOR over the live dict, so the ja |  |
| Y16 | YES | PY-ONLY (parallel: S02) codes come from the `random` module rather tha | review-security: Link codes minted from predictable random (Mersenne Twister) |
| Y17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| Y18 | YES | parallel: S02 session tokens come from the `random` module, which is s | review-security: Session tokens minted from predictable random (Mersenne Twis |
| Y19 | YES | parallel: S03 every issued session token is printed in cleartext | review-security: Issued session token written to stdout/logs |
| Y20 | YES | parallel: S04 verify never looks at expires_at, so a token works forev | review-bugs: verify() never checks expires_at — expired sessions are hono; review-security: verify() ignores expires_at, so sessions never expire |
| Y21 | YES | parallel: S07 the admin check is always true, so require_admin rejects | review-bugs: require_admin() is always False — 'or' where 'and' is needed; review-security: require_admin() always returns False |
| Y22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-security: Password hashing is unsalted single-iteration SHA-256 |
| Y23 | YES | parallel: S06 bearer_token indexes [1] without checking, so a request  | review-bugs: bearer_token() crashes with IndexError when the Authorizatio; review-security: bearer_token() crashes on empty/malformed Authorization head |
| Y24 | YES | secret_equals is documented as not leaking how much matched and uses = | review-security: secret_equals() is not constant-time despite its docstring |
| Y25 | — | PY-ONLY a bare `except:` in parse_lifetime catches KeyboardInterrupt a |  |
| Y26 | YES | PY-ONLY revoke deletes by key without a guard, so revoking an unknown  | review-bugs: revoke() raises KeyError for an unknown or already-revoked t |
| Y27 | YES | parallel: CA02 peek reads the dict without the lock every other method | review-concurrency: peek/stats/utilization/warm touch cache state outside the lo |
| Y28 | YES | parallel: CA08 eviction drops the first key in insertion order, not th | review-bugs: set() evicts by insertion order, not LRU |
| Y29 | — | parallel: CA06 warm's peek and set take the lock separately, so the ja |  |
| Y30 | YES | PY-ONLY sweep deletes from the dict while iterating it, raising Runtim | review-bugs: sweep() deletes entries while iterating the dict — RuntimeEr |
| Y31 | YES | PY-ONLY utilization floor-divides before scaling so it always reports  | review-bugs: utilization() floors to 0% via integer division and crashes  |
| Y32 | YES | PY-ONLY flush holds the non-reentrant Lock and calls delete, which tak | review-bugs: flush() deadlocks: it re-acquires the non-reentrant lock it ; review-concurrency: Cache.flush deadlocks: it calls delete() while already holdi |
| Y33 | — | parallel: CA07 stats reads the hit counter outside the lock while get  |  |
| Y34 | — | parallel: CF01 a malformed LINKD_PORT is swallowed silently with no lo |  |
| Y35 | — | parallel: CF02 the parse-error branch logs 'keeping default' and then  |  |
| Y36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and st | review-bugs: LINKD_TIMEOUT_MS and LINKD_CACHE_TTL_MS are stored raw, 1000 |
| Y37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed but the result is never assigned |
| Y38 | — | PY-ONLY (parallel: CF05) load_file wraps the read in a bare `except:`, |  |
| Y39 | YES | parallel: CF06 validate logs the problems it finds and returns True, s | review-bugs: validate() collects problems but unconditionally returns Tru |
| Y40 | YES | parallel: EX01 the report name comes from the query string; os.path.jo | review-security: Path traversal via report name enables arbitrary file write+ |
| Y41 | YES | parallel: EX02 the report is written with default permissions and with | review-bugs: Export file handles are never closed on the error path; review-security: export_csv never sets restrictive permissions despite claimi |
| Y42 | YES | parallel: EX03 CSV rows are built by string formatting, so a comma or  | review-concurrency: Concurrent /report requests to the same name truncate and in |
| Y43 | YES | archive_all builds a filename straight from the owner string, so an ow | review-security: archive_all builds the filename from owner without sanitizat |
| Y44 | YES | PY-ONLY (parallel: EX06/EX07) one fixed /tmp name, and the handle is n | review-bugs: write_snapshot() renames without flush/close, and /tmp may b; review-security: write_snapshot uses a fixed /tmp path followed by os.rename  |
| Y45 | YES | PY-ONLY the default `rows=[]` is shared across calls, so every parse a | review-bugs: parse_report() has a mutable default rows=[] that accumulate |
| Y46 | — | the errors counter is reported by snapshot and never incremented anywh |  |
| Y47 | YES | PY-ONLY the janitor deletes from store.links while consuming the gener | review-bugs: Janitor deletes from store.links while iterating a generator; review-concurrency: Janitor thread deletes from store.links while iterating a ge |
| Y48 | YES | parallel: X03 warm_cache collects its threads and never joins them, so | review-bugs: warm_cache() returns without joining its threads; review-concurrency: warm_cache spawns one thread per pair and never joins them,  |
| Y49 | YES | PY-ONLY check_targets indexes problems[0] unconditionally, so the all- | review-bugs: check_targets() raises IndexError when every target is fine |
| Y50 | YES | PY-ONLY count_for_owner indexes the dict directly, raising KeyError fo | review-bugs: count_for_owner() raises KeyError for an owner with no links |
| Y51 | — | summarize accumulates into its parameter; the default is an immutable  |  |
| Y52 | — | the janitor thread is not a daemon and is never joined, so the interpr |  |
| Y53 | YES | parallel: X01 the counters are incremented from the janitor thread and | review-concurrency: Metrics counters use unsynchronized += from concurrent reque |
| Y54 | — | PY-ONLY importing this module builds the whole service and STARTS A TH |  |
| Y55 | — | parallel: S06 an absent Authorization header becomes the empty string, |  |
| Y56 | YES | parallel: S08 the 401 response echoes the rejected token back to the c | review-bugs: Public endpoints /l and /out are gated behind a session toke |
| Y57 | YES | parallel: C28 the link owner is taken from the request body instead of | review-bugs: create_link() 500s on a payload missing target/owner instead; review-security: create_link takes owner from request body, not the session |
| Y58 | YES | parallel: C17 delete_link never checks the owner its docstring promise | review-security: delete_link has no ownership check (IDOR) |
| Y59 | YES | parallel: C18 /out redirects to any address in ?next= -- an open redir | review-security: Open redirect via unvalidated ?next= on /out |
| Y60 | YES | parallel: C19 the admin endpoint is gated on a request header the call | review-security: Admin endpoint gated only by a client-controlled header |
| Y61 | YES | parallel: C20 /stats divides by the link count, raising ZeroDivisionEr | review-bugs: /stats crashes with ZeroDivisionError on an empty store |
| Y62 | — | parallel: C16 redirect lowercases the code before lookup, but codes ar |  |
| Y63 | YES | parallel: C23 wildcard CORS combined with Allow-Credentials on every r | review-security: CORS allows wildcard origin together with credentials |
| Y64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| Y65 | — | report catches only IOError, so the ValueError and KeyError the same c |  |
| Y66 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) store.py:45 — create() silently overwrites an existing link on a code collision
- (medium) store.py:78 — Readers scan store.links while request threads and the janitor mutate it, with no locking 
- (medium) store.py:101 — get-then-delete TOCTOU: concurrent deletes of the same code raise KeyError and 500
- (low) store.py:60 — link.hits += 1 is an unsynchronized read-modify-write across concurrent /l resolves
- (low) auth.py:43 — verify iterates the sessions dict while issue/revoke mutate it from other threads (latent)
