// Package create holds the create-design pipeline's mechanics: the assignment
// snapshot, and (as later steps land) the anonymization labels, the cap formula,
// the provenance footer, and the atomic publish. The orchestrator drives the
// phases; this package is everything about them that is not agent scheduling.
//
// The pipeline itself is specified in docs/design/create-design.md, which was
// reviewed by review-design and revised twice; the decisions embodied here cite
// the findings that forced them.
package create

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Snapshotted describes the copy a run's phases will work against.
type Snapshotted struct {
	// Dir is the directory agents run in. For a file assignment it holds that one
	// file; for a directory assignment it is the copy of that directory.
	Dir string
	// Entry is the assignment's entry point relative to Dir: the file itself, or
	// "" when the assignment is a whole directory.
	Entry string
	// Files and Bytes are what was copied, for the provenance footer and the log.
	Files int
	Bytes int64
	// SkippedLinks counts symlinks left behind, for the same reporting: a snapshot
	// that silently dropped part of the assignment would misreport what was
	// designed against.
	SkippedLinks int
}

// Snapshot copies the assignment at src into dst and returns what the phases
// should run against.
//
// A COPY, not an inventory. The first draft of the specification pinned hashes
// and read the files live, and its own review called that what it is: drift
// detection, not a snapshot. Phases of one run must read the same bytes, and the
// only way to make that true is for the bytes the phases read to be the run's
// own.
//
// Copying also makes the exclusions structural rather than rule-based: `.git`
// and `.fixpoint` are never copied (VCS metadata and the tool's own logs are not
// the assignment), nor is anything under exclude -- the caller passes the -out
// path, so a rerun cannot ingest its previous deliverable and design against its
// own answer.
//
// Symlinks are skipped and counted, never followed and never replicated:
// following one ingests files outside the assignment, replicating one lets the
// snapshot read outside itself later -- both unfreeze exactly what the copy
// exists to freeze.
//
// maxBytes bounds the copy and a breach is a refusal, not a truncation: an
// assignment too large for the prompt caps fails here, at startup, before any
// session is paid for -- and a partial assignment silently passed on would be
// reviewed as though it were whole.
func Snapshot(src, dst string, exclude []string, maxBytes int64) (Snapshotted, error) {
	info, err := os.Stat(src)
	if err != nil {
		return Snapshotted{}, fmt.Errorf("assignment: %w", err)
	}
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return Snapshotted{}, err
	}
	if !info.IsDir() {
		out := Snapshotted{Dir: dst, Entry: filepath.Base(src)}
		n, err := copyFile(src, filepath.Join(dst, out.Entry), maxBytes)
		if err != nil {
			return Snapshotted{}, overBudget(err, src, maxBytes)
		}
		out.Files, out.Bytes = 1, n
		return out, nil
	}
	out := Snapshotted{Dir: dst}
	budget := maxBytes
	err = filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if excluded(path, rel, exclude) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		switch {
		case d.IsDir():
			return os.MkdirAll(filepath.Join(dst, rel), 0o700)
		case d.Type()&os.ModeSymlink != 0:
			out.SkippedLinks++
			return nil
		case !d.Type().IsRegular():
			// Sockets, devices, fifos: not assignment content, and copying one can
			// block forever.
			return nil
		}
		n, err := copyFile(path, filepath.Join(dst, rel), budget)
		if err != nil {
			return overBudget(err, path, maxBytes)
		}
		budget -= n
		out.Files++
		out.Bytes += n
		return nil
	})
	if err != nil {
		return Snapshotted{}, err
	}
	return out, nil
}

// excluded reports whether a source path is kept out of the snapshot. The
// built-ins are matched on any path segment by NAME (`.git`, `.fixpoint`);
// exclude entries are absolute paths matched against this path exactly or as a
// parent.
func excluded(abs, rel string, exclude []string) bool {
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if seg == ".git" || seg == ".fixpoint" {
			return true
		}
	}
	for _, e := range exclude {
		if e == "" {
			continue
		}
		if abs == e || strings.HasPrefix(abs, e+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// copyFile copies one regular file, refusing -- not truncating -- at the budget.
func copyFile(src, dst string, budget int64) (int64, error) {
	if budget < 0 {
		budget = 0
	}
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return 0, err
	}
	// Read one byte past the budget: hitting the limit exactly is fine, and the
	// extra byte is what distinguishes "fits" from "was cut".
	n, err := io.Copy(out, io.LimitReader(in, budget+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return n, err
	}
	if n > budget {
		return n, errOverBudget
	}
	return n, nil
}

// errOverBudget is copyFile's breach signal; overBudget turns it into the
// message, naming the WHOLE budget rather than whatever remained when the breach
// happened -- "exceeds the 3-byte budget" on a megabyte limit would read as a
// bug, not a refusal.
var errOverBudget = errors.New("assignment over the snapshot budget")

func overBudget(err error, at string, maxBytes int64) error {
	if errors.Is(err, errOverBudget) {
		return fmt.Errorf("assignment exceeds the %d-byte snapshot budget at %s; an assignment too large for the prompt caps must be split, not sampled", maxBytes, at)
	}
	return err
}
