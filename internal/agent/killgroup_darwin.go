//go:build darwin

package agent

// epermMeansGroupGone reports whether an EPERM from kill(-pgid, SIGKILL) may be
// read as "the group is already gone". On darwin it always is: the BSD kill(2)
// macOS inherits answers EPERM, not ESRCH, once the group holds nothing
// signalable, so a group whose leader has exited and been reaped reports EPERM on
// every call. Without this mapping TestSuperviseSuccessRacingDeadline failed 5 of
// 5 on macOS and passed 5 of 5 on Linux, which in a real run is a successful agent
// invocation reported as canceled -- a complete review discarded, or a landed
// commit called failed -- on every macOS run that hits the window.
//
// The probe the other platforms use cannot separate the two states here, since
// kill(-pgid, 0) answers EPERM for an empty group exactly as the SIGKILL did. So
// darwin keeps the unconditional downgrade and with it the blind spot: a group
// still holding an unsignalable descendant reads as finished. That is bounded by
// what this call site is -- a group this process created with Setpgid for its own
// child, so a genuine permission failure needs that child to change credentials,
// which no agent CLI does -- and by darwin being a development platform here,
// while the containment guarantee that matters is the Linux one.
func epermMeansGroupGone(int) bool { return true }
