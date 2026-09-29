package implement

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// RemoveCreated deletes the repo-relative paths an attempt created and returns
// the ones it left in place, each with the reason. census is the listing that
// named them, for each path's censused type (nil: files only, the un-ignored
// census never lists a directory).
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
// Nothing outside the repository, either (i11): every call goes through an
// os.Root, which refuses to resolve out of dir, and a path is refused outright
// when a parent component is a symlink or its own type is no longer the
// censused one. A window remains between that Lstat and the removal, but it is
// confined to dir by the Root: a racer who swaps a parent for a symlink to
// another directory in the repository can redirect one unlink or one rmdir of
// an empty directory there, never a recursive delete and never outside.
func RemoveCreated(dir string, created []string, census map[string]IgnoredStat) []string {
	if len(created) == 0 {
		return nil
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return []string{fmt.Sprintf("%s (%v)", strings.Join(created, ", "), err)}
	}
	defer func() { _ = root.Close() }()
	// Reverse order puts every path before its own prefix: children first.
	paths := slices.Clone(created)
	slices.Sort(paths)
	slices.Reverse(paths)
	var left []string
	for _, path := range paths {
		isDir := census[path].Dir || strings.HasSuffix(path, "/")
		if why := removeOne(root, filepath.FromSlash(strings.TrimSuffix(path, "/")), isDir); why != "" {
			left = append(left, fmt.Sprintf("%s (%s)", path, why))
		}
	}
	return left
}

// removeOne removes one path under root; "" means it is gone.
func removeOne(root *os.Root, rel string, isDir bool) string {
	// Top down, so the first symlink is named rather than resolved through.
	var parents []string
	for p := filepath.Dir(rel); p != "."; p = filepath.Dir(p) {
		parents = append(parents, p)
	}
	for _, p := range slices.Backward(parents) {
		st, err := root.Lstat(p)
		if os.IsNotExist(err) {
			return "" // a missing parent: the path is gone already
		}
		if err != nil {
			return "not touched: " + err.Error()
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return "not touched: " + filepath.ToSlash(p) + " is a symlink, and deleting through it could reach outside the repository"
		}
	}
	st, err := root.Lstat(rel)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		return "not touched: " + err.Error()
	}
	if st.IsDir() != isDir {
		return fmt.Sprintf("not touched: it is a %s now, not what the census recorded", st.Mode().Type())
	}
	if isDir {
		sweepEmptyDirs(root, rel)
	}
	if err := root.Remove(rel); err != nil && !os.IsNotExist(err) {
		if isDir {
			return "left in place: it holds entries neither census recorded -- " + err.Error()
		}
		return err.Error()
	}
	return ""
}

// sweepEmptyDirs removes the empty directories beneath rel, deepest first. A
// DirEntry's type comes from the directory itself, unfollowed, so a symlink is
// never descended; an rmdir that fails leaves the entry for the caller's own
// removal to report.
func sweepEmptyDirs(root *os.Root, rel string) {
	f, err := root.Open(rel)
	if err != nil {
		return
	}
	entries, err := f.ReadDir(-1)
	_ = f.Close()
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			child := filepath.Join(rel, e.Name())
			sweepEmptyDirs(root, child)
			_ = root.Remove(child)
		}
	}
}
