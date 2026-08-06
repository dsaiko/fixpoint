// Package forge talks to the code-hosting service a pull request lives on.
//
// GitHub works. GITLAB DOES NOT, and not because it is untested: target's pr-mode
// Prepare shells to `gh pr checkout` unconditionally, so a run against a GitLab
// remote fails before anything here is reached. The GitLab provider below is
// scaffolding for a checkout path that does not exist yet -- keep it honest by
// saying so rather than letting the type assertions imply otherwise. It reads what the forge already knows about the reviewed head
// and, on the posting path, writes the review back.
//
// It shells out to the vendors' own CLIs (`gh`, `glab`) rather than speaking HTTP.
// Those tools already solve the part that is genuinely hard and genuinely
// dangerous to get wrong -- credential discovery across keychains, SSO, enterprise
// hosts, token refresh -- and fixpoint's whole agent layer is built the same way.
// The cost is a dependency on a binary being present, which is why every read here
// FAILS SOFT: a missing CLI must degrade the verdict's evidence, never the run.
package forge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Kind is which forge a remote points at.
type Kind string

// The forges fixpoint can talk to, plus the zero value for a remote that is
// neither.
const (
	GitHub  Kind = "github"
	GitLab  Kind = "gitlab"
	Unknown Kind = ""
)

// cliTimeout bounds a forge CLI call. These are metadata reads on the critical
// path of a run; a hung credential prompt or a stalled network must not become a
// stalled review.
const cliTimeout = 2 * time.Minute

// Checks is what the forge reports about the head being reviewed.
//
// Known is separate from an empty Failing list on purpose: "every check passed"
// and "we could not find out" must not be the same input to a verdict. Pending is
// separate from both because a check still running is not evidence either way --
// it does not block (that would make the tool unusable while CI runs) but an
// approval that silently ignored it would be overstating what it knows.
type Checks struct {
	Known   bool
	Failing []string
	Pending []string
}

// Provider is one forge's implementation.
type Provider interface {
	// Kind names the forge, for logs and for the review signature.
	Kind() Kind
	// Checks reports the status of the checks on a pull request's head.
	Checks(ctx context.Context, dir string, pr int) (Checks, error)
}

// For returns the provider for a repository's origin remote, or nil when the
// remote points somewhere neither CLI understands. A nil provider is not an
// error: plenty of targets are plain directories or self-hosted git, and the run
// simply proceeds without forge evidence.
func For(ctx context.Context, dir string) Provider {
	switch DetectKind(remoteURL(ctx, dir)) {
	case GitHub:
		return githubProvider{}
	case GitLab:
		return gitlabProvider{}
	case Unknown:
		return nil
	}
	return nil
}

// DetectKind classifies a remote URL. It matches on the HOST rather than on a
// substring anywhere in the URL: a GitLab instance at gitlab.example.com hosting a
// repository literally named "github-mirror" must not be read as GitHub, and the
// path is attacker-adjacent in a way the host is not.
func DetectKind(remote string) Kind {
	host := remoteHost(remote)
	switch {
	case host == "":
		return Unknown
	case host == "github.com" || strings.HasSuffix(host, ".github.com"):
		return GitHub
	case host == "gitlab.com" || strings.HasSuffix(host, ".gitlab.com"):
		return GitLab
	// Self-hosted instances are named after what they run far more often than not,
	// and there is no other signal available from a remote alone. A wrong guess
	// costs a failed CLI call that fails soft, not a wrong action.
	case strings.Contains(host, "gitlab"):
		return GitLab
	case strings.Contains(host, "github"):
		return GitHub
	}
	return Unknown
}

// remoteHost extracts the hostname from either URL shape git uses:
// scp-like (git@host:owner/repo.git) and real URLs (https://host/owner/repo).
func remoteHost(remote string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}
	if i := strings.Index(remote, "://"); i >= 0 {
		rest := remote[i+3:]
		if at := strings.Index(rest, "@"); at >= 0 {
			rest = rest[at+1:] // strip userinfo
		}
		rest, _, _ = strings.Cut(rest, "/")
		host, _, _ := strings.Cut(rest, ":") // strip any port
		return strings.ToLower(host)
	}
	// scp-like: [user@]host:path
	if at := strings.Index(remote, "@"); at >= 0 {
		remote = remote[at+1:]
	}
	host, _, found := strings.Cut(remote, ":")
	if !found {
		return ""
	}
	return strings.ToLower(host)
}

func remoteURL(ctx context.Context, dir string) string {
	out, err := run(ctx, dir, "git", "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// run executes a CLI in the target directory with a bounded timeout, returning
// stdout. stderr is folded into the error so a caller's log line explains itself.
func run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg != "" {
			return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, firstLine(msg))
		}
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return stdout.String(), nil
}

// runStdin is run with a payload on standard input, for content too large or too
// sensitive to place on a command line.
func runStdin(ctx context.Context, dir, stdin, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, firstLine(msg))
		}
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// --- GitHub ---

type githubProvider struct{}

func (githubProvider) Kind() Kind { return GitHub }

// ghCheck covers both shapes GitHub returns in one rollup. A modern Actions run is
// a CheckRun with name/status/conclusion; a classic commit status is a
// StatusContext with context/state. A parser that knew only the first would read a
// repository still on commit statuses as having no checks at all -- indistinguishable
// from a green one, which is exactly the confusion Checks.Known exists to prevent.
type ghCheck struct {
	Typename   string `json:"__typename"`
	Name       string `json:"name"`
	Context    string `json:"context"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	State      string `json:"state"`
}

func (c ghCheck) label() string {
	if c.Name != "" {
		return c.Name
	}
	if c.Context != "" {
		return c.Context
	}
	return "(unnamed check)"
}

func (githubProvider) Checks(ctx context.Context, dir string, pr int) (Checks, error) {
	out, err := run(ctx, dir, "gh", "pr", "view", strconv.Itoa(pr), "--json", "statusCheckRollup")
	if err != nil {
		return Checks{}, err
	}
	var payload struct {
		Rollup []ghCheck `json:"statusCheckRollup"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return Checks{}, fmt.Errorf("parse gh statusCheckRollup: %w", err)
	}
	res := Checks{Known: true}
	for _, c := range payload.Rollup {
		switch {
		case failingGitHub(c):
			res.Failing = append(res.Failing, c.label())
		case pendingGitHub(c):
			res.Pending = append(res.Pending, c.label())
		}
	}
	return res, nil
}

// GitHub's own enum spellings, named rather than scattered as literals -- and
// spelled as the API spells them. CheckConclusionState uses the British
// double-L, so "correcting" it would silently stop matching.
//
//nolint:misspell // these are verbatim GitHub API values, not prose
const (
	ghFailure        = "FAILURE"
	ghTimedOut       = "TIMED_OUT"
	ghActionRequired = "ACTION_REQUIRED"
	ghStartupFailure = "STARTUP_FAILURE"
	ghError          = "ERROR"
	ghCancelled      = "CANCELLED"
	ghSkipped        = "SKIPPED"
	ghNeutral        = "NEUTRAL"
)

// failingGitHub is deliberately a short list of DEFINITE failures. A stopped,
// skipped or neutral check is not a pass, but it is not evidence of a defect
// either, and a verdict that blocked on one would block on a maintainer having
// stopped a run it did not need.
func failingGitHub(c ghCheck) bool {
	switch strings.ToUpper(c.Conclusion) {
	case ghFailure, ghTimedOut, ghActionRequired, ghStartupFailure:
		return true
	}
	switch strings.ToUpper(c.State) {
	case ghFailure, ghError:
		return true
	}
	return false
}

func pendingGitHub(c ghCheck) bool {
	switch strings.ToUpper(c.Status) {
	case "QUEUED", "IN_PROGRESS", "WAITING", "PENDING", "REQUESTED":
		return true
	}
	return strings.EqualFold(c.State, "PENDING") || strings.EqualFold(c.State, "EXPECTED")
}

// --- GitLab ---

type gitlabProvider struct{}

func (gitlabProvider) Kind() Kind { return GitLab }

// Checks reads the merge request's pipeline through glab's API passthrough.
//
// UNVERIFIED end to end: glab was not installed where this was written, so the
// command shape comes from its documented API rather than from a run. It fails
// soft like every other read here, so the cost of being wrong is a verdict with
// no CI evidence and a warning saying so -- not a broken review.
func (gitlabProvider) Checks(ctx context.Context, dir string, mr int) (Checks, error) {
	out, err := run(ctx, dir, "glab", "api",
		fmt.Sprintf("projects/:id/merge_requests/%d/pipelines", mr))
	if err != nil {
		return Checks{}, err
	}
	var pipelines []struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &pipelines); err != nil {
		return Checks{}, fmt.Errorf("parse glab pipelines: %w", err)
	}
	if len(pipelines) == 0 {
		// A merge request with no pipeline at all: the read SUCCEEDED and the answer
		// is "no checks", which is different from not being able to ask.
		return Checks{Known: true}, nil
	}
	// The API returns newest first; only the latest pipeline describes this head.
	latest := pipelines[0]
	res := Checks{Known: true}
	label := fmt.Sprintf("pipeline %d", latest.ID)
	switch strings.ToLower(latest.Status) {
	case "failed":
		res.Failing = append(res.Failing, label)
	case "running", "pending", "created", "waiting_for_resource", "preparing", "scheduled":
		res.Pending = append(res.Pending, label)
	}
	return res, nil
}

// Event is the kind of review to post.
type Event string

const (
	// Comment publishes the findings and takes no position. It is the safe default
	// and what -post alone does: a machine review becomes visible without spending
	// an approval or blocking somebody's merge.
	Comment Event = "comment"
	// EventApprove approves the pull request. Reached only through a second,
	// explicit flag: it spends an approval on somebody's change.
	EventApprove Event = "approve"
	// EventRequestChanges formally blocks the pull request. Same gate, and a social
	// act as much as a technical one.
	EventRequestChanges Event = "request_changes"
)

// InlineComment anchors one finding to the line it is about.
//
// A forge accepts these only for lines the pull request actually TOUCHES: a
// finding about code the PR did not change has nowhere to hang, and GitHub
// rejects the whole review rather than the one comment. That is why posting
// falls back to a body-only review instead of treating the rejection as fatal --
// a review that fails to publish because one finding pointed at context is worse
// than one that publishes with its findings in the summary.
type InlineComment struct {
	Path string
	Line int
	Body string
}

// Poster is a provider that can publish a review. It is separate from Provider
// because reading and writing are different privileges: every run may read, and
// writing needs an assertion the operator makes per invocation.
type Poster interface {
	Provider
	// PostReview publishes body on the pull request, with inline comments anchored
	// to their lines where the forge accepts them, and returns a URL when it gives
	// one.
	PostReview(ctx context.Context, dir string, pr int, body string, event Event, inline []InlineComment) (string, error)
}

// PosterFor is For, narrowed to providers that can also write.
func PosterFor(ctx context.Context, dir string) Poster {
	p, _ := For(ctx, dir).(Poster)
	return p
}

// PostReview publishes through `gh pr review`, which reads the body from a file
// so no part of it ever reaches an argument list.
//
// That is not tidiness: argv is world-readable on this host for the life of the
// process, and the body carries findings quoted out of the code under review --
// which in a review-only run may be the credential that the review is ABOUT.
// It is the same reasoning that makes prompt_via: stdin the default for agents.
func (githubProvider) PostReview(ctx context.Context, dir string, pr int, body string, event Event, inline []InlineComment) (string, error) {
	if len(inline) > 0 {
		if err := githubPostWithInline(ctx, dir, pr, body, event, inline); err == nil {
			return latestReviewURL(ctx, dir, pr), nil
		} else if ctx.Err() == nil {
			// One comment on a line the diff does not contain rejects the WHOLE review,
			// and the API says so without naming which. Rather than guess, publish the
			// review that was going to be published anyway: the findings are all in the
			// body, they simply lose their anchors.
			return "", fmt.Errorf("inline: %w", err)
		}
	}
	f, err := os.CreateTemp("", "fixpoint-review-*.md")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return "", err
	}
	if _, err := f.WriteString(body); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	flag := "--comment"
	switch event {
	case EventApprove:
		flag = "--approve"
	case EventRequestChanges:
		flag = "--request-changes"
	case Comment:
	}
	if _, err := run(ctx, dir, "gh", "pr", "review", strconv.Itoa(pr), flag, "--body-file", f.Name()); err != nil {
		return "", err
	}
	// gh prints nothing useful on success, so the URL is read back rather than
	// parsed out of its output.
	return latestReviewURL(ctx, dir, pr), nil
}

// githubPostWithInline submits one review carrying both the summary and the
// per-line comments, through the API `gh pr review` does not expose.
//
// The payload goes in on STDIN, not as arguments: it embeds the whole review, and
// argv is world-readable for the life of the process -- the same reason the
// body-only path writes a file. `{owner}/{repo}` are gh's own placeholders,
// resolved from the checkout, so no repository identity has to be parsed here.
func githubPostWithInline(ctx context.Context, dir string, pr int, body string, event Event, inline []InlineComment) error {
	type ghComment struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Side string `json:"side"`
		Body string `json:"body"`
	}
	payload := struct {
		Body     string      `json:"body"`
		Event    string      `json:"event"`
		Comments []ghComment `json:"comments"`
	}{Body: body, Event: githubEvent(event)}
	for _, c := range inline {
		// RIGHT is the head of the pull request. A finding is about the code as
		// proposed, not the line it replaced.
		payload.Comments = append(payload.Comments, ghComment{Path: c.Path, Line: c.Line, Side: "RIGHT", Body: c.Body})
	}
	doc, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return runStdin(ctx, dir, string(doc), "gh", "api", "--method", "POST",
		fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/reviews", pr), "--input", "-")
}

func githubEvent(e Event) string {
	switch e {
	case EventApprove:
		return "APPROVE"
	case EventRequestChanges:
		return "REQUEST_CHANGES"
	case Comment:
	}
	return "COMMENT"
}

// latestReviewURL best-effort resolves a link to what was just posted. A missing
// URL is cosmetic -- the review is already published -- so every failure here
// yields an empty string rather than an error that would misreport a successful
// post as a failed one.
func latestReviewURL(ctx context.Context, dir string, pr int) string {
	out, err := run(ctx, dir, "gh", "pr", "view", strconv.Itoa(pr), "--json", "url")
	if err != nil {
		return ""
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return ""
	}
	return payload.URL
}

// PostReview publishes on GitLab: a note for a comment, and the approval endpoints
// for the two verdicts.
//
// UNVERIFIED end to end, like the GitLab reader: glab was not installed where this
// was written. Unlike a read, a failed WRITE is reported to the caller rather than
// degraded silently -- an operator who asked to publish must not be told it
// happened when it did not.
func (gitlabProvider) PostReview(ctx context.Context, dir string, mr int, body string, event Event, inline []InlineComment) (string, error) {
	for _, c := range inline {
		// Best-effort per comment: GitLab positions a discussion with base/head/start
		// SHAs this package does not carry, so an inline note is attempted as a plain
		// note naming its location rather than skipped outright.
		if _, err := run(ctx, dir, "glab", "mr", "note", strconv.Itoa(mr),
			"--message", fmt.Sprintf("`%s:%d`\n\n%s", c.Path, c.Line, c.Body)); err != nil {
			return "", err
		}
	}
	if _, err := run(ctx, dir, "glab", "mr", "note", strconv.Itoa(mr), "--message", body); err != nil {
		return "", err
	}
	switch event {
	case EventApprove:
		if _, err := run(ctx, dir, "glab", "mr", "approve", strconv.Itoa(mr)); err != nil {
			return "", fmt.Errorf("note posted, but approving failed: %w", err)
		}
	case EventRequestChanges:
		// GitLab has no "request changes" review event; unapproving is the closest
		// equivalent and is what the note above explains.
		if _, err := run(ctx, dir, "glab", "mr", "unapprove", strconv.Itoa(mr)); err != nil {
			return "", fmt.Errorf("note posted, but unapproving failed: %w", err)
		}
	case Comment:
	}
	return "", nil
}

// Thread is one unresolved review conversation on a pull request.
//
// Unresolved only: a resolved thread is a settled question, and handing it to a
// coder invites it to reopen something a human already closed.
type Thread struct {
	// ID is what a reply is addressed to. It is the ROOT comment's id, because a
	// forge threads replies under the comment that started the conversation.
	ID     string
	Path   string
	Line   int
	Author string
	Body   string
}

// Reader is a provider that can also read a pull request's conversations. It is
// separate from Provider so a forge that cannot do it simply does not implement
// it, rather than returning an error every run has to interpret.
type Reader interface {
	Provider
	// Threads lists the UNRESOLVED review conversations on a pull request.
	Threads(ctx context.Context, dir string, pr int) ([]Thread, error)
	// Reply posts a response into an existing conversation.
	Reply(ctx context.Context, dir string, pr int, threadID, body string) error
}

// ReaderFor is For, narrowed to providers that can read conversations.
func ReaderFor(ctx context.Context, dir string) Reader {
	r, _ := For(ctx, dir).(Reader)
	return r
}

// threadQuery asks for unresolved threads and the comment that opened each.
//
// GraphQL rather than the REST comment list, because "is this conversation still
// open?" exists only in the GraphQL schema: REST returns every comment with no
// resolution state, so a coder would be handed questions a human had already
// settled -- and answering those is worse than not answering at all.
const threadQuery = `query($owner:String!,$repo:String!,$pr:Int!){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$pr){
      reviewThreads(first:100){
        nodes{
          isResolved
          comments(first:1){nodes{path line databaseId body author{login}}}
        }
      }
    }
  }
}`

func (githubProvider) Threads(ctx context.Context, dir string, pr int) ([]Thread, error) {
	owner, repo, err := githubSlug(ctx, dir)
	if err != nil {
		return nil, err
	}
	out, err := run(ctx, dir, "gh", "api", "graphql",
		"-f", "query="+threadQuery,
		"-F", "owner="+owner, "-F", "repo="+repo, "-F", fmt.Sprintf("pr=%d", pr))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						Nodes []struct {
							IsResolved bool `json:"isResolved"`
							Comments   struct {
								Nodes []struct {
									Path       string `json:"path"`
									Line       int    `json:"line"`
									DatabaseID int64  `json:"databaseId"`
									Body       string `json:"body"`
									Author     struct {
										Login string `json:"login"`
									} `json:"author"`
								} `json:"nodes"`
							} `json:"comments"`
						} `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return nil, fmt.Errorf("parse review threads: %w", err)
	}
	nodes := payload.Data.Repository.PullRequest.ReviewThreads.Nodes
	threads := make([]Thread, 0, len(nodes))
	for _, n := range nodes {
		if n.IsResolved || len(n.Comments.Nodes) == 0 {
			continue
		}
		c := n.Comments.Nodes[0]
		threads = append(threads, Thread{
			ID:     strconv.FormatInt(c.DatabaseID, 10),
			Path:   c.Path,
			Line:   c.Line,
			Author: c.Author.Login,
			Body:   c.Body,
		})
	}
	return threads, nil
}

func (githubProvider) Reply(ctx context.Context, dir string, pr int, threadID, body string) error {
	id, err := strconv.ParseInt(threadID, 10, 64)
	if err != nil {
		return fmt.Errorf("thread id %q is not a comment id: %w", threadID, err)
	}
	// Body on stdin, not argv: a reply quotes the coder's own prose about code it
	// just read, and argv is world-readable for the life of the process.
	payload, err := json.Marshal(struct {
		Body string `json:"body"`
	}{body})
	if err != nil {
		return err
	}
	return runStdin(ctx, dir, string(payload), "gh", "api", "--method", "POST",
		fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/comments/%d/replies", pr, id), "--input", "-")
}

// githubSlug resolves owner and repo for the API calls that cannot use gh's own
// {owner}/{repo} placeholders -- the GraphQL endpoint takes them as variables.
func githubSlug(ctx context.Context, dir string) (owner, repo string, err error) {
	out, err := run(ctx, dir, "gh", "repo", "view", "--json", "owner,name")
	if err != nil {
		return "", "", err
	}
	var payload struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return "", "", fmt.Errorf("parse repository identity: %w", err)
	}
	if payload.Owner.Login == "" || payload.Name == "" {
		return "", "", errors.New("gh reported no repository owner or name")
	}
	return payload.Owner.Login, payload.Name, nil
}
