//go:build unix

package implement

import "golang.org/x/sys/unix"

// DiskFree reports the free bytes on the filesystem holding path, and whether
// the platform could answer. min_free_disk is a refusal input, so "could not
// answer" must be distinguishable from "zero free".
//
// The block size is read through statfsBlockSize, which is per-platform for a
// reason worth stating: Statfs_t.Bsize is int64 on Linux and uint32 on Darwin,
// and no single expression is clean on both. `int64(st.Bsize)` is redundant on
// Linux (unconvert); `uint64(st.Bsize)` is an overflow conversion there
// (gosec G115) and unremarkable on Darwin. A //nolint cannot bridge that
// either, because nolintlint reports a directive that did not fire -- so
// suppressing on one platform fails on the other. Splitting the one differing
// conversion into files that are each compiled on one platform is what makes
// every suppression apply exactly where it is true. Both failures came from
// the release gate's Linux runner, minutes after macOS passed.
func DiskFree(path string) (int64, bool) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, false
	}
	// Bavail is uint64 on every platform here, so the product needs no
	// conversion; only the result is narrowed.
	free := st.Bavail * statfsBlockSize(&st)
	return int64(free), true //nolint:gosec // filesystem sizes, not attacker input
}
