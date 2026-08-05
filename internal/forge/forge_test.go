package forge

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The HOST decides, not a substring of the URL. A GitLab instance hosting a
// repository named "github-mirror" must not be driven with `gh`, and the path is
// attacker-adjacent in a way the host is not.
func TestDetectKindMatchesOnTheHost(t *testing.T) {
	for _, tc := range []struct {
		remote string
		want   Kind
	}{
		{"git@github.com:dsaiko/fixpoint.git", GitHub},
		{"https://github.com/dsaiko/fixpoint.git", GitHub},
		{"ssh://git@github.com:22/dsaiko/fixpoint.git", GitHub},
		{"https://user:tok@github.com/o/r.git", GitHub},
		{"git@gitlab.com:group/proj.git", GitLab},
		{"https://gitlab.example.com/group/proj.git", GitLab},
		{"git@git.company.io:team/github-mirror.git", Unknown},
		{"https://gitlab.example.com/team/github-mirror.git", GitLab},
		{"https://bitbucket.org/team/proj.git", Unknown},
		{"/srv/git/bare.git", Unknown},
		{"", Unknown},
	} {
		t.Run(tc.remote, func(t *testing.T) {
			if got := DetectKind(tc.remote); got != tc.want {
				t.Errorf("DetectKind(%q) = %q, want %q", tc.remote, got, tc.want)
			}
		})
	}
}

// Both shapes come back in one rollup: a modern Actions run is a CheckRun with
// name/status/conclusion, a classic commit status is a StatusContext with
// context/state. Parsing only the first would read a repository still on commit
// statuses as having no checks -- indistinguishable from a green one.
func TestGitHubChecksReadsBothRollupShapes(t *testing.T) {
	raw := `[
	  {"__typename":"CheckRun","name":"Build","status":"COMPLETED","conclusion":"SUCCESS"},
	  {"__typename":"CheckRun","name":"Unit Tests","status":"COMPLETED","conclusion":"FAILURE"},
	  {"__typename":"CheckRun","name":"Slow","status":"IN_PROGRESS","conclusion":""},
	  {"__typename":"CheckRun","name":"Redundant","status":"COMPLETED","conclusion":"` + ghCancelled + `"},
	  {"__typename":"CheckRun","name":"Optional","status":"COMPLETED","conclusion":"SKIPPED"},
	  {"__typename":"StatusContext","context":"ci/legacy","state":"ERROR"},
	  {"__typename":"StatusContext","context":"ci/queued","state":"PENDING"},
	  {"__typename":"StatusContext","context":"ci/green","state":"SUCCESS"}
	]`
	var rollup []ghCheck
	if err := json.Unmarshal([]byte(raw), &rollup); err != nil {
		t.Fatal(err)
	}
	var failing, pending []string
	for _, c := range rollup {
		switch {
		case failingGitHub(c):
			failing = append(failing, c.label())
		case pendingGitHub(c):
			pending = append(pending, c.label())
		}
	}
	if want := []string{"Unit Tests", "ci/legacy"}; !reflect.DeepEqual(failing, want) {
		t.Errorf("failing = %v, want %v", failing, want)
	}
	if want := []string{"Slow", "ci/queued"}; !reflect.DeepEqual(pending, want) {
		t.Errorf("pending = %v, want %v", pending, want)
	}
}

// A stopped or skipped check is not a pass, but it is not evidence of a defect
// either: blocking on one would block on a maintainer stopping a run they did not
// need.
func TestCancelledAndSkippedChecksDoNotBlock(t *testing.T) {
	for _, conclusion := range []string{ghCancelled, ghSkipped, ghNeutral, "SUCCESS"} {
		c := ghCheck{Name: "x", Status: "COMPLETED", Conclusion: conclusion}
		if failingGitHub(c) {
			t.Errorf("conclusion %q was treated as a failure", conclusion)
		}
	}
	for _, conclusion := range []string{ghFailure, ghTimedOut, ghActionRequired, ghStartupFailure} {
		if !failingGitHub(ghCheck{Name: "x", Status: "COMPLETED", Conclusion: conclusion}) {
			t.Errorf("conclusion %q should be a failure", conclusion)
		}
	}
}

// A check with neither a name nor a context still has to appear in a verdict's
// reasons; an empty string there would render as a stray comma.
func TestUnnamedCheckStillGetsALabel(t *testing.T) {
	if got := (ghCheck{}).label(); got == "" {
		t.Error("label() = empty; an unnamed failing check would vanish from the reasons")
	}
}
