# glm-5.2:cloud · code · run 20260812-211519 (repeat 1)

recall **12/12** · 15 finding(s), 0 unmatched · 174445 tokens · 155s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: NewStore never initializes the quota map; first Create panic; review-concurrency: Store maps and Link.Hits are accessed concurrently with no s |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve expiry check is inverted: valid links are deleted, e; review-concurrency: Store maps and Link.Hits are accessed concurrently with no s |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates `from` twice and never validates `to` |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page panics with slice out of range whenever the page extend |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate uses integer division before the multiply, alway |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink ignores strconv.Atoi error on TTL, silently creat |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview never closes the target response body, leaking conne |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead swallows NewRequest errors and returns a nil reques |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver, so the resolved counter ne; review-concurrency: AddResolved has a value receiver, copying the mutex and losi |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-concurrency: Store maps and Link.Hits are accessed concurrently with no s; review-concurrency: AddResolved has a value receiver, copying the mutex and losi |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: WarmCache calls wg.Add inside the goroutine, so Wait can ret; review-concurrency: WarmCache calls wg.Add inside the spawned goroutine, racing  |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-bugs: CheckTargets leaks goroutines when it returns on the first f; review-concurrency: CheckTargets leaks goroutines blocked on an unbuffered chann |
