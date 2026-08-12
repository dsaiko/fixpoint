# gpt-oss:120b-cloud · design · run 20260812-222405 (repeat 1)

recall **6/8** · 6 finding(s), 1 unmatched · 56759 tokens · 40s

| seed | found | note | matched by |
|---|---|---|---|
| D01 | YES | retrying the same POST after a timeout re-applies edits the server alr | design-failure: Partial‑failure handling on server can cause duplicate proce |
| D02 | — | §5 claims one transaction per request AND that rows 1..N-1 survive a f |  |
| D03 | YES | conflict resolution by client wall clock silently discards edits, cont | design-failure: Conflict resolution depends on client wall‑clock timestamps |
| D04 | YES | a failed token refresh wipes notes.db including the unsynced outbox —  | design-failure: Auth revocation wipes local `notes.db` and discards unsynced |
| D05 | YES | no outbox cap plus whole-outbox-in-one-POST: a long-offline device pro | design-failure: Unbounded outbox growth on prolonged offline periods; design-failure: Partial‑failure handling on server can cause duplicate proce |
| D06 | YES | desktop client and CLI open the same SQLite file with no locking story | design-failure: Conflict resolution depends on client wall‑clock timestamps; design-failure: Auth revocation wipes local `notes.db` and discards unsynced |
| D07 | YES | healthz checks process liveness only; an instance with a dead DB conne | design-failure: Health check does not verify database connectivity |
| D08 | — | rolling deploys guarantee old/new overlap; 'deploys are quick' is not  |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) DESIGN.md:36 — Infinite client retry loop with no back‑off ceiling
