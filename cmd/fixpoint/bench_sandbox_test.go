package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
			// The mock agent answers its n-th invocation with the n-th response: fixpoint's
			// own agent ping comes first and takes one, then one per lens. A lens that ran
			// out of responses would fail its round and make the run below vacuous.
			for n := 1; n <= tc.lenses+1; n++ {
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
			// from several of them interleaves. The prompt is read first and handed
			// on, so each call records how long it was: a lens carries a prompt of
			// kilobytes, fixpoint's own agent ping a few words.
			scratch := t.TempDir()
			wrapper := "#!/bin/sh\n" +
				"in=$(mktemp '" + scratch + "/in.XXXXXX')\n" +
				"cat >\"$in\"\n" +
				"printf 'CALL %s %s\\n' \"$(wc -c <\"$in\" | tr -d ' ')\" \"$*\" >>'" + rec + "'\n" +
				"shift 3\nexec \"$@\" <\"$in\"\n"
			if err := os.WriteFile(filepath.Join(bin, "sandbox.sh"), []byte(wrapper), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			var buf bytes.Buffer
			got := run([]string{tc.name, "-target", f.repo, "-trusted-bundle", "-trusted-target", "-agents-sandboxed"}, &buf, &buf)
			// The run itself must complete, with a valid response from every lens: a
			// review-only run that ended cleanly exits 0 (nothing to fix) or 4, and any
			// reviewer or lens error would show in the output.
			if got != 0 && got != 4 {
				t.Fatalf("run exited %d, want 0 or 4 (review-only completed):\n%s", got, buf.String())
			}
			if !strings.Contains(buf.String(), "0 reviewer error(s)") {
				t.Errorf("the run reports reviewer errors:\n%s", buf.String())
			}
			if !strings.Contains(buf.String(), "1 of 1 reviewer(s) completed every lens") {
				t.Errorf("the reviewer did not complete every lens:\n%s", buf.String())
			}
			// Every lens line of the summary shows zero errors.
			lensErrors := regexp.MustCompile(`(?m)^\s+(review-\S+|design-\S+|\S+)\s+\d+\s+\d+\s+\d+\s+\d+\s+\d+\s+(\d+)\s`)
			for _, m := range lensErrors.FindAllStringSubmatch(buf.String(), -1) {
				if m[2] != "0" {
					t.Errorf("lens %s reports %s error(s):\n%s", m[1], m[2], buf.String())
				}
			}

			data, err := os.ReadFile(rec)
			if err != nil {
				t.Fatalf("no lens reached the wrapper:\n%s", buf.String())
			}
			want := strings.Join([]string{f.repo, "ro", "--", f.script, "--sandbox", "danger-full-access"}, " ")
			lenses, pings := 0, 0
			for i, c := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var size int
				if _, err := fmt.Sscanf(c, "CALL %d", &size); err != nil {
					t.Fatalf("call %d is not a recorded call: %q", i+1, c)
				}
				args := strings.SplitN(c, " ", 3)[2]
				if args != want {
					t.Errorf("call %d reached the wrapper as\n  %s\nwant\n  %s", i+1, args, want)
				}
				if size >= 1000 {
					lenses++
				} else {
					pings++
				}
			}
			// Exactly one wrapped call per lens: a lens that bypassed the wrapper
			// would be missing here, and the agent ping must not stand in for it.
			if lenses != tc.lenses {
				t.Errorf("%d lens call(s) reached the wrapper, want exactly %d (plus at most one ping, saw %d):\n%s\n%s", lenses, tc.lenses, pings, data, buf.String())
			}
			if pings > 1 {
				t.Errorf("%d ping calls, want at most 1", pings)
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
