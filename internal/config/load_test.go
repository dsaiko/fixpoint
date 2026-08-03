package config

import "testing"

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
