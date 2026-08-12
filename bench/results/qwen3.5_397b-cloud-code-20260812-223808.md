# qwen3.5:397b-cloud · code · run 20260812-223808 (repeat 1)

recall **10/12** · 12 finding(s), 0 unmatched · 205879 tokens · 80s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: nil map panic in quota tracking; review-concurrency: Store maps accessed without synchronization |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: inverted expiry check deletes valid links |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice, never validates 'to' |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page() slice bounds off-by-one causes panic |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate returns 0 due to integer division |
| C06 | — | strconv.Atoi error discarded; bad TTL silently becomes 0 hours |  |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead can return nil request causing panic |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-concurrency: AddResolved uses value receiver, increments a copy |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store maps accessed without synchronization; review-concurrency: quota map never initialized, causes nil panic |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: WaitGroup.Add called inside goroutine causes race; review-concurrency: wg.Add(1) called inside goroutine causes race with Wait |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-concurrency: CheckTargets leaks goroutines on early return |
