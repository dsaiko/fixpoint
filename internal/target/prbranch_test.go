package target

import (
	"fmt"
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
	c := New(config.Target{Mode: config.ModePR, Path: dir})
	c.UseGitEnv(agent.EnvWithoutCredentials(nil))
	return c.ResolvePRFromBranch(t.Context())
}

// openPR renders one `gh pr list` row.
func openPR(number int, owner, base string) string {
	return fmt.Sprintf(`{"number":%d,"baseRefName":%q,"headRepositoryOwner":{"login":%q}}`, number, base, owner)
}

// viewPR renders a `gh pr view` answer.
func viewPR(number int, state, owner, head, base string) string {
	return fmt.Sprintf(`{"number":%d,"state":%q,"url":"https://example.test/pull/%d","baseRefName":%q,"headRefName":%q,"headRepositoryOwner":{"login":%q}}`,
		number, state, number, base, head, owner)
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

// stubGHRecording is stubGH plus a record of every argv gh was called with. A
// stub that ignores its arguments would keep passing if the probe asked about the
// wrong head, the wrong state, or nothing at all (review run 20260909-213147,
// finding i16).
func stubGHRecording(t *testing.T, view, list string) string {
	t.Helper()
	resp := t.TempDir()
	calls := filepath.Join(resp, "calls")
	for name, content := range map[string]string{"view": view, "list": list} {
		if err := os.WriteFile(filepath.Join(resp, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bin := t.TempDir()
	script := `#!/bin/sh
printf '%s\n' "$*" >> ` + calls + `
if [ "$1" = pr ] && [ "$2" = view ]; then cat ` + resp + `/view; exit 0; fi
if [ "$1" = pr ] && [ "$2" = list ]; then cat ` + resp + `/list; exit 0; fi
echo "unexpected gh call: $*" >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return calls
}

func ghArgv(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("gh was never called: %v", err)
	}
	return string(b)
}

// The probe has to ask about the head gh ANSWERED with, not the branch on disk:
// gh resolves through a branch's push configuration, so the two can differ, and
// asking about a name no pull request has would prove the uniqueness of nothing.
func TestResolvePRFromBranchProbesTheResolvedHead(t *testing.T) {
	dir := onBranch(t)
	calls := stubGHRecording(t,
		viewPR(170, "OPEN", "o", "their-name-for-it", "main"),
		"["+openPR(170, "o", "main")+"]")

	got, err := resolve(t, dir)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Number != 170 {
		t.Errorf("resolved #%d, want #170", got.Number)
	}
	argv := ghArgv(t, calls)
	if !strings.Contains(argv, "--head their-name-for-it") {
		t.Errorf("the probe did not ask about the head gh resolved:\n%s", argv)
	}
	if strings.Contains(argv, "--head "+branchUnderTest) {
		t.Errorf("the probe asked about the local branch name instead:\n%s", argv)
	}
	// Only OPEN pull requests are siblings; counting merged ones would refuse a
	// perfectly unambiguous branch.
	if !strings.Contains(argv, "--state open") {
		t.Errorf("the probe did not restrict the listing to open pull requests:\n%s", argv)
	}
}

// Counting alone accepted listings that could not prove anything. Every case here
// used to pass (review run 20260909-213147, findings i17, i19, and the
// truncation medium).
func TestResolvePRFromBranchRefusesAnUnprovableListing(t *testing.T) {
	many := make([]string, 0, prBranchCandidates)
	for i := range prBranchCandidates {
		// Another fork's same-named branch: filtered out by owner, but it still fills
		// the limit, so a sibling of OURS could be the row that did not fit.
		many = append(many, openPR(900+i, "someone-else", "main"))
	}
	cases := []struct {
		name, list, want string
	}{
		{"empty", "[]", "appears 0 times"},
		{"without the resolved PR", "[" + openPR(171, "o", "main") + "]", "appears 0 times"},
		{"the resolved PR twice", "[" + openPR(170, "o", "main") + "," + openPR(170, "o", "release") + "]", "appears 2 times"},
		{"truncated", "[" + strings.Join(many, ",") + "]", "truncated"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := onBranch(t)
			stubGH(t, viewPR(170, "OPEN", "o", branchUnderTest, "main"), "", tc.list, 0)

			_, err := resolve(t, dir)
			if err == nil {
				t.Fatal("a listing that proves nothing was accepted as proof of uniqueness")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal does not say what was wrong (want %q): %v", tc.want, err)
			}
			if !strings.Contains(err.Error(), "-pr") {
				t.Errorf("the refusal does not tell the operator what to do: %v", err)
			}
		})
	}
}

// The branch is read before gh is asked, and gh consults the checkout again on
// its own, so the answer is only about this branch if the branch stayed put. The
// stub switches it mid-resolution, which is what a concurrent fixpoint run -- or
// the operator -- does (review run 20260909-213147, finding i10).
func TestResolvePRFromBranchRefusesWhenTheBranchMovesUnderIt(t *testing.T) {
	dir := onBranch(t)
	resp := t.TempDir()
	if err := os.WriteFile(filepath.Join(resp, "view"),
		[]byte(viewPR(170, "OPEN", "o", branchUnderTest, "main")), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := `#!/bin/sh
if [ "$2" = view ]; then cat ` + resp + `/view; exit 0; fi
if [ "$2" = list ]; then
  git checkout -q -b somewhere-else
  printf '%s' '[` + openPR(170, "o", "main") + `]'
  exit 0
fi
exit 1
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := resolve(t, dir)
	if err == nil {
		t.Fatal("the resolution stood on a branch nobody is on any more")
	}
	if !strings.Contains(err.Error(), "somewhere-else") || !strings.Contains(err.Error(), branchUnderTest) {
		t.Errorf("the refusal does not name both branches: %v", err)
	}
}
