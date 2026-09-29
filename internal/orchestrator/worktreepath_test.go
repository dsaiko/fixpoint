package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/gitenv"
)

// A run over a SUBDIRECTORY of a checkout reviews a branch that owns the whole
// checkout, so a PATH entry elsewhere in it (/repo/bin, next to the reviewed
// /repo/src) is as much the branch's as one inside target.path. Both preflight
// refusals of a binary resolved through PATH -- the agent CLI's and the pinned
// helpers' -- must measure against the work tree (review run 20260929-141502, i9).
func TestPreflightMeasuresPATHAgainstTheWorkTree(t *testing.T) {
	setup := func(t *testing.T, planted string) (*fixture, string) {
		t.Helper()
		f := newFixture(t, config.Loop{MaxIterations: 1, CleanRoundsToStop: 1, ReviewOnly: true})
		f.cfg.Loop.TrustedTarget = false
		sub := filepath.Join(f.repo, "src")
		bin := filepath.Join(f.repo, "bin")
		for _, d := range []string{sub, bin} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(bin, planted), []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		f.cfg.Target.Path = sub
		// Registered before the PATH change so cleanups (LIFO) restore PATH first and
		// re-pin against the real one.
		t.Cleanup(gitenv.PinTools)
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		return f, bin
	}

	t.Run("agent CLI", func(t *testing.T) {
		f, bin := setup(t, "claude")
		a := f.cfg.Agents["mock"]
		a.Command = []string{"claude"}
		f.cfg.Agents["mock"] = a
		// The gap: Validate's own measure, against target.path, sees nothing.
		if dir := config.TargetSuppliedPATHDir("claude", f.cfg.Target.Path); dir != "" {
			t.Fatalf("precondition: TargetSuppliedPATHDir against target.path = %q, want \"\"", dir)
		}
		err := f.orchestrator().PreflightGuards(t.Context())
		if err == nil || !strings.Contains(err.Error(), bin) {
			t.Fatalf("PreflightGuards() = %v, want a refusal naming %s", err, bin)
		}
		f.cfg.Loop.TrustedTarget = true
		if err := f.orchestrator().PreflightGuards(t.Context()); err != nil {
			t.Errorf("with -trusted-target: %v, want nil", err)
		}
	})

	t.Run("pinned helper", func(t *testing.T) {
		f, _ := setup(t, "gh")
		gitenv.PinTools()
		err := f.orchestrator().PreflightGuards(t.Context())
		if err == nil || !strings.Contains(err.Error(), "gh resolves to") {
			t.Fatalf("PreflightGuards() = %v, want the gh refusal", err)
		}
	})
}
