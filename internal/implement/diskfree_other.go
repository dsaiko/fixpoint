//go:build !unix

package implement

// DiskFree is unsupported here; the check is skipped and says so.
func DiskFree(string) (int64, bool) { return 0, false }
