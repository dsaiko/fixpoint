package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
	"github.com/dsaiko/fixpoint/internal/model"
)

// The exact leak review run 20260929-113519 (i2) described: attempt 1 runs the
// build, leaves ignored output behind and breaks the output contract; attempt 2
// writes a source file and reports implemented, and the gate passes only while
// that ignored output exists. The discard used to stash and assert clean, both
// blind to ignored paths, so attempt 2's step 2 censused attempt 1's build as
// pre-existing, step 6 kept it, and the gate passed on another session's
// artifacts -- two sessions, one commit, attributed to one.
func TestRunImplementDiscardDeletesTheAttemptsIgnoredOutput(t *testing.T) {
	counter := filepath.Join(t.TempDir(), "sessions")
	coder := "n=$(cat " + counter + " 2>/dev/null || echo 0); n=$((n+1)); echo $n > " + counter + "\n" +
		"if [ \"$n\" = 1 ]; then mkdir -p scratch && printf 'built\\n' > scratch/build.tmp; echo 'no contract block'; exit 0; fi\n" +
		implementReply("printf 'work\\n' > \"src_$$.txt\"", `{"status": "implemented", "notes": "done"}`)
	f := newImplementFixture(t, coder, config.Verify{
		Policy:   config.VerifyMustPass,
		Timeout:  config.Duration(time.Minute),
		Commands: []config.VerifyCommand{{Name: "needs-build", Run: []string{"test", "-f", "scratch/build.tmp"}}},
	})
	f.cfg.Implement.GitignoreSeed = []string{"scratch/"}
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Tasks[0].Outcome == outcomeImplemented {
		t.Errorf("T01 was implemented on a gate that passed only on attempt 1's discarded build output\nlog:\n%s", f.logs())
	}
	if _, err := os.Stat(filepath.Join(f.out, "scratch", "build.tmp")); !os.IsNotExist(err) {
		t.Errorf("the discarded attempt's ignored output survived: %v", err)
	}
}

// Every discard exit deletes what the attempt created under an ignore rule --
// the stash, the checkout-and-clean variant, and the gate's own droppings --
// while the tree stays clean relative to HEAD.
func TestRunImplementEveryDiscardDeletesCreatedIgnoredPaths(t *testing.T) {
	failingGate := config.Verify{
		Policy:   config.VerifyMustPass,
		Timeout:  config.Duration(time.Minute),
		Commands: []config.VerifyCommand{{Name: "build-then-fail", Run: []string{"sh", "-c", "mkdir -p scratch && printf 'o\\n' > scratch/gate.out; false"}}},
	}
	off := config.Verify{Policy: config.VerifyOff}
	cases := []struct {
		name     string
		coder    string
		verify   config.Verify
		maxBytes int64
		residue  []string
	}{
		{
			name:    "broken output contract",
			coder:   "mkdir -p scratch && printf 'x\\n' > scratch/build.tmp\nprintf 'work\\n' > src.txt\necho 'no contract block'\n",
			verify:  off,
			residue: []string{"scratch/build.tmp"},
		},
		{
			name:    "failed gate",
			coder:   implementReply("printf 'work\\n' > src.txt", `{"status": "implemented", "notes": "done"}`),
			verify:  failingGate,
			residue: []string{"scratch/gate.out"},
		},
		{
			name:     "byte bound",
			coder:    implementReply("mkdir -p scratch && printf 'x\\n' > scratch/build.tmp\nprintf '%0200d' 0 > src.txt", `{"status": "implemented", "notes": "done"}`),
			verify:   off,
			maxBytes: 64,
			residue:  []string{"scratch/build.tmp"},
		},
		{
			name:    "credential-shaped file",
			coder:   implementReply("mkdir -p scratch && printf 'x\\n' > scratch/build.tmp\nprintf 'ok\\n' > src.txt\nprintf 'TOKEN=sk-live\\n' > .env", `{"status": "implemented", "notes": "done"}`),
			verify:  off,
			residue: []string{"scratch/build.tmp"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newImplementFixture(t, tc.coder, tc.verify)
			f.cfg.Implement.GitignoreSeed = []string{"scratch/"}
			if tc.maxBytes > 0 {
				f.cfg.Implement.MaxTaskBytes = config.ByteSize(tc.maxBytes)
			}
			sum, err := f.run(t)
			if err != nil {
				t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
			}
			if sum.Tasks[0].Outcome != outcomeFailed {
				t.Fatalf("T01 = %+v, want failed\nlog:\n%s", sum.Tasks[0], f.logs())
			}
			for _, rel := range tc.residue {
				if _, err := os.Stat(filepath.Join(f.out, rel)); !os.IsNotExist(err) {
					t.Errorf("%s survived the discard: %v", rel, err)
				}
			}
			if st := gitOutAt(t, f.out, "status", "--porcelain"); strings.TrimSpace(st) != "" {
				t.Errorf("the discard left the tree dirty:\n%s", st)
			}
		})
	}
}

// The infrastructure exit is a discard too (review run 20260929-125352, i2):
// a session that runs the build and then dies with no reply -- a 429 mid-turn,
// a crashed CLI -- is retried without consuming an attempt, and the retry's
// step 2 used to census the dead session's build output as pre-existing. Every
// other ignored-output test exits through a reply, so this branch's cleanup
// could be dropped with the suite green.
func TestRunImplementInfraDiscardDeletesTheAttemptsIgnoredOutput(t *testing.T) {
	counter := filepath.Join(t.TempDir(), "sessions")
	coder := "n=$(cat " + counter + " 2>/dev/null || echo 0); n=$((n+1)); echo $n > " + counter + "\n" +
		"if [ \"$n\" = 1 ]; then mkdir -p scratch && printf 'built\\n' > scratch/build.tmp; exit 1; fi\n" +
		implementReply("printf 'work\\n' > \"src_$$.txt\"", `{"status": "implemented", "notes": "done"}`)
	f := newImplementFixture(t, coder, config.Verify{
		Policy:   config.VerifyMustPass,
		Timeout:  config.Duration(time.Minute),
		Commands: []config.VerifyCommand{{Name: "needs-build", Run: []string{"test", "-f", "scratch/build.tmp"}}},
	})
	f.cfg.Implement.GitignoreSeed = []string{"scratch/"}
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if !strings.Contains(f.logs(), "infrastructure failure") {
		t.Fatalf("session 1 was not treated as infrastructure, so this proves nothing\nlog:\n%s", f.logs())
	}
	if sum.Tasks[0].Outcome == outcomeImplemented {
		t.Errorf("T01 was implemented on a gate that passed only on a dead session's build output\nlog:\n%s", f.logs())
	}
	if _, err := os.Stat(filepath.Join(f.out, "scratch", "build.tmp")); !os.IsNotExist(err) {
		t.Errorf("the dead session's ignored output survived: %v", err)
	}
}

// The census used to record ignored FILES only, so the discard deleted
// scratch/sub/build.tmp and left scratch/sub/ behind -- invisible to GitClean
// -- and a gate that checks for the directory passed on the discarded attempt's
// residue (review run 20260929-125352, i4).
func TestRunImplementDiscardDeletesTheAttemptsIgnoredDirectories(t *testing.T) {
	counter := filepath.Join(t.TempDir(), "sessions")
	coder := "n=$(cat " + counter + " 2>/dev/null || echo 0); n=$((n+1)); echo $n > " + counter + "\n" +
		"if [ \"$n\" = 1 ]; then mkdir -p scratch/sub scratch/empty && printf 'built\\n' > scratch/sub/build.tmp; echo 'no contract block'; exit 0; fi\n" +
		implementReply("printf 'work\\n' > \"src_$$.txt\"", `{"status": "implemented", "notes": "done"}`)
	f := newImplementFixture(t, coder, config.Verify{
		Policy:   config.VerifyMustPass,
		Timeout:  config.Duration(time.Minute),
		Commands: []config.VerifyCommand{{Name: "needs-dir", Run: []string{"test", "-d", "scratch"}}},
	})
	f.cfg.Implement.GitignoreSeed = []string{"scratch/"}
	sum, err := f.run(t)
	if err != nil {
		t.Fatalf("runImplement() = %v\nlog:\n%s", err, f.logs())
	}
	if sum.Tasks[0].Outcome == outcomeImplemented {
		t.Errorf("T01 was implemented on a gate that passed only on attempt 1's discarded directory\nlog:\n%s", f.logs())
	}
	if _, err := os.Stat(filepath.Join(f.out, "scratch")); !os.IsNotExist(err) {
		t.Errorf("the discarded attempt's ignored directory survived: %v", err)
	}
}

// An attempt canceled after writing ONLY ignored output leaves GitClean true,
// and cleanUpInterrupted returned on that before the ignored reconciliation
// existed there -- so the build output stood for -continue's next step 2 to
// census as pre-existing (review run 20260929-125352, i12). The cancel waits for
// the session to say it has written, so a clean result cannot come from a
// cancel that landed before the write.
func TestRunImplementInterruptedAttemptDeletesItsIgnoredOutput(t *testing.T) {
	written := filepath.Join(t.TempDir(), "written")
	f := newImplementFixture(t,
		"mkdir -p scratch && printf 'built\\n' > scratch/build.tmp\ntouch "+written+"\nsleep 60\n",
		config.Verify{Policy: config.VerifyOff})
	f.cfg.Implement.GitignoreSeed = []string{"scratch/"}
	if err := os.WriteFile(f.planFile, []byte(f.planJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	logf, logs := captureLog()
	f.logs = logs
	o, err := New(&config.Loaded{Config: f.cfg, Source: config.Source{Config: "t.yaml"}}, logf)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		for {
			if _, err := os.Stat(written); err == nil {
				cancel()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}()
	var sum model.RunSummary
	if err := o.runImplement(ctx, &sum); err == nil {
		t.Fatalf("an interrupted run reported success\nlog:\n%s", f.logs())
	}
	if _, err := os.Stat(written); err != nil {
		t.Fatalf("the session never wrote, so this proves nothing: %v\nlog:\n%s", err, f.logs())
	}
	if _, err := os.Stat(filepath.Join(f.out, "scratch")); !os.IsNotExist(err) {
		t.Errorf("the interrupted attempt's ignored output survived the cleanup: %v\nlog:\n%s", err, f.logs())
	}
}

// When the discard cannot delete what the attempt created, the run must STOP
// naming the path: the next attempt would otherwise build on a dead session's
// output with nothing recording it (review run 20260929-125352, i17). A
// read-only directory makes the unlink fail; root ignores the mode, so there
// the precondition cannot be built.
func TestRunImplementStuckIgnoredDiscardStopsTheRun(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root deletes from a read-only directory, so the removal cannot be made to stick")
	}
	f := newImplementFixture(t,
		"mkdir -p scratch && printf 'x\\n' > scratch/build.tmp && chmod 500 scratch\necho 'no contract block'\n",
		config.Verify{Policy: config.VerifyOff})
	f.cfg.Implement.GitignoreSeed = []string{"scratch/"}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(f.out, "scratch"), 0o700) })
	sum, err := f.run(t)
	var stop runStopError
	if !errors.As(err, &stop) {
		t.Fatalf("runImplement() = %v, want a run stop\nsummary: %+v\nlog:\n%s", err, sum.Tasks, f.logs())
	}
	if !strings.Contains(err.Error(), "the discard could not delete") || !strings.Contains(err.Error(), "scratch/build.tmp") {
		t.Errorf("the stop does not name the stuck path: %v", err)
	}
}
