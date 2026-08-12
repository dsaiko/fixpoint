# deepseek-v4-pro:cloud · code · run 20260812-221953 (repeat 1)

recall **12/12** · 15 finding(s), 0 unmatched · 215179 tokens · 99s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: nil map write panics on every Create call |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: expiration check is inverted — non-expired links are deleted |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice, never validates 'to' |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: off-by-one in Page slice bound causes panic |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: integer division truncation makes SuccessRate always return  |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: strconv.Atoi error ignored — invalid TTL silently becomes ze |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: resp.Body is never closed in preview, leaking connections |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: mustHead returns nil *http.Request on error, causing nil-der |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved uses value receiver, mutations are silently lost; review-concurrency: AddResolved uses value receiver — mutex is copied, no synchr |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-bugs: AddResolved uses value receiver, mutations are silently lost; review-concurrency: Store has no mutex — concurrent map access is a fatal runtim |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: wg.Add(1) inside goroutine — Wait may return before any work; review-concurrency: WarmCache calls wg.Add inside the goroutine — Wait may retur |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-bugs: goroutine leak in CheckTargets on early error return; review-concurrency: CheckTargets leaks goroutines on first error — senders block |
