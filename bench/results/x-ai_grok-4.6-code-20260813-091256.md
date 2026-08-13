# x-ai/grok-4.6 · code · run 20260813-091256 (repeat 1)

recall **12/12** · 13 finding(s), 0 unmatched · 95190 tokens · 313s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Create panics on a nil quota map |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve deletes live links and serves expired ones |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates the old code twice and never the new one |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page slice end is len+1 and panics |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate integer division is always 0 or 100 |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid TTL is silently turned into a zero duration |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the target response body |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns a nil request and CheckTargets will panic |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved uses a value receiver so counts are discarded; review-concurrency: AddResolved copies Metrics and its mutex |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store map and Link fields used concurrently with no lock; review-concurrency: AddResolved copies Metrics and its mutex |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-concurrency: WarmCache calls WaitGroup.Add inside the goroutine |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-concurrency: CheckTargets leaks goroutines on the first error |
