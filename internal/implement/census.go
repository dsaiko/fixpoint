package implement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Census is the digest record of the un-ignored working tree's changes
// relative to HEAD (§5.2 step 6): untracked paths and modified tracked paths,
// each with content digests. Digests, not just paths -- a task's new files
// are untracked by definition, and a gate that rewrites one changes no path
// set at all. A deleted tracked path carries the empty digest.
type Census struct {
	Untracked map[string]string
	Modified  map[string]string
	// Bytes is the total size of the changed files, the value the byte bound
	// judges.
	Bytes int64
}

// Paths is every path the census names, untracked and modified together,
// sorted: the set a caller must decide about before any of it is staged.
func (c Census) Paths() []string {
	out := make([]string, 0, len(c.Untracked)+len(c.Modified))
	for p := range c.Untracked {
		out = append(out, p)
	}
	for p := range c.Modified {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// ByteBoundError reports a session whose changes exceed implement.
// max_task_bytes; the attempt fails BEFORE anything is hashed or gated, and
// the caller routes the oversized tree through the checkout-and-clean discard
// (§5.2 step 6, §5.3).
type ByteBoundError struct {
	Bytes, Limit int64
	At           string
}

func (e ByteBoundError) Error() string {
	return fmt.Sprintf("the session's changes reach %d bytes at %s, over implement.max_task_bytes (%d) -- one runaway generation must not consume the object database, the deadline and the disk", e.Bytes, e.At, e.Limit)
}

// TakeCensus records the un-ignored changes at dir. maxBytes > 0 enforces the
// byte bound during the walk, sizes first, so an oversized tree refuses
// before a single file is hashed.
func (g Git) TakeCensus(ctx context.Context, dir string, maxBytes int64) (Census, error) {
	c := Census{Untracked: map[string]string{}, Modified: map[string]string{}}
	out, err := g.run(ctx, dir, "status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=all")
	if err != nil {
		return c, err
	}
	type entry struct {
		path      string
		untracked bool
	}
	records := strings.Split(strings.Trim(out, "\x00"), "\x00")
	entries := make([]entry, 0, len(records))
	for _, rec := range records {
		if len(rec) < 4 {
			continue
		}
		status, path := rec[:2], rec[3:]
		if status == "??" {
			entries = append(entries, entry{path, true})
			continue
		}
		entries = append(entries, entry{path, false})
	}
	// Sizes first, then hashes: the bound must trip before the expensive half.
	for _, e := range entries {
		if st, err := os.Lstat(filepath.Join(dir, e.path)); err == nil && st.Mode().IsRegular() {
			c.Bytes += st.Size()
			if maxBytes > 0 && c.Bytes > maxBytes {
				return c, ByteBoundError{Bytes: c.Bytes, Limit: maxBytes, At: e.path}
			}
		}
	}
	for _, e := range entries {
		digest, err := contentDigest(filepath.Join(dir, e.path))
		if err != nil {
			return c, err
		}
		if e.untracked {
			c.Untracked[e.path] = digest
		} else {
			c.Modified[e.path] = digest
		}
	}
	return c, nil
}

// contentDigest hashes a worktree path; a deleted or non-regular path digests
// to "" (deletion is a recorded state, and a symlink's content is its target
// string, hashed so a retargeted link reads as a change).
func contentDigest(path string) (string, error) {
	st, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		t, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256([]byte("symlink\x00" + t))
		return hex.EncodeToString(sum[:]), nil
	}
	if !st.Mode().IsRegular() {
		return "", nil
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// IgnoredStat is one ignored path's cheap fingerprint. A stat walk, not a
// digest walk -- node_modules/ makes hashing the ignored tree unaffordable --
// and honestly scoped (§5.2 step 2): it reliably detects CREATION; its
// modification diff is a journal signal, never an enforcement mechanism.
type IgnoredStat struct {
	Size    int64
	ModTime time.Time
	// Dir marks a wholly-ignored directory. Its entry carries no stat, so it is
	// only ever created, never modified: a directory's mtime moves with every
	// build that touches its contents, and that churn is already reported per
	// file.
	Dir bool
}

// TakeIgnoredCensus records every ignored file, and every wholly-ignored
// directory, minus fixpoint's own artifact root -- on a continued run
// `.fixpoint/` is the run's scratch inside the project, and a census that reads
// the run's own journal writes would fail every task on the tool's bookkeeping
// (review run 20260813-003817). exclude names further artifact roots, repo-
// relative: logs.dir is configurable, and a `runlogs/` the operator ignored as
// told was censused, so the prompts and replies an attempt logs after step 2
// read as created and every discard deleted them (review run 20260929-141502,
// i7).
//
// Directories because git lists only files, so a discard that deleted a
// session's scratch/build.tmp left the scratch/ it made -- invisible to
// GitClean -- and the next attempt's `test -d scratch` gate passed on it
// (review run 20260929-125352). A directory counts when git itself reports it
// wholly ignored (--directory, which also reports an empty one) or lies beneath
// one on the way to an ignored file; git never reports a directory holding a
// tracked or un-ignored file that way. It CAN report one holding ignored files
// that were already there -- its tracked files deleted and the deletion staged
// -- which is why DiffIgnored never calls a directory above a first-census path
// created, and RemoveCreated never deletes recursively (review run
// 20260929-133423, i7). An empty directory nested inside a PRE-EXISTING ignored
// one is not seen: finding it means walking node_modules/ on every census,
// which findNestedGit prunes precisely to avoid.
func (g Git) TakeIgnoredCensus(ctx context.Context, dir string, exclude ...string) (map[string]IgnoredStat, error) {
	out, err := g.run(ctx, dir, "ls-files", "--others", "--ignored", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	dirs, err := g.run(ctx, dir, "ls-files", "--others", "--ignored", "--exclude-standard", "--directory", "-z")
	if err != nil {
		return nil, err
	}
	roots := append([]string{".fixpoint"}, exclude...)
	artifactRoot := func(path string) bool {
		for _, r := range roots {
			if path == r || strings.HasPrefix(path, r+"/") {
				return true
			}
		}
		return false
	}
	census := map[string]IgnoredStat{}
	for _, path := range strings.Split(strings.Trim(dirs, "\x00"), "\x00") {
		if p, isDir := strings.CutSuffix(path, "/"); isDir && p != "" && !artifactRoot(p) {
			census[p] = IgnoredStat{Dir: true}
		}
	}
	for _, path := range strings.Split(strings.Trim(out, "\x00"), "\x00") {
		if path == "" || artifactRoot(path) {
			continue
		}
		st, err := os.Lstat(filepath.Join(dir, path))
		if err != nil {
			continue // deleted between listing and stat: not present, not censused
		}
		census[path] = IgnoredStat{Size: st.Size(), ModTime: st.ModTime()}
		recordIgnoredParents(census, path)
	}
	return census, nil
}

// recordIgnoredParents adds the directories between path and the nearest
// wholly-ignored directory above it. Nothing is recorded when no such directory
// exists: src/debug.log's parent is src/, which holds tracked files, and a
// census that listed it could have it deleted as "created".
func recordIgnoredParents(census map[string]IgnoredStat, path string) {
	var between []string
	for p := filepath.ToSlash(filepath.Dir(path)); p != "." && p != "/"; p = filepath.ToSlash(filepath.Dir(p)) {
		if census[p].Dir {
			for _, d := range between {
				census[d] = IgnoredStat{Dir: true}
			}
			return
		}
		between = append(between, p)
	}
}

// DiffIgnored compares two ignored censuses: created paths are deleted before
// the gate runs (a coder that ran the build loses nothing the gate does not
// recreate); modified paths are reported, not failed -- build churn rewrites
// caches in place from task 2 onward, and stat metadata cannot carry an
// integrity claim anyway. The clean-clone check is the enforcement (§7.2).
//
// A path whose type changed counts as created (review run 20260929-133423,
// i3): an ignored file the attempt replaced with a directory is the attempt's
// directory, and diffed as "modified" it survived the discard for the next
// gate's `test -d` to pass on. The file it replaced is gone already, so
// removing the directory takes only what the attempt made.
//
// A directory is NEVER created while a first-census path lies beneath it
// (i7): git reports a directory wholly ignored by its current contents, so one
// whose tracked files the task deleted reads as new at step 6 while it still
// holds the operator's certs/dev.key. Its children are diffed on their own.
func DiffIgnored(pre, post map[string]IgnoredStat) (created, modified []string) {
	var holdsPre map[string]bool // every directory above a first-census path, built on first need
	for path, st := range post {
		prev, existed := pre[path]
		if st.Dir && !prev.Dir {
			if holdsPre == nil {
				holdsPre = ancestorsOf(pre)
			}
			if holdsPre[path] {
				continue
			}
		}
		switch {
		case !existed, prev.Dir != st.Dir:
			created = append(created, path)
		case prev != st:
			modified = append(modified, path)
		}
	}
	sort.Strings(created)
	sort.Strings(modified)
	return created, modified
}

// ancestorsOf is the set of directories strictly above any census path.
func ancestorsOf(census map[string]IgnoredStat) map[string]bool {
	out := map[string]bool{}
	for path := range census {
		for p := filepath.ToSlash(filepath.Dir(strings.TrimSuffix(path, "/"))); p != "." && p != "/" && !out[p]; p = filepath.ToSlash(filepath.Dir(p)) {
			out[p] = true
		}
	}
	return out
}

// GateDiff classifies every post-gate difference against the pre-gate census
// (§5.2 step 7).
type GateDiff struct {
	// MutatedSources: the gate modified or deleted a tracked file or a
	// pre-existing untracked file -- a distinct task failure; tool-written
	// bytes must not land under the coder's attribution.
	MutatedSources []string
	// GateGenerated: config-named committed files the gate created or rewrote
	// (lockfiles); staged into the commit with gate attribution.
	GateGenerated []string
	// Output: un-ignored paths that did not exist before the gate ran;
	// excluded from the commit and removed after EVERY gate run.
	Output []string
}

// ClassifyGateDiff compares the pre- and post-gate censuses. gateGenerated is
// the operator's implement.gate_generated list, matched by exact path.
func ClassifyGateDiff(pre, post Census, gateGenerated []string) GateDiff {
	gg := map[string]bool{}
	for _, g := range gateGenerated {
		gg[g] = true
	}
	var d GateDiff
	// Presence, not just the digest: a path missing from the post census MATCHES
	// HEAD again, which is not the "" a deletion digests to. Defaulting it to ""
	// let a gate regenerate a tracked file the coder deleted, byte-identical to
	// HEAD, unseen -- and the commit then staged a HEAD-identical path and the
	// deletion silently left an "implemented" task.
	changed := func(path, was string) {
		now, found := post.Untracked[path]
		if !found {
			now, found = post.Modified[path]
		}
		if found && now == was {
			return
		}
		if gg[path] {
			d.GateGenerated = append(d.GateGenerated, path)
			return
		}
		d.MutatedSources = append(d.MutatedSources, path)
	}
	for path, was := range pre.Untracked {
		changed(path, was)
	}
	for path, was := range pre.Modified {
		changed(path, was)
	}
	for path := range post.Untracked {
		if _, existed := pre.Untracked[path]; existed {
			continue
		}
		if _, existed := pre.Modified[path]; existed {
			continue
		}
		if gg[path] {
			d.GateGenerated = append(d.GateGenerated, path)
			continue
		}
		d.Output = append(d.Output, path)
	}
	// A tracked file first modified BY the gate (absent from pre entirely).
	for path := range post.Modified {
		if _, existed := pre.Modified[path]; existed {
			continue
		}
		if _, existed := pre.Untracked[path]; existed {
			continue
		}
		if gg[path] {
			d.GateGenerated = append(d.GateGenerated, path)
			continue
		}
		d.MutatedSources = append(d.MutatedSources, path)
	}
	sort.Strings(d.MutatedSources)
	sort.Strings(d.GateGenerated)
	sort.Strings(d.Output)
	return d
}
