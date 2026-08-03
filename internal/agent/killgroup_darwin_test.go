//go:build darwin

package agent

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

// stubGroupPopulated makes the process-group membership query answer from the test
// instead of the kernel. It is the only way to reach the reply that matters -- a
// group still holding a member this process cannot signal -- since a test cannot
// create a process under other credentials. Restoring the real query is left to
// t.Cleanup so a failing assertion cannot leak the stub into later tests.
func stubGroupPopulated(t *testing.T, populated, known bool) {
	t.Helper()
	orig := groupPopulated
	t.Cleanup(func() { groupPopulated = orig })
	groupPopulated = func(int) (bool, bool) { return populated, known }
}

// On darwin the EPERM from kill(-pgid, SIGKILL) is ambiguous -- an empty group and
// a group of unsignalable members both answer it, and so does the signal-0 probe
// -- so what decides the downgrade is the kern.proc.pgrp membership listing.
// Drive KillProcessGroup through all three replies, because both directions of
// getting this wrong cost something: downgrading a populated group reports an
// uncontained descendant as cleanly contained, and refusing to downgrade an empty
// one rewrites a leader that merely raced its deadline into a cancel error --
// a complete review discarded, or a landed commit called failed.
func TestKillProcessGroupEPERMFollowsGroupMembershipOnDarwin(t *testing.T) {
	cmd := &exec.Cmd{Process: &os.Process{Pid: stubbedPid}}

	// The group is empty: EPERM is all darwin has to say about a group that is gone.
	stubGroupKill(t, syscall.EPERM, syscall.EPERM)
	stubGroupPopulated(t, false, true)
	err := KillProcessGroup(cmd)
	if !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("KillProcessGroup(EPERM, empty group) = %v, want an error wrapping os.ErrProcessDone; on darwin an empty group is what answers EPERM", err)
	}
	if errors.Is(err, syscall.EPERM) {
		t.Errorf("KillProcessGroup leaked the raw errno %v; os/exec does not recognize it as already-finished", err)
	}

	// The kernel would not answer, so nothing was learned: keep the downgrade every
	// macOS run has relied on rather than failing successful runs on a probe that
	// does not work.
	stubGroupKill(t, syscall.EPERM, syscall.EPERM)
	stubGroupPopulated(t, false, false)
	if err := KillProcessGroup(cmd); !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("KillProcessGroup(EPERM, membership unknown) = %v, want an error wrapping os.ErrProcessDone", err)
	}

	// The group still holds something this process cannot signal: containment
	// provably failed and the errno has to reach the caller.
	stubGroupKill(t, syscall.EPERM, syscall.EPERM)
	stubGroupPopulated(t, true, true)
	err = KillProcessGroup(cmd)
	if errors.Is(err, os.ErrProcessDone) {
		t.Errorf("KillProcessGroup(EPERM, populated group) = %v, want the EPERM; downgrading it reports a group that still holds a descendant as cleanly contained", err)
	}
	if !errors.Is(err, syscall.EPERM) {
		t.Errorf("KillProcessGroup(EPERM, populated group) = %v, want the errno surfaced so the step reports the failed containment", err)
	}
}

// The stubs above pin the composition; this pins the query itself against the real
// kernel, which is the half that cannot be reasoned out from the errno tables. It
// has to separate a group with a live member from one whose leader has exited AND
// been reaped -- the two states darwin's kill(2) reports identically.
func TestDarwinGroupPopulatedSeparatesLiveFromReaped(t *testing.T) {
	live := exec.Command("sleep", "60")
	live.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := live.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	t.Cleanup(func() {
		_ = KillProcessGroup(live)
		_ = live.Wait()
	})
	populated, known := darwinGroupPopulated(live.Process.Pid)
	if !known {
		t.Fatalf("darwinGroupPopulated(live group) reported the kernel gave no answer; the EPERM downgrade then has nothing to consult")
	}
	if !populated {
		t.Errorf("darwinGroupPopulated(live group) = false, want true; an EPERM would then report an uncontained group as finished")
	}

	reaped := exec.Command("true")
	reaped.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := reaped.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	if err := reaped.Wait(); err != nil { // reaps the leader, emptying the group
		t.Fatalf("Wait() = %v", err)
	}
	if populated, _ := darwinGroupPopulated(reaped.Process.Pid); populated {
		t.Errorf("darwinGroupPopulated(reaped leader) = true, want false; the group is empty and a command that raced its deadline is otherwise rewritten into a cancel error")
	}
}
