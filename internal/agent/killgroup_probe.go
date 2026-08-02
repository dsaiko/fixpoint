//go:build !darwin

package agent

import (
	"errors"
	"syscall"
)

// epermMeansGroupGone reports whether an EPERM from kill(-pgid, SIGKILL) may be
// read as "the group is already gone".
//
// Off darwin -- Linux is what fixpoint runs on -- ESRCH and EPERM mean opposite
// things. kill on a process group succeeds if it signals at least one member,
// answers ESRCH only when the group is EMPTY, and answers EPERM only when the
// group is NON-empty and not one remaining member is signalable (a descendant
// that changed credentials, e.g. one a `sudo`- or container-based verify command
// left behind after its leader exited). EPERM is therefore the single reply that
// proves containment failed, and downgrading it on the strength of the errno
// alone would report a live, uncontained process group -- holding whatever
// environment the command was given -- as a clean finish, with nothing in the run
// output saying so. The process-group kill is the only containment there is; when
// it provably did not contain, this fails closed and the step surfaces the errno.
//
// The reply can still be stale, which is the case worth rescuing: the last
// unsignalable member may exit between the kill and now, leaving the group as
// empty as an ESRCH would have said. Probe with signal 0 -- same per-process
// permission check, nothing delivered -- and downgrade only when the kernel now
// answers ESRCH for the group itself.
func epermMeansGroupGone(pid int) bool {
	return errors.Is(groupKill(pid, 0), syscall.ESRCH)
}
