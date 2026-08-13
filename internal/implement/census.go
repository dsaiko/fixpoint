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
}

// TakeIgnoredCensus records every ignored file, minus fixpoint's own artifact
// root -- on a continued run `.fixpoint/` is the run's scratch inside the
// project, and a census that reads the run's own journal writes would fail
// every task on the tool's bookkeeping (review run 20260813-003817).
func (g Git) TakeIgnoredCensus(ctx context.Context, dir string) (map[string]IgnoredStat, error) {
	out, err := g.run(ctx, dir, "ls-files", "--others", "--ignored", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	census := map[string]IgnoredStat{}
	for _, path := range strings.Split(strings.Trim(out, "\x00"), "\x00") {
		if path == "" || path == ".fixpoint" || strings.HasPrefix(path, ".fixpoint/") {
			continue
		}
		st, err := os.Lstat(filepath.Join(dir, path))
		if err != nil {
			continue // deleted between listing and stat: not present, not censused
		}
		census[path] = IgnoredStat{Size: st.Size(), ModTime: st.ModTime()}
	}
	return census, nil
}

// DiffIgnored compares two ignored censuses: created paths are deleted before
// the gate runs (a coder that ran the build loses nothing the gate does not
// recreate); modified paths are reported, not failed -- build churn rewrites
// caches in place from task 2 onward, and stat metadata cannot carry an
// integrity claim anyway. The clean-clone check is the enforcement (§7.2).
func DiffIgnored(pre, post map[string]IgnoredStat) (created, modified []string) {
	for path, st := range post {
		prev, existed := pre[path]
		switch {
		case !existed:
			created = append(created, path)
		case prev != st:
			modified = append(modified, path)
		}
	}
	sort.Strings(created)
	sort.Strings(modified)
	return created, modified
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
	changed := func(path, was string) {
		now := ""
		if v, ok := post.Untracked[path]; ok {
			now = v
		} else if v, ok := post.Modified[path]; ok {
			now = v
		}
		if now == was {
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
