package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
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
