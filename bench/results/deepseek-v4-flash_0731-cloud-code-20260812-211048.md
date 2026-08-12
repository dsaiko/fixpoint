# deepseek-v4-flash:0731-cloud · code · run 20260812-211048 (repeat 1)

recall **12/12** · 14 finding(s), 0 unmatched · 154297 tokens · 213s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Create panics on nil quota map |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Expiry check in Resolve is inverted |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates from twice and never validates to |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page panics with slice bounds out of range |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate computes the percentage with integer division fi |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Atoi error on ttl is swallowed |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview leaks the response body |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead can return nil and client.Do(nil) panics |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved value receiver discards the increment; review-concurrency: Metrics.AddResolved has a value receiver: increments are los |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-bugs: AddResolved value receiver discards the increment; review-concurrency: Store's links map and Link fields are accessed without synch |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-concurrency: WarmCache calls wg.Add inside the goroutine, racing with wg. |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-bugs: CheckTargets leaks goroutines after the first failure; review-concurrency: CheckTargets leaks goroutines and response bodies when the f |
