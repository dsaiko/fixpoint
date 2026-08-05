//go:build darwin

package agent

import (
	"math"
	"syscall"
	"time"
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
	kernProcPID  = 1
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
	n, known := procListLen(kernProcPgrp, pgid)
	return n > 0, known
}

// groupEmptiesShortly reports whether pgid's group becomes empty within a bounded
// wait, which is what separates the leader's own unreaped zombie from a member this
// process cannot kill.
//
// It WAITS rather than interpreting one snapshot, and that is the point. Two
// earlier attempts read the listing once -- then compared it against a
// single-process listing to spot "only the leader" -- and both stayed flaky across
// the 400-iteration sweep, because every such reading races cmd.Wait reaping the
// leader between one sysctl and the next. There is no snapshot that answers this;
// the race has an outcome, so wait for it. cmd.Wait reaps within milliseconds and
// then the listing is unambiguous, while a live member this process cannot signal
// never leaves.
//
// Polling in cmd.Cancel is safe: Cancel runs on the goroutine os/exec starts for
// the context watch, and Wait proceeds independently on the caller's, so this
// cannot block the reap it is waiting for. The budget is small enough to be
// invisible next to an agent invocation and is spent only on the EPERM path.
const (
	groupDrainWait  = 5 * time.Millisecond
	groupDrainPolls = 40 // 200ms total
)

// groupEmptiesShortly is a var for the same reason groupPopulated is: the reply
// that matters -- a group that never empties because a member is unsignalable --
// cannot be created from a test. Nothing in production reassigns it.
var groupEmptiesShortly = groupDrains

func groupDrains(pgid int) bool {
	for i := range groupDrainPolls {
		n, known := procListLen(kernProcPgrp, pgid)
		if !known {
			return false
		}
		if n == 0 {
			return true
		}
		if i < groupDrainPolls-1 {
			time.Sleep(groupDrainWait)
		}
	}
	return false
}

// procListLen returns the byte length of the kern.proc.<kind>.<arg> listing.
// known is false when the kernel did not answer the question at all, which is not
// the same as an empty listing and must not be read as one. A reply too long for
// the buffer answers ENOMEM and is reported as procListBytes+1 -- more than fits,
// which for every caller here means "more than one entry".
func procListLen(kind, arg int) (n int, known bool) {
	if arg <= 0 || arg > math.MaxInt32 {
		return 0, false
	}
	//nolint:gosec // G115: kind is a package constant and arg is range-checked above.
	mib := [4]int32{ctlKern, kernProc, int32(kind), int32(arg)}
	buf := make([]byte, procListBytes)
	size := uintptr(len(buf))
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
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, 0)
	switch errno {
	case 0:
		return int(size), true
	case syscall.ENOMEM:
		// The listing did not fit, so it holds more than this buffer -- and therefore
		// more than the one entry any caller here compares against.
		return procListBytes + 1, true
	default:
		return 0, false
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
// The listing includes ZOMBIES, and that is where the first version of this went
// wrong. It argued that an ordinary same-credentials zombie is signalable, so the
// kill would have succeeded and never reached here -- which is false on darwin,
// where kill answers EPERM whenever it signaled nothing, and a zombie takes no
// signal. cmd.Wait races cmd.Cancel, so the ordinary case of a command completing
// as its deadline expires arrives here with the leader exited, unreaped and still
// listed. Hence the bounded wait below rather than membership alone.
func epermMeansGroupGone(pid int) bool {
	populated, known := groupPopulated(pid)
	if !known || !populated {
		return true
	}
	// The group holds something -- but the leader's own unreaped zombie is something,
	// and it is not a containment failure. So wait for the reap instead of trying to
	// read the difference out of one snapshot: an empty group is proof it was only
	// the zombie, and a member this process cannot signal never leaves.
	//
	// The membership query above stays honest about what the group holds, which is
	// what makes it usable for the live case; the judgement about whether that
	// membership excuses an EPERM belongs here, where the errno's meaning is known.
	return groupEmptiesShortly(pid)
}
