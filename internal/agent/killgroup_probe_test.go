//go:build !darwin

package agent

import (
	"os/exec"
	"syscall"
	"testing"
)

// Off darwin, EPERM from the group kill means the group is NON-empty and nothing
// left in it is signalable -- containment provably failed. The only reason to
// read it as "already finished" is the race in which that last member exits
// between the kill and the check, so epermMeansGroupGone must answer for the
// group's state now, not for the errno. Both halves matter: say false for an
// empty group and a command that merely raced its deadline is rewritten into a
// cancel error; say true for a live one and an escaped descendant is reported as
// cleanly contained.
func TestEPERMOnlyMeansGroupGoneWhenItIs(t *testing.T) {
	reaped := exec.Command("true")
	reaped.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := reaped.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	if err := reaped.Wait(); err != nil { // reaps the leader, emptying the group
		t.Fatalf("Wait() = %v", err)
	}
	if !epermMeansGroupGone(reaped.Process.Pid) {
		t.Errorf("epermMeansGroupGone(reaped leader) = false, want true; the group is empty and there is nothing left to contain")
	}

	live := exec.Command("sleep", "60")
	live.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := live.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	t.Cleanup(func() {
		_ = KillProcessGroup(live)
		_ = live.Wait()
	})
	if epermMeansGroupGone(live.Process.Pid) {
		t.Errorf("epermMeansGroupGone(live group) = true, want false; an EPERM would then report an uncontained group as finished")
	}
}
