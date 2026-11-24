package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The two SHIPPED bench configs, loaded and run through the real entry point with
// a fake candidate and a recording wrapper named sandbox.sh on PATH. Every lens
// must reach the wrapper with the staged target, `ro`, `--`, and the candidate's
// own argv. bench/run_test.sh stubs ./fixpoint out, and the CLI test beside this one
// uses its own synthetic config, so without this a literal `rw`, a misspelled
// wrapper name or a dropped sandbox block in config/bench-*.yaml leaves every test
// green while real runs fail to start or let the lens write what it reviews.
func TestShippedBenchConfigsReachTheWrapperWithTheStagedTargetReadOnly(t *testing.T) {
	for _, tc := range []struct {
		name   string
		lenses int
	}{
		{"bench-code", 3},
		{"bench-design", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// newFixture changes directory into the scratch repository.
			cfgDir, err := filepath.Abs(filepath.Join("..", "..", "config"))
			if err != nil {
				t.Fatal(err)
			}
			f := newFixture(t)
			for n := 1; n <= tc.lenses; n++ {
				f.respond(n, reviewResponse(t))
			}
			// The shipped bundle, as ~/.fixpoint: the config, its base and every
			// prompt. Only the candidate agent is made up, as run.sh generates it.
			bundle := filepath.Join(os.Getenv("HOME"), ".fixpoint")
			for _, name := range []string{"bench-code.yaml", "bench-design.yaml", "defaults.yaml"} {
				copyFile(t, filepath.Join(cfgDir, name), filepath.Join(bundle, name))
			}
			prompts, err := os.ReadDir(filepath.Join(cfgDir, "prompts"))
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range prompts {
				copyFile(t, filepath.Join(cfgDir, "prompts", p.Name()), filepath.Join(bundle, "prompts", p.Name()))
			}
			agent := fmt.Sprintf("model: fake\ncommand:\n  - %s\n  - --sandbox danger-full-access\nprompt_via: stdin\ntimeout: 1m\ncan_edit: false\n", f.script)
			if err := os.MkdirAll(filepath.Join(bundle, "agents"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bundle, "agents", "bench-candidate.yaml"), []byte(agent), 0o600); err != nil {
				t.Fatal(err)
			}

			// The recording wrapper, found by name through PATH exactly as run.sh
			// arranges, then handing over to the agent the way the real one does.
			bin := t.TempDir()
			rec := filepath.Join(t.TempDir(), "calls")
			// One printf per call: the lenses run concurrently, and a loop of echoes
			// from several of them interleaves.
			wrapper := "#!/bin/sh\n" +
				"printf 'CALL %s\\n' \"$*\" >>'" + rec + "'\n" +
				"shift 3\nexec \"$@\"\n"
			if err := os.WriteFile(filepath.Join(bin, "sandbox.sh"), []byte(wrapper), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			var buf bytes.Buffer
			// The exit code is review-only's own business; what matters is below.
			run([]string{tc.name, "-target", f.repo, "-trusted-bundle", "-trusted-target", "-agents-sandboxed"}, &buf, &buf)

			data, err := os.ReadFile(rec)
			if err != nil {
				t.Fatalf("no lens reached the wrapper:\n%s", buf.String())
			}
			calls := strings.Split(strings.TrimSpace(string(data)), "\n")
			// Every lens, plus fixpoint's own agent ping before the round.
			if len(calls) < tc.lenses {
				t.Fatalf("%d call(s) reached the wrapper, want at least %d:\n%s\n%s", len(calls), tc.lenses, data, buf.String())
			}
			want := "CALL " + strings.Join([]string{f.repo, "ro", "--", f.script, "--sandbox", "danger-full-access"}, " ")
			for i, c := range calls {
				if c != want {
					t.Errorf("call %d reached the wrapper as\n  %s\nwant\n  %s", i+1, c, want)
				}
			}
		})
	}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
