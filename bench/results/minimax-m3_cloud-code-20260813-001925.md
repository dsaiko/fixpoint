# minimax-m3:cloud · code · run 20260813-001925 (repeat 1)

recall **12/12** · 23 finding(s), 3 unmatched · 242538 tokens · 127s

| seed | found | note | matched by |
|---|---|---|---|
| C01 | YES | s.quota is never initialized in NewStore; first Create panics with nil | review-bugs: nil map write panics in Create; review-bugs: Owner quota increments after link creation, never decremente |
| C02 | YES | expiry comparison inverted: live links are deleted, expired ones resol | review-bugs: Resolve inverts expiry: live links look expired, expired one; review-concurrency: Resolve deletes an entry the moment it is expired but the co |
| C03 | YES | Rename validates `from` twice; `to` is never validated | review-bugs: Rename validates 'from' twice and 'to' not at all |
| C04 | YES | Page clamps end to len(all)+1; last page slices out of range | review-bugs: Page can return out-of-bounds slice on overflow |
| C05 | YES | used/len*100 in integer arithmetic is 0 for every rate under 100% | review-bugs: SuccessRate integer-divides to zero |
| C06 | YES | strconv.Atoi error discarded; bad TTL silently becomes 0 hours | review-bugs: createLink silently ignores TTL parse errors; review-bugs: createLink forwards store error text to clients |
| C07 | YES | preview never closes resp.Body; connections leak | review-bugs: preview leaks response body on ReadAll error |
| C08 | YES | mustHead logs and returns nil request; client.Do(nil) panics | review-bugs: CheckTargets can deadlock on bad request construction; review-concurrency: CheckTargets leaks the results channel and the in-flight gor |
| X01 | YES | AddResolved has a value receiver; the mutex is copied and the incremen | review-bugs: AddResolved locks a value receiver, not the shared Metrics |
| X02 | YES | janitor iterates and deletes s.links concurrently with HTTP handlers;  | review-bugs: AddResolved locks a value receiver, not the shared Metrics; review-concurrency: Store has no synchronization; concurrent requests race on th |
| X03 | YES | wg.Add inside the goroutine races wg.Wait; WarmCache can return before | review-bugs: WarmCache launches goroutine before wg.Add and ignores HEAD ; review-concurrency: WarmCache spawns goroutines that do wg.Add(1) inside the gor |
| X04 | YES | CheckTargets returns on first error; remaining sends to the unbuffered | review-bugs: CheckTargets can deadlock on bad request construction; review-bugs: CheckTargets leaks goroutines after the first error |

Unmatched findings (noise, or genuinely new — skim before dismissing):
- (high) main.go:97 — preview calls strings.ToValidUTF8 after Fprintf is already done
- (low) store.go:125 — validateCode linear-scans a 64-char string per rune
- (low) store.go:137 — newCode emits 8-char codes from 6 random bytes
