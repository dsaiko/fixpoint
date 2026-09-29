package implement

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"golang.org/x/sys/unix"
)

// RemoveCreated deletes the repo-relative paths an attempt created and returns
// the ones it left in place, each with the reason. census is the listing that
// named them, for each path's censused type (nil: files only, the un-ignored
// census never lists a directory). first is the attempt's step 2 ignored
// census, consulted only for a nested repository.
//
// Nothing recursive, because a census is not a full view of what existed: git
// reports a directory wholly ignored by its CURRENT contents, so a directory
// the task emptied of tracked files reads as created while it still holds the
// operator's ignored .env (review run 20260929-133423, i7). Files are unlinked;
// directories are removed deepest first with a plain rmdir, which fails on a
// non-empty directory, so bytes the censuses never saw survive and are
// reported instead of taken. Empty directories beneath a created one are swept
// on the way: git does not list them, and they hold no bytes to lose.
//
// The one exception is an entry git reports with a trailing slash: a nested
// repository, which git lists as a unit and never descends. Absent from the
// census before, it is the attempt's clone or `git init`, and rmdir can never
// empty it -- the run stopped on every such fixture (review run
// 20260929-141502, i2). It is removed whole unless first records a path
// beneath it, which would mean the attempt ran `git init` over bytes that were
// already there.
//
// Nothing outside the repository and nothing but the named entry (i14): no
// path is resolved twice. Every parent is opened from a pinned repository-root
// descriptor with O_NOFOLLOW, one component at a time, and the entry is
// unlinked relative to the last of them. A parent that is a symlink is refused
// by the open itself, and one swapped for a symlink after its open changes
// nothing, because the descriptor still names the directory that was checked.
// The previous os.Root plus Lstat left a window in which an in-repository
// symlink redirected the unlink onto a pre-existing file. POSIX only, like the
// rest of the program (.goreleaser.yaml builds no Windows target).
func RemoveCreated(dir string, created []string, census, first map[string]IgnoredStat) []string {
	if len(created) == 0 {
		return nil
	}
	root, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return []string{fmt.Sprintf("%s (%v)", strings.Join(created, ", "), err)}
	}
	defer func() { _ = unix.Close(root) }()
	// Reverse order puts every path before its own prefix: children first.
	paths := slices.Clone(created)
	slices.Sort(paths)
	slices.Reverse(paths)
	var left []string
	for _, path := range paths {
		if why := removeOne(root, path, census[path].Dir, first); why != "" {
			left = append(left, fmt.Sprintf("%s (%s)", path, why))
		}
	}
	return left
}

// removeOne removes one repo-relative path below the root descriptor; ""
// means it is gone.
func removeOne(root int, path string, censusDir bool, first map[string]IgnoredStat) string {
	rel, nested := strings.CutSuffix(path, "/")
	parts := strings.Split(rel, "/")
	parent, why := openParents(root, parts[:len(parts)-1])
	if parent < 0 {
		return why
	}
	if parent != root {
		defer func() { _ = unix.Close(parent) }()
	}
	afterParentsPinned()
	name := parts[len(parts)-1]
	var st unix.Stat_t
	if err := unix.Fstatat(parent, name, &st, unix.AT_SYMLINK_NOFOLLOW); errors.Is(err, unix.ENOENT) {
		return ""
	} else if err != nil {
		return "not touched: " + err.Error()
	}
	isDir := st.Mode&unix.S_IFMT == unix.S_IFDIR
	if isDir != (censusDir || nested) {
		return "not touched: it is a " + fileKind(st.Mode) + " now, not what the census recorded"
	}
	switch {
	case nested:
		if holdsFirstCensusPath(first, rel) {
			return "left in place: a nested repository over paths that existed before the attempt"
		}
		if err := removeTree(parent, name); err != nil {
			return "left in place: " + err.Error()
		}
		return ""
	case isDir:
		if fd, err := openDir(parent, name); err == nil {
			sweepEmptyDirs(fd)
			_ = unix.Close(fd)
		}
		if err := unix.Unlinkat(parent, name, unix.AT_REMOVEDIR); err != nil && !errors.Is(err, unix.ENOENT) {
			return "left in place: it holds entries neither census recorded -- " + err.Error()
		}
		return ""
	default:
		if err := unix.Unlinkat(parent, name, 0); err != nil && !errors.Is(err, unix.ENOENT) {
			return err.Error()
		}
		return ""
	}
}

// openParents walks parts down from root without following a symlink and
// returns the last directory's descriptor, root itself when there are no
// parts. -1 with "" means a component is missing, so the path is gone already.
func openParents(root int, parts []string) (int, string) {
	cur := root
	for i, name := range parts {
		next, err := openDir(cur, name)
		why := ""
		if err != nil {
			why = "not touched: " + strings.Join(parts[:i+1], "/") + ": " + err.Error()
			var st unix.Stat_t
			if errors.Is(err, unix.ENOENT) {
				why = ""
			} else if unix.Fstatat(cur, name, &st, unix.AT_SYMLINK_NOFOLLOW) == nil && st.Mode&unix.S_IFMT == unix.S_IFLNK {
				why = "not touched: " + strings.Join(parts[:i+1], "/") + " is a symlink, and deleting through it could reach outside the repository"
			}
		}
		if cur != root {
			_ = unix.Close(cur)
		}
		if err != nil {
			return -1, why
		}
		cur = next
	}
	return cur, ""
}

// afterParentsPinned runs between pinning a path's parents and touching the
// entry: the window i14's parent swap races, which a test cannot hit on
// demand. Nothing in production reassigns it.
var afterParentsPinned = func() {}

func openDir(parent int, name string) (int, error) {
	return unix.Openat(parent, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
}

// holdsFirstCensusPath reports whether first records any path strictly
// beneath rel.
func holdsFirstCensusPath(first map[string]IgnoredStat, rel string) bool {
	for p := range first {
		if strings.HasPrefix(p, rel+"/") {
			return true
		}
	}
	return false
}

// readNames lists the directory open at fd. The listing reads a duplicate, so
// fd stays the caller's to close.
func readNames(fd int) ([]os.DirEntry, error) {
	dup, err := unix.Dup(fd)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(dup), "")
	defer func() { _ = f.Close() }()
	return f.ReadDir(-1)
}

// sweepEmptyDirs removes the empty directories beneath the directory open at
// fd, deepest first. A DirEntry's type comes from the directory itself,
// unfollowed, and every descent is an O_NOFOLLOW open relative to fd, so a
// symlink is never descended; an rmdir that fails leaves the entry for the
// caller's own removal to report.
func sweepEmptyDirs(fd int) {
	entries, err := readNames(fd)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if child, err := openDir(fd, e.Name()); err == nil {
			sweepEmptyDirs(child)
			_ = unix.Close(child)
		}
		_ = unix.Unlinkat(fd, e.Name(), unix.AT_REMOVEDIR)
	}
}

// removeTree deletes name below parent and everything in it, through
// descriptors only: a directory is opened O_NOFOLLOW relative to the one that
// listed it, and every other entry -- a symlink included -- is unlinked where
// it stands, never resolved.
func removeTree(parent int, name string) error {
	fd, err := openDir(parent, name)
	if err != nil {
		return err
	}
	entries, err := readNames(fd)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				err = errors.Join(err, removeTree(fd, e.Name()))
			} else if uerr := unix.Unlinkat(fd, e.Name(), 0); uerr != nil && !errors.Is(uerr, unix.ENOENT) {
				err = errors.Join(err, fmt.Errorf("%s: %w", e.Name(), uerr))
			}
		}
	}
	_ = unix.Close(fd)
	if err != nil {
		return err
	}
	if err := unix.Unlinkat(parent, name, unix.AT_REMOVEDIR); err != nil && !errors.Is(err, unix.ENOENT) {
		return err
	}
	return nil
}

// fileKind names a stat mode's type. Generic because Stat_t.Mode is uint16 on
// Darwin and uint32 on Linux, and a conversion is redundant on one of them.
func fileKind[M uint16 | uint32](mode M) string {
	switch mode & unix.S_IFMT {
	case unix.S_IFDIR:
		return "directory"
	case unix.S_IFLNK:
		return "symlink"
	case unix.S_IFREG:
		return "regular file"
	}
	return "special file"
}
