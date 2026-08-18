//go:build unix

package implement

import "golang.org/x/sys/unix"

// DiskFree reports the free bytes on the filesystem holding path, and whether
// the platform could answer. min_free_disk is a refusal input, so "could not
// answer" must be distinguishable from "zero free".
func DiskFree(path string) (int64, bool) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, false
	}
	// The block size is the one field whose TYPE differs across the platforms
	// this file serves: int64 on Linux, uint32 on Darwin. Converting it to
	// uint64 is therefore necessary on both, while `int64(st.Bsize)` was
	// redundant on Linux and tripped unconvert there -- caught by the release
	// gate's Linux runner on its first run, having passed on macOS. Bavail is
	// already uint64 everywhere, so the multiplication needs no conversion at
	// all and only the result is narrowed.
	free := st.Bavail * uint64(st.Bsize)
	return int64(free), true //nolint:gosec // filesystem sizes, not attacker input
}
