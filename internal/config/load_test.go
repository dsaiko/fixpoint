package config

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// The shipped fix configs take their language-shaped keys from gates/go.yaml and
// everything else from defaults.yaml, and both halves have to actually arrive.
// `extends` merges PER KEY, so a decode that replaced a block wholesale would
// blank max_iterations, clean_rounds_to_stop and commit_policy -- fix-pr is the
// config that still sets one `loop:` key (max_iterations: 3) over the base, and
// so the one that exercises that merge; fix-code and fix-branch set none, and
// prove the base arrives untouched. The skip globs come from the GATE now, not
// the task config, and Source.Gate has to say so.
//
// Validate is deliberately not called: it checks that every agent's binary is on
// PATH, which says nothing about inheritance and would make this fail on a machine
// that simply has no agent CLIs installed.
func TestShippedFixConfigsInheritLoopAndTakeTheirGlobsFromTheGate(t *testing.T) {
	wantIterations := map[string]int{"fix-code": 5, "fix-branch": 5, "fix-pr": 3}
	for _, name := range []string{"fix-code", "fix-branch", "fix-pr"} {
		t.Run(name, func(t *testing.T) {
			l, err := LoadBundle(&Resolver{Bundles: []string{"../../config"}}, name, "", Overrides{PR: 1})
			if err != nil {
				t.Fatal(err)
			}
			if l.Config.Verify.Gate != "go" || !strings.HasSuffix(l.Source.Gate, filepath.Join(gatesDir, "go"+configExt)) {
				t.Errorf("gate = %q from %q, want go from gates/go.yaml", l.Config.Verify.Gate, l.Source.Gate)
			}
			loop := l.Config.Loop
			if got := loop.FinalSkipRunEdits; len(got) != 1 || got[0] != "**/*_test.go" {
				t.Errorf("final_skip_run_edits = %v, want [**/*_test.go] from the go gate", got)
			}
			if loop.MaxIterations != wantIterations[name] {
				t.Errorf("max_iterations = %d, want %d", loop.MaxIterations, wantIterations[name])
			}
			if loop.MaxFinalPasses != 1 {
				t.Errorf("max_final_passes = %d, want 1 inherited from defaults.yaml", loop.MaxFinalPasses)
			}
			if loop.CleanRoundsToStop != 2 {
				t.Errorf("clean_rounds_to_stop = %d, want 2 inherited from defaults.yaml", loop.CleanRoundsToStop)
			}
			if loop.CommitPolicy != CommitPerFix {
				t.Errorf("commit_policy = %q, want %q inherited from defaults.yaml", loop.CommitPolicy, CommitPerFix)
			}
		})
	}
}

// Every shipped gate file is decoded here, the way the loader decodes it, with the
// rules Config.Validate applies to what it inlines. Otherwise six of the seven ship
// untouched by any test -- only go.yaml is reached, through fix-code -- and a typo'd
// key, a duplicate name or an empty `run:` surfaces for the first time on an
// operator's machine, mid-invocation, after they pointed a fix run at a real
// project with `-gate rust`. Keyed by name, so a new gate has to be registered here
// -- the same guard bundle_agents_test.go gives the agent files.
func TestShippedGatesDecodeAndCarryUsableCommands(t *testing.T) {
	want := map[string]bool{"go": true, "rust": true, "node": true, "java": true, "python": true, "dotnet": true, "cpp": true}
	paths, err := filepath.Glob(filepath.Join("..", "..", projectBundleDir, gatesDir, "*"+configExt))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), configExt)
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			if !want[name] {
				t.Fatalf("gates/%s%s is not registered in this test -- add it, and decide what it should look like", name, configExt)
			}
			var g Gate
			if err := decodeInto(path, &g); err != nil {
				t.Fatal(err)
			}
			if len(g.Commands) == 0 {
				t.Fatal("no commands; the loader refuses an empty gate, so this file could never be used")
			}
			names := map[string]bool{}
			for i, c := range g.Commands {
				if c.Name == "" {
					t.Errorf("commands[%d] has no name", i)
				}
				if names[c.Name] {
					t.Errorf("commands[%d] duplicates name %q", i, c.Name)
				}
				names[c.Name] = true
				if len(c.Run) == 0 {
					t.Errorf("commands[%s].run is empty", c.Name)
				}
			}
			if len(g.SkipRunEdits) == 0 {
				t.Error("no skip_run_edits: the closing round would review the test files this run wrote, which is the measured non-convergence the key exists for")
			}
			// The same rule Config.Validate applies to loop.final_skip_run_edits,
			// which these are inlined into: a blank pattern compiles to ^$ and sits
			// in the config looking like an active rule.
			for i, glob := range g.SkipRunEdits {
				if strings.TrimSpace(glob) == "" {
					t.Errorf("skip_run_edits[%d] is blank; Validate would refuse the inlined list at load time, on an operator's machine", i)
				}
			}
			// The two keys are the file's whole reason to exist, and the rule they
			// stand on -- Validate's, on the inlined result -- is applied here too.
			v := Verify{Policy: VerifyNoRegressions, Timeout: Duration(time.Minute), Commands: g.Commands}
			if err := v.validate(); err != nil {
				t.Errorf("Validate would refuse the inlined gate: %v", err)
			}
		})
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("gates/%s%s is registered here but not shipped", name, configExt)
		}
	}
}

// The verb in a config's name is a SAFETY claim: a `review-` config never invokes
// the coder, so it cannot modify a file. That claim is what lets review-code and
// review-branch run without the fix-round trust gate, and what makes review-pr
// safe to point at externally-authored code. Nothing in the code enforces it --
// loop.review_only is an ordinary boolean -- so a new review- config that simply
// forgets the line would quietly gain the ability to edit the target.
//
// The strategy assertion is a weaker claim but the same kind: a review run has one
// round and nothing after it, so a lens seen by only one model is a lens whose
// blind spot nothing catches.
func TestShippedReviewConfigsAreReviewOnlyAndFanOutToTheWholePanel(t *testing.T) {
	names, err := filepath.Glob(filepath.Join("..", "..", projectBundleDir, "review-*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 3 {
		t.Fatalf("found %d review-* configs, want at least review-code, review-branch and review-pr: %v", len(names), names)
	}
	for _, path := range names {
		name := strings.TrimSuffix(filepath.Base(path), ".yaml")
		t.Run(name, func(t *testing.T) {
			l, err := LoadBundle(&Resolver{Bundles: []string{filepath.Join("..", "..", projectBundleDir)}}, name, "", Overrides{})
			if err != nil {
				t.Fatal(err)
			}
			if !l.Config.Loop.ReviewOnly {
				t.Errorf("%s does not set loop.review_only: a review- config that invokes the coder can modify the target, which is the one thing its name promises it will not do", name)
			}
			if l.Config.Roles.Review.Strategy != StrategyAll {
				t.Errorf("%s strategy = %q, want %q: a one-round review has nothing after it to catch what a single model missed",
					name, l.Config.Roles.Review.Strategy, StrategyAll)
			}
		})
	}
}

// A review- config must not have to name a coder. It never invokes one, and naming
// it would put a write-capable agent in a configuration whose entire promise is
// that nothing modifies the target -- leaving a reader to work out from the loop
// settings that it never runs.
func TestShippedReviewConfigsDeclareNoCoder(t *testing.T) {
	names, err := filepath.Glob(filepath.Join("..", "..", projectBundleDir, "review-*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range names {
		name := strings.TrimSuffix(filepath.Base(path), ".yaml")
		t.Run(name, func(t *testing.T) {
			l, err := LoadBundle(&Resolver{Bundles: []string{filepath.Join("..", "..", projectBundleDir)}}, name, "", Overrides{})
			if err != nil {
				t.Fatal(err)
			}
			if a := l.Config.Roles.Coder.Agent; a != "" {
				t.Errorf("%s names coder %q; a review config invokes none", name, a)
			}
			// The judge takes its place, and must be read-only.
			j := l.Config.Roles.Judge
			if j.Agent == "" {
				t.Fatalf("%s has neither a coder nor a judge; nothing filters its findings", name)
			}
			if l.Config.Agents[j.Agent].CanEdit {
				t.Errorf("%s judge %q is write-capable", name, j.Agent)
			}
		})
	}
}

// An agent used ONLY as the judge must be resolved from its own
// agents/<name>.yaml like every other role's agent. While referencedAgents left
// the judge out, that file was never read and Validate then rejected the judge as
// undefined -- so the only judge a config could actually declare was an inline
// one, which is also the form that skipped the common agent check.
func TestJudgeAgentIsResolvedFromItsOwnFile(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"task": "target: {mode: directory}\n" + taskBody + "  judge: {agent: arbiter, prompt: judge}\n",
	}, []string{"fix", "review-bugs", "judge"}, []string{"mock"})
	// Not via bundle's agent helper: that writes can_edit: true, which the judge
	// must not have.
	if err := os.WriteFile(filepath.Join(dir, agentsDir, "arbiter"+configExt), []byte("command: [true]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{})
	if err != nil {
		t.Fatalf("LoadBundle() = %v, want a judge declared in agents/arbiter.yaml to load", err)
	}
	a, ok := l.Config.Agents["arbiter"]
	if !ok {
		t.Fatalf("agents = %v, want the judge's own declaration resolved", l.Config.Agents)
	}
	if len(a.Command) != 1 || a.Command[0] != "true" {
		t.Errorf("agents.arbiter.command = %v, want the file's [true]", a.Command)
	}
	if l.Config.Roles.Judge.PromptPath == "" {
		t.Error("roles.judge.prompt_path is empty; the judge prompt was not resolved")
	}
}

// A document target is read whole and shown to every agent, so a symlink is an
// exfiltration primitive: an untrusted checkout ships `DESIGN.md ->
// ~/.aws/credentials` and the panel reads the destination. Refused at the flag
// (review run 20260813-124710).
func TestTargetOverrideRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "DESIGN.md")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatal(err)
	}
	var cfg Config
	err := Overrides{Target: link}.applyTarget(&cfg)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("applyTarget(symlink) = %v, want a refusal naming the symlink", err)
	}
	if cfg.Target.Document != "" {
		t.Errorf("the symlink was accepted as a document: %q", cfg.Target.Document)
	}
}

// The leaf rule is not enough: Lstat resolves every parent before it stats the
// leaf, so a checkout shipping `assignment -> ../../.aws` and a documented
// `-target assignment/credentials` passed -- the leaf really is a regular file
// -- while the bytes handed to every agent were the operator's cloud keys
// (review run 20260813-180828).
//
// The check is UNCONDITIONAL. An earlier version ran only when the target lay
// under the working directory, on the reasoning that an absolute path elsewhere
// was named on purpose; three reviewers pointed out that this conflates "the
// operator named the destination" with "the target is outside cwd", and that
// the case the rule exists for is a path inside an UNTRUSTED checkout which
// need not sit under cwd at all (review run 20260814-024946). Both shapes are
// asserted here, from a working directory that has nothing to do with either.
func TestTargetOverrideRefusesAPathThatEscapesThroughASymlinkedParent(t *testing.T) {
	secrets := filepath.Join(t.TempDir(), "dotaws")
	if err := os.MkdirAll(secrets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secrets, "credentials"), []byte("[default]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	checkout := filepath.Join(t.TempDir(), "checkout")
	if err := os.MkdirAll(filepath.Join(checkout, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secrets, filepath.Join(checkout, "assignment")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	escaping := filepath.Join(checkout, "assignment", "credentials")

	// Somewhere with no relationship to the checkout: the operator is sitting in
	// their own tree and naming an absolute path into someone else's.
	t.Chdir(t.TempDir())
	var cfg Config
	err := Overrides{Target: escaping}.applyTarget(&cfg)
	if err == nil {
		t.Fatalf("an absolute path escaping through a symlinked parent was accepted; target.path = %q", cfg.Target.Path)
	}
	if !strings.Contains(err.Error(), "leaves the directory it sits in") {
		t.Errorf("the refusal must name the link that left its tree: %v", err)
	}

	// From a project SUBDIRECTORY, reaching back out: outside cwd, inside the
	// project, and the shape the cwd precondition let through.
	t.Chdir(filepath.Join(checkout, "src"))
	var sub Config
	if err := (Overrides{Target: escaping}).applyTarget(&sub); err == nil {
		t.Errorf("a relative escape from a project subdirectory was accepted; target.path = %q", sub.Target.Path)
	}

	// The rule is about ESCAPE, not about symlinks: a link that stays inside the
	// directory it sits in redirects nothing the operator did not already name,
	// and refusing every symlinked ancestor would refuse every path on macOS,
	// where /var is itself a link to private/var.
	inside := filepath.Join(checkout, "docs")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inside, "DESIGN.md"), []byte("# d\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(inside, filepath.Join(checkout, "linked")); err != nil {
		t.Fatal(err)
	}
	var ok Config
	if err := (Overrides{Target: filepath.Join(checkout, "linked", "DESIGN.md")}).applyTarget(&ok); err != nil {
		t.Errorf("a symlink that stays inside the tree was refused: %v", err)
	}
}

// The go gate's fmt check is the one shipped command whose failure semantics are
// hand-rolled shell, and it has regressed once already: the first version decided
// purely from gofmt's stdout, so a missing gofmt -- or one that exits 2 on a file
// it cannot parse -- left the check green. Decoding the file proves nothing about
// that; this runs the argv the gate actually ships, read from the file rather than
// restated, against the three outcomes that matter.
func TestShippedGoGateFmtCommandFailsForTheRightReasons(t *testing.T) {
	if _, err := exec.LookPath("gofmt"); err != nil {
		t.Skip("gofmt not on PATH")
	}
	var g Gate
	if err := decodeInto(filepath.Join("..", "..", projectBundleDir, gatesDir, "go"+configExt), &g); err != nil {
		t.Fatal(err)
	}
	var fmtCmd []string
	for _, c := range g.Commands {
		if c.Name == "fmt" {
			fmtCmd = c.Run
		}
	}
	if len(fmtCmd) < 3 || fmtCmd[0] != "sh" {
		t.Fatalf("go gate has no `fmt` command of the shape [sh -c ...]: %v", fmtCmd)
	}
	run := func(t *testing.T, dir string, env []string) (int, string) {
		t.Helper()
		// /bin/sh by absolute path, so the PATH the case sets governs only what the
		// wrapper looks up (gofmt) and not whether the wrapper can start at all.
		cmd := exec.Command("/bin/sh", fmtCmd[1:]...)
		cmd.Dir, cmd.Env = dir, env
		out, err := cmd.CombinedOutput()
		if err == nil {
			return 0, string(out)
		}
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("running the fmt check: %v\n%s", err, out)
		}
		return exit.ExitCode(), string(out)
	}
	clean := "package main\n\nfunc main() {}\n"
	messy := "package main\n\nfunc   main( ) {\n}\n"

	t.Run("formatted tree passes", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte(clean), 0o600); err != nil {
			t.Fatal(err)
		}
		if code, out := run(t, dir, os.Environ()); code != 0 {
			t.Errorf("exit %d on a gofmt-clean tree:\n%s", code, out)
		}
	})
	t.Run("unformatted file fails and is named", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "messy.go"), []byte(messy), 0o600); err != nil {
			t.Fatal(err)
		}
		code, out := run(t, dir, os.Environ())
		if code == 0 {
			t.Fatalf("exit 0 on an unformatted tree; the check cannot fail:\n%s", out)
		}
		if !strings.Contains(out, "messy.go") {
			t.Errorf("the failure should name the file:\n%s", out)
		}
	})
	t.Run("missing gofmt fails rather than passing on empty output", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte(clean), 0o600); err != nil {
			t.Fatal(err)
		}
		// A PATH holding nothing: the wrapper's `gofmt` lookup fails, and the
		// substitution's status has to be what decides, not its empty stdout.
		if code, out := run(t, dir, []string{"PATH=" + t.TempDir()}); code == 0 {
			t.Errorf("exit 0 with no gofmt on PATH -- the wrong-machine case reads as a clean tree:\n%s", out)
		}
	})
}

// The python gate's compile check is a small program rather than a shell one-liner
// -- an ast.parse walk that emits no bytecode, replacing a compileall wrapper whose
// EXIT trap could not run when Supervise SIGKILLed the process group and left a
// bytecode mirror in $TMPDIR each time. Like the go gate's fmt command it is
// executed here against what matters: it passes a valid tree, fails and names the
// file on a syntax error, skips virtualenvs, and leaves neither a __pycache__ in
// the tree nor anything in $TMPDIR.
func TestShippedPythonGateCompileCommandFailsForTheRightReasons(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	var g Gate
	if err := decodeInto(filepath.Join("..", "..", projectBundleDir, gatesDir, "python"+configExt), &g); err != nil {
		t.Fatal(err)
	}
	var compile []string
	for _, c := range g.Commands {
		if c.Name == "compile" {
			compile = c.Run
		}
	}
	if len(compile) < 2 || compile[0] != "python3" {
		t.Fatalf("python gate has no `compile` command of the shape [python3 ...]: %v", compile)
	}
	run := func(t *testing.T, dir, tmp string) (int, string) {
		t.Helper()
		cmd := exec.Command(compile[0], compile[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "TMPDIR="+tmp)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return 0, string(out)
		}
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("running the compile check: %v\n%s", err, out)
		}
		return exit.ExitCode(), string(out)
	}
	noCache := func(t *testing.T, dir string) {
		t.Helper()
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err == nil && d.IsDir() && d.Name() == "__pycache__" {
				t.Errorf("the check wrote %s into the tree; a fix commit would carry it", path)
			}
			return nil
		})
	}
	entries := func(t *testing.T, dir string) int {
		t.Helper()
		es, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		return len(es)
	}

	t.Run("valid tree passes, broken venv ignored", func(t *testing.T) {
		dir, tmp := t.TempDir(), t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "ok.py"), []byte("x = 1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, ".venv"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".venv", "broken.py"), []byte("def (\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		before := entries(t, tmp)
		if code, out := run(t, dir, tmp); code != 0 {
			t.Errorf("exit %d on a valid tree (the broken file is in .venv and must be skipped):\n%s", code, out)
		}
		noCache(t, dir)
		if after := entries(t, tmp); after != before {
			t.Errorf("$TMPDIR grew from %d to %d entries; the check must leave nothing behind", before, after)
		}
	})
	t.Run("syntax error in a project file fails and is named", func(t *testing.T) {
		dir, tmp := t.TempDir(), t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "bad.py"), []byte("def (\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		code, out := run(t, dir, tmp)
		if code == 0 {
			t.Fatalf("exit 0 on a tree with a syntax error:\n%s", out)
		}
		if !strings.Contains(out, "bad.py") {
			t.Errorf("the failure should name the file:\n%s", out)
		}
		noCache(t, dir)
	})
}

// PRFromBranch is the invocation saying no -pr was typed, and a config must not
// be able to say it: the fork refusal and the branch resolution both hang off it,
// and the bundle may be the reviewed repository's own. The yaml:"-" tag is what
// enforces that, and a tag is exactly the kind of thing a refactor drops.
func TestPRFromBranchCannotComeFromYAML(t *testing.T) {
	var c Config
	if err := yaml.Unmarshal([]byte("target:\n  mode: pr\n  pr_from_branch: true\n  prfrombranch: true\n  PRFromBranch: true\n"), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Target.PRFromBranch {
		t.Error("a config file set target.PRFromBranch; it is an assertion about how the command was typed and only the flag layer may make it")
	}
	if c.Target.Mode != ModePR {
		t.Fatalf("mode = %q, want pr -- the fixture must otherwise parse, or this proves nothing", c.Target.Mode)
	}
	// The flag layer can, and that is the only way in.
	Overrides{PRFromBranch: true}.apply(&c)
	if !c.Target.PRFromBranch {
		t.Error("the override did not reach the config")
	}
}
