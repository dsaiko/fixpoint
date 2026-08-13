package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The shipped bundle has to actually deliver what a task config sets while keeping
// what the base holds. fix-code.yaml and fix-branch.yaml each declare a `loop:`
// block naming exactly one key, and `extends` merges PER KEY -- so a decode that
// replaced the block wholesale would silently blank max_iterations,
// clean_rounds_to_stop and commit_policy for both fix configs, and the run would
// take its behavior from Go's zero values instead of from defaults.yaml.
//
// Validate is deliberately not called: it checks that every agent's binary is on
// PATH, which says nothing about inheritance and would make this fail on a machine
// that simply has no agent CLIs installed.
func TestShippedFixConfigsInheritLoopWhileSettingTheirOwnSkipGlobs(t *testing.T) {
	for _, name := range []string{"fix-code", "fix-branch"} {
		t.Run(name, func(t *testing.T) {
			l, err := LoadBundle(&Resolver{Bundles: []string{"../../config"}}, name, "", Overrides{})
			if err != nil {
				t.Fatal(err)
			}
			loop := l.Config.Loop
			if got := loop.FinalSkipRunEdits; len(got) != 1 || got[0] != "**/*_test.go" {
				t.Errorf("final_skip_run_edits = %v, want [**/*_test.go] from the task config", got)
			}
			if loop.MaxIterations != 5 {
				t.Errorf("max_iterations = %d, want 5 inherited from defaults.yaml", loop.MaxIterations)
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
