package forge

import (
	"encoding/json"
	"os"
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

// stubGHRollup puts a fake `gh` on PATH that answers the rollup read -- and only
// that read, matched on the WHOLE argument list, so a change to the command Checks
// runs fails this test instead of quietly returning an unknown rollup. Any other
// invocation exits nonzero, which is what Checks sees when gh cannot answer.
func stubGHRollup(t *testing.T, pr int, stdout string) (dir string) {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"'pr view " + strconv.Itoa(pr) + " --json statusCheckRollup') printf '%s' '" + stdout + "' ;;\n" +
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
	dir := stubGHRollup(t, 7, `{"statusCheckRollup":[
	  {"__typename":"CheckRun","name":"Build","status":"COMPLETED","conclusion":"SUCCESS"},
	  {"__typename":"CheckRun","name":"Unit Tests","status":"COMPLETED","conclusion":"FAILURE"},
	  {"__typename":"CheckRun","name":"Slow","status":"IN_PROGRESS","conclusion":""},
	  {"__typename":"CheckRun","name":"Redundant","status":"COMPLETED","conclusion":"`+ghCancelled+`"},
	  {"__typename":"CheckRun","name":"Optional","status":"COMPLETED","conclusion":"SKIPPED"},
	  {"__typename":"StatusContext","context":"ci/legacy","state":"ERROR"},
	  {"__typename":"StatusContext","context":"ci/queued","state":"PENDING"},
	  {"__typename":"StatusContext","context":"ci/green","state":"SUCCESS"}
	]}`)

	got, err := (githubProvider{}).Checks(t.Context(), dir, 7)
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
	dir := stubGHRollup(t, 7, `{"statusCheckRollup":[]}`)

	got, err := (githubProvider{}).Checks(t.Context(), dir, 7)
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
	dir := stubGHRollup(t, 7, `{"statusCheckRollup":[]}`)

	got, err := (githubProvider{}).Checks(t.Context(), dir, 9)
	if err == nil {
		t.Fatal("Checks() = nil although gh failed")
	}
	if got.Known {
		t.Error("Known = true although the rollup was never read -- an approval would rest on checks nobody saw")
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
// repository dismisses stale reviews. It cannot be un-posted -- it must not be
// reported as a clean approval either.
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
	for _, want := range []string{"PUBLISHED", "dismissed"} {
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
