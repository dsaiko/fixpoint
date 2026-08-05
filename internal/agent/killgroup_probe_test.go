//go:build !darwin

package agent

import (
	"errors"
	"os"
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

// The predicate above is only half the rule: what contains a group off darwin is
// KillProcessGroup CONSULTING it before downgrading an EPERM. Drive that
// composition end to end, because dropping the condition -- or inverting it --
// leaves the predicate's own test green while every EPERM again reads as a clean
// finish. Cases: the stale EPERM worth rescuing (the last member exited between
// the kill and the probe, so the probe answers ESRCH for the group); the EPERM
// that proves containment failed (the member is still there and still
// unsignalable, so the probe answers EPERM too); and a probe that succeeds
// outright, which is a member this process CAN signal still sitting in the group.
// Only the first may become os.ErrProcessDone.
func TestKillProcessGroupEPERMFollowsTheProbe(t *testing.T) {
	cmd := &exec.Cmd{Process: &os.Process{Pid: stubbedPid}}

	stubGroupKill(t, syscall.EPERM, syscall.ESRCH)
	err := KillProcessGroup(cmd)
	if !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("KillProcessGroup(EPERM, group now empty) = %v, want an error wrapping os.ErrProcessDone; a command that merely raced its deadline is otherwise rewritten into a cancel error", err)
	}
	if errors.Is(err, syscall.EPERM) {
		t.Errorf("KillProcessGroup leaked the raw errno %v; os/exec does not recognize it as already-finished", err)
	}

	for _, probe := range []error{syscall.EPERM, nil} {
		stubGroupKill(t, syscall.EPERM, probe)
		err := KillProcessGroup(cmd)
		if errors.Is(err, os.ErrProcessDone) {
			t.Errorf("KillProcessGroup(EPERM, probe = %v) = %v, want the EPERM; downgrading it reports a group that still holds a descendant as cleanly contained", probe, err)
		}
		if !errors.Is(err, syscall.EPERM) {
			t.Errorf("KillProcessGroup(EPERM, probe = %v) = %v, want the errno surfaced so the step reports the failed containment", probe, err)
		}
	}
}
