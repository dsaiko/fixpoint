package forge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

// A review is about one commit, and only publishing may decide whether that is
// still the commit the pull request proposes. Every answer other than "yes"
// refuses: an approval attached to a head nobody read is the whole risk, and an
// author who pushes while the panel runs must not be able to collect one.
func TestPostingRequiresTheReviewedHead(t *testing.T) {
	const reviewed = "0123456789abcdef0123456789abcdef01234567"
	for _, tc := range []struct {
		name             string
		reviewed, actual string
		wantErr          string
	}{
		{"the head is still the reviewed commit", reviewed, reviewed, ""},
		{"oid case does not decide it", reviewed, strings.ToUpper(reviewed), ""},
		{"the author pushed after the review", reviewed, "fedcba9876543210fedcba9876543210fedcba98", "has moved since it was reviewed"},
		{"the run never recorded what it reviewed", "", reviewed, "did not record which commit it reviewed"},
		{"the forge names no head", reviewed, "", "reported no head commit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := requireHead(7, tc.reviewed, tc.actual)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("requireHead() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("requireHead() = nil, want a refusal mentioning %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("refusal does not explain itself (%q): %v", tc.wantErr, err)
			}
		})
	}
}

// stubGH puts a fake `gh` on PATH. It answers the head read, records the review
// payload it is given on stdin, and fails any other call so an unexpected
// invocation shows up as an error rather than as a silent success.
func stubGH(t *testing.T, head string) (dir string, payload func() string) {
	t.Helper()
	bin := t.TempDir()
	capture := filepath.Join(bin, "payload.json")
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"*'--json headRefOid'*) printf '{\"headRefOid\":\"" + head + "\"}' ;;\n" +
		"*'--json url'*) printf '{\"url\":\"https://example.test/pr/7\"}' ;;\n" +
		"*'api --method POST'*) cat > " + capture + " ;;\n" +
		"*'pr review'*) printf 'body-only review submitted\\n' > " + capture + " ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	// Prepended, not replaced: the stub still needs the shell's own utilities, and
	// coming first is what makes it shadow any real gh on this host.
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return t.TempDir(), func() string {
		raw, err := os.ReadFile(capture)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

// The GitHub submission carries commit_id, so the forge records the review against
// the commit that was reviewed rather than against whatever the branch points at
// when it lands -- which also closes the gap between the head read and the post.
func TestGitHubReviewIsSubmittedAgainstTheReviewedCommit(t *testing.T) {
	const reviewed = "0123456789abcdef0123456789abcdef01234567"
	dir, payload := stubGH(t, reviewed)

	if _, err := (githubProvider{}).PostReview(t.Context(), dir, 7, reviewed, "the review",
		EventApprove, []InlineComment{{Path: "a.go", Line: 1, Body: "here"}}); err != nil {
		t.Fatalf("PostReview() = %v", err)
	}
	var got struct {
		CommitID string `json:"commit_id"`
		Event    string `json:"event"`
	}
	if err := json.Unmarshal([]byte(payload()), &got); err != nil {
		t.Fatalf("the submitted payload does not parse (%v): %s", err, payload())
	}
	if got.CommitID != reviewed {
		t.Errorf("commit_id = %q, want the reviewed head %q -- an approval must name the commit it approves", got.CommitID, reviewed)
	}
	if got.Event != "APPROVE" {
		t.Errorf("event = %q, want APPROVE", got.Event)
	}
}

// And nothing is submitted at all once the pull request has moved: not the review
// with its anchors, and not the body-only fallback that would otherwise approve the
// new head with no anchors to make the mismatch visible.
func TestGitHubPostRefusesAfterTheHeadMoved(t *testing.T) {
	dir, payload := stubGH(t, "fedcba9876543210fedcba9876543210fedcba98")

	_, err := (githubProvider{}).PostReview(t.Context(), dir, 7, "0123456789abcdef0123456789abcdef01234567",
		"the review", EventApprove, []InlineComment{{Path: "a.go", Line: 1, Body: "here"}})
	if err == nil {
		t.Fatal("PostReview() = nil; an approval was published against a commit that was never reviewed")
	}
	if !strings.Contains(err.Error(), "has moved since it was reviewed") {
		t.Errorf("refusal does not explain itself: %v", err)
	}
	if got := payload(); got != "" {
		t.Errorf("something was submitted anyway: %s", got)
	}
	// The retry in both callers keys on this classification, so a refusal must not
	// carry it: a stale head is not an anchor problem, and re-posting body-only is
	// exactly what must not happen.
	if AnchorRejection(err) {
		t.Errorf("a stale-head refusal must not look like an anchor rejection: %v", err)
	}
}

// stubGHRefusingInline answers the head read, then fails the inline submission
// with the given stderr line, the way gh reports whatever the API answered. The
// returned func reports what -- if anything -- the body-only path submitted.
func stubGHRefusingInline(t *testing.T, head, stderr string) (dir string, submitted func() string) {
	t.Helper()
	bin := t.TempDir()
	capture := filepath.Join(bin, "payload.json")
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"*'--json headRefOid'*) printf '{\"headRefOid\":\"" + head + "\"}' ;;\n" +
		"*'--json url'*) printf '{\"url\":\"https://example.test/pr/7\"}' ;;\n" +
		"*'api --method POST'*) echo \"" + stderr + "\" >&2; exit 1 ;;\n" +
		"*'pr review'*) printf 'body-only review submitted\\n' > " + capture + " ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return t.TempDir(), func() string {
		raw, err := os.ReadFile(capture)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

// Only a validation failure is an anchor rejection. GitHub checks every comment's
// path and line before it creates anything, so 422 means no review exists and the
// callers may safely send the same one body-only. Nothing else licenses that: an
// expired token, a 5xx, or the client-side timeout firing on a request the API had
// already accepted may all leave a review ON the pull request, and a retry then
// posts a second one under the operator's identity.
func TestOnlyAValidationFailureIsAnAnchorRejection(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	for _, tc := range []struct {
		name   string
		stderr string
		want   bool
	}{
		{"a comment outside the diff", "gh: Validation Failed (HTTP 422)", true},
		{"expired credentials", "gh: Bad credentials (HTTP 401)", false},
		{"the forge is unwell", "gh: Server Error (HTTP 502)", false},
		{"the request never completed", "error connecting to api.github.com", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, submitted := stubGHRefusingInline(t, head, tc.stderr)

			_, err := (githubProvider{}).PostReview(t.Context(), dir, 7, head, "the review",
				EventApprove, []InlineComment{{Path: "a.go", Line: 9, Body: "here"}})
			if err == nil {
				t.Fatal("PostReview() = nil although the submission failed")
			}
			if got := AnchorRejection(err); got != tc.want {
				t.Errorf("AnchorRejection(%v) = %v, want %v -- the callers re-post body-only on true, so classifying %q that way duplicates the review",
					err, got, tc.want, tc.stderr)
			}
			// Whatever the classification, PostReview must not publish the body-only
			// review itself: that is the caller's decision, taken after it has been
			// told which kind of failure this was.
			if got := submitted(); got != "" {
				t.Errorf("a body-only review was submitted from inside PostReview: %s", got)
			}
		})
	}
}
