//go:build darwin

package agent

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

// Darwin's downgrade is unconditional, and deliberately so: the BSD kill(2) macOS
// inherits answers EPERM -- not ESRCH -- once the group holds nothing signalable,
// so a reaped leader's group reports EPERM on every call and the signal-0 probe
// answers EPERM for it just the same. Without the downgrade a macOS run that hits
// that window reports a successful agent invocation as canceled. This pins the
// platform split from the darwin side, so the day the probe can tell the two
// states apart here, this test is what says the blind spot was intentional rather
// than letting the mapping be dropped by accident on a platform where it is the
// only thing keeping successful runs from being discarded.
func TestKillProcessGroupDowngradesEPERMOnDarwin(t *testing.T) {
	stubGroupKill(t, syscall.EPERM, syscall.EPERM)
	err := KillProcessGroup(&exec.Cmd{Process: &os.Process{Pid: stubbedPid}})
	if !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("KillProcessGroup(EPERM) = %v, want an error wrapping os.ErrProcessDone; on darwin an empty group is what answers EPERM", err)
	}
	if errors.Is(err, syscall.EPERM) {
		t.Errorf("KillProcessGroup leaked the raw errno %v; os/exec does not recognize it as already-finished", err)
	}
}
