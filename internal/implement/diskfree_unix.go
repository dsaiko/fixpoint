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
	return int64(st.Bavail) * int64(st.Bsize), true //nolint:gosec // sizes, not attacker input
}
