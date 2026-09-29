package forge

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A forge CLI that leaves a child behind holding its stdout must not hold the
// read with it. `gh` forks credential helpers and git subprocesses routinely, and
// one of those can outlive the leader: with a pipe os/exec owned, cmd.Wait would
// then wait for an EOF that only the descendant can send, and the reads on the
// critical path of a run -- the checks rollup, the head read before a post --
// would hang past cliTimeout and past a Ctrl-C, since cancellation kills the
// leader alone. The helpers own their pipes and kill the process group instead,
// so the call returns on the leader's exit.
func TestALeftBehindChildDoesNotHoldTheRead(t *testing.T) {
	bin := t.TempDir()
	// Exits at once, having backgrounded a child that inherited stdout and will hold
	// its write end far longer than any bound this test is willing to wait for.
	script := "#!/bin/sh\nsleep 120 &\nprintf 'answered'\n"
	if err := os.WriteFile(filepath.Join(bin, "lingerer"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	type result struct {
		out string
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := run(t.Context(), t.TempDir(), "lingerer")
		done <- result{out, err}
	}()
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("run() = %v", got.err)
		}
		if got.out != "answered" {
			t.Errorf("stdout = %q, want the leader's own output %q", got.out, "answered")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("run() is still blocked long after the CLI itself exited -- a descendant holding the stdout pipe is outlasting the call, so cliTimeout does not bound it")
	}
}

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

// reviewedSHA is the commit the Checks tests pretend the run reviewed, and
// stubbedPR the pull request the stub below answers about. A test that asks about
// any other number is asking a gh that cannot answer, which is the point of one.
const (
	reviewedSHA = "0123456789abcdef0123456789abcdef01234567"
	stubbedPR   = 7
)

// stubGHRollup puts a fake `gh` on PATH that answers the rollup read -- and only
// that read, matched on the WHOLE argument list, so a change to the command Checks
// runs fails this test instead of quietly returning an unknown rollup. Any other
// invocation exits nonzero, which is what Checks sees when gh cannot answer.
//
// It answers with head and rollup TOGETHER because Checks asks for them together:
// the binding is only sound if one call returns both, so a split back into two
// reads shows up here as an unexpected invocation.
func stubGHRollup(t *testing.T, head, rollup string) (dir string) {
	t.Helper()
	bin := t.TempDir()
	stdout := `{"headRefOid":"` + head + `","statusCheckRollup":` + rollup + `}`
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"'pr view " + strconv.Itoa(stubbedPR) + " --json headRefOid,statusCheckRollup') printf '%s' '" + stdout + "' ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return t.TempDir()
}

// Both shapes come back in one rollup: a modern Actions run is a CheckRun with
// name/status/conclusion, a classic commit status is a StatusContext with
// context/state. Parsing only the first would read a repository still on commit
// statuses as having no checks -- indistinguishable from a green one. Driven
// through Checks, because everything between the command and Checks.Failing fails
// in the same silent direction: an unread rollup is Known false, which never
// blocks, so red CI would stop reaching the verdict gate.
func TestGitHubChecksReadsBothRollupShapes(t *testing.T) {
	dir := stubGHRollup(t, reviewedSHA, `[
	  {"__typename":"CheckRun","name":"Build","status":"COMPLETED","conclusion":"SUCCESS"},
	  {"__typename":"CheckRun","name":"Unit Tests","status":"COMPLETED","conclusion":"FAILURE"},
	  {"__typename":"CheckRun","name":"Slow","status":"IN_PROGRESS","conclusion":""},
	  {"__typename":"CheckRun","name":"Redundant","status":"COMPLETED","conclusion":"`+ghCancelled+`"},
	  {"__typename":"CheckRun","name":"Optional","status":"COMPLETED","conclusion":"SKIPPED"},
	  {"__typename":"StatusContext","context":"ci/legacy","state":"ERROR"},
	  {"__typename":"StatusContext","context":"ci/queued","state":"PENDING"},
	  {"__typename":"StatusContext","context":"ci/green","state":"SUCCESS"}
	]`)

	got, err := (githubProvider{}).Checks(t.Context(), dir, stubbedPR, reviewedSHA)
	if err != nil {
		t.Fatalf("Checks() = %v", err)
	}
	if !got.Known {
		t.Error("Known = false although gh answered -- the verdict would carry no CI evidence")
	}
	if want := []string{"Unit Tests", "ci/legacy"}; !reflect.DeepEqual(got.Failing, want) {
		t.Errorf("Failing = %v, want %v", got.Failing, want)
	}
	if want := []string{"Slow", "ci/queued"}; !reflect.DeepEqual(got.Pending, want) {
		t.Errorf("Pending = %v, want %v", got.Pending, want)
	}
}

// A pull request with no checks at all is an ANSWER: Known stays true, so the
// verdict says CI reported nothing rather than leaving it unknown. "No checks
// configured" and "we could not ask" are different facts and only one of them is
// worth a reason.
func TestGitHubChecksKnowsAnEmptyRollupIsStillAnAnswer(t *testing.T) {
	dir := stubGHRollup(t, reviewedSHA, `[]`)

	got, err := (githubProvider{}).Checks(t.Context(), dir, stubbedPR, reviewedSHA)
	if err != nil {
		t.Fatalf("Checks() = %v", err)
	}
	if !got.Known {
		t.Error("Known = false for a rollup with no entries -- indistinguishable from gh being unable to answer")
	}
	if len(got.Failing) != 0 || len(got.Pending) != 0 {
		t.Errorf("Failing/Pending = %v/%v, want empty", got.Failing, got.Pending)
	}
}

// And when gh cannot answer, the checks are UNKNOWN rather than empty -- empty
// reads as green. The stub answers for pull request 7 alone, so this also pins
// that the number Checks is asked about is the one it puts on the command line.
func TestGitHubChecksAreUnknownWhenGHCannotAnswer(t *testing.T) {
	dir := stubGHRollup(t, reviewedSHA, `[]`)

	got, err := (githubProvider{}).Checks(t.Context(), dir, 9, reviewedSHA)
	if err == nil {
		t.Fatal("Checks() = nil although gh failed")
	}
	if got.Known {
		t.Error("Known = true although the rollup was never read -- an approval would rest on checks nobody saw")
	}
}

// The rollup belongs to the pull request, so it describes whatever the request
// proposes when it is read. An author can force-push a green commit, let this read
// see it and push the reviewed one back: the panel and every human still read the
// commit with the failing CI, while the verdict would rest on the green one's
// checks. Evidence about another commit is worth less than none, so the read
// refuses and CI stays unknown.
func TestGitHubChecksRefuseAHeadThatIsNotTheReviewedCommit(t *testing.T) {
	const pushed = "fedcba9876543210fedcba9876543210fedcba98"
	dir := stubGHRollup(t, pushed, `[
	  {"__typename":"CheckRun","name":"Build","status":"COMPLETED","conclusion":"SUCCESS"}
	]`)

	got, err := (githubProvider{}).Checks(t.Context(), dir, stubbedPR, reviewedSHA)
	if err == nil {
		t.Fatal("Checks() = nil for a rollup belonging to another commit -- the verdict would credit the reviewed commit with a pushed commit's green CI")
	}
	if got.Known {
		t.Error("Known = true although no check was seen for the reviewed commit")
	}
	if !strings.Contains(err.Error(), "another commit") {
		t.Errorf("refusal does not say whose checks these were: %v", err)
	}
}

// And a run that never recorded its head has nothing to bind to. Reading the rollup
// anyway would mean accepting whatever the pull request proposes at that moment,
// which is the same hole with no evidence that it has been exploited.
func TestChecksRefuseWhenTheReviewedCommitWasNotRecorded(t *testing.T) {
	dir := stubGHRollup(t, reviewedSHA, `[]`)

	if _, err := (githubProvider{}).Checks(t.Context(), dir, stubbedPR, ""); err == nil {
		t.Error("Checks() = nil with no reviewed commit to bind to")
	}
	if _, err := (gitlabProvider{}).Checks(t.Context(), dir, stubbedPR, ""); err == nil {
		t.Error("gitlab Checks() = nil with no reviewed commit to bind to")
	}
}

// stubGlabPipelines puts a fake `glab` on PATH that answers the merge request's
// pipeline list -- matched on the WHOLE argument list, so a change to the command
// Checks runs fails this test instead of quietly reporting no pipeline. Any other
// invocation exits nonzero, which is what Checks sees when glab cannot answer.
func stubGlabPipelines(t *testing.T, pipelines string) (dir string) {
	t.Helper()
	bin := t.TempDir()
	// The list goes through a file rather than into the script: it is quote-heavy
	// JSON, and embedding it would test the shell escaping, not the parse.
	body := filepath.Join(bin, "pipelines.json")
	if err := os.WriteFile(body, []byte(pipelines), 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"'api projects/:id/merge_requests/" + strconv.Itoa(stubbedPR) + "/pipelines') cat " + body + " ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "glab"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return t.TempDir()
}

// GitLab returns the pipeline list newest first, and the newest entry belongs to
// the newest PUSH. Taking it would be the GitHub hole in another shape: a later
// commit's green pipeline standing in for the reviewed commit's failing one. The
// reviewed commit's own pipeline is the evidence, however old it is in the list.
func TestGitLabChecksReadThePipelineOfTheReviewedCommit(t *testing.T) {
	const pushed = "fedcba9876543210fedcba9876543210fedcba98"
	dir := stubGlabPipelines(t, `[
	  {"id":22,"sha":"`+pushed+`","status":"success"},
	  {"id":11,"sha":"`+reviewedSHA+`","status":"failed"}
	]`)

	got, err := (gitlabProvider{}).Checks(t.Context(), dir, stubbedPR, reviewedSHA)
	if err != nil {
		t.Fatalf("Checks() = %v", err)
	}
	if !got.Known {
		t.Error("Known = false although glab answered -- the verdict would carry no CI evidence")
	}
	if want := []string{"pipeline 11"}; !reflect.DeepEqual(got.Failing, want) {
		t.Errorf("Failing = %v, want %v -- pipeline 22 ran on a later push, so reporting it credits the reviewed commit with another commit's green CI", got.Failing, want)
	}
	if len(got.Pending) != 0 {
		t.Errorf("Pending = %v, want empty", got.Pending)
	}
}

// And when every pipeline is about some other commit there is no evidence to
// report. An empty pipeline list means "no checks" (Known stays true); this means
// "we cannot say", which is a refusal, so a verdict never rests on it.
func TestGitLabChecksRefusePipelinesThatBelongToOtherCommits(t *testing.T) {
	const pushed = "fedcba9876543210fedcba9876543210fedcba98"
	dir := stubGlabPipelines(t, `[{"id":22,"sha":"`+pushed+`","status":"success"}]`)

	got, err := (gitlabProvider{}).Checks(t.Context(), dir, stubbedPR, reviewedSHA)
	if err == nil {
		t.Fatal("Checks() = nil although no pipeline ran on the reviewed commit -- the verdict would credit it with a pushed commit's green CI")
	}
	if got.Known {
		t.Error("Known = true although no pipeline was seen for the reviewed commit")
	}
	if !strings.Contains(err.Error(), shortSHA(reviewedSHA)) {
		t.Errorf("the refusal does not name the commit it wanted a pipeline for: %v", err)
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

// A review with no anchors is bound to the reviewed commit just the same. It is the
// common case -- most findings never reach an inline comment -- and it is where the
// anchor-rejection fallback lands, so `gh pr review`, which cannot name a commit,
// must not be how it is published: that would approve whatever the branch points at
// when the submission arrives.
func TestABodyOnlyReviewIsAlsoBoundToTheReviewedCommit(t *testing.T) {
	const reviewed = "0123456789abcdef0123456789abcdef01234567"
	dir, payload := stubGH(t, reviewed)

	if _, err := (githubProvider{}).PostReview(t.Context(), dir, 7, reviewed, "the review",
		EventApprove, nil); err != nil {
		t.Fatalf("PostReview() = %v", err)
	}
	raw := payload()
	if raw == "body-only review submitted\n" {
		t.Fatal("the review went out through `gh pr review`, which carries no commit_id")
	}
	var got struct {
		CommitID string `json:"commit_id"`
		Event    string `json:"event"`
		Body     string `json:"body"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("the submitted payload does not parse (%v): %s", err, raw)
	}
	if got.CommitID != reviewed {
		t.Errorf("commit_id = %q, want the reviewed head %q -- an approval must name the commit it approves", got.CommitID, reviewed)
	}
	if want := githubEvent(EventApprove); got.Event != want || got.Body != "the review" {
		t.Errorf("event/body = %q/%q, want %q/%q", got.Event, got.Body, want, "the review")
	}
	// Present-but-empty would be a claim about lines that this review does not make.
	if strings.Contains(raw, "\"comments\"") {
		t.Errorf("a review with no anchors still sent a comments list: %s", raw)
	}
}

// A 422 on a submission that carried no anchors is not an anchor rejection: there is
// nothing to drop, and re-posting body-only would fail the same way. Classifying it
// as one is how a caller's fallback turns one failure into two submissions.
func TestAValidationFailureWithoutAnchorsIsNotAnAnchorRejection(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	dir, _ := stubGHRefusingInline(t, head, "gh: Validation Failed (HTTP 422)")

	_, err := (githubProvider{}).PostReview(t.Context(), dir, 7, head, "the review", EventApprove, nil)
	if err == nil {
		t.Fatal("PostReview() = nil although the submission failed")
	}
	if AnchorRejection(err) {
		t.Errorf("a body-only failure was reported as an anchor rejection: %v", err)
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

// stubGHMovingHead answers the FIRST head read with before and every later one with
// after, which is the author pushing into the window between requireHead's read and
// the submission. It records the submitted payload like stubGH, so a test can tell
// "refused before publishing" apart from "published, then noticed".
func stubGHMovingHead(t *testing.T, before, after string) (dir string, payload func() string) {
	t.Helper()
	bin := t.TempDir()
	capture := filepath.Join(bin, "payload.json")
	seen := filepath.Join(bin, "head-read")
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"*'--json headRefOid'*) if [ -f " + seen + " ]; then printf '{\"headRefOid\":\"" + after + "\"}'; " +
		"else : > " + seen + "; printf '{\"headRefOid\":\"" + before + "\"}'; fi ;;\n" +
		"*'--json url'*) printf '{\"url\":\"https://example.test/pr/7\"}' ;;\n" +
		"*'api --method POST'*) cat > " + capture + " ;;\n" +
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

// requireHead reads the head and the submission is a separate round trip, so a push
// landing in between passes the check on a stale snapshot. commit_id keeps the review
// itself about the reviewed commit, but GitHub accepts an approval for a commit that
// is no longer the head, and that approval counts toward the pull request unless the
// repository dismisses stale reviews.
//
// Reporting it is not enough -- an approval nobody can see the report of still
// satisfies branch protection over unreviewed code -- so it is WITHDRAWN. This case
// is the one where withdrawal is impossible: the stub's submission answers with
// nothing, so no review id was ever learned and there is no handle to dismiss. The
// operator has to be told to do it by hand rather than left believing it was
// handled.
func TestAnApprovalThatLandedOnAMovedHeadIsNotReportedAsSuccess(t *testing.T) {
	const reviewed = "0123456789abcdef0123456789abcdef01234567"
	dir, payload := stubGHMovingHead(t, reviewed, "fedcba9876543210fedcba9876543210fedcba98")

	url, err := (githubProvider{}).PostReview(t.Context(), dir, 7, reviewed, "the review", EventApprove, nil)
	if err == nil {
		t.Fatal("PostReview() = nil; an approval on a head nobody reviewed was reported as a clean post")
	}
	if payload() == "" {
		t.Fatal("nothing was submitted -- this must exercise the post-submission check, not requireHead")
	}
	for _, want := range []string{"PUBLISHED", "withdrawing it failed", "by hand"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the report does not tell the operator what happened (%q): %v", want, err)
		}
	}
	// The review is on the pull request, so the caller must still be able to name it.
	if url == "" {
		t.Error("no URL was returned for a review that was published")
	}
	// A second submission here would put two reviews on somebody's pull request.
	if AnchorRejection(err) {
		t.Errorf("a published-then-moved approval must not look like an anchor rejection: %v", err)
	}
}

// stubGHDismissingReview moves the head between the two reads like stubGHMovingHead,
// but its submission answers the way the reviews API really does -- with the created
// review's id and permalink -- so the dismissal has a handle to work with. It records
// the argument list of any PUT, which is where the withdrawal goes.
func stubGHDismissingReview(t *testing.T, before, after string) (dir string, dismissal func() string) {
	t.Helper()
	bin := t.TempDir()
	seen := filepath.Join(bin, "head-read")
	put := filepath.Join(bin, "put.argv")
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"*'--json headRefOid'*) if [ -f " + seen + " ]; then printf '{\"headRefOid\":\"" + after + "\"}'; " +
		"else : > " + seen + "; printf '{\"headRefOid\":\"" + before + "\"}'; fi ;;\n" +
		"*'--json url'*) printf '{\"url\":\"https://example.test/pr/7\"}' ;;\n" +
		"*'api --method POST'*) cat > /dev/null; " +
		"printf '{\"id\":2938471,\"html_url\":\"https://example.test/pr/7#pullrequestreview-2938471\"}' ;;\n" +
		"*'--method PUT'*) printf '%s' \"$*\" > " + put + " ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return t.TempDir(), func() string {
		raw, err := os.ReadFile(put)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

// And when the id IS known -- which is the normal case, since the create response
// carries it -- the approval is actually taken off the pull request. Nothing else
// removes it: the report alone leaves a branch-protection approval standing over a
// commit no reviewer read. The id has to come from the submission's own response;
// the pull request URL that `gh pr view` returns carries no review id at all, so a
// dismissal derived from it could never fire.
func TestAnApprovalOnAMovedHeadIsWithdrawn(t *testing.T) {
	const reviewed = "0123456789abcdef0123456789abcdef01234567"
	dir, dismissal := stubGHDismissingReview(t, reviewed, "fedcba9876543210fedcba9876543210fedcba98")

	url, err := (githubProvider{}).PostReview(t.Context(), dir, 7, reviewed, "the review", EventApprove, nil)
	if err == nil {
		t.Fatal("PostReview() = nil; an approval on a head nobody reviewed was reported as a clean post")
	}
	got := dismissal()
	if got == "" {
		t.Fatal("no dismissal was attempted -- the approval is still on the pull request")
	}
	if !strings.Contains(got, "pulls/7/reviews/2938471/dismissals") {
		t.Errorf("the dismissal did not name the review that was just posted: %s", got)
	}
	if !strings.Contains(got, "event=DISMISS") {
		t.Errorf("the dismissal did not ask for a dismissal: %s", got)
	}
	// The operator still has to know: the approval was visible for as long as it took.
	for _, want := range []string{"the head moved", "withdrawn"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the report does not say what happened (%q): %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "by hand") {
		t.Errorf("the operator was sent to do by hand what was already done: %v", err)
	}
	// The permalink the submission answered with, not the pull request's own URL:
	// it is what names the review that was published.
	if want := "https://example.test/pr/7#pullrequestreview-2938471"; url != want {
		t.Errorf("URL = %q, want the review permalink %q", url, want)
	}
}

// The withdrawal is the compensating control for an approval that already landed,
// so it cannot be disabled by the run being interrupted. A Ctrl-C arriving in the
// window between the submission returning and the confirmation that follows it
// cancels the run's context, and on that context the head read and the dismissal
// would both fail instantly -- leaving the approval standing over unread code
// exactly when nobody is watching for the report. So the confirmation runs
// detached: cancellation stops the submission, never the repair of one that landed.
func TestAnApprovalOnAMovedHeadIsWithdrawnEvenWhenTheRunWasInterrupted(t *testing.T) {
	const reviewed = "0123456789abcdef0123456789abcdef01234567"
	const moved = "fedcba9876543210fedcba9876543210fedcba98"
	// Both reads answer the moved head: the submission has already happened here.
	dir, dismissal := stubGHDismissingReview(t, moved, moved)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := confirmApproval(ctx, dir, 7, reviewed, EventApprove, "2938471",
		"https://example.test/pr/7#pullrequestreview-2938471")
	if err == nil {
		t.Fatal("confirmApproval() = nil; an approval on a head nobody reviewed was reported as a clean post")
	}
	got := dismissal()
	if got == "" {
		t.Fatal("no dismissal was attempted after an interrupt -- the approval is still on the pull request")
	}
	if !strings.Contains(got, "pulls/7/reviews/2938471/dismissals") {
		t.Errorf("the dismissal did not name the review that was just posted: %s", got)
	}
	if !strings.Contains(err.Error(), "withdrawn") {
		t.Errorf("the report does not say the approval was taken off: %v", err)
	}
	if strings.Contains(err.Error(), "by hand") {
		t.Errorf("the operator was sent to do by hand what was already done: %v", err)
	}
}

// The submission has to be detached too, not only the confirmation that follows it. An
// interrupt landing while the APPROVE POST is in flight kills gh after GitHub has
// already created the review, so the submission reports a failure and carries back no
// review id -- and an approval nobody can name is an approval nobody can dismiss. So
// once the pre-submit head check has passed, cancellation stops the approval from
// going out at all or it does not touch it: never halfway.
func TestAnApprovalIsSubmittedOnAContextAnInterruptCannotKill(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	submit, err := approvalSubmitContext(ctx, 7, EventApprove)
	if err != nil {
		t.Fatalf("approvalSubmitContext() = %v on a live run", err)
	}
	// The interrupt arrives in the window between the head check and the POST.
	cancel()
	if err := submit.Err(); err != nil {
		t.Errorf("the approval submission was canceled mid-flight (%v) -- it can land on the forge with no id to withdraw it by", err)
	}

	// A comment grants nothing, so there is no repair to keep alive and Ctrl-C stays
	// as responsive as it was.
	plain, err := approvalSubmitContext(ctx, 7, Comment)
	if err != nil {
		t.Fatalf("approvalSubmitContext() = %v for a comment", err)
	}
	if plain.Err() == nil {
		t.Error("a comment review outlives cancellation, spending responsiveness on a post that grants nothing")
	}
}

// Before the POST, though, cancellation means what it says: detaching the submission
// must not turn an interrupt that arrived earlier into an approval published on a run
// the operator already stopped.
func TestAnAlreadyInterruptedRunSubmitsNoApproval(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := approvalSubmitContext(ctx, 7, EventApprove); err == nil {
		t.Fatal("approvalSubmitContext() = nil on an interrupted run -- the approval would be published anyway")
	} else if !errors.Is(err, context.Canceled) {
		t.Errorf("the refusal does not carry the cancellation: %v", err)
	}
}

// Only the approval is alarmed on. A comment review that lands on a moved head is
// stale, not dangerous, and turning that into a nonzero exit trains operators to
// ignore the one message that matters.
func TestACommentReviewIsNotAlarmedOnWhenTheHeadMoves(t *testing.T) {
	const reviewed = "0123456789abcdef0123456789abcdef01234567"
	dir, _ := stubGHMovingHead(t, reviewed, "fedcba9876543210fedcba9876543210fedcba98")

	if _, err := (githubProvider{}).PostReview(t.Context(), dir, 7, reviewed, "the review", Comment, nil); err != nil {
		t.Errorf("PostReview() = %v, want nil -- a comment grants nothing", err)
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

// stubGlab puts a fake `glab` on PATH. It answers the head read, records every
// invocation's argument list and whatever arrived on its standard input, and fails
// anything else so an unexpected call is an error rather than a silent success.
func stubGlab(t *testing.T, head string) (dir string, argv, stdin func() string) {
	t.Helper()
	bin := t.TempDir()
	args := filepath.Join(bin, "argv.txt")
	body := filepath.Join(bin, "stdin.txt")
	script := "#!/bin/sh\necho \"$*\" >> " + args + "\ncase \"$*\" in\n" +
		"*notes*|*approve*) cat >> " + body + " ;;\n" +
		"*merge_requests/7) printf '{\"diff_refs\":{\"head_sha\":\"" + head + "\"}}' ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "glab"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	read := func(path string) func() string {
		return func() string {
			raw, err := os.ReadFile(path)
			if err != nil {
				return ""
			}
			return string(raw)
		}
	}
	return t.TempDir(), read(args), read(body)
}

// The review never reaches an argument list, on GitLab as on GitHub. argv is
// world-readable for the life of the process and the review quotes the code it is
// about -- which in a review-only run may be the credential the review is ABOUT,
// and redaction is shape-based, so it cannot be assumed to have masked it.
func TestGitLabNotesKeepTheReviewOffTheCommandLine(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	const secret = "sk-ant-api03-nothing-real-here"
	dir, argv, stdin := stubGlab(t, head)

	if _, err := (gitlabProvider{}).PostReview(t.Context(), dir, 7, head, "the review quotes "+secret,
		EventApprove, []InlineComment{{Path: "a.go", Line: 9, Body: "so does this comment: " + secret}}); err != nil {
		t.Fatalf("PostReview() = %v", err)
	}
	if got := argv(); strings.Contains(got, secret) {
		t.Errorf("the note text reached the argument list: %s", got)
	}
	// And it did arrive: both notes, as JSON bodies on stdin.
	got := stdin()
	if n := strings.Count(got, secret); n != 2 {
		t.Errorf("stdin carried the text %d time(s), want 2 (the inline note and the body): %s", n, got)
	}
	if !strings.Contains(got, `"body"`) {
		t.Errorf("the note was not posted as an API payload: %s", got)
	}
}

// The inline note names its location, and that location is agent-authored: unlike
// the body and the comment text, it never went through SanitizeText when the review
// was rendered. A backtick in the path would close the code span this line wraps it
// in and leave the rest of the path rendering as live markdown -- a mention, in a
// note posted under the operator's identity.
func TestGitLabInlineNotePathCannotEscapeItsCodeSpan(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	dir, _, stdin := stubGlab(t, head)

	if _, err := (gitlabProvider{}).PostReview(t.Context(), dir, 7, head, "the review", Comment,
		[]InlineComment{{Path: "a`@victim **b**.go", Line: 9, Body: "the finding"}}); err != nil {
		t.Fatalf("PostReview() = %v", err)
	}
	// Decoded, because json.Marshal escapes the `<` of the mention break: what
	// matters is the note GitLab renders, not its wire encoding.
	var note struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(strings.NewReader(stdin())).Decode(&note); err != nil {
		t.Fatalf("decoding the first note: %v", err)
	}
	if strings.Contains(note.Body, "a`@victim") {
		t.Errorf("the path closed its code span, so what follows renders as markdown: %s", note.Body)
	}
	if !strings.Contains(note.Body, "`a&#96;@<!---->victim **b**.go:9`") {
		t.Errorf("the path was not escaped and sanitized in place: %s", note.Body)
	}
}

// An approval must name the commit it is about. requireHead runs before the notes are
// posted, so it alone would leave a window: an author who pushes after the head is
// read collects an approval for code no reviewer saw. sha on the approve endpoint is
// what closes it -- GitLab answers 409 when it no longer matches the source branch.
func TestGitLabApprovalIsBoundToTheReviewedCommit(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	dir, argv, stdin := stubGlab(t, head)

	if _, err := (gitlabProvider{}).PostReview(t.Context(), dir, 7, head, "the review", EventApprove, nil); err != nil {
		t.Fatalf("PostReview() = %v", err)
	}
	if got := argv(); !strings.Contains(got, "merge_requests/7/approve") {
		t.Errorf("the approval did not go through the endpoint that accepts a commit: %s", got)
	}
	if got := stdin(); !strings.Contains(got, `"sha":"`+head+`"`) {
		t.Errorf("the approval did not carry the reviewed commit, so it lands on whatever the head is when it arrives: %s", got)
	}
}

// stubGHThreads puts a fake `gh` on PATH that answers the repository-identity read
// and the reviewThreads GraphQL query with response, recording the argv of the query
// and the argv and stdin of a reply POST. Anything else exits nonzero, so a change to
// the commands Threads or Reply run surfaces as an error rather than as a pull
// request that appears to have no conversations.
func stubGHThreads(t *testing.T, response string) (dir string, query, replyArgv, replyBody func() string) {
	t.Helper()
	bin := t.TempDir()
	// The response goes through a file rather than into the script: a GraphQL payload
	// is full of quotes, and embedding it would test the escaping, not the parse.
	body := filepath.Join(bin, "response.json")
	if err := os.WriteFile(body, []byte(response), 0o600); err != nil {
		t.Fatal(err)
	}
	queryFile, argvFile, stdinFile := filepath.Join(bin, "query.txt"), filepath.Join(bin, "argv.txt"), filepath.Join(bin, "stdin.json")
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"'repo view --json owner,name') printf '%s' '{\"owner\":{\"login\":\"dsaiko\"},\"name\":\"fixpoint\"}' ;;\n" +
		"'api graphql'*) printf '%s' \"$*\" > " + queryFile + "; cat " + body + " ;;\n" +
		"'api --method POST'*) printf '%s' \"$*\" > " + argvFile + "; cat > " + stdinFile + " ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	read := func(path string) func() string {
		return func() string {
			raw, err := os.ReadFile(path)
			if err != nil {
				return ""
			}
			return string(raw)
		}
	}
	return t.TempDir(), read(queryFile), read(argvFile), read(stdinFile)
}

// One resolved thread -- a question a human already settled, and reopening it is
// worse than never answering. One open thread with the comment that started it. One
// open thread whose comments came back empty, which has no root comment to address a
// reply to. databaseId is deliberately past int32 so the id keeps its width.
const reviewThreadsPayload = `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[
  {"isResolved":true,"comments":{"nodes":[{"path":"settled.go","line":3,"databaseId":11,"body":"already handled","author":{"login":"dsaiko"}}]}},
  {"isResolved":false,"comments":{"nodes":[{"path":"internal/forge/forge.go","line":42,"databaseId":2147483648,"body":"why origin only?","author":{"login":"dsaiko"}}]}},
  {"isResolved":false,"comments":{"nodes":[]}}
]}}}}}`

// The whole answer-the-humans feature rests on this parse, and it fails in the
// quietest possible direction: a renamed GraphQL field still unmarshals, yields no
// threads, and a fix run then proceeds exactly as if nobody had commented. So the
// shape is pinned end to end -- through Threads, against a payload in the API's own
// form, not against a hand-unmarshalled struct.
func TestThreadsAreTheUnresolvedConversationsThatStillHaveARoot(t *testing.T) {
	dir, query, _, _ := stubGHThreads(t, reviewThreadsPayload)

	got, err := (githubProvider{}).Threads(t.Context(), dir, 7)
	if err != nil {
		t.Fatalf("Threads() = %v", err)
	}
	want := []Thread{{
		ID:     "2147483648",
		Path:   "internal/forge/forge.go",
		Line:   42,
		Author: "dsaiko",
		Body:   "why origin only?",
		// The whole exchange, root included: reading only the root made an answered
		// conversation look exactly like an untouched one.
		Comments: []ThreadComment{{Author: "dsaiko", Body: "why origin only?"}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Threads() = %+v, want %+v -- resolved threads and threads with no root comment are not conversations to answer", got, want)
	}
	// And the query still asks for what the parse reads, addressed at this pull
	// request: a drift on either side reads as a quiet pull request.
	for _, field := range []string{"owner=dsaiko", "repo=fixpoint", "pr=7", "reviewThreads", "isResolved", "databaseId"} {
		if !strings.Contains(query(), field) {
			t.Errorf("the graphql call does not carry %q: %s", field, query())
		}
	}
	// The two String! variables go through gh's untyped -f. Under -F, a checkout of a
	// repo named 2048 or null becomes a JSON number or JSON null, the server rejects
	// the page against String!, and the pull request reads as having no conversations.
	for _, typed := range []string{"-f owner=", "-f repo="} {
		if !strings.Contains(query(), typed) {
			t.Errorf("the graphql call does not pass %q untyped: %s", typed, query())
		}
	}
}

// The same payload read the other way. A resolved conversation is a question a
// human settled -- and settling one is how a maintainer says handled, or won't
// fix, so the finding it carries is the one a later review must not repeat.
// AllThreads is the read that answers "what has this pull request already been
// told", and dropping the resolved threads from it made every closed finding come
// back as a brand-new comment. The thread with no root comment stays out of both:
// it has nothing to read.
func TestAllThreadsKeepsTheConversationsAHumanHasSettled(t *testing.T) {
	dir, _, _, _ := stubGHThreads(t, reviewThreadsPayload)

	got, err := (githubProvider{}).AllThreads(t.Context(), dir, 7)
	if err != nil {
		t.Fatalf("AllThreads() = %v", err)
	}
	want := []Thread{{
		ID:       "11",
		Path:     "settled.go",
		Line:     3,
		Author:   "dsaiko",
		Body:     "already handled",
		Comments: []ThreadComment{{Author: "dsaiko", Body: "already handled"}},
	}, {
		ID:       "2147483648",
		Path:     "internal/forge/forge.go",
		Line:     42,
		Author:   "dsaiko",
		Body:     "why origin only?",
		Comments: []ThreadComment{{Author: "dsaiko", Body: "why origin only?"}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AllThreads() = %+v, want %+v -- a resolved conversation is something this pull request has already been told", got, want)
	}
}

// The review SUMMARIES, which is where most findings live: an anchor has to fall
// inside the pull request's own diff, and most findings point at code the change
// did not touch. This parse fails the same quiet way the thread one does -- a
// renamed field still unmarshals, yields no reviews, and every body-only finding
// of every earlier review is then reported as new.
func TestReviewsAreTheSummariesAlreadySubmitted(t *testing.T) {
	payload := `{"data":{"repository":{"pullRequest":{"reviews":{"nodes":[
	  {"body":"## Changes requested\n\n<!-- ai-panel run 20260101-000000 finding deadbeefcafe -->","author":{"login":"dsaiko"}},
	  {"body":"looks good","author":{"login":"stranger"}}
	]}}}}}`
	dir, query, _, _ := stubGHThreads(t, payload)

	got, err := (githubProvider{}).Reviews(t.Context(), dir, 7)
	if err != nil {
		t.Fatalf("Reviews() = %v", err)
	}
	want := []Review{
		{Author: "dsaiko", Body: "## Changes requested\n\n<!-- ai-panel run 20260101-000000 finding deadbeefcafe -->"},
		{Author: "stranger", Body: "looks good"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Reviews() = %+v, want %+v", got, want)
	}
	// Everyone's reviews are read and the author filtering happens in Go, so the
	// body of a stranger's review must arrive here rather than be filtered away by
	// the server: what makes a review ours is the marker AND the account.
	for _, field := range []string{"owner=dsaiko", "repo=fixpoint", "pr=7", "reviews", "body", "login"} {
		if !strings.Contains(query(), field) {
			t.Errorf("the graphql call does not carry %q: %s", field, query())
		}
	}
}

// A conversation longer than one page of comments, whose last word is this tool's.
// The first page stops at hasNextPage, and the reply carrying the marker is only on
// the second -- which is the shape that made AnsweredByMachine read the wrong
// comment and answer the same thread on every run.
const longThreadFirstPage = `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[
  {"id":"PRRT_kwDOAbCdEf","isResolved":false,"comments":{
    "pageInfo":{"hasNextPage":true,"endCursor":"c100"},
    "nodes":[{"path":"internal/forge/forge.go","line":42,"databaseId":2147483648,"body":"why origin only?","author":{"login":"dsaiko"}}]}}
]}}}}}`

const longThreadTail = `{"data":{"node":{"comments":{
  "pageInfo":{"hasNextPage":false,"endCursor":"c200"},
  "nodes":[{"path":"internal/forge/forge.go","line":42,"databaseId":2147483649,"body":"answered <!-- ai-panel run 20260807 -->","author":{"login":"dsaiko"}}]}}}}`

// The comments of one thread page separately from the thread list, so a long
// conversation came back cut at its hundredth message. AnsweredByMachine reads the
// LAST comment: with the tail missing it never saw this tool's own reply, called the
// thread live, and answered it again every run -- exactly what the marker exists to
// stop. So the tail is followed, and the test asserts both halves arrive in order.
func TestAConversationIsReadPastItsFirstPageOfComments(t *testing.T) {
	bin := t.TempDir()
	first, tail := filepath.Join(bin, "first.json"), filepath.Join(bin, "tail.json")
	for path, body := range map[string]string{first: longThreadFirstPage, tail: longThreadTail} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// The tail query is the one that names the thread type, so the stub answers on
	// that rather than on call order: a Threads that never asks for the rest fails
	// here by returning a truncated conversation, not by running out of responses.
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"'repo view --json owner,name') printf '%s' '{\"owner\":{\"login\":\"dsaiko\"},\"name\":\"fixpoint\"}' ;;\n" +
		"*PullRequestReviewThread*) cat " + tail + " ;;\n" +
		"'api graphql'*) cat " + first + " ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := (githubProvider{}).Threads(t.Context(), t.TempDir(), 7)
	if err != nil {
		t.Fatalf("Threads() = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Threads() = %+v, want one conversation", got)
	}
	want := []ThreadComment{
		{Author: "dsaiko", Body: "why origin only?"},
		{Author: "dsaiko", Body: "answered <!-- ai-panel run 20260807 -->"},
	}
	if !reflect.DeepEqual(got[0].Comments, want) {
		t.Errorf("Comments = %+v, want %+v -- the second page was dropped", got[0].Comments, want)
	}
	// The root is still the comment that opened the thread: it is what a reply is
	// addressed to, and the tail must not displace it.
	if got[0].ID != "2147483648" || got[0].Body != "why origin only?" {
		t.Errorf("root = %q/%q, want the first comment of the first page", got[0].ID, got[0].Body)
	}
	if !got[0].AnsweredByMachine("dsaiko") {
		t.Error("a thread whose last comment is this tool's reply reads as unanswered, so it is answered again every run")
	}
}

// A thread list page that never clears hasNextPage: a malformed cursor or a server
// that keeps saying "more" walks the loop into its bound.
const endlessThreadListPage = `{"data":{"repository":{"pullRequest":{"reviewThreads":{
  "pageInfo":{"hasNextPage":true,"endCursor":"c100"},
  "nodes":[
  {"id":"PRRT_kwDOAbCdEf","isResolved":false,"comments":{
    "pageInfo":{"hasNextPage":false,"endCursor":""},
    "nodes":[{"path":"internal/forge/forge.go","line":42,"databaseId":2147483648,"body":"why origin only?","author":{"login":"dsaiko"}}]}}
]}}}}}`

// One whole thread, whose comments never stop paging: the same non-termination one
// level down, where githubThreadTail rather than Threads has to refuse.
const endlessThreadTail = `{"data":{"node":{"comments":{
  "pageInfo":{"hasNextPage":true,"endCursor":"c200"},
  "nodes":[{"path":"internal/forge/forge.go","line":42,"databaseId":2147483649,"body":"still talking","author":{"login":"dsaiko"}}]}}}}`

// Both pagination loops stop at maxPages, and stopping is an ERROR rather than the
// pages read so far. Handing back what was accumulated would be the defect the
// pagination was added for, wearing a nil error: triage and the coder get half an
// exchange as if it were the whole one, and AnsweredByMachine -- which reads the LAST
// comment -- calls an answered thread live and answers it again under the operator's
// name. So what is asserted is that nothing comes back with the error.
func TestAConversationThatNeverStopsPagingIsAnErrorNotAPartialRead(t *testing.T) {
	for _, tc := range []struct {
		name string
		// tail is the response to the query that reads one thread's later comments;
		// list is the response to the thread-list query.
		list, tail string
		wantErr    string
	}{
		{"thread list never ends", endlessThreadListPage, longThreadTail, "pages of review threads"},
		{"one conversation never ends", longThreadFirstPage, endlessThreadTail, "read conversation PRRT_kwDOAbCdEf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			list, tail := filepath.Join(bin, "list.json"), filepath.Join(bin, "tail.json")
			for path, body := range map[string]string{list: tc.list, tail: tc.tail} {
				if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			// Dispatch on the query text, as the paging test above does: the stub is
			// endless by construction, so a loop without a bound hangs here instead of
			// running out of canned responses.
			script := "#!/bin/sh\ncase \"$*\" in\n" +
				"'repo view --json owner,name') printf '%s' '{\"owner\":{\"login\":\"dsaiko\"},\"name\":\"fixpoint\"}' ;;\n" +
				"*PullRequestReviewThread*) cat " + tail + " ;;\n" +
				"'api graphql'*) cat " + list + " ;;\n" +
				"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
			if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			got, err := (githubProvider{}).Threads(t.Context(), t.TempDir(), 7)
			if err == nil {
				t.Fatalf("Threads() = %d conversations, nil; a read that never reached the end must not read as the whole exchange", len(got))
			}
			if got != nil {
				t.Errorf("Threads() = %d conversations alongside the error; the pages read so far are a partial exchange, not an answer", len(got))
			}
			// The message has to say what stopped the read -- a bound hit, not a forge
			// that lost the thread -- and name the bound so the operator can tell.
			if !strings.Contains(err.Error(), tc.wantErr) || !strings.Contains(err.Error(), "100") {
				t.Errorf("Threads() = %v, want an error naming %q and the %d-page bound", err, tc.wantErr, maxPages)
			}
		})
	}
}

// The review summaries page too, and past the hundredth review the first page is no
// longer the whole story. The first page stops at hasNextPage; the review carrying
// this tool's marker is only on the second.
const reviewsFirstPage = `{"data":{"repository":{"pullRequest":{"reviews":{
  "pageInfo":{"hasNextPage":true,"endCursor":"r100"},
  "nodes":[{"body":"looks good","author":{"login":"stranger"}}]}}}}}`

const reviewsLastPage = `{"data":{"repository":{"pullRequest":{"reviews":{
  "pageInfo":{"hasNextPage":false,"endCursor":""},
  "nodes":[{"body":"## Changes requested\n\n<!-- ai-panel run 20260101-000000 finding deadbeefcafe -->","author":{"login":"dsaiko"}}]}}}}}`

// A reviews page that never clears hasNextPage: the same non-termination the thread
// list has, one query over.
const endlessReviewsPage = `{"data":{"repository":{"pullRequest":{"reviews":{
  "pageInfo":{"hasNextPage":true,"endCursor":"r100"},
  "nodes":[{"body":"still reviewing","author":{"login":"dsaiko"}}]}}}}}`

// stubGHReviews answers the first reviews query with first and every query carrying
// an after argument with next, recording the last such argv. Dispatching on after=
// rather than on call order is what makes the cursor itself the thing under test: a
// Reviews that forgets to thread endCursor through never reaches next.
func stubGHReviews(t *testing.T, first, next string) (dir string, afterArgv func() string) {
	t.Helper()
	bin := t.TempDir()
	firstFile, nextFile := filepath.Join(bin, "first.json"), filepath.Join(bin, "next.json")
	for path, body := range map[string]string{firstFile: first, nextFile: next} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	argvFile := filepath.Join(bin, "argv.txt")
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"'repo view --json owner,name') printf '%s' '{\"owner\":{\"login\":\"dsaiko\"},\"name\":\"fixpoint\"}' ;;\n" +
		"*after=*) printf '%s' \"$*\" > " + argvFile + "; cat " + nextFile + " ;;\n" +
		"'api graphql'*) cat " + firstFile + " ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return t.TempDir(), func() string {
		raw, err := os.ReadFile(argvFile)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

// A pull request reviewed more than a hundred times, whose earliest review is this
// tool's own. Losing a page here is the quiet failure the pagination exists to stop:
// a review that was never read looks exactly like one that never happened, and every
// body-only finding it carried -- the majority of a review -- is reprinted as new.
func TestReviewsAreReadPastTheirFirstPage(t *testing.T) {
	dir, afterArgv := stubGHReviews(t, reviewsFirstPage, reviewsLastPage)

	got, err := (githubProvider{}).Reviews(t.Context(), dir, 7)
	if err != nil {
		t.Fatalf("Reviews() = %v", err)
	}
	want := []Review{
		{Author: "stranger", Body: "looks good"},
		{Author: "dsaiko", Body: "## Changes requested\n\n<!-- ai-panel run 20260101-000000 finding deadbeefcafe -->"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Reviews() = %+v, want %+v -- the second page was dropped", got, want)
	}
	// The cursor the first page handed back is what asks for the second one, so a
	// call that omits it or sends the wrong value reads the first page forever.
	if !strings.Contains(afterArgv(), "after=r100") {
		t.Errorf("the second graphql call does not carry the first page's cursor: %s", afterArgv())
	}
}

// The reviews loop stops at maxPages, and stopping is an ERROR rather than the pages
// read so far -- for the reason the thread loops refuse: a partial read of what this
// pull request has already been told is indistinguishable from a quiet one, and every
// finding on the pages that were missed comes back as new under the operator's name.
func TestReviewsThatNeverStopPagingAreAnErrorNotAPartialRead(t *testing.T) {
	dir, _ := stubGHReviews(t, endlessReviewsPage, endlessReviewsPage)

	got, err := (githubProvider{}).Reviews(t.Context(), dir, 7)
	if err == nil {
		t.Fatalf("Reviews() = %d reviews, nil; a read that never reached the end must not read as every review submitted", len(got))
	}
	if got != nil {
		t.Errorf("Reviews() = %d reviews alongside the error; the pages read so far are a partial history, not an answer", len(got))
	}
	if !strings.Contains(err.Error(), "pages of reviews") || !strings.Contains(err.Error(), "100") {
		t.Errorf("Reviews() = %v, want an error naming %q and the %d-page bound", err, "pages of reviews", maxPages)
	}
}

// An unreadable answer and an empty one are different facts. readForgeThreads warns
// on an error and proceeds with no conversations, so the two only stay distinguishable
// if the parse refuses to call a malformed response an empty list.
func TestAThreadListThatDoesNotParseIsNotAQuietPullRequest(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response string
		wantErr  bool
	}{
		{"no threads", `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[]}}}}}`, false},
		{"malformed", "<html>proxy error</html>", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, _, _, _ := stubGHThreads(t, tc.response)

			got, err := (githubProvider{}).Threads(t.Context(), dir, 7)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Threads() = %+v, nil; a response that does not parse must not read as a pull request nobody commented on", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Threads() = %v", err)
			}
			if len(got) != 0 {
				t.Errorf("Threads() = %+v, want none", got)
			}
		})
	}
}

// Threads renders databaseId as a decimal string and Reply parses it back with
// ParseInt: the two ends are one contract, so the round trip is what is tested rather
// than either half. The body travels on stdin for the reason a review body does --
// argv is world-readable for the life of the process.
func TestAReplyGoesToTheCommentThreadsNamed(t *testing.T) {
	dir, _, argv, body := stubGHThreads(t, reviewThreadsPayload)

	threads, err := (githubProvider{}).Threads(t.Context(), dir, 7)
	if err != nil || len(threads) != 1 {
		t.Fatalf("Threads() = %+v, %v", threads, err)
	}
	const answer = "fixed by pinning the parse"
	if err := (githubProvider{}).Reply(t.Context(), dir, 7, threads[0].ID, answer); err != nil {
		t.Fatalf("Reply() = %v", err)
	}
	if want := "pulls/7/comments/2147483648/replies"; !strings.Contains(argv(), want) {
		t.Errorf("the reply was addressed to %q, want the root comment %q", argv(), want)
	}
	if strings.Contains(argv(), answer) {
		t.Errorf("the reply body was placed on the command line: %s", argv())
	}
	var got struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal([]byte(body()), &got); err != nil {
		t.Fatalf("the reply payload does not parse (%v): %s", err, body())
	}
	if got.Body != answer {
		t.Errorf("body = %q, want %q", got.Body, answer)
	}
}

// A thread id that is not a comment id is refused before anything is posted. GitHub's
// GraphQL node id for a review thread (PRRT_...) is the plausible wrong value here,
// and appending it to the replies path would POST to an endpoint that means nothing.
func TestAReplyToSomethingThatIsNotACommentIDPostsNothing(t *testing.T) {
	dir, _, argv, body := stubGHThreads(t, reviewThreadsPayload)

	err := (githubProvider{}).Reply(t.Context(), dir, 7, "PRRT_kwDOAbCdEf", "the answer")
	if err == nil {
		t.Fatal("Reply() = nil for a thread id that is not a comment id")
	}
	if !strings.Contains(err.Error(), "not a comment id") {
		t.Errorf("refusal does not explain itself: %v", err)
	}
	if argv() != "" || body() != "" {
		t.Errorf("something was posted anyway: %s / %s", argv(), body())
	}
}

// Requesters decides Origin.Author and Origin.External, which is the provenance a
// coder prompt and a commit message carry to say that untrusted third-party text
// commissioned a change. Each row below is one of the facts its doc commits to, and
// the shapes that matter are the ones a real conversation takes: answered twice with
// somebody writing between the answers, one person speaking again under a different
// capitalisation, an author GitHub would not name, and somebody copying the marker
// into a comment of their own.
func TestRequestersNamesTheLiveRequest(t *testing.T) {
	const marker = "answered <!-- ai-panel run 20260807 -->"
	for _, tc := range []struct {
		name     string
		me       string
		comments []ThreadComment
		want     []string
	}{
		{
			// Never answered: the whole conversation is the live request.
			name: "no reply of ours",
			me:   "dsaiko",
			comments: []ThreadComment{
				{Author: "dsaiko", Body: "missing guard"},
				{Author: "stranger", Body: "above the loop"},
			},
			want: []string{"dsaiko", "stranger"},
		},
		{
			// What was said before our answer is settled; only the follow-up is asking.
			name: "one reply of ours",
			me:   "dsaiko",
			comments: []ThreadComment{
				{Author: "dsaiko", Body: "missing guard"},
				{Author: "dsaiko", Body: marker},
				{Author: "stranger", Body: "now rewrite the parser"},
			},
			want: []string{"stranger"},
		},
		{
			// Anchored on the LAST marker: the person who wrote between the two answers
			// has already been answered, so naming them would credit the wrong request.
			name: "answered twice with a person in between",
			me:   "dsaiko",
			comments: []ThreadComment{
				{Author: "dsaiko", Body: "missing guard"},
				{Author: "dsaiko", Body: marker},
				{Author: "middle", Body: "also the second call site"},
				{Author: "dsaiko", Body: marker},
				{Author: "stranger", Body: "and the parser"},
			},
			want: []string{"stranger"},
		},
		{
			// One account is one requester however they capitalise their login, and the
			// spelling recorded is the one they first used.
			name: "a repeat speaker in mixed case",
			me:   "dsaiko",
			comments: []ThreadComment{
				{Author: "Stranger", Body: "missing guard"},
				{Author: "dsaiko", Body: "which loop?"},
				{Author: "sTrAnGeR", Body: "the outer one"},
			},
			want: []string{"Stranger", "dsaiko"},
		},
		{
			// A deleted account comes back with author.login absent, so the live request
			// has a speaker the forge cannot name. Dropping it would leave the sole
			// requester equal to this run's own account and label the request internal --
			// "we do not know who asked" is exactly what makes it external.
			name: "an author the forge could not name",
			me:   "dsaiko",
			comments: []ThreadComment{
				{Author: "dsaiko", Body: "missing guard"},
				{Author: "dsaiko", Body: marker},
				{Author: "", Body: "now rewrite the parser"},
			},
			want: []string{""},
		},
		{
			// The marker is copyable, so on its own it would let anybody who can comment
			// move the window past their own message: the pull request's author writes
			// what they want done with a marker pasted under it, a maintainer replies,
			// and the request is recorded as the maintainer's alone -- internal, with the
			// third party's text gone from the provenance. A marker not written by this
			// run's account closes nothing.
			name: "a marker copied into somebody else's comment",
			me:   "dsaiko",
			comments: []ThreadComment{
				{Author: "stranger", Body: "rewrite the parser " + marker},
				{Author: "dsaiko", Body: "which parser?"},
			},
			want: []string{"stranger", "dsaiko"},
		},
		{
			// No login to compare against: nothing can be proven ours, so the window is
			// the whole conversation. That names everyone, which is what makes the
			// request external -- the direction that says more.
			name: "the login could not be read",
			me:   "",
			comments: []ThreadComment{
				{Author: "dsaiko", Body: "missing guard"},
				{Author: "dsaiko", Body: marker},
				{Author: "stranger", Body: "now rewrite the parser"},
			},
			want: []string{"dsaiko", "stranger"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Thread{Author: "dsaiko", Comments: tc.comments}.Requesters(tc.me)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Requesters(%q) = %q, want %q", tc.me, got, tc.want)
			}
		})
	}
}

// AnsweredByMachine drops a conversation out of every later run, so the same
// forgeable marker decides whether a thread is ever read again. A third party who
// could end a thread with one would silence the maintainer's question underneath
// it; the account that posted the comment is the half they cannot forge.
func TestOnlyOurOwnAccountsMarkerAnswersAConversation(t *testing.T) {
	const marker = "answered <!-- ai-panel run 20260807 -->"
	for _, tc := range []struct {
		name string
		me   string
		last ThreadComment
		want bool
	}{
		{"our reply", "dsaiko", ThreadComment{Author: "dsaiko", Body: marker}, true},
		{"our reply, other capitalisation", "DSaiko", ThreadComment{Author: "dsaiko", Body: marker}, true},
		{"a marker somebody else wrote", "dsaiko", ThreadComment{Author: "stranger", Body: marker}, false},
		{"the operator asking again", "dsaiko", ThreadComment{Author: "dsaiko", Body: "and the parser?"}, false},
		// Without a login there is nothing to check the author against, so nothing is
		// provably ours. Letting the marker alone stand in would mean a failed `gh api
		// user` is all it takes for a copied marker to bury a maintainer's thread; the
		// caller leaves conversations unread instead of guessing (see readForgeThreads).
		{"no login, our marker", "", ThreadComment{Author: "dsaiko", Body: marker}, false},
		{"no login, no marker", "", ThreadComment{Author: "dsaiko", Body: "and the parser?"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			th := Thread{Author: "dsaiko", Comments: []ThreadComment{
				{Author: "dsaiko", Body: "missing guard"}, tc.last,
			}}
			if got := th.AnsweredByMachine(tc.me); got != tc.want {
				t.Errorf("AnsweredByMachine(%q) = %v, want %v", tc.me, got, tc.want)
			}
		})
	}
	if (Thread{}).AnsweredByMachine("dsaiko") {
		t.Error("a conversation with no comments reads as answered")
	}
}

// gitRepoWithRemotes creates a repository whose remotes are exactly the given
// name/URL pairs, in the given order. A name is written straight into the config
// rather than through `git remote add` so a test can create one git itself would
// refuse to name.
func gitRepoWithRemotes(t *testing.T, remotes [][2]string) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	for _, r := range remotes {
		gitIn(t, dir, "config", "remote."+r[0]+".url", r[1])
	}
	return dir
}

// stubGHRepoView puts a fake `gh` on PATH that answers the base-repository read
// with url, or -- when url is empty -- fails the way gh does when it cannot
// resolve one (not authenticated, no remote it recognizes). Every other
// invocation fails too, so a test row that expects gh NOT to be consulted fails
// loudly if it is.
func stubGHRepoView(t *testing.T, url string) {
	t.Helper()
	bin := t.TempDir()
	answer := "echo 'none of the git remotes point to a known GitHub host' >&2; exit 1"
	if url != "" {
		answer = "printf '%s' '{\"url\":\"" + url + "\"}'"
	}
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"'repo view --json url') " + answer + " ;;\n" +
		"*) echo \"unexpected: $*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
}

// The forge is not always on the remote named origin. `git clone -o upstream`, and
// a mirror setup where origin is an internal git host and the forge is a second
// remote, are checkouts pr mode supports -- target.ghRemote resolves the base
// remote by identity, not by name. Reading origin alone left those runs with a
// nil provider AFTER the whole panel had run: no CI evidence, no conversations,
// no replies, and the requested post silently skipped, every one of them fail-soft.
func TestTheForgeRemoteIsFoundWhateverItIsNamed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		remotes [][2]string
		// ghBase is what the stub `gh` answers the base-repository read with; ""
		// makes it fail the way gh does when it cannot resolve one.
		ghBase string
		want   Kind
	}{
		{"origin is the forge", [][2]string{{"origin", "git@github.com:o/r.git"}}, "", GitHub},
		{"cloned with -o upstream", [][2]string{{"upstream", "https://github.com/o/r.git"}}, "", GitHub},
		{
			"origin is an internal mirror, the forge is a second remote",
			[][2]string{{"origin", "git@git.internal.example:o/r.git"}, {"github", "https://github.com/o/r.git"}},
			"",
			GitHub,
		},
		{
			// Two remotes on the SAME forge are not a disagreement: gh is never asked,
			// and the stub above would fail if it were.
			"a fork and its upstream are both on GitHub",
			[][2]string{{"origin", "https://github.com/me/r.git"}, {"upstream", "https://github.com/o/r.git"}},
			"",
			GitHub,
		},
		{
			// The finding this test row exists for: the pull request was checked out
			// from the GitHub remote, so driving `glab` against merge request N of the
			// GitLab mirror on origin would read, and with -post-verdict act on,
			// something nobody reviewed. gh names the repository it resolved, and that
			// is the one the review is about -- whatever origin points at.
			"origin is a GitLab mirror and the PR came from GitHub",
			[][2]string{{"origin", "git@gitlab.com:g/p.git"}, {"github", "https://github.com/o/r.git"}},
			"https://github.com/o/r",
			GitHub,
		},
		{
			// Same checkout, but nothing can say which forge the pull request came
			// from. No provider at all rather than a guess: the reads lose their
			// evidence and a requested post fails loudly.
			"remotes disagree and gh cannot name the base repository",
			[][2]string{{"origin", "git@gitlab.com:g/p.git"}, {"github", "https://github.com/o/r.git"}},
			"",
			Unknown,
		},
		{"no remote is on a forge", [][2]string{{"origin", "git@git.internal.example:o/r.git"}}, "", Unknown},
		{"no remotes at all", nil, "", Unknown},
		{
			// A remote NAME is repo-controlled config that would become a positional
			// argument to git. One shaped like an option is skipped, not handed over.
			"an option-like remote name is refused rather than resolved",
			[][2]string{{"-x", "https://github.com/o/r.git"}},
			"",
			Unknown,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := gitRepoWithRemotes(t, tc.remotes)
			stubGHRepoView(t, tc.ghBase)
			if got := providerKind(t.Context(), dir); got != tc.want {
				t.Errorf("resolved forge = %q, want %q", got, tc.want)
			}
			p := For(t.Context(), dir)
			if tc.want == Unknown {
				if p != nil {
					t.Errorf("For() = %T, want nil", p)
				}
				return
			}
			if p == nil {
				t.Fatalf("For() = nil, want a %s provider", tc.want)
			}
			if p.Kind() != tc.want {
				t.Errorf("For().Kind() = %q, want %q", p.Kind(), tc.want)
			}
		})
	}
}

// A run has to record WHICH REPOSITORY it reviewed, because a later -post-run has
// nothing else to bind its submission to: the summary's path is a path, and the
// checkout occupying it can be a different repository on the same forge by then.
// The identity comes from gh -- the same resolution `gh pr checkout` used -- and an
// unanswerable read yields nothing rather than a guess, which the callers treat as
// the absence of the fact.
func TestRepoIDIsTheRepositoryGHResolvedForTheCheckout(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ghBase string
		want   string
	}{
		{"the repository gh names", "https://github.com/o/r", "github.com/o/r"},
		{"gh cannot name one", "", ""},
		{
			// A host alone is a forge, not a repository on it: comparing that would
			// accept every repository there, which is the comparison this exists to make.
			"a URL that names no repository",
			"https://github.com", "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := gitRepoWithRemotes(t, [][2]string{{"origin", "https://github.com/o/r.git"}})
			stubGHRepoView(t, tc.ghBase)
			if got := RepoID(t.Context(), dir); got != tc.want {
				t.Errorf("RepoID() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The identity has to be the SAME STRING for the same repository however its URL is
// spelled, or the replay refuses the ordinary case: a run recorded from an https
// remote and replayed in a checkout gh describes with a trailing .git, a port, or
// different capitalization is the same repository, and a refusal there would break
// the mode for everybody. It must still separate repositories that differ only in
// owner or host, which is the whole point of comparing it.
func TestCanonicalRepoNamesOneRepositoryOneWay(t *testing.T) {
	for _, tc := range []struct {
		remote string
		want   string
	}{
		{"https://github.com/Owner/Repo", "github.com/owner/repo"},
		{"https://GitHub.com/owner/repo.git", "github.com/owner/repo"},
		{"https://user:token@github.com:443/owner/repo/", "github.com/owner/repo"},
		{"git@github.com:owner/repo.git", "github.com/owner/repo"},
		{"ssh://git@gitlab.example.com/group/sub/proj.git", "gitlab.example.com/group/sub/proj"},
		{"https://github.com/owner/other", "github.com/owner/other"},
		{"https://github.com/other/repo", "github.com/other/repo"},
		{"https://ghe.example.com/owner/repo", "ghe.example.com/owner/repo"},
		// Neither URL shape, so nothing is named.
		{"/srv/git/repo.git", ""},
		{"", ""},
	} {
		if got := canonicalRepo(tc.remote); got != tc.want {
			t.Errorf("canonicalRepo(%q) = %q, want %q", tc.remote, got, tc.want)
		}
	}
}

// Everything else in this file binds a review to one commit while the run lasts.
// The approval itself is not bound after it: both forges keep counting one toward
// the merge requirements once the branch moves, unless the repository clears
// approvals on a push. The notice is the only thing that tells an operator so, so
// it has to be there on every approval -- and on nothing else, or it becomes the
// warning every comment prints and nobody reads.
func TestAnApprovalSaysItOutlivesTheCommitItWasGivenFor(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	for name, tc := range map[string]struct {
		kind  Kind
		event Event
		want  []string
	}{
		"github approval": {GitHub, EventApprove, []string{"#7", "0123456789ab", "Dismiss stale pull request approvals"}},
		"gitlab approval": {GitLab, EventApprove, []string{"!7", "0123456789ab", "Remove all approvals"}},
		"comment":         {GitHub, Comment, nil},
		"request changes": {GitHub, EventRequestChanges, nil},
	} {
		t.Run(name, func(t *testing.T) {
			got := ApprovalNotice(tc.kind, tc.event, 7, head)
			if len(tc.want) == 0 {
				if got != "" {
					t.Errorf("ApprovalNotice(%s) = %q, want nothing -- it grants no approval", tc.event, got)
				}
				return
			}
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("the notice does not say %q: %s", want, got)
				}
			}
		})
	}
}

// A review permalink is the only handle this run holds on the approval it just
// posted, and asking the API which review is ours would race anything else
// posting. Anything that is not a plain numeric id must fail closed: dismissing
// the wrong review is worse than reporting that we could not dismiss ours.
func TestReviewIDFromURL(t *testing.T) {
	for name, tc := range map[string]struct{ url, want string }{
		"a real permalink": {"https://github.com/o/r/pull/3#pullrequestreview-2938471", "2938471"},
		"no marker":        {"https://github.com/o/r/pull/3", ""},
		"empty":            {"", ""},
		"not numeric":      {"https://github.com/o/r/pull/3#pullrequestreview-2938471/../../9", ""},
		"nothing after":    {"https://github.com/o/r/pull/3#pullrequestreview-", ""},
	} {
		t.Run(name, func(t *testing.T) {
			if got := reviewIDFromURL(tc.url); got != tc.want {
				t.Errorf("reviewIDFromURL(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}
