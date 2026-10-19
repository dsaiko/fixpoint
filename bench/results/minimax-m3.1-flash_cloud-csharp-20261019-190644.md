# minimax-m3.1-flash:cloud · csharp · run 20261019-190644 (repeat 2)

recall **27/68** · 30 finding(s), 3 unmatched · 120987 tokens · 4775s

| seed | found | note | matched by |
|---|---|---|---|
| N01 | — | CS-ONLY (parallel: C01) the Dictionary indexer throws KeyNotFoundExcep |  |
| N02 | YES | parallel: C02 expiry comparison inverted: live links are deleted, expi | review-bugs: Resolve() inverts the expiry check: live links are deleted,  |
| N03 | YES | parallel: C03 rename validates `from` twice; `to` is never validated | review-bugs: Rename() validates `from` twice and never validates `to` |
| N04 | YES | parallel: C04 page clamps end past the list size, so GetRange throws o | review-bugs: Page() overflow path sets end to all.Count + 1, making GetRa |
| N05 | — | parallel: C05 integer division before scaling yields 0 for every rate  |  |
| N06 | YES | parallel: C09 delete drops the link but never returns the owner's quot | review-bugs: Delete() never decrements the owner's quota, contradicting i |
| N07 | YES | parallel: C10 top sorts by hits ascending, so the leaderboard shows th | review-bugs: Top() sorts ascending by Hits, returning the least-followed  |
| N08 | YES | parallel: C11 ByOwner matches owners by substring, so one owner sees a | review-bugs: ByOwner() uses substring matching, returning links owned by  |
| N09 | YES | parallel: C12 Extend re-dates an already-expired link, resurrecting a  | review-bugs: Extend() resurrects an already-expired link, contradicting t |
| N10 | — | parallel: C14 Quotas hands out the store's own dictionary while the do |  |
| N11 | YES | parallel: C13 ImportAll stores as it validates, so a bad code leaves t | review-bugs: ImportAll() is not atomic: a later invalid code leaves earli |
| N12 | YES | parallel: C15 Prune counts removals into `removed` and then returns th | review-bugs: Prune() returns the total link count instead of the number o |
| N13 | — | CS-ONLY Prune removes from the dictionary while enumerating its Keys,  |  |
| N14 | — | CS-ONLY expiry is computed from DateTime.Now, so every link's lifetime |  |
| N15 | YES | CS-ONLY (parallel: S02) codes come from System.Random, which is not a  | review-bugs: NewCode() mints short codes from a non-cryptographic Random, |
| N16 | — | CS-ONLY Followed returns a deferred LINQ query with a side-effecting l |  |
| N17 | — | parallel: S01 the signing secret falls back to a hardcoded development |  |
| N18 | — | parallel: S02 session tokens come from System.Random, so they are pred |  |
| N19 | — | parallel: S03 every issued session token is logged in cleartext |  |
| N20 | YES | parallel: S04 Verify never looks at ExpiresAt, so a token works foreve | review-bugs: Verify() never checks Session.ExpiresAt, so tokens stay vali |
| N21 | YES | parallel: S07 RequireAdmin returns true from both branches, so every s | review-bugs: RequireAdmin() returns true on both branches, so the admin r |
| N22 | YES | parallel: S05 passwords are stored as a single unsalted SHA-256, crack | review-bugs: HashPassword uses an unsalted SHA-256 digest, unsuitable for |
| N23 | YES | parallel: S06 BearerToken indexes parts[1] without checking, so a malf | review-bugs: BearerToken() indexes parts[1] with no bounds check, throwin |
| N24 | YES | SecretEquals is documented as not leaking how much matched and uses == | review-bugs: SecretEquals is a plain string comparison, not a constant-ti |
| N25 | — | the sessions dictionary is read and written from every request task wi |  |
| N26 | — | sessions are removed only by an explicit Revoke, so an untouched sessi |  |
| N27 | — | CS-ONLY every method locks on `this`, which any external caller can al |  |
| N28 | — | parallel: CA02 Peek reads the dictionary with no lock at all while loc |  |
| N29 | — | parallel: CA07 Stats reads the hit counter outside the lock while Get  |  |
| N30 | YES | parallel: CA08 eviction drops whatever key comes first, not the least  | review-bugs: Cache eviction picks entries.Keys.First(), which is insertio |
| N31 | — | parallel: CA06 Warm's Peek and Set are separately locked, so the janit |  |
| N32 | — | CS-ONLY Sweep removes entries while enumerating Keys, throwing Invalid |  |
| N33 | — | parallel: C05 Utilization divides before scaling so it always reports  |  |
| N34 | YES | parallel: CF01 TryParse's result is discarded and it writes 0 into Por | review-bugs: Malformed LINKD_PORT overwrites the default with 0 instead o |
| N35 | YES | parallel: CF02 the parse-error branch logs 'keeping default' and then  | review-bugs: A malformed LINKD_TIMEOUT_MS leaves the timeout at zero inst |
| N36 | YES | parallel: CF03 LINKD_CACHE_TTL_MS is documented in milliseconds and bu | review-bugs: LINKD_CACHE_TTL_MS is read as milliseconds but assigned as s |
| N37 | YES | parallel: CF04 LINKD_FETCH_LIMIT is parsed, validated and self-assigne | review-bugs: LINKD_FETCH_LIMIT is parsed and then discarded |
| N38 | — | parallel: CF05 LoadFile swallows every exception, so an unreadable con |  |
| N39 | — | parallel: CF06 Validate logs the problems it finds and returns true, s |  |
| N40 | — | LoadFile parses cache_size with int.Parse inside the try's successor,  |  |
| N41 | YES | parallel: EX01 the report name comes from the query string; Path.Combi | review-bugs: ExportCsv()'s name is not confined to dir, so a caller-suppl |
| N42 | — | CS-ONLY the StreamWriter is closed only on the success path with no `u |  |
| N43 | — | parallel: EX03 CSV rows are built by interpolation, so a comma or quot |  |
| N44 | YES | ArchiveAll builds a filename straight from the owner string, so an own | review-bugs: ArchiveAll() uses the owner name as a filename without valid |
| N45 | — | parallel: EX02 the report holding every owner's links is written with  |  |
| N46 | — | parallel: EX06/EX07 the snapshot temp file has one fixed name in /tmp  |  |
| N47 | YES | CS-ONLY async void: the caller cannot await it and an exception inside | review-bugs: ExportCsvAsync is async void and writes only a header, ignor |
| N48 | — | parallel: X01 AddCreated increments with ++ while AddResolved uses Int |  |
| N49 | — | the errors counter is reported by Snapshot and never assigned anywhere |  |
| N50 | — | CS-ONLY (parallel: X02) the janitor mutates the store's dictionary whi |  |
| N51 | — | parallel: X03 WarmCache starts a Task per link and awaits none of them |  |
| N52 | YES | CS-ONLY blocking on .Result is sync-over-async: it ties up a thread-po | review-bugs: CheckTargets() blocks on async work with .Result, risking a  |
| N53 | YES | CS-ONLY CountForOwner indexes the dictionary directly, throwing KeyNot | review-bugs: CountForOwner() throws KeyNotFoundException for an owner wit |
| N54 | — | CS-ONLY FollowedSummary enumerates the deferred query twice -- once fo |  |
| N55 | — | parallel: S06 an absent Authorization header becomes the empty string  |  |
| N56 | — | parallel: S08 the 401 response echoes the rejected token back to the c |  |
| N57 | YES | parallel: C17 the delete branch never checks the owner its comment pro | review-bugs: DELETE /links deletes any code without checking it belongs t |
| N58 | — | parallel: C18 /out redirects to any address in ?next= -- an open redir |  |
| N59 | — | parallel: C19 the admin endpoint is gated on a request header the call |  |
| N60 | YES | parallel: C20 /stats divides by the link count, throwing DivideByZeroE | review-bugs: Stats divides by store.Count, throwing DivideByZeroException |
| N61 | — | parallel: C16 redirect lowercases the code before lookup although code |  |
| N62 | — | parallel: C23 wildcard CORS combined with Allow-Credentials on every r |  |
| N63 | — | parallel: C28 the link owner comes from the query string instead of th |  |
| N64 | — | parallel: C29 /report exports every owner's links to any authenticated |  |
| N65 | — | offset and size are parsed with unguarded int.Parse, so any non-numeri |  |
| N66 | — | parallel: X02 every request runs on a thread-pool thread against a Sto |  |
| N67 | — | parallel: C26 the accept loop never exits, so StopJanitor is unreachab |  |
| N68 | — | HARVESTED from calibration (2026-09-01, reported by both baselines). e |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) Config.cs:126 — Validate() always returns true and only prints, so a bad config never fails startup
- (medium) Program.cs:183 — Report() catches only IOException, so a bad export name crashes the request instead of ret
- (medium) Store.cs:41 — Create() counts quota per owner but a rename or import can leave quota inconsistent
