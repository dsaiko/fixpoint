# mistral-large-3:675b-cloud · code · run 20260812-235041 (repeat 1)

recall **7/12** · 8 finding(s), 1 unmatched · 131066 tokens · 1250s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: Quota map is never initialized, causing nil map assignment p |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Incorrect expiration check logic in Resolve method |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Copy-paste error in Rename method validation |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Off-by-one error in Page method slice calculation |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: Integer division truncation in SuccessRate calculation |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: TTL parsing ignores errors from strconv.Atoi |
| C07 | — | preview never closes resp.Body; connections leak |  |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: Unchecked error in mustHead function |
| X01 | — | AddResolved has a value receiver; the mutex is copied and the incremen |  |
| X02 | — | janitor iterates and deletes s.links concurrently with HTTP handlers;  |  |
| X03 | — | wg.Add inside the goroutine races wg.Wait; WarmCache can return before |  |
| X04 | — | CheckTargets returns on first error; remaining sends to the unbuffered |  |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (low) main.go:39 — http.ListenAndServe error handling is too broad
