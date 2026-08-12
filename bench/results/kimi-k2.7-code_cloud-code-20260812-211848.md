# kimi-k2.7-code:cloud · code · run 20260812-211848 (repeat 1)

recall **11/12** · 12 finding(s), 0 unmatched · 114418 tokens · 94s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Nil quota map causes panic on link creation; review-concurrency: Store methods access shared maps without synchronization |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve treats live links as expired |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates from twice and never validates to |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page panics when offset+size exceeds link count |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate always reports 0% until every link is used |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: Invalid TTL string is silently treated as zero |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: Preview response body is never closed |
| C08 | — | mustHead logs and returns nil request; client.Do(nil) panics |  |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved has a value receiver so resolved counter never u; review-concurrency: AddResolved value receiver copies mutex and loses updates |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-bugs: AddResolved has a value receiver so resolved counter never u; review-concurrency: Store methods access shared maps without synchronization |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-concurrency: WaitGroup Add called inside goroutine causes premature retur |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-concurrency: Unbuffered results channel leaks goroutines on early return |
