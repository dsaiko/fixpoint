# mistral-large-3:675b-cloud · code · run 20260812-205635 (repeat 1)

recall **1/12** · 4 finding(s), 3 unmatched · 281329 tokens · 637s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | — | s.quota is never initialized in NewStore; first Create panics with nil |  |
| C02 | — | expiry comparison inverted: live links are deleted, expired ones resol |  |
| C03 | — | Rename validates `from` twice; `to` is never validated |  |
| C04 | — | Page clamps end to len(all)+1; last page slices out of range |  |
| C05 | — | used/len*100 in integer arithmetic is 0 for every rate under 100% |  |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | — | AddResolved has a value receiver; the mutex is copied and the incremen |  |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Copied mutex in NewWorker enables data races |
| X03 | — | wg.Add inside the goroutine races wg.Wait; WarmCache can return before |  |
| X04 | — | CheckTargets returns on first error; remaining sends to the unbuffered |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) worker.go:23 — Goroutine leak on worker shutdown
- (high) worker.go:32 — Missing context propagation in worker goroutine
- (medium) main.go:17 — TOCTOU race on store.data
