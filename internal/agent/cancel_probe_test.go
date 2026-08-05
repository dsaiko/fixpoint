package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dsaiko/fixpoint/internal/config"
)

func TestCancelProbe(t *testing.T) {
	// A few iterations to shake out cancel/kill races without amplifying a rare
	// scheduling stall into a flaky build: at 30x, a 1/100 readiness stall became
	// a ~1/3 failure rate. The readiness deadline is generous so a loaded runner's
	// slow-to-start child does not fatal, and cancel is always issued before the
	// helper returns so no child is left running.
	for i := range 5 {
		ctx, cancel := context.WithCancel(t.Context())
		dir := t.TempDir()
		a := config.Agent{
			Command:   []string{script(t, "touch ready\nsleep 60")},
			PromptVia: "stdin",
			Timeout:   config.Duration(time.Minute),
		}
		done := make(chan Result, 1)
		go func() { done <- Run(ctx, a, "", dir) }()
		ready := filepath.Join(dir, "ready")
		deadline := time.Now().Add(30 * time.Second)
		for {
			if _, err := os.Stat(ready); err == nil {
				break
			}
			if time.Now().After(deadline) {
				cancel()
				t.Fatalf("iter %d: child never signaled readiness", i)
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
		select {
		case res := <-done:
			// Canceling a live process must surface as an error; a nil error would
			// mean the child exited successfully rather than being killed.
			if res.Err == nil {
				t.Fatalf("iter %d: Run() err = nil after cancel, want cancellation error", i)
			}
		case <-time.After(10 * time.Second):
			out, _ := exec.Command("sh", "-c", "ps -axo pid,pgid,ppid,stat,command | grep -E 'sleep|agent.sh' | grep -v grep").Output()
			fmt.Printf("iter %d HANG; surviving processes:\n%s\n", i, out)
			t.Fatal("hang")
		}
	}
}
