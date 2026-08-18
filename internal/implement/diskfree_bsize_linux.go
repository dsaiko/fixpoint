//go:build linux

package implement

import "golang.org/x/sys/unix"

// statfsBlockSize widens Linux's int64 block size. Negative is not a value the
// kernel reports for it, and a filesystem block size cannot exceed the range
// either way.
func statfsBlockSize(st *unix.Statfs_t) uint64 {
	return uint64(st.Bsize) //nolint:gosec // a block size is small and never negative
}
