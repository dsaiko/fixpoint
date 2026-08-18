//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package implement

import "golang.org/x/sys/unix"

// statfsBlockSize widens the BSD block size, which is an unsigned 32-bit value
// -- so this conversion is both necessary and incapable of overflowing, and
// needs no suppression. That asymmetry with the Linux file is the whole reason
// the two exist.
func statfsBlockSize(st *unix.Statfs_t) uint64 {
	return uint64(st.Bsize)
}
