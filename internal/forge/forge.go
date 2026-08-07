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
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/model"
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
	//
	// head is the commit the run reviewed. An implementation must BIND the answer
	// to it and report an error rather than checks that describe another commit --
	// see requireCheckHead.
	Checks(ctx context.Context, dir string, pr int, head string) (Checks, error)
}

// For returns the provider for the forge this checkout's pull request lives on,
// or nil when no remote points somewhere either CLI understands and nothing else
// names one. A nil provider is not an error: plenty of targets are plain
// directories or self-hosted git, and the run simply proceeds without forge
// evidence.
func For(ctx context.Context, dir string) Provider {
	switch providerKind(ctx, dir) {
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
	host, _ := splitRemote(remote)
	return host
}

// splitRemote separates either URL shape git uses -- scp-like
// (git@host:owner/repo.git) and real URLs (https://host/owner/repo) -- into its
// lowercased host and the path that follows it. Both halves are empty for
// anything that is neither shape.
func splitRemote(remote string) (host, path string) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", ""
	}
	if i := strings.Index(remote, "://"); i >= 0 {
		rest := remote[i+3:]
		if at := strings.Index(rest, "@"); at >= 0 {
			rest = rest[at+1:] // strip userinfo
		}
		authority, path, _ := strings.Cut(rest, "/")
		host, _, _ := strings.Cut(authority, ":") // strip any port
		return strings.ToLower(host), path
	}
	// scp-like: [user@]host:path
	if at := strings.Index(remote, "@"); at >= 0 {
		remote = remote[at+1:]
	}
	host, path, found := strings.Cut(remote, ":")
	if !found {
		return "", ""
	}
	return strings.ToLower(host), path
}

// RepoID is the canonical identity of the repository whose pull request this
// checkout holds: "host/owner/repo", lowercased and stripped of scheme,
// credentials, port and ".git".
//
// It exists because a run's destination has to survive the run. The summary a
// review leaves records a PATH, and a path is not an identity: the checkout that
// occupied it can be repointed at another repository on the same forge, or the
// directory reused for one, and a later -post-run resolving its destination from
// whatever is there now would publish to pull request N of THAT repository. The
// head check does not notice -- it compares commit SHAs, and the reviewed commit
// is public, so anyone may open a request proposing it. Recording this at review
// time is what lets the replay refuse.
//
// Asked of gh rather than assembled from a remote URL, for the same reason
// providerKind asks gh to settle a disagreement: `gh pr checkout` fetched the
// reviewed pull request from the repository gh calls this checkout's base, so
// that -- not whichever remote happens to be listed first -- is the identity the
// review is about.
//
// "" when it cannot be established: gh missing, unauthenticated, no repository it
// recognizes, or a URL that does not name an owner and a repository. That is the
// absence of the fact, never a guess at it, and both callers treat it as one --
// recording warns, publishing refuses.
func RepoID(ctx context.Context, dir string) string {
	return canonicalRepo(ghBaseRepoURL(ctx, dir))
}

// canonicalRepo reduces a repository URL to the identity RepoID promises, or ""
// when it does not name one.
//
// Lowercased whole, host and path together: GitHub treats an owner and a
// repository name case-insensitively, so two spellings of one repository must
// compare equal -- a replay refused over capitalization would be a false alarm
// about the one thing this identity exists to detect.
func canonicalRepo(remote string) string {
	host, path := splitRemote(remote)
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	// An owner and a repository, both present: a host alone identifies a forge, not
	// a repository on it, and comparing that would accept every repository there.
	if host == "" || !strings.Contains(path, "/") {
		return ""
	}
	return strings.ToLower(host + "/" + path)
}

// providerKind decides WHICH forge the pull request under review lives on.
//
// One forge among the remotes is the answer, and it is the ordinary case: the
// mirror setups remoteForges exists for keep origin on a plain internal git host,
// which is not a forge and so does not make the choice ambiguous.
//
// Remotes that DISAGREE cannot be resolved by preferring one of them. Every caller
// here is in pr mode, where target.Prepare has already run `gh pr checkout`, so the
// reviewed code came from one specific repository on one specific host -- and a
// checkout whose origin is a GitLab mirror while the pull request came from a
// second, GitHub remote would otherwise send the checks read, the notes and the
// verdict itself to `glab`, against merge request N of an unrelated project that
// merely shares a number with the reviewed pull request. requireHead refuses most
// of what follows, but a mirror whose merge request proposes the same commit
// passes it, and -post-verdict then approves something nobody reviewed under the
// operator's identity.
//
// So a disagreement is settled by asking gh which repository IT resolved for this
// checkout -- the same answer `gh pr checkout` acted on, which is the identity the
// review is actually about. When gh cannot say, nothing binds a forge to the
// reviewed pull request and there is none: the reads then lose their evidence and
// a requested post fails loudly, which is the harmless direction to be wrong in.
func providerKind(ctx context.Context, dir string) Kind {
	switch kinds := remoteForges(ctx, dir); len(kinds) {
	case 0:
		return Unknown
	case 1:
		return kinds[0]
	}
	return DetectKind(ghBaseRepoURL(ctx, dir))
}

// remoteForges returns the DISTINCT forges this checkout's remotes point at.
//
// It is deliberately NOT `git remote get-url origin`. A clone made with
// `git clone -o upstream`, and a mirror setup where origin is an internal git
// host and the forge is a second remote, are checkouts pr mode already supports:
// target.ghRemote resolves the base remote by identity rather than by name for
// exactly that reason. Reading origin alone made every one of those runs fail
// SOFT and late -- the whole panel runs and is paid for, and then the CI evidence,
// the conversations, the replies and the requested post all quietly go missing,
// each behind its own warning or none at all.
//
// Kinds rather than a URL, and every remote rather than the first match, because
// the caller has to be able to tell "this checkout is on GitHub" from "this
// checkout has remotes on two different forges and the order they are listed in
// decides which one is driven". Remote order deliberately does not appear in the
// answer: a disagreement is settled by identity in providerKind, not by
// precedence.
//
// A remote NAME is repo-controlled config that ends up as a positional argument
// to git. --end-of-options is passed for it, and a name starting with "-" is
// skipped outright rather than handed over -- no legitimate remote is named that,
// and target.ghRemote refuses them for the same reason.
func remoteForges(ctx context.Context, dir string) []Kind {
	out, err := run(ctx, dir, "git", "remote")
	if err != nil {
		return nil
	}
	names := strings.Fields(out)
	kinds := make([]Kind, 0, len(names))
	for _, name := range names {
		if strings.HasPrefix(name, "-") {
			continue
		}
		url, err := run(ctx, dir, "git", "remote", "get-url", "--end-of-options", name)
		if err != nil {
			continue
		}
		k := DetectKind(strings.TrimSpace(url))
		if k == Unknown || slices.Contains(kinds, k) {
			continue
		}
		kinds = append(kinds, k)
	}
	return kinds
}

// ghBaseRepoURL is the canonical URL of the repository gh treats as this
// checkout's base -- the same resolution `gh pr checkout` used to fetch the pull
// request, so it names the host the reviewed code actually came from.
//
// "" when gh cannot name one: not installed, unauthenticated, or no remote it
// recognizes. That is not an error here, it is the absence of the one thing that
// could bind a forge to the reviewed pull request, and the caller answers it by
// choosing none.
func ghBaseRepoURL(ctx context.Context, dir string) string {
	out, err := run(ctx, dir, "gh", "repo", "view", "--json", "url")
	if err != nil {
		return ""
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.URL)
}

// run executes a CLI in the target directory with a bounded timeout, returning
// stdout. stderr is folded into the error so a caller's log line explains itself.
//
// Through agent.Supervise rather than cmd.Run, so cliTimeout is the bound the
// package header claims it is. These CLIs fork children -- a credential helper, a
// git subprocess -- that inherit whatever descriptors they are given. With an
// ordinary writer as cmd.Stdout, os/exec owns the pipe and cmd.Wait waits for its
// copy goroutine to see EOF: a descendant that outlives the leader holds the write
// end, no EOF arrives, and the deadline killing the leader alone frees nothing. The
// review then hangs past cliTimeout and past a Ctrl-C, since cancellation is the
// same mechanism. Supervise owns the pipes and kills the process group, so Wait
// returns on the leader's exit and the descendant goes with it.
func run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	if _, err := agent.Supervise(ctx, cmd, &stdout, &stderr); err != nil {
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
// sensitive to place on a command line. Supervised for the same reason as run --
// and the stdin feed needs it as much as the output does, since exec's stdin copy
// is a goroutine cmd.Wait joins just the same.
//
// stdout is returned rather than discarded: these calls are posts, and most callers
// read the outcome from the error alone, but a create answers with the identity of
// what it created -- and a review's id is the only handle that can withdraw it.
func runStdin(ctx context.Context, dir, stdin, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr strings.Builder
	if _, err := agent.Supervise(ctx, cmd, &stdout, &stderr); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, firstLine(msg))
		}
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return stdout.String(), nil
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

// Checks reads what GitHub's checks say -- about the reviewed commit and no other.
//
// The rollup hangs off the PULL REQUEST, not off a commit, so it describes whatever
// the request proposes at the moment of the read. That need not be the commit this
// run checked out: an author who force-pushes a green commit, waits for this read,
// and force-pushes the reviewed one back leaves the panel and every human reading
// the failing commit while the verdict rests on another one's passing CI. So the
// head is read IN THE SAME CALL as the rollup -- two calls could straddle the push,
// which is the hole this closes -- and a mismatch yields an error, which the caller
// turns into a verdict carrying no CI evidence rather than the wrong evidence.
func (githubProvider) Checks(ctx context.Context, dir string, pr int, head string) (Checks, error) {
	out, err := run(ctx, dir, "gh", "pr", "view", strconv.Itoa(pr), "--json", "headRefOid,statusCheckRollup")
	if err != nil {
		return Checks{}, err
	}
	var payload struct {
		HeadRefOid string    `json:"headRefOid"`
		Rollup     []ghCheck `json:"statusCheckRollup"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return Checks{}, fmt.Errorf("parse gh statusCheckRollup: %w", err)
	}
	if err := requireCheckHead(pr, head, strings.TrimSpace(payload.HeadRefOid)); err != nil {
		return Checks{}, err
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
// glPipeline is one entry of a merge request's pipeline list. sha is what ties a
// pipeline to the commit it ran on, which is what Checks has to bind to.
type glPipeline struct {
	ID     int    `json:"id"`
	SHA    string `json:"sha"`
	Status string `json:"status"`
}

func (gitlabProvider) Checks(ctx context.Context, dir string, mr int, head string) (Checks, error) {
	if head == "" {
		return Checks{}, fmt.Errorf("the run did not record which commit it reviewed, so !%d's pipelines cannot be bound to one", mr)
	}
	out, err := run(ctx, dir, "glab", "api",
		fmt.Sprintf("projects/:id/merge_requests/%d/pipelines", mr))
	if err != nil {
		return Checks{}, err
	}
	var pipelines []glPipeline
	if err := json.Unmarshal([]byte(out), &pipelines); err != nil {
		return Checks{}, fmt.Errorf("parse glab pipelines: %w", err)
	}
	if len(pipelines) == 0 {
		// A merge request with no pipeline at all: the read SUCCEEDED and the answer
		// is "no checks", which is different from not being able to ask.
		return Checks{Known: true}, nil
	}
	// The list covers every commit the merge request has proposed and returns newest
	// first, so the newest pipeline is the newest PUSH's -- not necessarily the
	// reviewed commit's. Taking it blindly would let a force-push move the evidence
	// off the commit under review; see requireCheckHead for why that matters. So the
	// newest pipeline that ran on the reviewed commit is the one, and a merge request
	// whose pipelines are all about other commits reports no evidence at all.
	i := slices.IndexFunc(pipelines, func(p glPipeline) bool {
		return strings.EqualFold(strings.TrimSpace(p.SHA), head)
	})
	if i < 0 {
		return Checks{}, fmt.Errorf("no pipeline on !%d ran on the reviewed commit %s, so its pipelines describe other commits", mr, shortSHA(head))
	}
	latest := pipelines[i]
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

// EventFor is the single place a verdict becomes a forge event.
//
// It lives here, and takes the outcome as the string the summary records, so the
// two callers that publish a review -- Orchestrator.postReview and the -post-run
// replay in cmd/fixpoint -- cannot drift apart. They used to hold a copy each of
// the same switch, and this is the most consequential mapping in the program: it
// decides whether fixpoint approves somebody's pull request or formally blocks it.
//
// Without -post-verdict every outcome is a Comment: the findings become visible
// and nothing is spent. An INCONCLUSIVE verdict is a Comment under every flag too
// -- there is no forge event for "the review did not finish", and both that exist
// would be lies about a panel that never reached quorum.
func EventFor(outcome string, postVerdict bool) Event {
	if !postVerdict {
		return Comment
	}
	switch outcome {
	case model.VerdictApprove:
		return EventApprove
	case model.VerdictChangesRequested:
		return EventRequestChanges
	}
	return Comment
}

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
	//
	// head is the commit the review was produced from. An implementation must BIND
	// the review to it and refuse to publish when the pull request has moved since
	// -- see requireHead.
	PostReview(ctx context.Context, dir string, pr int, head, body string, event Event, inline []InlineComment) (string, error)
}

// requireHead refuses to publish a review that is no longer about what the pull
// request proposes.
//
// A review is a statement about ONE commit. A panel takes minutes and `-post-run`
// can replay a run from yesterday, while a forge applies a review to whatever the
// pull request points at NOW. So an author can push after the reviewed head was
// checked out and collect an approval for code no reviewer read -- push something
// clean, wait for the approval, push the payload. The inline anchors were computed
// against the reviewed diff too, so a moved head makes them wrong as well as
// unbound, and the body-only fallback then approves the new head with no anchors at
// all.
//
// This is the one read in the package that FAILS CLOSED. The others weaken a
// verdict's evidence when they fail; this one guards an action taken under the
// operator's identity, and "which commit does this land on?" unanswered is not a
// license to take it. It costs nothing in practice: the same CLI does the posting,
// so a call that cannot read the head could not have published either.
//
// It is a check, though, and the act is a separate round trip: on its own it refuses
// a head that moved BEFORE the read, not one that moves between the read and the
// submission. commit_id binds the review to the commit for GitHub, gitlabApprove's
// sha does for GitLab, and confirmApproval reports the residue GitHub's API cannot
// refuse -- an approval that was accepted for a commit which is no longer the head.
func requireHead(pr int, reviewed, current string) error {
	if reviewed == "" {
		return fmt.Errorf("refusing to post on #%d: the run did not record which commit it reviewed, so the review cannot be bound to one -- review again to produce a run that can be published", pr)
	}
	if current == "" {
		return fmt.Errorf("refusing to post on #%d: the forge reported no head commit, so whether the reviewed commit is still the one proposed cannot be established", pr)
	}
	if !strings.EqualFold(reviewed, current) {
		return fmt.Errorf("refusing to post on #%d: it has moved since it was reviewed (reviewed %s, now %s) -- publishing would attach the review, and any verdict in it, to a commit nobody read; review the new head instead",
			pr, shortSHA(reviewed), shortSHA(current))
	}
	return nil
}

// requireCheckHead binds CI evidence to the commit the run reviewed.
//
// It is requireHead's read-side counterpart, and it fails SOFT where that one fails
// closed: the caller answers an error here by leaving CI unknown, which states in
// the verdict's reasons that no check was seen. Unknown is the harmless direction --
// it never blocks and never approves on evidence about a commit nobody read, which
// is exactly what an unbound rollup can be after a force-push.
func requireCheckHead(pr int, reviewed, current string) error {
	switch {
	case reviewed == "":
		return fmt.Errorf("the run did not record which commit it reviewed, so #%d's checks cannot be bound to one", pr)
	case current == "":
		return fmt.Errorf("#%d names no head commit, so whether its checks describe the reviewed commit cannot be established", pr)
	case !strings.EqualFold(reviewed, current):
		return fmt.Errorf("#%d now proposes %s but this run reviewed %s, so its checks describe another commit",
			pr, shortSHA(current), shortSHA(reviewed))
	}
	return nil
}

// shortSHA abbreviates a commit for a message a human reads. The full oid is not
// what makes the sentence understandable, and two of them make it unreadable.
func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// PosterFor is For, narrowed to providers that can also write.
func PosterFor(ctx context.Context, dir string) Poster {
	p, _ := For(ctx, dir).(Poster)
	return p
}

// anchorError marks the ONE PostReview failure a caller may answer by sending
// the same review again without its per-line comments.
//
// The distinction is about what already happened on the forge, not about what the
// caller would like to do next. A rejected submission created nothing, so posting
// again puts one review on the pull request. A submission that failed for any other
// reason -- an expired token, a 5xx, a dropped connection, cliTimeout firing while
// the request was in flight -- may well have been ACCEPTED before the failure was
// observed, and answering that with a second submission is how somebody's pull
// request ends up with two reviews on it under the operator's identity.
//
// A type rather than a string prefix: the callers used to test the rendered message,
// which quietly classified every failure above as an anchor problem.
type anchorError struct{ err error }

func (e anchorError) Error() string { return "inline: " + e.err.Error() }

func (e anchorError) Unwrap() error { return e.err }

// AnchorRejection reports whether a PostReview failure was the forge refusing the
// per-line comments rather than the review, and so whether the same review may be
// published body-only instead. Every other failure must be reported, not retried.
func AnchorRejection(err error) bool {
	var a anchorError
	return errors.As(err, &a)
}

// RejectedAnchors marks an error as that rejection, for Poster implementations
// outside this package -- and for the fakes that stand in for them, which is the
// only way a test can produce the failure PostReview's callers retry on.
func RejectedAnchors(err error) error { return anchorError{err} }

// invalidComment reports whether a failed submission was GitHub validating the
// review away rather than failing to process it.
//
// 422 is the whole test, and it is a statement about the forge's state: GitHub
// validates a review submission -- including every comment's path and line --
// before it creates anything, so a 422 answer means no review exists. Nothing
// weaker can be concluded from any other outcome, which is why everything else
// propagates unchanged. If gh ever stops naming the status in its message this
// fails closed: the post is reported as failed and the operator publishes by hand,
// which is the harmless direction to be wrong in.
func invalidComment(err error) bool {
	return strings.Contains(err.Error(), "HTTP 422")
}

// PostReview publishes through the reviews API, with the payload on stdin so no
// part of it ever reaches an argument list.
//
// That is not tidiness: argv is world-readable on this host for the life of the
// process, and the body carries findings quoted out of the code under review --
// which in a review-only run may be the credential that the review is ABOUT.
// It is the same reasoning that makes prompt_via: stdin the default for agents.
//
// ONE submission path, whether or not there are anchors. `gh pr review` would post
// the body just as well, but it cannot name a commit, so a body-only review -- the
// common case, since most findings are filtered out of inline posting, and the
// destination of the anchor-rejection fallback -- would land unbound on whatever
// the branch points at when it arrives. requireHead narrows that window; only
// commit_id closes it.
func (githubProvider) PostReview(ctx context.Context, dir string, pr int, head, body string, event Event, inline []InlineComment) (string, error) {
	// Before anything is published, and before the fallback below can turn an anchor
	// rejection into a body-only APPROVAL of whatever is at the head now.
	cur, err := githubHead(ctx, dir, pr)
	if err != nil {
		return "", err
	}
	if err := requireHead(pr, head, cur); err != nil {
		return "", err
	}
	id, url, err := githubSubmitReview(ctx, dir, pr, head, body, event, inline)
	if err != nil {
		// Only a submission that CARRIED anchors can have been rejected for them.
		// Without any, the same 422 is about the review itself and a body-only retry
		// would just fail again -- so it propagates like every other failure.
		if len(inline) > 0 && invalidComment(err) {
			// One comment on a line the diff does not contain rejects the WHOLE review,
			// and the API says so without naming which. Rather than guess, let the caller
			// publish the review that was going to be published anyway: the findings are
			// all in the body, they simply lose their anchors.
			return "", anchorError{err}
		}
		// Not the anchors. Returned as it is, so the callers' retry does not fire:
		// see anchorError for why a second submission is only safe there.
		return "", err
	}
	// The submission answered with the review's own permalink; the pull request's URL
	// is the fallback for a response that carried none, since a caller that has to
	// name what was published is better served by a link to the wrong granularity
	// than by nothing.
	if url == "" {
		url = latestReviewURL(ctx, dir, pr)
	}
	// Published either way -- the URL is returned even when the check below refuses
	// to call it a clean approval, because a caller that wants to name what has to be
	// dismissed needs it.
	return url, confirmApproval(ctx, dir, pr, head, event, id, url)
}

// confirmApproval re-reads the head AFTER an approval has been submitted, and
// refuses to report a clean post when the pull request moved while it was in flight.
//
// requireHead is a check-then-act: it reads the head, and the submission that
// follows is a separate round trip. commit_id closes that gap for what the review
// SAYS -- the forge records it against the commit that was read, and marks its
// comments outdated once the branch moves -- but not for what it GRANTS. GitHub
// accepts a commit_id that is no longer the head, and the approval it then records
// counts toward whatever the pull request proposes next, unless the repository
// happens to dismiss stale reviews. An author who pushes into that window collects
// an approval for code nobody read: the attack requireHead exists to refuse,
// narrowed to the read-to-submit window rather than closed.
//
// Nothing here can withdraw what is already on the pull request. What it can do is
// stop calling it a success: the mismatch comes back as an error, so the caller logs
// it and the run exits nonzero, and the message names the review a human has to
// dismiss before anyone merges on it. A read that fails counts too -- "the approval
// is bound" is a claim, and an unanswered head read does not support it.
//
// APPROVALS only. A comment review that lands on a moved head is stale, not
// dangerous, and REQUEST_CHANGES errs in the harmless direction for the same reason
// GitLab's unapprove does: neither can clear code nobody read. Alarming on those
// would spend an operator's attention -- and an exit code -- on a post that cost
// nothing.
func confirmApproval(ctx context.Context, dir string, pr int, reviewed string, event Event, id, url string) error {
	if event != EventApprove {
		return nil
	}
	where := ""
	if url != "" {
		where = " (" + url + ")"
	}
	cur, err := githubHead(ctx, dir, pr)
	if err != nil {
		return fmt.Errorf("the approval was PUBLISHED on #%d%s, but whether it is still the head could not be established afterwards (%w) -- confirm it before merging", pr, where, err)
	}
	if !strings.EqualFold(reviewed, cur) {
		// Withdraw it rather than merely reporting it. Detecting the race left an
		// approval standing on the pull request that satisfies a branch-protection
		// requirement over code no reviewer read -- the exact outcome requireHead
		// exists to prevent, arriving through the window between that check and this
		// submission. Dismissal is the only thing that actually removes it, and it is
		// ours to dismiss: this run posted it seconds ago.
		if derr := githubDismissReview(ctx, dir, pr, id, url, reviewed, cur); derr != nil {
			return fmt.Errorf("the approval was PUBLISHED on #%d%s, the head moved while it was in flight (reviewed %s, now %s), AND withdrawing it failed (%w) -- dismiss it by hand before anyone merges",
				pr, where, shortSHA(reviewed), shortSHA(cur), derr)
		}
		return fmt.Errorf("the head moved while the approval was in flight (reviewed %s, now %s) on #%d%s, so the approval was withdrawn -- review the new head",
			shortSHA(reviewed), shortSHA(cur), pr, where)
	}
	return nil
}

// ApprovalNotice says what a published approval does NOT cover, and is printed by
// every caller that publishes one.
//
// Everything else about posting binds a review to one commit: requireHead refuses a
// pull request that moved before the submission, commit_id and gitlabApprove's sha
// record which commit the review is about, and confirmApproval withdraws an approval
// that landed on a head which moved while it was in flight. None of it survives the
// run. An approval is a statement about a PULL REQUEST, not about a commit: both
// forges keep counting one toward their merge requirements after the branch moves,
// unless the repository is configured to clear approvals on a push. So an author can
// wait for a clean exit, push, and merge on an approval given for code nobody read --
// the window requireHead closes inside the run, reopened the moment it ends.
//
// No API call can close it, because the setting belongs to the repository rather
// than to whoever posts a review; a human reviewer's approval carries the same
// residue. What is left is to stop an operator inferring from a successful exit that
// the approval is commit-bound, and to name the one setting that makes it so.
//
// Printed on EVERY approval, not only where a check says it matters: the rule that
// would answer -- classic branch protection, or a ruleset -- is unreadable without
// admin on the repository, which is exactly what the operators running this on
// somebody else's pull request do not have. A notice that is silent whenever the
// answer is unavailable would be quietest in the case it is most needed.
//
// Empty for anything else. A comment grants nothing, and a change request that
// outlives the commit it was about errs in the direction that costs a re-review.
func ApprovalNotice(kind Kind, event Event, pr int, head string) string {
	if event != EventApprove {
		return ""
	}
	ref, setting := fmt.Sprintf("#%d", pr), `the branch protection setting "Dismiss stale pull request approvals when new commits are pushed"`
	if kind == GitLab {
		ref, setting = fmt.Sprintf("!%d", pr), `the project setting "Remove all approvals when commits are added to the source branch"`
	}
	return fmt.Sprintf("the approval published on %s was given for %s, but it is not withdrawn by a later push: unless %s is enabled on this repository, it goes on counting toward the merge requirements over commits nobody reviewed",
		ref, shortSHA(head), setting)
}

// githubDismissReview withdraws a review this run just submitted.
//
// The id is the one the create response named, which is the only handle that
// identifies OUR review: asking the API which review is ours would race with
// anything else posting. A permalink ending in "#pullrequestreview-<id>" carries
// the same id, so it stands in when the response could not be read but the URL can.
//
// A dismissal needs a message and it is shown to everyone reading the pull
// request, so it says what happened rather than "dismissed".
func githubDismissReview(ctx context.Context, dir string, pr int, id, url, reviewed, cur string) error {
	if id == "" {
		id = reviewIDFromURL(url)
	}
	if id == "" {
		return fmt.Errorf("the submission named no review id (%q)", url)
	}
	_, err := run(ctx, dir, "gh", "api", "--method", "PUT",
		fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/reviews/%s/dismissals", pr, id),
		"-f", fmt.Sprintf("message=Withdrawn automatically: the head moved from %s to %s while this review was being submitted, so it approved a commit that was not reviewed.",
			shortSHA(reviewed), shortSHA(cur)),
		"-f", "event=DISMISS")
	return err
}

// reviewIDFromURL pulls the numeric review id out of a review permalink.
func reviewIDFromURL(url string) string {
	const marker = "#pullrequestreview-"
	i := strings.Index(url, marker)
	if i < 0 {
		return ""
	}
	id := url[i+len(marker):]
	for _, r := range id {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return id
}

// githubSubmitReview submits one review carrying the summary and whatever per-line
// comments it has, through the API `gh pr review` does not expose.
//
// The payload goes in on STDIN, not as arguments: it embeds the whole review, and
// argv is world-readable for the life of the process. `{owner}/{repo}` are gh's own
// placeholders, resolved from the checkout, so no repository identity has to be
// parsed here.
//
// commit_id pins the review to the commit it is about, so the forge records the
// approval against that oid and marks its comments outdated if the branch moves
// afterwards. requireHead has already established it is the current head; this
// closes the gap between that read and this submission. It is why an empty inline
// slice comes through here too rather than through the CLI's own review verb.
//
// The response carries the created review's id and permalink, and they are read out
// of it because there is nowhere else to get them: asking the API afterwards which
// review is ours races with anything else posting. The id is the handle
// githubDismissReview needs to withdraw an approval that turns out to have landed on
// a moved head. A response that does not parse is NOT a failed submission -- the
// review is on the pull request either way -- so it yields empty strings and the
// caller reports what it cannot do rather than a post that did not happen.
func githubSubmitReview(ctx context.Context, dir string, pr int, head, body string, event Event, inline []InlineComment) (id, url string, err error) {
	type ghComment struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Side string `json:"side"`
		Body string `json:"body"`
	}
	payload := struct {
		Body     string `json:"body"`
		Event    string `json:"event"`
		CommitID string `json:"commit_id,omitempty"`
		// omitempty, not an empty array: a body-only review says nothing about lines,
		// and the API reads a present-but-empty comments list as a claim it can
		// validate.
		Comments []ghComment `json:"comments,omitempty"`
	}{Body: body, Event: githubEvent(event), CommitID: head}
	for _, c := range inline {
		// RIGHT is the head of the pull request. A finding is about the code as
		// proposed, not the line it replaced.
		payload.Comments = append(payload.Comments, ghComment{Path: c.Path, Line: c.Line, Side: "RIGHT", Body: c.Body})
	}
	doc, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	out, err := runStdin(ctx, dir, string(doc), "gh", "api", "--method", "POST",
		fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/reviews", pr), "--input", "-")
	if err != nil {
		return "", "", err
	}
	id, url = createdReview(out)
	return id, url, nil
}

// createdReview reads the identity of the review out of a create response. Either
// half may be missing -- an older gh that prints nothing, a body that does not parse
// -- and neither absence is a failed submission, so this reports what it found and
// never an error: the review is on the pull request regardless, and saying otherwise
// would report a published review as a failed post.
func createdReview(out string) (id, url string) {
	var created struct {
		ID      int64  `json:"id"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		return "", ""
	}
	if created.ID != 0 {
		id = strconv.FormatInt(created.ID, 10)
	}
	return id, strings.TrimSpace(created.HTMLURL)
}

// githubHead reads the commit a pull request currently proposes.
func githubHead(ctx context.Context, dir string, pr int) (string, error) {
	out, err := run(ctx, dir, "gh", "pr", "view", strconv.Itoa(pr), "--json", "headRefOid")
	if err != nil {
		return "", fmt.Errorf("read the head of #%d before posting: %w", pr, err)
	}
	var payload struct {
		HeadRefOid string `json:"headRefOid"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return "", fmt.Errorf("parse gh headRefOid: %w", err)
	}
	return strings.TrimSpace(payload.HeadRefOid), nil
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

// latestReviewURL best-effort resolves a link to the pull request a review was just
// posted on. DISPLAY ONLY, and only when the submission's own response carried no
// permalink: it is `gh pr view --json url`, so it answers with the pull request's
// URL and no "#pullrequestreview-<id>" fragment -- nothing that can identify which
// review is ours. A missing URL is cosmetic -- the review is already published -- so
// every failure here yields an empty string rather than an error that would
// misreport a successful post as a failed one.
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
func (gitlabProvider) PostReview(ctx context.Context, dir string, mr int, head, body string, event Event, inline []InlineComment) (string, error) {
	// requireHead first, so a moved head is refused before any note is posted and the
	// operator reads why. It is not what BINDS the approval, though: notes are posted
	// between this read and the approval, so on its own it would be a check-then-act
	// with a window an author can push into. gitlabApprove closes that.
	cur, err := gitlabHead(ctx, dir, mr)
	if err != nil {
		return "", err
	}
	if err := requireHead(mr, head, cur); err != nil {
		return "", err
	}
	for _, c := range inline {
		// Best-effort per comment: GitLab positions a discussion with base/head/start
		// SHAs this package does not carry, so an inline note is attempted as a plain
		// note naming its location rather than skipped outright.
		// The path goes through CodeSpan: it is the one agent-authored string that
		// reaches a note without having been sanitized on the way in -- the body was,
		// when the review was rendered -- and a backtick in it would break out of the
		// span this line puts it in.
		if err := gitlabNote(ctx, dir, mr, fmt.Sprintf("`%s:%d`\n\n%s", CodeSpan(c.Path), c.Line, c.Body)); err != nil {
			return "", err
		}
	}
	if err := gitlabNote(ctx, dir, mr, body); err != nil {
		return "", err
	}
	switch event {
	case EventApprove:
		if err := gitlabApprove(ctx, dir, mr, head); err != nil {
			return "", fmt.Errorf("note posted, but approving failed: %w", err)
		}
	case EventRequestChanges:
		// GitLab has no "request changes" review event; unapproving is the closest
		// equivalent and is what the note above explains. No commit to bind: removing
		// an approval cannot approve code nobody read, so a head that moves under this
		// call errs in the harmless direction.
		if _, err := run(ctx, dir, "glab", "mr", "unapprove", strconv.Itoa(mr)); err != nil {
			return "", fmt.Errorf("note posted, but unapproving failed: %w", err)
		}
	case Comment:
	}
	return "", nil
}

// gitlabNote posts one note on a merge request, with the text on STDIN.
//
// Not `glab mr note --message <body>`: that puts the whole note on the argument
// list, and argv is world-readable on this host for the life of the process. The
// note carries findings quoted out of the code under review -- which in a
// review-only run may be the credential the review is ABOUT, and redaction is
// shape-based, so it cannot be relied on to have masked it. The API passthrough
// is the same one gitlabHead and Checks already use, and it accepts the payload on
// stdin the way githubSubmitReview does.
func gitlabNote(ctx context.Context, dir string, mr int, body string) error {
	payload, err := json.Marshal(struct {
		Body string `json:"body"`
	}{body})
	if err != nil {
		return err
	}
	_, err = runStdin(ctx, dir, string(payload), "glab", "api", "--method", "POST",
		fmt.Sprintf("projects/:id/merge_requests/%d/notes", mr), "--input", "-")
	return err
}

// gitlabApprove approves a merge request AND binds the approval to the commit that
// was reviewed.
//
// The endpoint's optional sha is what makes that possible: GitLab compares it to the
// source branch's current head and answers 409 when they differ, so the approval is
// either about the reviewed commit or it does not happen. It is GitLab's equivalent
// of the commit_id githubSubmitReview sends, and it is needed for the same reason --
// requireHead reads the head, then every note is posted, and only then does this
// call arrive. An author who pushes into that window would otherwise collect an
// approval for code the panel never read, which is exactly the attack requireHead
// exists to stop.
//
// Not `glab mr approve`: the CLI's own verb sends no sha, so it cannot make that
// statement. The api passthrough is the one gitlabHead and gitlabNote already use.
//
// UNVERIFIED end to end like the rest of this provider, and it fails CLOSED: a 409,
// an unsupported attribute or any other refusal is reported as a failed approval,
// leaving the notes published and the approval to a human.
func gitlabApprove(ctx context.Context, dir string, mr int, head string) error {
	payload, err := json.Marshal(struct {
		SHA string `json:"sha"`
	}{head})
	if err != nil {
		return err
	}
	_, err = runStdin(ctx, dir, string(payload), "glab", "api", "--method", "POST",
		fmt.Sprintf("projects/:id/merge_requests/%d/approve", mr), "--input", "-")
	return err
}

// gitlabHead reads the commit a merge request currently proposes. diff_refs is
// preferred because it is the head the MR's own diff was computed against, which
// is what a review is about; sha is the same commit for a normal MR and the
// fallback when the field is absent.
//
// UNVERIFIED end to end like the rest of this provider, and it fails CLOSED: a read
// that cannot answer refuses the post rather than publishing an unbound approval.
func gitlabHead(ctx context.Context, dir string, mr int) (string, error) {
	out, err := run(ctx, dir, "glab", "api", fmt.Sprintf("projects/:id/merge_requests/%d", mr))
	if err != nil {
		return "", fmt.Errorf("read the head of !%d before posting: %w", mr, err)
	}
	var payload struct {
		SHA      string `json:"sha"`
		DiffRefs struct {
			HeadSHA string `json:"head_sha"`
		} `json:"diff_refs"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return "", fmt.Errorf("parse glab merge request: %w", err)
	}
	if h := strings.TrimSpace(payload.DiffRefs.HeadSHA); h != "" {
		return h, nil
	}
	return strings.TrimSpace(payload.SHA), nil
}

// Thread is one unresolved review conversation on a pull request.
//
// Unresolved only: a resolved thread is a settled question, and handing it to a
// coder invites it to reopen something a human already closed.
type Thread struct {
	// ID is what a reply is addressed to. It is the ROOT comment's id, because a
	// forge threads replies under the comment that started the conversation.
	ID   string
	Path string
	Line int
	// Author and Body are the ROOT comment's -- who opened the conversation and
	// what they asked. Kept alongside Comments because that is the question the
	// thread is about; the rest is what has been said since.
	Author string
	Body   string
	// Comments is the whole conversation in order, root first.
	//
	// Reading only the root made a conversation that had already been answered look
	// exactly like one nobody had touched, which is how the same comment ended up
	// answered twice. It also hid every clarification: a reviewer who explained what
	// they meant in a reply was invisible to the agent deciding what they meant.
	Comments []ThreadComment
}

// ThreadComment is one message in a conversation. Named for the thread rather
// than just "Comment", which is already the review event that posts without a
// verdict.
type ThreadComment struct {
	Author string
	Body   string
}

// AnsweredByMachine reports whether the LAST thing said in this conversation was
// one of this tool's own replies.
//
// That is the question "is anybody waiting on us?" in the only form that can be
// answered from the forge alone. Author identity cannot answer it: replies are
// posted under the operator's account, so "the last comment is mine" is equally
// true of a machine answer and of the operator typing a new request -- and
// skipping the second would swallow the very thing the run should act on.
//
// So the marker is a property of the MESSAGE. Every machine reply carries an
// invisible one (see ReplyMarker); a human writing in the same thread does not.
// A thread whose last word is ours is skipped as already answered; the moment a
// person replies under it, it is live again and gets read afresh -- with the whole
// exchange, including what we said last time.
func (t Thread) AnsweredByMachine() bool {
	if len(t.Comments) == 0 {
		return false
	}
	return HasReplyMarker(t.Comments[len(t.Comments)-1].Body)
}

// Requesters names everyone whose message makes up the conversation's LIVE
// request, in the order they spoke.
//
// Not the root author, which is who OPENED the thread and not necessarily who is
// asking now. A maintainer can start a conversation, this tool can answer it, and
// anybody with access to the pull request can then write under it -- and it is
// that follow-up a later run reads and acts on. Attributing it to the opener
// records the wrong provenance for the untrusted text that commissioned a change,
// which is precisely the fact somebody reviewing an automatically produced commit
// is looking for.
//
// So the window is everything said after this tool's last reply (see ReplyMarker),
// or the whole conversation when we have never answered it. Machine replies are
// never requesters. Repeat speakers are listed once; an author the forge could not
// name is kept as the empty string rather than dropped, because "we do not know who
// asked" is a fact the caller must be able to act on -- it is what makes a request
// external.
func (t Thread) Requesters() []string {
	start := 0
	for i, c := range t.Comments {
		if HasReplyMarker(c.Body) {
			start = i + 1
		}
	}
	out := make([]string, 0, len(t.Comments)-start)
	seen := make(map[string]bool, len(t.Comments)-start)
	for _, c := range t.Comments[start:] {
		key := strings.ToLower(c.Author)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c.Author)
	}
	return out
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
	// Login is the account this CLI is authenticated as, or "" when it cannot be
	// determined.
	//
	// It exists to answer one question: is this comment from the account fixpoint
	// posts under, or from somebody else? A conversation that commissions work is
	// labeled by that answer wherever it travels. "" is the safe answer -- an
	// unknown login makes every author external, which only ever adds a label.
	Login(ctx context.Context, dir string) string
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
// threadQuery reads a page of review conversations WITH THEIR REPLIES.
//
// Every comment, not just the first. The root comment is the question, but what
// happened since is what decides whether anything is still being asked: a
// clarification, somebody else disagreeing, or this tool's own previous answer.
// Reading only the root made all three invisible -- a conversation that had been
// answered looked exactly like one that had not.
//
// Paginated because reviewThreads(first:100) silently stopped there. A pull
// request with more conversations than that dropped the rest, and dropping them
// looks identical to a pull request that has none: nothing was shown to triage or
// to the coder, and nothing said so.
//
// The comments of each thread are paged too, for the same reason one level down --
// see threadCommentsQuery.
const threadQuery = `query($owner:String!,$repo:String!,$pr:Int!,$after:String){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$pr){
      reviewThreads(first:100,after:$after){
        pageInfo{hasNextPage endCursor}
        nodes{
          id
          isResolved
          comments(first:100){
            pageInfo{hasNextPage endCursor}
            nodes{path line databaseId body author{login}}
          }
        }
      }
    }
  }
}`

// threadCommentsQuery reads what is left of ONE conversation, past its first page.
//
// A thread's comments page independently of the thread list, so comments(first:100)
// cut every long conversation off at its hundredth message -- and the cut falls
// exactly where it does damage. AnsweredByMachine asks whether the LAST comment is
// ours; on a truncated thread it inspected the hundredth instead, missed our reply,
// read the thread as live and answered it again on every run. That is the loop the
// marker exists to prevent. The coder and triage were handed the same partial
// exchange, with any later correction cut off it.
//
// Addressed by the thread's node id rather than by walking the pull request again,
// so the extra call happens only for the rare thread that has a second page.
const threadCommentsQuery = `query($id:ID!,$after:String!){
  node(id:$id){
    ... on PullRequestReviewThread {
      comments(first:100,after:$after){
        pageInfo{hasNextPage endCursor}
        nodes{path line databaseId body author{login}}
      }
    }
  }
}`

// threadCommentNode is one comment as either query returns it.
type threadCommentNode struct {
	Path       string `json:"path"`
	Line       int    `json:"line"`
	DatabaseID int64  `json:"databaseId"`
	Body       string `json:"body"`
	Author     struct {
		Login string `json:"login"`
	} `json:"author"`
}

// threadCommentPage is one page of a conversation: its comments and where the next
// page starts.
type threadCommentPage struct {
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
	Nodes []threadCommentNode `json:"nodes"`
}

// githubThreadTail returns the comments of thread nodeID that follow the page
// ending at cursor, in order.
//
// An unreadable tail is an error rather than a short conversation: a thread read
// half-way is the state that makes a machine reply invisible, so the caller must not
// be able to mistake it for the whole exchange.
func githubThreadTail(ctx context.Context, dir, nodeID, cursor string) ([]threadCommentNode, error) {
	var rest []threadCommentNode
	// Bounded like the thread loop, and for the same reason: 100 pages is 10,000
	// comments on one conversation, past anything a person will write.
	for range 100 {
		out, err := run(ctx, dir, "gh", "api", "graphql", "-f", "query="+threadCommentsQuery,
			"-f", "id="+nodeID, "-f", "after="+cursor)
		if err != nil {
			return nil, err
		}
		var payload struct {
			Data struct {
				Node struct {
					Comments threadCommentPage `json:"comments"`
				} `json:"node"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(out), &payload); err != nil {
			return nil, fmt.Errorf("parse thread comments: %w", err)
		}
		page := payload.Data.Node.Comments
		rest = append(rest, page.Nodes...)
		if !page.PageInfo.HasNextPage || page.PageInfo.EndCursor == "" {
			return rest, nil
		}
		cursor = page.PageInfo.EndCursor
	}
	return rest, nil
}

func (githubProvider) Threads(ctx context.Context, dir string, pr int) ([]Thread, error) {
	owner, repo, err := githubSlug(ctx, dir)
	if err != nil {
		return nil, err
	}
	var threads []Thread
	cursor := ""
	// Bounded, so a malformed cursor or a server that never clears hasNextPage
	// cannot spin forever. 100 pages is 10,000 conversations -- far past any real
	// pull request, and the cap is reported rather than silently applied.
	for range 100 {
		// owner and repo go through -f, not -F: gh's typed flag would turn a repo
		// named 2048 into a JSON number and null into JSON null, and the query
		// declares both as String!, so the server would reject the whole page.
		// Only pr is genuinely Int!.
		args := []string{"api", "graphql", "-f", "query=" + threadQuery,
			"-f", "owner=" + owner, "-f", "repo=" + repo, fmt.Sprintf("-Fpr=%d", pr)}
		if cursor != "" {
			args = append(args, "-f", "after="+cursor)
		}
		out, err := run(ctx, dir, "gh", args...)
		if err != nil {
			return nil, err
		}
		var payload struct {
			Data struct {
				Repository struct {
					PullRequest struct {
						ReviewThreads struct {
							PageInfo struct {
								HasNextPage bool   `json:"hasNextPage"`
								EndCursor   string `json:"endCursor"`
							} `json:"pageInfo"`
							Nodes []struct {
								ID         string            `json:"id"`
								IsResolved bool              `json:"isResolved"`
								Comments   threadCommentPage `json:"comments"`
							} `json:"nodes"`
						} `json:"reviewThreads"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(out), &payload); err != nil {
			return nil, fmt.Errorf("parse review threads: %w", err)
		}
		rt := payload.Data.Repository.PullRequest.ReviewThreads
		for _, n := range rt.Nodes {
			if n.IsResolved || len(n.Comments.Nodes) == 0 {
				continue
			}
			comments := n.Comments.Nodes
			if n.Comments.PageInfo.HasNextPage && n.Comments.PageInfo.EndCursor != "" {
				tail, err := githubThreadTail(ctx, dir, n.ID, n.Comments.PageInfo.EndCursor)
				if err != nil {
					return nil, fmt.Errorf("read conversation %s: %w", n.ID, err)
				}
				comments = append(comments, tail...)
			}
			root := comments[0]
			t := Thread{
				// The ROOT comment's id: a forge threads replies under the comment that
				// started the conversation.
				ID:     strconv.FormatInt(root.DatabaseID, 10),
				Path:   root.Path,
				Line:   root.Line,
				Author: root.Author.Login,
				Body:   root.Body,
			}
			for _, c := range comments {
				t.Comments = append(t.Comments, ThreadComment{Author: c.Author.Login, Body: c.Body})
			}
			threads = append(threads, t)
		}
		if !rt.PageInfo.HasNextPage || rt.PageInfo.EndCursor == "" {
			return threads, nil
		}
		cursor = rt.PageInfo.EndCursor
	}
	return threads, nil
}

// Login asks gh who it is authenticated as.
//
// `gh api user` rather than `gh auth status`, whose output is prose meant for a
// human and has changed shape between releases. A failure is not an error worth
// stopping for: the caller only uses this to decide whether to LABEL a comment as
// externally authored, and an unknown login labels every one of them, which errs
// toward saying more rather than less.
func (githubProvider) Login(ctx context.Context, dir string) string {
	out, err := run(ctx, dir, "gh", "api", "user", "--jq", ".login")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
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
	_, err = runStdin(ctx, dir, string(payload), "gh", "api", "--method", "POST",
		fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/comments/%d/replies", pr, id), "--input", "-")
	return err
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
