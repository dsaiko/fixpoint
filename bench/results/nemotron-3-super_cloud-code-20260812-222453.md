# nemotron-3-super:cloud · code · run 20260812-222453 (repeat 1)

recall **4/12** · 4 finding(s), 1 unmatched · 519241 tokens · 765s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-concurrency: Unprotected concurrent access to Store maps and Link.Hits |
| C02 | — | expiry comparison inverted: live links are deleted, expired ones resol |  |
| C03 | — | Rename validates `from` twice; `to` is never validated |  |
| C04 | — | Page clamps end to len(all)+1; last page slices out of range |  |
| C05 | — | used/len*100 in integer arithmetic is 0 for every rate under 100% |  |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-concurrency: Metrics.AddResolved uses value receiver, causing data race |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Unprotected concurrent access to Store maps and Link.Hits; review-concurrency: Metrics.AddResolved uses value receiver, causing data race |
| X03 | — | wg.Add inside the goroutine races wg.Wait; WarmCache can return before |  |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-concurrency: Goroutine leak in CheckTargets on early return |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (medium) main.go:86 — Preview handler does not use request context for outgoing HTTP request
