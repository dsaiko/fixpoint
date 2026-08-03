//go:build darwin

package agent

import (
	"math"
	"syscall"
	"unsafe"
)

// The MIB of darwin's kern.proc.pgrp.<pgid> sysctl -- the process table filtered
// to a single process group, which is what `ps -g` reads. CTL_KERN, KERN_PROC and
// KERN_PROC_PGRP are the ABI constants from <sys/sysctl.h>; the standard library
// exports neither them nor a MIB-taking sysctl on darwin, so both the constants
// and the call are spelled out here.
const (
	ctlKern      = 1
	kernProc     = 14
	kernProcPgrp = 2
)

// procListBytes bounds the reply darwinGroupPopulated asks the kernel for. Only
// the LENGTH of that reply is ever looked at, never one byte of its contents: the
// entries are struct kinfo_proc, whose layout varies with architecture and OS
// version, and "does this group hold anything" needs no field of it. A small
// buffer is therefore enough -- a listing too long for it answers ENOMEM, which
// is itself proof that the group listed something.
const procListBytes = 4096

// darwinGroupPopulated reports whether pgid's process group currently holds any
// process. known is false when the kernel did not answer the question at all (a
// rejected MIB on some future kernel, or a pgid that cannot be one), which is not
// the same as an empty group and must not be read as one.
func darwinGroupPopulated(pgid int) (populated, known bool) {
	if pgid <= 0 || pgid > math.MaxInt32 {
		return false, false
	}
	mib := [4]int32{ctlKern, kernProc, kernProcPgrp, int32(pgid)}
	buf := make([]byte, procListBytes)
	n := uintptr(len(buf))
	// Ask for the listing itself, not for its size: a size-only query (a nil buffer)
	// cannot answer this, because for KERN_PROC the kernel pads its estimate, so an
	// empty group comes back with a non-zero size. With a real buffer the kernel
	// reports how many bytes it actually wrote.
	//
	// The pointers are the audit G103 asks for: three arguments a raw sysctl cannot
	// be made without, each the address of a local this frame keeps alive across the
	// call, none of them arithmetic, and the reply is never dereferenced -- only its
	// length is read.
	//nolint:gosec // G103: unsafe.Pointer is unavoidable for a MIB-taking sysctl; audited just above.
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])), uintptr(len(mib)),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)), 0, 0)
	switch errno {
	case 0:
		return n > 0, true
	case syscall.ENOMEM:
		// The listing did not fit, which it cannot fail to do unless there is one.
		return true, true
	default:
		return false, false
	}
}

// groupPopulated is the process-group membership query epermMeansGroupGone
// consults. It is a var solely so a test can drive every reply: the state that
// makes the answer matter -- a live group member running under other credentials
// -- cannot be created from a test. Nothing in production reassigns it.
var groupPopulated = darwinGroupPopulated

// epermMeansGroupGone reports whether an EPERM from kill(-pgid, SIGKILL) may be
// read as "the group is already gone".
//
// On darwin the errno alone cannot decide it, and that is the whole difficulty
// here. The BSD kill(2) macOS inherits answers EPERM -- not ESRCH -- whenever the
// call signaled nothing, so BOTH an empty group and a group whose every remaining
// member is unsignalable report EPERM, and the signal-0 probe the other platforms
// use answers EPERM for an empty group exactly as the SIGKILL did. The two states
// need opposite handling. Reading EPERM as still-live turns a leader that merely
// raced its deadline into a cancel error (without the downgrade
// TestSuperviseSuccessRacingDeadline failed 5 of 5 on macOS and passed 5 of 5 on
// Linux, which in a real run is a complete review discarded or a landed commit
// called failed on every macOS run that hits the window). Reading it as gone
// reports a group that still holds a descendant this process cannot kill -- one a
// sudo- or container-based agent or verify command left behind under other
// credentials -- as cleanly contained, while it keeps running inside the target
// repository alongside the verification and the commit that follow.
//
// So ask the kernel what the group holds instead of asking the errno: the
// kern.proc.pgrp listing answers for membership regardless of whether this
// process could signal those members. Only an empty group -- or a kernel that
// would not answer, where the historical downgrade is the safer default because
// it is what every macOS run has relied on -- is read as gone.
//
// A group holding nothing but an unreaped zombie under other credentials is
// reported as still populated, since the listing includes zombies. That is an
// over-report of a failed containment, not an under-report: an ordinary
// same-credentials zombie is signalable, so the kill would have succeeded and
// never reached here at all.
func epermMeansGroupGone(pid int) bool {
	populated, known := groupPopulated(pid)
	return !known || !populated
}
