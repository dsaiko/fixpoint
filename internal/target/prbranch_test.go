package target

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/agent"
	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/testfixture"
)

// stubGH puts a `gh` on PATH that replies from files, so the resolution can be
// driven through the production code path -- runInput, the hardened environment,
// the real argv -- rather than by calling the parsers directly. view/list hold
// stdout; view.status makes `gh pr view` fail the way an absent pull request
// does. PATH is PREPENDED, not replaced: git has to stay real, since the branch
// half of the answer comes from an actual repository.
func stubGH(t *testing.T, view, viewErr, list string, viewStatus int) {
	t.Helper()
	resp := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(resp, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("view", view)
	write("view.err", viewErr)
	write("list", list)

	bin := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = pr ] && [ \"$2\" = view ]; then\n" +
		"  cat " + resp + "/view\n" +
		"  cat " + resp + "/view.err >&2\n" +
		"  exit " + strconv.Itoa(viewStatus) + "\n" +
		"fi\n" +
		"if [ \"$1\" = pr ] && [ \"$2\" = list ]; then cat " + resp + "/list; exit 0; fi\n" +
		"echo \"unexpected gh call: $*\" >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// onBranch is a repository with a named branch checked out, which is what the
// resolution reads before it ever calls gh.
func onBranch(t *testing.T) string {
	t.Helper()
	dir := testfixture.GitRepo(t)
	testfixture.GitRun(t, dir, "checkout", "-q", "-b", branchUnderTest)
	return dir
}

// branchUnderTest is the branch every case below checks out. It is a constant so
// the name in a stub's JSON and the name on disk cannot drift apart.
const branchUnderTest = "feat/x"

func resolve(t *testing.T, dir string) (BranchPR, error) {
	t.Helper()
	return ResolvePRFromBranch(t.Context(), config.Target{Mode: config.ModePR, Path: dir},
		agent.EnvWithoutCredentials(nil))
}

// The whole point of the feature: on the branch a pull request is open from,
// nobody has to type its number.
func TestResolvePRFromBranchTakesTheBranchesPR(t *testing.T) {
	dir := onBranch(t)
	stubGH(t,
		`{"number":170,"state":"OPEN","url":"https://github.com/o/r/pull/170","baseRefName":"main","headRefName":"feat/x","headRepositoryOwner":{"login":"o"}}`,
		"", `[{"number":170,"baseRefName":"main","headRepositoryOwner":{"login":"o"}}]`, 0)

	got, err := resolve(t, dir)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Number != 170 || got.Branch != "feat/x" || got.Base != "main" {
		t.Errorf("got %+v, want #170 from feat/x into main", got)
	}
	if got.URL == "" {
		t.Error("the URL is empty; it is the only thing the log line can show for a number nobody typed")
	}
}

// gh picks one silently when a head branch has several open pull requests --
// same head, different bases. Acting on that guess would put a review, and with
// -post a verdict, on whichever one it happened to list first.
func TestResolvePRFromBranchRefusesAmbiguity(t *testing.T) {
	dir := onBranch(t)
	stubGH(t,
		`{"number":170,"state":"OPEN","url":"u","baseRefName":"main","headRefName":"feat/x","headRepositoryOwner":{"login":"o"}}`,
		"", `[{"number":170,"baseRefName":"main","headRepositoryOwner":{"login":"o"}},
		      {"number":171,"baseRefName":"release","headRepositoryOwner":{"login":"o"}}]`, 0)

	_, err := resolve(t, dir)
	if err == nil {
		t.Fatal("an ambiguous branch resolved; one of the two pull requests would have been reviewed by chance")
	}
	for _, want := range []string{"#170", "#171", "-pr"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not mention %q: %v", want, err)
		}
	}
}

// Two forks can hold the same branch NAME, and `gh pr list --head` matches the
// name alone. Those are different branches, gh's own resolution tells them apart
// by head owner, and refusing them would break the fork workflow this is most
// useful in.
func TestResolvePRFromBranchIgnoresAnotherForksSameName(t *testing.T) {
	dir := onBranch(t)
	stubGH(t,
		`{"number":170,"state":"OPEN","url":"u","baseRefName":"main","headRefName":"feat/x","headRepositoryOwner":{"login":"mine"}}`,
		"", `[{"number":170,"baseRefName":"main","headRepositoryOwner":{"login":"mine"}},
		      {"number":900,"baseRefName":"main","headRepositoryOwner":{"login":"someone-else"}}]`, 0)

	got, err := resolve(t, dir)
	if err != nil {
		t.Fatalf("a same-named branch on another fork was treated as ambiguity: %v", err)
	}
	if got.Number != 170 {
		t.Errorf("resolved #%d, want #170 -- the head owner decides which branch is this one", got.Number)
	}
}

// gh answers with a merged or closed pull request when that is all the branch
// has. Reviewing merged work by inference is never what the command meant, and a
// fix run would commit onto it.
func TestResolvePRFromBranchRefusesAClosedPR(t *testing.T) {
	for _, state := range []string{"MERGED", "CLOSED"} {
		t.Run(state, func(t *testing.T) {
			dir := onBranch(t)
			stubGH(t, `{"number":170,"state":"`+state+`","url":"u","baseRefName":"main","headRefName":"feat/x","headRepositoryOwner":{"login":"o"}}`,
				"", `[]`, 0)

			_, err := resolve(t, dir)
			if err == nil {
				t.Fatalf("a %s pull request was accepted", state)
			}
			// The number is in the refusal so an operator who does mean it can say so.
			if !strings.Contains(err.Error(), "#170") || !strings.Contains(err.Error(), "-pr 170") {
				t.Errorf("the refusal does not name the number to pass explicitly: %v", err)
			}
		})
	}
}

// A branch with no pull request is the ordinary mistake -- the work is not up
// yet -- and the message has to say what to do rather than surface gh's own
// wording alone.
func TestResolvePRFromBranchReportsNoPullRequest(t *testing.T) {
	dir := onBranch(t)
	stubGH(t, "", "no pull requests found for branch \"feat/x\"\n", `[]`, 1)

	_, err := resolve(t, dir)
	if err == nil {
		t.Fatal("a branch with no pull request resolved")
	}
	if !strings.Contains(err.Error(), "feat/x") || !strings.Contains(err.Error(), "-pr") {
		t.Errorf("the refusal names neither the branch nor the flag: %v", err)
	}
}

// A detached HEAD names no branch. `rev-parse --abbrev-ref HEAD` would answer
// the literal string "HEAD" here and send the resolution looking for a pull
// request whose head branch is called HEAD.
func TestResolvePRFromBranchRefusesDetachedHEAD(t *testing.T) {
	dir := onBranch(t)
	testfixture.GitRun(t, dir, "checkout", "-q", "--detach")
	stubGH(t, `{"number":1,"state":"OPEN","url":"u","baseRefName":"main","headRefName":"HEAD","headRepositoryOwner":{"login":"o"}}`,
		"", `[]`, 0)

	_, err := resolve(t, dir)
	if err == nil {
		t.Fatal("a detached checkout resolved a pull request")
	}
	if !strings.Contains(err.Error(), "not on a branch") {
		t.Errorf("the refusal does not say why: %v", err)
	}
}

func TestResolvePRFromBranchRefusesANonRepository(t *testing.T) {
	stubGH(t, "", "", `[]`, 0)
	if _, err := resolve(t, t.TempDir()); err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("a directory that is not a repository gave %v", err)
	}
}

// The probe is what proves the number is safe to act on, so "could not tell" is
// not "unique": a failed listing refuses, and names the number it could not
// prove so the operator can pass it.
func TestResolvePRFromBranchRefusesWhenTheProbeFails(t *testing.T) {
	dir := onBranch(t)
	stubGH(t, `{"number":170,"state":"OPEN","url":"u","baseRefName":"main","headRefName":"feat/x","headRepositoryOwner":{"login":"o"}}`,
		"", "not json at all", 0)

	_, err := resolve(t, dir)
	if err == nil {
		t.Fatal("an unprovable resolution was acted on")
	}
	if !strings.Contains(err.Error(), "-pr") {
		t.Errorf("the refusal does not tell the operator what to do: %v", err)
	}
}
